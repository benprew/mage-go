# RON fixed upkeep payments and controller damage

Capability IDs: P4-IMPLICIT-FIXED-MANA and its P5-CONTROLLER-DAMAGE prerequisite.
Status: supported shapes below. These families remain partial.

## Selection from fresh evidence

The strict loader and AST inventory started at 919 compiled records, 1,526
rejected records, 271 compiled unique names and 715 rejected unique names.
Baseline engine short tests and build passed. The collection rejection is an
expected compiler gap, not a build or environment failure. Complete candidate
records and nested expressions were read from RON, with reprints counted once.

| Candidate | Distinct immediate names | Estimated complete-card unlocks | Other blockers and downstream benefit | Effort |
| --- | --- | ---: | --- | --- |
| Fixed controller upkeep payments | Force of Nature, Phantasmal Forces, Junún Efreet, Sunken City, Palladia-Mors, Vaevictis Asmadi | 6 | Force needs the controller damage recipient adapter. Shared payments also advance Conversion, Demonic Hordes, Arcades Sabboth, Breeding Pit, Nicol Bolas, Forethought Amulet and the two Chants; their other rules remain blocked. | Small adapter; medium timing, payer, identity and payment tests. Existing engine services are sufficient. |
| Temporary animation | Mishra's Factory, Jade Statue, Thelonite Druid | Up to 3 | Temporary artifact/creature type and subtype composition, base P/T, collection animation, combat-only activation, end-of-combat duration and matching sacrifice cost. Enables other temporary type changes. | Large combined engine/compiler and layer, duration, timing and cost test scope. |
| Characteristic readers and branches | Blood Lust, Berserk, Kry Shield, Subdue | 1 | Blood Lust needs toughness comparison, conditional branches and subtraction. Berserk retains casting/delayed-trigger gaps; Kry Shield and Subdue retain prevention gaps. Enables other numeric effects and branches. | Medium compiler scope and resolution value tests. |

The fixed payment pair has the largest credible immediate gain for the effort.
The selection contract and acceptance cases were stated before implementation.
The controller damage prerequisite gives three additional full-card unlocks:
Elves of Deep Shadow, Juzám Djinn and Serendib Efreet. The initial estimate
counted the six payment cards and missed these direct prerequisite users. Their
complete records were then read, and gameplay tests were added before declaring
coverage. Actual gain is nine unique cards and 20 records, rather than six names.
No second slice was started.

## Exact compiler contract and reuse boundary

In an action sequence, the adjacent pair
`MayCost(PayMana(list))/1`, `Unless(CostWasPaid, list)/2` compiles to one
`mage.UnlessPaid(mage.You, mage.Costs(...), mage.Sequence(...))`.
`PayMana/1` uses existing fixed generic and colored mana syntax, including an
empty list. The unpaid list must contain at least one supported action.
Nested unpaid lists can contain independent payment pairs. Spell, activation
and supported trigger action sequences use the same adapter. `MayCost` always
selects the stored resolving controller; it does not inherit another player
selector. `PlayerMayCost/2` retains its existing explicit-player contract.
Factories create fresh costs and effects. No payment result is stored globally.

`PermanentDealsDamage(ThisPermanent, value, Player(You))/3` now compiles to
`mage.DealDamageFrom(mage.This, mage.You, mage.ValueAtResolution(value))`.
The source requires a permanent-card scope. Values use the existing Integer/1
and bound ValueX compiler. Player/1 here accepts only the symbol You. Other
player references, tuple You(), other damage sources and explicit player
recipients for SpellDealsDamage remain rejected. Existing raw target-recipient
forms retain their prior behavior.

RON owns these spellings, arity checks and lexical rejection. The engine owns
cost transactions and rollback, mana sources, controller capture, source
incarnations, LKI, damage prevention and events. No engine API was added, no
Go-card fallback was used, and no runtime registry or source asset changed.
Inventory now assigns cost operands to MayCost/1 and player-reference operands
to Player/1 in recipient context. Syntax roles still report support unverified.

Relevant local rules: CR 118 (complete payments), 603.3a (trigger controller),
608.2c–d (sequence and resolution choices), 608.2g (mana abilities during
resolution), 608.2h and 113.7a (source information and ability independence).
Sacrifice still requires the resolving player to control the permanent. If
control changes after triggering, the original player receives the payment
choice but cannot sacrifice the new controller's permanent. Damage still
applies to the original trigger controller. Departure or phasing does not
cancel a queued trigger; sacrifice cannot follow a returned incarnation.

## Gameplay and declaration evidence

All tests are in `pkg/mage/dsl/ron/upkeep_payment_test.go` and compile complete
repository RON records. Synthetic fixtures test shared contracts only.

| Cards or contract | Tests |
| --- | --- |
| Force of Nature, Phantasmal Forces, Junún Efreet, Sunken City, Palladia-Mors, Vaevictis Asmadi | TestRONUpkeepPaymentRealCards: own/opponent upkeep, no announcement payment, resolution payment/refusal/inability, colored affordability, mana sources and rollback, departure/return/phasing, later control change, prevention |
| Sunken City, all creature keywords, all three Vaevictis pumps | TestRONUpkeepPaymentOtherRules: both players' blue creatures, boost ends on unpaid sacrifice, flying/trample, each colored activation |
| Independent Force of Nature instances and games | TestRONUpkeepPaymentInstancesAndClones: separate payer/body, clone state isolation, independent resolution. Cloned BasePlayers use their default decline choice, so live payment and clone refusal have different correct outcomes. |
| Local/nested results, zero payment, fresh factory effects | TestRONUpkeepPaymentZeroAndNested |
| Juzám Djinn, Serendib Efreet, Elves of Deep Shadow | TestRONUpkeepDamageAdditionalCards: upkeep controller/timing, flying, tap cost, immediate mana-ability execution and self damage |
| Force of Nature source LKI and event identity | TestRONUpkeepDamageLKI: departure, return under the other controller, phasing, control change; damage event keeps the original source incarnation and recipient |
| Invalid arity/cost/result/binding/continuation declarations | TestRONUpkeepPaymentInvalid and TestRONUpkeepDamageInvalid |
| Nested semantic reporting roles | TestRONUpkeepPaymentInventory |

The original optional-payment tests retain target legality, stack copies and
payment transactions. This slice adds no new simultaneous operation or
replacement-ordering contract. The Force test proves use of existing prevention.
Local Oracle data in LEA/ARN/DRK/LEG agrees with the nine unlocked records. No
source defect was corrected. Gaea's Liege and Fastbond observations remain open.

TDD: after correcting test fixture API names, the real-record tests failed on
unsupported MayCost and reporting failed on the missing cost role. Implementation
made those contracts pass. Later test failures were incorrect sacrifice-control,
clone decision and ability-index expectations, corrected using engine behavior.
The full RON package then failed only on its two old exact inventory counts;
those counts were updated without changing gameplay assertions.

## Remaining records and next dependencies

Fresh AST inspection walked nested forms despite rejected ancestors. The fixed
unpaid payment forms also occur in Conversion, Demonic Hordes, Hasran Ogress,
Dance of Many, Stasis, Energy Flux, Breeding Pit, Goblin Flotilla, Thelon's Chant,
Tourach's Chant, Arcades Sabboth, Chromium, Cosmic Horror, Forethought Amulet,
Nicol Bolas and The Tabernacle at Pendrell Vale. Their blockers include land-type
layers, opponent sacrifice choices, attack/cast/entry/damage events, tokens and
links, player rules, static trigger grants, IsntAttacking, Rampage, MustCost
results and replacement pipelines. No full-card coverage is claimed for them.
Paid If/IfElse forms remain rejected, including Brass Man, Farmstead, Crystal
Rod, Iron Star, Ivory Cup, Mana Vault, Soul Net, Throne of Bone, Wooden Sphere,
Island Fish Jasconius, Spitting Slug, Imprison, Puppet Master, Rohgahh of Kher
Keep, Tablet of Epityr, Urza's Chalice and Urza's Miter. Nonmana and X payments
remain unsupported. Later diagnostics now expose these exact remaining gaps.

Other direct controller-damage users retain unrelated blockers: Fastbond,
Lord of the Pit, Bottle of Suleiman, Erg Raiders, Serendib Djinn, Electric Eel,
Wormwood Treefolk, Mishra's War Machine, Psychic Allergy, Elder Spawn, Primordial
Ooze, Voodoo Doll and Yawgmoth Demon. No source condition was changed to hide a
blocker. First-error movement is not evidence that their full rules execute.

Next ranked candidates, to reassess in the next run:

1. Shared spell-cast events plus adjacent paid fixed mana branches: Crystal Rod,
   Iron Star, Ivory Cup, Throne of Bone, Wooden Sphere and Urza's Chalice offer
   six credible unique unlocks if the event/payer contract is complete.
2. Temporary animation for Mishra's Factory, Jade Statue and Thelonite Druid;
   three possible unlocks with the dependencies listed above.
3. Shared numeric filters and conditional branches for Blood Lust and the three
   downstream numeric cards; one immediate unlock.

Singleton adapters remain deferred. Complete collection support, gameplay
mapping, runtime cutover, native startup, Go-card/generator deletion and unused
DSL removal gates remain unmet. No caller cleanup audit or removal is claimed.
