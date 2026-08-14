package room

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inmanuscript/TBG-RSE/internal/game"
	"github.com/inmanuscript/TBG-RSE/internal/store"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Manager struct {
	mu     sync.Mutex
	rooms  map[string]*Room
	store  *store.Store
	logger *log.Logger
}

type Room struct {
	mu           sync.Mutex
	Code         string
	HostPlayerID string
	State        game.GameState
	AuditLogs    []game.AuditLog
	tokens       map[string]string
	clients      map[*Client]struct{}
}

type Client struct {
	conn     *websocket.Conn
	room     *Room
	playerID string
	sendMu   sync.Mutex
}

type inboundMessage struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

type outboundMessage struct {
	Event   string `json:"event"`
	Payload any    `json:"payload"`
}

func NewManager(st *store.Store, logger *log.Logger) *Manager {
	if logger == nil {
		logger = log.Default()
	}
	m := &Manager{
		rooms:  make(map[string]*Room),
		store:  st,
		logger: logger,
	}
	if st != nil {
		persisted, err := st.LoadAllRooms()
		if err != nil {
			logger.Printf("load rooms: %v", err)
		} else {
			for _, pr := range persisted {
				game.EnsureStateDefaults(&pr.State)
				r := &Room{
					Code:         pr.Code,
					HostPlayerID: pr.HostPlayerID,
					State:        pr.State,
					AuditLogs:    pr.AuditLogs,
					tokens:       make(map[string]string),
					clients:      make(map[*Client]struct{}),
				}
				for id, meta := range pr.PlayersMeta {
					r.tokens[id] = meta.ReconnectToken
				}
				m.rooms[pr.Code] = r
			}
			logger.Printf("restored %d rooms from sqlite", len(persisted))
		}
	}
	return m
}

func (m *Manager) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		m.logger.Printf("ws accept: %v", err)
		return
	}
	client := &Client{conn: conn}
	defer func() {
		m.detachClient(client)
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()

	ctx := r.Context()
	for {
		var msg inboundMessage
		if err := wsjson.Read(ctx, conn, &msg); err != nil {
			return
		}
		if err := m.handleAction(client, msg); err != nil {
			_ = client.send(outboundMessage{
				Event:   "ERROR",
				Payload: map[string]string{"code": "ACTION_FAILED", "message": err.Error()},
			})
		}
	}
}

func (m *Manager) detachClient(c *Client) {
	if c.room == nil {
		return
	}
	r := c.room
	playerID := c.playerID
	r.mu.Lock()
	delete(r.clients, c)
	r.mu.Unlock()
	c.room = nil
	if playerID != "" {
		m.handlePlayerDisconnect(r, playerID)
	}
}

func (m *Manager) handlePlayerDisconnect(r *Room, playerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.isPlayerOnline(playerID) {
		return
	}
	detail, ok := game.AbsentPlayer(&r.State, playerID)
	if !ok {
		return
	}
	r.appendAuditLocked(playerID, "DISCONNECT", detail, nil)
	m.persistLocked(r)
	state := game.CloneGameState(r.State)
	r.broadcastAllLocked(outboundMessage{Event: "STATE_UPDATE", Payload: state})
	r.broadcastAllLocked(outboundMessage{
		Event: "NOTIFICATION",
		Payload: map[string]string{
			"player_name": "System",
			"message":     detail,
			"timestamp":   time.Now().Format("15:04:05"),
		},
	})
}

func (m *Manager) handleAction(client *Client, msg inboundMessage) error {
	switch msg.Action {
	case "CREATE_ROOM":
		var p struct {
			Name  string `json:"name"`
			Color string `json:"color"`
			Seat  int    `json:"seat"`
		}
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("invalid payload")
		}
		return m.createRoom(client, p.Name, p.Color, p.Seat)
	case "JOIN_ROOM":
		var p struct {
			RoomCode        string `json:"room_code"`
			Name            string `json:"name"`
			Color           string `json:"color"`
			Seat            int    `json:"seat"`
			ReclaimPlayerID string `json:"reclaim_player_id"`
		}
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("invalid payload")
		}
		return m.joinRoom(client, p.RoomCode, p.Name, p.Color, p.Seat, p.ReclaimPlayerID)
	case "RECONNECT":
		var p struct {
			RoomCode       string `json:"room_code"`
			PlayerID       string `json:"player_id"`
			ReconnectToken string `json:"reconnect_token"`
		}
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("invalid payload")
		}
		return m.reconnect(client, p.RoomCode, p.PlayerID, p.ReconnectToken)
	case "PEEK_ROOM":
		var p struct {
			RoomCode string `json:"room_code"`
		}
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("invalid payload")
		}
		return m.peekRoom(client, p.RoomCode)
	case "UPDATE_RESOURCE":
		var p struct {
			Target   string `json:"target"`
			Resource string `json:"resource"`
			Delta    int    `json:"delta"`
		}
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("invalid payload")
		}
		return m.withPlayer(client, func(r *Room, pl *game.PlayerState) error {
			detail, auditDelta, err := game.ApplyResourceDelta(pl, game.UpdateTarget(p.Target), game.ResourceType(p.Resource), p.Delta)
			if err != nil {
				return err
			}
			r.appendAuditLocked(client.playerID, "RESOURCE_CHANGE", detail, auditDelta)
			m.commitLocked(r, pl.Name, detail)
			return nil
		})
	case "BUY_CARDS":
		var p struct {
			Count int `json:"count"`
		}
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("invalid payload")
		}
		return m.withPlayer(client, func(r *Room, pl *game.PlayerState) error {
			if r.State.Phase != game.PhaseResearch {
				return fmt.Errorf("not in research phase")
			}
			detail, auditDelta, err := game.BuyCards(pl, p.Count, r.State.Generation)
			if err != nil {
				return err
			}
			r.appendAuditLocked(client.playerID, "RESEARCH", detail, auditDelta)
			finished := game.MaybeFinishResearch(&r.State)
			msg := detail
			if finished {
				msg = detail + " → Action phase"
			}
			m.commitLocked(r, pl.Name, msg)
			return nil
		})
	case "STANDARD_PROJECT":
		var p struct {
			Kind      string `json:"kind"`
			CardsSold int    `json:"cards_sold"`
		}
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("invalid payload")
		}
		return m.withPlayer(client, func(r *Room, pl *game.PlayerState) error {
			if err := game.CanAct(&r.State, client.playerID); err != nil {
				return err
			}
			detail, auditDelta, err := game.ApplyStandardProject(&r.State, client.playerID, p.Kind, p.CardsSold)
			if err != nil {
				return err
			}
			advanced, err := game.ClaimAction(&r.State, client.playerID)
			if err != nil {
				return err
			}
			r.appendAuditLocked(client.playerID, "STANDARD_PROJECT", detail, auditDelta)
			msg := detail
			if advanced {
				msg += " → next player"
			}
			if game.AllPassed(&r.State) || r.State.Phase == game.PhaseProductionWait {
				msg += " → production ready"
			}
			m.commitLocked(r, pl.Name, msg)
			return nil
		})
	case "SHORTCUT": // plant/heat conversion (consumes an action)
		var p struct {
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("invalid payload")
		}
		return m.withPlayer(client, func(r *Room, pl *game.PlayerState) error {
			if err := game.CanAct(&r.State, client.playerID); err != nil {
				return err
			}
			detail, auditDelta, err := game.ApplyConversion(&r.State, client.playerID, p.Kind)
			if err != nil {
				return err
			}
			advanced, err := game.ClaimAction(&r.State, client.playerID)
			if err != nil {
				return err
			}
			r.appendAuditLocked(client.playerID, "CONVERSION", detail, auditDelta)
			msg := detail
			if advanced {
				msg += " → next player"
			}
			m.commitLocked(r, pl.Name, msg)
			return nil
		})
	case "UPDATE_GLOBAL_PARAM":
		var p struct {
			ParamID    string `json:"param_id"`
			DeltaSteps int    `json:"delta_steps"`
			GrantTR    bool   `json:"grant_tr"`
		}
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("invalid payload")
		}
		return m.withPlayer(client, func(r *Room, pl *game.PlayerState) error {
			detail, auditDelta, err := game.UpdateGlobalParam(&r.State, p.ParamID, p.DeltaSteps, client.playerID, p.GrantTR)
			if err != nil {
				return err
			}
			r.appendAuditLocked(client.playerID, "GLOBAL_PARAM", detail, auditDelta)
			m.commitLocked(r, pl.Name, detail)
			return nil
		})
	case "CONFIGURE_GLOBAL_PARAMS":
		var p struct {
			Params map[string]game.GlobalParamDef `json:"params"`
		}
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("invalid payload")
		}
		return m.withPlayer(client, func(r *Room, pl *game.PlayerState) error {
			if client.playerID != r.HostPlayerID {
				return fmt.Errorf("only the host can configure global parameters")
			}
			if err := game.ConfigureGlobalParams(&r.State, p.Params); err != nil {
				return err
			}
			detail := fmt.Sprintf("%s updated global parameter settings", pl.Name)
			r.appendAuditLocked(client.playerID, "GLOBAL_PARAM_CONFIG", detail, nil)
			m.commitLocked(r, pl.Name, detail)
			return nil
		})
	case "CLAIM_ACTION":
		return m.withPlayer(client, func(r *Room, pl *game.PlayerState) error {
			if err := game.CanAct(&r.State, client.playerID); err != nil {
				return err
			}
			before := r.State.ActionsThisTurn
			advanced, err := game.ClaimAction(&r.State, client.playerID)
			if err != nil {
				return err
			}
			detail := fmt.Sprintf("%s took an action (%d/2)", pl.Name, before+1)
			if advanced {
				detail = fmt.Sprintf("%s took action 2/2 → next player", pl.Name)
			}
			r.appendAuditLocked(client.playerID, "ACTION", detail, nil)
			m.commitLocked(r, pl.Name, detail)
			return nil
		})
	case "END_TURN":
		return m.withPlayer(client, func(r *Room, pl *game.PlayerState) error {
			if err := game.EndTurn(&r.State, client.playerID); err != nil {
				return err
			}
			detail := fmt.Sprintf("%s ended turn", pl.Name)
			r.appendAuditLocked(client.playerID, "TURN", detail, nil)
			m.commitLocked(r, pl.Name, detail)
			return nil
		})
	case "PASS":
		return m.withPlayer(client, func(r *Room, pl *game.PlayerState) error {
			all, err := game.Pass(&r.State, client.playerID)
			if err != nil {
				return err
			}
			detail := fmt.Sprintf("%s passed (out this generation)", pl.Name)
			if all {
				detail += " — all passed, confirm production"
			}
			r.appendAuditLocked(client.playerID, "PASS", detail, nil)
			m.commitLocked(r, pl.Name, detail)
			return nil
		})
	case "READY_PRODUCTION":
		return m.withPlayer(client, func(r *Room, pl *game.PlayerState) error {
			shouldProduce, err := game.ToggleReady(&r.State, client.playerID)
			if err != nil {
				return err
			}
			readyMsg := "is ready for production"
			if !pl.IsReady {
				readyMsg = "cancelled production ready"
			}
			detail := fmt.Sprintf("%s %s", pl.Name, readyMsg)
			r.appendAuditLocked(client.playerID, "READY_TOGGLE", detail, nil)
			if shouldProduce {
				game.ExecuteProduction(&r.State)
				prod := fmt.Sprintf("Production executed → Generation %d (Research)", r.State.Generation)
				r.appendAuditLocked("", "PRODUCTION_EXECUTED", prod, nil)
				m.commitLocked(r, "System", prod)
				return nil
			}
			m.commitLocked(r, pl.Name, detail)
			return nil
		})
	case "UPDATE_TAG":
		var p struct {
			Tag   string `json:"tag"`
			Delta int    `json:"delta"`
		}
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("invalid payload")
		}
		return m.withPlayer(client, func(r *Room, pl *game.PlayerState) error {
			detail, err := game.ApplyTagDelta(pl, p.Tag, p.Delta)
			if err != nil {
				return err
			}
			r.appendAuditLocked(client.playerID, "TAG_CHANGE", detail, map[string]int{p.Tag: p.Delta})
			m.commitLocked(r, pl.Name, detail)
			return nil
		})
	case "UPDATE_SCORE":
		var p struct {
			Field string `json:"field"`
			Delta int    `json:"delta"`
		}
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			return fmt.Errorf("invalid payload")
		}
		return m.withPlayer(client, func(r *Room, pl *game.PlayerState) error {
			detail, err := game.ApplyScoreDelta(pl, p.Field, p.Delta)
			if err != nil {
				return err
			}
			r.appendAuditLocked(client.playerID, "SCORE_CHANGE", detail, map[string]int{p.Field: p.Delta})
			m.commitLocked(r, pl.Name, detail)
			return nil
		})
	case "END_GAME":
		return m.withPlayer(client, func(r *Room, pl *game.PlayerState) error {
			// Anyone may end the game — restricting this to the host left the
			// room with no way to end it if the host disconnected for good.
			game.EndGame(&r.State)
			detail := "Game ended — VP helper open"
			r.appendAuditLocked(client.playerID, "GAME_END", detail, nil)
			m.commitLocked(r, "System", detail)
			return nil
		})
	default:
		return fmt.Errorf("unknown action: %s", msg.Action)
	}
}

func (m *Manager) withPlayer(client *Client, fn func(*Room, *game.PlayerState) error) error {
	r := client.room
	if r == nil || client.playerID == "" {
		return fmt.Errorf("not in a room")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	pl := r.State.Players[client.playerID]
	if pl == nil {
		return fmt.Errorf("player not found")
	}
	return fn(r, pl)
}

func (m *Manager) commitLocked(r *Room, playerName, message string) {
	m.persistLocked(r)
	state := game.CloneGameState(r.State)
	r.broadcastAllLocked(outboundMessage{Event: "STATE_UPDATE", Payload: state})
	r.broadcastAllLocked(outboundMessage{
		Event: "NOTIFICATION",
		Payload: map[string]string{
			"player_name": playerName,
			"message":     message,
			"timestamp":   time.Now().Format("15:04:05"),
		},
	})
}

func (m *Manager) createRoom(client *Client, name, color string, seat int) error {
	if name == "" {
		return fmt.Errorf("name required")
	}
	if color == "" {
		color = "#FF5555"
	}
	if seat == 0 {
		seat = 1
	}
	tmp := game.NewGameState("tmp")
	if err := game.ValidateSeat(&tmp, seat, ""); err != nil {
		return err
	}
	m.mu.Lock()
	code := m.generateCodeLocked()
	playerID := uuid.NewString()
	token := randomToken()
	r := &Room{
		Code:         code,
		HostPlayerID: playerID,
		State:        game.NewGameState(code),
		tokens:       map[string]string{playerID: token},
		clients:      make(map[*Client]struct{}),
	}
	r.State.Players[playerID] = game.NewPlayer(playerID, name, color, seat)
	game.AddPlayerToTurnOrder(&r.State, playerID)
	m.rooms[code] = r
	m.mu.Unlock()

	m.attach(client, r, playerID)
	m.persist(r)
	return client.sendJoined(r, playerID, token)
}

func (r *Room) isPlayerOnline(playerID string) bool {
	for c := range r.clients {
		if c.playerID == playerID {
			return true
		}
	}
	return false
}

func (m *Manager) joinRoom(client *Client, code, name, color string, seat int, reclaimPlayerID string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name required")
	}
	m.mu.Lock()
	r := m.rooms[normalizeCode(code)]
	m.mu.Unlock()
	if r == nil {
		return fmt.Errorf("room not found")
	}
	if reclaimPlayerID != "" {
		return m.reclaimSeat(client, r, reclaimPlayerID, name)
	}
	return m.joinRoomNew(client, r, name, color, seat)
}

func (m *Manager) joinRoomNew(client *Client, r *Room, name, color string, seat int) error {
	if color == "" {
		color = "#55AAFF"
	}

	r.mu.Lock()
	if r.State.Phase == game.PhaseEnded {
		r.mu.Unlock()
		return fmt.Errorf("game already ended")
	}
	if !game.AllowNewJoin(&r.State) {
		r.mu.Unlock()
		return fmt.Errorf("game already started; reclaim an offline seat")
	}
	if game.ColorInUse(&r.State, color, "") {
		r.mu.Unlock()
		return fmt.Errorf("color already taken")
	}
	if seat == 0 {
		taken := map[int]bool{}
		for _, p := range r.State.Players {
			taken[p.Seat] = true
		}
		for s := 1; s <= game.MaxSeats; s++ {
			if !taken[s] {
				seat = s
				break
			}
		}
	}
	if err := game.ValidateSeat(&r.State, seat, ""); err != nil {
		r.mu.Unlock()
		return err
	}
	playerID := uuid.NewString()
	token := randomToken()
	r.State.Players[playerID] = game.NewPlayer(playerID, name, color, seat)
	game.AddPlayerToTurnOrder(&r.State, playerID)
	r.tokens[playerID] = token
	r.mu.Unlock()

	m.attach(client, r, playerID)
	m.persist(r)
	if err := client.sendJoined(r, playerID, token); err != nil {
		return err
	}
	r.broadcastStateAndNotify(name, fmt.Sprintf("%s joined (seat %d)", name, seat), playerID)
	return nil
}

func (m *Manager) reclaimSeat(client *Client, r *Room, reclaimPlayerID, name string) error {
	r.mu.Lock()
	if r.State.Phase == game.PhaseEnded {
		r.mu.Unlock()
		return fmt.Errorf("game already ended")
	}
	pl, ok := r.State.Players[reclaimPlayerID]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("player not found")
	}
	if r.isPlayerOnline(reclaimPlayerID) {
		r.mu.Unlock()
		return fmt.Errorf("player is already online")
	}
	if strings.TrimSpace(name) != strings.TrimSpace(pl.Name) {
		r.mu.Unlock()
		return fmt.Errorf("name does not match seat owner")
	}
	token := randomToken()
	r.tokens[reclaimPlayerID] = token
	seat := pl.Seat
	displayName := pl.Name
	r.mu.Unlock()

	m.attach(client, r, reclaimPlayerID)
	m.persist(r)
	if err := client.sendJoined(r, reclaimPlayerID, token); err != nil {
		return err
	}
	r.broadcastStateAndNotify(displayName, fmt.Sprintf("%s reconnected (seat %d)", displayName, seat), reclaimPlayerID)
	return nil
}

func (m *Manager) peekRoom(client *Client, code string) error {
	m.mu.Lock()
	r := m.rooms[normalizeCode(code)]
	m.mu.Unlock()
	if r == nil {
		return client.send(outboundMessage{
			Event: "ROOM_INFO",
			Payload: map[string]any{
				"room_code":    normalizeCode(code),
				"found":        false,
				"taken_colors": []string{},
				"taken_seats":  []int{},
				"player_count": 0,
			},
		})
	}
	r.mu.Lock()
	takenColors := game.TakenColors(&r.State)
	takenSeats := game.TakenSeats(&r.State)
	count := len(r.State.Players)
	phase := r.State.Phase
	generation := r.State.Generation
	allowNewJoin := game.AllowNewJoin(&r.State)
	players := make([]map[string]any, 0, count)
	for _, p := range r.State.Players {
		players = append(players, map[string]any{
			"id":     p.ID,
			"name":   p.Name,
			"seat":   p.Seat,
			"color":  p.Color,
			"online": r.isPlayerOnline(p.ID),
		})
	}
	r.mu.Unlock()
	sort.Slice(players, func(i, j int) bool {
		return players[i]["seat"].(int) < players[j]["seat"].(int)
	})
	return client.send(outboundMessage{
		Event: "ROOM_INFO",
		Payload: map[string]any{
			"room_code":      normalizeCode(code),
			"found":          true,
			"taken_colors":   takenColors,
			"taken_seats":    takenSeats,
			"player_count":   count,
			"phase":          phase,
			"generation":     generation,
			"allow_new_join": allowNewJoin,
			"players":        players,
		},
	})
}

func (m *Manager) reconnect(client *Client, code, playerID, token string) error {
	m.mu.Lock()
	r := m.rooms[normalizeCode(code)]
	m.mu.Unlock()
	if r == nil {
		return fmt.Errorf("room not found")
	}
	r.mu.Lock()
	expected, ok := r.tokens[playerID]
	if !ok || expected == "" || expected != token {
		r.mu.Unlock()
		return fmt.Errorf("invalid reconnect token")
	}
	if _, ok := r.State.Players[playerID]; !ok {
		r.mu.Unlock()
		return fmt.Errorf("player not found")
	}
	r.mu.Unlock()

	m.attach(client, r, playerID)
	return client.sendJoined(r, playerID, token)
}

func (m *Manager) attach(client *Client, r *Room, playerID string) {
	m.detachClient(client)
	var stale []*Client
	r.mu.Lock()
	for c := range r.clients {
		if c.playerID == playerID && c != client {
			stale = append(stale, c)
			delete(r.clients, c)
			c.room = nil
		}
	}
	client.room = r
	client.playerID = playerID
	r.clients[client] = struct{}{}
	r.mu.Unlock()
	for _, c := range stale {
		_ = c.conn.Close(websocket.StatusNormalClosure, "replaced")
	}
}

func (m *Manager) persist(r *Room) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m.persistLocked(r)
}

func (m *Manager) persistLocked(r *Room) {
	if m.store == nil {
		return
	}
	tokens := make(map[string]string, len(r.tokens))
	for k, v := range r.tokens {
		tokens[k] = v
	}
	logs := append([]game.AuditLog(nil), r.AuditLogs...)
	state := game.CloneGameState(r.State)
	if err := m.store.SaveRoom(r.Code, r.HostPlayerID, state, tokens, logs); err != nil {
		m.logger.Printf("persist room %s: %v", r.Code, err)
	}
}

func (r *Room) appendAuditLocked(playerID, typ, detail string, delta map[string]int) {
	if delta == nil {
		delta = map[string]int{}
	}
	r.AuditLogs = append(r.AuditLogs, game.AuditLog{
		ID:        uuid.NewString(),
		Timestamp: time.Now().UTC(),
		PlayerID:  playerID,
		Type:      typ,
		Detail:    detail,
		Delta:     delta,
	})
	if len(r.AuditLogs) > game.AuditLogLimit {
		r.AuditLogs = r.AuditLogs[len(r.AuditLogs)-game.AuditLogLimit:]
	}
}

func (r *Room) broadcastStateAndNotify(playerName, message, playerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state := game.CloneGameState(r.State)
	r.broadcastAllLocked(outboundMessage{Event: "STATE_UPDATE", Payload: state})
	r.broadcastAllLocked(outboundMessage{
		Event: "NOTIFICATION",
		Payload: map[string]string{
			"player_name": playerName,
			"message":     message,
			"timestamp":   time.Now().Format("15:04:05"),
		},
	})
	_ = playerID
}

func (r *Room) broadcastAllLocked(msg outboundMessage) {
	for c := range r.clients {
		_ = c.send(msg)
	}
}

func (c *Client) send(msg outboundMessage) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return wsjson.Write(ctx, c.conn, msg)
}

func (c *Client) sendJoined(r *Room, playerID, token string) error {
	r.mu.Lock()
	state := game.CloneGameState(r.State)
	code := r.Code
	host := r.HostPlayerID
	r.mu.Unlock()
	return c.send(outboundMessage{
		Event: "ROOM_JOINED",
		Payload: map[string]any{
			"room_code":       code,
			"player_id":       playerID,
			"reconnect_token": token,
			"host_player_id":  host,
			"state":           state,
		},
	})
}

func (m *Manager) generateCodeLocked() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for {
		b := make([]byte, 6)
		_, _ = rand.Read(b)
		code := make([]byte, 6)
		for i := range b {
			code[i] = alphabet[int(b[i])%len(alphabet)]
		}
		s := string(code)
		if _, exists := m.rooms[s]; !exists {
			return s
		}
	}
}

func normalizeCode(code string) string {
	out := make([]byte, 0, len(code))
	for i := 0; i < len(code); i++ {
		c := code[i]
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		if c == ' ' || c == '-' {
			continue
		}
		out = append(out, c)
	}
	return string(out)
}

func randomToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
