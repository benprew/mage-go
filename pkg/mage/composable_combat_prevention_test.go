package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombatPreventionIncarnation(t *testing.T) {
	for _, change := range []string{"none", "return", "phase cycle", "control", "expire"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Creature", "", 3, 3)
			g.PutOnBattlefield(card, a.PlayerID())
			g.PreventCombatDamageToAndBy(card.ID())
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(card.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "expire":
				g.effects.ClearReplacementsEndOfTurn()
			}
			clone := g.Clone()
			clone.damage.SetResolvingCombatDamage(true)
			clone.DealDamageToPlayer(clone.GetPlayer(b.PlayerID()), 2, card.ID())
			clone.DealDamageToPermanent(clone.FindPermanent(card.ID()), 1, uuid.Nil)
			prevented := change == "none" || change == "phase cycle" || change == "control"
			wantLife, wantDamage := 18, 1
			if prevented {
				wantLife, wantDamage = 20, 0
			}
			if clone.GetPlayer(b.PlayerID()).Life() != wantLife || clone.FindPermanent(card.ID()).Damage != wantDamage {
				t.Fatal("prevention did not preserve lifetime")
			}
			if b.Life() != 20 || g.FindPermanent(card.ID()).Damage != 0 {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestTypedCombatPreventionSubject(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 3, 3)
	other := NewCreature("Other", "", 3, 3)
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	body := PreventCombatDamageToAndBy(This)
	if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), []uuid.UUID{other.ID()}); err != nil {
		t.Fatal(err)
	}
	g.damage.SetResolvingCombatDamage(true)
	g.DealDamageToPlayer(b, 2, source.ID())
	g.DealDamageToPermanent(g.FindPermanent(source.ID()), 2, other.ID())
	g.DealDamageToPlayer(a, 2, other.ID())
	if b.Life() != 20 || g.FindPermanent(source.ID()).Damage != 0 || a.Life() != 18 {
		t.Fatal("wrong prevention subject")
	}
}

func TestTypedControlledTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	own := NewCreature("Own", "", 2, 2)
	other := NewCreature("Other", "", 2, 2)
	g.PutOnBattlefield(own, a.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	target := DeclarePermanentTarget("creature", IsCreature, TargetYouControl())
	ability := Spell(Sequence(UntapPermanent(target), PreventCombatDamageToAndBy(target)), WithTargets(target))
	card := NewInstant("Spell", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	possible := ability.Targets()[0].Possible(a.PlayerID(), card, g)
	if len(possible) != 1 || possible[0] != own.ID() {
		t.Fatal("controlled target included opponent")
	}
	g.TapPermanent(g.FindPermanent(own.ID()))
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{own.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	g.AddControlEffect(ControlEffectSpec{SourceID: other.ID(), TargetID: own.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	g.ResolveStack()
	if !g.FindPermanent(own.ID()).Tapped || len(g.effects.replacements) != 0 {
		t.Fatal("target stayed legal after control change")
	}
}
