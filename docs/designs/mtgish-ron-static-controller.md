# RON controlled static creature layers

Capability: P1-STATIC-CONTROLLER. Complete for the context below.

## Selection and fresh evidence

Initial user changes to `.gitignore` and `mtgish-ron-execution.md` are preserved.
Baseline compiler tests and `make build` passed. Strict loading and AST inventory
both exited 1 for collection rejection: 859 compiled records, 1,586 rejected
records, 251 accepted unique names and 735 rejected unique names. The collection
contains 2,445 records and 986 unique names. Inventory includes descendants of
rejected expressions and does not certify support.

Complete candidate records and nested forms were inspected. Reprints count once.

| Candidate | Distinct names | Estimated full-card compilation gain | Other blockers | Downstream benefit | Estimated effort |
| --- | --- | --- | --- | --- | --- |
| Controlled static creature layers, selected | 15; listed below | 5: Castle, Fortified Area, Goblin War Drums, Jacques le Vert, Orcish Oriflamme | Source exclusion, supertype/banding, payments, named filters | Source-relative static subjects; later controlled source-exclusion layers and separate target/collection adapters | Small compiler adapter, medium lifecycle tests |
| Variable temporary P/T (`AdjustPTX`) | 6: Berserk, Blood Lust, Great Defender, Howl from Beyond, Kry Shield, Subdue | 2: Great Defender, Howl from Beyond | Conditions, prevention, casting timing, delayed triggers, controller filters, ActionPermanent scope | Captured numeric values, mana-value and power readers | Medium value/capture adapters and tests |
| Optional fixed counter payments | 4: Force Spike, Power Sink, Thrull Wizard, Vodalian Mage | 2: Force Spike, Vodalian Mage | Alternative payments for Thrull Wizard; X, player actions, mana loss and collections for Power Sink | Payer, decision and independent payment-result scopes | Medium/large compiler and transactional decision tests |

Selected static names: Adventurers' Guildhouse, Arcades Sabboth, Castle,
Cathedral of Serra, Fortified Area, Goblin War Drums, Jacques le Vert, Kobold
Drill Sergeant, Kobold Overlord, Kobold Taskmaster, Mountain Stronghold, Orcish
Oriflamme, Rohgahh of Kher Keep, Seafarer's Quay, Unholy Citadel.

This slice has the largest credible gain for the adapter effort and uses exact
existing engine services. The prior ledger estimated four unlocks; fresh record
inspection also identified Orcish Oriflamme: its IsAttacking predicate is already
supported. Actual gain matches this run's five-card estimate. Two engine contract
defects found by lifecycle tests increased effort; both are prerequisites for
correct execution of this shape and were fixed within this slice.

## Signatures, contexts and reuse

`EachPermanentLayerEffect/2` accepts a creature-restricted filter and a list of
existing fixed `AdjustPT(integer, integer)/2` and basic `AddAbility([symbols])/1`
modifications. The new filter shape is
`ControlledByAPlayer(SinglePlayer(You))/1`, alone within a creature conjunction
or below nonempty recursive `And([filters])/1`. Repeated identical controller
conjuncts have the same meaning. Existing source-independent filters, including
Or groups without controller predicates, retain their behavior.

Only this static context gains controller support. The complete filter must
require creatures. Controller predicates below Or, other payer/reference kinds,
malformed selectors, tuple You(), unbound targets/loop references and empty And
are rejected at compilation. Target filters, Aura-host filters, loops and
controller-reference families remain unsupported where previously unsupported.
No generic source-dependent predicate or fabricated binding is introduced.

RON owns shape validation, conjunction decomposition and immutable filter plans.
Factories construct fresh continuous effects. You is lowered through the source
of each engine effect, not a captured controller UUID or mutable compiler slot.
Engine reuse: BoostControlledCreatures and GrantKeywordToControlled, shared
filters, printed-ability registration, control layer reconciliation, dependency
ordering, phasing, clone ownership and continuous-effect lifecycle.

Static membership reads the current source controller and current permanent
characteristics. It includes a matching creature source. Later entries qualify;
tap status, combat membership, color and control changes alter membership.
Source removal, phasing and ability removal stop the applicable effect. Returning
or phasing in causes current static rules to apply again. Effects use layer 6
for keywords and layer 7c for fixed P/T. Control changes apply in layer 2 first.
No announcement, target identity, cost, LKI value, replacement, simultaneous
zone-operation or resolving-action contract is added. Those services retain
existing regression coverage; static rules do not need those bindings.

Two engine prerequisites were established:

- `controlContinuousEffect.isActive` distinguishes an absent target from a
  phased target. A phased target suspends application without expiring an
  otherwise valid duration. Cleanup still expires turn-limited effects;
  conditions still expire when their required objects or facts become absent.
  Clone latches remain separate. Rules: CR 702.26b/d/f and 611.2b.
- Existing targeted/global `RemoveAllAbilities` services now carry typed removal
  membership through `abilityRemovalContinuousEffect`. Printed static effects
  depend on matching removal effects in the same layer. The dependency sorter
  orders removal first, and execution checks whether the printed effect is still
  active. Both printed and direct removers are recognized. Target-removal expiry
  state is deep-cloned. This covers these explicit removal services; it does not
  claim a complete dependency detector for all continuous-effect forms.
  Rules: CR 613.1f, 613.8a/b and 611.3a/b.

## TDD and card-to-test mapping

The real-card tests first failed with ControlledByAPlayer compiler errors for
all five selected cards. Declaration tests also exposed acceptance of empty
nested And. After lowering, all five lifecycle tests failed when control expired
on phasing. A new engine test reproduced that defect independently. Keyword
source-removal tests then failed for Fortified Area and Goblin War Drums before
the dependency fix. Inventory tests failed on an unknown controller operand role
before role propagation was added. All of these tests pass after implementation.
Initial test setup used a nonexistent Attacking field; it was corrected to use
the engine combat service before confirming the intended compiler failures.

| Test | Card/rules evidence |
| --- | --- |
| TestRONControlledStaticCards | All five real cards: P/T and keyword membership, opposing creatures excluded, owner distinct from controller, source and recipient control changes, matching source included for Jacques, clone isolation, source phase-out/in, departure and return |
| TestRONControlledStaticMembership | Castle: late entry, tap/untap, suppression/restoration; Jacques: green membership and layer-5 color change; Orcish: attacker membership and combat removal; Fortified: Wall restriction, fixed bonus and banding; Goblin War Drums: menace; both grants: late entries, recipient phase-out/in and source ability removal |
| TestRONControlledStaticNestedConjunctionAndInstances | Synthetic real engine execution: nested And, ordinary Or, mixed P/T and flying, validation, separate factory instances and accumulated effects after control change |
| TestRONControlledStaticRejectsInvalidFilters | Malformed/wrong-scope controller operands, unsupported controller disjunction and empty conjunction |
| TestRONControlledStaticInventoryRoles | Controller selector/reference roles in nested AST inspection; no support claim |
| TestControlEffectSurvivesTargetPhasing | Engine duration retention, clone isolation, phase-in and cleanup expiry while phased |
| TestConditionalControlStillExpiresDuringPhasing | Engine conditional source and target requirements remain enforced |
| TestAbilityRemovalOrdersBeforePrintedKeywordGrant | Engine targeted/global removal dependency, unrelated abilities preserved, restoration and clone expiry isolation |
| TestPrintedAbilityRemovalOrdersBeforePrintedGrant | Printed global remover, source phasing, source return and restoration |

Existing combat tests retain Banding and Menace rule evidence. New tests exercise
RON application of those keywords, not a new combat executor. Existing RON and
engine regression suites remain required. These five cards gain direct gameplay
evidence; compilation and gameplay coverage remain separate. No collection-wide
all-rules gameplay certification is claimed.

## Measurements and remaining gaps

Final strict loading: **874 compiled records, 1,571 rejected records, 256 accepted
unique names and 730 rejected unique names**. Gain: **15 records and five unique
names**, exactly the estimate. No previously accepted record regressed. No
remaining record changed its first diagnostic. Strict loading returns no partial
registry. Inventory assigns player_filter and player_reference roles below
ControlledByAPlayer; all syntax support labels remain unverified.

The ten other static names remain blocked: Arcades Sabboth and Rohgahh by MayCost;
three Kobold lords by Other(ThisPermanent); the five legendary lands by
IsSupertype and BandsWithOthers. Rohgahh also requires IsNamed and payment/list
rules; Arcades also requires IsntAttacking. First errors do not enumerate all
remaining gaps.

Leading first errors: PermanentRuleEffect 110, AsPermanentEnters 90,
ActivatedWithModifiers 61, instant/sorcery CastEffect 53, rule If 47,
ChooseADamageSource 46, MayCost 43, WhenAPlayerCastsASpell 43, Targeted_Modal 42,
HostController upkeep reference 35, ControlledByAPlayer in other contexts 28.
These are compiler/engine-context gaps, not build failures. No source defect was
corrected. All 12 source hashes still match the vocabulary snapshot, so no
snapshot refresh is required. Previously reported Gaea's Liege/Fastbond concerns
remain open.

Next ranked candidates:

1. Static source exclusion: Other(ThisPermanent) for the three Kobold lords,
   using the controlled static subject prerequisite and existing engine services.
2. Variable temporary P/T: Howl from Beyond and Great Defender; then shared
   numeric readers for the other four records.
3. Optional fixed counter payments: Force Spike and Vodalian Mage, then
   alternatives and X/result/collection work.

Full validation command results are recorded in the progress ledger. No runtime
cutover, Go-card removal, obsolete-generator retirement or DSL-function deletion
occurred. Their gates remain unmet. No whole-repository removal audit is claimed.
