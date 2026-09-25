package antiquities

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestCrumbleTypedDefinition(t *testing.T) {
	card, err := mage.CreateCard("Crumble")
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

func TestCrumbleOutcomes(t *testing.T) {
	for _, scenario := range []string{"destroy", "indestructible", "regenerate", "illegal", "return", "own", "zero", "X", "control"} {
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
			card := mage.NewArtifact("Crumble target", cost)
			card.SetOwner(owner)
			target := g.PutOnBattlefield(card, owner)
			if scenario == "indestructible" {
				target.GrantBaseAttr(core.Indestructible)
			}
			if scenario == "regenerate" {
				g.AddRegenerationShield(target.ID())
			}
			spell, err := mage.CreateCard("Crumble")
			if err != nil {
				t.Fatal(err)
			}
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.Green, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if scenario == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: uuid.New(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			if scenario == "illegal" || scenario == "return" {
				g.RemoveFromBattlefield(target)
				if scenario == "return" {
					g.PutOnBattlefield(card, owner)
				}
				gain = 0
			}
			g.ResolveStack()
			wantA, wantB := 20, 20+gain
			if scenario == "own" || scenario == "control" {
				wantA, wantB = 20+gain, 20
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life A=%d B=%d, want A=%d B=%d", a.Life(), b.Life(), wantA, wantB)
			}
			survives := scenario == "indestructible" || scenario == "return"
			if (g.FindPermanent(card.ID()) != nil) != survives {
				t.Fatal("wrong artifact destination")
			}
		})
	}
}
