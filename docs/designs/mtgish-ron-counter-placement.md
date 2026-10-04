# RON single-permanent counter placement

Capability: P5-COUNTER-PLACEMENT. Completed 2026-10-04.

## Selection evidence

Fresh RON/CLI tests and `make build` passed. Strict and semantic-inventory
loaders both exited 1: 1,015 compiled and 1,430 rejected records; 319 accepted
and 667 rejected unique names. This was collection rejection, not a build
failure. The positioned parser inventory walks nested expressions. Complete
records were read, including descendants hidden by earlier errors. There are
63 unique names containing PutCounters, not 63 supported cards. Reprints count once.

| Candidate | Distinct affected names | Estimated full unlocks | Other blockers | Downstream capability | Effort |
| --- | --- | ---: | --- | --- | --- |
| Single-permanent counter placement | Armor Thrull, Fungal Bloom, Dwarven Weaponsmith directly; all 63 names below | 3 | Opponent cast filtering blocks Citanul; removal/results block five Batteries; other contexts, values, entry, choices and layers block the rest | Typed actions for target/source/host and triggers; concrete Battery and spore-counter prerequisites | Small adapter, medium gameplay/identity/replacement tests; unusual counters already exist |
| Saved sacrifice characteristics | Diamond Valley, Life Chisel, Priest of Yawgmoth; Homarid Spawning Bed downstream | 2 | Priest also needs dynamic mana; Bed needs token creation and mana-value result | Per-payment characteristics for independent activations, mana and tokens | Medium typed payment/result adapter; substantial LKI, rollback and copy/clone tests |
| Variable optional mana payment | In the Eye of Chaos, Invoke Prejudice | 1 | Invoke also needs Opponent and ExceptFor/SharesAColorWithAPermanent with controlled creature selection | Captured spell mana-value payment for two cast triggers | Medium adapter, substantial payment/choice/copy/clone tests |

Counter placement won on credible immediate multi-card gain for small adapter
effort and downstream reuse. The prior Citanul estimate was reassessed: its
Opponent cast filter remains unsupported. Selection, contract, reuse boundary
and acceptance cases were stated before edits. No selected engine prerequisite,
source correction or environment repair was needed.

## Exact supported contract

`compileCounterPlacement(Node, scope) (effectFactory, error)` accepts the
resolving action `PutCounters/1` with a list of exactly one instruction:

- `ACounterOfTypeOnPermanent/2(counter, permanentRef)` places one counter.
- `NumberCountersOfTypeOnPermanent/3(value, counter, permanentRef)` places the
  resolved number. Values are existing Integer/1 with an integer literal or
  bound ValueX. Nonpositive values place nothing. No new X binding is introduced.

These work inside existing supported spell, activation, trigger and optional
payment continuation action lists. They are not costs, static modifications or
entry operations. ThisPermanent requires permanent-source scope, HostPermanent
requires Aura scope, and Ref_TargetPermanent requires the same action plan's
typed declaration. Event, numbered-target, loop, result, player and collection
references remain rejected here.

`compilePermanentCounterType(Node) (core.CounterType, error)` validates before
factory construction. Supported named permanent counters are Carrion, Charge,
Corpse, Credit, Cube, Doom, Dream, Glyph, Hatchling, Hunger, Intervention,
Javelin, Matrix, Mire, Net, Pin, Pupa, Scream, Sleep, Spore, Storage, Tide, Time,
Vitality and Wind, each with the RON Counter suffix. Supported PTCounter/2
literal pairs are +1/+1, -1/-1, +1/+0, +1/+2, -2/-2, +0/-1, +0/-2, +0/+1,
+2/+0 and +0/+2. Signed zero parses as zero. Unknown kinds, poison/player
instructions, unsupported coordinates and nonliteral coordinates are errors.
The +2/+2 counter in Soul Exchange needs a new engine counter kind; it is not
mapped to another kind and the card remains rejected.

Empty or multiple instruction lists remain errors: multiple objects/kinds need
a simultaneous-operation contract before lowering. Dynamic characteristic,
history, event and result values, removal/restrictions, placement-as-payment,
entry counters and conditional layer rules remain unsupported. Strict loading
never installs a partial collection registry.

Lowering uses `mage.AddCountersTo(subject.build(bindings), counter,
mage.ValueAtResolution(amount))`. Targets, actions and abilities are fresh per
factory; counter/value plans are immutable. Subjects use existing typed instance
bindings. No global result, card-name branch, Go-card fallback, new DSL export
or game-time declaration panic was added. Inventory supplies counter_instruction,
permanent_counter_type, permanent_reference, value and integer_literal roles
beneath these forms. Syntax stays unverified and rejected descendants are walked.

## Reuse and rules

Read pkg/mage/doc.go, RON API, engine ability-instance/payment contracts, counter
enum/primitives, placement action and replacement pipeline. Reused services:
AddCountersTo, ValueAtResolution, typed references/target legality, source and
attachment LKI, transactional costs, ability/spell copies, clones, P/T layers,
cleanup, AddCountersWithReplacement and affected-player replacement rechecks.
No engine API or execution service changed.

CR 122.1/1a establish kinds and P/T; 122.2 and 400.7 establish zone identity.
Source LKI retains a sacrificed ability's source but does not put counters on a
departed object. Host references use the current attachment at resolution when
the original Aura remains, or its saved attachment after departure. They cannot
follow a returned Aura/host. Phased subjects are unavailable; a phase cycle
retains incarnation. CR 602.2 and existing transactions establish rollback.
CR 616.1/1e/1f establish affected-controller choice and rechecks. Only one packet
on one permanent is supported; no new batch, entry or removal contract is claimed.

## Complete-record and nested audit

This is the parser inventory's complete 63-name PutCounters set. Remaining
blockers include nested forms rather than only first loader errors.

| Distinct names | Remaining blockers |
| --- | --- |
| Armor Thrull; Fungal Bloom; Dwarven Weaponsmith | None; complete RON records validate and have all-rule gameplay tests |
| Black Mana Battery; Blue Mana Battery; Green Mana Battery; Red Mana Battery; White Mana Battery | Any-number removal cost, typed removed amount and dynamic mana |
| Citanul Druid | Opponent cast filter |
| Ashnod's Transmogrant | Permanent-duration type addition |
| Feral Thallid; Thorn Thallid | Fixed counter-removal activation cost |
| Elvish Farmer; Thallid; Thallid Devourer | Fixed removal cost and tokens; Farmer's sacrifice and Devourer's bonus are supported |
| Spore Flower | Fixed removal cost and temporary combat-damage prevention |
| Ebon Praetor | Removal and sacrificed-subtype payment condition |
| Living Armor | ManaValueOfPermanent resolving counter amount |
| Dwarven Armorer | ChooseAnAction |
| City of Shadows | Exile cost, counter count and dynamic mana |
| Armageddon Clock | Draw-step event, counter count, each-player damage, removal and other-player activation/upkeep modifiers |
| Cyclone | Per-counter mana payment, counter count and multiple-recipient damage |
| Cyclopean Tomb | Counter-bound land-type duration, graveyard event, delayed trigger and historical removal selection |
| Fungusaur | Damage event and event permanent |
| Living Artifact | Player-damage event/amount and optional removal payment |
| Rock Hydra | Entry counters, prevention/replacement, counter condition/removal and future prevention |
| Scavenging Ghoul; Khabál Ghoul | Any-player end-step and death count; Scavenging Ghoul also needs removal cost |
| Sengir Vampire | Matching death trigger and source damage history |
| Unstable Mutation | Host-controller upkeep reference |
| Fasting | Counter condition, draw-step replacement and draw event |
| Marsh Viper; Pit Scorpion; Serpent Generator | Damage-to-player events and player poison counters; Generator also needs tokens |
| Necropolis | Graveyard exile cost/result and exiled card mana value |
| Bottomless Vault; Dwarven Hold; Hollow Trees; Icatian Store; Sand Silos | Tapped entry, optional untap rule, intervening-if, removal/payment amount and dynamic mana |
| Delif's Cube | Controller-relative target, attack/unblocked delayed event, combat rule and removal cost |
| Homarid; Tidal Influence | Entry, counter conditions, state trigger and removal; Influence also needs name/cast restrictions and conditional global layers |
| Icatian Moneychanger | Entry, entering-permanent reference and counter-count life |
| Soul Exchange | Additional exile casting cost/result, graveyard target/placement, paid subtype condition, returned object and missing +2/+2 engine counter |
| Thelon's Chant; Tourach's Chant | Player entry event, event player, optional chosen-permanent counter payment and damage branch |
| Tourach's Gate | Aura controller filter, removal/condition, host tap cost/state restriction |
| Cocoon | Aura controller filter, counter condition, untap rule, forced removal payment and permanent keyword layer |
| Glyph of Delusion | Multiple targets, combat history, power value and granted counter-dependent untap rule/removal trigger |
| Infinite Authority | Combat event union, toughness filter, event references, future triggers/payment/permanent layers |
| Lesser Werewolf | Combat-relative targets, power condition and declare-blockers timing |
| Life Matrix | Permanent granted activation and removal cost |
| Osai Vultures | Intervening-if end-step/death history and fixed removal cost |
| Primordial Ooze | Attack rule, saved counter-count X and dynamic optional mana |
| Rasputin Dreamweaver | Entry, removal cost, prevention, intervening-if and maximum-counter rule |
| Spirit Shackle | Host-tap event and captured event permanent |
| Takklemaggot | Host-controller upkeep/death, dead controller, choices/results, graveyard Aura return and granted trigger |
| Tetravus | Entry, optional removal/exile results, tokens and linked created objects |
| Triassic Egg | Modes, optional hand placement and counter-count activation condition |
| Venarian Gold | Counter conditions/untap rule, host-controller upkeep and removal; bound-X placement is supported |
| Voodoo Doll | Intervening-if end step, counter count and X payment/modifier |
| Whirling Dervish | Intervening-if any-player end step and damage history |

No source defect was corrected. All 12 source hashes match the vocabulary
snapshot; no RON change requires a refresh. Known Gaea's Liege/Fastbond questions
remain open. Missing +2/+2 support is an engine gap, not a source defect.

## Gameplay mapping

| Card/context | Tests |
| --- | --- |
| Armor Thrull; Fungal Bloom; Dwarven Weaponsmith (ATQ/3ED), all rules | TestRONCounterPlacementRealCards: validation, tap/mana/source or matching sacrifice, upkeep timing, delayed placement, opposing targets, source return, target return/phasing/control/type legality, pending clones, P/T and cleanup persistence |
| Armor Thrull; Dwarven Weaponsmith; context errors | TestRONCounterPlacementPaymentAndContexts: unpayable tap/sacrifice, missing artifact rollback, self-target sacrificed/returned, cost/static rejection, cross-rule target and spell/permanent mismatch |
| Source, Aura host, upkeep trigger | TestRONCounterPlacementSourceAndHost: fixed three counters, source/host identity, phase cycle/control, resolution attachment, departed Aura attachment LKI, returned Aura/host and pending clones |
| Typed targets/copies/replacements | TestRONCounterPlacementCopiesAndReplacement: independent factories, interleaved targets, another-controller copy, applicable doublers once, controller filter, pending clone and later mutation isolation |
| Replacement order/prevention | TestRONCounterPlacementReplacementOrder: target controller chooses add-then-double or double-then-add; source controller does not choose; prevented placement permits the next life action |
| Amounts/payment | TestRONCounterPlacementAmountsAndRollback: negative/zero/positive literals, mana/tap rollback, spell X, failed target pays nothing, copied spell retains X |
| P/T kinds | TestRONCounterPlacementSignedPT: -2/-2, -1/-1, signed-zero -0/-1 and -0/-2, +1/+0, +0/+1, +2/+0 and +0/+2; +1/+1 and +1/+2 covered by real cards/copies |
| Declarations/inventory | TestRONCounterPlacementInvalid; TestRONCounterPlacementInventory: kind/arity/list size, unknown/malformed kinds/coordinates, values/player instructions/scope and nested role reporting |

Initial gameplay tests failed on the missing PutCounters adapter. After lowering,
the noncreature Fungus test exposed an incorrect test expectation: its subtype
no longer qualifies after type removal. Expected placement was corrected to zero.
Two fixture build errors were corrected to use core events and the actual count
type. Existing inventory count assertions now use the measured baseline; no
rejection or gameplay assertion was weakened.

## Measurements and next work

Estimated and actual gain agree: three unique names, four records. Strict loader:
1,015 -> 1,019/2,445 compiled; 1,430 -> 1,426 rejected. Accepted unique names:
319 -> 322; rejected: 667 -> 664. No accepted record regressed. Strict and full
inventory issues match. Nodes/signatures remain 72,938/8,528; unknown roles fall
17,801 -> 17,378. Global engine_validated/behavior_verified stay null. Three
additional cards have complete-record gameplay evidence; compilation and global
behavior coverage remain separate.

Required checks and lint review are in the progress ledger. Strict loading still
rejects atomically. Startup, Go-card definitions/generators and DSL exports retain
callers. Removal gates stay closed; no deletion/unused-DSL audit is claimed.

Ranked next candidates to reassess: (1) fixed removal activation costs, directly
Feral Thallid/Thorn Thallid and prerequisites for spore-token cards; (2) typed
saved sacrifice toughness, directly Diamond Valley/Life Chisel, with shared
results for Priest/Bed; (3) any-number removal with typed payment amount and
dynamic mana, concretely the five Batteries. Variable optional mana has one
direct unlock. Full loading/gameplay, native/runtime cutover, Go-card/generator
removal and unused-DSL cleanup remain open. This run stops after this slice.
