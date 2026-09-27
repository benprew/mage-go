# Card Author Composable DSL Reference

This document is the authoritative, comprehensive reference for the composable card Domain-Specific Language (DSL) in `mage-go`.

Cards import `pkg/mage/dsl` (and `pkg/mage/core` for enums/types). All card definitions, abilities, costs, targets, and effects are built using the typed combinators exported by `dsl`.

Typed token creation uses `DefineToken(name, options...)` with `TokenTypes`, `TokenSubtypes`, `TokenColors`, `TokenPT`, `TokenKeywords`, and `TokenAbilities`. Creature tokens require `TokenPT`; other tokens omit it. Omit `TokenColors` for a colorless token. `CreateTokenCopies(player, count, definition)` evaluates the player and count at resolution and enters all copies as one group. A nonpositive count creates no tokens. Each token ability has its own binding scope and cannot read a binding from the creating ability.

---

## Table of Contents

1. [Core Design Principles](#1-core-design-principles)
2. [Card Registration & Lifecycle](#2-card-registration--lifecycle)
3. [Subjects and Selectors](#3-subjects-and-selectors)
4. [Target Declarations](#4-target-declarations)
5. [Pure State Conditions & Filters](#5-pure-state-conditions--filters)
6. [Resolving Effects (Actions & Verbs)](#6-resolving-effects-actions--verbs)
7. [Control Flow, Choices, and Branches](#7-control-flow-choices-and-branches)
8. [Saved Bindings & Snapshots](#8-saved-bindings--snapshots)
9. [Triggered Abilities](#9-triggered-abilities)
10. [Spells, Activations, and Costs](#10-spells-activations-and-costs)
11. [Continuous Effects, Modifications, and Durations](#11-continuous-effects-modifications-and-durations)
12. [Replacements and Prevention](#12-replacements-and-prevention)
13. [Values and Dynamic Calculations](#13-values-and-dynamic-calculations)
14. [Representative Card Implementations](#14-representative-card-implementations)

---

## 1. Core Design Principles

1. **One Concept, One Function**: Functions do not bundle multiple concepts into their names (e.g. use `WhenState(Controls(HasSubType("Dwarf")), Sacrifice(This))` instead of legacy `SacrificeIfControls`).
2. **Compile-Time Rejection over Runtime Panics**: Sealed Go interfaces distinguish roles. You cannot pass a `PlayerRef` to `Destroy`, use a `ResolvingEffect` as a cost, or install a resolving effect as a static rule.
3. **Registration Validation**: Value-dependent constraints (empty sequences, duplicate binding names, unbound variable reads) fail during card registration with clear diagnostics identifying the card and expression path.
4. **Typed Bindings over String Maps**: Target declarations and resolution-time captures use Go variables (`creature := TargetPermanent("creature", IsCreature)`), avoiding string-key dictionaries.
5. **No Universal Object Identity**: Physical card IDs are distinguished from rules objects. `PermanentRef` resolves only the captured battlefield incarnation. Phasing preserves incarnation; zone changes (leave-and-return) create a brand-new object.

---

## 2. Card Registration & Lifecycle

Cards register in `init()` in their set package:

```go
func init() {
    Register("Grizzly Bears", NewGrizzlyBears)
}
```

Card factories construct a card blueprint. Characteristics and abilities are added via builder methods:

```go
func NewGrizzlyBears() Card {
    return NewCreature("Grizzly Bears", "{1}{G}", 2, 2).
        WithSubTypes("Bear")
}
```

For non-creature cards:
* `NewInstant(name, manaCost)`
* `NewSorcery(name, manaCost)`
* `NewEnchantment(name, manaCost)`
* `NewArtifact(name, manaCost)`
* `NewLand(name)`

Abilities are attached with `.WithAbility(...)`, `.WithManaAbility(...)`, or `.WithStaticAbility(...)`.

---

## 3. Subjects and Selectors

Operations accept explicit subjects rather than baking the subject into the verb name:

### Singular Subjects (`PermanentRef`)
* `This`: The permanent or spell hosting the ability. For battlefield actions, resolves the battlefield incarnation.
* `AttachedPermanent`: The permanent to which the source Aura/Equipment is currently attached. If the source left the battlefield, reads the last attachment via Last-Known Information (CR 608.2h).
* `From(binding)`: The permanent captured in a `PermanentBinding` (from a target declaration or earlier `Snapshot`).
* `LowestPower(PermanentSet)`: Selects the permanent with the lowest power among a set.

### Permanent Sets (`PermanentSet`)
* `MatchingPermanents(filter)`: All permanents on the battlefield matching the filter.
* `ControlledPermanents(filter)`: All permanents controlled by the ability controller matching the filter.
* `PermanentsOwnedBy(player PlayerRef, filter PermanentFilter) PermanentSet`: Battlefield permanents owned by the resolved player, regardless of their controllers. Use it in a resolving spell, activation, or trigger body. A zero-value filter selects all permanent types. The set excludes phased-out permanents and reads current battlefield membership when consumed. The player can be `You`, `OwnerOf(...)`, or a declared player target. A missing player yields an empty set.

    ```go
    func ownedArtifactCount() dsl.Card {
        player := dsl.TargetPlayer("player")
        return dsl.NewInstant("Owned artifact count", "{1}{U}",
            dsl.WithAbility(dsl.Spell(
                dsl.GainLife(dsl.You,
                    dsl.CountPermanents(dsl.PermanentsOwnedBy(player, dsl.IsArtifact))),
                dsl.WithTargets(player))))
    }
    ```
* `BlockingOrBlockedBy(permanentRef) PermanentSet`: Creatures currently blocking or blocked by the referenced permanent. The set uses current combat links and battlefield incarnations when an action reads it. A phased-out, returned, or absent creature is excluded. Use it in a resolving body; `GrantEachKeyword` captures its members when `Until` resolves.

    ```go
    func combatWatcher() dsl.Card {
        return dsl.NewCreature("Combat Watcher", "{1}{G}", 2, 4,
            dsl.WithAbility(dsl.WhenEvent(dsl.BlocksOrBecomesBlockedBy(dsl.This),
                dsl.Until(dsl.EndOfTurn,
                    dsl.GrantEachKeyword(dsl.BlockingOrBlockedBy(dsl.This), dsl.FirstStrike)))))
    }
    ```
* `WithoutPermanent(set, ref)`: Excludes one specific permanent (e.g. `WithoutPermanent(MatchingPermanents(HasSubType("Zombie")), This)`).

### Player Selectors (`PlayerRef` and `PlayerSet`)
* `You`: The stored resolving controller of the spell or ability.
* `ActivePlayer`: The player whose turn it currently is.
* `FromPlayer(binding)`: The player captured in a `PlayerBinding` (from target, snapshot, or event pattern).
* `ControllerOf(permanentRef)`: The player who controls the referenced permanent at resolution time.
* `OwnerOf(permanentRef) PlayerRef`: The player who owns the referenced battlefield permanent at resolution time. It accepts one `PermanentRef`, including `This` or a declared permanent target, in a resolving spell, activation, or trigger body. An absent, phased-out, or returned incarnation supplies no player. Use `SnapshotPlayer` when a later instruction must keep the owner after the permanent leaves.

```go
target := dsl.TargetPermanent("permanent", dsl.AnyPermanent)
owner := dsl.BindPlayer("owner")
spell := dsl.Spell(dsl.Sequence(
    dsl.SnapshotPlayer(dsl.OwnerOf(target), owner),
    dsl.GainLife(dsl.FromPlayer(owner), dsl.Fixed(1)),
), dsl.WithTargets(target))
_ = spell
```

* `EachPlayer`: Collection of all players in active-player-first order.
* `OpponentsOf(playerRef)`: All opponents of the specified player (for 2-player, the single opponent).

---

## 4. Target Declarations

Targets are declared as typed bindings and attached to abilities using `WithTargets(...)`:

```go
target := TargetPermanent("creature", IsCreature)

Spell(
    Destroy(target),
    WithTargets(target),
)
```

### Permanent Targets
* `TargetPermanent(label, filter, options...)`: Declares a single permanent target matching the filter.
* `TargetPermanents(label, min, max, filter, options...)`: Declares multiple permanent targets.
* `TargetPermanentsExactlyX(label, filter, options...)`: Declares exactly the announced X as a permanent target set. `TargetPermanentsUpToX(label, filter, options...)` declares zero through X targets. Both return a `PermanentSet` and a target declaration. They work in spells and activations through `WithTargets`; use `WithTargetGroups` or `WithActivationTargetGroups` when another variable group is present. Choose X before targets. The target count stays fixed on a copy, and an illegal target is absent at resolution. The legacy `TargetXPermanents` has a different return type and remains available to unmigrated cards.

```go
lands := dsl.TargetPermanentsExactlyX("lands", dsl.IsLand)
member := dsl.BindPermanent("land")
ability := dsl.Activate(dsl.Costs(dsl.Mana("{X}"), dsl.TapCost(dsl.This)),
    dsl.ForEach(lands, member, dsl.Untap(dsl.From(member))),
    dsl.WithTargets(lands))
_ = dsl.NewArtifact("Example", "", dsl.WithAbility(ability))
```

  For “up to two target creatures,” use `TargetPermanents("creatures", 0, 2, IsCreature)`
  with `WithTargets(creatures)`. Dauntless Onslaught and Tandem Tactics pass this
  collection to `Until(EndOfTurn, ModifyEachPT(creatures, ...))`. Only targets
  that remain legal receive the boost. A cast with zero targets still resolves;
  a cast whose chosen targets all become illegal does not resolve. Tandem
  Tactics gains life after the boost with `Sequence` and `GainLife(You, Fixed(2))`.
  Spell and ability copies keep the recorded number and position of chosen targets
  in this group. A copy that chooses new targets cannot move a choice to a
  different group. `WhenEvent` and `WhenEventModes` can declare multiple
  variable-size groups: the controller chooses each group separately when the
  trigger goes on the stack. `Spell(effect, WithTargetGroups(bindings...))`
  declares independent groups for `Game.CastSpellByNameWithTargetGroups(playerID,
  name, groups, optionalX)`. The cast argument has one UUID slice per declared
  group, including an empty slice for an unchosen optional group. The flat cast
  path rejects this declaration before payment. `Activate(costs, effect,
  WithActivationTargetGroups(bindings...))` uses
  `Game.ActivateAbilityByIndexWithTargetGroups(playerID, permanentID,
  abilityIndex, groups)` for independent activation target groups. The flat
  activation path rejects that declaration before payment. Delayed
  declarations still allow only one variable-size group.

```go
creatures := dsl.TargetPermanents("creatures", 0, 2, dsl.IsCreature)
ability := dsl.Spell(dsl.DestroyEach(creatures), dsl.WithTargets(creatures))
_ = ability
```

For a required opponent target and two independent optional creature targets:

```go
opponent := dsl.TargetPlayer("opponent", dsl.TargetOpponentOnly())
first := dsl.TargetPermanents("first", 0, 1, dsl.IsCreature)
second := dsl.TargetPermanents("second", 0, 1, dsl.IsCreature)
ability := dsl.Spell(dsl.Sequence(
    dsl.LoseLife(opponent, dsl.Fixed(1)),
    dsl.Until(dsl.EndOfTurn, dsl.ModifyEachPT(first, dsl.Fixed(1), dsl.Fixed(1))),
    dsl.Until(dsl.EndOfTurn, dsl.ModifyEachPT(second, dsl.Fixed(-1), dsl.Fixed(-1))),
), dsl.WithTargetGroups(opponent, first, second))
_ = dsl.NewInstant("Example", "", dsl.WithAbility(ability))
```

The cast input `[][]uuid.UUID{{opponentID}, {}, {creatureID}}` chooses only the
second optional creature group. Each group is checked before costs are paid.
Its target positions remain fixed through resolution and copying.

An activation can use the same group shape:

```go
first := dsl.TargetPermanents("first", 0, 1, dsl.IsCreature)
second := dsl.TargetPermanents("second", 0, 1, dsl.IsCreature)
member := dsl.BindPermanent("member")
ability := dsl.Activate(dsl.Costs(dsl.TapCost(dsl.This)),
    dsl.Sequence(dsl.DestroyEach(first),
        dsl.ForEach(second, member, dsl.Tap(dsl.From(member)))),
    dsl.WithActivationTargetGroups(first, second))
_ = dsl.NewArtifact("Example", "", dsl.WithAbility(ability))
```

The activation input `[][]uuid.UUID{{}, {creatureID}}` selects only the second
group. Group counts and legal choices are checked before activation costs are
paid. Copying and resolving the ability keep the group identity and target
incarnation.

For two independent optional targets on a trigger, declare separate bindings:

```go
first := dsl.TargetPermanents("first", 0, 1, dsl.IsCreature)
second := dsl.TargetPermanents("second", 0, 1, dsl.IsCreature)
member := dsl.BindPermanent("member")
ability := dsl.WhenEvent(dsl.Enters(dsl.This),
    dsl.Sequence(
        dsl.DestroyEach(first),
        dsl.ForEach(second, member, dsl.Tap(dsl.From(member))),
    ),
    dsl.WithTargets(first, second),
)
_ = dsl.NewCreature("Example", "", 1, 1, dsl.WithAbility(ability))
```

Each choice keeps its own group at resolution and on a stack copy. A choice
in one group does not fill an empty group. The same permanent can be chosen
once in each group if it is legal for both.

* Target Options:
  * `TargetYouControl()`: Target must be controlled by the ability controller.
  * `TargetOtherThanSource()`: Target must be a different permanent from the source.
  * `TargetAttachedToYouControl()`: Target must be attached to a permanent controlled by the ability controller. This does not restrict the attachment's controller.
  * `TargetYouOwn()`: Target must be owned by the ability controller.
  * `TargetOpponentControls()`: Target must be controlled by an opponent.
  * `TargetOpponentOnly()`: Target must be an opponent (for player targets).
  * `TargetChosenByOpponent()`: The opponent makes the target choice when the ability is put on the stack.

### Player Targets
* `TargetPlayer(label, options...)`: Declares a target player.
* `TargetDamageRecipient(label, filter, options...)`: Declares a target creature or player.

### Spell Targets
* `TargetSpell(label, filters...)`: Declares one spell target on the stack. Card filters restrict legal choices at announcement and resolution.
* `ControllerOfSpell(spell StackObjectRef) PlayerRef`: Reads the current controller of a legal spell reference at resolution. `TargetSpell` supplies the reference. The player is absent if the stack object is gone. It works in resolving effects, including `UnlessPaid` and `IfPaid`; it is not a target declaration.
* `ManaValueOfSpell(spell StackObjectRef) ResolutionValue`: Reads the mana value of a legal spell on the stack at resolution. The announced X contributes once per `{X}` in its mana cost. An absent spell supplies zero. Use `SnapshotNumber` before `Counter` when a later instruction needs the value after the spell leaves the stack. It works in resolving effect bodies, not target eligibility at announcement.
* `Counter(spell)`: Counters the referenced spell if it remains legal and can be countered. A copy disappears from the stack when countered. Declare the binding with `WithTargets(spell)`.

```go
spell := TargetSpell("spell")
ability := Spell(
    UnlessPaid(ControllerOfSpell(spell), Costs(ManaPayment("{1}")), Counter(spell)),
    WithTargets(spell),
)
_ = NewInstant("Example", "{U}", WithAbility(ability))

valueSpell := TargetSpell("value spell")
valueAbility := Spell(
    GainLifeFor(You, ManaValueOfSpell(valueSpell)),
    WithTargets(valueSpell),
)
_ = NewInstant("Value example", "{U}", WithAbility(valueAbility))
```

### Graveyard Card Targets
* `TargetGraveyardCard(label, options...)`: Declares one card target in your graveyard by default. `InAnyGraveyard()` permits any player's graveyard. `GraveyardCardFilter(filter)` restricts its characteristics. Declare the binding with `WithTargets(card)`.
* `ReturnCardToHand(card)`: Moves the bound graveyard card to its owner's hand. A card that leaves and returns to the graveyard is a new object, so the old binding cannot move it.
* `ReturnCardToBattlefield(card, controller)`: Moves the bound graveyard card onto the battlefield under the selected player's control. The card keeps its owner. A stale graveyard binding does nothing.

---

## 5. Pure State Conditions & Filters

State conditions must be **pure functions** of game state (no random choices, mutations, or player prompts):

### State Conditions (`StateCondition`)
* `Controls(filter)`: Ability controller controls at least one permanent matching filter (includes source).
* `ControlsNone(filter)`: Ability controller controls zero permanents matching filter.
* `BattlefieldHasNone(filter)`: Neither player controls a permanent matching filter.
* `ControllerHandSizeIs(size int) StateCondition`: Check the ability controller's current number of cards in hand. It works in `ActivationIf`, `IfState`, `WhenState`, `WithInterveningIf`, and `While`. A missing controller does not match. Registration rejects a negative size.
* `HasUniqueHighestLife() StateCondition`: True when one player has more life than every other player. Ties return false. The condition reads all players at each check and does not depend on the ability controller or source. It works in `ActivationIf`, `IfState`, `WhenState`, `WithInterveningIf`, and `While`.

  ```go
  ability := dsl.Activate(
      dsl.Costs(),
      dsl.Draw(dsl.You, dsl.Fixed(1)),
      dsl.ActivationIf(dsl.HasUniqueHighestLife()),
  )
  _ = ability
  ```
* `SourceHasCounters(counter, min)`: The source permanent has at least `min` counters of the specified type.
* `SourceTappedCondition()` and `SourceUntappedCondition()`: The current battlefield source is tapped or untapped, respectively. Both return false when the source is absent or phased out. They work in `While`, `WhenState`, `WithInterveningIf`, `ActivationIf`, and `IfState` on permanent abilities.
* `PermanentIsTapped(subject PermanentRef)` and `PermanentIsUntapped(subject PermanentRef)`: Check `This` or `AttachedPermanent` at state evaluation. Both return false if the source or selected permanent is absent or phased out. A returned source or attachment is a new object. Use these conditions in `While`, `WhenState`, `WithInterveningIf`, `ActivationIf`, or `IfState` on permanent abilities. Registration rejects target and saved bindings in this state context.
* `PermanentIsAttacking(subject PermanentRef) StateCondition`: Check whether `This` or `AttachedPermanent` is in combat as an attacker now. A creature put onto the battlefield attacking matches, although it did not attack for trigger purposes. An absent, phased-out, or returned object does not match. Use it in `While`, `WhenState`, `WithInterveningIf`, `ActivationIf`, or `IfState` on permanent abilities. Registration rejects targets and saved bindings in this state context.
* `PermanentIsBlocking(subject PermanentRef) StateCondition`: Check whether `This` or `AttachedPermanent` is in combat as a blocker now. A creature put onto the battlefield blocking matches even though it did not block for trigger purposes. An absent, phased-out, removed, or returned object does not match. Use it in `While`, `WhenState`, `WithInterveningIf`, `ActivationIf`, or `IfState` on permanent abilities. Registration rejects targets and saved bindings in this state context.
* `SourceAttackedThisTurnCondition()`: The current battlefield source was declared as an attacker this turn. It returns false before an attack, after the source leaves or phases out, or after it returns as a new object. Use it in `While`, `WhenState`, `WithInterveningIf`, `ActivationIf`, or `IfState` on a permanent ability.
* `SourceAttackedOrBlockedThisTurnCondition() StateCondition`: The current battlefield source was declared as an attacker or blocker this turn. It uses the source incarnation and returns false if the source is absent, phased out, or returned as a new object. Turn cleanup clears the history. Use it in pure state contexts such as `IfState`, `While`, `WhenState`, `WithInterveningIf`, and `ActivationIf` on a permanent ability. It does not mean "this combat".
* `SourceAttackedOrBlockedThisCombatCondition() StateCondition`: The current battlefield source was declared as an attacker or blocker in the current combat. It returns false before a declaration, after the combat phase, for a phased-out or absent source, and after that card returns as a new object. A later combat in the same turn starts with no combat history. Use it in `IfState`, `While`, `WhenState`, `WithInterveningIf`, or `ActivationIf` on a permanent ability. It reads declarations, so a creature put onto the battlefield attacking or blocking does not match.

```go
func handGate() dsl.Card {
    return dsl.NewLand("Hand gate", dsl.WithAbility(dsl.Activate(
        dsl.Costs(dsl.TapCost(dsl.This)),
        dsl.Draw(dsl.You, dsl.Fixed(1)),
        dsl.ActivationIf(dsl.ControllerHandSizeIs(7)),
    )))
}
```

```go
func combatHistoryWatcher() dsl.Card {
    return dsl.NewCreature("Combat History Watcher", "{2}", 2, 2,
        dsl.WithAbility(dsl.At(dsl.EachEndOfCombat,
            dsl.IfState(dsl.SourceAttackedOrBlockedThisCombatCondition(),
                dsl.GainLife(dsl.You, dsl.Fixed(1))))))
}
```

```go
func historyWatcher() dsl.Card {
    return dsl.NewCreature("History Watcher", "{2}", 2, 2,
        dsl.WithAbility(dsl.At(dsl.EachEndStep,
            dsl.IfState(dsl.SourceAttackedOrBlockedThisTurnCondition(),
                dsl.GainLife(dsl.You, dsl.Fixed(1))))))
}
```
* `SourceBlockingCondition()`: The current battlefield source is blocking now. It returns false when the source is absent, phased out, removed from combat, or a new incarnation of the same card. Use it in `While`, `WhenState`, `WithInterveningIf`, `ActivationIf`, or `IfState` on a permanent ability. It does not report whether the source blocked earlier in combat.

```go
var _ = NewCreature("Untapped guardian", "", 1, 1,
    WithAbility(Static(While(
        SourceUntappedCondition(),
        ModifyPT(This, Fixed(0), Fixed(3)),
    ))),
)

var _ = NewAura("Untapped cloak", "",
    WithAbility(Static(While(
        PermanentIsUntapped(AttachedPermanent),
        GrantKeyword(AttachedPermanent, Shroud),
    ))),
)

var _ = NewCreature("Attack watcher", "", 2, 2,
    WithAbility(WhenEvent(Attacks(This),
        IfState(SourceAttackedThisTurnCondition(), GainLife(You, Fixed(1))))),
)

var _ = NewCreature("Attacking guardian", "", 2, 2,
    WithAbility(Static(While(
        PermanentIsAttacking(This),
        ModifyPT(This, Fixed(0), Fixed(2)),
    ))),
)

var _ = NewCreature("Blocking guardian", "", 2, 2,
    WithAbility(Static(While(
        PermanentIsBlocking(This),
        ModifyPT(This, Fixed(0), Fixed(2)),
    ))),
)

var _ = NewAura("Blocking cloak", "",
    WithAbility(Static(While(
        PermanentIsBlocking(AttachedPermanent),
        GrantKeyword(AttachedPermanent, Shroud),
    ))),
)
```
* Logical combinators:
  * `AllConditions(conds...)`: Logical AND.
  * `AnyCondition(conds...)`: Logical OR.
  * `NotCondition(cond)`: Logical NOT.
* `StateConditionFunc(desc, deps, fn)`: Pure read-only escape hatch with declared dependencies.

### Permanent & Card Filters (`PermanentFilter`, `CardFilter`)
* Type filters: `IsCreature`, `IsArtifact`, `IsLand`, `IsEnchantment`, `IsPlaneswalker`, `IsToken`, `IsNonToken`.
* Characteristic filters: `HasSubType("Dwarf")`, `HasColorFilter(Red)`, `HasKeywordFilter(Flying)`.
* `HasPowerEqual(n)` and `HasToughnessEqual(n)`: Match current power or toughness exactly, including continuous effects and counters. Combine both with `IsCreature` for Pendelhaven's 1/1 target; legality is checked at activation and resolution.
* `HasToughnessGTE(n)`: Matches current toughness at least `n`, including continuous effects and counters. Combine with `IsCreature` for creature targets. Valorous Stance uses this filter in its destruction mode; check it at casting and resolution.
* Combinators: `And(filters...)`, `Or(filters...)`, `Not(filter)`.
* `IsAttachedTo(filter)`: Matches attachments whose current, unphased host matches the filter. This does not require an Aura subtype or restrict either controller.

---

## 6. Resolving Effects (Actions & Verbs)

Resolving effects implement `ResolvingEffect` and mutate the game during resolution:

* `GrantLandPlays(PlayerRef, ResolutionValue)`: Adds land plays for the selected player for this turn. Grants accumulate; the amount is read once at resolution. An absent player or nonpositive amount does nothing. This does not play a land, grant access to another zone, or override normal land-play timing.

### Permanent Actions
* `Sacrifice(PermanentRef, ...SacrificeResultOption)`: Stored controller sacrifices the permanent if it is present, unphased, and still controlled by that player. `SacrificeAmountInto(*NumberBinding)` saves 1 after a successful instruction and 0 otherwise. The result is local to this resolution and can be read by later `FromNumber` expressions in a spell, activation, or trigger body. This is a resolving action result, not a cost-payment capture.

```go
package example

import . "github.com/benprew/mage-go/pkg/mage/dsl"

func sacrificeAbility() {
    done := BindNumber("sacrificed")
    ability := Activate(Costs(), Sequence(
        Sacrifice(This, SacrificeAmountInto(done)),
        GainLife(You, FromNumber(done)),
    ))
    _ = ability
}
```
* `Destroy(PermanentRef)`: Destroys the permanent (allows regeneration).
* `DestroyWithoutRegeneration(PermanentRef)`: Destroys the captured permanent without regeneration. The prohibition belongs to this destruction action; it does not change permanent keywords or last-known characteristics. Indestructible still applies.
* `DestroyEach(PermanentSet)`: Simultaneously destroys all permanents in the set. For multiple card types, use one `MatchingPermanents(Or(...))` collection; a permanent with several matching types is destroyed only once.
* `DestroyEachWithoutRegeneration(PermanentSet)`: Simultaneously destroys the set without regeneration. Indestructible still applies; the instruction does not change permanent keywords.
* `Exile(PermanentRef)`: Moves the permanent to the exile zone.
* `Tap(PermanentRef, ...TapResultOption) ResolvingEffect`: Taps the referenced battlefield incarnation. `TapAmountInto(*NumberBinding) TapResultOption` saves 1 when it changes from untapped to tapped, or 0 when it is already tapped, absent, phased out, or a returned incarnation. The option may occur once. The result is local to the resolving spell or ability. Existing `Tap(subject)` calls remain valid.
* `IfPositive(ResolutionValue, ResolvingEffect) ResolvingEffect`: Runs the body only when the value is greater than zero. Use it after an action result in `Sequence`; a binding written only inside its body cannot be read after the branch.
* `IfCardHasType(CardRef, core.CardType, ResolvingEffect) ResolvingEffect` and `IfCardLacksType(CardRef, core.CardType, ResolvingEffect) ResolvingEffect`: Check a saved hand or graveyard card, or a legal graveyard card target, at resolution. Both skip the body when the reference is absent. The body may read a binding written before the branch; a binding written in the body cannot be read after it. These branches work in resolving spell, activation, and trigger bodies, not in pure state or static conditions.

For example, this synthetic spell reveals one drawn card and discards it if it is not a land card:

```go
package example

import (
    dsl "github.com/benprew/mage-go/pkg/mage/dsl"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func drawRevealDiscard() dsl.ResolvingEffect {
    drawn := dsl.BindCard("drawn")
    return dsl.Sequence(
        dsl.Draw(dsl.You, dsl.Fixed(1), dsl.DrawCardInto(drawn)),
        dsl.RevealCard(dsl.FromCard(drawn)),
        dsl.IfCardLacksType(dsl.FromCard(drawn), core.TypeLand,
            dsl.DiscardCard(dsl.FromCard(drawn))),
    )
}
```

For example, this synthetic activation gains life only when the source becomes tapped during resolution:

```go
package example

import dsl "github.com/benprew/mage-go/pkg/mage/dsl"

func tapAbility() {
    done := dsl.BindNumber("tap result")
    ability := dsl.Activate(dsl.Costs(), dsl.Sequence(
        dsl.Tap(dsl.This, dsl.TapAmountInto(done)),
        dsl.IfPositive(dsl.FromNumber(done), dsl.GainLife(dsl.You, dsl.Fixed(2))),
    ))
    _ = ability
}
```
* `Fight(PermanentRef, PermanentRef)`: The two referenced creatures fight if both are present and creatures at resolution. A creature fighting itself deals twice its power to itself.
* `Untap(PermanentRef)`: Untaps the permanent.
* `RemoveFromCombat(PermanentRef)`: Removes the captured permanent from combat. Removing a blocker does not make its attacker unblocked. An absent or different incarnation is unaffected.
* `ReturnToHand(PermanentRef)`: Bounces the permanent to its owner's hand.
* `ReturnEachToHand(PermanentSet) ResolvingEffect`: Returns the selected battlefield incarnations to their owners' hands in one simultaneous instruction. Use it in a spell, activation, or trigger body. It excludes phased-out objects and tokens do not remain in hand. The set is read once at resolution; later returned incarnations are not part of that instruction.

```go
package example

import dsl "github.com/benprew/mage-go/pkg/mage/dsl"

func returnOwnedArtifacts() dsl.Card {
    player := dsl.TargetPlayer("player")
    return dsl.NewInstant("Return owned artifacts", "{1}{U}", dsl.WithAbility(dsl.Spell(
        dsl.ReturnEachToHand(dsl.PermanentsOwnedBy(player, dsl.IsArtifact)),
        dsl.WithTargets(player),
    )))
}
```

For two declared creature targets, use `Spell(Fight(first, second), WithTargets(first, second))`. Each argument reads its own target binding; an absent target prevents the fight.

For an Aura entry trigger, use `WhenEvent(Enters(This), Tap(AttachedPermanent))`.

The engine event `core.EvtCreatureBecomesBlocked` reports an attacking creature's first transition from unblocked to blocked. Its `SourceID` is the attacker. It fires once when one or more blockers are declared for that attacker, or when a creature enters blocking an unblocked attacker. Use `WhenEvent(BecomesBlocked(This), body)` for a battlefield source. This pattern does not select or bind a blocker.

The engine event `core.EvtCreatureBlockedBy` reports each new attacker and blocker pair. `SourceID` is the attacker, `TargetID` is the blocker, and `PlayerID` is the attacker's controller when the pair forms. The event fires after legal blocker declarations settle, and it also fires when a creature enters blocking an attacker. It fires for a new blocker even if that attacker was already blocked. Engine and test-only definitions can use `mage.NewTriggered(core.EvtCreatureBlockedBy, false, body)` with `mage.EventSourceIsSelf{}` on a battlefield source. This engine event has no typed `EventPattern` or blocker binding yet.

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func blockedWatcher() mage.Card {
    return mage.NewCreature("Blocked Watcher", "", 2, 2,
        mage.WithAbility(mage.NewTriggered(core.EvtCreatureBecomesBlocked, false,
            mage.GainLifeFor(mage.You, mage.Fixed(1))).SetConditionData(mage.EventSourceIsSelf{})))
}

func pairWatcher() mage.Card {
    return mage.NewCreature("Pair Watcher", "", 2, 2,
        mage.WithAbility(mage.NewTriggered(core.EvtCreatureBlockedBy, false,
            mage.GainLifeFor(mage.You, mage.Fixed(1))).SetConditionData(mage.EventSourceIsSelf{})))
}
```
The attachment is read at resolution, so moving the Aura changes the affected
permanent. If the Aura has left, its last attachment is used. A returned Aura
or host is a new incarnation and does not redirect the old reference.

### Player Actions
* `Draw(PlayerRef, ResolutionValue, ...DrawResultOption) ResolvingEffect`: Instructs one player to draw cards in a spell, activation, or trigger body. `DrawAmountInto(*NumberBinding) DrawResultOption` saves the number of cards actually drawn by this instruction. It writes zero for an absent player, an empty library, or a fully replaced draw. Read it later in the same `Sequence` with `FromNumber`. `DrawCardInto(*CardBinding) DrawResultOption` saves the card from one completed draw. It requires `Fixed(1)` and cannot be combined with `DrawAmountInto`. `FromCard` refers to that card only during its original hand visit. `DrawEach` does not accept these options.

```go
drawn := dsl.BindNumber("drawn")
ability := dsl.Spell(dsl.Sequence(
    dsl.Draw(dsl.You, dsl.Fixed(2), dsl.DrawAmountInto(drawn)),
    dsl.GainLife(dsl.You, dsl.FromNumber(drawn)),
))
_ = ability
```
For a single-card result in a spell, activation, or trigger body:

```go
func drawnCardSpell() dsl.Card {
    drawnCard := dsl.BindCard("drawn card")
    return dsl.NewInstant("Synthetic Draw", "{U}",
        dsl.Spell(dsl.Sequence(
            dsl.Draw(dsl.You, dsl.Fixed(1), dsl.DrawCardInto(drawnCard)),
            dsl.RevealCard(dsl.FromCard(drawnCard)),
            dsl.DiscardCard(dsl.FromCard(drawnCard)),
        )))
}
```

Hand cards have an engine visit ID. A completed draw creates a new visit;
discarding the card ends it. `FromCard(drawnCard)` is a `CardRef` while the
original card remains in that hand visit. `DiscardCard(CardRef) ResolvingEffect`
discards that exact card if it is still in its owner's hand. It uses effect
discard replacements and does nothing if the reference is absent or in another
zone. Use it in spell, activation, and trigger resolving bodies. Hand reveal
uses `RevealCard(CardRef) ResolvingEffect`. It shows the referenced card to
all players and leaves it in its zone. The engine stores a public snapshot in
`Game.PublicReveals()` in reveal order. This is a record of what was shown,
not an ongoing effect that keeps a card revealed. An absent reference has no
effect. Multi-card draw-result consumers remain open.
* `DrawEach(PlayerSet, ResolutionValue)`: Draws cards for each player in turn order.
* `Discard(PlayerRef, ResolutionValue)`: Player chooses and discards cards from hand.
* `DiscardAtRandom(PlayerRef, ResolutionValue)`: Discards cards at random from hand. Goblin Lore uses `Spell(Sequence(Draw(You, Fixed(4)), DiscardAtRandom(You, Fixed(3))))`; draw first, then select randomly from the entire current hand through the discard replacement and event services.
* `DiscardEntireHand(PlayerRef)`: Discards all cards in the player's hand.
* `Mill(PlayerRef, ResolutionValue)`: Mills cards from the top of the library to the graveyard.
* `GainLife(PlayerRef, ResolutionValue)`: Gains life through the event-aware life gain pipeline.
* `LoseLife(PlayerRef, ResolutionValue)`: Causes player to lose life.
* `LoseLifeEach(PlayerSet, ResolutionValue)`: Causes each player in the set to lose life.

### Damage Actions
The engine event payload `core.GameEvent.SourceControllerID uuid.UUID` is available to engine trigger conditions for `EvtDamageDealt`. It records the controller of the damage source captured by the damage action, even if that source leaves before the event is observed. It is set for positive damage to a player or permanent after replacement and prevention. This is an engine integration field; the typed event pattern that will bind it for card declarations is not yet available. An engine condition can read it without looking up the source card again:

```go
func sourceWasControlledBy(evt *core.GameEvent, player uuid.UUID) bool {
    return evt.Type == core.EvtDamageDealt && evt.SourceControllerID == player
}
```

This example uses `core` from `github.com/benprew/mage-go/pkg/mage/core` and `uuid` from `github.com/google/uuid`.

* `DealDamageFrom(DamageSourceRef, DamageRecipientRef, ResolutionValue, ...DamageResultOption) ResolvingEffect`: Deals one damage action from `This` to a player or battlefield permanent. Use it in `Spell`, `Activate`, or a triggered resolving body. The source keeps its resolving incarnation and last known characteristics after departure. A missing source or recipient has no effect. The current `DealDamage(ValueSource)` remains for unmigrated positional callers.
* `DamageSourceOf(PermanentRef) DamageSourceRef`: Uses a typed permanent reference as the source of `DealDamageFrom` or `DealDamageEach`. A declared target or `From` snapshot keeps its captured battlefield incarnation; if that permanent has left, the damage instruction uses its departure characteristics and controller. A phased-out or missing permanent without departure information cannot supply damage. The argument must be a `PermanentRef`; player and stack-object references are rejected by Go. This source can be used in a spell, activation, or trigger body.

```go
other := dsl.TargetPermanent("other creature", dsl.IsCreature)
ability := dsl.Activate(
    dsl.Costs(dsl.Mana("{G}{G}"), dsl.TapCost(dsl.This)),
    dsl.Sequence(
        dsl.DealDamageFrom(dsl.This, other, dsl.PowerOf(dsl.This)),
        dsl.DealDamageFrom(dsl.DamageSourceOf(other), dsl.This, dsl.PowerOf(other)),
    ),
    dsl.WithTargets(other),
)
_ = ability
```
* `DamageRecipients(PlayerSet, PermanentSet) DamageRecipientSet`: Combines the selected players and battlefield permanents for one damage instruction. Both sets are required. `TargetPermanents(name, minimum, maximum, filter, options...)` supplies a declared permanent target set when used with `WithTargets`. Triggered abilities, spells, and activations can use multiple variable-size groups through their respective group announcement paths. An illegal target is absent from the batch.
* `TargetDamageRecipients(name string, minimum, maximum int) *DamageTargetsBinding`: Declares one bounded target group of players, creatures, and planeswalkers. It is both a `TargetBinding` and a `DamageRecipientSet`. Declare it with `WithTargets`, `WithTargetGroups`, or `WithActivationTargetGroups` as appropriate. Registration rejects invalid bounds. The target announcement rejects duplicate or illegal choices before payment. At resolution, an illegal, phased-out, or returned permanent is absent from the batch; each remaining recipient is dealt damage simultaneously. This collection cannot be used as the singular recipient of `DealDamageFrom`.
* `DealDamageEach(DamageSourceRef, DamageRecipientSet, ResolutionValue, ...DamageEachResultOption) ResolvingEffect`: Deals the same amount to each selected recipient simultaneously. Use it in a `Spell`, `Activate`, or triggered resolving body. The source and amount are read once before recipients are collected. The action uses the shared damage replacement and prevention pipeline.

```go
targets := dsl.TargetPermanents("creatures", 1, 2, dsl.IsCreature)
ability := dsl.Spell(
    dsl.DealDamageEach(dsl.This,
        dsl.DamageRecipients(dsl.EachPlayer, targets), dsl.Fixed(1)),
    dsl.WithTargets(targets),
)
_ = ability
```

```go
targets := dsl.TargetDamageRecipients("recipients", 1, 2)
ability := dsl.Spell(
    dsl.DealDamageEach(dsl.This, targets, dsl.Fixed(1)),
    dsl.WithTargets(targets),
)
_ = ability
```
* `DamageTotalInto(*NumberBinding) DamageEachResultOption`: Saves the sum of damage actually dealt by one `DealDamageEach` instruction after replacement, redirection, and prevention. It writes zero when no damage is dealt. The result is local to the resolving spell or ability. Use `FromNumber` in a later instruction. Each batch can declare this option once.
* `BindDamageResults(string) *DamageResultsBinding`, `DamageResultsInto(*DamageResultsBinding) DamageEachResultOption`: Save each final positive damage action from one `DealDamageEach` instruction. The binding is empty if the source or recipients are absent, damage is zero, or all damage is prevented. It can be used with `DamageTotalInto` on the same instruction.
* `BindDamageResult(string) *DamageResultBinding`, `ForEachDamageResult(*DamageResultsBinding, *DamageResultBinding, ResolvingEffect) ResolvingEffect`: Run a body once for each saved final action, after the complete batch. The body is sequential; it does not deal the batch damage. Each iteration has its own bindings.
* `DamageResultAmount(*DamageResultBinding) ResolutionValue`, `DamageResultPlayer(*DamageResultBinding) PlayerRef`, `DamageResultPermanent(*DamageResultBinding) PermanentRef`, and `DamageResultSource(*DamageResultBinding) DamageSourceRef`: Read the actual amount, final player or permanent, and captured source in a result body. A player reference is absent for permanent damage, and a permanent reference is absent for player damage. A permanent reference keeps its battlefield incarnation and becomes absent after it leaves or phases out. The source keeps damage-time characteristics after departure. These readers require a dominating `ForEachDamageResult` member declaration.
* `DamageAmountInto(*NumberBinding) DamageResultOption`: Saves the amount dealt by that instruction after damage replacement, redirection, and prevention. It writes zero when no damage is dealt.
* `DamagePlayerInto(*PlayerBinding) DamageResultOption`: Saves the player who actually receives positive damage. A redirected-to-permanent, prevented, zero, or absent action clears the binding. Use `FromPlayer` in later instructions. Each result option can occur once per instruction; the amount and player options can occur together.
* `DamagePermanentInto(*PermanentBinding) DamageResultOption`: Saves the battlefield incarnation that actually receives positive damage, after redirection. A redirected-to-player, prevented, zero, or absent action clears the binding. Use `From` in later instructions. Each result option can occur once per instruction and can combine with the amount and player options.
* `BindDamageSource(string) *DamageSourceBinding`, `DamageSourceInto(*DamageSourceBinding) DamageResultOption`, and `FromDamageSource(*DamageSourceBinding) DamageSourceRef`: Save the source of positive damage and use its captured incarnation and characteristics in a later damage instruction in the same resolution. Zero, prevented, and absent damage clear the binding. The source option can combine with the amount and recipient options, once per instruction.
* `PreventNextDamage(DamageRecipientRef, ResolutionValue)`: Installs a prevention shield on creature or player.
* `PreventAllCombatDamage()`: Prevents all combat damage for the turn (Fog).
* `PreventCombatDamageToAndBy(PermanentRef)`: Prevents combat damage to and by a specific permanent.

The engine APIs `(*Game).CaptureDamageSource(uuid.UUID) DamageSource`, `(*Game).DealDamageToPermanentFromSource(*Permanent, int, DamageSource)`, and `(*Game).DealDamageToPlayerFromSource(Player, int, DamageSource)` serve engine and integration code. Capture freezes a permanent's incarnation, controller, colors, types, and keywords. The damage methods keep that source through replacement and prevention. The existing UUID-based damage methods remain available and capture the source when called. A planeswalker loses loyalty counters after unprevented damage; a planeswalker creature also has damage marked.

Engine and integration code can submit one mixed damage event with `(*Game).DealSimultaneousDamage(...DamageAssignment)`. Use `PlayerDamageAssignment(Player, int, DamageSource)` and `PermanentDamageAssignment(*Permanent, int, DamageSource)` to build its parts. Capture the source before the instruction. Each permanent assignment keeps its battlefield incarnation; a returned or phased-out recipient receives no damage. The engine prepares all assignments with the existing damage replacement and prevention service, applies their results, then sends the queued events and damage callbacks. `DealDamageEach` is the composable adapter for a player set and permanent set.

```go
package example

import "github.com/benprew/mage-go/pkg/mage/dsl"

func simultaneousDamage() dsl.Card {
    total := dsl.BindNumber("damage dealt")
    return dsl.NewSorcery("Test Burst", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
        dsl.DealDamageEach(dsl.This,
            dsl.DamageRecipients(dsl.EachPlayer, dsl.MatchingPermanents(dsl.IsCreature)),
            dsl.Fixed(2), dsl.DamageTotalInto(total)),
        dsl.GainLife(dsl.You, dsl.FromNumber(total)),
    ))))
}
```

```go
package example

import "github.com/benprew/mage-go/pkg/mage/dsl"

func damageByRecipient() dsl.Card {
    results := dsl.BindDamageResults("results")
    item := dsl.BindDamageResult("item")
    return dsl.NewSorcery("Test Results", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(
        dsl.DealDamageEach(dsl.This,
            dsl.DamageRecipients(dsl.EachPlayer, dsl.MatchingPermanents(dsl.IsCreature)),
            dsl.Fixed(2), dsl.DamageResultsInto(results)),
        dsl.ForEachDamageResult(results, item,
            dsl.GainLife(dsl.You, dsl.DamageResultAmount(item))),
    ))))
}
```

```go
package example

import "github.com/benprew/mage-go/pkg/mage"

func damageTogether(game *mage.Game, sourcePermanent, creature *mage.Permanent, player mage.Player) {
    source := game.CaptureDamageSource(sourcePermanent.ID())
    game.DealSimultaneousDamage(
        mage.PlayerDamageAssignment(player, 2, source),
        mage.PermanentDamageAssignment(creature, 2, source),
    )
}
```

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func damageWalker() {
    a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
    game := mage.NewGame(a, b)
    walker := game.PutOnBattlefield(mage.NewPlaneswalker("Test Walker", "", 4), b.PlayerID())
    source := game.PutOnBattlefield(mage.NewCreature("Source", "", 1, 1), a.PlayerID())
    game.DealDamageToPermanentFromSource(walker, 2, game.CaptureDamageSource(source.ID()))
    _ = walker.Counters[core.Loyalty]
}
```

The typed constructor can use a declared damage target and its actual result:

```go
package example

import "github.com/benprew/mage-go/pkg/mage/dsl"

func damageAbility() {
    recipient := dsl.TargetDamageRecipient("recipient")
    dealt := dsl.BindNumber("damage dealt")
    damagedPlayer := dsl.BindPlayer("damaged player")
    ability := dsl.Spell(dsl.Sequence(
        dsl.DealDamageFrom(dsl.This, recipient, dsl.Fixed(3), dsl.DamageAmountInto(dealt), dsl.DamagePlayerInto(damagedPlayer)),
        dsl.GainLife(dsl.FromPlayer(damagedPlayer), dsl.FromNumber(dealt)),
    ), dsl.WithTargets(recipient))
    _ = ability
}
```

`DamageSourceRef` accepts `This` and `FromDamageSource(binding)`. `DamageRecipientRef` accepts `You`, `ActivePlayer`, typed player and permanent references, and `TargetDamageRecipient`. `TargetPlayerOrPlaneswalkerDamage(name string) *DamageTargetBinding` declares one player-or-planeswalker target and excludes creatures. It works in `Spell`, `Activate`, and `WhenEvent` through `WithTargets`, and with `DealDamageFrom` or `PreventNextDamage`. The action rechecks a target's battlefield incarnation at resolution. The result amount is damage dealt, not the recipient's resulting life loss or loyalty loss.

```go
package example

import "github.com/benprew/mage-go/pkg/mage/dsl"

var recipient = dsl.TargetPlayerOrPlaneswalkerDamage("recipient")
var spell = dsl.Spell(dsl.DealDamageFrom(dsl.This, recipient, dsl.Fixed(2)), dsl.WithTargets(recipient))
```

This spell uses one captured source for two sequential damage instructions:

```go
package example

import "github.com/benprew/mage-go/pkg/mage/dsl"

func twoDamageInstructions() {
    source := dsl.BindDamageSource("first damage source")
    ability := dsl.Spell(dsl.Sequence(
        dsl.DealDamageFrom(dsl.This, dsl.You, dsl.Fixed(1), dsl.DamageSourceInto(source)),
        dsl.DealDamageFrom(dsl.FromDamageSource(source), dsl.You, dsl.Fixed(2)),
    ))
    _ = ability
}
```

For a permanent result in a spell, the saved recipient can be used by a later instruction:

```go
recipient := dsl.TargetDamageRecipient("recipient")
damaged := dsl.BindPermanent("damaged permanent")
ability := dsl.Spell(dsl.Sequence(
    dsl.DealDamageFrom(dsl.This, recipient, dsl.Fixed(2), dsl.DamagePermanentInto(damaged)),
    dsl.AddCounters(dsl.From(damaged), dsl.P1P1, dsl.Fixed(1)),
), dsl.WithTargets(recipient))
_ = ability
```

This example uses `dsl` from `github.com/benprew/mage-go/pkg/mage/dsl`. A player result leaves `damaged` absent, so the counter instruction has no subject. Collection damage and group totals remain separate work.

### Counter Actions
* `AddCounters(PermanentRef, CounterType, ResolutionValue)`: Adds counters through the replacement pipeline.
* `RemoveCountersFrom(PermanentRef, CounterType, ResolutionValue)`: Removes up to the specified count.
* `RemoveAllCounters(PermanentRef, CounterType)`: Removes all counters of that kind.

### Library & Mana Actions
* `ScryAtResolution(ResolutionValue)`: Scries cards equal to the amount.
* `SurveilAtResolution(ResolutionValue)`: Surveils cards equal to the amount.
* `AddManaAmount(Color, ResolutionValue)`: Adds mana of the specified color to the pool.
* `AddManaOfAnyOneColor(ResolutionValue)`: The resolving controller chooses one color and adds the full amount in that color. It works in resolving spell, activation, and trigger bodies. A targetless activation that can produce mana resolves as a mana ability. A fixed positive amount in a tap-only mana activation is available to automatic payment planning; a dynamic amount is not.

```go
package example

import dsl "github.com/benprew/mage-go/pkg/mage/dsl"

func choiceLand() *dsl.BaseCard {
    return dsl.NewLand("Choice land", dsl.WithAbility(dsl.Activate(
        dsl.Costs(dsl.TapCost(dsl.This)),
        dsl.AddManaOfAnyOneColor(dsl.Fixed(2)),
    )))
}
```

---

## 7. Control Flow, Choices, and Branches

* `Sequence(effects...)`: Executes resolving effects in order, threading a single resolution frame.
* `ForEach(set, memberBinding, body)`: Iterates sequentially over permanents in the set.
* `FlipCoin(playerRef, OnWin(effect), OnLoss(effect))`: Resolves a coin flip through the game's deterministic RNG.
* `IfState(condition, effect)`: Resolution-time branch based on pure game state.
* `IsYourTurn() StateCondition`: Checks whether the active player is the stored controller when the condition is evaluated. Use it with `IfState` in a spell, activation, or trigger body; it also works in pure static and state conditions. It does not restrict announcement or activation. For example:

```go
package example

import dsl "github.com/benprew/mage-go/pkg/mage/dsl"

func turnCheck() *dsl.BaseCard {
    return dsl.NewInstant("Turn check", "{W}",
        dsl.Spell(dsl.IfState(dsl.IsYourTurn(),
            dsl.GainLife(dsl.You, dsl.Fixed(1)))))
}
```
* `IfPaid(playerRef, costGroup, effect)`: Offers payment transaction; executes body only if paid.
* `UnlessPaid(playerRef, costGroup, effect)`: Offers payment transaction; executes body if declined/unpaid.
  For "destroy this creature unless you pay {1}", use
  `UnlessPaid(You, Costs(Mana("{1}")), Destroy(This))`. A granted trigger
  keeps its triggering controller and source incarnation through resolution.
* `ChoosePermanent(playerRef, set, binding)`: Player chooses a permanent at resolution, storing it in `binding`.
* `ChooseRandomPermanent(set, binding)`: Selects a random permanent from the set at resolution.
* `ChooseRandomPlayer(playerSet, binding)`: Selects a random player from the set at resolution.

For a random tap, use `ChooseRandomPermanent(MatchingPermanents(filter), chosen)`
followed by `Tap(From(chosen))`. The collection excludes phased-out permanents;
an empty choice consumes no random value. The choice does not declare a target.

---

## 8. Saved Bindings & Snapshots

Bindings bridge resolution steps without string dictionaries:

```go
host := BindPermanent("host")

Sequence(
    Snapshot(AttachedPermanent, host),
    Sacrifice(This),
    AddCounters(From(host), P1P1, Fixed(1)),
)
```

* `BindPermanent(label)`: Declares a permanent binding variable.
* `BindPlayer(label)`: Declares a player binding variable.
* `BindNumber(label)`: Declares a numeric binding variable.
* `Snapshot(subject, binding)`: Captures the subject's current battlefield incarnation.
* `SnapshotPlayer(subject, binding)`: Captures a player reference at resolution.
* `SnapshotNumber(value, binding)`: Evaluates and freezes a numeric value at resolution.
* `From(binding)` / `FromPlayer(binding)` / `FromNumber(binding)`: Reads the saved value in later instructions.

`SacrificeCreatureCost()` and `SacrificeMatchingCost(filter, text)` select one
other permanent controlled by the payer. `SacrificeMatchingIncludingSourceCost`
can select the source. `SacrificeNMatchingCost(count, filter, text)` selects the
specified number of other permanents. These costs work in `Costs(...)` for
`Activate`, `WithAdditionalCost` for spells, and payment branches. The payer
chooses during payment preparation. The transaction validates and pays those
same battlefield incarnations. A returned or phased-out choice cannot be
silently replaced with another permanent at commit.

```go
ability := Activate(
    Costs(SacrificeCreatureCost()),
    GainLifeFor(You, Fixed(1)),
)
_ = NewArtifact("Offering", "", WithAbility(ability))
```

`SacrificeCreatureIncludingSourceCostWithStats(power, toughness *NumberBinding)`
uses the same prepared-payment transaction and also permits the source when it
is a creature. Use it only as an activation cost or spell additional cost.
It saves the chosen creature's power and toughness immediately before payment
in the resulting stack instance. The source can leave as payment; the saved
numbers remain available at resolution.

```go
power, toughness := BindNumber("paid power"), BindNumber("paid toughness")
ability := Activate(
    Costs(SacrificeCreatureIncludingSourceCostWithStats(power, toughness)),
    GainLifeFor(You, FromNumber(toughness)),
)
_ = NewCreature("Offering", "", 2, 2, WithAbility(ability))
```

Other sacrifice costs do not yet produce a typed paid-permanent value. A
resolving `SacrificeAmountInto` binding reports a separate action and cannot
read this payment.

Crumble captures `ControllerOf(artifact)` with `SnapshotPlayer` and
`ManaValueOf(artifact)` with `SnapshotNumber` before `DestroyWithoutRegeneration`.
Its later `GainLife` reads those saved values, including when destruction fails.

Launch Party captures `ControllerOf(creature)` with `SnapshotPlayer` before
`Destroy(creature)`, then uses `LoseLife(FromPlayer(controller), Fixed(2))`.
Life loss still occurs if regeneration or indestructible prevents destruction,
and it emits a life-loss event. Attach `WithAdditionalCost(SacrificeCreatureCost())`
to the card alongside `Spell`; choose the target before sacrifice payment.
Sacrificing the sole target makes it illegal, so neither instruction resolves.

Assassin's Strike uses the same controller snapshot before `Destroy`, followed
by `Discard(FromPlayer(controller), Fixed(1))`. The captured player chooses the
card, and the discard uses the shared replacement and event services. Discard
still occurs when regeneration or indestructible prevents destruction.

Swords to Plowshares captures `ControllerOf(creature)` and `PowerOf(creature)`
before `Exile(creature)`, then gains life with those saved values. Nonpositive
power gains no life. `Exile` retains exile targeting metadata through `Sequence`;
it does not report destruction that regeneration can replace.

---

## 9. Triggered Abilities

Use distinct constructors for distinct trigger lifecycles:

### Event Triggers (`WhenEvent`)
Triggers on rules events dispatched by the engine:
* `WhenEvent(EventPattern, ResolvingEffect, ...TriggerOption)`
* Event patterns:
  * `LandTappedForMana().WithPlayer(*PlayerBinding)`: Match a land's tap-cost mana ability when it resolves and produces mana. `WithPlayer` is optional and saves the activating player when the event occurs. Use it in `WhenEvent` or `WhenEventModes` on a battlefield permanent. The binding is local to each trigger and is not a target. For example:

    ```go
    player := dsl.BindPlayer("mana player")
    ability := dsl.WhenEvent(
        dsl.LandTappedForMana().WithPlayer(player),
        dsl.DealDamageFrom(dsl.This, dsl.FromPlayer(player), dsl.Fixed(1)))
    ```

  * `Enters(This)`: Source enters the battlefield.
  * `Dies(This)`: Source dies (moves from battlefield to graveyard).
  * `LeavesBattlefield(This)`: Source moves from the battlefield to any other zone. Destruction, exile, and return to hand match. Phasing does not match. Use it in `WhenEvent` or `WhenEventModes` on a battlefield permanent. The pending trigger keeps its controller and source incarnation.
  * `OneOrMoreCreaturesYouControlAttack().WithCount(*NumberBinding)`: Match once when one or more creatures controlled by this permanent's controller are declared as attackers in one declare attackers step. `WithCount` is optional. It saves the number actually declared for this trigger instance; read it with `FromNumber`. Creatures put onto the battlefield attacking do not count. Use it in `WhenEvent` or `WhenEventModes` on a battlefield permanent. For example:

    ```go
    package main

    import "github.com/benprew/mage-go/pkg/mage/dsl"

    var count = dsl.BindNumber("declared attackers")
    var _ = dsl.NewEnchantment("Attack watcher", "",
        dsl.WithAbility(dsl.WhenEvent(
            dsl.OneOrMoreCreaturesYouControlAttack().WithCount(count),
            dsl.GainLife(dsl.You, dsl.FromNumber(count)))))

    func main() {}
    ```
  * `EachCreatureYouControlAttacks().WithAttacker(*PermanentBinding).WithDefendingPlayer(*PlayerBinding)`: Match once per creature you control that is declared as an attacker. Either capture is optional. `From` reads the attacking battlefield incarnation; `FromPlayer` reads the defending player. Both values belong to that trigger instance. A creature put onto the battlefield attacking does not match. Use this pattern in `WhenEvent` or `WhenEventModes` on a battlefield permanent. The combat service currently records player defenders only.

    ```go
    package main

    import "github.com/benprew/mage-go/pkg/mage/dsl"

    var attacker = dsl.BindPermanent("attacker")
    var defender = dsl.BindPlayer("defender")
    var _ = dsl.NewEnchantment("Attack watcher", "",
        dsl.WithAbility(dsl.WhenEvent(
            dsl.EachCreatureYouControlAttacks().WithAttacker(attacker).WithDefendingPlayer(defender),
            dsl.Sequence(dsl.Tap(dsl.From(attacker)),
                dsl.GainLife(dsl.FromPlayer(defender), dsl.Fixed(1))))))

    func main() {}
    ```
  * `CreatureYouSacrifice() EventPattern`: Match each creature sacrificed by the source controller, including the source and tokens. It uses the completed sacrifice event and the creature type at sacrifice time. The pattern has no target or sacrificed-object binding. Use it in `WhenEvent` or `WhenEventModes` on a battlefield permanent. For example:

    ```go
    mage.NewCreature("Sacrifice Watcher", "", 1, 1,
        mage.WithAbility(mage.WhenEvent(
            mage.CreatureYouSacrifice(),
            mage.GainLifeFor(mage.You, mage.Fixed(1)))))
    ```
  * `SpellCast().OfColor(core.Color) EventPattern`: Match a spell cast by any player while the source is on the battlefield. `OfColor` is optional and accepts one of the five Magic colors. It checks the spell's effective color on the stack when the cast event occurs, including color changes. The pattern does not target or bind the spell. Use it in `WhenEvent` or `WhenEventModes`.
  * `SpellCast().OfType(core.CardType) EventPattern`: Match a cast spell of one nonland card type. The filter reads the spell on the stack at cast time, including more than one type. A repeated filter or a land type fails registration. It composes with `OfColor`, `ByOpponent`, and `WithCaster` in `WhenEvent` and `WhenEventModes` on a battlefield source. For example:
    ```go
    watcher := dsl.NewCreature("Watcher", "{2}{U}", 1, 2,
        dsl.WithAbility(dsl.WhenEvent(
            dsl.SpellCast().OfType(core.TypeArtifact),
            dsl.Draw(dsl.You, dsl.Fixed(1)),
        )),
    )
    _ = watcher
    ```
  * `SpellCast().ByOpponent().WithCaster(*PlayerBinding) EventPattern`: `ByOpponent` matches a caster who is an opponent of the source controller when the spell is cast. `WithCaster` is optional and saves that event player for this trigger instance. Read it with `FromPlayer` in the trigger body. The two filters compose with `OfColor`. A missing or repeated caster binding is rejected at registration. This does not bind or target the spell.

    ```go
    package main

    import "github.com/benprew/mage-go/pkg/mage/dsl"

    var caster = dsl.BindPlayer("spell caster")
    var _ = dsl.NewArtifact("Cast Watcher", "{1}",
        dsl.WithAbility(dsl.WhenEvent(
            dsl.SpellCast().ByOpponent().WithCaster(caster),
            dsl.GainLife(dsl.FromPlayer(caster), dsl.Fixed(1)),
        )),
    )

    func main() {}
    ```

    ```go
    package main

    import (
        "github.com/benprew/mage-go/pkg/mage/core"
        "github.com/benprew/mage-go/pkg/mage/dsl"
    )

    var _ = dsl.NewArtifact("Blue Spell Watcher", "{1}",
        dsl.WithAbility(dsl.WhenEvent(
            dsl.SpellCast().OfColor(core.Blue),
            dsl.GainLife(dsl.You, dsl.Fixed(1)),
        )),
    )

    func main() {}
    ```
  * `ArtifactBecomesTapped().WithArtifact(*PermanentBinding).WithController(*PlayerBinding) EventPattern`: Match any artifact when it changes from untapped to tapped. Both methods are optional. `WithArtifact` saves the tapped artifact's battlefield incarnation for `From`; `WithController` saves its controller at the event for `FromPlayer`. Use the pattern with `WhenEvent` or `WhenEventModes` on a battlefield permanent. The artifact is not a target. A saved artifact that leaves and returns, or phases out, is absent at resolution. Entering tapped and tapping an already tapped artifact do not match. For example:
    ```go
    package main

    import "github.com/benprew/mage-go/pkg/mage/dsl"

    var tapped = dsl.BindPermanent("tapped artifact")
    var controller = dsl.BindPlayer("tap controller")
    var _ = dsl.NewEnchantment("Artifact watcher", "",
        dsl.WithAbility(dsl.WhenEvent(
            dsl.ArtifactBecomesTapped().WithArtifact(tapped).WithController(controller),
            dsl.Sequence(
                dsl.AddCounters(dsl.From(tapped), dsl.Charge, dsl.Fixed(1)),
                dsl.GainLife(dsl.FromPlayer(controller), dsl.Fixed(1))))))

    func main() {}
    ```
  * `ArtifactAbilityActivatedWithoutTapCost().AttachedToSource().WithArtifact(*PermanentBinding).WithController(*PlayerBinding) EventPattern`: Match an activated ability of a battlefield artifact when its activation cost has no `{T}`. `AttachedToSource` optionally requires the artifact to be attached to the trigger source. Either binding is optional. `WithArtifact` saves the battlefield incarnation for `From`; `WithController` saves its controller at activation for `FromPlayer`. A sacrificed artifact has no live permanent reference at resolution, but its saved controller remains available. Use this pattern with `WhenEvent` or `WhenEventModes` on a battlefield permanent. For example:
    ```go
    package main

    import "github.com/benprew/mage-go/pkg/mage/dsl"

    var controller = dsl.BindPlayer("artifact controller")
    var _ = dsl.NewEnchantment("Ability watcher", "",
        dsl.WithAbility(dsl.WhenEvent(
            dsl.ArtifactAbilityActivatedWithoutTapCost().WithController(controller),
            dsl.GainLife(dsl.FromPlayer(controller), dsl.Fixed(1)))))

    func main() {}
    ```
  * `OpponentLandBecomesTapped().OfSubtype(string).WithLand(*PermanentBinding) EventPattern`: Match an opponent-controlled land when it changes from untapped to tapped. Both methods are optional. `OfSubtype` requires a valid land subtype at registration. `WithLand` saves the tapped land's battlefield incarnation for `From` in the trigger body. A land that leaves and returns, or phases out, is absent from that reference. Use the pattern with `WhenEvent` or `WhenEventModes` on a battlefield permanent. The condition uses the land's type, subtype, and controller when it becomes tapped. It does not target the land.
    ```go
    package main

    import "github.com/benprew/mage-go/pkg/mage/dsl"

    var tapped = dsl.BindPermanent("tapped Forest")
    var _ = dsl.NewEnchantment("Forest watcher", "",
        dsl.WithAbility(dsl.WhenEvent(
            dsl.OpponentLandBecomesTapped().OfSubtype("Forest").WithLand(tapped),
            dsl.AddCounters(dsl.From(tapped), dsl.Charge, dsl.Fixed(1)))))

    func main() {}
    ```
    ```go
    package main

    import "github.com/benprew/mage-go/pkg/mage/dsl"

    var _ = dsl.NewCreature("Leaving watcher", "", 1, 1,
        dsl.WithAbility(dsl.WhenEvent(dsl.LeavesBattlefield(dsl.This),
            dsl.GainLife(dsl.You, dsl.Fixed(1)))))

    func main() {}
    ```
  * `BecomesTapped(This) EventPattern`: Match the battlefield source when it changes from untapped to tapped. Use it with `WhenEvent` or `WhenEventModes` on a permanent. It does not match entering tapped or a second tap while already tapped. The pending trigger keeps its event-time controller and source incarnation. For example:

    ```go
    package main

    import "github.com/benprew/mage-go/pkg/mage/dsl"

    var _ = dsl.NewLand("Tap watcher",
        dsl.WithAbility(dsl.WhenEvent(dsl.BecomesTapped(dsl.This),
            dsl.GainLife(dsl.You, dsl.Fixed(1)))))

    func main() {}
    ```
  * `Attacks(This)`: Source is declared as an attacker. A creature put onto the battlefield attacking does not match.
  * `Attacks(This).WithDefendingPlayer(*PlayerBinding)`: Save the player defending against the source when it is declared as an attacker. Read the saved player with `FromPlayer` in the trigger body. The binding is instance-local and is not a target. It remains the same after the attacker leaves combat or changes controller. Use this pattern in `WhenEvent` or `WhenEventModes` on a battlefield creature. A creature put onto the battlefield attacking does not match. For example:

    ```go
    defender := dsl.BindPlayer("defender")
    dsl.NewCreature("Example", "", 2, 2,
        dsl.WithAbility(dsl.WhenEvent(
            dsl.Attacks(dsl.This).WithDefendingPlayer(defender),
            dsl.GainLife(dsl.FromPlayer(defender), dsl.Fixed(1)))))
    ```

  * `Blocks(This)`: Source is declared as a blocker. It matches once when the source blocks one or more attackers, after block legality checks finish. A block removed by a minimum-blocker restriction does not match. A creature put onto the battlefield blocking does not match. Use this pattern in `WhenEvent` or `WhenEventModes` on a battlefield creature.

    ```go
    var _ = dsl.NewCreature("Block watcher", "", 2, 2,
        dsl.WithAbility(dsl.WhenEvent(dsl.Blocks(dsl.This),
            dsl.GainLife(dsl.You, dsl.Fixed(1)))))
    ```
  * `BecomesBlocked(This)`: Source first becomes blocked in combat. It matches once even when multiple creatures block it, and also matches when a creature enters blocking an unblocked source. It is available in `WhenEvent` and `WhenEventModes` on a battlefield source. It does not bind or target a blocker.
  * `BecomesTargetOfSpell(This) EventPattern`: Match the battlefield source when a spell targets it. An activated ability targeting the source does not match. Use it in `WhenEvent` or `WhenEventModes`; the pattern has no target or stack-object binding. For example:

    ```go
    package main

    import "github.com/benprew/mage-go/pkg/mage/dsl"

    var _ = dsl.NewCreature("Spell watcher", "", 2, 2,
        dsl.WithAbility(dsl.WhenEvent(
            dsl.BecomesTargetOfSpell(dsl.This), dsl.Sacrifice(dsl.This))))
    ```
  * `AttacksUnblocked(This)`: Source is attacking and no creature blocks it when legal blocker declarations finish. A creature that entered attacking can match. Removing every blocker from an already blocked creature does not make it match. Use `.WithDefendingPlayer(*PlayerBinding)` to save the event-time defending player. Use it in `WhenEvent` or `WhenEventModes` on a battlefield source. For example:

    ```go
    defender := dsl.BindPlayer("defender")
    ability := dsl.WhenEvent(
        dsl.AttacksUnblocked(dsl.This).WithDefendingPlayer(defender),
        dsl.Draw(dsl.FromPlayer(defender), dsl.Fixed(1)),
    )
    ```
  * `BecomesBlockedBy(This)`: Source becomes blocked by a creature. It matches once for each new attacker and blocker pair, including a creature that enters blocking. Add `.OfSubtype(string)` to require a creature subtype at the event. Add `.WithBlocker(*PermanentBinding)` to save that blocker's battlefield incarnation for the trigger body. The saved blocker is not a target and becomes absent if it leaves and returns or phases out. This pattern is valid in `WhenEvent` and `WhenEventModes` on a battlefield source.

    `BlocksOrBecomesBlockedBy(This).OfSubtype(string)` matches one group when the source blocks or becomes blocked by one or more creatures of that subtype. It checks the other creatures at event detection and triggers once for a group with several matches. A creature put onto the battlefield blocking can cause the attacker to match, but it did not itself block. Use this pattern in `WhenEvent` or `WhenEventModes` on a battlefield creature.

    ```go
    package main

    import "github.com/benprew/mage-go/pkg/mage/dsl"

    var _ = dsl.NewCreature("Orc watcher", "", 2, 2,
        dsl.WithAbility(dsl.WhenEvent(
            dsl.BlocksOrBecomesBlockedBy(dsl.This).OfSubtype("Orc"),
            dsl.GainLife(dsl.You, dsl.Fixed(1)))))

    func main() {}
    ```

    ```go
    package example

    import (
        "github.com/benprew/mage-go/pkg/mage/core"
        "github.com/benprew/mage-go/pkg/mage/dsl"
    )

    func wallWatcher() dsl.Card {
        blocker := dsl.BindPermanent("blocking Wall")
        return dsl.NewCreature("Wall Watcher", "", 2, 2,
            dsl.WithAbility(dsl.WhenEvent(
                dsl.BecomesBlockedBy(dsl.This).OfSubtype("Wall").WithBlocker(blocker),
                dsl.AddCounters(dsl.From(blocker), core.P1P1, dsl.Fixed(1)))))
    }
    ```

    ```go
    var _ = dsl.NewCreature("Blocked watcher", "", 2, 2,
        dsl.WithAbility(dsl.WhenEvent(dsl.BecomesBlocked(dsl.This),
            dsl.GainLife(dsl.You, dsl.Fixed(1)))))
    ```

  * `CreatureDies()`: Any creature dies.
  * `LandEnters().OfSubtype(string).WithActor(*PlayerBinding).WithController(*PlayerBinding)`: Match a land entering the battlefield. Each method is optional. `OfSubtype` filters by one land subtype. `WithActor` saves the player who put the land onto the battlefield; `WithController` saves its controller at entry. The two players can differ. Use this pattern with `WhenEvent` or `WhenEventModes` on a battlefield permanent. A saved player remains fixed after the land or trigger source leaves. Registration rejects an invalid subtype, missing or repeated binding, or one binding used for both players. An entry without a known actor does not match a pattern with `WithActor`. For example:

    ```go
    package example

    import "github.com/benprew/mage-go/pkg/mage/dsl"

    func swampWatcher() dsl.Card {
        actor := dsl.BindPlayer("entry actor")
        return dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(
            dsl.LandEnters().OfSubtype("Swamp").WithActor(actor),
            dsl.DealDamageFrom(dsl.This, dsl.FromPlayer(actor), dsl.Fixed(2)))))
    }
    ```

    This example uses `dsl` from `github.com/benprew/mage-go/pkg/mage/dsl`. `PutOnBattlefieldBy(card, controller, actor)` supplies the actor for an effect entry; normal land play supplies its player. The existing `PutOnBattlefield(card, controller)` does not supply an actor.
  * `PermanentDies()`: Any permanent moves from the battlefield to a graveyard. Add `.OfType(core.TypeLand)` or another permanent card type to test the departing object's last-known type. `WithController`, `WithOwner`, and `Nontoken` use the same event-time captures and filters as creature-death patterns. The source must be a battlefield permanent. This pattern is valid in `WhenEvent` and `WhenEventModes` and declares no target. `OfType` accepts creature, land, artifact, enchantment, or planeswalker; registration rejects other types and repeated type filters.
    ```go
    former := dsl.BindPlayer("former land controller")
    dsl.WhenEvent(
        dsl.PermanentDies().OfType(core.TypeLand).WithController(former),
        dsl.DealDamageFrom(dsl.This, dsl.FromPlayer(former), dsl.Fixed(2)),
    )
    ```
  * `AnotherCreatureDies()`: A creature other than the source dies, regardless of its controller.
  * Death patterns accept `.WithCard(*CardBinding)`. `BindCard(name)` declares the binding, and `FromCard(binding)` is a `CardRef` for graveyard actions such as `ReturnCardToHand`. It saves the card's first graveyard visit when the trigger occurs. The reference is absent if a token dies, the card does not enter the graveyard, or it leaves and returns before resolution. This is an event binding, not a target. It works in `WhenEvent` and `WhenEventModes` on a battlefield permanent.

    ```go
    package main

    import "github.com/benprew/mage-go/pkg/mage/dsl"

    func watcher() dsl.Card {
        dead := dsl.BindCard("dead creature")
        return dsl.NewCreature("Watcher", "", 1, 1,
            dsl.WithAbility(dsl.WhenEvent(
                dsl.AnotherCreatureDies().WithCard(dead),
                dsl.ReturnCardToHand(dsl.FromCard(dead)))))
    }

    func main() {}
    ```
  * `CreatureYouControlDies()`: A creature controlled by ability controller dies.
  * `OpponentCreatureDies()`: A creature controlled by an opponent of the ability controller dies. Use it in `WhenEvent` or `WhenEventModes` on a battlefield permanent. The filter uses the dying creature's controller and type before departure. It includes tokens. `.WithController(*PlayerBinding)` saves that controller for the resolving body; it does not declare a target.
    ```go
    former := dsl.BindPlayer("former controller")
    ability := dsl.WhenEvent(
        dsl.OpponentCreatureDies().WithController(former),
        dsl.GainLife(dsl.FromPlayer(former), dsl.Fixed(1)),
    )
    _ = ability
    ```
  * `AnotherCreatureYouControlDies()`: A different creature controlled by the ability controller dies. The controller and creature type are checked before departure; the dying creature is not a target.
  * `AnotherCreatureYouControlEnters()`: A different creature enters under the ability controller's control. Add `.Nontoken()` to exclude tokens. The check uses the entering permanent after entry effects and continuous effects apply. This pattern is valid in `WhenEvent` and `WhenEventModes` on a battlefield permanent. The entering creature is not a target.
    ```go
    dsl.WhenEvent(
        dsl.AnotherCreatureYouControlEnters().Nontoken(),
        dsl.GainLife(dsl.You, dsl.Fixed(1)),
    )
    ```
    Add `.OfSubtype(string)` to require one creature subtype at entry. The string must be nonempty and must not name a subtype in another family. Filters can be combined:
    ```go
    dsl.WhenEvent(
        dsl.AnotherCreatureYouControlEnters().Nontoken().OfSubtype("Dragon"),
        dsl.GainLife(dsl.You, dsl.Fixed(1)),
    )
    ```
    Add `.WithPermanent(*PermanentBinding)` to save the entering battlefield incarnation. Read it with `From(binding)` in the trigger body. The reference does not follow the card if it leaves and returns. The binding is local to each trigger and is not a target.
    ```go
    entered := dsl.BindPermanent("entered creature")
    dsl.WhenEvent(
        dsl.AnotherCreatureYouControlEnters().WithPermanent(entered),
        dsl.AddCounters(dsl.From(entered), core.P1P1, dsl.Fixed(1)),
    )
    ```
  * Creature-death patterns support `.Nontoken()`: Match only a creature that was not a token before it died. For example, `dsl.WhenEvent(dsl.AnotherCreatureDies().Nontoken(), dsl.GainLife(dsl.You, dsl.Fixed(1)))` is valid on a battlefield permanent. This does not choose or target the dying creature.
  * Creature-death patterns support `.WithoutColor(core.Color)`: Match a creature that did not have the specified color immediately before it died. Supply one of the five Magic colors. This filter can combine with `.Nontoken()` and the existing event bindings. Use it with `WhenEvent` or `WhenEventModes` on a battlefield permanent.
    ```go
    dsl.WhenEvent(
        dsl.CreatureDies().WithoutColor(core.Black),
        dsl.AddCounters(dsl.This, core.Husk, dsl.Fixed(1)),
    )
    ```
  * `AttachedCreatureDies()`: The creature attached to the source dies. The event check uses the attachment's battlefield incarnation, so a returned copy of the same card does not match. It also matches when the source Aura leaves in the same batch.
  * Each creature-death pattern supports `.WithController(*PlayerBinding)`: Save the dying creature's controller before departure. Read the player with `FromPlayer` in the trigger body. This is an event participant, not a target; the binding is local to each trigger resolution.
  * Each creature-death pattern supports `.WithOwner(*PlayerBinding)`: Save the dying creature's owner at event detection. Read the player with `FromPlayer` in the trigger body. It works with `.WithController` and `.WithToughness` when each output has its own binding. It is available in `WhenEvent` and `WhenEventModes` on battlefield permanents.
    ```go
    owner := dsl.BindPlayer("dying creature owner")
    dsl.WhenEvent(
        dsl.AnotherCreatureDies().WithOwner(owner),
        dsl.GainLife(dsl.FromPlayer(owner), dsl.Fixed(1)),
    )
    ```
  * Each creature-death pattern supports `.WithToughness(*NumberBinding)`: Save the dying creature's toughness at event detection, including continuous effects. Read the number with `FromNumber` in the trigger body. The value remains fixed after the creature leaves or returns. This is an event value, not a target. It can combine with `.WithController`.
  * `DealsDamage(source PermanentRef, amountBinding *NumberBinding) EventPattern`: `This` or `AttachedPermanent` deals positive damage to any player or permanent. Use the pattern in `WhenEvent` or `WhenEventModes` on a battlefield permanent. The binding saves the final amount after prevention and replacement in that trigger's frame. `AttachedPermanent` checks the attachment when damage occurs; a later attachment or controller change does not change a queued trigger. Registration rejects other permanent references, nil or unnamed amount bindings, and use outside a battlefield trigger.
    ```go
    func linkedAura() dsl.Card {
        dealt := dsl.BindNumber("damage dealt")
        return dsl.NewAura("Linked Aura", "{W}", dsl.WithAbility(
            dsl.WhenEvent(dsl.DealsDamage(dsl.AttachedPermanent, dealt),
                dsl.GainLife(dsl.You, dsl.FromNumber(dealt))),
        ))
    }
  * `DealsDamageToPlayer(source PermanentRef, playerBinding *PlayerBinding).WithAmount(amountBinding)`: `This` or `AttachedPermanent` deals positive damage to a player. The player binding saves the damaged player, not a target. `WithAmount` optionally saves the final damage amount. Use this pattern in `WhenEvent` or `WhenEventModes` on a battlefield permanent. An attached source is checked when damage occurs; queued bindings stay fixed if the attachment or its controller later changes. Registration rejects other source references and invalid bindings.
  * `DealsDamageToPlayer(source, playerBinding).ToController().WithAmount(amountBinding)`: Filter the player recipient to the trigger controller when damage occurs. `ToController` must follow `DealsDamageToPlayer`; registration rejects it on opponent or combat damage patterns. The damaged player and final amount remain event bindings. Use `This` or `AttachedPermanent` as the source in a battlefield `WhenEvent` or `WhenEventModes` ability.
    ```go
    func attachedDamageWatcher() dsl.Card {
        player := dsl.BindPlayer("damaged player")
        amount := dsl.BindNumber("damage dealt")
        return dsl.NewAura("Damage Watcher", "{W}", dsl.WithAbility(
            dsl.WhenEvent(
                dsl.DealsDamageToPlayer(dsl.AttachedPermanent, player).ToController().WithAmount(amount),
                dsl.GainLife(dsl.FromPlayer(player), dsl.FromNumber(amount)),
            ),
        ))
    }
    ```
  * `DealsDamageToPlayer(source, playerBinding).WithSourceController(controllerBinding)`: Save the damage source's controller when positive damage occurs. Use `This` or `AttachedPermanent` in a battlefield `WhenEvent` or `WhenEventModes` ability. The controller and damaged player are separate event bindings, not targets. `ToController` and `WithAmount` can be combined; call `WithSourceController` before `WithAmount`. Prevention, departure, return, and later control changes do not change a queued capture. Registration rejects nil, repeated, shared, unnamed, and out-of-scope bindings. This option is not supported on other damage patterns.
    ```go
    func sourceControllerWatcher() dsl.Card {
        player := dsl.BindPlayer("damaged player")
        controller := dsl.BindPlayer("source controller")
        amount := dsl.BindNumber("damage dealt")
        return dsl.NewAura("Source Controller Watcher", "{U}", dsl.WithAbility(
            dsl.WhenEvent(
                dsl.DealsDamageToPlayer(dsl.AttachedPermanent, player).ToController().WithSourceController(controller).WithAmount(amount),
                dsl.DealDamageFrom(dsl.This, dsl.FromPlayer(controller), dsl.FromNumber(amount)),
            ),
        ))
    }
    ```
  * `DealsCombatDamageToPlayer(This, playerBinding).WithAmount(amountBinding)`: A battlefield source deals combat damage to a player. Use it in `WhenEvent` or `WhenEventModes`. It saves the damaged player and, when requested, the final positive amount after prevention. The player is an event participant, not a target. Registration rejects nil, unnamed, or out-of-scope bindings. The amount capture is optional.
    ```go
    func combatWatcher() dsl.Card {
        player := dsl.BindPlayer("damaged player")
        amount := dsl.BindNumber("combat damage")
        return dsl.NewCreature("Combat Watcher", "{2}{G}", 2, 2,
            dsl.WithAbility(dsl.WhenEvent(
                dsl.DealsCombatDamageToPlayer(dsl.This, player).WithAmount(amount),
                dsl.GainLife(dsl.FromPlayer(player), dsl.FromNumber(amount)),
            )))
    }
    ```
  * `DealsDamageToOpponent(source, playerBinding)`: Source deals damage to opponent.
  * `ControllerIsDealtDamage(amountBinding) EventPattern`: A battlefield source's controller is dealt positive damage by any source. The pattern saves the actual amount after damage replacement and prevention. Use `FromNumber(amountBinding)` in a `WhenEvent` or `WhenEventModes` body. The damaged player is not a target. Registration rejects a nil or unnamed binding and a read outside the trigger.
    ```go
    func damageWatcher() dsl.Card {
        amount := dsl.BindNumber("damage dealt")
        return dsl.NewEnchantment("Damage Watcher", "{G}", dsl.WithAbility(
            dsl.WhenEvent(dsl.ControllerIsDealtDamage(amount),
                dsl.GainLife(dsl.You, dsl.FromNumber(amount))),
        ))
    }
    ```
  * `ReceivesDamage(This, amountBinding).WithSourceController(controllerBinding)`: A battlefield source permanent receives positive damage. The pattern saves the final amount and, when requested, the damage source's controller at event time. Use it in `WhenEvent` or `WhenEventModes`. The controller is an event participant, not a target. The amount binding is required; the controller capture is optional. Registration rejects nil, unnamed, and out-of-scope bindings.
    ```go
    func damageRecipient() dsl.Card {
        amount := dsl.BindNumber("damage")
        controller := dsl.BindPlayer("source controller")
        return dsl.NewCreature("Damage Recipient", "{4}{U}", 2, 5,
            dsl.WithAbility(dsl.WhenEvent(
                dsl.ReceivesDamage(dsl.This, amount).WithSourceController(controller),
                dsl.Mill(dsl.FromPlayer(controller), dsl.FromNumber(amount)),
            )))
    }
    ```
  * `ControllerDrawsCard(playerBinding)`: Source controller draws a card.
  * `OpponentDrawsCard(playerBinding) EventPattern`: An opponent of the source controller draws one card. The pattern saves the drawer in a `PlayerBinding` when the draw occurs. Read it with `FromPlayer` in a `WhenEvent` or `WhenEventModes` body. A draw that is replaced or fails on an empty library does not match. The saved player is not a target.
    ```go
    func drawWatcher() dsl.Card {
        drawer := dsl.BindPlayer("drawer")
        return dsl.NewEnchantment("Draw Watcher", "{B}{B}{B}", dsl.WithAbility(
            dsl.WhenEvent(dsl.OpponentDrawsCard(drawer),
                dsl.DealDamageFrom(dsl.This, dsl.FromPlayer(drawer), dsl.Fixed(1))),
        ))
    }
    ```
  * `OpponentDiscardsCard(playerBinding) EventPattern`: An opponent of the battlefield source controller discards one card. Each completed discard triggers once, including a discard moved to another zone by a replacement. The pattern saves the discarding player at event time; a failed discard does not trigger. Use it with `WhenEvent` or `WhenEventModes`. The saved player is not a target.
    ```go
    func discardWatcher() dsl.Card {
        discarder := dsl.BindPlayer("discarder")
        return dsl.NewEnchantment("Discard Watcher", "{1}{B}", dsl.WithAbility(
            dsl.WhenEvent(dsl.OpponentDiscardsCard(discarder),
                dsl.LoseLife(dsl.FromPlayer(discarder), dsl.Fixed(2))),
        ))
    }
    ```
  * `ControllerGainsLife(playerBinding)`: Source controller gains a positive amount of life. Saves that player at event detection without declaring a target. A replaced gain triggers only if life is actually gained. Copies retain the saved player while `You` uses the copy's controller.
  * `OpponentLosesLife(playerBinding)`: Opponent loses life (can chain `.WithAmount(binding)`).
  * `AttachedControllerUpkeep(hostBinding, playerBinding)`: Upkeep of enchanted permanent's controller.
  * `AttachedBecomesTapped(hostBinding)`: Captures the attached permanent when it becomes tapped. Read with `From(hostBinding)`; it is not a target. Attachment moves and source departure do not change the captured incarnation.

For a synthetic creature-death trigger, the same declaration supplies the event value and the body read:

```go
func toughnessWatcher() dsl.Card {
    toughness := dsl.BindNumber("former toughness")
    return dsl.NewCreature("Toughness Watcher", "", 1, 1,
        dsl.WithAbility(dsl.WhenEvent(
            dsl.CreatureDies().WithToughness(toughness),
            dsl.GainLife(dsl.You, dsl.FromNumber(toughness)))))
}
```

An Aura can capture the attached creature's former controller and toughness for one trigger resolution:

```go
package example

import "github.com/benprew/mage-go/pkg/mage/dsl"

func bondWatcher() dsl.Card {
    controller := dsl.BindPlayer("former controller")
    toughness := dsl.BindNumber("former toughness")
    return dsl.NewAura("Bond Watcher", "{1}{U}",
        dsl.WithAbility(dsl.WhenEvent(
            dsl.AttachedCreatureDies().WithController(controller).WithToughness(toughness),
            dsl.DealDamageFrom(dsl.This, dsl.FromPlayer(controller), dsl.FromNumber(toughness)))))
}
```

The pattern is valid in `WhenEvent` and `WhenEventModes` on a battlefield source. The captured values are available in the resolving body after the Aura or creature leaves.

For a source combat trigger:

```go
package example

import "github.com/benprew/mage-go/pkg/mage/dsl"

func combatWatcher() dsl.Card {
    return dsl.NewCreature("Combat Watcher", "{1}{G}", 2, 2,
        dsl.WithAbility(dsl.WhenEvent(dsl.Blocks(dsl.This),
            dsl.Until(dsl.EndOfTurn, dsl.ModifyPT(dsl.This, dsl.Fixed(2), dsl.Fixed(2))))))
}
```

The trigger keeps its controller if the source leaves before resolution. The boost applies only to the original battlefield incarnation. `Attacks` and `Blocks` accept the sealed source role; they do not declare a target.

For a source that rewards another controlled creature's death:

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage/core"
    "github.com/benprew/mage-go/pkg/mage/dsl"
)

func deathWatcher() dsl.Card {
    return dsl.NewCreature("Death Watcher", "{1}{G}", 2, 2,
        dsl.WithAbility(dsl.WhenEvent(dsl.AnotherCreatureYouControlDies(),
            dsl.Sequence(dsl.GainLife(dsl.You, dsl.Fixed(1)),
                dsl.AddCounters(dsl.This, core.P1P1, dsl.Fixed(1))))))
}
```

The pattern is valid in `WhenEvent` and `WhenEventModes` on a battlefield permanent. The trigger can resolve after its source leaves. A source counter applies only if the original incarnation is still present.

To use the former controller after a creature dies:

```go
package example

import "github.com/benprew/mage-go/pkg/mage/dsl"

func deathControllerWatcher() dsl.Card {
    former := dsl.BindPlayer("former controller")
    return dsl.NewCreature("Death Controller Watcher", "{1}{B}", 1, 1,
        dsl.WithAbility(dsl.WhenEvent(dsl.CreatureDies().WithController(former),
            dsl.GainLife(dsl.FromPlayer(former), dsl.Fixed(1)))))
}
```

The binding uses the dying permanent's controller at the death event. A later change of control or a stack copy does not change that saved player. The pattern still requires a battlefield source.

### State Triggers (`WhenState`)
Triggers whenever game state becomes true (CR 603.8):
* `WhenState(StateCondition, ResolvingEffect, ...TriggerOption)`
* Arming & Suppression: Triggers immediately when state is true (even between resolution instructions). Stays suppressed while its instance is pending, on the stack, or resolving. Rearms after that instance leaves the stack.

### Turn-Step Triggers (`At` and `AtNext`)
Triggers at step transitions:
* `At(TurnStepPattern, ResolvingEffect, ...TriggerOption)`: Recurring step triggers (`YourUpkeep`, `EachUpkeep`, `EachEndStep`, `YourFirstMainPhase`, `YourPostcombatMainPhase`).
* `AtNext(DelayedStepPattern, ResolvingEffect, ...DelayedStepOption)`: **One-shot delayed trigger** installed during resolution.
  * Options: `WithTargets(...)`, `WithInterveningIf(condition)`.
  * `TurnStepBegins(step, playerBinding)`: Captures the future step's active player.

### Modal Triggers
* `WhenEventModes(pattern, modes...)`: Choose a mode when the trigger is placed on stack.
* `AtModes(stepPattern, modes...)`: Modal turn-step trigger.
* `AtNextModes(stepPattern, modalOptions...)`: Modal one-shot delayed step trigger.

---

## 10. Spells, Activations, and Costs

### Spells
* `Spell(ResolvingEffect, ...SpellOption)`: Instants and sorceries.
* `SpellModes(modes...)`: Modal spells.
* Options: `WithTargets(...)`, `WithAdditionalCost(cost)`, `WithSelfCostReduction(condition, amount)`.

### Activated Abilities
* `Activate(CostGroup, ResolvingEffect, ...ActivationOption)`: Standard activation.
* `ActivateModes(CostGroup, modes...)`: Choose-one activated ability.
* Costs (`Costs(...)`):
  * `Mana(costString)`: Mana payment (e.g. `Mana("{1}{U}")`).
  * `TapCost(This)`: {T} tap cost.
  * `SacrificeSourceCost()`: Sacrifice source as cost.
  * `SacrificeCreatureCostWithStats(power, toughness *NumberBinding)`: Sacrifice one creature and save its current power and toughness during payment. Both bindings are required. `FromNumber` reads either value in the resolving body. This cost works in `Activate`, `ActivateModes`, and spell additional costs. An activation excludes its source from the choice.
  * `SacrificeMatchingCostWithCard(filter PermanentFilter, text string, card *CardBinding) Cost`: Sacrifice one other matching permanent and save its card in the new graveyard visit. Use `FromCard(card)` in the resolving body of `Spell`, `Activate`, or `ActivateModes`. A token has no card in the graveyard, so its saved reference is absent. A later zone move also makes the saved reference absent. Registration requires a named card binding and nonempty cost text.

    ```go
    paid := dsl.BindCard("paid creature")
    spell := dsl.NewInstant("Offering", "",
        dsl.WithAbility(dsl.Spell(
            dsl.IfCardHasType(dsl.FromCard(paid), core.TypeCreature,
                dsl.GainLife(dsl.You, dsl.Fixed(2))))),
        dsl.WithAdditionalCost(dsl.SacrificeMatchingCostWithCard(
            dsl.IsCreature, "Sacrifice a creature", paid)))
    _ = spell
    ```
  * `LifePayCost(amount)`: Pay life (permits paying exact life total CR 119.4).
  * `LifePayCostInto(amount int, paid *NumberBinding) Cost`: Pay a fixed amount of life and save the committed amount for the resolving ability. Use it in `Activate`, `ActivateModes`, or a spell additional cost. Registration rejects a negative amount, a nil or unnamed binding, and duplicate producers. It is not a resolution-time `IfPaid` or `UnlessPaid` cost. Use `FromNumber(paid)` in the body; zero life is a valid payment.
  * `LifePayXCostInto(paid *NumberBinding) Cost`: Pay the announced X life as a direct activation cost or direct spell additional cost. The cost locks X before payment and saves the committed amount for `FromNumber(paid)` in that stack instance. X must be nonnegative and cannot exceed the payer's life total. This is not a resolution-time payment.

    ```go
    paid := dsl.BindNumber("life paid")
    spell := dsl.NewInstant("Variable life", "",
        dsl.WithAbility(dsl.Spell(dsl.GainLife(dsl.You, dsl.FromNumber(paid)))),
        dsl.WithAdditionalCost(dsl.LifePayXCostInto(paid)))
    _ = spell
    ```
  * `RemoveCountersCost(counter, count)`: Remove counters from source.
  * `RemoveCountersCostInto(counter core.CounterType, count int, paid *NumberBinding) Cost`: Remove a fixed nonnegative number of counters from the source as an activated ability cost. `FromNumber(paid)` reads the committed count in that activation's resolving body. Payment locks the source battlefield incarnation; failed payment saves no value. Registration rejects a nil or unnamed binding, a negative count, duplicate producers, and use as a spell additional cost or resolution-time optional cost.

    ```go
    paid := dsl.BindNumber("paid counters")
    ability := dsl.Activate(
        dsl.Costs(dsl.RemoveCountersCostInto(core.Charge, 2, paid)),
        dsl.GainLife(dsl.You, dsl.FromNumber(paid)),
    )
    _ = ability
    ```
  * `RemoveAnyCountersCost(counter core.CounterType, paid *NumberBinding) Cost`: In an activated ability cost, the payer chooses a number from zero through the source's current count and removes exactly that many counters. `FromNumber(paid)` reads the committed count in the resolving body. The choice is made once during payment preparation. The cost requires a battlefield source and cannot be used as a spell additional cost or an `IfPaid`/`UnlessPaid` cost.
  * `DiscardCost(n int) Cost`: Discard exactly `n` cards from the payer's hand. Use it in `Costs(...)` or as a spell additional cost. The payer's card choice is locked during total-cost validation and reused at commit. Each chosen discard uses the shared replacement action and emits a discard event only when a card is discarded. A replacement that applies only when an effect causes a discard does not apply to this cost.
  * `DiscardCardCostInto(card *CardBinding) Cost`: Discard one chosen card and save its new graveyard visit in the spell or activation frame. Use `FromCard(card)` in the resolving body. The reference is absent if a replacement sends the card to a different zone or it leaves the graveyard before resolution. Registration requires a named binding and a dominating payment producer. This cost uses the same locked choice and replacement path as `DiscardCost(1)`.
  * `DiscardRandomCost(count)`: Discard cards at random.

```go
ability := dsl.Activate(
    dsl.Costs(dsl.DiscardCost(1)),
    dsl.Draw(dsl.You, dsl.Fixed(1)),
)
_ = ability
```

```go
paid := dsl.BindCard("discarded land")
ability := dsl.Activate(
    dsl.Costs(dsl.DiscardCardCostInto(paid)),
    dsl.IfCardHasType(dsl.FromCard(paid), core.TypeLand,
        dsl.GainLife(dsl.You, dsl.Fixed(2))),
)
_ = ability
```

```go
paid := dsl.BindNumber("life paid")
ability := dsl.Activate(
    dsl.Costs(dsl.LifePayCostInto(3, paid)),
    dsl.GainLife(dsl.You, dsl.FromNumber(paid)),
)
```

```go
paid := dsl.BindNumber("paid charge counters")
ability := dsl.Activate(
    dsl.Costs(dsl.RemoveAnyCountersCost(core.Charge, paid)),
    dsl.GainLife(dsl.You, dsl.FromNumber(paid)),
)
_ = ability
```

```go
package example

import . "github.com/benprew/mage-go/pkg/mage/dsl"

func lifeForSacrificedToughness() {
    power := BindNumber("paid power")
    toughness := BindNumber("paid toughness")
    ability := Activate(
        Costs(SacrificeCreatureCostWithStats(power, toughness)),
        GainLife(You, FromNumber(toughness)),
    )
    _ = ability
}

func spellUsingSacrificedPower() {
    power := BindNumber("paid power")
    toughness := BindNumber("paid toughness")
    spell := NewInstant("Offering", "{1}{R}",
        WithAbility(Spell(GainLife(You, FromNumber(power)))),
        WithAdditionalCost(SacrificeCreatureCostWithStats(power, toughness)))
    _ = spell
}
```

For a tap ability that prevents blocking, compose the target, cost, and duration:

```go
creature := TargetPermanent("creature", And(IsCreature, HasPowerLTE(2)))
WithAbility(Activate(Costs(TapCost(This)),
    Until(EndOfTurn, CantBeBlocked(creature)), WithTargets(creature)))
```

Dwarven Warriors uses this form. Suspicious Bookcase omits the power filter and
adds `Mana("{3}")` to `Costs`. Target restrictions are checked on resolution;
a power increase after resolution does not end the blocking restriction.

* Activation Options:
  * `WithTargets(...)`: Declares target bindings.
  * `OnlyAsSorcery()`: Sorcery-speed timing.
  * `OnlyDuring(step, moreSteps...)`: Restricts casting or activation to any specified step (e.g. `DeclareAttackers` or `Upkeep`). Returns `StepTimingOption`, which is accepted by `Spell` and all activation contexts. It adds a restriction; it does not grant permission to cast a sorcery outside normal timing.
  * `OnlyOnYourTurn()`: Active player only.
  * `ActivationLimit(maxPerTurn)`: Max activations per turn.
  * `AnyPlayerMayActivate()`: Permits either player to activate and pay for
    the ability. `You` refers to the activator at resolution. Other costs,
    timing rules, and restrictions still apply. Use this option with `Activate`.
    Feral Hydra combines it with `Costs(Mana("{3}"))` and
    `AddCounters(This, P1P1, Fixed(1))`; the counter applies only to the
    original source incarnation, even when the opponent activates the ability.
  * `ActivationIf(condition)`: Pure state condition required to activate.
  * `CaptureNumberBeforePayment(value, binding)`: Saves amount before mana production & cost payment.
  * `CapturePermanentBeforePayment(subject, binding)`: Saves permanent (e.g. attachment) before sacrifice cost.

`SacrificeMatchingCostWithManaValue(filter, text, binding) Cost` chooses one matching permanent that the payer controls, except the source, and sacrifices it as an activation cost or spell additional cost. It saves that permanent's mana value at payment time in the `NumberBinding`. Use `FromNumber` in the resolving body. The binding belongs to that spell or ability instance, including copies. It is not available in `IfPaid` or `UnlessPaid` resolution-time costs. The text and binding must be present at registration. A zero-value filter matches all permanents.

```go
func manaValueOffering() dsl.Card {
    paid := dsl.BindNumber("paid mana value")
    return dsl.NewArtifact("Mana Value Offering", "", dsl.WithAbility(
        dsl.Activate(
            dsl.Costs(dsl.SacrificeMatchingCostWithManaValue(
                dsl.IsArtifact, "Sacrifice an artifact", paid)),
            dsl.GainLife(dsl.You, dsl.FromNumber(paid)))))
}

---

## 11. Continuous Effects, Modifications, and Durations

Continuous effects are broken down into **modifications** and their **durations**:

### Continuous Modifications (`ContinuousModification`)
* `ModifyPT(subject, powerValue, toughnessValue)`: Power/toughness boost (Layer 7c). For a repeatable self-pump activation, use `Activate(Costs(Mana("{R}")), Until(EndOfTurn, ModifyPT(This, Fixed(1), Fixed(0))))`. Each resolution adds a separate bonus to the original source incarnation. Frozen Shade, Granite Gargoyle, Shivan Dragon, Wall of Fire, and Wall of Water use this form.
  For targeted artifact activations, declare `TargetPermanent("creature", IsCreature)`, pass it to `ModifyPT` and `WithTargets`, and group all payments in `Costs`. Coral Helm uses `Mana("{3}")` and `DiscardRandomCost(1)`; Elven Lyre uses `Mana("{1}")`, `TapCost(This)`, and `SacrificeSourceCost()`. Both install +2/+2 with `Until(EndOfTurn, ...)`; the boost does not require the source to remain on the battlefield.
  For Atog's sacrifice payment, use `Costs(SacrificeMatchingIncludingSourceCost(IsArtifact, "Sacrifice an artifact"))`. This permits sacrificing the source if it is an artifact. `SacrificeArtifactCost` excludes the source and does not express that requirement. Dragon Engine uses `Costs(Mana("{2}"))` with the same typed self-pump form.
  Dragon Hatchling, Furnace Whelp, and Molten Ravager use `{R}` for +1/+0; Minotaur Sureshot uses `{1}{R}` for +1/+0; Jousting Dummy uses `{3}` for +1/+0; Tibalt’s Rager uses `{1}{R}` for +2/+0. Each uses the same untargeted `This` modification with an explicit end-of-turn duration.
  Carrion Ants uses `{1}` for +1/+1; Wall of Opposition uses `{1}` for +1/+0; Killer Bees uses `{G}` for +1/+1; Arcades Sabboth uses `{W}` for +0/+1; Pavel Maliki uses `{B}{R}` for +1/+0. Vaevictis Asmadi has three separate +1/+0 activations costing `{B}`, `{R}`, and `{G}`. Each uses the same typed source modification and explicit duration.
  Fire Drake and Mindful Biomancer add `ActivationLimit(1)` to their typed source-boost activations; Vampire Bats adds `ActivationLimit(2)`. Use the limit as an `Activate` option, separate from the end-of-turn duration. Costs are `{R}`, `{2}{G}`, and `{B}`, respectively.
  Fallen Angel uses `Activate(Costs(SacrificeMatchingIncludingSourceCost(IsCreature, "Sacrifice a creature")), Until(EndOfTurn, ModifyPT(This, Fixed(2), Fixed(1))))`. Its cost permits sacrificing Fallen Angel itself; `SacrificeCreatureCost` excludes the source.
  Osai Vultures uses `Activate(Costs(RemoveCountersCost(Carrion, 2)), Until(EndOfTurn, ModifyPT(This, Fixed(1), Fixed(1))))`. Remove the counters as payment before the bonus resolves. The ability has no mana or tap cost, targets, or per-turn limit.
  Mishra’s Factory and Pendelhaven use the same typed activation with `Costs(TapCost(This))`. Their target filters require an Assembly-Worker creature and a creature with current power and toughness each equal to 1, respectively. These restrictions are checked at activation and resolution; they do not end a resolved bonus. Either player’s qualifying creature can be targeted.
  Ghosts of the Damned, Pradesh Gypsies, and Tuknir Deathlock use `TargetPermanent("creature", IsCreature)` with `ModifyPT` and `Until(EndOfTurn, ...)` for -1/-0, -2/-0, and +2/+2, respectively. All three include `TapCost(This)` in `Costs`; Pradesh adds `Mana("{1}{G}")` and Tuknir adds `Mana("{R}{G}")`. Either player’s creature, including the source, can be targeted.
  For an entry boost, use `WhenEvent(Enters(This), Until(EndOfTurn, ModifyPT(creature, Fixed(0), Fixed(3))), WithTargets(creature))` with `TargetPermanent("creature", IsCreature)`. Affa Guard Hound uses this form. Declare the target on the trigger, not with `WithCastTarget`: choose it when the trigger goes on the stack, after the creature enters. The creature can target itself, and entry without casting also triggers the ability.
  Angelic Page combines `IsCreature` with `Or(IsAttacking, IsBlocking)` in its typed target and pays `Costs(TapCost(This))` for +1/+1. Either player’s attacking or blocking creature qualifies. The target must still qualify at resolution, but leaving combat afterward does not end the end-of-turn bonus.
  Dwarven Lieutenant uses `TargetPermanent("dwarf", And(IsCreature, HasSubType("Dwarf")))`, `Costs(Mana("{1}{R}"))`, and `Until(EndOfTurn, ModifyPT(dwarf, Fixed(1), Fixed(0)))`. Either player's Dwarf creature, including the source, qualifies. The subtype restriction is checked at activation and resolution; losing the subtype after resolution does not end the bonus.
  Icatian Lieutenant uses the same subtype-filtered form for Soldier creatures, with `{1}{W}` and +1/+0. Icatian Priest uses `IsCreature`, `{1}{W}{W}`, and +1/+1. Both can target either player's qualifying creature, including themselves, and have no tap cost or activation limit.
* `ModifyEachPT(permanentSet, powerValue, toughnessValue)`: Collection power/toughness modification (Layer 7c). For “creatures you control,” use `ControlledPermanents(IsCreature)`. Install with `Until(EndOfTurn, ...)` to capture membership at resolution; later entries and returned cards do not receive the boost. Fortify uses this in `SpellModes`, Inspired Charge and Shield Wall in `Spell`, and Assault Formation in `Activate`. For attacking or blocking creatures, use `MatchingPermanents(And(IsCreature, IsAttacking))` or `MatchingPermanents(And(IsCreature, IsBlocking))`; Army of Allah, Morale, and Piety use these forms. The collection includes either player’s creatures. Combat membership is checked at resolution, and leaving combat afterward does not end the bonus.
  Inspiring Captain and Inspiring Unicorn use the same controlled collection for their +1/+1 trigger bodies. Captain installs it with `WhenEvent(Enters(This), ...)`; Unicorn retains `AttacksTrigger` until a typed attack-event adapter is available. Both capture creatures at resolution and expire at cleanup.
* `ChangeController(subject, player)`: Changes one permanent's controller in Layer 2. Install it with `Until(EndOfTurn, ChangeController(creature, You))`. The effect captures the permanent incarnation and the evaluated player when the ability resolves. It survives source departure, retains the target through phasing, and expires at cleanup. Registration rejects this modification inside `While`, `UntilSourceLeaves`, or another `Until` duration.
* `GrantKeyword(subject, keyword)`: Grants keyword ability (Layer 6).
  Blighted Bat uses `Activate(Costs(Mana("{1}")), Until(EndOfTurn, GrantKeyword(This, Haste)))`. The ability has no target, tap cost, or activation limit. Its printed flying ability remains separate.
  Jump uses `Spell(Until(EndOfTurn, GrantKeyword(creature, Flying)), WithTargets(creature))` with `TargetPermanent("creature", IsCreature)`. Pixie Queen uses the same body in `Activate` with `Costs(Mana("{G}{G}{G}"), TapCost(This))`. Either player's creature can be targeted, including Pixie Queen itself. The resolved grant remains after the source leaves and expires at cleanup.
  Icatian Scout and Svyelunite Priest use a `TargetPermanent("creature", IsCreature)` binding with `WithTargets` and `Until(EndOfTurn, GrantKeyword(creature, ...))`. Both pay mana and `TapCost(This)`. The Priest combines `OnlyDuring(core.Upkeep)` with `OnlyOnYourTurn()` to restrict activation to its controller's upkeep. Either player's creature can be targeted, including the source.
  Order of Leitbur and Order of the Ebon Hand use `Activate(Costs(Mana("{W}")), Until(EndOfTurn, GrantKeyword(This, FirstStrike)))` and the corresponding `{B}` cost. Their separate +1/+0 activations use `ModifyPT(This, Fixed(1), Fixed(0))` with `{W}{W}` or `{B}{B}` and the same explicit end-of-turn duration.
  Icatian Infantry uses separate `{1}` activations for first strike and banding with the same typed source grant and duration. River Merfolk grants mountainwalk for `{U}`, Vodalian Knights grants flying for `{U}`, and Orcish Veteran grants first strike for `{R}`. These activations have no targets, tap costs, or per-turn limits.
  Alabaster Mage uses `TargetPermanent("creature", IsCreature, TargetYouControl())`, `{1}{W}`, and a temporary lifelink grant. Check the controller restriction at activation and resolution; a later control change does not end the resolved grant. Emerald Dragonfly uses `{G}{G}` and `GrantKeyword(This, FirstStrike)` with the same end-of-turn duration, keeping its printed flying ability. Neither activation has a tap cost or activation limit.
* `RemoveKeyword(subject, keyword)`: Removes all instances of one keyword in Layer 6. Use `Until(EndOfTurn, RemoveKeyword(creature, Forestwalk))` for Scarwood Hag's second activation. Declare the creature with `TargetPermanent` and `WithTargets`; pay `Costs(TapCost(This))`. The first activation uses `GrantKeyword` and adds `Mana("{G}{G}{G}{G}")`. Both can target either player's creature, including the Hag. Removal uses timestamp order, so a later grant can restore the keyword. A resolved removal survives source departure, preserves the recipient through phasing, expires at cleanup, and does not affect a returned incarnation.
* `RemoveLandwalk(subject)`: Removes all landwalk abilities in layer 6, including legendary landwalk and subtype-based landwalk. Hammerheim uses `Activate(Costs(TapCost(This)), Until(EndOfTurn, RemoveLandwalk(creature)), WithTargets(creature))` with `TargetPermanent("creature", IsCreature)`. Either player's creature can be targeted. Later grants restore landwalk in timestamp order; unrelated abilities remain. The resolved removal survives source departure and recipient phasing, expires at cleanup, and does not affect a returned incarnation.
* `RemoveAbilities(PermanentRef)`: Removes all abilities from one permanent in Layer 6.
* `RemoveAbilitiesEach(PermanentSet)`: Removes all abilities from each permanent in a set. `Until` captures the set at resolution; `While` reevaluates it while active. Later Layer 6 grants can restore abilities. Rules restrictions, such as regeneration prohibitions, remain in effect.
* `CantBeBlocked(subject)`: Prevents all creatures from blocking the permanent. Install with `Until(EndOfTurn, CantBeBlocked(subject))`; this is a rules restriction, so ability removal does not erase it. Later blockers are also restricted. Source departure does not end the duration, and a returned subject is a new object.
* `GrantEachKeyword(permanentSet, keyword)`: Grants keyword to set.
* `SetColors(subject, colors...)`: Replaces colors (Layer 5).
* `SetEachColors(permanentSet, colors...)`: Replaces colors for set.
* `SetSubtypeFamily(subject, family, subtypes...)`: Replaces one subtype family (Layer 4). Wishful Merfolk combines `SetSubtypeFamily(This, SubtypeCreature, "Human")` and `RemoveKeyword(This, Defender)` in one `Until(EndOfTurn, ...)` activation. This replaces its creature subtypes and removes defender only on the original source incarnation. Both changes expire at cleanup.
* `AddSubtypeFamily(subject, family, subtypes...)`: Adds subtypes within a family (Layer 4).
* `ActivatedAbilityModification(subject, costs, effect, options...)`: Grants activated ability (Layer 6).
* `ActivatedModesModification(subject, costs, options...)`: Grants modal activated ability.

For “all creatures get -N/-M until end of turn,” use
`Until(EndOfTurn, ModifyEachPT(MatchingPermanents(IsCreature), Fixed(-N), Fixed(-M)))`.
Hell Swarm, Marsh Gas, Languish, and Bone Flute use this form. Both players'
creatures are included, with no targets. Bone Flute installs the effect through
`Activate(Costs(Mana("{2}"), TapCost(This)), ...)`. The resolved effect remains
if the Flute leaves. Temporary-effect metadata reports a detrimental outcome
when its summed fixed power or toughness modifier is negative; signed values
and collection metadata remain available to AI inspection.

For “Target creature can't be regenerated this turn,” use a typed creature
target and `Until(EndOfTurn, GrantKeyword(creature, CantRegenerate))`. The
prohibition remains after the source leaves. It expires at cleanup and does
not prevent the creation of regeneration shields.

### Durations & Installation
* `Until(Duration, modifications...)`: Installs temporary continuous effects (e.g. `EndOfTurn`, `EndOfCombat`, `Indefinite`).
* `UntilSourceLeaves(This, modifications...)`: Effect lasts until the source leaves the battlefield.
* `While(condition, modifications...)`: Static conditional continuous effect.
* `Static(rules...)`: Installs static rules on the permanent.

### Static Ability Grants
* `GrantActivatedAbility(subject, costs, body, options...)`: Statically grants an ability to a single subject (e.g. `AttachedPermanent`).
* `GrantEachActivatedAbility(set, costs, body, options...)`: Statically grants an ability to each permanent in a set.
* `GrantEachActivatedModes(set, costs, options...)`: Statically grants a modal ability.

---

## 12. Replacements and Prevention

* `PreventNextDamage(recipient, amount)`: Installs a prevention shield on permanent or player.
* `SkipNextUntap(PermanentRef)`: Prevents untap during the controller's next untap step.
* `SkipNextUntapOf(PermanentRef, PlayerRef)`: Prevents untap during the specified player's next untap step.
* `SkipDrawStepForLife(amount)`: Static rule to skip draw step and gain life (CR 614.10b).
* `EntersWithNCounters(counter, count)`: Replacement effect placing counters as the permanent enters.
* `Regenerate(PermanentRef)`: Installs a regeneration shield on the target or source.

---

## 13. Values and Dynamic Calculations

Operations taking amounts accept `ResolutionValue`:

* `Fixed(n)`: Static constant integer.
* `XValue()`: The X announced during casting or activation.
* `ManaValueOf(PermanentRef)`: Current mana value from the permanent's card; absent references supply zero. X contributes zero on the battlefield. Use `SnapshotNumber` before moving the permanent when a later instruction needs that value. This adapter uses the existing `ManaCost.CMC()` service and does not add mana-symbol support.
* `ManaValueOfSpell(StackObjectRef)`: Current mana value of a spell on the stack. For example, `target := TargetSpell("spell"); ability := Spell(GainLifeFor(You, ManaValueOfSpell(target)), WithTargets(target))` compiles with the typed target and value roles. The value uses the stack object's announced X and the existing mana-cost parser; mana symbols that the parser cannot represent need a separate engine contract.
* `PowerOf(PermanentRef)`: Current power of the permanent at resolution.
* `ToughnessOf(PermanentRef)`: Current toughness of the permanent at resolution.
* `CountersOn(PermanentRef, CounterType)`: Current count of counters on the live permanent.
* `SourceCounters(CounterType)`: Counters on the source permanent, or its last-known count if departed.
* `CountPermanents(PermanentSet)`: Counts the current collection when the value is used. Controlled collections use the resolving controller and exclude phased-out permanents. Use `SnapshotNumber` to preserve an earlier count. Aerial Assault counts controlled flying creatures after its destruction instruction. Take Heart uses `CountPermanents(ControlledPermanents(And(IsCreature, IsAttacking)))` after its typed end-of-turn +2/+2 boost. Count current attackers when the gain resolves, independently of the creature target's controller or combat status.
* `CreaturesDiedThisTurn()`: Number of creatures that died this turn across all players.
* Arithmetic:
  * `MultiplyValues(a, b)`: The product of two resolution values. Use `MultiplyValues(Fixed(2), ValueAtResolution(XValue()))` for twice the announced X. Both operands keep their typed binding context.
  * `SubtractValues(a, b)`: $a - b$.
  * `MinValue(a, b)`: $\min(a, b)$.
  * `MaxValue(a, b)`: $\max(a, b)$.
* Adapters:
  * `ValueAtResolution(legacyValueSource)`: Adapts legacy value sources.

---

## 14. Representative Card Implementations

### Goblins of the Flarg (State Trigger)
```go
WithAbility(WhenState(
    Controls(HasSubType("Dwarf")),
    Sacrifice(This),
))
```

### Island-Dependent Creatures (State Trigger with Negation)
```go
WithAbility(WhenState(
    ControlsNone(HasSubType("Island")),
    Sacrifice(This),
))
```

### Blood Divination (Additional Cost Spell)
```go
Spell(
    Draw(You, Fixed(3)),
    WithAdditionalCost(SacrificeCreatureCost()),
)
```

### Targeted Land Destruction
```go
land := TargetPermanent("land", IsLand)
WithAbility(Spell(Destroy(land), WithTargets(land)))
```

The target can be a land controlled by either player. `Destroy` allows regeneration.

### Land Destruction with a Sacrifice Cost

Strip Mine uses `TargetPermanent("land", IsLand)` and
`Activate(Costs(TapCost(This), SacrificeSourceCost()), Destroy(land), WithTargets(land))`.
Select the target before paying the tap and sacrifice costs. Either player's
land can be targeted, including Strip Mine itself. The ability resolves after
the source leaves; a self-target is then absent. Its separate mana ability uses
`Activate(Costs(TapCost(This)), AddManaAmount(Colorless, Fixed(1)))`, which
resolves immediately and supports automatic mana planning.

### Filtered Destruction Activations

Royal Assassin uses `TargetPermanent("creature", And(IsCreature, IsTapped))`
and `Activate(Costs(TapCost(This)), Destroy(creature), WithTargets(creature))`.
Exorcist uses `And(IsCreature, HasColorFilter(Black))` and adds `Mana("{1}{W}")`
to the cost group. Either player's qualifying creature can be targeted. The
restriction is checked again at resolution; regeneration and indestructible
still apply.

Merfolk Assassin uses `And(IsCreature, HasKeywordFilter(Islandwalk))` with
`Costs(TapCost(This))`. Savaen Elves uses `IsAuraOnLand` with
`Costs(Mana("{G}{G}"), TapCost(This))`. `IsAuraOnLand` requires the Aura
subtype, the enchantment type, and an attachment to a current land permanent.
Neither the Aura nor its land has a controller restriction. Both activations
use `Destroy(target)` and `WithTargets(target)`; their filters are checked
again at resolution.

### Filtered Permanent Destruction
```go
WithAbility(Spell(DestroyEach(MatchingPermanents(IsLand))))
WithAbility(Spell(DestroyEach(MatchingPermanents(HasSubType("Island")))))
WithAbility(Spell(DestroyEach(MatchingPermanents(HasSubType("Plains")))))
WithAbility(Spell(DestroyEach(MatchingPermanents(HasSubType("Forest")))))
WithAbility(Spell(DestroyEach(MatchingPermanents(And(IsCreature, HasColorFilter(Black))))))
```

These spells have no targets. The collection includes both players' matching
permanents at resolution, excludes phased-out permanents, and is destroyed as
one simultaneous group. Regeneration and indestructible still apply. Island
matching uses the subtype, including nonbasic Islands, rather than the card name.
The same rule applies to Plains and Forests. Black creature matching includes
multicolored black creatures and excludes black permanents that are not creatures.

`ManaValueAtMost(PermanentSet, ResolutionValue) PermanentSet` keeps members
whose current mana value is no greater than the value read when the set is
used. It can filter a set for `DestroyEach` in a spell, activation, or trigger.
The set and value are required. `FromNumber` can read an announced cost payment
from the same spell or ability instance. For example, this synthetic spell
uses the X life paid during casting as its destruction threshold:

```go
paid := dsl.BindNumber("life paid")
spell := dsl.NewInstant("Sweep", "",
    dsl.WithAbility(dsl.Spell(dsl.DestroyEach(dsl.ManaValueAtMost(
        dsl.MatchingPermanents(dsl.Or(dsl.IsArtifact, dsl.IsCreature)),
        dsl.FromNumber(paid))))),
    dsl.WithAdditionalCost(dsl.LifePayXCostInto(paid)))
```

This filter reads the limit once per set evaluation. It does not target or
change destruction ordering. Existing `DestroyEach` replacement and batch
rules still apply.

### Fasting (Static Replacement + Upkeep Counters + Event Trigger)
```go
// Skip draw step for life
WithAbility(Static(SkipDrawStepForLife(2)))

// Put hunger counter, destroy if threshold reached
WithAbility(At(YourUpkeep, Sequence(
    AddCounters(This, Hunger, Fixed(1)),
    IfState(SourceHasCounters(Hunger, 5), Destroy(This)),
)))

// Destroy if controller draws a card
WithAbility(WhenEvent(
    ControllerDrawsCard(BindPlayer("drawer")),
    Destroy(This),
))
```

### Thrull Retainer (Activation Capture Before Sacrifice)
```go
host := BindPermanent("host")
WithAbility(Activate(
    Costs(SacrificeSourceCost()),
    Regenerate(From(host)),
    CapturePermanentBeforePayment(AttachedPermanent, host),
))
```

### Erosion (Attachment Event Capture + Payment Branch)
```go
land := BindPermanent("land")
player := BindPlayer("player")

WithAbility(WhenEvent(
    AttachedControllerUpkeep(land, player),
    UnlessPaid(FromPlayer(player),
        Costs(EitherCost(Mana("{1}"), LifePayCost(1))),
        Destroy(From(land)),
    ),
))
```

### Teleport (Spell Step Restriction)

```go
creature := TargetPermanent("creature", IsCreature)
Spell(
    Until(EndOfTurn, CantBeBlocked(creature)),
    WithTargets(creature),
    OnlyDuring(core.DeclareAttackers),
)
```

The casting restriction applies on either player's turn, before choices and
payment, including alternate-cost casts and casts of card copies. It does not
restrict resolution or copying a spell already on the stack. The target may be
any creature, including a creature that is not attacking.

### Base Power and Toughness

`Until(EndOfTurn, SetBasePowerToughness(creature, Fixed(0), Fixed(2)))`
sets both base values through the existing layer executor. It preserves bonuses,
counters, abilities, types, and colors. Temporary effects capture the subject's
incarnation and both values at resolution. `While` evaluates them during layer
evaluation. Base values are not reported as additive boosts in AI metadata.

`Until(EndOfTurn, SetBasePower(creature, Fixed(0)))` changes only base power.
Base toughness remains subject to other layer effects; it is not captured at
resolution. This replaces `SetPowerUntilEndOfTurn` for Singing Tree and Island
of Wak-Wak. The old UUID-based engine `SetBasePower` constructor is replaced
by this typed modification; install it with an explicit duration.

Disharmony uses `OnlyDuring(core.BeginCombat, core.DeclareAttackers)` for
“during combat before blockers are declared.” This timing restriction does not
remove its requirement for an attacking creature target. The constructor copies
the supplied steps; later caller changes do not change the definition.

### Aura Targets on Controlled Creatures

Miracle Worker uses `TargetPermanent("aura", And(IsEnchantment,
HasSubType("Aura"), IsAttachedTo(IsCreature)), TargetAttachedToYouControl())`
with `Activate(Costs(TapCost(This)), Destroy(aura), WithTargets(aura))`.
Either player's Aura qualifies if its current host is a creature you control.
The host is not a target. Check the attachment, host type, and host controller
again at resolution. Moving the Aura to another qualifying creature keeps it
legal; returning the Aura from another zone creates a new target incarnation.

### Enchanted Creature Destruction

Ramses Overdark uses `TargetPermanent("creature", And(IsCreature, IsEnchanted))`
with `Activate(Costs(TapCost(This)), Destroy(creature), WithTargets(creature))`.
`IsEnchanted` requires an attached Aura enchantment; Equipment does not count.
Either player's creature can qualify, regardless of the Aura's controller.
Check the restriction at activation and resolution. An Aura that leaves or
phases out no longer qualifies, but another attached Aura can keep the target
legal. Destruction permits regeneration and respects indestructible.

### Fixed Land Mana Activations

Hammerheim, Karakas, and Pendelhaven use
`Activate(Costs(TapCost(This)), AddManaAmount(color, Fixed(1)))` for red,
white, and green mana, respectively. These untargeted abilities resolve
immediately, emit the tapped-for-mana event, and support automatic payment.
Their separate utility abilities use the stack.

Desert, Elephant Graveyard, and Library of Alexandria use the same declaration
with `Colorless` for `{T}: Add {C}`. The mana activation has no target, step,
or hand-size restriction. Those restrictions belong to the separate utility
abilities. Automatic payment can use the fixed colorless production.

The test harness prefers non-mana abilities using the engine's
`IsActivatedManaAbility` classification, including typed fixed and dynamic
production. Use `ActivateAbilityIndex` to select a particular mana ability
when a permanent has multiple activated abilities.

Guardian Idol and Mana Crypt use `Activate(Costs(TapCost(This)),
AddManaAmount(Colorless, Fixed(n)))` with one and two mana, respectively.
These untargeted abilities resolve immediately and support automatic payment.
Guardian Idol's entry-tapped rule prevents payment of the tap cost until it
untaps. Its animation ability and Mana Crypt's upkeep ability are separate.

### Optional Entry Target with a Dependent Value

Brightmare declares `TargetPermanents("creature", 0, 1, IsCreature)` on
`WhenEvent(Enters(This), ..., WithTargets(creatures))`. It uses
`ForEach(creatures, creature, Sequence(Tap(From(creature)),
GainLife(You, PowerOf(From(creature)))))` with a `BindPermanent` member.
The collection has at most one member, so the loop preserves the singular
instruction without an implicit first-member conversion. No chosen target means
no tap or life gain. The target is chosen when the entry trigger goes on the
stack, and Brightmare itself can be chosen. An already tapped target still
supplies its current power for life gain. An illegal target prevents resolution.

### Sacrifice for a Source Counter and Life Gain

Blood Host uses `Activate(Costs(Mana("{1}{B}"), SacrificeCreatureCost()),
Sequence(AddCounters(This, P1P1, Fixed(1)), GainLife(You, Fixed(2))))`.
The sacrifice cost requires another creature. The ability has no targets or
tap cost. At resolution, add the counter to the original source incarnation,
then gain life for the stored controller. An absent source does not prevent
life gain, and a returned Blood Host does not receive the old ability's counter.
