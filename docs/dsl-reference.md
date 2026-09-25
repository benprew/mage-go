# Card Author Composable DSL Reference

This document is the authoritative, comprehensive reference for the composable card Domain-Specific Language (DSL) in `mage-go`.

Cards import `pkg/mage/dsl` (and `pkg/mage/core` for enums/types). All card definitions, abilities, costs, targets, and effects are built using the typed combinators exported by `dsl`.

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
* `WithoutPermanent(set, ref)`: Excludes one specific permanent (e.g. `WithoutPermanent(MatchingPermanents(HasSubType("Zombie")), This)`).

### Player Selectors (`PlayerRef` and `PlayerSet`)
* `You`: The stored resolving controller of the spell or ability.
* `ActivePlayer`: The player whose turn it currently is.
* `FromPlayer(binding)`: The player captured in a `PlayerBinding` (from target, snapshot, or event pattern).
* `ControllerOf(permanentRef)`: The player who controls the referenced permanent at resolution time.
* `EachPlayer`: Collection of all players in active-player-first order.
* `OpponentsOf(playerRef)`: All opponents of the specified player (for 2-player, the single opponent).

---

## 4. Target Declarations

Targets are declared as typed bindings and attached to abilities using `WithTargets(...)`:

```go
target := TargetPermanent("creature", IsCreature)

Spell(
    DealDamage(This, target, Fixed(3)),
    WithTargets(target),
)
```

### Permanent Targets
* `TargetPermanent(label, filter, options...)`: Declares a single permanent target matching the filter.
* `TargetPermanents(label, min, max, filter, options...)`: Declares multiple permanent targets.
  For “up to two target creatures,” use `TargetPermanents("creatures", 0, 2, IsCreature)`
  with `WithTargets(creatures)`. Dauntless Onslaught and Tandem Tactics pass this
  collection to `Until(EndOfTurn, ModifyEachPT(creatures, ...))`. Only targets
  that remain legal receive the boost. A cast with zero targets still resolves;
  a cast whose chosen targets all become illegal does not resolve. Tandem
  Tactics gains life after the boost with `Sequence` and `GainLife(You, Fixed(2))`.
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

---

## 5. Pure State Conditions & Filters

State conditions must be **pure functions** of game state (no random choices, mutations, or player prompts):

### State Conditions (`StateCondition`)
* `Controls(filter)`: Ability controller controls at least one permanent matching filter (includes source).
* `ControlsNone(filter)`: Ability controller controls zero permanents matching filter.
* `BattlefieldHasNone(filter)`: Neither player controls a permanent matching filter.
* `SourceHasCounters(counter, min)`: The source permanent has at least `min` counters of the specified type.
* Logical combinators:
  * `AllConditions(conds...)`: Logical AND.
  * `AnyCondition(conds...)`: Logical OR.
  * `NotCondition(cond)`: Logical NOT.
* `StateConditionFunc(desc, deps, fn)`: Pure read-only escape hatch with declared dependencies.

### Permanent & Card Filters (`PermanentFilter`, `CardFilter`)
* Type filters: `IsCreature`, `IsArtifact`, `IsLand`, `IsEnchantment`, `IsPlaneswalker`, `IsToken`, `IsNonToken`.
* Characteristic filters: `HasSubType("Dwarf")`, `HasColorFilter(Red)`, `HasKeywordFilter(Flying)`.
* `HasToughnessGTE(n)`: Matches current toughness at least `n`, including continuous effects and counters. Combine with `IsCreature` for creature targets. Valorous Stance uses this filter in its destruction mode; check it at casting and resolution.
* Combinators: `And(filters...)`, `Or(filters...)`, `Not(filter)`.
* `IsAttachedTo(filter)`: Matches attachments whose current, unphased host matches the filter. This does not require an Aura subtype or restrict either controller.

---

## 6. Resolving Effects (Actions & Verbs)

Resolving effects implement `ResolvingEffect` and mutate the game during resolution:

* `GrantLandPlays(PlayerRef, ResolutionValue)`: Adds land plays for the selected player for this turn. Grants accumulate; the amount is read once at resolution. An absent player or nonpositive amount does nothing. This does not play a land, grant access to another zone, or override normal land-play timing.

### Permanent Actions
* `Sacrifice(PermanentRef)`: Stored controller sacrifices the permanent. Fails if they no longer control it.
* `Destroy(PermanentRef)`: Destroys the permanent (allows regeneration).
* `DestroyWithoutRegeneration(PermanentRef)`: Destroys the captured permanent without regeneration. The prohibition belongs to this destruction action; it does not change permanent keywords or last-known characteristics. Indestructible still applies.
* `DestroyEach(PermanentSet)`: Simultaneously destroys all permanents in the set. For multiple card types, use one `MatchingPermanents(Or(...))` collection; a permanent with several matching types is destroyed only once.
* `DestroyEachWithoutRegeneration(PermanentSet)`: Simultaneously destroys the set without regeneration. Indestructible still applies; the instruction does not change permanent keywords.
* `Exile(PermanentRef)`: Moves the permanent to the exile zone.
* `Tap(PermanentRef)`: Taps the permanent.
* `Untap(PermanentRef)`: Untaps the permanent.
* `RemoveFromCombat(PermanentRef)`: Removes the captured permanent from combat. Removing a blocker does not make its attacker unblocked. An absent or different incarnation is unaffected.
* `ReturnToHand(PermanentRef)`: Bounces the permanent to its owner's hand.

For an Aura entry trigger, use `WhenEvent(Enters(This), Tap(AttachedPermanent))`.
The attachment is read at resolution, so moving the Aura changes the affected
permanent. If the Aura has left, its last attachment is used. A returned Aura
or host is a new incarnation and does not redirect the old reference.

### Player Actions
* `Draw(PlayerRef, ResolutionValue)`: Instructs player to draw cards.
* `DrawEach(PlayerSet, ResolutionValue)`: Draws cards for each player in turn order.
* `Discard(PlayerRef, ResolutionValue)`: Player chooses and discards cards from hand.
* `DiscardAtRandom(PlayerRef, ResolutionValue)`: Discards cards at random from hand. Goblin Lore uses `Spell(Sequence(Draw(You, Fixed(4)), DiscardAtRandom(You, Fixed(3))))`; draw first, then select randomly from the entire current hand through the discard replacement and event services.
* `DiscardEntireHand(PlayerRef)`: Discards all cards in the player's hand.
* `Mill(PlayerRef, ResolutionValue)`: Mills cards from the top of the library to the graveyard.
* `GainLife(PlayerRef, ResolutionValue)`: Gains life through the event-aware life gain pipeline.
* `LoseLife(PlayerRef, ResolutionValue)`: Causes player to lose life.
* `LoseLifeEach(PlayerSet, ResolutionValue)`: Causes each player in the set to lose life.

### Damage Actions
* `DealDamage(DamageSourceRef, DamageRecipientRef, ResolutionValue)`: Deals damage.
* `PreventNextDamage(DamageRecipientRef, ResolutionValue)`: Installs a prevention shield on creature or player.
* `PreventAllCombatDamage()`: Prevents all combat damage for the turn (Fog).
* `PreventCombatDamageToAndBy(PermanentRef)`: Prevents combat damage to and by a specific permanent.

### Counter Actions
* `AddCounters(PermanentRef, CounterType, ResolutionValue)`: Adds counters through the replacement pipeline.
* `RemoveCountersFrom(PermanentRef, CounterType, ResolutionValue)`: Removes up to the specified count.
* `RemoveAllCounters(PermanentRef, CounterType)`: Removes all counters of that kind.

### Library & Mana Actions
* `ScryAtResolution(ResolutionValue)`: Scries cards equal to the amount.
* `SurveilAtResolution(ResolutionValue)`: Surveils cards equal to the amount.
* `AddManaAmount(Color, ResolutionValue)`: Adds mana of the specified color to the pool.

---

## 7. Control Flow, Choices, and Branches

* `Sequence(effects...)`: Executes resolving effects in order, threading a single resolution frame.
* `ForEach(set, memberBinding, body)`: Iterates sequentially over permanents in the set.
* `FlipCoin(playerRef, OnWin(effect), OnLoss(effect))`: Resolves a coin flip through the game's deterministic RNG.
* `IfState(condition, effect)`: Resolution-time branch based on pure game state.
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
  * `Enters(This)`: Source enters the battlefield.
  * `Dies(This)`: Source dies (moves from battlefield to graveyard).
  * `CreatureDies()`: Any creature dies.
  * `CreatureYouControlDies()`: A creature controlled by ability controller dies.
  * `DealsDamage(source, amountBinding)`: Source deals damage to anything; captures amount.
  * `DealsDamageToPlayer(source, playerBinding)`: Source deals damage to player; captures player.
  * `DealsDamageToOpponent(source, playerBinding)`: Source deals damage to opponent.
  * `ControllerDrawsCard(playerBinding)`: Source controller draws a card.
  * `ControllerGainsLife(playerBinding)`: Source controller gains a positive amount of life. Saves that player at event detection without declaring a target. A replaced gain triggers only if life is actually gained. Copies retain the saved player while `You` uses the copy's controller.
  * `OpponentLosesLife(playerBinding)`: Opponent loses life (can chain `.WithAmount(binding)`).
  * `AttachedControllerUpkeep(hostBinding, playerBinding)`: Upkeep of enchanted permanent's controller.
  * `AttachedBecomesTapped(hostBinding)`: Captures the attached permanent when it becomes tapped. Read with `From(hostBinding)`; it is not a target. Attachment moves and source departure do not change the captured incarnation.

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
  * `LifePayCost(amount)`: Pay life (permits paying exact life total CR 119.4).
  * `RemoveCountersCost(counter, count)`: Remove counters from source.
  * `DiscardRandomCost(count)`: Discard cards at random.

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
  * `ActivationIf(condition)`: Pure state condition required to activate.
  * `CaptureNumberBeforePayment(value, binding)`: Saves amount before mana production & cost payment.
  * `CapturePermanentBeforePayment(subject, binding)`: Saves permanent (e.g. attachment) before sacrifice cost.

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
* `GrantKeyword(subject, keyword)`: Grants keyword ability (Layer 6).
  Blighted Bat uses `Activate(Costs(Mana("{1}")), Until(EndOfTurn, GrantKeyword(This, Haste)))`. The ability has no target, tap cost, or activation limit. Its printed flying ability remains separate.
  Jump uses `Spell(Until(EndOfTurn, GrantKeyword(creature, Flying)), WithTargets(creature))` with `TargetPermanent("creature", IsCreature)`. Pixie Queen uses the same body in `Activate` with `Costs(Mana("{G}{G}{G}"), TapCost(This))`. Either player's creature can be targeted, including Pixie Queen itself. The resolved grant remains after the source leaves and expires at cleanup.
  Icatian Scout and Svyelunite Priest use a `TargetPermanent("creature", IsCreature)` binding with `WithTargets` and `Until(EndOfTurn, GrantKeyword(creature, ...))`. Both pay mana and `TapCost(This)`. The Priest combines `OnlyDuring(core.Upkeep)` with `OnlyOnYourTurn()` to restrict activation to its controller's upkeep. Either player's creature can be targeted, including the source.
  Order of Leitbur and Order of the Ebon Hand use `Activate(Costs(Mana("{W}")), Until(EndOfTurn, GrantKeyword(This, FirstStrike)))` and the corresponding `{B}` cost. Their separate +1/+0 activations use `ModifyPT(This, Fixed(1), Fixed(0))` with `{W}{W}` or `{B}{B}` and the same explicit end-of-turn duration.
  Icatian Infantry uses separate `{1}` activations for first strike and banding with the same typed source grant and duration. River Merfolk grants mountainwalk for `{U}`, Vodalian Knights grants flying for `{U}`, and Orcish Veteran grants first strike for `{R}`. These activations have no targets, tap costs, or per-turn limits.
  Alabaster Mage uses `TargetPermanent("creature", IsCreature, TargetYouControl())`, `{1}{W}`, and a temporary lifelink grant. Check the controller restriction at activation and resolution; a later control change does not end the resolved grant. Emerald Dragonfly uses `{G}{G}` and `GrantKeyword(This, FirstStrike)` with the same end-of-turn duration, keeping its printed flying ability. Neither activation has a tap cost or activation limit.
* `RemoveKeyword(subject, keyword)`: Removes all instances of one keyword in Layer 6. Use `Until(EndOfTurn, RemoveKeyword(creature, Forestwalk))` for Scarwood Hag's second activation. Declare the creature with `TargetPermanent` and `WithTargets`; pay `Costs(TapCost(This))`. The first activation uses `GrantKeyword` and adds `Mana("{G}{G}{G}{G}")`. Both can target either player's creature, including the Hag. Removal uses timestamp order, so a later grant can restore the keyword. A resolved removal survives source departure, preserves the recipient through phasing, expires at cleanup, and does not affect a returned incarnation.
* `CantBeBlocked(subject)`: Prevents all creatures from blocking the permanent. Install with `Until(EndOfTurn, CantBeBlocked(subject))`; this is a rules restriction, so ability removal does not erase it. Later blockers are also restricted. Source departure does not end the duration, and a returned subject is a new object.
* `GrantEachKeyword(permanentSet, keyword)`: Grants keyword to set.
* `SetColors(subject, colors...)`: Replaces colors (Layer 5).
* `SetEachColors(permanentSet, colors...)`: Replaces colors for set.
* `SetSubtypeFamily(subject, family, subtypes...)`: Replaces one subtype family (Layer 4).
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
