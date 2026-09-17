package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestComposablePaymentAtomicity(t *testing.T) {
	for _, accept := range []bool{false, true} {
		for _, total := range []int{3, 25} {
			t.Run(map[bool]string{false: "decline", true: "accept"}[accept]+Fixed(total).Text(), func(t *testing.T) {
				base := NewBasePlayer("A")
				var a Player = base
				if accept {
					a = &acceptingMayPlayer{base}
				}
				b := NewBasePlayer("B")
				g := NewGame(a, b)
				effect := UnlessPaid(You, Costs(LifePayCost(1), LifePayCost(total-1)), GainLifeFor(ActivePlayer, Fixed(4)))
				if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want := 24
				if accept && total == 3 {
					want = 17
				}
				if got := g.GetPlayer(a.PlayerID()).Life(); got != want {
					t.Fatalf("life = %d, want %d", got, want)
				}
			})
		}
	}
}

func TestComposablePaymentSavedSubject(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Land")
	land.SetOwner(b.PlayerID())
	g.PutOnBattlefield(land, b.PlayerID())
	saved := BindPermanent("land")
	effect := Sequence(Snapshot(This, saved), UnlessPaid(ControllerOf(From(saved)), Costs(ManaPayment("{1}")), Destroy(From(saved))))
	if err := ApplyEffect(g, effect, land.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(land.ID()) != nil {
		t.Fatal("unpaid branch lost its saved subject")
	}
	if err := ValidateCardDefinition("Payment", NewCreature("Payment", "", 1, 1, WithAbility(Activate(Costs(), effect)))); err != nil {
		t.Fatal(err)
	}
	bad := UnlessPaid(You, Costs(), Snapshot(This, saved))
	if err := ValidateCardDefinition("Scope", NewCreature("Scope", "", 1, 1, WithAbility(Activate(Costs(), Sequence(bad, Destroy(From(saved))))))); err == nil {
		t.Fatal("payment branch exported an assignment")
	}
}

func TestComposableIfPaid(t *testing.T) {
	for _, accept := range []bool{false, true} {
		base := NewBasePlayer("A")
		var player Player = base
		if accept {
			player = &acceptingMayPlayer{base}
		}
		g := NewGame(player, NewBasePlayer("B"))
		effect := IfPaid(You, Costs(LifePayCost(3)), GainLifeFor(You, Fixed(4)))
		if err := ApplyEffect(g, effect, uuid.Nil, player.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := 20
		if accept {
			want = 21
		}
		if got := g.GetPlayer(player.PlayerID()).Life(); got != want {
			t.Fatalf("life = %d, want %d", got, want)
		}
	}
}
