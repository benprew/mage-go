package mage

import (
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageOpponentBindingValidation(t *testing.T) {
	for _, tt := range []struct {
		name   string
		player *PlayerBinding
		want   string
	}{
		{"valid", BindPlayer("opponent"), ""},
		{"nil", nil, "non-nil"},
		{"empty", BindPlayer(""), "name"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			card := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
				DealsDamageToOpponent(This, tt.player), DrawFor(You, Fixed(1)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
	p := BindPlayer("opponent")
	card := NewCreature("Scope", "", 1, 1,
		WithAbility(WhenEvent(DealsDamageToOpponent(This, p), DrawFor(FromPlayer(p), Fixed(1)))),
		WithAbility(WhenEvent(Enters(This), DrawFor(FromPlayer(p), Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("scope error = %v", err)
	}
}

func TestDamageOpponentBindingFiltering(t *testing.T) {
	for _, combat := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		player := BindPlayer("damaged opponent")
		source := NewCreature("Source", "", 2, 2, WithAbility(WhenEvent(
			DealsDamageToOpponent(This, player), GainLifeFor(FromPlayer(player), Fixed(3)))))
		source.SetOwner(a.PlayerID())
		g.PutOnBattlefield(source, a.PlayerID())
		g.damage.resolvingCombatDamage = combat
		g.DealDamageToPlayer(a, 1, source.ID())
		g.DealDamageToPlayer(b, 1, a.PlayerID())
		g.AddPreventionShield(b.PlayerID(), 2)
		g.DealDamageToPlayer(b, 2, source.ID())
		g.DealDamageToPlayer(b, 0, source.ID())
		g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: source.ID(), TargetID: source.ID(), Amount: 1, Flag: combat})
		g.PutTriggersOnStack()
		if g.StackSize() != 0 {
			t.Fatal("unmatched damage triggered")
		}
		g.DealDamageToPlayer(b, 2, source.ID())
		g.PutTriggersOnStack()
		if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 {
			t.Fatal("opponent damage must produce one trigger without targets")
		}
		g.ResolveStack()
		if a.Life() != 19 || b.Life() != 20 {
			t.Fatalf("life totals = %d, %d", a.Life(), b.Life())
		}
	}
}

func TestDamageOpponentBindingControlCopyClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("damaged opponent")
	source := NewCreature("Source", "", 2, 2, WithAbility(WhenEvent(
		DealsDamageToOpponent(This, player), DiscardAtRandom(FromPlayer(player), Fixed(1)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	for _, name := range []string{"First", "Second", "Third"} {
		b.AddToHand(NewSorcery(name, ""))
	}
	g.DealDamageToPlayer(b, 1, source.ID())
	g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	g.DealDamageToPlayer(b, 1, source.ID())
	g.DealDamageToPlayer(a, 1, source.ID())
	g.SetRandomResults([]int{1, 0, 7})
	clone := g.Clone()
	for _, game := range []*Game{clone, g} {
		game.PutTriggersOnStack()
		if game.StackSize() != 2 {
			t.Fatalf("control change produced %d triggers", game.StackSize())
		}
		for _, obj := range game.StackObjects() {
			if obj.Controller == a.PlayerID() {
				if game.CopyStackObjectDirect(obj, b.PlayerID(), true) == nil {
					t.Fatal("copy failed")
				}
				break
			}
		}
		game.RemoveFromBattlefield(game.FindPermanent(source.ID()))
		game.PutOnBattlefield(source, a.PlayerID())
		game.ResolveStack()
		hand := game.GetPlayer(b.PlayerID()).Hand()
		if len(hand) != 1 || hand[0].Name() != "Third" || game.RandIntn(10) != 7 {
			t.Fatal("capture, random queue, or copy changed the discard recipient")
		}
		if game == clone && len(b.Hand()) != 3 {
			t.Fatal("clone changed the original hand")
		}
	}
}
