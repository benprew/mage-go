package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTurnStepPlayerCapture(t *testing.T) {
	for _, modal := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		player := BindPlayer("step player")
		pattern := TurnStepBegins(EachUpkeep, player)
		body := Sequence(GainLifeFor(FromPlayer(player), Fixed(3)), GainLifeFor(You, Fixed(1)))
		ability := WhenEvent(pattern, body)
		if modal {
			ability = WhenEventModes(pattern, ModeOf("Life", body), ModeOf("Other", DrawFor(You, Fixed(1))))
		}
		source := NewEnchantment("Watcher", "", WithAbility(ability))
		if err := ValidateCardDefinition(source.Name(), source); err != nil {
			t.Fatal(err)
		}
		perm := g.PutOnBattlefield(source, a.PlayerID())
		g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: b.PlayerID(), TargetID: a.PlayerID()})
		g.RemoveFromBattlefield(perm)
		g.PutOnBattlefield(source, b.PlayerID())
		pendingClone := g.Clone()
		for _, game := range []*Game{g, pendingClone} {
			game.PutTriggersOnStack()
			original := game.StackPeek()
			if original == nil || len(original.Targets) != 0 {
				t.Fatal("missing nontargeted trigger")
			}
			if game.CopyStackObjectDirect(original, b.PlayerID(), false) == nil {
				t.Fatal("missing copy")
			}
			clone := game.Clone()
			clone.ResolveStack()
			if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("clone changed original")
			}
			game.ResolveStack()
			for _, resolved := range []*Game{game, clone} {
				if resolved.GetPlayer(a.PlayerID()).Life() != 21 || resolved.GetPlayer(b.PlayerID()).Life() != 27 {
					t.Fatal("event player or controller changed")
				}
				if _, ok := FromPlayer(player).resolvePlayer(&EffectContext{Game: resolved}); ok {
					t.Fatal("binding leaked")
				}
			}
		}
	}
}

func TestTurnStepPlayerFiltering(t *testing.T) {
	for _, tt := range []struct {
		name    string
		pattern TurnStepPattern
		event   EventType
		first   bool
		own     bool
	}{
		{"each upkeep", EachUpkeep, EvtUpkeep, false, false},
		{"own upkeep", YourUpkeep, EvtUpkeep, false, true},
		{"end step", EachEndStep, EvtEndStep, false, false},
		{"own end step", YourEndStep, EvtEndStep, false, true},
		{"combat end", EachEndOfCombat, EvtEndOfCombat, false, false},
		{"draw step", EachDrawStep, EvtDrawStep, false, false},
		{"first main", YourFirstMainPhase, EvtMainPhase, true, true},
		{"postcombat main", YourPostcombatMainPhase, EvtMainPhase, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := BindPlayer("player")
			source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(TurnStepBegins(tt.pattern, p), GainLifeFor(FromPlayer(p), Fixed(1)))))
			g.PutOnBattlefield(source, a.PlayerID())
			g.FireEvent(GameEvent{Type: EvtLifeLost, PlayerID: a.PlayerID()})
			if tt.event == EvtMainPhase {
				g.FireEvent(GameEvent{Type: tt.event, PlayerID: a.PlayerID(), Flag: !tt.first})
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("unmatched event triggered")
			}
			for _, player := range []Player{a, b} {
				g.FireEvent(GameEvent{Type: tt.event, PlayerID: player.PlayerID(), Flag: tt.first})
			}
			g.PutTriggersOnStack()
			want := 2
			if tt.own {
				want = 1
			}
			if g.StackSize() != want {
				t.Fatalf("triggers = %d, want %d", g.StackSize(), want)
			}
			g.ResolveStack()
			wantB := 21
			if tt.own {
				wantB = 20
			}
			if a.Life() != 21 || b.Life() != wantB {
				t.Fatalf("life = %d/%d", a.Life(), b.Life())
			}
		})
	}
}

func TestTurnStepPlayerDelayedCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := BindPlayer("step player")
	body := Sequence(AtNext(EachEndStep, GainLifeFor(FromPlayer(p), Fixed(3))), SnapshotPlayer(You, p))
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(TurnStepBegins(EachUpkeep, p), body)))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	perm := g.PutOnBattlefield(source, a.PlayerID())
	for _, player := range []Player{a, b} {
		g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: player.PlayerID()})
	}
	g.PutTriggersOnStack()
	g.ResolveStack()
	g.RemoveFromBattlefield(perm)
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		if game.StackSize() != 2 {
			t.Fatal("delayed captures shared")
		}
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 23 || game.GetPlayer(b.PlayerID()).Life() != 23 {
			t.Fatal("delayed player changed")
		}
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: b.PlayerID()})
		game.PutTriggersOnStack()
		if game.StackSize() != 0 {
			t.Fatal("delayed trigger repeated")
		}
	}
}

func TestTurnStepPlayerValidation(t *testing.T) {
	p := BindPlayer("player")
	for _, tt := range []struct {
		pattern EventPattern
		want    string
	}{
		{TurnStepBegins(EachUpkeep, p), ""},
		{TurnStepBegins(EachUpkeep, nil), "non-nil"},
		{TurnStepBegins(EachUpkeep, BindPlayer("")), "name"},
	} {
		card := NewEnchantment("Watcher", "", WithAbility(WhenEvent(tt.pattern, GainLifeFor(You, Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tt.want == "" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("validation = %v", err)
		}
	}
	card := NewEnchantment("Scope", "", WithAbility(WhenEvent(TurnStepBegins(EachUpkeep, p), GainLifeFor(FromPlayer(p), Fixed(1)))), WithAbility(At(EachEndStep, GainLifeFor(FromPlayer(p), Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("scope = %v", err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("nil pattern accepted")
		}
	}()
	TurnStepBegins(nil, p)
}

func TestTurnStepPlayerUpkeepAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := BindPlayer("step player")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(TurnStepBegins(EachUpkeep, p), GainLifeFor(FromPlayer(p), Fixed(3)))))
	g.PutOnBattlefield(source, b.PlayerID())
	phased := g.PhaseOut(source.ID())
	g.doUpkeepActions()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phased)
	g.doUpkeepActions()
	if g.StackSize() != 1 || a.Life() != 20 {
		t.Fatal("upkeep trigger did not use the stack")
	}
	g.PhaseOut(source.ID())
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 20 {
		t.Fatal("upkeep event lost its player after source phased out")
	}
}
