package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatureDiesDamagedByThisTurnIdentity(t *testing.T) {
	for _, tc := range []struct {
		name          string
		damage        bool
		returnVictim  bool
		returnSource  bool
		phaseSource   bool
		phaseVictim   bool
		preventDamage bool
		want          int
	}{
		{name: "dealt damage", damage: true, want: 1},
		{name: "no damage"},
		{name: "returned victim", damage: true, returnVictim: true},
		{name: "returned source", damage: true, returnSource: true},
		{name: "phased source", damage: true, phaseSource: true},
		{name: "phased victim", damage: true, phaseVictim: true, want: 1},
		{name: "prevented", damage: true, preventDamage: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			sourceCard := NewCreature("Source", "", 1, 5, WithAbility(WhenEvent(
				CreatureDies().DamagedByThisTurn(), GainLifeFor(You, Fixed(1)))))
			sourceCard.SetOwner(a.PlayerID())
			victimCard := NewCreature("Victim", "", 1, 5)
			victimCard.SetOwner(b.PlayerID())
			if err := ValidateCardDefinition(sourceCard.Name(), sourceCard); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			if tc.preventDamage {
				g.AddPreventionShield(victim.ID(), 1)
			}
			if tc.damage {
				g.DealDamageToPermanentFromSource(victim, 1, g.CaptureDamageSource(source.ID()))
			}
			if tc.returnVictim {
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(victimCard, b.PlayerID())
			}
			if tc.returnSource {
				g.RemoveFromBattlefield(source)
				source = g.PutOnBattlefield(sourceCard, a.PlayerID())
			}
			if tc.phaseSource {
				g.PhaseOut(source.ID())
			}
			if tc.phaseVictim {
				g.PhaseIn(g.PhaseOut(victim.ID()))
			}
			g.DestroyPermanent(victim)
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.want {
				t.Fatalf("stack=%d, want %d", got, tc.want)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("clone changed original life")
			}
			g.ResolveStack()
			if a.Life() != 20+tc.want {
				t.Fatalf("life=%d, want %d", a.Life(), 20+tc.want)
			}
		})
	}
}

func TestCreatureDiesDamagedByThisTurnSimultaneous(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	sourceCard := NewCreature("Source", "", 1, 5, WithAbility(WhenEvent(
		CreatureDies().DamagedByThisTurn(), GainLifeFor(You, Fixed(1)))))
	sourceCard.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	victimCard := NewCreature("Victim", "", 1, 5)
	victimCard.SetOwner(b.PlayerID())
	victim := g.PutOnBattlefield(victimCard, b.PlayerID())
	g.DealDamageToPermanentFromSource(victim, 1, g.CaptureDamageSource(source.ID()))
	g.DestroyPermanents([]*Permanent{source, victim})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack=%d, want 1", g.StackSize())
	}
}

func TestCreatureDiesDamagedByThisTurnRejectsInvalidPattern(t *testing.T) {
	for _, pattern := range []creatureDiesPattern{
		PermanentDies().DamagedByThisTurn(),
		CreatureDies().DamagedByThisTurn().DamagedByThisTurn(),
	} {
		card := NewArtifact("Invalid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid damage-history pattern was accepted")
		}
	}
}

func TestCreatureDiesDamagedByThisTurnRequiresEventIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 5), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 5), b.PlayerID())
	g.DealDamageToPermanentFromSource(victim, 1, g.CaptureDamageSource(source.ID()))
	cond := eventCreatureDamagedBySourceThisTurn{}
	if cond.CheckTriggerCond(&core.GameEvent{SourceID: victim.ID()}, g, source.ID(), a.PlayerID()) {
		t.Fatal("an event without incarnation matched damage history")
	}
}
