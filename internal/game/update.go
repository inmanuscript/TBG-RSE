package game

import "fmt"

type UpdateTarget string

const (
	TargetStock      UpdateTarget = "stock"
	TargetProduction UpdateTarget = "production"
	TargetTR         UpdateTarget = "tr"
)

func ValidateResourceType(r ResourceType) error {
	switch r {
	case MC, Steel, Titanium, Plant, Energy, Heat:
		return nil
	default:
		return fmt.Errorf("unknown resource: %s", r)
	}
}

func ApplyResourceDelta(p *PlayerState, target UpdateTarget, resource ResourceType, delta int) (detail string, auditDelta map[string]int, err error) {
	if p == nil {
		return "", nil, fmt.Errorf("player is nil")
	}
	if delta == 0 {
		return "", nil, fmt.Errorf("delta must be non-zero")
	}

	switch target {
	case TargetTR:
		next := p.TR + delta
		if next < 0 {
			return "", nil, fmt.Errorf("TR cannot be negative")
		}
		p.TR = next
		return fmt.Sprintf("%s TR %+d → %d", p.Name, delta, p.TR), map[string]int{"tr": delta}, nil

	case TargetStock:
		if err := ValidateResourceType(resource); err != nil {
			return "", nil, err
		}
		rs := p.Resources[resource]
		next := rs.Stock + delta
		if next < 0 {
			return "", nil, fmt.Errorf("%s stock cannot be negative", resource)
		}
		rs.Stock = next
		p.Resources[resource] = rs
		return fmt.Sprintf("%s %s stock %+d → %d", p.Name, resource, delta, rs.Stock),
			map[string]int{string(resource) + "_stock": delta}, nil

	case TargetProduction:
		if err := ValidateResourceType(resource); err != nil {
			return "", nil, err
		}
		rs := p.Resources[resource]
		next := rs.Production + delta
		min := 0
		if resource == MC {
			min = MCProductionMin
		}
		if next < min {
			return "", nil, fmt.Errorf("%s production cannot be below %d", resource, min)
		}
		rs.Production = next
		p.Resources[resource] = rs
		return fmt.Sprintf("%s %s production %+d → %d", p.Name, resource, delta, rs.Production),
			map[string]int{string(resource) + "_production": delta}, nil

	default:
		return "", nil, fmt.Errorf("unknown target: %s", target)
	}
}

func spendMC(p *PlayerState, amount int) error {
	mc := p.Resources[MC]
	if mc.Stock < amount {
		return fmt.Errorf("need %d MC (have %d)", amount, mc.Stock)
	}
	mc.Stock -= amount
	p.Resources[MC] = mc
	return nil
}

func addProduction(p *PlayerState, r ResourceType, delta int) {
	rs := p.Resources[r]
	rs.Production += delta
	p.Resources[r] = rs
}

// ApplyConversion spends plants/heat for board conversions (counts as a game action).
func ApplyConversion(state *GameState, playerID string, kind string) (detail string, auditDelta map[string]int, err error) {
	if state == nil {
		return "", nil, fmt.Errorf("state is nil")
	}
	p, ok := state.Players[playerID]
	if !ok || p == nil {
		return "", nil, fmt.Errorf("player not found")
	}
	auditDelta = make(map[string]int)

	switch kind {
	case "greenery":
		rs := p.Resources[Plant]
		if rs.Stock < ShortcutCost {
			return "", nil, fmt.Errorf("need %d plants for greenery (have %d)", ShortcutCost, rs.Stock)
		}
		rs.Stock -= ShortcutCost
		p.Resources[Plant] = rs
		p.Score.GreeneryTiles++
		auditDelta["Plant_stock"] = -ShortcutCost
		auditDelta["greenery_tiles"] = 1

		if state.GlobalParams != nil {
			if param, exists := state.GlobalParams[ParamOxygen]; exists && param.Enabled {
				if param.Current < param.Max {
					param.Current += param.Step
					if param.Current > param.Max {
						param.Current = param.Max
					}
					state.GlobalParams[ParamOxygen] = param
					p.TR++
					auditDelta["tr"] = 1
					auditDelta["oxygen"] = param.Step
					return fmt.Sprintf("%s spent %d Plants for Greenery (+1 Greenery, Oxygen %d%s, +1 TR)", p.Name, ShortcutCost, param.Current, param.Unit), auditDelta, nil
				}
				return fmt.Sprintf("%s spent %d Plants for Greenery (+1 Greenery, Oxygen already MAX %d%s, no TR)", p.Name, ShortcutCost, param.Current, param.Unit), auditDelta, nil
			}
		}
		return fmt.Sprintf("%s spent %d Plants for Greenery (+1 Greenery)", p.Name, ShortcutCost), auditDelta, nil

	case "temperature":
		rs := p.Resources[Heat]
		if rs.Stock < ShortcutCost {
			return "", nil, fmt.Errorf("need %d heat for temperature (have %d)", ShortcutCost, rs.Stock)
		}
		rs.Stock -= ShortcutCost
		p.Resources[Heat] = rs
		auditDelta["Heat_stock"] = -ShortcutCost

		if state.GlobalParams != nil {
			if param, exists := state.GlobalParams[ParamTemperature]; exists && param.Enabled {
				if param.Current < param.Max {
					param.Current += param.Step
					if param.Current > param.Max {
						param.Current = param.Max
					}
					state.GlobalParams[ParamTemperature] = param
					p.TR++
					auditDelta["tr"] = 1
					auditDelta["temperature"] = param.Step
					return fmt.Sprintf("%s spent %d Heat for Temperature (%d%s, +1 TR)", p.Name, ShortcutCost, param.Current, param.Unit), auditDelta, nil
				}
				return fmt.Sprintf("%s spent %d Heat for Temperature (Temp already MAX %d%s, no TR)", p.Name, ShortcutCost, param.Current, param.Unit), auditDelta, nil
			}
		}
		p.TR++
		auditDelta["tr"] = 1
		return fmt.Sprintf("%s spent %d Heat for Temperature (+1 TR)", p.Name, ShortcutCost), auditDelta, nil

	default:
		return "", nil, fmt.Errorf("unknown conversion: %s", kind)
	}
}

// ApplyStandardProject applies base-game standard projects.
// sell_patents uses cardsSold (MC gained = cardsSold).
func ApplyStandardProject(state *GameState, playerID string, kind string, cardsSold int) (detail string, auditDelta map[string]int, err error) {
	if state == nil {
		return "", nil, fmt.Errorf("state is nil")
	}
	p, ok := state.Players[playerID]
	if !ok || p == nil {
		return "", nil, fmt.Errorf("player not found")
	}
	auditDelta = make(map[string]int)

	switch kind {
	case "sell_patents":
		if cardsSold < 1 {
			return "", nil, fmt.Errorf("sell at least 1 card")
		}
		mc := p.Resources[MC]
		mc.Stock += cardsSold
		p.Resources[MC] = mc
		return fmt.Sprintf("%s sold %d patent(s) (+%d MC)", p.Name, cardsSold, cardsSold),
			map[string]int{"MC_stock": cardsSold}, nil

	case "power_plant":
		if err := spendMC(p, 11); err != nil {
			return "", nil, err
		}
		addProduction(p, Energy, 1)
		return fmt.Sprintf("%s Standard Project: Power Plant (−11 MC, +1 Energy prod)", p.Name),
			map[string]int{"MC_stock": -11, "Energy_production": 1}, nil

	case "asteroid":
		if err := spendMC(p, 14); err != nil {
			return "", nil, err
		}
		auditDelta["MC_stock"] = -14
		if state.GlobalParams != nil {
			if param, exists := state.GlobalParams[ParamTemperature]; exists && param.Enabled {
				if param.Current < param.Max {
					param.Current += param.Step
					if param.Current > param.Max {
						param.Current = param.Max
					}
					state.GlobalParams[ParamTemperature] = param
					p.TR++
					auditDelta["tr"] = 1
					auditDelta["temperature"] = param.Step
					return fmt.Sprintf("%s Standard Project: Asteroid (−14 MC, Temp %d%s, +1 TR)", p.Name, param.Current, param.Unit), auditDelta, nil
				}
				return fmt.Sprintf("%s Standard Project: Asteroid (−14 MC, Temp already MAX %d%s, no TR)", p.Name, param.Current, param.Unit), auditDelta, nil
			}
		}
		p.TR++
		auditDelta["tr"] = 1
		return fmt.Sprintf("%s Standard Project: Asteroid (−14 MC, +1 TR)", p.Name), auditDelta, nil

	case "aquifer":
		if err := spendMC(p, 18); err != nil {
			return "", nil, err
		}
		auditDelta["MC_stock"] = -18
		if state.GlobalParams != nil {
			if param, exists := state.GlobalParams[ParamOceans]; exists && param.Enabled {
				if param.Current < param.Max {
					param.Current += param.Step
					if param.Current > param.Max {
						param.Current = param.Max
					}
					state.GlobalParams[ParamOceans] = param
					p.TR++
					auditDelta["tr"] = 1
					auditDelta["oceans"] = param.Step
					return fmt.Sprintf("%s Standard Project: Aquifer (−18 MC, Oceans %d/%d, +1 TR)", p.Name, param.Current, param.Max), auditDelta, nil
				}
				return fmt.Sprintf("%s Standard Project: Aquifer (−18 MC, Oceans already MAX %d, no TR)", p.Name, param.Current), auditDelta, nil
			}
		}
		p.TR++
		auditDelta["tr"] = 1
		return fmt.Sprintf("%s Standard Project: Aquifer (−18 MC, +1 TR)", p.Name), auditDelta, nil

	case "greenery_project":
		if err := spendMC(p, 23); err != nil {
			return "", nil, err
		}
		p.Score.GreeneryTiles++
		auditDelta["MC_stock"] = -23
		auditDelta["greenery_tiles"] = 1
		if state.GlobalParams != nil {
			if param, exists := state.GlobalParams[ParamOxygen]; exists && param.Enabled {
				if param.Current < param.Max {
					param.Current += param.Step
					if param.Current > param.Max {
						param.Current = param.Max
					}
					state.GlobalParams[ParamOxygen] = param
					p.TR++
					auditDelta["tr"] = 1
					auditDelta["oxygen"] = param.Step
					return fmt.Sprintf("%s Standard Project: Greenery (−23 MC, +1 Greenery, Oxygen %d%s, +1 TR)", p.Name, param.Current, param.Unit), auditDelta, nil
				}
				return fmt.Sprintf("%s Standard Project: Greenery (−23 MC, +1 Greenery, Oxygen already MAX %d%s, no TR)", p.Name, param.Current, param.Unit), auditDelta, nil
			}
		}
		return fmt.Sprintf("%s Standard Project: Greenery (−23 MC, +1 Greenery)", p.Name), auditDelta, nil

	case "city":
		if err := spendMC(p, 25); err != nil {
			return "", nil, err
		}
		p.Score.CityTiles++
		return fmt.Sprintf("%s Standard Project: City (−25 MC)", p.Name),
			map[string]int{"MC_stock": -25, "city_tiles": 1}, nil

	default:
		return "", nil, fmt.Errorf("unknown standard project: %s", kind)
	}
}

// UpdateGlobalParam directly updates a global parameter with optional TR grant.
func UpdateGlobalParam(state *GameState, paramID string, deltaSteps int, playerID string, grantTR bool) (detail string, auditDelta map[string]int, err error) {
	if state == nil {
		return "", nil, fmt.Errorf("state is nil")
	}
	if deltaSteps == 0 {
		return "", nil, fmt.Errorf("deltaSteps must be non-zero")
	}
	param, exists := state.GlobalParams[paramID]
	if !exists || !param.Enabled {
		return "", nil, fmt.Errorf("global parameter %q is not active", paramID)
	}
	p := state.Players[playerID]

	if deltaSteps > 0 && param.Current >= param.Max {
		return "", nil, fmt.Errorf("%s is already at maximum (%d%s)", param.Name, param.Max, param.Unit)
	}
	if deltaSteps < 0 && param.Current <= param.Min {
		return "", nil, fmt.Errorf("%s is already at minimum (%d%s)", param.Name, param.Min, param.Unit)
	}

	target := param.Current + (deltaSteps * param.Step)
	if target > param.Max {
		target = param.Max
	}
	if target < param.Min {
		target = param.Min
	}
	actualStepDelta := (target - param.Current) / param.Step
	param.Current = target
	state.GlobalParams[paramID] = param

	auditDelta = map[string]int{paramID: target}
	trGained := 0
	if grantTR && p != nil {
		if actualStepDelta > 0 {
			p.TR += actualStepDelta
			trGained = actualStepDelta
			auditDelta["tr"] = trGained
		} else if actualStepDelta < 0 {
			if p.TR+actualStepDelta >= 0 {
				p.TR += actualStepDelta
				trGained = actualStepDelta
				auditDelta["tr"] = trGained
			}
		}
	}

	actorName := "System"
	if p != nil {
		actorName = p.Name
	}
	stepSign := "+"
	if deltaSteps < 0 {
		stepSign = ""
	}
	trMsg := ""
	if trGained != 0 {
		trMsg = fmt.Sprintf(" (%+d TR)", trGained)
	}
	detail = fmt.Sprintf("%s updated %s %s%d%s → %d%s%s", actorName, param.Name, stepSign, deltaSteps*param.Step, param.Unit, target, param.Unit, trMsg)
	return detail, auditDelta, nil
}

// ConfigureGlobalParams validates and applies host-configured parameter definitions.
func ConfigureGlobalParams(state *GameState, params map[string]GlobalParamDef) error {
	if state == nil {
		return fmt.Errorf("state is nil")
	}
	for id, p := range params {
		if p.Step <= 0 {
			return fmt.Errorf("step for %s must be > 0", id)
		}
		if p.Min > p.Max {
			return fmt.Errorf("min cannot be greater than max for %s", id)
		}
		if p.Current < p.Min || p.Current > p.Max {
			return fmt.Errorf("current value for %s must be between min and max", id)
		}
	}
	if state.GlobalParams == nil {
		state.GlobalParams = make(map[string]GlobalParamDef)
	}
	for id, p := range params {
		p.ID = id
		state.GlobalParams[id] = p
	}
	return nil
}

// AreGlobalParamsMaxed returns whether all enabled params required for game end are at max.
func AreGlobalParamsMaxed(state *GameState) bool {
	if state == nil || len(state.GlobalParams) == 0 {
		return false
	}
	hasRequired := false
	for _, p := range state.GlobalParams {
		if p.Enabled && p.RequiredEnd {
			hasRequired = true
			if p.Current < p.Max {
				return false
			}
		}
	}
	return hasRequired
}

// Deprecated name kept for older call sites.
func ApplyShortcut(p *PlayerState, kind string) (string, map[string]int, error) {
	if p == nil {
		return "", nil, fmt.Errorf("player is nil")
	}
	st := &GameState{Players: map[string]*PlayerState{p.ID: p}, GlobalParams: DefaultGlobalParams()}
	return ApplyConversion(st, p.ID, kind)
}

func ApplyTagDelta(p *PlayerState, tag string, delta int) (detail string, err error) {
	if p == nil {
		return "", fmt.Errorf("player is nil")
	}
	if delta == 0 {
		return "", fmt.Errorf("delta must be non-zero")
	}
	valid := false
	for _, t := range AllTags {
		if t == tag {
			valid = true
			break
		}
	}
	if !valid {
		return "", fmt.Errorf("unknown tag: %s", tag)
	}
	EnsurePlayerDefaults(p)
	next := p.Tags[tag] + delta
	if next < 0 {
		return "", fmt.Errorf("tag count cannot be negative")
	}
	p.Tags[tag] = next
	return fmt.Sprintf("%s tag %s %+d → %d", p.Name, tag, delta, next), nil
}

func ApplyScoreDelta(p *PlayerState, field string, delta int) (detail string, err error) {
	if p == nil {
		return "", fmt.Errorf("player is nil")
	}
	if delta == 0 {
		return "", fmt.Errorf("delta must be non-zero")
	}
	var ptr *int
	switch field {
	case "greenery_tiles":
		ptr = &p.Score.GreeneryTiles
	case "city_tiles":
		ptr = &p.Score.CityTiles
	case "city_adj_greenery":
		ptr = &p.Score.CityAdjGreenery
	case "milestone":
		ptr = &p.Score.Milestone
	case "award":
		ptr = &p.Score.Award
	case "cards":
		ptr = &p.Score.Cards
	case "other":
		ptr = &p.Score.Other
	default:
		return "", fmt.Errorf("unknown score field: %s", field)
	}
	next := *ptr + delta
	if next < 0 {
		return "", fmt.Errorf("%s cannot be negative", field)
	}
	*ptr = next
	return fmt.Sprintf("%s score.%s %+d → %d (total VP %d)", p.Name, field, delta, next, p.TotalVP()), nil
}
