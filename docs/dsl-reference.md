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

In this end-of-turn context, `LosesAbility(keyword)` removes all instances of
one supported keyword. `Landwalk(IsLandType(symbol))` is accepted for both
`LosesAbility/1` and the existing `AddAbility([keyword, ...])` form. The supported
land symbols are Plains, Island, Swamp, Mountain, Forest and Desert.
`LosesAbility(AnyLandwalk)` uses the engine's complete landwalk removal service,
including legendary landwalk. These operations use layer 6 and timestamp order:
a later grant can restore an ability. Removal does not prohibit later grants.
Static removal, `AnyBandsWithOthers`, `TheChosenAbility`, non-keyword removal and
other durations remain unsupported. The tapped-source duration below also
accepts these modifications. Evidence is in
`TestRONTemporaryKeyword*` and `TestRONTemporaryLayersDeclareTypedReads`.

Temporary `CreatePermanentLayerEffectUntil/3` also accepts
`SetPower(Integer(n))` for any integer literal. It sets base power in layer 7b
through `SetBasePower` and preserves base toughness. Bonuses and counters still
apply in layer 7c. Existing typed subjects and durations apply. Dynamic values
and static `SetPower` remain unsupported. Island of Wak-Wak requires a flying
creature target; Singing Tree requires an attacking creature target. The engine
checks these restrictions at announcement and resolution. After resolution,
losing flying or leaving combat does not end the effect. Evidence:
`TestRONTemporaryPower*`.

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
Captured spell sources and unknown battlefield incarnations do not match these
permanent filters. UUID-only actions keep live permanent lookup. The shared
lookup is internal; the public signatures above do not change.
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
abilities remain separate unsupported shapes. Conditional rules have the
restricted contract below.
See [the exact contract and tests](designs/mtgish-ron-static-permanents.md).

## RON optional fixed mana payments

An adjacent `PlayerMayCost(player, PayMana([...]))` and
`Unless(CostWasPaid, [actions])` pair builds one `UnlessPaid` effect.
Each pair keeps its own result, including nested unpaid bodies. Payment uses
the engine's total-cost transaction during resolution. Spell-controller payer
references use the current controller of the declared stack target.
Variable and alternative payments remain rejected. Optional sacrifice payments
use the adapter below; other nonmana payments remain rejected.
Contract and tests: [RON optional payments](designs/mtgish-ron-optional-payment.md).

### RON optional sacrifice payments and branch polarity

In supported spell, activation and trigger action sequences,
`MayCost(SacrificeAPermanent(filter))` uses the resolving controller as payer.
`PlayerMayCost(player, SacrificeAPermanent(filter))` uses the existing typed
player-reference grammar. The filter uses the activation sacrifice-cost grammar,
including nonempty recursive `And`/`Or`. It selects one current, unphased
matching permanent the payer controls, including the source if it matches.
The selected card moves to its owner's graveyard. Regeneration does not apply.

Each payment must have an adjacent `If(condition, [actions])` or
`Unless(condition, [actions])`, with a nonempty action body. Symbol
`CostWasPaid` or `CostWasntPaid` supplies the condition. `If(CostWasPaid, ...)`
and `Unless(CostWasntPaid, ...)` run after successful payment;
`If(CostWasntPaid, ...)` and `Unless(CostWasPaid, ...)` run after decline or
failed payment. Both conditions also work with the existing fixed mana costs.
The payment result stays local to this pair, including nested pairs.
Detached result reads, plural sacrifice costs and other optional costs remain
errors. This does not add a general resolving `If` adapter.

The adapter uses `SacrificeMatchingIncludingSourceCost`, `IfPaid`, `UnlessPaid`
and the shared cost transaction. `TestRONOptionalSacrifice*` covers complete
Elder Spawn and Yawgmoth Demon records, declaration rejection, payment choices,
source identity, phasing, copies, clones, nested results and native loading.

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
spell filters and `SpellCast`. The opponent scope and event references have
separate contracts below. An adjacent fixed-mana MayCost or PlayerMayCost
followed by `If(CostWasPaid, [actions])` uses `IfPaid`. Each nonempty body has
its own payment result, including nested pairs. Payment is offered at resolution
to the stored ability controller or explicit typed payer. X and other nonmana
payments remain separate shapes; optional sacrifice support is described above.
See [the contract and tests](designs/mtgish-ron-spell-cast-life.md).

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
Token source damage-to-player triggers are supported as described below. Other
token abilities, supertype options, copying, creation modifiers and saved token
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

## RON typed permanent damage recipients

RON damage actions also accept `Permanent(Ref_TargetPermanent)` as the recipient
in `SpellDealsDamage(ThisSpell, value, recipient)` and
`PermanentDealsDamage(ThisPermanent, value, recipient)`. The enclosing action
must declare its own single `TargetPermanent`. Spell, mode, activation and
supported trigger action lists use that declaration's fresh
`mage.PermanentTargetBinding` with `mage.DealDamageFrom`.
The wrapper does not select a new target or bypass the target filter at
resolution. Unbound, numbered, player, source and host operands remain rejected
in this shape. Existing fixed and announced-X values retain their contracts.
Evidence: `TestRONPermanentDamage*`.

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
announced X scope. Other amounts and tuple forms of the singular action remain
rejected. These actions lower to `mage.DiscardAtRandom`. Random discard costs
have the separate declaration contract below.

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

## RON resolving continuous durations

### Next-upkeep duration

`CreatePermanentLayerEffectUntil(subject, modifications, UntilNextUpkeep(You))`
accepts the existing singular permanent references and temporary modifications.
It applies in resolving action bodies, including spells, activations, supported
triggers, and supported delayed actions. The duration operand must be bare `You`;
other players and malformed declarations are rejected. Collection modifications,
static operations, costs, and control changes gain no new support.

The adapter uses
`mage.UntilNextUpkeep(player mage.PlayerRef, modifications ...mage.ContinuousModification)`.
The engine supports validated typed player references, saves the player at
resolution, and captures the existing modification subjects and values. An absent
player starts no effect. Changes to the source or recipient controller do not
change the saved player. Source departure, return, ability removal, and phasing
do not end the duration. Recipient departure breaks the captured identity;
recipient phasing suspends application but does not postpone expiry.

The effect ends as the next actual upkeep of the saved player begins, before
upkeep event detection and priority. It lasts through cleanup and other players'
upkeeps. A skipped upkeep does not expire it; an extra upkeep can expire it in
the same turn. This operation creates no delayed trigger or stack object.
`Until(UntilYourNextTurn, ...)` remains an invalid declaration; the explicit
player adapter supplies the lifetime policy.

Erhnam Djinn uses this duration with its controller-upkeep trigger and opposing
non-Wall creature target. Xenic Poltergeist still needs `SetCardtypes` and
`SetPowerAndToughnessBoth` adapters. Halfdane remains rejected for `ManualPT`;
its source duration also differs from local Oracle text (end of next upkeep).

### Tapped-source duration

`CreatePermanentLayerEffectUntil(subject, modifications,
ForAsLongAsPermanentRemainsTapped(ThisPermanent))` supports the existing typed
subjects and temporary modifications in resolving permanent abilities, including
activations and supported triggers. The source operand must be the symbol
`ThisPermanent`. Host, target, event and numbered source operands are rejected.
Spell-only, static, cost, collection-layer and control-duration contexts are not
added. Existing `UntilEndOfTurn` declarations retain their behavior.

The compiler lowers to this engine service:

```go
func UntilSourceUntaps(source sourceRef, modifications ...ContinuousModification) ResolvingEffect
```

Call it with `mage.This`. The service captures the source incarnation and
modification subjects/values at resolution. It starts only if that source is
present, phased in and tapped. An untap replaced by removal of a stun counter
does not end it. Actual untapping, source departure and source phasing end it
permanently; retapping or phase-in cannot restart it. Control changes do not
end this tapped-state duration. Target phasing suspends application, and target
return does not change the captured membership.

Tawnos's Weaponry, Spirit Shield and Zelyon Sword use this duration with their
printed mana/tap costs and P/T bonuses. Their separate permission to remain
tapped during untap still permits the choice after the target leaves.
Evidence: `TestRONSourceTappedDuration*` and `TestUntilSourceUntaps*`.

## RON opponent-land tap triggers

`TriggerA(WhenAPermanentBecomesTapped(And([IsLandType(subtype),
ControlledByAPlayer(Opponent)])), actions)` uses
`mage.OpponentLandBecomesTapped().OfSubtype(subtype)`. The conjunction must
contain exactly these two filters, in either order. The subtype must be a symbol
that the engine accepts as a land subtype. Existing subtype spelling conversion
applies, including `Urzas` and `PowerPlant`. Extra filters, disjunctions,
source/host selectors, other controller selectors and event-result reads remain
unsupported in this context. The existing trigger action compiler supplies
per-instance actions and local targets; this event does not declare a
`Trigger_ThatPermanent` or `Trigger_ThatPlayer` binding.

Matching uses the land's type, subtype and controller when it becomes tapped.
The pending trigger retains its controller. Mana tapping and other tap operations
can trigger it; entry tapped and tapping an already tapped land cannot. Source
or land departure and later control changes do not erase a pending trigger.
Phased sources do not trigger. Copies use the copy controller for `You`.

Real RON evidence: Lifetap (all six reprints) and Lifeblood in
`TestRONOpponentLandTapRealCards` and `TestRONOpponentLandTapPaymentAndBatch`.
Declaration and syntax evidence: `TestRONOpponentLandTapInvalidAndInventory`
and `TestRONOpponentLandTapSubtypeAndOrder`.

### RON Aura host tap events

On an Aura, `TriggerA(WhenAPermanentBecomesTapped(SinglePermanent(HostPermanent)),
actions)` declares `Trigger_ThatPermanent` for that action body. The reference
reads the permanent captured when the enchanted permanent becomes tapped.
It is distinct from `HostPermanent`, which reads the attachment at resolution.
Existing typed permanent actions can use this binding. It does not declare
`Trigger_ThatPlayer` or supply a controller query or damage-source adapter.

Each factory call creates its own `BindPermanent` declaration. Lowering uses
`AttachedBecomesTapped` and `From`; the engine retains separate captures for
pending events, copies and clones. The exact form executes Blight and Spirit
Shackle. Other `SinglePermanent` tap-event subjects remain unsupported.

## RON global combat prevention

In a resolving action list, the RON form
`CreatePreventDamageUntil(CombatDamageWouldBeDealt, [PreventThatDamage], UntilEndOfTurn)`
uses `mage.PreventAllCombatDamage`. The event and duration must be symbols.
The operation list must contain exactly the symbol `PreventThatDamage`.
The adapter rejects other events, operations and durations before factory creation.
It does not declare a source, recipient, target or result binding.

The shared engine installs prevention at resolution and removes it at cleanup.
It applies to all combat damage for the rest of the turn, including later combats,
and leaves noncombat damage unchanged. Fog, Darkness and Holy Day use this
form in spells; Spore Flower uses it after its three-counter activation cost.
Tests: `TestRONGlobalCombatPrevention*`. Other `CreatePreventDamageUntil` shapes
and future amount shields remain separate gaps.

## RON source combat prevention

`CreatePreventDamageUntil(CombatDamageWouldBeDealtByCreature(ref), [PreventThatDamage], UntilEndOfTurn)`
accepts a bound typed permanent reference in a resolving action. The event is
an exact one-argument tuple; its operation list has exactly the symbol
`PreventThatDamage`, and the duration is exactly the symbol `UntilEndOfTurn`.
Other events, operations, durations, unbound references and cost/static/rule
contexts remain declaration errors. The existing target filter checks creature
legality at announcement and resolution.

The adapter builds `mage.PreventCombatDamageByThisTurn(PermanentRef)` with
per-instance bindings. It captures the resolved incarnation and installs the
existing source prevention replacement until cleanup. It prevents combat damage
by that source, including later combats, but allows damage to that source and
noncombat damage. A returned permanent is a new source; phasing and control
changes preserve the captured identity. Captured old-source damage keeps its
identity after departure. `PreventCombatDamageBy` and `PreventCombatDamageByEach`
still end at end of combat.

Real RON tests cover Horn of Deafening, Lady Evangela and both Subdue instructions.
Telekinesis still requires its separate future untap action. Filtered source sets,
recipient prevention and finite shields are outside this supported shape.

## RON controller restriction on permanent targets

`TargetPermanent(ControlledByAPlayer(SinglePlayer(You)))` restricts a target
by the controller of the spell or ability. It can occur inside nested `And`
filters with existing permanent filters and `Other(ThisPermanent)`.
The adapter uses `mage.TargetYouControl()` on each fresh target declaration.
This restriction is available in supported spell, mode, activation and trigger
contexts. Ownership does not determine target legality. The target must remain
under that controller at resolution. A later control change does not end an
already resolved characteristic modification.

Only the exact `SinglePlayer(You)` shape is supported here. Opponent expressions,
other player references, ownership restrictions and disjunctions containing
controller restrictions remain errors. Static and collection filters retain
their separate adapters. Ashnod's Battle Gear uses this target restriction with
the existing source-tapped duration and optional untap services.

## RON combat prevention to and by one permanent

The resolving action has this exact form:

```ron
CreatePreventDamageUntil(
    Or([
        CombatDamageWouldBeDealtToRecipient(Permanent(Ref_TargetPermanent)),
        CombatDamageWouldBeDealtByCreature(Ref_TargetPermanent),
    ]),
    [PreventThatDamage],
    UntilEndOfTurn,
)
```

`Or/1` takes a list with exactly one recipient event and one source event,
in either order. Both events require one argument. The recipient uses the
`Permanent/1` wrapper. Both references must identify the same lexical permanent
plan. Supported references are the existing scoped `Ref_TargetPermanent`,
`ThisPermanent`, `HostPermanent`, and bound `Trigger_ThatPermanent` forms.
Unbound, numbered, player, different-subject, duplicate-event, additional-event,
wrong-operation and wrong-duration declarations fail compilation.

This action is available in supported spells, modes, activations and trigger
bodies. It is not a cost, layer modification or static `PreventDamage` rule.
The compiler builds `mage.PreventCombatDamageToAndBy(PermanentRef)` with fresh
instance bindings. The engine owns incarnation capture, replacement selection,
source snapshots and cleanup. This is a union of two events for one object;
it does not introduce a general event-union executor.

Real RON tests cover every rule of Maze of Ith, Ebony Horse (ARN and 3ED), and
Elvish Scout. They retain attacking-creature target checks, the controller
restriction for Horse and Scout, costs, untap and prevention. Untap does not
remove an attacker from combat. Failure to untap due to a stun counter does not
stop prevention. Neither source departure nor later recipient control changes
end installed prevention. Returned permanents have new incarnations.

Finite shields, static Aura prevention (including Gaseous Form), different
source/recipient unions and other durations remain separate gaps.

## RON hand-size values and ordinary upkeep events

`TheNumberOfCardsInPlayersHand(player)` reads the selected player's current hand
when the action resolves. It accepts the existing typed player references,
including `You`, declared `Ref_TargetPlayer`, and `Trigger_ThatPlayer` in a
matching event scope. `Minus(left, right)` subtracts two supported resolution
values recursively. Intermediate results retain their sign; life and damage
services treat nonpositive final amounts as zero.

These values are supported where `compileInstanceValue` is used: resolving
life, draw, damage and existing token-count actions. They do not establish
static values, casting costs, chosen-player references, damage-result players,
or arbitrary temporary P/T value support. Unbound reads remain errors.

`TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer), body)` and the
`SinglePlayer(You)` form use the same per-instance player-binding service as
ordinary end-step events. `Trigger_ThatPlayer` saves the player whose upkeep
began; `You` remains the resolving ability's controller. Other player filters
and intervening-if upkeep declarations retain their separate adapters.

`SpellDealsDamage` and `PermanentDealsDamage` now build typed damage actions
with resolution values. `Player(player)` accepts a valid typed player reference;
existing bare target recipients and targeted `Permanent(...)` remain supported.
The shared engine supplies target checks, source snapshots and prevention.
Full-record evidence: Ivory Tower (ATQ/3ED), Storm World (LEG), and Copper Tablet
(LEA/LEB/2ED/CED/CEI) in `TestRONHandSize*`.

## RON static filtered blocking restriction

`PermanentRuleEffect(ThisPermanent, [CantBeBlockedExceptByDefenders(filter)])`
and the Aura form with `HostPermanent` support exactly one operation. The filter
uses the existing source-independent permanent-filter grammar, including nested
`And`/`Or`, type, subtype, color and keyword queries. Each possible blocker must
pass this filter and every other combat restriction. `Defenders` here means
blocking creatures; it does not require the defender keyword. Reach does not
satisfy a filter that requires flying.

The compiler uses `mage.CantBeBlockedExceptBy(subject, filter)` in a static
`While` rule. The Aura reads its current attachment each layer pass. Source
ability removal, source departure or source phasing stops the rule; recipient
ability removal does not remove an Aura's rule. The rule does not grant an
ability to the recipient. Blocker characteristics are queried when block
legality is checked. A restriction change after a legal block does not remove
that block.

The typed engine modification also supports `Until(EndOfTurn, ...)`: it captures
its subject incarnation at resolution, survives source departure and a phase
cycle, and expires at cleanup. A returned card does not inherit the restriction.
This run adds only the static RON spelling, not new temporary RON operations.
Existing source and target constructors remain available. Evidence:
`TestFilteredBlocking*` and `TestRONFilteredBlocking*`.

### RON Aura host-controller upkeep

`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(HostController)), body)`
is supported on Auras. It uses `AttachedControllerUpkeep` with fresh event
bindings in each factory instance. `Trigger_ThatPlayer` is the player whose
upkeep caused the trigger. This binding is available only inside `body`.
A later control or attachment change does not change that player.

`HostPermanent` keeps its existing resolution-time attachment reference. Thus
Unstable Mutation puts its counter on the creature enchanted at resolution.
The event's internal host capture does not replace that reference. A departed
Aura uses the existing attachment LKI and cannot follow its new incarnation.

Complete-record tests cover Cursed Land, Feedback, Wanderlust, Warp Artifact,
and Unstable Mutation. This adapter does not add a resolving `HostController`
reference, optional payments, host-death triggers, or a host-controller end-step
pattern. Unsupported declarations remain errors.

## RON static collection attack restrictions

`EachPermanentRuleEffect(filter, [CantAttack])` accepts one bare `CantAttack`
symbol on a permanent. It lowers the static filter plan to
`Static(CantAttackEach(PermanentSet))`. The collection reads final characteristics
at attack declaration, including later flying grants or removal. Source control
and collection membership are current. The rule does not grant an ability to
recipients and does not remove creatures already in combat.

Static filters also accept `ExceptFor(filter)` for a pure complement and
`IsNamed(NamedCard("name"))` with a nonempty string. A complement with a contextual
controller or source exclusion is rejected. Existing conjunctions,
`ControlledByAPlayer(SinglePlayer(You))`, and `Other(ThisPermanent)` keep their
scope rules; source-dependent disjunctions remain errors. These adapters also
serve existing static layer contexts and the existing end-of-turn temporary
collection P/T adapter. Temporary collections keep their existing resolution-time
membership capture. No current RON record uses these new filters in that temporary
context. They do not add generic target filters,
card-zone filters or name choices.

Complete RON gameplay evidence covers Moat, Evil Eye of Orms-by-Gore, and
Akron Legionnaire in `TestRONAttackRuleCards`, `Lifecycle`, `FlyingChanges`,
`Native`, `NativeCast`, `StaticFilterScopes`, and `StrictLoadAndInventory`.
Other `CantAttack` contexts, temporary rules, attacking-player restrictions,
and mixed operation lists remain unsupported.

## RON static Aura control

On an Aura with `EnchantPermanent/1`, `PermanentLayerEffect/2` accepts
`HostPermanent` and a modification list containing `SetController(You)/1`.
`You` must be a bare symbol. This modification can accompany the existing
supported Aura modifications. Each factory creates a separate `ControlAttached`
continuous effect through `WithStaticAbility`. The controller is the Aura's
current controller in layer 2, not a player saved when the spell was cast.

`ThisPermanent`, collection control, chosen/target players, resolving actions,
and other control contexts remain unsupported. Control Magic and Steal Artifact
are covered by `TestRONStaticControl*`; their creature/artifact enchant filters
still govern casting and attachment legality. Collection loading remains strict.

## Static basic land type prerequisite

The existing engine constructors `BecomesBasicLandsEffect(filter, subtypes...)`
and `BecomesBasicLandAttachedEffect(subtypes...)` are available for fixed basic
land types. The collection effect now detects layer-4 selection dependencies;
the shared executor reevaluates remaining dependencies after each application.
The attachment effect requires the host incarnation saved by `Game.Attach`.
Do not construct attachments by setting `AttachedTo` alone.

Each new basic type supplies its intrinsic mana ability. The change removes old
land types and printed rules-text abilities, retains other subtype families and
supertypes, and permits separately granted abilities. Details and test evidence
are in [the engine contract](engine-rules-contracts.md#static-basic-land-types-dependency-and-attachment-identity).

RON supports `PermanentLayerEffect(HostPermanent, [SetLandType(BasicType)])`
on Auras and `EachPermanentLayerEffect(PureFilter, [SetLandType(BasicType)])`.
`BasicType` is a symbol: Plains, Island, Swamp, Mountain or Forest. Host effects
can include the existing P/T and keyword modifications. Collection effects must
contain only the type operation; combined operations need a shared membership
capture contract. Source-dependent collection filters remain rejected.

`IsSupertype(Basic)` and `IsNonSupertype(Basic)` read the basic supertype, not the
land subtype. Both filter forms also support Legendary. These forms use the
existing pure predicate and negation services. Conversion, Evil Presence and
Blood Moon have RON gameplay evidence. `SetLandTypeVariable`, resolving type
changes and entry choices remain unsupported.

## RON conditional static modifications

`If(condition, [layer_rules...])` on a permanent accepts a nonempty body of
`PermanentLayerEffect(subject, modifications)` and
`EachPermanentLayerEffect(filter, modifications)`. Singular subjects use
`ThisPermanent` or, on an Aura, `HostPermanent`. Collections use the existing
static filter adapter, including controller selection and source exclusion.

Conditions accept `PlayerPassesFilter(You, ControlsA(filter))`,
`PlayerPassesFilter(You, ControlsNo(filter))`,
`APlayerPassesFilter(Opponent, ControlsA(filter))`, or
`PermanentPassesFilter(subject, IsUntapped)`. Controller filters accept existing
card-type, subtype, color, supertype, tap/combat-state and `AnyPermanent` filters,
with nonempty recursive `And`/`Or`. Symbol `IsNonToken` is supported in these
static condition filters and lowers to `Not(IsToken)`. They reject ability and
P/T queries. `Opponent` is a player filter here, not a player reference. Only
`ControlsA` is supported on this opponent form.

Modifications accept fixed `AdjustPT(power, toughness)` or nonempty
`AddAbility([keyword...])` with existing basic keywords and supported landwalk
forms. One `If` body must use only layer 6 grants or only layer 7c bonuses.
Nested conditions, other body rules, mixed layers, chosen/event/target references,
counters, other player conditions, type/base-value changes and replacements remain
errors. No resolving `If` action support is implied.

The adapter reuses `PlayerControls`, `OpponentControls`, `ControlsNone`, `PermanentIsUntapped`,
`While`, `Static`, typed source/attachment references, permanent sets and
continuous modifications. Conditions and membership are live at each layer pass.
Factories create separate static rules and modifications without chosen state.
`TestRONConditionalStatic*` covers Kird Ape, Sedge Troll, Giant Tortoise,
Spectral Cloak and Angelic Voices, their reprints and invalid declarations.
`TestRONOpponentStatic*` covers Water Wurm, Beasts of Bogardan and Ivory
Guardians, including token exclusion, current colors, named recipients on both
battlefields, cumulative bonuses, protection and layer ordering.

`mage.OpponentControls(PermanentFilter) StateCondition` is a pure condition for
`While`, `IfState` and other state-condition consumers. It uses the context
controller's current opponent and current unphased battlefield membership.
An absent or unknown controller does not satisfy the condition. The declaration
holds no chosen state and is safe to reuse across factories and cloned games.

## RON static blocking prohibitions

`PermanentRuleEffect(ThisPermanent, [CantBlockAttackers(filter)])` prevents the
source from blocking attackers that match the filter. It lowers to
`SourceCanBlockOnly(Not(filter))`. The source restriction uses exactly one
operation; `HostPermanent`, target and event subjects remain rejected for this
form.

`PermanentRuleEffect(subject, [CantBeBlockedByDefenders(filter)])` prevents
creatures that match the filter from blocking the subject. Its subject is
`ThisPermanent` or the current `HostPermanent` of an Aura. It lowers to the
existing typed static `CantBeBlockedExceptBy(subject, Not(filter))` rule. This
form also uses exactly one operation. Here, `Defenders` means candidate blocking
creatures, regardless of whether they have defender.

Both forms reuse the existing permanent-filter grammar, including nested
`And`/`Or`, card type, creature subtype, color and fixed power comparisons.
`PowerIs(GreaterThanOrEqualTo(Integer(n)))`, `LessThanOrEqualTo` and `EqualTo`
read current power when block legality is checked. Filters do not create target
bindings or choices. All other combat restrictions still apply. Removing the
source ability stops the static rule. Removing a host's abilities does not stop
an Aura's rule. Phasing suspends the rule; attachment changes select the new
host, and a departed host is not followed through its return.

`TestRONBlockingProhibitionCards`, `CurrentCharacteristics`, `Lifecycle`,
`Host`, `NativeCombat`, `DeclaredBlock`, `Invalid`, `Inventory` and
`StrictCollection` (each with prefix `TestRONBlockingProhibition`) cover these
forms. The five complete-card mappings are Ironclaw Orcs, Brassclaw Orcs,
Orcish Veteran, Amrou Kithkin and Bog Rats. Other records with these operations
can still have unsupported rules. Temporary forms are unchanged.

## RON static attack permissions

Permanent rules accept these exact forms:

```ron
PermanentRuleEffect(HostPermanent, [CanAttackAsThoughItDidntHaveDefender])
PermanentRuleEffect(HostPermanent, [CanAttackAsThoughItHadHaste])
```

Each operation is a symbol in a one-element list. The subject is symbol
`ThisPermanent` or, only on an Aura, symbol `HostPermanent`. Target/event reads,
operation arguments, mixed lists and temporary or collection contexts remain
rejected. Both forms use typed current references and fresh static abilities per
factory, through the shared `While` service. The first reuses `AttrCanAttack`;
the second uses `mage.AttackAsThoughHaste` and preserves tap-cost restrictions.
They do not grant haste or remove defender from the host's characteristics.

Animate Wall and Instill Energy now compile in all six reprints each. Instill
Energy's separate zero-mana untap activation uses the existing controller-turn
and once-per-turn checks. No Go definition or production registry is replaced.

### RON temporary combat rules

`CreatePermanentRuleEffectUntil(subject, [operation], UntilEndOfTurn)` accepts
exactly one of these additional operations:

- Symbol `CanAttackAsThoughItDidntHaveDefender`.
- Symbol `CanAttackAsThoughItHadHaste`.
- Tuple `CantBeBlockedByDefenders(permanent_filter)`.

These forms accept symbol `ThisPermanent` on a permanent or a declared
`Ref_TargetPermanent`. They work in resolving activation, spell, mode and trigger
bodies. Aura hosts, event references, multiple operations and other durations
remain unsupported. The existing symbol `CantBeBlocked` keeps its target-only
boundary.

RON builds each typed target and effect in the same factory instance. It uses
`Until(EndOfTurn, ...)` with `GrantKeywordModification(ref, AttrCanAttack)`,
`AttackAsThoughHaste(ref)`, or `CantBeBlockedExceptBy(ref, Not(filter))`.
The engine captures the subject incarnation at resolution. A blocking filter
reads current blocker characteristics at declaration. Each attack permission
leaves other attack restrictions in force. Attack-only haste does not permit
summoning-sick tap costs or grant the haste ability.

Tower of Coireall uses the filtered rule with its creature target and tap cost.
Wall of Wonder uses the source defender permission after its temporary +4/-4.
`TestRONTemporaryCombatRule*` supplies real-record gameplay, native combat,
invalid-declaration, payment, target-copy, lifecycle and context evidence.

### RON static prevention from permanent sources

The top-level permanent rule supports this complete shape:

```ron
PreventDamage(
    DamageWouldBeDealtByAPermanentToRecipient(
        source_filter,
        Permanent(ThisPermanent)
    ),
    [PreventThatDamage]
)
```

`source_filter` uses the existing permanent filter grammar. The recipient is
only the source of the static ability. The operation list must contain exactly
the symbol `PreventThatDamage`. Player recipients, Aura hosts, event or target
references, general damage sources, combat-only events, finite amounts and
additional operations remain unsupported in this context. These declarations
fail compilation before a factory is returned.

Each factory builds a fresh static ability using `PreventDamageFromTo` and
`IsID`. It prevents all matching damage to that permanent while the ability
is active. It covers combat and noncombat damage without a finite shield.
Current source characteristics and retained source incarnations use the engine
source filter contract. This does not make permanent filters match spell sources.
The rule follows the recipient across control changes and phase cycles; a
returned recipient gets its own new static ability. Removing that ability
removes the prevention. Other instances and cloned games keep separate state.

Tests: `TestRONStaticPreventionCards`, `TestRONStaticPreventionLifecycle`,
`TestRONStaticPreventionPendingActivation`, `TestRONStaticPreventionCombatAndSpell`,
`TestRONStaticPreventionReplacementOrder`, `TestRONStaticPreventionOtherRules`,
`TestRONStaticPreventionNativeLandwalk`, `TestRONStaticPreventionStrictLoadAndInstances`
and `TestRONStaticPreventionInvalid`. These execute the complete Argothian Pixies,
Desert Nomads and Uncle Istvan records, including their other rules.

## RON mixed-recipient damage

In resolving action lists, the existing `SpellDealsDamage(ThisSpell, amount,
recipient)` and `PermanentDealsDamage(ThisPermanent, amount, recipient)` now
accept this recipient shape:

```ron
MultipleRecipients([
    EachPermanent(And([IsCardtype(Creature), DoesntHaveAbility(Flying)])),
    EachPlayer(AnyPlayer)
])
```

Supply exactly one `EachPermanent/1` and one `EachPlayer/1`, in either order.
The permanent operand uses the existing context-free permanent filter grammar;
the player operand must be symbol `AnyPlayer`. Duplicate sets, singular members,
empty lists, player-history filters, and unbound references are rejected during
compilation. This does not add multiple damage amounts or target groups.

The adapter builds `mage.DamageRecipients(mage.EachPlayer,
mage.MatchingPermanents(filter))` and passes it to
`mage.DealDamageEach(mage.This, recipients, amount)`. Each card factory builds
fresh effects. Existing amount forms, action contexts, and source restrictions
remain in force. The engine selects recipients at resolution and deals the
same amount to all of them in one simultaneous batch. No target declaration is
created. Earthquake, Hurricane, and Inferno use this complete shape.

Evidence: `TestRONMixedDamage*` in `pkg/mage/dsl/ron/mixed_damage_test.go`.

## RON simultaneous damage packets

Resolving action lists accept these forms:

```ron
SpellDealsMultipleDamage(ThisSpell, [
    DamageToRecipients(Integer(4), Ref_AnyTarget),
    DamageToRecipients(Integer(2), Player(You))
])
PermanentDealsMultipleDamage(ThisPermanent, [
    DamageToRecipients(Integer(2), Ref_AnyTarget),
    DamageToRecipients(Integer(3), Permanent(ThisPermanent))
])
```

The action has arity two. Its second operand is a nonempty list of
`DamageToRecipients(amount, recipient)/2`. The source must match the enclosing
spell or permanent context. Each amount uses the existing resolution-value
compiler. A recipient is the in-scope symbol `Ref_AnyTarget`, `Player(ref)` with
an existing typed player reference, or `Permanent(ref)` with an existing typed
permanent reference. Collections, numbered target groups and undeclared reads
remain errors. This adds no target declaration or target-choice service.
Spells, supported activations, supported triggers and supported modal action
bodies use the same lowering and retain their existing declaration scopes.

The compiler builds fresh `mage.DamagePacketTo(recipient, amount)` values for
`mage.DealDamagePackets(mage.This, packets...)`. The engine reads all recipients
and positive amounts before damage. Overlapping recipient identities receive
one combined amount before replacement processing. All assignments use one
captured source and one existing simultaneous batch. Nontargeted self damage
still requires the enclosing targeted spell or ability to resolve. The source
as a damage recipient must remain its original, unphased incarnation; as a
damage source it can use retained LKI after departure.

Orcish Artillery, Psionic Entity, Brothers of Fire and Psionic Blast use this
complete shape. Evidence: `TestRONDamagePackets*` and `TestDamagePackets*`.
The general `DamageToRecipients` family remains partial: recipient sets,
opponent-chosen targets, history target filters and rounded/counting values
still require their own adapters and contracts.

## RON draw-step intervening-if triggers

Permanent cards support this exact rule shape:

```ron
TriggerI(AtTheBeginningOfAPlayersDrawStep(players),
    PermanentPassesFilter(ThisPermanent, state), actions)
```

`players` is symbol `AnyPlayer` or `SinglePlayer(You)`. `state` is symbol
`IsTapped` or `IsUntapped`. The body uses the existing action-list compiler and
may read `Trigger_ThatPlayer` as a typed player reference. Each factory creates
its own binding; independent events, copies and cloned games keep their captured
player. `You` in the body uses the resolving ability controller. The controller
filter uses the source controller at detection. Other event players, conditions,
subjects and spell contexts remain declaration errors.

The adapter lowers to `TurnStepBegins(EachDrawStep, binding)` and
`WithInterveningIf(SourceTappedStateWithLKI(state == IsTapped))`. The engine
checks the condition before the normal draw and again at resolution. A departed
source uses the last-known tapped state of its captured incarnation. A returned
card cannot replace that information. A source that phases out after detection
keeps its tapped state for this information read. Phased sources do not trigger.
Normal draws and trigger draws use the shared replacement pipeline.

The complete-card mappings are Howling Mine and Mana Vault, each in all six
reprints. `TestRONDrawStepCards`, `ConditionsAndIdentity`, `VaultPayment`,
`Bindings`, `Invalid`, `InventoryAndReprints`, `BeforeNormalDrawReplacement` and
`VaultDamagePrevention` (all with prefix `TestRONDrawStep`) cover their rules
and interactions. The vocabulary family remains partial: this does not add
other intervening-if events, conditions, or ordinary `TriggerA` draw-step rules.

`mage.SourceTappedStateWithLKI(bool) StateCondition` is a source-only condition
for information reads at resolution. At detection, with no captured incarnation,
it requires a live phased-in source. With a captured incarnation, it reads that
object's current tapped state or its departure snapshot. Existing
`PermanentIsTapped`, `PermanentIsUntapped`, `SourceTappedCondition` and
`SourceUntappedCondition` keep their live-only semantics.

## RON permanent information and singular exile

In supported resolving action bodies, `ControllerOfPermanent(ref)` is a typed
player reference. `PowerOfPermanent(ref)` and `ManaValueOfPermanent(ref)` are
resolution values. They read the exact incarnation's current information, or
its saved departure/phase information. `ref` can be the local permanent target,
a captured `Trigger_ThatPermanent`, `ThisPermanent` on a permanent, or
`HostPermanent` on an Aura. An unbound or wrong-role reference fails compilation.
Each factory builds reads against its own declarations; modes and abilities
cannot share a target scope.

`PlayerAction(ControllerOfPermanent(ref), GainLife(value))` and its `LoseLife`
form use the indicated player. Other player contexts retain their existing
support limits. `Exile([Permanent(ref)], [])` supports one permanent with no
modifiers in spell, modal, activation and ordinary trigger bodies. It performs
no action on a departed or phased incarnation. Larger lists, card subjects,
collections, face-down modifiers and linked results remain unsupported.
See [the rules contract](engine-rules-contracts.md#permanent-information-through-typed-references)
for engine APIs, lifecycle behavior, and test evidence.

## RON fixed entry counters

`AsPermanentEnters(ThisPermanent, [EntersWithNumberCounters(Integer(n), counter)])`
accepts one entry operation on a permanent card. The source is the symbol
`ThisPermanent`. The amount is a nonnegative integer literal that fits the
engine's current counter storage (0–255); larger declarations are rejected.
`counter` uses the same validated permanent counter types as `PutCounters`,
including `PTCounter(+1, +0)`, `PTCounter(+1, +1)` and named counters.
The adapter uses `WithAbility(EntersWithNCounters(counter, n))`, with a fresh
ability for each card factory invocation. Counters exist before entry triggers
are detected. Entry is not a stack action; the permanent spell still resolves
through the stack. X, computed entry values, other source references and combined
entry-operation lists remain unsupported. Existing `EntersTapped` support remains.

Triskelion's complete ATQ record is exercised by `TestRONEntryCountersTriskelion`,
`Lifecycle`, `Replacements` and `TargetRecheckAndPrevention` in the RON package.
`TestRONEntryCountersCompleteRecordInventory` exercises only entry clauses from
all 13 records of eight fixed-counter names; it does not establish their remaining
rules. See the [rules contract](engine-rules-contracts.md#ron-fixed-entry-counters)
and [progress ledger](designs/mtgish-ron-progress.md).

## RON player-target life actions

`PlayerAction(Ref_TargetPlayer, GainLife(value))` and
`PlayerAction(Ref_TargetPlayer, LoseLife(value))` use the declared singular
`TargetPlayer(AnyPlayer)` binding. `value` uses the existing resolution-value
compiler, including `Integer(n)` and announced `ValueX`. The target can belong
to a spell, activation, trigger, or selected spell mode. Each factory creates
its own target declaration. `PlayerAction(You, ...)` and bare life actions
still use the action controller, even inside a player-target action list.
Sibling actions do not inherit an explicit player's scope.

Life actions now also honor existing non-event spell-controller references.
Step players and permanent-controller LKI reads retain their established
behavior. Spell-cast event players remain rejected in this slice. Unbound,
incompatible and mode-external references remain declaration errors.
The engine executes life changes through `GainLifeFor` and `LoseLifeFor`,
with its normal replacement and notification services. Life loss does not
use damage prevention. `TestRONPlayerLife*` covers these contracts.

Stream of Life has complete RON gameplay evidence for all six reprints.
Healing Salve and Alabaster Potion have life-clause evidence only; their
complete records remain rejected because finite prevention is unsupported.

## RON opponent targets and spell-cast events

The symbol `Opponent` is supported in `TargetPlayer(Opponent)`,
`TargetPermanent(ControlledByAPlayer(Opponent))`, and
`WhenAPlayerCastsASpell(Opponent, spellFilter)`. Permanent target filters can
combine the controller restriction with other supported filters in nested
`And` lists. These target forms work in spells, activations, triggers, and
selected spell modes. Each factory creates separate typed target declarations.

Targets use the spell or ability controller at announcement and resolution.
An opponent-controlled copy that keeps its controller as its target cannot
resolve. Source control changes or departure do not change a pending ability's
controller. Spell-cast events use `SpellCast().ByOpponent()` to check the source
controller at the event. The existing spell and caster bindings stay separate
for each pending trigger. Tuple-shaped `Opponent()`, player references used as
filters, and controller-dependent target disjunctions remain rejected.
This does not add general opponent player-reference or collection support.
Evidence: `TestRONOpponent*`, including complete Citanul Druid gameplay.

## RON optional tap or untap

`MayAction(TapOrUntapPermanent(ref))` accepts a typed permanent reference in
its current action scope: `ThisPermanent`, Aura `HostPermanent`, declared
`Ref_TargetPermanent`, or bound `Trigger_ThatPermanent`. The nested action must
have exactly one reference. Other `MayAction` shapes and a bare
`TapOrUntapPermanent` remain unsupported.

The compiler lowers this pair to `mage.MayTapOrUntap` with fresh instance
bindings. It works in spell actions, selected spell modes, activations, and
trigger actions. The resolving controller chooses whether to change the
permanent's current tapped state. Target choice and costs occur before this
choice. Decline does not refund costs or stop later instructions. An absent or
illegal subject receives no choice. Normal tap/untap events and replacements
still apply. See `TestRONTapChoice*` for complete Twiddle and Hyperion Blacksmith
RON gameplay and shared scope, copy, clone, event, and rejection evidence.

## RON resolution-time battlefield counts

`TheNumberOfPermanentsOnTheBattlefield(filter)` supplies a resolution value.
The filter accepts the existing source-independent permanent filters. Nested
`And([...])` can include one `ControlledByAPlayer(SinglePlayer(player))` clause.
The player uses the current typed reference scope, including `You`, a declared
`Ref_TargetPlayer`, or an event-bound `Trigger_ThatPlayer`. Controller clauses
inside `Or`, multiple controller clauses, collection players, unbound references,
and damage-loop players remain rejected.

`Twice(value)` multiplies any supported instance resolution value by two.
It composes with counts and subtraction and retains signed intermediate values.
These forms work in the resolving action consumers of `compileInstanceValue`;
they do not add static, CDA, entry, cost, beginning-of-turn, or temporary P/T
value support. Each instruction reads current collection membership. Phased
permanents are absent; tokens and returned permanents can be current members.
Ownership does not determine control.

The adapter uses `CountPermanents`, `MatchingPermanents`,
`PermanentsControlledBy`, and `MultiplyValues`. Typed references keep factory,
copy, event, and target bindings separate. Full-card evidence is Karma
(LEA/LEB/2ED/3ED/CED/CEI) and Jovial Evil (LEG), in `TestRONBattlefieldCount*`.

### RON land mana events

RON supports `TriggerA(WhenAPlayerTapsAPermanentForMana(AnyPlayer,
IsCardtype(Land)), body)` for non-mana resolving actions. Each factory has its
own typed event bindings. `Trigger_ThatPlayer` saves the activating player;
`Trigger_ThatPermanent` saves the land incarnation. Information reads use the
existing LKI and phase-information contracts. These names do not escape the
trigger body.

Two complete mana-bonus bodies use the shared mana-bonus service so automatic
payment and actual production agree:

```ron
TriggerA(WhenAPlayerTapsAPermanentForMana(AnyPlayer, IsCardtype(Land)),
  ActionList([PlayerAction(Trigger_ThatPlayer,
    AddMana(Trigger_AnyManaTypeProduced))]))
TriggerA(WhenAPermanentIsTappedForMana(SinglePermanent(HostPermanent)),
  ActionList([PlayerAction(ControllerOfPermanent(Trigger_ThatPermanent),
    AddMana(ManaProduceG))]))
```

The attached form needs an Aura scope and one concrete production. White, blue,
black, red, green and colorless productions work; `And` with exactly one concrete
production also works. The bonus goes to the player who taps the host for mana.
These bonuses add one mana, resolve immediately, and do not create stack objects.
The produced-type form permits only a type produced by the triggering ability;
it does not copy the amount produced or the types of other bonuses.

Other mana-producing bodies in these events remain declaration errors. Subtype
events, other player filters, temporary triggers, source/target mana events and
attached non-mana bodies remain unsupported. This does not add general
`PlayerAction(..., AddMana(...))` support outside this event scope.

### RON enchanted permanent filter

The bare symbol `IsEnchanted` is a permanent filter. It uses the shared
`mage.IsEnchanted` query: a permanent must have an attached, present Aura.
Equipment alone does not qualify. The filter composes with the supported
`And` and `Or` forms and works in existing permanent target and static damage
source filters. Target legality is checked at announcement and resolution;
prevention reads source information when damage would occur. The query does
not choose or save an attachment. Constructor forms such as `IsEnchanted()`
and `IsEnchanted(ThisPermanent)` are invalid.

RON gameplay tests: `TestRONEnchantedFilter*` cover Ramses Overdark and Wall of
Putrid Flesh. Enchanted Being still requires an unsupported combat-only static
prevention event; this filter does not establish support for that event.

## RON land and artifact graveyard-event adapters

`TriggerA(WhenAPermanentIsPutIntoAPlayersGraveyard(filter, AnyPlayer), actions)`
accepts `IsCardtype(Land)` or `IsCardtype(Artifact)`. It also accepts
`And([IsCardtype(type), ControlledByAPlayer(SinglePlayer(You))])` in either
order, with exactly one type and one controller restriction. The type and
controller are checked before the permanent leaves the battlefield. A move
from another zone, phasing, regeneration, or an exile destination does not
match. Tokens and the source itself can match.

Inside this trigger, `Trigger_ThatPlayer` is the owner of the destination
graveyard. `ControllerOfDeadPermanent` is the permanent's controller before
it left. Both use separate `PlayerBinding` values created for each factory
instance. Each pending trigger saves its own values. A later return or control
change does not redirect them. These symbols are not available in other event
contexts. Singular source/host selectors, other types, restricted destination
players and `TriggerI` remain unsupported for this event.

The adapter uses `PermanentDies().OfType(type).WithOwner(owner)` and
`WithController(controller)`, with `YouControlled()` for the controller
restriction. Existing action lists, optional mana payments and typed targets
apply. No new engine API is required.

Dingus Egg's six records use `ControllerOfDeadPermanent`, in accord with the
local Oracle text in `data/LEA.json`. Tablet of Epityr uses the existing optional
payment action. Full-card evidence is in `TestRONGraveyardEvent*`, including
strict loading and native casting. This does not establish support for Lich,
Cyclopean Tomb or Urza's Miter.

### RON end-step intervening-if triggers

`TriggerI(AtTheBeginningOfAPlayersEndStep(players), condition, actions)` accepts
`AnyPlayer` or `SinglePlayer(You)`. The actions can read the captured
`Trigger_ThatPlayer`; that binding belongs to this trigger and this factory.
Supported conditions are:

- `NoPermanentsPassFilter(filter, OnTheBattlefield)`, with an existing permanent
  filter. The query includes both players and excludes phased permanents.
- `PermanentPassesFilter(ThisPermanent,
  DealtDamageToAnyPlayerThisTurn(Opponent))`. Damage history belongs to the
  source incarnation. Zero damage and prevented damage do not satisfy it.

The engine checks the condition at the event and at resolution. A false event
condition creates no trigger. A false resolution condition performs no actions.
The pending trigger keeps its controller and event player after a control
change. Source actions cannot affect a returned incarnation. Sacrifice still
requires the trigger controller to control the permanent. History and counters
remain separate between factories and cloned games. Other end-step conditions,
including tapped state, death history and attack history, remain unsupported.
Evidence: `TestRONEndStepIntervening*` covers the complete Pestilence and
Whirling Dervish records, reprints, native casting and engine gameplay.

### RON artifact event adapters

In `TriggerA(event, actions)`, the following event forms create a fresh typed
`Trigger_ThatPermanent` binding per rule and factory:

- `WhenAPermanentBecomesTapped(IsCardtype(Artifact))`.
- `WhenAPermanentBecomesTapped(And([IsCardtype(Artifact), ControlledByAPlayer(Opponent)]))`;
  the two conjunction members may occur in either order.
- `WhenAPlayerActivatesAnAbility(player, And([AbilityOfAPermanent(IsCardtype(Artifact)), DoesntHaveTapSelfInCost]))`,
  where `player` is symbol `AnyPlayer` or `Opponent`.
- The same activation form with `AbilityOfPermanent(HostPermanent)` in an Aura.

The two activation conjunction members may occur in either order. Arity,
source kind, non-tap symbol spelling, duplicates and lexical reads are checked
before factory construction. Other activation filters remain unsupported.
`ControllerOfPermanent(Trigger_ThatPermanent)` uses the saved incarnation's
current controller or departure LKI at resolution. `Trigger_ThatPlayer` is not
introduced by these forms. Haunting Wind, Powerleech and Artifact Possession
use these adapters. The attached tap adapter and land tap adapters remain
available with their existing contracts.


### RON random discard costs

`DiscardACardAtRandom` is a bare cost symbol in `Activated/2` and
`ActivatedWithModifiers/3`, including nested cost `And([costs])` lists.
It creates `mage.DiscardRandomCost(1)` for each factory instance. Tuple forms,
random-number discard costs and optional random payments remain rejected.
The payer is the activation controller; declaring a target does not change it.

The engine locks a random hand selection after deterministic payments. It uses
hand-visit identity and the game's random service, with no hand-choice prompt.
Shared discard replacements and simultaneous batch events apply to the payment.
An unpaid activation changes no live resources or scripted random results.
Copies resolve their effects without paying the original cost again.

Complete-record mappings: Coral Helm and Draconian Cylix in
`TestRONRandomDiscardCosts` and `TestRONRandomDiscardCostNativeLoadAndInstances`.
The latter loads a strict native fixture collection; production cutover remains
pending complete collection support.

### RON land dependence

`PermanentRuleEffect(ThisPermanent, [CantAttackUnlessDefendingPlayer(
PlayerPassesFilter(DefendingPlayer, ControlsA(IsLandType(Island))))])` uses
`Static(CanAttackOnlyIfDefendingPlayerControls(filter))`. In this two-player
engine, the nonactive player is the defender. The rule checks current subtype
and control for attack legality; it does not remove a declared attacker.

`TriggerA(WhenAPlayerControlsNoPermanents(SinglePlayer(You),
IsLandType(Island)), actions)` uses `WhenState(ControlsNone(filter), actions)`.
The condition uses the current source controller. Once triggered, the ability
resolves even if that player acquires a matching permanent. This is a state
trigger, with no intervening-if check and no event-player binding.

These forms accept `IsLandType` with Plains, Island, Swamp, Mountain, Forest or
Desert. The attack rule accepts only the source, one operation, `DefendingPlayer`
and `ControlsA`. The state event accepts only `SinglePlayer(You)`. Actions and
targets use the existing action compiler and fresh factory declarations.
Other filters, chosen players, Aura hosts and inverse attack conditions remain
unsupported in these contexts. The entire collection still fails strict loading.

## RON source attack triggers

`TriggerA(WhenACreatureAttacks(SinglePermanent(ThisPermanent)), actions)`
uses `mage.Attacks(mage.This)`. Replace the event with
`WhenACreatureAttacksAndIsntBlocked` to use `mage.AttacksUnblocked(mage.This)`.
Only the source permanent is supported in these two event contexts.
`Trigger_ThatCreature` is a typed source reference inside the action body. The
engine captures the source incarnation for each pending ability. The reference
is not available in another rule, and it does not select a returned permanent.
`PermanentDealsDamage` accepts this reference as its source in this context.

Temporary layer modifications in these action bodies accept the bare duration
`UntilEndOfCombat` as well as the existing `UntilEndOfTurn` and source-tapped
forms. Combat expiry uses `mage.Until(core.EndOfCombat, modifications...)`.
Other temporary-modification contexts retain their existing duration support.
Host events, event defending-player references, delayed events and non-mana
optional payments need separate adapters. These forms remain rejected.

### RON one-shot delayed step actions

`CreateFutureTrigger(event, ActionList([...]))` accepts bare
`AtTheBeginningOfTheNextEndStep`, bare `AtTheNextEndOfCombat`, and
`AtTheBeginningOfPlayersNextUpkeep(You)`. It lowers to `mage.AtNext` with
`EachEndStep`, `EachEndOfCombat`, or `YourUpkeep`. The nonempty body uses the
existing action compiler and may schedule another supported delayed action.

The body can read existing typed target declarations, source references, saved
number/permanent/player bindings, and announced X. Each scheduling resolution
captures its own frame. The delayed body declares no new targets. A captured
permanent reference keeps its original incarnation and does not repeat the
original target filter. Source departure and ability removal do not cancel a
schedule. Actions still require a present, phased-in subject; information reads
use their existing LKI contracts.

Other events, new target declarations or modes in the delayed body, and event
spell reads remain compiler errors. The engine does not yet save event spell
bindings in this frame. Next upkeep accepts only bare `You`; other players need
an explicit scheduling contract. `AtNextEndOfCombatThisTurn` is distinct from
`AtTheNextEndOfCombat` and remains unsupported. This adapter does not add casting
restrictions, combat events, choices, histories, links, or finite prevention.

### RON combat-pair triggers

`TriggerA` accepts `WhenACreatureBlocksACreature(subject, filter)` and
`WhenACreatureBecomesBlockedByACreature(subject, filter)` with
`SinglePermanent(ThisPermanent)` or, in an Aura, `SinglePermanent(HostPermanent)`.
The filter is `IsCardtype(Creature)` or a one-argument `And` list containing it
exactly once, optionally with one `IsNonCreatureType(subtype)` and one color
condition. The color condition is `IsColor(color)` or a nonempty
`Or([IsColor(color), ...])`. Each subtype must pass creature-family validation.
Other type, history, numeric, controller and nested conjunction forms remain
unsupported. Duplicate top-level conditions are errors.

An event union is exactly `Or([blockingEvent, blockedByEvent])`, in either order.
Both branches must have the same position-free subject and filter declarations.
It lowers to the engine's per-opponent combat-pair event, not two abilities or a
once-per-combat group. Arbitrary unions and unequal branch declarations remain
errors. Inventory walks nested events even in unsupported unions.

The blocking event provides `Trigger_ThatOtherCreature`; the blocked-by event
provides `Trigger_TheBlockingCreature`; the union provides only
`Trigger_ThatOtherCreature`. Each is a typed event binding available only in its
body, including supported delayed step actions. Each factory builds a fresh
binding. The matching engine patterns save the opposing incarnation at detection.
Aura events read the current attachment at detection. Later attachment, color,
subtype or controller changes do not recheck filters or redirect the saved object.
Departure of the source does not end pending or delayed abilities. A returned
opponent is a new object; a phased-out opponent cannot be destroyed.

The compiler uses `BlocksCreatureMatching`, `BecomesBlockedByCreatureMatching`,
and `BlocksOrBecomesBlockedByCreatureMatching` with a typed source/host reference,
immutable `PermanentFilter`, and `WithOpponent(binding)`. Previous source-only
engine patterns remain available. `WhenACreatureBlocks`, group events and other
event-reference spellings still require separate adapters.

### RON indefinite type changes

`CreatePermanentLayerEffect(reference, [AddCardtype(type)])` accepts Creature,
Artifact, Enchantment or Land. `CreatePermanentLayerEffect(reference,
[SetLandType(type)])` accepts Plains, Island, Swamp, Mountain or Forest.
Each form requires exactly one operation and a bound singular permanent
reference in a resolving action body. Empty or combined operations, unsupported
card types, nonbasic land types and malformed or unbound references are errors.
Other indefinite modification families remain unsupported.

The compiler builds each subject with the action instance's bindings and uses
`Until(Indefinite, AddCardType(...))` or
`Until(Indefinite, SetSubtypeFamily(..., SubtypeLand, ...))`. Resolution captures
one incarnation. The effect survives cleanup and source departure. Phasing
preserves an installed effect; departure and return do not preserve it. An
absent or illegal target at resolution receives no modification. Basic land
subtype replacement uses the engine's printed-ability removal and intrinsic
mana rules; it preserves other card types, subtype families and supertypes.

Complete RON gameplay evidence covers Ashnod's Transmogrant and Thelonite Monk
in `TestRONIndefiniteType*`. The latter can sacrifice itself as its green creature
payment. Counter replacement on Transmogrant does not stop its next instruction.

## RON attachment target filters

`TargetPermanent(AttachedToAPermanent(hostFilter))` selects an attachment by its
current battlefield host. Combine it with `IsEnchantmentType(Aura)` in `And`
to select Auras. This adapter works in spell, activation and trigger targets.
The host filter accepts existing pure permanent filters and nonempty nested
`And`/`Or` lists. A conjunctive `ControlledByAPlayer(SinglePlayer(You))` restricts
the host to the stack object's controller. It does not restrict the Aura's
controller or owner. The host controller restriction can occur once.

The compiler lowers the host query to `IsAttachedTo` and the controller rule to
`TargetAttachedToYouControl`. The target remains the attachment; the host is
read again at resolution. Reattachment to a qualifying host keeps the target
legal. Source departure does not change the activation controller. Copies use
their own controller. Each card factory builds a separate target declaration.

Opponent/event/target player references, controller rules inside a disjunction,
source exclusions and nested attachment queries are rejected in this host
context. This slice does not add collection attachment filters, movement,
activated modes or replacement operations. Miracle Worker and Savaen Elves have
complete RON gameplay tests in `attachment_target_test.go`.

### RON plural targets and temporary colors

The RON compiler accepts `OneOrMoreTargetPermanents(filter)` as the sole target
group of a spell, mode, activation, or trigger. Its typed collection reference
is `Ref_TargetPermanents`; the singular `Ref_TargetPermanent` cannot read this
group. Existing permanent target filters and options apply to each member.

`CreateEachPermanentLayerEffectUntil(Ref_TargetPermanents,
[SetColor(SimpleColorList([colors]))], UntilEndOfTurn)` replaces the colors of
legal targets at resolution. The list can contain White, Blue, Black, Red, and
Green. An empty list makes the targets colorless. Supply one modification.
The compiler rejects delayed-action use of this form until that context has
contract tests. Collection P/T effects retain their existing filter context.

The adapter uses `DeclarePermanentTargets`, `SetEachColors`, and `Until`.
Targets are chosen at announcement. The color effect captures each legal
incarnation at resolution and ends at cleanup. The compiler creates fresh target
bindings for every factory call.

### RON fixed and X permanent target collections

`NumberTargetPermanents(Integer(n), filter)` declares exactly `n` targets.
The integer must be nonnegative. `NumberTargetPermanents(ValueX, filter)`
declares exactly the announced X targets. Both use the sole target group of
`Targeted` and bind the typed collection `Ref_TargetPermanents`. Spell, mode,
activation and trigger bodies share this declaration adapter. X payment remains
an independent cost contract; this adapter does not add activation-X costs.

`TapEachPermanent(Ref_TargetPermanents)` taps the legal group in one engine
operation. `Exile([EachPermanent(Ref_TargetPermanents)], [])` exiles the legal
group in one zone-change batch. The exile form requires one subject and no
modifiers. These actions also accept existing one-or-more target declarations.
They reject singular, unbound and delayed-context collection reads. Existing
filter-based tap and singular exile forms keep their contracts.

The adapters use `DeclarePermanentTargets`, `DeclareXPermanentTargets`,
`TapEach` and `ExileEach`. Zero targets permit resolution; if a nonempty group
has no legal targets, the spell does not resolve. Factory calls and spell copies
keep separate chosen targets and announced values.

## RON conditional fixed mana

The supported context is an unmodified `Activated` rule with the sole cost
`TapPermanent(ThisPermanent)` and the sole action:

```ron
IfElse(
    PlayerPassesFilter(You, And([
        ControlsA(And([IsLandType(Urzas), IsLandType(Mine)])),
        ControlsA(And([IsLandType(Urzas), IsLandType(PowerPlant)])),
    ])),
    [AddMana(And([ManaProduceC, ManaProduceC, ManaProduceC]))],
    [AddMana(ManaProduceC)],
)
```

`ControlsA(filter)` may stand alone or occur in nonempty, nested `And` conditions.
Its permanent filter accepts nonempty `And`/`Or` combinations of `AnyPermanent`,
card types, positive/negative land or creature subtypes, Aura subtype,
positive/negative colors, and positive/negative supertypes. Subtype validation
uses the engine family contract, which permits custom unknown subtypes and
rejects known family mismatches. Tap and combat status filters are rejected
because solver queries precede the tap cost.

Each branch has exactly one `AddMana`, with a concrete mana symbol or a nonempty
`And` of concrete symbols. Repeated colors and multiple fixed colors are valid.
Empty branches, additional actions or costs, modifiers, player choices, dynamic
amounts, bound references and other condition forms remain unsupported.
Factories create fresh `ManaAbility` instances and use the shared engine
`NewConditionalManaAbility` service. General `IfElse` action support is still a
gap. Tests: `TestRONConditionalMana*`; engine contract: `TestConditionalMana*`.

## RON source and attachment filter conditions

`PermanentPassesFilter(subject, filter)` in a static `If` accepts bare
`ThisPermanent`, or bare `HostPermanent` on an Aura. It lowers to
`mage.PermanentMatchesFilter(PermanentRef, PermanentFilter) StateCondition`.
The existing `IsUntapped` form retains `PermanentIsUntapped`.

The filter uses the existing static condition grammar: nonempty nested
`And([filters])`/`Or([filters])`, `IsNonToken`, permanent type/subtype,
color/supertype predicates, tapped state, and combat membership. Operands must
have the existing exact shapes. Ability and P/T filters remain rejected so a
conditional grant or bonus cannot read the characteristics it changes. Target,
event, saved and unresolved references remain rejected. No resolving `If` action
or new conditional body shape is added.

The engine checks the live source incarnation and, for attachments, the live
attachment incarnation before applying the filter. It uses the shared battlefield
query service and stores no choices or captured game state. Conditions read the
current attachment and characteristics on each layer pass. Departure or phasing
makes the condition false; a phase cycle restores it if the same objects still
qualify. Control changes alone do not end the condition. No LKI is read.

`TestPermanentMatchesFilter*` covers the engine contract.
`TestRONPermanentFilter*` covers complete Goblin Caves gameplay, Aura legality
and costs, attachment/type changes, source and recipient membership, phasing,
ability removal, independent factory instances and clone isolation. Goblin
Shrine shares the condition but remains rejected for its leave-battlefield
trigger; its complete gameplay is not verified.

## RON unlimited recipient prevention

Resolving actions accept these two complete forms:

```ron
CreatePreventDamageUntil(
    DamageWouldBeDealtToRecipient(Permanent(Ref_TargetPermanent)),
    [PreventThatDamage], UntilEndOfTurn)
CreatePreventDamageUntil(
    DamageWouldBeDealtByAPermanentToRecipient(
        And([IsCardtype(Creature), HasAbility(Flying)]), Player(You)),
    [PreventThatDamage], UntilEndOfTurn)
```

Recipients reuse typed damage-packet references: `Permanent(reference)`,
`Player(reference)`, or a bound `Ref_AnyTarget`. Existing source, Aura attachment,
event, saved, and singular target scopes apply. Unbound or incompatible reads
are compile errors. Each factory builds references with its own bindings.
These actions work in the existing resolving spell, mode, activation, trigger,
and delayed-action contexts; they add no static or replacement-operation scope.
The operation list must contain exactly one bare `PreventThatDamage`, and the
only supported duration is bare `UntilEndOfTurn`.

Source filters reuse supported type, subtype, supertype, color, keyword, token,
and tapped-state filters with nonempty nested `And`/`Or`. Bare `IsAttacking`
uses `DamageSourceIsAttacking()` so absent sources use saved combat information.
`IsBlocking` and `PowerIs` remain rejected in this context. A zero engine filter
means any source; RON uses the first form to express that choice. The second form
requires a permanent source and does not match a spell source.

The engine API is `PreventDamageThisTurn(DamageRecipientRef, PermanentFilter)`.
It captures the recipient at resolution, preserves permanent incarnation,
rechecks source properties at damage time, and expires at cleanup. It prevents
combat and noncombat damage without consuming a finite shield. It does not
prevent damage dealt by the recipient to other objects or players.

Complete gameplay evidence covers Indestructible Aura, Scarecrow, and
Al-abara's Carpet. The latter two records include their Oracle source
restrictions: flying creatures, and attacking creatures without flying,
respectively. Other finite, chosen-source, redirection, and additional-operation
prevention declarations remain unsupported.

### RON all-player hand and fixed draw actions

Resolving action lists accept `EachPlayerAction(AnyPlayer, DiscardHand)` and
`EachPlayerAction(AnyPlayer, DrawNumberCards(Integer(n)))`, where `n` is a
nonnegative integer. `AnyPlayer` and `DiscardHand` are bare symbols. These forms
use `DiscardEach(EachPlayer)` and `DrawEach(EachPlayer, Fixed(n))`.

Discard selects both current hands at resolution and uses one simultaneous
batch. All discard moves finish before discard events. Draw uses individual
draw actions; the active player completes all draws before the other player.
Both operations use the shared replacement pipeline. Empty hands and zero
draws permit later instructions. Existing spell, mode, activation, trigger and
delayed action lists use the same lowering. The forms create no targets or
loop-local player binding and preserve surrounding target scopes.

Other player filters, action-list bodies, shuffle actions, optional discards,
X and player-dependent draw amounts remain compile errors in this context.
Complete Wheel of Fortune gameplay evidence is in `TestRONEachPlayerWheel`,
`TestRONEachPlayerCopies` and `TestRONEachPlayerReplacements`.

### RON current-hand operations

`PlayerAction(player, RevealHand)` reveals that player's current hand at
resolution. `RevealHand` must be a bare symbol.
`PlayerAction(player, DiscardCards(filter))` discards all matching cards in one
batch. `TheNumberOfCardsOfTypeInPlayersHand(filter, player)` is a current-hand
resolution value. Player references use the existing typed lexical scope;
these forms create no new binding or target declaration.

Hand filters accept `IsCardtype(type)`, `IsNonCardtype(type)`, `IsColor(color)`,
`IsNonColor(color)` and nonempty nested `And([...])` or `Or([...])` forms. They
read card characteristics, not permanent characteristics. Unbound players,
malformed declarations, battlefield predicates and empty logical filters are
compile errors. The adapters apply in resolving spell, mode, activation,
trigger and delayed action scopes; they do not add cost, entry, static or
replacement-operation forms. The count uses existing resolution-value contexts.

The engine primitives are `RevealHandFor`, `DiscardMatchingHand` and
`CountMatchingHand`. Complete card evidence is in `TestRONHandOperationsCards`,
`TestRONHandOperationsCopiesAndPrevention` and
`TestRONHandOperationsEmptyAndReplacements` for Amnesia and Inquisition.

## RON damage-amount triggers

`TriggerA(WhenAPermanentDealsDamage(SinglePermanent(ThisPermanent)), body)`
uses `DealsDamage(This, amount)`. An Aura can use `HostPermanent` in the same
shape, which uses `DealsDamage(AttachedPermanent, amount)`.
`Trigger_AmountOfDamageDealt` is a bare value symbol in that trigger's resolving
body. Each factory call creates a number binding. The value is the actual damage
after prevention, for combat or noncombat damage to a player or permanent.
Zero and fully prevented damage do not trigger. Each event keeps its own amount.
The body has no damaged-player or damaged-permanent binding in this shape.
Other source collections remain unsupported. Opponent-damage events use the
separate shape below.

The engine captures the event before stack placement. Later source departure,
phasing, control changes, or attachment changes do not change the saved amount.
The trigger keeps its controller; a copied trigger uses its copy controller for
`You`. `TestRONDamageAmountCards`, `BatchAndInstances`, `Combat`, `Casting`,
`AuraTargetLegality`, `Invalid`, and `Inventory` provide the RON evidence.


## RON opponent-damage player triggers

`TriggerA(WhenAPermanentDealsDamageToAPlayer(SinglePermanent(ThisPermanent),
Opponent), body)` uses `DealsDamageToOpponent(This, player)`. Each factory
creates a separate player binding. Bare `Trigger_ThatPlayer` reads the damaged
opponent in the event body's resolving scopes. It is not a target. The event
checks the source controller when actual combat or noncombat damage occurs.
Zero or fully prevented damage does not trigger. Later source control changes,
departure, return or phasing cannot redirect the captured player. Pending
triggers, copies and clones use the existing resolution frame services.
This shape supplies no amount or permanent binding. Other source references
and player filters remain unsupported.

`PlayerAction(player, DiscardHand)` requires a bare action symbol and a bound
player reference. It uses `DiscardMatchingHand(player, CardFilter{})` to select
all cards in that player's current hand at resolution. Replacements prepare all
moves before discard events. This adds no bare action, EachPlayerAction, cost,
entry, static or replacement-operation context.

`TestRONDamagePlayerCards`, `Casting`, `Combat`, `Instances`,
`DiscardReplacement`, `HandBatch`, `Invalid` and `Inventory` cover complete
Hypnotic Specter and Nicol Bolas records. `TestRONUpkeepPaymentRealCards`
also covers Nicol Bolas's complete upkeep payment and sacrifice rule.

## RON player poison counters and token damage triggers

`PutCounters([ACounterOfTypeOnPlayer(PoisonCounter, player)])` places one counter.
`NumberCountersOfTypeOnPlayer(Integer(n), PoisonCounter, player)` places a fixed,
nonnegative amount, including zero. The list must contain one instruction.
These resolving actions accept existing typed player references, including
`You`, declared player targets and event-local `Trigger_ThatPlayer`. Other player
counter types, dynamic amounts, costs and entry operations remain unsupported.
They use `mage.AddPoisonFor(subject PlayerRef, amount ResolutionValue)`; the
engine resolves the player and amount in the current frame and adds positive
amounts through the existing player poison state. Nil or unbound declarations
fail definition validation. Absent references and nonpositive amounts do nothing.
Poison placement is separate from damage and life loss.

`TriggerA(WhenAPermanentDealsDamageToAPlayer(SinglePermanent(ThisPermanent),
AnyPlayer), body)` extends the existing source/opponent form. It uses
`DealsDamageToPlayer(This, binding)`, captures the actual damaged player, and
permits damage to the source controller. It supplies no amount or permanent
binding. Other source and player filters remain unsupported.

The ability list of `TokenWithPT/6` and `NamedTokenWithPT/7` can contain this
source damage-player `TriggerA/2` with bare `AnyPlayer` or `Opponent`, alongside
existing keywords. The token trigger body uses existing resolving action forms
in a fresh permanent scope. It cannot read targets, X, payment results or event
bindings from the creating ability. Each card factory builds a fresh immutable
token declaration with fresh trigger bindings; `TokenAbilities` gives created
tokens separate runtime ability identities. Token creation and copying of the
creating stack ability retain their existing controller and batch contracts.

Evidence: `TestAddPoisonFor*`, `TestRONPoisonCards`, `CastingAndCombat`,
`GeneratorPaymentAndCopies`, `FactoryInstances`, `TokenScope`, `PlacementAndLoss`,
`Invalid` and `Inventory` in their poison test files. The complete RON card
mappings are Marsh Viper, Pit Scorpion and Serpent Generator.
