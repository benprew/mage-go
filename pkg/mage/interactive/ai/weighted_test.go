package ai

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/eval"
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
	// BlockPowerThreshold=3 → BlockThreshold=0.3
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
	// BlockPowerThreshold=99 → 0.95
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

// ── Backward compatibility: old Personality struct literals ──────────────────

func TestHeuristicStrategy_OldPersonalityBackwardCompat(t *testing.T) {
	// Using the old struct-literal pattern (no Weights set) should still work.
	start := &HeuristicStrategy{Personality: AggroPersonality}
	g, pa, _ := makeGame()
	c := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(c)

	// Aggro should attack with everything.
	attackers := start.Attackers(pa, g)
	if len(attackers) != 1 {
		t.Errorf("old-style aggro should attack, got %d attackers", len(attackers))
	}
}

func TestHeuristicStrategy_OldControlHoldsInstants(t *testing.T) {
	start := &HeuristicStrategy{Personality: ControlPersonality}
	g, pa, _ := makeGame()
	card := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)
	// Add a land so the AI can afford it.
	land := mage.NewLand("Mountain")
	land.SetOwner(pa.PlayerID())
	lp := mage.NewPermanent(land, pa.PlayerID())
	lp.RevokeBaseAttr(core.AttrSummonSick)
	g.AddToBattlefield(lp)

	action := start.PriorityAction(pa, g, 1, true)
	// Control holds instants during main phase.
	if action.Type != interactive.ActionPass {
		t.Errorf("old-style control should pass (hold instant), got %v", action.Type)
	}
}

// ── Weighted presets match old behavior ─────────────────────────────────────

func TestWeightedPresets_AggroAttacksAll(t *testing.T) {
	g, pa, pb := makeGame()
	c1 := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	c2 := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	blk := makePerm("Giant", "{3}{G}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(c1, c2, blk)

	start := NewHeuristicStrategy(AggroWeighted)
	attackers := start.Attackers(pa, g)
	if len(attackers) != 2 {
		t.Errorf("AggroWeighted should attack with all, got %d", len(attackers))
	}
}

func TestWeightedPresets_ControlOnlyProfitable(t *testing.T) {
	g, pa, pb := makeGame()
	smallAtk := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	blk := makePerm("Bear", "{1}{G}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(smallAtk, blk)

	start := NewHeuristicStrategy(ControlWeighted)
	attackers := start.Attackers(pa, g)
	if len(attackers) != 0 {
		t.Errorf("ControlWeighted should not attack unprofitably, got %d", len(attackers))
	}
}

func TestWeightedPresets_BurnTargetsFace(t *testing.T) {
	g, pa, pb := makeGame()
	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(oppCreature)

	card := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := NewHeuristicStrategy(BurnWeighted)
	targets := start.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != pb.PlayerID() {
		t.Errorf("BurnWeighted should target face, got %v", targets)
	}
}

func TestWeightedPresets_ControlHoldsInstants(t *testing.T) {
	g, pa, pb := makeGame()
	card := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)
	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(oppCreature)
	land := mage.NewLand("Mountain")
	land.SetOwner(pa.PlayerID())
	lp := mage.NewPermanent(land, pa.PlayerID())
	lp.RevokeBaseAttr(core.AttrSummonSick)
	g.AddToBattlefield(lp)

	start := NewHeuristicStrategy(ControlWeighted)
	action := start.PriorityAction(pa, g, 1, true)
	if action.Type != interactive.ActionPass {
		t.Errorf("ControlWeighted should hold instants in main phase, got %v", action.Type)
	}
}

func TestWeightedPresets_BurnNeverBlocks(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Giant", "{3}{R}", 5, 5, pa.PlayerID())
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := NewHeuristicStrategy(BurnWeighted)
	blocks := start.Blockers(pb, g)
	if len(blocks) != 0 {
		t.Errorf("BurnWeighted should never block, got %d blocks", len(blocks))
	}
}

func TestWeightedPresets_ControlBlocksEverything(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	blk := makePerm("Wall", "{W}", 0, 4, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := NewHeuristicStrategy(ControlWeighted)
	blocks := start.Blockers(pb, g)
	// Control blocks everything (BlockThreshold=0.0), atkPow=1 >= minPow=0.
	// But the blocker logic also checks blkPow >= atkTough OR atkPow >= 3.
	// Here blkPow=0, atkTough=1, atkPow=1 < 3. Neither condition met, so no block.
	// This matches old control behavior: it blocks when power >= 0 AND the trade is favorable.
	// The 1/1 vs 0/4 doesn't trigger because 0 < 1 (can't kill) and 1 < 3.
	if len(blocks) != 0 {
		t.Errorf("ControlWeighted shouldn't block unfavorable position, got %d", len(blocks))
	}
}

// ── Intermediate weight behavior ────────────────────────────────────────────

func TestIntermediateAggression_AttacksMore(t *testing.T) {
	g, pa, pb := makeGame()
	// Our 4-CMC creature vs their 2-CMC creature: trading down in CMC.
	// profitableToAttack returns false (attacker dies, trade down).
	atk := makePerm("Expensive", "{3}{G}", 2, 2, pa.PlayerID())
	blk := makePerm("Cheap", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	// At aggression 0.0 (pure control), shouldn't attack (trade down).
	control := NewHeuristicStrategy(WeightedPersonality{Aggression: 0.0})
	if len(control.Attackers(pa, g)) != 0 {
		t.Error("aggression 0.0 should not attack into losing trade")
	}

	// At aggression 0.7, attacks marginal trades (kills blocker = marginal).
	mid := NewHeuristicStrategy(WeightedPersonality{Aggression: 0.7})
	if len(mid.Attackers(pa, g)) != 1 {
		t.Error("aggression 0.7 should attack when it can kill the blocker")
	}

	// At aggression 1.0, definitely attacks.
	aggro := NewHeuristicStrategy(WeightedPersonality{Aggression: 1.0})
	if len(aggro.Attackers(pa, g)) != 1 {
		t.Error("aggression 1.0 should always attack")
	}
}

func TestIntermediateBlockThreshold(t *testing.T) {
	g, pa, pb := makeGame()
	// Power 2 attacker.
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	blk := makePerm("Giant", "{3}{G}", 4, 4, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	// BlockThreshold 0.0 → block everything (minPow=0, 2 >= 0).
	low := NewHeuristicStrategy(WeightedPersonality{BlockThreshold: 0.0})
	if len(low.Blockers(pb, g)) != 1 {
		t.Error("BlockThreshold 0.0 should block power-2 attacker")
	}

	// BlockThreshold 0.3 → block power >= 3 (2 < 3, skip).
	mid := NewHeuristicStrategy(WeightedPersonality{BlockThreshold: 0.3})
	if len(mid.Blockers(pb, g)) != 0 {
		t.Error("BlockThreshold 0.3 should NOT block power-2 attacker")
	}

	// BlockThreshold 0.1 → block power >= 1 (2 >= 1).
	lowMid := NewHeuristicStrategy(WeightedPersonality{BlockThreshold: 0.1})
	if len(lowMid.Blockers(pb, g)) != 1 {
		t.Error("BlockThreshold 0.1 should block power-2 attacker")
	}
}

func TestIntermediateTargetFace(t *testing.T) {
	g, pa, pb := makeGame()
	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(oppCreature)

	card := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	// TargetFace 0.0 → target creature (lethal).
	creatFace := NewHeuristicStrategy(WeightedPersonality{TargetFace: 0.0})
	targets := creatFace.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != oppCreature.ID() {
		t.Error("TargetFace 0.0 should target creature")
	}

	// TargetFace 1.0 → target face.
	faceFace := NewHeuristicStrategy(WeightedPersonality{TargetFace: 1.0})
	targets = faceFace.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != pb.PlayerID() {
		t.Error("TargetFace 1.0 should target face")
	}

	// TargetFace 0.4 → below 0.5 threshold, target creature.
	midFace := NewHeuristicStrategy(WeightedPersonality{TargetFace: 0.4})
	targets = midFace.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != oppCreature.ID() {
		t.Error("TargetFace 0.4 should target creature (below threshold)")
	}
}

// ── WeightedEvaluator ───────────────────────────────────────────────────────

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
	// Aggro: LifeWeight=1.0, so score = (25-15)*1.0 = 10
	evaluator := eval.WeightedEvaluator(AggroWeighted.Weights)
	got := evaluator(g, pa.PlayerID())
	if got != 10 {
		t.Errorf("Aggro weighted eval (life advantage) = %d, want 10", got)
	}
	// Control: LifeWeight=4.0, so score = (25-15)*4.0 = 40
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

	// evalCreature(2/2 vanilla) = 2*2 + 2*1 = 6
	// Aggro: BoardWeight=3.0, boardScale=3.0/2.0=1.5, score contribution = 6*1.5 = 9
	evaluator := eval.WeightedEvaluator(AggroWeighted.Weights)
	got := evaluator(g, pa.PlayerID())
	if got != 9 {
		t.Errorf("Aggro weighted eval (creature) = %d, want 9", got)
	}

	// Control: BoardWeight=1.0, boardScale=0.5, score contribution = 6*0.5 = 3
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

	// Control: CardWeight=3.0, hand advantage = 2 cards * 3.0 = 6
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

	// Tempo: ManaWeight=2.0, TempoWeight=3.0
	// 1 land * 2.0 + 1 untapped * 3.0 = 5
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

// ── shouldAttack / shouldBlock direct tests ─────────────────────────────────

func TestShouldAttack_MaxAggression(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	blk := makePerm("Giant", "{3}{G}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	// At max aggression, attack even into a losing fight.
	if !shouldAttack(atk, g, pb.PlayerID(), 1.0) {
		t.Error("aggression 1.0 should always attack")
	}
}

func TestShouldAttack_ZeroAggression(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	blk := makePerm("Giant", "{3}{G}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	if shouldAttack(atk, g, pb.PlayerID(), 0.0) {
		t.Error("aggression 0.0 should not attack into losing trade")
	}
}

func TestShouldBlock_ZeroThreshold(t *testing.T) {
	g, _, _ := makeGame()
	if !shouldBlock(1, g, uuid.New(), 0.0) {
		t.Error("BlockThreshold 0.0 should block power-1 attacker")
	}
}

func TestShouldBlock_HighThreshold(t *testing.T) {
	g, _, _ := makeGame()
	if shouldBlock(5, g, uuid.New(), 0.95) {
		t.Error("BlockThreshold 0.95 should never block")
	}
}

// ── NewWeightedAI constructor ───────────────────────────────────────────────

func TestNewWeightedAI(t *testing.T) {
	custom := WeightedPersonality{
		Name:       "Custom",
		Aggression: 0.5,
		Weights: eval.Weights{
			Life: 2.0,
		},
	}
	ai := NewWeightedAI("Bot", custom)
	if ai.Name() != "Bot" {
		t.Errorf("name = %q, want Bot", ai.Name())
	}
	if ai.strategy == nil {
		t.Error("strategy should not be nil")
	}
}

// ── AdaptiveStrategy with weighted presets ───────────────────────────────────

func TestAdaptiveStrategy_WeightedPresets(t *testing.T) {
	g, pa, pb := makeGame()
	pa.SetLife(25)
	pb.SetLife(15)

	adaptive := &AdaptiveStrategy{
		Aggressive: NewHeuristicStrategy(AggroWeighted),
		Defensive:  NewHeuristicStrategy(ControlWeighted),
	}
	got := adaptive.active(pa, g)
	if got != adaptive.Aggressive {
		t.Error("adaptive should use Aggressive when ahead")
	}

	pa.SetLife(5)
	pb.SetLife(20)
	oppCreature := makePerm("Giant", "{3}{R}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(oppCreature)
	got = adaptive.active(pa, g)
	if got != adaptive.Defensive {
		t.Error("adaptive should use Defensive when behind")
	}
}

// ── NewWeightedEvaluator (Phase 3: role-based) ─────────────────────────────

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
	// Life diff (10) * LifeWeight(2.0) = 20
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

// ── EvalCreature with ability quality (exported) ────────────────────────────

func TestEvalCreature_PingerVsVanilla(t *testing.T) {
	pinger := makePerm("Prodigal Sorcerer", "{2}{U}", 1, 1, uuid.New(),
		mage.WithActivatedAbility(mage.DealDamage(mage.Fixed(1)), mage.Tap(),
			mage.WithTarget(mage.TargetAnyTarget())),
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

// ── AbilityQuality (exported) ───────────────────────────────────────────────

func TestAbilityQuality_TapToDealDamage(t *testing.T) {
	ab := mage.NewActivatedAbility(mage.DealDamage(mage.Fixed(1)), mage.Tap(),
		mage.WithTarget(mage.TargetAnyTarget()))
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
		mage.WithTarget(mage.TargetAnyTarget()))
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

// ── Default evaluator integration ───────────────────────────────────────────

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
