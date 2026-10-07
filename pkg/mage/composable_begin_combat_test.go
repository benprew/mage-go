package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestBeginCombatPatterns(t *testing.T) {
	for _, own := range []bool{false, true} {
		t.Run(map[bool]string{false: "each", true: "your"}[own], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			var step TurnStepPattern = EachBeginCombat
			if own {
				step = YourBeginCombat
			}
			player := BindPlayer("combat player")
			card := NewEnchantment("Observer", "", WithAbility(WhenEvent(TurnStepBegins(step, player), GainLifeFor(FromPlayer(player), Fixed(1)))))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(card, a.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtEndOfCombat, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("wrong boundary triggered")
			}
			for _, p := range []Player{a, b, a} {
				g.FireEvent(core.GameEvent{Type: core.EvtBeginCombat, PlayerID: p.PlayerID()})
			}
			g.PutTriggersOnStack()
			want := 3
			if own {
				want = 2
			}
			if g.StackSize() != want || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("event selection or early resolution")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			wantB := 21
			if own {
				wantB = 20
			}
			if a.Life() != 22 || b.Life() != wantB || clone.GetPlayer(b.PlayerID()).Life() != wantB {
				t.Fatal("pending events shared capture")
			}
		})
	}
}

func TestBeginCombatDelayedAndAPNAP(t *testing.T) {
	for _, own := range []bool{false, true} {
		t.Run(map[bool]string{false: "each", true: "your"}[own], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			var step TurnStepPattern = EachBeginCombat
			if own {
				step = YourBeginCombat
			}
			player := BindPlayer("future combat player")
			effect := AtNext(TurnStepBegins(step, player), GainLifeFor(FromPlayer(player), Fixed(2)))
			source := NewArtifact("Schedule", "", WithAbility(Activate(Costs(), effect)))
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(source, a.PlayerID())
			if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.RemoveFromBattlefield(p)
			g.FireEvent(core.GameEvent{Type: core.EvtBeginCombat, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if own {
				if g.StackSize() != 0 {
					t.Fatal("wrong player's delayed step")
				}
				g.FireEvent(core.GameEvent{Type: core.EvtBeginCombat, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
			}
			if g.StackSize() != 1 {
				t.Fatal("missing delayed step")
			}
			clone := g.Clone()
			clone.ResolveStack()
			g.ResolveStack()
			recipient := b.PlayerID()
			if own {
				recipient = a.PlayerID()
			}
			if g.GetPlayer(recipient).Life() != 22 || clone.GetPlayer(recipient).Life() != 22 {
				t.Fatal("delayed player capture")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtBeginCombat, PlayerID: a.PlayerID()})
			g.ResolveStack()
			if g.GetPlayer(recipient).Life() != 22 {
				t.Fatal("delayed trigger repeated")
			}
		})
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for _, player := range []Player{a, b} {
		binding := BindPlayer("combat player")
		g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(WhenEvent(TurnStepBegins(EachBeginCombat, binding), GainLifeFor(You, Fixed(1))))), player.PlayerID())
	}
	g.SetActivePlayerIndex(0)
	g.FireEvent(core.GameEvent{Type: core.EvtBeginCombat, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 || g.StackPeek().Controller != b.PlayerID() {
		t.Fatal("combat triggers did not use APNAP order")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatal("independent controllers")
	}
}

func TestBeginCombatSkippedNativeStep(t *testing.T) {
	g := newPriorityTestGame()
	a := g.ActivePlayerObj()
	p := BindPlayer("combat player")
	g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(WhenEvent(TurnStepBegins(YourBeginCombat, p), GainLifeFor(FromPlayer(p), Fixed(1))))), a.PlayerID())
	g.SkipNextCombatPhase()
	g.turns.BuildNextTurn()
	for {
		step, ok := g.turns.PopNextStep()
		if !ok {
			t.Fatal("missing main phase")
		}
		if step == core.PostcombatMain {
			break
		}
		if step >= core.PrecombatMain {
			g.RunStepWithPriority(step)
		}
	}
	if a.Life() != 20 {
		t.Fatal("skipped combat triggered")
	}
	g.RunStepWithPriority(core.BeginCombat)
	if a.Life() != 21 {
		t.Fatal("later combat did not trigger")
	}
}
