package room

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func TestWSCreateResearchPassProduce(t *testing.T) {
	mgr := NewManager(nil, nil)
	srv := httptest.NewServer(http.HandlerFunc(mgr.HandleWS))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	wsURL := "ws" + srv.URL[len("http"):]
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	write := func(action string, payload any) {
		t.Helper()
		if err := wsjson.Write(ctx, conn, map[string]any{"action": action, "payload": payload}); err != nil {
			t.Fatal(err)
		}
	}
	readEvent := func(want string) map[string]any {
		t.Helper()
		for {
			var msg map[string]any
			if err := wsjson.Read(ctx, conn, &msg); err != nil {
				t.Fatalf("read %s: %v", want, err)
			}
			if msg["event"] == want {
				return msg
			}
		}
	}

	write("CREATE_ROOM", map[string]string{"name": "Alice", "color": "#ff0000"})
	readEvent("ROOM_JOINED")

	write("BUY_CARDS", map[string]any{"count": 0})
	st := readEvent("STATE_UPDATE")
	readEvent("NOTIFICATION")
	phase := phaseOf(st)
	if phase != "ACTION" {
		t.Fatalf("phase after research=%s", phase)
	}

	write("PASS", map[string]any{})
	st = readEvent("STATE_UPDATE")
	readEvent("NOTIFICATION")
	if phaseOf(st) != "PRODUCTION_WAIT" {
		t.Fatalf("phase after pass=%s", phaseOf(st))
	}

	write("READY_PRODUCTION", map[string]any{})
	st = readEvent("STATE_UPDATE")
	gen, phase := genPhase(st)
	if phase != "RESEARCH" || gen != 2 {
		t.Fatalf("after prod gen=%d phase=%s", gen, phase)
	}
}

func phaseOf(msg map[string]any) string {
	_, p := genPhase(msg)
	return p
}

func genPhase(msg map[string]any) (int, string) {
	raw, _ := json.Marshal(msg["payload"])
	var st struct {
		Phase      string `json:"phase"`
		Generation int    `json:"generation"`
	}
	_ = json.Unmarshal(raw, &st)
	return st.Generation, st.Phase
}
