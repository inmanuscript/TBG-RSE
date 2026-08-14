package room

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func TestReclaimThenUpdateResource(t *testing.T) {
	mgr := NewManager(nil, nil)
	srv := httptest.NewServer(http.HandlerFunc(mgr.HandleWS))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	wsURL := "ws" + srv.URL[len("http"):]
	conn1, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	write := func(conn *websocket.Conn, action string, payload any) {
		t.Helper()
		if err := wsjson.Write(ctx, conn, map[string]any{"action": action, "payload": payload}); err != nil {
			t.Fatal(err)
		}
	}
	readEvent := func(conn *websocket.Conn, want string) map[string]any {
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

	write(conn1, "CREATE_ROOM", map[string]any{"name": "Alice", "color": "#ff0000", "seat": 1})
	joined := readEvent(conn1, "ROOM_JOINED")
	payload := joined["payload"].(map[string]any)
	roomCode := payload["room_code"].(string)
	playerID := payload["player_id"].(string)
	_ = conn1.Close(websocket.StatusNormalClosure, "")

	conn2, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close(websocket.StatusNormalClosure, "")

	write(conn2, "JOIN_ROOM", map[string]any{
		"room_code":         roomCode,
		"name":              "Alice",
		"reclaim_player_id": playerID,
	})
	readEvent(conn2, "ROOM_JOINED")
	// Drain broadcast from reclaim (STATE_UPDATE + NOTIFICATION).
	readEvent(conn2, "STATE_UPDATE")
	readEvent(conn2, "NOTIFICATION")

	write(conn2, "UPDATE_RESOURCE", map[string]any{
		"target":   "stock",
		"resource": "MC",
		"delta":    5,
	})
	st := readEvent(conn2, "STATE_UPDATE")
	readEvent(conn2, "NOTIFICATION")

	rawPlayers := st["payload"].(map[string]any)["players"].(map[string]any)
	pl := rawPlayers[playerID].(map[string]any)
	resources := pl["resources"].(map[string]any)
	mc := resources["MC"].(map[string]any)
	if mc["stock"].(float64) != 5 {
		t.Fatalf("mc stock=%v want 5", mc["stock"])
	}
}
