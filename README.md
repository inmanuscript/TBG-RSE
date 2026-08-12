# TBG-RSE

**T**erraforming Mars **B**oard **G**ame **R**esource **S**ynchronization **E**ngine

Self-hosted companion app for *Terraforming Mars*: track resources, production, turns, and generation progress with live sync across devices on your LAN.

Single Go binary with an embedded Vue SPA. No cloud account required.

> **Unofficial fan project.** *Terraforming Mars* is a trademark of FryxGames / Stronghold Games.  
> This software is not affiliated with, endorsed by, or associated with the rights holders.  
> It does not include official artwork, card text, or other copyrighted game assets.

## Features

- **Multi-room** — short room codes; SQLite persistence for reconnect / server restart
- **Seat & color** — pick clockwise seat (1–5) and a unique player color when joining
- **Own-board editing** — change your stock / production / TR anytime; opponents are view-only
- **Turn management** — up to 2 actions per turn; Pass with 0 actions leaves you out for the generation
- **Research phase** — buy 0–4 cards at 3 MC each
- **Standard projects & conversions** — power plant, asteroid, aquifer, city, greenery, patent sale; plant/heat conversions
- **Tag counters** — manual Building / Space / Science / … tracking
- **Production automation** — Energy→Heat, MC from TR+production, then other productions
- **VP helper** — end-game score sheet (TR auto + manual tile / milestone / award / card VP)
- **Live feed** — toasts and activity log over WebSocket

## Requirements

| Purpose | Tool |
|---------|------|
| Run (prebuilt binary) | None beyond the OS |
| Build from source | Go 1.22+, Node.js 20+ |

## Quick start

```bash
# Build (Windows)
./scripts/build.ps1

# Build (Unix)
./scripts/build.sh

# Run
./tbg-rse.exe -addr :8080 -db tbg-rse.db   # Windows
./tbg-rse      -addr :8080 -db tbg-rse.db   # Unix
```

Open `http://<host>:8080` from tablets / phones on the same network.  
Create a room (choose seat + color), share the room code, others join with free seats/colors.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `:8080` | HTTP / WebSocket listen address |
| `-db` | `tbg-rse.db` | SQLite file for room recovery |

Security is intentionally minimal (private LAN). Do not expose this to the public Internet without additional hardening.

## Build from source (manual)

```bash
cd web
npm install
npm run build          # → webui/dist
cd ..
go build -o tbg-rse ./cmd/tbg-rse
```

## Development

Terminal 1 — API + WebSocket:

```bash
go run ./cmd/tbg-rse -addr :8080
```

Terminal 2 — Vite (proxies `/ws` to `:8080`):

```bash
cd web
npm run dev
```

## Game flow

1. **Lobby** — create or join; pick **seat (1–5, clockwise = turn order)** and a unique color  
2. **Research** — each player buys 0–4 cards (3 MC each)  
3. **Action** — seat order; up to 2 actions (claim action / projects / conversions), then End Turn  
   - **Pass** with 0 actions → out for the rest of this generation  
4. When everyone has passed → **Production** (confirm) → next generation Research (first player rotates)  
5. Host **End Game** → VP helper  

### Standard projects

| Project | Cost / effect |
|---------|----------------|
| Sell patents | +1 MC per card sold |
| Power plant | −11 MC, +1 Energy production |
| Asteroid | −14 MC, +1 TR |
| Aquifer | −18 MC, +1 TR |
| Greenery | −23 MC, +1 greenery tile (score) |
| City | −25 MC, +1 city tile (score) |

Plant 8 → Greenery and Heat 8 → Temperature also consume an action (temperature +1 TR).

### Production order

1. Energy stock → Heat stock  
2. MC stock += max(0, TR + MC production)  
3. Steel / Titanium / Plant / Energy / Heat stock += production  
4. Generation += 1; Research opens; first-player seat rotates  

## Stack

| Layer | Tech |
|-------|------|
| Backend | Go, `nhooyr.io/websocket`, modernc SQLite |
| Frontend | Vue 3, Vite, Tailwind CSS, Lucide |
| Ship | `go:embed` of `webui/dist` into one binary |

```
Clients  ──WebSocket──►  Go Hub (in-memory rooms + SQLite)
                ▲
         HTTP (embedded SPA)
```

## License

This project’s source code is released under the [MIT License](LICENSE).

*Terraforming Mars* and related names remain the property of their respective owners.  
Use of those names here is for identification of an unofficial companion tool only.
