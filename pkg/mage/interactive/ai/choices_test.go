package ai

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func aiGame() (*mage.Game, *AIPlayer, *mage.BasePlayer) {
	bot := NewAIPlayer("Bot", &dummyStrategy{})
	opp := mage.NewBasePlayer("Opp")
	g := mage.NewGame(bot, opp)
	return g, bot, opp
}

func battlefieldCreature(g *mage.Game, ownerID uuid.UUID, name string, power, toughness int) *mage.Permanent {
	c := mage.NewCreature(name, "{1}", power, toughness)
	c.SetOwner(ownerID)
	perm := mage.NewPermanent(c, ownerID)
	perm.RevokeBaseAttr(core.AttrSummonSick)
	g.AddToBattlefield(perm)
	return perm
}

func TestChoosePermanent_SacrificeKeepsBest(t *testing.T) {
	g, bot, _ := aiGame()
	weak := battlefieldCreature(g, bot.PlayerID(), "Bird", 1, 1)
	strong := battlefieldCreature(g, bot.PlayerID(), "Dragon", 5, 5)

	got := bot.ChoosePermanent([]*mage.Permanent{strong, weak}, "sacrifice a creature", g)
	if got != weak {
		t.Fatalf("sacrifice should give up the weakest creature, got %s", got.Card.Name())
	}
}

func TestChoosePermanent_DestroyPicksBest(t *testing.T) {
	g, bot, opp := aiGame()
	weak := battlefieldCreature(g, opp.PlayerID(), "Bird", 1, 1)
	strong := battlefieldCreature(g, opp.PlayerID(), "Dragon", 5, 5)

	got := bot.ChoosePermanent([]*mage.Permanent{weak, strong}, "destroy target creature", g)
	if got != strong {
		t.Fatalf("destroy should pick the most valuable creature, got %s", got.Card.Name())
	}
}

func TestChooseCardsFromHand_DiscardsSurplusLandKeepsBomb(t *testing.T) {
	g, bot, _ := aiGame()
	// Flood the board with lands so a hand land is surplus.
	for range 5 {
		land := mage.NewLand("Forest", mage.WithManaAbility(core.Green))
		land.SetOwner(bot.PlayerID())
		perm := mage.NewPermanent(land, bot.PlayerID())
		g.AddToBattlefield(perm)
	}

	handLand := mage.NewLand("Forest", mage.WithManaAbility(core.Green))
	handLand.SetOwner(bot.PlayerID())
	bomb := mage.NewCreature("Dragon", "{4}{R}{R}", 6, 6)
	bomb.SetOwner(bot.PlayerID())
	bot.AddToHand(handLand)
	bot.AddToHand(bomb)

	discards := bot.ChooseCardsFromHand(1, "discard to hand size", g)
	if len(discards) != 1 || discards[0].ID() != handLand.ID() {
		t.Fatalf("should discard the surplus land, not the bomb; got %v", discards)
	}
}

func TestChooseCardsFromHand_KeepsLandWhenManaLight(t *testing.T) {
	g, bot, _ := aiGame()
	// No lands in play: the land is precious, pitch the filler spell instead.
	handLand := mage.NewLand("Forest", mage.WithManaAbility(core.Green))
	handLand.SetOwner(bot.PlayerID())
	filler := mage.NewCreature("Bird", "{1}", 1, 1)
	filler.SetOwner(bot.PlayerID())
	bot.AddToHand(handLand)
	bot.AddToHand(filler)

	discards := bot.ChooseCardsFromHand(1, "discard", g)
	if len(discards) != 1 || discards[0].ID() != filler.ID() {
		t.Fatalf("should keep the land when mana-light; got %v", discards)
	}
}

func TestChooseManaColor_MostNeededInHand(t *testing.T) {
	_, bot, _ := aiGame()
	c := mage.NewCreature("Goblins", "{R}{R}{R}", 3, 3)
	c.SetOwner(bot.PlayerID())
	bot.AddToHand(c)

	if got := bot.ChooseManaColor("add mana"); got != core.Red {
		t.Fatalf("expected Red (most-needed pip), got %v", got)
	}
}

func TestChooseNumber_DiscardMinElseMax(t *testing.T) {
	_, bot, _ := aiGame()
	if got := bot.ChooseNumber(0, 3, "discard cards"); got != 0 {
		t.Fatalf("discard should choose the minimum, got %d", got)
	}
	if got := bot.ChooseNumber(0, 3, "deal damage"); got != 3 {
		t.Fatalf("damage should choose the maximum, got %d", got)
	}
}

func TestChooseDamageDistribution_KillsMultipleCreatures(t *testing.T) {
	g, bot, opp := aiGame()
	a := battlefieldCreature(g, opp.PlayerID(), "GoblinA", 1, 2)
	b := battlefieldCreature(g, opp.PlayerID(), "GoblinB", 1, 3)

	dist := bot.ChooseDamageDistribution([]uuid.UUID{a.ID(), b.ID()}, 5, "Fireball", g)
	if dist[a.ID()] != 2 || dist[b.ID()] != 3 {
		t.Fatalf("should assign exactly lethal to each creature, got %v", dist)
	}
}

func TestChooseDamageDistribution_RemainderToFace(t *testing.T) {
	g, bot, opp := aiGame()
	a := battlefieldCreature(g, opp.PlayerID(), "GoblinA", 1, 2)

	dist := bot.ChooseDamageDistribution([]uuid.UUID{a.ID(), opp.PlayerID()}, 5, "Fireball", g)
	if dist[a.ID()] != 2 {
		t.Fatalf("creature should take lethal (2), got %v", dist)
	}
	if dist[opp.PlayerID()] != 3 {
		t.Fatalf("remainder (3) should go to the opponent's face, got %v", dist)
	}
}

func TestChooseCardFromLibrary_PrefersSpellOverLand(t *testing.T) {
	g, bot, _ := aiGame()
	land := mage.NewLand("Forest", mage.WithManaAbility(core.Green))
	land.SetOwner(bot.PlayerID())
	spell := mage.NewCreature("Dragon", "{4}{R}{R}", 6, 6)
	spell.SetOwner(bot.PlayerID())

	got := bot.ChooseCardFromLibrary([]mage.Card{land, spell}, "search", g)
	if got != spell {
		t.Fatalf("tutor should fetch the spell over a land, got %s", got.Name())
	}
}

func TestChooseTargets_PrefersOpponentBestCreature(t *testing.T) {
	g, bot, opp := aiGame()
	mine := battlefieldCreature(g, bot.PlayerID(), "MyBear", 2, 2)
	theirsWeak := battlefieldCreature(g, opp.PlayerID(), "TheirBird", 1, 1)
	theirsStrong := battlefieldCreature(g, opp.PlayerID(), "TheirDragon", 5, 5)

	got := bot.ChooseTargets([]uuid.UUID{mine.ID(), theirsWeak.ID(), theirsStrong.ID()}, 1, 1, g)
	if len(got) != 1 || got[0] != theirsStrong.ID() {
		t.Fatalf("should target the opponent's best creature, got %v", got)
	}
}

func TestChooseScryPlacement_BottomsFloodedLand(t *testing.T) {
	g, bot, _ := aiGame()
	for range 5 {
		land := mage.NewLand("Forest", mage.WithManaAbility(core.Green))
		land.SetOwner(bot.PlayerID())
		perm := mage.NewPermanent(land, bot.PlayerID())
		g.AddToBattlefield(perm)
	}

	topLand := mage.NewLand("Forest", mage.WithManaAbility(core.Green))
	topLand.SetOwner(bot.PlayerID())
	topSpell := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	topSpell.SetOwner(bot.PlayerID())

	bottom, top := bot.ChooseScryPlacement([]mage.Card{topLand, topSpell}, "scry", g)
	if len(bottom) != 1 || bottom[0] != topLand.ID() {
		t.Fatalf("flooded surplus land should be bottomed, got bottom=%v", bottom)
	}
	if len(top) != 1 || top[0] != topSpell.ID() {
		t.Fatalf("spell should be kept on top, got top=%v", top)
	}
}
