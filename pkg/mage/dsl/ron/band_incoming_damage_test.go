package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronBandSourcePlayer struct {
	*mage.BasePlayer
	choices int
}

func (p *ronBandSourcePlayer) GetBandingDamageAssignment(_ *mage.Game, _ *mage.Permanent, recipients []*mage.Permanent, amount int) map[uuid.UUID]int {
	return map[uuid.UUID]int{recipients[0].ID(): amount}
}

func (p *ronBandSourcePlayer) ChooseNumber(_, _ int, _ string) int {
	p.choices++
	return 1
}

func TestRONBandIncomingDamage(t *testing.T) {
	for _, partner := range []struct {
		set, name   string
		color       core.Color
		cost, power int
	}{
		{"LEB", "Timber Wolves", core.Green, 1, 1},
		{"ARN", "War Elephant", core.White, 4, 2},
	} {
		t.Run(partner.name, func(t *testing.T) {
			a := &ronBandSourcePlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			cards := []mage.Card{repositoryCard(t, "LEB", "Benalish Hero"), repositoryCard(t, partner.set, partner.name)}
			for i, card := range cards {
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("unpaid cast changed state")
				}
				color, cost := core.White, 1
				if i == 1 {
					color, cost = partner.color, partner.cost
				}
				a.ManaPool().Add(color, cost)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(card.ID()) != nil || a.ManaPool().TotalMana() != 0 {
					t.Fatal("casting timing or mana payment")
				}
				g.ResolveStack()
				g.FindPermanent(card.ID()).RevokeBaseAttr(core.AttrSummonSick)
			}
			hero := g.FindPermanent(cards[0].ID())
			member := g.FindPermanent(cards[1].ID())
			if !hero.HasKeyword(core.Banding) || !member.HasKeyword(core.Banding) || member.CurrentPower(g) != partner.power {
				t.Fatal("RON band metadata")
			}
			lion := g.PutOnBattlefield(repositoryCard(t, "LEB", "Savannah Lions"), b.PlayerID())
			mammoth := g.PutOnBattlefield(repositoryCard(t, "LEB", "War Mammoth"), b.PlayerID())
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{hero.ID(), member.ID()})
			g.GetCombat().AddBand([]uuid.UUID{hero.ID(), member.ID()})
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: lion.ID(), AttackerID: hero.ID()}, {BlockerID: mammoth.ID(), AttackerID: hero.ID()}})
			first, second := g.CaptureDamageSource(lion.ID()), g.CaptureDamageSource(mammoth.ID())
			g.AddPreventionShield(hero.ID(), 3)
			clone := g.Clone()
			g.ExecuteCombatDamage()
			if a.choices != 1 || g.FindPermanent(hero.ID()) != nil || g.FindPermanent(member.ID()) == nil || b.Life() != 20 {
				t.Fatal("finite shield allocation, band membership or trample scope")
			}
			if !g.HasDealtDamageToPermanentFromSource(first, hero) || !g.HasDealtDamageToPermanentFromSource(second, hero) {
				t.Fatal("RON combat combined two blocker identities")
			}
			if clone.FindPermanent(hero.ID()) == nil || clone.HasDealtDamageToPermanentFromSource(first, clone.FindPermanent(hero.ID())) {
				t.Fatal("combat mutated pending clone")
			}
			clone.ExecuteCombatDamage()
			if clone.FindPermanent(hero.ID()) != nil || !clone.HasDealtDamageToPermanentFromSource(second, hero) {
				t.Fatal("cloned RON combat lost source packets")
			}
		})
	}
}
