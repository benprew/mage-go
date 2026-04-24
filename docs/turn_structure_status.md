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
3. **CR 509.1h (no-trample, blocker destroyed after blocks)** — not an engine bug. `doNormalBlockedDamage` already skips dead blockers via `FindPermanent(bid) == nil` and, without trample, routes 0 damage to the defender. Test: `TestTurnStructureCombatRemoval/CR_509.1h_...`. The cast is scheduled at `CombatDamage` rather than `DeclareBlockers` because of a harness ordering quirk (see below), not because of CR semantics. Trample interaction (CR 702.19b — damage that would have gone to removed blockers still "counts" against the attacker's trample accounting) is *not* yet correct; defer until #7's damage-assignment DSL lands.

   **Harness ordering bug — follow-up:** `gametest/harness.go:348` runs `executeOrderedActions` before `RunStepWithPriority`, which means a `CastSpell(turn, step, ...)` resolves *before* that step's turn-based actions instead of during its priority round. Per CR, TBAs run first and priority opens afterwards (e.g. CR 509.2 grants priority only after `doDeclareBlockers`). The current inversion prevents writing the natural 509.1h test (cast at `DeclareBlockers` targeting the declared blocker). Fixing this globally would likely churn many existing tests that implicitly rely on "ordered action fires at the top of step X"; handle as its own focused change.

### Next up (priority order)

4. **`EvtCleanup` event + cleanup priority round** (CR 514.3, 514.3a)
   - Add `core.EvtCleanup` to `pkg/mage/core/event.go` / `event_enumer.go`, fired at the start of cleanup's turn-based action.
   - Extend `RunStepWithPriority(Cleanup)` to give priority iff SBAs fired or triggers are waiting; after resolution, start another cleanup step. Today it already has partial support — see the `if g.doCleanupActions()` branch in `priority.go`. Needs an observable for "priority was denied" (a counter on `Game` the DSL can assert).
   - Unblocks two skipped tests in `turn_cleanup_test.go`.

5. **Instant-speed "can't attack / can't block" effects** (CR 506.4a)
   - Today only an attachment-based `PreventAttachedFromAttacking` exists, checked at declaration.
   - Add a targeted continuous effect applied at instant speed that survives post-declaration. Piggyback on the existing layered continuous-effects system.
   - Unlocks gotcha control cards like *Marshal's Anthem*-style "target creature can't attack this turn."
   - Unblocks one skip in `turn_combat_test.go`.

6. **Attacks-alone / blocks-alone selector** (CR 506.5)
   - Trivial: `Combat.IsAttackingAlone(permID)` / `IsBlockingAlone(permID)` given the current attacker/blocker tracking.
   - Four variants per CR: "attacks alone," "attacking alone," "blocks alone," "blocking alone."
   - Unlocks *Silverblade Paladin*, *Cathedral of War*, *Wingmate Roc*, etc.
   - Unblocks one skip in `turn_combat_test.go`.

7. **Multi-blocker damage-assignment DSL** (CR 510.1c)
   - Gametest-only; engine likely already supports the mechanic.
   - `tg.AssignCombatDamage(attacker, map[blocker]int)`.
   - `tg.ChooseBlockerOrder(attacker, blockers...)` — damage-assignment order, needed for trample (CR 702.19b) and classic "4/4 vs two 2/3s" scenarios.
   - Unblocks one skip in `turn_combat_damage_test.go`.

8. **Runtime color-change effect** (CR 508.2a)
   - Existing `ChangeColorEffect` / `ColorOverride` didn't apply in the failing test. Investigate layer-5 registration on resolving instants. Likely a bug or missing integration rather than a new primitive.
   - Unblocks one skip in `turn_combat_attackers_test.go`.

9. **`EvtBetweenSteps` / TBA-fence invariant** (CR 500.12)
   - Negative invariant; hardest to observe, lowest payoff.
   - Defer until after #3–#8 land; then add a shared assertion helper that verifies events fired during a turn-based action are associated with the enclosing step.

10. **Harness ordering: scripted actions should fire at the player's first priority in the target step** (CR 117 / 509.2 / general priority correctness)
    - **Current behavior:** `gametest/harness.go:348` (and the `PlayToEnd` mirror near line 832) runs `executeOrderedActions` immediately before `RunStepWithPriority`. That resolves each `CastSpell(turn, step, ...)` at the top of the step — *before* the step's turn-based actions (`doUntap`, `doUpkeepActions`, `doDeclareBlockers`, …) and before upkeep/begin-combat/etc. triggers are placed on the stack. Per CR 117.1b the active player receives priority only after TBAs and after triggered abilities have been put on the stack, so the harness inverts the CR ordering on every step.
    - **Observable symptom:** `CR 509.1h attacker still blocked after blocker destroyed` cannot be scripted at `DeclareBlockers` — the removal spell resolves before `doDeclareBlockers`, so the blocker dies before it can block. The test currently works around this by scheduling the cast at `CombatDamage` (which is also not CR-correct, just happens to produce the right observable).
    - **Proposed design:** drain scripted actions through `RunPriorityRound` rather than executing them out-of-band.
      1. Add `Game.ScheduledPriorityActions map[uuid.UUID][]PriorityAction` — a per-player FIFO.
      2. The default handler (and `autoPassHandler` in the harness) checks the queue before returning `PriorityPass`: if the player whose turn it is to act has a queued action, pop and return it.
      3. Harness `CastSpell` / `ActivateAbility` translate into a `PriorityAction` plus a "gate" predicate that the handler uses to decide *when* to release it into the queue (match on `g.Turn`, `g.Step`, and player). One clean encoding: keep the existing `castActions` / `activateActions` slices; the handler scans the slice for an action matching `(g.Turn, g.Step, playerRef)` and pops+returns the first match.
      4. Remove the pre-step `executeOrderedActions` call. Actions now flow through the engine's priority loop, so resolution, stack ordering, and SBA/trigger interleaving are handled by `RunPriorityRound` — no special-case `ResolveStack()` in the harness.
    - **Responses (`CastInResponseTo`, `ActivateInResponseTo`):** currently imperative ("after the main cast hits the stack, cast these, then resolve"). In the new model, a response is another queued `PriorityAction` gated on `g.Stack.Peek().SourceID == <cast's CardID>` (or similar). Needs its own gate predicate on the queued entry.
    - **Expected blast radius:** card-side tests that rely on "scripted action resolves before upkeep/begin-combat triggers land on the stack" will fail and need to be rewritten. A one-shot audit at the time of landing this: `go test ./... 2>&1 | grep '^--- FAIL'` after removing the eager call revealed ~13 failures across `cards/legends`, `cards/limited`, `cards/antiquities`, and one in `pkg/mage/gametest` (`TestDiscardCost`). Most are "cast X at Upkeep to interact with an upkeep trigger" patterns — the fix is to express the interaction as a response (`CastInResponseTo`) or reschedule to the prior main phase where the effect should have been set up.
    - **Why it wasn't done in this session:** scope. The failures are real CR-correctness issues masked by the harness quirk; fixing them is a genuine 13-test rewrite, not a mechanical pass. Warrants its own branch.

## Remaining skipped tests

| File | Rule | Gap |
|---|---|---|
| `turn_general_test.go` | 500.12 | no between-steps observable (negative invariant) |
| `turn_beginning_test.go` | 503.2 | no extra-upkeep-step primitive |
| `turn_combat_test.go` | 506.4a | no instant-speed "can't attack" effect |
| `turn_combat_test.go` | 506.5 | no attacks-alone selector |
| `turn_combat_blockers_test.go` | 509.1h | dedup — covered by `TestTurnStructureCombatRemoval/CR_509.1h_...` in `turn_combat_test.go` |
| `turn_combat_attackers_test.go` | 508.2a | color override didn't apply |
| `turn_combat_damage_test.go` | 510.1c | no multi-blocker damage-assignment DSL |
| `turn_cleanup_test.go` | 514.3 | no observable for priority denial in cleanup |
| `turn_cleanup_test.go` | 514.3a | no `EvtCleanup` event |

Note: 503.2 also needs card-side wiring (*Paradox Haze*, *Obeka*) to be exercised end-to-end, but the engine primitive is the blocker.

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
