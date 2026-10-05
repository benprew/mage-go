# DSL reference

This checkout has the legacy facade in `pkg/mage/dsl/dsl.go`. It does not have
`pkg/mage/dsl/composable.go`. Proposed signatures in the migration plan are
not implemented APIs.

## RON card loading

`pkg/mage/dsl/ron` parses the card records in `cards/mtgish/*.ron` and compiles
supported rules into the existing engine. It has these loading APIs:

```go
func Parse(reader io.Reader, filename string) ([]Node, error)
func Compile(card Node) (mage.CardFactory, error)
func LoadFS(files fs.FS, pattern string) (*mage.Registry, error)
func LoadInto(registry *mage.Registry, files fs.FS, pattern string) error
```

`LoadInto` replaces the entire destination registry only after every record
compiles. A failed load leaves its factories and name aliases unchanged. The
loader never calls an existing Go factory to substitute for unsupported RON.
Reprints with equal definitions share a factory. Conflicting definitions fail
the load. Every factory creates separate card, ability, target, and effect state.

This startup function compiles against the current API:

```go
package example

import (
    "os"
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/dsl/ron"
)

func LoadCards(directory string) error {
    return ron.LoadInto(mage.DefaultRegistry, os.DirFS(directory), "*.ron")
}
```

The parser supports the syntax used by the supplied files: constructor names,
named fields, lists, tuples, strings, signed decimal integers, trailing commas,
and line and nested block comments. Records may span lines. Raw strings and
Rust Unicode string escapes are supported. This is a card-file parser, not a
general implementation of every RON extension.

The initial compiler supports card metadata, color indicators, simple keywords,
color protection, basic landwalk, tap/sacrifice/mana costs, single target
declarations, source damage to a bound target, batch destruction, drawing,
controller life changes, mana production, temporary boosts and keyword grants,
Aura boosts and grants, creature-filtered global boosts and grants, self entry
and death triggers, and controller upkeep triggers. It rejects unsupported
constructors, target-role mismatches, unbound references, and unknown fields.
It does not discard unsupported rules from otherwise supported cards.

The public adapters include `ActionList`, `Targeted`, `SpellActions`, `Activated`,
`TriggerA`, `Integer`, `PayMana`, `IsCardtype`, `IsCreatureType`,
`SpellDealsDamage`, `PermanentDealsDamage`, `DrawNumberCards`,
`DestroyPermanent`, `DestroyPermanentNoRegen`, `DestroyEachPermanent`, and
`DestroyEachPermanentNoRegen`. Their typed Go parameters use existing engine
values. The compiler validates the RON references before it calls these adapters.

Temporary RON layers use typed permanent references. In existing spell,
activation and trigger action lists, `CreatePermanentLayerEffectUntil/3`
accepts `ThisPermanent`, Aura `HostPermanent`, or the local
`Ref_TargetPermanent`, fixed `AdjustPT/2` and basic keyword `AddAbility/1`
modifications, and `UntilEndOfTurn`. One engine `Until` installs each action's
modifications together. Subjects are captured at resolution and retain their
incarnations until expiry. The compiler rejects empty modifications and invalid
or out-of-scope reads. Exact contracts and RON tests are in
[the temporary-layer slice](designs/mtgish-ron-temporary-layers.md).

Run the full collection check with:

```sh
go run ./cmd/loadcards -dir cards/mtgish
go run ./cmd/loadcards -dir cards/mtgish -json
```

Both commands return a nonzero exit status if any record fails. JSON diagnostics
include record counts and each failed card's location and first compilation
error. The current collection still has unsupported rules. Existing application
registration therefore remains in place until the complete collection compiles.
Remaining families include conditional costs and actions, other trigger forms,
replacement and prevention effects, modes, copy effects, zone movement,
permanent rule restrictions, dynamic values, and multiple result/target bindings.

## Damage attribution through retained callers

The existing engine services have these signatures:

```go
func (g *Game) GetArtifactDamageTaken(playerID uuid.UUID) int
func (g *Game) SetOnDamageDealt(callback func(sourceName, targetName string, amount int, isCombat bool))
func (ds *DamageSystem) CombatDamageSourcesThisStep(controllerID, recipientID uuid.UUID) map[uuid.UUID]int
```

These are engine services, not new sealed DSL expressions. Legacy resolving
damage effects use them through the shared damage system. Artifact history and
combat aggregation read the captured permanent incarnation's current information
or retained LKI. Callbacks report that source's name and the action's combat flag.
Copied abilities keep their original source attribution. History records damage
after prevention and redirection. History keys and event participants remain UUIDs.

This example compiles against the current engine and legacy DSL facade:

```go
package example

import (
    "github.com/google/uuid"
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/dsl"
)

func ArtifactHistory(g *mage.Game, player uuid.UUID) int {
    return g.GetArtifactDamageTaken(player)
}

func DamageSource() dsl.Card {
    return dsl.NewCreature("Example artifact damage source", "", 1, 3,
        dsl.WithActivatedAbility(dsl.DealDamage(dsl.Fixed(2)),
            dsl.GenericCost(0), dsl.WithTarget(dsl.TargetDamageAnyTarget())))
}
```

`TestDamageAttributionSequence` makes a synthetic source an artifact, returns it
under another controller, and resolves its damage instruction. The history uses
the original source's artifact type. The example is a compiling declaration;
the test supplies the characteristic change and source departure.
No production cards or legacy exports are changed. Typed `DealDamageFrom`,
spell-source LKI, recipient identity, and immutable event captures remain open.

## Retained singular damage

```go
func DealDamage(amount ValueSource) Effect
func TargetDamageAnyTarget() Target
```

`dsl.DealDamage` retains its single-argument signature and positional target
behavior for production callers. The resolving effect can be used in a
targeted spell or a legacy activated or triggered ability with a declared
target. It uses the shared damage service. Damage to a planeswalker now removes
loyalty after prevention and replacement. A creature-planeswalker also receives
the existing creature damage result. Excess damage still counts in full for
damage events and lifelink. State-based actions handle zero loyalty.

This compiling declaration uses only the existing facade:

```go
package example

import "github.com/benprew/mage-go/pkg/mage/dsl"

func DamageSpell() dsl.Card {
    return dsl.NewInstant("Example damage spell", "{R}",
        dsl.NewTargetedSpell(dsl.TargetDamageAnyTarget(),
            dsl.DealDamage(dsl.Fixed(3))),
    )
}
```

The test-only spell in `TestPlaneswalkerDamageSpellResolution` covers actual
announcement and resolution against a planeswalker. Engine tests cover
prevention, protection, replacement order, excess damage, and clone isolation.

`DealDamageFrom(source, recipient, amount)` remains proposed. Source LKI,
sealed expression roles, explicit recipient bindings, batches, and actual-damage
result bindings are not supplied by this change. No legacy export is retired.

## Lifelink from a departed permanent

The retained `DealDamage(amount ValueSource) Effect` also applies lifelink from
live permanent characteristics or retained battlefield LKI. This works in
legacy targeted activated abilities, including an ability whose source was
sacrificed to pay its cost. Player and permanent damage both use the shared
life-gain modifiers and replacements. Life-gain events report actual gain.

This compiling synthetic declaration exercises source departure during payment:

```go
package example

import "github.com/benprew/mage-go/pkg/mage/dsl"

func LifelinkSource() dsl.Card {
    return dsl.NewCreature("Example lifelink source", "", 1, 1,
        dsl.WithKeyword(dsl.Lifelink),
        dsl.WithActivatedAbility(dsl.DealDamage(dsl.Fixed(2)),
            dsl.GenericCost(0),
            dsl.WithTarget(dsl.TargetDamageAnyTarget()),
            dsl.WithCost(dsl.SacrificeSourceCost())),
    )
}
```

No public signature changed. The later damage propagation prerequisite below
adds incarnation selection and retained LKI for captured permanent sources.
Spell-source LKI and `DealDamageFrom` remain absent. Production cards remain
unmigrated.

## Engine prerequisite: permanent incarnation references

Implemented in `mage`, without a new DSL export:

```go
func (g *Game) PermanentRef(id uuid.UUID) (PermanentRef, bool)
func (r PermanentRef) ID() uuid.UUID
func (g *Game) LookupPermanentRef(ref PermanentRef) LKIView
```

These engine queries capture a present battlefield incarnation and read its
current characteristics or retained LKI. They work outside resolution as well
as during engine actions. Capture rejects absent and phased-out permanents.
Existing references retain phase-time information while the object is absent;
phase-in resumes live reads without changing identity. Leave-and-return creates
a different reference. Departure snapshots survive cleanup. The zero reference
reads as absent. Reference values can be copied into instance-local storage and
used in cloned games; callers must retain the complete value, not only `ID()`.

This compiling engine example captures a source before its departure:

```go
package example

import "github.com/benprew/mage-go/pkg/mage"

func DepartedSourcePower(g *mage.Game, p *mage.Permanent) int {
    ref, ok := g.PermanentRef(p.ID())
    if !ok {
        return 0
    }
    g.BouncePermanentToHand(p)
    if view := g.LookupPermanentRef(ref); view != nil {
        return view.ViewPower()
    }
    return 0
}
```

This is an engine identity prerequisite, not a sealed author expression role.
The existing `LKIView` remains an engine reader with its existing read-only
ability convention. Returned departure snapshots have independent permanent
fields, but card and ability definitions follow existing shared-definition
ownership. Captured references retain snapshots for the game's lifetime; there
is no release API yet. Damage lifelink now consumes these references through
the action propagation described below. `DealDamageFrom` remains proposed.
No card wiring or gap status is complete.

## Engine prerequisite: ability source references

Implemented in `mage`; the legacy DSL signatures stay available:

```go
func (obj *StackObject) SourcePermanentRef() PermanentRef
func (g *Game) ResolvingSourcePermanentRef() PermanentRef
```

Activated abilities capture the battlefield source before payment. Ordinary
battlefield event and state triggers capture the source when they become
pending; leave triggers use the departing source. Pending-to-stack transfer,
ability copying, and game cloning preserve the complete reference. A copied
ability can have a different controller while retaining the original source.
This is identity capture, not an announcement-time characteristic snapshot:
`LookupPermanentRef` reads current characteristics until departure or phasing,
then retained LKI. Leave-and-return cannot substitute the new incarnation.

The resolution query works in engine effects and nested resolution; it restores
the previous reference on exit, including target failure. Outside resolution,
on spells, on delayed triggers, and on non-battlefield-source abilities it
returns zero. Manually constructed stack objects need engine capture to carry
an identity; a UUID alone does not supply one. Direct printed mana abilities do
not use the stack or this resolution query. The activation path for resolving
mana abilities does capture its source.

This complete example uses the retained facade and the new engine query:

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/dsl"
    "github.com/google/uuid"
)

func AbilitySource() dsl.Card {
    effect := dsl.FuncEffect("Gain life equal to the source power",
        mage.EffectProperties{},
        func(g *mage.Game, source, controller uuid.UUID, targets []uuid.UUID) error {
            view := g.LookupPermanentRef(g.ResolvingSourcePermanentRef())
            if view != nil {
                g.PlayerGainLife(g.GetPlayer(controller), view.ViewPower())
            }
            return nil
        })
    return dsl.NewCreature("Example ability source", "", 2, 3,
        dsl.WithActivatedAbility(effect, dsl.GenericCost(0),
            dsl.WithCost(dsl.SacrificeSourceCost())))
}
```

`TestAbilitySourceRefSacrificeResolution` tests this pattern with a synthetic
card. Damage now carries the captured source as described below. Spell-source
identity, delayed-source capture, protection/deathtouch integration, typed roles,
and `DealDamageFrom` remain unimplemented. No card wiring or gap is complete.

## Engine prerequisite: damage action source propagation

Implemented in `mage`, with no new DSL author export:

```go
func (a *DamageToPlayerAction) SourcePermanentRef() PermanentRef
func (a *DamageToCreatureAction) SourcePermanentRef() PermanentRef
func (a *DamageToPlayerAction) RedirectToPermanent(id uuid.UUID) *DamageToCreatureAction
func (a *DamageToCreatureAction) RedirectToPlayer(id uuid.UUID) *DamageToPlayerAction
```

These queries and transformations are for engine damage replacements. Both
`WithAmount` methods preserve the captured source. The shared damage service
captures before replacements, preferring a matching resolving ability source
over the present incarnation. Built-in redirection preserves that identity.
Direct damage from a present permanent also captures its identity. UUID-only
action constructors keep their signatures and return zero references.

This complete engine example redirects a player damage action without losing
its original source:

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/google/uuid"
)

func RedirectDamage(a *mage.DamageToPlayerAction, recipient uuid.UUID) *mage.DamageToCreatureAction {
    return a.RedirectToPermanent(recipient)
}
```

The retained `dsl.DealDamage(amount ValueSource) Effect` now uses the exact
permanent incarnation for lifelink in captured battlefield abilities, including
copies and triggers. Source departure/return, cleanup, and phasing retain the
original lifelink information. Current characteristics remain current until
departure. An explicitly different damage source does not inherit the resolving
ability's reference. Uncaptured absent sources keep legacy UUID lookup.

Protection, deathtouch, notification identity, history, and recipient lookup
retain their existing contracts. Spell/delayed-source identity, typed roles,
`DealDamageFrom`, simultaneous batches, and damage results remain unimplemented.
No production card is migrated and no inventory or gap is complete.

## Engine prerequisite: permanent source filters in prevention rules

The existing `WithFrom(from PermanentFilter)` option now uses captured permanent
source information in the shared damage-prevention rule. No signature changed.
The other relevant signatures remain:

```go
func (g *Game) AddDamagePreventionRule(opts ...damagePreventionRuleOption)
func (g *Game) PreventAllDamageFrom(sourceID uuid.UUID)
func WithFrom(from PermanentFilter) damagePreventionRuleOption
func WithToPlayer(playerID uuid.UUID) damagePreventionRuleOption
```

`damagePreventionRuleOption` is package-private; callers compose options without
naming its type. `AddDamagePreventionRule` installs a rule for the current effect
cycle. Use it during continuous-effect application, or when no intervening
cycle rebuild occurs before damage. `PreventAllDamageFrom` installs its existing
source-ID rule until end of turn. Neither function is a typed DSL declaration.

For captured damage, source filters read current characteristics while the same
permanent is present and retained characteristics after departure or phasing.
Copies, redirections, and cleanup preserve that identity. Custom filters must
read the supplied permanent rather than perform another UUID lookup. Player and
permanent recipients retain their existing selection and legality contracts.
This complete engine example composes existing options:

```go
package example

import (
    "github.com/google/uuid"
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func PreventRedDamageThisCycle(g *mage.Game, player uuid.UUID) {
    g.AddDamagePreventionRule(
        mage.WithFrom(mage.HasColorFilter(core.Red)),
        mage.WithToPlayer(player),
    )
}
```

At this earlier slice boundary, the separate color/type prevention executors
were unchanged. The contract below now extends their source reads. Protection,
deathtouch, spell-source identity, typed `DealDamageFrom`, and card migration
remain separate work. All production damage exports remain available.

## Engine prerequisite: color and card-type prevention source reads

Existing public signatures are unchanged:

```go
func (g *Game) AddColorPrevention(playerID uuid.UUID, color core.Color)
func (g *Game) AddTypePrevention(playerID uuid.UUID, ct core.CardType)
func AddColorPreventionStep(color core.Color) Effect
```

The two game methods install player-only, one-use prevention rules until end
of turn. They prevent the first matching damage event in full, then expire.
They can be called from resolving effects; the color step remains usable in
legacy sequences. They are engine installers, not sealed typed declarations.

Both rules read current color/type characteristics of the original permanent
incarnation, or retained departure/phase-time LKI. This includes gained and
lost characteristics, copied abilities, and source return followed by another
departure. Missing captured information rejects the match without using the
returned card. Nonmatching events do not consume the rule. Uncaptured live
permanents use current characteristics; uncaptured color reads reuse
`Game.EffectiveColors`, including spell color changes. Uncaptured card-type
reads retain the existing card lookup for nonpermanent sources. This does not
establish spell incarnation/LKI semantics.

This complete example compiles:

```go
package example

import (
    "github.com/google/uuid"
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func PreventNextArtifactDamageThisTurn(g *mage.Game, player uuid.UUID) {
    g.AddTypePrevention(player, core.TypeArtifact)
}
```

These installers do not choose a damage source. They do not implement the
complete Circle of Protection requirements. Chosen-source identity, artifact
redirection source matching, protection, deathtouch, notifications, recipient
identity, typed damage adapters, and production card wiring remain open.

## Engine prerequisite: protection in the damage pipeline

Existing public signatures remain available:

```go
func ProtectionFromColor(c core.Color) *ProtectionAbility
func ProtectionFromColors(cs ...core.Color) *ProtectionAbility
func ProtectionFromCardType(ct core.CardType) *ProtectionAbility
func ProtectionFromSubType(subtype string) *ProtectionAbility
func ProtectionFromAll() *ProtectionAbility
func (g *Game) DealDamageToPermanent(perm *Permanent, amount int, sourceID uuid.UUID)
```

Protection damage prevention now participates in recipient-chosen replacement
ordering and is checked again after redirection to a permanent. Permanent
sources use the action's captured incarnation: current characteristics while
present, retained departure or phase-time LKI while absent. A returned card
cannot replace the original source's characteristics. Color, card-type,
subtype, and everything protection reuse existing abilities, including wrapped
grants and Aura exemptions. Noncolor custom filters retain their CardFilter
contract; effective type and subtype queries use the supplied permanent view.
Uncaptured nonpermanent sources keep existing card and effective-color lookup;
spell incarnation/LKI is still open.

The same service supports legacy resolving damage effects, sequences, divided
damage, fight instructions, combat damage, and direct engine actions. The
existing `AttrDamageCantBePreventedOrRedirected` bypass now also bypasses
protection on direct permanent damage. This does not establish general CR
615.12 semantics. Targeting, blocking, and attachment checks remain separate.
No sealed typed declaration or `DealDamageFrom` adapter is introduced here.

This complete example compiles and uses nontargeted damage:

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func DamageProtectedPermanent(g *mage.Game, controller mage.Player) {
    source := g.PutOnBattlefield(mage.NewCreature("Test red source", "{R}", 1, 3), controller.PlayerID())
    recipient := g.PutOnBattlefield(mage.NewCreature("Test protected recipient", "", 1, 3,
        mage.WithAbility(mage.ProtectionFromColor(core.Red))), controller.PlayerID())
    g.DealDamageToPermanent(recipient, 2, source.ID())
}
```

Production cards remain unmigrated. Deathtouch, notifications, spell and recipient
identity, batch damage, results, and typed adapters remain separate work.

## Engine prerequisite: deathtouch damage and destruction

Existing signatures are preserved:

```go
func DealDamage(amount ValueSource) Effect
func WithKeyword(kw core.Keyword) CardOption
func (g *Game) DealDamageToPermanent(perm *Permanent, amount int, sourceID uuid.UUID)
func (g *Game) CheckStateBasedActions()
```

The damage service now keeps actual marked damage and recent deathtouch damage
as separate facts. Permanent sources use the captured incarnation's current
keywords or retained LKI, before notification callbacks. Returning the source
cannot replace it. Positive damage after replacements and prevention can cause
destruction at the next state-based check. That check uses the shared simultaneous
destruction service, including regeneration and indestructibility. Each check
consumes the fact, even when destruction cannot occur.

Supported contexts include existing resolving damage effects, sequences,
triggered and activated abilities (including copied abilities), fight and
combat damage, and direct engine actions. Target legality remains an
announcement/resolution requirement. A copy retains the original source card
for target checks. No sealed typed constructor or `DealDamageFrom` is introduced.

This complete engine example compiles:

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func DealDeathtouchDamage(g *mage.Game, controller mage.Player) {
    source := g.PutOnBattlefield(mage.NewCreature("Test source", "", 1, 3,
        mage.WithKeyword(core.Deathtouch)), controller.PlayerID())
    recipient := g.PutOnBattlefield(mage.NewCreature("Test recipient", "", 1, 10), controller.PlayerID())
    g.DealDamageToPermanent(recipient, 1, source.ID())
    g.CheckStateBasedActions()
}
```

Spell-source identity/LKI, typed recipient bindings, damage batches and results,
source notifications, and production card migration remain open. The separate
legacy `BasiliskTouch` behavior is outside this contract.

## Engine prerequisite: artifact-source damage redirection

The existing public signature is unchanged:

```go
func (g *Game) SetArtifactDamageRedirect(controllerID, permID uuid.UUID)
```

This installer redirects artifact-source damage from the specified player to
an existing permanent for the current effect cycle. Use it in continuous-effect
application, or during resolution when no cycle rebuild precedes the damage.
It does not test whether the recipient is untapped; declarations must supply
that condition. An absent or phased recipient makes the rule inactive.

Source matching now reads the action's captured permanent incarnation, using
current types while present and retained departure/phase-time types while absent.
A later incarnation cannot supply those types. Missing captured information
rejects the match. Uncaptured live sources use current types; other uncaptured
sources keep the existing card lookup, without a spell-identity guarantee.
Replacement ordering, amount, combat flag, and original source reference use
the shared damage pipeline. This complete engine example compiles:

```go
package example

import "github.com/benprew/mage-go/pkg/mage"

func RedirectArtifactDamageThisCycle(g *mage.Game, player mage.Player, recipient *mage.Permanent) {
    g.SetArtifactDamageRedirect(player.PlayerID(), recipient.ID())
}
```

There is no new DSL export. Martyrs of Korlis still requires a typed replacement
and untapped condition. All production callers remain available. Notification
identity, spell/recipient identity, typed singular damage, batches, results, and
card migration remain separate work.

## Engine prerequisite: immutable damage-event source colors

Existing DSL signatures remain available:

```go
func (c EventSourceWasColor) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool
func (c EventSourceWasNotColor) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool
```

For permanent-source damage, both predicates use colors captured from the damage
action's original incarnation before life and damage notifications. Current
colors apply while present; departure or phase-time LKI applies while absent.
Later changes, return, cleanup, event value copies, and game clones cannot change
this capture. Missing captured information satisfies neither predicate. Other
events and uncaptured nonpermanent sources retain legacy UUID lookup behavior.
These predicates remain usable with legacy triggered abilities; they are not
sealed composable declarations.

`core.GameEvent.DamageSourceColors` is an immutable `core.EventSourceColors`
value. Its engine inspection methods are `Captured() bool`, `Known() bool`,
`HasColor(Color) bool`, `SourceID() uuid.UUID`, and `Incarnation() uuid.UUID`.
`core.NewEventSourceColors(sourceID, incarnation uuid.UUID, colors []Color,
known bool) EventSourceColors` copies the color input into the value. A zero
value means there is no capture; a captured unknown source is distinct from a
known source with no colors. Damage executors produce this value, after
replacement processing and before result notifications. It is not a resolution
selector or a damage source declaration.

This complete example compiles:

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func BlackDamageWatcher() mage.Card {
    condition := mage.EventSourceWasColor{Color: core.Black}
    return mage.NewEnchantment("Test black damage watcher", "",
        mage.WithAbility(mage.NewTriggered(core.EvtDamageDealt, false,
            mage.GainLife(1)).SetCondition(condition.CheckTriggerCond)))
}
```

Controller/type/subtype/filter and self-identity event predicates, spell LKI,
recipient identity, result bindings, typed damage, and production card migration
remain separate work. All production exports and callers remain available.

## Permanent damage-source controller capture

Implemented engine inspection API in `pkg/mage/core`:

```go
NewEventSourceController(sourceID, incarnation, controller uuid.UUID, known bool) EventSourceController
(EventSourceController).Captured() bool
(EventSourceController).Known() bool
(EventSourceController).SourceID() uuid.UUID
(EventSourceController).Incarnation() uuid.UUID
(EventSourceController).ControllerID() uuid.UUID
```

`GameEvent.DamageSourceController` contains this immutable value on damage
from a captured permanent. Both player and permanent damage paths save it
before life or damage notifications. Current information applies to the original
incarnation; retained departure or phase-time information applies while absent.
Returning the card cannot replace the controller. Unknown captured information
never falls back to UUID lookup. A source without a controller matches neither
controller predicate; its owner is not a substitute.

The existing engine conditions `EventSourceControlledByController{}` and
`EventSourceControlledByOpponent{}` retain their
`CheckTriggerCond(*GameEvent, GameReader, uuid.UUID, uuid.UUID) bool` signatures.
They use the saved controller for captured damage events. Other events and
uncaptured nonpermanent sources retain legacy lookup. These are event conditions,
not pure state predicates. The facade does not export these conditions here.
The scalar inspection API is not a sealed author expression or typed damage
adapter; type/subtype, self matching, and spell participants remain separate work.

This complete synthetic definition compiles with the current engine API:

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func DamageControllerWatcher() mage.Card {
    condition := mage.EventSourceControlledByController{}
    ability := mage.NewTriggered(core.EvtDamageDealt, false, mage.GainLife(1)).
        SetCondition(condition.CheckTriggerCond)
    return mage.NewEnchantment("Test damage controller watcher", "", mage.WithAbility(ability))
}
```

This demonstrates only the controller condition. A creature/combat/opponent
restriction and optional draw require additional declarations for Coastal Piracy.

## Permanent damage-source type capture

Implemented engine inspection API in `pkg/mage/core`:

```go
NewEventSourceTypes(sourceID, incarnation uuid.UUID, types []CardType, subtypes []string, known bool) EventSourceTypes
(EventSourceTypes).Captured() bool
(EventSourceTypes).Known() bool
(EventSourceTypes).SourceID() uuid.UUID
(EventSourceTypes).Incarnation() uuid.UUID
(EventSourceTypes).HasType(CardType) bool
(EventSourceTypes).HasSubType(string) bool
```

`GameEvent.DamageSourceTypes` stores immutable copies of effective types and
subtypes from the original permanent incarnation. Both damage recipient paths
capture them before life or damage notifications. Current information applies
while the original incarnation is present; retained LKI applies during absence,
including phasing. Cleanup and a returned incarnation cannot replace the value.
Unknown captured information matches no type or subtype.

The existing `mage.EventSourceHasType`, `mage.EventSourceHasSubType`, and
`mage.EventSourceWasOfType` conditions retain their
`CheckTriggerCond(*GameEvent, GameReader, uuid.UUID, uuid.UUID) bool` signatures.
They use the capture for permanent-source damage; other events and uncaptured
nonpermanent sources retain existing lookup. This does not implement a general
permanent filter, spell-source capture, sealed expression, or typed damage
adapter. Legacy DSL exports remain compatible.

This complete synthetic definition compiles:

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func CreatureDamageWatcher() mage.Card {
    condition := mage.EventSourceWasOfType{Type: core.TypeCreature}
    ability := mage.NewTriggered(core.EvtDamageDealt, false, mage.GainLife(1)).
        SetCondition(condition.CheckTriggerCond)
    return mage.NewEnchantment("Test creature damage watcher", "", mage.WithAbility(ability))
}
```

This example tests only source type. Coastal Piracy still needs controller,
combat, opponent, optional draw, and typed event declarations.

## Retained self-source damage conditions

These existing engine declarations keep their signatures:

```go
func (EventSourceIsSelf) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool
func (EventSourceIsSelfDamageToPlayer) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool
func (EventSourceIsSelfDamageToOpponent) CheckTriggerCond(evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID) bool
func DealsDamageToOpponentTrigger(effect Effect, optional bool) *GenericTriggered
```

For damage events with a permanent-source capture, self matching requires a
present source with the captured battlefield incarnation. Departure, returning,
phasing out, and missing captured information do not match. Phasing in keeps the
same identity. Copies preserve the source, and clones use their own battlefield.
Already pending triggers continue after source departure. Uncaptured events and
non-damage events retain UUID matching. These are retained trigger-time engine
conditions, not sealed typed expressions or resolution-time predicates.

This complete example compiles with the retained engine API:

```go
package example

import "github.com/benprew/mage-go/pkg/mage"

func SelfDamageSource() mage.Card {
    return mage.NewCreature("Example self damage source", "", 1, 4,
        mage.WithAbility(mage.DealsDamageToOpponentTrigger(mage.GainLife(1), false)),
        mage.WithActivatedAbility(mage.DealDamage(mage.Fixed(2)),
            mage.GenericCost(0), mage.WithTarget(mage.TargetDamageAnyTarget())))
}
```

The ability deals damage through the shared damage service. If the source leaves
and returns before the activation resolves, its new trigger does not match the
old source's damage. Typed `DealDamageFrom`, spell and recipient identity,
full permanent-filter event captures, batches, and results remain proposed work.

## Damage-event permanent-filter identity

The retained engine condition has this signature:

```go
func (c EventSourceMatchesPermanentFilter) CheckTriggerCond(
    evt *GameEvent, g GameReader, sourceID, controllerID uuid.UUID,
) bool
```

For captured permanent damage, these existing filters use immutable event
information: `AnyPermanent`, the zero filter, `IsCreature`, `IsArtifact`,
`IsEnchantment`, `IsLand`, `IsPlaneswalker`, `HasSubType(string)`,
`HasColorFilter(core.Color)`, `ControlledBy(uuid.UUID)`,
`NotControlledBy(uuid.UUID)`, `IsID(uuid.UUID)`, `NotID(uuid.UUID)`,
`Named(string)`, `OwnedBy(uuid.UUID)`, `NotOwnedBy(uuid.UUID)`, and `IsToken`.
`Not(PermanentFilter)`, `And(...PermanentFilter)`, and `Or(...PermanentFilter)`
use captures when every leaf supports them. `And` and `Or` copy their inputs.
Their ordinary `Match(*Permanent, *Game)` behavior remains a current-state query.

Captures describe the original incarnation at damage execution, before life or
damage notifications. Later characteristic changes, departure, phasing, return,
cleanup, and cloned-game changes cannot change the match. Missing or inconsistent
required captures match nothing, including under negation or disjunction.
Controller filters require a known nonzero controller. ID filters compare the
saved UUID; they do not implement self-incarnation matching.

Filters outside this list, including custom filters, power, attachment, combat,
keyword and supertype filters, retain identity-safe lookup.
If a composition includes such a filter, the entire composition uses this
existing contract: current information for the same present incarnation or
retained LKI for departure/phasing, with the supplied game for state queries.
Uncaptured events and other event types keep current UUID lookup.

This is a retained engine condition; this checkout's `dsl` facade does not
export it. A complete compiling engine example is:

```go
package example

import (
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func Watcher() mage.Card {
    condition := mage.EventSourceMatchesPermanentFilter{
        Filter: mage.And(mage.IsCreature, mage.HasColorFilter(core.Green)),
    }
    return mage.NewEnchantment("Example damage filter watcher", "",
        mage.WithAbility(mage.NewTriggered(core.EvtDamageDealt, false,
            mage.GainLife(1)).SetCondition(condition.CheckTriggerCond)))
}
```

This establishes captured characteristic matching for the listed filters only.
Full permanent/game snapshots and exact typed declarations remain separate work.
Existing production callers stay compatible; no typed damage adapter is added.


## Damage-event source name, owner, and token status

Both shared damage executors save `core.GameEvent.DamageSourceIdentity` before
life and damage notifications. It contains an immutable `core.EventSourceIdentity`:

```go
func NewEventSourceIdentity(sourceID, incarnation, owner uuid.UUID,
    name string, token, known bool) EventSourceIdentity
func (s EventSourceIdentity) Captured() bool
func (s EventSourceIdentity) Known() bool
func (s EventSourceIdentity) SourceID() uuid.UUID
func (s EventSourceIdentity) Incarnation() uuid.UUID
func (s EventSourceIdentity) OwnerID() uuid.UUID
func (s EventSourceIdentity) Name() string
func (s EventSourceIdentity) IsToken() bool
```

`Named(name string) PermanentFilter`, `OwnedBy(playerID uuid.UUID) PermanentFilter`,
`NotOwnedBy(playerID uuid.UUID) PermanentFilter`, and the `IsToken PermanentFilter`
value now support captured damage matching through `EventSourceMatchesPermanentFilter`.
They work in triggered conditions and supported `And`/`Or`/`Not` compositions.
Ordinary permanent and prevention filtering retain their current/LKI contracts.
Captures use the existing live/LKI owner query, independently of ability control;
that service falls back to the permanent controller for ownerless engine fixtures.
A captured zero owner is unknown for owner filters. Empty names and nontoken status
remain known values. Missing, unknown, or mismatched captures never match, including
through negation. Uncaptured and non-damage events retain current lookup.

This complete engine example compiles:

```go
package example

import (
    "github.com/google/uuid"
    "github.com/benprew/mage-go/pkg/mage"
    "github.com/benprew/mage-go/pkg/mage/core"
)

func IdentityWatcher(owner uuid.UUID) mage.Card {
    condition := mage.EventSourceMatchesPermanentFilter{
        Filter: mage.And(mage.Named("Example source"), mage.OwnedBy(owner),
            mage.Not(mage.IsToken)),
    }
    return mage.NewEnchantment("Example identity watcher", "",
        mage.WithAbility(mage.NewTriggered(core.EvtDamageDealt, false,
            mage.GainLife(1)).SetCondition(condition.CheckTriggerCond)))
}
```

No production caller changes. Spell-source identity, recipient identity, typed
`DealDamageFrom`, other intrinsic filters, batches, and results remain open.

## RON single spell targets

`Targeted([TargetSpell(filter)], ActionList([CounterSpell(Ref_TargetSpell)]))`
uses a typed declaration local to the rule. Each factory creates a fresh target
binding. Supported filters are `AnySpell`, `IsCardtype(type)`, `IsColor(color)`,
and recursive nonempty `And([filters])` and `Or([filters])`. Filters match spell
cards on the stack and exclude abilities. The engine checks legality at both
announcement and resolution. Countering uses the selected stack object and puts
its card in its owner's graveyard, unless it cannot be countered. Copies and
clones keep separate target choices. Undeclared, incompatible, malformed and
cross-rule reads fail compilation. Controller, mana-value and target-history
filters, modes and multiple spell target groups remain unsupported.

Contract and card tests: [RON spell scopes](designs/mtgish-ron-spell-scopes.md).

### RON controlled static creature layers

In `EachPermanentLayerEffect`, use
`ControlledByAPlayer(SinglePlayer(You))` inside an `And` filter
for fixed P/T changes or basic keyword grants. Nested nonempty conjunctions are
supported. The engine reads the source's current controller each time it applies
the layers. Matching sources are included. Current membership changes
with control, tap status, combat status, color and later entries. Controller
predicates inside `Or` and other controller reference forms remain rejected.
This adapter does not add controller support to target or collection filters.
See [the supported contract and tests](designs/mtgish-ron-static-controller.md).

### RON static source exclusion

In an `EachPermanentLayerEffect` conjunction, use
`Other(ThisPermanent)` to exclude the source from fixed P/T changes and basic
keyword grants. It can be combined with `ControlledByAPlayer(SinglePlayer(You))`
and nested nonempty `And` filters. Each source excludes only itself; other
instances can affect it. Source-dependent predicates below `Or`, other reference
operands and target/cost/action contexts remain rejected.
See [the contract and card tests](designs/mtgish-ron-static-other.md).

### RON captured temporary P/T values

`CreatePermanentLayerEffectUntil` supports `AdjustPTX(power, toughness, value)`
with `PlusX`, `MinusX`, `Zero` or `Integer(n)` coordinates. Values can be bound
`ValueX`, `Integer(n)`, or `ManaValueOfPermanent` of the modified permanent.
In that reader, `ActionPermanent` names the modification's subject. The binding
is local to the modification. Numeric values are captured at resolution and
remain fixed until end of turn. Other-object characteristic reads, power and
toughness readers, arithmetic and static AdjustPTXY remain unsupported.
See [the exact contexts and tests](designs/mtgish-ron-temporary-values.md).

## RON static landwalk grants

`AddAbility([Landwalk(IsLandType(subtype))])` is supported inside the existing
`PermanentLayerEffect(HostPermanent, modifications)` Aura context and
`EachPermanentLayerEffect(filter, modifications)` context.
The subtype must be a symbol: Plains, Island, Swamp, Mountain, Forest or Desert.
These grants use the same validated landwalk form as printed RON keywords.
They can be mixed with existing fixed P/T modifications and basic keywords.
Global source exclusion and controller conjunctions retain their existing
contracts. Each factory creates fresh continuous effects.

General permanent subjects are also supported as described below. Other
landwalk filters, temporary grants and rule restrictions remain separate
compiler contexts. See
[the static-landwalk slice](designs/mtgish-ron-static-landwalk.md) for evidence.

## RON general static permanent subjects

`EachPermanentLayerEffect(filter, modifications)` accepts the existing permanent
filters without an added creature restriction for fixed `AdjustPT(power,
toughness)` and basic keyword or supported landwalk `AddAbility(list)` grants.
`IsCreatureType(Goblin)` selects Goblin permanents; add `IsCardtype(Creature)`
only when the declaration requires creatures. Keywords can apply to
noncreatures. A P/T modification can apply when its recipient becomes a creature.

Nonempty recursive `And` and `Or` filters retain their logical membership.
`Other(ThisPermanent)` and `ControlledByAPlayer(SinglePlayer(You))` work at the
root or inside conjunctions. Source-dependent predicates within a disjunction
remain rejected. Each factory builds a fresh typed permanent set and static rule.
The engine reads current membership, controller, subtype, type and phase status
on each layer pass. No target or action-result binding is introduced.

General sets lower to `MatchingPermanents`, `ControlledPermanents`,
`WithoutPermanent`, `ModifyEachPT`, `GrantEachKeyword`, `While` and `Static`.
Existing explicit-creature and Aura adapters retain their services. Static
numeric P/T, independent base P/T/color/type changes, granted activated/triggered
abilities and conditional rules remain separate unsupported shapes.
See [the exact contract and tests](designs/mtgish-ron-static-permanents.md).

## RON optional fixed mana payments

An adjacent `PlayerMayCost(player, PayMana([...]))` and
`Unless(CostWasPaid, [actions])` pair builds one `UnlessPaid` effect.
Each pair keeps its own result, including nested unpaid bodies. Payment uses
the engine's total-cost transaction during resolution. Spell-controller payer
references use the current controller of the declared stack target.
Variable, alternative and nonmana payments remain rejected.
Contract and tests: [RON optional payments](designs/mtgish-ron-optional-payment.md).

## RON static animation

`EachPermanentLayerEffect` supports a combined static animation list with exactly
one `SetPT(PT(power, toughness))`, one `AddCardtype(Creature)`, and optionally one
`SetColor(SimpleColorList(colors))`. Values are signed integer literals. The
list order does not change layer order. Color symbols are White, Blue, Black,
Red and Green; an empty list makes the recipients colorless. Omission keeps
current colors. Other modifications cannot share this animation list.

The existing static permanent filters apply. Factories lower the complete
combination to `Static(AnimateEach(set, power, toughness, colors...))`.
The engine selects subjects in layer 4 and retains those incarnations for
layers 5 and 7b within the same application. Later applications use current
membership. Other static, Aura and temporary type/base/color shapes remain
separate compiler contexts. See [the contract and tests](designs/mtgish-ron-static-animation.md).

RON action sequences also accept `MayCost(PayMana([...]))` followed by
`Unless(CostWasPaid, [actions])`. This pair uses the stored resolving controller
as payer. Costs are fixed mana costs; an empty list still offers a choice.
`PermanentDealsDamage(ThisPermanent, value, Player(You))` uses that controller
as the recipient and the engine's damage source, LKI and prevention services.
Other explicit damage-player references remain unsupported. Shapes, limits and
tests: [RON upkeep payments](designs/mtgish-ron-upkeep-payment.md).

## RON fixed draw and activation life cost

Symbol `DrawACard` draws one card in supported action contexts. In
`PlayerAction(player, DrawACard)`, the typed player reference selects the player;
otherwise the resolving controller draws. Activation cost
`PayLife(Integer(n))` accepts a nonnegative fixed integer and uses the existing
atomic total-cost payment. Optional and dynamic life costs remain rejected.
Fixed `AnyManaColor` production asks for one color for the complete amount.
See [the exact shape and gameplay report](designs/mtgish-ron-fixed-draw.md).

## RON spell-cast life triggers and paid bodies

`TriggerA(WhenAPlayerCastsASpell(AnyPlayer, filter), actions)` uses the existing
spell filters and `WheneverSpellCastTrigger`. Other caster scopes and event
references remain rejected. An adjacent fixed-mana MayCost or PlayerMayCost
followed by `If(CostWasPaid, [actions])` uses `IfPaid`. Each nonempty body has
its own payment result, including nested pairs. Payment is offered at resolution
to the stored ability controller or explicit typed payer. X and nonmana
payments remain separate shapes. See [the contract and tests](designs/mtgish-ron-spell-cast-life.md).

## RON fixed selected discard

In supported spell, activation and trigger action lists, symbol `DiscardACard`
and `DiscardNumberCards(Integer(n))` use `mage.Discard` with a fixed amount.
The integer must be nonnegative. Bare actions affect the resolving controller;
`PlayerAction(player, action)` uses an existing typed player reference, including
an explicitly declared `Ref_TargetPlayer`. Each factory builds fresh effects
and target bindings. A target declaration alone does not change the recipient
of a bare discard action.

The affected player selects cards at resolution, after preceding draws. Short
hands discard as many cards as possible; zero does not ask for a choice. Selected
cards move through the shared discard replacement batch before discard events
are sent. Dynamic amounts, saved discard results, discard costs, and undeclared
event players remain unsupported RON forms. Evidence:
`TestRONFixedDiscardRealCards`, `TestRONFixedDiscardContexts`,
`TestRONFixedDiscardReplacementAndEvents`,
`TestRONFixedDiscardZeroEmptyAndContinuation`,
`TestRONFixedDiscardActivationRollback`, `TestRONFixedDiscardInvalid` and
`TestRONFixedDiscardInventory`.

## RON spell-cast event references

Inside `TriggerA(WhenAPlayerCastsASpell(AnyPlayer, filter), actions)`, symbol
`Trigger_ThatSpell` reads the cast spell and symbol `Trigger_ThatPlayer` reads
its caster. CounterSpell accepts the event spell; supported draws/discards and
fixed PlayerMayCost accept the event player. ControllerOfSpell accepts the event
spell and reads its current controller. The event references do not add targets.
The compiler rejects these references outside this event and rejects explicit
event-player life actions. Existing spell filters and payment continuations retain
their shapes. See [the supported contract and tests](designs/mtgish-ron-spell-cast-references.md).

## RON temporary animation and base values

In the existing end-of-turn `CreatePermanentLayerEffectUntil/3` action, fixed
`SetPT(PT(integer, integer))` sets base P/T; `AddCardtype` supports Creature,
Artifact, Land and Enchantment. `AddCreatureType(symbol)` adds a creature subtype;
`SetCreatureType(symbol)` replaces the creature subtype family. Existing types
and abilities remain, and all modifications share one engine `Until` installation.
For becoming a named artifact creature, use replacement as required by CR 205.1b.
The compiler rejects invalid types/subtype families before building a factory.

Permanent-source `TargetPermanent` filters accept `Other(ThisPermanent)` alone
or under `And` with existing filters. They use the engine's source-exclusion
option at announcement and resolution. `Other` in disjunctions or with other
reference kinds remains unsupported. Exact scope, engine reuse, source corrections
and gameplay evidence: [temporary animation](designs/mtgish-ron-temporary-animation.md).
Dynamic base values, collection animation and additional durations remain gaps.

### RON temporary collection P/T

`CreateEachPermanentLayerEffectUntil(filter, [AdjustPT(integer, integer), ...],
UntilEndOfTurn)` accepts a nonempty modification list in supported resolving
action contexts. It reuses generic permanent filters and
`ControlledByAPlayer(SinglePlayer(You))` alone or under conjunction. Controller
selectors under disjunction, source exclusions, target collections and other
modifications/durations remain errors. You means the resolving controller.

The compiler reuses the immutable permanent-set filter plan and lowers to
`Until(EndOfTurn, ModifyEachPT(...))`. Members are captured at resolution;
later entries are excluded, and later control or combat changes do not change
membership. Phased objects are excluded. Installed effects keep incarnation
identity and expire at cleanup. See
[the collection report](designs/mtgish-ron-temporary-collection.md) for exact
contexts, full-record audit and gameplay mappings.

## RON activation timing

On permanent cards, `ActivatedWithModifiers(cost, actions, modifier)` accepts
`ActivateOnlyIf(PlayerPassesFilter(You, IsTheirTurn))` and
`ActivateOnlyIf(PlayerPassesFilter(You, IsTheirUpkeep))`. Costs and action bodies
use the same builders as `Activated`. The engine checks the current activating
controller's turn, and the upkeep step where required, before payment. The
restriction does not repeat when the ability resolves. Every factory creates
fresh costs, targets, effects and abilities. The activation limit forms below
can be combined with this timing. Other modifier forms remain errors. See [the timing slice](designs/mtgish-ron-activation-timing.md)
for exact shapes and real-card evidence.


## RON activation limits

Permanent `ActivatedWithModifiers` accepts the symbol `ActivateOnlyOnceEachTurn`
and `ActivateNoMoreThanNumberTimesEachTurn(Integer(n))` for a positive fixed
integer. A nonempty `And([modifier, ...])` combines supported limits and timing;
nested conjunctions are allowed. More than one numbered maximum is rejected.
Dynamic or nonpositive limits remain errors. Costs and actions retain their
existing context checks.

Limits belong to an ability of one permanent incarnation. Failed activation
consumes no use; control changes and phase cycles retain uses. Returned cards
start unused. Copies do not consume uses, cloned games keep separate counts,
and final cleanup resets phased sources too. The shared engine service copies
runtime action state before it records a successful activation; card definitions
remain unchanged. See [activation limits](designs/mtgish-ron-activation-limits.md)
for exact shapes, dependencies and real RON gameplay evidence.

RON activation costs accept `SacrificeAPermanent(filter)` through the existing
matching-sacrifice transaction. The activating player chooses one matching
permanent they control, including the source when it matches. Costs are fresh
per factory; nested filter `And`/`Or` lists must be nonempty. Saved sacrifice
results, source-relative filters and resolution/optional/casting contexts remain
unsupported. See [the exact contract and tests](designs/mtgish-ron-sacrifice-costs.md).

## RON single-permanent counter placement

`PutCounters([ACounterOfTypeOnPermanent(counter, permanentRef)])` places one
counter. `NumberCountersOfTypeOnPermanent(value, counter, permanentRef)` in the
same one-instruction list uses an integer literal value or already bound ValueX.
Nonpositive amounts place nothing. Typed source, Aura host and declared permanent
target references use the same scope and incarnation rules as other actions.
Placement uses AddCountersTo and the shared replacement pipeline. Counters
remain through cleanup; returned objects do not receive pending placement.

Named permanent kinds and supported literal PTCounter pairs are validated before
factory construction. Multiple instructions, player counters, unsupported
values/kinds, entry/removal and placement-as-cost remain errors. See
[the exact shapes, inventory audit and tests](designs/mtgish-ron-counter-placement.md).

## RON fixed source-counter activation costs

`RemoveCounters([ACounterOfTypeFromPermanent(counter, ThisPermanent)])` removes
one counter as an activation cost. `NumberCountersOfTypeFromPermanent(Integer(n),
counter, ThisPermanent)` uses a nonnegative fixed amount, including zero. The list
must contain one instruction. Supported permanent counter types match placement.
Use these costs in `Activated` or `ActivatedWithModifiers`, including `And` costs.
Factories create fresh `RemoveCountersCost` instances; payment uses the shared
atomic transaction before resolution. Other source references, X/any/all amounts,
removal actions and optional counter payments remain errors. See
[the exact contexts and tests](designs/mtgish-ron-counter-costs.md).

## RON basic creature-token creation

RON `CreateTokens/2` lowers one creature-token instruction and an empty modifier
list to `mage.CreateTokenCopies(mage.You, count, definition)`. Instructions accept
`TokenWithPT/6`, `NamedTokenWithPT/7`, or `NumberTokens/2`. Printed characteristics
use `DefineToken` with typed P/T, types, colors, subtypes and keyword options.
Counts use fixed integers or in-scope announced spell X at resolution. Empty
optional options are omitted. Each action builds a fresh effect; the engine's
immutable token definition contains no chosen state.

See [the supported shapes and card tests](designs/mtgish-ron-token-creation.md).
Token abilities, supertype options, copying, creation modifiers and saved token
results remain unsupported by this adapter.

### RON sacrifice payment values

In `Activated/2` and `ActivatedWithModifiers/3`, a single
`SacrificeAPermanent/1` cost can supply a saved number to its resolving actions.
`ToughnessOfTheSacrificedCreature` requires the exact filter
`IsCardtype(Creature)`. `ManaValueOfTheSacrificedPermanent` accepts the existing
matching-sacrifice filters. Both references are symbols. The number is captured
at payment, before sacrifice, and belongs to that activation.

The supported consumers are `GainLife/1`, `LoseLife/1`, `DrawNumberCards/1`, and
`NumberTokens/2` in the existing basic token-creation forms. These references do
not enable dynamic mana, damage, static values, optional sacrifice payments, or
spell additional costs. A declaration cannot combine both characteristic
references or use more than one sacrifice cost, including source sacrifice.
Each factory builds the cost and its consumers with the same private engine
`NumberBinding`; copies keep the paid number without paying again.

### RON source counter payment results

An activation can pay one
`RemoveCounters([AnyNumberOfCountersOfTypeFromPermanent(counter, ThisPermanent)])`
cost. Symbol `TheNumberOfCountersRemovedThisWay` reads its saved paid count in
that activation's life, draw, basic token-count or repeated-mana actions.
`AddManaRepeated(production, TheNumberOfCountersRemovedThisWay)` adds the count
in each declared production; `AnyManaColor` chooses one color for the amount.
Zero is a valid payment. Each factory builds a private typed number binding.
Unbound, tuple, cross-rule, non-source and ambiguous counter-payment forms fail
compilation. Optional resolution payments remain unsupported. The shared engine
transaction locks the choice and preserves rollback and source identity.
Mana production queries do not choose counter payments; choice-cost abilities
remain outside automatic mana planning. See the
[counter payment report](designs/mtgish-ron-counter-payment.md) for shapes and tests.

## RON end-step trigger adapter

Permanent-source RON rules accept
`TriggerA(AtTheBeginningOfAPlayersEndStep(AnyPlayer), actions)` and
`TriggerA(AtTheBeginningOfAPlayersEndStep(SinglePlayer(You)), actions)`.
The first form matches either player's end step. The second form uses the
source's current controller when the event occurs. Both use the engine's step
patterns and pending-trigger path.

Within these action lists, symbol `Trigger_ThatPlayer` is a typed reference to
the player whose end step began. It is separate from `You`, which refers to the
pending ability's controller. It can supply existing player actions, including
life, draw, discard and optional fixed mana payment. Each factory builds a fresh
player binding and any declared targets. Targets are chosen at stack placement.

Other end-step player filters, `TriggerI`, and `CreateFutureTrigger` remain
unsupported. This adapter does not provide history queries or a delayed trigger.
Evidence: `TestRONEndStep*`, including complete Ball Lightning and Breeding Pit
RON gameplay; [slice report](designs/mtgish-ron-end-step.md).

### RON tapped entry

Permanent cards accept exactly
`AsPermanentEnters(ThisPermanent, [EntersTapped])`. The compiler lowers this
form to the engine's `EntersTapped` entry attribute. The permanent is tapped
before entry effects and triggers inspect it. Entry does not create a tap event
or put a tapped-entry ability on the stack. Normal untapping is permitted.
Re-entry applies the rule again; phasing does not. Card copies retain the rule,
and cloned games keep separate tapped states.

The source must be the symbol `ThisPermanent`, and the list must contain only
the symbol `EntersTapped`. Empty lists, other subjects, tuple-shaped symbols,
additional operations, entry choices and entry counters remain unsupported.
`EntersTapped` is not a general action or a top-level keyword declaration.
See `TestRONEntryTapped*` and the [slice report](designs/mtgish-ron-entry-tapped.md).

## RON creature death history in counter amounts

`PutCounters([NumberCountersOfTypeOnPermanent(amount, counter, subject)])`
accepts `NumCreaturesOrPlaneswalkersThatDiedThisTurn(IsCardtype(Creature))`
as its amount in supported spell, activation and trigger action lists. This
exact filter lowers to `mage.CreaturesDiedThisTurn()`. The count is read when
placement resolves. It includes tokens and creatures from either player,
including deaths before the source entered. It does not count exile, return to
hand, phasing or prevented destruction. Other history filters remain errors.
The existing typed subject and counter replacement contracts still apply.
RON gameplay evidence: `TestRONDeathHistory*` for Khabál Ghoul and all six
Scavenging Ghoul records. Intervening-if history conditions remain unsupported.

## RON storage-land lifecycle

RON supports `PermanentRuleEffect(ThisPermanent,
[MayChooseNotToUntapDuringUntap])` and
`TriggerI(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),
PermanentPassesFilter(ThisPermanent, IsTapped), actions)` on permanent cards.
The first uses the ability layer's optional untap attribute. The second checks
current tapped state at detection and resolution through WithInterveningIf.
Actions use the shared compiler and fresh instance bindings. Other operations,
subjects, conditions and events remain declaration errors. Full-card and invalid
shape evidence is recorded in the storage-land lifecycle entry in
[the progress ledger](designs/mtgish-ron-progress.md).

## RON next-untap restriction action

`PermanentDoesntUntapDuringControllersNextUntap/1` accepts an in-scope symbol
`ThisPermanent`, `HostPermanent`, or `Ref_TargetPermanent`. The shared resolving
action compiler supports it in spell, activation and supported trigger action
lists. The permanent source, Aura host or single typed target must exist in that
declaration context. Wrong arity, tuple references, undeclared reads and reads
from another rule fail compilation.

The action lowers to `mage.SkipNextUntap` with the instance's typed reference.
It does not tap the permanent or prevent other untap effects. The restriction
expires during the next untap step of the permanent's current controller, even
if the permanent is untapped. Control changes do not capture the old controller.
Repeated applications expire together. See
[the contract and tests](designs/mtgish-ron-next-untap.md).
Collection actions remain unsupported by this adapter. The self static form
is described below.

## RON self untap prohibition

The permanent-rule compiler accepts
`PermanentRuleEffect(ThisPermanent, [DoesntUntapDuringControllersUntap])`.
It creates a source-bound static ability through `Static`, `While`, and
`GrantKeywordModification(This, AttrDoesNotUntap)`. The existing optional-untap
form still uses `AttrMayNotUntap`. The operation list must contain exactly one
symbol. Only the source symbol `ThisPermanent` is accepted in this context.

The prohibition applies during each untap step of the current controller.
Other untap effects remain permitted. Ability removal disables the prohibition;
cleanup can restore it. Conditional, Aura-host, collection, and temporary rule
forms remain rejected. See [the slice ledger](designs/mtgish-ron-progress.md)
for the real-card tests and supported contexts.

## RON global landwalk blocking overrides

`EachPermanentRuleEffect(And([IsCardtype(Creature),
HasAbility(Landwalk(IsLandType(subtype)))]),
[CanBeBlockedAsThoughItDidntHave(Landwalk(IsLandType(subtype)))])`
uses the existing global landwalk blocking rule. The two filter terms may be
in either order. Both subtype operands must match. Supported subtypes are
Plains, Island, Swamp, Mountain, Forest and Desert.

The creature keeps its landwalk ability. Only blocking checks ignore that
landwalk; other evasion abilities still apply. The source must be phased in,
on the battlefield, and have its static ability. Each factory builds a fresh
effect. Extra filters, other ability operands, mixed operation lists, and
self-only or temporary contexts remain unsupported. This adapter does not add
general landwalk filter support to targets or other selectors.

## RON choose-one spell modes

`SpellActions(Modal_ChooseOne([mode, mode, ...]))` accepts at least two
mode bodies. Each body uses the supported `ActionList` or `Targeted` grammar.
Each mode has its own target scope. A target reference cannot read a declaration
from another mode. Each factory call builds fresh targets and effects. Internal
declaration names include a mode namespace; RON references keep their spelling.

The adapter uses `NewModalSpell`. The controller chooses one mode and only its
targets during casting. Resolution executes only that mode and checks its target
legality. Copies retain the mode and can choose new targets for that mode.
Mode labels use effect sequence text with readable target names. All mode bodies must compile;
an unsupported body rejects the entire record. Nested modes, activation modes,
trigger modes, and `Targeted_Modal` remain unsupported. Life actions in player
target scopes and future prevention remain separate gaps for Healing Salve
and Alabaster Potion.

RON `PutPermanentIntoItsOwnersHand(reference)` uses `mage.ReturnToHand`.
The argument must be one in-scope permanent reference: `Ref_TargetPermanent`,
`ThisPermanent` on a permanent source, or `HostPermanent` on an Aura.
The existing action compiler accepts it in spells, choose-one spell modes,
activations and supported triggers. Target declarations are built per factory
instance. Unsupported enclosing forms still fail compilation.
The action reads the current object through its captured incarnation and moves
its card to its owner's hand. Phased or departed objects are absent; a returned
card is a new object. An Aura uses its current attachment, or the last attachment
of its captured source after departure. It cannot follow a returned source.

## RON typed single damage targets

Symbol `AnyTarget` in the existing single-target `Targeted` form now creates a
fresh `mage.DeclareDamageTarget` declaration. Symbol `Ref_AnyTarget` in
`SpellDealsDamage(ThisSpell, value, Ref_AnyTarget)` or
`PermanentDealsDamage(ThisPermanent, value, Ref_AnyTarget)` reads that instance's
recipient through `mage.DealDamageFrom`. The engine validates the declaration
and read together. The recipient can be a player, creature or planeswalker;
an ordinary land cannot be the target. Values retain the existing fixed/X
resolution contract.

Spell modes have separate declaration names and bindings. Copies retain their
mode and can change the recipient through the engine's target service. Target
legality, permanent incarnation, source LKI, damage replacement and prevention
remain engine behavior. This adapter also works in existing activation and
trigger action lists. Numbered references, multiple target groups, target unions
with other filters, and future prevention remain unsupported shapes. Evidence:
`TestRONDamageScope*` and the migration ledger's P0-TYPED-DAMAGE-RECIPIENT entry.

## RON targeted temporary blocking restriction

`CreatePermanentRuleEffectUntil(Ref_TargetPermanent, [CantBeBlocked],
UntilEndOfTurn)` is supported in resolving action lists with a declared
`TargetPermanent`. Spell, choose-one spell mode, activated ability and supported
trigger contexts use the same per-instance target binding. Other rule operations,
subjects and durations remain rejected.

Permanent filters accept `PowerIs(LessThanOrEqualTo(Integer(n)))`,
`PowerIs(GreaterThanOrEqualTo(Integer(n)))` and `PowerIs(EqualTo(Integer(n)))`
for signed integer constants. These filters read current power at each query;
combine them with `IsCardtype(Creature)` when the subject must be a creature.
Dynamic bounds and other comparison forms remain unsupported.

Dwarven Warriors and Tawnos's Wand use the power-at-most-two target and the
same blocking restriction. Target legality is checked at announcement and
resolution. A later power change does not end a resolved restriction. The
restriction follows the captured incarnation until cleanup and survives source
departure, ability removal and a phase cycle. It does not follow a returned card.

## RON collection untap restrictions and tap actions

`EachPermanentRuleEffect(filter, [DoesntUntapDuringControllersUntap])` accepts
one bare restriction operation and a supported, source-independent permanent
filter. The compiler uses `PreventUntapDuringStepForMatching`. The engine checks
final characteristics after phasing and before the untap operation. It does not
grant an ability to the selected permanents. Other untap effects remain legal.
The rule stops when its source leaves, phases out, or loses its abilities.
Membership is selected again at each untap decision.

`IsSupertype(Legendary)` is supported in permanent filters, including nested
`And` and `Or`. Other supertypes and card/spell filter contexts remain unsupported.
The filter uses `IsLegendary`.

`TapEachPermanent(filter)` accepts a supported, source-independent permanent
filter in resolving spell, mode, activation and trigger action lists. It lowers
to `TapEach(MatchingPermanents(filter))`. Selection occurs at resolution, excludes
phased permanents, and taps the selected permanents together before tap events.
Target-group, chosen-result and player-dependent collection selectors remain
unsupported by this adapter.

Source-entry triggers now use `WhenEvent(Enters(This), body)`. An event object is
not a target declaration. A nontargeted trigger can resolve after its source
leaves or phases out. See the progress ledger for complete real-card mappings.

### RON random discard actions

In resolving spell, mode, activation and supported trigger action lists,
`DiscardACardAtRandom` is a bare symbol. `DiscardNumberCardsAtRandom(Integer(n))`
requires a nonnegative integer; `DiscardNumberCardsAtRandom(ValueX)` requires an
announced X scope. Other amounts, tuple forms of the singular action, and random
discard costs remain rejected. These actions lower to `mage.DiscardAtRandom`.

Direct actions affect the resolving controller. `PlayerAction(player, action)`
uses the existing typed player reference plan: `You`, declared `Ref_TargetPlayer`,
or an available step/cast event or spell-controller reference. Each factory
builds its own target/event bindings. A player read without its declaration is
an error. A target declaration alone does not change the direct action's player.
Selection occurs at resolution before the shared simultaneous discard batch;
it does not ask the affected player to choose cards.

Full-record gameplay mappings: Hymn to Tourach, Mind Twist (all six reprints),
Ring of Renewal and Gwendlyn Di Corci in `TestRONRandomDiscard*`. The engine
contract is documented in the random-discard section of
[engine-rules-contracts.md](engine-rules-contracts.md).

### RON printed rampage

A permanent's top-level `Rules` list accepts `Rampage(n)` with one nonnegative
integer literal. It lowers to `mage.RampageTrigger(n)` with a fresh ability per
factory construction. Zero is legal and still creates the trigger. Bare symbols,
negative values, `Integer(n)`, X, extra arguments, and resolving-action uses are
rejected during compilation. `AddAbility([Rampage(n)])` and selected ability
choices remain unsupported contexts.

Rampage counts blockers at resolution and grants its fixed bonus until cleanup.
Its source remains the original incarnation. Full-record gameplay mappings:
`TestRONRampageRealCards` covers Aerathi Berserker, Chromium, Craw Giant, Frost
Giant, Hunding Gjornersen, Marhault Elsdragon and Wolverine Pack.
`TestRONRampageNativeCombat` checks declared-blocker legality, flying and trample.
Chromium also uses the shared `TestRONUpkeepPayment*` lifecycle tests.
