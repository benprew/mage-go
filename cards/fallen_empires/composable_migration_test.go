package fallen_empires

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestSporeCloudCombatSnapshot(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Spore Cloud")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.Attack(1, gametest.PlayerA, "Hill Giant")
	g.Block(1, gametest.PlayerB, "Grizzly Bears", "Hill Giant")
	g.CastSpell(1, core.DeclareBlockers, gametest.PlayerA, "Spore Cloud")
	g.StopAt(3, core.PrecombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertTapped(gametest.PlayerB, "Grizzly Bears", true)
	g.AssertTapped(gametest.PlayerA, "Hill Giant", true)
}

func TestGoblinKitesGrantsFlying(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Kites")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Goblin Kites", "Grizzly Bears")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Flying, true)
}

func TestSaprolingSacrificeAbilities(t *testing.T) {
	for _, name := range []string{"Elvish Farmer", "Thallid Devourer"} {
		t.Run(name, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			source := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, name)
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Thallid")
			g.FindPermanent(source).AddCounter(core.Spore, 3)
			player := g.GetPlayer(gametest.PlayerA)
			if err := g.ActivateAbilityByIndex(player.PlayerID(), source, 1, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			g.AssertPermanentCount(gametest.PlayerA, "Saproling", 1)
			for _, p := range g.Battlefield {
				if p.Name() == "Saproling" && !slices.Contains(p.Colors(), core.Green) {
					t.Fatal("Saproling is not green")
				}
			}
			if err := g.ActivateAbilityByIndex(player.PlayerID(), source, 2, nil); err != nil {
				t.Fatal(err)
			}
			g.AssertPermanentCount(gametest.PlayerA, "Saproling", 0)
			g.ResolveStack()
			if name == "Elvish Farmer" {
				g.AssertLife(gametest.PlayerA, 22)
			} else {
				g.AssertPowerToughness(gametest.PlayerA, name, 3, 4)
			}
		})
	}
}

func TestGoblinKitesDelayedSacrifice(t *testing.T) {
	for _, scenario := range []string{"win", "loss", "source leaves", "creature returns"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			source := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Kites")
			creature := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
			g.SetCoinFlipResults([]bool{scenario == "win"})
			player := g.GetPlayer(gametest.PlayerA)
			if err := g.ActivateAbilityByIndex(player.PlayerID(), source, 0, []uuid.UUID{creature}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if scenario == "source leaves" {
				g.RemoveFromBattlefield(g.FindPermanent(source))
			}
			if scenario == "creature returns" {
				card := g.FindPermanent(creature).Card
				g.RemoveFromBattlefield(g.FindPermanent(creature))
				g.PutOnBattlefield(card, player.PlayerID())
			}
			g.StopAt(2, core.PrecombatMain)
			g.Execute()
			want := 0
			if scenario == "win" || scenario == "creature returns" {
				want = 1
			}
			g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", want)
			if want == 1 {
				g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Flying, false)
			}
		})
	}
}

func TestGoblinKitesRejectsIllegalTargets(t *testing.T) {
	for _, scenario := range []string{"opponent", "toughness", "noncreature"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			source := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Kites")
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
			owner, name := gametest.PlayerA, "Grizzly Bears"
			switch scenario {
			case "opponent":
				owner = gametest.PlayerB
			case "toughness":
				name = "Hill Giant"
			case "noncreature":
				name = "Forest"
			}
			target := g.AddCard(core.ZoneBattlefield, owner, name)
			if err := g.ActivateAbilityByIndex(g.GetPlayer(gametest.PlayerA).PlayerID(), source, 0, []uuid.UUID{target}); err == nil {
				t.Fatal("illegal target accepted")
			}
			if g.StackSize() != 0 {
				t.Fatal("illegal activation reached the stack")
			}
		})
	}
}

func TestBalmOfRestorationPreventionMode(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Balm of Restoration")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	target := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.ChooseMode(gametest.PlayerA, 1)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Balm of Restoration", "Hill Giant")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Balm of Restoration", 0)
	g.DealDamageToPermanent(g.FindPermanent(target), 3, uuid.Nil)
	if got := g.FindPermanent(target).Damage; got != 1 {
		t.Fatalf("damage = %d, want 1", got)
	}
}
