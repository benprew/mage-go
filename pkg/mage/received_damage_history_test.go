package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestReceivedDamageHistoryIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Recipient", "", 1, 8)
	p := g.PutOnBattlefield(card, a.PlayerID())
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 8), b.PlayerID())
	filter := WasDealtDamageThisTurn
	if filter.Match(nil, g) || filter.Match(p, nil) || filter.Match(p, g) {
		t.Fatal("absent history matched")
	}
	g.DealDamageToPermanentFromSource(p, 1, g.CaptureDamageSource(source.ID()))
	g.AddRegenerationShield(p.ID())
	g.DestroyPermanent(p)
	if p.Damage != 0 || !filter.Match(p, g) {
		t.Fatal("regeneration erased actual damage history")
	}
	phased := g.PhaseOut(p.ID())
	if filter.Match(p, g) {
		t.Fatal("phased recipient matched")
	}
	g.PhaseIn(phased)
	g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	if !filter.Match(g.FindPermanent(p.ID()), g) {
		t.Fatal("phase cycle or control change erased history")
	}
	g.RemoveFromBattlefield(p)
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if filter.Match(returned, g) || filter.Match(p, g) {
		t.Fatal("returned or departed visit matched")
	}
}

func TestReceivedDamageHistoryOutcomesAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 8), a.PlayerID())
	source := g.CaptureDamageSource(g.PutOnBattlefield(NewCreature("Source", "", 1, 8), b.PlayerID()).ID())
	g.AddPreventionShield(p.ID(), 2)
	g.DealDamageToPermanentFromSource(p, 2, source)
	g.DealDamageToPermanentFromSource(p, 0, source)
	g.DealDamageToPermanentFromSource(p, -1, source)
	phased := g.PhaseOut(p.ID())
	g.DealDamageToPermanentFromSource(p, 1, source)
	g.PhaseIn(phased)
	if WasDealtDamageThisTurn.Match(p, g) {
		t.Fatal("prevented, nonpositive or phased damage entered history")
	}
	clone := g.Clone()
	q := clone.FindPermanent(p.ID())
	clone.DealDamageToPermanentFromSource(q, 1, source)
	if !WasDealtDamageThisTurn.Match(q, clone) || WasDealtDamageThisTurn.Match(p, g) {
		t.Fatal("clone shared new history")
	}
	g.DealDamageToPermanentFromSource(p, 1, source)
	clone = g.Clone()
	g.damage.ClearEndOfTurn()
	if WasDealtDamageThisTurn.Match(p, g) || !WasDealtDamageThisTurn.Match(clone.FindPermanent(p.ID()), clone) {
		t.Fatal("cleanup shared history")
	}
}

func TestReceivedDamageHistorySpellAndBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("First", "", 1, 8), a.PlayerID())
	q := g.PutOnBattlefield(NewCreature("Second", "", 1, 8), a.PlayerID())
	seen := 0
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(EvtDamageDealt, false).SetCondition(func(_ *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		game := reader.(*Game)
		if !WasDealtDamageThisTurn.Match(game.FindPermanent(p.ID()), game) || !WasDealtDamageThisTurn.Match(game.FindPermanent(q.ID()), game) {
			t.Error("batch observation preceded recipient history")
		}
		seen++
		return false
	}))), a.PlayerID())
	spell := NewInstant("Damage spell", "", WithAbility(NewSpellAbility(&funcEffect{fn: func(game *Game, sourceID, _ uuid.UUID, _ []uuid.UUID) error {
		source := game.CaptureDamageSource(sourceID)
		if !source.Valid() || source.Incarnation() != uuid.Nil {
			t.Fatal("spell fixture has no valid spell source")
		}
		game.DealSimultaneousDamage(PermanentDamageAssignment(game.FindPermanent(p.ID()), 1, source), PermanentDamageAssignment(game.FindPermanent(q.ID()), 1, source))
		return nil
	}})))
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	if err := g.CastSpellByID(b.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if seen != 2 || !WasDealtDamageThisTurn.Match(p, g) || !WasDealtDamageThisTurn.Match(q, g) {
		t.Fatal("spell or batch damage was not recorded")
	}
}

func TestReceivedDamageHistoryCleanupPriority(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 8), a.PlayerID())
	g.DealDamageToPermanentFromSource(p, 1, g.CaptureDamageSource(p.ID()))
	g.RegisterDelayedTrigger(&DelayedTrigger{EventType: EvtCleanup, Controller: b.PlayerID(), Effects: []Effect{GainLifeFor(You, Fixed(1))}})
	if !g.doCleanupActions() || p.Damage != 0 || !WasDealtDamageThisTurn.Match(p, g) {
		t.Fatal("cleanup priority erased history with marked damage")
	}
	g.ResolveStack()
	g.RunStepWithPriority(Cleanup)
	if WasDealtDamageThisTurn.Match(p, g) {
		t.Fatal("final cleanup retained history")
	}
}

func TestReceivedDamageHistoryRedirection(t *testing.T) {
	for _, prevent := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewCreature("Source", "", 1, 8), a.PlayerID())
		first := g.PutOnBattlefield(NewCreature("First", "", 1, 8), b.PlayerID())
		second := g.PutOnBattlefield(NewCreature("Second", "", 1, 8), b.PlayerID())
		g.SetAttackerDamageRedirect(source.ID(), second.ID())
		if prevent {
			g.AddPreventionShield(second.ID(), 1)
		}
		g.DealDamageToPlayerFromSource(b, 1, g.CaptureDamageSource(source.ID()))
		if b.Life() != 20 || WasDealtDamageThisTurn.Match(first, g) || WasDealtDamageThisTurn.Match(second, g) == prevent {
			t.Fatal("history did not use the final positive damage recipient")
		}
	}
}
