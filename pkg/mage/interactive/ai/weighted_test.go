package ai

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

// ── ToWeighted conversion ───────────────────────────────────────────────────

func TestToWeighted_AggroPersonality(t *testing.T) {
	wp := AggroPersonality.ToWeighted()
	if wp.Aggression != 1.0 {
		t.Errorf("Aggro.Aggression = %f, want 1.0", wp.Aggression)
	}
	if wp.HoldInstants != 0.0 {
		t.Errorf("Aggro.HoldInstants = %f, want 0.0", wp.HoldInstants)
	}
	if wp.TargetFace != 0.0 {
		t.Errorf("Aggro.TargetFace = %f, want 0.0", wp.TargetFace)
	}
	if wp.CurvePreference != 0.0 {
		t.Errorf("Aggro.CurvePreference = %f, want 0.0 (CheapestFirst)", wp.CurvePreference)
	}
	if wp.BlockThreshold != 0.3 {
		t.Errorf("Aggro.BlockThreshold = %f, want 0.3", wp.BlockThreshold)
	}
}

func TestToWeighted_ControlPersonality(t *testing.T) {
	wp := ControlPersonality.ToWeighted()
	if wp.Aggression != 0.0 {
		t.Errorf("Control.Aggression = %f, want 0.0", wp.Aggression)
	}
	if wp.HoldInstants != 1.0 {
		t.Errorf("Control.HoldInstants = %f, want 1.0", wp.HoldInstants)
	}
	if wp.BlockThreshold != 0.0 {
		t.Errorf("Control.BlockThreshold = %f, want 0.0", wp.BlockThreshold)
	}
	if wp.CurvePreference != 1.0 {
		t.Errorf("Control.CurvePreference = %f, want 1.0 (MostExpensiveFirst)", wp.CurvePreference)
	}
}

func TestToWeighted_BurnPersonality(t *testing.T) {
	wp := BurnPersonality.ToWeighted()
	if wp.Aggression != 1.0 {
		t.Errorf("Burn.Aggression = %f, want 1.0", wp.Aggression)
	}
	if wp.TargetFace != 1.0 {
		t.Errorf("Burn.TargetFace = %f, want 1.0", wp.TargetFace)
	}
	if wp.BlockThreshold != 0.95 {
		t.Errorf("Burn.BlockThreshold = %f, want 0.95", wp.BlockThreshold)
	}
}

func TestToWeighted_PreservesName(t *testing.T) {
	for _, p := range []Personality{AggroPersonality, ControlPersonality, MidrangePersonality, TempoPersonality, BurnPersonality} {
		wp := p.ToWeighted()
		if wp.Name != p.Name {
			t.Errorf("ToWeighted().Name = %q, want %q", wp.Name, p.Name)
		}
	}
}

// ── eval.WeightedEvaluator ──────────────────────────────────────────────────

func TestWeightedEvaluator_EmptyBoard(t *testing.T) {
	g, pa, _ := makeGame()
	evaluator := eval.WeightedEvaluator(MidrangeWeighted.Weights)
	got := evaluator(g, pa.PlayerID())
	if got != 0 {
		t.Errorf("WeightedEvaluator(empty) = %d, want 0", got)
	}
}

func TestWeightedEvaluator_LifeAdvantage(t *testing.T) {
	g, pa, pb := makeGame()
	pa.SetLife(25)
	pb.SetLife(15)
	evaluator := eval.WeightedEvaluator(AggroWeighted.Weights)
	got := evaluator(g, pa.PlayerID())
	if got != 10 {
		t.Errorf("Aggro weighted eval (life advantage) = %d, want 10", got)
	}
	evaluator2 := eval.WeightedEvaluator(ControlWeighted.Weights)
	got2 := evaluator2(g, pa.PlayerID())
	if got2 != 40 {
		t.Errorf("Control weighted eval (life advantage) = %d, want 40", got2)
	}
}

func TestWeightedEvaluator_BoardWeight(t *testing.T) {
	g, pa, _ := makeGame()
	perm := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(perm)

	evaluator := eval.WeightedEvaluator(AggroWeighted.Weights)
	got := evaluator(g, pa.PlayerID())
	if got != 9 {
		t.Errorf("Aggro weighted eval (creature) = %d, want 9", got)
	}

	evaluator2 := eval.WeightedEvaluator(ControlWeighted.Weights)
	got2 := evaluator2(g, pa.PlayerID())
	if got2 != 3 {
		t.Errorf("Control weighted eval (creature) = %d, want 3", got2)
	}
}

func TestWeightedEvaluator_CardAdvantage(t *testing.T) {
	g, pa, _ := makeGame()
	c1 := mage.NewCreature("Card", "{1}", 1, 1)
	c2 := mage.NewCreature("Card2", "{1}", 1, 1)
	pa.AddToHand(c1)
	pa.AddToHand(c2)

	evaluator := eval.WeightedEvaluator(ControlWeighted.Weights)
	got := evaluator(g, pa.PlayerID())
	if got != 6 {
		t.Errorf("Control weighted eval (card advantage) = %d, want 6", got)
	}
}

func TestWeightedEvaluator_TempoBonus(t *testing.T) {
	g, pa, _ := makeGame()
	land := mage.NewLand("Forest")
	land.SetOwner(pa.PlayerID())
	lp := mage.NewPermanent(land, pa.PlayerID())
	lp.RevokeBaseAttr(core.AttrSummonSick)
	g.AddToBattlefield(lp)

	evaluator := eval.WeightedEvaluator(TempoWeighted.Weights)
	got := evaluator(g, pa.PlayerID())
	if got != 5 {
		t.Errorf("Tempo weighted eval (land+tempo) = %d, want 5", got)
	}
}

func TestWeightedEvaluator_NilPlayer(t *testing.T) {
	g, _, _ := makeGame()
	evaluator := eval.WeightedEvaluator(MidrangeWeighted.Weights)
	got := evaluator(g, uuid.New())
	if got != 0 {
		t.Errorf("WeightedEvaluator(nil player) = %d, want 0", got)
	}
}

// ── eval.NewWeightedEvaluator (Phase 3: role-based) ────────────────────────

func TestNewWeightedEvaluator_EmptyBoard(t *testing.T) {
	g, pa, _ := makeGame()
	evaluator := eval.NewWeightedEvaluator(MidrangeWeighted.Weights)
	got := evaluator(g, pa.PlayerID())
	if got != 0 {
		t.Errorf("NewWeightedEvaluator(empty) = %d, want 0", got)
	}
}

func TestNewWeightedEvaluator_NilPlayer(t *testing.T) {
	g, _, _ := makeGame()
	evaluator := eval.NewWeightedEvaluator(MidrangeWeighted.Weights)
	got := evaluator(g, uuid.New())
	if got != 0 {
		t.Errorf("NewWeightedEvaluator(nil player) = %d, want 0", got)
	}
}

func TestNewWeightedEvaluator_LifeAdvantage(t *testing.T) {
	g, pa, pb := makeGame()
	pa.SetLife(25)
	pb.SetLife(15)
	evaluator := eval.NewWeightedEvaluator(MidrangeWeighted.Weights)
	got := evaluator(g, pa.PlayerID())
	if got < 15 {
		t.Errorf("NewWeightedEvaluator(life advantage) = %d, want >= 15", got)
	}
}

func TestNewWeightedEvaluator_AggroValuesCreatures(t *testing.T) {
	g, pa, _ := makeGame()
	perm := makePerm("Hill Giant", "{3}{R}", 3, 3, pa.PlayerID())
	g.AddToBattlefield(perm)

	aggro := eval.NewWeightedEvaluator(AggroWeighted.Weights)
	control := eval.NewWeightedEvaluator(ControlWeighted.Weights)

	aggroScore := aggro(g, pa.PlayerID())
	controlScore := control(g, pa.PlayerID())

	if aggroScore <= controlScore {
		t.Errorf("Aggro should value creatures more: aggro=%d, control=%d", aggroScore, controlScore)
	}
}

func TestNewWeightedEvaluator_ControlValuesHand(t *testing.T) {
	g, pa, _ := makeGame()
	for range 5 {
		c := mage.NewInstant("Spell", "{1}{U}", mage.NewSpellAbility(mage.DrawCards(mage.Fixed(1))))
		pa.AddToHand(c)
	}

	aggro := eval.NewWeightedEvaluator(AggroWeighted.Weights)
	control := eval.NewWeightedEvaluator(ControlWeighted.Weights)

	aggroScore := aggro(g, pa.PlayerID())
	controlScore := control(g, pa.PlayerID())

	if controlScore <= aggroScore {
		t.Errorf("Control should value hand more: control=%d, aggro=%d", controlScore, aggroScore)
	}
}

func TestNewWeightedEvaluator_TempoValuesUntappedMana(t *testing.T) {
	g, pa, _ := makeGame()
	for range 5 {
		land := mage.NewLand("Island")
		land.SetOwner(pa.PlayerID())
		lp := mage.NewPermanent(land, pa.PlayerID())
		lp.RevokeBaseAttr(core.AttrSummonSick)
		g.AddToBattlefield(lp)
	}

	tempo := eval.NewWeightedEvaluator(TempoWeighted.Weights)
	aggro := eval.NewWeightedEvaluator(AggroWeighted.Weights)

	tempoScore := tempo(g, pa.PlayerID())
	aggroScore := aggro(g, pa.PlayerID())

	if tempoScore <= aggroScore {
		t.Errorf("Tempo should value untapped mana more: tempo=%d, aggro=%d", tempoScore, aggroScore)
	}
}

// ── eval.EvalCreature with ability quality ─────────────────────────────────

func TestEvalCreature_PingerVsVanilla(t *testing.T) {
	pinger := makePerm("Prodigal Sorcerer", "{2}{U}", 1, 1, uuid.New(),
		mage.WithActivatedAbility(mage.DealDamage(mage.Fixed(1)), mage.Tap(),
			mage.WithTarget(mage.TargetDamageAnyTarget())),
	)
	vanilla := makePerm("Bear", "{1}{G}", 1, 1, uuid.New())

	pingerScore := eval.EvalCreature(pinger)
	vanillaScore := eval.EvalCreature(vanilla)

	if pingerScore <= vanillaScore {
		t.Errorf("Pinger(%d) should score higher than vanilla 1/1(%d)", pingerScore, vanillaScore)
	}
	if pingerScore-vanillaScore < 3 {
		t.Errorf("Pinger advantage(%d) should be at least 3", pingerScore-vanillaScore)
	}
}

func TestEvalCreature_DrawCreatureVsVanilla(t *testing.T) {
	drawer := makePerm("Draw Engine", "{2}{U}", 1, 1, uuid.New(),
		mage.WithActivatedAbility(mage.DrawCards(mage.Fixed(1)), mage.Tap()),
	)
	vanilla := makePerm("Bear", "{1}{G}", 1, 1, uuid.New())

	drawerScore := eval.EvalCreature(drawer)
	vanillaScore := eval.EvalCreature(vanilla)

	if drawerScore <= vanillaScore {
		t.Errorf("Draw creature(%d) should score higher than vanilla(%d)", drawerScore, vanillaScore)
	}
	if drawerScore-vanillaScore < 4 {
		t.Errorf("Draw creature advantage(%d) should be at least 4", drawerScore-vanillaScore)
	}
}

// ── eval.AbilityQuality ─────────────────────────────────────────────────────

func TestAbilityQuality_TapToDealDamage(t *testing.T) {
	ab := mage.NewActivatedAbility(mage.DealDamage(mage.Fixed(1)), mage.Tap(),
		mage.WithTarget(mage.TargetDamageAnyTarget()))
	got := eval.AbilityQuality(ab)
	if got != 4 {
		t.Errorf("AbilityQuality(tap to ping) = %d, want 4", got)
	}
}

func TestAbilityQuality_TapToDraw(t *testing.T) {
	ab := mage.NewActivatedAbility(mage.DrawCards(mage.Fixed(1)), mage.Tap())
	got := eval.AbilityQuality(ab)
	if got != 5 {
		t.Errorf("AbilityQuality(tap to draw) = %d, want 5", got)
	}
}

func TestAbilityQuality_ExpensiveDraw(t *testing.T) {
	ab := mage.NewActivatedAbility(mage.DrawCards(mage.Fixed(1)), mage.ManaCostOf("{3}{U}"),
		mage.WithCost(mage.Tap()))
	got := eval.AbilityQuality(ab)
	if got != 3 {
		t.Errorf("AbilityQuality(expensive draw) = %d, want 3", got)
	}
}

func TestAbilityQuality_ExpensivePump(t *testing.T) {
	ab := mage.NewActivatedAbility(
		mage.FuncEffect("pump", mage.EffectProperties{Outcome: mage.OutcomeBenefit},
			func(g *mage.Game, s, c uuid.UUID, t []uuid.UUID) error { return nil }),
		mage.ManaCostOf("{5}"),
	)
	got := eval.AbilityQuality(ab)
	if got != 1 {
		t.Errorf("AbilityQuality(expensive pump) = %d, want 1", got)
	}
}

func TestAbilityQuality_CheapBenefit(t *testing.T) {
	ab := mage.NewActivatedAbility(
		mage.FuncEffect("buff", mage.EffectProperties{Outcome: mage.OutcomeBenefit},
			func(g *mage.Game, s, c uuid.UUID, t []uuid.UUID) error { return nil }),
		mage.ManaCostOf("{1}"),
	)
	got := eval.AbilityQuality(ab)
	if got != 2 {
		t.Errorf("AbilityQuality(cheap benefit) = %d, want 2", got)
	}
}

func TestAbilityQuality_FreeTapAbility(t *testing.T) {
	ab := mage.NewActivatedAbility(
		mage.FuncEffect("tap effect", mage.EffectProperties{},
			func(g *mage.Game, s, c uuid.UUID, t []uuid.UUID) error { return nil }),
		mage.Tap(),
	)
	got := eval.AbilityQuality(ab)
	if got != 2 {
		t.Errorf("AbilityQuality(free tap) = %d, want 2", got)
	}
}

func TestAbilityQuality_PingerBeatsPump(t *testing.T) {
	pinger := mage.NewActivatedAbility(mage.DealDamage(mage.Fixed(1)), mage.Tap(),
		mage.WithTarget(mage.TargetDamageAnyTarget()))
	pump := mage.NewActivatedAbility(
		mage.FuncEffect("pump", mage.EffectProperties{Outcome: mage.OutcomeBenefit},
			func(g *mage.Game, s, c uuid.UUID, t []uuid.UUID) error { return nil }),
		mage.ManaCostOf("{5}"),
	)

	pingerQ := eval.AbilityQuality(pinger)
	pumpQ := eval.AbilityQuality(pump)

	if pingerQ <= pumpQ {
		t.Errorf("Pinger(%d) should beat expensive pump(%d)", pingerQ, pumpQ)
	}
}

// ── DefaultEvaluator integration ────────────────────────────────────────────

func TestDefaultEvaluator_LethalBonusIntegrated(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(3)
	perm := makePerm("Giant", "{3}{R}", 5, 5, pa.PlayerID())
	g.AddToBattlefield(perm)

	score := eval.DefaultEvaluator(g, pa.PlayerID())
	if score < eval.LethalBonus {
		t.Errorf("DefaultEvaluator with lethal should include LethalBonus, got %d", score)
	}
}

func TestDefaultEvaluator_UntappedManaSourcesIncludeManaCreatures(t *testing.T) {
	g1, pa1, _ := makeGame()
	g2, pa2, _ := makeGame()

	land := mage.NewLand("Forest")
	land.SetOwner(pa1.PlayerID())
	lp := mage.NewPermanent(land, pa1.PlayerID())
	lp.RevokeBaseAttr(core.AttrSummonSick)
	g1.AddToBattlefield(lp)

	land2 := mage.NewLand("Forest")
	land2.SetOwner(pa2.PlayerID())
	lp2 := mage.NewPermanent(land2, pa2.PlayerID())
	lp2.RevokeBaseAttr(core.AttrSummonSick)
	g2.AddToBattlefield(lp2)

	elf := makePerm("Llanowar Elves", "{G}", 1, 1, pa2.PlayerID(), mage.WithManaAbility(core.Green))
	g2.AddToBattlefield(elf)

	score1 := eval.DefaultEvaluator(g1, pa1.PlayerID())
	score2 := eval.DefaultEvaluator(g2, pa2.PlayerID())

	if score2 <= score1 {
		t.Errorf("Board with mana elf (%d) should score higher than land only (%d)", score2, score1)
	}
}
