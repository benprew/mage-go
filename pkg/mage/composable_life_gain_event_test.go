package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestControllerLifeGainEventFiltering(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		requested, replaced, want int
		replace, consume          bool
	}{
		{name: "positive", requested: 3, want: 3},
		{name: "zero"},
		{name: "negative", requested: -1},
		{name: "replacement", requested: 3, replace: true, replaced: 7, want: 7},
		{name: "replacement zero", requested: 3, replace: true},
		{name: "replacement negative", requested: 3, replace: true, replaced: -1},
		{name: "consumed", requested: 3, replace: true, consume: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			player := BindPlayer("gainer")
			source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
				ControllerGainsLife(player), LoseLifeFor(FromPlayer(player), Fixed(1)))))
			g.PutOnBattlefield(source, a.PlayerID())
			g.gainLifeWithEvent(b, 2)
			g.PlayerLoseLife(a, 1)
			for _, amount := range []int{0, -1} {
				g.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: a.PlayerID(), Amount: amount})
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("unmatched event triggered")
			}
			if tc.replace {
				g.effects.AddReplacement(&lifeOutcomeReplacement{amount: tc.replaced, consume: tc.consume})
			}
			g.gainLifeWithEvent(a, tc.requested)
			g.PutTriggersOnStack()
			wantTriggers := 0
			if tc.want > 0 {
				wantTriggers = 1
			}
			if g.StackSize() != wantTriggers {
				t.Fatalf("stack = %d, want %d", g.StackSize(), wantTriggers)
			}
			g.ResolveStack()
			if a.Life() != 19+tc.want-wantTriggers || b.Life() != 22 {
				t.Fatalf("life = %d, %d", a.Life(), b.Life())
			}
		})
	}
}

func TestControllerLifeGainEventLifecycle(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		for _, modal := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			player := BindPlayer("gainer")
			body := Sequence(LoseLifeFor(FromPlayer(player), Fixed(2)), LoseLifeFor(You, Fixed(1)))
			if delayed {
				body = AtNext(EachEndStep, body)
			}
			pattern := ControllerGainsLife(player)
			trigger := WhenEvent(pattern, body)
			if modal {
				trigger = WhenEventModes(pattern, ModeOf("Gain", body), ModeOf("Other", LoseLifeFor(You, Fixed(9))))
			}
			source := NewEnchantment("Watcher", "", WithAbility(trigger))
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.gainLifeWithEvent(a, 1)
			g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			g.gainLifeWithEvent(a, 1)
			g.gainLifeWithEvent(b, 1)
			pendingClone := g.Clone()
			for _, game := range []*Game{g, pendingClone} {
				game.PutTriggersOnStack()
				if game.StackSize() != 2 {
					t.Fatalf("stack = %d", game.StackSize())
				}
				for _, obj := range game.StackObjects() {
					if len(obj.Targets) != 0 {
						t.Fatal("event player became a target")
					}
					if obj.Controller == a.PlayerID() {
						game.CopyStackObjectDirect(obj, b.PlayerID(), false)
						break
					}
				}
				game.RemoveFromBattlefield(game.FindPermanent(source.ID()))
				game.PutOnBattlefield(source, a.PlayerID())
				cloned := game.Clone()
				for _, resolved := range []*Game{game, cloned} {
					resolved.ResolveStack()
					if delayed {
						resolved.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
						resolved.PutTriggersOnStack()
						resolved.ResolveStack()
					}
					if resolved.GetPlayer(a.PlayerID()).Life() != 17 || resolved.GetPlayer(b.PlayerID()).Life() != 17 {
						t.Fatalf("life = %d, %d", resolved.GetPlayer(a.PlayerID()).Life(), resolved.GetPlayer(b.PlayerID()).Life())
					}
				}
			}
		}
	}
}

func TestControllerLifeGainEventSourceIdentity(t *testing.T) {
	for _, returned := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerGainsLife(BindPlayer("gainer")), Destroy(This))))
		source.SetOwner(a.PlayerID())
		g.PutOnBattlefield(source, a.PlayerID())
		phased := g.PhaseOut(source.ID())
		g.gainLifeWithEvent(a, 1)
		g.PutTriggersOnStack()
		if g.StackSize() != 0 {
			t.Fatal("phased source triggered")
		}
		g.PhaseIn(phased)
		g.gainLifeWithEvent(a, 1)
		if returned {
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
		} else {
			phased = g.PhaseOut(source.ID())
			g.PhaseIn(phased)
		}
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatal("pending trigger lost")
		}
		g.ResolveStack()
		if (g.FindPermanent(source.ID()) != nil) != returned {
			t.Fatal("incorrect source incarnation")
		}
	}
}

func TestControllerLifeGainEventValidation(t *testing.T) {
	p := BindPlayer("gainer")
	for _, tt := range []struct {
		pattern EventPattern
		want    string
	}{
		{ControllerGainsLife(p), ""}, {ControllerGainsLife(nil), "non-nil"},
		{ControllerGainsLife(BindPlayer("")), "name"}, {ControllerGainsLife(BindPlayer("other")), "producer"},
	} {
		card := NewEnchantment("Watcher", "", WithAbility(WhenEvent(tt.pattern, LoseLifeFor(FromPlayer(p), Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tt.want == "" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("want %q: %v", tt.want, err)
		}
	}
	card := NewEnchantment("Scope", "", WithAbility(WhenEvent(ControllerGainsLife(p), LoseLifeFor(FromPlayer(p), Fixed(1)))), WithAbility(At(YourUpkeep, LoseLifeFor(FromPlayer(p), Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("scope: %v", err)
	}
}
