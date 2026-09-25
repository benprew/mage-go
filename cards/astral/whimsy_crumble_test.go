package astral

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestWhimsyCrumble(t *testing.T) {
	for _, scenario := range []string{"destroy", "regenerate", "indestructible", "empty"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			card := mage.NewArtifact("Artifact", "{4}")
			card.SetOwner(b.PlayerID())
			if scenario != "empty" {
				p := g.PutOnBattlefield(card, b.PlayerID())
				if scenario == "regenerate" {
					g.AddRegenerationShield(p.ID())
				}
				if scenario == "indestructible" {
					p.GrantBaseAttr(core.Indestructible)
				}
			}
			creature := mage.NewCreature("Nonartifact", "", 2, 2)
			creature.SetOwner(a.PlayerID())
			g.PutOnBattlefield(creature, a.PlayerID())
			if err := applyWhimsyAction(g.Game, uuid.New(), a.PlayerID(), 5); err != nil {
				t.Fatal(err)
			}
			want := 24
			if scenario == "empty" {
				want = 20
			}
			if a.Life() != 20 || b.Life() != want {
				t.Fatalf("life A=%d B=%d, want A=20 B=%d", a.Life(), b.Life(), want)
			}
			if (g.FindPermanent(card.ID()) != nil) != (scenario == "indestructible") {
				t.Fatal("wrong artifact destination")
			}
			if g.FindPermanent(creature.ID()) == nil {
				t.Fatal("destroyed a nonartifact")
			}
		})
	}
}
