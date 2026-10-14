package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestRepeatedManaPayment(t *testing.T) {
	for _, scenario := range []string{"paid", "declined", "short", "wrong-color", "mana-source", "zero", "zero-decline", "negative", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: scenario != "declined" && scenario != "zero-decline"}
			g := NewGame(a, NewBasePlayer("B"))
			count := 2
			if scenario == "zero" || scenario == "zero-decline" {
				count = 0
			}
			if scenario == "negative" {
				count = -2
			}
			a.ManaPool().Add(Green, 2)
			a.ManaPool().Add(Colorless, 4)
			if scenario == "short" {
				a.ManaPool().Clear()
				a.ManaPool().Add(Green, 1)
			}
			if scenario == "wrong-color" {
				a.ManaPool().Clear()
				a.ManaPool().Add(Blue, 6)
			}
			var lands []*Permanent
			if scenario == "mana-source" {
				a.ManaPool().Clear()
				a.ManaPool().Add(Colorless, 4)
				for range 2 {
					lands = append(lands, g.PutOnBattlefield(NewLand("Mana", WithManaAbility(Green)), a.PlayerID()))
				}
			}
			before := a.ManaPool().TotalMana()
			effect := PayRepeatedManaOrElse(You, ParseManaCost("{2}{G}"), Fixed(count), GainLifeFor(You, Fixed(3)), GainLifeFor(You, Fixed(1)))
			if scenario == "clone" {
				clone := g.Clone()
				clone.players[0] = &directOptionalAcceptPlayer{clone.GetPlayer(a.PlayerID())}
				if err := ApplyEffect(clone, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if a.Life() != 20 || a.ManaPool().TotalMana() != before {
					t.Fatal("clone changed live payment")
				}
			}
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			paid := scenario != "declined" && scenario != "short" && scenario != "wrong-color" && scenario != "zero-decline"
			wantLife, wantMana := 21, before
			if paid {
				wantLife = 23
				if count > 0 {
					wantMana = 0
				}
			}
			if a.Life() != wantLife || a.ManaPool().TotalMana() != wantMana {
				t.Fatalf("life/mana = %d/%d, want %d/%d", a.Life(), a.ManaPool().TotalMana(), wantLife, wantMana)
			}
			for _, land := range lands {
				if !g.FindPermanent(land.ID()).Tapped {
					t.Fatal("mana ability did not pay")
				}
			}
		})
	}
}

func TestRepeatedManaPaymentCaptureAndRollback(t *testing.T) {
	for _, fail := range []bool{false, true} {
		a := &resolutionManaChoicePlayer{BasePlayer: NewBasePlayer("A")}
		g := NewGame(a, NewBasePlayer("B"))
		a.AddToHand(NewLand("First"))
		a.AddToHand(NewLand("Second"))
		a.ManaPool().Add(Green, 2)
		a.choose = func() {
			a.AddToHand(NewLand("Later"))
			if fail {
				a.ManaPool().Clear()
				a.ManaPool().Add(Green, 1)
			}
		}
		effect := PayRepeatedManaOrElse(You, ParseManaCost("{G}"), HandSizeOf(You), GainLifeFor(You, Fixed(3)), GainLifeFor(You, Fixed(1)))
		if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		wantLife, wantMana := 23, 0
		if fail {
			wantLife, wantMana = 21, 1
		}
		if a.Life() != wantLife || a.ManaPool().TotalMana() != wantMana || a.offers != 1 {
			t.Fatal("count changed during offer or failed payment spent mana")
		}
	}
}

func TestRepeatedManaPaymentDeclarations(t *testing.T) {
	body := GainLifeFor(You, Fixed(1))
	for _, effect := range []ResolvingEffect{
		PayRepeatedManaOrElse(nil, ParseManaCost("{G}"), Fixed(1), body, body),
		PayRepeatedManaOrElse(You, ParseManaCost("{G}"), nil, body, body),
		PayRepeatedManaOrElse(You, ParseManaCost("{G}"), Fixed(1), nil, body),
		PayRepeatedManaOrElse(You, ParseManaCost("{G}"), Fixed(1), body, nil),
		PayRepeatedManaOrElse(You, ParseManaCost("{X}"), Fixed(1), body, body),
		PayRepeatedManaOrElse(You, ManaCost{Green: -1}, Fixed(1), body, body),
	} {
		c := NewInstant("Invalid", "", Spell(effect))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid repeated mana declaration accepted")
		}
	}
	valid := NewInstant("Valid", "", Spell(PayRepeatedManaOrElse(You, ParseManaCost("{G}"), Fixed(1), body, body)))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
}

func TestRepeatedManaPaymentLimitsAndReduction(t *testing.T) {
	a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	g.effects.Rules.ActivationCostReductions[p.ID()] = 9
	a.ManaPool().Add(Colorless, 3)
	if err := ApplyEffect(g, UnlessRepeatedManaPaid(You, ParseManaCost("{2}"), Fixed(2), GainLifeFor(You, Fixed(1))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.offers != 0 || a.Life() != 21 || a.ManaPool().TotalMana() != 3 {
		t.Fatal("activation reduction changed resolving payment")
	}
	maximum := int(^uint(0) >> 1)
	for _, cost := range []ManaCost{{Generic: 2}, {Generic: 1, Green: 1}} {
		if err := ApplyEffect(g, IfRepeatedManaPaid(You, cost, Fixed(maximum), GainLifeFor(You, Fixed(1))), p.ID(), a.PlayerID(), nil); err == nil {
			t.Fatal("overflow accepted")
		}
	}
	if a.offers != 0 || a.Life() != 21 || a.ManaPool().TotalMana() != 3 {
		t.Fatal("overflow changed state")
	}
}

func TestRepeatedManaPaymentAllColorsAndAbsentPlayer(t *testing.T) {
	a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	for _, color := range []Color{White, Blue, Black, Red, Green, Colorless} {
		a.ManaPool().Add(color, 2)
	}
	effect := IfRepeatedManaPaid(You, ParseManaCost("{1}{W}{U}{B}{R}{G}"), Fixed(2), GainLifeFor(You, Fixed(1)))
	if err := ApplyEffect(g, effect, uuid.Nil, uuid.Nil, nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || a.ManaPool().TotalMana() != 12 {
		t.Fatal("absent payer changed state")
	}
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 21 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("complete colored cost did not repeat")
	}
}

func TestRepeatedManaPaymentWaitingDeclarations(t *testing.T) {
	plain := GainLifeFor(You, Fixed(1))
	source := GainLifeFor(You, CountersOnWithLKI(This, Charge))
	for _, tc := range []struct {
		count        ResolutionValue
		paid, unpaid ResolvingEffect
		want         bool
	}{
		{Fixed(0), plain, plain, false},
		{Fixed(0), source, plain, true},
		{Fixed(0), plain, source, true},
		{CountersOnWithLKI(This, Charge), plain, plain, true},
	} {
		effect := PayRepeatedManaOrElse(You, ParseManaCost("{G}"), tc.count, tc.paid, tc.unpaid).(*resolutionManaPayment)
		if effect.refersToSourcePermanent() != tc.want {
			t.Fatal("waiting declaration omitted a branch or read values")
		}
	}
}
