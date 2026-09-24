package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestOpponentLifeEventFiltering(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player, amount := BindPlayer("loser"), BindNumber("lost")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentLosesLife(player).WithAmount(amount), GainLifeFor(FromPlayer(player), FromNumber(amount)))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PlayerLoseLife(a, 2)
	g.PlayerLoseLife(b, 0)
	g.PlayerLoseLife(b, -1)
	g.FireEvent(GameEvent{Type: EvtLifeLost, PlayerID: b.PlayerID(), Amount: 0})
	g.AddPreventionShield(b.PlayerID(), 2)
	g.DealDamageToPlayer(b, 2, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("unmatched event triggered")
	}
	g.PlayerLoseLife(b, 3)
	g.DealDamageToPlayer(b, 4, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d", g.StackSize())
	}
	for _, obj := range g.StackObjects() {
		if len(obj.Targets) != 0 {
			t.Fatal("event player became a target")
		}
	}
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 20 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
}

func TestOpponentLifeEventLifecycle(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		for _, modal := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			player, amount := BindPlayer("loser"), BindNumber("lost")
			body := Sequence(GainLifeFor(FromPlayer(player), FromNumber(amount)), GainLifeFor(You, Fixed(1)))
			if delayed {
				body = AtNext(EachEndStep, body)
			}
			pattern := OpponentLosesLife(player).WithAmount(amount)
			trigger := WhenEvent(pattern, body)
			if modal {
				trigger = WhenEventModes(pattern, ModeOf("Gain", body), ModeOf("Other", GainLifeFor(You, Fixed(1))))
			}
			source := NewEnchantment("Watcher", "", WithAbility(trigger))
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			g.PlayerLoseLife(b, 3)
			g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			g.PlayerLoseLife(b, 2)
			g.PlayerLoseLife(a, 4)
			pendingClone := g.Clone()
			for _, game := range []*Game{g, pendingClone} {
				game.PutTriggersOnStack()
				if game.StackSize() != 2 {
					t.Fatalf("stack = %d", game.StackSize())
				}
				for _, obj := range game.StackObjects() {
					if obj.Controller == a.PlayerID() {
						game.CopyStackObjectDirect(obj, b.PlayerID(), false)
						break
					}
				}
				game.RemoveFromBattlefield(game.FindPermanent(source.ID()))
				game.PutOnBattlefield(source, a.PlayerID())
				stackClone := game.Clone()
				for _, resolved := range []*Game{game, stackClone} {
					resolved.ResolveStack()
					if delayed {
						resolved.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
						resolved.PutTriggersOnStack()
						resolved.ResolveStack()
					}
					if resolved.GetPlayer(a.PlayerID()).Life() != 21 || resolved.GetPlayer(b.PlayerID()).Life() != 23 {
						t.Fatalf("life = %d, %d", resolved.GetPlayer(a.PlayerID()).Life(), resolved.GetPlayer(b.PlayerID()).Life())
					}
				}
			}
		}
	}
}

func TestOpponentLifeEventValidation(t *testing.T) {
	p, n := BindPlayer("player"), BindNumber("amount")
	read := GainLifeFor(FromPlayer(p), FromNumber(n))
	for _, tt := range []struct {
		pattern EventPattern
		want    string
	}{
		{OpponentLosesLife(p).WithAmount(n), ""},
		{OpponentLosesLife(nil).WithAmount(n), "non-nil"},
		{OpponentLosesLife(p).WithAmount(nil), "non-nil"},
		{OpponentLosesLife(p), "producer"},
		{OpponentLosesLife(p).WithAmount(BindNumber("player")), "duplicate"},
		{OpponentLosesLife(BindPlayer("")).WithAmount(n), "name"},
	} {
		card := NewEnchantment("Watcher", "", WithAbility(WhenEvent(tt.pattern, read)))
		err := ValidateCardDefinition(card.Name(), card)
		if tt.want == "" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("want %q: %v", tt.want, err)
		}
	}
	card := NewEnchantment("Scope", "", WithAbility(WhenEvent(OpponentLosesLife(p).WithAmount(n), read)), WithAbility(At(YourUpkeep, read)))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("scope: %v", err)
	}
}

func TestOpponentLifeEventPaymentAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("loser")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(OpponentLosesLife(player), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(source, a.PlayerID())
	phased := g.PhaseOut(source.ID())
	g.PlayerLoseLife(b, 2)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn(phased)
	cost := LifePayCost(3)
	if !cost.CanPay(source.ID(), b.PlayerID(), g) {
		t.Fatal("cost unavailable")
	}
	if err := cost.Pay(source.ID(), b.PlayerID(), g); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack = %d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 15 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
}
