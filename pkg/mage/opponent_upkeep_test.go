package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestOpponentUpkeepCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := BindPlayer("upkeep player")
	card := NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentUpkeep(p), Sequence(
		GainLifeFor(FromPlayer(p), Fixed(2)), GainLifeFor(You, Fixed(1)),
		AtNext(EachEndStep, GainLifeFor(FromPlayer(p), Fixed(3))),
	))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(card, a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("wrong player or step triggered")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("detection did not use current control or acted before resolution")
	}
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, a.PlayerID())
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 23 || game.GetPlayer(b.PlayerID()).Life() != 23 {
			t.Fatal("pending events lost separate players or controllers")
		}
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		if game.StackSize() != 2 {
			t.Fatal("delayed event frames shared")
		}
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 26 || game.GetPlayer(b.PlayerID()).Life() != 26 {
			t.Fatal("delayed player changed")
		}
	}
}

func TestOpponentUpkeepSourceLifetime(t *testing.T) {
	for _, scenario := range []string{"phased", "cycle", "removed", "pending-phase", "pending-remove", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := BindPlayer("upkeep player")
			source := g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentUpkeep(p), GainLifeFor(FromPlayer(p), Fixed(1))))), a.PlayerID())
			switch scenario {
			case "phased":
				g.PhaseOut(source.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "removed":
				g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.Indefinite))
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			want := 1
			if scenario == "phased" || scenario == "removed" {
				want = 0
			}
			if g.StackSize() != want {
				t.Fatal("source lifetime did not control detection")
			}
			switch scenario {
			case "pending-phase":
				g.PhaseOut(source.ID())
			case "pending-remove":
				g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.Indefinite))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				want++
			}
			g.ResolveStack()
			if b.Life() != 20+want || a.Life() != 20 {
				t.Fatal("pending or copied event player changed")
			}
		})
	}
}

func TestOpponentUpkeepDeclarations(t *testing.T) {
	for _, binding := range []*PlayerBinding{nil, BindPlayer("")} {
		card := NewEnchantment("Invalid", "", WithAbility(WhenEvent(OpponentUpkeep(binding), GainLifeFor(You, Fixed(1)))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid event binding accepted")
		}
	}
	p := BindPlayer("local")
	card := NewEnchantment("Unbound", "", WithAbility(WhenEvent(OpponentUpkeep(p), GainLifeFor(FromPlayer(p), Fixed(1)))), WithAbility(At(YourUpkeep, GainLifeFor(FromPlayer(p), Fixed(1)))))
	if ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("event binding escaped its ability")
	}
}
