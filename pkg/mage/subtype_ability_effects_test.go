package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func subtypeTestPermanent(t *testing.T) (*Game, *Permanent) {
	t.Helper()
	player := NewBasePlayer("Alice")
	g := NewGame(player, NewBasePlayer("Bob"))
	card := NewCreature("Forest Dryad", "{G}", 1, 1,
		WithCardType(TypeLand),
		WithSubTypes("Forest", "Dryad"),
		WithManaAbility(Green),
	)
	card.SetOwner(player.PlayerID())
	return g, g.PutOnBattlefield(card, player.PlayerID())
}

func TestSetSubtypesReplacesOnlyTheSpecifiedFamily(t *testing.T) {
	g, permanent := subtypeTestPermanent(t)
	g.AddContinuousEffect(SetSubtypes(permanent.ID(), SubtypeCreature, Indefinite, "Human"))

	if !permanent.HasSubType("Human") || permanent.HasSubType("Dryad") {
		t.Fatalf("creature subtypes: Human=%v Dryad=%v", permanent.HasSubType("Human"), permanent.HasSubType("Dryad"))
	}
	if !permanent.HasSubType("Forest") {
		t.Fatal("setting creature subtypes removed the Forest land subtype")
	}
}

func TestSetLandSubtypesPreservesCreatureSubtypes(t *testing.T) {
	g, permanent := subtypeTestPermanent(t)
	g.AddContinuousEffect(SetSubtypes(permanent.ID(), SubtypeLand, Indefinite, "Island"))

	if !permanent.HasSubType("Island") || permanent.HasSubType("Forest") {
		t.Fatalf("land subtypes: Island=%v Forest=%v", permanent.HasSubType("Island"), permanent.HasSubType("Forest"))
	}
	if !permanent.HasSubType("Dryad") {
		t.Fatal("setting land subtypes removed the Dryad creature subtype")
	}
	if countManaAbilityColor(permanent, Green) != 0 || countManaAbilityColor(permanent, Blue) != 1 {
		t.Fatalf("mana after setting Island: green=%d blue=%d", countManaAbilityColor(permanent, Green), countManaAbilityColor(permanent, Blue))
	}
}

func TestSetCreatureSubtypesPreservesArtifactSubtypes(t *testing.T) {
	player := NewBasePlayer("Alice")
	g := NewGame(player, NewBasePlayer("Bob"))
	card := NewCreature("Equipped Construct", "{3}", 2, 2,
		WithCardType(TypeArtifact),
		WithSubTypes("Equipment", "Construct"),
	)
	card.SetOwner(player.PlayerID())
	permanent := g.PutOnBattlefield(card, player.PlayerID())

	g.AddContinuousEffect(SetSubtypes(permanent.ID(), SubtypeCreature, Indefinite, "Human"))

	if !permanent.HasSubType("Equipment") || !permanent.HasSubType("Human") || permanent.HasSubType("Construct") {
		t.Fatalf("subtypes: Equipment=%v Human=%v Construct=%v", permanent.HasSubType("Equipment"), permanent.HasSubType("Human"), permanent.HasSubType("Construct"))
	}
}

func TestSetSubtypesCanRemoveEverySubtypeInOneFamily(t *testing.T) {
	g, permanent := subtypeTestPermanent(t)
	g.AddContinuousEffect(SetSubtypes(permanent.ID(), SubtypeCreature, Indefinite))

	if permanent.HasSubType("Dryad") {
		t.Fatal("creature subtype was not removed")
	}
	if !permanent.HasSubType("Forest") {
		t.Fatal("removing creature subtypes removed the land subtype")
	}
}

func TestSetSubtypesRestoresTheFamilyWhenEffectEnds(t *testing.T) {
	g, permanent := subtypeTestPermanent(t)
	player := g.GetPlayer(permanent.ControllerID())
	sourceCard := NewEnchantment("Subtype Source", "{1}{U}")
	sourceCard.SetOwner(player.PlayerID())
	source := g.PutOnBattlefield(sourceCard, player.PlayerID())
	effect := SetSubtypes(permanent.ID(), SubtypeCreature, WhileOnBattlefield, "Human")
	effect.SetSourceID(source.ID())
	g.AddContinuousEffect(effect)

	g.RemoveFromBattlefield(source)
	if permanent.HasSubType("Human") || !permanent.HasSubType("Dryad") || !permanent.HasSubType("Forest") {
		t.Fatalf("restored subtypes: Human=%v Dryad=%v Forest=%v", permanent.HasSubType("Human"), permanent.HasSubType("Dryad"), permanent.HasSubType("Forest"))
	}
}

func TestAddSubtypesRetainsExistingSubtypes(t *testing.T) {
	g, permanent := subtypeTestPermanent(t)
	g.AddContinuousEffect(AddSubtypes(permanent.ID(), SubtypeCreature, Indefinite, "Druid", "Human"))

	for _, subtype := range []string{"Forest", "Dryad", "Druid", "Human"} {
		if !permanent.HasSubType(subtype) {
			t.Fatalf("missing subtype %q after additive effect", subtype)
		}
	}
}

func TestSetAndAddSubtypesUseLayerFourTimestampOrder(t *testing.T) {
	t.Run("later set replaces earlier addition", func(t *testing.T) {
		g, permanent := subtypeTestPermanent(t)
		g.AddContinuousEffect(AddSubtypes(permanent.ID(), SubtypeCreature, Indefinite, "Druid"))
		g.AddContinuousEffect(SetSubtypes(permanent.ID(), SubtypeCreature, Indefinite, "Human"))

		if permanent.HasSubType("Druid") || !permanent.HasSubType("Human") {
			t.Fatalf("subtypes after add then set: Druid=%v Human=%v", permanent.HasSubType("Druid"), permanent.HasSubType("Human"))
		}
	})

	t.Run("later addition augments earlier set", func(t *testing.T) {
		g, permanent := subtypeTestPermanent(t)
		g.AddContinuousEffect(SetSubtypes(permanent.ID(), SubtypeCreature, Indefinite, "Human"))
		g.AddContinuousEffect(AddSubtypes(permanent.ID(), SubtypeCreature, Indefinite, "Druid"))

		if !permanent.HasSubType("Druid") || !permanent.HasSubType("Human") {
			t.Fatalf("subtypes after set then add: Druid=%v Human=%v", permanent.HasSubType("Druid"), permanent.HasSubType("Human"))
		}
	})
}

func TestAddSubtypesRequiresCorrespondingCardType(t *testing.T) {
	player := NewBasePlayer("Alice")
	g := NewGame(player, NewBasePlayer("Bob"))
	card := NewArtifact("Relic", "{1}")
	card.SetOwner(player.PlayerID())
	permanent := g.PutOnBattlefield(card, player.PlayerID())

	g.AddContinuousEffect(AddSubtypes(permanent.ID(), SubtypeCreature, Indefinite, "Human"))
	if permanent.HasSubType("Human") {
		t.Fatal("noncreature permanent gained a creature subtype")
	}
}

func TestSubtypeChangeRejectsKnownSubtypeFromAnotherFamily(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("SetSubtypes accepted Forest as a creature subtype")
		}
	}()
	SetSubtypes(uuid.New(), SubtypeCreature, Indefinite, "Forest")
}

func TestAddSubtypesChecksCardTypeAtItsLayerFourTimestamp(t *testing.T) {
	newArtifact := func(t *testing.T) (*Game, *Permanent) {
		t.Helper()
		player := NewBasePlayer("Alice")
		g := NewGame(player, NewBasePlayer("Bob"))
		card := NewArtifact("Animated Relic", "{1}")
		card.SetOwner(player.PlayerID())
		return g, g.PutOnBattlefield(card, player.PlayerID())
	}
	becomeCreature := func(targetID uuid.UUID) ContinuousEffect {
		return FuncContinuousEffect(LayerType, Indefinite, func(g *Game, _ uuid.UUID) error {
			g.effects.GrantAttr(targetID, AttrIsCreature)
			return nil
		}, func(*Game, uuid.UUID) bool { return true })
	}

	t.Run("type gained before subtype", func(t *testing.T) {
		g, permanent := newArtifact(t)
		g.AddContinuousEffect(becomeCreature(permanent.ID()))
		g.AddContinuousEffect(AddSubtypes(permanent.ID(), SubtypeCreature, Indefinite, "Human"))
		if !permanent.HasSubType("Human") {
			t.Fatal("permanent did not gain subtype after gaining its corresponding type")
		}
	})

	t.Run("subtype attempted before type", func(t *testing.T) {
		g, permanent := newArtifact(t)
		g.AddContinuousEffect(AddSubtypes(permanent.ID(), SubtypeCreature, Indefinite, "Human"))
		g.AddContinuousEffect(becomeCreature(permanent.ID()))
		if permanent.HasSubType("Human") {
			t.Fatal("subtype applied before the permanent had its corresponding type")
		}
	})
}

func TestAddBasicLandSubtypeRetainsRulesTextAndAddsIntrinsicMana(t *testing.T) {
	player := NewBasePlayer("Alice")
	g := NewGame(player, NewBasePlayer("Bob"))
	printed := NewActivatedAbility(GainLife(1), ManaCostOf("{1}"))
	card := NewLand("Utility Mountain",
		WithSubTypes("Mountain"),
		WithManaAbility(Red),
		WithAbility(printed),
	)
	card.SetOwner(player.PlayerID())
	land := g.PutOnBattlefield(card, player.PlayerID())

	g.AddContinuousEffect(AddSubtypes(land.ID(), SubtypeLand, Indefinite, "Forest"))

	if !land.HasSubType("Mountain") || !land.HasSubType("Forest") {
		t.Fatalf("additive land subtypes: Mountain=%v Forest=%v", land.HasSubType("Mountain"), land.HasSubType("Forest"))
	}
	if !hasRuntimeAbility(land, printed) {
		t.Fatal("additive basic land subtype removed printed rules text")
	}
	if countManaAbilityColor(land, Red) != 1 || countManaAbilityColor(land, Green) != 1 {
		t.Fatalf("mana after additive Forest: red=%d green=%d", countManaAbilityColor(land, Red), countManaAbilityColor(land, Green))
	}
}

func abilityRemovalTestPermanent(t *testing.T) (*Game, Player, *Permanent, Ability) {
	t.Helper()
	player := NewBasePlayer("Alice")
	g := NewGame(player, NewBasePlayer("Bob"))
	printed := NewActivatedAbility(GainLife(1), ManaCostOf("{1}"))
	card := NewCreature("Skilled Creature", "{2}", 2, 2,
		WithKeyword(Indestructible),
		WithKeyword(DoesNotUntapKW),
		WithAbility(printed),
	)
	card.SetOwner(player.PlayerID())
	return g, player, g.PutOnBattlefield(card, player.PlayerID()), printed
}

func TestRemoveAllAbilitiesRemovesAndRestoresPrintedAbilities(t *testing.T) {
	g, player, target, printed := abilityRemovalTestPermanent(t)
	sourceCard := NewEnchantment("Ability Blank", "{1}{W}")
	sourceCard.SetOwner(player.PlayerID())
	source := g.PutOnBattlefield(sourceCard, player.PlayerID())
	effect := RemoveAllAbilities(target.ID(), WhileOnBattlefield)
	effect.SetSourceID(source.ID())
	g.AddContinuousEffect(effect)

	if target.HasKeyword(Indestructible) || target.HasAttr(DoesNotUntapKW) || hasRuntimeAbility(target, printed) {
		t.Fatalf("abilities remain while removed: indestructible=%v doesNotUntap=%v activated=%v", target.HasKeyword(Indestructible), target.HasAttr(DoesNotUntapKW), hasRuntimeAbility(target, printed))
	}
	if !target.HasType(TypeCreature) || !target.HasAttr(AttrCanAttack) || !target.HasAttr(AttrCanBlock) {
		t.Fatal("ability removal changed card type or basic creature capabilities")
	}

	g.RemoveFromBattlefield(source)
	if !target.HasKeyword(Indestructible) || !target.HasAttr(DoesNotUntapKW) || !hasRuntimeAbility(target, printed) {
		t.Fatalf("abilities not restored: indestructible=%v doesNotUntap=%v activated=%v", target.HasKeyword(Indestructible), target.HasAttr(DoesNotUntapKW), hasRuntimeAbility(target, printed))
	}
}

func TestRemoveAllAbilitiesUsesLayerSixTimestampOrder(t *testing.T) {
	newGrantedAbility := func(target *Permanent) Ability {
		ability := NewActivatedAbility(GainLife(2), ManaCostOf("{2}"))
		ability.SetSource(target.ID())
		ability.SetController(target.ControllerID())
		return ability
	}
	grant := func(target *Permanent, ability Ability) ContinuousEffect {
		return FuncContinuousEffect(LayerAbility, Indefinite, func(g *Game, _ uuid.UUID) error {
			g.effects.GrantAttr(target.ID(), Flying)
			target.RuntimeAbilities = append(target.RuntimeAbilities, WrapGrantedAbility(ability))
			return nil
		}, func(*Game, uuid.UUID) bool { return true })
	}

	t.Run("later removal removes earlier grant", func(t *testing.T) {
		g, _, target, _ := abilityRemovalTestPermanent(t)
		ability := newGrantedAbility(target)
		g.AddContinuousEffect(grant(target, ability))
		g.AddContinuousEffect(RemoveAllAbilities(target.ID(), Indefinite))

		if target.HasKeyword(Flying) || hasRuntimeAbility(target, ability) {
			t.Fatalf("earlier grant survived later removal: flying=%v activated=%v", target.HasKeyword(Flying), hasRuntimeAbility(target, ability))
		}
	})

	t.Run("later grant survives earlier removal", func(t *testing.T) {
		g, _, target, _ := abilityRemovalTestPermanent(t)
		ability := newGrantedAbility(target)
		g.AddContinuousEffect(RemoveAllAbilities(target.ID(), Indefinite))
		g.AddContinuousEffect(grant(target, ability))

		if !target.HasKeyword(Flying) || !hasRuntimeAbility(target, ability) {
			t.Fatalf("later grant did not survive earlier removal: flying=%v activated=%v", target.HasKeyword(Flying), hasRuntimeAbility(target, ability))
		}
	})
}

func TestRemoveAllAbilitiesFromAllAffectsOnlyMatchingPermanents(t *testing.T) {
	g, player, creature, _ := abilityRemovalTestPermanent(t)
	landCard := NewLand("Indestructible Land", WithKeyword(Indestructible))
	landCard.SetOwner(player.PlayerID())
	land := g.PutOnBattlefield(landCard, player.PlayerID())
	sourceCard := NewEnchantment("Humility Test Source", "{2}{W}{W}")
	sourceCard.SetOwner(player.PlayerID())
	source := g.PutOnBattlefield(sourceCard, player.PlayerID())
	effect := RemoveAllAbilitiesFromAll(IsCreature)
	effect.SetSourceID(source.ID())
	g.AddContinuousEffect(effect)

	if creature.HasKeyword(Indestructible) {
		t.Fatal("matching creature retained indestructible")
	}
	if !land.HasKeyword(Indestructible) {
		t.Fatal("nonmatching land lost indestructible")
	}
}

func TestRemoveAllAbilitiesFromAllSeesEarlierLayerTypeChanges(t *testing.T) {
	player := NewBasePlayer("Alice")
	g := NewGame(player, NewBasePlayer("Bob"))
	landCard := NewLand("Animated Indestructible Land",
		WithSubTypes("Forest"),
		WithManaAbility(Green),
		WithKeyword(Indestructible),
	)
	landCard.SetOwner(player.PlayerID())
	land := g.PutOnBattlefield(landCard, player.PlayerID())
	animate := FuncContinuousEffect(LayerType, Indefinite, func(g *Game, _ uuid.UUID) error {
		g.effects.GrantAttr(land.ID(), AttrIsCreature)
		return nil
	}, func(*Game, uuid.UUID) bool { return true })
	g.AddContinuousEffect(animate)
	sourceCard := NewEnchantment("Humility Test Source", "{2}{W}{W}")
	sourceCard.SetOwner(player.PlayerID())
	source := g.PutOnBattlefield(sourceCard, player.PlayerID())
	effect := RemoveAllAbilitiesFromAll(IsCreature)
	effect.SetSourceID(source.ID())
	g.AddContinuousEffect(effect)

	if !land.HasType(TypeCreature) {
		t.Fatal("precondition: land was not animated")
	}
	if land.HasKeyword(Indestructible) {
		t.Fatal("ability removal missed a land made into a creature in layer 4")
	}
	if countManaAbilityColor(land, Green) != 0 {
		t.Fatal("animated Forest retained its intrinsic mana ability after losing all abilities")
	}
}

func TestRemoveAllAbilitiesSuppressesAndRestoresPrintedStaticAbilities(t *testing.T) {
	player := NewBasePlayer("Alice")
	g := NewGame(player, NewBasePlayer("Bob"))
	victimCard := NewCreature("Boost Recipient", "{G}", 1, 1)
	victimCard.SetOwner(player.PlayerID())
	victim := g.PutOnBattlefield(victimCard, player.PlayerID())
	abilitySourceCard := NewCreature("Static Ability Source", "{1}{G}", 1, 1,
		WithStaticAbility(FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
			g.MutablePermanent(victim.ID()).BoostPT(1, 1)
			return nil
		})),
	)
	abilitySourceCard.SetOwner(player.PlayerID())
	abilitySource := g.PutOnBattlefield(abilitySourceCard, player.PlayerID())
	if power, toughness := victim.CurrentPower(g), victim.CurrentToughness(g); power != 2 || toughness != 2 {
		t.Fatalf("P/T before ability removal = %d/%d, want 2/2", power, toughness)
	}
	removalSourceCard := NewEnchantment("Ability Removal Source", "{2}{W}")
	removalSourceCard.SetOwner(player.PlayerID())
	removalSource := g.PutOnBattlefield(removalSourceCard, player.PlayerID())
	effect := RemoveAllAbilities(abilitySource.ID(), WhileOnBattlefield)
	effect.SetSourceID(removalSource.ID())
	g.AddContinuousEffect(effect)

	if power, toughness := victim.CurrentPower(g), victim.CurrentToughness(g); power != 1 || toughness != 1 {
		t.Fatalf("P/T while static ability removed = %d/%d, want 1/1", power, toughness)
	}

	g.RemoveFromBattlefield(removalSource)
	if power, toughness := victim.CurrentPower(g), victim.CurrentToughness(g); power != 2 || toughness != 2 {
		t.Fatalf("P/T after static ability restored = %d/%d, want 2/2", power, toughness)
	}
}

func TestSubtypeFamilyStateClonesWithoutSharingSlices(t *testing.T) {
	_, permanent := subtypeTestPermanent(t)
	permanent.setSubtypeFamily(SubtypeCreature, []string{"Human"})
	permanent.addSubtypeFamily(SubtypeCreature, []string{"Druid"})
	clone := &Permanent{}
	clonePermanentInto(clone, permanent)

	clone.subtypeFamilyOverrides[SubtypeCreature][0] = "Wizard"
	clone.subtypeFamilyAdditions[SubtypeCreature][0] = "Rogue"
	if permanent.subtypeFamilyOverrides[SubtypeCreature][0] != "Human" {
		t.Fatal("clone subtype override mutation leaked to original")
	}
	if permanent.subtypeFamilyAdditions[SubtypeCreature][0] != "Druid" {
		t.Fatal("clone subtype addition mutation leaked to original")
	}
}
