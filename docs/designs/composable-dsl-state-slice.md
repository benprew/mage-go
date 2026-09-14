# Composable DSL: state conditions and sacrifice

This implements the design's first vertical slice. Later families remain on
legacy adapters; this is not completion of the full DSL migration.
The engine reference is the repository Comprehensive Rules revision effective
February 27, 2026: CR 400.7, 603.8, 608.2, and 701.21a.

## Contracts and capability inventory

| Expression | Contract and engine executor | Evidence |
| --- | --- | --- |
| `WhenState(condition, effect)` | Uses `GenericTriggered`, pending storage, APNAP placement, and normal stack resolution. Condition is not an intervening-if. Observe complete derived-state changes and instruction boundaries; retain transient triggers and suppress outstanding instances. | `TestStateSacrificeMigration`, `TestAuditStateTriggerTransient`, `TestAuditStateTriggerLifecycle`, `TestComposableTransientCharacteristics` |
| `Controls(filter)` | Current trigger controller; includes source; ignores phased-out objects; uses effective permanent characteristics. | `TestComposableStateSacrifice`, `TestComposableConditions`, `TestComposableConditionTruthTable` |
| `ControlsNone(filter)`, `BattlefieldHasNone(filter)` | Controller-relative and global empty-set queries, respectively. | `TestComposableConditionTruthTable`, `TestStateSacrificeMigration` |
| `AllConditions`, `AnyCondition`, `NotCondition` | Pure short-circuit composition; reject empty conjunction/disjunction and nil children at registration. | `TestComposableConditions`, `TestComposableConditionTruthTable`, `TestComposableDefinitionValidation` |
| `StateConditionFunc` | Trusted pure callback; declares source-choice/battlefield dependencies and a description. Receives copied immutable views, without game/player/card/permanent mutators. No randomness, choices, or mutable captures. | `TestComposableStateReaderSnapshot`, Jihad migration; registration checks declarations, not arbitrary callback behavior |
| `Sacrifice(This)` | Stored resolving controller instructs sacrifice only of the captured battlefield incarnation that they still control. Absence is a no-op. Uses `DoSacrifice`, its LKI, and zone/sacrifice events. Does not test the trigger condition again. No action-result binding in this slice. | `TestAuditSacrificeSourceController`, `TestComposableSourceReentry`, `TestComposableSourceLifetimes`, `TestComposableSequenceSourceIdentity` |
| `Sequence` | Ordered execution over `CompositeEffects`; shared context and instruction-boundary observations. Keeps child metadata and derives legacy properties. | `TestComposableMetadata`, `TestComposableSequenceSourceIdentity` |
| `IfState` | Pure condition evaluated during resolution, then optional instruction. Added for the existing Pestilence resolution branch. | `TestComposableMetadata`, existing Pestilence card tests |
| Definition inspection | Named child paths, roles, descriptions, source-context requirement, and structural metadata. Registry validates outside its lock before storing factories; aggregate validation loads all sets in the DSL tests. | `TestComposableDefinitionValidation`, `TestComposableRegistrationBoundary`, `TestRegisteredDefinitions` |

Jihad's state callback has no captured state. It reads the source's chosen
player and color, then current colors and token status of battlefield snapshots.
This replaces its printed-card-color check. Its existing entry-choice and static
boost implementations are separate, unmigrated abilities; this slice does not
claim to correct those abilities.

The source adapter carries the existing private incarnation through event and
state pending triggers, delayed triggers, activated abilities, stack clones,
and effect contexts. No public IDs, zone registry, or action executor changed.
A cloned stack must also retain the original state-trigger suppression key.
A complete layer recalculation now observes state triggers; individual layer
updates do not. Existing simultaneous-action observation batches still apply.

## Removed exports and semantic parity

| Removed export | Replacement | Callers and verification |
| --- | --- | --- |
| `SacrificeIfControls`, `SacrificeIfControlsAbility` | `WhenState(Controls(HasSubType("Dwarf")), Sacrifice(This))` | Goblins of the Flarg; source-inclusive matching and stack timing in `TestComposableStateSacrifice` and `TestStateSacrificeMigration`. Removed engine SBA type switch. |
| `SacrificeUnlessLand`, `SacrificeUnlessLandAbility` | `WhenState(ControlsNone(HasSubType("Island")), Sacrifice(This))` | Vodalian Knights, Giant Shark, Pirate Ship, Sea Serpent. `TestStateSacrificeMigration` plus existing card suites. Removed engine SBA type switch. |
| DSL `NewStateTriggered` | `WhenState` | Serendib Djinn, Drop of Honey, Mana Vortex, Jihad. Engine constructor remains for legacy engine tests/adapters. |
| `ControllerHasNoPermanentMatching` | `ControlsNone` | Dandân, Island Fish Jasconius, Merchant Ship, Serendib Djinn. Former zone-change approximations become state triggers. |
| `NoBattlefieldPermanentMatching` | `BattlefieldHasNone` | Drop of Honey, Mana Vortex; Pestilence uses `IfState` at its existing resolution boundary. |
| `SacrificeSource`, `SacrificeSourceStep` | `Sacrifice(This)` | All callers listed below. Same resolving action at top level and within pipelines. Source text remains diagnostic; properties preserve the empty legacy summary. Mana Vortex retains its explicit detrimental hint and text in its existing pipeline envelope. Identity and controller corrections apply consistently. |

Sacrifice costs and target/gathered sacrifice adapters remain. Their typed
binding migration is explicitly a later slice. No untyped target replacement
was introduced. `Effect` remains an alias for `ResolvingEffect`.

## Oracle clause coverage

Each row's action is `effects[0] -> Sacrifice(This)`. The shared migration test
asserts the source survives state-based actions, its trigger uses the stack,
and a response that removes its triggering condition does not cancel sacrifice.

| Card | Trigger expression / Oracle clause | Behavioral coverage |
| --- | --- | --- |
| Goblins of the Flarg | Controls a Dwarf, including itself if its types change | `TestStateSacrificeMigration/Goblins_of_the_Flarg`, `TestComposableStateSacrifice` |
| Vodalian Knights | Controls no Islands | `TestStateSacrificeMigration/Vodalian_Knights`, existing Vodalian Knights restriction tests |
| Giant Shark | Controls no Islands | `TestStateSacrificeMigration/Giant_Shark`, `TestGiantShark` |
| Pirate Ship | Controls no Islands | `TestStateSacrificeMigration/Pirate_Ship` |
| Sea Serpent | Controls no Islands | `TestStateSacrificeMigration/Sea_Serpent`, `TestSeaSerpent` |
| Dandân | Controls no Islands | `TestStateSacrificeMigration/Dandân`, existing Dandân attack/land tests |
| Island Fish Jasconius | Controls no Islands | `TestStateSacrificeMigration/Island_Fish_Jasconius` |
| Merchant Ship | Controls no Islands | `TestStateSacrificeMigration/Merchant_Ship` |
| Serendib Djinn | Controls no lands | `TestStateSacrificeMigration/Serendib_Djinn` |
| Drop of Honey | No creatures on battlefield | `TestStateSacrificeMigration/Drop_of_Honey` |
| Mana Vortex | No lands on battlefield | `TestStateSacrificeMigration/Mana_Vortex`, `TestManaVortex` |
| Jihad | Chosen opponent has no nontoken permanent of chosen current color | `TestStateSacrificeMigration/Jihad`, `TestJihad`, `TestJihadCurrentColorAndTokenCondition`, snapshot tests |

## Remaining package boundary and validation work

Migrated expressions compile through `pkg/mage/dsl`. The files that still use
engine imports keep them for unrelated legacy abilities. Existing DSL exports
of mutable engine types are not a completed package boundary. Target and saved
bindings, all-zone identity adapters, canonical action families, continuous
modifications, typed event patterns, spell/activation builders, replacement
constructors, and generator conversion remain later slices.

Validation currently traverses migrated nodes, ordinary effect-bearing
abilities, triggered modes, pipelines, composites, and the legacy loop/branch
containers used by this slice. It does not claim definite-assignment analysis
for future bindings or inspect arbitrary legacy callbacks. Those capabilities
must arrive with their families. Value errors for migrated nodes are detected
before games start. Compile-fail fixtures use `go/types` and test one invalid
category per fixture.

## Source-sacrifice caller inventory

| File | Card or engine caller |
| --- | --- |
| `cards/arabian/creatures.go` | Dandân |
| `cards/arabian/creatures.go` | Island Fish Jasconius |
| `cards/arabian/creatures.go` | Merchant Ship |
| `cards/arabian/creatures.go` | Serendib Djinn |
| `cards/arabian/enchantments.go` | Drop of Honey |
| `cards/arabian/enchantments.go` | Jihad |
| `cards/fallen_empires/creatures.go` | engine adapter / regression |
| `cards/fallen_empires/creatures.go` | Deep Spawn |
| `cards/fallen_empires/creatures.go` | Mindstab Thrull |
| `cards/fourthedition/creatures.go` | Ball Lightning |
| `cards/jumpstart/creatures.go` | Departed Deckhand |
| `cards/jumpstart/enchantments.go` | Curious Obsession |
| `cards/jumpstart/lands.go` | Rupture Spire |
| `cards/legends/creatures.go` | Time Elemental |
| `cards/legends/enchantments.go` | Cocoon |
| `cards/limited/enchantments.go` | Pestilence |
| `cards/thedark/artifacts.go` | Runesword |
| `pkg/mage/effect_pay_mana_unless.go` | engine adapter / regression |
| `pkg/mage/engine_audit_test.go` | engine adapter / regression |
| `pkg/mage/gametest/triggers_target_damage_player_test.go` | engine adapter / regression |

Pestilence retains its legacy resolution-only condition in this slice. Its
Oracle intervening-if also requires a creation-time check; that known behavior
belongs to the event/intervening-if slice and is not claimed as corrected here.

Files with migrated expressions that still directly import `pkg/mage`:

- `cards/jumpstart/creatures.go`

- `cards/jumpstart/enchantments.go`

- `cards/jumpstart/lands.go`

- `cards/legends/creatures.go`

- `cards/limited/creatures.go`

- `cards/thedark/artifacts.go`

- `cards/thedark/creatures.go`

- `cards/thedark/enchantments.go`

## Verification

Passed after implementation:

- Focused DSL, state-trigger, source-lifetime, registration, and compile-fail tests.
- `go test ./pkg/mage/... -short -count=1`.
- All seven affected card-package suites.
- `go test ./... -short -count=1`.
- `go build ./...` and `go vet ./...`.
- `GOLANGCI_LINT_CACHE=/tmp/mage-dsl-lint-cache make lint`: zero issues.

The lint cache uses a writable temporary directory. Long simulations were not run.
