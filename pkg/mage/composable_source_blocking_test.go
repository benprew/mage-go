package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceBlockingConditionState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Blocker", "", 2, 2)
	attacker := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(card, a.PlayerID())
	g.PutOnBattlefield(attacker, b.PlayerID())
	incarnation := g.FindPermanent(card.ID()).incarnationID
	ctx := StateContext{SourceID: card.ID(), ControllerID: a.PlayerID(), SourceIncarnation: incarnation}
	condition := SourceBlockingCondition()
	if condition.checkState(g, ctx) {
		t.Fatal("source matched before blocking")
	}
	g.combat.AddAttacker(attacker.ID(), a.PlayerID())
	g.combat.AddBlockerIncarnation(card.ID(), incarnation, attacker.ID())
	if !condition.checkState(g, ctx) || !condition.checkState(g.Clone(), ctx) {
		t.Fatal("blocking source was not observed in game and clone")
	}
	clone := g.Clone()
	clone.combat.RemoveFromCombat(card.ID())
	if condition.checkState(clone, ctx) || !condition.checkState(g, ctx) {
		t.Fatal("clone combat state was shared")
	}
	g.FindPermanent(card.ID()).PhasedOut = true
	if condition.checkState(g, ctx) {
		t.Fatal("phased-out source matched")
	}
	g.FindPermanentIncludingPhased(card.ID()).PhasedOut = false
	if !condition.checkState(g, ctx) {
		t.Fatal("phased-in blocker lost identity")
	}
	phased := g.PhaseOut(card.ID())
	g.PhaseIn(phased)
	if condition.checkState(g, ctx) {
		t.Fatal("phasing kept the source in combat")
	}
	g.combat.AddBlockerIncarnation(card.ID(), incarnation, attacker.ID())
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	if condition.checkState(g, ctx) {
		t.Fatal("absent source matched")
	}
	g.PutOnBattlefield(card, a.PlayerID())
	if condition.checkState(g, ctx) || condition.checkState(g, StateContext{SourceID: card.ID()}) {
		t.Fatal("returned source inherited blocking state")
	}
}

func TestSourceBlockingConditionDeclaration(t *testing.T) {
	condition := SourceBlockingCondition()
	card := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(Blocks(This),
		IfState(condition, GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	meta := InspectDefinition(condition)
	if meta.Role != "state-condition" || meta.Text != "source is blocking" {
		t.Fatalf("condition metadata = %+v", meta)
	}
	spell := NewInstant("Invalid", "", WithEffect(IfState(condition, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell validation = %v", err)
	}
}

func TestSourceBlockingConditionResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Watcher", "", 2, 2, WithAbility(WhenEvent(Blocks(This),
		IfState(SourceBlockingCondition(), GainLifeFor(You, Fixed(1))))))
	attacker := NewCreature("Attacker", "", 2, 2)
	g.PutOnBattlefield(card, a.PlayerID())
	g.PutOnBattlefield(attacker, b.PlayerID())
	g.combat.AddAttacker(attacker.ID(), a.PlayerID())
	g.combat.AddBlockerIncarnation(card.ID(), g.FindPermanent(card.ID()).incarnationID, attacker.ID())
	g.FireEvent(GameEvent{Type: EvtDeclaredBlocker, SourceID: card.ID(), TargetID: attacker.ID(), Flag: true})
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || a.Life() != 20 {
		t.Fatal("clone failed to resolve its blocking trigger")
	}
	g.combat.RemoveFromCombat(card.ID())
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("source resolved after it stopped blocking")
	}
}

func TestSourceBlockingConditionStaticAndInterveningIf(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := SourceBlockingCondition()
	card := NewCreature("Guardian", "", 2, 2,
		WithAbility(Static(While(condition, ModifyPT(This, Fixed(0), Fixed(2))))),
		WithAbility(WhenEvent(Blocks(This), GainLifeFor(You, Fixed(1)), WithInterveningIf(condition))))
	attacker := NewCreature("Attacker", "", 2, 2)
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, a.PlayerID())
	g.PutOnBattlefield(attacker, b.PlayerID())
	g.ApplyContinuousEffects()
	if got := g.FindPermanent(card.ID()).CurrentToughness(g); got != 2 {
		t.Fatalf("before block toughness = %d", got)
	}
	g.combat.AddAttacker(attacker.ID(), a.PlayerID())
	g.combat.AddBlockerIncarnation(card.ID(), g.FindPermanent(card.ID()).incarnationID, attacker.ID())
	g.ApplyContinuousEffects()
	if got := g.FindPermanent(card.ID()).CurrentToughness(g); got != 4 {
		t.Fatalf("blocking toughness = %d", got)
	}
	g.FireEvent(GameEvent{Type: EvtDeclaredBlocker, SourceID: card.ID(), TargetID: attacker.ID(), Flag: true})
	g.PutTriggersOnStack()
	g.RemoveFromCombat(card.ID())
	g.ApplyContinuousEffects()
	if got := g.FindPermanent(card.ID()).CurrentToughness(g); got != 2 {
		t.Fatalf("after combat removal toughness = %d", got)
	}
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("intervening condition did not recheck blocking state")
	}
}

func TestCombatDeparturePreservesAttackerBlockedStatus(t *testing.T) {
	for _, departure := range []string{"leave", "phase"} {
		t.Run(departure, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			attacker := NewCreature("Attacker", "", 2, 2)
			blocker := NewCreature("Blocker", "", 2, 2)
			g.PutOnBattlefield(attacker, a.PlayerID())
			g.PutOnBattlefield(blocker, b.PlayerID())
			g.combat.AddAttacker(attacker.ID(), b.PlayerID())
			g.combat.AddBlockerIncarnation(blocker.ID(), g.FindPermanent(blocker.ID()).incarnationID, attacker.ID())
			oldIncarnation := g.FindPermanent(blocker.ID()).incarnationID
			if departure == "leave" {
				g.RemoveFromBattlefield(g.FindPermanent(blocker.ID()))
				g.PutOnBattlefield(blocker, b.PlayerID())
			} else {
				phased := g.PhaseOut(blocker.ID())
				g.PhaseIn(phased)
			}
			if !g.combat.GroupFor(attacker.ID()).Blocked {
				t.Fatal("departed blocker remained in combat or attacker became unblocked")
			}
			if departure == "leave" {
				if !g.IsBlockingIncarnation(blocker.ID(), oldIncarnation) ||
					g.IsBlockingIncarnation(blocker.ID(), g.FindPermanent(blocker.ID()).incarnationID) {
					t.Fatal("return changed the old combat relationship")
				}
			} else if g.IsBlockingInCombat(blocker.ID()) {
				t.Fatal("phased blocker stayed in combat")
			}
		})
	}
}
