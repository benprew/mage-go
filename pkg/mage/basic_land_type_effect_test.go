package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func basicLandTypeTestGame(t *testing.T) (*Game, Player, *Permanent, *Permanent, Ability) {
	t.Helper()
	player := NewBasePlayer("Alice")
	g := NewGame(player, NewBasePlayer("Bob"))

	sourceCard := NewCreature("Type Changer", "{G}", 1, 1)
	sourceCard.SetOwner(player.PlayerID())
	source := g.PutOnBattlefield(sourceCard, player.PlayerID())

	printedAbility := NewActivatedAbility(GainLife(1), ManaCostOf("{1}"))
	landCard := NewLand("Utility Mountain",
		WithSubTypes("Mountain"),
		WithManaAbility(Red),
		WithAbility(printedAbility),
	)
	landCard.SetOwner(player.PlayerID())
	land := g.PutOnBattlefield(landCard, player.PlayerID())

	return g, player, source, land, printedAbility
}

func countManaAbilityColor(perm *Permanent, color Color) int {
	total := 0
	for _, ability := range perm.RuntimeAbilities {
		for _, production := range ManaProductionsForAbility(ability) {
			if production.Color == color {
				total += production.Amount
			}
		}
	}
	return total
}

func hasRuntimeAbility(perm *Permanent, abilityID Ability) bool {
	for _, ability := range perm.RuntimeAbilities {
		if ability.AbilityID() == abilityID.AbilityID() {
			return true
		}
	}
	return false
}

func TestBecomesBasicLandTargetEffectReplacesPrintedAbilitiesAndGrantsMana(t *testing.T) {
	g, _, source, land, printedAbility := basicLandTypeTestGame(t)
	effect := BecomesBasicLandTargetEffect(land.ID(), WhileOnBattlefield, "Forest")
	effect.SetSourceID(source.ID())
	g.AddContinuousEffect(effect)

	if !land.HasSubType("Forest") || land.HasSubType("Mountain") {
		t.Fatalf("land subtypes after override: Forest=%v Mountain=%v", land.HasSubType("Forest"), land.HasSubType("Mountain"))
	}
	if got := countManaAbilityColor(land, Green); got != 1 {
		t.Fatalf("green mana produced = %d, want 1", got)
	}
	if got := countManaAbilityColor(land, Red); got != 0 {
		t.Fatalf("red mana produced = %d, want 0", got)
	}
	if hasRuntimeAbility(land, printedAbility) {
		t.Fatal("printed rules-text ability was not removed")
	}
}

func TestBecomesBasicLandTargetEffectAddsIntrinsicManaDuringLayerFourApply(t *testing.T) {
	g, _, _, land, _ := basicLandTypeTestGame(t)
	effect := BecomesBasicLandTargetEffect(land.ID(), Indefinite, "Forest")
	if effect.GetLayer() != LayerType {
		t.Fatalf("effect layer = %v, want layer 4 type-changing effects", effect.GetLayer())
	}
	if err := effect.Apply(&EffectContext{Game: g}); err != nil {
		t.Fatal(err)
	}

	if got := countManaAbilityColor(land, Green); got != 1 {
		t.Fatalf("green mana produced immediately by layer-four effect = %d, want 1", got)
	}
}

func TestAutoTapForCostUsesManaFromChangedBasicLandSubtype(t *testing.T) {
	player := NewBasePlayer("Alice")
	g := NewGame(player, NewBasePlayer("Bob"))
	card := NewLand("Mishra's Factory", WithManaAbility(Colorless))
	card.SetOwner(player.PlayerID())
	land := g.PutOnBattlefield(card, player.PlayerID())
	g.AddContinuousEffect(BecomesBasicLandTargetEffect(land.ID(), Indefinite, "Island"))

	if err := g.AutoTapForCost(player.PlayerID(), ManaCost{Blue: 1}); err != nil {
		t.Fatalf("AutoTapForCost({U}) failed after land became an Island: %v", err)
	}
	if !land.Tapped {
		t.Fatal("changed land was not tapped to pay {U}")
	}
}

func TestAutoTapForCostUsesManaFlareBonusFromChangedBasicLandSubtype(t *testing.T) {
	player := NewBasePlayer("Alice")
	g := NewGame(player, NewBasePlayer("Bob"))
	card := NewLand("Mishra's Factory", WithManaAbility(Colorless))
	card.SetOwner(player.PlayerID())
	land := g.PutOnBattlefield(card, player.PlayerID())
	flare := NewEnchantment("Mana Flare", "{2}{R}", WithAbility(NewManaFlareAbility(IsLand)))
	flare.SetOwner(player.PlayerID())
	g.PutOnBattlefield(flare, player.PlayerID())
	g.AddContinuousEffect(BecomesBasicLandTargetEffect(land.ID(), Indefinite, "Island"))

	if !g.CanAfford(player.PlayerID(), ManaCost{Blue: 2}, nil) {
		t.Fatal("mana solver did not consider {U}{U} affordable")
	}
	if err := g.AutoTapForCost(player.PlayerID(), ManaCost{Blue: 2}); err != nil {
		t.Fatalf("AutoTapForCost({U}{U}) failed with Mana Flare after land became an Island: %v", err)
	}
	if !land.Tapped {
		t.Fatal("changed land was not tapped to pay {U}{U}")
	}
}

func TestBecomesBasicLandTargetEffectClearsOnlyLandSubtypes(t *testing.T) {
	player := NewBasePlayer("Alice")
	g := NewGame(player, NewBasePlayer("Bob"))
	landCard := NewLand("Animated Gate",
		WithCardType(TypeCreature),
		WithSubTypes("Mountain", "Gate", "Elemental"),
		WithManaAbility(Red),
	)
	landCard.SetOwner(player.PlayerID())
	land := g.PutOnBattlefield(landCard, player.PlayerID())

	effect := BecomesBasicLandTargetEffect(land.ID(), Indefinite, "Forest")
	if err := effect.Apply(&EffectContext{Game: g}); err != nil {
		t.Fatal(err)
	}

	if land.HasSubType("Mountain") || land.HasSubType("Gate") {
		t.Fatalf("old land subtypes remain: Mountain=%v Gate=%v", land.HasSubType("Mountain"), land.HasSubType("Gate"))
	}
	if !land.HasSubType("Forest") || !land.HasSubType("Elemental") {
		t.Fatalf("new and nonland subtypes: Forest=%v Elemental=%v", land.HasSubType("Forest"), land.HasSubType("Elemental"))
	}
}

func TestBecomesBasicLandTargetEffectRestoresPrintedAbilitiesWhenEffectEnds(t *testing.T) {
	g, _, source, land, printedAbility := basicLandTypeTestGame(t)
	effect := BecomesBasicLandTargetEffect(land.ID(), WhileOnBattlefield, "Forest")
	effect.SetSourceID(source.ID())
	g.AddContinuousEffect(effect)

	g.RemoveFromBattlefield(source)

	if land.HasSubType("Forest") || !land.HasSubType("Mountain") {
		t.Fatalf("land subtypes after effect ended: Forest=%v Mountain=%v", land.HasSubType("Forest"), land.HasSubType("Mountain"))
	}
	if got := countManaAbilityColor(land, Green); got != 0 {
		t.Fatalf("green mana produced after effect ended = %d, want 0", got)
	}
	if got := countManaAbilityColor(land, Red); got != 1 {
		t.Fatalf("red mana produced after effect ended = %d, want 1", got)
	}
	if !hasRuntimeAbility(land, printedAbility) {
		t.Fatal("printed rules-text ability was not restored")
	}
}

func TestBecomesBasicLandTargetEffectSuppressesAndRestoresPrintedKeywords(t *testing.T) {
	player := NewBasePlayer("Alice")
	g := NewGame(player, NewBasePlayer("Bob"))
	sourceCard := NewCreature("Type Changer", "{G}", 1, 1)
	sourceCard.SetOwner(player.PlayerID())
	source := g.PutOnBattlefield(sourceCard, player.PlayerID())
	landCard := NewLand("Indestructible Mountain",
		WithSubTypes("Mountain"),
		WithManaAbility(Red),
		WithKeyword(Indestructible),
	)
	landCard.SetOwner(player.PlayerID())
	land := g.PutOnBattlefield(landCard, player.PlayerID())
	if !land.HasKeyword(Indestructible) {
		t.Fatal("precondition: land should have printed indestructible")
	}

	effect := BecomesBasicLandTargetEffect(land.ID(), WhileOnBattlefield, "Forest")
	effect.SetSourceID(source.ID())
	g.AddContinuousEffect(effect)

	if land.HasKeyword(Indestructible) {
		t.Fatal("printed indestructible was not removed")
	}

	g.RemoveFromBattlefield(source)
	if !land.HasKeyword(Indestructible) {
		t.Fatal("printed indestructible was not restored")
	}
}

func TestBecomesBasicLandTargetEffectPreservesGrantedKeywords(t *testing.T) {
	g, player, source, land, _ := basicLandTypeTestGame(t)
	grantSourceCard := NewEnchantment("Indestructibility Grant", "{W}")
	grantSourceCard.SetOwner(player.PlayerID())
	grantSource := g.PutOnBattlefield(grantSourceCard, player.PlayerID())

	typeEffect := BecomesBasicLandTargetEffect(land.ID(), WhileOnBattlefield, "Forest")
	typeEffect.SetSourceID(source.ID())
	g.AddContinuousEffect(typeEffect)
	grantEffect := FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		g.effects.GrantAttr(land.ID(), Indestructible)
		return nil
	})
	grantEffect.SetSourceID(grantSource.ID())
	g.AddContinuousEffect(grantEffect)

	if !land.HasKeyword(Indestructible) {
		t.Fatal("indestructible granted by another effect was removed")
	}
}

func TestBecomesBasicLandTargetEffectSuppressesCopiedKeywords(t *testing.T) {
	g, player, source, land, _ := basicLandTypeTestGame(t)
	copyCard := NewCreature("Flying Copy Source", "{1}{U}", 1, 1, WithKeyword(Flying))
	copyCard.SetOwner(player.PlayerID())
	copySource := g.PutOnBattlefield(copyCard, player.PlayerID())
	g.effects.AddCopyEffect(land.ID(), copySource)
	g.ApplyContinuousEffects()
	if !land.HasKeyword(Flying) {
		t.Fatal("precondition: copy effect should grant flying")
	}

	typeEffect := BecomesBasicLandTargetEffect(land.ID(), WhileOnBattlefield, "Forest")
	typeEffect.SetSourceID(source.ID())
	g.AddContinuousEffect(typeEffect)

	if land.HasKeyword(Flying) {
		t.Fatal("keyword generated by a copy effect was not removed")
	}
}

func TestBecomesBasicLandTargetEffectPreservesAbilitiesGrantedByOtherEffects(t *testing.T) {
	g, player, source, land, _ := basicLandTypeTestGame(t)
	grantSourceCard := NewEnchantment("Mana Grant", "{U}")
	grantSourceCard.SetOwner(player.PlayerID())
	grantSource := g.PutOnBattlefield(grantSourceCard, player.PlayerID())

	typeEffect := BecomesBasicLandTargetEffect(land.ID(), WhileOnBattlefield, "Forest")
	typeEffect.SetSourceID(source.ID())
	g.AddContinuousEffect(typeEffect)
	grantEffect := FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		ability := NewManaAbility(Blue)
		ability.SetSource(land.ID())
		ability.SetController(land.ControllerID())
		land.RuntimeAbilities = append(land.RuntimeAbilities, WrapGrantedAbility(ability))
		return nil
	})
	grantEffect.SetSourceID(grantSource.ID())
	g.AddContinuousEffect(grantEffect)

	if got := countManaAbilityColor(land, Green); got != 1 {
		t.Fatalf("green mana produced = %d, want 1", got)
	}
	if got := countManaAbilityColor(land, Blue); got != 1 {
		t.Fatalf("mana from granted ability = %d, want 1 blue", got)
	}
}

func TestBecomesBasicLandTargetEffectUsesFinalSubtypeForIntrinsicMana(t *testing.T) {
	g, player, firstSource, land, _ := basicLandTypeTestGame(t)
	secondSourceCard := NewCreature("Later Type Changer", "{U}", 1, 1)
	secondSourceCard.SetOwner(player.PlayerID())
	secondSource := g.PutOnBattlefield(secondSourceCard, player.PlayerID())

	forestEffect := BecomesBasicLandTargetEffect(land.ID(), WhileOnBattlefield, "Forest")
	forestEffect.SetSourceID(firstSource.ID())
	g.AddContinuousEffect(forestEffect)
	islandEffect := BecomesBasicLandTargetEffect(land.ID(), WhileOnBattlefield, "Island")
	islandEffect.SetSourceID(secondSource.ID())
	g.AddContinuousEffect(islandEffect)

	if land.HasSubType("Forest") || !land.HasSubType("Island") {
		t.Fatalf("final land subtypes: Forest=%v Island=%v", land.HasSubType("Forest"), land.HasSubType("Island"))
	}
	if got := countManaAbilityColor(land, Green); got != 0 {
		t.Fatalf("green mana produced = %d, want 0", got)
	}
	if got := countManaAbilityColor(land, Blue); got != 1 {
		t.Fatalf("blue mana produced = %d, want 1", got)
	}
}

func TestBecomesBasicLandTargetEffectSuppressesAndRestoresPrintedStaticAbilities(t *testing.T) {
	player := NewBasePlayer("Alice")
	g := NewGame(player, NewBasePlayer("Bob"))

	typeChangerCard := NewCreature("Type Changer", "{G}", 1, 1)
	typeChangerCard.SetOwner(player.PlayerID())
	typeChanger := g.PutOnBattlefield(typeChangerCard, player.PlayerID())
	targetCard := NewCreature("Boosted Creature", "{1}{G}", 1, 1)
	targetCard.SetOwner(player.PlayerID())
	target := g.PutOnBattlefield(targetCard, player.PlayerID())
	landCard := NewLand("Static Mountain",
		WithSubTypes("Mountain"),
		WithManaAbility(Red),
		WithStaticAbility(FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
			g.MutablePermanent(target.ID()).BoostPT(1, 1)
			return nil
		})),
	)
	landCard.SetOwner(player.PlayerID())
	land := g.PutOnBattlefield(landCard, player.PlayerID())
	if power, toughness := target.CurrentPower(g), target.CurrentToughness(g); power != 2 || toughness != 2 {
		t.Fatalf("P/T with printed static ability = %d/%d, want 2/2", power, toughness)
	}

	effect := BecomesBasicLandTargetEffect(land.ID(), WhileOnBattlefield, "Forest")
	effect.SetSourceID(typeChanger.ID())
	g.AddContinuousEffect(effect)
	if power, toughness := target.CurrentPower(g), target.CurrentToughness(g); power != 1 || toughness != 1 {
		t.Fatalf("P/T while printed static ability suppressed = %d/%d, want 1/1", power, toughness)
	}

	g.RemoveFromBattlefield(typeChanger)
	if power, toughness := target.CurrentPower(g), target.CurrentToughness(g); power != 2 || toughness != 2 {
		t.Fatalf("P/T after printed static ability restored = %d/%d, want 2/2", power, toughness)
	}
}

func TestBecomesBasicLandTargetUntilSourceLeavesSurvivesPhasing(t *testing.T) {
	g, _, source, land, _ := basicLandTypeTestGame(t)
	effect := BecomesBasicLandTargetUntilSourceLeaves(land.ID(), "Forest")
	effect.SetSourceID(source.ID())
	g.AddContinuousEffect(effect)

	phased := g.PhaseOut(source.ID())
	g.ApplyContinuousEffects()
	if !land.HasSubType("Forest") || countManaAbilityColor(land, Green) != 1 {
		t.Fatal("basic land type effect ended when its source phased out")
	}

	g.PhaseIn(phased)
	g.RemoveFromBattlefield(source)
	if land.HasSubType("Forest") || !land.HasSubType("Mountain") {
		t.Fatal("basic land type effect did not end when its source left the battlefield")
	}
}

func TestBecomesBasicLandTargetEffectSurvivesTargetPhasing(t *testing.T) {
	g, _, source, land, _ := basicLandTypeTestGame(t)
	effect := BecomesBasicLandTargetUntilSourceLeaves(land.ID(), "Forest")
	effect.SetSourceID(source.ID())
	g.AddContinuousEffect(effect)

	phased := g.PhaseOut(land.ID())
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()

	if !land.HasSubType("Forest") || land.HasSubType("Mountain") {
		t.Fatal("basic land type effect ended when its target phased out")
	}
	if got := countManaAbilityColor(land, Green); got != 1 {
		t.Fatalf("green mana produced after target phased in = %d, want 1", got)
	}
}
