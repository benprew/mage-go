package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronNontramplePlayer struct {
	*mage.BasePlayer
	calls, orders int
}

func (p *ronNontramplePlayer) GetBlockerOrder(_ *mage.Game, _ *mage.Permanent, _ []*mage.Permanent, _ int) []uuid.UUID {
	p.orders++
	return nil
}
func (p *ronNontramplePlayer) GetCombatDamageAssignment(_ *mage.Game, _ *mage.Permanent, blockers []*mage.Permanent, amount int) map[uuid.UUID]int {
	p.calls++
	return map[uuid.UUID]int{blockers[0].ID(): 1, blockers[1].ID(): amount - 1}
}

func TestRONNontrampleAssignment(t *testing.T) {
	for _, tc := range []struct {
		name        string
		color       core.Color
		cost, power int
	}{
		{"Craw Wurm", core.Green, 6, 6}, {"Hill Giant", core.Red, 4, 3}, {"Savannah Lions", core.White, 1, 2}, {"Grizzly Bears", core.Green, 2, 2}, {"Serra Angel", core.White, 5, 4}, {"Air Elemental", core.Blue, 5, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &ronNontramplePlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			card := repositoryCard(t, "LEB", tc.name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("unpaid cast changed state")
			}
			a.ManaPool().Add(tc.color, tc.cost)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil || a.ManaPool().TotalMana() != 0 {
				t.Fatal("cast timing or payment")
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			first := g.PutOnBattlefield(repositoryCard(t, "LEB", "Wall of Air"), b.PlayerID())
			second := g.PutOnBattlefield(repositoryCard(t, "LEB", "Wall of Air"), b.PlayerID())
			amulet := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Amulet of Kroog"), b.PlayerID())
			if err := g.ActivateAbilityByIndex(b.PlayerID(), amulet.ID(), 0, []uuid.UUID{second.ID()}); err == nil || amulet.Tapped || g.StackSize() != 0 {
				t.Fatal("unpaid activation changed state")
			}
			b.ManaPool().Add(core.Colorless, 2)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), amulet.ID(), 0, []uuid.UUID{second.ID()}); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(amulet.ID()).Tapped || b.ManaPool().TotalMana() != 0 || g.StackSize() != 1 {
				t.Fatal("shield payment or stack timing")
			}
			g.ResolveStack()
			source = g.FindPermanent(source.ID())
			first = g.FindPermanent(first.ID())
			second = g.FindPermanent(second.ID())
			information := g.CaptureDamageSource(source.ID())
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{source.ID()})
			if source.Tapped == (tc.name == "Serra Angel") {
				t.Fatal("vigilance or attack tapping")
			}
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: first.ID(), AttackerID: source.ID()}, {BlockerID: second.ID(), AttackerID: source.ID()}})
			clone := g.Clone()
			g.ExecuteCombatDamage()
			first = g.FindPermanent(first.ID())
			second = g.FindPermanent(second.ID())
			if first.Damage != 1 || second.Damage != tc.power-2 || a.Life() != 20 || b.Life() != 20 || a.orders != 0 || a.calls != 1 {
				t.Fatalf("damage=%d/%d calls=%d/%d", first.Damage, second.Damage, a.orders, a.calls)
			}
			if !g.HasDealtDamageToPermanentFromSource(information, first) {
				t.Fatal("source history lost")
			}
			if (tc.power > 2) != g.HasDealtDamageToPermanentFromSource(information, second) {
				t.Fatal("prevented damage history incorrect")
			}
			if clone.FindPermanent(first.ID()).Damage != 0 || clone.FindPermanent(second.ID()).Damage != 0 {
				t.Fatal("combat mutated clone")
			}
			clone.ExecuteCombatDamage()
			cloneFirst := clone.FindPermanent(first.ID())
			if tc.power >= 5 {
				if cloneFirst != nil {
					t.Fatal("clone fallback did not deal lethal damage")
				}
			} else if cloneFirst == nil || cloneFirst.Damage != tc.power {
				t.Fatal("clone fallback damage")
			}
			if clone.FindPermanent(second.ID()).Damage != max(tc.power-6, 0) || first.Damage != 1 || second.Damage != tc.power-2 {
				t.Fatal("clone combat leaked or lost prevention")
			}

		})
	}
}

func TestRONNontrampleAssignmentDeclarations(t *testing.T) {
	for _, rule := range []string{`Trample()`, `Activated(Trample,ActionList([GainLife(Integer(1))]))`, `SpellActions(ActionList([Trample]))`, `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([PermanentDealsDamage(ThisPermanent,Integer(1),Ref_TargetPlayer)])))`} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid declaration %s", rule)
		}
	}
}
