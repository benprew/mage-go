package limited

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func castAnteSpell(t *testing.T, g *gametest.TestGame, name string, targets ...mage.Card) {
	t.Helper()
	player := g.GetPlayer(gametest.PlayerA)
	player.ManaPool().Add(core.Black, 3)
	ids := make([]uuid.UUID, 0, len(targets))
	for _, target := range targets {
		ids = append(ids, target.ID())
	}
	g.SetStep(core.PrecombatMain)
	if err := g.CastSpellByName(player.PlayerID(), name, ids); err != nil {
		t.Fatalf("cast %s: %v", name, err)
	}
	g.ResolveTopOfStack()
}

func TestAnteSpellsUsePipelines(t *testing.T) {
	for _, name := range []string{"Contract from Below", "Darkpact", "Demonic Attorney"} {
		t.Run(name, func(t *testing.T) {
			card, err := mage.CreateCard(name)
			if err != nil {
				t.Fatal(err)
			}
			ability := mage.UnwrapAbility(card.Abilities()[0]).(*mage.SpellAbility)
			if _, ok := ability.Effects()[0].(*mage.PipelineData); !ok {
				t.Fatalf("%s effect is %T, want *mage.PipelineData", name, ability.Effects()[0])
			}
		})
	}
}

func TestContractFromBelow(t *testing.T) {
	g := gametest.NewTestGameWithAnte(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Contract from Below")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Black Lotus")
	for range 7 {
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Island")
	}

	castAnteSpell(t, g, "Contract from Below")

	g.AssertAnteCount(gametest.PlayerA, "Black Lotus", 1)
	if got := len(g.GetPlayer(gametest.PlayerA).Hand()); got != 7 {
		t.Fatalf("hand size = %d, want 7", got)
	}
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 1)
}

func TestDarkpactTargetsOwnedCardInSharedAnteAndExchangesExactly(t *testing.T) {
	g := gametest.NewTestGameWithAnte(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Darkpact")
	g.AddCard(core.ZoneAnte, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneAnte, gametest.PlayerB, "Hill Giant")
	owned := g.GetPlayer(gametest.PlayerB).Ante()[0].ID()
	opponentOwned := g.GetPlayer(gametest.PlayerB).Ante()[1].ID()
	if err := g.ChangeOwner(owned, g.GetPlayer(gametest.PlayerA).PlayerID()); err != nil {
		t.Fatal(err)
	}
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Black Lotus")
	top := g.GetPlayer(gametest.PlayerA).Library()[0].ID()
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Island")

	card, err := mage.CreateCard("Darkpact")
	if err != nil {
		t.Fatal(err)
	}
	target := card.CastTargets()[0]
	possible := target.Possible(g.GetPlayer(gametest.PlayerA).PlayerID(), card, g.Game)
	if !slices.Contains(possible, owned) || slices.Contains(possible, opponentOwned) {
		t.Fatalf("Darkpact targets = %v, want own ante card only", possible)
	}

	castAnteSpell(t, g, "Darkpact", g.FindCardAnywhere(owned))

	g.AssertLibraryTop(gametest.PlayerA, "Grizzly Bears", "Island")
	g.AssertAnteCount(gametest.PlayerA, "Grizzly Bears", 0)
	g.AssertAnteCount(gametest.PlayerB, "Grizzly Bears", 0)
	g.AssertAnteCount(gametest.PlayerA, "Black Lotus", 1)
	if g.CardZone(top) != core.ZoneAnte {
		t.Fatalf("library top zone = %s, want Ante", g.CardZone(top))
	}
}

func TestDemonicAttorneyAntesEachPlayersLibraryTop(t *testing.T) {
	g := gametest.NewTestGameWithAnte(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Demonic Attorney")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Black Lotus")
	aTop := g.GetPlayer(gametest.PlayerA).Library()[0].ID()
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Island")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Mox Ruby")
	bTop := g.GetPlayer(gametest.PlayerB).Library()[0].ID()
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Mountain")

	castAnteSpell(t, g, "Demonic Attorney")

	if g.CardZone(aTop) != core.ZoneAnte || g.CardZone(bTop) != core.ZoneAnte {
		t.Fatal("each player's top card was not moved to ante")
	}
	g.AssertLibraryTop(gametest.PlayerA, "Island")
	g.AssertLibraryTop(gametest.PlayerB, "Mountain")
}
