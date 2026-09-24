package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttachedControllerUpkeepCapture(t *testing.T) {
	for _, change := range []string{"move", "source returns", "host returns", "phase host", "phase in", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			host, player := BindPermanent("host"), BindPlayer("player")
			body := Sequence(AddCountersTo(From(host), Charge, Fixed(1)), GainLifeFor(FromPlayer(player), Fixed(3)), GainLifeFor(You, Fixed(1)))
			source := NewAura("Watcher", "", WithAbility(WhenEvent(AttachedControllerUpkeep(host, player), body)))
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
			g.PutOnBattlefield(first, b.PlayerID())
			g.PutOnBattlefield(second, a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.Attach(source.ID(), first.ID())
			g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("wrong step or controller triggered")
			}
			g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: b.PlayerID()})
			want := uint8(2)
			switch change {
			case "move":
				g.Attach(source.ID(), second.ID())
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
					if resolved.GetPlayer(a.PlayerID()).Life() != 21 || resolved.GetPlayer(b.PlayerID()).Life() != 27 {
						t.Fatal("captured player or copy controller changed")
					}
				}
			}
		})
	}
}

func TestAttachedControllerUpkeepDelayedModes(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host, player := BindPermanent("host"), BindPlayer("player")
	body := Sequence(AtNext(EachEndStep, Sequence(AddCountersTo(From(host), Charge, Fixed(1)), GainLifeFor(FromPlayer(player), Fixed(2)))), Snapshot(This, host), SnapshotPlayer(You, player))
	source := NewAura("Watcher", "", WithAbility(WhenEventModes(AttachedControllerUpkeep(host, player), ModeOf("Delay", body), ModeOf("Other", DrawFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("unattached source triggered")
	}
	g.Attach(source.ID(), first.ID())
	phased := g.PhaseOut(source.ID())
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phased)
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
	g.Attach(source.ID(), second.ID())
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: b.PlayerID()})
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
		if game.GetPlayer(a.PlayerID()).Life() != 22 || game.GetPlayer(b.PlayerID()).Life() != 22 {
			t.Fatal("delayed player overwritten")
		}
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
		game.PutTriggersOnStack()
		if game.StackSize() != 0 {
			t.Fatal("delayed trigger repeated")
		}
	}
}

func TestAttachedControllerUpkeepValidation(t *testing.T) {
	for _, pattern := range []EventPattern{
		AttachedControllerUpkeep(nil, BindPlayer("player")),
		AttachedControllerUpkeep(BindPermanent("host"), nil),
		AttachedControllerUpkeep(BindPermanent(""), BindPlayer("player")),
		AttachedControllerUpkeep(BindPermanent("same"), BindPlayer("same")),
	} {
		card := NewAura("Invalid", "", WithAbility(WhenEvent(pattern, DrawFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid binding accepted")
		}
	}
	host, player := BindPermanent("host"), BindPlayer("player")
	card := NewAura("Scope", "", WithAbility(WhenEvent(AttachedControllerUpkeep(host, player), DrawFor(You, Fixed(1)))), WithAbility(At(EachEndStep, TapSubject(From(host)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("binding escaped ability")
	}
}
