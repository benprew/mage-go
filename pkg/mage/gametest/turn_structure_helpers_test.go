// Package gametest: shared helpers for turn-structure combat tests.
// The combatEventRecorder and its associated card registration are used by
// both the combat-phase general tests (CR 506.1) and the declare-attackers
// tests (CR 508.8), so they live here to avoid duplication.
package gametest

import (
	"sync"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// combatEventRecorder captures combat-phase events in the order they fire.
// Tests reset it before use and register (once) creatures whose triggers append
// their event type into the shared slice.
type combatEventRecorder struct {
	mu     sync.Mutex
	events []core.EventType
}

func (r *combatEventRecorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = nil
}

func (r *combatEventRecorder) record(et core.EventType) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, et)
}

func (r *combatEventRecorder) snapshot() []core.EventType {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]core.EventType, len(r.events))
	copy(out, r.events)
	return out
}

var combatRec = &combatEventRecorder{}

// registerCombatEventRecorder registers a vanilla 1/1 creature whose triggers
// append each of the four combat-phase events into combatRec. Registering once
// per process is sufficient since the factory closes over combatRec (a pointer
// to a package-level recorder).
func registerCombatEventRecorder() {
	name := "Combat Event Recorder"
	if mage.CardRegistered(name) {
		return
	}
	mage.Register(name, func() mage.Card {
		recEffect := func(et core.EventType) mage.Effect {
			return mage.FuncEffect("record combat event",
				mage.EffectProperties{},
				func(_ *mage.Game, _, _ uuid.UUID, _ []uuid.UUID) error {
					combatRec.record(et)
					return nil
				})
		}
		return mage.NewCreature(name, "{1}", 1, 1,
			mage.WithSubTypes("Spirit"),
			mage.WithAbility(mage.NewTriggered(core.EvtBeginCombat, false,
				recEffect(core.EvtBeginCombat))),
			mage.WithAbility(mage.NewTriggered(core.EvtDeclaredAttacker, false,
				recEffect(core.EvtDeclaredAttacker)).
				SetCondition(func(evt *core.GameEvent, _ mage.GameReader, sourceID, _ uuid.UUID) bool {
					return evt.SourceID == sourceID
				})),
			mage.WithAbility(mage.NewTriggered(core.EvtBlockersDecl, false,
				recEffect(core.EvtBlockersDecl))),
			mage.WithAbility(mage.NewTriggered(core.EvtEndOfCombat, false,
				recEffect(core.EvtEndOfCombat))),
		)
	})
}
