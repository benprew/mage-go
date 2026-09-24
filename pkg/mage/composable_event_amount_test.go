package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageEventAmountLifecycle(t *testing.T) {
	for _, modal := range []bool{false, true} {
		for _, delayed := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			player, amount := BindPlayer("damaged"), BindNumber("damage amount")
			pattern := DealsDamageToPlayer(This, player).WithAmount(amount)
			body := GainLifeFor(FromPlayer(player), FromNumber(amount))
			if delayed {
				body = AtNext(EachEndStep, body)
			}
			trigger := WhenEvent(pattern, body)
			if modal {
				trigger = WhenEventModes(pattern, ModeOf("Gain", body), ModeOf("Other", DrawFor(You, Fixed(1))))
			}
			source := NewCreature("Source", "", 1, 1, WithAbility(trigger))
			source.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(source, a.PlayerID())
			g.AddPreventionShield(b.PlayerID(), 2)
			g.DealDamageToPlayer(b, 5, source.ID())
			g.DealDamageToPlayer(a, 1, source.ID())
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			g.PutOnBattlefield(source, b.PlayerID())
			pendingClone := g.Clone()
			for _, game := range []*Game{g, pendingClone} {
				game.PutTriggersOnStack()
				if game.StackSize() != 2 {
					t.Fatalf("stack = %d", game.StackSize())
				}
				for _, obj := range game.StackObjects() {
					game.CopyStackObjectDirect(obj, b.PlayerID(), false)
				}
				stackClone := game.Clone()
				for _, resolved := range []*Game{game, stackClone} {
					resolved.ResolveStack()
					if delayed {
						delayedClone := resolved.Clone()
						for _, final := range []*Game{resolved, delayedClone} {
							final.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
							final.PutTriggersOnStack()
							final.ResolveStack()
							if final.GetPlayer(a.PlayerID()).Life() != 21 || final.GetPlayer(b.PlayerID()).Life() != 23 {
								t.Fatal("delayed event amount lost or shared")
							}
						}
					}
					if resolved.GetPlayer(a.PlayerID()).Life() != 21 || resolved.GetPlayer(b.PlayerID()).Life() != 23 {
						t.Fatal("event amount lost or shared")
					}
					if FromNumber(amount).resolveValue(&EffectContext{Game: resolved}) != 0 {
						t.Fatal("amount escaped resolution")
					}
				}
			}
		}
	}
}

func TestDamageEventAmountValidation(t *testing.T) {
	p, n := BindPlayer("player"), BindNumber("amount")
	read := GainLifeFor(You, FromNumber(n))
	for _, tt := range []struct {
		name    string
		trigger TriggeredAbility
		want    string
	}{
		{"valid", WhenEvent(DealsDamageToOpponent(This, p).WithAmount(n), read), ""},
		{"unbound", WhenEvent(DealsDamageToPlayer(This, p), read), "producer"},
		{"unrelated", WhenEvent(Enters(This), read), "producer"},
		{"nil", WhenEvent(DealsDamageToPlayer(This, p).WithAmount(nil), read), "non-nil"},
		{"empty", WhenEvent(DealsDamageToPlayer(This, p).WithAmount(BindNumber("")), read), "name"},
		{"duplicate", WhenEvent(DealsDamageToPlayer(This, p).WithAmount(BindNumber("player")), read), "duplicate"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			card := NewCreature("Source", "", 1, 1, WithAbility(tt.trigger))
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
	card := NewCreature("Cross ability", "", 1, 1,
		WithAbility(WhenEvent(DealsDamageToPlayer(This, p).WithAmount(n), read)), WithAbility(At(YourUpkeep, read)))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("scope error = %v", err)
	}
}

func TestDamageEventAmountFrameIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("amount")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	capture := &eventAmountCapture{n, 3}
	if err := capture.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := (&eventAmountCapture{n, 9}).Apply(&EffectContext{Game: clone}); err != nil {
		t.Fatal(err)
	}
	if FromNumber(n).resolveValue(ctx) != 3 {
		t.Fatal("clone shared number map")
	}
	clone.resolution.Clear()
	if len(clone.resolution.numberBindings) != 0 {
		t.Fatal("clear retained number map")
	}
	if err := ApplyEffect(g, GainLifeFor(You, FromNumber(n)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || FromNumber(n).resolveValue(ctx) != 3 {
		t.Fatal("nested resolution did not isolate and restore number map")
	}
	g.PutOnBattlefield(NewCreature("First", "", 1, 1), a.PlayerID())
	g.PutOnBattlefield(NewCreature("Second", "", 1, 1), a.PlayerID())
	loop := ForEach(MatchingPermanents(IsCreature), BindPermanent("member"), Sequence(
		GainLifeFor(You, FromNumber(n)), &eventAmountCapture{n, 9}))
	if err := loop.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 26 || FromNumber(n).resolveValue(ctx) != 3 {
		t.Fatal("loop did not isolate and restore number map")
	}
}

func TestDamageEventAmountAnyRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount := BindNumber("damage")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
		DealsDamage(This, amount), GainLifeFor(You, FromNumber(amount)))))
	source.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	other := NewCreature("Other", "", 1, 10)
	other.SetOwner(b.PlayerID())
	g.PutOnBattlefield(other, b.PlayerID())
	g.FireEvent(GameEvent{Type: EvtDamageDealt, SourceID: other.ID(), TargetID: a.PlayerID(), Amount: 9})
	g.FireEvent(GameEvent{Type: EvtLifeLost, SourceID: source.ID(), TargetID: a.PlayerID(), Amount: 9})
	g.DealDamageToPermanent(g.FindPermanent(other.ID()), 4, source.ID())
	g.DealDamageToPlayer(b, 2, source.ID())
	g.AddPreventionShield(b.PlayerID(), 3)
	g.DealDamageToPlayer(b, 3, source.ID())
	g.DealDamageToPlayer(b, 0, source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 26 || b.Life() != 18 {
		t.Fatal("damage amounts or filtering incorrect")
	}
}

func TestDamageEventAmountCopyRetarget(t *testing.T) {
	a, b := newRecordingPlayer("A"), newRecordingPlayer("B")
	g := NewGame(a, b)
	amount, target := BindNumber("damage"), DeclarePlayerTarget("recipient")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(
		DealsDamage(This, amount), GainLifeFor(target, FromNumber(amount)), WithTargets(target))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.DealDamageToPlayer(b, 3, source.ID())
	a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
	g.PutTriggersOnStack()
	original := g.StackPeek()
	if original == nil {
		t.Fatal("missing trigger")
	}
	b.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
	copied := g.CopyStackObjectDirect(original, b.PlayerID(), true)
	if copied == nil || copied.Targets[0] != b.PlayerID() || original.Targets[0] != a.PlayerID() {
		t.Fatal("targets were not independent")
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 20 {
		t.Fatal("retargeting changed the captured amount")
	}
}
