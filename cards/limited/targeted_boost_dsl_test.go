package limited

import (
	"fmt"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestTargetedBoostTypedDefinitions(t *testing.T) {
	for _, name := range []string{"Howl from Beyond", "Righteousness"} {
		t.Run(name, func(t *testing.T) {
			card, err := mage.CreateCard(name)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(name, card); err != nil {
				t.Fatal(err)
			}
			ability := mage.UnwrapAbility(card.Abilities()[0]).(*mage.SpellAbility)
			if len(ability.Targets()) != 1 {
				t.Fatal("expected one target")
			}
			if _, ok := ability.Targets()[0].(mage.TargetBinding); !ok {
				t.Fatal("missing typed creature target")
			}
			props := ability.Effects()[0].Properties()
			if props.Outcome != mage.OutcomeBenefit || props.Mass {
				t.Fatalf("boost metadata = %+v", props)
			}
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			var want []uuid.UUID
			for _, player := range []*mage.BasePlayer{a, b} {
				for _, blocking := range []bool{false, true} {
					p := g.PutOnBattlefield(mage.NewCreature("Creature", "", 2, 2), player.PlayerID())
					if blocking {
						attacker := uuid.New()
						g.GetCombat().AddAttacker(attacker, player.PlayerID())
						g.GetCombat().AddBlocker(p.ID(), attacker)
					}
					if name == "Howl from Beyond" || blocking {
						want = append(want, p.ID())
					}
				}
				g.PutOnBattlefield(mage.NewArtifact("Artifact", ""), player.PlayerID())
			}
			got := ability.Targets()[0].Possible(a.PlayerID(), card, g)
			if len(got) != len(want) {
				t.Fatalf("targets = %v, want %v", got, want)
			}
			for _, id := range want {
				if !slices.Contains(got, id) {
					t.Fatalf("missing legal target %v", id)
				}
			}
		})
	}
}

func TestTargetedBoostResolution(t *testing.T) {
	for _, tc := range []struct {
		name                string
		x, power, toughness int
		stopBlocking        bool
	}{
		{"Howl from Beyond", 0, 2, 2, false},
		{"Howl from Beyond", 5, 7, 2, false},
		{"Righteousness", 0, 9, 9, false},
		{"Righteousness", 0, 2, 2, true},
	} {
		t.Run(fmt.Sprintf("%s/x=%d/stopBlocking=%v", tc.name, tc.x, tc.stopBlocking), func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			card, err := mage.CreateCard(tc.name)
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			creature := mage.NewCreature("Target", "", 2, 2)
			creature.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(creature, b.PlayerID())
			attacker := uuid.New()
			g.GetCombat().AddAttacker(attacker, b.PlayerID())
			g.GetCombat().AddBlocker(p.ID(), attacker)
			ability := mage.UnwrapAbility(card.Abilities()[0]).(*mage.SpellAbility)
			obj := &mage.StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{p.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects(), XValue: tc.x}
			g.PushStack(obj)
			if tc.stopBlocking {
				g.RemoveFromCombat(p.ID())
			}
			g.ResolveStack()
			g.SetXValue(12)
			g.RemoveFromCombat(p.ID())
			check := func(game *mage.Game, power, toughness int) {
				t.Helper()
				game.ApplyContinuousEffects()
				p := game.FindPermanent(creature.ID())
				if p.CurrentPower(game) != power || p.CurrentToughness(game) != toughness {
					t.Fatalf("P/T = %d/%d, want %d/%d", p.CurrentPower(game), p.CurrentToughness(game), power, toughness)
				}
			}
			check(g, tc.power, tc.toughness)
			clone := g.Clone()
			clone.GetEffects().RemoveEndOfTurn()
			check(clone, 2, 2)
			check(g, tc.power, tc.toughness)
			g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
			g.PutOnBattlefield(creature, b.PlayerID())
			check(g, 2, 2)
		})
	}
}
