package legends

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestDivineOfferingTypedDefinition(t *testing.T) {
	card, err := mage.CreateCard("Divine Offering")
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	ability := mage.UnwrapAbility(card.Abilities()[0]).(*mage.SpellAbility)
	if _, ok := ability.Targets()[0].(mage.TargetBinding); !ok {
		t.Fatal("missing typed artifact target")
	}
	if !ability.Effects()[0].Properties().Destroys {
		t.Fatal("missing destruction metadata")
	}
}

func TestDivineOfferingOutcomes(t *testing.T) {
	for _, scenario := range []string{"destroy", "indestructible", "regenerate", "illegal", "return", "own", "zero", "X"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			owner := b.PlayerID()
			if scenario == "own" {
				owner = a.PlayerID()
			}
			cost, gain := "{4}", 4
			if scenario == "zero" {
				cost, gain = "", 0
			}
			if scenario == "X" {
				cost, gain = "{X}{2}", 2
			}
			card := mage.NewArtifact("Offering target", cost)
			card.SetOwner(owner)
			target := g.PutOnBattlefield(card, owner)
			if scenario == "indestructible" {
				target.GrantBaseAttr(core.Indestructible)
			}
			if scenario == "regenerate" {
				g.AddRegenerationShield(target.ID())
			}
			spell, err := mage.CreateCard("Divine Offering")
			if err != nil {
				t.Fatal(err)
			}
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.White, 2)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if scenario == "illegal" || scenario == "return" {
				g.RemoveFromBattlefield(target)
				if scenario == "return" {
					g.PutOnBattlefield(card, owner)
				}
				gain = 0
			}
			g.ResolveStack()
			if a.Life() != 20+gain || b.Life() != 20 {
				t.Fatalf("life A=%d B=%d, want A=%d B=20", a.Life(), b.Life(), 20+gain)
			}
			survives := scenario == "indestructible" || scenario == "regenerate" || scenario == "return"
			if (g.FindPermanent(card.ID()) != nil) != survives {
				t.Fatal("wrong artifact destination")
			}
			if scenario == "regenerate" && !g.FindPermanent(card.ID()).Tapped {
				t.Fatal("regeneration did not tap artifact")
			}
		})
	}
}
