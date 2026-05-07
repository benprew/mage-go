# Writing Cards for mage-go — Video Slideshow & Script

> **Format:** YouTube tutorial (~60-90 min). Each "slide" section below is a visual beat.
> Speaker notes are in *italic*. Code blocks are shown on screen. Diagrams are described
> in `[DIAGRAM: ...]` annotations for the editor to build.

---

## SLIDE 0 — Title Card

```
╔══════════════════════════════════════════╗
║   Writing Cards for mage-go             ║
║   A deep dive into the MTG rules engine ║
║                                         ║
║   From Grizzly Bears to custom events   ║
╚══════════════════════════════════════════╝
```

*Welcome! Today we're going to build Magic: The Gathering cards in Go — starting
from the simplest vanilla creature and ending with custom engine events and
replacement effects. Everything is test-driven. Every card we write, we write
the test first.*

---

## SLIDE 1 — Four Rules

[BUILD: reveal one at a time]

1. **Follow Oracle text exactly**
2. **Never simplify**
3. **TDD — test first**
4. **Consult XMage**

*These are non-negotiable. If Oracle says "nontoken creature an opponent controls
with power 3 or greater" — you check every single word. A simplified implementation
is a wrong implementation.*

---

## Part 1: The Basics

## SLIDE 2 — Section: The Basics

*Constructors, options, registration.*

---

## SLIDE 3 — Grizzly Bears: Oracle

```
Grizzly Bears     {1}{G}
Creature — Bear
              2/2
```

*The simplest possible card. No abilities, just a body.*

---

## SLIDE 4 — Grizzly Bears: Test

```go
func TestGrizzlyBears(t *testing.T) {
    g := gametest.NewTestGame(t)
    g.AddCard(core.ZoneBattlefield,
        gametest.PlayerA, "Grizzly Bears")
    g.StopAt(1, core.BeginCombat)
    g.Execute()

    g.AssertPowerToughness(                    // ← HIGHLIGHT
        gametest.PlayerA, "Grizzly Bears", 2, 2)
}
```

*NewTestGame gives you two players at 20 life. AddCard places cards. StopAt freezes.
Execute runs. Then assert.*

---

## SLIDE 5 — Grizzly Bears: Implementation

```go
mage.Register("Grizzly Bears", func() mage.Card {
    return mage.NewCreature(
        "Grizzly Bears",
        "{1}{G}",
        2, 2,
        mage.WithSubTypes("Bear"),
    )
})
```

*Five lines. Register takes a name and factory. NewCreature takes name, cost, P/T, options.*

---

## SLIDE 6 — Registration Diagram

[DIAGRAM: init() → Register(name, factory) → DefaultRegistry → CreateCard(name)]

```go
// cards/limited/test.go — ensures init() runs
var _ = registerCreatures
var _ = registerSpells
```

*Without the blank-id references, Go's test binary won't link the card package.*

---

## SLIDE 7 — Constructors & Options (split)

**Left column: Constructors**
```
NewCreature(name, cost, p, t, ...opts)
NewInstant(name, cost, spell, ...opts)
NewSorcery(name, cost, spell, ...opts)
NewEnchantment(name, cost, ...opts)
NewAura(name, cost, ...opts)
NewArtifact(name, cost, ...opts)
NewEquipment(name, cost, ...opts)
NewLand(name, ...opts)
NewToken(name, p, t, types, ...)
```

Template helpers: `NewLuckyCharm`, `NewLandDestruction`, `NewBoostAura`

**Right column: CardOption Functions**
```
WithSubTypes("Human", "Soldier")
WithSuperTypes(core.Legendary)
WithKeyword(core.Flying)
WithCardType(core.TypeArtifact)
WithManaAbility(core.Green)
WithAnyColorMana()
WithAdditionalCost(cost)
WithExpansion("Arabian Nights")
WithCumulativeUpkeep("{1}")
WithActivatedAbility(e, c, ...o)
WithStaticAbility(effects...)
WithAbility(ability)
WithETBEffect(effect)
```

*Pick the constructor. Customize with options. No inheritance.*

---

## SLIDE 8 — Serra Angel (split)

**Left: Oracle + Code**
```
Serra Angel     {3}{W}{W}
Creature — Angel
Flying, vigilance
              4/4
```

```go
mage.NewCreature("Serra Angel", "{3}{W}{W}", 4, 4,
    mage.WithSubTypes("Angel"),
    mage.WithKeyword(core.Flying),      // ← HIGHLIGHT
    mage.WithKeyword(core.Vigilance),   // ← HIGHLIGHT
)
```

**Right: Behavior Test**
```go
// Vigilance means she doesn't tap
g.Attack(1, gametest.PlayerA, "Serra Angel")
g.StopAt(1, core.EndStep)
g.Execute()

g.AssertTapped(gametest.PlayerA, "Serra Angel", false) // ← HIGHLIGHT
g.AssertLife(gametest.PlayerB, 16)
```

*Test what the keyword DOES, not just that it exists.*

---

## Part 2: Spells & Effects

## SLIDE 9 — Section: Spells & Effects

*The atomic building blocks.*

---

## SLIDE 10 — Lightning Bolt Pipeline

[DIAGRAM: Target + Effect = Spell]

```go
mage.NewInstant("Lightning Bolt", "{R}",
    mage.NewTargetedSpell(
        mage.TargetAnyTarget(),           // ← HIGHLIGHT
        mage.DealDamage(mage.Fixed(3)),   // ← HIGHLIGHT
    ),
)
```

*Three components. Target, Effect, NewTargetedSpell wires them together.*

---

## SLIDE 11 — Lightning Bolt: Test

```go
g := gametest.NewTestGame(t)
g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")

g.CastSpell(1, core.PrecombatMain,
    gametest.PlayerA, "Lightning Bolt", "PlayerB")

g.StopAt(1, core.BeginCombat)
g.Execute()
g.AssertLife(gametest.PlayerB, 17)
```

---

## SLIDE 12 — Effects: Damage & Removal

**Damage:**
- `DealDamage(amount)` — damage to target
- `DealDamageToAllCreatures(amount, filter)` — mass creature damage
- `DealDamageToPlayers(amount, selector)` — damage to selected players
- `PreventAllCombatDamage()` — Fog
- `PreventDamageToTarget(amount)` — prevention shield

**Removal:**
- `DestroyTarget()` / `DestroyTargetPermanent()` / `DestroyTargetLand()` / `DestroyTargetArtifact()`
- `DestroyAllCreatures()` / `DestroyAllLands()` / `DestroyAllEnchantments()`
- `DestroyAllMatching(filter, text)` / `DestroyAllMatchingNoRegen(f, text)`
- `ExileTarget()` / `SacrificeSource()`

---

## SLIDE 13 — Effects: Life, Cards, Graveyard

**Life:** `GainLife`, `GainLifeTarget`, `LoseLife`, `SacrificeCreatureOrDamage`

**Card Manipulation:** `DrawCards`, `DrawCardsActivePlayer`, `DiscardCards`, `DiscardRandom`, `DiscardHandAndDraw`, `SearchLibraryToHand`, `SearchLibraryToTop`, `ShuffleLibrary`

**Graveyard:** `ReturnFromGraveyardToBattlefield`, `ReturnFromGraveyardToHandTarget`, `ReturnSourceToHand`

---

## SLIDE 14 — Effects: Combat, Permanents, Misc

**Combat:** `BoostUntilEndOfTurn`, `BoostMatchingUntilEndOfTurn`, `BoostAllMatchingUntilEndOfTurn`, `DoubleTargetPower`, `GrantKeywordUntilEndOfTurn`, `MakeUnblockableUntilEndOfTurn`, `SetPTUntilEndOfTurn`, `SetPowerUntilEndOfTurn`, `RemoveFromCombat`

**Counters:** `AddCounters`, `RemoveCountersFromSource`

**Permanents:** `Tap`, `UntapTarget`, `UntapSource`, `TapOrUntapTarget`, `TapAllLands`, `TapAttachedCreature`

**Misc:** `AddMana`, `AddAnyMana`, `CreateToken`, `CloneTarget`, `CopySpellOnStack`, `AttachToTarget`, `ControlChangeTarget`, `ExtraTurn`, `ChangeColorEffect`, `CounterSpell`, `CounterSpellIfColor`, `CounterSpellIfXMeetsCMC`, `RegenerateSource`, `RegenerateTarget`, `ChooseColor`, `DestroyTargetAtEndOfTurn`, `ReplaceKeywordEffect`, `CompositeEffects`

*50+ pre-built effects. You almost never need a custom one.*

---

## SLIDE 15 — ValueSource

```go
Fixed(3)                        // constant
XValue()                       // reads g.CurrentX
CountBattlefield(who, filter)  // count permanents
CountZone(zone, who, filter)   // count cards in zone
```

Examples:
```go
mage.DealDamage(mage.Fixed(3))                          // Lightning Bolt
mage.DealDamage(mage.XValue())                          // Fireball
mage.DealDamage(mage.CountBattlefield(                  // Drain based on Swamps
    mage.SelectController(),
    mage.And(mage.IsLand, mage.HasSubType("Swamp"))))
```

*ValueSource is the secret to composable effects.*

---

## SLIDE 16 — Targets: Battlefield, Player, Stack, Zone (split)

**Battlefield:**
```
TargetCreature(filters...)
TargetPermanent(filters...)
TargetControlledCreature()
TargetControlledPermanent()
TargetLand()
TargetArtifact()
TargetArtifactOrEnchantment()
```

**Player / Stack / Zone:**
```
TargetPlayer()
TargetOpponent()
TargetAnyTarget()                    // creature or player
TargetSpellOnStack(filters...)
TargetCreatureInYourGraveyard()
TargetCardInYourGraveyard(filters...)
TargetCreatureInHand()
```

*Auto-checks hexproof, shroud, protection via CanBeTargetedBy.*

---

## SLIDE 17 — Filters (split)

**Left: PermanentFilter**

Pre-built values: `AnyPermanent`, `IsCreature`, `IsArtifact`, `IsEnchantment`, `IsLand`, `IsTapped`, `IsUntapped`, `IsAttacking`, `IsBlocking`

Constructors: `HasSubType`, `HasColorFilter`, `HasKeywordFilter`, `NotHasKeywordFilter`, `HasPowerLTE`, `HasPowerGTE`, `ControlledBy`, `NotControlledBy`, `Named`, `NotID`, `IsID`, `IsBandedWith`, `HasExpansion`

**Right: Combinators & CardFilter**

`And(filters...)`, `Or(filters...)`, `Not(filter)`

```go
// "nonblack creature"
mage.TargetCreature(mage.Not(mage.HasColorFilter(core.Black)))
// "target Djinn or Efreet"
mage.TargetCreature(mage.Or(mage.HasSubType("Djinn"), mage.HasSubType("Efreet")))
```

CardFilter: `IsCreatureCard`, `IsArtifactCard`, `NewCardFilter("label", func)`

---

## SLIDE 18 — PermanentSelector & PlayerSelector (split)

**PermanentSelector:** `SelectTarget` (targets[0]) / `SelectSource` (source permanent)

```go
mage.AddCounters(core.P1P1, mage.Fixed(1), mage.SelectSource)  // Sengir: self
mage.AddCounters(core.P1P1, mage.Fixed(1), mage.SelectTarget)  // spell: target
```

**PlayerSelector:** `SelectController()`, `SelectActivePlayer()`, `SelectEachPlayer()`, `SelectEachOpponent()`, `SelectAttachedController()`, `SelectEventController()`

```go
mage.DealDamageToPlayers(mage.Fixed(1), mage.SelectController())          // Juzam
mage.DealDamageToPlayers(mage.Fixed(2), mage.SelectAttachedController())  // Psychic Venom
```

---

## Part 3: Abilities

## SLIDE 19 — Section: Abilities

*Activated, triggered, static, and protection.*

---

## SLIDE 20 — Activated Abilities

[DIAGRAM: Cost → Target? → Effect]

```go
// Prodigal Sorcerer: {T}: Deal 1 damage to any target
mage.WithActivatedAbility(
    mage.DealDamage(mage.Fixed(1)),           // effect
    mage.Tap(),                               // cost (taps source)
    mage.WithTarget(mage.TargetAnyTarget()),  // target
)
```

---

## SLIDE 21 — Costs Catalog (split)

**Mana:** `ManaCostOf("{1}{R}")`, `GenericCost(n)`, `XManaCost()`

**Tap:** `Tap()` (also usable as effect)

**Sacrifice:** `SacrificeSourceCost()`, `SacrificeCreatureCost()`, `SacrificeArtifactCost()`

**Life:** `LifePayCost(amount)`

**Resources:** `RemoveCountersCost(ct, n)`, `DiscardCost(n)`, `DiscardRandomCost(n)`, `ExileFromGraveyardCost(n)`, `ExileSourceCost()`, `ReturnToHandCost(filter)`, `TapCreatureCost()`

Card-level: `mage.WithAdditionalCost(mage.SacrificeCreatureCost())`

---

## SLIDE 22 — AbilityOption Modifiers

```
WithCost(cost)                       // additional cost
WithTarget(target)                   // targeting
WithEffect(effect)                   // additional effect
WithUpkeepOnly()                     // only during upkeep
WithOncePerTurn()                    // once per turn
WithAnyPlayerMay()                   // any player can activate
WithControlledSinceTurnStart()       // no haste
```

```go
// Northern Paladin: {W}{W}, {T}: Destroy target black permanent
mage.WithActivatedAbility(
    mage.DestroyTargetPermanent(),
    mage.ManaCostOf("{W}{W}"),
    mage.WithCost(mage.Tap()),
    mage.WithTarget(mage.TargetPermanent(mage.HasColorFilter(core.Black))),
)
```

---

## SLIDE 23 — Triggered: Sengir Vampire (split)

*[Same as before — oracle + code on left, test on right]*

---

## SLIDE 24 — All 20 Trigger Constructors

```
EntersBattlefieldTrigger          AttacksTrigger
BlocksTrigger                     DiesCreatureTrigger
AnyCreatureDiesTrigger            CreatureDealtDamageBySourceDiesTrigger
DealsDamageToOpponentTrigger      WhenDamageDealtToThisTrigger
BeginningOfUpkeepTrigger          BeginningOfEachUpkeepTrigger
BeginningOfEachDrawStepTrigger    BeginningOfAttachedControllerUpkeepTrigger
WheneverSpellCastTrigger          WheneverEnchantmentCastTrigger
WhenAttachedBecomesTappedTrigger  WheneverLandEntersBattlefieldTrigger
WhenOpponentPermanentBecomesTappedTrigger
PutIntoGraveyardFromBattlefieldTrigger
SacrificeAtUpkeepUnlessPay
```

`false` = mandatory · `true` = "you may"

---

## SLIDE 25 — Custom Triggers

```go
// "Whenever an opponent's Swamp becomes tapped"
mage.NewTriggered(core.EvtTapped, false, effect).
  SetCondition(func(evt *core.GameEvent,
      g *mage.Game,
      sourceID, controllerID uuid.UUID) bool {
    perm := g.FindPermanent(evt.SourceID)
    return perm != nil &&
      perm.Controller != controllerID &&
      perm.HasSubType("Swamp")
  })
```

---

## SLIDE 26 — Protection Abilities

```
ProtectionFromColor(core.Black)
ProtectionFromColors(core.Black, core.Red)
ProtectionFromCardType(core.TypeArtifact)
ProtectionFromSubType("Goblin")
ProtectionFromAll()
```

```go
// Black Knight
mage.NewCreature("Black Knight", "{B}{B}", 2, 1,
    mage.WithKeyword(core.FirstStrike),
    mage.WithAbility(mage.ProtectionFromColor(core.White)),
)
```

---

## SLIDE 27 — Special Ability Types

**ETB & Lifecycle:** `ETBWithTargets`, `ETBEffect`, `EntersWithXCounters`, `CopyCreatureOnETB`, `SacrificeUnlessLand`

**Mana:** `NewManaAbility`, `ManaBonusAbility`, `NewManaBonusAbility`, `NewAttachedManaBonusAbility`, `NewEquipAbility`

---

## Part 4: Continuous Effects

## SLIDE 28 — Section: Continuous Effects & The Layer System

---

## SLIDE 29 — Layer Table

| # | Layer   | What            | Example            |
|---|---------|-----------------|--------------------|
| 1 | Copy    | Copy effects    | Clone              |
| 2 | Control | Controller      | Control Magic      |
| 4 | Type    | Type changes    | Animate            |
| 5 | Color   | Color changes   | Lace effects       |
| 6 | Ability | Add/remove      | Keyword grants     |
| 7 | P/T     | Power/toughness | Giant Growth, lords|

*Effects don't go on the stack. Re-applied from scratch every cycle.*

---

## SLIDE 30 — Three Primitives

**FuncContinuousEffect** — general purpose (layer, duration, func, conditions)
**AttachedEffect** — auras & equipment (layer, func receiving src+tgt)
**TargetEffect** — temporary by ID (layer, duration, targetID, func)

*Every continuous effect wraps one of these three.*

---

## SLIDE 31 — Durations, ActiveCondition, SourceCondition

**Durations:** `WhileOnBattlefield`, `EndOfTurn`, `EndOfCombat`, `Indefinite`

**ActiveCondition:** `SourceAttached`, `SourceUntapped`, `SourceTapped`, `WithSourceCondition(cond)`

**SourceCondition:** `WhileSourceAttacking`, `WhileSourceUntapped`, `WhileControlling(filter)`

---

## SLIDE 32 — Goblin King (split)

*[Same as before — oracle + WithStaticAbility on left, test including removal on right]*

---

## SLIDE 33 — Pre-Built: Attached Effects

```
BoostAttached(p, t, AttachType)
GrantAbilityToAttached(keyword, AttachType)
GrantProtectionToAttached(color, AttachType)
RemoveKeywordFromAttached(keyword, AttachType)
ChangeAttachedSubTypes(newSubTypes)
GrantActivatedAbilityToAttached(effect, cost, at)
PreventAttachedFromUntapping(AttachType)
PreventAttachedFromAttacking(AttachType)
ControlChangeContinuous()
BoostAttachedByCount(filter, powerFn, toughFn)
```

Shortcut: `mage.NewBoostAura("Holy Strength", "{W}", 1, 2)`

---

## SLIDE 34 — Pre-Built: Target & Global Effects (split)

**Target Effects** (by ID, temporary):
```
TemporaryBoost          TemporaryKeyword
KeywordReplacement      SetBasePT
SetBasePower            ColorOverride
TemporaryAnimate        TemporaryAnimateUntilEndOfCombat
PreventBlockingUntilEndOfCombat
```

**Global Effects** (while source exists):
```
BoostAllCreatures / IncludingSelf / Controlled
BoostSelf                    PTEqualsCount / Controlled
GrantKeywordToAll            GrantActivatedAbilityToAll
PreventUntapForMatching      PreventAllUntaps
AnimateLands                 LimitLandUntaps
AllowUnlimitedLandPlays      IncreaseSpellCostForColor
ReduceSpellCostForColor      ChangeSubTypesForAll
ManaConversion               BodyguardContinuous
PersonalIncarnationRedirect
PreventFromAttackingIfDefendingPlayerControls
PreventDamageFromTo
```

---

## SLIDE 35 — The Attr System

**Capability attrs:** `AttrCanAttack`, `AttrCanBlock`, `AttrHasPowerToughness`, `AttrSummonSick`, `AttrDoesNotUntap`, `AttrEntersTapped`, `AttrMustAttack`, `AttrMustBeBlocked`, `AttrMayNotUntap`

**Type-identity attrs:** `AttrIsCreature`, `AttrIsLand`, `AttrIsArtifact`, `AttrIsEnchantment`

**Keyword attrs:** `Flying`, `Reach`, `FirstStrike`, `DoubleStrike`, `Trample`, `Vigilance`, `Haste`, `Menace`, `Fear`, `Deathtouch`, `Lifelink`, `Defender`, `Banding`, `Indestructible`, `Hexproof`, `Shroud`, `Forestwalk`, `Islandwalk`, `Swampwalk`, `Mountainwalk`, `Plainswalk`, `Desertwalk`, `UnblockableKW`, `CantBeBlockedByWalls`, `CantBeBlockedExceptByWalls`, `CanBlockAny`, `CanBlockAdditional`, `BasiliskTouch`, `CantRegenerate`

```go
g.Effects.GrantAttr(perm.ID(), core.Flying)         // give flying
g.Effects.RevokeAttr(perm.ID(), core.AttrCanAttack)  // prevent attacking
```

`baseAttrs` (intrinsic) + `grantedAttrs` (effect-cycle deltas) → `HasAttr(a)` = sum > 0

---

## Part 5: FuncEffect & GameMutator

## SLIDE 36 — Section: FuncEffect

---

## SLIDE 37 — FuncEffect Anatomy

```go
mage.FuncEffect(
    "description",
    mage.EffectProperties{Outcome: mage.OutcomeDetriment},
    func(g mage.GameMutator, sourceID, controller uuid.UUID,
         targets []uuid.UUID) error {
        // full GameMutator access
        return nil
    },
)
```

*Always set Outcome. The AI uses it for targeting.*

---

## SLIDE 38 — Swords to Plowshares (split)

**Left: Oracle** — Exile target creature. Controller gains life equal to its power.

**Right: FuncEffect implementation** showing `ExilePermanent` + `CurrentPower` + `PlayerGainLife`

---

## SLIDE 39 — Serendib Djinn: Three Branches

Three test cards: Non-Island (20 life), Island (17 life), No Lands (17 life + Djinn gone)

*One test per Oracle clause. Written BEFORE the implementation.*

---

## SLIDE 40 — GameReader & GameMutator (split)

**GameReader** (read-only): `GetPlayer`, `GetOpponent`, `ActivePlayerObj`, `AllPlayers`, `FindPermanent`, `FindPermanentByName`, `FindCardAnywhere`, `AnyBattlefield`, `FilterBattlefield`, `CountBattlefield`, `XValue`, `ModeValue`, `GetResolvingCard`, `FindStackObject`, `CombatGroups`

**GameMutator** (read + mutations): `PlayerGainLife`, `FireEvent`, `PutOnBattlefield`, `RemoveFromBattlefield`, `DestroyPermanent`, `ExilePermanent`, `Sacrifice`, `DealDamageToPlayer`, `DealDamageToPermanent`, `CounterSpellOnStack`, `PushStack`, `Attach`, `RegisterDelayedTrigger`, `GrantExtraTurn`, `RemoveFromCombat`, `AddContinuousEffect`, `ApplyContinuousEffects`, `TryPayCostFromLands`, `FlipCoin`

*Mutations go through the replacement pipeline. The API is safe by design.*

---

## Part 6: Deep Engine

## SLIDE 41 — Section: Deep Engine

*Replacements, EffectManager, GameRules.*

---

## SLIDE 42 — Replacement Pipeline

[DIAGRAM vertical: Deal 5 damage → DamageToPlayerAction{5} → Prevention Shield absorb 3 → {2} → Ali from Cairo cap → {1} → executeAction: 1 damage]

---

## SLIDE 43 — Action Types

Five concrete types: `*DamageToPlayerAction`, `*DamageToCreatureAction`, `*DestroyPermanentAction`, `*LifeGainAction`, `*DrawCardAction`

All implement `Action` interface. Damage actions expose `Amount()`, `IsCombatDamage()`, `WithAmount(n)`.

---

## SLIDE 44 — ReplacementEffect Interface

```go
type ReplacementEffect interface {
    Matches(Action, GameReader) bool
    Replace(Action, GameMutator) Action
    SourceID() uuid.UUID
    IsActive(GameReader) bool
}
```

Replace returns: modified action, different type (redirect), or `nil` (prevent).

---

## SLIDE 45 — Two Replacement Lists

**em.replacements** — persistent (one-shot shields, turn-scoped). `AddReplacement`, `PrependReplacement`.

**em.cycleReplacements** — cleared each Apply() cycle, re-registered by continuous effects. `AddCycleReplacement`.

---

## SLIDE 46 — 17 Built-In Replacements

**Damage Prevention:** `preventionShield`, `fog`, `forcefield`, `colorPrevention`, `typePrevention`, `reverseDamage`, `damagePreventionRule`

**Damage Redirection:** `bodyguard` (cycle), `playerDamageRedirect` (cycle), `artifactDamageRedirect` (cycle), `creatureDamageRedirect`, `attackerDamageRedirect`

**Destruction:** `regeneration`

**Life Gain:** `lichLifeGain`, `minimumLife` (cycle)

**Draw:** `skipDraw`, `drawReplacement`

---

## SLIDE 47 — GameMutator → Replacement Registration

```
g.AddPreventionShield(playerID, amount)     → preventionShieldReplacement
g.AddRegenerationShield(permID)             → regenerationReplacement
g.SetPreventCombatDamage()                  → fogReplacement
g.AddForcefieldShield(playerID)             → forcefieldReplacement
g.AddColorPrevention(playerID, color)       → colorPreventionReplacement
g.AddTypePrevention(playerID, cardType)     → typePreventionReplacement
g.AddReverseDamageShield(playerID)          → reverseDamageReplacement
g.SetCreatureDamageRedirect(cID, pID)       → creatureDamageRedirectReplacement
g.SetAttackerDamageRedirect(aID, absID)     → attackerDamageRedirectReplacement
g.SetSkipNextDraw(playerID)                 → skipDrawReplacement
g.SetDrawReplacement(playerID, count)       → drawReplacementEffect
g.SetLichActive(playerID, sourceID)         → lichLifeGainReplacement
g.PreventAllDamageFrom(sourceID)            → damagePreventionRuleReplacement
g.SetMinimumLife(playerID)                  → minimumLifeReplacement (cycle)
g.SetArtifactDamageRedirect(ctrlID, pID)    → artifactDamageRedirectReplacement (cycle)
g.AddReplacementEffect(r)                   → custom
```

---

## SLIDE 48 — Custom Replacement Example

```go
type halveDamageReplacement struct {
    replacementBase
    playerID uuid.UUID
}

func (r *halveDamageReplacement) Matches(a Action, _ GameReader) bool {
    act, ok := a.(*DamageToPlayerAction)
    return ok && act.PlayerID() == r.playerID
}

func (r *halveDamageReplacement) Replace(a Action, _ GameMutator) Action {
    act := a.(*DamageToPlayerAction)
    return act.WithAmount(act.Amount() / 2)
}

func (r *halveDamageReplacement) IsActive(_ GameReader) bool { return true }
```

*Register via FuncContinuousEffect + AddCycleReplacement.*

---

## SLIDE 49 — EffectManager.Apply(g)

[DIAGRAM vertical:
1. Reset grantedAttrs, powerBonus, toughBonus, overrides, controller
2. Strip effect-granted runtime abilities
3. Remove effects whose source is gone
4. Apply effects in layer order: 1→2→4→5→6→7
5. Rules.ResetPerCycle() + Damage.ResetPerCycle()
6. Sync mana conversions & write attr deltas
]

Also: `GrantAttr`/`RevokeAttr`, `PreventBlockPair`/`IsBlockPrevented`

---

## SLIDE 50 — DamageSystem & GameRules (split)

**DamageSystem** (`g.Effects.Damage`):
- Reflection: `SetDamageReflection`, `GetDamageReflection` (Eye for an Eye)
- Legacy API (delegates): `AddDamagePreventionRule` (with `WithFrom`/`WithTo`/`WithOneShot`), `AddRegenerationShield`, `SetArtifactDamageRedirect`

**GameRules** (`g.Effects.Rules`):
- Fields: `LandUntapMax`, `ArtifactUntapMax`, `UnlimitedLandPlays`, `SpellCostIncreases`, `SpellCostReductions`, `ManaConversion`
- Lich: `SetLichActive`/`IsLichActive`/`GetLichPermanent`
- Channel: `SetChannelActive`/`IsChannelActive`
- Sanctuary: `SetSanctuaryActive`/`IsSanctuaryActive`
- Draw: `SetSkipNextDraw`/`ShouldSkipDraw`
- Life: `SetMinimumLife`/`HasMinimumLife`
- Hand: `SetMaxHandSize`/`GetMaxHandSize`
- Cast block: `AddExpansionCastBlock`/`IsExpansionBlocked`

---

## SLIDE 51 — Adding New Events

Three steps: 1. Define const in core/events.go. 2. FireEvent in engine. 3. Cards listen with NewTriggered.

---

## Part 7: Testing

## SLIDE 52 — Section: TDD Best Practices

---

## SLIDE 53 — The TDD Cycle

[DIAGRAM: RED (write test) → GREEN (implement) → REFACTOR (pre-built effects? edge cases?)]

*If you can't write a test for it, you don't understand the card.*

---

## SLIDE 54 — Five Testing Patterns

1. **Positive AND negative** — triggers on own, NOT on opponent
2. **Source removal** — removing king removes boost
3. **Stack interaction** — CastInResponseTo
4. **Across turns** — 3 upkeeps of Juzam
5. **Combat math** — first strike kills before normal damage

---

## SLIDE 55 — The Complexity Ladder

| Level | Description | API |
|-------|-------------|-----|
| 1 | Vanilla creature | NewCreature + WithSubTypes |
| 2 | Keywords | WithKeyword |
| 3 | Pre-built effects | DealDamage, DestroyTarget (50+) |
| 4 | Targets + filters | TargetCreature(Not(HasColor(...))) |
| 5 | Costs & selectors | 15 costs, PermanentSelector, PlayerSelector |
| 6 | Activated abilities | WithActivatedAbility + AbilityOptions |
| 7 | Triggered abilities | 20 convenience constructors |
| 8 | Custom triggers | NewTriggered + SetCondition |
| 9 | Continuous effects | 30+ pre-built (attached/target/global) |
| 10 | FuncContinuousEffect | Custom layer logic + conditions |
| 11 | FuncEffect | Custom one-off + GameMutator API |
| 12 | Replacement effects | 17 built-in + ReplacementEffect interface |
| 13 | Attr system | GrantAttr, RevokeAttr, base/granted |
| 14 | New events | FireEvent, listen |
| 15 | Engine rules | GameRules + DamageSystem |

*Most cards live in levels 1–7.*

---

## SLIDE 56 — Closing

[DIAGRAM: Oracle Text → Test → Implement → Verify]

1. Follow Oracle text exactly
2. Never simplify
3. Write the test first
4. Consult XMage

The reference: `pkg/mage/doc.go`

*Happy brewing.*
