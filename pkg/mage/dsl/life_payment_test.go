package dsl_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

type lifePaymentPlayer struct {
	*mage.BasePlayer
	accept bool
}

func (p *lifePaymentPlayer) ChooseMayAbility(string) bool { return p.accept }

func TestTypedLifePaymentBranchesAtBoundary(t *testing.T) {
	for _, accept := range []bool{false, true} {
		for _, total := range []int{3, 4} {
			for _, ifPaid := range []bool{false, true} {
				t.Run(fmt.Sprintf("accept=%v/total=%d/ifPaid=%v", accept, total, ifPaid), func(t *testing.T) {
					a := &lifePaymentPlayer{mage.NewBasePlayer("A"), accept}
					b := mage.NewBasePlayer("B")
					g := mage.NewGame(a, b)
					a.SetLife(3)
					player, amount := dsl.BindPlayer("payer"), dsl.BindNumber("amount")
					watcher := dsl.NewEnchantment("Payment watcher", "", dsl.WithAbility(dsl.WhenEvent(
						dsl.OpponentLosesLife(player).WithAmount(amount), dsl.GainLife(dsl.You, dsl.FromNumber(amount)),
					)))
					g.PutOnBattlefield(watcher, b.PlayerID())
					costs := dsl.Costs(dsl.LifePayCost(1), dsl.LifePayCost(total-1))
					body := dsl.GainLife(dsl.You, dsl.Fixed(5))
					var effect dsl.ResolvingEffect
					if ifPaid {
						effect = dsl.IfPaid(dsl.You, costs, body)
					} else {
						effect = dsl.UnlessPaid(dsl.You, costs, body)
					}
					if err := mage.ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					paid := accept && total == 3
					want := 3
					if paid {
						want = 0
					}
					if paid == ifPaid {
						want += 5
					}
					if a.Life() != want {
						t.Fatalf("life = %d, want %d", a.Life(), want)
					}
					g.PutTriggersOnStack()
					wantTriggers := 0
					if paid {
						wantTriggers = 2
					}
					if g.StackSize() != wantTriggers {
						t.Fatalf("triggers = %d, want %d", g.StackSize(), wantTriggers)
					}
					g.ResolveStack()
					wantGain := 20
					if paid {
						wantGain += 3
					}
					if b.Life() != wantGain {
						t.Fatalf("watcher life = %d, want %d", b.Life(), wantGain)
					}
				})
			}
		}
	}
}

func TestTypedLifePaymentCastAndActivationBoundary(t *testing.T) {
	for _, spell := range []bool{false, true} {
		for _, life := range []int{2, 3} {
			t.Run(fmt.Sprintf("spell=%v/life=%d", spell, life), func(t *testing.T) {
				a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				a.SetLife(life)
				land := dsl.NewLand("Payment land", dsl.WithManaAbility(core.Green))
				g.PutOnBattlefield(land, a.PlayerID())
				var err error
				if spell {
					card := dsl.NewSorcery("Life spell", "{G}", dsl.WithAbility(dsl.Spell(dsl.GainLife(dsl.You, dsl.Fixed(5)))), dsl.WithAdditionalCost(dsl.LifePayCost(3)))
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					err = g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0)
					if life == 2 && len(a.Hand()) != 1 {
						t.Fatal("failed cast removed the card")
					}
				} else {
					card := dsl.NewArtifact("Life device", "", dsl.WithAbility(dsl.Activate(
						dsl.Costs(dsl.Mana("{G}"), dsl.LifePayCost(1), dsl.LifePayCost(2)), dsl.GainLife(dsl.You, dsl.Fixed(5)),
					)))
					g.PutOnBattlefield(card, a.PlayerID())
					err = g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil)
				}
				if (err == nil) != (life == 3) {
					t.Fatalf("payment error = %v", err)
				}
				wantLife, wantStack := life, 0
				if life == 3 {
					wantLife, wantStack = 0, 1
				}
				if a.Life() != wantLife || g.StackSize() != wantStack {
					t.Fatalf("life/stack = %d/%d", a.Life(), g.StackSize())
				}
				if g.FindPermanent(land.ID()).Tapped != (life == 3) || a.ManaPool().TotalMana() != 0 {
					t.Fatal("incorrect mana payment or rollback")
				}
			})
		}
	}
}
