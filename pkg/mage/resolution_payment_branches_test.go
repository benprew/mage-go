package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestResolutionPaymentBranches(t *testing.T) {
	for _, state := range []string{"paid", "declined", "short", "zero", "nested"} {
		t.Run(state, func(t *testing.T) {
			a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: state != "declined"}
			g := NewGame(a, NewBasePlayer("B"))
			cost := Costs(LifePayCost(2))
			if state == "short" {
				a.SetLife(1)
			}
			if state == "zero" {
				cost = Costs()
			}
			paid := GainLifeFor(You, Fixed(3))
			if state == "nested" {
				paid = PayOrElse(You, Costs(LifePayCost(50)), GainLifeFor(You, Fixed(100)), GainLifeFor(You, Fixed(3)))
			}
			effect := PayOrElse(You, cost, paid, GainLifeFor(You, Fixed(7)))
			card := NewInstant("Branches", "", WithAbility(Spell(effect)))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			want := 21
			switch state {
			case "declined":
				want = 27
			case "short":
				want = 8
			case "zero":
				want = 23
			}
			if g.GetPlayer(a.PlayerID()).Life() != want {
				t.Fatalf("life = %d, want %d", g.GetPlayer(a.PlayerID()).Life(), want)
			}
		})
	}
}

func TestResolutionPaymentBranchesDeclaration(t *testing.T) {
	saved := BindPermanent("branch-local")
	for _, effect := range []ResolvingEffect{
		PayOrElse(nil, Costs(), GainLifeFor(You, Fixed(1)), GainLifeFor(You, Fixed(2))),
		PayOrElse(You, CostGroup{}, GainLifeFor(You, Fixed(1)), GainLifeFor(You, Fixed(2))),
		PayOrElse(You, Costs(), nil, GainLifeFor(You, Fixed(2))),
		PayOrElse(You, Costs(), GainLifeFor(You, Fixed(1)), nil),
		Sequence(PayOrElse(You, Costs(), Snapshot(This, saved), GainLifeFor(You, Fixed(1))), TapSubject(From(saved))),
		PayOrElse(You, Costs(), Snapshot(This, saved), TapSubject(From(saved))),
	} {
		card := NewInstant("Invalid", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid declaration")
		}
	}
}

func TestResolutionPaymentBranchesRollbackAndClone(t *testing.T) {
	a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	effect := PayOrElse(You, Costs(LifePayCost(15), LifePayCost(15)), GainLifeFor(You, Fixed(100)), GainLifeFor(You, Fixed(2)))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.GetPlayer(a.PlayerID()).Life() != 22 {
		t.Fatal("failed total payment spent life or ran paid branch")
	}
	before := g.GetPlayer(a.PlayerID()).Life()
	if err := ApplyEffect(g, effect, uuid.Nil, uuid.New(), nil); err != nil {
		t.Fatal(err)
	}
	if g.GetPlayer(a.PlayerID()).Life() != before {
		t.Fatal("absent payer ran a branch")
	}
	clone := g.Clone()
	if err := ApplyEffect(clone, PayOrElse(You, Costs(), GainLifeFor(You, Fixed(3)), GainLifeFor(You, Fixed(7))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.GetPlayer(a.PlayerID()).Life() != before || clone.GetPlayer(a.PlayerID()).Life() == before {
		t.Fatal("clone payment leaked")
	}
}
