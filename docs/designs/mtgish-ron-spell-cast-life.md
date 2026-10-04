# RON spell-cast life triggers

Capability P6-SPELL-CAST-LIFE, completed 2026-10-04. This slice adds compiler
adapters for existing engine contracts. It does not change engine execution.

## Candidate selection

Fresh AST inventory and strict loader: 2,445 records, 986 unique names,
949 accepted records, 1,496 rejected records, 285 accepted unique names.
Baseline RON tests, engine short tests and build pass. Collection rejection is
expected and does not indicate a build failure. User changes to `.gitignore`
and `mtgish-ron-execution.md` are excluded.

| Candidate | Distinct affected names | Estimated complete unlocks | Other blockers and downstream benefit | Effort |
| --- | --- | --- | --- | --- |
| Any-player spell-cast life triggers and fixed paid bodies | Crystal Rod, Iron Star, Ivory Cup, Throne of Bone, Wooden Sphere, Urza's Chalice, Sol'kanar the Swamp King | 7 names / 32 records | All seven full records require only event and paid-body adapters. Event references later enable Presence of the Master, Nether Void and In the Eye of Chaos; fixed paid bodies enable Urza's Miter after its event/intervening condition | Small adapters, substantial event/payment/identity test matrix |
| Temporary animation | Mishra's Factory, Jade Statue, Thelonite Druid | Up to 3 names | SetPT, added types/subtypes and captured collection animation; Jade Statue needs combat-only timing and end-of-combat duration; Druid needs creature sacrifice. Enables later temporary characteristic changes | Medium to large layer, duration, timing and payment tests |
| Fixed selected discard actions | Jalum Tome, Bazaar of Baghdad; partial users Disrupting Scepter, Recall, Mindstab Thrull | 2 names | Scepter needs controller-turn activation restrictions; Recall needs typed discard results, graveyard return and self-exile; Thrull needs combat event/player and sacrifice payment. Enables hidden-zone actions and later discard results | Small adapters, medium choice/empty-hand/replacement/sequence tests |

The seven-card event/payment slice gives the largest credible gain for the
adapter effort. The initial commentary identified Jalum Tome's immediate gain;
the complete-record audit adds Bazaar of Baghdad, without changing the ranking.
Reprints count once. First-error frequency (43 spell-cast records) includes
unsupported caster scopes and event references and is not the unlock estimate.

All candidate records were read in full. The AST inventory includes descendants
of unsupported parents. Spell-cast records also include Verduran Enchantress
(controller-only caster and MayAction), Citanul Druid (opponent caster and
counters), Invoke Prejudice (opponent caster, shared-color condition, event
references and variable payment), Mana Vortex (FromStack, self-spell scope,
sacrifice, upkeep and state events), Presence of the Master (event spell),
Nether Void (event spell/player) and In the Eye of Chaos (event references and
variable payment). All remain rejected. The full Rohgahh of Kher Keep record
was also read: its now-supported paid continuation exposes CreatePermanentsList,
name filters and player choice/control operations. No additional card is
claimed complete.

## Shapes, dependencies and reuse boundary

- `TriggerA(WhenAPlayerCastsASpell(AnyPlayer, filter), actions)` has exactly two
  trigger operands. AnyPlayer must be a symbol. Existing pure spell filters
  support AnySpell, IsColor(color), IsCardtype(type), and nonempty And/Or lists.
  Lowering uses `WheneverSpellCastTrigger(effect, false, filter)`.
- Adjacent `MayCost(PayMana(costs))` or
  `PlayerMayCost(player, PayMana(costs))` followed by
  `If(CostWasPaid, [actions])` lowers to `IfPaid`. The continuation must be
  nonempty and CostWasPaid must be a symbol. Existing fixed-mana cost checks
  reject X and nonmana costs. Existing typed player references apply.
- The paired plan consumes its own result. Nesting and successive pairs build
  separate effects and costs per factory call. CostWasPaid cannot escape its
  adjacent continuation. The existing Unless form retains its behavior.
- Paid bodies work in supported spell, activation, source entry/death and
  controller-upkeep action contexts. The new event form introduces no event
  bindings. Other caster scopes and Trigger_ThatSpell/Trigger_ThatPlayer remain
  declaration errors. Broader If conditions and multiple continuations are
  separate shapes.
- Inventory assigns the event operands player_filter/spell_filter roles and If
  operands condition/action roles. These roles describe syntax; they do not
  mark every condition or event reference as supported.

RON owns spelling, shape validation, scope and per-factory lowering. Engine
services own cast events, pending triggers/APNAP, event-time filtering, stacked
controller capture, source incarnations, phasing, pure mana affordability and
transactional payment, player decisions, life gain and clone state. No mutable
payment result is shared and no global last-action slot is used.

CR 601.2 describes successful casting; CR 603.2/603.3a establish event-time
triggering and its controller; CR 603.5 makes payment a resolution choice.
Copies put directly on the stack do not cast (CR 707.10). The local Oracle
records in data/LEA.json, ATQ.json and LEG.json agree with every selected rule.
No source data correction or engine prerequisite is needed.

## Card-to-test mappings

All tests are in `pkg/mage/dsl/ron/spell_cast_life_test.go`.

| Card or contract | Evidence |
| --- | --- |
| Crystal Rod, Iron Star, Ivory Cup, Throne of Bone, Wooden Sphere, Urza's Chalice | TestRONSpellCastLifeRealCards: full real RON records, both casters, matching/nonmatching events, pay/refuse/inability, pooled and battlefield mana, payment after announcement, failed casts, copies, countered spell, source departure/return/control/phasing, clone isolation |
| Sol'kanar the Swamp King | Same real-card test for mandatory life gain; TestRONSpellCastLifeFilterCombinations checks swampwalk with/without defending Swamp |
| Throne of Bone instances, multicolor spell | TestRONSpellCastLifeIndependentInstances: two sources per controller, four distinct triggers, APNAP through engine placement, one payer unable to pay second body |
| AnySpell, And, Or, multicolor/type event filters | TestRONSpellCastLifeFilterCombinations |
| Zero-cost refusal/payment, nested/successive paid branches, fresh factories, explicit targeted payer and draw recipient, unavailable colored payment with unused partial mana source | TestRONSpellCastLifePaymentContexts |
| Activation/upkeep/entry paid bodies | TestRONSpellCastLifeSharedContexts |
| Invalid arities, tuple AnyPlayer, unsupported caster/self-spell filters, unbound event refs, orphan/nonadjacent/empty/invalid paid bodies, X/nonmana payment | TestRONSpellCastLifeInvalid |
| Event/payment child roles | TestRONSpellCastLifeInventory |

The upkeep invalid-declaration regression now uses a second orphan If after a
valid pair. Its original single paid pair is supported and has positive
upkeep gameplay evidence. Inventory assertions change to fresh exact counts.
Tests were not weakened to hide an unsupported execution path.

Tests initially failed on unsupported spell-cast events and missing roles.
Fixture corrections supplied engine priority placement, supported legal spells,
the active caster for permanent spells, and explicit clone payment decisions.
A nested-payment fixture matched ManaCostGeneric with an overly broad string
check and incorrectly left a Forest untapped; the fixture now checks the exact
colored token. Existing explicit player life actions reject targeted scope;
the typed payer test uses the supported targeted draw body instead. This
separate life-action compiler gap is not changed by this slice.

## Measurements and validation

Actual gain equals the estimate: 7 names and 32 records. The five colored
artifacts each contribute six reprints; Urza's Chalice and Sol'kanar each one.
Accepted records: **981/2,445**. Rejected records: **1,464**. Accepted unique
names: **292/986**. Rejected unique names: **694**. No previously accepted
record regresses. Twelve remaining diagnostics change: six Enchantress reprints
and Citanul Druid/Invoke Prejudice reach caster-filter errors; Nether Void,
Presence of the Master and In the Eye of Chaos reach unbound event references;
Rohgahh reaches the unsupported permanent-list producer.

Compilation and gameplay evidence remain separate. Global engine_validated and
behavior_verified totals remain null. The seven names above have direct RON
behavior evidence for their full rule lists; this is not collection-wide proof.
All RON file hashes match the vocabulary snapshot. No snapshot refresh is
needed. Gaea's Liege and Fastbond source observations remain outside this slice.

Validation results are recorded in the progress ledger. Complete loading,
collection-wide gameplay, runtime embedding/startup, Go-card and generator
removal, and unused-DSL removal gates remain open. No deletion or partial
production registry is introduced.

Next ranking: (1) typed spell-cast event references (Presence of the Master and
Nether Void, with In the Eye of Chaos needing variable payments); (2) fixed
selected discard for Jalum Tome and Bazaar of Baghdad; (3) temporary animation
for Mishra's Factory, Jade Statue and Thelonite Druid. Reassess dependencies,
complete records and effort next run. No next slice was started.
