package legends

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestComposableColorSpellsMultipleTargets(t *testing.T) {
	for _, tc := range []struct {
		name  string
		color core.Color
	}{
		{"Dwarven Song", core.Red}, {"Heaven's Gate", core.White}, {"Sea Kings' Blessing", core.Blue}, {"Sylvan Paradise", core.Green}, {"Touch of Darkness", core.Black},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a := g.GetPlayer(gametest.PlayerA)
			first := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
			second := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Azure Drake")
			g.AddCard(core.ZoneHand, gametest.PlayerA, tc.name)
			a.ManaPool().Add(tc.color, 1)
			if err := g.CastSpellByID(a.PlayerID(), a.Hand()[0].ID(), []uuid.UUID{first, second}, 0); err != nil {
				t.Fatal(err)
			}
			g.RemoveFromBattlefield(g.FindPermanent(first))
			g.ResolveStack()
			colors := g.FindPermanent(second).Colors()
			if len(colors) != 1 || colors[0] != tc.color {
				t.Fatalf("colors = %v", colors)
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			colors = g.FindPermanent(second).Colors()
			if len(colors) != 1 || colors[0] != core.Blue {
				t.Fatalf("colors after cleanup = %v", colors)
			}
		})
	}
}

func TestComposableSpiritShackleCapturesHost(t *testing.T) {
	for _, change := range []string{"move aura", "remove aura", "return host"} {
		t.Run(change, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			host := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Durkwood Boars")
			other := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Azure Drake")
			aura := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Spirit Shackle")
			g.Attach(aura, host)
			g.TapPermanent(g.FindPermanent(host))
			switch change {
			case "move aura":
				g.Attach(aura, other)
			case "remove aura":
				g.RemoveFromBattlefield(g.FindPermanent(aura))
			case "return host":
				card := g.FindPermanent(host).Card
				g.RemoveFromBattlefield(g.FindPermanent(host))
				g.PutOnBattlefield(card, card.Owner())
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 {
				t.Fatal("expected one untargeted trigger")
			}
			g.ResolveStack()
			want := 1
			if change == "return host" {
				want = 0
			}
			if int(g.FindPermanent(host).Counters[core.M0M2]) != want || g.FindPermanent(other).Counters[core.M0M2] != 0 {
				t.Fatal("counter applied to the wrong incarnation")
			}
		})
	}
}

func TestComposableCaretakerCanSacrificeItself(t *testing.T) {
	g := gametest.NewTestGame(t)
	source := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hell's Caretaker")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
	a := g.GetPlayer(gametest.PlayerA)
	target := a.Graveyard()[0].ID()
	g.SetStep(core.Upkeep)
	g.SetActivePlayerIndex(1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source, 0, []uuid.UUID{target}); err == nil {
		t.Fatal("activated during opponent upkeep")
	}
	g.SetActivePlayerIndex(0)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source, 0, []uuid.UUID{target}); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(source) != nil {
		t.Fatal("source was not sacrificed as a cost")
	}
	g.ResolveStack()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Hell's Caretaker", 1)
}

func TestComposableAlabasterPotionModes(t *testing.T) {
	card, err := mage.CreateCard("Alabaster Potion")
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g := gametest.NewTestGame(t)
	a := g.GetPlayer(gametest.PlayerA)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Alabaster Potion")
	a.ManaPool().Add(core.White, 5)
	if err := g.CastSpellByID(a.PlayerID(), a.Hand()[0].ID(), []uuid.UUID{a.PlayerID()}, 3); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.AssertLife(gametest.PlayerA, 23)
}

func TestComposableAlabasterPotionPrevention(t *testing.T) {
	g := gametest.NewTestGame(t)
	a := g.GetPlayer(gametest.PlayerA)
	creature := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Durkwood Boars")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Alabaster Potion")
	a.ManaPool().Add(core.White, 5)
	if err := g.CastSpellByID(a.PlayerID(), a.Hand()[0].ID(), []uuid.UUID{creature}, 3); err == nil {
		t.Fatal("life-gain mode accepted a creature")
	}
	g.ChooseMode(gametest.PlayerA, 1)
	if err := g.CastSpellByID(a.PlayerID(), a.Hand()[0].ID(), []uuid.UUID{creature}, 3); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.DealDamageToPermanent(g.FindPermanent(creature), 4, uuid.New())
	if g.FindPermanent(creature).Damage != 1 {
		t.Fatalf("damage = %d, want 1", g.FindPermanent(creature).Damage)
	}
}
