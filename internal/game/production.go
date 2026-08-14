package game

import "fmt"

// BuyCards spends 3 MC per card during RESEARCH phase. The cap depends on
// generation — generation 1 deals a larger starting hand than later ones.
func BuyCards(p *PlayerState, count int, generation int) (detail string, auditDelta map[string]int, err error) {
	if p == nil {
		return "", nil, fmt.Errorf("player is nil")
	}
	if p.ResearchDone {
		return "", nil, fmt.Errorf("already finished research")
	}
	max := MaxCardsBuyForGeneration(generation)
	if count < 0 || count > max {
		return "", nil, fmt.Errorf("card buy count must be 0–%d", max)
	}
	cost := count * CardBuyCost
	if cost > 0 {
		if err := spendMC(p, cost); err != nil {
			return "", nil, err
		}
	}
	p.ResearchDone = true
	detail = fmt.Sprintf("%s bought %d card(s) (−%d MC)", p.Name, count, cost)
	return detail, map[string]int{"MC_stock": -cost, "cards_bought": count}, nil
}

// MaybeFinishResearch moves to ACTION when everyone is done.
func MaybeFinishResearch(state *GameState) bool {
	if state.Phase != PhaseResearch || len(state.Players) == 0 {
		return false
	}
	for _, p := range state.Players {
		if !p.ResearchDone {
			return false
		}
	}
	StartActionPhase(state)
	return true
}

func StartActionPhase(state *GameState) {
	RebuildTurnOrderBySeat(state)
	for _, p := range state.Players {
		p.Passed = false
		p.IsReady = false
		p.ResearchDone = false
	}
	state.Phase = PhaseAction
	state.ActionsThisTurn = 0
	if len(state.TurnOrder) == 0 {
		state.ActivePlayerID = ""
		return
	}
	if state.FirstPlayerIndex < 0 || state.FirstPlayerIndex >= len(state.TurnOrder) {
		state.FirstPlayerIndex = 0
	}
	state.ActivePlayerID = state.TurnOrder[state.FirstPlayerIndex]
}

// RebuildTurnOrderBySeat sets turn order to ascending seat (clockwise from seat 1).
func RebuildTurnOrderBySeat(state *GameState) {
	type seatPlayer struct {
		id   string
		seat int
	}
	list := make([]seatPlayer, 0, len(state.Players))
	for id, p := range state.Players {
		seat := p.Seat
		if seat < 1 {
			seat = MaxSeats + 1
		}
		list = append(list, seatPlayer{id: id, seat: seat})
	}
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].seat < list[i].seat || (list[j].seat == list[i].seat && list[j].id < list[i].id) {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	order := make([]string, len(list))
	for i, sp := range list {
		order[i] = sp.id
	}
	state.TurnOrder = order
}

// ValidateSeat checks seat range and uniqueness.
func ValidateSeat(state *GameState, seat int, exceptPlayerID string) error {
	if seat < 1 || seat > MaxSeats {
		return fmt.Errorf("seat must be 1–%d", MaxSeats)
	}
	for id, p := range state.Players {
		if id == exceptPlayerID {
			continue
		}
		if p.Seat == seat {
			return fmt.Errorf("seat %d already taken", seat)
		}
	}
	return nil
}

func TakenSeats(state *GameState) []int {
	out := make([]int, 0, len(state.Players))
	for _, p := range state.Players {
		if p.Seat >= 1 && p.Seat <= MaxSeats {
			out = append(out, p.Seat)
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// AddPlayerToTurnOrder rebuilds clockwise order after a join.
func AddPlayerToTurnOrder(state *GameState, playerID string) {
	RebuildTurnOrderBySeat(state)
	if state.Phase == PhaseAction && state.ActivePlayerID == "" && len(state.TurnOrder) > 0 {
		state.ActivePlayerID = state.TurnOrder[0]
	}
}

// ColorInUse reports whether another player already uses this color.
func ColorInUse(state *GameState, color string, exceptPlayerID string) bool {
	want := normalizeColor(color)
	for id, p := range state.Players {
		if id == exceptPlayerID {
			continue
		}
		if normalizeColor(p.Color) == want {
			return true
		}
	}
	return false
}

func normalizeColor(c string) string {
	out := make([]byte, 0, len(c))
	for i := 0; i < len(c); i++ {
		ch := c[i]
		if ch >= 'a' && ch <= 'z' {
			ch -= 32
		}
		out = append(out, ch)
	}
	return string(out)
}

func TakenColors(state *GameState) []string {
	out := make([]string, 0, len(state.Players))
	seen := map[string]bool{}
	for _, p := range state.Players {
		c := normalizeColor(p.Color)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, p.Color)
	}
	return out
}

func requireActive(state *GameState, playerID string) (*PlayerState, error) {
	if state.Phase != PhaseAction {
		return nil, fmt.Errorf("not in action phase")
	}
	p, ok := state.Players[playerID]
	if !ok {
		return nil, fmt.Errorf("player not found")
	}
	if p.Passed {
		return nil, fmt.Errorf("you have passed this generation")
	}
	if state.ActivePlayerID != playerID {
		return nil, fmt.Errorf("not your turn")
	}
	return p, nil
}

func CanAct(state *GameState, playerID string) error {
	_, err := requireActive(state, playerID)
	return err
}

func ClaimAction(state *GameState, playerID string) (advanced bool, err error) {
	if _, err := requireActive(state, playerID); err != nil {
		return false, err
	}
	state.ActionsThisTurn++
	if state.ActionsThisTurn >= MaxActionsPerTurn {
		AdvanceTurn(state)
		return true, nil
	}
	return false, nil
}

func EndTurn(state *GameState, playerID string) error {
	if _, err := requireActive(state, playerID); err != nil {
		return err
	}
	if state.ActionsThisTurn < 1 {
		return fmt.Errorf("take an action first, or Pass to leave this generation")
	}
	AdvanceTurn(state)
	return nil
}

func Pass(state *GameState, playerID string) (allPassed bool, err error) {
	p, err := requireActive(state, playerID)
	if err != nil {
		return false, err
	}
	if state.ActionsThisTurn > 0 {
		return false, fmt.Errorf("already took an action this turn — use End Turn instead")
	}
	p.Passed = true
	AdvanceTurn(state)
	return AllPassed(state), nil
}

// AbsentPlayer marks a disconnected player out for the current phase.
func AbsentPlayer(state *GameState, playerID string) (detail string, ok bool) {
	if state == nil {
		return "", false
	}
	p, exists := state.Players[playerID]
	if !exists || p == nil {
		return "", false
	}
	switch state.Phase {
	case PhaseResearch:
		if p.ResearchDone {
			return "", false
		}
		p.ResearchDone = true
		detail = fmt.Sprintf("%s disconnected → research skipped", p.Name)
		if MaybeFinishResearch(state) {
			detail += " → Action phase"
		}
		return detail, true
	case PhaseAction:
		if p.Passed {
			return "", false
		}
		p.Passed = true
		detail = fmt.Sprintf("%s disconnected → passed (out this generation)", p.Name)
		AdvanceTurn(state)
		if state.Phase == PhaseProductionWait {
			detail += " — all passed, confirm production"
		}
		return detail, true
	case PhaseProductionWait:
		if p.IsReady {
			return "", false
		}
		p.IsReady = true
		return fmt.Sprintf("%s disconnected → ready for production", p.Name), true
	default:
		return "", false
	}
}

func AllPassed(state *GameState) bool {
	if len(state.Players) == 0 {
		return false
	}
	for _, p := range state.Players {
		if !p.Passed {
			return false
		}
	}
	return true
}

func AdvanceTurn(state *GameState) {
	state.ActionsThisTurn = 0
	if len(state.TurnOrder) == 0 {
		state.ActivePlayerID = ""
		return
	}
	if AllPassed(state) {
		state.ActivePlayerID = ""
		state.Phase = PhaseProductionWait
		for _, p := range state.Players {
			p.IsReady = true
		}
		return
	}
	start := indexOf(state.TurnOrder, state.ActivePlayerID)
	if start < 0 {
		start = state.FirstPlayerIndex
	}
	n := len(state.TurnOrder)
	for i := 1; i <= n; i++ {
		idx := (start + i) % n
		id := state.TurnOrder[idx]
		if pl := state.Players[id]; pl != nil && !pl.Passed {
			state.ActivePlayerID = id
			return
		}
	}
	state.ActivePlayerID = ""
	state.Phase = PhaseProductionWait
}

func indexOf(ids []string, id string) int {
	for i, v := range ids {
		if v == id {
			return i
		}
	}
	return -1
}

func ExecuteProduction(state *GameState) {
	for _, p := range state.Players {
		producePlayer(p)
		p.IsReady = false
		p.Passed = false
		p.ResearchDone = false
	}
	state.Generation++
	if len(state.TurnOrder) > 0 {
		state.FirstPlayerIndex = (state.FirstPlayerIndex + 1) % len(state.TurnOrder)
	}
	state.ActivePlayerID = ""
	state.ActionsThisTurn = 0
	state.Phase = PhaseResearch
}

func producePlayer(p *PlayerState) {
	energy := p.Resources[Energy]
	heat := p.Resources[Heat]
	heat.Stock += energy.Stock
	energy.Stock = 0
	p.Resources[Energy] = energy
	p.Resources[Heat] = heat

	mc := p.Resources[MC]
	gain := p.TR + mc.Production
	if gain < 0 {
		gain = 0
	}
	mc.Stock += gain
	p.Resources[MC] = mc

	for _, r := range []ResourceType{Steel, Titanium, Plant, Energy, Heat} {
		rs := p.Resources[r]
		rs.Stock += rs.Production
		p.Resources[r] = rs
	}
}

func ToggleReady(state *GameState, playerID string) (shouldProduce bool, err error) {
	p, ok := state.Players[playerID]
	if !ok {
		return false, fmt.Errorf("player not found")
	}
	if state.Phase == PhaseProductionWait {
		return true, nil
	}
	p.IsReady = !p.IsReady
	allReady := true
	anyReady := false
	for _, pl := range state.Players {
		if pl.IsReady {
			anyReady = true
		} else {
			allReady = false
		}
	}
	if allReady && len(state.Players) > 0 {
		state.Phase = PhaseProductionWait
		return true, nil
	}
	if anyReady {
		state.Phase = PhaseProductionWait
	}
	return false, nil
}

func EndGame(state *GameState) {
	state.Phase = PhaseEnded
	state.ActivePlayerID = ""
	state.ActionsThisTurn = 0
}
