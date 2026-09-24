package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttachedBecomesTappedCapture(t *testing.T) {
	for _, change := range []string{"none", "move", "source leaves", "source returns", "host returns", "phase host", "phase in", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			host := BindPermanent("host")
			body := Sequence(AddCountersTo(From(host), Charge, Fixed(1)), GainLifeFor(You, Fixed(1)))
			source := NewAura("Watcher", "", WithAbility(WhenEvent(AttachedBecomesTapped(host), body)))
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
			g.PutOnBattlefield(first, b.PlayerID())
			g.PutOnBattlefield(second, a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.Attach(source.ID(), first.ID())
			g.TapPermanent(g.FindPermanent(second.ID()))
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("wrong step or controller triggered")
			}
			g.TapPermanent(g.FindPermanent(first.ID()))
			want := uint8(2)
			switch change {
			case "move":
				g.Attach(source.ID(), second.ID())
			case "source leaves":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "source returns":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(source, b.PlayerID())
				g.Attach(source.ID(), second.ID())
			case "host returns":
				g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
				g.PutOnBattlefield(first, b.PlayerID())
				want = 0
			case "phase host":
				g.PhaseOut(first.ID())
				want = 0
			case "phase in":
				phased := g.PhaseOut(first.ID())
				g.PhaseIn(phased)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: first.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
			}
			pendingClone := g.Clone()
			for _, game := range []*Game{g, pendingClone} {
				game.PutTriggersOnStack()
				if game.StackSize() != 1 || len(game.StackPeek().Targets) != 0 {
					t.Fatal("missing nontargeted trigger")
				}
				if game.CopyStackObjectDirect(game.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("missing copy")
				}
				clone := game.Clone()
				clone.ResolveStack()
				if game.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("clone leaked")
				}
				game.ResolveStack()
				for _, resolved := range []*Game{game, clone} {
					if resolved.FindPermanentIncludingPhased(first.ID()).Counters[Charge] != want || resolved.FindPermanent(second.ID()).Counters[Charge] != 0 {
						t.Fatal("captured host changed")
					}
					if resolved.GetPlayer(a.PlayerID()).Life() != 21 || resolved.GetPlayer(b.PlayerID()).Life() != 21 {
						t.Fatal("captured player or copy controller changed")
					}
				}
			}
		})
	}
}

func TestAttachedBecomesTappedDelayedModes(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := BindPermanent("host")
	body := Sequence(AtNext(EachEndStep, Sequence(AddCountersTo(From(host), Charge, Fixed(1)))), Snapshot(This, host))
	source := NewAura("Watcher", "", WithAbility(WhenEventModes(AttachedBecomesTapped(host), ModeOf("Delay", body), ModeOf("Other", DrawFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtTapped, SourceID: first.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("unattached source triggered")
	}
	g.Attach(source.ID(), first.ID())
	phased := g.PhaseOut(source.ID())
	g.FireEvent(GameEvent{Type: EvtTapped, SourceID: first.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phased)
	g.FireEvent(GameEvent{Type: EvtTapped, SourceID: first.ID(), PlayerID: a.PlayerID()})
	g.Attach(source.ID(), second.ID())
	g.FireEvent(GameEvent{Type: EvtTapped, SourceID: second.ID(), PlayerID: b.PlayerID()})
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("events lost separate captures")
	}
	g.ResolveStack()
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if game.FindPermanent(first.ID()).Counters[Charge] != 1 || game.FindPermanent(second.ID()).Counters[Charge] != 1 {
			t.Fatal("delayed host overwritten")
		}
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
		game.PutTriggersOnStack()
		if game.StackSize() != 0 {
			t.Fatal("delayed trigger repeated")
		}
	}
}

func TestAttachedBecomesTappedValidation(t *testing.T) {
	for _, pattern := range []EventPattern{
		AttachedBecomesTapped(nil),
		AttachedBecomesTapped(BindPermanent("")),
	} {
		card := NewAura("Invalid", "", WithAbility(WhenEvent(pattern, DrawFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid binding accepted")
		}
	}
	host := BindPermanent("host")
	card := NewAura("Scope", "", WithAbility(WhenEvent(AttachedBecomesTapped(host), DrawFor(You, Fixed(1)))), WithAbility(At(EachEndStep, TapSubject(From(host)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("binding escaped ability")
	}
}
