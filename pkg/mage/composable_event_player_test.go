package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamagePlayerBinding(t *testing.T) {
	for _, combat := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		player := BindPlayer("damaged player")
		source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
			DealsDamageToPlayer(This, player), GainLifeFor(FromPlayer(player), Fixed(3)))))
		source.SetOwner(a.PlayerID())
		if err := ValidateCardDefinition(source.Name(), source); err != nil {
			t.Fatal(err)
		}
		g.PutOnBattlefield(source, a.PlayerID())
		g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: source.ID(), TargetID: a.PlayerID(), Amount: 1, Flag: combat})
		g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: source.ID(), TargetID: b.PlayerID(), Amount: 2, Flag: combat})
		g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
		clone := g.Clone()
		for _, game := range []*Game{g, clone} {
			game.PutTriggersOnStack()
			if game.StackSize() != 2 {
				t.Fatalf("stack size = %d", game.StackSize())
			}
			for _, obj := range game.StackObjects() {
				if len(obj.Targets) != 0 || len(obj.TargetSpecs) != 0 {
					t.Fatal("event participant became a target")
				}
			}
			game.ResolveStack()
			if game.GetPlayer(a.PlayerID()).Life() != 23 || game.GetPlayer(b.PlayerID()).Life() != 23 {
				t.Fatal("event participant lost or shared")
			}
		}
	}
}

func TestDamagePlayerBindingValidation(t *testing.T) {
	p := BindPlayer("player")
	for _, tt := range []struct {
		name    string
		trigger TriggeredAbility
		want    string
	}{
		{"valid", WhenEvent(DealsDamageToPlayer(This, p), DrawFor(FromPlayer(p), Fixed(1))), ""},
		{"missing", WhenEvent(Enters(This), DrawFor(FromPlayer(p), Fixed(1))), "producer"},
		{"nil", WhenEvent(DealsDamageToPlayer(This, nil), DrawFor(You, Fixed(1))), "non-nil"},
		{"empty", WhenEvent(DealsDamageToPlayer(This, BindPlayer("")), DrawFor(You, Fixed(1))), "name"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCreature("Test", "", 1, 1, WithAbility(tt.trigger))
			err := ValidateCardDefinition(c.Name(), c)
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestDamagePlayerBindingCopyAndTargets(t *testing.T) {
	for _, modal := range []bool{false, true} {
		a, b := newRecordingPlayer("A"), newRecordingPlayer("B")
		g := NewGame(a, b)
		player := BindPlayer("damaged")
		target := DeclarePlayerTarget("target")
		body := Sequence(GainLifeFor(FromPlayer(player), Fixed(3)), GainLifeFor(target, Fixed(1)))
		trigger := WhenEvent(DealsDamageToPlayer(This, player), body, WithTargets(target))
		if modal {
			trigger = WhenEventModes(DealsDamageToPlayer(This, player), ModeOf("Life", body, WithTargets(target)), ModeOf("Other", DrawFor(You, Fixed(1))))
		}
		source := NewCreature("Source", "", 1, 1, WithAbility(trigger))
		source.SetOwner(a.PlayerID())
		if err := ValidateCardDefinition(source.Name(), source); err != nil {
			t.Fatal(err)
		}
		perm := g.PutOnBattlefield(source, a.PlayerID())
		g.DealDamageToPlayer(b, 1, source.ID())
		a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
		g.PutTriggersOnStack()
		original := g.StackPeek()
		if original == nil || len(original.Targets) != 1 {
			t.Fatal("missing declared target")
		}
		g.RemoveFromBattlefield(perm)
		g.PutOnBattlefield(source, b.PlayerID())
		b.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
		cp := g.CopyStackObjectDirect(original, b.PlayerID(), true)
		if cp == nil || cp.Targets[0] != b.PlayerID() || original.Targets[0] != a.PlayerID() {
			t.Fatal("copy target isolation failed")
		}
		clone := g.Clone()
		clone.ResolveStack()
		if a.Life() != 20 || b.Life() != 19 {
			t.Fatal("clone changed original game")
		}
		g.ResolveStack()
		for _, game := range []*Game{g, clone} {
			if game.GetPlayer(a.PlayerID()).Life() != 21 || game.GetPlayer(b.PlayerID()).Life() != 26 {
				t.Fatal("copy changed event participant")
			}
			if _, ok := FromPlayer(player).resolvePlayer(&EffectContext{Game: game}); ok {
				t.Fatal("event binding leaked")
			}
		}
	}
}

func TestDamagePlayerBindingEventFiltering(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := BindPlayer("damaged")
	source := NewCreature("Source", "", 1, 4, WithAbility(WhenEvent(DealsDamageToPlayer(This, p), GainLifeFor(FromPlayer(p), Fixed(3)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	other := NewCreature("Other", "", 1, 4)
	other.SetOwner(b.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: other.ID(), TargetID: b.PlayerID(), Amount: 1})
	g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: source.ID(), TargetID: other.ID(), Amount: 1})
	g.FireEvent(GameEvent{Type: EvtLifeLost, SourceID: source.ID(), TargetID: b.PlayerID(), Amount: 1})
	g.AddPreventionShield(b.PlayerID(), 2)
	g.DealDamageToPlayer(b, 2, source.ID())
	g.DealDamageToPlayer(b, 0, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("unmatched or prevented damage triggered")
	}
	g.DealDamageToPlayer(a, 1, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("damage to controller did not trigger")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatal("wrong damage recipient")
	}
}

func TestDamagePlayerBindingScope(t *testing.T) {
	p := BindPlayer("damaged")
	read := GainLifeFor(FromPlayer(p), Fixed(1))
	for _, card := range []Card{
		NewCreature("Cross ability", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, p), read)), WithAbility(At(YourUpkeep, read))),
	} {
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "producer") {
			t.Fatalf("scope error = %v", err)
		}
	}
}

func TestDamagePlayerBindingDelayedCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := BindPlayer("damaged")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(DealsDamageToPlayer(This, p), AtNext(EachEndStep, GainLifeFor(FromPlayer(p), Fixed(3))))))
	source.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	g.DealDamageToPlayer(b, 1, source.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if game.GetPlayer(b.PlayerID()).Life() != 22 || game.GetPlayer(a.PlayerID()).Life() != 20 {
			t.Fatal("delayed trigger lost player")
		}
	}
}
