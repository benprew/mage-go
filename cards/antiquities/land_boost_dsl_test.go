package antiquities

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestLandBoostTypedDefinition(t *testing.T) {
	card, err := mage.CreateCard("Mishra's Factory")
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Mishra's Factory", card); err != nil {
		t.Fatal(err)
	}
	ability := mage.UnwrapAbility(card.Abilities()[2]).(mage.ActivatedAbility)
	if ability.SorcerySpeed() || len(ability.Targets()) != 1 {
		t.Fatal("expected one target and instant timing")
	}
	if _, ok := ability.Targets()[0].(mage.TargetBinding); !ok {
		t.Fatal("missing typed creature target")
	}
	meta := mage.InspectDefinition(ability.Effects()[0])
	if meta.Role != "until" || meta.Properties.PowerBoost != 1 || meta.Properties.ToughnessBoost != 1 || meta.Properties.Mass || meta.Properties.Outcome != mage.OutcomeBenefit {
		t.Fatalf("metadata = %+v", meta)
	}
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	var want []uuid.UUID
	for _, player := range []*mage.BasePlayer{a, b} {
		for _, stats := range [][2]int{{1, 1}, {1, 2}, {2, 1}} {
			for _, subtype := range []string{"Assembly-Worker", "Soldier"} {
				p := g.PutOnBattlefield(mage.NewCreature("Candidate", "", stats[0], stats[1], mage.WithSubTypes(subtype)), player.PlayerID())
				if subtype == "Assembly-Worker" {
					want = append(want, p.ID())
				}
			}
		}
		g.PutOnBattlefield(mage.NewArtifact("Noncreature", "", mage.WithSubTypes("Assembly-Worker")), player.PlayerID())
	}
	got := ability.Targets()[0].Possible(a.PlayerID(), card, g)
	if len(got) != len(want) {
		t.Fatalf("targets = %v, want %v", got, want)
	}
	for _, id := range want {
		if !slices.Contains(got, id) {
			t.Fatalf("missing target %v", id)
		}
	}
}

func TestLandBoostResolution(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		t.Run(map[bool]string{false: "resolves", true: "target changes"}[invalid], func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			source := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mishra's Factory")
			target := g.PutOnBattlefield(mage.NewCreature("Target", "", 1, 1, mage.WithSubTypes("Assembly-Worker")), b.PlayerID())
			g.SetStep(core.Upkeep)
			g.SetActivePlayerIndex(1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source, 2, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source).Tapped || g.StackSize() != 1 {
				t.Fatal("tap payment or stack entry missing")
			}
			if invalid {
				g.AddContinuousEffect(mage.SetSubtypes(target.ID(), mage.SubtypeCreature, core.EndOfTurn, "Soldier"))
				g.ApplyContinuousEffects()
			}
			g.RemoveFromBattlefield(g.FindPermanent(source))
			g.ResolveStack()
			g.ApplyContinuousEffects()
			p, q := 1+1, 1+1
			if invalid {
				p, q = 1, 1
			}
			if target.CurrentPower(g.Game) != p || target.CurrentToughness(g.Game) != q {
				t.Fatalf("P/T = %d/%d, want %d/%d", target.CurrentPower(g.Game), target.CurrentToughness(g.Game), p, q)
			}
			if !invalid {
				clone := g.Clone()
				clone.GetEffects().RemoveEndOfTurn()
				clone.ApplyContinuousEffects()
				returned := clone.FindPermanent(target.ID())
				if returned.CurrentPower(clone) != 1 || returned.CurrentToughness(clone) != 1 {
					t.Fatal("boost did not expire")
				}
				if target.CurrentPower(g.Game) != 1+1 {
					t.Fatal("clone changed original boost")
				}
			}
		})
	}
}
