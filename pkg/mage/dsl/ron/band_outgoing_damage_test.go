package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronOutgoingBandPlayer struct{ *mage.BasePlayer }

func (p *ronOutgoingBandPlayer) ChooseNumber(_, _ int, _ string) int { return 0 }

func TestRONBandOutgoingDamage(t *testing.T) {
	for _, tc := range []struct {
		set, name   string
		color       core.Color
		cost, power int
	}{
		{"LEB", "Timber Wolves", core.Green, 1, 1},
		{"LEB", "Mesa Pegasus", core.White, 2, 1},
		{"ARN", "War Elephant", core.White, 4, 2},
		{"FEM", "Icatian Phalanx", core.White, 5, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := mage.NewBasePlayer("A")
			b := &ronOutgoingBandPlayer{BasePlayer: mage.NewBasePlayer("B")}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			cards := []mage.Card{repositoryCard(t, "LEB", "Benalish Hero"), repositoryCard(t, tc.set, tc.name)}
			for i, card := range cards {
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("unpaid card cast changed state")
				}
				color, cost := core.White, 1
				if i == 1 {
					color, cost = tc.color, tc.cost
				}
				a.ManaPool().Add(color, cost)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().TotalMana() != 0 || g.FindPermanent(card.ID()) != nil {
					t.Fatal("cost or casting timing")
				}
				g.ResolveStack()
				g.FindPermanent(card.ID()).RevokeBaseAttr(core.AttrSummonSick)
			}
			hero, member := g.FindPermanent(cards[0].ID()), g.FindPermanent(cards[1].ID())
			blocker := g.PutOnBattlefield(repositoryCard(t, "LEB", "War Mammoth"), b.PlayerID())
			amulet := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Amulet of Kroog"), b.PlayerID())
			if err := g.ActivateAbilityByIndex(b.PlayerID(), amulet.ID(), 0, []uuid.UUID{blocker.ID()}); err == nil || amulet.Tapped || g.StackSize() != 0 {
				t.Fatal("unpaid shield activation changed state")
			}
			b.ManaPool().Add(core.Colorless, 2)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), amulet.ID(), 0, []uuid.UUID{blocker.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{hero.ID(), member.ID()})
			g.GetCombat().AddBand([]uuid.UUID{hero.ID(), member.ID()})
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: hero.ID()}})
			first, second := g.CaptureDamageSource(hero.ID()), g.CaptureDamageSource(member.ID())
			clone := g.Clone()
			g.ExecuteCombatDamage()
			if g.FindPermanent(blocker.ID()).Damage != tc.power || b.Life() != 20 {
				t.Fatal("outgoing damage or trample used the band's combined power")
			}
			if !g.HasDealtDamageToPermanentFromSource(first, blocker) {
				t.Fatal("hero damage history lost")
			}
			if tc.power == 2 && !g.HasDealtDamageToPermanentFromSource(second, blocker) {
				t.Fatal("other source damage history lost")
			}
			if clone.FindPermanent(blocker.ID()).Damage != 0 {
				t.Fatal("combat changed pending clone")
			}
			clone.ExecuteCombatDamage()
			if clone.FindPermanent(blocker.ID()).Damage != tc.power {
				t.Fatal("clone lost band sources or RON shield")
			}
		})
	}
}

func TestRONBandOutgoingInvalidDeclarations(t *testing.T) {
	for _, rule := range []string{`Banding()`, `Banding(ThisPermanent)`, `Activated(Banding,ActionList([GainLife(Integer(1))]))`, `SpellActions(ActionList([Banding]))`} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid declaration: %s", rule)
		}
	}
}
