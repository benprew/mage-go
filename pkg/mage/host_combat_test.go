package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestHostCombatDefinition(t *testing.T) {
	for _, build := range []func(*PermanentBinding) EventPattern{AttachedAttacks, AttachedBlocks, AttachedAttacksUnblocked, AttachedAttacksOrBlocks} {
		for _, valid := range []bool{false, true} {
			var binding *PermanentBinding
			if valid {
				binding = BindPermanent("host")
			}
			c := NewAura("Watcher", "", WithAbility(WhenEvent(build(binding), TapSubject(From(binding)))))
			err := ValidateCardDefinition("Watcher", c)
			if (err == nil) != valid {
				t.Fatalf("valid=%t err=%v", valid, err)
			}
		}
	}
}

func TestHostCombatEventIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindPermanent("host")
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 8), b.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Watcher", "", WithAbility(WhenEvent(AttachedAttacksOrBlocks(saved), TapSubject(From(saved))))), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	pattern := aura.RuntimeAbilities[0].(*GenericTriggered)
	for _, kind := range []EventType{EvtDeclaredAttacker, EvtDeclaredBlocker} {
		event := GameEvent{Type: kind, SourceID: host.ID(), SourceIncarnation: host.incarnationID, Flag: true}
		if !pattern.CheckEventType(kind) || !pattern.CheckTrigger(&event, g) {
			t.Fatal("valid event rejected")
		}
		event.SourceIncarnation = uuid.New()
		if pattern.CheckTrigger(&event, g) {
			t.Fatal("stale event matched")
		}
	}
	if pattern.CheckEventType(EvtAttacksUnblocked) {
		t.Fatal("union matched an unblocked event")
	}
	event := GameEvent{Type: EvtDeclaredBlocker, SourceID: host.ID()}
	if pattern.CheckTrigger(&event, g) {
		t.Fatal("second combat pair triggered singular block event")
	}
	event.Type = EvtDeclaredAttacker
	g.PhaseOut(host.ID())
	if pattern.CheckTrigger(&event, g) {
		t.Fatal("phased host matched")
	}
}
