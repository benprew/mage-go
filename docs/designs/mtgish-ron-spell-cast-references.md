# RON spell-cast event references

Capability P6-SPELL-CAST-REFERENCES, 2026-10-04: complete for the contexts below.
This is one slice; other cast-event forms remain unsupported.

## Selection and semantic audit

Baseline focused RON/CLI tests, engine short tests and build pass. Strict loader
rejects the collection: 983 compiled records, 1,462 rejected, 294 accepted names
of 986. Initial user changes to `.gitignore` and the execution design are preserved.
The fresh full AST inventory includes nested forms under unsupported parents.
Reprints count once; first diagnostics are not estimates of full-card support.

| Candidate | Distinct affected names | Estimated full unlocks | Other blockers and downstream use | Effort |
| --- | --- | ---: | --- | --- |
| Cast-event spell/caster references | Presence of the Master, Nether Void; downstream In the Eye of Chaos, Invoke Prejudice, Mana Vortex | 2 | Chaos needs variable optional payment; Invoke needs opponent/color-sharing filters and variable payment; Vortex needs stack source, self-spell filter, sacrifice payment and more events. Event identity enables their counter/payment bodies. | Small adapter change; medium identity/payment gameplay matrix. Existing engine bindings suffice. |
| Temporary animation | Mishra's Factory, Jade Statue, Thelonite Druid | 1 directly; up to 3 with prerequisites | Factory needs temporary set P/T, added types/subtype. Statue also needs combat timing and end-of-combat duration. Druid also needs selected sacrifice cost and collection animation. Shared temporary modifiers also serve Sorceress Queen after its Other target-filter adapter. | Medium to large; layers, duration, membership, identity and cost tests. |
| Own-turn activation restriction | Disrupting Scepter, Instill Energy, Rag Man, Wand of Ith, Gwendlyn Di Corci, Nebuchadnezzar | 1 | Energy also needs attack permission and once-per-turn limit; Rag Man/Gwendlyn need random discard; Wand needs random reveal, saved results, life payments and conditions; Nebuchadnezzar needs X cost, name choice and random selection. Enables exact turn restrictions for these activations. | Small adapter plus timing/rollback matrix; later capabilities still needed. |

Cast references give the largest credible immediate multi-card gain for this
boundary. No engine contract is missing. Animation and activation modifiers are
separate slices. Selection, reuse and acceptance were stated before editing.

Complete distinct records were read for all three candidate groups. All
`WhenAPlayerCastsASpell` records were read, including the seven accepted life
cards (Crystal Rod, Iron Star, Ivory Cup, Throne of Bone, Wooden Sphere,
Urza's Chalice, Sol'kanar the Swamp King) and Citanul Druid/Verduran Enchantress.
The latter two still need opponent or controller filters plus counters or
optional drawing. The inventory finds seven cast-event shapes, including
Mana Vortex beneath unsupported FromStack. The AnyPlayer shapes are AnySpell,
IsCardtype and IsColor; existing nonempty And/Or spell filters also remain valid.
Nested event refs outside this event do not gain a producer.

Local `data/LEG.json` agrees with both new full records: Presence counters an
Enchantment spell cast by either player; Nether counters any cast spell unless
its caster pays three mana. Nether's World supertype is retained. No RON source
changed. All source hashes match the vocabulary snapshot; no refresh is needed.
The Gaea's Liege/Fastbond observations remain unresolved source concerns.

## Supported contract and reuse

Exact declaration: `TriggerA(WhenAPlayerCastsASpell(AnyPlayer, filter), actions)`.
AnyPlayer is a symbol; filter uses the existing `compileSpellFilter` shapes:
symbol AnySpell; IsCardtype(type)/1; IsColor(color)/1; And(list)/1 or Or(list)/1
with at least one recursively supported operand. Other player/event forms remain
errors. Actions use the existing ActionList/1 or single-group Targeted/2 shape.

Symbol `Trigger_ThatSpell` is a typed spell ref only inside that event's actions.
It is supported where `compileSpellRef` is used: CounterSpell/1 and
ControllerOfSpell/1 player references. Symbol `Trigger_ThatPlayer` is a typed
player ref where `compilePlayerRef` is used: PlayerAction/2 and PlayerMayCost/2
with the already supported actions/costs. DrawACard, fixed DrawNumberCards,
DiscardACard and fixed DiscardNumberCards retain their existing shapes.
Fixed PayMana and adjacent If/Unless(CostWasPaid, nonempty-list) reuse existing
payment lowering. The event does not enable variable amounts, mana value queries,
other costs, new conditions, event damage recipients or life actions. Explicit
life actions through an event player or event-spell controller stay rejected
because that old adapter still acts on You. Bare life actions retain You.

`castEventPlan` is an immutable lexical identity with spell/caster names.
`spellReferencePlan.event` and `playerReferencePlan.event` point to that identity.
`actionPlan.buildWithEvents(map[*castEventPlan]castEventBindings)` adds an event
environment while retaining fresh target maps. Each trigger factory creates fresh
`BindSpell`/`BindPlayer` declarations; no game state lives in a compiler plan.
Single target groups and event declarations coexist without replacing each other.
Wrong types, tuple-shaped refs, absent producers and cross-rule reads fail at
compile time. Engine validation also rejects reads without the event producer.

Lowering reuses `WhenEvent(SpellCast().WithSpell(...).WithCaster(...), Sequence(...))`,
then the existing `SpellCastMatchesCardFilters` condition. That condition is the
same service previously used by WheneverSpellCastTrigger, so accepted life cards
retain their existing filter behavior. `FromSpell` reads the captured stack UUID;
`FromPlayer` reads the event caster; `ControllerOfSpell` reads current controller.
Target bindings remain distinct, and event refs do not become targets.

Engine-owned contracts: pending event capture/placement and APNAP, resolution
binding maps, Counter/CounterStackObject and uncounterable rules, source LKI and
incarnations, game cloning, stack copying, player decisions, transactional optional
mana payment and mana-source selection, world state actions, draw/discard services.
No engine service, Go definition, runtime registry, fallback or DSL export changes.
CR 603.2/603.3/603.5 governs detection, placement and decisions; 608.2c/d/g/k
covers order, choices, mana abilities and untargeted identity. CR 701.6a covers
countering. Phasing/source departure cannot erase a queued event. No new permanent
LKI query, replacement ordering or simultaneous-operation service is introduced.
The pre-existing engine regressions continue to verify those services.

## Acceptance and gameplay evidence

Tests were written first. After fixing a test helper name, the initial focused
run failed on unbound Trigger_ThatSpell/Trigger_ThatPlayer. The independent-cast
fixture uses Dark Ritual as the second cast because a second artifact cannot be
cast at sorcery speed while triggers are stacked. Later test compilation fixes
used the actual card CloneFrom API to change characteristics. Expectations were
not weakened. The old invalid CounterSpell event case now uses a malformed tuple;
positive cases prove the newly supported symbol. Exact inventory counts changed.

| Card/context | Test mapping and behavior |
| --- | --- |
| Presence of the Master, Nether Void | TestRONSpellCastReferenceRealCards: all records' rules; engine validation; both casters; matching/nonmatching filters; no announcement-time effect; no targets; refusal, insufficient mana, three land mana sources; failed casts; departure/return, source control and phasing before/after cast; uncounterable and absent spells; owner graveyard; copy and recast identity; changed spell type/controller; independent clone outcomes; copied trigger keeps payer. |
| Synthetic independent factories/events | TestRONSpellCastReferenceIndependentEvents: two factory instances, two pending casts, correct caster draws and exact counters. |
| Synthetic declarations/target coexistence | TestRONSpellCastReferenceDeclarationAndTarget: fresh factory validation; engine producer validation; ordinary spell target plus event spell reference; event caster draw; both exact stack objects countered. |
| Synthetic caster/current-controller reads | TestRONSpellCastReferenceCasterAndCurrentController: changed stack controller; captured caster draws and current controller discards. |
| Invalid shapes/scopes | TestRONSpellCastReferenceInvalid: spell/activation/noncast trigger reads, wrong reference type, tuple reference, escaped player binding, unsupported explicit event life forms. |
| Seven accepted cast-life cards | Retained TestRONSpellCastLife* matrix: filter combinations, life rules, independent payments, clone/source lifecycle, failed casts and cast-copy distinctions. |

Synthetic tests do not count as new inventory card coverage. Two newly accepted
cards have gameplay evidence for their full RON records; this is not collection
completion. Global engine_validated and behavior_verified totals remain unmeasured.

## Measurements, checks and next work

Actual gain equals estimate: two names, two records. Accepted records: 983 -> 985;
rejected: 1,462 -> 1,460. Accepted unique names: 294 -> 296; rejected names: 692
-> 690. No accepted-record regression. In the Eye of Chaos's diagnostic changes
from missing caster reference to unsupported optional PayManaX. No data defect or
environment failure is part of this slice. Final checks are recorded in the ledger.

Ranked next candidates: temporary animation (Factory and Sorceress Queen can
share set-P/T modifiers, with Queen's Other target filter and Statue/Druid's
other prerequisites still required); shared
activation modifiers (audit once-per-turn and timing combinations for multi-card
unlocks); variable optional mana payments (Chaos directly; Invoke after filters).
Reassess complete records and nested inventory before selecting another slice.
Runtime cutover, complete card gameplay, Go-card/generator deletion and DSL cleanup
remain gated. No second slice was started and no unused-DSL audit is claimed.
