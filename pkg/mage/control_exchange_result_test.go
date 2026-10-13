package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestControlExchangeResult(t *testing.T) {
	for _, change := range []string{"none", "same", "blocked", "phase", "phase-cycle", "return", "identical"} {
		t.Run(change, func(t *testing.T) {
			g, a, b, first := controllerTestGame(t)
			card := NewArtifact("Second", "")
			card.SetOwner(b.PlayerID())
			second := g.PutOnBattlefield(card, b.PlayerID())
			left, right := BindPermanent("left"), BindPermanent("right")
			ctx := &EffectContext{Game: g, SourceID: first.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, left).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			ctx.SourceID = second.ID()
			if err := Snapshot(This, right).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			result := BindControlExchangeResult("exchange")
			producer := ExchangePermanentControlInto(From(left), From(right), result)
			if err := producer.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if err := IfControlExchanged(result, GainLifeFor(You, Fixed(1))).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 21 {
				t.Fatal("successful exchange did not save its result")
			}
			switch change {
			case "same":
				g.AddControlEffect(ControlEffectSpec{TargetID: second.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "blocked":
				g.MutablePermanent(first.ID()).GrantBaseAttr(core.AttrCantChangeControl)
			case "phase":
				g.PhaseOut(second.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(second.ID()))
			case "return":
				g.BouncePermanentToHand(second)
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "identical":
				producer = ExchangePermanentControlInto(From(left), From(left), result)
			}
			if err := producer.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if err := IfControlExchanged(result, GainLifeFor(You, Fixed(1))).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 21
			if change == "none" || change == "phase-cycle" {
				want++
			}
			if a.Life() != want {
				t.Fatal("failure reused an old successful outcome")
			}
		})
	}
}

func TestControlExchangeResultDeclaration(t *testing.T) {
	result := BindControlExchangeResult("exchange")
	producer := ExchangePermanentControlInto(This, This, result)
	for _, tc := range []struct {
		name  string
		body  ResolvingEffect
		valid bool
	}{
		{"valid", Sequence(producer, IfControlExchanged(result, GainLifeFor(You, Fixed(1)))), true},
		{"unbound", IfControlExchanged(result, GainLifeFor(You, Fixed(1))), false},
		{"nil-result", ExchangePermanentControlInto(This, This, nil), false},
		{"nil-read", IfControlExchanged(nil, GainLifeFor(You, Fixed(1))), false},
		{"nil-body", Sequence(producer, IfControlExchanged(result, nil)), false},
		{"nil-ref", ExchangePermanentControlInto(nil, This, result), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewArtifact("Test", "", WithAbility(Activate(Costs(), tc.body)))
			if (ValidateCardDefinition(c.Name(), c) == nil) != tc.valid {
				t.Fatal("wrong declaration validation")
			}
		})
	}
}
