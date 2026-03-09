# AI v2 Improvements

## Overview

Follow-up improvements to the AI after the Phase 0-5 plan was completed. These address remaining blindspots in combat math, move generation, evaluation, and decision-making.

## Implementation Order

```
1. First Strike / Deathtouch in Combat Math     [combat.go, eval/lethal.go]
2. Mulligan Logic                                [ai.go, game_loop.go]
3. Gang Blocking 3+ Creatures                    [combat.go]
4. Lifelink in Combat / Lethal / Race            [combat.go, eval/lethal.go]
5. Hand Quality in Eval (castability-aware)       [eval/eval.go]
6. Search-Based Blocking                          [search.go]
7. Multi-Spell Turns in Search                   [search.go]
8. X-Spell Move Generation                       [movegen.go]
9. Modal Spell Move Generation                   [movegen.go, ai.go]
10. Better SearchPlayer Choice Stubs              [search_player.go]
11. Smarter Spell Resolution in Search            [search.go]
```

Items 1-5 are self-contained heuristic improvements (no search changes). Items 6-11 improve the minimax search quality.

---

## 1. First Strike / Deathtouch in Combat Math

**Files:** `ai/combat.go` (`evaluateCombatOutcome`), `eval/lethal.go`

### Problem

`evaluateCombatOutcome` (combat.go:52-77) treats all damage as simultaneous. A 1/1 first striker vs a 3/1 should kill the 3/1 before taking damage. A 1/1 deathtouch should trade with a 6/6.

### Changes

**`evaluateCombatOutcome`:**
- Split combat into two sub-steps when any creature has first/double strike:
  1. First strike damage: creatures with FirstStrike or DoubleStrike deal damage. Check lethal on blockers/attackers.
  2. Normal damage: surviving creatures without FirstStrike deal damage. Creatures with DoubleStrike deal damage again.
- Deathtouch: any damage from a deathtouch source is lethal (1 >= toughness for kill check).

**`eval/lethal.go`:**
- `estimatePushThroughDamage`: account for first strike when estimating blocker survival.
- `CalculateLethal`: deathtouch creatures count as "kills any single blocker" for push-through.

### Tests
- 2/2 first striker vs 3/1 blocker: attacker survives, blocker dies.
- 1/1 deathtouch vs 6/6: both die (trade).
- 1/1 deathtouch first strike vs 6/6: attacker kills blocker, doesn't take damage.
- 4/4 double strike vs 3/3 blocker: blocker dies on first strike, 4 trample through on second if trample.
- Lethal calc with deathtouch attacker gets through when opponent only has big blockers.

---

## 2. Mulligan Logic

**Files:** `ai/ai.go`, `pkg/mage/interactive/game_loop.go`

### Problem

AI always keeps its opening hand. A hand of 7 lands or 0 lands should be mulliganed.

### Changes

**`ai/ai.go` — add `ShouldMulligan(hand []mage.Card) bool`:**
- Count lands and spells in hand.
- Mulligan if: 0-1 lands, 6-7 lands, or no castable spells (all spells CMC > land count + 2).
- On mulligan to 6: slightly more lenient (accept 1-5 lands).
- On mulligan to 5: accept anything with 2+ lands.
- Never mulligan below 5.

**`game_loop.go` — integrate mulligan step:**
- After drawing opening hands, before game starts, offer mulligan.
- For AI: call `ShouldMulligan`, if true, shuffle hand into library, draw N-1.
- For human: present mulligan choice via channel (new `PromptMulligan` type or use existing `ChoiceMay`).

### Tests
- Hand with 0 lands: mulligan.
- Hand with 7 lands: mulligan.
- Hand with 3 lands + 4 spells: keep.
- Hand with 1 land + 6 seven-drops: mulligan.
- Mulligan to 6 with 2 lands: keep.

---

## 3. Gang Blocking 3+ Creatures

**Files:** `ai/combat.go` (`findGangBlocks`)

### Problem

`findGangBlocks` only tries exactly 2 blockers (combat.go:133). A 7/7 might need 3 small creatures to kill.

### Changes

- After checking 2-blocker combos, try 3-blocker combos if no 2-blocker combo works.
- Cap at 3 blockers (combinatorics: `C(n,3)` is manageable for typical board sizes ≤ 10).
- Relax value threshold: allow gang block if attacker score > 80% of combined blocker score (currently must be strictly greater).

### Tests
- 7/7 attacker, three 3/3 blockers: gang block kills it.
- 5/5 attacker, three 1/1 tokens: don't gang block (losing 3 creatures for 1, unless lethal threatened).
- When lethal threatened: always gang block regardless of value.

---

## 4. Lifelink in Combat / Lethal / Race

**Files:** `ai/combat.go`, `eval/lethal.go`

### Problem

Lifelink is never accounted for in combat scoring, lethal detection, or race calculation.

### Changes

**`evaluateCombatOutcome`:**
- Track lifelink damage dealt by our attackers (adds to our life in evaluation).
- Track lifelink damage dealt by opponent blockers (adds to their life).
- Adjust `cs.Score` by net lifelink difference.

**`eval/lethal.go`:**
- `CalculateLethal`: lifelink attackers going unblocked reduce opponent's effective life less (opponent might gain life from their lifelink blockers).
- `CalculateRace`: lifelink creatures effectively reduce the clock (they deal damage AND gain life, so opponent's clock gets longer).

### Tests
- 3/3 lifelink attacking into empty board: 3 damage + 3 life gained.
- Race calc with lifelink: creature that deals 3 and gains 3 has a better clock than vanilla 3/3.
- Combat score prefers attacking with lifelink creature when racing.

---

## 5. Hand Quality in Eval (castability-aware)

**Files:** `eval/eval.go` (`handQuality`)

### Problem

`handQuality` (eval.go:192-211) gives all non-lands a flat 2 points. A hand of 7-drops with 2 lands scores the same as a hand of 2-drops with 2 lands.

### Changes

- Score each card by castability: if `CMC <= availableMana`, full value (2 points). If `CMC <= availableMana + 2`, partial value (1 point). If `CMC > availableMana + 3`, no value (dead card).
- Lands: 1 point if < 5 lands, 0 if >= 5 lands (diminishing returns).
- Excess lands beyond 6: negative value (flood).

### Tests
- Hand with [2-drop, 3-drop], 3 mana available: both fully valued.
- Hand with [7-drop], 2 mana available: valued at 0 (dead card).
- Hand with 6 lands: excess lands penalized.

---

## 6. Search-Based Blocking

**Files:** `ai/search.go`

### Problem

`SearchStrategy.Blockers()` (search.go:131-133) delegates entirely to heuristic fallback. The minimax engine is never used for blocking.

### Changes

- Generate blocker assignment sets (similar to `GenerateAttackerSets`):
  - No blocks.
  - Greedy single-block (current heuristic result).
  - Best single block per attacker.
  - Gang block the biggest attacker + single block rest.
- For each assignment set, clone game, apply blocks, run 1-ply eval.
- Pick assignment set with best eval score.
- Fall back to heuristic if timeout or too many combinations.

### Tests
- AI finds blocking line that prevents lethal when heuristic would not block.
- AI gang-blocks a must-kill threat when heuristic would single-block something else.

---

## 7. Multi-Spell Turns in Search

**Files:** `ai/search.go`

### Problem

Search commits to one move per ply. Can't find "bolt the blocker, then attack for lethal" sequences.

### Changes

- After applying a non-pass move in minimax, generate moves again for the same player (same priority window).
- Treat "pass" as ending the priority window (switches to opponent or advances step).
- Limit per-turn move chains to 3 (prevent explosion).
- Track accumulated moves in the search state.

### Tests
- AI finds "Lightning Bolt blocker + attack for lethal" 2-move sequence.
- AI casts two spells in one turn when both are valuable.
- Search doesn't explode (stays within node budget with move chaining).

---

## 8. X-Spell Move Generation

**Files:** `ai/movegen.go`

### Problem

`expandSpellMoves` doesn't generate X variants. An X=5 Fireball should be considered alongside X=3.

### Changes

- In `GeneratePriorityMoves`, detect X costs on cards.
- Generate moves for X = 1, max/2, max (where max = available mana - fixed costs).
- Each X variant gets its own `Move` with `XValue` field set.
- `applySpellCast` in search.go: use `m.XValue` when resolving X damage.

### Tests
- Fireball with 5 mana: generates X=1, X=2, X=4 variants.
- Search picks X value that exactly kills the best target.

---

## 9. Modal Spell Move Generation

**Files:** `ai/movegen.go`, `ai/ai.go`

### Problem

`AIPlayer.ChooseMode` (ai.go:86-95) is hardcoded for specific cards. Modal spells in movegen always use mode 0.

### Changes

**`movegen.go`:**
- Detect modal spells (cards where `ChooseMode` is called during resolution).
- For now: generate one move per mode (up to 3 modes).
- Set a `ModeIndex` field on `Move`.

**`ai.go` (`ChooseMode`):**
- Replace hardcoded switch with heuristic evaluation:
  - "Gain life" mode: prefer when life < 10.
  - "Deal damage" mode: prefer when opponent has low life or good targets.
  - "Draw cards" mode: prefer when hand size < 3.
  - "Destroy" mode: prefer when opponent has targets.
- Fall back to mode 0 for unknown modes.

### Tests
- Healing Salve at 20 life: picks prevent damage mode.
- Healing Salve at 5 life: picks gain life mode.
- Charm with 3 modes: generates 3 moves in movegen.

---

## 10. Better SearchPlayer Choice Stubs

**Files:** `pkg/mage/search_player.go`

### Problem

SearchPlayer always picks first candidate, mode 0, declines may abilities, discards cheapest. These defaults are often wrong.

### Changes

- `ChooseTargets`: prefer opponent's permanents for detriment effects (check if any target belongs to opponent).
- `ChoosePermanent`: when reason contains "sacrifice", pick lowest-value own permanent. When reason contains "destroy", pick highest-value opponent permanent.
- `ChooseMayAbility`: accept if description contains "draw" or "damage" or "destroy".
- `ChooseMode`: use same heuristic as improved `AIPlayer.ChooseMode`.
- `ChooseNumber`: for "damage" context, pick max. For "discard" context, pick min.

### Tests
- SearchPlayer sacrifices worst creature (not best).
- SearchPlayer accepts "draw a card" may ability.
- SearchPlayer targets opponent's creature for destruction.

---

## 11. Smarter Spell Resolution in Search

**Files:** `ai/search.go` (`applySpellCast`)

### Problem

`applySpellCast` (search.go:367-438) only resolves the first matching effect category. Multi-effect spells, life gain, and buff effects are ignored.

### Changes

- Process ALL effects in order (remove early `return` statements).
- Add handling for:
  - Life gain effects (`props.LifeGain > 0`).
  - Buff effects (`BoostUntilEndOfTurn`): apply power/toughness bonus to target permanent.
  - Token creation: add a vanilla permanent with the token's stats.
  - Bounce effects: remove permanent from battlefield, add card to hand.
- For X spells: substitute X value from move before resolving damage.

### Tests
- Spell that draws 2 AND deals 3: both effects resolve.
- Spell that gains 3 life: life gain applied in clone.
- Giant Growth on a creature: power/toughness updated in clone.

---

## Dependency Graph

```
1. First Strike/Deathtouch ──┐
2. Mulligan                  │  (independent)
3. Gang Block 3+             │
4. Lifelink                  │
5. Hand Quality              │
                             ▼
6. Search Blocking ─────── requires 1, 3, 4
7. Multi-Spell Search ──── independent
8. X-Spell Movegen ──────── independent
9. Modal Spell Movegen ──── independent
10. SearchPlayer Stubs ──── independent
11. Spell Resolution ────── requires 8
```

Items 1-5 can be done in any order. Items 6-11 can mostly be done in parallel except 11 depends on 8.
