package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceCombatEvents(t *testing.T) {
	for _, tc := range []struct {
		name  string
		event EventType
		flag  bool
		make  func(sourceRef) EventPattern
	}{
		{"attacks", EvtDeclaredAttacker, false, Attacks},
		{"blocks", EvtDeclaredBlocker, true, Blocks},
		{"becomes blocked", EvtCreatureBecomesBlocked, false, BecomesBlocked},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(tc.make(This),
				Sequence(Until(EndOfTurn, ModifyPT(This, Fixed(2), Fixed(2))), GainLifeFor(You, Fixed(1))))))
			source.SetOwner(a.PlayerID())
			other := NewCreature("Other", "", 1, 1)
			other.SetOwner(b.PlayerID())
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(other, b.PlayerID())
			g.FireEvent(GameEvent{Type: tc.event, SourceID: other.ID(), PlayerID: b.PlayerID(), Flag: tc.flag})
			g.FireEvent(GameEvent{Type: EvtEntersAttacking, SourceID: source.ID(), PlayerID: a.PlayerID()})
			g.FireEvent(GameEvent{Type: EvtEntersBlocking, SourceID: source.ID(), PlayerID: a.PlayerID()})
			g.FireEvent(GameEvent{Type: tc.event, SourceID: source.ID(), PlayerID: a.PlayerID(), Flag: tc.flag})
			if tc.event == EvtDeclaredBlocker {
				g.FireEvent(GameEvent{Type: tc.event, SourceID: source.ID(), PlayerID: a.PlayerID(), Flag: false})
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 {
				t.Fatal("wrong trigger count or unexpected target")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || g.FindPermanent(source.ID()).CurrentPower(g) != 1 {
				t.Fatal("clone changed original game")
			}
			g.ResolveStack()
			for _, game := range []*Game{g, clone} {
				if game.GetPlayer(a.PlayerID()).Life() != 21 || game.FindPermanent(source.ID()).CurrentPower(game) != 3 {
					t.Fatal("source combat trigger did not resolve")
				}
			}
		})
	}
}

func TestSourceCombatEventIncarnationAndPhasing(t *testing.T) {
	for _, change := range []string{"leaves", "returns", "phased", "phases in"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(BecomesBlocked(This),
				Sequence(Until(EndOfTurn, ModifyPT(This, Fixed(2), Fixed(2))), GainLifeFor(You, Fixed(1))))))
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.FireEvent(GameEvent{Type: EvtCreatureBecomesBlocked, SourceID: source.ID()})
			switch change {
			case "leaves", "returns":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "returns" {
					g.PutOnBattlefield(source, b.PlayerID())
				}
			case "phased", "phases in":
				phased := g.PhaseOut(source.ID())
				if change == "phases in" {
					g.PhaseIn(phased)
				}
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("trigger lost after source changed")
			}
			g.ResolveStack()
			if a.Life() != 21 || b.Life() != 20 {
				t.Fatal("trigger controller changed")
			}
			if change == "returns" && g.FindPermanent(source.ID()).CurrentPower(g) != 1 {
				t.Fatal("new incarnation received old bonus")
			}
		})
	}
}

func TestSourceCombatEventValidation(t *testing.T) {
	for _, pattern := range []EventPattern{Attacks(This), Blocks(This), BecomesBlocked(This)} {
		card := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		meta := InspectDefinition(card.Abilities()[0])
		if meta.Role != "event-trigger" || len(meta.Children) == 0 || meta.Children[0].Role != "event-pattern" {
			t.Fatal("combat event missing from inspection")
		}
	}
	spell := NewInstant("Invalid", "", WithEffect(GainLifeFor(You, Fixed(1))))
	spell.AddAbility(WhenEvent(BecomesBlocked(This), GainLifeFor(You, Fixed(1))))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("spell event validation = %v", err)
	}
}

func TestSourceBecomesBlockedAbsentOrPhasedBeforeEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(BecomesBlocked(This), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(source, a.PlayerID())
	phase := g.PhaseOut(source.ID())
	g.FireEvent(GameEvent{Type: EvtCreatureBecomesBlocked, SourceID: source.ID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phase)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.FireEvent(GameEvent{Type: EvtCreatureBecomesBlocked, SourceID: source.ID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("absent source triggered")
	}
}

func TestSourceCombatEventCopyAndTarget(t *testing.T) {
	a, b := newRecordingPlayer("A"), newRecordingPlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("player")
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(BecomesBlocked(This),
		GainLifeFor(target, Fixed(2)), WithTargets(target))))
	source.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
	g.FireEvent(GameEvent{Type: EvtCreatureBecomesBlocked, SourceID: source.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	original := g.StackPeek()
	if original == nil || len(original.Targets) != 1 || original.Targets[0] != a.PlayerID() {
		t.Fatal("trigger target was not announced")
	}
	copied := g.CopyStackObjectDirect(original, b.PlayerID(), false)
	if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != a.PlayerID() {
		t.Fatal("copy lost target")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || g.StackSize() != 2 {
		t.Fatal("clone changed original stack")
	}
	g.ResolveStack()
	if a.Life() != 24 || b.Life() != 20 {
		t.Fatal("copied combat trigger changed target")
	}
}
