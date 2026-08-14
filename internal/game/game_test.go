package game

import "testing"

func TestApplyResourceDeltaStock(t *testing.T) {
	p := NewPlayer("1", "Alice", "#ff0000", 1)
	_, delta, err := ApplyResourceDelta(p, TargetStock, Plant, 5)
	if err != nil {
		t.Fatal(err)
	}
	if p.Resources[Plant].Stock != 5 {
		t.Fatalf("stock=%d", p.Resources[Plant].Stock)
	}
	if delta["Plant_stock"] != 5 {
		t.Fatalf("delta=%v", delta)
	}
	_, _, err = ApplyResourceDelta(p, TargetStock, Plant, -6)
	if err == nil {
		t.Fatal("expected negative stock error")
	}
}

func TestApplyResourceDeltaProductionMC(t *testing.T) {
	p := NewPlayer("1", "Alice", "#ff0000", 1)
	_, _, err := ApplyResourceDelta(p, TargetProduction, MC, -5)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = ApplyResourceDelta(p, TargetProduction, MC, -1)
	if err == nil {
		t.Fatal("expected MC production floor error")
	}
	_, _, err = ApplyResourceDelta(p, TargetProduction, Steel, -1)
	if err == nil {
		t.Fatal("expected steel production floor error")
	}
}

func TestApplyConversionAndProjects(t *testing.T) {
	p := NewPlayer("1", "Alice", "#ff0000", 1)
	_, _, err := ApplyConversion(p, "greenery")
	if err == nil {
		t.Fatal("expected insufficient plants")
	}
	rs := p.Resources[Plant]
	rs.Stock = 8
	p.Resources[Plant] = rs
	_, _, err = ApplyConversion(p, "greenery")
	if err != nil {
		t.Fatal(err)
	}
	if p.Score.GreeneryTiles != 1 {
		t.Fatalf("greenery tiles=%d", p.Score.GreeneryTiles)
	}

	p.Resources[MC] = ResourceState{Stock: 11}
	_, _, err = ApplyStandardProject(p, "power_plant", 0)
	if err != nil {
		t.Fatal(err)
	}
	if p.Resources[Energy].Production != 1 || p.Resources[MC].Stock != 0 {
		t.Fatalf("mc=%d energyProd=%d", p.Resources[MC].Stock, p.Resources[Energy].Production)
	}
}

func TestExecuteProductionOrder(t *testing.T) {
	state := NewGameState("ROOM")
	p := NewPlayer("1", "Alice", "#ff0000", 1)
	p.TR = 20
	p.Resources[MC] = ResourceState{Stock: 10, Production: -3}
	p.Resources[Energy] = ResourceState{Stock: 5, Production: 2}
	p.Resources[Heat] = ResourceState{Stock: 1, Production: 1}
	p.Resources[Plant] = ResourceState{Stock: 0, Production: 3}
	p.IsReady = true
	state.Players[p.ID] = p
	state.Phase = PhaseProductionWait
	state.Generation = 1
	state.TurnOrder = []string{p.ID}

	ExecuteProduction(&state)

	if state.Generation != 2 {
		t.Fatalf("gen=%d", state.Generation)
	}
	if state.Phase != PhaseResearch {
		t.Fatalf("phase=%s", state.Phase)
	}
	if p.Resources[Energy].Stock != 2 {
		t.Fatalf("energy stock=%d want 2", p.Resources[Energy].Stock)
	}
	if p.Resources[Heat].Stock != 7 {
		t.Fatalf("heat stock=%d want 7", p.Resources[Heat].Stock)
	}
	if p.Resources[MC].Stock != 27 {
		t.Fatalf("mc stock=%d want 27", p.Resources[MC].Stock)
	}
}

func TestAbsentPlayerDuringAction(t *testing.T) {
	state := NewGameState("R")
	a := NewPlayer("a", "A", "#1", 1)
	b := NewPlayer("b", "B", "#2", 2)
	state.Players["a"] = a
	state.Players["b"] = b
	StartActionPhase(&state)

	if state.ActivePlayerID != "a" {
		t.Fatalf("active=%s", state.ActivePlayerID)
	}
	detail, ok := AbsentPlayer(&state, "a")
	if !ok || detail == "" {
		t.Fatalf("absent a ok=%v detail=%q", ok, detail)
	}
	if !a.Passed || state.ActivePlayerID != "b" {
		t.Fatalf("after absent a: passed=%v active=%s", a.Passed, state.ActivePlayerID)
	}
}

func TestAbsentPlayerDuringResearch(t *testing.T) {
	state := NewGameState("R")
	a := NewPlayer("a", "A", "#1", 1)
	b := NewPlayer("b", "B", "#2", 2)
	b.ResearchDone = true
	state.Players["a"] = a
	state.Players["b"] = b

	detail, ok := AbsentPlayer(&state, "a")
	if !ok {
		t.Fatal("expected absent")
	}
	if state.Phase != PhaseAction {
		t.Fatalf("phase=%s detail=%q", state.Phase, detail)
	}
}

func TestTurnPassAndActions(t *testing.T) {
	state := NewGameState("ROOM")
	a := NewPlayer("a", "A", "#1", 1)
	b := NewPlayer("b", "B", "#2", 2)
	state.Players["a"] = a
	state.Players["b"] = b
	StartActionPhase(&state)

	if state.ActivePlayerID != "a" {
		t.Fatalf("active=%s want a (seat 1)", state.ActivePlayerID)
	}
	if state.TurnOrder[0] != "a" || state.TurnOrder[1] != "b" {
		t.Fatalf("order=%v", state.TurnOrder)
	}

	all, err := Pass(&state, "a")
	if err != nil || all {
		t.Fatalf("pass all=%v err=%v", all, err)
	}
	if !a.Passed || state.ActivePlayerID != "b" {
		t.Fatalf("passed=%v active=%s", a.Passed, state.ActivePlayerID)
	}

	adv, err := ClaimAction(&state, "b")
	if err != nil || adv {
		t.Fatalf("claim adv=%v err=%v", adv, err)
	}
	adv, err = ClaimAction(&state, "b")
	if err != nil || !adv {
		t.Fatalf("second claim adv=%v err=%v", adv, err)
	}
	if state.ActivePlayerID != "b" || state.ActionsThisTurn != 0 {
		t.Fatalf("active=%s actions=%d", state.ActivePlayerID, state.ActionsThisTurn)
	}

	all, err = Pass(&state, "b")
	if err != nil || !all {
		t.Fatalf("final pass all=%v err=%v", all, err)
	}
	if state.Phase != PhaseProductionWait {
		t.Fatalf("phase=%s", state.Phase)
	}
}

func TestColorInUse(t *testing.T) {
	state := NewGameState("R")
	state.Players["a"] = NewPlayer("a", "A", "#FF5555", 1)
	if !ColorInUse(&state, "#ff5555", "") {
		t.Fatal("expected color in use")
	}
	if ColorInUse(&state, "#55AAFF", "") {
		t.Fatal("expected free color")
	}
}

func TestRebuildTurnOrderBySeat(t *testing.T) {
	state := NewGameState("R")
	state.Players["c"] = NewPlayer("c", "C", "#3", 3)
	state.Players["a"] = NewPlayer("a", "A", "#1", 1)
	state.Players["b"] = NewPlayer("b", "B", "#2", 2)
	RebuildTurnOrderBySeat(&state)
	want := []string{"a", "b", "c"}
	for i := range want {
		if state.TurnOrder[i] != want[i] {
			t.Fatalf("order=%v want %v", state.TurnOrder, want)
		}
	}
	if err := ValidateSeat(&state, 2, ""); err == nil {
		t.Fatal("expected seat taken")
	}
	if err := ValidateSeat(&state, 4, ""); err != nil {
		t.Fatal(err)
	}
}

func TestAllowNewJoin(t *testing.T) {
	state := NewGameState("R")
	if !AllowNewJoin(&state) {
		t.Fatal("generation 1 research should allow new join")
	}
	state.Phase = PhaseAction
	if AllowNewJoin(&state) {
		t.Fatal("action phase should not allow new join")
	}
	state.Phase = PhaseResearch
	state.Generation = 2
	if AllowNewJoin(&state) {
		t.Fatal("generation 2 research should not allow new join")
	}
}

func TestBuyCardsAndFinishResearch(t *testing.T) {
	state := NewGameState("R")
	a := NewPlayer("a", "A", "#1", 1)
	a.Resources[MC] = ResourceState{Stock: 12}
	state.Players["a"] = a
	_, _, err := BuyCards(a, 4, state.Generation)
	if err != nil {
		t.Fatal(err)
	}
	if a.Resources[MC].Stock != 0 || !a.ResearchDone {
		t.Fatalf("mc=%d done=%v", a.Resources[MC].Stock, a.ResearchDone)
	}
	if !MaybeFinishResearch(&state) {
		t.Fatal("should finish research")
	}
	if state.Phase != PhaseAction {
		t.Fatalf("phase=%s", state.Phase)
	}
}

func TestBuyCardsGenerationCap(t *testing.T) {
	a := NewPlayer("a", "A", "#1", 1)
	a.Resources[MC] = ResourceState{Stock: 100}

	// Generation 1 deals a larger starting hand — up to 10 may be bought.
	if _, _, err := BuyCards(a, 10, 1); err != nil {
		t.Fatalf("gen1 buy 10: %v", err)
	}
	if _, _, err := BuyCards(a, 11, 1); err == nil {
		t.Fatal("gen1 buy 11 should fail")
	}

	// Later generations cap at 4.
	b := NewPlayer("b", "B", "#2", 2)
	b.Resources[MC] = ResourceState{Stock: 100}
	if _, _, err := BuyCards(b, 4, 2); err != nil {
		t.Fatalf("gen2 buy 4: %v", err)
	}
	c := NewPlayer("c", "C", "#3", 3)
	c.Resources[MC] = ResourceState{Stock: 100}
	if _, _, err := BuyCards(c, 5, 2); err == nil {
		t.Fatal("gen2 buy 5 should fail")
	}
}

func TestTotalVP(t *testing.T) {
	p := NewPlayer("1", "A", "#1", 1)
	p.TR = 30
	p.Score = ScoreSheet{GreeneryTiles: 5, CityTiles: 2, CityAdjGreenery: 3, Milestone: 5, Award: 5, Cards: 7, Other: 1}
	if p.TotalVP() != 30+5+2+3+5+5+7+1 {
		t.Fatalf("vp=%d", p.TotalVP())
	}
}
