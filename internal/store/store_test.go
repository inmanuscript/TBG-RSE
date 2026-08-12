package store_test

import (
	"path/filepath"
	"testing"

	"github.com/inmanuscript/TBG-RSE/internal/game"
	"github.com/inmanuscript/TBG-RSE/internal/store"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	state := game.NewGameState("ABC123")
	state.Generation = 3
	state.Phase = game.PhaseProductionWait
	p := game.NewPlayer("pid1", "Alice", "#ff0000", 2)
	p.TR = 25
	p.IsReady = true
	p.Resources[game.Plant] = game.ResourceState{Stock: 8, Production: 2}
	state.Players[p.ID] = p

	logs := []game.AuditLog{{
		ID:       "log1",
		PlayerID: "pid1",
		Type:     "RESOURCE_CHANGE",
		Detail:   "test",
		Delta:    map[string]int{"Plant_stock": 1},
	}}
	tokens := map[string]string{"pid1": "tok123"}

	if err := st.SaveRoom("ABC123", "pid1", state, tokens, logs); err != nil {
		t.Fatal(err)
	}

	rooms, err := st.LoadAllRooms()
	if err != nil {
		t.Fatal(err)
	}
	if len(rooms) != 1 {
		t.Fatalf("rooms=%d", len(rooms))
	}
	r := rooms[0]
	if r.Code != "ABC123" || r.HostPlayerID != "pid1" {
		t.Fatalf("%+v", r)
	}
	if r.State.Generation != 3 || r.State.Phase != game.PhaseProductionWait {
		t.Fatalf("state %+v", r.State)
	}
	got := r.State.Players["pid1"]
	if got == nil || got.Name != "Alice" || got.TR != 25 || !got.IsReady {
		t.Fatalf("player %+v", got)
	}
	if got.Resources[game.Plant].Stock != 8 || got.Resources[game.Plant].Production != 2 {
		t.Fatalf("resources %+v", got.Resources[game.Plant])
	}
	if r.PlayersMeta["pid1"].ReconnectToken != "tok123" {
		t.Fatalf("token %+v", r.PlayersMeta)
	}
	if len(r.AuditLogs) != 1 || r.AuditLogs[0].Detail != "test" {
		t.Fatalf("logs %+v", r.AuditLogs)
	}
}

