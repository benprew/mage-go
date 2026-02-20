# Modal Spells Implementation Plan

## Status: In Progress

Build passes, all tests pass. TUI view rendering done. Needs manual TUI testing.

## Overview

MTG modal spells ("Choose one —") require the mode to be chosen at cast time (rule 601.2b), before targets. The chosen mode is locked in on the stack and read during resolution.

## What's Done

### Engine

- `Card` interface: added `Modes() []string` — returns nil for non-modal cards
- `BaseCard`: added `modes` field, `Modes()`, `SetModes()`, copied in `Copy()`/`CloneFrom()`
- `StackObject`: added `ModeChoice int` — stores the chosen mode on the stack
- `Game`: added `CurrentMode int` — set from `StackObject.ModeChoice` during `ResolveStackObject`, cleared after resolution (same pattern as `CurrentX`)
- `CastSpellByName` and `CastSpellByID`: after building the `StackObject`, if `card.Modes()` is non-empty, calls `player.ChooseMode()` and stores result in `obj.ModeChoice`

### Player Interface

- `Player` interface: added `ChooseMode(modes []string, reason string) int`
- `BasePlayer`: default returns 0 (first mode)
- `TestPlayer`: pops from scripted `chooseMode` queue, defaults to 0

### Test Harness

- `TestGame.ChooseMode(player, mode)`: queues a mode choice for the player
- Queue is consumed at cast time by the engine calling `player.ChooseMode()`

### Cards

- **Healing Salve**: calls `SetModes()` with two modes, effect reads `g.CurrentMode`
- **Twiddle**: NOT modal (Oracle text has no "Choose one"). Uses `player.ChooseMode()` at resolution time inside the effect, which is rules-correct for resolution-time choices.

### TUI

- `HumanPlayer` type in `interactive/` wraps `BasePlayer`, overrides `ChooseMode` to return a pre-set `PendingMode` value
- `PriorityAction`: added `ModeChoice int`
- `ActionOption`: added `Modes []string`
- `GetAvailableActions`: populates `Modes` from `card.Modes()`
- `RunGameLoop`: sets `HumanPlayer.PendingMode` from `action.ModeChoice` before calling `executeAction`
- `cmd/tui/main.go`: uses `interactive.NewHumanPlayer()` instead of `mage.NewBasePlayer()`
- `Model`: added `selectingMode`, `modeOptions`, `modeCursor`, `pendingModeOpt` state
- `handleEnter`: modal spells enter mode selection before target selection
- `handleModeKey`: up/down/enter/esc for mode picker
- `view.go`: renders "Choose Mode" menu with cursor

## Data Flow

```
Test harness:  g.ChooseMode(P, 1) → queue → CastSpellByName → player.ChooseMode() → StackObject.ModeChoice → g.CurrentMode
TUI:           user picks mode → PriorityAction.ModeChoice → HumanPlayer.PendingMode → player.ChooseMode() → StackObject.ModeChoice → g.CurrentMode
AI:            BasePlayer.ChooseMode() returns 0 → StackObject.ModeChoice → g.CurrentMode
```

## What's Left

- Manual TUI testing with Healing Salve in a deck
- AI heuristics for mode selection (currently always picks mode 0)

## Design Notes

- **Fork copies modes**: `StackObject.ModeChoice` is copied when a spell is copied on the stack, preserving the chosen mode. Correct per rules.
- **Resolution-time choices** (Twiddle's tap/untap) are separate from cast-time modality. They call `player.ChooseMode()` inside `Effect.Apply`. The TUI human player defaults to mode 0 for these — same pre-existing limitation as all other `Choose*` methods.
