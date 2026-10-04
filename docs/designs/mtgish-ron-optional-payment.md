# RON optional fixed mana payments

An action list can contain an adjacent pair:

```text
PlayerMayCost(player, PayMana([fixed mana symbols]))
Unless(CostWasPaid, [actions])
```

The compiler builds one `mage.UnlessPaid` effect for each pair. The result is
local to that effect; it is not stored in a shared last-payment slot. Unpaid
actions can contain independent nested pairs. Each factory builds fresh costs,
effects and target bindings. Ordinary actions retain their written order.

The player can be `You`, a declared `Ref_TargetPlayer`,
`ControllerOfSpell(Ref_TargetSpell)`, or `ControllerOfTargetSpell` in a declared
single-spell target scope. Both spell-controller forms read the current stack
controller at resolution through `mage.ControllerOfSpell`. They follow the
stack object's identity and the copy's own target frame, not its card's owner
or another cast of the same card.

Payment occurs during resolution. Decline or inability to pay runs the unpaid
body; an accepted complete payment skips it. Available mana abilities can pay
the cost. The engine's transaction prevents partial payment. Empty mana lists
still require a choice to pay zero. Existing target validation prevents an
illegal target from receiving a payment offer. Counter restrictions apply even
if payment fails. A source that leaves or phases out does not stop its stacked
ability. Relevant rules: CR 113.7a and 608.2b–d, 608.2g.

Declarations reject missing/nonadjacent continuations, tuple `CostWasPaid()`,
standalone or reused payment conditions, empty unpaid bodies, unbound/wrong-role
player or spell references, variable mana costs, alternative costs and nonmana
costs. `MayCost`, `PayManaX`, paid branches, loop payment bindings, event payer
references and general conditionals remain unsupported.

Inventory assigns the player and cost roles to `PlayerMayCost`, condition and
action roles to `Unless`, and spell-reference roles to `ControllerOfSpell`.
These reporting roles do not imply execution support for rejected ancestors.

| Cards or contract | Tests in `pkg/mage/dsl/ron/optional_payment_test.go` |
| --- | --- |
| Force Spike, Vodalian Mage | `TestRONOptionalPaymentRealCards`: announcement costs, resolution-only offers, accepted/declined/unaffordable payments, mana sources, source departure/phasing, illegal targets, uncounterable spells, current controller |
| Stack identity, copies, clones | `TestRONOptionalPaymentCopiesClonesAndIdentity` |
| Local and nested results; fresh factory effects | `TestRONOptionalPaymentIndependentContinuations` |
| Zero payment and unavailable colored payment | `TestRONOptionalPaymentZeroCostAndRollback` |
| Declaration boundaries | `TestRONOptionalPaymentInvalid` |
| Reporting operand roles | `TestRONOptionalPaymentInventoryRoles` |

Local Oracle records in `data/LEG.json` and `data/FEM.json` agree with the
complete selected RON records. No RON data or engine implementation changes
are required. The production registry remains unchanged.
