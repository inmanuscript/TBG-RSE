package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/inmanuscript/TBG-RSE/internal/game"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type PersistedRoom struct {
	Code         string
	HostPlayerID string
	State        game.GameState
	PlayersMeta  map[string]PlayerMeta
	AuditLogs    []game.AuditLog
}

type PlayerMeta struct {
	ReconnectToken string
}

type roomExtras struct {
	TurnOrder        []string                       `json:"turn_order"`
	ActivePlayerID   string                         `json:"active_player_id"`
	ActionsThisTurn  int                            `json:"actions_this_turn"`
	FirstPlayerIndex int                            `json:"first_player_index"`
	GlobalParams     map[string]game.GlobalParamDef `json:"global_params,omitempty"`
}

type playerExtras struct {
	Passed       bool            `json:"passed"`
	ResearchDone bool            `json:"research_done"`
	Seat         int             `json:"seat"`
	Tags         map[string]int  `json:"tags"`
	Score        game.ScoreSheet `json:"score"`
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS rooms (
  code TEXT PRIMARY KEY,
  host_player_id TEXT NOT NULL,
  generation INTEGER NOT NULL,
  phase TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  extras_json TEXT NOT NULL DEFAULT '{}'
);
CREATE TABLE IF NOT EXISTS players (
  room_code TEXT NOT NULL,
  player_id TEXT NOT NULL,
  name TEXT NOT NULL,
  color TEXT NOT NULL,
  tr INTEGER NOT NULL,
  is_ready INTEGER NOT NULL,
  resources_json TEXT NOT NULL,
  reconnect_token TEXT NOT NULL,
  extras_json TEXT NOT NULL DEFAULT '{}',
  PRIMARY KEY (room_code, player_id),
  FOREIGN KEY (room_code) REFERENCES rooms(code)
);
CREATE TABLE IF NOT EXISTS audit_logs (
  id TEXT PRIMARY KEY,
  room_code TEXT NOT NULL,
  ts TEXT NOT NULL,
  player_id TEXT NOT NULL,
  type TEXT NOT NULL,
  detail TEXT NOT NULL,
  delta_json TEXT NOT NULL,
  FOREIGN KEY (room_code) REFERENCES rooms(code)
);
CREATE INDEX IF NOT EXISTS idx_audit_room_ts ON audit_logs(room_code, ts);
`)
	if err != nil {
		return err
	}
	// Best-effort columns for DBs created before extras_json existed.
	_, _ = s.db.Exec(`ALTER TABLE rooms ADD COLUMN extras_json TEXT NOT NULL DEFAULT '{}'`)
	_, _ = s.db.Exec(`ALTER TABLE players ADD COLUMN extras_json TEXT NOT NULL DEFAULT '{}'`)
	return nil
}

func (s *Store) SaveRoom(code, hostPlayerID string, state game.GameState, tokens map[string]string, logs []game.AuditLog) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	extras, err := json.Marshal(roomExtras{
		TurnOrder:        state.TurnOrder,
		ActivePlayerID:   state.ActivePlayerID,
		ActionsThisTurn:  state.ActionsThisTurn,
		FirstPlayerIndex: state.FirstPlayerIndex,
		GlobalParams:     state.GlobalParams,
	})
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = tx.Exec(`
INSERT INTO rooms(code, host_player_id, generation, phase, updated_at, extras_json)
VALUES(?,?,?,?,?,?)
ON CONFLICT(code) DO UPDATE SET
  host_player_id=excluded.host_player_id,
  generation=excluded.generation,
  phase=excluded.phase,
  updated_at=excluded.updated_at,
  extras_json=excluded.extras_json
`, code, hostPlayerID, state.Generation, state.Phase, now, string(extras))
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM players WHERE room_code=?`, code)
	if err != nil {
		return err
	}
	for _, p := range state.Players {
		resJSON, err := json.Marshal(p.Resources)
		if err != nil {
			return err
		}
		px, err := json.Marshal(playerExtras{
			Passed:       p.Passed,
			ResearchDone: p.ResearchDone,
			Seat:         p.Seat,
			Tags:         p.Tags,
			Score:        p.Score,
		})
		if err != nil {
			return err
		}
		token := tokens[p.ID]
		ready := 0
		if p.IsReady {
			ready = 1
		}
		_, err = tx.Exec(`
INSERT INTO players(room_code, player_id, name, color, tr, is_ready, resources_json, reconnect_token, extras_json)
VALUES(?,?,?,?,?,?,?,?,?)
`, code, p.ID, p.Name, p.Color, p.TR, ready, string(resJSON), token, string(px))
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(`DELETE FROM audit_logs WHERE room_code=?`, code)
	if err != nil {
		return err
	}
	for _, log := range logs {
		deltaJSON, err := json.Marshal(log.Delta)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`
INSERT INTO audit_logs(id, room_code, ts, player_id, type, detail, delta_json)
VALUES(?,?,?,?,?,?,?)
`, log.ID, code, log.Timestamp.UTC().Format(time.RFC3339Nano), log.PlayerID, log.Type, log.Detail, string(deltaJSON))
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) LoadAllRooms() ([]PersistedRoom, error) {
	type roomRow struct {
		code, host, extras string
		gen                int
		phase              string
	}
	rows, err := s.db.Query(`SELECT code, host_player_id, generation, phase, COALESCE(extras_json,'{}') FROM rooms`)
	if err != nil {
		return nil, err
	}
	var roomRows []roomRow
	for rows.Next() {
		var rr roomRow
		if err := rows.Scan(&rr.code, &rr.host, &rr.gen, &rr.phase, &rr.extras); err != nil {
			rows.Close()
			return nil, err
		}
		roomRows = append(roomRows, rr)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var rooms []PersistedRoom
	for _, rr := range roomRows {
		pr := PersistedRoom{
			Code:         rr.code,
			HostPlayerID: rr.host,
			State:        game.NewGameState(rr.code),
			PlayersMeta:  make(map[string]PlayerMeta),
		}
		pr.State.Generation = rr.gen
		pr.State.Phase = rr.phase
		var rex roomExtras
		_ = json.Unmarshal([]byte(rr.extras), &rex)
		pr.State.TurnOrder = rex.TurnOrder
		pr.State.ActivePlayerID = rex.ActivePlayerID
		pr.State.ActionsThisTurn = rex.ActionsThisTurn
		pr.State.FirstPlayerIndex = rex.FirstPlayerIndex
		if rex.GlobalParams != nil {
			pr.State.GlobalParams = rex.GlobalParams
		}

		prows, err := s.db.Query(`
SELECT player_id, name, color, tr, is_ready, resources_json, reconnect_token, COALESCE(extras_json,'{}')
FROM players WHERE room_code=?
`, pr.Code)
		if err != nil {
			return nil, err
		}
		for prows.Next() {
			var (
				id, name, color, resJSON, token, extras string
				tr, ready                               int
			)
			if err := prows.Scan(&id, &name, &color, &tr, &ready, &resJSON, &token, &extras); err != nil {
				prows.Close()
				return nil, err
			}
			p := game.NewPlayer(id, name, color, 1)
			p.TR = tr
			p.IsReady = ready != 0
			if err := json.Unmarshal([]byte(resJSON), &p.Resources); err != nil {
				prows.Close()
				return nil, fmt.Errorf("resources for %s: %w", id, err)
			}
			var px playerExtras
			_ = json.Unmarshal([]byte(extras), &px)
			p.Passed = px.Passed
			p.ResearchDone = px.ResearchDone
			if px.Seat >= 1 {
				p.Seat = px.Seat
			}
			if px.Tags != nil {
				p.Tags = px.Tags
			}
			p.Score = px.Score
			game.EnsurePlayerDefaults(p)
			pr.State.Players[id] = p
			pr.PlayersMeta[id] = PlayerMeta{ReconnectToken: token}
		}
		prows.Close()
		if err := prows.Err(); err != nil {
			return nil, err
		}

		lrows, err := s.db.Query(`
SELECT id, ts, player_id, type, detail, delta_json
FROM audit_logs WHERE room_code=? ORDER BY ts ASC
`, pr.Code)
		if err != nil {
			return nil, err
		}
		for lrows.Next() {
			var id, ts, playerID, typ, detail, deltaJSON string
			if err := lrows.Scan(&id, &ts, &playerID, &typ, &detail, &deltaJSON); err != nil {
				lrows.Close()
				return nil, err
			}
			parsed, err := time.Parse(time.RFC3339Nano, ts)
			if err != nil {
				parsed, _ = time.Parse(time.RFC3339, ts)
			}
			delta := map[string]int{}
			_ = json.Unmarshal([]byte(deltaJSON), &delta)
			pr.AuditLogs = append(pr.AuditLogs, game.AuditLog{
				ID:        id,
				Timestamp: parsed,
				PlayerID:  playerID,
				Type:      typ,
				Detail:    detail,
				Delta:     delta,
			})
		}
		lrows.Close()
		if err := lrows.Err(); err != nil {
			return nil, err
		}

		game.EnsureStateDefaults(&pr.State)
		rooms = append(rooms, pr)
	}
	return rooms, nil
}
