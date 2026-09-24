package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestTypedPreventionSavedAmount(t *testing.T) {
	for _, power := range []int{-2, 0, 3} {
		t.Run(Fixed(power).Text(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Source", "", power, 4)
			g.PutOnBattlefield(source, a.PlayerID())
			saved := BindPermanent("amount source")
			body := Sequence(Snapshot(This, saved), PreventNextDamage(You, PowerOf(From(saved))))
			card := NewCreature("Shield", "", 2, 3, WithAbility(Activate(Costs(), body)))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			clone := g.Clone()
			clone.DealDamageToPlayer(clone.GetPlayer(a.PlayerID()), 5, uuid.Nil)
			want := 15 + max(power, 0)
			if got := clone.GetPlayer(a.PlayerID()).Life(); got != want {
				t.Fatalf("life = %d, want %d", got, want)
			}
			if a.Life() != 20 {
				t.Fatal("clone consumed original shield")
			}
			g.DealDamageToPlayer(a, 2, uuid.Nil)
			g.DealDamageToPlayer(a, 3, uuid.Nil)
			if a.Life() != want {
				t.Fatalf("life = %d, want %d", a.Life(), want)
			}
		})
	}
}

func TestTypedPreventionTargetAmount(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	subject := NewCreature("Amount", "", 3, 4)
	g.PutOnBattlefield(subject, a.PlayerID())
	recipient := DeclarePlayerTarget("recipient")
	amount := DeclarePermanentTarget("amount", IsCreature)
	ability := Spell(PreventNextDamage(recipient, PowerOf(amount)), WithTargets(recipient, amount))
	card := NewInstant("Shield", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.pushStack(&StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID(), subject.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	g.ResolveStack()
	g.RemoveFromBattlefield(g.FindPermanent(subject.ID()))
	g.DealDamageToPlayer(b, 5, uuid.Nil)
	if b.Life() != 18 || a.Life() != 20 {
		t.Fatalf("life = %d/%d", a.Life(), b.Life())
	}
}

func TestTypedPreventionAmountValidation(t *testing.T) {
	saved := BindPermanent("missing")
	target := DeclarePermanentTarget("undeclared", IsCreature)
	for _, amount := range []ResolutionValue{nil, PowerOf(nil), PowerOf(From(saved)), PowerOf(target), ValueAtResolution(nil)} {
		card := NewInstant("Invalid shield", "", WithAbility(Spell(PreventNextDamage(You, amount))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("accepted %T", amount)
		}
	}
}

func TestTypedPreventionUnusedCapacityExpires(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if err := ApplyEffect(g, PreventNextDamage(You, Fixed(5)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPlayer(a, 1, uuid.Nil)
	g.effects.RemoveEndOfTurn()
	g.effects.ClearReplacementsEndOfTurn()
	g.DealDamageToPlayer(a, 2, uuid.Nil)
	if a.Life() != 18 {
		t.Fatalf("life = %d, want 18", a.Life())
	}
}

func TestTypedPreventionSavedRecipientLifetime(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "phase cycle"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Recipient", "", 3, 6)
			g.PutOnBattlefield(c, a.PlayerID())
			saved := BindPermanent("recipient")
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
			}
			if err := PreventNextDamage(From(saved), PowerOf(From(saved))).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{c.ID()})
			}
			p := g.FindPermanent(c.ID())
			g.DealDamageToPermanent(p, 4, uuid.Nil)
			want := 1
			if change == "return" || change == "phase" {
				want = 4
			}
			if p.Damage != want {
				t.Fatalf("damage = %d, want %d", p.Damage, want)
			}
		})
	}
}
