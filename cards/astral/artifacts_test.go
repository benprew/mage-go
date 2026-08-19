package astral

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

const pandoraTestCreature = "Pandora Test Angel"

func registerPandoraTestCards() {
	if mage.CardRegistered(pandoraTestCreature) {
		return
	}
	mage.Register(pandoraTestCreature, func() mage.Card {
		return mage.NewCreature(pandoraTestCreature, "{3}{W}", 3, 4,
			mage.WithSubTypes("Angel"),
			mage.WithKeyword(core.Flying),
			mage.WithAbility(mage.EntersBattlefieldTrigger(mage.GainLife(3), false)),
		)
	})
}

func activatePandorasBox(t *testing.T, g *gametest.TestGame) {
	t.Helper()
	player := g.GetPlayer(gametest.PlayerA)
	box := g.FindPermanentByName("Pandora's Box", player.PlayerID())
	if box == nil {
		t.Fatal("Pandora's Box not found")
	}
	player.ManaPool().Add(core.Colorless, 3)
	if err := g.ActivateAbilityByIndex(player.PlayerID(), box.ID(), 0, nil); err != nil {
		t.Fatalf("activate Pandora's Box: %v", err)
	}
	g.ResolveStack()
}

func TestPandorasBox_CreatureOnlyLibraryCopyAndIndependentFlips(t *testing.T) {
	registerPandoraTestCards()
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pandora's Box")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Pandora's Box")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, pandoraTestCreature)

	a := g.GetPlayer(gametest.PlayerA)
	b := g.GetPlayer(gametest.PlayerB)
	creatureID := b.Library()[0].ID()
	g.SetRandomResults([]int{0})
	g.SetCoinFlipResults([]bool{true, false})

	activatePandorasBox(t, g)

	if len(a.Library()) != 1 || a.Library()[0].Name() != "Pandora's Box" {
		t.Fatalf("PlayerA library changed: %#v", a.Library())
	}
	if len(b.Library()) != 1 || b.Library()[0].ID() != creatureID {
		t.Fatalf("chosen creature moved or PlayerB library changed: %#v", b.Library())
	}
	g.AssertPermanentCount(gametest.PlayerA, pandoraTestCreature, 1)
	g.AssertPermanentCount(gametest.PlayerB, pandoraTestCreature, 0)
	g.AssertLife(gametest.PlayerA, 23)
	g.AssertLife(gametest.PlayerB, 20)

	tokenCopy := g.FindPermanentByName(pandoraTestCreature, a.PlayerID())
	if tokenCopy == nil {
		t.Fatal("token copy not found")
	}
	if !tokenCopy.IsToken || !tokenCopy.Card.IsToken() {
		t.Fatal("copy is not represented as a token")
	}
	if tokenCopy.Card.ID() == creatureID {
		t.Fatal("token reused the library card identity")
	}
	if tokenCopy.CurrentPower(g.Game) != 3 || tokenCopy.CurrentToughness(g.Game) != 4 || !tokenCopy.HasSubType("Angel") || !tokenCopy.HasKeyword(core.Flying) {
		t.Fatalf("token did not copy characteristics: %d/%d subtypes=%v flying=%v", tokenCopy.CurrentPower(g.Game), tokenCopy.CurrentToughness(g.Game), tokenCopy.Card.SubTypes(), tokenCopy.HasKeyword(core.Flying))
	}
}

func TestPandorasBox_BothHeadsCreateDistinctTokens(t *testing.T) {
	registerPandoraTestCards()
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pandora's Box")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, pandoraTestCreature)
	g.SetRandomResults([]int{0})
	g.SetCoinFlipResults([]bool{true, true})

	activatePandorasBox(t, g)

	a := g.GetPlayer(gametest.PlayerA)
	b := g.GetPlayer(gametest.PlayerB)
	aCopy := g.FindPermanentByName(pandoraTestCreature, a.PlayerID())
	bCopy := g.FindPermanentByName(pandoraTestCreature, b.PlayerID())
	if aCopy == nil || bCopy == nil {
		t.Fatalf("expected one copy for each player: a=%v b=%v", aCopy, bCopy)
	}
	if aCopy.ID() == bCopy.ID() || aCopy.Card.ID() == bCopy.Card.ID() {
		t.Fatal("players received the same token object")
	}
	if len(aCopy.RuntimeAbilities) != 1 || len(bCopy.RuntimeAbilities) != 1 ||
		aCopy.RuntimeAbilities[0].Source() != aCopy.ID() || bCopy.RuntimeAbilities[0].Source() != bCopy.ID() {
		t.Fatal("token copies do not have independent ability instances")
	}
	if aCopy.Card.Owner() != a.PlayerID() || bCopy.Card.Owner() != b.PlayerID() {
		t.Fatalf("token owners are wrong: a=%s b=%s", aCopy.Card.Owner(), bCopy.Card.Owner())
	}
	g.AssertLife(gametest.PlayerA, 23)
	g.AssertLife(gametest.PlayerB, 23)
}

func TestPandorasBox_NoCreatureCardsDoesNothing(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pandora's Box")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Pandora's Box")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Pandora's Box")
	g.SetCoinFlipResults([]bool{true, true})

	activatePandorasBox(t, g)

	for _, perm := range g.AllBattlefield() {
		if perm.Name() != "Pandora's Box" {
			t.Fatalf("unexpected permanent created: %s", perm.Name())
		}
	}
	g.AssertPermanentCount(gametest.PlayerA, "Pandora's Box", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Pandora's Box", 0)
}
