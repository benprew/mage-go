package antiquities

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestAntiquitiesSelfPumpTypedDefinitions(t *testing.T) {
	for _, tc := range []struct {
		name             string
		power, toughness int
	}{
		{"Atog", 2, 2}, {"Dragon Engine", 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card, err := mage.CreateCard(tc.name)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(tc.name, card); err != nil {
				t.Fatal(err)
			}
			ability := mage.UnwrapAbility(card.Abilities()[0]).(mage.ActivatedAbility)
			if ability.SorcerySpeed() || len(ability.Targets()) != 0 {
				t.Fatal("expected no targets and instant timing")
			}
			metadata := mage.InspectDefinition(ability.Effects()[0])
			if metadata.Role != "until" {
				t.Fatalf("effect role = %q, want until", metadata.Role)
			}
			props := metadata.Properties
			if props.Mass || props.Outcome != mage.OutcomeBenefit || props.PowerBoost != tc.power || props.ToughnessBoost != tc.toughness {
				t.Fatalf("boost metadata = %+v", props)
			}
		})
	}
}

func TestAtogSacrificePayment(t *testing.T) {
	for _, scenario := range []string{"own artifact", "opponent artifact", "phased artifact", "nonartifact", "artifact source"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a := g.GetPlayer(gametest.PlayerA)
			card, err := mage.CreateCard("Atog")
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "artifact source" {
				card.(*mage.BaseCard).AddType(core.TypeArtifact)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			payment := source.ID()
			if scenario != "artifact source" {
				owner, name := gametest.PlayerA, "Ornithopter"
				if scenario == "opponent artifact" {
					owner = gametest.PlayerB
				}
				if scenario == "nonartifact" {
					name = "Grizzly Bears"
				}
				payment = g.AddCard(core.ZoneBattlefield, owner, name)
				if scenario == "phased artifact" {
					g.PhaseOut(payment)
				}
			}
			source.Tapped = true
			g.SetStep(core.Upkeep)
			g.SetActivePlayerIndex(1)
			err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			if scenario != "own artifact" && scenario != "artifact source" {
				if err == nil || g.StackSize() != 0 || len(a.Graveyard()) != 0 {
					t.Fatal("invalid payment changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(payment) != nil || len(a.Graveyard()) != 1 || g.StackSize() != 1 {
				t.Fatal("sacrifice was not paid before resolution")
			}
			g.ResolveStack()
			if scenario == "artifact source" {
				g.AssertPermanentCount(gametest.PlayerA, "Atog", 0)
				g.AssertGraveyardCount(gametest.PlayerA, "Atog", 1)
			} else {
				g.AssertPowerToughness(gametest.PlayerA, "Atog", 3, 4)
			}
		})
	}
}
