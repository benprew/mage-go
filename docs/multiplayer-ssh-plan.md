# Plan: Multiplayer SSH Server via Charmbracelet Wish

## Context

The game currently supports one human vs one AI via a local Bubble Tea TUI (`cmd/tui/`). The game loop (`RunGameLoop` in `interactive.go`) communicates with a single human player over channels (`toTUI`/`fromTUI`) and computes AI decisions inline. To support two human players over the network, we need an SSH server (Wish) that accepts two connections, pairs them, and runs the game with each player getting their own TUI session.

## Step 1: Extract TUI into shared package `internal/tui/`

The TUI code lives in `cmd/tui/` as `package main` (1037 lines total across 5 files), making it impossible to import from a second binary. Move the reusable parts to `internal/tui/` so both `cmd/tui/` and `cmd/server/` can share them.

**Files to create:**
- `internal/tui/model.go` — Model struct, NewModel, Update, handleKey, handleEnter, handleTargetKey, handleBlockerAssignKey, tryInspect, getTargetChoices, getAttackers (from `cmd/tui/model.go`)
- `internal/tui/view.go` — View method, all render helpers (from `cmd/tui/view.go`)
- `internal/tui/styles.go` — Refactored styles as a `Styles` struct with a constructor taking `*lipgloss.Renderer` (from `cmd/tui/styles.go`)
- `internal/tui/decks.go` — DeckEntry, buildDeck, drawOpeningHand, deck lists (from `cmd/tui/decks.go`)

**Key changes during extraction:**
- Export types: `Model`, `Styles`, `CardDetail`, `GameStateMsg`, `WaitForGameState`, `DeckEntry`, `BuildDeck`, `DrawOpeningHand`, `AttackerInfo`
- **Styles refactor**: Replace package-level `lipgloss.NewStyle()` calls with a `Styles` struct and `NewStyles(r *lipgloss.Renderer) Styles` constructor. The `Model` gets a `Styles` field. For local play, use `lipgloss.DefaultRenderer()`. For SSH, use `bubbletea.MakeRenderer(sess)`.
- `Model` fields `toGame`/`fromGame` become exported `ToGame`/`FromGame`

**Update `cmd/tui/main.go`** to import `internal/tui` and become a thin wrapper:
```go
styles := tui.NewStyles(lipgloss.DefaultRenderer())
model := tui.NewModel(fromTUI, toTUI, styles)
```

**Files to modify:**
- `cmd/tui/main.go` — import `internal/tui`, remove local model/view/styles/decks
- `cmd/tui/model.go` — delete (moved)
- `cmd/tui/view.go` — delete (moved)
- `cmd/tui/styles.go` — delete (moved)
- `cmd/tui/decks.go` — delete (moved)

**Verify**: `go build ./cmd/tui && go run ./cmd/tui` still works as before.

## Step 2: Add `RunMultiplayerGameLoop` to `interactive.go`

New function alongside the existing `RunGameLoop` (which stays unchanged for local AI play).

**New types:**
```go
type PlayerChannels struct {
    ToPlayer   chan GameMsg          // game writes, player reads
    FromPlayer <-chan PriorityAction // player writes, game reads
}
```

**New function:**
```go
func RunMultiplayerGameLoop(g *Game, channels [2]PlayerChannels)
```

**Differences from `RunGameLoop`:**
- No AI logic — both players communicate via channels
- Per-player snapshots: `SnapshotGameState(g, 0)` for player 0, `SnapshotGameState(g, 1)` for player 1
- After any player acts, send a state update to the *other* player so their TUI refreshes
- Undo disabled (`CanUndo: false` always) — too complex for multiplayer
- Disconnect detection: use `select` with timeout on `FromPlayer` reads; if channel closes or times out, notify remaining player and return
- On game over, send final message to both players then close both `ToPlayer` channels

The existing helper functions are reused directly: `GetAvailableActions`, `executeAction`, `getEligibleAttackers`, `getEligibleBlockers`, `performAttack`, `performBlock`, `attackerOptions`, `blockerOptions`, `reportCombatResults`, `describeAction`.

**File to modify:** `interactive.go` — add ~150 lines (the new function + type)

## Step 3: Create SSH server `cmd/server/`

**Files to create:**

### `cmd/server/main.go` — Wish SSH server entry point
- Create a Wish server on `:2222` (configurable via flag/env)
- Use `wish.WithHostKeyPath(".ssh/server_ed25519")` for host key (auto-generated on first run)
- Use Wish `bubbletea.Middleware` to serve a Bubble Tea program per SSH session
- In the handler: get player name from `sess.User()`, call `lobby.Join(name)`, return a `tui.Model` wired to the player's channels
- If lobby rejects (game in progress), return a simple model showing "Server is full, try again later"
- Graceful shutdown on SIGINT/SIGTERM

### `cmd/server/lobby.go` — Lobby/pairing logic
```go
type Lobby struct {
    mu         sync.Mutex
    waiting    *PlayerSession  // first player, nil if nobody waiting
    gameActive bool
}

type PlayerSession struct {
    Name     string
    ToGame   chan mage.PriorityAction // player writes actions here
    FromGame chan mage.GameMsg        // player reads state here
}
```

**Flow:**
1. First player connects: `lobby.Join()` creates `PlayerSession`, stores as `lobby.waiting`, returns it. The player's TUI model starts and shows "Waiting for game to start..." (already handled by `View()` when `state == nil`).
2. Second player connects: `lobby.Join()` creates second `PlayerSession`, builds the game (two `BasePlayer`s, decks, opening hands), starts `go RunMultiplayerGameLoop(game, channels)`, sets `gameActive = true`, clears `waiting`, returns session. The game loop sends the first `GameMsg` to both players, transitioning both TUIs to game mode.
3. Third+ player connects while game active: `lobby.Join()` returns nil, server shows "full" message.
4. When game loop returns (game over or disconnect): reset `gameActive = false`, clear sessions, lobby is ready for new connections.

**Disconnection handling:** When a player's SSH session drops, their `ToGame` channel closes (Bubble Tea program exits). The game loop's `select` on `FromPlayer` detects this, sends a "opponent disconnected" `GameMsg` to the remaining player, and returns. The lobby cleans up.

**New dependency:** `github.com/charmbracelet/wish` (add via `go get`)

## Step 4: Update TUI model for multiplayer awareness

Minor changes to `internal/tui/model.go`:
- When `CanUndo` is false, don't show the undo hint in the view
- Handle channel closure gracefully (when game loop closes `ToPlayer`, the `WaitForGameState` cmd returns a `GameOver` message)
- The existing `View()` already handles `state == nil` ("Waiting for game to start...") which covers the lobby wait state

## Summary of all file changes

| File | Action | Description |
|------|--------|-------------|
| `internal/tui/model.go` | Create | Extracted from `cmd/tui/model.go`, exported types |
| `internal/tui/view.go` | Create | Extracted from `cmd/tui/view.go` |
| `internal/tui/styles.go` | Create | Refactored styles with renderer parameter |
| `internal/tui/decks.go` | Create | Extracted deck building utilities |
| `cmd/tui/main.go` | Modify | Import `internal/tui`, thin wrapper |
| `cmd/tui/model.go` | Delete | Moved to `internal/tui/` |
| `cmd/tui/view.go` | Delete | Moved to `internal/tui/` |
| `cmd/tui/styles.go` | Delete | Moved to `internal/tui/` |
| `cmd/tui/decks.go` | Delete | Moved to `internal/tui/` |
| `interactive.go` | Modify | Add `PlayerChannels` type + `RunMultiplayerGameLoop` |
| `cmd/server/main.go` | Create | Wish SSH server entry point |
| `cmd/server/lobby.go` | Create | Lobby pairing logic |
| `go.mod` / `go.sum` | Modify | Add `charmbracelet/wish` dependency |

## Verification

1. After Step 1: `go build ./cmd/tui && go run ./cmd/tui` — local game works unchanged
2. After Step 2: `go build ./...` — compiles with new function
3. After Step 3-4: Start server with `go run ./cmd/server`, connect from two terminals with `ssh localhost -p 2222`. First player sees "Waiting...", second player triggers game start, both see the game from their perspective, game plays to completion.
4. Test disconnect: kill one SSH session mid-game, verify other player gets notified.