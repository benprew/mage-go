# DSL reference

This checkout has the legacy facade in `pkg/mage/dsl/dsl.go`. It does not have
`pkg/mage/dsl/composable.go`. Proposed signatures in the migration plan are
not implemented APIs.

## Static spell targeting restrictions

`CantBeTargetedBySpells(PermanentRef, CardFilter) StaticRule` supplies a live
restriction on matching spells. Use it with `Static`. Supply `This` or
`AttachedPermanent` for source and Aura-host rules. Definition validation rejects
nil or unbound references and a zero filter. Spell filters use the spell's current
information view, including color changes. Ability sources remain separate.

For a direct pre-stack target query, pass `SpellTargetSource(card)` to
`Target.Possible` or `Permanent.CanBeTargetedBy`. The returned card view is
immutable and keeps the physical card identity. Engine casting, grouped/modal
choices, alternate-zone casting, copy retargeting, resolution, interactive spell
choices, AI spell choices and the test harness supply this context. Existing
ability queries retain their source contract.

RON supports `PermanentRuleEffect(subject, [CantBeTheTargetOfSpells(filter)])`
in the top-level static rule context. `subject` is
bare `ThisPermanent`, or bare `HostPermanent` on an Aura. `filter` reuses the
spell filter grammar: bare `AnySpell`, `IsEnchantmentType(Aura)`, `IsCardtype`,
`IsColor`, and nonempty nested `And`/`Or` lists. Wrong arity, constructor-form
references, unbound references and unsupported spell filters remain declaration
errors. One spell restriction can share its list with one other-Aura restriction,
as described below. Other combined lists and ability
targeting remain separate capabilities.

`CantBeTargetedBySpellsIf(PermanentRef, CardFilter, StateCondition) StaticRule`
adds a pure condition to the same rule. The condition is required and is checked
at each target query, with the source's current controller and incarnation.
Ability targets remain permitted. Use the existing `NotCondition` to invert it.

RON top-level static `If(condition, rules)`, `Unless(condition, rules)` and
`IfElse(condition, trueRules, falseRules)` accept nonempty lists of the existing
conditional modifications. Each branch keeps the existing same-layer limit.
A branch can instead contain exactly one
`PermanentRuleEffect(subject, [CantBeTheTargetOfSpells(filter)])`.
Existing conditional attack and untap bodies retain their support. Branches
cannot declare targets, costs, keywords as bare rules, triggers or resolving actions.
Both branches must compile, including the currently inactive branch.

`IsPlayersTurn(You)` reads the current controller's turn. The exact condition
`PermanentPassesFilter(ThisPermanent, Or([AttackedThisTurn, BlockedThisTurn]))`
reads the original source incarnation's turn history; the two operands can be
reversed. Host history, separate history predicates and other logical history
combinations remain unsupported. Static P/T counts also accept
`ControlledByAPlayer(Opponent)`, including within a nonempty `And` filter, through
the shared opponents and permanent-set services. The count reads current control,
types and phasing. It is not a saved resolution value or an all-zone CDA.
Angry Mob and Lurker use these forms with complete-record gameplay tests.

## Static Aura attachment restrictions

`CantBeEnchantedExceptBy(subject, exception PermanentRef) StaticRule` permits
only the specified Aura to enchant the subject. Both references must be valid
and bound. Use `Static` with `AttachedPermanent` and `This` for an Aura that
forbids other Auras. The rule follows the current attachment and its incarnation.
It checks Aura spell targets, direct attachment, optional hand entry, and existing
attachment legality. It does not restrict ability targets or Equipment.
Illegal attached Auras go to their owners' graveyards in one zone batch;
indestructibility and regeneration cannot prevent that state-based action.

RON supports `PermanentRuleEffect(subject,
[CantBeEnchantedByAnEnchantment(And([Other(ThisPermanent),
IsEnchantmentType(Aura)]))])` at the top level. The two filter operands can be
reversed. The subject must be bare `ThisPermanent`, or bare `HostPermanent`
on an Aura. Exactly these two filter operands are supported. A list can combine
this operation with one `CantBeTheTargetOfSpells(filter)` in either order.
Unknown operations, duplicates, extra filter conditions, other exceptions,
conditional contexts and resolving durations remain rejected.
Anti-Magic Aura and Consecrate Land use these forms with complete-record tests.

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
`IsEnchantmentType(Aura)`, and recursive nonempty `And([filters])` and
`Or([filters])`. Filters match spell cards on the stack and exclude abilities. The engine checks legality at both
announcement and resolution. Countering uses the selected stack object and puts
its card in its owner's graveyard, unless it cannot be countered. Copies and
clones keep separate target choices. Undeclared, incompatible, malformed and
cross-rule reads fail compilation.

`TargetsAPermanent(ControlledByAPlayer(SinglePlayer(You)))` is supported alone
or once in a top-level `And([filters])` for `TargetSpell`. Other conjuncts can
use recursive intrinsic spell filters. It uses
`mage.DeclareSpellTargetTargetingControlledPermanent` with the same per-factory
binding as `Ref_TargetSpell`. The selected spell must currently target a
permanent you control. A target that gained shroud can still qualify; absent,
returned and phased-out targets cannot. Checks run at announcement and
resolution. Copies use their own controllers and stack identities. This form
works in existing spell, mode, activation and trigger target declarations.

Other permanent predicates, players, duplicate target-history predicates and
history predicates nested inside logical operators remain compile errors.
Other enchantment subtypes, controller and mana-value spell filters remain
unsupported. No new modes or multiple spell target groups are added.
Contract: [controlled permanent spell targets](engine-rules-contracts.md#spell-targets-that-target-a-controlled-permanent).
Tests: `TestRONControlledPermanentCounter*`.

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
operands remain rejected in this static context.
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
numeric P/T, independent base P/T/color/type changes and granted activated
abilities remain separate unsupported shapes. Controller-upkeep trigger grants use the
restricted contract under "RON granted controller-upkeep triggers" below.
Conditional rules have the
restricted contract below.
See [the exact contract and tests](designs/mtgish-ron-static-permanents.md).

## RON optional fixed mana payments

An adjacent `PlayerMayCost(player, PayMana([...]))` and
`Unless(CostWasPaid, [actions])` pair builds one `UnlessPaid` effect.
Each pair keeps its own result, including nested unpaid bodies. Payment uses
the engine's total-cost transaction during resolution. Spell-controller payer
references use the current controller of the declared stack target.
Variable mana payments use the separate adapter documented below. Fixed
mana/life alternatives and fixed life payments use the adapter at the end of
this document. Optional sacrifice payments use the adapter below; other
nonmana payments remain rejected.
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
atomic total-cost payment. Optional fixed life costs use the resolution adapter
at the end of this document. Dynamic life costs remain rejected.
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
are sent. Dynamic amounts, saved discard results, and undeclared
event players remain unsupported RON action forms. The selected discard cost
contract is documented below. Evidence:
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
The supported collection animation and equal base-value extensions are described
in the temporary collection animation contract below. Other shapes remain gaps.

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


## RON activation step restrictions

`ActivatedWithModifiers/3` also accepts
`ActivateOnlyIf(IsDuringCombat)`, `ActivateOnlyIf(IsDuringEndOfCombatStep)`,
`ActivateOnlyIf(IsDuringDeclareBlockersStep)`, and `ActivateOnlyIf(IsDuringUpkeep)`.
The conditions are bare symbols. Combat includes beginning of combat, declare
attackers, declare blockers, both damage steps, and end of combat. Each form
permits either player's turn. Existing controller permissions still apply.

These forms lower to `WithActivationCondition` and can be combined through the
existing modifier `And([modifiers])`. Conditions, own-turn restrictions and use
limits remain conjunctive. Queries and mana planning check the same conditions
without payment. Resolution does not repeat the announcement restriction.
Malformed tuples and other step predicates remain errors.

In an immediate activation body, `CreatePermanentLayerEffectUntil/3` accepts
bare `UntilEndOfCombat` for supported singular permanent modifications.
It reuses `Until(core.EndOfCombat, modifications...)`, captures each subject's
incarnation at resolution, and expires after end-of-combat priority. Spell,
delayed and collection modification contexts gain no new duration support.
The existing source-attack trigger context retains its duration support.

Evidence: `TestRONActivationSteps*` covers Jade Statue and Desert through their
complete repository records, including animation, damage, timing, payment,
identity, phasing, copies, clones, combat and native expiry. Tolaria and Lesser
Werewolf have only the timing prerequisite; their other rules remain unsupported.

## RON activation limits

`ActivatedWithModifiers(cost, actions, ActivateOnlyAsASorcery)` accepts the bare
symbol as an activation modifier. Nonempty, nested `And` modifier lists can
combine it with existing timing conditions and limits. The compiler applies
`mage.WithSorcerySpeed()` after the other options, so an own-turn option cannot
erase the main-phase or empty-stack restriction. Costs, references and bodies
retain their existing declaration checks. Tuples such as
`ActivateOnlyAsASorcery()` and uses in actions or costs reject.

The existing engine permits activation in either main phase of the activating
player's turn with an empty stack. The priority loop supplies priority; this
adapter adds no priority permission. The restriction applies at activation,
before choices and payment, and does not apply again when the ability or a copy
resolves. Mana planning uses the same timing restriction as explicit activation.
Evidence: `TestRONSorceryActivation*`. Gaea's Touch and Illusionary Mask have
only this prerequisite supported; their complete records still reject at their
action bodies. The repository prerequisite tests use neutral test bodies and
do not establish complete gameplay for those cards.

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
same one-instruction list uses the typed resolving values described below.
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
not enable activation mana, damage, static values, or optional sacrifice
payments. A declaration cannot combine both characteristic
references or use more than one sacrifice cost, including source sacrifice.
Each factory builds the cost and its consumers with the same private engine
`NumberBinding`; copies keep the paid number without paying again.

For a nonmodal instant or sorcery, one
`CastEffect(AdditionalCastingCost(SacrificeAPermanent(filter)))` supplies the
same saved values to `SpellActions`. It uses the existing matching-sacrifice
filters and consumers above. `AddManaRepeated(production,
ManaValueOfTheSacrificedPermanent)` also accepts one fixed production symbol:
`ManaProduceW`, `ManaProduceU`, `ManaProduceB`, `ManaProduceR`, `ManaProduceG`, or
`ManaProduceC`. The spell pays its printed mana and sacrifice during casting;
the mana action runs during resolution. Each factory builds the additional cost
and effects with the same private binding. Copies retain the payment and use
their own controller. Goblin Grenade and Sacrifice use this form.

Malformed, duplicate, permanent-card and modal declarations remain errors.
Other additional costs, optional costs, activation mana from sacrifice values,
compound/any-color productions and restricted mana remain unsupported.

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
an unsupported body rejects the entire record. Nested modes,
trigger modes, and `Targeted_Modal` remain unsupported. Supported player life
actions and finite prevention can run in each mode.

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
trigger action lists. Numbered references, multiple target groups and target unions
with arbitrary filters remain unsupported shapes. Evidence:
`TestRONDamageScope*` and the migration ledger's P0-TYPED-DAMAGE-RECIPIENT entry.

`TargetPlayerOrPermanent(AnyPlayer, IsCardtype(Planeswalker))` in the same
single-target form builds `mage.DeclarePlayerOrPlaneswalkerTarget`.
`TargetPlayerOrPermanent(Opponent, IsCardtype(Planeswalker))` instead builds
`mage.DeclareOpponentOrPlaneswalkerTarget`. Only the player branch excludes the
resolving controller; either player's planeswalker can be selected. Copies
check the player restriction against their own controller.
`Ref_TargetPlayerOrPermanent` reads that declaration as a damage recipient.
In the `AnyPlayer` form, either player or either player's planeswalker can be selected. A creature that
is also a planeswalker qualifies; a creature without that type does not.
This exact shape works in supported spells, choose-one spell modes, activations
and triggers, including damage packets. The reference is not a singular player
or permanent reference. It cannot be used as one. Other player filters, permanent
filters, arities and unbound or cross-mode reads fail compilation.
Each factory and spell mode builds its own declaration. Copies and clones use
the engine's target identity and legality services. This adapter adds no engine
API. Evidence: `TestRONPlayerPlaneswalker*`; complete records for Axelrod
Gunnarson and Witch Hunter supply the gameplay mappings.

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
`SetPowerAndToughnessBoth` adapters. Halfdane uses the separate end-of-next-upkeep adapter documented below.

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
and Unstable Mutation. Host-death triggers and host-controller end-step patterns
remain unsupported. Resolving references use the separate contract below.

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
counter modifications, other player conditions, type/base-value changes and replacements remain
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
references, general damage sources, finite amounts and additional operations
remain unsupported in this shape. Static combat events have the separate
contract below. These declarations
fail compilation before a factory is returned.

Each factory builds a fresh typed static ability using
`PreventDamageFromMatchingPermanents(This, source_filter)`. It prevents all matching damage to that permanent while the ability
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
other intervening-if events or conditions.

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

## RON land-entry events

Top-level permanent `TriggerA` rules accept these events:

```ron
WhenAPermanentEntersTheBattlefield(IsCardtype(Land))
WhenAPlayerPutsAPermanentOnTheBattlefield(AnyPlayer, IsLandType(Swamp))
```

The second form accepts the five basic land subtypes. Other filters, player
restrictions, malformed envelopes and tuple-spelled references are rejected.
Both events declare `Trigger_ThatPermanent` only inside their action body. The
actor form also declares `Trigger_ThatPlayer`, the player who caused entry.
An entry without a known actor does not match that form. Setup entry can still
match the first form. Playing a land and putting a land onto the battlefield
both cause entry; phasing does not.

Each factory uses a fresh `PermanentBinding` with
`LandEnters().WithPermanent(binding)`. Actor events also use `OfSubtype` and
`WithActor` with a separate `PlayerBinding`. Each pending event saves its own
land incarnation and actor. A returned land cannot replace an old reference.
`ControllerOfPermanent(Trigger_ThatPermanent)` reads the controller at resolution
through `ControllerOfWithLKI`; it does not use the actor or freeze entry control.
Actions on the land require the original, present incarnation. Copies and clones
retain independent resolution state.

`TestRONLandEntry*` covers Ankh of Mishra's complete six records and both Chant
event prerequisites. The Chants still reject their optional counter-placement
payments; these tests do not establish complete Chant gameplay.

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

Other events and new target declarations or modes in the delayed body remain
compiler errors. Bound cast-event spell reads use the saved spell frame described
in "Delayed spell-event references" below. Next upkeep accepts only bare `You`; other players need
an explicit scheduling contract. `AtNextEndOfCombatThisTurn` is distinct from
`AtTheNextEndOfCombat` and remains unsupported. This adapter does not add casting
restrictions, combat events, choices, histories, links, or finite prevention.

### RON combat-pair triggers

`TriggerA` accepts `WhenACreatureBlocksACreature(subject, filter)` and
`WhenACreatureBecomesBlockedByACreature(subject, filter)` with
`SinglePermanent(ThisPermanent)` or, in an Aura, `SinglePermanent(HostPermanent)`.
The filter is `IsCardtype(Creature)`, `IsCreatureType(subtype)`, or a
one-argument `And` list. A conjunction must contain a creature type condition
or a required creature subtype. It permits one of each, one
`IsNonCreatureType(subtype)`, one color condition, and one fixed toughness
comparison. The color condition is `IsColor(color)` or a nonempty
`Or([IsColor(color), ...])`. Each subtype must pass creature-family validation.
`IsCreatureType` also requires the creature card type. It cannot match a
noncreature with that subtype. `ToughnessIs` accepts `EqualTo(Integer(n))`,
`LessThanOrEqualTo(Integer(n))`, or `GreaterThanOrEqualTo(Integer(n))` and reads
current toughness at detection. Other type, history, numeric, controller and
nested conjunction forms remain unsupported. Duplicate conditions of one kind
are errors.

An event union is exactly `Or([blockingEvent, blockedByEvent])`, in either order.
Both branches must have the same position-free subject and filter declarations.
It lowers to the engine's per-opponent combat-pair event, not two abilities or a
once-per-combat group. Arbitrary unions and unequal branch declarations remain
errors. Inventory walks nested events even in unsupported unions.

The blocking event provides `Trigger_ThatOtherCreature`; the blocked-by event
provides `Trigger_TheBlockingCreature`; the union provides
`Trigger_ThatOtherCreature`. All three pair-event forms also provide
`Trigger_ThatOtherPermanent` as an alias for the same opposing incarnation.
The alias is available in resolving action bodies and supported delayed step
actions. It does not grant support to action contexts that reject event
references, such as temporary combat rules. Each is a typed event binding
available only in its body, including supported delayed step actions. Each factory builds a fresh
binding. The matching engine patterns save the opposing incarnation at detection.
Aura events read the current attachment at detection. Later attachment, color,
subtype, toughness or controller changes do not recheck filters or redirect
the saved object.
Departure of the source does not end pending or delayed abilities. A returned
opponent is a new object; a phased-out opponent cannot be destroyed.

The compiler uses `BlocksCreatureMatching`, `BecomesBlockedByCreatureMatching`,
and `BlocksOrBecomesBlockedByCreatureMatching` with a typed source/host reference,
immutable `PermanentFilter`, and `WithOpponent(binding)`. Previous source-only
engine patterns remain available. Source-only `WhenACreatureBlocks` uses the
adapter described below. Host block events, group events and other
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

Static condition filters also accept `HasACounterOfType(counter)` and
`HasNumberCountersOfType(EqualTo(Integer(n)), counter)`, where `n` is a
nonnegative integer literal. They use `mage.HasCounter` and
`mage.CounterCountEquals`. Counter types use the existing permanent-counter
grammar, including supported P/T pairs. Counts above 255 remain integer
comparisons; they do not wrap to the engine's counter storage type.
These filters compose with nonempty `And`/`Or` and also work in the existing
`ControlsA`/`ControlsNo` static player conditions. Dynamic thresholds, other
comparisons, resolving conditions and counter-dependent durations are not added.
Conditional body restrictions remain unchanged, including rejection of
`PermanentRuleEffect` bodies.

`TestRONStaticCounter*` covers the predicate contract and the exact Homarid
and Tidal Influence static P/T clauses. Merseine and Venarian Gold use their
exact parsed predicates with test-only keyword bodies. These four complete
records still reject other unsupported rules; none is a full-card gameplay
coverage claim.

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

Other player filters, action-list bodies, optional discards,
X and player-dependent draw amounts remain compile errors in this context.
Complete Wheel of Fortune gameplay evidence is in `TestRONEachPlayerWheel`,
`TestRONEachPlayerCopies` and `TestRONEachPlayerReplacements`.

### RON all-player library shuffle actions

`EachPlayerAction(AnyPlayer, ShuffleHandAndGraveyardIntoLibrary)` moves both
players' hands and graveyards into their own libraries, then shuffles each
library. Later fixed draw actions remain separate instructions. This uses
`ShuffleHandsAndGraveyardsIntoLibraries(EachPlayer)`.

The adjacent pair `EachPlayerAction(AnyPlayer, ShuffleHandIntoLibrary)` followed
by `EachPlayerAction(AnyPlayer, DrawNumberCards(NumCardsShuffledIntoLibraryThisWay))`
uses `ShuffleHandsIntoLibrariesAndDraw(EachPlayer)`. The engine keeps each
player's moved count local to that resolution. It finishes all shuffles before
any draw. This pair creates no readable result outside its continuation.
Missing, nonadjacent, incompatible, or malformed continuations are compile
errors. Other shuffle-result consumers remain unsupported.

Both forms use resolving action lists, including mode and activation bodies.
They create no targets. They shuffle empty libraries and hands, preserve
owners, and exclude battlefield, exile, and stack cards. Movement is not
discard. Zone events see all completed moves; graveyard departures report one
batch per player. Draws use the existing replacement pipeline and active-player
order. The resolving spell moves to the graveyard after its instructions.
Evidence: `TestRONLibraryRecycle*` and engine `TestLibraryRecycle*`.

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

## RON casting windows

Nonmodal instant and sorcery records can declare one
`CastEffect(CantBeCastUnless(condition))` beside `SpellActions`. Rule order does
not change the restriction. Conditions support nonempty nested `And` lists,
`IsAPlayersTurn(You|Opponent)` and `IsPlayersTurn(You|Opponent)`, and bare
`IsDuringUpkeep`, `IsDuringDeclareAttackersStep`, `IsDuringDeclareBlockersStep`
and `IsAfterUpkeep`. The last form checks whether the first upkeep in this turn has ended.
It permits later upkeep steps (CR 503.2); it does not infer history from a step
number. A skipped upkeep does not supply a completed upkeep. Turn references use the caster, not the card owner.

Lowering uses `WithCastingCondition(StateCondition)` with `IsDuringSteps`,
`AfterFirstUpkeep`, `IsYourTurn`, `NotCondition` and `AllConditions`. The condition is checked before
casting choices and payment on each casting path. It grants no priority or
sorcery timing permission. Copying a spell on the stack and resolving it do not
repeat this check. Casting a copy of a card does repeat it. Modal spells,
permanent cards, other CastEffect operations, other predicates and duplicate
CastEffect declarations remain errors.

Combat conditions also support bare `IsDuringCombat`, `IsBeforeAttackersDeclared`,
`IsBeforeBlockersDeclared`, and `IsBeforeCombatDamageStep`.
The before conditions use the first combat of the turn. A nonempty nested `And`
that includes `IsDuringCombat` uses the current combat for its before conditions,
regardless of operand order. This permits the window in an extra combat phase
(CR 506.7c/d). `IsAfterCombat` remains unsupported.
Skipped blockers/damage close their windows when the attackers step ends;
a skipped combat phase closes them when the precombat main phase ends.
First-strike damage closes the combat-damage window. Tuple forms remain errors.

`ActivateOnlyIf(IsBeforeCombatDamageStep)` uses the same first-combat history
through the existing activation gate. It does not grant priority or remove other
activation restrictions. Angus Mackenzie uses this form with its mana/tap cost
and existing end-of-turn combat prevention. Other before/after activation
predicates are not added in this slice.

Resolving `UntapEachPermanent(filter)` uses `UntapEach` and the existing typed
collection services. Filters support the existing pure permanent filters and
controller restrictions expressed as
`ControlledByAPlayer(SinglePlayer(player-reference))` inside `And`. One
controller restriction is allowed. Selection occurs at resolution and excludes
phased permanents. Declared player targets and event players use their own
instance bindings. Chosen-payment and plural-target collections remain errors
in this action context. Reset combines this adapter with its casting window;
Teleport combines its window with the existing temporary blocking restriction.

### Numeric RON permanent targets

Use `PowerAndToughnessIsEqualTo(PT(p,t))` for exact current P/T.
Use `ToughnessIs(EqualTo(Integer(n)))`, `LessThanOrEqualTo`, or
`GreaterThanOrEqualTo` for a fixed current toughness comparison.
In permanent target declarations, use
`ToughnessIs(LessThan(PowerOfPermanent(ThisPermanent)))` for a strict source
power comparison. The compiler requires a permanent source scope and composes
this restriction with `And` filters and controller restrictions. The engine
checks it at announcement and resolution with retained source information.
The Go API is `TargetToughnessLessThanSourcePower()` in
`DeclarePermanentTarget` or `DeclarePermanentTargets`.

## RON win/loss coin branches

Resolving actions support `FlipACoin_OnLose([actions])` and
`FlipACoin_OnWinAndLose([win-actions], [loss-actions])`. Each supplied list must
contain at least one action. The compiler checks both branches, including
unbound references and adjacent optional-payment continuations. Branches use
the surrounding lexical target, event, X and payment scopes. They cannot declare
targets or export payment results. Nested coin branches are supported.

Lowering uses `FlipCoin(You, OnWin(Sequence(...)), OnLoss(Sequence(...)))`,
omitting OnWin for the loss-only form. One flip occurs at resolution, and only
the matching branch runs. An omitted win branch does nothing. You is the
resolving controller, including on a copied ability. Independent instances use
their own target declarations; the engine owns random outcomes and clone state.

These forms work in supported spell/mode, activation, trigger and delayed
resolving action lists. Delayed bodies retain captured target incarnations and
do not target again. No cost, static, entry, replacement or PlayerAction body
context is added. Bare heads/tails `FlipACoin`, result predicates, repeat
procedures and additional combat actions remain unsupported.

Complete-card evidence: `TestRONCoinBottle`, `BottleDamageAndLKI`, `Captain`,
`Kites`, `CastingAndInstances`, `NestedSequencing`, and `InvalidAndInventory`.
The three mappings are Bottle of Suleiman, Goblin Kites and Orcish Captain.


## RON source block events

`TriggerA(WhenACreatureBlocks(SinglePermanent(ThisPermanent)), body)` uses
`mage.Blocks(mage.This)`. The body is a supported resolving action list or
single target declaration with actions. `Trigger_ThatCreature` denotes the
captured source incarnation in this body and its supported delayed actions.
The binding cannot escape to another ability. Host subjects and filtered
collections remain unsupported. The event has exactly one positional argument;
its selector has exactly one symbol reference.

The event triggers once when the source blocks, including when it blocks
multiple attackers. Entering the battlefield blocking does not trigger it.
The shared engine selects final legal block assignments and preserves pending
trigger controllers, source identity, copies and clone isolation.

`IsntEnchanted` is a zero-argument symbol permanent filter. It lowers to
`mage.Not(mage.IsEnchanted)` and reads current Aura attachments. A phased Aura
does not count. Targets use this filter at announcement and resolution;
equipment does not count as an Aura. Tuple spellings remain invalid.

`TestRONSourceBlock*` maps the complete Time Elemental record to native combat,
next-end-of-combat sacrifice and damage, casting, activation payments, target
rechecks, source LKI, identity, phasing, copies, clones and factory isolation.
Other block consumers retain their separate unsupported actions.

## RON static and indefinite keyword removal

`EachPermanentLayerEffect(IsCardtype(Creature), [LosesAbility(keyword)])`
removes one fixed keyword from all current creatures in layer 6. The list must
contain exactly one modification. Fixed keyword symbols and fixed
`Landwalk(IsLandType(...))` declarations use the existing keyword validator.
Other selectors, Aura hosts, `AnyLandwalk`, chosen abilities and combined static
modifications remain unsupported in this context.

`CreatePermanentLayerEffect(reference, [LosesAbility(keyword)])` installs an
indefinite layer-6 removal on the permanent selected at resolution. This action
uses the existing source, host, target and event reference scopes. It
accepts the fixed keywords and `AnyLandwalk` supported by temporary removal.
The list must contain exactly one modification. References must be bound in the
current scope. An absent or phased subject receives no effect. A returned card
cannot inherit the removal; a phase cycle retains an installed removal.

Engine API: `mage.RemoveEachKeyword(mage.PermanentSet, core.Attr)` returns a
`ContinuousModification`. `Until` captures the set at resolution; `While` reads
current membership at each layer application. Both reuse keyword removal and
timestamp order. A later grant can restore the keyword.

Evidence: `TestRemoveEachKeyword*` and `TestRONLastingKeyword*`. The complete
Gravity Sphere and Elder Land Wurm records have native RON gameplay tests.

## RON batch return to hand

`PutEachPermanentIntoItsOwnersHand(filter)`, `DestroyEachPermanent(filter)` and
`DestroyEachPermanentNoRegen(filter)` accept contextual battlefield collection
predicates. Nonempty `And([filters])` and `Or([filters])` nest freely and keep
player restrictions local to each operand. Repeated or overlapping predicates
select each incarnation once. Characteristic leaves use existing permanent
filters. `OwnedByAPlayer(SinglePlayer(player-reference))` and
`ControlledByAPlayer(SinglePlayer(player-reference))` use typed player references
from the current resolving scope. `ControlledByAPlayer(Opponent)` selects the
resolving controller's opponents. `AttachedToAPermanent(host-filter)` uses the
same collection grammar for hosts, including nested ownership/control predicates.
It checks the current attachment and the exact host incarnation.

Collections read current membership at each instruction. Absent or phased
objects do not qualify; an untargeted returned card can qualify as a new object.
LKI does not restore collection membership. Selection does not target members:
shroud does not protect them. Return and destruction use the existing simultaneous
batches, replacement pipeline, indestructibility and regeneration contracts.
Remove Enchantments returns its owned collection before selecting the destruction
collection. No collection or chosen state is saved in the compiled factory.

For permanent targets, `OwnedByAPlayer(SinglePlayer(You))` lowers to
`TargetYouOwn()` and can be combined with existing conjunction restrictions.
Target ownership and control are checked at announcement and resolution against
the stack object's controller. Other target owner references and contextual
ownership in target disjunctions remain unsupported. Existing attachment target
syntax retains its separate contract.

Engine primitives are `UnionPermanents(sets ...PermanentSet) PermanentSet`,
`IntersectPermanents(sets ...PermanentSet) PermanentSet`,
`AttachmentsOn(hosts PermanentSet) PermanentSet`, and
`PermanentsControlledByPlayers(players PlayerSet, filter PermanentFilter)
PermanentSet`. Union/intersection copy their input slices and emit live members
once in battlefield order. Empty compositions, nil children and unbound nested
references fail definition validation. Existing RON scopes construct bindings
separately for each factory. `TestPermanentSet*`, `TestRONContextualCollection*`
and `TestRONReturnCollection*` supply contract and gameplay evidence.

## RON combat removal and temporary blocking prohibition

Resolving actions `RemoveCreatureFromCombat(ref)/1` and
`RemoveCreatureFromCombatAndUnblockBlockers(ref)/1` use existing scoped typed
permanent references. They are available in supported spell, mode, activation
and trigger action lists, including coin branches. They declare no targets.
Wrong arity, unbound and incompatible references, and cost/static contexts are
rejected. The ordinary action lowers to `RemovePermanentFromCombat`. The second
lowers to `RemovePermanentFromCombatAndUnblock` and explicitly makes qualifying
attackers unblocked. It checks both current blocking membership and the set of
blocker incarnations recorded for that attacker during this combat. Removing a
different blocker does not make the remaining blocker historically sole.
A returned card cannot replace an old blocker. Combat reset clears the history;
game clones keep independent history maps. No attack-unblocked trigger is fired
by removal after blocker declaration.

`CreatePermanentRuleEffectUntil(ref, [CantBlock], UntilEndOfTurn)/3` accepts the
symbol operation, one scoped source or singular target reference, and the symbol
duration. It lowers to `Until(EndOfTurn, CantBlock(ref))`. This rules restriction
uses the shared blocker-filter service, survives recipient ability removal and
phase cycles, and applies in later combats this turn. It keeps the captured
incarnation and expires at cleanup. It does not remove an already declared
block. Other operation lists and durations retain their current support limits.

Complete records for Mijae Djinn (ARN and 3ED) and Ydwen Efreet (ARN) use these
forms with existing coin and source-combat triggers. Evidence:
`TestRONCombatExitNative`, `PendingIdentity`, `Vigilance`, `TargetContexts`,
`Invalid` and `InventoryAndReprints`. False Orders, Imprison and Disharmony gain
shared prerequisites but retain other compiler gaps; no full-card support is
claimed for those records.

## RON announced X activation payments

`Activated/2` and `ActivatedWithModifiers/3` accept the cost
`PayManaX(manaSymbols, ValueX)/2`. The first operand is a mana list with at
least one bare `ManaCostX`; fixed generic and colored symbols can share the
list. The second operand must be bare `ValueX`. Multiple X symbols and X
costs use the same announced value. Non-X `PayMana/1` costs can compose with
these costs through `And([costs])/1`.

The activation declares X for its action and target scope. A
`NumberTargetPermanents(ValueX, filter)/2` target requires that declaration.
`UntapEachPermanent(Ref_TargetPermanents)/1` reads that instance's typed target
collection and lowers to `mage.UntapEach`. Existing filter-based untap forms
keep their behavior. Undeclared, malformed, singular and out-of-scope collection
reads fail compilation.

Payments lower to `mage.ManaCostPayment`. The shared transaction substitutes
announced X before applying an activation cost reduction, then plans and pays
mana and nonmana costs atomically. The stack object retains X even if another
action changes the game's announcement state. Reductions do not change the
number of X targets or the effect's X value. Optional resolution-time payments,
defined-X modifiers, and rounded-half values still require separate adapters.

Evidence: `TestRONXActivation*` and `TestXActivationPayment*`. The complete
Candelabra of Tawnos record has native casting, activation and resolution tests.

## RON recipient-relative player damage

`SpellDealsDamage(ThisSpell, amount, Player(player-reference))` and
`PermanentDealsDamage(ThisPermanent, amount, Player(player-reference))` bind
bare `DealsDamage_ThatPlayer` within `amount` to that typed player reference.
Existing player references and resolution values retain their contracts.
The binding is unavailable to the recipient declaration, other actions,
other modes, delayed bodies, and permanent or mixed recipient amounts.

The same damage actions accept `EachPlayer(AnyPlayer)` or
`EachPlayer(Opponent)`. Each factory creates its own amount-member binding.
`DealsDamage_ThatPlayer` reads that member during amount evaluation. The engine
selects players in turn order, evaluates all amounts before any damage, and
uses one simultaneous batch. Opponents are relative to the resolving controller,
including copied spells. Nonpositive amounts deal no damage. Prevention and
redirection operate through the existing replacement pipeline; changing a
recipient there does not recalculate the original amount.

Malformed player collections and unbound reads are compile errors. General
player filters, permanent-recipient bindings, mixed collections and discard
result values remain unsupported. `TestRONDamageRecipientValue*` covers both
complete Legends records, Storm Seeker and Typhoon, plus declaration, scope,
instance, activation, copy, clone, payment, timing and simultaneous contracts.
Mind Bomb still requires optional discard and saved per-player results.

## Simultaneous allocation for finite recipient shields

The existing `mage.PreventNextDamage(recipient, amount)` and
`Game.AddPreventionShield(recipientID, amount)` APIs now support affected-player
allocation among simultaneous damage sources. `DealSimultaneousDamage` and
native combat steps retain all assignments until replacement choices finish.
The affected player or permanent's current controller uses `ChooseNumber` with
legal bounds to allocate limited shield capacity. Replacement ordering uses
`ChooseMode`, including a choice that applies to a later packet. Source
incarnations, shield consumption, cleanup and game clones use the existing
engine services. Single-damage callers keep their current signatures.

This is an engine prerequisite. The RON compiler still rejects finite
`CreateFuturePreventDamage` declarations. No new RON shape or card factory is
accepted. See the finite-shield contract and test mappings in
[engine-rules-contracts.md](engine-rules-contracts.md).

## Source identity for incoming damage to an attacking band

The optional engine player service `mage.BandingDamageAssigner` supplies a
separate assignment for each blocker. `GetBandingDamageAssignment` receives the
game, blocking source, eligible band members and exact source amount. The result
must assign that amount among those members. Nil or invalid results use bounded
`Player.ChooseNumber` choices. This service has no RON spelling.

Existing `BandingDamageDistributor` callers can still supply one map of
recipient totals. The engine preserves valid totals while assigning each
blocking source separately. Sources keep their own identities and abilities
through finite prevention, redirection, actual damage and cloned games. Losing
banding restores the blocking controller's assignment choice. Blocker links are
recorded for every band member so a departed member does not erase other links.

Only incoming damage to ordinary attacking bands has this new contract.
Outgoing band aggregation remains a migration prerequisite. See
[engine-rules-contracts.md](engine-rules-contracts.md) for signatures and tests.
No additional RON declaration is accepted.

## RON finite recipient prevention

The resolving action
`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(amount, recipient), [PreventThatDamage])`
now lowers to `mage.PreventNextDamage`. `amount` is `Integer(n)` with a
nonnegative literal, or bound bare `ValueX`. Zero installs no effective shield.
`recipient` uses existing typed `Permanent(reference)`, `Player(reference)`, or
bound bare `Ref_AnyTarget`. Targets must be declared in the enclosing action
scope. Spell modes keep separate target bindings. Resolving activations,
triggers and delayed bodies use the same adapter and their existing scopes.

Resolution captures the recipient and reads the amount once. The engine owns
shield consumption, incarnation checks, phasing, simultaneous allocation,
replacement choices, game cloning and cleanup. Unbound or incompatible
references, wrong arity, extra operations, source-specific events, and negative
literal amounts remain compile errors. This action grants no activated modes,
player permissions or additional prevention operations. Guardian Angel and
Balm of Restoration still fail complete-record compilation.

Evidence: `TestRONFinitePrevention*` uses complete records for ten cards and
synthetic scope declarations. The outgoing attacking-band source assignment
limitation remains recorded in the engine contract; these adapters do not
establish that separate combat behavior.

## RON single-card library searches

The resolving action compiler accepts these complete sequences:

```ron
SearchLibrary([FindAGenericCard, PutFoundCardsIntoHand, Shuffle])
SearchLibrary([FindACardOfType(And([IsSupertype(Basic), IsCardtype(Land)])),
               PutFoundCardsOntoBattlefield([EntersNormally]), Shuffle])
```

The basic/land conjunction permits either operand order. No other filter,
number of found cards, destination, reveal instruction, entry modifier, or
instruction order is enabled. Symbols must be bare symbols. Declarations fail
at compilation for malformed or unsupported sequences. Searches are resolving
actions in the existing spell, activation, trigger and delayed action scopes;
they are not costs, static rules, entry operations or explicit player actions.
They use the resolving object's controller and do not produce a result binding.

Lowering reuses `SearchLibraryToHand` and `SearchLibraryToBattlefield`. The engine
chooses at resolution from that controller's current library. An unqualified
search must find a card if one exists; a search for a basic land may find none.
The library is shuffled even if nothing is found. Hand movement and normal
battlefield entry precede the shuffle. Normal entry restrictions and counter
replacements still apply. A prohibited entry leaves the card in the library.
Neither shape reveals a card moved to hand. Complete Demonic Tutor and Untamed
Wilds records have gameplay tests in `TestRONLibrarySearchCards` and
`TestRONLibrarySearchCopiesAndEntry`; declaration, inventory, source-controller
and entry contracts have separate tests in the same file.

## Source identity for outgoing damage from an attacking band

Ordinary attacking bands now assign outgoing damage separately for each member.
The existing `CombatDamageAssigner` receives the member and its own amount.
When an ordinary banding blocker changes the chooser, `BandingDamageAssigner`
or `BandingDamageDistributor` receives each attacking source separately.
Source trample permits an unassigned remainder for the defender only after the
complete band's assignments supply lethal damage to all blockers. Other members'
assigned damage and deathtouch can supply lethal damage; they cannot grant trample.
Invalid maps use a legal default plan. Caller maps remain unchanged.

Every packet keeps its own incarnation and damage information through prevention,
redirection, source departure, actual damage histories, lifelink and game cloning.
`TestRONBandOutgoingDamage` maps complete Benalish Hero, Timber Wolves, Mesa
Pegasus, War Elephant, Icatian Phalanx, War Mammoth and Amulet of Kroog records to
this native combat interaction. This resolves the outgoing aggregation prerequisite
listed above. No additional RON declaration is accepted. See
[engine-rules-contracts.md](engine-rules-contracts.md) for the exact context,
validation rules and engine tests.

## Normal nontrample combat division

The existing `mage.CombatDamageAssigner.GetCombatDamageAssignment` also permits
free division for a normal blocked attacker without trample. With multiple
eligible blockers, return nonnegative amounts for their IDs totaling the exact
source amount. No lethal order applies (CR 510.1c). The engine does not call
`GetBlockerOrder` for this context. Nil or invalid maps use the legal default;
one blocker receives all damage without a choice. Zero and negative sources
make no assignment. No remaining blocker means no player damage.

Combat membership keeps the original incarnations. Separate captured source
packets enter the shared simultaneous damage service, including finite shield
allocation and actual lifelink. Existing player implementations keep their
signatures. The trample and banding contexts keep their existing behavior.
No RON shape is added. `TestRONNontrampleAssignment` supplies native combat
interaction evidence for six complete RON creature records with Wall of Air
and Amulet of Kroog. See the normal nontrample contract in
[engine-rules-contracts.md](engine-rules-contracts.md) for exact test mappings
and remaining contexts.

## RON choose-one activated modes

`Activated(cost, Modal_ChooseOne([mode, mode, ...]))` and
`ActivatedWithModifiers(cost, Modal_ChooseOne([mode, mode, ...]), modifiers)`
accept at least two bodies with the supported `ActionList` or `Targeted` grammar.
All bodies must compile. Each mode has a separate target namespace and fresh
factory bindings. Activation costs and supported modifiers apply to the ability.
Payment-result bindings are available in each mode; target declarations stay
local to their mode. Nested modes and triggered modes remain unsupported.

The adapter uses `NewModalActivated` and the existing activation transaction.
The controller chooses a legal mode and its targets before payment. Only that
mode resolves. Copies keep the selected mode and can select new targets for it.
Source sacrifice does not stop resolution. Target legality and incarnation
checks still apply. Finite shields start at resolution and expire at cleanup.
Balm of Restoration has RON gameplay evidence for both modes, casting, costs,
rollback, copies, clones, recipient identity and simultaneous damage.

## RON resolution-time variable mana payments

`MayCost(PayManaX(mana, amount))` and
`PlayerMayCost(player, PayManaX(mana, amount))` require an adjacent nonempty
`If` or `Unless` body testing bare `CostWasPaid` or `CostWasntPaid`. `mana` is
an existing mana-symbol list containing at least one X. Fixed colored/generic
symbols and multiple X symbols retain their normal payment requirements.
`amount` is `Integer(n)` with n >= 0, a bound bare `ValueX`, or
`ManaValueOfSpell(typed-spell-reference)`. Spell references can be the current
scope's `Ref_TargetSpell` or cast event's `Trigger_ThatSpell`. They are built
with the current factory's target/event declarations. Other values, missing
continuations and unbound/incompatible references reject compilation.

These resolving forms work in supported spell, activation, trigger and delayed
lists; bound delayed cast-event spell reads use the saved frame. They do not
announce X or add a cost to casting/activation. The engine reads the amount once at the
payment action, then uses `IfManaPaid`/`UnlessManaPaid` and the shared cost
transaction. An unavailable cost offers no choice. A zero cost still requires
acceptance. Each adjacent pair keeps its own outcome. Activation-cost reductions
do not apply. RON `ManaValueOfSpell` uses `ManaValueOfSpellWithLKI` in the engine.
It includes the referenced spell's announced X,
uses current information while it is on the stack, and retains its departure
mana value by stack object ID. A recast card does not replace this information.

`TestRONResolutionMana*` covers the complete In the Eye of Chaos record,
explicit casters, source lifetime, spell departure, payments, copies, clones,
multiple targets, declaration errors and inventory roles. Power Sink still
requires its unpaid `PlayerActions` body and is not a complete supported card.

### Player-scoped mana loss and tap collections

`LoseUnspentManaFor(player PlayerRef) ResolvingEffect` removes all mana from that
player's pool, including restricted mana. It preserves life, mana-production
history and other players' pools. Supply a typed player reference; definition
validation rejects missing targets and bindings.

`HasManaAbility PermanentFilter` checks current runtime abilities with the shared
activated and triggered mana classifiers. Tapped state, payment availability and
current production do not change the classification. Ability removal and grants
do change the result.

`ControllerOfSpellWithLKI(spell StackObjectRef) PlayerRef` reads the live spell's
controller or the controller saved at departure. It uses the stack object's
identity, so a later cast of the same card cannot replace that information.
`ControllerOfSpell` remains a live-only query.

RON resolving `PlayerActions(player, [actions])` captures its player once with
`SnapshotPlayer`. `ActionPlayer` is available only in that body. Nested bodies
have separate bindings; a sibling action keeps its prior scope. This slice
supports nested `PlayerActions`, explicit `PlayerAction`, `TapEachPermanent`,
`LoseUnspentMana`, and the existing draw, life, discard and hand-reveal player
actions. Other body actions remain compile errors. Spells, activations, triggers
and delayed resolving lists use the same adapter; costs, static declarations
and entry operations do not.

`TapEachPermanent(filter)` accepts one `ControlledByAPlayer(SinglePlayer(ref))`
restriction, directly or within nested `And` filters, through the existing
controlled collection service. Controller restrictions inside `Or`, duplicate
controller restrictions and unbound player reads remain errors.
`HasAbility(ManaAbility)` and `DoesntHaveAbility(ManaAbility)` use the current
mana-ability filter. `LoseUnspentMana` must be a bare symbol, either in a player
body, an explicit `PlayerAction`, or a resolving list for its controller.

Mana Short and Power Sink use these shared forms. Their complete RON records
are tested for all six reprints by `TestRONManaDenial*`. Compilation and complete
collection gameplay remain separate measurements.

## RON live static combat prevention

Top-level `PreventDamage(event, [PreventThatDamage])` on permanents accepts:

- `CombatDamageWouldBeDealtByCreature(reference)`.
- `CombatDamageWouldBeDealtToRecipient(Permanent(reference))`.
- `CombatDamageWouldBeDealtByACreatureToRecipient(filter, Permanent(reference))`.
- `Or([CombatDamageWouldBeDealtToRecipient(Permanent(reference)),
  CombatDamageWouldBeDealtByCreature(reference)])`, in either order, with
  exactly the same reference in both events.

References are `ThisPermanent` and, on an Aura, `HostPermanent`. Source filters accept `And`/`Or`, type/subtype,
color and supertype filters, `AnyPermanent`, `IsTapped`, `IsUntapped`,
`IsEnchanted`, `IsntEnchanted` and `IsAttacking`. The filtered source event
also requires a creature. `IsBlocking` and other relational/numeric source
filters remain declaration errors. Enchanted and attacking sources use retained
incarnation information after departure or phasing; live sources use current
state.
Target/event references, player recipients, mismatched unions, finite amounts,
and additional operations fail compilation. Resolving prevention retains its
separate duration contract.

These adapters use `Static(PreventCombatDamageFrom(PermanentRef))` and
`Static(PreventCombatDamageTo(PermanentRef, PermanentFilter))`. They read the
current subject for every proposed event and prevent repeated combat packets.
They do not prevent noncombat damage. Source phasing or departure suspends the
rule; changing an Aura's host redirects its rule. No subject is saved at entry.
The engine replacement pipeline owns ordering, rechecks, and unpreventable
packets. `PermanentRuleEffect(reference, [CantAttack])` uses
`Static(CantAttack(PermanentRef))` with the same reference restrictions. This
attack rule remains separate from the host's abilities and does not remove a
creature already in combat.

Evidence: `TestStaticCombatPrevention*` and `TestRONStaticCombatPrevention*`.
Complete RON records: Demonic Torment, Gaseous Form, Enchanted Being.


## RON controller spell-cost reductions

Permanent rules accept `PlayerEffect(You,
[DecreaseSpellCost(filter, [CostReduceGeneric(n)])])`. The amount is one
nonnegative integer literal. Filters accept bare `AnySpell`,
`IsCardtype(Creature|Artifact|Enchantment|Instant|Sorcery)`, and nonempty nested
`And([filters])` or `Or([filters])`. A union applies its reduction once even
when a spell has both matching types. Other player selectors, operations,
reduction kinds, values and contexts remain declaration errors.

The compiler uses `ReduceSpellCostStatic`, `FixedAmount`, `SpellAny`,
`SpellHasType`, `SpellsAnd` and `SpellsOr`. Each factory builds a fresh static
ability and effect. The current controller receives the reduction while the
source is present and has its ability. Phasing makes the source inactive.
Different sources add their reductions. Only generic mana is reduced, including
announced X and additional mana costs. Colored mana and nonmana costs remain
required. Cost increases precede reductions, and the final cost cannot be
negative. The existing payment transaction locks the total before payment.

Real RON tests: `TestRONSpellCostReductionCards`, `Casting`, `Lifecycle`,
`Totals`, `StackingAndColoredCosts`, `Shapes`, `StrictRecords`,
`LockedBeforeSourcePayment`, `Invalid` and `Inventory`
(all names have the `TestRONSpellCostReduction` prefix).

## Fixed-count RON sacrifice costs

`SacrificeNumberPermanents(Integer(n), filter)` accepts a positive integer literal
and the existing sacrifice filter vocabulary, including nonempty nested `And`
and `Or`. It works in activation costs (including `And` cost groups) and in
`MayCost`/`PlayerMayCost` paired with an adjacent `If`/`Unless` payment result.
Dynamic counts, empty logical filters, and action/static contexts remain errors.

The adapter uses `SacrificeNMatchingIncludingSourceCost(n, filter, text)`.
Matching permanents must belong to the payer's current control and be phased in.
The source is eligible when it matches. All distinct choices are locked before
any sacrifice and rechecked by incarnation before payment. One count cost is
one simultaneous departure batch; separate cost declarations remain separate
payments. The total-cost service protects mana and other resources when a locked
count payment becomes stale. Existing `SacrificeNMatchingCost` excludes the
source as before. Positive counts are checked during definition validation.

`ThatEnteringPermanent` is a typed source reference only within
`TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), body)`.
It uses the stack source's saved incarnation, including in copied triggers and
clones. Other events, activation bodies, and tuple spellings reject this name.
This permits Mold Demon's unpaid sacrifice and Electric Eel's entry damage.

### RON all-player untap limits

The compiler accepts
`EachPlayerEffect(AnyPlayer, [CantUntapMoreThanNumberPermanentsDuringTheirUntapStep(Integer(n), filter)])`
in a permanent's rules, with exactly one operation and nonnegative literal `n`.
`filter` uses the existing permanent-filter grammar. It lowers to
`Static(LimitUntapsDuringStep(filter, n))`; it does not grant a recipient ability.
Other player filters, dynamic limits and other operations remain unsupported.
Smoke and Damping Field use this form. The engine saves membership and lets the
active player choose a legal set before simultaneous untap. Overlapping filters
apply together. The spell, cost, entry and resolving-action contexts do not gain
this declaration.

### RON library-top inspection

Resolving action lists accept:

- `LookAtTheTopNumberCardsOfPlayersLibrary(player, Integer(n),
  [LeaveRemainingCardsOnTopOfLibraryInSameOrder])`.
- The same form with `[PutTheRemainingCardsOnTopOfLibraryInAnyOrder]`.
- `MayHavePlayerAction(player, Shuffle)`.

`n` is a nonnegative integer literal. Each form reads an existing typed player
reference; it does not declare a target. Unbound references, dynamic counts,
other ordering operations, multiple operations, and other optional actions
remain errors. Rules, costs and entry operations do not gain these actions.
The resolving controller sees the cards and chooses their order or optional
shuffle, even when the library belongs to another player. Factories build fresh
target references. Copies can retain or change their own targets.

Lowering uses `LookAtLibraryTop(You, player, n, reorder)` and
`MayShuffleLibrary(You, player)`. Short libraries supply all remaining cards;
empty libraries permit a later optional shuffle. Inspection neither reveals
cards publicly nor changes their zones. Evidence: `TestRONLibraryInspection*`.

### Library inspection engine API

`LookAtLibraryTop(viewer, library PlayerRef, count int, reorder bool)` records
private card snapshots at resolution. `Game.LibraryLooks(viewerID)` returns
independent copies of that viewer's history. History records past observations;
it does not grant access to the current library order. Optional
`LibraryLookObserver` callers receive private copies before the next choice.
`LibraryTopOrderChooser` callers choose a complete order of the inspected cards.
Other players use the existing scry choice service; both returned partitions
stay on top. Invalid orders retain the complete original order.

`MayShuffleLibrary(chooser, library PlayerRef)` offers a resolution decision to
the chooser and uses the selected player's existing shuffle service on acceptance.
Both APIs validate their player references. Negative inspection counts fail
card-definition validation. Missing players perform no actions or choices.
HumanPlayer displays private cards and supplies ordering choices through its
own choice channels. The Player interface remains unchanged.

## RON fixed milling

`MillNumberCards(Integer(n))` supports a nonnegative fixed count in resolving
player actions, including `PlayerAction(player, ...)` and `PlayerActions`.
The enclosing player scope supplies the subject. A bare action uses the ability
controller. Typed player targets and local `ActionPlayer` bindings remain
separate between instances and copies. Dynamic counts and all-player milling
are not supported by this adapter.

`MayCost(MillNumberCards(Integer(n)))` and
`PlayerMayCost(player, MillNumberCards(Integer(n)))` require a positive fixed
count and an adjacent `If` or `Unless` payment continuation. They use `MillCost`
through the existing optional payment transaction. A mill action moves as many
cards as possible; a mill cost requires its complete adjusted count. Millstone
and Deep Spawn use these contexts.

`Mill` and `MillCost` use `ChooseCardFromLibrary` to let the library owner order
cards that move together to the graveyard. The choice reason identifies the
graveyard operation. All order choices precede the moves. An invalid selection
uses the first remaining card; zero or one card needs no order choice. Payment
queries and transaction validation make no live order choices. Existing library
order remains the default for `BasePlayer` and cloned players.

### Mana value targets with announced X

Use `TargetManaValueEqualX()` with a typed permanent target to compare its
current mana value with the action's announced X. Use
`DeclareSpellTargetManaValueX(name, filters...)` for a typed spell target.
The target spell's own X contributes to its mana value; X in a permanent's cost
is zero. Target checks and copy choices use the action's stored X.

RON lowers `TargetPermanent(And([IsCardtype(Artifact),
ManaValueIs(EqualTo(ValueX))]))` and
`TargetSpell(And([AnySpell, ManaValueIs(EqualTo(ValueX))]))` to these services.
The compiler requires announced X in the scope. Other comparisons, static
filters, disjunctions and combined spell mana value/permanent-target restrictions
remain unsupported. Detonate and Spell Blast have complete RON gameplay tests.

## RON singular graveyard exile and source-death cards

`Exile([CardInGraveyards(ref)], [])` lowers to
`mage.ExileGraveyardCard(CardRef) ResolvingEffect`. It accepts one declared
graveyard target or `Trigger_ThatGraveyardCard` in the body of
`WhenAPermanentDies(SinglePermanent(ThisPermanent))`. Modifiers, plural subjects,
unbound references and callable reference symbols are rejected.

The death body receives a fresh card binding per factory. Detection saves the
first graveyard visit through `PermanentDies().WithCard(binding)`, restricted by
`EventSourceIsSelf`. Copies retain the captured visit; a card that leaves and
returns is a new object. Existing `PutGraveyardCardIntoHand` can also read this
typed binding. Battlefield return still requires a creature card target.
Host death has the separate binding contract below. General filtered death-card
bindings remain unsupported.

Grave Robbers uses the existing any-graveyard artifact target, mana and tap
payment, and life action. An illegal sole target prevents both exile and life
gain. Cyclopean Mummy uses its captured death card without a target. Phasing
does not cause a death event. See `TestRONGraveRobbers*`,
`TestRONCyclopeanMummyDeathCard`, `TestRONGraveyardExile*` and
`TestRONDeathCard*`.

## RON fixed landwalk keyword filters

`Landwalk(IsSupertype(Legendary))` lowers to the engine's legendary landwalk
keyword. The existing printed, static grant, temporary grant and removal
contexts use the same validator. `HasAbility(Landwalk(filter))` and
`DoesntHaveAbility(Landwalk(filter))` now accept fixed `IsLandType` filters
for Plains, Island, Swamp, Mountain, Forest and Desert, and
`IsSupertype(Legendary)`. They query the current keyword, including layer
changes, and targeted abilities check the filter again at resolution.
The existing `ManaAbility` predicate keeps its separate ability-class behavior.
Unknown, variable, combined and callable-symbol forms remain rejected.
Legendary landwalk blocking overrides remain rejected: the existing override
service supports subtype landwalk only.

`TestRONFixedLandwalkLivonya` and `NativeCombat` execute Livonya Silone's full
record, including first strike. `TestRONFixedLandwalkAssassin`,
`AssassinRollback` and `InstancesAndRetargetedCopy` execute Merfolk Assassin's
full record with live keyword legality, source and target identity, phasing,
regeneration, costs, copies and clone isolation. `Filters`, `GrantRemoval`,
`Invalid` and `Inventory` cover the shared shapes and semantic roles.

## RON rounded resolving values

`HalfRoundedDown(value)` and `HalfRoundedUp(value)` each require one supported
resolving value. They can nest around announced `ValueX`, battlefield counts,
supported captured results, characteristic reads and existing arithmetic.
Bindings remain local to the enclosing action and must exist before the read.
The compiler lowers these forms through the shared engine division services
with a fixed divisor of two. These forms are supported by the consumers of
`compileInstanceValue`, including damage packets. Static layer, entry, cost and
replacement value contexts do not gain support from this adapter.

Banshee uses the announced activation X for half-down targeted damage and
half-up self damage. Eternal Flame uses the resolving controller's current
Mountain count for full targeted damage and half-up self damage. Both use one
simultaneous damage instruction. The corrected Eternal Flame source matches
`data/DRK.json`; it previously halved targeted damage and rounded self damage
down. Evidence: `TestRONRoundedDamage*` and `TestOpponentOrPlaneswalkerTarget*`.

## RON source-conditioned control

`GainControlOfPermanentUntil(subject, duration)` supports typed permanent
subjects in existing resolving permanent action scopes. `duration` is exactly
`UntilPermanentChangesControl(You, ThisPermanent)` or
`Or([UntilPermanentChangesControl(You, ThisPermanent),
ForAsLongAsPermanentRemainsTapped(ThisPermanent)])`. `Or` gives the two endpoints:
either loss of source control or loss of its tapped state ends the duration.
It is not a disjunction of conditions that keep the effect active. Other players,
sources, durations, missing endpoints and incompatible bindings remain errors.
Spell-source, static, entry, cost and replacement-operation contexts gain no
support. Each factory creates its own typed target declarations.

The compiler lowers these forms to
`mage.GainControlWhileSourceControlled(subject, requireTapped)`. The action
keeps the resolving controller and the original source and target incarnations.
It starts only if the source duration is valid at resolution. Source departure,
phasing and loss of control end both forms; untapping also ends the second.
Target phasing preserves a valid duration. Ability removal does not erase an
installed effect. Copies keep their source identity and use their own controller
and target frame. No source LKI or returned object can start the old duration.

Aladdin and Thrull Champion use the control-only endpoint. Rubinia Soulsinger
and Willow Satyr use both endpoints and retain their independent optional untap
permission after the target leaves. Thrull Champion's global Thrull bonus uses
the existing static creature layer. Tests: `TestRONSourceControlCards`,
`RollbackAndCasting`, `UntapAndInstances`, `CopiesAndContexts`, `Invalid`, and
`Inventory`. Compilation counts remain separate from gameplay coverage.

## RON spell-or-permanent color changes

On a nonpermanent spell, `SpellActions(Targeted_Modal(branches))` supports
exactly two `Targeted` branches for this shape: one `TargetSpell(AnySpell)`
with `CreateSpellEffect(Ref_TargetSpell, [SetColor(SimpleColorList([color]))])`,
and one `TargetPermanent(AnyPermanent)` with
`CreatePermanentLayerEffect(Ref_TargetPermanent, [SetColor(SimpleColorList([color]))])`.
Each branch has one target and one action in its `ActionList`. Both branches
must use the same single actual color. Branch order is free. Extra actions,
filters, unbound or cross-category references, different colors, empty or plural
colors, casting modifiers and permanent-card contexts remain errors.

These alternatives lower to one union target, not a modal spell. No mode choice
occurs. A copied spell can choose a permanent in place of a spell, or a spell in
place of a permanent. Each factory creates its own declaration. This shape
covers Chaoslace, Deathlace, Lifelace, Purelace and Thoughtlace, including all
six reprints. `TestRONLace*` covers their complete records. Text-changing
`Targeted_Modal` consumers remain unsupported.

The engine API is `DeclareSpellOrPermanentTarget(name string)
*SpellOrPermanentTargetBinding` and `SetObjectColor(object
*SpellOrPermanentTargetBinding, color core.Color) ResolvingEffect`, with
`WithTargets(object)`. Declaration validation requires a declared nonnil binding
and one of White, Blue, Black, Red or Green. The target uses stack object IDs
and battlefield incarnation IDs. Resolution reuses `Game.ChangeObjectColor`.
The color replaces existing colors without changing mana symbols. It lasts
through cleanup, control changes and phasing, and carries from a permanent
spell to the permanent it becomes. Later zone changes end it. Color changes
are not copiable values. Existing layer timestamps and protection checks apply.

## RON activation history at resolution

`BindActivationHistory(name string) *ActivationHistoryBinding` declares history
for one activated ability. Supply it once with
`WithActivationHistory(binding *ActivationHistoryBinding) ActionOption` on
`NewAction(ActionActivated, ...)`. Use
`IfActivationsAtLeast(binding *ActivationHistoryBinding, minimum int,
body ResolvingEffect) ResolvingEffect` to read the current turn's count during
resolution. Validation rejects missing declarations, nil or empty bindings,
negative thresholds, duplicate options, spell declarations and one binding
shared by different abilities.

The RON compiler supports this resolving action in nonmodal activated abilities
that use the stack:

```ron
If(TimesThisAbilityHasBeenActivatedThisTurnIs(GreaterThanOrEqualTo(Integer(4))),
   [CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,
       ActionList([SacrificePermanent(ThisPermanent)]))])
```

The threshold can be any nonnegative integer. The branch must have at least one
supported action. Its declarations use the enclosing ability's lexical scope.
A delayed body cannot read this enclosing history condition. Spell, static,
trigger and modal contexts remain unsupported. Mana bodies support fixed-color
production with history-gated delayed instructions, as specified below.
The obsolete resolution-count spelling remains rejected.

Dragon Whelp and Nalathni Dragon use this shape. All identical records now count
activations, as Oracle requires. Farrelite Priest and Initiates of the Ebon Hand
use the same per-ability history with immediate mana resolution and automatic
payment planning. Their complete RON gameplay is covered by `TestRONHistoryMana*`.

## Mandatory resolution sacrifice payments

`IfRequiredPaid(player PlayerRef, costs CostGroup, body ResolvingEffect)` and
`UnlessRequiredPaid(player PlayerRef, costs CostGroup, body ResolvingEffect)`
use the shared total-cost transaction without an optional payment choice.
The supported group contains one or more matching sacrifice costs. Empty groups,
other cost types, and captured payment values are rejected by definition
validation. This restriction does not require a player to activate mana abilities
for a mandatory cost (CR 118.3c).

A source-excluding cost excludes the source incarnation saved by the resolving
ability. A returned card is a new object and can pay that old ability.
Payment branches depend on the start of a legal payment, including a sacrifice
whose move is replaced. Failed total payments leave the game unchanged.

RON supports adjacent `MustCost(SacrificeAPermanent(filter))` and
`If(condition, [actions])` or `Unless(condition, [actions])` in supported spell,
activation, and trigger action sequences. The condition is the symbol
`CostWasPaid` or `CostWasntPaid`. Bodies must be nonempty. Each pair keeps its
own result, including nested pairs. The current resolving controller pays.
The filter uses the existing sacrifice filter grammar, with
`Other(ThisPermanent)` also permitted at the root or in nested `And` filters
when the source is a permanent. Source exclusions inside `Or` remain rejected.
Mandatory tap, mana, plural sacrifice, saved sacrifice values, and
`IfElse` continuations remain unsupported RON shapes.

`SacrificePermanentAction` sends sacrifice through the shared replacement
pipeline. `WithExile(sourceID)` redirects the move to exile. The controller
chooses among replacements in the same priority tier; each replacement applies
once and the pipeline checks the changed action again. A redirected sacrifice
fires sacrifice and zone-change events with the final destination. It does not
fire a death event or update creature-death history. Regeneration does not
replace sacrifice.

Evidence: `TestRequiredPayment*` and `TestRONRequiredPayment*`. The complete RON
Lord of the Pit records use this adapter; Transmute Artifact still requires
search and saved-payment-value support.

## RON live static P/T bonuses

The RON compiler supports singular static `AdjustPTXY(PlusX, PlusY, x, y)` and
`AdjustPTForEach(power, toughness, count)` on `ThisPermanent` or a declared
`HostPermanent`. The per-member multipliers are signed integer literals. Values
use integer constants, `Twice`, `HalfRoundedDown`, `HalfRoundedUp`, and supported
battlefield counts. The exact Aura/attachment count uses `AttachedAuras`.

These adapters reuse `Static`, `While`, `ModifyPT`, `CountPermanents`,
`PermanentsControlledBy`, `MatchingPermanents`, `AttachedAuras`, `MultiplyValues`,
`DivideRoundDown` and `DivideRoundUp`. Values are live on each layer pass; they
are not wrapped in `ValueAtResolution`. P/T-dependent values, X and event
results are rejected in this static context. Combined modification lists and
collection subjects remain separate contexts.

Native `CopySpellOnStack` can retarget Aura casting declarations. A permanent
spell copy enters as a token with separate ability and static-effect state.
The normal entry path applies replacements and events; the copy has no mana
spent to cast it. Illegal and countered copies leave no card in another zone.

## RON singular entry counters

`AsPermanentEnters(ThisPermanent, [EntersWithACounter(counter)])` accepts one
entry operation on a permanent. It uses the same validated permanent counter
types as fixed entry counters, including named counters and supported
`PTCounter` pairs. The operation creates a fresh `EntersWithNCounters(counter,
1)` ability for each factory call. The source must be the bare `ThisPermanent`
symbol. Additional operations, other subjects and resolving-action contexts
remain rejected. Entry and counter replacements use the existing engine service.

`TestRONSingularEntryCounter*` executes the complete Icatian Javelineers record
and the exact entry clauses of Homarid and Tidal Influence. Homarid now has
complete counter-cycle coverage. Tidal Influence still rejects its independent casting restriction. See the
[rules contract](engine-rules-contracts.md#ron-singular-entry-counters).

## RON source counter-state cycles

`TriggerA(WhenAPermanentHasNumberCountersOfType(SinglePermanent(ThisPermanent),
GreaterThanOrEqualTo(Integer(n)), counter), body)` supports a nonnegative literal
threshold and the validated permanent counter kinds. Other comparisons, dynamic
thresholds, host subjects and collections remain declaration errors. The body
can read `Trigger_ThatPermanent` as the original source through the engine's
typed source reference. The read is local to this trigger; other rules cannot
use it. Each factory creates a fresh state-trigger ability.

`RemoveCounters([AllCountersOfTypeFromPermanent(counter, subject)])` supports
one resolving instruction with a typed source, Aura host, declared permanent
target or bound permanent event reference. It uses `RemoveAllCounters`, reads
the live count at resolution and removes only that kind. It is not a payment.
`RemoveCounters([ACounterOfTypeFromPermanent(counter, subject)])` uses the same
typed references and removes one available counter with `RemoveCountersFrom`
and `Fixed(1)`. No counter is required for the action to resolve. Later
instructions still run when no counter remains. Plural instruction lists and
numbered or chosen removal shapes remain rejected in actions; existing counter
payment forms remain available in costs.

`WhenState` and `SourceHasCounters` supply arming and the native stack lifecycle.
The threshold is not an intervening-if condition. Reducing the count after the
trigger queues does not cancel its reset. A returned source cannot replace the
original source; phasing makes it absent for the action. Native ability copies,
permanent-spell copies and game clones retain separate execution state.

Evidence: `TestRONCounterState*` in `counter_state_test.go`; full Homarid gameplay,
exact Tidal Influence reset clause, declaration rejection, inventory roles,
entry replacements, targets, hosts, state arming and instance identity. See the
[rules contract](engine-rules-contracts.md#ron-source-counter-state-cycles).

## RON source and Aura-host untap rules

`PermanentRuleEffect(ThisPermanent,[DoesntUntapDuringControllersUntap])` and
`PermanentRuleEffect(HostPermanent,[DoesntUntapDuringControllersUntap])` lower to
`mage.PreventUntapDuringStep`. The host reference requires Aura scope. Static
`If(condition,[PermanentRuleEffect(subject,[DoesntUntapDuringControllersUntap])])`
uses the same rule and the existing static condition compiler, including live
source or host counter checks. The body must contain exactly this one operation
and one rule. Returned objects do not inherit old attachment references.

The restriction is checked before simultaneous untap, after characteristic layers.
Source removal, phasing and ability removal disable it. Removing the recipient's
abilities does not disable a separate source's rule. Other untap effects remain
legal. Optional source untap retains its existing adapter; optional host untap and
mixed conditional rule/layer bodies remain unsupported. `TestRONHostStaticUntap*`
and `TestSubjectUntap*` supply declaration and execution evidence. The four
inventory consumers retain other blockers; this is not full-card support.

## RON resolving single-counter removal

The single-counter action uses the existing source, Aura host, declared target
and bound event scopes. Each factory builds its own references. It adds no
payment, replacement or entry context. Invalid counter kinds, arity, lists and
unbound references fail compilation.

Evidence: `TestRONSingleCounterRemoval*` in `counter_removal_test.go` covers
complete Venarian Gold gameplay and eight exact removal clauses from rejected
records. See the [rules contract](engine-rules-contracts.md#ron-resolving-single-counter-removal).

### RON completed card draws

`TriggerA(WhenAPlayerDrawsACard(players), actions)` accepts `AnyPlayer`,
`Opponent`, or `SinglePlayer(You)`. The event supplies the typed player reference
`Trigger_ThatPlayer` only within its action body. Each factory builds a separate
binding. The adapter uses `PlayerDrawsCard`, `OpponentDrawsCard`, or
`ControllerDrawsCard`; it does not declare a target. Other player filters and
card references remain unsupported in this context.

### RON resolving Aura host controller

Bare `HostController` is a typed player reference in an Aura's resolving actions.
It lowers to `ControllerOfWithLKI(AttachedPermanent)`. `HostController()` and
uses outside an Aura reject during compilation. The reference reads the current
attachment and its controller when the action runs. It does not read the Aura's
controller or the upkeep event's saved `Trigger_ThatPlayer`.

A departed Aura supplies its last attachment. A departed or phased host supplies
its retained controller information. Returned objects cannot replace the saved
incarnations. These information reads do not permit actions on absent hosts.
The existing `HostPermanent` action still requires a present host.

Paralyze's complete record now compiles, including entry tap, normal untap
restriction, and optional four-mana payment by the resolving host controller.
Payment remains optional even when the host is already untapped or absent.
Curse Artifact still rejects optional `SacrificePermanent(HostPermanent)`;
Erosion still rejects alternative `Or` mana/life costs. Neither full card is
claimed supported. Evidence: `TestRONHostController*` in `pkg/mage/dsl/ron`.


## RON fixed alternative resolution payments

In supported spell, activation and trigger action lists, `MayCost(cost)` or
`PlayerMayCost(player, cost)` accepts `PayLife(Integer(n))`, with a nonnegative
integer literal, and `Or([cost, cost, ...])`. Every alternative must be fixed
`PayMana([...])`, fixed `PayLife(Integer(n))`, or another `Or` of those costs.
Each `Or` requires at least two alternatives. X, dynamic values, conjunctions,
other cost types and malformed declarations remain rejected in this context.
Activation costs and spell additional costs do not gain this RON grammar.

The adjacent nonempty `If` or `Unless` continuation uses symbol `CostWasPaid`
or `CostWasntPaid`. Its result belongs to that payment, including nested and
consecutive pairs. Zero payments still require acceptance. After acceptance,
the engine offers payable alternatives to the resolving payer; one payable
option needs no mode choice. Mana abilities can fund mana options. Only the
selected option is paid, through the existing total-cost transaction. Resolution
payments ignore activation cost reductions during affordability and locking.

`Or` lowers to fresh `EitherCost` declarations, and life payment lowers to
`LifePayCost`. Existing `IfPaid`/`UnlessPaid`, typed player and spell references,
attachment information, countering, destruction, phasing and copying services
keep their ownership. The engine's private mana-policy check is shared by
resolution affordability and payment locking. Existing casting and activation
callers retain their policies.

Full-record behavior mappings: `TestRONAlternativePaymentErosion`,
`ThrullWizard`, `InstancesAndRetarget`, and `NativeUpkeepAndLifeClause` in
`pkg/mage/dsl/ron/alternative_payment_test.go`. The last test covers only the
exact upkeep clause of Season of the Witch; its full record remains rejected
for the independent end-step history filter. `Declarations`,
`NestedAndInventory`, and `PolarityAndZero` establish grammar, scope, result and
zero-payment contracts. Inventory role labels do not assert gameplay support.

### RON optional sacrifice of a declared permanent

`MayCost(SacrificePermanent(ref))` and
`PlayerMayCost(player, SacrificePermanent(ref))` accept a bound singular permanent
reference in resolving action lists. The adjacent nonempty `If` or `Unless`
continuation uses the symbols `CostWasPaid` or `CostWasntPaid`; all four branch
polarities work. Results stay local to each pair, including nested pairs.
`MustCost`, composite/alternative costs, and collections gain no new support.

The adapter uses `IfSacrificePaid` or `UnlessSacrificePaid`. These engine effects
resolve the subject once and use the existing matching-sacrifice transaction
with its captured ID and incarnation. The payer must control the subject.
Missing, phased, returned, or uncontrolled subjects cannot pay. A declined or
failed payment does not sacrifice another object. Zone replacements can change
the destination without changing successful payment.

An Aura host-controller upkeep exposes `Trigger_ThatPermanent` as the captured
host and `Trigger_ThatPlayer` as the upkeep player. These bindings are local to
the trigger and are fresh in each factory instance. Curse Artifact uses these
references because its Oracle text refers to that artifact and player. Its
complete record is covered by `TestRONTypedSacrificeCurseArtifact` and
`TestRONTypedSacrificeNativeUpkeep`. The remaining `TestRONTypedSacrifice*` tests
cover rejection, independent targets, copies, nested results and the source
unblocked-event clause. Mindstab Thrull's defending-player binding is supported
by the contract below.

## RON defending-player references and singular targets

`WhenACreatureAttacks(SinglePermanent(ThisPermanent))` and
`WhenACreatureAttacksAndIsntBlocked(SinglePermanent(ThisPermanent))` expose
`Trigger_DefendingPlayer` in their action bodies. The compiler lowers this
reference to a fresh `PlayerBinding` supplied to `WithDefendingPlayer`. It
retains the event player after source departure, return, control changes,
ability removal and copies. Other events and separate abilities cannot read it.

`TargetPermanent(ControlledByPlayer(Trigger_DefendingPlayer))` selects a permanent
that this saved player currently controls. The restriction can appear inside
nested `And` filters with existing restrictions. `ControlledByPlayer(You)` uses
the ability controller. Other player expressions, `Or` combinations and plural
target contexts remain unsupported by this adapter. Each target factory receives
its instance's bindings before it constructs the declaration.

The engine API `TargetControlledBy(PlayerRef) PermanentTargetOption` accepts
`You` and `FromPlayer` of a saved event binding. Definition validation checks
the event producer before checking target reads and rejects missing or invalid
references. Trigger target selection reads a private query frame made from the
existing event captures, without running effects or offering payment. The stack
object retains those player values for legality checks and retargeting copies.
Clones and copies have separate maps. Random target choices consume the live
random stream; candidate queries leave the live resolution frame unchanged.

Mindstab Thrull and Necrite use their complete FEM records. Targets are chosen
when Necrite's trigger goes on the stack; sacrifice is offered at resolution.
If its target becomes illegal, the ability fails to resolve and no sacrifice is
offered. Mindstab Thrull discards from the saved defender's current hand only
after payment. Source return or loss of control prevents payment of the old
source. Existing zone replacements can exile a sacrificed source. Necrite's
destruction forbids regeneration and still respects indestructibility.

Evidence: `TestRONDefendingPlayerNative`, `Pending`, `InstancesAndCopy`,
`ScopeAndInventory` and `Declarations` in `pkg/mage/dsl/ron`, and
`TestTargetPlayerReferenceEvent`, `Definition` and `SnapshotsAndQueries` in mage.

## RON targets that share a blocking pair with the source

Permanent target filters accept `IsBlockingAttacker(ThisPermanent)` and
`IsBlockedByDefender(ThisPermanent)`, alone or in `And` filters. The former
selects a blocker of the source; the latter selects an attacker blocked by the
source. `Or([IsBlockingAttacker(ThisPermanent),
IsBlockedByDefender(ThisPermanent)])` permits either direction, in either order.
Other combat disjunctions, host/event/target references, and non-target contexts
remain unsupported. An empty target disjunction is invalid.

The compiler lowers these forms to the engine's `TargetBlocksSource`,
`TargetBlockedBySource`, and `TargetInCombatWithSource` options on fresh typed
target declarations. No target choice is stored in the compiled plan. The engine
checks live combat membership and both incarnations at selection and resolution.
LKI cannot restore an ended relation. Tapping preserves it; phasing, control
changes, and combat exit end it. Copies and clones use the existing target services.

Evidence: `TestRONCombatTargetRealDeclarations`, `Declarations`, and
`NativeFactories`. Lesser Werewolf uses the resolving property branch described
below. Sentinel still requires indefinite dynamic base toughness. The target
clauses alone do not establish full-card gameplay support.

### Resolving permanent property conditions

RON resolving action lists support
`If(PermanentPassesFilter(subject, filter), [actions])`. `subject` uses the
existing singular permanent references and their lexical scope: source, Aura
host, declared target, or a supplied event reference. Each factory builds fresh
bindings. Branch bodies use the existing action sequence compiler, including
nested branches and adjacent payment continuations. The body must be nonempty.

Pure property filters accept `AnyPermanent`, tapped/untapped, the existing
supported supertype, card type, subtype, color and keyword predicates, fixed
power/toughness comparisons and fixed P/T equality. Nonempty `And` and `Or`
compose these predicates. Numeric limits are fixed integers; their supported
comparison forms are the existing `compileFilter` forms. Historical, controller,
attachment and combat-relation filters remain errors in this context.
`IsAttacking` and `IsBlocking` are supported as bare complete conditions through
`IfPermanentAttacking` and `IfPermanentBlocking`; they require current combat
membership and cannot occur inside a property conjunction or disjunction.

Pure properties lower to
`mage.IfPermanentMatches(PermanentRef, PermanentFilter, ResolvingEffect)
ResolvingEffect`. The predicate receives current information or the retained
snapshot for the same incarnation. It evaluates at its position in the resolving
sequence, rather than at announcement. LKI permits information reads, not actions
on absent objects. Invalid/unbound references, malformed filters and empty bodies
are declaration errors. Static `If`, intervening-if, activation restrictions,
unsupported historical conditions do not gain support here. Resolving property
`Unless` and `IfElse` use the alternatives described below.
Conditional mana production in activation branches is rejected until mana
classification and solver planning support the same condition.

Evidence: `TestIfPermanentMatches*`, `TestRONResolvingCondition*` and the existing
permanent-information, target, combat-state and payment regression tests.

## RON combat relation collections and captured prevention

Resolving `TapEachPermanent(And([IsCardtype(Creature),
IsBlockingAttacker(permanent-ref)]))` selects current blockers with `BlockersOf`.
`IsBlockedByDefender(permanent-ref)` selects current blocked attackers with
`BlockedAttackersOf`. The conjunction requires exactly the creature type and
one relation; either order is accepted. Additional filters and generic unions
remain unsupported in this collection context. Existing typed source, host,
target and event references retain their scope checks and per-instance bindings.

`CreatePreventDamageUntil(CombatDamageWouldBeDealtByACreature(Or([
SinglePermanent(permanent-ref), relation-collection])), [PreventThatDamage],
duration)` accepts exactly one singular reference and one collection in either
order. Duration is `UntilEndOfCombat` or `UntilEndOfTurn`. Each instruction
captures its selected objects at resolution. The prevention sequence uses
`PreventCombatDamageBy` / `PreventCombatDamageByEach` for combat duration, or
`PreventCombatDamageByThisTurn` / `PreventCombatDamageByEachThisTurn` for turn
duration. This adapter applies in existing resolving action scopes, including
conditional bodies; it does not add static prevention, costs or entry operations.

`PreventCombatDamageByEachThisTurn(subjects PermanentSet) ResolvingEffect` uses
the existing installation and replacement services. Definition validation checks
its collection and bound references. Each source keeps its original incarnation;
departure LKI, phasing and control changes do not substitute a returned object.
The effect excludes noncombat damage and expires in native cleanup. Collections
are selected once, and duplicate incarnations install one rule per instruction.

Complete RON gameplay evidence: `TestRONCombatCollectionCards`,
`SewersAttacking`, `AnnouncementLegality`, `Invalid`, and `StrictInventory`.

## RON permanent collection damage

In resolving action lists, `SpellDealsDamage/3` and `PermanentDealsDamage/3`
accept `EachPermanent(filter)`, `EachPermanent(Ref_TargetPermanents)`, and
`EachPermanent(And([Ref_TargetPermanents, property-filter]))`. The intersection
requires exactly one plural reference and one property filter, in either order.
The property filter uses the supported resolving property grammar. A live
battlefield collection uses the supported permanent filters, including attacking
and blocking status and nonempty nested `And`/`Or`. This action declares no targets.

`FilterPermanentSet(PermanentSet, PermanentFilter)` keeps current matching
members from a typed set. `PermanentDamageRecipients(PermanentSet)` places that
set in the existing damage batch without player recipients. Both services retain
definition validation and the original set's identity and target checks. A zero
engine filter keeps all members. RON empty logical filters are errors.

Sandstorm selects attacking creatures when it resolves. Winter Blast first taps
its legal X creature targets, then reads flying on those same legal targets.
A ground creature remains a legal target and is tapped. Later flying changes
change damage eligibility without changing the target declaration. Plural target
references remain local to each spell, mode, activation or trigger; the existing
rejection of delayed plural target reads remains in force. These additions do
not add collection recipients to replacement operations or damage packets.
Evidence: `TestRONCollectionDamage*` and `TestFilterPermanentSet*`.

## Captured temporary characteristic values

`CreatePermanentLayerEffectUntil(subject, [SetPT(ManualPT(power, toughness))],
 duration)` accepts two resolution values in spell, activation and supported
trigger action bodies. `AdjustPTX` accepts the same characteristic operands.
`PowerOfPermanent(ref)` and `ToughnessOfPermanent(ref)` read typed permanent
references with exact-incarnation information. `ActionPermanent` refers only to
the modified subject inside these values. `Minus(left, right)` can nest these
reads and supported integer/X values. Existing `ManaValueOfPermanent` restrictions
remain in force. Static, CDA, entry and indefinite `ManualPT` forms remain errors.

The engine captures both coordinates before it installs the modifications.
Later characteristic changes cannot recalculate those values. Existing layer
7b/7c ordering, target legality, identity, phasing and copy isolation apply.
`ToughnessOfWithLKI(PermanentRef) ResolutionValue` provides the reusable toughness
query; it reads current layered toughness, retained departure/phase toughness,
or zero for an unknown incarnation. It does not permit actions on absent objects.

`UntilEndOfNextUpkeep(You)` saves the resolving controller and ends after priority
in that player's next actual upkeep. A duration created during that upkeep waits
for a later upkeep. Skipped upkeeps do not expire it; extra upkeeps can. Source
control changes do not redirect expiry. The engine API is
`UntilEndOfNextUpkeep(PlayerRef, ...ContinuousModification) ResolvingEffect`.
`UntilNextUpkeep(You)` retains its beginning-of-upkeep endpoint.

Halfdane's complete record uses `ManualPT` and the end-of-upkeep duration.
`TestRONTemporaryCharacteristicHalfdane` covers its casting, trigger, targets,
values, identity and duration; `Values`, `Invalid`, `InvalidShapes` and
`LoadAndInventory` cover shared value forms and strict loading. Berserk still
needs its `AttackedThisTurn` delayed condition; Blood Lust still needs resolving
`IfElse`. Their partial compilation progress is not full-card gameplay evidence.

## RON paid selection and simultaneous untap

The compiler accepts this adjacent pair in an `ActionList`:

```text
PlayerAction(player, ChooseAnyNumberPermanentsAndPayManaForEach(
  And([filter, ControlledByPlayer(PlayerAction_ThatPlayer)]), mana)),
PlayerAction(player, UntapEachPermanent(TheChosenPermanents))
```

Both player references must identify the same typed plan. The controller token
is local to the choice. `filter` must require `IsTapped` through a bare predicate
or an `And` conjunction and otherwise uses the supported permanent filters.
The outer `And` contains exactly one such filter and one controller clause,
in either order. `mana` is the existing nonnegative fixed mana-symbol list;
X is rejected. The result must be a bare `TheChosenPermanents` in the adjacent
untap. Other consumers, intervening actions, unbound reads and different player
references remain errors. This is a complete choice/payment/untap procedure,
not general support for saved permanent collections.

The pair lowers to `mage.UntapWithManaPayment(player, filter, cost)` in resolving
spell, activation, trigger and branch action lists. The engine owns the local
selection, incarnation capture, total mana transaction and untap batch. Choices
are made at resolution, not announcement. They are not targets; shroud does not
exclude them. An upkeep trigger retains its event player through source control
changes, departure, stack copying and game cloning. Each resolution has its own
choice and payment result. Magnetic Mountain (ARN and 3ED) and Thelon's Curse
(FEM) have complete-record gameplay tests in `TestRONPaidUntapCards`.

### RON exile activation payments

`Exile([Permanent(ThisPermanent)], [])` is an activation cost. It uses
`ExileSourceCost` and can be combined with mana, tap, life, and other supported
costs in `And`. Exile happens during payment; the body runs at resolution.

`Exile([ACardInGraveyards(And([cardFilter,
InAPlayersGraveyard(SinglePlayer(You))]))], [])` chooses one matching card from
the activator's graveyard. Exactly one graveyard exile cost is supported in an
activation. The controller clause must occur once in the outer conjunction.
Other conjuncts use the existing intrinsic card type/color filters and nonempty
logical combinations. Plural, foreign-graveyard, and permanent-choice payments
remain unsupported. The object list has one element and the operation list is
empty. These forms are not resolving actions or resolution costs.

`ManaValueOfExiled(TheCardExiledThisWay)` reads a typed number saved by that
activation's graveyard payment. Each factory creates a fresh number binding;
pending activations, modes, copies, and clones retain their own paid values.
Source exile cannot produce this result. Undeclared, malformed, duplicate, and
cross-activation results are compile errors. `NumberCountersOfTypeOnPermanent`
accepts this value without extending other counter-amount contexts.

Bare `ShuffleGraveyardIntoLibrary` in a resolving action list lowers to
`ShuffleGraveyardFor(You)`. It moves only the resolving controller's graveyard,
then shuffles that library even when the graveyard is empty. A changed-controller
copy uses its own controller. This action makes no target declaration.
Evidence: `TestRONExilePayment*` and `TestGraveyardOnlyShuffleBatch`.

## RON opponent entry choice

`AsPermanentEnters(ThisPermanent,[ChooseAPlayer(Opponent)])` lowers to
`mage.ChooseOpponentAsEnters()`. It saves the entry controller's opponent on
the permanent before battlefield registration. The choice stays with that
incarnation through control changes and phasing. A returned card chooses again.

With this declaration, an upkeep trigger can select
`SinglePlayer(TheChosenPlayer)`. Its body reads the saved event player through
`Trigger_ThatPlayer`. Black Vise and The Rack use that reference for both
`TheNumberOfCardsInPlayersHand` and damage, so their amounts use the current hand
at resolution and their recipients survive source departure and stack copying.

Exactly one opponent choice operation and declaration are supported. The rules
may list the upkeep consumer before its entry declaration. Duplicate and unbound
declarations are errors. Combined entry choices, other player filters, resolving
choices, chosen-player action reads and chosen-player end-step triggers remain
unsupported. The legacy opponent ETB helpers retain their current behavior.

## RON end-of-combat triggers

`TriggerA(AtTheEndOfCombat, body)` accepts the bare event symbol and a supported
action list, including its standard target declaration. It lowers to
`mage.At(mage.EachEndOfCombat, ...)`. Both players' combats match. This event
supplies no `Trigger_ThatPlayer` or `Trigger_ThatPermanent` binding.

The supported intervening shape is
`TriggerI(AtTheEndOfCombat, ACombatPermanentPassesFilter(ThisPermanent,
Or([AttackedThisCombat, BlockedThisCombat])), body)`. The two history symbols can
occur in either order, once each. Other subjects, predicates, combinations and
event tuple forms remain errors. It lowers to
`mage.WithInterveningIf(mage.SourceAttackedOrBlockedThisCombatCondition())`.
The original incarnation must have attacked or blocked in the current combat
at detection and resolution. Entering attacking does not count as attacking
(CR 508.4). Leaving combat does not erase a declaration. Phasing prevents the
condition from matching; a phase cycle preserves history. A new incarnation
has no history from the old one. Each factory builds separate abilities and
target declarations through the existing action-plan service.

Evidence: `TestRONEndCombat*` tests the entry and combat rules extracted from
complete Clockwork Avian and Clockwork Beast records, plus synthetic plain-event,
identity and invalid-declaration contracts. Their restoration rules remain
unsupported. Neither clockwork nor The Wretched has complete gameplay coverage
from this adapter, and their complete records still fail strict loading.

## RON bounded optional counter placement

`PutCountersWithRestriction([UptoNumberCountersOfTypeOnPermanent(amount,
 counter, subject)], CantCauseTheNumberOfCountersOfTypeOnPermanentToBeGreaterThan(
 counter, subject, Integer(limit)))` accepts one placement and one restriction.
The counter and typed permanent reference must match. The limit is an integer
literal from zero through 255; the amount uses existing resolving counter values,
including
announced `ValueX` only in an X-enabled scope. Permanent counter types use the
existing placement validator. Source, attachment, target and event references
retain their existing scope requirements. This is a resolving action, not a
cost, static modification, entry operation or replacement declaration.

The adapter builds `mage.AddCountersUpTo` with fresh instance bindings. At
resolution, the effect controller chooses zero through the smaller of the
amount and available counter room. An absent or phased subject, nonpositive
amount or full subject causes no choice. Invalid choice responses place nothing.
The action limit applies to its counter on its original subject after counter
replacements; it does not remove excess counters supplied by other actions.

Clockwork Avian and Clockwork Beast now compile from their complete records.
`TestRONBoundedCounters*` covers casting, choices, payment, upkeep timing,
identity, target legality, copies, clones, replacement ordering and instances.
`TestRONEndCombatClockwork` and `TestRONEndCombatInterveningIdentity` now use
complete records for entry, characteristics and combat-history interactions.
Other counter instructions and restriction shapes remain errors.

## Static prevention by source card type

`mage.PreventDamageFromCardType(subject mage.PermanentRef, kind core.CardType)
mage.StaticRule` prevents combat and noncombat damage from sources with the
specified card type. Compose it with `mage.Static`. The subject is live: `This`
protects the ability source, and `AttachedPermanent` follows the current host.
Invalid card types, nil subjects, and undeclared bindings fail definition
validation. Source types use the exact damage source's current or retained
information. Captured spells can match; a returned permanent cannot replace an
older source. The rule uses the existing static lifecycle and damage pipeline.

RON accepts the static rule
`PreventDamage(DamageWouldBeDealtByASourceToRecipient(IsCardtype(type),
Permanent(subject)), [PreventThatDamage])`, where `subject` is `ThisPermanent`
or a declared Aura's `HostPermanent`. Other filter shapes and resolving
contexts remain unsupported. Argothian Treefolk uses the complete shape.
Artifact Ward uses its host context, but its separate ability-target restriction
still prevents complete loading.

## RON prevention from blocked creatures

A permanent static rule accepts
`PreventDamage(DamageWouldBeDealtByAPermanentToRecipient(And([IsCardtype(Creature), IsBlockedByDefender(ThisPermanent)]), Permanent(ThisPermanent)), [PreventThatDamage])`.
The two filter operands can occur in either order. Both references must name the
permanent source. Other types, references, recipients, operations, filter shapes
and resolving/cost contexts fail compilation. This shape does not extend the
ordinary permanent filter or target compiler.

Each factory builds `Static(PreventDamageFromBlockedCreatures(This))` with its
own ability. The engine checks the current blocking pair for each proposed
combat or noncombat damage event. Former pairs cannot match through LKI.
Wall of Vapor has complete RON gameplay coverage in
`TestRONBlockedDamagePreventionWall`, `StrictInventory` and `Ordering`.
Wall of Shadows shares this rule but still fails its separate targeting rule.

RON resolving `CreatePermanentLayerEffect(subject, [SetToughness(value)])`
sets base toughness indefinitely through `mage.SetBaseToughness`. It preserves
base power, bonuses and counters. The value is captured at resolution.
Supported values are integer literals, binary nested `Plus`/`Minus`, typed
permanent power/toughness reads, and a matching graveyard-card count with one
`InAPlayersGraveyard(SinglePlayer(playerRef))` and one card filter in `And`.
The operands can appear in either order. Existing lexical bindings apply.
Static and temporary RON forms and chosen-number values remain unsupported.
Sentinel and Wall of Tombstones have complete-record gameplay evidence in
`TestRONIndefiniteToughness*`; shared engine contracts are documented in
`engine-rules-contracts.md` under “RON indefinite base toughness.”

### Optional RON entry from hand

`MayAction(PutACardFromHandOnBattlefield(filter, [EntersNormally]))` is a
resolving action. It uses `mage.MayPutFromHandOntoBattlefield(mage.You, filter)`.
The controller makes the optional choice during resolution. Copies use their
own controller. The action has no target and does not cast or play the card.

The card filter accepts bare `IsPermanent`, `IsSupertype(Basic|Legendary|Snow|World)`,
`IsLandType(subtype)`, `IsCreatureType(subtype)`, existing card-type and color
filters, and nonempty nested `And`/`Or` lists. Subtype families are validated.
Only one bare `EntersNormally` entry operation is supported. Explicit player
loops, repeated processes, other entry operations and action-result reads remain
separate contracts. The engine excludes instants and sorceries from entry.

The engine keeps each offered hand visit through both choices. Invalid, absent
or returned choices cannot move a card. Entry uses the existing preparation,
registration, attachment and entry-event services. A land-entry prohibition
keeps the card in hand with its original visit. Auras use the existing permanent
host contract: legal attachment choices ignore shroud, obey protection and
exclude phased hosts. No legal host leaves the Aura in hand.

Complete Gaea's Touch behavior is covered by `TestRONOptionalHandEntryGaeasTouch`
and `TestRONOptionalHandEntryTimingAndRollback`. Goblin Wizard's temporary
protection and Eureka's repeated player procedure remain unsupported. Their
complete records still reject. Triassic Egg now has a corrected graveyard mode
and live activation-condition support; see the activation-property section below.


## RON controller restrictions on enchant abilities

On an Aura, `EnchantPermanent(filter)` accepts
`ControlledByAPlayer(SinglePlayer(You))` or `ControlledByAPlayer(Opponent)`.
One controller restriction can occur alone or in a nonempty nested `And`
with existing permanent property filters. Operand order does not matter.
A property-only `Or` retains the existing filter grammar; a controller
restriction inside `Or` is rejected. Duplicate or conflicting controller
restrictions, other player references, malformed constructors and enchant
rules on non-Auras are compile errors.

RON lowers these restrictions to fresh `TargetPermanentYouControl(filter)`
or `TargetPermanentOpponentControls(filter)` cast targets. Every factory
has its own casting target state. Casting and resolution use the spell's
controller; a copied spell uses its own controller. Direct attachment,
optional hand entry and state-based actions use the Aura's current or
prospective controller. Ownership does not determine the restriction.
Shroud and hexproof affect spell targeting, but do not prevent untargeted
attachment or invalidate an existing attachment.

Evidence: `TestRONEnchantControllerClauses`, `Invalid`, `NestedFilters` and
`Reporting`. Gameplay tests execute the real enchant clauses from Cocoon,
Relic Bind and Tourach's Gate, including casting costs and metadata. They
are clause evidence: all three complete records remain rejected for later
unsupported operations. No full-card behavior claim follows from these tests.

## RON resolving counter conditions

Inside a resolving `If(PermanentPassesFilter(subject, filter), [actions...])`,
counter filters accept `HasACounterOfType(counter)`,
`HasNoCountersOfType(counter)`, and
`HasNumberCountersOfType(comparison(Integer(n)), counter)`. The comparison is
`EqualTo` or `GreaterThanOrEqualTo`; `n` must be a nonnegative integer literal.
The counter uses the existing permanent counter grammar, including supported
`PTCounter` coordinates. Nonempty `And` and `Or` combine these with the existing
resolving property filters. Other comparisons, dynamic thresholds, malformed
constructors and unbound subjects are declaration errors.

The condition reads the count when its action runs, after earlier instructions.
It uses current information or LKI for the exact subject incarnation. Retained
information permits a condition read, but cannot permit an action on an absent,
returned or phased-out object. Each factory constructs its own typed targets;
copies and clones retain their own bindings and state. Conditional mana
activations remain rejected until solver planning supports their complete body.
Static conditions, activation restrictions and exiled-card conditions retain
their separate grammars.

`mage.CounterCountAtLeast(counter, minimum)` compares the stored count as an
`int`; a threshold above 255 cannot wrap to a smaller count. Presence and absence
reuse `HasCounter` and `CounterCountEquals`. Execution reuses
`IfPermanentMatches`, counter actions and the existing reference service.

Evidence: `TestCounterCountAtLeast` and `TestRONResolvingCounter*`. The actual
Fasting upkeep and Tourach's Gate counter-decay clauses have gameplay coverage.
Fasting still rejects its draw-step replacement, and Gate still rejects its
host tap cost. These tests do not establish complete-card support.

### RON live activation property restrictions

`ActivatedWithModifiers(cost, body, ActivateOnlyIf(PermanentPassesFilter(subject,
filter)))` checks the current source or Aura host before activation payment.
`subject` is bare `ThisPermanent`, or bare `HostPermanent` on an Aura. Targets,
event bindings and other subjects are not available in this declaration scope.
Ordinary and modal bodies use the same restriction. Nested modifier `And` lists
combine property, turn, step and usage restrictions without replacing them.

Filters use the existing resolving property filter grammar: current types,
subtypes, colors, supertypes, tapped state, keywords and supported P/T comparisons;
nonempty nested `And`/`Or`; counter presence/absence; and counter counts with
`EqualTo(Integer(n))` or `GreaterThanOrEqualTo(Integer(n))`. Counter thresholds
must be nonnegative literals, and counter kinds must be known. This filter
reads live information, without departure LKI. Static and resolving condition
contexts retain their separate reference and evaluation contracts.

Lowering uses `mage.WithActivationStateCondition(mage.PermanentMatchesFilter(...))`.
The engine validates the typed declaration and uses the same pure activation
check for mana planning and actual payment. Restrictions do not repeat at
resolution or when a stack ability is copied. Changing counters or tapping the
host after activation does not stop the ability. Phased, absent and stale hosts
cannot qualify; changing the attachment changes the live subject.

`TestRONActivationProperty*` covers these contracts and both complete Triassic
Egg modes. Its corrected graveyard mode targets a creature card in your
graveyard at announcement; the hand mode makes an optional choice at resolution.
Tourach's Gate has activation-condition clause evidence only; its complete record
still rejects the separate host tap cost.

## RON last-turn attack conditions

Static `PermanentPassesFilter(subject, AttackedDuringLastPlayersTurn(player))`
accepts bare `ThisPermanent` or an Aura's `HostPermanent`. The player is bare
`You` or `ItsController`: the latter reads the current subject controller,
which can differ from the Aura controller. The condition reads the player's
most recent completed turn and the live subject's exact incarnation. Current
turn attacks do not replace that history. Phasing preserves history; absent,
phased and returned objects cannot use an old incarnation's history.

Static `If(condition, [PermanentRuleEffect(subject, [CantAttack])])` now uses
`mage.CantAttackIf(PermanentRef, StateCondition)`. The body has exactly one rule
and one bare operation. It reuses the existing static condition grammar,
including the last-turn condition. Existing conditional untap bodies keep
`PreventUntapDuringStep`. Mixed bodies and additional operations still reject.
History predicates in resolving, activation, cost, target and nested property
filter contexts are not added by this slice.

Engine `PermanentAttackedDuringLastTurn(PermanentRef, PlayerRef) StateCondition`
accepts `This` or `AttachedPermanent`; the player is `You` or `ControllerOf`
a live source/attachment. Invalid declarations fail definition validation.
Turn cleanup archives incarnation history per player; cloning copies all maps.
The existing ID-only history API remains available to its callers.
Evidence: `TestLastTurnAttack*` and `TestRONLastTurnAttack*`, including complete
Giant Turtle, Goblin Rock Sled and Tangle Kelp records.

## RON static source combat attributes

`PermanentRuleEffect(ThisPermanent, [MustAttack])` and
`PermanentRuleEffect(ThisPermanent, [CanBlockAnAdditionalCreature])` accept one
bare operation on the live permanent source. These declarations lower to
`Static(While(condition, GrantKeywordModification(This, attribute)))` with
`AttrMustAttack` or `CanBlockAdditional`. Each factory builds a fresh ability.
Tuple operations, tuple references, target references, Aura hosts, collections,
conditional bodies and combined operation lists remain unsupported here.

Native attacker declaration includes a source with `MustAttack` when it can
attack. Tapping, summoning sickness, phasing, attack restrictions and unpaid
attack costs still prevent declaration. The requirement applies in each combat
and reads the controller, rather than the owner. Additional blocking permits
this source to block two distinct attackers; it does not permit blocking a
flying attacker without flying or reach. It does not require a block. Ability
removal changes later declarations and leaves existing combat pairs intact.

Complete-record evidence: Juggernaut and Two-Headed Giant of Foriys map to
`TestRONSourceCombatAttributeCast`, `RequiredAttack`, `AdditionalBlock`,
`AttackControlAndFinality`, `DeclaredBlockFinality`, `GiantTrample`, `Lifecycle`,
`Invalid` and `Reprints` (all have the `TestRONSourceCombatAttribute` prefix).
Primordial Ooze remains rejected for its unresolved `CreateValueX` action.
This slice does not establish additive multiple extra-block grants, all-zone
CDA values, arbitrary combat requirements or new combat decision services.

## RON damage-received triggers

Permanent rules support `TriggerA` with either
`WhenAPermanentIsDealtDamage(SinglePermanent(ThisPermanent))` or
`WhenAPlayerIsDealtDamage(SinglePlayer(You))`. The first form uses
`ReceivesDamage(This, amount)`; the second uses
`ControllerIsDealtDamage(amount)`. Both save `Trigger_AmountOfDamageDealt`
through a fresh `NumberBinding` for each factory instance. Counter placement
can use this saved number in `NumberCountersOfTypeOnPermanent`.

Only the permanent event supplies `Trigger_ThatCreatureOrPlaneswalker`. It
resolves through `This` and keeps the original source incarnation. It is not a
target. The player event checks the source controller when damage occurs; it
does not supply `Trigger_ThatPlayer`. Other subjects, malformed references and
reads outside the matching event body remain declaration errors.

Fungusaur's complete RON record is supported. Living Artifact's damage clause
is supported, but its complete record still rejects optional counter payment.
Lich retains its other unsupported contracts. This adapter does not grant
support for general damage-recipient filters or optional counter payments.

RON optional counter-placement payment accepts
`PlayerMayCost(payer, PutCounters([ACounterOfTypeOnAPermanent(counter,
And([IsCardtype(Creature), ControlledByAPlayer(SinglePlayer(payer))]))]))`
and implicit `MayCost` with `You` as payer. The two filters can appear in either
order. The controller reference must be the same bound player reference as the
payer. Counter types use the existing permanent counter vocabulary. The cost
requires one instruction and one counter; plural amounts, other filters, and
other controller references remain rejected.

These payments require an adjacent `If` or `Unless` with `CostWasPaid` or
`CostWasntPaid`, in the existing resolving spell, activation, mode, or trigger
action sequence. The cost and continuation stay together. A later payment or
sibling action cannot read that result. Selection occurs during resolution,
without targeting. A shrouded creature can pay. The engine locks its incarnation
and verifies current control and presence before payment. Counter replacements
can change or prevent placement without changing successful payment (CR 118.11–12).

`PutCounterOnMatchingCost(counter core.CounterType, filter PermanentFilter) Cost`
is the shared engine primitive. It uses the payer's current permanents, permits
the source, and treats a zero filter as unrestricted. Availability queries do
not choose or place counters. Preparation locks one choice; commit does not
choose again. The shared cost transaction and counter replacement pipeline
supply rollback, game isolation, and replacement ordering.

Evidence: `TestPutCounterCost*`, `TestRONCounterPlacementPayment*` and the existing
`TestRONLandEntry*` event contracts. The complete FEM records for Thelon's Chant
and Tourach's Chant now load and have gameplay tests for both their rules.

## Beginning-of-combat RON triggers

`TriggerA(AtTheBeginningOfCombatDuringAPlayersTurn(players), actions)` supports
bare `AnyPlayer` and `SinglePlayer(You)`. Each combat begins a separate event,
including extra combats. `You` uses the source controller at detection.
`Trigger_ThatPlayer` is a typed, local event-player reference. Each factory call
creates a separate binding, and each pending event saves its player.

The adapter uses `TurnStepBegins` with `EachBeginCombat` or `YourBeginCombat`.
Supported action lists retain their existing target and payment contracts.
Temporary permanent layer effects in this immediate body can use
`UntilEndOfCombat`. A nested delayed-step body does not inherit that permission.
Other player filters, malformed tuples, and event references outside their body
are rejected during compilation. This does not add `TriggerI`, combat-scoped
installed triggers, optional nonmana effect payments, or conditional attack tap
rules.

`TestRONBeginCombatBinding`, `BatteringRamClause`, `NativeAndControl`,
`ConsumerBoundaries`, `Invalid`, `ScopeRejection`, and `Inventory` cover this
contract. The Battering Ram test executes its complete combat-start RON clause;
the rest of that card remains rejected. Goblin Flotilla and Johan tests execute
their original event declarations with test-only bodies. These mappings do not
claim complete gameplay coverage for any of the three cards.

## RON granted controller-upkeep triggers

Supported declarations are
`PermanentLayerEffect(HostPermanent, [AddAbility([TriggerA(event, body)])])`
for an Aura and
`EachPermanentLayerEffect(filter, [AddAbility([TriggerA(event, body)])])`.
Each contains exactly one modification and one granted ability. `event` must be
`AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You))`. Other events, player
selectors and combined modifications remain rejected.

The collection uses the existing static permanent filter adapter, including
its controller and source-exclusion forms. A host grant follows the current
attachment. The grant operates in layer 6. `You` and `ThisPermanent` inside
`body` refer to the recipient's trigger controller and source, respectively.
The body receives a fresh permanent scope with the ordinary upkeep event's
`Trigger_ThatPlayer` binding. It can declare its own targets. It cannot read the
donor's attachment, entry choices, X, targets or event/payment results.
Existing action and adjacent-payment restrictions still apply.

The adapter uses `GrantTriggeredAbility(PermanentRef, *GenericTriggered)` or
`GrantEachTriggeredAbility(PermanentSet, *GenericTriggered)`. Each factory builds
fresh declarations. The engine keeps ability identity separate for each grant
and each pair of donor/recipient incarnations. Layer passes and control changes
do not change that identity. Phasing suspends the grant without changing it.
Pending triggers keep the recipient incarnation and detection controller even
if the grant ends before resolution.

Complete record mappings are Energy Flux (artifact sacrifice unless {2} is
paid), Farmstead (optional {W}{W}, then gain 1 life), and The Tabernacle at
Pendrell Vale (creature destruction unless {1} is paid).
`TestRONGrantedUpkeep*` covers their casting or land play, native upkeep,
payments, recipient identity, attachment, control, phasing, copies, clones,
ability removal, later members, multiple donors, malformed declarations and
strict loading. Compilation and collection-wide gameplay coverage stay separate.

## RON continuous public visibility

In a permanent declaration, the compiler supports exactly one operation in:

- `EachPlayerEffect(AnyPlayer, [PlaysWithHandRevealed])`
- `EachPlayerEffect(AnyPlayer, [PlaysWithTopOfLibraryRevealed])`

These are static player rules, not actions or one-time reveals. The operation
and `AnyPlayer` must be symbols. Tuple operations, extra arguments, combined
operations and other player selectors are rejected. Existing all-player untap
limit declarations keep their compiler and engine behavior.

The compiler lowers the declarations to `mage.Static(mage.RevealAllHands())` or
`mage.Static(mage.RevealAllLibraryTops())`. Each factory builds its own ability
and effect state. Visibility follows current contents while the printed ability
is active, regardless of its controller. The engine owns source lifecycle,
phasing, ability suppression and cloned visibility flags. Public hands appear
in interactive snapshots without enabling the debug hand-display option.

Complete-card mappings: Field of Dreams and Revelation use
`TestRONPublicVisibilityCards`, `Lifecycle`, and `Contents`;
`TestRONPublicVisibilityInvalid` and `InventoryAndAtomicLoad` cover declaration
validation, nested semantic roles and strict loading. See the continuous public
visibility contract for engine and snapshot evidence.

## RON referenced permanent damage and source tap events

`PermanentDealsDamage(source, amount, recipient)` accepts typed permanent
references for `source` with singular `Permanent(ref)`, `Player(ref)`, or bound
`Ref_AnyTarget` / `Ref_TargetPlayerOrPermanent` recipients. Source and recipient
references use their own source, Aura host, target or event scope. The compiler
uses `DealDamageFrom(DamageSourceOf(source), recipient, amount)` and existing
resolution values. Explicit player recipients also provide the existing
recipient-value scope. Collection damage retains its existing supported forms.

`WhenAPermanentBecomesTapped(SinglePermanent(ThisPermanent))` uses
`BecomesTapped(This)`. Its body may read `Trigger_ThatPermanent` as the exact
trigger source. That name is unavailable outside the event's body. The existing
host-tap form keeps its captured host binding. Factories build separate target
and event declarations; pending copies and clones use the engine's stack frames.

Evidence: `TestRONReferencedDamage*` covers the complete City of Brass and
Tracker records, typed host/event/target sources, invalid declarations,
incarnation and clone isolation, announcement costs, prevention and sequential
resolution. These adapters add no source-choice or damage-result operation.

## RON destruction result branches

In supported resolving action sequences, `MustCost(DestroyPermanent(ref))`
followed immediately by `If(condition, [actions])` or
`Unless(condition, [actions])` attempts destruction and branches on its result.
`condition` is the symbol `CostWasPaid` or `CostWasntPaid`. Bodies are nonempty.
The reference must be a bound permanent reference in that context. This includes
source, host, target, and supported event references; it does not introduce new
bindings. Activation costs and `IfElse` do not accept this mandatory shape.
The optional `MayCost` shape uses the choice contract below.

This RON spelling denotes the actual destruction result, as Cosmic Horror's
"destroyed this way" clause requires. It uses `DestroyWithResult`,
`DestroyedThisWayInto`, `FromNumber` and `IfPositive`, rather than a mandatory
cost transaction. Regeneration, indestructibility, absence and phasing produce
zero. Each declaration builds its own typed result binding with a position-based
name. Nested pairs and copied or cloned resolution frames retain separate state.
Existing delayed effects retain the original references. Cosmic Horror executes
its complete record; Infinite Authority still requires a captured combat subject
for `Trigger_ThatCreature` and remains rejected.

### RON selected single-card discard costs

Bare `DiscardACard` in `Activated/2` or `ActivatedWithModifiers/3`, including
nested cost `And([costs])`, lowers to a fresh `mage.DiscardCost(1)`. The
activation controller chooses one card during payment. The whole mana, tap and
discard payment must succeed before the ability enters the stack. Copies do
not pay an activation cost again.

Resolving action sequences accept `MayCost(DiscardACard)` and
`PlayerMayCost(player, DiscardACard)` immediately followed by `If/2` or
`Unless/2` with bare `CostWasPaid` or `CostWasntPaid` and a nonempty action list.
The explicit payer must be a bound typed player reference. These forms reuse
`IfPaid`/`UnlessPaid` and the same total-cost transaction. Payment and choice
occur at resolution; each adjacent pair has its own outcome. Changing or
returning the source does not replace the saved ability controller. A copied
ability offers its own payment to its own controller.

Tuple discard declarations, plural costs, filtered costs, saved discarded-card
reads, mandatory resolving discards, optional cost conjunctions and discard
alternatives are not added. Unsupported complete records still reject.
Dwarven Armorer's cost/target/first counter-choice clause and Mishra's War
Machine's upkeep/payment/damage clause have engine tests in
`TestRONChosenDiscardCostArmorerClause` and `TestRONChosenDiscardCostWarMachineClause`.
Armorer's complete choice now uses the resolving-choice adapter below. War
Machine's actual-damage-result rule remains a gap. Land's Edge
still needs a typed discarded-card result. Its shared activation permission
is supported by the adapter described below.
Other evidence: `TestRONChosenDiscardCostOptional`, `IndependentPayers`,
`ReplacementsAndInstances`, `Invalid` and `StrictBoundaries`.


## RON choices during resolution

`ChooseAnAction/1` accepts a list of at least two alternatives. Each alternative
is a list containing exactly one `PutCounters([ACounterOfTypeOnPermanent(
PTCounter(p, t), Ref_TargetPermanent)])` action. P/T coordinates use the existing
counter type grammar. The permanent target must be declared outside the choice.
Choices that add targets, use another subject, use plural counters, or run other
actions remain errors. This shape executes Dwarven Armorer's complete record.

`ChooseAnAbility/1` accepts at least two distinct keywords supported by the
existing temporary keyword adapter. It must be followed immediately by
`CreatePermanentLayerEffectUntil/3`, whose modification list contains
`AddAbilityVariable(TheChosenAbility)`. The reference is a bare symbol and is
bound only in that adjacent continuation. Other supported modifications, such
as `AdjustPT`, may share the list. Subjects and durations retain their existing
typed contracts. Fixed branches lower the selected ability without a mutable
result slot. Temporary grants and choices also accept `Rampage(n)` with a bare
nonnegative integer. Different amounts are distinct alternatives. Saved choices
outside the continuation, wrong references, duplicate options and missing
continuations remain errors. This shape executes Urza's Avenger and Gabriel
Angelfire. `HasAbility(AnyRampage)` and `DoesntHaveAbility(AnyRampage)` read current
triggered abilities through `mage.HasRampage`; Rapid Fire checks at resolution
after its first-strike grant. Rampage is not an Attr: its temporary grant uses
`mage.TriggeredAbilityModification` with the shared becomes-blocked pattern and
rampage effect. Static RON rampage grants and rampage removal remain unsupported.

Both forms use `mage.ChooseAtResolution(PlayerRef, ...ResolutionOption)`.
`ResolutionOption` has `Label string`, `Body ResolvingEffect` and optional
`When StateCondition`. The condition supplies any choice restrictions; arbitrary
action legality is not inferred. Zero available options do nothing. One option
runs without a prompt. Multiple options use the selected player's `ChooseMode`;
an invalid answer uses the first available option. Definition validation checks
all bodies, conditions and references, and prevents branch results from escaping.

Choices occur during resolution after targets and payments. The selected branch
uses the current frame, including target groups, controller, source incarnation,
event bindings and saved information. Copies make separate choices. Each factory
builds fresh effects and target bindings. Existing counters, layers, transaction,
replacement, copy and clone services own the resulting actions.

Evidence: `TestResolutionChoice*`, `TestRONResolutionChoice*`, existing
`TestRONChosenDiscardCostArmorerClause`, counter placement, temporary keyword
and next-upkeep duration contracts. Modal/spell and ordinary activation target
bindings have definition tests; the ordinary trigger has resolution and expiry
tests. These mappings do not establish collection-wide gameplay coverage.

## Mana ability activation payment plans

`ManaTap.ActivationPayment` is an optional `*ManaActivationPayment`, a fixed
array of counts indexed by color. `SolveMana` now searches all legal color
payments for each supported mana ability's activation cost. It saves the chosen
payment with that action's production choices. For example, a Celestial Prism
can spend red and green mana for its `{2}` activation and retain white mana for
a final `{W}{U}` cost. Generic, colored, hybrid and existing color conversion
payments use the same mana-pool legality rules. Spell-restricted mana cannot
pay an activation cost.

`ManaTap.SourceIncarnation` saves the discovered source's incarnation. Execution
checks that incarnation, current ability, production, controller, tap legality,
current reduced cost and saved payment before it spends mana or taps the source.
A phase cycle keeps the incarnation. A departed or returned source cannot execute
an old discovered plan. `Game.Clone` can execute the same plan independently.
Callers that omit either new field retain the previous behavior for that field.

The RON syntax and supported cost contexts do not change. Celestial Prism,
Apprentice Wizard and Fire Sprites reuse their existing full-record compilation
and the shared solver. This service is a prerequisite for repeatable non-tap
conversion abilities. Those abilities, mana bodies with activation history, and
Standing Stones' additional life cost still need planner support.

## Temporary declared triggered abilities

`TriggeredAbilityModification(PermanentRef, *GenericTriggered)
ContinuousModification` installs a declared battlefield trigger in layer 6.
Use it with `Until`, source-bound durations, saved-player upkeep durations, or
`While`. The subject uses the outer context; the trigger has an independent
permanent-source and binding scope. Spell grants cannot read the spell's source
as a permanent. Granted bodies cannot read outer targets or casting facts.
Nil declarations, unbound references, legacy event declarations and triggers
from other zones fail definition validation.

Each installation captures recipient incarnation and supplies a separate ability
object and stable ability UUID. Layer reapplication does not accumulate triggers.
The recipient's controller at detection controls the trigger. Copies of the
source effect install separate grants. Phasing suspends the grant and preserves
identity; leaving and returning ends its application. Pending triggers survive
expiry and ability removal, with their original event frame. Existing layer
ordering, durations, APNAP, target legality and cloning services apply.

Evidence: `TestTemporaryTrigger*`, `TestRONTemporaryRampage*`, existing
`TestGrantedTrigger*` and `TestRampage*`. Gabriel Angelfire's four choices and
saved-player next-upkeep duration, and Rapid Fire's casting restriction,
resolution-time rampage condition and end-of-turn grants have complete RON tests.

## Permanent-source next-event prevention prerequisite

`PreventNextDamageFromPermanent(subject PermanentRef, recipient PlayerRef,
filter PermanentFilter) ResolvingEffect` installs one shield until cleanup.
The subject must be present and pass the filter when the effect resolves.
The recipient is saved at resolution. Targets and saved references use their
existing declaration and incarnation checks.

The next matching damage event to that player is prevented in full. Current
source properties are checked again; departure or phasing uses the retained
view for that exact incarnation. A returned card cannot use or consume the old
shield. Nonmatching and zero damage do not consume it. All matching packets
from one simultaneous event are prevented with one use. Separate installations,
copied activations and cloned games keep separate consumption state.

The zero filter matches all permanent sources. Filter callbacks must read the
supplied permanent, rather than look up its UUID again. This prerequisite does
not choose a source or support spell-to-permanent source continuity. RON
`ChooseADamageSource` and its continuation remain rejected. The existing UUID
`Game.AddSourcePrevention` installer retains its caller behavior.

### Engine planning prerequisite: repeatable fixed mana conversion

The existing `Activate(Costs(ManaPayment(...)), AddMana(...))` engine API can
participate in automatic payment without a tap cost when the body is fixed and
mana-only (or includes the deferred history shape below), and the reduced cost
consumes at least its produced mana count.
The source may be tapped or summoning-sick. Automatic planning excludes X costs,
activation limits, arbitrary activation conditions, targets, non-mana costs and
mixed bodies except history-gated delayed instructions. Explicit activation
retains the existing broader behavior.
No tap bonus or tapped-for-mana event is generated for a conversion.

RON vocabulary is unchanged. History-bearing fixed-color conversions now use
the synchronous execution contract below.
`TestRepeatableMana*` and `TestRONRepeatableManaPayment` establish this engine
prerequisite; they do not establish full Farrelite Priest or Initiates of the
Ebon Hand gameplay support.

### Fixed mana production with activation history

RON supports nonmodal `Activated` bodies with an `ActionList` containing fixed
`AddMana(ManaProduceW|ManaProduceU|ManaProduceB|ManaProduceR|ManaProduceG|ManaProduceC)`
actions and `If(TimesThisAbilityHasBeenActivatedThisTurnIs(
GreaterThanOrEqualTo(Integer(n))), [CreateFutureTrigger(event, ActionList(...))])`.
The threshold is nonnegative. Each branch contains one or more supported delayed
step instructions. Existing step-event and delayed-body restrictions apply.
Immediate branch actions, conditional production, any-color production, extra
immediate actions and history reads inside delayed bodies are rejected.

Farrelite Priest and Initiates of the Ebon Hand each pay `{1}`, produce one fixed
mana, and create a next-end-step sacrifice delay at four or more activations.
Explicit activation and automatic payment share the same per-instance history.
Each successful activation resolves immediately without entering the stack.
The planner admits these bodies under its existing cost and timing restrictions,
reads fixed output without executing effects, then executes the complete body
through normal immediate resolution after payment. No new public API is needed.
`TestHistoryMana*` and `TestRONHistoryMana*` cover payment, event timing, thresholds,
instance/clone isolation, cleanup, departure, control, phasing and delayed timing.

## RON attached permanent death bindings

`TriggerA(WhenAPermanentDies(SinglePermanent(HostPermanent)), body)` uses
`mage.AttachedPermanentDies()`. The source must be an Aura. The event matches
its attached incarnation moving from the battlefield to a graveyard, including
simultaneous Aura/host departure. It does not require the host to be a creature.
`mage.AttachedCreatureDies()` retains its existing creature restriction.

Each factory builds separate typed bindings. Within this trigger body,
`ControllerOfDeadPermanent` and `OwnerOfDeadPermanent` are saved player references,
`ToughnessOfDeadPermanent` is a saved number, `Trigger_ThatPermanent` is the
original departing permanent, and `Trigger_ThatGraveyardCard` is its first
graveyard visit. Callable symbols and reads outside this body are compile errors.
Existing player actions, permanent-source damage and graveyard-card actions can
read these bindings. Source-death and general filtered death contexts do not
receive the new host bindings.

`DeadGuestPermanentDealsDamage(value, recipient)` is a resolving action in this
host-death body. It lowers to `DealDamageFrom(DamageSourceOf(This), recipient,
value)`: the Aura deals the damage. The amount uses supported instance values;
the recipient uses the existing typed singular damage-recipient declarations.
It declares no target of its own. Values from the host death are captured at
detection, not recalculated from a returned card or a later attachment.

Creature Bond has complete RON gameplay evidence in `TestRONHostDeath*`.
Puppet Master and Takklemaggot can use the event bindings but retain unsupported
payment, choice, return and Aura-change instructions. Their complete records
still fail strict compilation.

## RON fixed matching tap payments

`TapAPermanent(filter)` and `TapNumberPermanents(Integer(n), filter)` are fixed
activation costs; `n` must be positive. They use
`mage.TapNMatchingIncludingSourceCost(n, filter, text) Cost`. This constructor
uses the existing exact-count payment transaction and permits the source when
it matches. `TapNOtherMatchingCost` keeps its source exclusion.

The payer chooses distinct untapped matching permanents they control. Each
choice retains its incarnation. A selected permanent can have summoning sickness;
a separate `TapPermanent(ThisPermanent)` cost still checks the tap symbol rule.
All selected permanents tap before their tap events are sent. Failed payments
leave mana, life and taps unchanged. Queries do not make choices.

Filters can use the existing fixed permanent filters and nonempty conjunctions.
A conjunction can contain one `ControlledByAPlayer(SinglePlayer(You))`; the engine
already checks the payer's control. Other players, unbound references, duplicate
controller clauses, or controller clauses inside disjunctions are rejected.
These adapters do not provide paid-object history or resolution tap payments.
Hand of Justice has complete RON gameplay tests. Vodalian War Machine's two
activation clauses have tests, but its death rule still needs paid-object history.


## RON activation permission for any player

`ActivatedWithModifiers(cost, actions, OtherPlayersMayActivate(AnyPlayer))`
permits either player to activate the ability. This modifier also works inside
nested `And([...])` modifier lists and with `Modal_ChooseOne` action bodies.
It lowers to `mage.WithAnyPlayerMay()` on each fresh action definition.
The compiler requires one bare `AnyPlayer` symbol and rejects duplicate
permissions, including duplicates in nested modifier lists. Other player
filters, top-level rules, action/cost uses and the
separate conditional-mana activation context remain unsupported.

The activator pays, chooses modes and targets, controls the stack ability and
supplies resolving `You`. Target controller restrictions are checked against
that player at announcement and resolution. Source characteristics and damage
ownership use the original source's current or retained information, rather
than the stack ability's controller. Permission does not bypass payment,
phasing, activation limits, timing or target legality.

`TestRONActivationPermissionEfreet` proves the complete Ifh-Bíff Efreet ARN
record, native casting, both activators, payment rollback, flying recipients,
shroud, phasing and simultaneous damage. `Identity` covers source control,
departure/return, phase cycles, ability removal, lifelink, copies and clones.
`ActorsAndTiming`, `ModalChoices`, `IndependentInstances`, `CostRollback`,
`TargetsAndPresence`, `EfreetPreventionAndMembership`, `Invalid` and
`StrictLoad` cover the remaining permission contracts and atomic loading.
Land's Edge shares the supported permission and discard cost, but its full
record still rejects `DiscardedCardPassesFilter`. It has no complete gameplay
mapping from this slice.

### RON grouped source combat events

`TriggerA` supports this exact event union, in either order:

```text
Or([
  WhenACreatureBlocksAnyNumberOfCreatures(SinglePermanent(ThisPermanent), IsCreatureType(S)),
  WhenACreatureBecomesBlockedByAnyNumberOfCreatures(SinglePermanent(ThisPermanent), IsCreatureType(S))
])
```

Both entries must use the same source and creature subtype. The compiler lowers
this union to `BlocksOrBecomesBlockedBy(This).OfSubtype(S)`. One matching new
group creates one trigger, even when multiple members match. These forms do not
support a standalone direction, an Aura host, or a singular opponent binding.

`WhenACreatureBecomesBlocked(SinglePermanent(ThisPermanent))` lowers to
`BecomesBlocked(This)`. It fires on the source's first transition to blocked,
including a transition caused by entry of a blocker. Later blockers do not
repeat this event. Both adapters permit `Trigger_ThatCreature` as the original
source in the resolving body. They do not supply `Trigger_DefendingPlayer` or
an opposing-creature reference. Each factory builds fresh actions and abilities.

Dwarven Soldier has complete RON gameplay coverage in `TestRONGroupedCombat*`.
Spitting Slug still rejects its unsupported `MayCost`/`IfElse` continuation;
this event adapter does not establish payment branches or collection grants.

## RON two-branch fixed mana payments

In resolving action sequences, adjacent `MayCost(PayMana([...]))` or
`PlayerMayCost(player, PayMana([...]))` and
`IfElse(condition, [true_actions], [false_actions])` form one payment operation.
The condition is the bare symbol `CostWasPaid` or `CostWasntPaid`. Both lists
must contain at least one supported action. Each branch can contain another
payment pair. The result cannot be read outside the pair. Fixed costs include
an explicit zero cost; X, mandatory payments and nonmana costs in this
three-argument continuation remain errors. Existing one-branch forms retain
their contracts. The adapter lowers to `mage.PayOrElse` with typed player and
body references, built separately for each factory instance.

`CreateEachPermanentLayerEffectUntil(collection, [AddAbility([keywords])],
UntilEndOfTurn)` supports nonempty lists of supported temporary keywords.
The collection uses the contextual collection grammar, including nonempty
`And`/`Or` and typed `IsBlockingAttacker(permanent)` and
`IsBlockedByDefender(permanent)`. These relations use current combat pairs.
An absent, phased or returned source cannot restore a pair through LKI.
Membership is captured when the grant begins. Leaving combat does not end it;
a recipient's zone change ends it, and cleanup removes it. Mixed keyword/P/T
modifications and other durations remain errors in this shape.

`TestRONPaymentBranches*` maps complete Spitting Slug RON to casting, both combat
triggers, multiple blockers, payment, first-strike damage, duration, identity,
phasing, copies and cloning. Synthetic tests cover nested results, polarity,
target legality, copy retargeting and invalid declarations. This does not claim
complete Cyclone, Imprison, Bronze Tablet or Transmute Artifact support.

## RON activated ability targets

`Targeted([TargetAbility(And([ActivatedAbility,
AbilityOfASource(IsCardtype(Artifact))]))], ActionList([CounterAbility(Ref_TargetAbility)]))`
declares one activated ability target. The two filter operands can appear in
either order. Exactly one bare `ActivatedAbility` and one
`AbilityOfASource(IsCardtype(type))` are supported. The source type uses the
existing card-type grammar. Other ability/source filters remain errors.

`ControllerOfAbility(Ref_TargetAbility)` is a typed player reference. It reads
the controller of the ability on the stack, independently of source control.
It can supply the payer of `PlayerMayCost`; the existing adjacent `Unless` or
`If` payment continuation selects its branch at resolution. `CounterAbility`
requires a bound ability reference in the same rule. Spell, permanent, delayed,
undeclared and cross-rule references are rejected. Factories build fresh target
bindings. Stack objects store their own choices; copies can select other targets.

The engine API is `DeclareActivatedAbilityTarget(name string, sourceType
core.CardType) *AbilityTargetBinding`, `ControllerOfAbility(AbilityRef) PlayerRef`,
and `CounterAbility(AbilityRef) ResolvingEffect`. The compiler imports these
services directly from `mage`; no legacy DSL export is required. Rust and Ayesha
Tanaka use their complete LEG RON records. Trigger-event ability references,
including Imprison's `Trigger_ThatActivatedAbility`, remain unsupported.

## RON source departure events and live exile collections

`TriggerA(WhenAPermanentLeavesTheBattlefield(SinglePermanent(ThisPermanent)), body)`
is supported on permanent definitions. The event and `SinglePermanent` each take
one argument. The source must be the bare `ThisPermanent` symbol. Other subject
filters and Aura-host events remain unsupported. The compiler uses
`WhenEvent(LeavesBattlefield(This), ...)` and creates a fresh ability per factory.
Within this body, `Trigger_ThatPermanent` is a typed reference to the original
source. It has the same incarnation and LKI behavior as `ThisPermanent`; it is
unbound in a separate rule. It supplies no card reference in the destination zone.

`Exile([EachPermanent(collection)], [])` supports the shared contextual collection
compiler in resolving actions. This includes permanent filters, nonempty `And`
and `Or`, controller/owner selections, combat-pair collections and attachments.
A declared `Ref_TargetPermanents` retains its typed target set. Unbound target
sets, malformed filters and exile modifiers are rejected. `ExileEach` captures
live collection members at resolution and uses one zone-change batch. This action
has no target unless the surrounding action declares one.

Complete-card mappings: `TestRONSourceLeaveShrine*` covers Goblin Shrine's
conditional basic-Mountain bonus and departure damage. `TestRONSourceLeaveHazezon*`
covers Hazezon Tamar's entry trigger, next-upkeep tokens and departure exile.
`TestRONSourceLeaveCasting`, `IndependentEvents`, `Invalid`, `Inventory` and
`StrictLoad` cover payment, declarations, independent pending events, role
reporting, fresh factories and atomic rejection. Existing `TestExileEach*` tests
supply the reusable batch, identity, phasing and token contracts.

## RON ordinary draw-step triggers

Permanent definitions support
`TriggerA(AtTheBeginningOfAPlayersDrawStep(players), actions)` with exactly one
event argument: bare `AnyPlayer` or `SinglePlayer(You)`. The existing action
compiler supplies the body; this declaration does not add missing body operations.
`Trigger_ThatPlayer` is a typed player reference local to that body. Each factory
creates a separate binding, and each pending event saves its player. `You` reads
the resolving ability controller. The controller filter reads the source at
detection; it is not checked again at resolution.

The adapter reuses `compileBoundStepAbility`, `TurnStepBegins(EachDrawStep, binding)`
and `EventPlayerIsController` for the `SinglePlayer(You)` restriction. It has no
intervening-if condition. Other players, malformed declarations, cross-rule
reads and instant/sorcery rule contexts fail compilation. Detection precedes the
normal draw; priority and trigger resolution follow it. Phased sources cannot
trigger, but departure or phasing after detection does not cancel a pending rule.
Copies retain the event player and use their own controller; clones keep separate
resolution state.

`TestRONOrdinaryDrawStep*` covers declarations, bindings, copies, clones, source
lifecycle and native timing. Armageddon Clock's unchanged RON event is tested
with an explicit probe body. Its complete record still rejects the source-counter
value. Sylvan Library still rejects optional drawing payment and has a separate
source-data defect in its drawn-card disposition. Neither has a complete-card
mapping from this prerequisite. Existing `TestRONDrawStep*` mappings for Howling
Mine and Mana Vault remain valid.

## RON counter information values

`TheTotalNumberOfCountersOfTypeAmongPermanents(counter, SinglePermanent(ref))`
is supported by the resolving value compiler. `counter` uses the validated
permanent counter vocabulary. `ref` is a typed source, Aura host, singular target,
or permanent supplied by the current event. A reference must be bound in that
rule and context. Collections and static, entry, cost and announced-X contexts
are not established by this adapter.

The adapter uses `mage.CountersOnWithLKI` with each factory's own bindings.
It reads current counters at execution or retained information for the exact
incarnation after departure or phasing. It cannot restore an absent target's
legality. A prior instruction can move a legal target and a later information
read can still use its retained counters. A sacrifice cost uses departure
information. No mutable count is stored in the compiled plan.

Complete-card evidence: Armageddon Clock, in ATQ and 3ED. Tests with prefix
`TestRONCounterInformation` cover its casting, upkeep placement, any-player
upkeep activation, payment rollback, countered payment, removal, draw timing,
simultaneous damage, prevention, copies, clones and incarnation changes.
Other consumers remain rejected at independent gaps. See
[the counter information contract](engine-rules-contracts.md#counter-information-values).

## RON restricted activation players

`OnlyPlayerMayActivate(OwnerOfPermanent(ref))` and
`OnlyPlayerMayActivate(ControllerOfPermanent(ref))` are activation modifiers.
`ref` must be the bare `ThisPermanent` symbol or, for an Aura,
`HostPermanent`. They work in ordinary and modal `ActivatedWithModifiers`
abilities and nested modifier `And` lists. Exactly one permission modifier is
permitted, including when combined with `OtherPlayersMayActivate`.
Announcement targets and event bindings cannot supply the permission subject.
Other player references, action/cost/rule uses and conditional-mana activations
remain unsupported. This does not add resolving `OwnerOfPermanent` support.

Lowering combines `WithAnyPlayerMay()` and `WithActivationStateCondition()`
with `PlayerIsPermanentOwner(PermanentRef) StateCondition` or
`PlayerIsPermanentController(PermanentRef) StateCondition`. These engine
conditions accept `This` and `AttachedPermanent`, compare
`StateContext.ControllerID`, and require the current source and attachment
incarnations. They use live information; absent or phased objects cannot grant
permission. Control and attachment changes affect future activations.

The eligible player pays costs, chooses modes and targets, controls the stack
ability, and supplies resolving `You`. Timing, limits and target legality still
apply. Permission is not repeated during resolution or for a stack copy.
No mutable choice is stored in the condition. Engine declaration validation
rejects nil, target and saved-binding subjects before gameplay.

Evidence: `TestActivationPlayerConditionIdentity` and `Declarations`;
`TestRONOnlyPlayerActivationRealModifierProbes`, `PresenceAndReattachment`,
`TimingTargetsAndInstances`, `RollbackAndModes`, `Inventory`, and
`InvalidAndBoundaries`. The probes retain the unchanged permission nodes from
Merseine and Personal Incarnation, with explicitly synthetic costs and bodies.
Neither complete card is supported: Merseine still needs
`PayManaCostOfPermanent(HostPermanent)` and Personal Incarnation still needs
`CreateFutureReplaceWouldDealDamage` and its independent death body.

### Counted and collection next-untap actions

Use `mage.SkipNextUntaps(subject, count)` for a fixed nonnegative number of
future untap steps of the subject's current controller. Use
`mage.SkipNextUntapEach(subjects)` to capture a typed collection at resolution
and suppress each member's next controller untap. Neither action taps subjects
or prevents untaps from other effects. Repeated restrictions overlap; they do
not add steps. Control changes and phasing preserve remaining steps. Departure
ends the restriction, and clones keep separate counts.

RON supports resolving
`PermanentDoesntUntapDuringControllersNextNumberUntaps(ref, Integer(n))` and
`EachPermanentDoesntUntapDuringControllersNextUntap(filter)`. Counts must be
nonnegative literals. References use the existing instance scope; filters use
contextual collection queries. These are actions, not costs or static rules.
Complete gameplay mappings cover Telekinesis and Spore Cloud. See the counted
untap contract in `engine-rules-contracts.md` for timing and identity evidence.

## RON resolution counter payments

Resolving permanent-source action lists accept `MayCost`, `PlayerMayCost`
and `MustCost` with `RemoveCounters([instruction])`, followed immediately by
`If` or `Unless` with `CostWasPaid` or `CostWasntPaid` and a nonempty action
list. Fixed instructions are `ACounterOfTypeFromPermanent(counter,
ThisPermanent)` and `NumberCountersOfTypeFromPermanent(Integer(n), counter,
ThisPermanent)` with nonnegative n. Optional payments also accept
`AnyNumberOfCountersOfTypeFromPermanent(counter, ThisPermanent)`.

A chosen payment's paid continuation can read the symbol
`TheNumberOfCountersRemovedThisWay`. The compiler makes a fresh NumberBinding
per factory instance and payment. The value is unavailable before payment,
after the continuation, or on its unpaid path. Fixed resolution payments do
not produce this value. Spell sources, other subjects, mandatory chosen counts,
multiple instructions, dynamic fixed counts and IfElse continuations remain
rejected. Existing activation counter payments retain their contract.

Lowering uses `PayCountersOrElse` or `PayChosenCountersOrElse` and the engine's
shared transaction. The payment occurs at resolution, with presence, phasing
and original-incarnation checks. It cannot use LKI to remove counters from an
absent object. Copies pay separately; clones keep separate state.

Living Artifact's six complete reprints compile and pass definition validation.
Its gameplay mapping covers enchant/casting costs and legality, actual damage
counter accumulation, controller upkeep timing, decline, insufficient counters,
source identity, control, phasing, copied triggers and clones. Cocoon and
Tetravus still fail strict compilation at their remaining rule adapters.
Synthetic mandatory and token-count tests establish the payment prerequisite;
they do not establish either card's complete gameplay. Evidence:
`TestRONResolutionCounter*` and `TestResolutionCounterPayment*`.

## RON untargeted activated ability grants

A static `PermanentLayerEffect(subject, [AddAbility([Activated(cost, body)])])`
can grant to `ThisPermanent` or an Aura's `HostPermanent`.
`EachPermanentLayerEffect(filter, [AddAbility([Activated(cost, body)])])` uses
an existing static permanent collection. A resolving
`CreatePermanentLayerEffect(reference, [AddAbility([Activated(cost, body)])])`
captures one typed permanent for an indefinite grant.

Each form requires exactly one modification and one `Activated` declaration.
The cost uses existing fixed source-cost adapters; X costs are rejected.
The body must be a nonempty, untargeted `ActionList` in a fresh permanent-source
scope. `ThisPermanent` and `You` refer to the recipient and activating player.
Donor targets, attachment references, events and payment results cannot enter
that scope. Targeted bodies, modes and `ActivatedWithModifiers` in the granted
declaration remain unsupported. The outer Life Matrix activation can still
have its own target and upkeep restriction.

The adapter uses `GrantActivatedAbility`, `GrantEachActivatedAbility` and
`ActivatedAbilityModification`. Each factory creates its own costs and body.
Static grants follow current membership; resolving grants keep captured
incarnations. Zombie Master and Life Matrix have complete RON test mappings
in `activated_grant_test.go`. Equinox still needs a targeted grant and a
spell-outcome condition.

## RON static attack payments

`PermanentRuleEffect(subject, [CantAttackUnlessCost(cost)])` supports
`ThisPermanent` on a permanent and `HostPermanent` on an Aura. The operation
requires exactly one `PayMana(list)` with fixed mana symbols or
`SacrificeNumberPermanents(Integer(n), filter)` with positive fixed `n` and an
existing sacrifice filter. X, combined costs, target references, other cost
forms and multiple operations remain compile errors in this context.

Each factory creates a fresh cost. The adapter uses `SourceCantAttackUnlessPays`
or `AttachedCantAttackUnlessPays`; payment uses `Game.PayAttackCosts` during
attack declaration. The attacker controller pays, even when an Aura has another
controller. Multiple live rules contribute separate costs to one transaction.
The rule follows current attachment, source abilities and phasing. It does not
use retained information to tax a former host. Casting and resolving the card
do not pay the attack cost. Leviathan's optional upkeep payment is separate.

Complete Brainwash and Leviathan records are mapped by `TestRONAttackCost*`:
casting and rollback, entry and upkeep, native attack choices, trample damage,
control, attachment changes, phase cycles, returned objects, copies, independent
clones, aggregate payment and invalid or stale sacrifice choices. Inventory
visits the operation's operand as a cost, including rejected descendants.

Attack payment may sacrifice the attacker when it matches the cost filter.
Both declaration paths require the paid source's original incarnation and
current control before they put it in combat. An animated Leviathan that is
also an Island can pay with itself; it then remains absent from combat.

### RON delayed departure actions through end of turn

`CreateTriggerUntil(WhenAPermanentLeavesTheBattlefield(SinglePermanent(ref)),
ActionList(actions), UntilEndOfTurn)` and the corresponding
`WhenAPermanentDies(SinglePermanent(ref))` form lower to `WhenLeavesThisTurn`
and `WhenDiesThisTurn`. They use resolving spell, activation, trigger and nested
supported delayed action scopes. `ref` must be a bound singular permanent
reference. The body uses the existing delayed action restrictions, captures the
outer bindings, and declares no new targets or departure-event bindings.

The duration is the bare `UntilEndOfTurn` symbol. Other events, durations,
plural subjects, wrong arities, unbound reads, empty bodies, modes and new body
target declarations remain compile errors. This shape does not add generic
`CreateTriggerUntil` event support. War Barge and Sandals of Abdallah use the
complete records, with their separate regeneration restrictions.

## RON hand and continuous-control activation restrictions

`ActivateOnlyIf(PermanentPassesFilter(subject,
ControlledSinceBeginningOfMostRecentTurn))` accepts a bare predicate with
`ThisPermanent` or an Aura's `HostPermanent`. It uses
`mage.ControlledSinceTurnStart(subject)`. The subject must remain present,
phased in, and under the activating player's control. An opponent's turn or
extra turn cannot grant permission before the controller's next turn begins.
Control changes and zone changes restart the requirement; phasing preserves
control history. Resolving and static filters do not accept this predicate.

`ActivateOnlyIf(PlayerPassesFilter(You,
NumCardsInHandIs(EqualTo(Integer(n)))))` accepts a nonnegative literal `n`.
It uses `mage.ControllerHandSizeIs(n)` before payment. The activating player's
current hand supplies the count, including when another player owns the source.
Other comparisons, dynamic counts, and other player references remain errors.
Both restrictions combine with existing modifiers and use the shared pure check
for mana planning. Stack copies and resolution do not repeat the check.

`TestRONActivationHistory*` maps the complete Library of Alexandria and Rocket
Launcher records to gameplay and declaration tests. Casting, land play, tap/mana
payment, target legality, delayed destruction, copies, clones, source return,
phasing, and current controller behavior use their existing engine services.

## Spell damage-source identity prerequisite

The engine APIs are:

```go
func (g *Game) CaptureSpellDamageSource(stackID uuid.UUID) DamageSource
func DamageSourceOfSpell(spell StackObjectRef) DamageSourceRef
func SnapshotDamageSource(source DamageSourceRef, binding *DamageSourceBinding) ResolvingEffect
func (s DamageSource) StackID() uuid.UUID
func (s DamageSource) SameObject(other DamageSource) bool
```

Capture uses a live stack object ID, including permanent spells, and rejects
abilities or absent spells. `DamageSourceOfSpell` accepts an existing typed spell
target or saved spell binding. Snapshot saves independent information in the
existing resolution environment; `FromDamageSource` can read it after departure.
The normal definition validator checks declarations and producer dominance.
An unsuccessful snapshot replaces the binding with an empty source.

Native spell damage retains its exact stack ID while resolving. A saved source
keeps identity, controller, owner, effective colors and card types through
departure and cloning. `SameObject` distinguishes separate casts, copied spells
and permanent incarnations. It does not link a permanent spell to the permanent
it becomes; that future prevention contract needs an explicit link. A source
without a stack ID or permanent incarnation cannot match by UUID alone.

These APIs are engine services. No new RON shape is accepted. In particular,
`ChooseADamageSource` remains rejected until enumeration, source bindings,
property rechecks and shield installation are complete. Existing callers and
the permanent-source prevention API remain available.


## RON attack history references

`AttackedThisTurn` and `DidntAttackThisTurn` are bare permanent-filter symbols.
They are supported in existing permanent-filter contexts, including target
filters and battlefield collections, and in resolving
`If(PermanentPassesFilter(reference, filter), [actions])` branches. Nonempty
`And`/`Or` branches retain the existing scope rules. A delayed branch keeps its
original permanent reference through `AtNext`; it reads history when the delayed
body resolves. No new intervening-if, static-condition or player-collection
filter context is added.

`TargetPlayer(AttackedThisTurn)` declares one player target. It lowers to
`DeclarePlayerTarget(name, TargetPlayerAttackedThisTurn())`; either player can
qualify. The engine checks the restriction at announcement and resolution.
Tuple forms, arguments, `TargetPlayer(DidntAttackThisTurn)` and unbound or
incompatible permanent references remain declaration errors.

The engine filters `AttackedThisTurn` and `DidntAttackThisTurn` read declaration
history for an exact permanent incarnation. `IfPermanentMatches` can read that
history through current or retained information. Absence does not permit an
action on that object. A returned card has separate history, even when both
incarnations attack in the same turn. Phasing and combat exit do not erase
history. A creature put into combat without declaration did not attack.

`TargetPlayerAttackedThisTurn()` composes with `TargetOpponentOnly()` in either
order. It uses the existing player target and turn tracker, without a separate
choice service. Targets and delayed bindings remain private to each factory;
copies and cloned games retain their existing identity contracts.

Evidence: `TestAttackHistoryReferenceIdentity`,
`TestAttackHistoryReferencePhasingAndCombat`, `TestAttackHistoryPlayerTarget`,
`TestAttackHistoryBothDeclarationPaths`, `TestRONAttackHistoryBerserk`,
`TestRONAttackHistoryFireAndBrimstone`, `TestRONAttackHistoryInvalidAndInventory`,
`TestRONAttackHistoryBerserkTimingAndLegality`,
`TestRONAttackHistoryBerserkCopyAndExpiry`,
`TestRONAttackHistoryFireAndBrimstoneBatch`, and
`TestRONAttackHistoryFilterContextsAndInstances`.
Erg Raiders still needs its source intervening-if and continuous-control
conditions. Season of the Witch still needs its inability-to-attack history
and exclusion filter. Their full records remain rejected.

## RON filtered compulsory blockers

RON supports
`PermanentRuleEffect(ThisPermanent, [AllCreaturesMustBlockIt(filter)])` and,
for an Aura with an enchant declaration,
`PermanentRuleEffect(HostPermanent, [AllCreaturesMustBlockIt(filter)])`.
The operation has exactly one permanent-filter argument. It uses the existing
filter compiler, including supported nonempty conjunctions/disjunctions.
Unbound targets, non-Aura hosts, malformed operations and incompatible filters
remain declaration errors. No temporary, collection, cost or entry context is
added by this adapter.

The compiler lowers the operation to
`Static(While(condition, AllCreaturesMustBlock(subject, filter)))`.
The engine owns live attachment identity, rule storage and the maximum legal
blocking assignment. It uses both native and search declaration paths and does
not redirect every block to the first compulsory attacker. Lure requires all
able creatures to block its current host. Marble Priest requires only able
Walls and retains its separate combat-damage prevention rule.

For Go engine consumers, `AllCreaturesMustBlock(PermanentRef, PermanentFilter)`
can also use the existing `Until` duration contract. The pure
`Game.CompleteBlockAssignments(uuid.UUID, []BlockAssignment) []BlockAssignment`
query keeps legal maximum proposals and completes unmet requirements without
writing combat state. The engine contract documents capacity, bands, minimum
blocker counts, identity and search behavior. Evidence is in
`TestBlockingRequirement*` and `TestRONBlockingRequirement*`.


## RON temporary collection animation and equal base values

`CreateEachPermanentLayerEffectUntil(collection,
[SetPT(PT(integer, integer)), AddCardtype(Creature)], UntilEndOfTurn)` accepts
exactly one fixed P/T pair and one Creature addition, in either order. Collections
use the existing contextual collection compiler, including typed controller,
intersection and union queries. Both operations share one `Until` installation.
It captures current members and their incarnations at resolution; later entries
and phased-out members do not join. Prior types, subtypes, colors and abilities
remain. This is the complete Thelonite Druid record, including its transactional
mana, tap and creature-sacrifice costs; the source can pay its own sacrifice.

In `CreatePermanentLayerEffectUntil/3`,
`SetCardtypes([Artifact, Creature])` accepts exactly those two types once each,
in either order. It lowers to two type additions because becoming an artifact
creature retains prior types under CR 205.1b. Other type-setting lists remain
errors; this does not establish general type replacement.
`SetPowerAndToughnessBoth(value)` lowers to the existing base-value operation
with the temporary-value compiler. Its supported values are fixed integers,
bound announced X, subtraction of supported values, and typed permanent
power/toughness or mana-value reads. `ActionPermanent` means this modification's
subject. A mana-value read must refer to that subject. The engine captures values
at resolution, rather than recalculating them on each layer pass.

Xenic Poltergeist uses these operations with the existing
`UntilNextUpkeep(You)` duration. Its noncreature-artifact target is checked at
announcement and resolution; the saved caster/controller selects the expiry
upkeep. Source departure does not end the installed effect. These additions
apply only to resolving temporary modifications, not static, entry or cost
contexts. Invalid references and declarations fail compilation.

Evidence: `TestRONExtendedAnimationCards`, `XenicLifetime`, `DruidMembership`,
`Copy`, `Invalid` and `StrictLoad` in `extended_animation_test.go`.

## RON numbered singular permanent groups

`Targeted([TargetPermanent(filter1), TargetPermanent(filter2), ...], actions)`
accepts two or more ordinary singular permanent declarations. The resolving body
uses `Ref_TargetPermanent1`, `Ref_TargetPermanent2`, and subsequent one-based
names. Each declaration has a separate typed plan and each factory call builds
separate engine target bindings. Single declarations retain the unnumbered
`Ref_TargetPermanent` spelling. A multiple-declaration body cannot use that alias.

This shape works in existing spell, activation, triggered and mode-local action
contexts. Existing delayed bodies capture each numbered reference separately.
Filters reuse the singular permanent target compiler. Each group requires one
legal permanent at announcement. A permanent can be chosen once in each separate
group when it meets both filters. The normal target service checks each group
again at resolution. Legal groups still act when another group is illegal; all
illegal groups prevent resolution. An action cannot act on an illegal target.
Copies can retarget each group, and clones keep their own resolving frames.
Delayed actions retain the captured incarnation and existing information reads.

Empty lists, mixed declaration kinds, plural groups, opponent-choice declarations,
unbound or malformed numbered references, mode-scope leaks, nested declarations,
and conflicting `You`/`Opponent` target controller restrictions are rejected.
This adds no dependent target filter, exchange, fight or graveyard group adapter.
Gauntlets of Chaos and Glyph of Delusion now reach their dependent-filter gaps;
Arena, Cuombajj Witches and Drafna's Restoration still need other target forms.
No complete-card gameplay mapping is added by this prerequisite.

Evidence: `TestRONSingularGroupsGameplay`, `InstancesAndRepeatedObject`,
`CopyRetarget`, `DelayedLKI`, `TriggeredAndModal`, `Invalid`, and
`StrictLoadAndConsumers`. Execution reuses `DeclarePermanentTarget`, `WithTargets`,
existing target legality, payment transactions, saved references and stack copies.

## RON opponent-chosen permanent targets

`TargetPermanentOfAPlayersChoice(filter, Opponent)` and
`TargetPermanentOfAPlayersChoiceTheyControl(filter, Opponent)` declare one
permanent target in `Targeted`. The second form requires the opponent to control
that permanent. The first form permits any matching permanent. The chooser
argument must be the bare `Opponent` symbol. Other chooser references and
`TargetPermanentOfPlayersChoice` remain unsupported.

These forms can occur alone or in numbered singular permanent groups with
ordinary `TargetPermanent` declarations. They use the existing target filters,
scopes, spell, activation, trigger and mode contexts. A single declaration uses
`Ref_TargetPermanent`; multiple declarations use one-based numbered references.
Conflicting controller restrictions fail compilation before factory construction.
Factories create separate typed target bindings for each declaration.

Lowering uses `DeclarePermanentTarget`, `TargetChosenByOpponent` and, for
`TheyControl`, `TargetOpponentControls`. The opponent chooses at announcement
or trigger placement. Target legality reads the action's controller and source;
the chooser does not bypass hexproof or other targeting restrictions. Targets
are checked again at resolution. Modal activation preserves the initial target
choice. Copy permissions use the existing copy service: the copy's controller
can choose new legal targets under CR 707.10c.

Evidence: `TestRONOpponentTargets*` and
`TestModalActivationOpponentChoosesOnce`. Preacher, Arena and Nova Pentacle have
source-record target-fragment tests and explicit next-blocker checks. Those
fragments establish targeting and original activation-cost behavior. They do
not establish complete-card gameplay or permit partial production loading.


## RON reference collections for resolving actions

`SinglePermanent(ref)` is a typed contextual collection operand. Symbolic
`Ref_TargetPermanents` uses its scoped numbered target group. Both can occur in
nonempty `And([...])` and `Or([...])` compositions with existing live filters,
controller/owner sets, attachments and combat relations. The action consumers
are `TapEachPermanent`, `UntapEachPermanent`, `DestroyEachPermanent`,
`DestroyEachPermanentNoRegen` and `PutEachPermanentIntoItsOwnersHand`.

These forms use `mage.SingletonPermanent(PermanentRef) PermanentSet`, existing
target sets, `IntersectPermanents` and `UnionPermanents`. The reference keeps its
identity and legality checks; collection selection does not add targets. Each
factory builds separate bindings. Action selection excludes absent and phased
members and uses existing batch services. Existing saved attachment and singular
delayed capture behavior remains in effect. Plural delayed target collections,
unbound or incompatible references and empty compositions reject. These forms
do not add support in costs, entry operations, static filters or damage sets.

Evidence: `TestSingletonPermanentSet*`, `TestRONReferenceCollection*`, and the
existing full RON tap/untap and contextual collection gameplay tests. Arena still
needs Fight; Volcanic Eruption still needs its actual destruction-result count.

### RON destruction counts

A resolving action sequence can pair `DestroyEachPermanent(selection)/1` or
`DestroyEachPermanentNoRegen(selection)/1` with its next action. That action can
read `TheNumberOfPermanentsThatDiedThisWay(IsCardtype(Creature))/1` or
`TheNumberOfPermanentsPutIntoAPlayersGraveyardThisWay(filter, AnyPlayer)/2`.
The creature count uses the characteristics before destruction. The graveyard
count requires the selection to guarantee the requested filter: the same
position-free declaration, or an operand of its nonempty `And` tree. A bound
`Ref_TargetPermanents` uses its declared target filter for this check. It retains
all target legality and incarnation checks. This proof permits the existing
engine count of all selected graveyard moves without ignoring a filter.

Each pair creates a fresh `NumberBinding`. Only the adjacent continuation can
read it. Reads before the producer, later actions, delayed bodies, mixed result
kinds, other death filters and restricted graveyard players are rejected. Each
factory instance has separate bindings. `Plus(left, right)/2` now composes two
supported resolution values with `AddValues`; announcement and static value
contexts do not gain this adapter.

Volcanic Eruption and Hellfire execute their complete collection records through
`DestroyEach`, `PermanentsPutIntoGraveyardThisWayInto`,
`CreaturesDiedThisWayInto`, `FromNumber` and the existing damage services.
No engine action or global result slot is added.

## Damage-source choice prerequisite

`mage.ChooseDamageSourceFrom(player, []mage.DamageSourceRef{...}, binding)`
saves a required non-targeted choice in `mage.BindDamageSource`. Use
`mage.FromDamageSource(binding)` in the subsequent resolving instructions.
Each reference supplies its own identity and information contract. The declared
list must be nonempty and contain only sources permitted by the calling rule.
An empty runtime list clears the binding. The operation copies its input slice,
keeps distinct spell stack objects, and removes exact duplicate sources.

Players can implement `mage.DamageSourceChooser` to choose among typed source
snapshots. Existing players use numbered options. Choice values remain in the
existing resolution frame and survive game cloning without sharing mutable
snapshot data.

RON `ChooseADamageSource` remains unsupported. Its adapter needs complete
CR 609.7a enumeration, property filters, lexical bindings and the consuming
shield/replacement operations. This prerequisite does not change the production
loader, accepted records, or existing UUID-based callers.

## Typed damage-source property filters

`ChooseDamageSourceMatchingFrom(player PlayerRef, sources []DamageSourceRef,
filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect`
filters an explicit legal list before the existing required choice. The caller
must supply the complete legal list. The zero filter permits any valid source.
`DamageSourceHasColor(Color)` accepts one of the five Magic colors;
`DamageSourceHasType(CardType)` accepts a valid engine card type.
`AllDamageSources(...DamageSourceFilter)` and
`AnyOfDamageSources(...DamageSourceFilter)` require nonempty operands and copy
the supplied slice. Invalid colors, types and nested declarations fail engine
validation. `Match(DamageSource) bool` is a pure information query; a missing
source never matches. The original unfiltered API remains available.

Choice evaluates each reference at resolution. Live references supply current
properties; saved/event references supply their retained information. The filter
does not make a fresh battlefield lookup by card ID. Exact duplicate removal,
source snapshots, local results, no-choice clearing, player compatibility and
cloning use the existing choice service. The source need not deal damage.

RON now validates the filter argument of `ChooseADamageSource/1`:
`AnyDamageSource` as a symbol, `IsColor/1`, `IsCardtype/1`, and recursive nonempty
`And([filters])/1` or `Or([filters])/1`. The RON card types use the current
compiler vocabulary. Nested semantic inventory assigns source-filter, color
and card-type roles even inside rejected records. The choice action still
rejects after filter validation: complete CR 609.7a enumeration, per-action
bindings and complete consuming shields remain prerequisites. Filter validation
alone cannot produce a partial production registry.


### RON combat defending player targets

For singular permanent targets in instant/sorcery `SpellActions`,
`ControlledByAPlayer(SpellDefendingPlayer)` selects current permanents controlled
by the nonactive player during combat. The `SinglePlayer(SpellDefendingPlayer)`
wrapper is also supported. Either player can cast the spell. The target control
restriction does not follow the caster. Nested `And` filters, modal spell targets
and numbered singular target groups reuse their existing target declarations.
Outside combat the engine supplies no defending player, so no target qualifies.
Use the card's independent `CastEffect` to establish its exact casting window.

For source attack or unblocked-attack trigger targets,
`ControlledByAPlayer(SinglePlayer(DefendingPlayer))` uses the event's saved
player. The bare `DefendingPlayer` form is also supported in this target context.
The existing `ControlledByPlayer(Trigger_DefendingPlayer)` form is unchanged.
The player is available before target selection and stays fixed through copies,
source departure and later turn changes. Control, incarnation, phasing and target
protection are checked again at resolution.

RON spell-defender symbols are accepted only in these spell target filters;
event-defender symbols require the matching event scope. Other references,
malformed wrappers and tuple symbols remain declaration errors.
`CombatDefendingPlayer() mage.PlayerRef` and `mage.TargetControlledBy` supply
the reusable engine contract. RON keeps the syntax and scope checks.

Blaze of Glory, False Orders and Floral Spuzzem have other unsupported operations.
The target prerequisite does not make their complete records executable.
`TestRONCombatDefender*` maps the supported target contracts; the existing
`TestRONDefendingPlayer*` covers accepted full-card regressions.

## RON upkeep player conditions

`TriggerI(AtTheBeginningOfAPlayersUpkeep(AnyPlayer | SinglePlayer(You)),
condition, actions)` supports these player conditions:

- `PlayerPassesFilter(You | Trigger_ThatPlayer, ControlsA(permanent_filter))`.
- `APlayerPassesFilter(Opponent, ControlsMorePermanentThanPlayer(You, permanent_filter))`.
- `APlayerPassesFilter(AnyPlayer, SinglePlayer(ThePlayerWithTheMostLife))`.

Filters use the existing static-condition filter compiler. Empty conjunctions,
unbound players, tuple symbols and other selectors are rejected. The event
player and action body share one fresh binding per factory. The engine checks
current conditions at detection and resolution. A tie does not satisfy the
highest-life condition. An opponent count must be strictly greater; ownership,
LKI and phased permanents cannot supply current control.

Spiritual Sanctuary has a complete RON gameplay mapping in
`TestRONSpiritualSanctuary` and `TestRONSpiritualSanctuaryLifecycle`. Land Tax
still needs its multi-card search; Ghazbán Ogre still needs its indefinite
control action. Their condition probes do not establish complete gameplay.

## RON end-step source state and creature death conditions

In `TriggerI(AtTheBeginningOfAPlayersEndStep(players), condition, actions)`,
`players` can be the symbol `AnyPlayer` or `SinglePlayer(You)`. This slice adds:

- `PermanentPassesFilter(ThisPermanent, IsUntapped)`.
- `PermanentPassesFilter(ThisPermanent, DidntAttackThisTurn)`.
- `ACreatureOrPlaneswalkerDiedThisTurn(IsCardtype(Creature))`.

The first two filters are symbols. Other subjects, filter tuples, death filters,
and contexts remain unsupported. The spelling of the third condition does not
permit a planeswalker-only death filter. All declarations are checked before
factory construction.

Lowering uses `SourceTappedStateWithLKI(false)`,
`SourceAttackHistoryThisTurn(false)` and `CreatureDiedThisTurnCondition()` with
`TurnStepBegins` and `WithInterveningIf`. Each condition is read at detection and
resolution. The resolving source keeps its original incarnation. An absent source
can supply information but cannot receive counters or a bonus.

Osai Vultures has a complete gameplay mapping in `TestRONOsaiVulturesEndStep`,
`Activation` and `NativeSteps`. `TestRONEndStepSourceState` and `StateRecords`
provide prerequisite evidence for Erg Raiders and Voodoo Doll. Their complete
records still reject at resolving `Unless` and activation `XIs`, respectively.
`TestRONEndStepStateDeclarations` preserves strict declaration rejection;
`StateRecords` covers all four selected records, independent factories,
semantic roles and atomic loading. No complete gameplay claim is made for
those two blocked names.

### Typed player damage-history values

```go
mage.DamageTakenThisTurn(subject mage.PlayerRef) mage.ResolutionValue
mage.ArtifactDamageTakenThisTurn(subject mage.PlayerRef) mage.ResolutionValue
```

These pure resolution-time values read actual damage taken this turn. Prevention
and redirection affect the saved amounts; ordinary life loss does not. Artifact
classification uses the source's type at damage time. Later type or zone changes
do not change history. An absent player supplies zero. Definition validation
checks the typed reference. Clones keep separate totals, and cleanup clears them.

RON `DamageDealtToPlayerThisTurn(player)` uses the first value.
`DamageDealtToPlayerThisTurnByPermanents(player, IsCardtype(Artifact))` uses the
second. Other source filters remain errors. References use existing resolution
bindings, including target, scoped action and delayed player references. These
values do not supply casting costs, entry amounts or static characteristics.

### Static fixed mana increases

Use `mage.IncreaseSpellCostStatic(filter, increase, controllerOnly)` with
`WithStaticAbility` to add fixed generic or colored mana to matching spells.
The increase uses the live source controller when `controllerOnly` is true;
otherwise, it applies to all casters. Increases precede reductions in the shared
spell payment transaction. It accepts nonnegative generic and W/U/B/R/G counts,
including zero; X, hybrid mana and nil filters are invalid declarations.
Source departure, phasing and ability removal stop the tax. A returned source
starts a new instance. Stack copies require no extra payment.

RON permanent rules support `PlayerEffect(You, [IncreaseSpellCost(filter,
PayMana(mana))])` and `EachPlayerEffect(AnyPlayer, [IncreaseSpellCost(filter,
PayMana(mana))])`. The operation list contains exactly one rule, and `mana`
contains fixed mana symbols. Spell-cost filters support `AnySpell`, nonland
`IsCardtype`, `IsColor` for W/U/B/R/G, and nonempty nested `And`/`Or`.
`IsColor` reads card colors, including an indicator. It also works in the existing
static generic-reduction context. This support does not include an
`ActivatedAbilityEffect` tax. Derelor is a complete RON consumer; Gloom still
requires that separate activation contract.

## RON static activation mana modifiers

Permanent definitions accept `ActivatedAbilityEffect(subject, operation)/2`:

- `AbilityOfAPermanent(filter)/1` with `IncreaseManaCost(manaSymbols)/1`.
  The filter accepts `IsCardtype(type)/1`, `IsColor(color)/1`, and nonempty
  nested `And(list)/1` or `Or(list)/1`. Mana symbols use fixed, nonnegative
  generic and W/U/B/R/G costs. X is rejected.
- An Aura accepts `AbilityOfPermanent(HostPermanent)/1` with
  `ReduceManaCostNotLessThanOne([CostReduceGeneric(n)])/1`, where `n` is a
  nonnegative integer literal. Other references and reduction lists fail
  declaration checks.

These lower to `IncreaseActivationCostStatic(filter PermanentFilter,
increase core.ManaCost) ContinuousEffect` and
`ReduceActivationCostStatic(subject PermanentRef, reduction, minimum int)
ContinuousEffect`. They use live source incarnations and typed Aura host
references. Sources can affect either player's abilities. Source departure,
phasing and printed ability removal disable them. Reattachment changes the
subject; a returned host does not inherit an old attachment.

The shared activation transaction combines all mana components and announced
X, adds increases once, then reduces generic mana. Colored mana is retained.
A minimum limits the reduction; it does not add mana to a zero cost. Multiple
sources accumulate. The legacy `AddActivationCostReduction` caller remains
supported. Ordinary activation affordability, actual payment, native mana
abilities and solver plans use the shared total. Spells and resolution payments
do not acquire these modifiers. Copies of abilities do not pay again.

Evidence: `TestStaticActivationCost*`, `TestRONActivationCost*`. Complete RON
Gloom tests cover both its spell and activation taxes. Complete Power Artifact
casts and Basalt Monolith interactions cover its artifact enchant restriction,
generic reduction, stacking, zero mana and one-mana minimum. All six Gloom
reprints and the Power Artifact record pass declaration and strict subset
loading tests; the production collection still rejects remaining gaps.

### Current card P/T queries for count CDAs

Use `g.CardPowerToughness(card)` when the current value of a card matters.
`card.Power()` and `card.Toughness()` remain printed values for copy construction.
The existing `PTEqualsCount`, `PTEqualsControlledCount` and
`ToughnessEqualsControlledCount` declarations work with this query outside the
battlefield. Counts use current nonphased permanents. A spell uses its controller;
a card without a controller uses its owner. Battlefield queries use layer results,
including ability removal, base overrides, bonuses, counters and switching.
Use saved-information services for LKI; this current-object query cannot preserve
a departed incarnation. See the count-CDA contract in `engine-rules-contracts.md`.
RON count-CDA lowering is described below.

### RON battlefield count CDAs

Creature rules can declare `CDA_Power(TheNumberOfPermanentsOnTheBattlefield(filter))`
and `CDA_Toughness(TheNumberOfPermanentsOnTheBattlefield(filter))` together with
identical position-free count expressions. They lower to one fresh
`PTEqualsCount` or `PTEqualsControlledCount` declaration per factory instance.
A toughness-only declaration requires a `ControlledByAPlayer(SinglePlayer(You))`
restriction and lowers to `ToughnessEqualsControlledCount`. Duplicate axes,
unequal expressions, power-only counts and noncreature declarations reject.

Count filters accept nonempty `And`/`Or`, explicit card names through
`IsNamed(NamedCard(nonempty_string))`, card types, creature/land/enchantment
subtypes, colors and supertypes, including their existing negative forms.
Controller restrictions must occur in conjunctions and use `You`; disjunctions
cannot contain a controller restriction. Object references, current P/T,
ability filters, arithmetic and entry/payment results remain unsupported.
The engine supplies owner/current spell controller context outside the
battlefield, layer 7a execution, phasing, per-incarnation binding and copying.

Evidence: `TestRONCharacteristicCount*` executes the complete Dakkon Blackblade,
Keldon Warlord, Nightmare, Plague Rats and People of the Woods records and checks
all twenty records, including reprints. The strict full collection still rejects
remaining gaps. Compilation counts do not establish full collection gameplay.


## Live-object damage-source enumeration prerequisite

`mage.Game.LiveDamageSources() []mage.DamageSource` reads current unphased
battlefield permanents, followed by live spells in stack insertion order.
Permanent spells and spell copies qualify. An ability on the stack does not
qualify as a spell. Objects need not have a damage ability. Shroud does not
restrict this non-targeted source choice. The query uses the current layer
results and does not run a layer pass, make decisions, or change game state.
Each returned source has independent snapshot data.

`mage.ChooseDamageSourceFromLiveObjects(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` enumerates those objects at resolution and uses the
existing filtered choice service. The result uses the current resolution frame.
The explicit-reference choice APIs retain their existing behavior.

This is a complete contract for the two live-object categories, not complete
CR 609.7a enumeration. Objects referred to by stack objects, waiting replacement
or prevention effects, or waiting delayed triggers, and face-up command-zone
objects require further engine contracts. RON `ChooseADamageSource/1` remains
rejected. Its consumer cannot use this restricted list as its whole legal list.
No source files or production registry change in this slice.

### Captured blocking rule permissions

Use `Until(EndOfTurn, BlockAnyNumber(subject), BlockEachAttacker(subject))` for a
permanent that can block every legal attacker and must do so. These operations
change rules; they do not grant abilities. The shared blocker planner respects
capacity, evasion and minimum blockers. Captured subjects keep their incarnation
through control changes and phasing. Ability removal does not remove the rules.

RON `CreatePermanentRuleEffectUntil(subject, operations, UntilEndOfTurn)` accepts
nonempty lists of the bare operations `CantBeRegenerated`,
`CanBlockAnyNumberOfCreatures`, and `MustBlockEachAttacker`. The subject is a
permanent source or declared permanent target. Regeneration prohibition uses
`Until` with `GrantKeywordModification(subject, CantRegenerate)`; it does not
prevent creation of a regeneration shield. Each factory keeps separate target
bindings. Unsupported operations and durations fail compilation.

### RON static untap-step rules

Permanent rules accept `EachPlayerEffect(AnyPlayer, [SkipsUntapStep])`, with
exactly one bare operation. It lowers to `Static(SkipUntapSteps())` and replaces
the whole step before phasing and untap actions. The existing replacement
pipeline chooses and rechecks competing replacements. Other player selectors,
tuple operations, multiple operations and resolving/cost/entry contexts remain
errors. Stasis also uses the existing controller-upkeep payment and sacrifice
adapters; the skip does not skip that upkeep.

A static `If(condition, [EachPlayerEffect(AnyPlayer,
[CantUntapMoreThanNumberPermanentsDuringTheirUntapStep(Integer(n), filter)])])`
uses `LimitUntapsDuringStepIf(filter, n, condition)`. The condition uses the
existing static-condition grammar and typed source/Aura-host references;
unbound targets and incompatible references remain errors. The limit is one
nonnegative literal and uses the existing permanent filter grammar. No other
all-player operation is supported inside this conditional body, and mixed
bodies remain rejected. This form does not add a target or mutable binding.

Winter Orb's complete record checks `PermanentPassesFilter(ThisPermanent,
IsUntapped)`. The engine reads that condition after phasing but before the untap
selection. If a tapped Orb untaps in the same operation, the operation keeps
its unrestricted selection. The next untap step reads its new state. Separate
instances, control changes and cloned games retain independent source contexts.
Stasis and Winter Orb have complete RON gameplay mappings in
`static_player_untap_test.go`; identical reprints also pass definition validation.


## RON resolving numeric consumers

`GainLifeForEach(value, count)` resolves both typed values, multiplies them with
`mage.MultiplyValues`, and sends one life-gain action through `mage.GainLifeFor`.
It uses the current action's player, including explicit `PlayerAction` and
`PlayerActions` scopes. Spell, mode, activation and ordinary trigger bodies use
the same instance value compiler as `GainLife`. The existing restriction on
life actions in a directly captured event-player scope still applies. This form
adds no static, entry, payment, or replacement-operation declaration.

`NumberCountersOfTypeOnPermanent(value, counter, ref)` now uses that same typed
instance value compiler. It supports arithmetic and in-scope permanent
mana-value, power, counter-information and saved payment/event values, along
with existing integer and announced-X values. The existing top-level
`NumCreaturesOrPlaneswalkersThatDiedThisTurn(IsCardtype(Creature))` form remains
supported. `UptoNumberCountersOfTypeOnPermanent` uses the same amount compiler;
its fixed cap, optional number choice and other restrictions stay in force.
Player counter placement retains its fixed-integer amount restriction.

Values are read when each action executes. A legal target's current mana value
supplies Living Armor's +0/+1 counter count. Icatian Moneychanger reads the
original source's retained credit counters after its sacrifice cost. Returned
objects cannot replace that information. An information read cannot make an
illegal, absent or phased recipient eligible for placement. Nonpositive final
amounts do nothing. Unknown, wrong-type and out-of-scope references fail at
compilation; each factory keeps its own target and payment bindings.

`TestRONNumericConsumer*` covers both complete RON records, sequencing,
resolution-time values, supported contexts, invalid declarations, LKI, timing,
target legality, copy retargeting, independent instances and clones, and
replacement ordering. These adapters add no engine service or new simultaneous
operation contract.


### RON resolving source exclusion

`SacrificeAPermanent(filter)` in a permanent's `Activated` or
`ActivatedWithModifiers` cost accepts `Other(ThisPermanent)` alone or in nested
nonempty `And` filters. Ordinary nonempty `Or` filters can select payment types;
source exclusion below `Or` remains rejected. Payment uses
`SacrificeMatchingCost`, so the source cannot pay. Existing costs without the
exclusion still permit the source. Sacrifice characteristic result bindings and
granted abilities retain their prior supported contexts.

`CreateEachPermanentLayerEffectUntil(filter, [AdjustPT(power, toughness), ...],
UntilEndOfTurn)` accepts the same source exclusion for a permanent source.
The engine captures matching recipients when the effect starts. Both players'
matching permanents qualify unless a controller filter restricts them. A
returned source is a new object and can receive the bonus. Later entries and
returned recipients cannot join the captured set. Phasing preserves a captured
recipient's identity. Other references, source-dependent disjunctions and a
spell's `ThisPermanent` remain declaration errors.

Evidence: `TestRONOrcGeneral`, `TestRONOrcGeneralRollback`,
`TestRONSourceExclusionInstancesAndCopy`, `TestRONSourceExclusionInvalid`.

## RON live nonattacking filters

Bare `IsntAttacking` is a permanent filter and lowers to
`mage.Not(mage.IsAttacking)`. It can appear in existing permanent target,
collection and static filter contexts, including nested `And`/`Or`.
Static `PermanentPassesFilter` conditions accept it on `ThisPermanent` or an
Aura's `HostPermanent`. Tuple forms are invalid. Existing reference and
source-dependent filter restrictions still apply. Card-zone and chosen damage
source filters do not gain this predicate.

Static membership reads current combat status at each layer pass. Untapping an
attacker does not make it nonattacking; removal from combat does. This predicate
is separate from attack history. Target filters check announcement and resolution.
`TestRONNonattacking*` covers the complete Arcades Sabboth record, live source
and attachment conditions, target legality, identity and clone isolation.
`TestRONUpkeepPayment*` includes its upkeep payment and sacrifice behavior.
Gaea's Liege still lacks conditional base-value and defending-player adapters;
this filter does not establish its gameplay support.

## Engine prerequisite for permanent property alternatives

The engine now exposes
`mage.IfPermanentMatchesElse(PermanentRef, PermanentFilter, ResolvingEffect,
ResolvingEffect) ResolvingEffect` and
`mage.UnlessPermanentMatches(PermanentRef, PermanentFilter, ResolvingEffect)
ResolvingEffect`. These services use one current or retained information read
for the exact object. Missing information runs neither body. Both branch bodies
are required and validated; branch-local producers do not escape their paths.
Both new services execute their conditions in AI simulations.

RON resolving action lists now accept
`IfElse(PermanentPassesFilter(subject, propertyFilter), [matchingActions],
[nonmatchingActions])` and
`Unless(PermanentPassesFilter(subject, propertyFilter), [nonmatchingActions])`.
They use the existing resolving property and counter filters, including nested
nonempty `And`/`Or`. Both bodies must be nonempty. Each branch compiles in its
own action sequence with fresh factory bindings. Payment results remain local
to their adjacent continuation; they cannot escape into another branch.
Static and combat-membership alternatives remain unsupported. Conditional mana
production in activation branches remains a declaration error.

The exact historical exception shape is
`Unless(PermanentPassesFilter(subject, CameUnderControlOfPlayerThisTurn(playerRef)),
[actions])`. It uses
`mage.UnlessPermanentCameUnderControlThisTurn(PermanentRef, PlayerRef,
ResolvingEffect) ResolvingEffect`. Each original incarnation retains the latest
acquisition turn for every player. Entry and actual control changes record it;
phasing and ability removal do not erase it. A later controller does not erase
another player's acquisition earlier in the same turn. Haste and summoning
sickness do not supply this history. Current, phased and departed information
use the same identity service; returned objects cannot replace it. Missing
object or player information runs no body. Other history shapes remain errors.

Evidence: `TestRONPropertyAlternatives*` covers the complete Blood Lust and
corrected Erg Raiders records, both property branches, nested conditions,
current reads, target legality, casting payments and rollback, retargeted copies,
factory and clone isolation, source LKI, phasing, control changes, damage
prevention and cleanup. `TestUnlessPermanentCameUnderControl*` supplies the
engine history and declaration contracts. Existing `TestPermanentPropertyBranch*`
and `TestRONResolvingCondition*` retain their behavior.

## Sacrifice-derived mana and restricted resolution production

`mage.AddRestrictedManaOfAnyOneColor(amount ResolutionValue, restriction
ManaRestriction) ResolvingEffect` reads the amount once at resolution. A positive
amount uses one controller color choice for all units. Zero or negative amounts
produce no mana and make no choice. Definition validation rejects a missing
amount or restriction. Production uses the existing mana-pool service and
production tracking. Fixed production metadata retains the restriction, so the
shared payment planner cannot use it for an incompatible spell or activation.
`CreatureSpellsOnly` uses the existing spell payment context (CR 106.6).

RON accepts `AddManaRepeatedWithModifiers(AnyManaColor, value,
CanOnlySpendOnSpells(IsCardtype(Creature)))` in resolving action lists, with the
existing supported resolution-value shapes. Other production and restriction
forms remain compile errors. The inventory reports the restriction role and its
nested spell filter independently of compiler success.

`AddManaRepeated(fixedProduction, ManaValueOfTheSacrificedPermanent)` now
accepts the existing unique matching sacrifice activation binding as well as an
additional casting cost binding. An absent, ambiguous, malformed or unrelated
binding remains an error. Each factory instance has its own typed number binding.
The shared payment transaction saves mana value before sacrifice. Copies retain
the paid value without another payment and use their own resolving controller.
Game clones keep separate pools and payment frames. Returned payment objects
cannot change the number (CR 400.7, 202.3, 707.10).

Priest of Yawgmoth remains an untargeted mana ability, including zero production:
payment and production finish immediately without the stack (CR 605.1a, 605.2,
605.3b). A tap symbol still checks summoning sickness. The matching sacrifice
permits its source if it is an artifact. Metamorphosis pays its additional cost
at casting and produces mana only when the spell resolves (CR 605.5b).
Automatic planning does not select arbitrary sacrifice payments; the existing
choice-cost exclusion remains in effect.

Evidence: `TestRestrictedResolutionMana`, `Declarations`, and `Planning` in
`pkg/mage/restricted_resolution_mana_test.go`; complete RON gameplay in
`TestRONSacrificeManaPriest`, `PriestLegality`, `Metamorphosis`,
`MetamorphosisRollback`, `ActualSpending`, `PendingValues`, `Invalid`, and
`RegistryAndInventory` in `pkg/mage/dsl/ron/sacrifice_mana_test.go`.

### RON resolving blocking-history collections

`WasBlockedByDefenderThisTurn(permanentRef)` accepts one bound singular
permanent reference in a resolving collection. It lowers to
`mage.BlockedByThisTurn` and composes through the existing collection `And/Or`
adapters. For example, both Glyph destruction clauses use
`And([IsCardtype(Creature), WasBlockedByDefenderThisTurn(Ref_TargetPermanent)])`.
The existing `DestroyEach` and `DestroyEachWithoutRegeneration` actions retain
batch destruction, regeneration and replacement behavior.

The reference belongs to its rule instance and keeps its original incarnation.
Saved blockers can supply history after departure or phasing. Collection members
must still be live and phased in. Invalid arity, unbound references, player
references and cross-rule reads are declaration errors. Inventory gives the
operand a permanent-reference role even below unsupported delayed actions.

Target-filter use, including Glyph of Delusion's dependent second target, is
still unsupported. `AtNextEndOfCombatThisTurn`, after-combat casting, death-result
loops and the RON last-block controller expression remain separate gaps. The
engine's typed `ControllerWhenLastBlockedThisTurn` supplies the controller
history prerequisite. The three complete Glyph records still reject. Tests
of extracted clauses do not claim complete gameplay coverage for those cards.

## RON resolving collection attack rules

`CreateEachPermanentRuleEffectUntil(filter, [CantAttack], UntilEndOfTurn)`
accepts the supported source-independent static permanent filters, including
nonempty logical combinations. Controller and source-exclusion selectors are
not supported in this restriction shape.

`CreateEachPermanentRuleEffectUntil(And([filter, ...,
ControlledByPlayer(TheActivePlayer)]), [MustAttack], UntilEndOfTurn)` accepts
exactly one active-player selector and one or more supported permanent filters.
It saves the active player at resolution and checks current creature membership
at attack declaration. Filter order does not matter. Other player selectors,
operation lists, durations and unbound references remain compile errors.
Both forms work in resolving action lists. They lower to
`CantAttackMatchingThisTurn` and `MustAttackControlledThisTurn`; they do not use
captured layer membership or give creatures an ability (CR 611.2c).

Festival's complete RON record is tested. Siren's Call's exact attack clause is
tested in a test-only extracted record; its full record still rejects on the
delayed destruction collection. Runesword's regeneration and exile clauses
remain unsupported. These forms do not establish other collection rule support.

### Turn-scoped unblocked attacker subscription

`WhenAttacksUnblockedThisTurn(subject PermanentRef, body ResolvingEffect)
ResolvingEffect` saves the watched incarnation and resolving controller. It uses
legal blocker-declaration events in each combat until cleanup. A phase cycle
keeps identity; a returned object cannot use the subscription. Source departure,
phasing, control changes and ability removal do not end the subscription.
Pending triggers retain the body after cleanup. Use `Snapshot` and `From` when
the body must read the watched object independently of the creator.

RON supports `CreateTriggerUntil(WhenACreatureAttacksAndIsntBlocked(
SinglePermanent(ref)), ActionList([actions...]), UntilEndOfTurn)` in resolving
contexts. The subject must be a bound singular permanent reference. The nonempty
body can use existing supported actions and the local `Trigger_ThatCreature`
reference. This reference names the watched incarnation; it does not name the
artifact that created the subscription. Parent references retain their bindings.
Each installation saves its own frame, including original targets and X.
Other delayed combat events, durations, target declarations inside this body,
and unsupported body operations still reject.

Delif's Cone and Delif's Cube use this event. Real-record clause tests establish
Cone's power/life expression and Cube's counter instruction with their actual
activation costs and target restrictions. Cone remains unsupported on optional
life gain. Cube's complete gameplay uses the assignment rule described below.

### Combat damage assignment suppression

`mage.AssignsNoCombatDamage(subject mage.PermanentRef) mage.ContinuousModification` can be installed with `mage.Until`. It changes a
combat rule and remains effective after ability removal. The resolved subject
keeps its incarnation; source departure and subject control changes do not end
an end-of-turn installation. A phase cycle keeps it. Cleanup ends it, including
while the subject is phased out. An absent or phased-out subject at installation
receives no rule. This service does not prevent damage or change power.

RON resolving action lists accept
`CreatePermanentRuleEffectUntil(permanentRef, [AssignsNoCombatDamage],
UntilEndOfTurn)`. There are exactly three arguments, one symbol operation, and
one symbol duration. The permanent reference must be bound in that action's
scope: source, target, Aura host, or a supported permanent event binding.
Factory construction uses that instance's typed references. Other durations and
mixed operation lists are not added by this adapter.

`TestRONCombatAssignmentCube` executes the complete Delif's Cube record,
including casting, its delayed combat trigger, cube counter payment and
regeneration. `RollbackAndIdentity`, `EventAndInstances`, `SourceAndHost` and
`Invalid` establish the supported scopes, legality, copies and declaration
rejections. The optional payment contract below adds three consumers.
Farrel's Mantle still needs its host unblocked event contract.

## RON optional resolution action payments

`MayCost(action)` and `PlayerMayCost(player, action)` accept these complete
singular action shapes in resolving sequences:

- `GainLife(value)` uses the current action player and a resolution value.
- `HavePermanentDealDamage(permanent, value, recipient)` uses a typed permanent
  source and one existing `Permanent(ref)`, `Player(ref)`, or singular union
  recipient. Source information follows `DamageSourceOf`, including LKI.
- `DestroyPermanent(permanent)` requires the original live, unphased object
  without indestructible. Regeneration can replace the paid instruction.

The payment must immediately precede `If(condition, [actions])`,
`Unless(condition, [actions])`, or `IfElse(condition, [actions], [actions])`.
The condition is the symbol `CostWasPaid` or `CostWasntPaid`; each listed
branch is nonempty. References and values must already be bound in the current
context. The adapter creates no new event, choice, or target binding.
`MayCost` asks the resolving controller. `PlayerMayCost` asks its explicit
player; it does not change the action's player or damage source.

Lowering uses `PayActionOrElse` with a typed `ResolutionActionPayment`.
The choice determines the branch (CR 118.11–12). Prevented damage, replaced
life gain, and regeneration still count as payment. Indestructible and absent
subjects cannot pay destruction. The payment captures its amount and object
identity before the choice, then rechecks the recipient before execution.
Nested payments cannot replace an outer result. Each payment is one action;
these forms do not add composite or announcement costs.

`TestRONActionPaymentRealCards`, `NativeCombat`, `Invalid`, `StrictLoad`, and
`BranchesAndChooser` cover Delif's Cone, Farrel's Zealot, and Floral Spuzzem.
Farrel's Mantle still needs its host event binding and a source-data review.

## RON indefinite fixed color and keyword changes

`CreatePermanentLayerEffect(reference, [SetColor(SimpleColorList([colors...]))])`
accepts distinct Magic colors. An empty list sets colorless. It replaces the
subject's colors in layer 5. Choices and unbound references are rejected.

`CreatePermanentLayerEffect(reference, [AddAbility([keywords...])])` accepts a
nonempty list of the existing fixed keyword declarations, including fixed
landwalk. The separate activated-ability grant form remains supported.
These resolving forms use `Until(Indefinite)`, `SetColors`, and
`GrantKeywordModification`. They capture the exact live subject at resolution.
Source departure does not end them. Returned subjects do not inherit them;
phasing preserves an installed effect. Later layer timestamps still apply.

Full-card evidence: `TestRONAislingIndefiniteColor`,
`TestRONAislingNativeCombat`, and `TestRONCocoonCompleteLifecycle`.
Declaration and recipient contracts are in `TestRONIndefiniteCharacteristic*`
and `TestRONIndefiniteKeywordIdentity`. This slice adds no engine service.

## RON scoped color choices

Resolving action lists support these adjacent pairs:

```ron
ChooseAColor(AnyColor),
CreatePermanentLayerEffect(permanent_reference, [SetColor(TheChosenColor)])

ChooseColors,
CreatePermanentLayerEffect(permanent_reference, [SetColor(TheChosenColors)])
```

The first pair offers exactly the five Magic colors. The second offers all 32
subsets, including no colors. Each choice is made at resolution by the resolving
ability or spell's controller. These are resolution choices, not announced modes.
The result belongs only to the adjacent color-setting continuation. Unbound,
wrong-kind, nonadjacent and escaped result reads are rejected. Entry choices,
chosen-color filters, other modifications and other durations remain unsupported
by this adapter. The existing typed permanent reference compiler supplies the
subject; the choice introduces no target declaration.

The compiler lowers each legal option to a fixed `SetColors` modification with
`Until(Indefinite)`, through `ChooseAtResolution`. No choice is stored on a source
permanent or in shared mutable declaration state. The existing layer capture
service owns incarnation identity, phase cycles, timestamps and game cloning.
Accepted Alchor's Tomb and Dream Coat records have gameplay mappings in
`color_choice_test.go`; this does not certify all accepted collection records.

## RON temporary ability grants

`CreatePermanentLayerEffectUntil(ref, [AddAbility([Protection(FromColor(
Colors([White])))])], UntilEndOfTurn)` accepts one or more of the five Magic
colors. Empty lists, colorless, unknown qualities, malformed constructors and
unbound subjects fail compilation. Protection is part of the existing temporary
ability plan, including its scoped `ChooseAnAbility` continuation. The plan keeps
an immutable color set; factory calls build separate target declarations.
Existing permanent references and temporary durations keep their scope checks.

`CreateEachPermanentLayerEffectUntil(Ref_TargetPermanents,
[AddAbility([Landwalk(IsLandType(Island))])], UntilEndOfTurn)` accepts nonempty
lists of existing temporary keywords, including fixed landwalk. It uses the
existing plural target binding from `NumberTargetPermanents` or
`OneOrMoreTargetPermanents`. Multiple `AddAbility` operations share one captured
installation. Mixed color/keyword modifications, plural protection/rampage and
delayed plural target reads remain unsupported.

`mage.GrantProtectionColors(subject mage.PermanentRef, colors ...core.Color)
mage.ContinuousModification` is the typed engine primitive. Definition validation
rejects missing subjects, undeclared targets and invalid color sets. Use it with
`Until` to capture the live subject incarnation in layer 6. Normal protection
rules apply to targeting, attachment, blocking and damage; there is no Ward
attachment exception. Repeated layer passes do not accumulate abilities.

Goblin Wizard and Part Water have complete RON gameplay mappings in
`temporary_ability_grant_test.go`. Their native costs, choices, targets, copies,
clones, object identity and cleanup are tested separately from loader acceptance.

### RON live static base values

RON supports `PermanentLayerEffect(subject, [SetPowerAndToughnessBoth(value)])`
for `ThisPermanent` and a declared Aura's `HostPermanent`, alone or inside an
existing static `If` body. This singular operation uses `SetBasePowerToughness`
and `While` in layer 7b. Its value is read on each layer pass, before bonuses,
counters and switches. It is not a characteristic-defining ability.

Static values accept `Integer/1`, `Plus/2`, `Minus/2`, `Twice/1`,
`HalfRoundedDown/1`, `HalfRoundedUp/1`, and the existing restricted battlefield
counts. `ManaValueOfPermanent/1` accepts the source or declared Aura host and
uses current information. These values also work in the existing singular
static `AdjustPTXY` and `AdjustPTForEach` adapters. Targets, event references,
announced X and P/T reads remain invalid in this context.

Combined modifications and collective base-value operations remain rejected.
This prerequisite does not accept the complete Animate Artifact, Angry Mob or
Gaea's Liege records. Their remaining rules require separate slices.

### RON basic land type choices

`AsPermanentEnters(ThisPermanent,[ChooseABasicLandType])` declares one saved
entry choice on a permanent. The operation must be a symbol. Duplicate choices,
combined operations and nonpermanent declarations are errors. Rule order does
not change this declaration's scope. On an Aura,
`PermanentLayerEffect(HostPermanent,[SetLandTypeVariable(TheChosenLandType)])`
uses that entry declaration and accepts exactly one modification.

The engine runs `ChooseBasicLandTypeAsEnters` before battlefield registration.
The entry controller chooses Plains, Island, Swamp, Mountain or Forest through
`ChooseString`. The permanent stores `core.BasicLandType` separately from its
chosen color and other entry choices. `BecomesChosenBasicLandTypeAttachedEffect`
uses the existing attachment and land-type services. It replaces land subtypes
and printed abilities, supplies intrinsic mana, preserves supertypes and follows
reattachment. Source phasing and departure stop the effect. Ability removal in
layer 6 does not undo the type change already applied in layer 4.

In an action sequence, bare `ChooseABasicLandType` must immediately precede
`CreatePermanentLayerEffectUntil(subject,
[AddAbility([Landwalk(IsLandTypeVariable(TheChosenLandType))])],duration)`.
The subject and duration use their existing supported contexts, including the
body of a supported delayed step trigger. This bounded pair declares no target
and exposes no result outside its continuation. The compiler builds five fixed
branches through the existing temporary keyword compiler. `ChooseAtResolution`
selects one branch for each invocation; no mutable compiler or global result
slot stores the choice. The controller chooses at resolution, even if the saved
source has left. A grant still requires its original live incarnation.

The complete Phantasmal Terrain records (six reprints) map to
`TestRONBasicLandChoiceTerrain`, `TestRONBasicLandChoiceTerrainIdentity` and
`TestRONBasicLandChoiceInventoryAndReprints`. Giant Slug maps to
`TestRONBasicLandChoiceSlug` and `TestRONBasicLandChoiceSlugIdentity`.
`TestRONBasicLandChoiceInvalid` and `TestRONBasicLandChoiceStrictLoading` cover
scope errors and atomic rejection. General subtype choices, plural choice
consumers, mixed modifications and other saved-land-choice reads remain errors.

### Conditional static damage redirection

Use `mage.RedirectDamageToPermanentIf(player, destination, sourceFilter,
condition)` with `mage.Static` for player damage from permanent sources.
Use `mage.RedirectDamageFromCardTypeToPermanentIf(player, destination, cardType,
condition)` for a type restriction that includes sources outside the battlefield.
Both constructors require a condition. The rule reads current controller and
attachment references, and the destination must remain a creature or planeswalker.
The shared replacement pipeline preserves source information and damage facts.
`mage.DamageSourceIsUnblocked()` reads current or retained combat status. An
attacker becomes unblocked only after blocker declaration; removal of its last
blocker does not make a blocked attacker unblocked.

RON `If`, `Unless` and `IfElse` static branches can contain one
`ReplaceWouldDealDamage(DamageWouldBeDealtByAPermanentToRecipient(filter,
Player(You)),[DealToCreatureOrPlaneswalkerInstead(ThisPermanent)])`.
`HostPermanent` is also valid in an Aura. Permanent filters accept nonempty
`And`/`Or`, bare `IsUnblocked`, and the supported type, subtype, supertype and
color predicates. `DamageWouldBeDealtByASourceToRecipient(IsCardtype(type),
Player(You))` includes captured spell sources. This does not add resolving
replacement operations or target declarations.


### Original permanent sources of stack abilities

`mage.Game.StackAbilityPermanentSources() []mage.DamageSource` reads original
permanent sources of abilities still on the stack, in insertion order with
exact duplicates removed. It keeps saved incarnation identities through
source departure, return and phasing. Live sources use current information;
absent sources use the shared phase/departure information reader. Returned
cards cannot replace the original source. Missing identities are excluded.
The query is pure and returns separate snapshot data.

`mage.ChooseDamageSourceFromStackAbilities(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` makes a filtered non-targeted choice from that category
at resolution. The shared choice service owns validation, decisions and
per-resolution output bindings. Empty lists and absent players clear the
result. Clones retain independent results.

This source-only API covers one CR 609.7a context. Other stack references,
waiting replacement/prevention and delayed-trigger references, and face-up
command-zone objects remain dependencies. General RON `ChooseADamageSource`
remains rejected; do not combine this and the live-object list and treat them
as the complete legal list. Existing Go callers remain compatible.

### Permanent targets referred to by stack objects

`mage.Game.StackTargetPermanentSources() []mage.DamageSource` reads exact
permanent target incarnations referred to by spells and abilities still on the
stack. Results follow stack insertion order, then target-list order, with exact
duplicates removed. Target legality does not remove this reference: shroud,
phasing or departure can make a target illegal while the object still refers
to it. Current information applies to live incarnations; phase information or
departure LKI applies to absent incarnations. Returned cards cannot replace them.

`mage.ChooseDamageSourceFromStackTargets(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` makes a filtered non-targeted choice from this category
at resolution. It uses the shared decision and per-resolution binding services.
Empty lists and absent players clear only this output. Definition validation
checks players, filters, bindings and producer-before-consumer ordering.

Copies retain original target incarnations unless retargeted. Only current
`Targets` entries qualify; old map entries cannot add candidates. Players,
spell targets, graveyard card visits and missing/unknown identities do not
qualify as permanent targets. Removing the final referring stack object ends
eligibility. The resolving object has already left the stack. Queries are pure,
return separate snapshot data and preserve independent clones.

This establishes one CR 609.7a category. Other saved stack references, waiting
replacement/prevention and delayed-trigger references, and face-up command
objects remain dependencies. General RON `ChooseADamageSource` stays rejected.
No existing caller changes and no new production RON shape is added.

### Spell targets referred to by stack objects

`mage.Game.StackTargetSpellSources() []mage.DamageSource` reads spell targets
of spells and abilities still on the stack. Results follow stack insertion and
target-list order, with exact duplicates removed. A spell need not be capable
of damage. Permanent spells qualify. Ability, player, permanent and source-card
IDs cannot substitute for spell stack IDs.

Live spells supply current characteristics and controller. Each referring stack
object saves the target spell's information. Every native departure path refreshes
that information before removal. A later cast or battlefield entry cannot replace
the old spell. Copies keep the saved reference; retargeted copies use their own
selected targets. Removing the final referring object ends this category's
reference. Queries return separate card data and make no choices or state changes.
Stack clones share immutable maps and copy them before departure updates.

`mage.ChooseDamageSourceFromStackSpellTargets(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` uses the shared filtered resolution choice service.
Declaration validation, producer-before-consumer checks, separate bindings,
empty/absent-player clearing and clone behavior remain the same. This service
covers only spell targets, not the complete CR 609.7a source list. Other saved
bindings, waiting effects, command objects and chosen-source shields remain
separate dependencies. No additional production RON shape is accepted.

### Permanent bindings referred to by stack objects

`mage.Game.StackBoundPermanentSources() []mage.DamageSource` reads exact
permanent incarnations saved by event, announcement, payment and delayed-frame
bindings on spells and abilities still on the stack. It reads `capturedPermanents`,
immutable event captures and captured delayed bodies in their sequence wrappers.
It does not execute an effect or evaluate an unproduced binding. Results follow stack insertion
order, then permanent ID and incarnation byte order within each object, with
exact duplicates removed. Queries return separate card snapshots.

Live incarnations use current information; phased and departed incarnations
use their separate retained information. Returned objects cannot replace them.
Shroud and damage capability do not restrict this non-targeted source category.
Copies keep saved references. Removal of the last referring stack object ends
eligibility. Pending triggers and the currently resolving object are excluded.

`mage.ChooseDamageSourceFromStackBindings(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` reuses the shared filtered resolution choice service.
Definition validation checks the player, filter, output and binding dominance.
Empty categories and absent players clear only that output. Choices and cloned
resolution frames stay separate.

This category includes saved permanent bindings, target groups and created-token
sets in delayed bodies after stack placement. It is not the complete CR 609.7a
source list. Other binding types, waiting replacement/prevention/delayed references
and face-up command objects remain separate dependencies. General RON
`ChooseADamageSource` remains rejected. Existing Go callers stay compatible.

### Damage-source event bindings referred to by stack objects

`mage.Game.StackBoundDamageSources() []mage.DamageSource` reads exact permanent
and spell identities saved by `eventDamageSourceCapture` within stack objects,
including nested sequence wrappers. It does not execute captures, evaluate future
action results, or read the active resolution frame. Results follow stack
insertion order and capture order, with exact duplicates removed.

Live permanent incarnations and spell stack IDs supply current information.
Permanent departure and phase information use the existing exact-incarnation
reader. Spell departure information is saved separately on each referring object,
with copy-on-write updates for clones. A returned permanent or recast spell cannot
replace the original object. The trigger body's event-time source binding remains
unchanged. Query results own separate card snapshots.

`mage.ChooseDamageSourceFromStackDamageBindings(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` uses the shared filtered non-targeted resolution choice
service. Definition validation, independent results, empty/absent-player clearing,
invalid-answer fallback and cloned resolution frames retain their existing rules.

Pending triggers and the currently resolving object are outside this category.
Removing the final referring stack object ends eligibility. This adds one source
category required by CR 609.7a; saved delayed frames, other typed references,
waiting effects and face-up command objects remain separate dependencies. General
RON `ChooseADamageSource` and the seven chosen-source shields remain rejected.
Existing production declarations and callers do not change.

### Spell-event bindings referred to by stack objects

`mage.Game.StackBoundSpellSources() []mage.DamageSource` reads spell references
saved by typed cast-event bindings on stack objects. Detection saves identity and
information before placement. Nested sequences expose immutable captures without
running them. Live spells use current information; departing spells use the shared
copy-on-write stack snapshot service. A spell absent before placement uses its
detection snapshot. Recasts and permanent entry retain separate identities.

The query returns copied card data in stack insertion/capture traversal order,
with exact duplicates removed. Nil, invalid and incompatible captures are
excluded. Pending triggers and the currently resolving object are outside the
category. Copies retain references; clones keep independent departure information.

`mage.ChooseDamageSourceFromStackSpellBindings(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` performs a filtered, non-targeted choice at resolution.
It uses shared definition validation, producer order, independent output bindings,
empty/absent-player clearing, invalid-answer fallback and cloned result frames.

This establishes the typed spell-event context of CR 609.7a. Other delayed bindings,
other reference types, waiting effects, face-up command objects and consuming
shields remain separate dependencies. General RON `ChooseADamageSource` remains
rejected; no new RON spelling or compiler adapter is established here.

## Delayed permanent frames referred to by stack objects

`StackBoundPermanentSources` and `ChooseDamageSourceFromStackBindings` also
read immutable `capturedDelayedBody` frames after delayed trigger placement.
Saved permanent bindings, singular/plural target groups and created-token sets
supply exact permanent incarnations. The shared reader sorts and deduplicates
them with other references in each stack object. Nil declarations and invalid
identities are excluded. Reading does not execute the body or a future producer.

Current objects supply current characteristics and control; absent and phased
objects use exact retained information. Copies preserve captures; game clones
keep independent state. Query and choice snapshots cannot mutate saved cards.
A saved target does not become a new target. Waiting triggers, pending triggers
and the currently resolving object stay outside this stack category.

No RON syntax or context is added. Stone Giant, Goblin Kites and War Barge use
existing delayed adapters; `TestRONStackDelayedPermanentSources` and
`TestRONStackDelayedPermanentSourceInstances` verify their native gameplay and
source choice through a synthetic category consumer. General `ChooseADamageSource`
and the complete shields still need other source categories and shield contracts.

## Permanent references held by waiting delayed abilities

`mage.Game.WaitingDelayedPermanentSources() []mage.DamageSource` reads saved
permanent bindings, singular/plural target groups, created-token sets and exact
permanent event subjects from delayed abilities that still wait to trigger.
Captured modal bodies and sequence wrappers use the same frame reader. The query
uses registration order, then sorted ID/incarnation pairs within each ability,
and removes exact duplicates. It does not run effects or make choices.

Live objects supply current properties. Departed and phased objects use the
shared exact-incarnation information reader. Returned objects cannot replace
saved references. Each result owns separate card data. One-shot detection ends
waiting eligibility before stack placement. A delay with a duration keeps waiting
until its existing lifecycle expires. Pending and stack objects have separate
categories; copies on the stack do not create new waiting abilities. Game clones
retain independent waiting state.

`mage.ChooseDamageSourceFromWaitingDelayedBindings(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` uses the shared filtered resolution-choice service,
declaration validation and binding dominance. Empty or absent-player choices
clear only that output. Invalid answers use the existing first-choice fallback.

This is a permanent-frame and exact-event-subject contract. Other typed saved
references, source reads in future bodies, untyped legacy IDs, waiting replacement
and prevention effects and command objects still need separate contracts. No RON
syntax is added. General `ChooseADamageSource` remains rejected.


## Source references in waiting delayed action bodies

`Game.WaitingDelayedPermanentSources` also reads explicit `This` subjects in
`Destroy`, `DestroyWithoutRegeneration`, `Sacrifice` and `ReturnToHand` bodies.
This context includes saved `capturedDelayedBody` wrappers, nested `Sequence`
bodies and each saved modal body. The typed reference interface does not execute
an action. A source ID alone does not qualify. An action that will create another
delayed ability does not supply its future body's references in this context.
Other source-reading action kinds and conditional wrappers remain separate forms.

The ability must have a captured source and a known permanent incarnation.
The reader uses the saved source ID/incarnation, the existing incarnation
information service and the existing deterministic deduplication service.
Current color and control, departure LKI, phasing and hidden characteristics use
the same contracts as saved target references. Returned cards cannot replace the
original source (CR 400.7, 603.7c, 603.7e, 609.7a and 702.26).

`ChooseDamageSourceFromWaitingDelayedBindings` can consume this category through
its existing filtered choice, validation and output services. One-shot detection
ends waiting eligibility before pending placement. Copies on the stack do not
create waiting abilities; clones keep independent waiting lifetimes. The actions
keep their existing legality: a player cannot sacrifice a source controlled by
another player. This slice adds no cost or replacement operation.

Evidence: `TestWaitingDelayedSourceBodies`, `TestWaitingDelayedSourceBodyBoundaries`
and `TestWaitingDelayedSourceBodyChoicesAndInstances` cover exact identity,
current and retained information, query snapshot isolation, modal and sequence
bodies, declaration rejection, independent filtered outputs, empty/absent-player
clearing, APNAP, stack copies and clone execution. `TestRONWaitingSourceBodies`
executes complete Dragon Whelp, Nalathni Dragon, Rocket Launcher and Rakalite
records, payment failure, delayed creation and native delayed actions.
These are added contracts for accepted records, not complete prevention-card
coverage. General RON `ChooseADamageSource` remains rejected.

## RON static maximum hand size

Top-level permanent rules support these complete shapes:

```text
PlayerEffect(You, [HasNoMaximumHandSize])
PlayerEffect(You, [SetMaximumHandSize(Integer(n))])
PlayerEffect(TheChosenPlayer, [HasNoMaximumHandSize])
PlayerEffect(TheChosenPlayer, [SetMaximumHandSize(Integer(n))])
```

`n` is a nonnegative integer literal. `TheChosenPlayer` requires the existing
`AsPermanentEnters(ThisPermanent, [ChooseAPlayer(Opponent)])` declaration on the
same card. Each list has exactly one operation. These are static declarations;
resolving actions, combined operation lists, target/event player references,
constructor-form symbols, dynamic amounts and negative maxima remain errors.
The inventory assigns the fixed amount the `value` role.

RON lowers these forms to `SetMaximumHandSizeStatic(PlayerRef, int)` and
`NoMaximumHandSizeStatic(PlayerRef)`, both returning `ContinuousEffect`.
`ChosenPlayerOf(PermanentRef) PlayerRef` supplies the saved entry choice through
a live incarnation; it reads no LKI. Use `WithStaticAbility` for native callers.
`Game.MaximumHandSize` and cleanup share the live rule query and timestamp order.
The existing direct hand-size setters remain available to current callers.
Cursed Rack is covered as a complete RON card. Library of Leng's discard
replacement remains unsupported; its hand-size rule alone has gameplay evidence.

### Delayed spell-event references

`AtNext`, `AtNextModes` and delayed permanent-event bodies now capture and restore
`SpellBinding` values. A `FromSpell` read keeps the original stack identity; a
counter action cannot affect a later cast of the same card. Other typed frames
retain their existing behavior.

`mage.Game.WaitingDelayedSpellSources() []mage.DamageSource` reads saved spell
frames only while their abilities wait to trigger. It uses current information
or exact departure information, returns independent snapshots, and deduplicates
in deterministic order. Placed delayed bodies use `StackBoundSpellSources`.

`mage.ChooseDamageSourceFromWaitingDelayedSpellBindings(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` chooses from this category through the shared filtered
choice service. Missing choices and absent players clear only this output.
General RON `ChooseADamageSource` remains rejected until its complete source list
and shield contracts are available.

In supported RON delayed bodies, bound `CounterSpell(Trigger_ThatSpell)` and
other existing spell-reference consumers now use the saved cast-event frame.
`Trigger_ThatSpell` requires an enclosing cast event. It does not become a target
and cannot refer to a later cast. This adds no RON spelling or production registry.

### Delayed damage-source frames

`AtNext`, `AtNextModes` and delayed permanent-event operations save the current
`DamageSourceBinding` frame. `FromDamageSource` reads the saved facts during
body execution. Separate executions receive separate maps, and the outer frame
is restored afterward.

`mage.Game.WaitingDelayedDamageSources() []mage.DamageSource` reads exact source
references held by waiting delayed frames. Queries use current or exact departure
information, preserve return/recast identity, and return separate snapshots.
Placed bodies supply `StackBoundDamageSources`. One-shot detection ends waiting
eligibility; duration-based delays retain it until cleanup.

`mage.ChooseDamageSourceFromWaitingDelayedDamageBindings(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` chooses from this category through the existing filtered
choice service. Empty categories and absent players clear only the named output.

RON's supported controller/source received-damage events create fresh internal
source captures that supported delayed bodies can retain. Existing amount and
recipient spellings keep their scope. No new source-reference spelling is added.
General RON `ChooseADamageSource` still requires the other source categories and
complete shield contracts.

### Waiting prevention permanent references

`mage.Game.WaitingPreventionPermanentSources() []mage.DamageSource` reads exact
permanent recipients saved by `PreventNextDamage` and `PreventDamageThisTurn`,
and exact source subjects saved by `PreventNextDamageFromPermanent`. It scans
active persistent and cycle replacements in registration order, removes exact
duplicates, and returns separate snapshots. Player IDs, legacy source IDs
without incarnations, filters over unspecified future members, unknown objects,
consumed shields and removed effects do not supply references.

`mage.ChooseDamageSourceFromWaitingPreventionReferences(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` uses this category with the shared non-targeted choice and
binding service. Missing players and empty filtered categories clear that output.

Existing RON lowering of
`CreateFuturePreventDamage(NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient(amount,
recipient), [PreventThatDamage])` and
`CreatePreventDamageUntil(DamageWouldBeDealtToRecipient(recipient),
[PreventThatDamage], UntilEndOfTurn)` now supplies these references after
resolution. The permanent-source-filter variant of the second declaration keeps
its existing behavior; its filter does not declare individual source objects.
No new RON syntax or partial `ChooseADamageSource` adapter is introduced.
Other prevention/replacement contexts, spell references and command-zone sources
remain separate prerequisites for general source choice.


### Waiting references from resolved combat prevention

`Game.WaitingPreventionPermanentSources() []DamageSource` also includes exact
subjects retained by `PreventCombatDamageToAndBy`, `PreventCombatDamageBy`,
`PreventCombatDamageByThisTurn`, `PreventCombatDamageByEach`, and
`PreventCombatDamageByEachThisTurn`. Collection members are saved at resolution;
later members cannot enter the reference list. A to-and-by subject appears once.
The shared query and `ChooseDamageSourceFromWaitingPreventionReferences` retain
the existing identity, information, deduplication, filtering and binding contract.
Combat prevention remains available after matching damage. End-of-combat or
end-of-turn removal ends its eligibility according to its declared duration.

RON uses the existing `CreatePreventDamageUntil` lowering:
`CombatDamageWouldBeDealtByCreature(permanent)` for one source;
`Or([CombatDamageWouldBeDealtToRecipient(Permanent(permanent)),
CombatDamageWouldBeDealtByCreature(permanent)])` for a to-and-by subject;
and `CombatDamageWouldBeDealtByACreature(Or([SinglePermanent(permanent),
collection]))` for the supported combat relation collections. Operations must be
`[PreventThatDamage]`. Singular forms permit `UntilEndOfTurn`; supported
collection unions also permit `UntilEndOfCombat`. Existing scope checks remain.
There is no new spelling or general `ChooseADamageSource` adapter.

### Waiting references from typed static prevention

`Game.WaitingPreventionPermanentSources() []DamageSource` also reads the current
subject of `PreventCombatDamageFrom`, `PreventCombatDamageTo`,
`PreventDamageFromCardType`, and `PreventDamageFromBlockedCreatures`.
`This` resolves the live rule source; `AttachedPermanent` resolves its current
host. The source incarnation must still match. Phased or absent sources and
subjects supply no reference. Reattachment changes the subject. Ability removal
uses the existing layer lifecycle; removing a host's abilities does not remove
an Aura's rule. Queries remain read-only and return separate, deduplicated
snapshots. Source filters and blocked attackers do not declare extra subjects.

Existing RON `PreventDamage(event, [PreventThatDamage])` lowering is reused:
`CombatDamageWouldBeDealtByCreature(subject)`,
`CombatDamageWouldBeDealtToRecipient(Permanent(subject))`, their matching `Or`
union, and `CombatDamageWouldBeDealtByACreatureToRecipient(filter,
Permanent(subject))` use the static combat rules. The supported source-card-type
and blocked-creature events use the other two typed rules. Existing declaration,
scope and filter restrictions remain. This adds no new spelling or general
`ChooseADamageSource` adapter. Legacy filter-factory static prevention and other
waiting replacements still need separate reference contracts.

## Typed static prevention from matching permanent sources

`PreventDamageFromMatchingPermanents(subject PermanentRef, from PermanentFilter)
StaticRule` prevents combat and noncombat damage to its current subject. `This`
and `AttachedPermanent` use the existing static subject validation. A zero filter
permits any permanent source. Spell sources cannot match, even when their card
characteristics match the filter. Exact departed sources use the existing LKI
service; a returned card cannot supply the old source's information.

RON lowers the existing top-level `PreventDamage(
DamageWouldBeDealtByAPermanentToRecipient(source_filter,
Permanent(ThisPermanent)), [PreventThatDamage])` to this rule. The supported RON
context stays self-only; Aura hosts, event/target references, player recipients,
finite amounts and additional operations stay rejected in this event shape.
The native service permits a live attachment subject without extending RON syntax.

The rule supplies its exact live subject through
`Game.WaitingPreventionPermanentSources`. Source filters do not enumerate their
matching members. Current subject resolution, source incarnation validation,
phasing, ability removal, clones and replacement rechecks reuse typed static
prevention. Legacy `PreventDamageFromTo` callers keep their existing contract.

Tests: `TestStaticPermanentFilterPrevention*`,
`TestWaitingStaticPermanentFilterPreventionStaleCycle`, and the expanded
`TestRONWaitingStaticPreventionReferences`. Complete Argothian Pixies, Desert
Nomads and Uncle Istvan records retain `TestRONStaticPrevention*` gameplay evidence.


### Live destinations of waiting static damage redirection

`mage.Game.WaitingRedirectionPermanentSources() []mage.DamageSource` reads the
exact current destination of each active typed static damage redirection rule.
`mage.ChooseDamageSourceFromWaitingRedirectionReferences(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` saves one matching destination through the shared choice
service. This category is separate from prevention references and from complete
CR 609.7a source enumeration.

RON retains conditional `ReplaceWouldDealDamage` lowering for the permanent-source
and single-card-type source events described above. The destination plan permits
`ThisPermanent`, or `HostPermanent` in an Aura, and builds its typed reference in
each factory call. No resolving choice syntax or additional static scope is added.
A false condition, phased or departed source, missing host, or destination that
is neither a creature nor a planeswalker supplies no reference. A returned source
cannot restore a saved cycle rule. Reattachment supplies the new live host.
Queries neither execute damage nor test a hypothetical matching source.

Complete Veteran Bodyguard and Martyrs of Korlis records map to
`TestRONWaitingRedirectionReferences` and the existing `TestRONStaticRedirect*`
tests. Other replacement categories, source-choice shields, command objects and
complete `ChooseADamageSource` lowering remain separate capabilities.

### Exact-object source prevention prerequisite

`PreventNextDamageFromExactSource(source DamageSourceRef, recipient PlayerRef,
filter DamageSourceFilter) ResolvingEffect` saves one exact permanent incarnation
or spell stack identity at resolution. It prevents the next positive damage event
to that player this turn. All matching simultaneous packets use one shield.
Nonmatching properties, recipients, returned objects and recasts do not consume
it. Cleanup expires it; clones keep separate consumption state. The filter uses
current object information when available and retained information after departure.

The RON lowering prerequisite now uses `PreventNextDamageFromSelectedSource`
with the same typed arguments. It accepts
`CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(
TheChosenDamageSource, Player(player)), [PreventThatDamage])` only with an existing
typed `chosenDamageSourcePlan` in the lexical scope and its factory-local binding.
The plan supplies the choice filter. It rejects unbound references, permanent
recipients, extra operations and call forms of the two symbols. `Player(player)`
uses the existing typed player reference grammar.

No production declaration creates that chosen-source scope yet. The seven
protection records remain rejected at `ChooseADamageSource`. Tests lower their
shield expressions with explicit native producers. The selected-source service
continues an active shield from an exact spell stack identity to the exact
permanent that spell becomes (CR 400.7c, 609.7a). This includes a chosen spell
copy that becomes a token with a new card ID. It links the shield after successful
battlefield registration and before entry effects and events. Failed entry, a
countered spell, a separate copy, direct entry and a later cast cannot create
that link. The original spell and its continued permanent share one consumption
state. Current properties are checked before damage; departure and phasing use
the existing exact information services. Returned objects cannot inherit it.
The continued permanent joins waiting prevention references. Clones copy both
identities and keep separate consumption. Complete source enumeration must
precede production source-choice lowering. Other prevention operations and
recipients require separate contracts.

### Waiting prevention spell choice prerequisite

`Game.WaitingPreventionSpellSources()` reads exact spell identities held by active
typed source shields. `ChooseDamageSourceFromWaitingPreventionSpellReferences(
player, filter, binding)` selects from this category through the shared resolution
choice service. Current spell data or retained departure data supplies properties;
the saved shield snapshot is the fallback. Recasts and copies keep separate stack
identities. Consumption and cleanup remove the reference. A continued permanent
supplies a separate incarnation through the permanent-reference query.

RON `compileChosenDamageSourceChoice` validates exactly
`ChooseADamageSource(filter)` and produces an immutable `chosenDamageSourcePlan`.
The category lowering `waitingPreventionSpellChoice(instanceBindings)` uses the
plan's filter, `You`, and its own factory-local chosen-source binding. It supports
synthetic category consumers alongside the existing scoped
`CreateFuturePreventDamage` lowering. Invalid arity/filter and absent bindings
fail declaration validation. The production compiler still rejects general
`ChooseADamageSource`: remaining replacement references and face-up command
objects need contracts before complete enumeration can run. No partial registry
is published. This establishes subrule evidence for the seven protection records,
not complete-card gameplay or new collection compilation coverage.


### Exact subjects held by waiting regeneration shields

`Game.WaitingRegenerationPermanentSources() []DamageSource` supplies exact
permanent subjects of active regeneration shields in both replacement lists.
CR 609.7a permits these referred objects even after departure; CR 701.19a keeps
the shield waiting until destruction or cleanup. Phasing and an inability to
regenerate do not remove the reference. The query reads current characteristics
or exact retained departure information. Returned incarnations cannot replace
old subjects. Invalid identities are excluded, duplicates are merged in list
order, and query results contain copied card data. Clones retain separate shield
lifetimes. This query uses the shared waiting permanent-reference collector.

`ChooseDamageSourceFromWaitingRegenerationReferences(PlayerRef,
DamageSourceFilter, *DamageSourceBinding) ResolvingEffect` supplies this category
through the shared choice service. Pure filters read current or retained data.
Each choice writes only its own binding; no matching candidate clears that
binding. Definition validation checks the player, filter, binding and producer
order. It introduces no new targeting, payment, destruction or replacement
ordering behavior.

RON `chosenDamageSourcePlan.waitingRegenerationChoice(instanceBindings)` lowers
an already validated `ChooseADamageSource(filter)` plan with `You` and its
factory-local binding. The existing scoped `CreateFuturePreventDamage` consumes
that binding. Production general source choices remain rejected until all legal
categories are implemented. No partial registry is installed.

Evidence: `TestWaitingRegenerationReferences`, `TestWaitingRegenerationChoices`
and `TestWaitingRegenerationSeparateSubjects` cover lifecycle, retained data,
phasing, control, duplicate and invalid identities, independent choices, shield
counts, returned objects and clones. `TestRONWaitingRegeneration` executes full
Drudge Skeletons, Death Ward and Living Wall records through payment and
resolution, including illegal targets and retained-source damage. The protection
subrule test `TestRONWaitingRegenerationProtection` lowers all seven protection
records, chooses a departed Living Wall retained by its regeneration shield,
and checks independent prevention consumption in original and cloned games.
`TestRONWaitingRegenerationInvalid` checks invalid choice syntax. These are
additional interaction and subrule tests, not new full-card compilation coverage.

## RON optional draw-step skip for life

Permanent rules support the exact operation sequence
`ReplaceWouldBeginDrawStep(PlayerWouldBeginTheirDrawStep(You),
[MayCost(SkipThisDrawStep), If(CostWasPaid, [GainLife(Integer(n))])])`,
where `n` is a nonnegative integer literal. Each factory constructs
`Static(SkipDrawStepForLife(n))`. Other players, operations and dynamic amounts
remain declaration errors. The optional replacement skips the whole step;
the gain occurs first in the next step that actually occurs (CR 614.10b).
`CostWasPaid` is local to the replacement and supplies no action-list result.

`Game.WaitingDrawStepLifePermanentSources` and
`ChooseDamageSourceFromWaitingDrawStepLifeReferences` supply one damage-source
category from active draw-step life replacements. Execution and enumeration
share source incarnation, phasing and ability-lifecycle checks. Inactive rules
and pending life gains cannot supply former source objects. Queries use current
characteristics and control, copy their data and merge duplicates. The choice
uses typed player/filter/binding declarations and independent resolution results.
RON `chosenDamageSourcePlan.waitingDrawStepLifeChoice` lowers the validated
filter with `You` and the factory-local binding. Production general
`ChooseADamageSource` still rejects until all legal categories exist.

Tests: `TestRONDrawStepLifeFasting`, `TestRONDrawStepLifeFastingUpkeep`,
`TestRONDrawStepLifeInstances`, `TestRONDrawStepLifeProtection`,
`TestRONDrawStepLifeInvalid`, `TestRONDrawStepLifeFixedAmounts`,
`TestWaitingDrawStepLifeReferences` and `TestWaitingDrawStepLifeChoices`.
Full Fasting gameplay and protection subrule evidence are separate claims.

### Waiting counter-prevention subjects

`PreventDamageToSourceByRemovingCounters(counterType core.CounterType)
ContinuousEffect` keeps its existing public signature. Use it in a static
ability. `Game.WaitingPreventionPermanentSources() []DamageSource` now includes
its exact source while that ability remains active, even with zero counters.
The query excludes phased or departed sources, removed rules and stale rules
for returned objects.
It uses current control and returns independent information snapshots.

RON `chosenDamageSourcePlan.waitingPreventionChoice(instanceBindings)` lowers
validated `ChooseADamageSource(filter)` to the shared waiting-prevention choice
service with `You` and the instance's typed damage-source binding. This is a
category-specific prerequisite. General production `ChooseADamageSource` stays
rejected until complete legal-source enumeration is available.

### Exact permanent sources of legacy finite and half-damage shields

`Game.AddSourcePreventionShield(playerID, sourceID uuid.UUID, amount int)` and
`Game.AddHalfDamageFromSourcePreventionShield(playerID, sourceID uuid.UUID)`
retain the incarnation when `sourceID` is a live permanent at registration.
Their damage match requires that exact incarnation and positive damage to the
specified player. Return cannot transfer the shield to the new object.
Departure and phasing do not end a resolved shield (CR 400.7, 609.7a, 615).
The finite shield retains its remaining amount across damage events and the
shared simultaneous packet pipeline. The half shield prevents the amount rounded
down in its next matching packet and then ends. Existing ordering stays in the
shared replacement service; this change does not establish a new half-shield
simultaneous-event contract.

Both rules supply their exact permanent source to
`Game.WaitingPreventionPermanentSources() []DamageSource` in ordinary and cycle
replacement lists. Current or retained exact information, deduplication, copied
query snapshots and cloned lifetime use the existing collector. Consumption and
cleanup end eligibility. Nonpositive finite amounts install no rule. Registrations
without a live permanent preserve their existing ID-based matching but do not
supply a waiting permanent or spell reference. Typed spell shields keep their
separate services; no permanent-spell continuation is added here.

RON uses the existing `chosenDamageSourcePlan.waitingPreventionChoice` lowering
for validated `ChooseADamageSource(filter)`, its factory-local binding and
`compileExactSourcePrevention` for the chosen-source shield. No duplicate adapter
is required. The seven protection records have choice/prevention subrule tests
against both native shield forms, including departure, return and phasing.
`TestRONWaitingLegacyPreventionInvalid` checks malformed declarations and binding
order. Complete source enumeration and complete protection records remain
unsupported. `TestWaitingLegacyPrevention*` covers native lifetime, information,
return exclusion, clone isolation, finite simultaneous damage and odd half damage.

### Exact permanent sources held by legacy reverse-damage shields

`Game.AddReverseDamageShield(playerID, sourceID uuid.UUID)` now saves the
incarnation of a live permanent at registration. Positive damage to that player
must come from the same incarnation. Departure and phasing keep the shield;
a returned permanent cannot consume it. Nonpositive damage and a consumed
rule cannot match. The existing replacement order, prevention, consumption
before life gain, life-gain replacements and event/history services remain in use.
This change adds no new simultaneous-event or unpreventable-damage contract.

The source joins `Game.WaitingPreventionPermanentSources() []DamageSource`.
The shared collector reads current or exact departure information, including
after LKI cleanup, and deduplicates ordinary and cycle replacement lists.
Queries return independent snapshots. Consumption and end-of-turn cleanup end
eligibility; game clones retain independent lifetimes. Unknown/nonpermanent IDs
retain their existing matching behavior but supply no exact waiting reference.
No spell capture or permanent-spell continuation is established by this API.

RON reuses `compileChosenDamageSourceChoice`, the per-instance
`chosenDamageSourcePlan.waitingPreventionChoice(instanceBindings)` lowering,
and `compileExactSourcePrevention` for the chosen-source consumer. Its supported
category now includes these shield references. `TestRONWaitingReverseDamageProtection`
executes the complete choice/prevention subrules from the six Circle of Protection
names and Greater Realm of Preservation with live, departed, returned and phased
sources and cloned games. `TestRONWaitingReverseDamageInvalid` checks malformed
choices, missing bindings and consumer-before-producer order.
`TestWaitingReverseDamageReferences`, `TestWaitingReverseDamageBoundaries` and
`TestWaitingReverseDamageDeduplication` cover native identity, information,
consumption, histories, compatibility and cloning. Existing reverse-damage and
life-outcome tests cover life-gain replacements and nested damage.

CR 400.7, 609.7a, 615.4–8 and 702.26 support the lifetime and reference contract.
Complete source enumeration remains unsupported. The seven protection names gain
subrule evidence only; the full Reverse Damage record, including its prevented
amount result, is not implemented by this change.

### Temporary exact permanent-source redirection

`mage.RedirectDamageFromPermanentThisTurn(source mage.PermanentRef,
player mage.PlayerRef, destination mage.PermanentRef) mage.ResolvingEffect`
captures all three references at resolution. It redirects each positive damage
packet from that exact source to that saved player until cleanup. The original
destination must be present as a creature or planeswalker when damage occurs.
Control changes do not change the saved player; a phase cycle keeps identity;
a returned source or destination cannot inherit the effect. Unbound identities at
installation produce no rule. Captured identities permit installation after
departure or phasing, without making those objects available for damage. Source information on a pending damage packet can
survive source departure. The effect is unlimited and is not consumed by damage.

RON supports the resolving action shape
`CreateReplaceWouldDealDamageUntil(DamageWouldBeDealtByPermanentToRecipient(
<permanent reference>, Player(<player reference>)),
[DealToCreatureOrPlaneswalkerInstead(<permanent reference>)], UntilEndOfTurn)`.
References use the existing lexical scopes and fresh factory bindings. Other
events, optional operations, collections and durations remain rejected.
Complete Shimian Night Stalker gameplay maps to `TestRONTemporaryRedirectGameplay`
and `TestRONTemporaryRedirectCopyAndInstances`.

`Game.WaitingRedirectionPermanentSources` also reads the saved source and
destination of this typed rule. Both remain referred objects until cleanup,
even when the destination cannot currently receive redirected damage. Current
or exact retained information supplies choice filters. The existing collector
deduplicates identities and returns isolated snapshots.
`chosenDamageSourcePlan.waitingRedirectionChoice(instanceBindings)` connects
this category to the existing RON choice/prevention plans. It is category-only;
general `ChooseADamageSource` stays rejected pending complete enumeration.
Legacy ID-only redirection installers retain their behavior and are not enumerated.

### Selected-source permanent-recipient redirection

The resolving RON shape is:

```text
CreateFutureReplaceWouldDealDamage(
  NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(
    TheChosenDamageSource, Permanent(<permanent reference>)),
  [DealToPlayerInstead(<player reference>)])
```

`compileSelectedSourceRedirect(Node, scope) (effectFactory, error)` requires a
lexically bound source-choice plan, one typed permanent recipient and exactly
one player-redirection operation. Each factory uses its own source binding and
existing target/player bindings. It lowers to
`mage.RedirectNextDamageFromSelectedSource(DamageSourceRef, PermanentRef,
PlayerRef, DamageSourceFilter) ResolvingEffect`. The choice plan supplies the
source filter for installation and damage-time rechecks.

The engine keeps exact identities, saves the destination player, redirects one
positive damage event until cleanup and continues selected permanent spells
before entry effects. All matching simultaneous packets use the rule together.
Other events, recipients, operations, unbound references and wrong arities remain
errors. `ChooseADamageSource` remains rejected in production until all its source
categories exist; the adapter does not provide a partial source list.

`Game.WaitingRedirectionSpellSources` supplies exact selected spells to the shared
waiting-reference choice service alongside existing permanent redirection refs.
Native and RON tests are mapped in `docs/engine-rules-contracts.md`. No RON source
record or vocabulary hash changed.

### Face-up command-object damage sources

`mage.ChooseDamageSourceFromCommandObjects(player mage.PlayerRef,
filter mage.DamageSourceFilter, binding *mage.DamageSourceBinding)
mage.ResolvingEffect` uses only face-up engine objects installed in the command
zone. The choice occurs at resolution and does not target. An object need not
have a damage ability. Invalid declarations fail definition validation; empty
choices and absent players clear only this result binding.

`Game.AddCommandObject(Card, bool) (uuid.UUID, error)` supplies a fresh visit ID.
The caller must remove the object from its previous zone first. Nil objects,
missing IDs and duplicate command IDs fail. `RemoveCommandObject(visit)` and
`SetCommandObjectFaceDown(visit, bool)` use that visit, so an old reference cannot
remove or change a returned object. These services do not grant casting
permission or install command abilities. `Game.CommandDamageSources()` returns
isolated face-up snapshots in insertion order. `DamageSource.CommandVisit()`
keeps command entries separate from permanent incarnations and stack spells.
Departure information and game clones keep exact visits separate.

RON `chosenDamageSourcePlan.commandChoice(instanceBindings)` lowers the validated
`ChooseADamageSource(filter)` AST to this engine service with `You` and its own
binding. Exact and selected-source prevention and selected-source redirection
accept these identities and use the existing damage and replacement pipeline.
This is one source category. Complete source enumeration, including explicit
references held by waiting rules, still precedes unrestricted production RON
source choices. No full-card unlock or production registry change is claimed.

### Waiting command visits in prevention and redirection

`Game.WaitingPreventionCommandSources() []DamageSource` reads exact command
visits from active exact-source and selected-source prevention shields.
`Game.WaitingRedirectionCommandSources() []DamageSource` reads the selected
command source of active typed redirection rules. The collector examines both
ordinary and cycle replacement storage, excludes other identity kinds and
invalid identities, deduplicates visits, and returns isolated snapshots.

CR 609.7a permits these references after departure. CR 400.7 keeps a returned
object separate. Current or exact departure information supplies properties;
hidden objects expose no characteristics. A retained reference does not require
the redirection recipient to remain present or unphased. Consumption and cleanup
remove availability. Existing replacement ordering and damage batches apply.

`ChooseDamageSourceFromWaitingCommandReferences(player PlayerRef,
filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect` chooses
from both query results at resolution through the shared choice service.
Definition validation checks player, filter, binding and producer order.
Nonmatching or empty queries and absent players clear only that result binding.
Game clones retain independent references and consumption.

RON `chosenDamageSourcePlan.waitingCommandChoice(instanceBindings)
mage.ResolvingEffect` lowers validated `ChooseADamageSource(filter)` with `You`
and the plan's per-instance binding. Full source choices stay rejected until the
remaining explicit-reference audit and category composition are complete.
No RON record or source hash changes.

Evidence: native `TestWaitingCommandSources`, `VisibilityAndCleanup`,
`Declarations`, and `ExcludeOtherIdentities` (all use the
`TestWaitingCommandSources` prefix); RON `TestRONWaitingCommandSourceProtection`,
`Invalid`, and `TimingAndCopies`. The RON tests use the complete records of six
Circles and Greater Realm of Preservation for their actual choice/prevention
bodies, plus the Red Circle's activation cost. These tests establish subrule
coverage, not full-card acceptance or complete gameplay validation.

### Waiting callback modifier source choice

`mage.ChooseDamageSourceFromWaitingModifierReferences(player, filter, binding)`
chooses among the explicit sources of active registered mill/life-gain modifiers
and their static factories. `Game.WaitingModifierPermanentSources()` reads that
category without calling callbacks or factories. Registered sources need their
saved live unphased incarnation; static sources also need their current ability.
The result uses the existing typed `DamageSourceBinding` and shared source filters.

RON `ChooseADamageSource(filter)` has a category adapter
`chosenDamageSourcePlan.waitingModifierChoice(instanceBindings)` for this context.
It uses `You` and the plan's instance binding. This prerequisite does not accept
unrestricted production source choices. Objects hidden in callback closures and
other legal categories need separate reference contracts.


### Waiting empty-library replacement source choice

`mage.ChooseDamageSourceFromWaitingEmptyLibraryReferences(player, filter, binding)`
chooses among the explicit sources of active empty-library draw replacements.
`Game.WaitingEmptyLibraryPermanentSources()` reads this category without calling
callbacks. Registration saves the current live source incarnation. A phased,
departed or returned source cannot supply the old registration. Phase-in restores
the same registration; a returned source requires a fresh registration.

The RON `chosenDamageSourcePlan.waitingEmptyLibraryChoice(instanceBindings)`
adapter lowers validated `ChooseADamageSource(filter)` with `You` and its own
binding. It completes this category only. Full production source choices remain
rejected until the legal source list is complete.

### Persistent legacy replacement source references

`mage.ChooseDamageSourceFromWaitingPersistentReplacementReferences(player,
filter, binding)` supplies one CR 609.7a source category.
`Game.WaitingPersistentReplacementPermanentSources()` reads the exact explicit
sources of registered life-gain-to-draw, sanctuary normal-draw and counter-doubling
replacements. General filters supply no matching-object references. Queries do
not execute replacements, make choices or modify registrations.

`SetLichActive(playerID, sourceID)`,
`AddIslandSanctuaryReplacement(playerID, sourceID)` and
`AddCounterDoubler(sourceID, counterType, filter)` save the live incarnation at
registration. Execution and enumeration require that incarnation to remain live
and unphased. Phasing suspends the registration; phase-in restores it. Departure
ends it, and a return needs a new registration. Registration before entry cannot
follow a later entry. These separate registrations keep their existing explicit
player and filter contracts; source ability removal does not remove them.

RON `ChooseADamageSource(filter)` has a validated per-instance
`waitingPersistentReplacementChoice` lowering for this category. Full production
source choices remain rejected until the complete legal list is established.
Evidence: native `TestWaitingPersistentReplacementSources*` and RON
`TestRONWaitingPersistentReplacementSource*`. The latter use full-record costs
and choice/prevention ASTs for six Circles and Greater Realm of Preservation in
a test-only category context. They do not establish full-card loading.

### Typed untap-step replacement source references

`mage.ChooseDamageSourceFromWaitingUntapSkipReferences(player, filter, binding)`
supplies the explicit source category of active typed `SkipUntapSteps` rules.
`Game.WaitingUntapSkipPermanentSources() []DamageSource` reads both replacement
lists without running replacements, choices, factories or layer passes. Each
source must keep the registered incarnation and the matching live static ability.
Phasing suspends availability; phase-in restores it. Departure and ability
removal stop it; a returned permanent needs its new static registration.

RON `chosenDamageSourcePlan.waitingUntapSkipChoice(instanceBindings)` lowers
validated `ChooseADamageSource(filter)` with `You` and its per-instance binding.
The common source filter and definition validation still apply. Empty or
nonmatching lists and absent players clear only that result. This category does
not complete CR 609.7a enumeration; production declarations remain rejected.
Evidence: `TestWaitingUntapSkipSources*` and
`TestRONWaitingUntapSkipSource*`, including actual RON Stasis as a source and
full-record protection costs and prevention ASTs in a test-only category context.

### Typed discard replacement source references

`mage.DiscardToLibraryReplacement() ContinuousEffect` retains its existing
calling form and now constructs a typed ability-layer effect. Its replacement
requires the original live, unphased incarnation and matching static ability.
Each effect-caused discard reads the current source controller. Declining keeps
the normal discard; cost discards cannot use the optional replacement.

`Game.WaitingDiscardPermanentSources() []DamageSource` reads exact explicit
sources from both replacement lists without choices, callbacks or layer passes.
`mage.ChooseDamageSourceFromWaitingDiscardReferences(player, filter, binding)`
uses shared source filters, decisions and resolution bindings.
RON `chosenDamageSourcePlan.waitingDiscardChoice(instanceBindings)` lowers
validated `ChooseADamageSource(filter)` with `You` and its instance binding.
Empty lists and absent players clear only that binding. Complete production
source choices still reject until the remaining CR 609.7a audit is complete.
This does not lower Library of Leng's `ReplaceWouldDiscard` declaration.

Evidence: `TestWaitingDiscardSourcesLifecycle`, `Choices`, `Execution` and
`InvalidRegistrations`; `TestRONWaitingDiscardSourceProtection`, `Invalid` and
`TimingAndRechecks`. Seven protection records supply their complete cost,
choice and prevention ASTs in a test-only category context. This establishes
category gameplay evidence, not complete production card loading.

### Exact subjects of waiting exile replacements

`Game.WaitingExileSources() []DamageSource` reads exact spell and permanent
subjects of active `AddExileIfWouldGoToGraveyardThisTurn` rules. A live or
resolving spell registration saves its stack identity. Its continuation saves
only the permanent that spell becomes, before entry effects run. Departure and
phasing retain both references until cleanup. Queries use current or retained
information, deduplicate exact identities and return separate snapshots.
Legacy registrations before casting retain their calling form and card marker;
they supply no exact waiting reference.

`mage.ChooseDamageSourceFromWaitingExileReferences(player, filter, binding)`
uses shared source filters, decisions, declaration validation and resolution
frames. RON `chosenDamageSourcePlan.waitingExileChoice(instanceBindings)` lowers
validated `ChooseADamageSource(filter)` with `You` and its instance binding.
Missing or nonmatching subjects and absent players clear only that result.
Full production source choices still reject until the remaining audit is complete.

Evidence: `TestWaitingExileSources*` and `TestRONWaitingExileSource*`.
The seven protection names supply their complete activation cost, choice and
prevention ASTs in a test-only category context. The spell test also proves
selected-source prevention continuation through the normal engine entry path.
This is category gameplay evidence; it does not establish full-card loading.

## RON conditional type and equal base values

A conditional static body may contain one
`PermanentLayerEffect(ThisPermanent | HostPermanent, [modifications])` with
one `SetPowerAndToughnessBoth(value)` and one or more
`AddCardtype(Artifact | Creature)` operations, including Creature. HostPermanent
requires an Aura declaration. The existing live static P/T value grammar
applies, including `ManaValueOfPermanent(HostPermanent)`. Other modifications,
duplicate base declarations, unbound references and resolution-only values
reject before factory publication. Other mixed body rules still reject.

The compiler lowers the complete list to one `Static(WhileWithSharedSubjects(condition, ...))`
rule. The engine selects the condition and exact subjects before the first
layer, retains them for the pass, and reads values in their applicable layer.
Animate Artifact therefore keeps its base P/T after its own type addition
makes the host a creature. A host that was already a creature is unaffected.
A fresh pass checks the condition again. This does not add sticky effects or
collective dynamic base-value lowering.

### RON static bands-with-other grants

The compiler accepts
`EachPermanentLayerEffect(filter, [AddAbility([BandsWithOthers(quality)])])`.
The collection selector uses existing static permanent filters, including
`ControlledByAPlayer(SinglePlayer(You))`. The quality is a pure permanent filter;
source/controller bindings, target references and empty compositions are
rejected. `And`, `IsSupertype`, `IsCardtype` and `IsNamed(NamedCard("name"))`
use their existing validated filter plans. Each factory builds its own grant.

This shape lowers to `GrantEachBandsWithOther(PermanentSet, PermanentFilter)`
inside `Static(While(...))`. The engine checks qualities at band formation and
combat damage assignment. All members of a quality band must match the quality;
only one matching member must hold the ability. Static land recipients must
also match the grant's color and current controller selection.

The supported context is an unconditional static collection layer with one or
more bands-with-other grants. Combined unrelated modifications, Aura/singular
subjects, printed declarations, token abilities and temporary grants remain
declaration errors. Ordinary banding removal uses the shared engine rule and
removes bands-with-other grants too.

### RON selective bands-with-other removal

`CreatePermanentLayerEffectUntil(subject, [LosesAbility(AnyBandsWithOthers)],
UntilEndOfTurn)` lowers to `RemoveBandsWithOther(PermanentRef)` in the existing
singular resolving modification context. `AnyBandsWithOthers` is a symbol;
call forms and use in `AddAbility` are invalid. Typed subject scope checks and
supported duration checks apply. Static collection removal remains unsupported.

The operation removes every quality grant in layer 6, preserves ordinary
banding, and uses the saved subject incarnation until cleanup. A later layer
grant can restore the ability. A formed band persists, while damage assignment
checks the abilities that remain. Shelkin Brownie uses selective removal;
Tolaria combines it with ordinary banding removal and an upkeep restriction.

### Precast exile references in damage-source choices

`mage.Game.CaptureExiledDamageSource(visit)` and
`mage.Game.StackBoundExileSources()` expose exact exile visits and references
saved by stack card/card-set payment bindings or nested event-card captures.
Use `mage.ChooseDamageSourceFromStackExileBindings(player, filter, binding)`
only for that category. A face-down card supplies no public characteristics.
Departure preserves the old visit's information; a return creates a new object.
A choice of the exile object does not follow a later spell or permanent.
`GrantCastFromExile` keeps its existing signature and grants permission only for
the present exile visit; departure or re-exile ends that permission.

RON's private `chosenDamageSourcePlan.stackExileChoice(instanceBindings)`
adapter consumes validated `ChooseADamageSource(filter)` plans with separate
instance bindings. It shares the existing exact-source prevention lowering.
General source choice remains unsupported in complete RON records until the
full CR 609.7a source list and remaining reference contracts are established.

### Precast graveyard references on the stack

`mage.Game.CaptureGraveyardDamageSource(visit)` and
`mage.Game.StackGraveyardSources()` read exact graveyard visits referred to by
stack targets, card/card-set payments and nested event-card captures.
`mage.ChooseDamageSourceFromStackGraveyardReferences(player, filter, binding)`
uses only that category. Current facts are read while the visit is present;
departure facts stay separate from returned cards. Copies retain original target
visits; retargeting captures the new target. Final referring stack departure
ends category availability. A graveyard source does not become a later spell
or permanent source. Game clones and queries keep separate mutable facts.

RON's private `chosenDamageSourcePlan.stackGraveyardChoice(instanceBindings)`
lowers `ChooseADamageSource(filter)` with separate instance bindings and the
existing selected-source prevention body. General production source choice still
rejects until the complete CR 609.7a list is supported. Waiting references and
delayed card frames remain separate contexts.

## RON hand source references on the stack

Validated `ChooseADamageSource(filter)` has an internal
`chosenDamageSourcePlan.stackHandChoice(instanceBindings)` lowering for exact
hand visits in singular/plural payment card bindings and nested event captures
on stack objects. It calls
`mage.ChooseDamageSourceFromStackHandReferences(mage.You, filter, binding)`.
The separate instance binding is consumed by the existing
`CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(
TheChosenDamageSource, Player(You)), [PreventThatDamage])` lowering.

Hidden hands supply no public characteristics. Public hands use current
visibility; saved departures keep only the facts public at departure. Cost
reveals keep their exact card public until the paid stack object leaves, including
resolution. Copies retain cost references without extending the original reveal. Returned cards, later spells
and permanents keep separate identities. Invalid declarations and premature
reads reject. Empty choices clear the result. General production source choice
remains unsupported until complete enumeration and its opaque-reference audit.


## RON choices with private hand inspection

Validated `ChooseADamageSource(filter)` plans also have a private
`chosenDamageSourcePlan.inspectableStackHandChoice(instanceBindings)` lowering.
It calls `mage.ChooseDamageSourceFromInspectableStackHandReferences` with `You`
and the instance's own binding. The choosing player can inspect their referenced
hand cards; other players receive public information only. Filters use that
player's information at resolution. Inspection does not reveal the hand.

`mage.Game.CaptureHandDamageSourceFor(viewer, visit)` reads a live visit and
`mage.Game.StackHandSourcesFor(viewer)` reads the existing stack-reference
category. Departures retain private facts and separate public visibility.
Queries and clones preserve exact visits and separate mutable snapshots.
The original public query and choice APIs remain available.

The chosen-source future prevention and native selected-source redirection
recheck actual hand characteristics through the engine's rules reader.
A private matching choice can therefore install a shield without revealing the
card. Returned visits, later spells and permanents remain separate objects.
Invalid declarations, missing bindings and reads before producers still reject;
empty choices clear the binding. Complete production `ChooseADamageSource`
remains unsupported pending the remaining reference categories and aggregation.


## Waiting damage-rule exile references

`Game.WaitingPreventionExileSources() []DamageSource` and
`Game.WaitingRedirectionExileSources() []DamageSource` read exact exile visits
saved by active exact/selected-source prevention and selected-source redirection.
Both reuse the command category's shared active-replacement walker and the
existing exile information service. They do not execute replacement callbacks.
Queries deduplicate exact visits and return separate mutable snapshots.
Consumption ends availability; end-of-turn cleanup removes temporary rules.
Cycle registrations keep their existing layer-cycle lifetime.

`ChooseDamageSourceFromWaitingPrecastExileReferences(player PlayerRef,
filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect` combines
these categories and resolves the choosing player at resolution. An absent player
or empty matching set clears the result. Definition validation rejects invalid
players, filters, bindings and reads before the producer. Other categories,
including exile-instead replacement subjects, retain their separate APIs.

Current exile facts apply while the visit remains. Departure retains the facts
at that visit's departure. Face-down exile has no characteristics, even when a
player can inspect the card. A later visit, spell or permanent cannot inherit
the source identity (CR 400.7, 406.3a, 609.7a–b). Existing damage replacement
ordering, rechecks, simultaneous packets, shield consumption and clone services
own execution; this query adds no replacement policy or zone transition.

RON `chosenDamageSourcePlan.waitingPrecastExileChoice(instanceBindings)` lowers
validated `ChooseADamageSource(filter)` to this category with `You` and a fresh
instance binding. Complete production enumeration remains rejected until other
required contexts are complete. No new RON spelling or source edit is needed.

Evidence: `TestWaitingExileSources`, `Declarations`, `VisibilityAndBoundaries`
and `RechecksAndBatch`; `TestRONWaitingPrecastExileProtection` maps the real
choice, payment and future-prevention declarations of all six Circles of
Protection and Greater Realm of Preservation to live/departed/returned, copied
and cloned waiting references. `TestRONWaitingPrecastExileInvalid` covers
scope/declaration rejection. These are subrule mappings, not full-card coverage.


## Waiting damage-rule hand references

`Game.WaitingPreventionHandSourcesFor(viewer uuid.UUID) []DamageSource` and
`Game.WaitingRedirectionHandSourcesFor(viewer uuid.UUID) []DamageSource` read
exact hand visits retained by active exact/selected-source prevention and
selected-source redirection. The shared replacement walker checks ordinary and
cycle rule lifetime without calling replacement callbacks, excludes malformed
and other-zone identities, deduplicates visits, and returns separate snapshots.
Consumption and cleanup end availability; cycle rules retain their layer lifetime.
No stack reference is required once the waiting rule retains the visit.

The existing hand information reader supplies current facts with the viewer's
inspection rights. The hand player can inspect private characteristics; other,
nil and unknown viewers receive public facts only. Departure keeps that visit's
private facts and separate public visibility. Later reveals and returns cannot
rewrite it. A waiting reference does not extend a cost reveal after the paying
stack object leaves. Inspection adds no unreferenced card to the source list.

`ChooseDamageSourceFromWaitingPrecastHandReferences(player PlayerRef,
filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect` combines
both categories using the choosing player's information at resolution, even when
that player differs from the controller. Empty matches or an absent player clear
the result. Definition validation checks player, filter, binding and producer
order. Options and saved choices keep separate mutable data.

RON `chosenDamageSourcePlan.waitingPrecastHandChoice(instanceBindings)` lowers
validated `ChooseADamageSource(filter)` with `You` and the instance binding.
Existing future prevention consumes it. General production enumeration still
rejects pending other categories; this API supplies only waiting hand references.
The engine's rules information reader still rechecks actual characteristics for
prevention and redirection. Hidden information does not stop an installed shield.
A returned visit, later spell or permanent cannot inherit it. Replacement order,
simultaneous damage, consumption, phasing and clone behavior reuse existing rules.

Rules: CR 400.2/400.7, 402.3, 609.7a–b, 615.9 and 701.20a.
Evidence: native `TestWaitingHandSources`, `Declarations`,
`VisibilityAndBoundaries`, `RechecksAndBatch`, `Inspection` and `RevealLifetime`;
RON `TestRONWaitingPrecastHandProtection` and `Invalid`. All six Circles of
Protection and Greater Realm of Preservation map their real payment, choice and
future-prevention declarations to this waiting-reference context. These mappings
establish subrule gameplay, not full-card acceptance.

## RON source choices from delayed hand and exile card frames

Validated `ChooseADamageSource(filter)` plans have an internal
`chosenDamageSourcePlan.waitingDelayedCardChoice(instanceBindings)` lowering.
It calls `mage.ChooseDamageSourceFromWaitingDelayedCardBindings(mage.You,
filter, binding)` with a separate binding for the factory instance. Existing
`CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(
TheChosenDamageSource, Player(You)), [PreventThatDamage])` consumes that binding.

The source category includes singular and plural hand/exile card bindings held
by waiting delayed bodies. Existing `CreateFutureTrigger/2` lowering reuses
`AtNext`, which now saves these frames. Trigger placement transfers saved card
references to stack queries. Each delayed execution restores its card frame
without changing the outer frame. Hand choices use the choosing player's private
inspection rights; other hands and face-down exile keep hidden characteristics.
Departure retains exact visit facts, and a returned card cannot replace them.
A saved reference cannot extend a cost reveal. Empty choices clear their binding;
invalid declarations and premature reads reject through shared validation.

`TestRONWaitingDelayedCardProtection` maps these declarations to Circle of
Protection: Black, Blue, Green, Red, White, Artifacts and Greater Realm of
Preservation. It executes hand draw-event and exile payment producers, RON
next-end-step bodies and paid protection activations across live, departed,
returned, copied and cloned contexts. `TestRONWaitingDelayedCardInvalid` covers
invalid syntax and binding declarations. General production source choice stays
rejected until all CR 609.7a categories and the opaque-reference audit are complete.
Waiting graveyard card frames have a separate contract below. Remaining precast
target contexts still prevent a complete legal source list.

## RON source choices from delayed graveyard card frames

Validated `ChooseADamageSource(filter)` plans support internal
`chosenDamageSourcePlan.waitingDelayedGraveyardChoice(instanceBindings)` lowering
to `mage.ChooseDamageSourceFromWaitingDelayedGraveyardBindings(mage.You,
filter, binding)`. Each instance supplies its own binding. Existing
`CreateFuturePreventDamage(NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient(
TheChosenDamageSource, Player(You)), [PreventThatDamage])` consumes the result.

`mage.Game.WaitingDelayedGraveyardSources() []mage.DamageSource` supplies exact
singular/plural graveyard visits saved by waiting delayed bodies and modes.
Current public facts or retained departure facts cannot follow a returned card.
Existing `CreateFutureTrigger/2` lowering uses shared card capture and restoration.
Detection, placement and final stack departure retain their existing lifetimes.

`TestRONWaitingDelayedGraveyardProtection` maps all six Circles of Protection and
Greater Realm of Preservation to this context using a real discard payment,
compiled RON delayed body, real protection payment and prevention declarations.
It covers live, departed, returned, copied and cloned choices, resolution timing,
failed payments, shield consumption and empty-category clearing. `Invalid`
checks malformed syntax, missing instance bindings and premature reads.
General production source choice remains rejected until all remaining CR 609.7a
categories and opaque references are audited. No full-card unlock is claimed.


## Exact hand-target source references

`Game.StackHandSources() []DamageSource` and
`Game.StackHandSourcesFor(viewer uuid.UUID) []DamageSource` now include exact
hand targets, alongside their existing payment/event/reveal references.
Announcement captures the physical card ID, hand visit and zone before payment.
A target discarded as a cost keeps its hand visit; it supplies no graveyard
reference. Failed payment installs no target frame. Stack insertion retains the
saved visit and also captures targets for direct stack insertion. Target
incarnations also use the hand visit for resolution legality. An absent visit
or a conflicting saved incarnation supplies no source. Queries only read
identities listed in the stack object's current targets; old retarget snapshots
cannot add choices. Sorting and exact deduplication use the existing reader.

A live visit supplies current facts. Departure supplies the same visit's saved
facts and public visibility. The hand player can inspect private facts; other
players receive only public information. Targeting does not reveal a hand card.
A later reveal cannot expose a hidden departed visit. Queries return separate
snapshots. No source continues to a returned card, spell or permanent.

Copies retain exact targets. An explicit legal new choice saves its current
visit, including a returned card with the same physical ID. Cancellation keeps
the original visit. Copy maps are separate; game clones share immutable target
identities through copy-on-write. The original returned hand target is illegal
at resolution. Illegal targets remain available as source references until the
last referring stack object leaves. Resolving hand frames retain these target
references until resolution ends, including cloned frames.

RON `chosenDamageSourcePlan.inspectableStackHandChoice(instanceBindings)`
reuses `ChooseDamageSourceFromInspectableStackHandReferences(You, filter,
binding)` for validated `ChooseADamageSource(filter)` in this context. Each
instance retains its own binding. The existing exact future-prevention adapter
consumes it. The public `stackHandChoice` lowering remains available. General
production source choice remains rejected pending complete source aggregation.
No additional source-choice syntax or targeting declaration is introduced.

Rules: CR 400.7, 402.3, 601.2c, 608.2b, 609.7a–b and 707.10c.
Phasing does not apply to hand cards. Payment rollback, damage batches,
replacement ordering, source-property rechecks and shield consumption use their
existing engine services.

Evidence: native `TestHandTargetSourcesIdentity`, `LegalityAndRetarget`,
`Boundaries`, `Visibility`, `ResolutionAndClone`, `SpellCopy`, and
`PaymentDeparture` and `FailedPayment`; RON
`TestRONHandTargetSourcesProtection` and `Invalid`. The RON test maps all six
Circles of Protection and Greater Realm of Preservation to their complete real
cost/choice/future-prevention declarations in this hand-target context. It does
not claim complete-card gameplay or compilation gains.
