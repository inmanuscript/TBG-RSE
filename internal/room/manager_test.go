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

func TestJoinBlockedAfterGameStart(t *testing.T) {
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
	defer conn1.Close(websocket.StatusNormalClosure, "")

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

	write(conn1, "CREATE_ROOM", map[string]string{"name": "Alice", "color": "#ff0000"})
	joined := readEvent(conn1, "ROOM_JOINED")
	roomCode := joined["payload"].(map[string]any)["room_code"].(string)

	write(conn1, "BUY_CARDS", map[string]any{"count": 0})
	readEvent(conn1, "STATE_UPDATE")
	readEvent(conn1, "NOTIFICATION")

	conn2, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close(websocket.StatusNormalClosure, "")

	write(conn2, "JOIN_ROOM", map[string]any{
		"room_code": roomCode,
		"name":      "Bob",
		"color":     "#00ff00",
		"seat":      2,
	})
	errMsg := readEvent(conn2, "ERROR")
	payload := errMsg["payload"].(map[string]any)
	if payload["message"] != "game already started; reclaim an offline seat" {
		t.Fatalf("error=%v", payload["message"])
	}
}

func TestReclaimOfflineSeat(t *testing.T) {
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
	rejoined := readEvent(conn2, "ROOM_JOINED")
	rePayload := rejoined["payload"].(map[string]any)
	if rePayload["player_id"] != playerID {
		t.Fatalf("player_id=%v want %s", rePayload["player_id"], playerID)
	}
	if rePayload["reconnect_token"] == "" {
		t.Fatal("expected new reconnect token")
	}
}

func TestPeekRoomIncludesPlayersAndJoinPolicy(t *testing.T) {
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
	defer conn1.Close(websocket.StatusNormalClosure, "")

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

	write(conn1, "CREATE_ROOM", map[string]string{"name": "Alice", "color": "#ff0000"})
	joined := readEvent(conn1, "ROOM_JOINED")
	roomCode := joined["payload"].(map[string]any)["room_code"].(string)

	write(conn1, "PEEK_ROOM", map[string]string{"room_code": roomCode})
	info := readEvent(conn1, "ROOM_INFO")
	peek := info["payload"].(map[string]any)
	if peek["allow_new_join"] != true {
		t.Fatalf("allow_new_join=%v", peek["allow_new_join"])
	}
	players, ok := peek["players"].([]any)
	if !ok || len(players) != 1 {
		t.Fatalf("players=%v", peek["players"])
	}

	write(conn1, "BUY_CARDS", map[string]any{"count": 0})
	readEvent(conn1, "STATE_UPDATE")
	readEvent(conn1, "NOTIFICATION")

	write(conn1, "PEEK_ROOM", map[string]string{"room_code": roomCode})
	info = readEvent(conn1, "ROOM_INFO")
	peek = info["payload"].(map[string]any)
	if peek["allow_new_join"] != false {
		t.Fatalf("allow_new_join=%v", peek["allow_new_join"])
	}
}

func TestDisconnectAutoPassDuringTurn(t *testing.T) {
	mgr := NewManager(nil, nil)
	srv := httptest.NewServer(http.HandlerFunc(mgr.HandleWS))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
	playerID := func(msg map[string]any) string {
		return msg["payload"].(map[string]any)["player_id"].(string)
	}

	write(conn1, "CREATE_ROOM", map[string]any{"name": "Alice", "color": "#ff0000", "seat": 1})
	joined1 := readEvent(conn1, "ROOM_JOINED")
	roomCode := joined1["payload"].(map[string]any)["room_code"].(string)
	id1 := playerID(joined1)

	conn2, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close(websocket.StatusNormalClosure, "")

	write(conn2, "JOIN_ROOM", map[string]any{
		"room_code": roomCode,
		"name":      "Bob",
		"color":     "#00ff00",
		"seat":      2,
	})
	joined2 := readEvent(conn2, "ROOM_JOINED")
	readEvent(conn2, "STATE_UPDATE")
	readEvent(conn2, "NOTIFICATION")
	id2 := playerID(joined2)

	write(conn1, "BUY_CARDS", map[string]any{"count": 0})
	readEvent(conn1, "STATE_UPDATE")
	readEvent(conn1, "NOTIFICATION")
	write(conn2, "BUY_CARDS", map[string]any{"count": 0})
	readEvent(conn1, "STATE_UPDATE")
	readEvent(conn1, "NOTIFICATION")
	readEvent(conn2, "STATE_UPDATE")
	readEvent(conn2, "NOTIFICATION")

	st := readEvent(conn1, "STATE_UPDATE")
	if phaseOf(st) != "ACTION" {
		t.Fatalf("phase=%s", phaseOf(st))
	}
	payload := st["payload"].(map[string]any)
	if payload["active_player_id"] != id1 {
		t.Fatalf("active=%v want %s", payload["active_player_id"], id1)
	}

	_ = conn1.Close(websocket.StatusNormalClosure, "")

	for {
		msg := readEvent(conn2, "STATE_UPDATE")
		p := msg["payload"].(map[string]any)
		if p["phase"] != "ACTION" {
			t.Fatalf("phase=%s", p["phase"])
		}
		if p["active_player_id"] == id2 {
			players := p["players"].(map[string]any)
			a := players[id1].(map[string]any)
			if a["passed"] != true {
				t.Fatalf("alice passed=%v", a["passed"])
			}
			return
		}
	}
}

func TestWSGlobalParams(t *testing.T) {
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

	// Update temperature directly via WS
	write("UPDATE_GLOBAL_PARAM", map[string]any{
		"param_id":    "temperature",
		"delta_steps": 1,
		"grant_tr":    true,
	})
	st := readEvent("STATE_UPDATE")
	readEvent("NOTIFICATION")

	payload := st["payload"].(map[string]any)
	gp := payload["global_params"].(map[string]any)
	temp := gp["temperature"].(map[string]any)
	if temp["current"].(float64) != -28 {
		t.Fatalf("temp=%v want -28", temp["current"])
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
