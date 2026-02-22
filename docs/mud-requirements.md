# MUD Requirements

Captured from design discussion, 2026-02-21.

## Overview

A MUD (Multi-User Dungeon) layer built on top of the existing SSH server
infrastructure. Players connect via SSH, wander a small world, encounter NPCs,
fight them using the existing MTG engine, and collect cards into a personal
collection. The world is inspired by Shandalar — a new original plane where
players are dueling wizards working to save the world from an encroaching
Darkness.

The game engine (`pkg/mage/`) is not touched. The MUD lives entirely in new
modules.

---

## World & Setting

- **Plane:** Original — "Shandalar", a new plane with MTG-canon flavor mixed in.
  Players are not planeswalkers but wizards fighting to save the world.
- **Size:** Small — 5–15 rooms. Every room has purpose.
- **Tone:** Shandalar (the 1996 MicroProse game) is the reference. Tense
  economy, meaningful combat, world exploration as meta-game.

---

## Identity & Persistence

- **Player identity:** SSH public key fingerprint — the same key = the same
  player regardless of username.
- **Save data location:** `data/players/<key-fingerprint>/` — JSON files.
  - `collection.json` — card collection + gold + last booster timestamp
  - `location.json` — last room ID
  - `decks.json` — saved named decks (active + sideboard)
- **Data format for world content:** Go code in `internal/worlddata/` — rooms
  and NPCs are hardcoded Go structs. Edit and recompile to change content.

---

## New Player Experience

1. Player connects, no save data found.
2. Placed in **The Threshold** (room 1).
3. A chest is present — player chooses one of the starter archetypes.
4. That deck becomes their starting collection. They can build/modify from it
   immediately.

---

## Card Acquisition

Players acquire cards through:

| Source | Notes |
|---|---|
| NPC loot | Guaranteed card(s) after winning a fight |
| Ante | If ante was in play, winner takes both face-down cards |
| Hidden treasure | Search rooms; some require luck or items |
| Shop NPC | Buy staple cards for gold; one booster pack per 24h |
| Quest rewards | Deferred — unlock after core loop is solid |
| PvP | Deferred — keep existing lobby for now |

---

## Combat & Stakes

### Ante Mechanic

Before the game begins, one card is drawn face-down from each player's deck.
The winner takes both.

- **Forced ante:** Some NPCs always play with ante. The flag lives in the NPC
  definition.
- **Proposed ante:** Players may propose ante before challenging any NPC. The
  NPC accepts based on their flavor/personality.

### Loss Consequences

Tunable — the system should support any combination of:
- Gold loss (always safe to implement)
- EXP loss (if EXP is added)
- Card loss (from collection — roguelite stakes)
- Ante card lost to NPC (always, if ante was in play)

The specific values will be dialed in after playtesting. Design for
tunability: each NPC or encounter type specifies its consequence type and
magnitude.

---

## Progression

### Gold Economy

- Gold is earned by winning combat encounters.
- Gold is spent at the shop and to unlock gold-gated doors.
- Economy balance is TBD — design for tunability, not a fixed number.

### Locked Rooms

Two types of room locks:
1. **Gold lock:** Pay N gold to pass (like Shandalar town entry fees).
2. **Boss lock:** Defeat a specific NPC to unlock a passage.

Some locks are one and the same (gatekeeper you can also pay past).

### Story Flags

Defeating specific NPCs sets flags tracked in the player's save data. Flags
can unlock rooms, change NPC dialog, or gate quest rewards.

---

## Collection & Deckbuilding

### Deck Rules

- Only cards the player owns (collection-bound)
- Minimum deck size (exact value TBD, suggest 20 for limited-style)
- Maximum 4 copies of any non-basic card
- One active deck + 15-card sideboard
- Between games in a series (e.g. best-of-3 against a boss), player may swap
  cards from sideboard into main deck

### Deckbuilder

- Accessible anywhere in the world (may be restricted to safe rooms in a
  future iteration)
- Full two-pane UI: collection on left, current deck on right
- Filter/search by card name
- Save multiple named decks; choose active deck before each encounter

---

## Shop

- One shop NPC in the world (Merchant Varro)
- **Staple stock:** A fixed list of cards always available for gold purchase
- **Booster:** One random booster pack (N cards from the limited set) available
  per 24-hour period; cooldown tracked in `collection.json`

---

## NPC Loot

Each NPC has:
- **Deterministic drops:** Cards always awarded on defeat (can be empty)
- **Loot table:** A weighted list; one (or more) cards drawn from it on defeat
- **Loot table count:** How many draws from the loot table (default 1)
- Harder NPCs may have better loot tables and higher loot counts

If ante was in play, the ante card is awarded separately and always.

---

## Multiplayer Presence

- Players in the same room see each other via periodic refresh (`tea.Tick`)
- No real-time chat for now
- PvP challenges between players in the same room: deferred (keep existing
  lobby for now)
- Card trading: deferred

---

## NPC Dialog

- Flavor lines only for now — a few rotating lines per NPC
- No branching dialog trees in MVP
- Future: dialog trees that unlock quests or rooms

---

## Visual Style

- Text-only MUD style for MVP
- Room description + exits + NPC list + player list
- ASCII art and two-panel layout as future polish

---

## MVP Definition

The minimum viable product that is worth showing to a friend:

1. Walk between rooms
2. Fight NPCs using the existing engine
3. Collect loot cards into a persistent collection
4. **Ante mechanic** — both forced (NPC) and proposed (player)
5. Edit your deck using the deckbuilder

Shop, locked rooms, and story progression are the next layer after MVP.
