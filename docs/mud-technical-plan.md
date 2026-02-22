# MUD Technical Plan

## Guiding Constraint

The MTG game engine (`pkg/mage/`) is not modified. The MUD is a pure wrapper
that sequences existing bubbletea programs and adds new ones around them.

---

## Architecture: Sequential Programs

The existing SSH handler runs two bubbletea programs in sequence (lobby →
game). The MUD extends this into a loop:

```
SSH connect
  → identify by key fingerprint → load collection + last location
  → loop:
      worldProgram  → Result{action, ...}
        "fight_npc"    → ante draw → runCombat (existing tui.Model) → award loot → save
        "deckbuilder"  → deckbuilderProgram → save
        "shop"         → shopProgram (future) → save
        "quit"         → break
  → save location → disconnect
```

`runProgram(sess, p)` already exists in `cmd/server/main.go` and is reused
unchanged.

---

## New Module Layout

```
internal/
  worlddata/          # Go-code world content (you author this)
    worlddata.go      # shared type definitions: RoomDef, NPCDef, DeckEntry, etc.
    rooms.go          # room graph — edit freely
    npcs.go           # NPC roster — edit freely

  world/              # shared runtime world state (server-wide singleton)
    world.go          # World{mu, Rooms, OnlinePlayers, DefeatedNPCs}
    room.go           # Room, runtime types converted from worlddata
    npc.go            # NPC runtime type

  collection/         # per-player card collection and persistence
    collection.go     # Collection{Cards, Gold, LastBooster} + Load/Save JSON
    deck.go           # Deck + Sideboard, validation, BuildDeck → []mage.Card
    builder.go        # bubbletea deckbuilder model

  mud/                # world navigation bubbletea model
    model.go          # mudModel state machine
    view.go           # text rendering
    result.go         # Result type — what worldProgram quits with
    messages.go       # tea.Msg types for async events

cmd/server/
  main.go             # SSH middleware — replace lobby loop with MUD loop
  player.go           # PlayerState, SSH key fingerprint, load/save
  lobby.go            # keep existing PvP lobby for now (minimal changes)
```

---

## Phase 1: Data Layer (no bubbletea)

### `internal/worlddata/`

Pure data definitions. No imports outside stdlib. The user edits `rooms.go`
and `npcs.go` to author world content without touching engine code.

**Key types:**

```go
type DeckEntry struct { Name string; Count int }

type LootEntry struct {
    Name        string
    Weight      int  // relative weight; 0 = always drops
}

type NPCDef struct {
    ID             string
    Name           string
    Description    string
    Deck           []DeckEntry
    Personality    string   // "aggro" | "control" | "midrange" | "tempo" | "burn"
    AnteForced     bool
    LootFixed      []string    // always dropped on win
    LootTable      []LootEntry // draw LootTableCount from this
    LootTableCount int
    GoldReward     int
    Dialog         []string
    RespawnMinutes int
}

type Treasure struct {
    Cards    []string
    SearchDC int      // 0 = always found
}

type RoomDef struct {
    ID          string
    Name        string
    Description string
    Exits       map[string]string // "north" → room ID
    NPCIDs      []string
    GoldLock    int    // 0 = free
    BossLock    string // NPC ID that must be defeated to enter
    Hidden      []Treasure
    StartChest  bool   // new player starter deck chest
}
```

Exported accessor functions: `Rooms() []RoomDef`, `NPCs() []NPCDef`,
`NPCByID(id string) (NPCDef, bool)`.

### `internal/collection/collection.go`

```go
type Collection struct {
    Cards       map[string]int // card name → count owned
    Gold        int
    LastBooster time.Time
    Decks       []SavedDeck
    ActiveDeck  int // index into Decks
}

type SavedDeck struct {
    Name      string
    Main      []DeckEntry
    Sideboard []DeckEntry
}
```

- `Load(dataDir, fingerprint string) (*Collection, error)`
- `(c *Collection) Save(dataDir, fingerprint string) error`
- `(c *Collection) Add(cards []string)` — add cards to collection
- `(c *Collection) Remove(card string) bool` — remove one copy (for card loss)
- `(c *Collection) SpendGold(amount int) bool` — returns false if insufficient

Persistence: `data/players/<fingerprint>/collection.json`

### `internal/collection/deck.go`

- `BuildDeck(entries []DeckEntry, ownerID uuid.UUID) ([]mage.Card, error)` —
  calls `mage.CreateCard`, returns error for unknown cards
- `Validate(deck SavedDeck, coll *Collection) []string` — returns list of
  validation errors (unknown cards, too many copies, deck too small)

---

## Phase 2: Runtime World State

### `internal/world/`

The `World` struct is a server-wide singleton. It holds:
- A map of runtime `Room` values (converted from `worlddata.RoomDef` at
  startup)
- `OnlinePlayers map[string]*WorldPlayer` — keyed by key fingerprint
- `DefeatedNPCs map[string]map[string]time.Time` — `[fingerprint][npcID]`

**Key operations:**
- `(w *World) Enter(fingerprint, roomID string)` — record player presence
- `(w *World) Leave(fingerprint string)`
- `(w *World) PlayersInRoom(roomID string) []string`
- `(w *World) RecordDefeat(fingerprint, npcID string)`
- `(w *World) NPCAvailable(fingerprint, npcID string) bool` — respawn check
- `(w *World) CanEnter(fingerprint, roomID string, coll *collection.Collection) (bool, string)` — checks gold/boss locks

NPC respawn: checked inline against `time.Since(defeatedAt) > respawnDuration`.
No background goroutines needed.

---

## Phase 3: MUD bubbletea Model

### `internal/mud/model.go`

State machine:

```go
type mudState int
const (
    stateRoom      mudState = iota  // viewing current room
    stateExamine                    // examining an NPC or item
    stateChallenge                  // confirm fight with NPC (deck select)
    stateLoot                       // viewing rewards after combat
    stateInventory                  // viewing collection summary
    stateDeckbuilder                // deckbuilder sub-model
)
```

`mudModel` holds:
- `*world.World` (shared server ref)
- `*collection.Collection` (this player's data)
- `roomID string`
- `fingerprint string`

**Key input bindings (stateRoom):**

| Key | Action |
|---|---|
| `n/s/e/w` | Move between rooms |
| `l` | Re-describe room (look) |
| `j/k` or `↑/↓` | Cycle through NPCs in room |
| `f` / `enter` | Fight selected NPC |
| `t` | Talk to NPC (cycle dialog) |
| `x` | Search room for hidden treasure |
| `i` | View inventory summary |
| `d` | Open deckbuilder |
| `q` | Quit to disconnect |

### `internal/mud/result.go`

```go
type Result struct {
    Action      string  // "fight_npc" | "deckbuilder" | "shop" | "quit"
    RoomID      string  // current room when quitting
    NPC         *worlddata.NPCDef
    AnteEnabled bool
    Collection  *collection.Collection
}
```

`mudModel` returns this from `Init()` → `tea.Quit` path; SSH handler type-asserts it.

---

## Phase 4: SSH Server Wiring

### `cmd/server/player.go`

```go
type PlayerState struct {
    Fingerprint string
    Username    string
    Collection  *collection.Collection
    RoomID      string
}

func loadPlayer(dataDir, fingerprint, username string) *PlayerState
func (ps *PlayerState) save(dataDir string) error
```

Key fingerprint extraction:

```go
// in SSH handler:
fingerprint := ssh.FingerprintSHA256(sess.PublicKey())
```

If the session has no public key (password auth), fall back to username as
fingerprint with a warning log.

### `cmd/server/main.go` — new MUD loop

Replace `gameMiddleware` with `mudMiddleware(world, dataDir)`:

```go
for {
    wm := mud.NewModel(w, ps.Fingerprint, ps.RoomID, ps.Collection)
    rawResult, _ := runProgram(sess, tea.NewProgram(wm, opts...))
    result, ok := rawResult.(mud.Result)
    ps.RoomID = result.RoomID
    ps.save(dataDir)

    switch result.Action {
    case "fight_npc":
        loot := runCombat(sess, result, ps, opts...)
        ps.Collection.Add(loot)
        ps.save(dataDir)

    case "deckbuilder":
        db := collection.NewBuilder(ps.Collection)
        runProgram(sess, tea.NewProgram(db, opts...))
        ps.save(dataDir)

    case "quit", "":
        return
    }
}
```

`runCombat` wraps the existing `tui.NewModel` / `StartAIGame` flow, with ante
card draw injected before game start.

---

## Phase 5: Deckbuilder UI

`internal/collection/builder.go` — a bubbletea model:

- Left pane: collection list with count badges, sorted by color then name
- Right pane: current deck (main + sideboard tab)
- `a` — add card to deck
- `r` — remove card from deck
- `s` — move card to/from sideboard
- `/` — search filter
- `v` — validate deck (show errors)
- `enter/q` — save and quit

The builder is the most UI-heavy piece and comes last.

---

## Ante Implementation

When `result.AnteEnabled == true` in the combat result:

1. Before game starts, draw one card at random from each player's shuffled deck
   (using `rand.Intn(len(deck))`).
2. Store the two ante cards separately (`anteMine`, `anteTheirs`).
3. Pass the modified deck (ante card removed) to `BuildDeck` / library add.
4. After game ends: winner's `Collection.Add([]string{anteMine, anteTheirs})`,
   loser's collection has already had their ante card removed before the game.

NPC ante cards do not go into an NPC's "collection" — they are effectively
removed from circulation if the player loses.

---

## Dependency Graph

```
worlddata   ←  world  ←  mud  ←  cmd/server
worlddata   ←  collection  ←  mud  ←  cmd/server
pkg/mage    ←  collection (BuildDeck)
pkg/mage    ←  cmd/server (existing game loop)
internal/tui (future)  ←  cmd/server
```

No circular imports. `worlddata` has no engine dependencies.

---

## Build Order

1. `internal/worlddata/` — types + content (`rooms.go`, `npcs.go`) ✓
2. `internal/collection/collection.go` + `deck.go`
3. `internal/world/`
4. `internal/mud/model.go` + `view.go` + `result.go`
5. `cmd/server/player.go` — identity + persistence wiring
6. `cmd/server/main.go` — replace lobby with MUD loop
7. `internal/collection/builder.go` — deckbuilder UI (last, most polish)

---

## What Is NOT Changed

- `pkg/mage/` — engine untouched
- `pkg/mage/interactive/` — AI player untouched
- `cards/limited/` — card set untouched
- Existing PvP lobby flow — kept, may be integrated as a room action later

---

## Open Questions

- Minimum deck size: 20 (limited-style) or 40 (constructed-style)?
- What cards are always in the shop staple list?
- Loss consequence magnitudes: how much gold on loss? Which NPCs trigger card
  loss?
- Booster pack composition: how many cards, which rarity distribution?
- Does the player have an XP/level, or is gold the only progression currency?
