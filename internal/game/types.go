package game

import "time"

type ResourceType string

const (
	MC       ResourceType = "MC"
	Steel    ResourceType = "Steel"
	Titanium ResourceType = "Titanium"
	Plant    ResourceType = "Plant"
	Energy   ResourceType = "Energy"
	Heat     ResourceType = "Heat"
)

var AllResources = []ResourceType{MC, Steel, Titanium, Plant, Energy, Heat}

const (
	PhaseResearch       = "RESEARCH"
	PhaseAction         = "ACTION"
	PhaseProductionWait = "PRODUCTION_WAIT"
	PhaseEnded          = "ENDED"
)

// AllowNewJoin reports whether a brand-new player may join the room.
func AllowNewJoin(state *GameState) bool {
	return state != nil && state.Phase == PhaseResearch && state.Generation == 1
}

const (
	InitialTR           = 20
	InitialGeneration   = 1
	MCProductionMin     = -5
	AuditLogLimit       = 200
	ShortcutCost        = 8
	MaxActionsPerTurn   = 2
	CardBuyCost         = 3
	MaxCardsBuy         = 4
	MaxCardsBuyFirstGen = 10
	MaxSeats            = 5
)

// MaxCardsBuyForGeneration returns the research card-buy cap for the given
// generation. Generation 1 deals a larger starting hand (10 cards); every
// later generation deals 4.
func MaxCardsBuyForGeneration(generation int) int {
	if generation <= InitialGeneration {
		return MaxCardsBuyFirstGen
	}
	return MaxCardsBuy
}

// Common TM tags (manual counters).
var AllTags = []string{
	"Building", "Space", "Science", "Power", "Earth", "Jovian",
	"Plant", "Microbe", "Animal", "City", "Wild", "Event",
}

type ResourceState struct {
	Stock      int `json:"stock"`
	Production int `json:"production"`
}

// ScoreSheet is a lightweight end-game VP helper (semi-manual).
type ScoreSheet struct {
	GreeneryTiles    int `json:"greenery_tiles"`     // 1 VP each
	CityTiles        int `json:"city_tiles"`         // 1 VP each
	CityAdjGreenery  int `json:"city_adj_greenery"`  // adjacent greenery VP
	Milestone        int `json:"milestone"`
	Award            int `json:"award"`
	Cards            int `json:"cards"`
	Other            int `json:"other"`
}

func (s ScoreSheet) TileVP() int {
	return s.GreeneryTiles + s.CityTiles + s.CityAdjGreenery
}

func (s ScoreSheet) ManualVP() int {
	return s.Milestone + s.Award + s.Cards + s.Other + s.TileVP()
}

type PlayerState struct {
	ID           string                         `json:"id"`
	Name         string                         `json:"name"`
	Color        string                         `json:"color"`
	Seat         int                            `json:"seat"` // 1..MaxSeats, clockwise table order
	TR           int                            `json:"tr"`
	IsReady      bool                           `json:"is_ready"`
	Passed       bool                           `json:"passed"`
	ResearchDone bool                           `json:"research_done"`
	Resources    map[ResourceType]ResourceState `json:"resources"`
	Tags         map[string]int                 `json:"tags"`
	Score        ScoreSheet                     `json:"score"`
}

func (p *PlayerState) TotalVP() int {
	if p == nil {
		return 0
	}
	return p.TR + p.Score.ManualVP()
}

// Global parameter IDs
const (
	ParamTemperature = "temperature"
	ParamOxygen      = "oxygen"
	ParamOceans      = "oceans"
	ParamVenus       = "venus"
)

type GlobalParamDef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Unit        string `json:"unit"`
	Current     int    `json:"current"`
	Min         int    `json:"min"`
	Max         int    `json:"max"`
	Step        int    `json:"step"`
	Enabled     bool   `json:"enabled"`
	RequiredEnd bool   `json:"required_end"`
}

func DefaultGlobalParams() map[string]GlobalParamDef {
	return map[string]GlobalParamDef{
		ParamTemperature: {
			ID:          ParamTemperature,
			Name:        "Temperature",
			Unit:        "°C",
			Current:     -30,
			Min:         -30,
			Max:         8,
			Step:        2,
			Enabled:     true,
			RequiredEnd: true,
		},
		ParamOxygen: {
			ID:          ParamOxygen,
			Name:        "Oxygen",
			Unit:        "%",
			Current:     0,
			Min:         0,
			Max:         14,
			Step:        1,
			Enabled:     true,
			RequiredEnd: true,
		},
		ParamOceans: {
			ID:          ParamOceans,
			Name:        "Oceans",
			Unit:        "",
			Current:     0,
			Min:         0,
			Max:         9,
			Step:        1,
			Enabled:     true,
			RequiredEnd: true,
		},
		ParamVenus: {
			ID:          ParamVenus,
			Name:        "Venus",
			Unit:        "%",
			Current:     0,
			Min:         0,
			Max:         30,
			Step:        2,
			Enabled:     false,
			RequiredEnd: false,
		},
	}
}

type GameState struct {
	RoomID           string                    `json:"room_id"`
	Generation       int                       `json:"generation"`
	Phase            string                    `json:"phase"`
	Players          map[string]*PlayerState   `json:"players"`
	TurnOrder        []string                  `json:"turn_order"`
	ActivePlayerID   string                    `json:"active_player_id"`
	ActionsThisTurn  int                       `json:"actions_this_turn"`
	FirstPlayerIndex int                       `json:"first_player_index"`
	GlobalParams     map[string]GlobalParamDef `json:"global_params"`
}

type AuditLog struct {
	ID        string         `json:"id"`
	Timestamp time.Time      `json:"timestamp"`
	PlayerID  string         `json:"player_id"`
	Type      string         `json:"type"`
	Detail    string         `json:"detail"`
	Delta     map[string]int `json:"delta"`
}

func NewPlayer(id, name, color string, seat int) *PlayerState {
	resources := make(map[ResourceType]ResourceState, len(AllResources))
	for _, r := range AllResources {
		resources[r] = ResourceState{}
	}
	tags := make(map[string]int, len(AllTags))
	for _, t := range AllTags {
		tags[t] = 0
	}
	if seat < 1 {
		seat = 1
	}
	return &PlayerState{
		ID:        id,
		Name:      name,
		Color:     color,
		Seat:      seat,
		TR:        InitialTR,
		Resources: resources,
		Tags:      tags,
	}
}

func NewGameState(roomID string) GameState {
	return GameState{
		RoomID:       roomID,
		Generation:   InitialGeneration,
		Phase:        PhaseResearch,
		Players:      make(map[string]*PlayerState),
		TurnOrder:    nil,
		GlobalParams: DefaultGlobalParams(),
	}
}

func CloneGameState(src GameState) GameState {
	dst := GameState{
		RoomID:           src.RoomID,
		Generation:       src.Generation,
		Phase:            src.Phase,
		Players:          make(map[string]*PlayerState, len(src.Players)),
		TurnOrder:        append([]string(nil), src.TurnOrder...),
		ActivePlayerID:   src.ActivePlayerID,
		ActionsThisTurn:  src.ActionsThisTurn,
		FirstPlayerIndex: src.FirstPlayerIndex,
		GlobalParams:     make(map[string]GlobalParamDef, len(src.GlobalParams)),
	}
	for id, p := range src.Players {
		dst.Players[id] = ClonePlayer(p)
	}
	for k, v := range src.GlobalParams {
		dst.GlobalParams[k] = v
	}
	return dst
}

func ClonePlayer(p *PlayerState) *PlayerState {
	if p == nil {
		return nil
	}
	resources := make(map[ResourceType]ResourceState, len(p.Resources))
	for k, v := range p.Resources {
		resources[k] = v
	}
	tags := make(map[string]int, len(p.Tags))
	for k, v := range p.Tags {
		tags[k] = v
	}
	cp := *p
	cp.Resources = resources
	cp.Tags = tags
	cp.Score = p.Score
	return &cp
}

// EnsurePlayerDefaults fills missing maps after DB load / older clients.
func EnsurePlayerDefaults(p *PlayerState) {
	if p.Resources == nil {
		p.Resources = make(map[ResourceType]ResourceState)
	}
	for _, r := range AllResources {
		if _, ok := p.Resources[r]; !ok {
			p.Resources[r] = ResourceState{}
		}
	}
	if p.Tags == nil {
		p.Tags = make(map[string]int)
	}
	for _, t := range AllTags {
		if _, ok := p.Tags[t]; !ok {
			p.Tags[t] = 0
		}
	}
}

func EnsureStateDefaults(state *GameState) {
	if state.Players == nil {
		state.Players = make(map[string]*PlayerState)
	}
	for _, p := range state.Players {
		EnsurePlayerDefaults(p)
	}
	if state.Phase == "" {
		state.Phase = PhaseResearch
	}
	if state.Generation == 0 {
		state.Generation = InitialGeneration
	}
	if state.GlobalParams == nil {
		state.GlobalParams = DefaultGlobalParams()
	} else {
		defaults := DefaultGlobalParams()
		for k, v := range defaults {
			if _, ok := state.GlobalParams[k]; !ok {
				state.GlobalParams[k] = v
			}
		}
	}
}
