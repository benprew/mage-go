package interactive

import (
	"testing"

	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// ── evaluateCombatOutcome (Phase 5D) ─────────────────────────────────────────

func TestEvaluateCombatOutcome_UnblockedDamage(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.Battlefield = append(g.Battlefield, atk)

	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, nil)
	if cs.DamageToOpponent != 2 {
		t.Errorf("DamageToOpponent = %d, want 2", cs.DamageToOpponent)
	}
	if cs.OurCreaturesLost != 0 {
		t.Errorf("OurCreaturesLost = %d, want 0", cs.OurCreaturesLost)
	}
	if cs.TheirCreaturesLost != 0 {
		t.Errorf("TheirCreaturesLost = %d, want 0", cs.TheirCreaturesLost)
	}
	_ = pb
}

func TestEvaluateCombatOutcome_BlockedTrade(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	blk := makePerm("Bear2", "{1}{G}", 2, 2, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, atk, blk)

	blocks := []mage.BlockAssignment{{
		BlockerID:  blk.ID(),
		AttackerID: atk.ID(),
	}}
	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, blocks)

	if cs.DamageToOpponent != 0 {
		t.Errorf("DamageToOpponent = %d, want 0 (blocked)", cs.DamageToOpponent)
	}
	if cs.OurCreaturesLost != 1 {
		t.Errorf("OurCreaturesLost = %d, want 1", cs.OurCreaturesLost)
	}
	if cs.TheirCreaturesLost != 1 {
		t.Errorf("TheirCreaturesLost = %d, want 1", cs.TheirCreaturesLost)
	}
}

func TestEvaluateCombatOutcome_AttackerSurvives(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Giant", "{3}{G}", 4, 4, pa.PlayerID())
	blk := makePerm("Elf", "{G}", 1, 1, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, atk, blk)

	blocks := []mage.BlockAssignment{{
		BlockerID:  blk.ID(),
		AttackerID: atk.ID(),
	}}
	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, blocks)

	if cs.OurCreaturesLost != 0 {
		t.Errorf("OurCreaturesLost = %d, want 0 (attacker survives)", cs.OurCreaturesLost)
	}
	if cs.TheirCreaturesLost != 1 {
		t.Errorf("TheirCreaturesLost = %d, want 1", cs.TheirCreaturesLost)
	}
	if cs.Score <= 0 {
		t.Errorf("Score = %d, want > 0 (favorable trade)", cs.Score)
	}
}

func TestEvaluateCombatOutcome_Trample(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Trampler", "{3}{G}", 5, 5, pa.PlayerID(), mage.WithKeyword(core.Trample))
	blk := makePerm("Elf", "{G}", 1, 1, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, atk, blk)

	blocks := []mage.BlockAssignment{{
		BlockerID:  blk.ID(),
		AttackerID: atk.ID(),
	}}
	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, blocks)

	// 5 power - 1 toughness blocker = 4 trample damage
	if cs.DamageToOpponent != 4 {
		t.Errorf("DamageToOpponent = %d, want 4 (trample)", cs.DamageToOpponent)
	}
	if cs.TheirCreaturesLost != 1 {
		t.Errorf("TheirCreaturesLost = %d, want 1", cs.TheirCreaturesLost)
	}
}

// ── findGangBlocks (Phase 5C) ────────────────────────────────────────────────

func TestFindGangBlocks_TwoSmallBlockBig(t *testing.T) {
	g, pa, pb := makeGame()
	// 4/4 attacker: evalCreature = 4*2+4 = 12
	// Two 2/2 blockers: evalCreature = (2*2+2)*2 = 12
	// 12 >= 12 (equal trade), and combined power (4) >= toughness (4)
	atk := makePerm("Giant", "{3}{G}", 4, 4, pa.PlayerID())
	b1 := makePerm("Bear1", "{1}{G}", 2, 2, pb.PlayerID())
	b2 := makePerm("Bear2", "{1}{G}", 2, 2, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, atk, b1, b2)

	available := []*mage.Permanent{b1, b2}
	gang := findGangBlocks(atk, available, g, pb.PlayerID(), false)

	if gang == nil {
		t.Fatal("expected gang block, got nil")
	}
	if len(gang) != 2 {
		t.Fatalf("expected 2 gang blockers, got %d", len(gang))
	}
}

func TestFindGangBlocks_NotWorthIt(t *testing.T) {
	g, pa, pb := makeGame()
	// Attacker is a small 2/2, blockers are two valuable 2/2s — not worth ganging
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	b1 := makePerm("Knight1", "{1}{W}", 2, 2, pb.PlayerID())
	b2 := makePerm("Knight2", "{1}{W}", 2, 2, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, atk, b1, b2)

	available := []*mage.Permanent{b1, b2}
	gang := findGangBlocks(atk, available, g, pb.PlayerID(), false)

	// Attacker evalCreature score for 2/2 is small; sum of two 2/2 blockers is larger.
	// Gang block should not be worthwhile.
	if gang != nil {
		t.Errorf("expected no gang block (not worthwhile), got %d blockers", len(gang))
	}
}

func TestFindGangBlocks_LethalOverridesTradeCheck(t *testing.T) {
	g, pa, pb := makeGame()
	// Even though the trade is bad, if they have lethal, we gang-block aggressively
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	b1 := makePerm("Elf1", "{G}", 1, 1, pb.PlayerID())
	b2 := makePerm("Elf2", "{G}", 1, 1, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, atk, b1, b2)

	available := []*mage.Permanent{b1, b2}
	gang := findGangBlocks(atk, available, g, pb.PlayerID(), true) // theyHaveLethal=true

	if gang == nil {
		t.Fatal("expected gang block when facing lethal, got nil")
	}
}

func TestFindGangBlocks_CantBlock(t *testing.T) {
	g, pa, pb := makeGame()
	// Flying attacker: ground blockers can't block
	atk := makePerm("Bird", "{1}{U}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.Flying))
	b1 := makePerm("Bear1", "{1}{G}", 2, 2, pb.PlayerID())
	b2 := makePerm("Bear2", "{1}{G}", 2, 2, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, atk, b1, b2)

	available := []*mage.Permanent{b1, b2}
	gang := findGangBlocks(atk, available, g, pb.PlayerID(), false)

	if gang != nil {
		t.Error("ground creatures should not gang-block a flyer")
	}
}

// ── Blockers with gang blocks integration ────────────────────────────────────

func TestBlockers_GangBlockIntegration(t *testing.T) {
	g, pa, pb := makeGame()
	// A 6/6 that no single 2/2 can kill.
	// evalCreature(6/6) = 6*2+6 = 18
	// evalCreature(2/2) = 2*2+2 = 6, sum = 12
	// 18 > 12, so gang block is worthwhile.
	// Use 2/2 blockers so the first pass won't assign a single blocker
	// (blkPow(2) < atkTough(6) and atkPow(6) >= 3, so the first pass WILL
	// assign a single blocker). We need the first pass to NOT block:
	// use a power threshold that skips in the single-blocker pass.
	// Actually the first pass assigns if blkPow >= atkTough OR atkPow >= 3.
	// Since atkPow(6) >= 3, it will single-block. So the gang block will only
	// trigger for attackers that the single pass doesn't find a blocker for.
	// Let's use an aggro personality with high block threshold to skip single blocks.
	atk := makePerm("Giant", "{4}{G}", 6, 6, pa.PlayerID())
	b1 := makePerm("Bear1", "{1}{G}", 3, 3, pb.PlayerID())
	b2 := makePerm("Bear2", "{1}{G}", 3, 3, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, atk, b1, b2)

	g.Combat.AddAttacker(atk.ID(), pb.PlayerID())

	// Use a custom personality with block threshold high enough to skip single blocks
	// but the gang block pass should still trigger for big attackers
	strat := NewHeuristicStrategy(WeightedPersonality{
		Name:           "Test",
		Aggression:     0.0,
		BlockThreshold: 0.8, // minPow = 8, so won't block 6-power in single pass
		HoldInstants:   0.0,
		LifeWeight:     2.0,
		BoardWeight:    2.0,
		CardWeight:     2.0,
		ManaWeight:     1.0,
		TempoWeight:    1.0,
	})
	blocks := strat.Blockers(pb, g)

	// Gang block should find two 3/3s to kill the 6/6
	if len(blocks) != 2 {
		t.Errorf("expected 2 gang blockers, got %d blocks", len(blocks))
	}
	if len(blocks) == 2 {
		if blocks[0].AttackerID != atk.ID() || blocks[1].AttackerID != atk.ID() {
			t.Error("both blockers should be assigned to the same attacker")
		}
	}
}

// ── holdBackValue (Phase 5B) ─────────────────────────────────────────────────

func TestHoldBackValue_ControlHoldsInstant(t *testing.T) {
	g, pa, pb := makeGame()

	// Give Control AI a Lightning Bolt (instant) and a sorcery
	bolt := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)

	// Add an opponent creature so the bolt has a target
	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, oppCreature)

	// Add lands so we can afford it
	addLands(g, pa, "Mountain", 3)

	// Control personality (HoldInstants=1.0) should want to hold the bolt
	hbv := holdBackValue(pa, g, ControlWeighted)
	if hbv <= 0 {
		t.Errorf("holdBackValue for Control with bolt = %f, want > 0", hbv)
	}
}

func TestHoldBackValue_AggroDoesNotHold(t *testing.T) {
	g, pa, pb := makeGame()

	bolt := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)

	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, oppCreature)

	addLands(g, pa, "Mountain", 3)

	// Aggro personality (HoldInstants=0.0) should not hold
	hbv := holdBackValue(pa, g, AggroWeighted)
	if hbv != 0 {
		t.Errorf("holdBackValue for Aggro = %f, want 0", hbv)
	}
}

// ── evaluateResponse (Phase 5A) ──────────────────────────────────────────────

func TestEvaluateResponse_CastsRemovalOnOpponentTurn(t *testing.T) {
	g, pa, pb := makeGame()

	bolt := mage.NewInstant("Lightning Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)

	// Opponent creature to target
	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, oppCreature)

	// Give mana
	addLands(g, pa, "Mountain", 1)

	strat := NewHeuristicStrategy(MidrangeWeighted)
	response := strat.evaluateResponse(pa, g)

	if response == nil {
		t.Fatal("expected evaluateResponse to find bolt, got nil")
	}
	if response.CardName != "Lightning Bolt" {
		t.Errorf("expected Lightning Bolt, got %s", response.CardName)
	}
}

func TestEvaluateResponse_PassWithNoInstants(t *testing.T) {
	g, pa, _ := makeGame()

	strat := NewHeuristicStrategy(MidrangeWeighted)
	response := strat.evaluateResponse(pa, g)

	if response != nil {
		t.Errorf("expected nil response with empty hand, got %v", response)
	}
}

// ── raceInformedAttack (Phase 5E) ────────────────────────────────────────────

func TestRaceInformedAttack_FavorableRaceAttacksAll(t *testing.T) {
	g, pa, pb := makeGame()
	creature := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.Battlefield = append(g.Battlefield, creature)

	race := RaceInfo{MyClock: 2, TheirClock: 4, Racing: true}
	if !raceInformedAttack(creature, g, pb.PlayerID(), race) {
		t.Error("should attack when racing favorably")
	}
}

func TestRaceInformedAttack_UnfavorableOnlyEvasion(t *testing.T) {
	g, pa, pb := makeGame()
	ground := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	flyer := makePerm("Bird", "{1}{U}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.Flying))
	g.Battlefield = append(g.Battlefield, ground, flyer)

	race := RaceInfo{MyClock: 4, TheirClock: 2, Racing: true}

	if raceInformedAttack(ground, g, pb.PlayerID(), race) {
		t.Error("ground creature should not attack when racing unfavorably")
	}
	if !raceInformedAttack(flyer, g, pb.PlayerID(), race) {
		t.Error("flying creature should attack when racing unfavorably")
	}
}

func TestRaceInformedAttack_TiedRaceTradesUp(t *testing.T) {
	g, pa, pb := makeGame()
	creature := makePerm("Giant", "{3}{G}", 4, 5, pa.PlayerID())
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, creature, blk)

	race := RaceInfo{MyClock: 3, TheirClock: 3, Racing: true}

	// 4/5 vs 2/2 is profitable, should attack
	if !raceInformedAttack(creature, g, pb.PlayerID(), race) {
		t.Error("should attack with profitable creature when race is tied")
	}
}

// ── raceInformedBlock (Phase 5E) ─────────────────────────────────────────────

func TestRaceInformedBlock_FavorableSkipsSmall(t *testing.T) {
	g, _, pb := makeGame()
	pb.SetLife(20)
	smallAtk := makePerm("Elf", "{G}", 1, 1, uuid.New())
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, smallAtk, blk)

	race := RaceInfo{MyClock: 2, TheirClock: 4, Racing: true}

	// 1 damage * 4 = 4 < 20 life, should skip blocking
	if raceInformedBlock(smallAtk, blk, g, race) {
		t.Error("should skip blocking small attacker when racing favorably")
	}
}

func TestRaceInformedBlock_UnfavorableBlocksAggressively(t *testing.T) {
	g, _, pb := makeGame()
	atk := makePerm("Giant", "{3}{R}", 4, 4, uuid.New())
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, atk, blk)

	race := RaceInfo{MyClock: 4, TheirClock: 2, Racing: true}

	if !raceInformedBlock(atk, blk, g, race) {
		t.Error("should block aggressively when racing unfavorably")
	}
}

// ── PriorityAction response integration ──────────────────────────────────────

func TestPriorityAction_ResponseOnOpponentTurn(t *testing.T) {
	g, pa, pb := makeGame()

	bolt := mage.NewInstant("Lightning Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)

	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, oppCreature)

	addLands(g, pa, "Mountain", 1)

	strat := NewHeuristicStrategy(MidrangeWeighted)
	// Non-main phase: should evaluate response and cast bolt
	action := strat.PriorityAction(pa, g, 0, false)

	if action.Type != ActionCastSpell {
		t.Errorf("expected ActionCastSpell response, got %v", action.Type)
	}
	if action.CardName != "Lightning Bolt" {
		t.Errorf("expected Lightning Bolt, got %s", action.CardName)
	}
}

// addLands is defined in search_test.go
