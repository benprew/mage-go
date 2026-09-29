package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// CR 613.1c: Layer 3 text-changing continuous effects must apply in layer order.
func TestCR613_LayerText(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := NewCreature("Bear", "{1}{G}", 2, 2)
	g.PutOnBattlefield(p, a.PlayerID())

	applied := false
	eff := FuncContinuousEffect(LayerText, WhileOnBattlefield, func(game *Game, srcID uuid.UUID) error {
		applied = true
		return nil
	})
	eff.SetSourceID(p.ID())
	g.AddContinuousEffect(eff)
	g.effects.Apply(g)

	if !applied {
		t.Fatal("LayerText continuous effect was not applied")
	}
}

// CR 502.1: Phased-out permanents controlled by the active player phase in during the untap step,
// and if tapped, they untap along with other permanents. Attached permanents phase in with their host (CR 702.26g).
func TestCR502_1_PhasingInDuringUntap(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.turns.SetActivePlayerIndex(0) // Player A's turn

	creature := NewCreature("Knight", "{1}{W}", 2, 2)
	aura := NewEnchantment("Holy Armor", "{W}")
	aura.subTypes = []string{"Aura"}

	g.PutOnBattlefield(creature, a.PlayerID())
	g.PutOnBattlefield(aura, a.PlayerID())
	g.Attach(aura.ID(), creature.ID())

	// Tap the creature, then phase it out (aura phases out indirectly).
	cPerm := g.MutablePermanent(creature.ID())
	cPerm.Tapped = true
	g.PhaseOut(creature.ID())

	if !g.FindPermanentIncludingPhased(creature.ID()).PhasedOut {
		t.Fatal("creature should be phased out")
	}
	if !g.FindPermanentIncludingPhased(aura.ID()).PhasedOut {
		t.Fatal("aura should be indirectly phased out")
	}

	// Run untap step.
	g.doUntap()

	cAfter := g.FindPermanent(creature.ID())
	if cAfter == nil || cAfter.PhasedOut {
		t.Fatal("creature should have phased in during untap step")
	}
	if cAfter.Tapped {
		t.Fatal("creature should have untapped during untap step")
	}

	aAfter := g.FindPermanent(aura.ID())
	if aAfter == nil || aAfter.PhasedOut {
		t.Fatal("aura should have phased in with its host")
	}
}

// CR 702.36a: Fear checks effective continuous colors, not just printed mana cost colors.
func TestCR702_36a_FearEffectiveColor(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)

	fearAttacker := NewCreature("Fear Monster", "{2}{B}", 2, 2, WithKeyword(Fear))
	greenBlocker := NewCreature("Bear", "{1}{G}", 2, 2)
	blackBlocker := NewCreature("Zombie", "{1}{B}", 2, 2)

	g.PutOnBattlefield(fearAttacker, a.PlayerID())
	g.PutOnBattlefield(greenBlocker, b.PlayerID())
	g.PutOnBattlefield(blackBlocker, b.PlayerID())

	// Normally green blocker cannot block fear attacker.
	if CanBlock(g.FindPermanent(greenBlocker.ID()), g.FindPermanent(fearAttacker.ID()), g) {
		t.Fatal("green creature should not be able to block fear attacker")
	}

	// Change green creature's color to Black.
	g.ChangeObjectColor(greenBlocker.ID(), Black)
	g.effects.Apply(g)

	if !slices.Contains(g.FindPermanent(greenBlocker.ID()).Colors(), Black) {
		t.Fatal("blocker should effectively be black")
	}
	if !CanBlock(g.FindPermanent(greenBlocker.ID()), g.FindPermanent(fearAttacker.ID()), g) {
		t.Fatal("green creature changed to black should now be able to block fear creature")
	}

	// Change black creature's color to Red.
	g.ChangeObjectColor(blackBlocker.ID(), Red)
	g.effects.Apply(g)

	if slices.Contains(g.FindPermanent(blackBlocker.ID()).Colors(), Black) {
		t.Fatal("blocker should no longer be black")
	}
	if CanBlock(g.FindPermanent(blackBlocker.ID()), g.FindPermanent(fearAttacker.ID()), g) {
		t.Fatal("black creature changed to red should no longer be able to block fear creature")
	}
}

// CR 701.5a: CounterStackObject should counter abilities on the stack by ID.
func TestCR701_5a_CounterAbilityOnStack(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)

	abilityObj := &StackObject{
		ID:         uuid.New(),
		Controller: a.PlayerID(),
		SourceID:   uuid.New(),
		IsAbility:  true,
		Effects: []Effect{
			FuncEffect("test effect", EffectProperties{}, func(g *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
				t.Fatal("countered ability should not resolve")
				return nil
			}),
		},
	}
	g.PushStack(abilityObj)
	if g.stack.Size() != 1 {
		t.Fatalf("stack size = %d, want 1", g.stack.Size())
	}

	g.CounterStackObject(abilityObj.ID)
	if g.stack.Size() != 0 {
		t.Fatalf("ability should have been removed from stack, size = %d", g.stack.Size())
	}
}

// CR 506.4: A permanent is removed from combat if its controller changes.
func TestCR506_4_RemoveFromCombatOnControlChange(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)

	attacker := NewCreature("Attacker", "{1}{R}", 2, 2)
	g.PutOnBattlefield(attacker, a.PlayerID())

	g.combat.AddAttacker(attacker.ID(), b.PlayerID())
	if !g.combat.IsAttacking(attacker.ID()) {
		t.Fatal("attacker should be in combat")
	}

	// Change controller in Layer 2.
	eff := TargetEffect(LayerControl, EndOfTurn, attacker.ID(), func(g *Game, target *Permanent) error {
		target.computedController = b.PlayerID()
		return nil
	})
	g.AddContinuousEffect(eff)
	g.effects.Apply(g)

	if g.combat.IsAttacking(attacker.ID()) {
		t.Fatal("creature whose control changed should be removed from combat (CR 506.4)")
	}
}

// CR 120.3 & CR 704.5h: Deathtouch damage marks actual damage, not inflated toughness.
// A creature receiving deathtouch damage has that damage marked and is destroyed by SBA,
// even if toughness is boosted before SBAs check.
func TestCR704_5h_DeathtouchDamageIntegrity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)

	snake := NewCreature("Snake", "{G}", 1, 1, WithKeyword(Deathtouch))
	g.PutOnBattlefield(snake, a.PlayerID())

	giant := NewCreature("Colossus", "{8}", 10, 10)
	giantPerm := g.PutOnBattlefield(giant, b.PlayerID())

	// Deal 1 deathtouch damage to 10/10.
	g.DealDamageToPermanent(giantPerm, 1, snake.ID())

	// Marked damage must be exactly 1 per CR 120.3, NOT 10.
	giantPerm = g.FindPermanent(giant.ID())
	if giantPerm.Damage != 1 {
		t.Fatalf("expected marked damage to be 1 per CR 120.3, got %d", giantPerm.Damage)
	}
	if !giantPerm.DeathtouchDamage {
		t.Fatal("expected DeathtouchDamage flag to be true")
	}

	// Instant-speed toughness boost before SBAs: 10/10 becomes 10/13.
	giantPerm.BoostPT(0, 3)
	if giantPerm.CurrentToughness(g) != 13 {
		t.Fatalf("toughness = %d, want 13", giantPerm.CurrentToughness(g))
	}
	if giantPerm.Damage != 1 {
		t.Fatalf("damage changed to %d, want 1", giantPerm.Damage)
	}

	// SBA check destroys the creature via CR 704.5h despite having 1 damage on a 13-toughness creature.
	g.CheckStateBasedActions()
	if g.FindPermanent(giant.ID()) != nil {
		t.Fatal("creature dealt deathtouch damage should be destroyed by SBA 704.5h")
	}
}
