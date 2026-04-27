package ai

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/eval"
)

// ── evaluateCombatOutcome (Phase 5D) ─────────────────────────────────────────

func TestEvaluateCombatOutcome_UnblockedDamage(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(atk)

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
	g.AddToBattlefield(atk, blk)

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
	g.AddToBattlefield(atk, blk)

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
	g.AddToBattlefield(atk, blk)

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
	g.AddToBattlefield(atk, b1, b2)

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
	g.AddToBattlefield(atk, b1, b2)

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
	g.AddToBattlefield(atk, b1, b2)

	available := []*mage.Permanent{b1, b2}
	gang := findGangBlocks(atk, available, g, pb.PlayerID(), true) // theyHaveLethal=true

	if gang == nil {
		t.Fatal("expected gang block when facing lethal, got nil")
	}
}

func TestFindGangBlocks_ThreeBlockersKillBig(t *testing.T) {
	g, pa, pb := makeGame()
	// 7/7 attacker: evalCreature = 7*2+7 = 21
	// Three 3/2 blockers: evalCreature = (3*2+2) = 8 each, total = 24
	// Combined power 9 >= 7 toughness — can kill.
	// Value: 21*100=2100 >= 24*80=1920. Passes.
	// No 2-blocker combo works: best pair power = 6 < 7 toughness.
	atk := makePerm("Wurm", "{5}{G}{G}", 7, 7, pa.PlayerID())
	b1 := makePerm("Soldier1", "{2}{W}", 3, 2, pb.PlayerID())
	b2 := makePerm("Soldier2", "{2}{W}", 3, 2, pb.PlayerID())
	b3 := makePerm("Soldier3", "{2}{W}", 3, 2, pb.PlayerID())
	g.AddToBattlefield(atk, b1, b2, b3)

	available := []*mage.Permanent{b1, b2, b3}
	gang := findGangBlocks(atk, available, g, pb.PlayerID(), false)

	if gang == nil {
		t.Fatal("expected 3-blocker gang block, got nil")
	}
	if len(gang) != 3 {
		t.Fatalf("expected 3 gang blockers, got %d", len(gang))
	}
}

func TestFindGangBlocks_ThreeTokensNotWorthIt(t *testing.T) {
	g, pa, pb := makeGame()
	// 5/5 attacker: evalCreature = 5*2+5 = 15
	// Three 1/1 tokens: evalCreature = (1*2+1) = 3 each, total = 9
	// Combined power 3 < 5 toughness — can't even kill it.
	atk := makePerm("Beast", "{3}{G}{G}", 5, 5, pa.PlayerID())
	b1 := makePerm("Token1", "{0}", 1, 1, pb.PlayerID())
	b2 := makePerm("Token2", "{0}", 1, 1, pb.PlayerID())
	b3 := makePerm("Token3", "{0}", 1, 1, pb.PlayerID())
	g.AddToBattlefield(atk, b1, b2, b3)

	available := []*mage.Permanent{b1, b2, b3}
	gang := findGangBlocks(atk, available, g, pb.PlayerID(), false)

	if gang != nil {
		t.Errorf("expected no gang block (can't kill 5/5 with three 1/1s), got %d blockers", len(gang))
	}
}

func TestFindGangBlocks_ThreeBlockersLethalOverride(t *testing.T) {
	g, pa, pb := makeGame()
	// 7/7 attacker: evalCreature = 7*2+7 = 21
	// Three 3/3 blockers: evalCreature = 9 each, total = 27
	// No 2-blocker pair works: 3+3=6 < 7 toughness.
	// Three blockers: combined power 9 >= 7 — can kill.
	// Value: 21*100=2100 < 27*80=2160. Fails without lethal.
	atk := makePerm("Wurm", "{5}{G}{G}", 7, 7, pa.PlayerID())
	b1 := makePerm("Knight1", "{2}{W}", 3, 3, pb.PlayerID())
	b2 := makePerm("Knight2", "{2}{W}", 3, 3, pb.PlayerID())
	b3 := makePerm("Knight3", "{2}{W}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(atk, b1, b2, b3)

	available := []*mage.Permanent{b1, b2, b3}

	// Without lethal, value check fails (21 < 80% of 27)
	gang := findGangBlocks(atk, available, g, pb.PlayerID(), false)
	if gang != nil {
		t.Errorf("expected no gang block without lethal (value-negative), got %d blockers", len(gang))
	}

	// With lethal, should gang block regardless
	gang = findGangBlocks(atk, available, g, pb.PlayerID(), true)
	if gang == nil {
		t.Fatal("expected gang block when facing lethal, got nil")
	}
	if len(gang) != 3 {
		t.Fatalf("expected 3 gang blockers when facing lethal, got %d", len(gang))
	}
}

func TestFindGangBlocks_TwoBlockersFail_ThreeSucceed(t *testing.T) {
	g, pa, pb := makeGame()
	// 6/6 attacker: evalCreature = 6*2+6 = 18
	// Two 2/2s + one 3/3: no 2-blocker pair has combined power >= 6.
	//   Best pair: 2+3=5 < 6, or 2+2=4 < 6. None work.
	// Three blockers: 2+2+3 = 7 >= 6 toughness — can kill.
	// Value: 18*100=1800. Blockers: 6+6+9=21. 21*80=1680. 1800 >= 1680. Passes.
	atk := makePerm("Giant", "{4}{G}{G}", 6, 6, pa.PlayerID())
	b1 := makePerm("Guard1", "{1}{W}", 2, 2, pb.PlayerID())
	b2 := makePerm("Guard2", "{1}{W}", 2, 2, pb.PlayerID())
	b3 := makePerm("Knight", "{2}{W}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(atk, b1, b2, b3)

	available := []*mage.Permanent{b1, b2, b3}
	gang := findGangBlocks(atk, available, g, pb.PlayerID(), false)

	if gang == nil {
		t.Fatal("expected 3-blocker gang block (2 blockers can't kill 6/6), got nil")
	}
	if len(gang) != 3 {
		t.Fatalf("expected 3 gang blockers, got %d", len(gang))
	}
}

func TestFindGangBlocks_CantBlock(t *testing.T) {
	g, pa, pb := makeGame()
	// Flying attacker: ground blockers can't block
	atk := makePerm("Bird", "{1}{U}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.Flying))
	b1 := makePerm("Bear1", "{1}{G}", 2, 2, pb.PlayerID())
	b2 := makePerm("Bear2", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, b1, b2)

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
	g.AddToBattlefield(atk, b1, b2)

	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	// Use a custom personality with block threshold high enough to skip single blocks
	// but the gang block pass should still trigger for big attackers
	start := NewHeuristicStrategy(WeightedPersonality{
		Name:           "Test",
		Aggression:     0.0,
		BlockThreshold: 0.8, // minPow = 8, so won't block 6-power in single pass
		HoldInstants:   0.0,
		Weights: eval.Weights{
			Life:  2.0,
			Board: 2.0,
			Card:  2.0,
			Mana:  1.0,
			Tempo: 1.0,
		},
	})
	blocks := start.Blockers(pb, g)

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
	g.AddToBattlefield(oppCreature)

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
	g.AddToBattlefield(oppCreature)

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
	g.AddToBattlefield(oppCreature)

	// Give mana
	addLands(g, pa, "Mountain", 1)

	start := NewHeuristicStrategy(MidrangeWeighted)
	response := start.evaluateResponse(pa, g)

	if response == nil {
		t.Fatal("expected evaluateResponse to find bolt, got nil")
	}
	if response.CardName != "Lightning Bolt" {
		t.Errorf("expected Lightning Bolt, got %s", response.CardName)
	}
}

func TestEvaluateResponse_PassWithNoInstants(t *testing.T) {
	g, pa, _ := makeGame()

	start := NewHeuristicStrategy(MidrangeWeighted)
	response := start.evaluateResponse(pa, g)

	if response != nil {
		t.Errorf("expected nil response with empty hand, got %v", response)
	}
}

// ── raceInformedAttack (Phase 5E) ────────────────────────────────────────────

func TestRaceInformedAttack_FavorableRaceAttacksAll(t *testing.T) {
	g, pa, pb := makeGame()
	creature := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(creature)

	race := eval.RaceInfo{MyClock: 2, TheirClock: 4, Racing: true}
	if !raceInformedAttack(creature, g, pb.PlayerID(), race) {
		t.Error("should attack when racing favorably")
	}
}

func TestRaceInformedAttack_UnfavorableOnlyEvasion(t *testing.T) {
	g, pa, pb := makeGame()
	ground := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	flyer := makePerm("Bird", "{1}{U}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.Flying))
	g.AddToBattlefield(ground, flyer)

	race := eval.RaceInfo{MyClock: 4, TheirClock: 2, Racing: true}

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
	g.AddToBattlefield(creature, blk)

	race := eval.RaceInfo{MyClock: 3, TheirClock: 3, Racing: true}

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
	g.AddToBattlefield(smallAtk, blk)

	race := eval.RaceInfo{MyClock: 2, TheirClock: 4, Racing: true}

	// 1 damage * 4 = 4 < 20 life, should skip blocking
	if raceInformedBlock(smallAtk, blk, g, race) {
		t.Error("should skip blocking small attacker when racing favorably")
	}
}

func TestRaceInformedBlock_UnfavorableBlocksAggressively(t *testing.T) {
	g, _, pb := makeGame()
	atk := makePerm("Giant", "{3}{R}", 4, 4, uuid.New())
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	race := eval.RaceInfo{MyClock: 4, TheirClock: 2, Racing: true}

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
	g.AddToBattlefield(oppCreature)

	addLands(g, pa, "Mountain", 1)

	start := NewHeuristicStrategy(MidrangeWeighted)
	// Non-main phase: should evaluate response and cast bolt
	action := start.PriorityAction(pa, g, 0, false)

	if action.Type != interactive.ActionCastSpell {
		t.Errorf("expected ActionCastSpell response, got %v", action.Type)
	}
	if action.CardName != "Lightning Bolt" {
		t.Errorf("expected Lightning Bolt, got %s", action.CardName)
	}
}

// ── First Strike / Deathtouch combat math ─────────────────────────────────────

func TestEvaluateCombatOutcome_FirstStrikeKillsBeforeDamageBack(t *testing.T) {
	// 2/2 first striker vs 3/1 blocker: first strike deals 2, kills 1-toughness blocker.
	// Blocker never gets to deal damage back. Attacker survives, blocker dies.
	g, pa, pb := makeGame()
	atk := makePerm("First Striker", "{1}{W}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.FirstStrike))
	blk := makePerm("Goblin", "{R}", 3, 1, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	blocks := []mage.BlockAssignment{{
		BlockerID:  blk.ID(),
		AttackerID: atk.ID(),
	}}
	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, blocks)

	if cs.OurCreaturesLost != 0 {
		t.Errorf("OurCreaturesLost = %d, want 0 (first striker kills blocker first)", cs.OurCreaturesLost)
	}
	if cs.TheirCreaturesLost != 1 {
		t.Errorf("TheirCreaturesLost = %d, want 1", cs.TheirCreaturesLost)
	}
}

func TestEvaluateCombatOutcome_DeathtouchTradesWithBig(t *testing.T) {
	// 1/1 deathtouch vs 6/6: deathtouch deals 1 damage which is lethal (deathtouch).
	// 6/6 deals 6 back. Both die.
	g, pa, pb := makeGame()
	atk := makePerm("Deathtouch", "{B}", 1, 1, pa.PlayerID(), mage.WithKeyword(core.Deathtouch))
	blk := makePerm("Wurm", "{4}{G}{G}", 6, 6, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	blocks := []mage.BlockAssignment{{
		BlockerID:  blk.ID(),
		AttackerID: atk.ID(),
	}}
	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, blocks)

	if cs.OurCreaturesLost != 1 {
		t.Errorf("OurCreaturesLost = %d, want 1 (deathtouch creature dies to 6 damage)", cs.OurCreaturesLost)
	}
	if cs.TheirCreaturesLost != 1 {
		t.Errorf("TheirCreaturesLost = %d, want 1 (deathtouch kills 6/6)", cs.TheirCreaturesLost)
	}
}

func TestEvaluateCombatOutcome_DeathtouchFirstStrikeSurvives(t *testing.T) {
	// 1/1 deathtouch + first strike vs 6/6: first strike deals 1 (lethal with deathtouch),
	// 6/6 dies before dealing damage. Attacker survives.
	g, pa, pb := makeGame()
	atk := makePerm("Deadly Striker", "{B}{W}", 1, 1, pa.PlayerID(),
		mage.WithKeyword(core.Deathtouch), mage.WithKeyword(core.FirstStrike))
	blk := makePerm("Wurm", "{4}{G}{G}", 6, 6, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	blocks := []mage.BlockAssignment{{
		BlockerID:  blk.ID(),
		AttackerID: atk.ID(),
	}}
	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, blocks)

	if cs.OurCreaturesLost != 0 {
		t.Errorf("OurCreaturesLost = %d, want 0 (deathtouch first striker kills before damage back)", cs.OurCreaturesLost)
	}
	if cs.TheirCreaturesLost != 1 {
		t.Errorf("TheirCreaturesLost = %d, want 1", cs.TheirCreaturesLost)
	}
}

func TestEvaluateCombatOutcome_DoubleStrikeVsBlocker(t *testing.T) {
	// Double strike 3/3 vs 4/4 blocker:
	// Step 1 (first strike): attacker deals 3, blocker at 4 toughness survives (3 < 4).
	// Step 2 (normal): attacker deals 3 again (total 6 >= 4, blocker dies).
	//   Blocker deals 4 (>= 3, attacker dies).
	// Result: both die.
	g, pa, pb := makeGame()
	atk := makePerm("Double Striker", "{1}{R}{W}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.DoubleStrike))
	blk := makePerm("Rhino", "{2}{G}{G}", 4, 4, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	blocks := []mage.BlockAssignment{{
		BlockerID:  blk.ID(),
		AttackerID: atk.ID(),
	}}
	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, blocks)

	if cs.OurCreaturesLost != 1 {
		t.Errorf("OurCreaturesLost = %d, want 1 (3/3 double strike dies to 4/4)", cs.OurCreaturesLost)
	}
	if cs.TheirCreaturesLost != 1 {
		t.Errorf("TheirCreaturesLost = %d, want 1 (4/4 takes 6 total from double strike)", cs.TheirCreaturesLost)
	}
}

func TestEvaluateCombatOutcome_DeathtouchTrample(t *testing.T) {
	// 4/4 deathtouch trample vs 6/6 blocker:
	// Deathtouch means only 1 damage needed to kill the blocker (lethal).
	// Remaining 3 power tramples through to opponent.
	g, pa, pb := makeGame()
	atk := makePerm("Deadly Trampler", "{2}{B}{G}", 4, 4, pa.PlayerID(),
		mage.WithKeyword(core.Deathtouch), mage.WithKeyword(core.Trample))
	blk := makePerm("Wurm", "{4}{G}{G}", 6, 6, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	blocks := []mage.BlockAssignment{{
		BlockerID:  blk.ID(),
		AttackerID: atk.ID(),
	}}
	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, blocks)

	if cs.TheirCreaturesLost != 1 {
		t.Errorf("TheirCreaturesLost = %d, want 1 (deathtouch kills 6/6)", cs.TheirCreaturesLost)
	}
	// 4 power - 1 lethal (deathtouch) = 3 trample damage
	if cs.DamageToOpponent != 3 {
		t.Errorf("DamageToOpponent = %d, want 3 (deathtouch trample: 1 to kill, 3 through)", cs.DamageToOpponent)
	}
	// Attacker takes 6 damage from blocker (6/6), so it dies (4 toughness)
	if cs.OurCreaturesLost != 1 {
		t.Errorf("OurCreaturesLost = %d, want 1 (4/4 takes 6 damage from 6/6)", cs.OurCreaturesLost)
	}
}

// ── Lifelink in combat ─────────────────────────────────────────────────────

func TestEvaluateCombatOutcome_LifelinkUnblocked(t *testing.T) {
	// 3/3 lifelink attacking into empty board: 3 damage + LifeGained=3
	g, pa, pb := makeGame()
	atk := makePerm("Lifelinker", "{1}{W}{W}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.Lifelink))
	g.AddToBattlefield(atk)

	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, nil)
	if cs.DamageToOpponent != 3 {
		t.Errorf("DamageToOpponent = %d, want 3", cs.DamageToOpponent)
	}
	if cs.LifeGained != 3 {
		t.Errorf("LifeGained = %d, want 3", cs.LifeGained)
	}
	_ = pb
}

func TestEvaluateCombatOutcome_LifelinkBlocked(t *testing.T) {
	// 3/3 lifelink attacks into 2/2 blocker. Attacker deals 3 to blocker (kills it),
	// gains 3 life from lifelink. Attacker takes 2 damage (survives).
	g, pa, pb := makeGame()
	atk := makePerm("Lifelinker", "{1}{W}{W}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.Lifelink))
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	blocks := []mage.BlockAssignment{{
		BlockerID:  blk.ID(),
		AttackerID: atk.ID(),
	}}
	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, blocks)
	if cs.LifeGained != 3 {
		t.Errorf("LifeGained = %d, want 3 (lifelink attacker dealt 3 to blocker)", cs.LifeGained)
	}
}

func TestEvaluateCombatOutcome_LifelinkBlocker(t *testing.T) {
	// Opponent's 2/2 lifelink blocks our 3/3. Blocker deals 2, gains 2 for opponent.
	g, pa, pb := makeGame()
	atk := makePerm("Giant", "{2}{G}", 3, 3, pa.PlayerID())
	blk := makePerm("Lifelink Bear", "{1}{W}", 2, 2, pb.PlayerID(), mage.WithKeyword(core.Lifelink))
	g.AddToBattlefield(atk, blk)

	blocks := []mage.BlockAssignment{{
		BlockerID:  blk.ID(),
		AttackerID: atk.ID(),
	}}
	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, blocks)
	if cs.OpponentLifeGained != 2 {
		t.Errorf("OpponentLifeGained = %d, want 2 (lifelink blocker dealt 2)", cs.OpponentLifeGained)
	}
}

func TestEvaluateCombatOutcome_LifelinkTrample(t *testing.T) {
	// 5/5 lifelink trampler blocked by 1/1. Deals 1 to blocker + 4 trample.
	// Total damage dealt = 5, lifelink gains 5.
	g, pa, pb := makeGame()
	atk := makePerm("Trampler", "{3}{W}{G}", 5, 5, pa.PlayerID(),
		mage.WithKeyword(core.Lifelink), mage.WithKeyword(core.Trample))
	blk := makePerm("Elf", "{G}", 1, 1, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	blocks := []mage.BlockAssignment{{
		BlockerID:  blk.ID(),
		AttackerID: atk.ID(),
	}}
	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, blocks)
	if cs.DamageToOpponent != 4 {
		t.Errorf("DamageToOpponent = %d, want 4 (trample)", cs.DamageToOpponent)
	}
	if cs.LifeGained != 5 {
		t.Errorf("LifeGained = %d, want 5 (all damage dealt gains life with lifelink)", cs.LifeGained)
	}
}

func TestEvaluateCombatOutcome_LifelinkScoreBonus(t *testing.T) {
	// Lifelink should make combat score higher than equivalent non-lifelink creature.
	g, pa, _ := makeGame()
	ll := makePerm("Lifelinker", "{1}{W}{W}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.Lifelink))
	vanilla := makePerm("Bear", "{2}{G}", 3, 3, pa.PlayerID())
	g.AddToBattlefield(ll, vanilla)

	csLL := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{ll.ID()}, nil)
	csVanilla := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{vanilla.ID()}, nil)

	if csLL.Score <= csVanilla.Score {
		t.Errorf("lifelink score (%d) should be > vanilla score (%d)", csLL.Score, csVanilla.Score)
	}
}

// addLands is defined in search_test.go
