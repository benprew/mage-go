package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type triggerOrderingTestPlayer struct {
	*BasePlayer
	orderFunc func(triggers []TriggerOrderInfo) []int
}

func (p *triggerOrderingTestPlayer) ChooseTriggerOrder(triggers []TriggerOrderInfo) []int {
	if p.orderFunc != nil {
		return p.orderFunc(triggers)
	}
	return nil
}

func TestCR603_3b_SimultaneousTriggerPlayerChoice(t *testing.T) {
	var resolutionOrder []string

	pA := &triggerOrderingTestPlayer{
		BasePlayer: NewBasePlayer("Alice"),
		orderFunc: func(triggers []TriggerOrderInfo) []int {
			// To have Card A resolve first, it must be put on top of the stack (placed second).
			// So put Card B on stack first, then Card A.
			res := make([]int, len(triggers))
			for i, trig := range triggers {
				if trig.SourceName == "Card A" {
					res[1] = i
				} else {
					res[0] = i
				}
			}
			return res
		},
	}
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	effectA := FuncEffect("effect A", EffectProperties{}, func(_ *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		resolutionOrder = append(resolutionOrder, "Card A")
		return nil
	})
	effectB := FuncEffect("effect B", EffectProperties{}, func(_ *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		resolutionOrder = append(resolutionOrder, "Card B")
		return nil
	})

	cardA := NewCreature("Card A", "{G}", 1, 1,
		WithAbility(WheneverSpellCastTrigger(effectA, false)),
	)
	cardB := NewCreature("Card B", "{G}", 1, 1,
		WithAbility(WheneverSpellCastTrigger(effectB, false)),
	)

	g.PutOnBattlefield(cardA, pA.PlayerID())
	g.PutOnBattlefield(cardB, pA.PlayerID())

	// Fire EvtSpellCast which triggers both
	g.FireEvent(GameEvent{
		Type:     EvtSpellCast,
		PlayerID: pA.PlayerID(),
	})

	g.ResolveStack()

	if len(resolutionOrder) != 2 {
		t.Fatalf("expected 2 resolved triggers, got %d", len(resolutionOrder))
	}
	if resolutionOrder[0] != "Card A" || resolutionOrder[1] != "Card B" {
		t.Fatalf("expected resolution order [Card A, Card B], got %v", resolutionOrder)
	}
}

func TestCR603_3b_APNAPOrder(t *testing.T) {
	var resolutionOrder []string

	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	effectA := FuncEffect("effect A", EffectProperties{}, func(_ *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		resolutionOrder = append(resolutionOrder, "Alice")
		return nil
	})
	effectB := FuncEffect("effect B", EffectProperties{}, func(_ *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		resolutionOrder = append(resolutionOrder, "Bob")
		return nil
	})

	cardA := NewCreature("Alice Card", "{G}", 1, 1,
		WithAbility(WheneverSpellCastTrigger(effectA, false)),
	)
	cardB := NewCreature("Bob Card", "{G}", 1, 1,
		WithAbility(WheneverSpellCastTrigger(effectB, false)),
	)

	g.PutOnBattlefield(cardA, pA.PlayerID())
	g.PutOnBattlefield(cardB, pB.PlayerID())

	// Active player is Alice (index 0)
	g.FireEvent(GameEvent{
		Type:     EvtSpellCast,
		PlayerID: pA.PlayerID(),
	})

	g.ResolveStack()

	if len(resolutionOrder) != 2 {
		t.Fatalf("expected 2 resolved triggers, got %d", len(resolutionOrder))
	}
	// Under APNAP, Alice (AP) puts her trigger on the stack first; Bob (NAP) puts his on top.
	// Bob's trigger resolves first.
	if resolutionOrder[0] != "Bob" || resolutionOrder[1] != "Alice" {
		t.Fatalf("expected APNAP resolution order [Bob, Alice], got %v", resolutionOrder)
	}
}

func TestCR603_3b_InvalidPermutationFallback(t *testing.T) {
	var resolutionOrder []string

	pA := &triggerOrderingTestPlayer{
		BasePlayer: NewBasePlayer("Alice"),
		orderFunc: func(triggers []TriggerOrderInfo) []int {
			// Return invalid permutation (duplicate indices)
			return []int{0, 0}
		},
	}
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	effect1 := FuncEffect("effect 1", EffectProperties{}, func(_ *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		resolutionOrder = append(resolutionOrder, "First")
		return nil
	})
	effect2 := FuncEffect("effect 2", EffectProperties{}, func(_ *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		resolutionOrder = append(resolutionOrder, "Second")
		return nil
	})

	card1 := NewCreature("First", "{G}", 1, 1,
		WithAbility(WheneverSpellCastTrigger(effect1, false)),
	)
	card2 := NewCreature("Second", "{G}", 1, 1,
		WithAbility(WheneverSpellCastTrigger(effect2, false)),
	)

	g.PutOnBattlefield(card1, pA.PlayerID())
	g.PutOnBattlefield(card2, pA.PlayerID())

	g.FireEvent(GameEvent{
		Type:     EvtSpellCast,
		PlayerID: pA.PlayerID(),
	})

	g.ResolveStack()

	if len(resolutionOrder) != 2 {
		t.Fatalf("expected 2 resolved triggers, got %d", len(resolutionOrder))
	}
	// Engine default reverses order of event generation: Second is put on stack first, First on top -> First resolves first
	if resolutionOrder[0] != "First" || resolutionOrder[1] != "Second" {
		t.Fatalf("expected fallback resolution order [First, Second], got %v", resolutionOrder)
	}
}
