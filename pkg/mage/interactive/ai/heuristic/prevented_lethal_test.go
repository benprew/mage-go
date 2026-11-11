package heuristic

import (
	"slices"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
)

func TestAttackers_GaseousFormLeviathan(t *testing.T) {
	for _, healthyCount := range []int{0, 1} {
		t.Run(map[int]string{0: "only prevented creature", 1: "healthy creature available"}[healthyCount], func(t *testing.T) {
			g, pa, pb := makeGame()
			pb.SetLife(3)
			put := func(name string, owner mage.Player) *mage.Permanent {
				t.Helper()
				card, err := mage.CreateCard(name)
				if err != nil {
					t.Fatal(err)
				}
				card.SetOwner(owner.PlayerID())
				perm := g.PutOnBattlefield(card, owner.PlayerID())
				perm.RevokeBaseAttr(core.AttrSummonSick)
				return perm
			}
			prevented := put("Segovian Leviathan", pa)
			aura := put("Gaseous Form", pb)
			g.Attach(aura.ID(), prevented.ID())
			put("Island", pb)
			var healthy *mage.Permanent
			if healthyCount > 0 {
				healthy = put("Segovian Leviathan", pa)
			}
			g.ApplyContinuousEffects()
			g.SetStep(core.DeclareAttackers)
			attackers := New(ai.MidrangeWeighted).Attackers(pa, g)
			if healthy == nil {
				if len(attackers) != 0 {
					t.Fatalf("prevented creature cannot deal lethal damage: %v", attackers)
				}
			} else {
				if !slices.Contains(attackers, healthy.ID()) {
					t.Fatalf("attack must include the healthy Leviathan: %v", attackers)
				}
				clone := g.Clone()
				clone.ExecuteAttackers(pa.PlayerID(), attackers)
				clone.ExecuteCombatDamage()
				if got := clone.GetPlayer(pb.PlayerID()).Life(); got != 0 {
					t.Fatalf("attack left opponent at %d life", got)
				}
			}
			if pb.Life() != 3 || prevented.Tapped || (healthy != nil && healthy.Tapped) {
				t.Fatal("attack selection changed the real game")
			}
		})
	}
}

func TestAttackers_IncludeEnoughDamageForPreventionShield(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(3)
	first := makePerm("First", "{2}{U}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.UnblockableKW))
	second := makePerm("Second", "{2}{U}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.UnblockableKW))
	g.AddToBattlefield(first, second)
	g.AddPreventionShield(pb.PlayerID(), 3)
	g.SetStep(core.DeclareAttackers)
	attackers := New(ai.MidrangeWeighted).Attackers(pa, g)
	if len(attackers) != 2 {
		t.Fatalf("both attackers are needed to get past the shield: %v", attackers)
	}
	g.DealDamageToPlayer(pb, 3, first.ID())
	if pb.Life() != 3 {
		t.Fatal("attack selection consumed the real prevention shield")
	}
}
