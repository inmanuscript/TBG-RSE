package room

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

// A solo host finishing their own research (even buying 0 cards) satisfies
// MaybeFinishResearch's "every current player is done" check and advances
// Phase to ACTION — but nobody has actually played a turn yet, so a second
// player using the room code must still be able to join. Regression test
// for the "just sharing the room code locks the room" bug.
func TestJoinAllowedAfterSoloResearch(t *testing.T) {
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
	st := readEvent(conn1, "STATE_UPDATE")
	readEvent(conn1, "NOTIFICATION")
	if phaseOf(st) != "ACTION" {
		t.Fatalf("phase after solo research=%s", phaseOf(st))
	}

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
	payload := joined2["payload"].(map[string]any)
	state := payload["state"].(map[string]any)
	// Bob never got a research step — the room should have reopened it
	// instead of leaving him stranded in the action phase with nothing bought.
	if state["phase"] != "RESEARCH" {
		t.Fatalf("phase after Bob joins=%v, want RESEARCH reopened for him", state["phase"])
	}
}

func TestJoinBlockedAfterActionTaken(t *testing.T) {
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

	// Bob joins while research is still open, so the pair reaches the action
	// phase together with a real generation under way.
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
	readEvent(conn2, "ROOM_JOINED")
	readEvent(conn1, "STATE_UPDATE")
	readEvent(conn1, "NOTIFICATION")

	write(conn1, "BUY_CARDS", map[string]any{"count": 0})
	readEvent(conn1, "STATE_UPDATE")
	readEvent(conn1, "NOTIFICATION")
	write(conn2, "BUY_CARDS", map[string]any{"count": 0})
	st := readEvent(conn1, "STATE_UPDATE")
	readEvent(conn1, "NOTIFICATION")
	if phaseOf(st) != "ACTION" {
		t.Fatalf("phase=%s", phaseOf(st))
	}

	// Alice actually plays her turn now — this is the point where the room
	// should genuinely lock out new joins.
	write(conn1, "CLAIM_ACTION", map[string]any{})
	readEvent(conn1, "STATE_UPDATE")
	readEvent(conn1, "NOTIFICATION")

	conn3, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn3.Close(websocket.StatusNormalClosure, "")

	write(conn3, "JOIN_ROOM", map[string]any{
		"room_code": roomCode,
		"name":      "Carol",
		"color":     "#0000ff",
		"seat":      3,
	})
	errMsg := readEvent(conn3, "ERROR")
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

	// Solo research finishing (and advancing Phase to ACTION) must not lock
	// new joins by itself — nobody has actually played a turn yet.
	write(conn1, "PEEK_ROOM", map[string]string{"room_code": roomCode})
	info = readEvent(conn1, "ROOM_INFO")
	peek = info["payload"].(map[string]any)
	if peek["allow_new_join"] != true {
		t.Fatalf("allow_new_join=%v, want true (no turn played yet)", peek["allow_new_join"])
	}

	// Once Alice actually claims her turn, the room is genuinely under way.
	write(conn1, "CLAIM_ACTION", map[string]any{})
	readEvent(conn1, "STATE_UPDATE")
	readEvent(conn1, "NOTIFICATION")

	write(conn1, "PEEK_ROOM", map[string]string{"room_code": roomCode})
	info = readEvent(conn1, "ROOM_INFO")
	peek = info["payload"].(map[string]any)
	if peek["allow_new_join"] != false {
		t.Fatalf("allow_new_join=%v, want false after a turn was claimed", peek["allow_new_join"])
	}
}

// Regression test: a dropped connection — however long it lasts — must
// never by itself skip the player's turn. Only an explicit LEAVE or a
// SKIP_PLAYER from someone else does that (see TestLeaveSkipsOwnTurn /
// TestSkipPlayerByOtherPlayer below). This also checks that the live
// "online" presence flag flips off promptly so the UI can offer the skip
// option, and flips back on when the player reconnects.
func TestDisconnectNeverSkipsTurnUntilExplicit(t *testing.T) {
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
	joined1 := readEvent(conn1, "ROOM_JOINED")
	p1 := joined1["payload"].(map[string]any)
	roomCode := p1["room_code"].(string)
	id1 := p1["player_id"].(string)
	token1 := p1["reconnect_token"].(string)

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
	readEvent(conn2, "ROOM_JOINED")
	readEvent(conn1, "STATE_UPDATE")
	readEvent(conn1, "NOTIFICATION")

	write(conn1, "BUY_CARDS", map[string]any{"count": 0})
	readEvent(conn1, "STATE_UPDATE")
	readEvent(conn1, "NOTIFICATION")
	write(conn2, "BUY_CARDS", map[string]any{"count": 0})
	st := readEvent(conn1, "STATE_UPDATE")
	readEvent(conn1, "NOTIFICATION")
	if phaseOf(st) != "ACTION" {
		t.Fatalf("phase=%s", phaseOf(st))
	}

	// Alice drops. Bob should promptly see her flip to offline, with her
	// turn state completely untouched.
	_ = conn1.Close(websocket.StatusNormalClosure, "")

	offlineSeen := false
	for !offlineSeen {
		msg := readEvent(conn2, "STATE_UPDATE")
		players := msg["payload"].(map[string]any)["players"].(map[string]any)
		alice := players[id1].(map[string]any)
		if alice["passed"] == true {
			t.Fatal("Alice's turn was skipped by a plain disconnect")
		}
		if alice["online"] == false {
			offlineSeen = true
		}
	}

	// Reconnect on a fresh connection — no expiry, no grace window to race.
	conn1b, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn1b.Close(websocket.StatusNormalClosure, "")
	write(conn1b, "RECONNECT", map[string]any{
		"room_code":       roomCode,
		"player_id":       id1,
		"reconnect_token": token1,
	})
	rejoined := readEvent(conn1b, "ROOM_JOINED")
	state := rejoined["payload"].(map[string]any)["state"].(map[string]any)
	players := state["players"].(map[string]any)
	alice := players[id1].(map[string]any)
	if alice["passed"] == true {
		t.Fatal("Alice's turn was skipped despite reconnecting")
	}
	if alice["online"] != true {
		t.Fatalf("alice online=%v after reconnect, want true", alice["online"])
	}
	if state["active_player_id"] != id1 {
		t.Fatalf("active_player_id=%v, want %s to still hold the turn", state["active_player_id"], id1)
	}
}

// The client sends LEAVE right before closing its own socket when the
// player deliberately steps away — this is the only thing that should skip
// their turn.
func TestLeaveSkipsOwnTurn(t *testing.T) {
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

	write(conn1, "CREATE_ROOM", map[string]any{"name": "Alice", "color": "#ff0000", "seat": 1})
	joined1 := readEvent(conn1, "ROOM_JOINED")
	id1 := joined1["payload"].(map[string]any)["player_id"].(string)

	write(conn1, "BUY_CARDS", map[string]any{"count": 0})
	st := readEvent(conn1, "STATE_UPDATE")
	readEvent(conn1, "NOTIFICATION")
	if phaseOf(st) != "ACTION" {
		t.Fatalf("phase=%s", phaseOf(st))
	}

	write(conn1, "LEAVE", map[string]any{})
	st = readEvent(conn1, "STATE_UPDATE")
	players := st["payload"].(map[string]any)["players"].(map[string]any)
	alice := players[id1].(map[string]any)
	if alice["passed"] != true {
		t.Fatalf("alice passed=%v after LEAVE, want true", alice["passed"])
	}
}

func TestSkipPlayerByOtherPlayer(t *testing.T) {
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
	joined1 := readEvent(conn1, "ROOM_JOINED")
	roomCode := joined1["payload"].(map[string]any)["room_code"].(string)
	id1 := joined1["payload"].(map[string]any)["player_id"].(string)

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
	readEvent(conn2, "ROOM_JOINED")
	readEvent(conn1, "STATE_UPDATE")
	readEvent(conn1, "NOTIFICATION")

	write(conn1, "BUY_CARDS", map[string]any{"count": 0})
	readEvent(conn1, "STATE_UPDATE")
	readEvent(conn1, "NOTIFICATION")

	// Bob can't skip Alice while she's still online.
	write(conn2, "SKIP_PLAYER", map[string]any{"target_player_id": id1})
	errMsg := readEvent(conn2, "ERROR")
	if errMsg["payload"].(map[string]any)["message"] == "" {
		t.Fatal("expected an error skipping an online player")
	}

	write(conn2, "BUY_CARDS", map[string]any{"count": 0})
	st := readEvent(conn1, "STATE_UPDATE")
	readEvent(conn1, "NOTIFICATION")
	if phaseOf(st) != "ACTION" {
		t.Fatalf("phase=%s", phaseOf(st))
	}

	// Alice vanishes mid-turn.
	_ = conn1.Close(websocket.StatusNormalClosure, "")
	for {
		msg := readEvent(conn2, "STATE_UPDATE")
		players := msg["payload"].(map[string]any)["players"].(map[string]any)
		if players[id1].(map[string]any)["online"] == false {
			break
		}
	}

	// Now Bob can skip her.
	write(conn2, "SKIP_PLAYER", map[string]any{"target_player_id": id1})
	st = readEvent(conn2, "STATE_UPDATE")
	notif := readEvent(conn2, "NOTIFICATION")
	players := st["payload"].(map[string]any)["players"].(map[string]any)
	alice := players[id1].(map[string]any)
	if alice["passed"] != true {
		t.Fatalf("alice passed=%v after SKIP_PLAYER, want true", alice["passed"])
	}
	msg := notif["payload"].(map[string]any)["message"].(string)
	if !strings.Contains(msg, "skipped by Bob") {
		t.Fatalf("notification=%q, want it to credit Bob", msg)
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
