package heuristic

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
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
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	b1 := makePerm("Knight1", "{1}{W}", 2, 2, pb.PlayerID())
	b2 := makePerm("Knight2", "{1}{W}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, b1, b2)

	available := []*mage.Permanent{b1, b2}
	gang := findGangBlocks(atk, available, g, pb.PlayerID(), false)

	if gang != nil {
		t.Errorf("expected no gang block (not worthwhile), got %d blockers", len(gang))
	}
}

func TestFindGangBlocks_LethalOverridesTradeCheck(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	b1 := makePerm("Elf1", "{G}", 1, 1, pb.PlayerID())
	b2 := makePerm("Elf2", "{G}", 1, 1, pb.PlayerID())
	g.AddToBattlefield(atk, b1, b2)

	available := []*mage.Permanent{b1, b2}
	gang := findGangBlocks(atk, available, g, pb.PlayerID(), true)

	if gang == nil {
		t.Fatal("expected gang block when facing lethal, got nil")
	}
}

func TestFindGangBlocks_ThreeBlockersKillBig(t *testing.T) {
	g, pa, pb := makeGame()
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
	atk := makePerm("Wurm", "{5}{G}{G}", 7, 7, pa.PlayerID())
	b1 := makePerm("Knight1", "{2}{W}", 3, 3, pb.PlayerID())
	b2 := makePerm("Knight2", "{2}{W}", 3, 3, pb.PlayerID())
	b3 := makePerm("Knight3", "{2}{W}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(atk, b1, b2, b3)

	available := []*mage.Permanent{b1, b2, b3}

	gang := findGangBlocks(atk, available, g, pb.PlayerID(), false)
	if gang != nil {
		t.Errorf("expected no gang block without lethal (value-negative), got %d blockers", len(gang))
	}

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
	atk := makePerm("Giant", "{4}{G}", 6, 6, pa.PlayerID())
	b1 := makePerm("Bear1", "{1}{G}", 3, 3, pb.PlayerID())
	b2 := makePerm("Bear2", "{1}{G}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(atk, b1, b2)

	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := New(ai.WeightedPersonality{
		Name:           "Test",
		Aggression:     0.0,
		BlockThreshold: 0.8,
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

	bolt := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)

	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(oppCreature)

	addLands(g, pa, "Mountain", 3)

	hbv := holdBackValue(pa, g, ai.ControlWeighted)
	if hbv <= 0 {
		t.Errorf("holdBackValue for Control with bolt = %f, want > 0", hbv)
	}
}

func TestHoldBackValue_AggroDoesNotHold(t *testing.T) {
	g, pa, pb := makeGame()

	bolt := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)

	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(oppCreature)

	addLands(g, pa, "Mountain", 3)

	hbv := holdBackValue(pa, g, ai.AggroWeighted)
	if hbv != 0 {
		t.Errorf("holdBackValue for Aggro = %f, want 0", hbv)
	}
}

// ── evaluateResponse (Phase 5A) ──────────────────────────────────────────────

func TestEvaluateResponse_CastsRemovalOnOpponentTurn(t *testing.T) {
	g, pa, pb := makeGame()

	bolt := mage.NewInstant("Lightning Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)

	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(oppCreature)

	addLands(g, pa, "Mountain", 1)

	start := New(ai.MidrangeWeighted)
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

	start := New(ai.MidrangeWeighted)
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
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)

	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(oppCreature)

	addLands(g, pa, "Mountain", 1)

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, false)

	if action.Type != interactive.ActionCastSpell {
		t.Errorf("expected ActionCastSpell response, got %v", action.Type)
	}
	if action.CardName != "Lightning Bolt" {
		t.Errorf("expected Lightning Bolt, got %s", action.CardName)
	}
}

// ── First strike / deathtouch / lifelink combat tests ──────────────────────

func TestEvaluateCombatOutcome_FirstStrikeKillsBeforeDamageBack(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Knight", "{1}{W}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.FirstStrike))
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	blocks := []mage.BlockAssignment{{
		BlockerID:  blk.ID(),
		AttackerID: atk.ID(),
	}}
	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, blocks)

	if cs.OurCreaturesLost != 0 {
		t.Errorf("first strike attacker should not die, got OurCreaturesLost=%d", cs.OurCreaturesLost)
	}
	if cs.TheirCreaturesLost != 1 {
		t.Errorf("blocker should die, got TheirCreaturesLost=%d", cs.TheirCreaturesLost)
	}
}

func TestEvaluateCombatOutcome_DeathtouchTradesWithBig(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Stinger", "{1}{B}", 1, 1, pa.PlayerID(), mage.WithKeyword(core.Deathtouch))
	blk := makePerm("Giant", "{3}{G}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	blocks := []mage.BlockAssignment{{
		BlockerID:  blk.ID(),
		AttackerID: atk.ID(),
	}}
	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, blocks)

	if cs.OurCreaturesLost != 1 {
		t.Errorf("deathtouch 1/1 should die, got OurCreaturesLost=%d", cs.OurCreaturesLost)
	}
	if cs.TheirCreaturesLost != 1 {
		t.Errorf("blocker should die to deathtouch, got TheirCreaturesLost=%d", cs.TheirCreaturesLost)
	}
}

// ── Regeneration during combat ─────────────────────────────────────────────

// regenerator builds a creature with a "{B}: Regenerate" activated ability.
func regenerator(name string, power, toughness int, owner uuid.UUID) *mage.Permanent {
	return makePerm(name, "{1}{B}", power, toughness, owner,
		mage.WithActivatedAbility(
			mage.RegenerateSource(),
			mage.ManaCostOf("{B}"),
		),
	)
}

func TestPriorityAction_RegeneratesDoomedBlocker(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.DeclareBlockers)
	addLands(g, pa, "Swamp", 1)

	skeleton := regenerator("Drudge Skeletons", 1, 1, pa.PlayerID())
	bear := makePerm("Grizzly Bears", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(skeleton, bear)

	g.GetCombat().AddAttacker(bear.ID(), pa.PlayerID())
	g.GetCombat().AddBlocker(skeleton.ID(), bear.ID())

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, false)

	if action.Type != interactive.ActionActivateAbility {
		t.Fatalf("expected regeneration activation, got %v", action.Type)
	}
	if action.PermanentID != skeleton.ID() {
		t.Errorf("expected regeneration on the doomed blocker, got %v", action.PermanentID)
	}
}

func TestPriorityAction_RegeneratesAttackerWithMarkedDamage(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.DeclareBlockers)
	addLands(g, pa, "Swamp", 1)

	// 2/3 attacker already carrying 2 damage from a burn spell this turn; it
	// will take 1 more from the blocker and reach lethal.
	troll := regenerator("Regen Troll", 2, 3, pa.PlayerID())
	troll.Damage = 2
	elf := makePerm("Llanowar Elves", "{G}", 1, 1, pb.PlayerID())
	g.AddToBattlefield(troll, elf)

	g.GetCombat().AddAttacker(troll.ID(), pb.PlayerID())
	g.GetCombat().AddBlocker(elf.ID(), troll.ID())

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, false)

	if action.Type != interactive.ActionActivateAbility || action.PermanentID != troll.ID() {
		t.Fatalf("expected regeneration on the doomed attacker, got %+v", action)
	}
}

func TestPriorityAction_DoesNotRegenerateSurvivingCreature(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.DeclareBlockers)
	addLands(g, pa, "Swamp", 1)

	// 3/3 regenerator blocked by a 1/1 — it survives, so no regeneration.
	troll := regenerator("Regen Troll", 3, 3, pa.PlayerID())
	elf := makePerm("Llanowar Elves", "{G}", 1, 1, pb.PlayerID())
	g.AddToBattlefield(troll, elf)

	g.GetCombat().AddAttacker(troll.ID(), pb.PlayerID())
	g.GetCombat().AddBlocker(elf.ID(), troll.ID())

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, false)

	if action.Type == interactive.ActionActivateAbility {
		t.Fatalf("did not expect regeneration of a surviving creature, got %+v", action)
	}
}

func TestPriorityAction_DoesNotDoubleRegenerate(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.DeclareBlockers)
	addLands(g, pa, "Swamp", 2)

	skeleton := regenerator("Drudge Skeletons", 1, 1, pa.PlayerID())
	bear := makePerm("Grizzly Bears", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(skeleton, bear)

	g.GetCombat().AddAttacker(bear.ID(), pa.PlayerID())
	g.GetCombat().AddBlocker(skeleton.ID(), bear.ID())

	// A shield is already in place; the AI must not waste a second activation.
	g.AddRegenerationShield(skeleton.ID())

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, false)

	if action.Type == interactive.ActionActivateAbility {
		t.Fatalf("did not expect a second regeneration activation, got %+v", action)
	}
}

func TestEvaluateCombatOutcome_DeathtouchFirstStrikeSurvives(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Stinger", "{1}{B}", 1, 1, pa.PlayerID(),
		mage.WithKeyword(core.Deathtouch), mage.WithKeyword(core.FirstStrike))
	blk := makePerm("Giant", "{3}{G}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	blocks := []mage.BlockAssignment{{
		BlockerID:  blk.ID(),
		AttackerID: atk.ID(),
	}}
	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, blocks)

	if cs.OurCreaturesLost != 0 {
		t.Errorf("deathtouch+FS should survive, got OurCreaturesLost=%d", cs.OurCreaturesLost)
	}
	if cs.TheirCreaturesLost != 1 {
		t.Errorf("blocker should die, got TheirCreaturesLost=%d", cs.TheirCreaturesLost)
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
	g, pa, pb := makeGame()
	atk := makePerm("Trampler", "{2}{B}", 4, 4, pa.PlayerID(),
		mage.WithKeyword(core.Deathtouch), mage.WithKeyword(core.Trample))
	blk := makePerm("Wall", "{1}{W}", 0, 4, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	blocks := []mage.BlockAssignment{{
		BlockerID:  blk.ID(),
		AttackerID: atk.ID(),
	}}
	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, blocks)

	if cs.DamageToOpponent < 3 {
		t.Errorf("deathtouch+trample should deliver excess to face, got DamageToOpponent=%d", cs.DamageToOpponent)
	}
}

func TestEvaluateCombatOutcome_LifelinkUnblocked(t *testing.T) {
	g, pa, _ := makeGame()
	atk := makePerm("Lifelinker", "{1}{W}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.Lifelink))
	g.AddToBattlefield(atk)

	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, nil)
	if cs.LifeGained != 2 {
		t.Errorf("LifeGained = %d, want 2 (unblocked lifelink)", cs.LifeGained)
	}
}

func TestEvaluateCombatOutcome_LifelinkBlocked(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Lifelinker", "{1}{W}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.Lifelink))
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	blocks := []mage.BlockAssignment{{BlockerID: blk.ID(), AttackerID: atk.ID()}}
	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, blocks)
	if cs.LifeGained != 2 {
		t.Errorf("LifeGained = %d, want 2 (lifelink dealt 2 damage)", cs.LifeGained)
	}
}

func TestEvaluateCombatOutcome_LifelinkBlocker(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	blk := makePerm("Lifelinker", "{1}{W}", 2, 2, pb.PlayerID(), mage.WithKeyword(core.Lifelink))
	g.AddToBattlefield(atk, blk)

	blocks := []mage.BlockAssignment{{BlockerID: blk.ID(), AttackerID: atk.ID()}}
	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, blocks)
	if cs.OpponentLifeGained != 2 {
		t.Errorf("OpponentLifeGained = %d, want 2", cs.OpponentLifeGained)
	}
}

func TestEvaluateCombatOutcome_LifelinkTrample(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Lifelinker", "{2}{W}", 5, 5, pa.PlayerID(),
		mage.WithKeyword(core.Lifelink), mage.WithKeyword(core.Trample))
	blk := makePerm("Elf", "{G}", 1, 1, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	blocks := []mage.BlockAssignment{{BlockerID: blk.ID(), AttackerID: atk.ID()}}
	cs := evaluateCombatOutcome(g, pa.PlayerID(), []uuid.UUID{atk.ID()}, blocks)
	if cs.DamageToOpponent != 4 {
		t.Errorf("DamageToOpponent = %d, want 4 (trample)", cs.DamageToOpponent)
	}
	if cs.LifeGained != 5 {
		t.Errorf("LifeGained = %d, want 5 (all damage dealt gains life with lifelink)", cs.LifeGained)
	}
}

func TestEvaluateCombatOutcome_LifelinkScoreBonus(t *testing.T) {
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
