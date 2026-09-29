package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCR613_4_Layer7Sublayers_CDAAndSetBase(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	// Creature with CDA: P/T = number of lands controlled (CR 613.4a: Layer 7a)
	cdaCreature := NewCreature("CDA Creature", "{3}", 0, 0,
		WithStaticAbility(PTEqualsControlledCount(IsLand)),
	)
	perm := g.PutOnBattlefield(cdaCreature, pA.PlayerID())

	// Put 4 lands on the battlefield under Alice
	for range 4 {
		g.PutOnBattlefield(NewLand("Forest"), pA.PlayerID())
	}

	g.ApplyContinuousEffects()

	// Initial P/T from Layer 7a CDA should be 4/4
	if perm.CurrentPower(g) != 4 || perm.CurrentToughness(g) != 4 {
		t.Fatalf("expected 4/4 from CDA, got %d/%d", perm.CurrentPower(g), perm.CurrentToughness(g))
	}

	// Now apply an effect setting base P/T to 0/2 (CR 613.4b: Layer 7b)
	// (Simulating Sorceress Queen)
	setEffect := TargetEffectWithSublayer(LayerPT, SublayerPT_Set, EndOfTurn, perm.ID(), func(_ *Game, target *Permanent) error {
		target.BasePTOverride = &[2]int{0, 2}
		return nil
	})
	g.AddContinuousEffect(setEffect)
	g.ApplyContinuousEffects()

	// Layer 7b overrides Layer 7a CDA! Must be 0/2
	if perm.CurrentPower(g) != 0 || perm.CurrentToughness(g) != 2 {
		t.Fatalf("expected 0/2 after Layer 7b set base P/T, got %d/%d", perm.CurrentPower(g), perm.CurrentToughness(g))
	}

	// Add a Layer 7c modification (+3/+3, e.g. Giant Growth)
	modEffect := TargetEffectWithSublayer(LayerPT, SublayerPT_Modify, EndOfTurn, perm.ID(), func(_ *Game, target *Permanent) error {
		target.powerBonus += 3
		target.toughBonus += 3
		return nil
	})
	g.AddContinuousEffect(modEffect)
	g.ApplyContinuousEffects()

	// Layer 7b (0/2) + Layer 7c (+3/+3) = 3/5
	if perm.CurrentPower(g) != 3 || perm.CurrentToughness(g) != 5 {
		t.Fatalf("expected 3/5 after Layer 7c modification, got %d/%d", perm.CurrentPower(g), perm.CurrentToughness(g))
	}
}

func TestCR613_5_GrayOgreExample(t *testing.T) {
	// CR 613.5 Example:
	// Gray Ogre (2/2).
	// An effect puts a +1/+1 counter on it (layer 7c), making it 3/3.
	// A spell gives +4/+4 until end of turn (layer 7c), making it 7/7.
	// An enchantment gives +0/+2 (layer 7c), making it 7/9.
	// An effect says "Target creature becomes 0/1 until end of turn" (layer 7b).
	// Final: 5/8 (0/1 with +4/+4 from spell, +0/+2 from enchantment, +1/+1 counter).
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	ogre := NewCreature("Gray Ogre", "{2}{R}", 2, 2)
	perm := g.PutOnBattlefield(ogre, pA.PlayerID())

	// +1/+1 counter
	perm.AddCounter(P1P1, 1)

	// +4/+4 until end of turn (Layer 7c)
	g.AddContinuousEffect(TargetEffectWithSublayer(LayerPT, SublayerPT_Modify, EndOfTurn, perm.ID(), func(_ *Game, target *Permanent) error {
		target.powerBonus += 4
		target.toughBonus += 4
		return nil
	}))

	// +0/+2 static/continuous (Layer 7c)
	g.AddContinuousEffect(TargetEffectWithSublayer(LayerPT, SublayerPT_Modify, EndOfTurn, perm.ID(), func(_ *Game, target *Permanent) error {
		target.toughBonus += 2
		return nil
	}))

	// Set base 0/1 until end of turn (Layer 7b)
	g.AddContinuousEffect(TargetEffectWithSublayer(LayerPT, SublayerPT_Set, EndOfTurn, perm.ID(), func(_ *Game, target *Permanent) error {
		target.BasePTOverride = &[2]int{0, 1}
		return nil
	}))

	g.ApplyContinuousEffects()

	if perm.CurrentPower(g) != 5 || perm.CurrentToughness(g) != 8 {
		t.Fatalf("expected 5/8 for CR 613.5 Gray Ogre example, got %d/%d", perm.CurrentPower(g), perm.CurrentToughness(g))
	}
}

func TestCR613_4d_SwitchPowerToughness(t *testing.T) {
	// CR 613.4d Example:
	// A 1/3 creature is given +0/+1 by an effect.
	// Then another effect switches the creature's power and toughness -> 4/1.
	// A new effect gives the creature +5/+0. Unswitched is 6/4, actual is 4/6.
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	creature := NewCreature("Test Creature", "{1}", 1, 3)
	perm := g.PutOnBattlefield(creature, pA.PlayerID())

	// +0/+1 effect (Layer 7c)
	g.AddContinuousEffect(TargetEffectWithSublayer(LayerPT, SublayerPT_Modify, EndOfTurn, perm.ID(), func(_ *Game, target *Permanent) error {
		target.toughBonus++
		return nil
	}))

	// Switch effect (Layer 7d)
	switchEff1 := TargetEffectWithSublayer(LayerPT, SublayerPT_Switch, EndOfTurn, perm.ID(), func(_ *Game, target *Permanent) error {
		target.SwitchPowerToughness()
		return nil
	})
	g.AddContinuousEffect(switchEff1)

	g.ApplyContinuousEffects()
	if perm.CurrentPower(g) != 4 || perm.CurrentToughness(g) != 1 {
		t.Fatalf("expected 4/1 after switch, got %d/%d", perm.CurrentPower(g), perm.CurrentToughness(g))
	}

	// +5/+0 effect (Layer 7c)
	g.AddContinuousEffect(TargetEffectWithSublayer(LayerPT, SublayerPT_Modify, EndOfTurn, perm.ID(), func(_ *Game, target *Permanent) error {
		target.powerBonus += 5
		return nil
	}))

	g.ApplyContinuousEffects()
	// Unswitched: power = 1 + 5 = 6, toughness = 3 + 1 = 4.
	// Switched: power = 4, toughness = 6.
	if perm.CurrentPower(g) != 4 || perm.CurrentToughness(g) != 6 {
		t.Fatalf("expected 4/6 with +5/+0 and switch, got %d/%d", perm.CurrentPower(g), perm.CurrentToughness(g))
	}

	// Second switch effect (Layer 7d): two switches cancel each other out -> 6/4 (CR 613.4d Example 3)
	switchEff2 := TargetEffectWithSublayer(LayerPT, SublayerPT_Switch, EndOfTurn, perm.ID(), func(_ *Game, target *Permanent) error {
		target.SwitchPowerToughness()
		return nil
	})
	g.AddContinuousEffect(switchEff2)
	g.ApplyContinuousEffects()

	if perm.CurrentPower(g) != 6 || perm.CurrentToughness(g) != 4 {
		t.Fatalf("expected 6/4 after double switch (cancelled), got %d/%d", perm.CurrentPower(g), perm.CurrentToughness(g))
	}
}

type testDependentEffect struct {
	ContinuousEffect
	dependsOnID uuid.UUID
}

func (e *testDependentEffect) DependsOn(other ContinuousEffect, _ *Game) bool {
	return other.SourceID() == e.dependsOnID
}

func TestCR613_8_ContinuousEffectDependency(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	var applyOrder []string

	sourceA := uuid.New()
	sourceB := uuid.New()

	// Effect B is created first (earlier timestamp), but depends on Effect A.
	effB := &testDependentEffect{
		ContinuousEffect: FuncContinuousEffect(LayerColor, EndOfTurn, func(_ *Game, _ uuid.UUID) error {
			applyOrder = append(applyOrder, "B")
			return nil
		}),
		dependsOnID: sourceA,
	}
	effB.SetSourceID(sourceB)

	// Effect A is created second (later timestamp).
	effA := FuncContinuousEffect(LayerColor, EndOfTurn, func(_ *Game, _ uuid.UUID) error {
		applyOrder = append(applyOrder, "A")
		return nil
	})
	effA.SetSourceID(sourceA)

	// Register B first (earlier timestamp), then A (later timestamp)
	g.AddContinuousEffect(effB)
	g.AddContinuousEffect(effA)

	applyOrder = nil
	g.ApplyContinuousEffects()

	// Under CR 613.8b: B depends on A, so A must be applied before B!
	if len(applyOrder) != 2 || applyOrder[0] != "A" || applyOrder[1] != "B" {
		t.Fatalf("expected dependency order [A, B], got %v", applyOrder)
	}
}

func TestCR613_8b_DependencyCycleRevertsToTimestamp(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	var applyOrder []string

	source1 := uuid.New()
	source2 := uuid.New()

	// Effect 1 depends on Effect 2
	eff1 := &testDependentEffect{
		ContinuousEffect: FuncContinuousEffect(LayerColor, EndOfTurn, func(_ *Game, _ uuid.UUID) error {
			applyOrder = append(applyOrder, "1")
			return nil
		}),
		dependsOnID: source2,
	}
	eff1.SetSourceID(source1)

	// Effect 2 depends on Effect 1 (cycle!)
	eff2 := &testDependentEffect{
		ContinuousEffect: FuncContinuousEffect(LayerColor, EndOfTurn, func(_ *Game, _ uuid.UUID) error {
			applyOrder = append(applyOrder, "2")
			return nil
		}),
		dependsOnID: source1,
	}
	eff2.SetSourceID(source2)

	g.AddContinuousEffect(eff1)
	g.AddContinuousEffect(eff2)

	applyOrder = nil
	g.ApplyContinuousEffects()

	// CR 613.8b: "If several dependent effects form a dependency loop, then this rule is ignored
	// and the effects in the dependency loop are applied in timestamp order."
	// Since eff1 was registered before eff2, order must be [1, 2].
	if len(applyOrder) != 2 || applyOrder[0] != "1" || applyOrder[1] != "2" {
		t.Fatalf("expected timestamp order [1, 2] on dependency cycle, got %v", applyOrder)
	}
}
