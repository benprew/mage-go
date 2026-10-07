package mage

import (
	"fmt"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type resolutionCounterPayer struct {
	*BasePlayer
	accept                bool
	choice                int
	mayCalls, numberCalls int
	onMay                 func()
	onNumber              func()
}

func (p *resolutionCounterPayer) ChooseMayAbility(string) bool {
	p.mayCalls++
	if p.onMay != nil {
		p.onMay()
	}
	return p.accept
}
func (p *resolutionCounterPayer) ChooseNumber(int, int, string) int {
	p.numberCalls++
	if p.onNumber != nil {
		p.onNumber()
	}
	return p.choice
}

func TestResolutionCounterPayment(t *testing.T) {
	for _, state := range []string{"fixed", "chosen", "zero", "declined", "short", "mandatory", "return", "phase", "phase-cycle", "control"} {
		t.Run(state, func(t *testing.T) {
			a := &resolutionCounterPayer{BasePlayer: NewBasePlayer("A"), accept: state != "declined", choice: 2}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			result := BindNumber("removed")
			paid := GainLifeFor(You, Fixed(4))
			var effect ResolvingEffect
			if state == "chosen" || state == "zero" {
				if state == "zero" {
					a.choice = 0
				}
				effect = PayChosenCountersOrElse(You, This, core.Charge, result, GainLifeFor(You, FromNumber(result)), GainLifeFor(You, Fixed(7)))
			} else {
				effect = PayCountersOrElse(You, This, core.Charge, 2, state != "mandatory", paid, GainLifeFor(You, Fixed(7)))
			}
			card := NewArtifact("Source", "", WithAbility(Activate(Costs(), effect)))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.MutablePermanent(p.ID()).AddCounter(core.Charge, 3)
			if state == "short" {
				g.MutablePermanent(p.ID()).RemoveCounter(core.Charge, 2)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				p = g.PutOnBattlefield(card, a.PlayerID())
				g.MutablePermanent(p.ID()).AddCounter(core.Charge, 3)
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			clone := g.Clone()
			g.ResolveStack()
			wantLife, wantCounters := 24, 1
			switch state {
			case "chosen":
				wantLife = 22
			case "zero":
				wantLife, wantCounters = 20, 3
			case "declined", "return", "phase":
				wantLife, wantCounters = 27, 3
			case "short":
				wantLife, wantCounters = 27, 1
			}
			if a.Life() != wantLife || int(g.FindPermanentIncludingPhased(p.ID()).Counters[core.Charge]) != wantCounters || b.Life() != 20 {
				t.Fatalf("life %d counters %d", a.Life(), g.FindPermanentIncludingPhased(p.ID()).Counters[core.Charge])
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone shared resolution")
			}
			if state == "mandatory" && a.mayCalls != 0 {
				t.Fatal("mandatory payment was optional")
			}
		})
	}
}

func TestResolutionCounterPaymentDeclarations(t *testing.T) {
	result := BindNumber("removed")
	for _, effect := range []ResolvingEffect{
		PayCountersOrElse(You, nil, core.Charge, 1, true, GainLifeFor(You, Fixed(1)), GainLifeFor(You, Fixed(2))),
		PayCountersOrElse(You, This, core.Charge, -1, true, GainLifeFor(You, Fixed(1)), GainLifeFor(You, Fixed(2))),
		PayCountersOrElse(You, This, core.NumCounters, 1, true, GainLifeFor(You, Fixed(1)), GainLifeFor(You, Fixed(2))),
		PayChosenCountersOrElse(You, This, core.Charge, nil, GainLifeFor(You, Fixed(1)), GainLifeFor(You, Fixed(2))),
		PayChosenCountersOrElse(You, This, core.Charge, result, GainLifeFor(You, FromNumber(result)), GainLifeFor(You, FromNumber(result))),
		Sequence(PayChosenCountersOrElse(You, This, core.Charge, result, GainLifeFor(You, FromNumber(result)), GainLifeFor(You, Fixed(2))), GainLifeFor(You, FromNumber(result))),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("accepted invalid declaration")
		}
	}
}

func TestResolutionCounterPaymentStaleChoice(t *testing.T) {
	for _, chosen := range []bool{false, true} {
		for _, change := range []string{"return", "phase", "short", "invalid-count"} {
			t.Run(fmt.Sprintf("%t/%s", chosen, change), func(t *testing.T) {
				a := &resolutionCounterPayer{BasePlayer: NewBasePlayer("A"), accept: true, choice: 2}
				g := NewGame(a, NewBasePlayer("B"))
				card := NewArtifact("Source", "")
				p := g.PutOnBattlefield(card, a.PlayerID())
				g.MutablePermanent(p.ID()).AddCounter(core.Charge, 3)
				mutate := func() {
					switch change {
					case "return":
						g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
						p = g.PutOnBattlefield(card, a.PlayerID())
						g.MutablePermanent(p.ID()).AddCounter(core.Charge, 3)
					case "phase":
						g.PhaseOut(p.ID())
					case "short":
						g.MutablePermanent(p.ID()).RemoveCounter(core.Charge, 3)
					case "invalid-count":
						a.choice = 4
					}
				}
				if chosen {
					a.onNumber = mutate
				} else {
					a.onMay = mutate
				}
				var effect = PayCountersOrElse(You, This, core.Charge, 2, true, GainLifeFor(You, Fixed(2)), GainLifeFor(You, Fixed(7)))
				if chosen {
					effect = PayChosenCountersOrElse(You, This, core.Charge, BindNumber("paid"), GainLifeFor(You, Fixed(2)), GainLifeFor(You, Fixed(7)))
				}
				if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want := 27
				if !chosen && change == "invalid-count" {
					want = 22
				}
				if a.Life() != want {
					t.Fatalf("life %d want %d", a.Life(), want)
				}
				wantCounters := 3
				if change == "short" {
					wantCounters = 0
				}
				if !chosen && change == "invalid-count" {
					wantCounters = 1
				}
				if int(g.FindPermanentIncludingPhased(p.ID()).Counters[core.Charge]) != wantCounters {
					t.Fatal("stale payment spent counters")
				}
			})
		}
	}
}

func TestResolutionCounterPaymentNestedResults(t *testing.T) {
	a := &resolutionCounterPayer{BasePlayer: NewBasePlayer("A"), accept: true, choice: 2}
	g := NewGame(a, NewBasePlayer("B"))
	outer, inner := BindNumber("outer"), BindNumber("inner")
	body := Sequence(PayChosenCountersOrElse(You, This, core.Charge, inner, GainLifeFor(You, FromNumber(inner)), GainLifeFor(You, Fixed(9))), GainLifeFor(You, FromNumber(outer)))
	effect := PayChosenCountersOrElse(You, This, core.Charge, outer, body, GainLifeFor(You, Fixed(8)))
	card := NewArtifact("Nested", "", WithAbility(Activate(Costs(), effect)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.MutablePermanent(p.ID()).AddCounter(core.Charge, 5)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 24 || g.FindPermanent(p.ID()).Counters[core.Charge] != 1 || a.numberCalls != 2 {
		t.Fatal("nested result or payment changed")
	}
	if _, ok := g.resolution.numberBindings[outer]; ok {
		t.Fatal("outer result escaped")
	}
	if _, ok := g.resolution.numberBindings[inner]; ok {
		t.Fatal("inner result escaped")
	}
}
