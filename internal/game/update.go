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
func ApplyConversion(p *PlayerState, kind string) (detail string, auditDelta map[string]int, err error) {
	if p == nil {
		return "", nil, fmt.Errorf("player is nil")
	}
	switch kind {
	case "greenery":
		rs := p.Resources[Plant]
		if rs.Stock < ShortcutCost {
			return "", nil, fmt.Errorf("need %d plants for greenery (have %d)", ShortcutCost, rs.Stock)
		}
		rs.Stock -= ShortcutCost
		p.Resources[Plant] = rs
		p.Score.GreeneryTiles++
		return fmt.Sprintf("%s spent %d Plants for Greenery", p.Name, ShortcutCost),
			map[string]int{"Plant_stock": -ShortcutCost, "greenery_tiles": 1}, nil
	case "temperature":
		rs := p.Resources[Heat]
		if rs.Stock < ShortcutCost {
			return "", nil, fmt.Errorf("need %d heat for temperature (have %d)", ShortcutCost, rs.Stock)
		}
		rs.Stock -= ShortcutCost
		p.Resources[Heat] = rs
		p.TR++
		return fmt.Sprintf("%s spent %d Heat for Temperature (+1 TR)", p.Name, ShortcutCost),
			map[string]int{"Heat_stock": -ShortcutCost, "tr": 1}, nil
	default:
		return "", nil, fmt.Errorf("unknown conversion: %s", kind)
	}
}

// ApplyStandardProject applies base-game standard projects.
// sell_patents uses cardsSold (MC gained = cardsSold).
func ApplyStandardProject(p *PlayerState, kind string, cardsSold int) (detail string, auditDelta map[string]int, err error) {
	if p == nil {
		return "", nil, fmt.Errorf("player is nil")
	}
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
		p.TR++
		return fmt.Sprintf("%s Standard Project: Asteroid (−14 MC, +1 TR)", p.Name),
			map[string]int{"MC_stock": -14, "tr": 1}, nil

	case "aquifer":
		if err := spendMC(p, 18); err != nil {
			return "", nil, err
		}
		p.TR++
		return fmt.Sprintf("%s Standard Project: Aquifer (−18 MC, +1 TR)", p.Name),
			map[string]int{"MC_stock": -18, "tr": 1}, nil

	case "greenery_project":
		if err := spendMC(p, 23); err != nil {
			return "", nil, err
		}
		p.Score.GreeneryTiles++
		return fmt.Sprintf("%s Standard Project: Greenery (−23 MC)", p.Name),
			map[string]int{"MC_stock": -23, "greenery_tiles": 1}, nil

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

// Deprecated name kept for older call sites.
func ApplyShortcut(p *PlayerState, kind string) (string, map[string]int, error) {
	return ApplyConversion(p, kind)
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
