# Fixed selected discard execution slice

Capability P5-FIXED-DISCARD; completed 2026-10-04. Compilation and gameplay
measurements remain separate. This run stops after one slice.

## Candidate audit and selection

Fresh strict loader and full AST inventory: 2,445 records, 986 distinct names,
981 compiled records, 1,464 rejected records; 292 accepted names and 694 rejected
names. Baseline RON tests, engine short tests and build passed. Collection
rejection is expected and is not a build or test failure.

The audit used the positioned parser inventory, including descendants of
unsupported parents. Complete records were read for the candidates below.
Identical Jade Statue and Disrupting Scepter reprints count once.

| Candidate | Distinct affected names | Estimated complete unlocks | Other blockers and downstream benefit | Effort |
| --- | --- | --- | --- | --- |
| Fixed selected discard | Jalum Tome, Bazaar of Baghdad, Disrupting Scepter, Mindstab Thrull; adjacent shapes in Recall, Dwarven Armorer, Mishra's War Machine, Land's Edge, Chains of Mephistopheles | 2: Jalum Tome and Bazaar | Scepter needs activation timing; Thrull needs combat event/player and optional sacrifice. Recall needs X, actual results and graveyard return. Remaining names use discard as a cost, not an action. Fixed discard supplies a tested action for Scepter/Thrull and a shared batch for later results. | Small compiler adapter and inventory role changes; shared engine batch repair; moderate choice/payment/context matrix |
| Typed spell-cast references | Presence of the Master, Nether Void, In the Eye of Chaos, Invoke Prejudice | 2: Presence and Nether Void | Eye needs mana-value and variable optional payment. Invoke also needs opponent caster and color-sharing filters. Enables exact event spell identity and event payer for these four cards. | Medium typed event environment and engine event binding audit; counter/leave-stack/copy/payment matrix |
| Temporary animation | Mishra's Factory, Jade Statue, Thelonite Druid | Up to 3 with prerequisites | Factory needs temporary type/subtype and base P/T. Statue additionally needs combat-only activation and end-of-combat expiry. Druid needs selected sacrifice cost and collection capture. Enables shared temporary type/layer changes. | Larger compiler/layer/timing/duration/cost work and interaction matrix |

Fixed discard has two concrete full unlocks with the smallest implementation
boundary. The engine audit found a prerequisite: selected multi-card discard
sent events after each individual move. CR 608.2f requires simultaneous moves.
This slice repairs that path with the existing shared batch, then adds adapters.
The initial selection and acceptance contract were stated before editing.

## Complete-record requirements and exact supported contexts

Jalum Tome (ATQ:40): artifact, Book, mana cost {3}; {2}, tap: draw one, then
select and discard one. Bazaar of Baghdad (ARN:8): land; tap: draw two, then
select and discard three. Neither uses a discard cost or a saved result.
Local `data/ATQ.json` and `data/ARN.json` Oracle text agrees with both records.
Candidate Oracle text was also checked in local LEA/ATQ/LEG/FEM data.

Supported signatures:

- Action symbol `DiscardACard` (not a zero-argument tuple).
- Action `DiscardNumberCards(Integer(n))`, with one nonnegative integer literal.
- `PlayerAction(player, action)` with either of those actions and the existing
  typed `You`, declared `Ref_TargetPlayer`, or supported spell-controller refs.
- These actions are accepted by existing spell, activation, source entry/death,
  controller-upkeep, spell-cast and optional-payment continuation action contexts.
  They add no event, target, payment or source-zone forms.
- Inventory marks the number operand as `value` in action and player-action roles.

Bare actions use the captured resolving controller. Each explicit player action
has its own lexical context; it does not redirect the following bare action.
Fresh factory calls build separate effects and target bindings. Dynamic amounts,
negative literals, malformed tuples, unbound/incompatible players, cost-role
DiscardACard and saved results remain errors. Recall's ValueX diagnostic now
names the unsupported fixed amount instead of the entire action.

RON owns syntax, validation and lowering to `mage.Discard(ref, mage.Fixed(n))`.
The engine owns choice, replacement, zone moves, events and trackers. No engine
API signature changes. The no-result typed Discard path already delegates to
legacy DiscardCards; this executor now uses `discardActionsTogether`. It prepares
replacements before moves and sends events after moves. Result-option paths are
outside this new batch contract. There is no separate RON zone executor.

Reuse: player target bindings, Sequence, DrawFor, fixed values, transactional
activation costs, stack controller capture, target/copy validation, phase status,
Game.Clone, DiscardAction, replacement rechecks and discardActionsTogether.
The implement-engine skill was used for the engine prerequisite. The user task
also requires RON adapters and gameplay evidence, supplied in this same slice.

Rules consulted: CR 101.3 (impossible portions), 113.7a (independent abilities),
121.2 (individual draws), 608.2b/c/d/f (legality, order, choices, simultaneous
actions), 616.1/f (replacement choice and rechecks), 701.9a/b (discard and chooser).
No new general source LKI, phasing or replacement ordering service is introduced.

## Card and contract test mapping

| Card or contract | Evidence |
| --- | --- |
| Jalum Tome, all printed rules | TestRONFixedDiscardRealCards and ActivationRollback: mana/tap payment before resolution, draw before affected-player choice, selected newly drawn card, empty initial hand, source departure/return/control/phasing, cloned stack resolution |
| Bazaar of Baghdad, all printed rules | Same tests: no mana cost, draw two before discard three, short hands, tap legality and captured controller; ReplacementAndEvents: affected-player choice, three replacement outcomes and complete destinations before events |
| Shared typed player scope and fresh instances | TestRONFixedDiscardContexts: explicit target versus following bare action, separate factory bindings, retargeted spell copy, entry and controller upkeep |
| Zero, empty hand and later instruction | TestRONFixedDiscardZeroEmptyAndContinuation |
| Invalid declarations and nested roles | TestRONFixedDiscardInvalid and Inventory |
| Engine simultaneous destinations and continuation | TestSelectedDiscardBatchEvents, normal graveyard and replaced library destinations |
| Engine replacements before moves, recheck prevention and clone isolation | TestSelectedDiscardBatchReplacementPreparation |
| Retained typed discard contracts | Existing TestTypedDiscardAmount* choice/replacement/short-hand/validation/copy tests, run in engine suite |
| Accepted RON records and strict registry | Existing RON/loader/inventory tests, updated exact collection counts |

The first RON tests failed for missing discard adapters and missing inventory
roles. The engine test failed because events saw partial hands and destinations.
Fixture build errors used unqualified core events, then an incorrect upkeep event
name; these were corrected to actual APIs. Full RON tests next exposed old exact
inventory counts; they were updated to the measured counts. No test expectation
was weakened to permit unsupported behavior.

## Measurements and validation

Estimated and actual gain agree: two unique cards, two records. Strict loader
now compiles **983/2,445** and rejects **1,462**. Accepted unique names: **294/986**;
rejected unique names: **692**. No previously accepted record regresses. Only
Recall's still-rejected diagnostic changes. Strict loader and inventory issues
are identical. Global engine_validated and behavior_verified remain null.
This slice adds RON gameplay evidence for two complete records; it does not
claim collection gameplay completion.

All 12 source hashes match the vocabulary snapshot. No RON source defect was
corrected, so no vocabulary refresh is necessary. Gaea's Liege and Fastbond
observations remain unresolved. The multi-card event issue was an engine
contract gap, and the missing forms were compiler adapters. No environment
failure occurred. The initial user edits to .gitignore and the execution design
remain excluded from the commit. No Go definitions, DSL exports, generator or
startup wiring changed. Removal gates remain closed; no caller-removal audit
is claimed.

| Check | Result |
| --- | --- |
| Baseline RON tests, engine short suite, build | PASS |
| Focused new engine and RON tests, `-count=1` | Expected failures before implementation; final PASS |
| RON/CLI tests, `-count=1` | PASS; repeated after lint |
| `go test ./pkg/mage/... -short -count=1` | PASS after exact inventory expectations were refreshed |
| `make test` | PASS |
| `make build` | PASS |
| `go vet ./...` | PASS |
| `make lint` | Final PASS, zero issues |
| `git diff --check` | PASS |
| Strict loader and full inventory | Expected exit 1, identical 983 compiled / 1,462 rejected counts and issues |

The first lint run flagged the existing private withDestination parameter after
a new test supplied another constant caller. The synthetic replacement now
copies its proposal and sets the test destination itself. Production APIs and
lint rules are unchanged. Automatic formatting only reordered test imports;
focused checks were repeated after review.

Next ranked candidates: typed spell-cast references; temporary animation;
activation modifiers/timing including Disrupting Scepter. Each needs fresh
inventory and complete-record audit before selection. Full collection gameplay,
loading, runtime embedding, native startup, Go-card/generator deletion and
unused-DSL cleanup remain open.
