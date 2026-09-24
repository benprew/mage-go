package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDelayedStepPlayerCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := BindPlayer("future player")
	prior := BindPlayer("earlier player")
	effect := Sequence(SnapshotPlayer(You, prior), SnapshotPlayer(You, p),
		AtNext(TurnStepBegins(EachUpkeep, p), Sequence(
			GainLifeFor(FromPlayer(p), Fixed(3)), GainLifeFor(FromPlayer(prior), Fixed(2)), GainLifeFor(You, Fixed(1)))),
		SnapshotPlayer(You, p))
	source := NewCreature("Source", "", 1, 1, WithAbility(Activate(Costs(), effect)))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	perm := g.PutOnBattlefield(source, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(perm)
	g.PutOnBattlefield(source, b.PlayerID())
	registeredClone := g.Clone()
	for _, game := range []*Game{g, registeredClone} {
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
		if len(game.DelayedTriggers()) != 1 {
			t.Fatal("wrong event consumed trigger")
		}
		event := GameEvent{Type: EvtUpkeep, PlayerID: b.PlayerID(), TargetID: a.PlayerID()}
		game.FireEvent(event)
		event.PlayerID = a.PlayerID()
		if game.StackSize() != 0 || len(game.triggers.Pending()) != 1 {
			t.Fatal("did not wait for priority")
		}
		pendingClone := game.Clone()
		for _, pending := range []*Game{game, pendingClone} {
			pending.PutTriggersOnStack()
			if pending.CopyStackObjectDirect(pending.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			stackClone := pending.Clone()
			stackClone.ResolveStack()
			if pending.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone leaked")
			}
			pending.ResolveStack()
			for _, resolved := range []*Game{pending, stackClone} {
				if resolved.GetPlayer(a.PlayerID()).Life() != 25 || resolved.GetPlayer(b.PlayerID()).Life() != 27 {
					t.Fatalf("life = %d/%d", resolved.GetPlayer(a.PlayerID()).Life(), resolved.GetPlayer(b.PlayerID()).Life())
				}
				resolved.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
				if len(resolved.triggers.Pending()) != 0 {
					t.Fatal("trigger repeated")
				}
				if _, ok := FromPlayer(p).resolvePlayer(&EffectContext{Game: resolved}); ok {
					t.Fatal("binding leaked")
				}
			}
		}
	}
}

func TestDelayedStepPlayerValidation(t *testing.T) {
	p := BindPlayer("future player")
	for _, tt := range []struct {
		body ResolvingEffect
		want string
	}{
		{AtNext(TurnStepBegins(EachUpkeep, p), GainLifeFor(FromPlayer(p), Fixed(1))), ""},
		{Sequence(AtNext(TurnStepBegins(EachUpkeep, p), GainLifeFor(FromPlayer(p), Fixed(1))), GainLifeFor(FromPlayer(p), Fixed(1))), "producer"},
		{AtNext(TurnStepBegins(EachUpkeep, nil), GainLifeFor(You, Fixed(1))), "non-nil"},
		{AtNext(TurnStepBegins(EachUpkeep, BindPlayer("")), GainLifeFor(You, Fixed(1))), "name"},
	} {
		card := NewSorcery("Schedule", "", Spell(tt.body))
		err := ValidateCardDefinition(card.Name(), card)
		if tt.want == "" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("validation = %v, want %s", err, tt.want)
		}
	}
}

func TestDelayedStepPlayerFiltersAndNestedCapture(t *testing.T) {
	for _, tt := range []struct {
		name  string
		step  TurnStepPattern
		event EventType
		flag  bool
		own   bool
	}{
		{"each upkeep", EachUpkeep, EvtUpkeep, false, false},
		{"your upkeep", YourUpkeep, EvtUpkeep, false, true},
		{"each end", EachEndStep, EvtEndStep, false, false},
		{"your end", YourEndStep, EvtEndStep, false, true},
		{"combat", EachEndOfCombat, EvtEndOfCombat, false, false},
		{"draw", EachDrawStep, EvtDrawStep, false, false},
		{"first main", YourFirstMainPhase, EvtMainPhase, true, true},
		{"postcombat main", YourPostcombatMainPhase, EvtMainPhase, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			player := BindPlayer("future player")
			body := AtNext(TurnStepBegins(tt.step, player), AtNext(EachEndStep, GainLifeFor(FromPlayer(player), Fixed(2))))
			source := NewCreature("Source", "", 1, 1, WithAbility(Activate(Costs(), body)))
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.PhaseOut(source.ID())
			if tt.event == EvtMainPhase {
				g.FireEvent(GameEvent{Type: tt.event, PlayerID: a.PlayerID(), Flag: !tt.flag})
				if len(g.triggers.Pending()) != 0 {
					t.Fatal("wrong phase triggered")
				}
			}
			g.FireEvent(GameEvent{Type: tt.event, PlayerID: b.PlayerID(), Flag: tt.flag})
			if tt.own {
				if len(g.triggers.Pending()) != 0 {
					t.Fatal("opponent step triggered")
				}
				g.FireEvent(GameEvent{Type: tt.event, PlayerID: a.PlayerID(), Flag: tt.flag})
			}
			g.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 || len(g.DelayedTriggers()) != 1 {
				t.Fatal("nested trigger resolved early")
			}
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
			g.ResolveStack()
			wantA, wantB := 20, 22
			if tt.own {
				wantA, wantB = 22, 20
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life = %d/%d", a.Life(), b.Life())
			}
		})
	}
}
