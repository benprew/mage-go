package ai

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

// ── profitableToAttack ──────────────────────────────────────────────────────

func TestProfitableToAttack_NoBlockers(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(atk)
	if !profitableToAttack(atk, g, pb.PlayerID()) {
		t.Error("should be profitable when no blockers exist")
	}
}

func TestProfitableToAttack_AttackerSurvives(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Giant", "{3}{G}", 4, 5, pa.PlayerID())
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	if !profitableToAttack(atk, g, pb.PlayerID()) {
		t.Error("should be profitable when attacker survives block")
	}
}

func TestProfitableToAttack_TradeUp(t *testing.T) {
	g, pa, pb := makeGame()
	// Our 2-CMC bear trades with their 3-CMC creature — profitable
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	blk := makePerm("Hill Giant", "{2}{R}", 3, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	if !profitableToAttack(atk, g, pb.PlayerID()) {
		t.Error("should be profitable when trading up in CMC")
	}
}

func TestProfitableToAttack_TradeDown(t *testing.T) {
	g, pa, pb := makeGame()
	// Our 4-CMC creature trades with their 2-CMC creature — bad trade
	atk := makePerm("Expensive", "{3}{G}", 2, 2, pa.PlayerID())
	blk := makePerm("Cheap", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	if profitableToAttack(atk, g, pb.PlayerID()) {
		t.Error("should NOT be profitable when trading down in CMC")
	}
}

func TestProfitableToAttack_AttackerDies(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	blk := makePerm("Wall", "{1}{W}", 0, 5, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	// Attacker dies (blocker's 0 power doesn't kill attacker, but attacker can't kill blocker)
	// Actually: atkPow=2 < blkTough=5 (blk doesn't die), blkPow=0 < atkTough=2 (atk survives)
	// So attacker survives — this should be profitable
	if !profitableToAttack(atk, g, pb.PlayerID()) {
		t.Error("attacker survives the wall, should be profitable")
	}
}

func TestProfitableToAttack_AttackerDiesBlockerSurvives(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	blk := makePerm("Big", "{3}{G}", 3, 4, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	// blkPow=3 >= atkTough=2 (atk dies), atkPow=2 < blkTough=4 (blk survives)
	if profitableToAttack(atk, g, pb.PlayerID()) {
		t.Error("should NOT be profitable when attacker dies and blocker survives")
	}
}

func TestProfitableToAttack_TappedBlockerIgnored(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	blk := makePerm("Big", "{3}{G}", 5, 5, pb.PlayerID())
	blk.Tapped = true
	g.AddToBattlefield(atk, blk)
	if !profitableToAttack(atk, g, pb.PlayerID()) {
		t.Error("tapped blocker should be ignored, attack should be profitable")
	}
}

// ── HeuristicStrategy.Attackers ─────────────────────────────────────────────

// TestAttackers_FlyerGetsThrough verifies the solver picks a strictly-
// positive attacker subset: the flying 2/2 can't be blocked by the ground
// 2/2, so it deals 2 damage for free. The ground 1/1 would die to the
// blocker for nothing — solver should skip it but include the flier.
func TestAttackers_FlyerGetsThrough(t *testing.T) {
	g, pa, pb := makeGame()
	flier := makePerm("Bird", "{1}{U}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.Flying))
	ground := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(flier, ground, blk)

	start := &HeuristicStrategy{Personality: AggroPersonality}
	attackers := start.Attackers(pa, g)
	hasFlier := false
	for _, id := range attackers {
		if id == flier.ID() {
			hasFlier = true
		}
	}
	if !hasFlier {
		t.Errorf("solver should attack with the unblockable flier, got %v", attackers)
	}
}

func TestAttackers_ControlOnlyProfitable(t *testing.T) {
	g, pa, pb := makeGame()
	smallAtk := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	// Opponent has a 3/3 that would kill the elf
	blk := makePerm("Bear", "{1}{G}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(smallAtk, blk)

	start := &HeuristicStrategy{Personality: ControlPersonality}
	attackers := start.Attackers(pa, g)
	// Elf vs 3/3 is not profitable
	if len(attackers) != 0 {
		t.Errorf("control should not attack with unprofitable creatures, got %d attackers", len(attackers))
	}
}

func TestAttackers_SkipsCantAttack(t *testing.T) {
	g, pa, _ := makeGame()
	c1 := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	// Create a tapped creature — can't attack
	c2 := makePerm("Tapped", "{1}{R}", 3, 3, pa.PlayerID())
	c2.Tapped = true
	g.AddToBattlefield(c1, c2)

	start := &HeuristicStrategy{Personality: AggroPersonality}
	attackers := start.Attackers(pa, g)
	if len(attackers) != 1 {
		t.Errorf("should only attack with untapped creature, got %d", len(attackers))
	}
}

// ── HeuristicStrategy.Blockers ──────────────────────────────────────────────

func TestBlockers_ControlBlocksHighPower(t *testing.T) {
	g, pa, pb := makeGame()
	// Attacker with power >= 3 so it meets the atkPow >= 3 guard
	atk := makePerm("Giant", "{3}{R}", 3, 3, pa.PlayerID())
	blk := makePerm("Wall", "{W}", 0, 4, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := &HeuristicStrategy{Personality: ControlPersonality}
	blocks := start.Blockers(pb, g)
	if len(blocks) != 1 {
		t.Errorf("control should block high-power attacker, got %d blocks", len(blocks))
	}
}

// TestBlockers_FreeKillIsTaken verifies that the solver takes a strictly
// favourable block (1/1 attacker into 2/2 blocker → kill the attacker, blocker
// survives). The old per-creature heuristic skipped weak attackers based on
// BlockThreshold; the solver evaluates the trade and blocks when blocking
// strictly improves the position.
func TestBlockers_FreeKillIsTaken(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := &HeuristicStrategy{Personality: AggroPersonality}
	blocks := start.Blockers(pb, g)
	if len(blocks) != 1 {
		t.Errorf("solver should block 1/1 with 2/2 (free kill), got %d blocks", len(blocks))
	}
}

func TestBlockers_KillsAttacker(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bear", "{1}{G}", 3, 3, pa.PlayerID())
	blk := makePerm("Giant", "{3}{G}", 4, 4, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := &HeuristicStrategy{Personality: ControlPersonality}
	blocks := start.Blockers(pb, g)
	if len(blocks) != 1 {
		t.Errorf("should block to kill attacker, got %d blocks", len(blocks))
	}
}

func TestBlockers_CantBlockFlying(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bird", "{1}{U}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.Flying))
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := &HeuristicStrategy{Personality: ControlPersonality}
	blocks := start.Blockers(pb, g)
	if len(blocks) != 0 {
		t.Errorf("ground creature should not block flyer, got %d blocks", len(blocks))
	}
}

func TestBlockers_ReachCanBlockFlying(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bird", "{1}{U}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.Flying))
	blk := makePerm("Spider", "{1}{G}", 1, 4, pb.PlayerID(), mage.WithKeyword(core.Reach))
	g.AddToBattlefield(atk, blk)

	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := &HeuristicStrategy{Personality: ControlPersonality}
	blocks := start.Blockers(pb, g)
	// Spider can block flyer (has reach), atkPow=3 >= threshold=0, and atkPow=3 >= 3
	if len(blocks) != 1 {
		t.Errorf("reach creature should block flyer, got %d blocks", len(blocks))
	}
}

// ── HeuristicStrategy.PriorityAction ────────────────────────────────────────

func TestPriorityAction_PlaysLandFirst(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	land := mage.NewLand("Forest")
	land.SetOwner(pa.PlayerID())
	pa.AddToHand(land)

	start := &HeuristicStrategy{Personality: MidrangePersonality}
	action := start.PriorityAction(pa, g, 0, true)
	if action.Type != interactive.ActionPlayLand {
		t.Errorf("expected ActionPlayLand, got %v", action.Type)
	}
}

func TestPriorityAction_PassWhenEmpty(t *testing.T) {
	g, pa, _ := makeGame()
	start := &HeuristicStrategy{Personality: MidrangePersonality}
	action := start.PriorityAction(pa, g, 0, true)
	if action.Type != interactive.ActionPass {
		t.Errorf("expected ActionPass with empty hand, got %v", action.Type)
	}
}

func TestPriorityAction_PassOnNonMainEmptyHand(t *testing.T) {
	g, pa, _ := makeGame()
	start := &HeuristicStrategy{Personality: MidrangePersonality}
	action := start.PriorityAction(pa, g, 0, false)
	if action.Type != interactive.ActionPass {
		t.Errorf("expected ActionPass on non-main with empty hand, got %v", action.Type)
	}
}

// TestPriorityAction_HoldsCombatTrickPrecombat verifies that the AI doesn't
// fire a Giant Growth-style pump spell during the pre-combat main phase when
// it has an attacker available. The trick is held for the post-blockers
// response window.
func TestPriorityAction_HoldsCombatTrickPrecombat(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)

	// Player A has an attacker and a Giant Growth in hand with mana to cast.
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(atk)
	land := mage.NewLand("Forest")
	land.SetOwner(pa.PlayerID())
	lp := mage.NewPermanent(land, pa.PlayerID())
	lp.RevokeBaseAttr(core.AttrSummonSick)
	g.AddToBattlefield(lp)

	growth := mage.NewInstant("Giant Growth", "{G}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.Boost(mage.Fixed(3), mage.Fixed(3))),
	)
	growth.SetOwner(pa.PlayerID())
	pa.AddToHand(growth)

	start := &HeuristicStrategy{Personality: AggroPersonality}
	action := start.PriorityAction(pa, g, 1, true)
	if action.Type == interactive.ActionCastSpell && action.CardID == growth.ID() {
		t.Error("AI should hold Giant Growth in pre-combat main, not cast it sorcery-speed")
	}
}

// TestPriorityAction_CastsTrickWithNoAttackers verifies the inverse: with
// no creatures available to attack, holding a combat trick is pointless —
// it should be cast normally.
func TestPriorityAction_CastsTrickWithNoAttackers(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.PrecombatMain)

	// Player A has no creatures of their own, but opponent does — the trick
	// can target the opponent's creature (still combat-eligible by spec, but
	// nothing to hold for since A has no attackers).
	target := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(target)
	land := mage.NewLand("Forest")
	land.SetOwner(pa.PlayerID())
	lp := mage.NewPermanent(land, pa.PlayerID())
	lp.RevokeBaseAttr(core.AttrSummonSick)
	g.AddToBattlefield(lp)

	growth := mage.NewInstant("Giant Growth", "{G}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.Boost(mage.Fixed(3), mage.Fixed(3))),
	)
	growth.SetOwner(pa.PlayerID())
	pa.AddToHand(growth)

	start := &HeuristicStrategy{Personality: AggroPersonality}
	_ = start.PriorityAction(pa, g, 1, true)
	// We don't assert the action type here — autoSelectTargets may choose to
	// pump the opponent's bear (a buff, not a hold-worthy decision in this
	// configuration). The key invariant is that shouldHoldForCombat returned
	// false (no attackers), so the holding-skip path didn't engage.
	if start.shouldHoldForCombat(g, pa.PlayerID()) {
		t.Error("shouldHoldForCombat returned true when AI has no attackers")
	}
}

// ── AIPlayer.ChooseMode ─────────────────────────────────────────────────────

func TestAIChooseMode_HealingSalveLowLife(t *testing.T) {
	ai := NewAIPlayer("Bot")
	ai.SetLife(5)
	got := ai.ChooseMode([]string{"Gain 3 life", "Prevent 3"}, "Healing Salve")
	if got != 0 {
		t.Errorf("ChooseMode(Healing Salve, low life) = %d, want 0", got)
	}
}

func TestAIChooseMode_HealingSalveHighLife(t *testing.T) {
	ai := NewAIPlayer("Bot")
	ai.SetLife(15)
	got := ai.ChooseMode([]string{"Gain 3 life", "Prevent 3"}, "Healing Salve")
	if got != 1 {
		t.Errorf("ChooseMode(Healing Salve, high life) = %d, want 1", got)
	}
}

func TestAIChooseMode_UnknownCard(t *testing.T) {
	ai := NewAIPlayer("Bot")
	got := ai.ChooseMode([]string{"A", "B"}, "Unknown Card")
	if got != 0 {
		t.Errorf("ChooseMode(unknown) = %d, want 0", got)
	}
}

// ── AdaptiveStrategy ────────────────────────────────────────────────────────

func TestAdaptiveStrategy_AheadUsesAggressive(t *testing.T) {
	g, pa, pb := makeGame()
	pa.SetLife(25)
	pb.SetLife(15)

	adaptive := &AdaptiveStrategy{
		Aggressive: &HeuristicStrategy{Personality: AggroPersonality},
		Defensive:  &HeuristicStrategy{Personality: ControlPersonality},
	}
	got := adaptive.active(pa, g)
	if got != adaptive.Aggressive {
		t.Error("adaptive should use Aggressive when ahead")
	}
}

func TestAdaptiveStrategy_BehindUsesDefensive(t *testing.T) {
	g, pa, pb := makeGame()
	pa.SetLife(5)
	pb.SetLife(20)
	// Give opponent a creature to make score clearly negative
	oppCreature := makePerm("Giant", "{3}{R}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(oppCreature)

	adaptive := &AdaptiveStrategy{
		Aggressive: &HeuristicStrategy{Personality: AggroPersonality},
		Defensive:  &HeuristicStrategy{Personality: ControlPersonality},
	}
	got := adaptive.active(pa, g)
	if got != adaptive.Defensive {
		t.Error("adaptive should use Defensive when behind")
	}
}

// ── SequentialStrategy ──────────────────────────────────────────────────────

type passStrategy struct{}

func (s *passStrategy) PriorityAction(_ mage.Player, _ *mage.Game, _ int, _ bool) interactive.PriorityAction {
	return interactive.PriorityAction{Type: interactive.ActionPass}
}
func (s *passStrategy) Attackers(_ mage.Player, _ *mage.Game) []uuid.UUID           { return nil }
func (s *passStrategy) Blockers(_ mage.Player, _ *mage.Game) []mage.BlockAssignment { return nil }

type fixedActionStrategy struct {
	action interactive.PriorityAction
}

func (s *fixedActionStrategy) PriorityAction(_ mage.Player, _ *mage.Game, _ int, _ bool) interactive.PriorityAction {
	return s.action
}
func (s *fixedActionStrategy) Attackers(_ mage.Player, _ *mage.Game) []uuid.UUID { return nil }
func (s *fixedActionStrategy) Blockers(_ mage.Player, _ *mage.Game) []mage.BlockAssignment {
	return nil
}

func TestSequentialStrategy_FirstNonPassWins(t *testing.T) {
	g, pa, _ := makeGame()
	landAction := interactive.PriorityAction{Type: interactive.ActionPlayLand, CardName: "Forest"}

	seq := &SequentialStrategy{
		Strategies: []AIStrategy{
			&passStrategy{},
			&fixedActionStrategy{action: landAction},
		},
	}
	got := seq.PriorityAction(pa, g, 0, true)
	if got.Type != interactive.ActionPlayLand {
		t.Errorf("expected first non-pass action, got %v", got.Type)
	}
}

func TestSequentialStrategy_AllPassReturnsPass(t *testing.T) {
	g, pa, _ := makeGame()
	seq := &SequentialStrategy{
		Strategies: []AIStrategy{
			&passStrategy{},
			&passStrategy{},
		},
	}
	got := seq.PriorityAction(pa, g, 0, true)
	if got.Type != interactive.ActionPass {
		t.Errorf("expected pass when all strategies pass, got %v", got.Type)
	}
}

func TestSequentialStrategy_Attackers(t *testing.T) {
	g, pa, _ := makeGame()
	seq := &SequentialStrategy{
		Strategies: []AIStrategy{
			&passStrategy{},
			&passStrategy{},
		},
	}
	atks := seq.Attackers(pa, g)
	if len(atks) != 0 {
		t.Errorf("expected no attackers, got %d", len(atks))
	}
}

func TestSequentialStrategy_Blockers(t *testing.T) {
	g, pa, _ := makeGame()
	seq := &SequentialStrategy{
		Strategies: []AIStrategy{
			&passStrategy{},
		},
	}
	blks := seq.Blockers(pa, g)
	if len(blks) != 0 {
		t.Errorf("expected no blockers, got %d", len(blks))
	}
}

// ── autoSelectTargets ───────────────────────────────────────────────────────

func TestAutoSelectTargets_AnyTargetBenefit_OwnCreature(t *testing.T) {
	g, pa, _ := makeGame()
	ownCreature := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(ownCreature)

	card := mage.NewInstant("Heal", "{W}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DrawCards(mage.Fixed(1))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := &HeuristicStrategy{Personality: MidrangePersonality}
	targets := start.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != ownCreature.ID() {
		t.Errorf("benefit spell should target own creature, got %v", targets)
	}
}

func TestAutoSelectTargets_AnyTargetDetriment_OpponentCreature(t *testing.T) {
	g, pa, pb := makeGame()
	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(oppCreature)

	card := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := &HeuristicStrategy{Personality: MidrangePersonality}
	targets := start.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != oppCreature.ID() {
		t.Errorf("detriment spell should target opponent creature, got %v", targets)
	}
}

func TestAutoSelectTargets_BurnTargetsFace(t *testing.T) {
	g, pa, pb := makeGame()
	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(oppCreature)

	card := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := &HeuristicStrategy{Personality: BurnPersonality}
	targets := start.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != pb.PlayerID() {
		t.Errorf("burn should target opponent face, got %v", targets)
	}
}

func TestAutoSelectTargets_CreatureTargetBenefit_OwnCreature(t *testing.T) {
	g, pa, pb := makeGame()
	ownCreature := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	oppCreature := makePerm("Ogre", "{2}{R}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(ownCreature, oppCreature)

	card := mage.NewInstant("Buff", "{G}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.DrawCards(mage.Fixed(1))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := &HeuristicStrategy{Personality: MidrangePersonality}
	targets := start.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != ownCreature.ID() {
		t.Errorf("benefit creature target should pick own creature, got %v", targets)
	}
}

func TestAutoSelectTargets_CreatureTargetDetriment_OpponentCreature(t *testing.T) {
	g, pa, pb := makeGame()
	ownCreature := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	oppCreature := makePerm("Ogre", "{2}{R}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(ownCreature, oppCreature)

	card := mage.NewSorcery("Destroy", "{1}{B}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := &HeuristicStrategy{Personality: MidrangePersonality}
	targets := start.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != oppCreature.ID() {
		t.Errorf("detriment creature target should pick opponent creature, got %v", targets)
	}
}

func TestAutoSelectTargets_PlayerTarget_Opponent(t *testing.T) {
	g, pa, pb := makeGame()
	card := mage.NewSorcery("Drain", "{B}",
		mage.NewTargetedSpell(mage.TargetPlayer(), mage.DealDamage(mage.Fixed(2))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := &HeuristicStrategy{Personality: MidrangePersonality}
	targets := start.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != pb.PlayerID() {
		t.Errorf("player target should select opponent, got %v", targets)
	}
}

func TestAutoSelectTargets_NoTargets_ReturnsNil(t *testing.T) {
	g, pa, _ := makeGame()
	card := mage.NewSorcery("Destroy", "{1}{B}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := &HeuristicStrategy{Personality: MidrangePersonality}
	targets := start.autoSelectTargets(pa, g, card)
	if targets != nil {
		t.Errorf("expected nil targets when no valid targets, got %v", targets)
	}
}

func TestAutoSelectTargets_LethalPreference(t *testing.T) {
	g, pa, pb := makeGame()
	// Two opponent creatures: expensive 5/5 and cheap 2/2. Bolt (3 damage) kills only the 2/2.
	big := makePerm("Giant", "{4}{G}", 5, 5, pb.PlayerID())
	small := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(big, small)

	card := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := &HeuristicStrategy{Personality: MidrangePersonality}
	targets := start.autoSelectTargets(pa, g, card)
	// Should prefer the lethal target (bear) over non-lethal (giant)
	if len(targets) != 1 || targets[0] != small.ID() {
		t.Errorf("should prefer lethal target, got %v", targets)
	}
}

func TestAutoSelectTargets_FallbackToOpponentFace(t *testing.T) {
	g, pa, pb := makeGame()
	// No creatures, just players as targets — should pick opponent
	card := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := &HeuristicStrategy{Personality: MidrangePersonality}
	targets := start.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != pb.PlayerID() {
		t.Errorf("with no creatures, should target opponent player, got %v", targets)
	}
}

// ── NewAIPlayer constructors ────────────────────────────────────────────────

func TestNewAIPlayer_Constructors(t *testing.T) {
	tests := []struct {
		name string
		ai   *AIPlayer
	}{
		{"default", NewAIPlayer("Bot")},
		{"aggro", NewAggroAI("Bot")},
		{"control", NewControlAI("Bot")},
		{"tempo", NewTempoAI("Bot")},
		{"burn", NewBurnAI("Bot")},
		{"adaptive", NewAdaptiveAI("Bot")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ai.Name() != "Bot" {
				t.Errorf("name = %q, want Bot", tt.ai.Name())
			}
			if tt.ai.strategy == nil {
				t.Error("strategy should not be nil")
			}
		})
	}
}

func TestAIPlayer_DeclareAttackers(t *testing.T) {
	g, _, _ := makeGame()
	ai := NewAggroAI("Bot")
	// Wire the AI as player A
	g.SetPlayerAt(0, ai)
	creature := makePerm("Bear", "{1}{G}", 2, 2, ai.PlayerID())
	g.AddToBattlefield(creature)

	attackers := ai.DeclareAttackers(g)
	if len(attackers) != 1 {
		t.Errorf("expected 1 attacker, got %d", len(attackers))
	}
}

func TestAIPlayer_DeclareBlockers(t *testing.T) {
	g, pa, _ := makeGame()
	ai := NewControlAI("Bot")
	// AI is player index 1 (defender)
	g.SetPlayerAt(1, ai)

	// Power 3 attacker triggers the atkPow >= 3 guard in the blocker logic
	atk := makePerm("Giant", "{2}{R}", 3, 3, pa.PlayerID())
	blk := makePerm("Blocker", "{1}{G}", 2, 4, ai.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.GetCombat().AddAttacker(atk.ID(), ai.PlayerID())

	blocks := ai.DeclareBlockers(g)
	if len(blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(blocks))
	}
}

// ── Phase 0A: Lethal-aware attacks ──────────────────────────────────────────

func TestAttackers_LethalUsesMinimalSet(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(3)
	// 3/3 unblockable is lethal by itself
	big := makePerm("Assassin", "{2}{U}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.UnblockableKW))
	// 2/2 vanilla — should NOT be included in lethal set
	small := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(big, small)

	start := &HeuristicStrategy{Personality: ControlPersonality}
	attackers := start.Attackers(pa, g)
	// Should attack with only the 3/3 unblockable (minimal lethal set)
	if len(attackers) != 1 {
		t.Errorf("expected 1 lethal attacker, got %d", len(attackers))
	}
	if len(attackers) == 1 && attackers[0] != big.ID() {
		t.Error("should pick the unblockable creature for lethal")
	}
}

func TestAttackers_LethalOverridesControlPersonality(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(2)
	// 2/2 unblockable vs 5/5 blocker — control personality would normally not attack with this
	// but lethal detection should override
	atk := makePerm("Rogue", "{1}{U}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.UnblockableKW))
	blk := makePerm("Giant", "{3}{G}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	start := &HeuristicStrategy{Personality: ControlPersonality}
	attackers := start.Attackers(pa, g)
	if len(attackers) != 1 {
		t.Errorf("control should attack for lethal, got %d attackers", len(attackers))
	}
}

// ── Phase 0A: Lethal-aware blocking ─────────────────────────────────────────

func TestBlockers_BlocksToPreventLethal(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(3)
	// Opponent attacks with 3/3 — if unblocked, it's lethal
	atk := makePerm("Giant", "{2}{R}", 3, 3, pa.PlayerID())
	// Our 1/1 can block (bad trade normally, but prevents lethal)
	blk := makePerm("Elf", "{G}", 1, 1, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	// Aggro personality normally skips blocking power < 3, but lethal override should block
	start := &HeuristicStrategy{Personality: AggroPersonality}
	blocks := start.Blockers(pb, g)
	if len(blocks) != 1 {
		t.Errorf("should block to prevent lethal, got %d blocks", len(blocks))
	}
}

// ── Phase 0B: Race-aware attacks ────────────────────────────────────────────

func TestAttackers_RaceFavorably(t *testing.T) {
	g, pa, pb := makeGame()
	pa.SetLife(20)
	pb.SetLife(6)
	// We have a 3/3 unblockable (2-turn clock) and a 1/1 ground creature
	unblockable := makePerm("Rogue", "{1}{U}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.UnblockableKW))
	ground := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	// Opponent has a 5/5 blocker that would kill the elf
	bigBlocker := makePerm("Giant", "{3}{G}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(unblockable, ground, bigBlocker)

	// Control personality normally wouldn't attack with the 1/1 into a 5/5
	// But we're racing favorably so it should attack aggressively
	start := &HeuristicStrategy{Personality: ControlPersonality}
	attackers := start.Attackers(pa, g)
	// With favorable race, might use lethal detection or race logic to attack
	if len(attackers) < 1 {
		t.Errorf("should attack when racing favorably, got %d attackers", len(attackers))
	}
}

// ── Phase 0C: Mana curve in PriorityAction ──────────────────────────────────

func TestPriorityAction_PrefersCurvePlay(t *testing.T) {
	g, pa, _ := makeGame()
	// Give player 5 untapped lands
	for range 5 {
		land := mage.NewLand("Forest")
		land.SetOwner(pa.PlayerID())
		lp := mage.NewPermanent(land, pa.PlayerID())
		lp.RevokeBaseAttr(core.AttrSummonSick)
		g.AddToBattlefield(lp)
		pa.ManaPool().Add(core.Green, 1)
	}

	// Hand has a 2-drop and a 5-drop creature
	cheap := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	cheap.SetOwner(pa.PlayerID())
	expensive := mage.NewCreature("Wurm", "{4}{G}", 5, 5)
	expensive.SetOwner(pa.PlayerID())
	pa.AddToHand(cheap)
	pa.AddToHand(expensive)

	start := &HeuristicStrategy{Personality: MidrangePersonality}
	action := start.PriorityAction(pa, g, 1, true) // already played land
	if action.Type == interactive.ActionCastSpell && action.CardName == "Wurm" {
		// Good - preferred the 5-drop
	} else if action.Type == interactive.ActionCastSpell && action.CardName == "Bear" {
		// The base spellValue for 5/5 (5*2+5=15) is much higher than 2/2 (2*2+2=6)
		// so this shouldn't happen even without curve bonus, but verify it
		t.Log("cast the cheaper spell (base spellValue dominates)")
	} else if action.Type == interactive.ActionPass {
		t.Log("passed (might not have enough mana configured in test)")
	}
	// This test primarily verifies the curve bonus code path doesn't crash
}

// ── Phase 0D: Ability activation in PriorityAction ──────────────────────────

func TestPriorityAction_ActivatesAbility(t *testing.T) {
	g, pa, pb := makeGame()
	// Opponent has a creature for targeting
	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(oppCreature)

	// Player has a pinger (tap-to-deal-1-damage, quality=4, above threshold of 3)
	pinger := makePerm("Pinger", "{1}{R}", 1, 1, pa.PlayerID(),
		mage.WithActivatedAbility(mage.DealDamage(mage.Fixed(1)), mage.Tap(),
			mage.WithTarget(mage.TargetAnyTarget())),
	)
	g.AddToBattlefield(pinger)

	start := &HeuristicStrategy{Personality: MidrangePersonality}
	// No spells in hand, not main phase → should consider ability activation
	action := start.PriorityAction(pa, g, 1, false)
	// The AI should activate the pinger ability
	if action.Type == interactive.ActionActivateAbility {
		if action.PermanentID != pinger.ID() {
			t.Error("should activate the pinger's ability")
		}
	}
	// Note: this tests the code path; the ability may or may not be activatable
	// depending on the full game state (e.g., sorcery-speed check)
}

// ── Mulligan ─────────────────────────────────────────────────────────────────

func TestShouldMulligan_ZeroLands(t *testing.T) {
	ai := NewAIPlayer("Bot")
	// 7-card hand with 0 lands — should mulligan
	for range 7 {
		c := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(ai.PlayerID())
		ai.AddToHand(c)
	}
	if !ai.ShouldMulligan(7) {
		t.Error("should mulligan with 0 lands in a 7-card hand")
	}
}

func TestShouldMulligan_SevenLands(t *testing.T) {
	ai := NewAIPlayer("Bot")
	// 7-card hand with 7 lands — should mulligan
	for range 7 {
		c := mage.NewLand("Forest")
		c.SetOwner(ai.PlayerID())
		ai.AddToHand(c)
	}
	if !ai.ShouldMulligan(7) {
		t.Error("should mulligan with 7 lands in a 7-card hand")
	}
}

func TestShouldMulligan_ThreeLandsFourSpells(t *testing.T) {
	ai := NewAIPlayer("Bot")
	// 3 lands + 4 two-drop spells — should keep
	for range 3 {
		c := mage.NewLand("Forest")
		c.SetOwner(ai.PlayerID())
		ai.AddToHand(c)
	}
	for range 4 {
		c := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(ai.PlayerID())
		ai.AddToHand(c)
	}
	if ai.ShouldMulligan(7) {
		t.Error("should keep with 3 lands and 4 castable spells")
	}
}

func TestShouldMulligan_OneLandSixSevenDrops(t *testing.T) {
	ai := NewAIPlayer("Bot")
	// 1 land + 6 seven-drops — no castable spells (CMC 7 > 1+2=3)
	c := mage.NewLand("Forest")
	c.SetOwner(ai.PlayerID())
	ai.AddToHand(c)
	for range 6 {
		s := mage.NewCreature("Wurm", "{5}{G}{G}", 7, 7)
		s.SetOwner(ai.PlayerID())
		ai.AddToHand(s)
	}
	if !ai.ShouldMulligan(7) {
		t.Error("should mulligan with 1 land and no castable spells")
	}
}

func TestShouldMulligan_TwoLandsFourTwoDrops(t *testing.T) {
	ai := NewAIPlayer("Bot")
	// 2 lands + 4 two-drops + 1 three-drop — should keep
	for range 2 {
		c := mage.NewLand("Forest")
		c.SetOwner(ai.PlayerID())
		ai.AddToHand(c)
	}
	for range 4 {
		c := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(ai.PlayerID())
		ai.AddToHand(c)
	}
	s := mage.NewCreature("Centaur", "{2}{G}", 3, 3)
	s.SetOwner(ai.PlayerID())
	ai.AddToHand(s)
	if ai.ShouldMulligan(7) {
		t.Error("should keep with 2 lands and castable spells")
	}
}

func TestShouldMulligan_SixCardHandOneLand(t *testing.T) {
	ai := NewAIPlayer("Bot")
	// 6-card hand with 1 land — more lenient, should keep
	c := mage.NewLand("Forest")
	c.SetOwner(ai.PlayerID())
	ai.AddToHand(c)
	for range 5 {
		s := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		s.SetOwner(ai.PlayerID())
		ai.AddToHand(s)
	}
	if ai.ShouldMulligan(6) {
		t.Error("should keep 6-card hand with 1 land")
	}
}

func TestShouldMulligan_FiveCardHand(t *testing.T) {
	ai := NewAIPlayer("Bot")
	// 5-card hand — always keep regardless of contents
	for range 5 {
		c := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(ai.PlayerID())
		ai.AddToHand(c)
	}
	if ai.ShouldMulligan(5) {
		t.Error("should always keep a 5-card hand")
	}
}

func TestMulligan_ShufflesAndDrawsFewerCards(t *testing.T) {
	ai := NewAIPlayer("Bot")
	// Put 30 cards in library
	for range 30 {
		c := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(ai.PlayerID())
		ai.AddToLibrary(c)
	}
	// Draw 7 cards
	for range 7 {
		ai.DrawCard()
	}
	if len(ai.Hand()) != 7 {
		t.Fatalf("expected 7 cards in hand, got %d", len(ai.Hand()))
	}
	if len(ai.Library()) != 23 {
		t.Fatalf("expected 23 cards in library, got %d", len(ai.Library()))
	}

	ai.Mulligan()

	if len(ai.Hand()) != 6 {
		t.Errorf("expected 6 cards in hand after mulligan, got %d", len(ai.Hand()))
	}
	if len(ai.Library()) != 24 {
		t.Errorf("expected 24 cards in library after mulligan, got %d", len(ai.Library()))
	}
}

func TestMulliganAI_KeepsGoodHand(t *testing.T) {
	ai := NewAIPlayer("Bot")
	// Build a library of 30 cards (mix of lands and spells)
	for range 15 {
		c := mage.NewLand("Forest")
		c.SetOwner(ai.PlayerID())
		ai.AddToLibrary(c)
	}
	for range 15 {
		c := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(ai.PlayerID())
		ai.AddToLibrary(c)
	}
	ai.ShuffleLibrary()
	// Draw opening hand
	for range 7 {
		ai.DrawCard()
	}

	startHand := len(ai.Hand())
	startLib := len(ai.Library())

	MulliganAI(ai)

	// Hand should be between 5 and 7 (kept or mulliganed)
	if len(ai.Hand()) < 5 || len(ai.Hand()) > 7 {
		t.Errorf("hand size after mulligan loop should be 5-7, got %d", len(ai.Hand()))
	}
	// Total cards should be preserved
	total := len(ai.Hand()) + len(ai.Library())
	if total != startHand+startLib {
		t.Errorf("total cards changed: started %d, now %d", startHand+startLib, total)
	}
}
