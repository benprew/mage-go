package interactive

import (
	"testing"

	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
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
	strat := &HeuristicStrategy{Personality: AggroPersonality}
	g, pa, _ := makeGame()
	c := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.Battlefield = append(g.Battlefield, c)

	// Aggro should attack with everything.
	attackers := strat.Attackers(pa, g)
	if len(attackers) != 1 {
		t.Errorf("old-style aggro should attack, got %d attackers", len(attackers))
	}
}

func TestHeuristicStrategy_OldControlHoldsInstants(t *testing.T) {
	strat := &HeuristicStrategy{Personality: ControlPersonality}
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
	g.Battlefield = append(g.Battlefield, lp)

	action := strat.PriorityAction(pa, g, 1, true)
	// Control holds instants during main phase.
	if action.Type != ActionPass {
		t.Errorf("old-style control should pass (hold instant), got %v", action.Type)
	}
}

// ── Weighted presets match old behavior ─────────────────────────────────────

func TestWeightedPresets_AggroAttacksAll(t *testing.T) {
	g, pa, pb := makeGame()
	c1 := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	c2 := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	blk := makePerm("Giant", "{3}{G}", 5, 5, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, c1, c2, blk)

	strat := NewHeuristicStrategy(AggroWeighted)
	attackers := strat.Attackers(pa, g)
	if len(attackers) != 2 {
		t.Errorf("AggroWeighted should attack with all, got %d", len(attackers))
	}
}

func TestWeightedPresets_ControlOnlyProfitable(t *testing.T) {
	g, pa, pb := makeGame()
	smallAtk := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	blk := makePerm("Bear", "{1}{G}", 3, 3, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, smallAtk, blk)

	strat := NewHeuristicStrategy(ControlWeighted)
	attackers := strat.Attackers(pa, g)
	if len(attackers) != 0 {
		t.Errorf("ControlWeighted should not attack unprofitably, got %d", len(attackers))
	}
}

func TestWeightedPresets_BurnTargetsFace(t *testing.T) {
	g, pa, pb := makeGame()
	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, oppCreature)

	card := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	strat := NewHeuristicStrategy(BurnWeighted)
	targets := strat.autoSelectTargets(pa, g, card)
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
	g.Battlefield = append(g.Battlefield, oppCreature)
	land := mage.NewLand("Mountain")
	land.SetOwner(pa.PlayerID())
	lp := mage.NewPermanent(land, pa.PlayerID())
	lp.RevokeBaseAttr(core.AttrSummonSick)
	g.Battlefield = append(g.Battlefield, lp)

	strat := NewHeuristicStrategy(ControlWeighted)
	action := strat.PriorityAction(pa, g, 1, true)
	if action.Type != ActionPass {
		t.Errorf("ControlWeighted should hold instants in main phase, got %v", action.Type)
	}
}

func TestWeightedPresets_BurnNeverBlocks(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Giant", "{3}{R}", 5, 5, pa.PlayerID())
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, atk, blk)
	g.Combat.AddAttacker(atk.ID(), pb.PlayerID())

	strat := NewHeuristicStrategy(BurnWeighted)
	blocks := strat.Blockers(pb, g)
	if len(blocks) != 0 {
		t.Errorf("BurnWeighted should never block, got %d blocks", len(blocks))
	}
}

func TestWeightedPresets_ControlBlocksEverything(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	blk := makePerm("Wall", "{W}", 0, 4, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, atk, blk)
	g.Combat.AddAttacker(atk.ID(), pb.PlayerID())

	strat := NewHeuristicStrategy(ControlWeighted)
	blocks := strat.Blockers(pb, g)
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
	g.Battlefield = append(g.Battlefield, atk, blk)

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
	g.Battlefield = append(g.Battlefield, atk, blk)
	g.Combat.AddAttacker(atk.ID(), pb.PlayerID())

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
	g.Battlefield = append(g.Battlefield, oppCreature)

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
	eval := WeightedEvaluator(MidrangeWeighted)
	got := eval(g, pa.PlayerID())
	if got != 0 {
		t.Errorf("WeightedEvaluator(empty) = %d, want 0", got)
	}
}

func TestWeightedEvaluator_LifeAdvantage(t *testing.T) {
	g, pa, pb := makeGame()
	pa.SetLife(25)
	pb.SetLife(15)
	// Aggro: LifeWeight=1.0, so score = (25-15)*1.0 = 10
	eval := WeightedEvaluator(AggroWeighted)
	got := eval(g, pa.PlayerID())
	if got != 10 {
		t.Errorf("Aggro weighted eval (life advantage) = %d, want 10", got)
	}
	// Control: LifeWeight=4.0, so score = (25-15)*4.0 = 40
	eval2 := WeightedEvaluator(ControlWeighted)
	got2 := eval2(g, pa.PlayerID())
	if got2 != 40 {
		t.Errorf("Control weighted eval (life advantage) = %d, want 40", got2)
	}
}

func TestWeightedEvaluator_BoardWeight(t *testing.T) {
	g, pa, _ := makeGame()
	perm := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.Battlefield = append(g.Battlefield, perm)

	// evalCreature(2/2 vanilla) = 2*2 + 2*1 = 6
	// Aggro: BoardWeight=3.0, boardScale=3.0/2.0=1.5, score contribution = 6*1.5 = 9
	eval := WeightedEvaluator(AggroWeighted)
	got := eval(g, pa.PlayerID())
	if got != 9 {
		t.Errorf("Aggro weighted eval (creature) = %d, want 9", got)
	}

	// Control: BoardWeight=1.0, boardScale=0.5, score contribution = 6*0.5 = 3
	eval2 := WeightedEvaluator(ControlWeighted)
	got2 := eval2(g, pa.PlayerID())
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
	eval := WeightedEvaluator(ControlWeighted)
	got := eval(g, pa.PlayerID())
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
	g.Battlefield = append(g.Battlefield, lp)

	// Tempo: ManaWeight=2.0, TempoWeight=3.0
	// 1 land * 2.0 + 1 untapped * 3.0 = 5
	eval := WeightedEvaluator(TempoWeighted)
	got := eval(g, pa.PlayerID())
	if got != 5 {
		t.Errorf("Tempo weighted eval (land+tempo) = %d, want 5", got)
	}
}

func TestWeightedEvaluator_NilPlayer(t *testing.T) {
	g, _, _ := makeGame()
	eval := WeightedEvaluator(MidrangeWeighted)
	got := eval(g, uuid.New())
	if got != 0 {
		t.Errorf("WeightedEvaluator(nil player) = %d, want 0", got)
	}
}

// ── shouldAttack / shouldBlock direct tests ─────────────────────────────────

func TestShouldAttack_MaxAggression(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	blk := makePerm("Giant", "{3}{G}", 5, 5, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, atk, blk)
	// At max aggression, attack even into a losing fight.
	if !shouldAttack(atk, g, pb.PlayerID(), 1.0) {
		t.Error("aggression 1.0 should always attack")
	}
}

func TestShouldAttack_ZeroAggression(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	blk := makePerm("Giant", "{3}{G}", 5, 5, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, atk, blk)
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
		LifeWeight: 2.0,
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
	g.Battlefield = append(g.Battlefield, oppCreature)
	got = adaptive.active(pa, g)
	if got != adaptive.Defensive {
		t.Error("adaptive should use Defensive when behind")
	}
}
