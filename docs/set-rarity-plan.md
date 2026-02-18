# Plan: Add Set and Rarity System

## Context

Cards in Magic: The Gathering belong to sets (e.g., Limited Edition Alpha) and each printing has a rarity (Common, Uncommon, Rare, etc.) and collector number. The codebase currently has no concept of sets or rarity. The Alpha set cards are already implemented across `cards/alpha_*.go` files but lack set metadata. We need a parallel registry system for set/printing data, separate from the card gameplay registry, because the same card can appear in multiple sets at different rarities.

## New Files

### 1. `set.go` — Types and set registry (root `mage` package)

Defines all new types following existing `keyword.go` pattern (iota enums with `String()` methods):

- **`Rarity`** — `RarityLand`, `RarityCommon`, `RarityUncommon`, `RarityRare`, `RarityMythic`, `RaritySpecial`, `RarityBonus` with `String()` and `Code()` methods
- **`SetType`** — `SetTypeCore`, `SetTypeExpansion`, `SetTypeSupplemental`, `SetTypePromotional`, `SetTypeCustom` with `String()`
- **`SetCard`** struct — `Name string`, `CollectorNumber string`, `Rarity Rarity`
- **`Set`** struct — `Name`, `Code`, `ReleaseDate time.Time`, `SetType`, `Cards []SetCard`

Registry functions (mirroring `registry.go` pattern):
- `RegisterSet(set *Set)`
- `GetSet(code string) *Set`
- `AllSets() []*Set`
- `SetsContainingCard(cardName string) []*Set`
- `CardInSet(setCode, cardName string) (SetCard, error)`
- `CardsByRarity(setCode string, rarity Rarity) []SetCard`

### 2. `set_test.go` — Unit tests (root `mage` package)

Tests for all enum `String()`/`Code()` methods, register/get round-trip, query functions, and error cases.

### 3. `cards/sets.go` — Helper for card set registrations

Contains `buildDate(year, month, day int) time.Time` helper used by all set registration files.

### 4. `cards/set_alpha.go` — Alpha set registration

Registers `"LEA"` (Limited Edition Alpha, 1993-08-05, Core) with all ~295 cards from the XMage reference at `~/mage/Mage.Sets/src/mage/sets/LimitedEditionAlpha.java`. Includes cards not yet implemented in the card registry — the set registry is pure data. Each entry has name, collector number, and rarity.

### 5. `cards/set_misc.go` — Non-Alpha set registrations

Registers minimal sets for cards in `creatures.go`, `spells.go`, `artifacts.go`, `enchantments.go` that aren't from Alpha:
- Mirrodin (MRD): Bonesplitter, Lightning Greaves
- Magic 2014 (M14): Elvish Mystic, Gladecover Scout
- Return to Ravnica (RTR): Fencing Ace
- Ravnica: City of Guilds (RAV): Boros Swiftblade
- Invasion (INV): Blurred Mongoose
- Worldwake (WWK): Kor Firewalker
- Magic 2010 (M10): Centaur Courser, Doom Blade, Wind Drake, Goblin Piker
- Urza's Legacy (ULG): Rancor
- Mirage (MIR): Pacifism
- Mage Test Cards (TST): Wraithbloom Cultivator

### 6. `cards/set_test.go` — Integration tests (cards package)

Verifies Alpha set is registered with correct metadata, spot-checks known card rarities (Black Lotus=Rare, Lightning Bolt=Common, Serra Angel=Uncommon, Forest=Land), and checks that set cards with implementations exist in the card registry.

## Implementation Order

1. Create `set.go` (zero dependencies on existing code)
2. Create `set_test.go`, run tests
3. Create `cards/sets.go` (buildDate helper)
4. Create `cards/set_alpha.go` (transcribe from XMage reference)
5. Create `cards/set_misc.go`
6. Create `cards/set_test.go`, run full suite

## Verification

```bash
go vet ./...
go test ./...
```
