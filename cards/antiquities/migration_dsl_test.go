package antiquities

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestAntiquitiesSacrificeArtifactSource(t *testing.T) {
	for _, name := range []string{"Sage of Lat-Nam", "Dwarven Weaponsmith", "Orcish Mechanics"} {
		t.Run(name, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a := g.GetPlayer(gametest.PlayerA)
			card, err := mage.CreateCard(name)
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			card.(*mage.BaseCard).AddType(core.TypeArtifact)
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.GrantBaseAttr(core.Haste)
			g.SetStep(core.Upkeep)
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
			ability := mage.UnwrapAbility(source.RuntimeAbilities[0]).(mage.ActivatedAbility)
			targets := ability.Targets()
			var ids []uuid.UUID
			if len(targets) > 0 {
				ids = targets[0].Possible(a.PlayerID(), card, g.Game)[:1]
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, ids); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(source.ID()) != nil {
				t.Fatal("source was not sacrificed as payment")
			}
			g.ResolveStack()
			if name == "Sage of Lat-Nam" && len(a.Hand()) != 1 {
				t.Fatal("ability did not draw after the source left")
			}
		})
	}
}

func TestUrzaLandsTypedMana(t *testing.T) {
	for _, tc := range []struct {
		name, first, second string
		amount              int
	}{
		{"Urza's Mine", "Power-Plant", "Tower", 2},
		{"Urza's Power Plant", "Mine", "Tower", 2},
		{"Urza's Tower", "Mine", "Power-Plant", 3},
	} {
		for _, scenario := range []string{"complete", "missing", "wrong subtype", "duplicate", "opponent", "phased"} {
			t.Run(tc.name+"/"+scenario, func(t *testing.T) {
				g := gametest.NewTestGame(t)
				a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
				id := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, tc.name)
				source := g.FindPermanent(id)
				if !mage.IsActivatedManaAbility(source.RuntimeAbilities[0]) {
					t.Fatal("ability must resolve as a mana ability")
				}
				first := mage.NewLand("first", mage.WithSubTypes("Urza's", tc.first))
				if scenario == "wrong subtype" {
					first = mage.NewLand("first", mage.WithSubTypes(tc.first))
				}
				owner := a.PlayerID()
				if scenario == "opponent" {
					owner = b.PlayerID()
				}
				p := g.PutOnBattlefield(first, owner)
				if scenario == "phased" {
					g.PhaseOut(p.ID())
				}
				if scenario != "missing" {
					g.PutOnBattlefield(mage.NewLand("second", mage.WithSubTypes("Urza's", tc.second)), a.PlayerID())
				}
				if scenario == "duplicate" {
					g.PutOnBattlefield(mage.NewLand("extra", mage.WithSubTypes("Urza's", tc.first, tc.second)), a.PlayerID())
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), id, 0, nil); err != nil {
					t.Fatal(err)
				}
				want := 1
				if scenario == "complete" || scenario == "duplicate" {
					want = tc.amount
				}
				if g.StackSize() != 0 || a.ManaPool().Count(core.Colorless) != want {
					t.Fatalf("stack=%d mana=%d, want immediate %d", g.StackSize(), a.ManaPool().Count(core.Colorless), want)
				}
			})
		}
	}
}
