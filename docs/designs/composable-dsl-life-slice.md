# Controller-life action slice

Status: complete. This is the controller-life family within later action slice 2;
other action and binding families remain open in the implementation inventory.
Rules reference: repository Comprehensive Rules, effective February 27, 2026,
CR 119 (life), 608 (resolution), and 614 (replacement effects).

## Removed exports and semantic parity

| Removed export | Canonical card expression | Executor |
| --- | --- | --- |
| `mage.GainLife(n)` | `dsl.GainLife(dsl.You, dsl.Fixed(n))` | Existing target-life executor and `Game.PlayerGainLife` |
| `GainLifeStep(n)` | `dsl.GainLife(dsl.You, dsl.Fixed(n))` | Same executor in any resolving composition |
| `GainLifeAmount(value)` | `dsl.GainLife(dsl.You, value)` | Same executor; evaluate value once at resolution |
| `mage.LoseLife(n)` | `dsl.LoseLife(dsl.You, dsl.Fixed(n))` | Existing target-life executor and `Game.PlayerLoseLife` |
| `LoseLifeAmount(value)` | `dsl.LoseLife(dsl.You, value)` | Same executor; evaluate value once at resolution |

`You` is the stored resolving controller. Source control changes and positional
player targets do not change that recipient. These instructions need no source
LKI. Values still read the original source, controller, targets, X, and event
amount. Recipient selection must not replace the targets used by a value.
Absent recipients and nonpositive values have no effect. The definition stores
no mutable resolution data; clones use their own game state.

Fixed positive gains retain `LifeGain` metadata. Dynamic values report no fixed
amount and are never evaluated with a nil game during inspection. The text
includes amount and recipient; inspection retains subject and amount children.
Existing target-life replacement and event behavior remains in use, including
Lich's replacement and life-loss events. General replacement-result event
accounting is outside this slice; this migration does not certify that broader
legacy behavior.

Three adapter defects were demonstrated by failing tests before correction:
metadata inspection evaluated dynamic values with a nil game; recipient
selection overwrote value targets; zero gain emitted a life-gain trigger.
The new adapter resolves amounts before passing a fixed amount to the executor
and stops before execution for nonpositive amounts.

## Clause and composition evidence

| Contract or clause | Expression | Regression |
| --- | --- | --- |
| You gain/lose life; source controller and targets differ | `GainLife(You, ...)`, `LoseLife(You, ...)` | `TestComposableLifeControllerAndClone` |
| Amount depends on resolution state | Dynamic amount, original target context | `TestComposableLifeValueContext`, `TestComposableLifeDynamicMetadata` |
| Gain zero or less | Positive-amount guard | `TestComposableLifeNonpositive` |
| Replaced gain draws cards; normal gain triggers once | Existing Lich replacement and gain event | `TestComposableLifeGainReplacement`, `TestLich` |
| Modified life gain | Existing modifier path | `TestAddLifeGainModifier_DoubleLifegain` |
| Absent recipient; invalid definitions | Subject and amount edges | `TestComposableLifeAbsentAndValidation` |
| Direct, sequence, loop, mode, trigger, activation | Shared resolving instruction | `TestTypedLifeComposition`, existing migrated modal, trigger, and activation tests |
| Clone isolation | Shared definition, separate game state | `TestComposableLifeControllerAndClone` |
| Permanent/set cannot be life recipient | Sealed `PlayerRef` | `life_permanent.go`, `life_collection.go` in compile-fail suite |

## Production caller inventory

The table lists every migrated production card caller. Only the life expression
changes; surrounding Oracle clauses, targets, costs, and trigger conditions stay
in their existing executors. Each listed package's complete tests pass. The
full registry test validates the migrated definitions. These are migration
parity claims, not a new audit of unrelated card clauses.

| File | Cards / factories |
| --- | --- |
| `cards/antiquities/creatures.go` | Onulet |
| `cards/antiquities/enchantments.go` | Powerleech |
| `cards/arabian/creatures.go` | Merchant Ship; El-Hajjâj |
| `cards/custom/modal.go` | Modal Test Artifact |
| `cards/custom/wraithbloom.go` | Wraithbloom Cultivator |
| `cards/fallen_empires/artifacts.go` | Balm of Restoration |
| `cards/jumpstart/artifacts.go` | Bubbling Cauldron |
| `cards/jumpstart/creatures.go` | Angel of Mercy; Bulwark Giant; Blood Host; Phyrexian Gargantua; Phyrexian Rager; Sangromancer; Tithebearer Giant; Ravenous Baloth; Sylvan Brushstrider; Thragtusk; Gingerbrute |
| `cards/jumpstart/enchantments.go` | Path of Bravery; Primeval Bounty |
| `cards/jumpstart/spells.go` | Aerial Assault; Agonizing Syphon; Battlefield Promotion; Douse in Gloom; Take Heart; Tandem Tactics; Time to Feed |
| `cards/legends/creatures.go` | Axelrod Gunnarson; Sol'kanar the Swamp King |
| `cards/legends/enchantments.go` | Lifeblood; Spirit Link |
| `cards/legends/spells.go` | Glyph of Life |
| `cards/limited/artifacts.go` | Soul Net |
| `cards/limited/creatures.go` | Personal Incarnation |
| `cards/limited/enchantments.go` | Lifetap; Living Artifact |
| `cards/secretsofstrixhaven/artifacts.go` | Ark of Hunger; Potioner's Trove |
| `cards/secretsofstrixhaven/creatures.go` | Stone Docent; Adventurous Eater // Have a Bite; Moseo, Vein's New Dean; Mindful Biomancer; Pestbrood Sloth; Shopkeeper's Bane; Bogwater Lumaret; Essenceknit Scholar; Lluwen, Exchange Student // Pest Friend; Pterafractyl; Tam, Observant Sequencer // Deep Sight; Teacher's Pest |
| `cards/secretsofstrixhaven/spells.go` | Grapple with Death; Oracle's Restoration; Pull from the Grave; Pursue the Past; Send in the Pest |
| `cards/thedark/artifacts.go` | Fountain of Youth |
| `cards/thedark/creatures.go` | Grave Robbers |
| `cards/thedark/enchantments.go` | Dark Heart of the Wood |

## Additional in-tree caller migrations

Engine factories and test definitions also use the typed replacements. This
list accounts for every additional file that called a removed export; tests
retain their existing assertions.

- `cards/astral/artifacts_test.go` (1 call).
- `pkg/mage/action_payment_test.go` (1 call).
- `pkg/mage/autotap_test.go` (6 calls).
- `pkg/mage/basic_land_type_effect_test.go` (1 call).
- `pkg/mage/card.go` (1 call).
- `pkg/mage/composable_binding_test.go` (1 call).
- `pkg/mage/composable_dsl_test.go` (3 calls).
- `pkg/mage/engine_audit_test.go` (2 calls).
- `pkg/mage/gametest/action_test.go` (1 call).
- `pkg/mage/gametest/attackers_declared_test.go` (2 calls).
- `pkg/mage/gametest/cast_alt_test.go` (4 calls).
- `pkg/mage/gametest/cost_reduction_test.go` (2 calls).
- `pkg/mage/gametest/dsl_test.go` (1 call).
- `pkg/mage/gametest/either_cost_test.go` (1 call).
- `pkg/mage/gametest/fight_test.go` (1 call).
- `pkg/mage/gametest/grant_trigger_attached_test.go` (1 call).
- `pkg/mage/gametest/graveyard_activated_test.go` (1 call).
- `pkg/mage/gametest/lifegain_modifier_test.go` (1 call).
- `pkg/mage/gametest/lki_test.go` (1 call).
- `pkg/mage/gametest/main_phase_event_test.go` (2 calls).
- `pkg/mage/gametest/mechanics_test.go` (2 calls).
- `pkg/mage/gametest/per_turn_trackers_test.go` (3 calls).
- `pkg/mage/gametest/rules_ch05_turn_structure_test.go` (4 calls).
- `pkg/mage/gametest/rules_ch06a_casting_resolving_test.go` (11 calls).
- `pkg/mage/gametest/rules_ch07a_keyword_actions_test.go` (1 call).
- `pkg/mage/gametest/triggers_life_discard_sacrifice_test.go` (2 calls).
- `pkg/mage/gametest/triggers_target_damage_player_test.go` (4 calls).
- `pkg/mage/gametest/turn_beginning_test.go` (2 calls).
- `pkg/mage/gametest/turn_combat_attackers_test.go` (1 call).
- `pkg/mage/gametest/turn_combat_damage_test.go` (2 calls).
- `pkg/mage/gametest/turn_combat_test.go` (1 call).
- `pkg/mage/gametest/turn_end_test.go` (2 calls).
- `pkg/mage/gametest/turn_general_test.go` (1 call).
- `pkg/mage/gametest/turn_main_test.go` (1 call).
- `pkg/mage/interactive/ai/search/search_test.go` (2 calls).
- `pkg/mage/keyword_sos_test.go` (1 call).
- `pkg/mage/mana_system_test.go` (1 call).
- `pkg/mage/modal_test.go` (3 calls).
- `pkg/mage/predefined_tokens.go` (1 call).
- `pkg/mage/random_target_test.go` (7 calls).
- `pkg/mage/spell_payment_test.go` (5 calls).
- `pkg/mage/subtype_ability_effects_test.go` (3 calls).
- `pkg/mage/tapped_for_mana_test.go` (1 call).
- `pkg/mage/target_spell_or_permanent_test.go` (1 call).

No callers or implementations of the removed exports remain. `GainLifeTarget`,
`TargetPlayerLoseLife`, and the string-variable helpers remain because their
positional, event-player, and saved-value callers need separate exact bindings.
Direct player methods are engine mutations, not these removed DSL constructors.


## Verification for the controller-life slice

- Focused life behavior, public composition, and compile-fail tests passed.
- All ten affected card-package suites passed.
- `go test ./pkg/mage/... -short -count=1` passed.
- `go test ./... -count=1` passed, including the long AI simulations.
- `go build ./...` and `go vet ./...` passed.
- `make lint` passed with zero issues. The sandbox prevented some optional
  cache writes; checks still completed successfully.
- `git diff --check` passed.
