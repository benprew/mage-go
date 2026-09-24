package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposablePermanentValueCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Subject", "", 2, 3)
	g.PutOnBattlefield(c, a.PlayerID())
	saved := BindPermanent("subject")
	effect := Sequence(Snapshot(This, saved), Until(EndOfTurn, ModifyPT(From(saved), PowerOf(From(saved)), ToughnessOf(From(saved)))))
	if err := ApplyEffect(g, effect, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
	}
	p := g.FindPermanent(c.ID())
	if p.CurrentPower(g) != 4 || p.CurrentToughness(g) != 6 {
		t.Fatal("values were not captured once")
	}
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(c.ID()))
	clone.PutOnBattlefield(c, a.PlayerID())
	clone.ApplyContinuousEffects()
	if clone.FindPermanent(c.ID()).CurrentPower(clone) != 2 || p.CurrentPower(g) != 4 {
		t.Fatal("capture followed a returned card or changed original")
	}
}

func TestComposablePermanentValueValidation(t *testing.T) {
	saved := BindPermanent("missing")
	effect := Until(EndOfTurn, ModifyPT(This, PowerOf(From(saved)), Fixed(0)))
	c := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), effect)))
	if err := ValidateCardDefinition(c.Name(), c); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("error = %v", err)
	}
}

func TestComposablePermanentValueLifetime(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "phase cycle", "absent", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Subject", "", -2, 3)
			g.PutOnBattlefield(c, a.PlayerID())
			saved := BindPermanent("subject")
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
				g.PutOnBattlefield(c, a.PlayerID())
			case "phase":
				g.PhaseOut(c.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(c.ID()))
			case "absent":
				g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: c.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
				g.ApplyContinuousEffects()
			}
			wantPower, wantToughness := -2, 3
			if change == "return" || change == "phase" || change == "absent" {
				wantPower, wantToughness = 0, 0
			}
			if got := PowerOf(From(saved)).resolveValue(ctx); got != wantPower {
				t.Fatalf("power = %d, want %d", got, wantPower)
			}
			if got := ToughnessOf(From(saved)).resolveValue(ctx); got != wantToughness {
				t.Fatalf("toughness = %d, want %d", got, wantToughness)
			}
		})
	}
}

func TestComposablePermanentValueTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := NewCreature("First", "", 2, 3), NewCreature("Second", "", 4, 5)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	x, y := DeclarePermanentTarget("first", IsCreature), DeclarePermanentTarget("second", IsCreature)
	ability := Spell(Until(EndOfTurn, ModifyPT(x, PowerOf(y), ToughnessOf(y))), WithTargets(x, y))
	card := NewInstant("Read second", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID(), second.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	clone := g.Clone()
	clone.ResolveStack()
	p := clone.FindPermanent(first.ID())
	if p.CurrentPower(clone) != 6 || p.CurrentToughness(clone) != 8 {
		t.Fatal("value did not read the second target")
	}
	if g.FindPermanent(first.ID()).CurrentPower(g) != 2 {
		t.Fatal("clone changed original")
	}
}

func TestComposablePermanentValueInvalidDefinitions(t *testing.T) {
	target := DeclarePermanentTarget("undeclared", IsCreature)
	for _, value := range []ResolutionValue{PowerOf(nil), ToughnessOf(nil), PowerOf(target), ValueAtResolution(nil)} {
		c := NewCreature("Invalid value", "", 1, 1, WithAbility(Activate(Costs(), Until(EndOfTurn, ModifyPT(This, value, Fixed(0))))))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatalf("accepted invalid value %T", value)
		}
	}
}

func TestComposablePermanentValueStaticEvaluation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 2, 3, WithAbility(Static(While(Controls(IsCreature), ModifyPT(This, PowerOf(This), Fixed(0))))))
	g.PutOnBattlefield(c, a.PlayerID())
	for range 3 {
		g.ApplyContinuousEffects()
	}
	if got := g.FindPermanent(c.ID()).CurrentPower(g); got != 4 {
		t.Fatalf("power = %d, want 4", got)
	}
	g.FindPermanent(c.ID()).Counters[P1P1] = 1
	g.ApplyContinuousEffects()
	if got := g.FindPermanent(c.ID()).CurrentPower(g); got != 6 {
		t.Fatalf("power = %d, want 6", got)
	}
}
