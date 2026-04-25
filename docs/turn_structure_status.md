# Turn Structure: Status & Plan

Tracks engine coverage of MTG Comprehensive Rules Chapter 5 (Turn Structure) and the primitives needed to close the remaining gaps.

## Test layout

All in `pkg/mage/gametest/`, one file per CR section:

| File | CR section |
|---|---|
| `turn_general_test.go` | 500 |
| `turn_beginning_test.go` | 501–504 |
| `turn_main_test.go` | 505 |
| `turn_combat_test.go` | 506, 507, 511 (combat phase meta + removal from combat) |
| `turn_combat_attackers_test.go` | 508 |
| `turn_combat_blockers_test.go` | 509 |
| `turn_combat_damage_test.go` | 510 |
| `turn_end_test.go` | 512–513 |
| `turn_cleanup_test.go` | 514 |
| `turn_structure_helpers_test.go` | shared helpers (combat event recorder) |

Engine-internals counterparts live in `pkg/mage/priority_test.go`. Per-CR test search: `grep -n "CR 5" pkg/mage/gametest/turn_*.go`.

## Engine primitives

### Done
1. **TurnSchedule** (commit `43db1a3`) — per-turn mutable step list on `Game.Schedule`. Methods: `AppendExtraStep`, `InsertStepAfter`, `SkipNextOccurrenceOfStep`, `SkipNextCombatPhase`, `SkipNextTurnFor`. `RunTurn` / `Run` / harness `Execute` all drive off the schedule. Unblocks CR 500.7, 500.9, 500.10, 500.11, 505.1a.
2. **Per-step mana pool empty** (commit `e7447cf`) — `Game.EmptyManaPools()` deferred at end of every `RunStep`/`RunStepWithPriority`. `ManaPool.ProducedThisTurn` tally + `CountProducedThisTurn` / `AssertManaProduced*` DSL preserves observability. Unblocks CR 500.5.
3. **`EvtCleanup` event + cleanup priority observable** (CR 514.3, 514.3a) — `core.EvtCleanup` added and fired at the top of `doCleanupActions`; `BeginningOfEachCleanupStepTrigger` helper exposes it to card code. `Game.CleanupPriorityRounds` counts how many times priority was granted during a cleanup step (normally 0; incremented only on 514.3a fallback). Both skipped cleanup tests now pass: `TestCleanup/CR_514.3_...` and `TestCleanup/CR_514.3a_...`.

4. **Instant-speed "can't attack this turn"** (commit `947a9cc`, CR 506.4a) — `PreventAttackingUntilEndOfTurn(permID)` continuous effect at `LayerAbility` revoking `AttrCanAttack`, plus `PreventAttackingTargetUntilEndOfTurn()` effect wrapper. Mirror of the existing block-prevention plumbing. Per-step continuous-effect re-application means already-declared attackers stay declared (CR 506.4a). Unskipped: `TestTurnStructureCombatRemoval/CR_506.4a_...`.

5. **Attacks-alone / blocks-alone selectors** (commit `90d3753`, CR 506.5) — `Combat.AttacksAlone` / `BlocksAlone` (snapshot at declaration) and `IsAttackingAlone` / `IsBlockingAlone` (live), with `SnapshotAttackedAlone` / `SnapshotBlockedAlone` wired into both `doDeclareAttackers`/`doDeclareBlockers` and the `ExecuteAttackers`/`ExecuteBlockers` AI search-clone paths. Unskipped: `TestTurnStructureCombatRemoval/CR_506.5_...`.

6. **Multi-blocker damage-assignment DSL** (commit `7940691`, CR 510.1c) — `CombatDamageAssigner` interface with `GetBlockerOrder` / `GetCombatDamageAssignment`; `doNormalBlockedDamage` consults it with CR 510.1c lethal-first / CR 702.19b trample validation, falling back to greedy split if absent or invalid. Harness exposes `tg.AssignCombatDamage(attacker, map[blocker]int)` and `tg.ChooseBlockerOrder(attacker, blockers...)`. Unskipped: `TestCombatDamage/CR_510.1c_...`.

7. **Runtime color override on declared attackers** (commit `894f04a`, CR 508.2a) — investigation only; the layer-5 pipeline (`ChangeColorEffect` → `ColorOverride` → indefinite `TargetEffect`) was already correct, likely fixed indirectly by the per-step continuous-effect re-application in #2/#3. Unskipped with a hard `t.Fatalf` guard so any regression surfaces: `TestDeclareAttackers/CR_508.2a_...`.

9. **Extra-upkeep-step primitive** (commit `5987183`, CR 503.2) — `Game.InsertStepAfter(anchor, step)` / `Game.AppendExtraStep(step)` already existed on `TurnSchedule` from item #1; exposed them on the `GameMutator` interface so card effects (Paradox Haze, Obeka) can insert an additional upkeep step into the current turn. Unskipped: `TestTurnStructureBeginning_Upkeep_MultipleUpkeepSteps` — installs an "at beginning of upkeep gain 1 life" pinger, stops at turn 3 Upkeep, calls `InsertStepAfter(Upkeep, Upkeep)`, drives both upkeep steps, and asserts +2 life across the pair. No new engine plumbing needed beyond the two interface methods.

8. **CR 509.1h (no-trample, blocker destroyed after blocks)** — not an engine bug. `doNormalBlockedDamage` already skips dead blockers via `FindPermanent(bid) == nil` and, without trample, routes 0 damage to the defender. Test: `TestTurnStructureCombatRemoval/CR_509.1h_...`. Trample interaction (CR 702.19b — damage that would have gone to removed blockers still "counts" against the attacker's trample accounting) is *not* yet correct; defer until #7's damage-assignment DSL lands.

10. **Harness ordering: scripted actions fire at the player's first priority in the target step** (commits `2b961c1`, `51e846d`, `d94aa34`, `981ac18`, CR 117 / 509.2) — scripted `CastSpell` / `ActivateAbility` / `CastInResponseTo` / `ActivateInResponseTo` now flow through `Game.OnPriority` via a scripted handler instead of being drained imperatively before `RunStepWithPriority`. Cast/activate actions are popped on the earliest-seq match for `(g.Turn, g.Step, playerRef)` when the stack is empty; responses are popped when the parent cast's spell is the current top of the stack (by card-name match). `autoPlayLands` also moves into the same handler. `CastSpellByID` now delegates to `CastSpellByName` after auto-tapping, unifying the priority-loop and harness cast paths (the previous divergence silently skipped additional costs, spell-cost modifiers, and mana restrictions). The harness no longer needs `Game.WithInStep` wrapping (the CR 500.12 observable still works via `RunStepWithPriority`). About a dozen card tests needed rewrites — the recurring shape was either "cast at FirstStrikeDamage when no first-strikers exist" (→ `DeclareBlockers`) or "script two same-step actions expecting a specific order" (→ `CastInResponseTo` / `ActivateInResponseTo`). The `TestIcatianMoneychanger_SacrificeGainsLife` expected value was genuinely CR-incorrect under the old harness and has been updated.

9. **TBA-fence invariant / `Game.InStep()` observable** (commit `91e45e5`, CR 500.12) — `Game.inStep` flag set true inside `RunStep` / `RunStepWithPriority` (defer-cleared on exit); `Game.OnFireEvent func(*Game, GameEvent)` hook invoked at the top of `FireEvent`; `Game.WithInStep(fn)` helper used by the harness to mark scripted pre-step actions (`executeOrderedActions`, `autoPlayLands` in both `Execute` and `PlayToEnd`) as logically part of the enclosing step's priority round. The CR 500.12 test records every fired event together with `g.InStep()` at fire time and asserts no event escaped a step. Unskipped: `TestTurnPhasesAdditional/CR_500.12_no_game_events_between_steps`.

### Next up (priority order)

(Nothing blocking. See "Card-side follow-ups" below.)

## Remaining skipped tests

| File | Rule | Gap |
|---|---|---|

Note: 503.2 is now exercised at the engine level via `GameMutator.InsertStepAfter`; card-side wiring (*Paradox Haze*, *Obeka*) can be added when those sets are implemented.

## Card-side follow-ups (not engine work)

TurnSchedule is in place but unused by any card. Candidate cards once their sets are in flight:
- *Stasis* — `SkipNextOccurrenceOfStep(Untap)` on a per-turn trigger.
- *Time Stop* — end the turn; needs to drain the schedule and skip straight to cleanup.
- *Angel's Grace* — partial turn protection.
- *Final Fortune* / *Time Walk* — already covered by `GrantExtraTurn`, but `SkipNextTurnFor` supports "then skip your next turn."
- *Paradox Haze* — `AppendExtraStep(Upkeep)` as an ETB / beginning-of-upkeep trigger.
- *Obeka, Splitter of Seconds* — insert additional upkeep steps after the current phase.
- *Static Orb* / *Winter Orb* — not a skip; a continuous restriction on untap targets. Orthogonal.

## How to pick up where we left off

1. Grep for `XXX:` in `pkg/mage/gametest/turn_*.go` to find the next skipped test.
2. Implement the corresponding primitive above.
3. Unskip the test, run `go test ./...`.
4. Update this file with a commit hash and strike the item off "Next up."
