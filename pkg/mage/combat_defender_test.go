package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombatDefendingPlayerQuery(t *testing.T) {
	g, a, b := newSpellCopyGame()
	ref := CombatDefendingPlayer()
	target := DeclarePermanentTarget("defender creature", IsCreature, TargetControlledBy(ref))
	card := NewInstant("Probe", "", Spell(TapSubject(target), WithTargets(target)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 4), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 4), b.PlayerID())
	for active := range 2 {
		g.SetActivePlayerIndex(active)
		for _, step := range core.AllSteps() {
			g.SetStep(step)
			for _, caster := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
				ctx := &EffectContext{Game: g, Controller: caster}
				id, ok := ref.resolvePlayer(ctx)
				inCombat := step >= core.BeginCombat && step <= core.EndCombat
				if ok != inCombat {
					t.Fatalf("step %v: available=%v", step, ok)
				}
				wantID, wantPermanent := b.PlayerID(), second.ID()
				if active == 1 {
					wantID, wantPermanent = a.PlayerID(), first.ID()
				}
				possible := target.Possible(caster, card, g)
				if inCombat && (id != wantID || !slices.Equal(possible, []uuid.UUID{wantPermanent})) {
					t.Fatal("defender followed the caster or missing attackers", step, id, possible)
				}
				if !inCombat && (id != uuid.Nil || len(possible) != 0) {
					t.Fatal("defender outside combat", step, id, possible)
				}
				if recipient, available := ref.resolveRecipient(ctx); recipient != id || available != ok {
					t.Fatal("recipient and player queries differ")
				}
			}
		}
	}
	g.SetStep(core.DeclareBlockers)
	g.SetActivePlayerIndex(0)
	clone := g.Clone()
	clone.SetActivePlayerIndex(1)
	if id, _ := ref.resolvePlayer(&EffectContext{Game: g}); id != b.PlayerID() {
		t.Fatal("clone changed the live defender")
	}
	if id, _ := ref.resolvePlayer(&EffectContext{Game: clone}); id != a.PlayerID() {
		t.Fatal("clone does not use its own turn")
	}
	if first.Tapped || second.Tapped || g.StackSize() != 0 {
		t.Fatal("query changed game state")
	}
}

func TestCombatDefendingPlayerMissing(t *testing.T) {
	g, _, _ := newSpellCopyGame()
	g.SetStep(core.BeginCombat)
	g.turns.SetActivePlayerIndex(-1)
	if id, ok := CombatDefendingPlayer().resolvePlayer(&EffectContext{Game: g}); ok || id != uuid.Nil {
		t.Fatal("missing active player supplied a defender")
	}
	g.turns.SetActivePlayerIndex(0)
	g.players = g.players[:1]
	if id, ok := CombatDefendingPlayer().resolvePlayer(&EffectContext{Game: g}); ok || id != uuid.Nil {
		t.Fatal("one player supplied a defender")
	}
	g.players = nil
	if id, ok := CombatDefendingPlayer().resolvePlayer(&EffectContext{Game: g}); ok || id != uuid.Nil {
		t.Fatal("missing players supplied a defender")
	}
}
