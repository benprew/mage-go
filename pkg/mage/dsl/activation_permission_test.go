package dsl_test

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAnyPlayerMayActivatePaymentAndController(t *testing.T) {
	for _, permission := range []bool{false, true} {
		t.Run(map[bool]string{false: "controller only", true: "any player"}[permission], func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			var options []dsl.ActivationOption
			if permission {
				options = append(options, dsl.AnyPlayerMayActivate())
			}
			card := dsl.NewArtifact("Source", "", dsl.WithAbility(dsl.Activate(
				dsl.Costs(dsl.Mana("{3}")), dsl.GainLife(dsl.You, dsl.Fixed(1)), options...)))
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(card, a.PlayerID())
			a.ManaPool().Add(core.Green, 3)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), card.ID(), 0, nil); err == nil || a.ManaPool().TotalMana() != 3 || g.StackSize() != 0 {
				t.Fatal("the opponent cannot spend the source controller's mana")
			}
			b.ManaPool().Add(core.Blue, 6)
			err := g.ActivateAbilityByIndex(b.PlayerID(), card.ID(), 0, nil)
			if !permission {
				if err == nil || b.ManaPool().TotalMana() != 6 || g.StackSize() != 0 {
					t.Fatal("the default permission must reject the opponent before payment")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if b.ManaPool().TotalMana() != 3 || g.StackSize() != 1 || g.StackPeek().Controller != b.PlayerID() {
					t.Fatal("the activator must pay and control the stacked ability")
				}
				clone := g.Clone()
				clone.ResolveStack()
				if clone.GetPlayer(b.PlayerID()).Life() != 21 || b.Life() != 20 || a.Life() != 20 {
					t.Fatal("You must refer to the activator without changing the original game")
				}
				if err := g.ActivateAbilityByIndex(b.PlayerID(), card.ID(), 0, nil); err != nil {
					t.Fatal("permission must not impose an activation limit", err)
				}
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
				t.Fatal("the source controller must retain permission", err)
			}
			g.ResolveStack()
			want := 20
			if permission {
				want = 22
			}
			if a.Life() != 21 || b.Life() != want || a.ManaPool().TotalMana() != 0 {
				t.Fatal("resolution must use each ability's stored activator")
			}
		})
	}
}

func TestAnyPlayerMayActivateRespectsRestrictions(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	card := dsl.NewArtifact("Source", "", dsl.WithAbility(dsl.Activate(
		dsl.Costs(dsl.Mana("{1}")), dsl.GainLife(dsl.You, dsl.Fixed(1)),
		dsl.AnyPlayerMayActivate(), dsl.OnlyDuring(core.Upkeep))))
	g.PutOnBattlefield(card, a.PlayerID())
	b.ManaPool().Add(core.Blue, 1)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), card.ID(), 0, nil); err == nil || b.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
		t.Fatal("permission must not bypass the timing restriction")
	}
}
