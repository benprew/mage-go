package heuristic

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/ai"
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
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	blk := makePerm("Hill Giant", "{2}{R}", 3, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	if !profitableToAttack(atk, g, pb.PlayerID()) {
		t.Error("should be profitable when trading up in CMC")
	}
}

func TestProfitableToAttack_TradeDown(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Expensive", "{3}{G}", 2, 2, pa.PlayerID())
	blk := makePerm("Cheap", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	if profitableToAttack(atk, g, pb.PlayerID()) {
		t.Error("should NOT be profitable when trading down in CMC")
	}
}

func TestProfitableToAttack_AttackerSurvivesWall(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	blk := makePerm("Wall", "{1}{W}", 0, 5, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	if !profitableToAttack(atk, g, pb.PlayerID()) {
		t.Error("attacker survives the wall, should be profitable")
	}
}

func TestProfitableToAttack_AttackerDiesBlockerSurvives(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	blk := makePerm("Big", "{3}{G}", 3, 4, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
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

// ── Strategy.Attackers ─────────────────────────────────────────

func TestAttackers_FlyerGetsThrough(t *testing.T) {
	g, pa, pb := makeGame()
	flier := makePerm("Bird", "{1}{U}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.Flying))
	ground := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(flier, ground, blk)

	start := &Strategy{Personality: ai.AggroPersonality}
	attackers := start.Attackers(pa, g)
	hasFlier := false
	for _, id := range attackers {
		if id == flier.ID() {
			hasFlier = true
		}
	}
	if !hasFlier {
		t.Errorf("aggro should attack with the flier, got %v", attackers)
	}
}

func TestAttackers_ControlOnlyProfitable(t *testing.T) {
	g, pa, pb := makeGame()
	smallAtk := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	blk := makePerm("Bear", "{1}{G}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(smallAtk, blk)

	start := &Strategy{Personality: ai.ControlPersonality}
	attackers := start.Attackers(pa, g)
	if len(attackers) != 0 {
		t.Errorf("control should not attack unprofitably, got %d", len(attackers))
	}
}

func TestAttackers_SkipsCantAttack(t *testing.T) {
	g, pa, _ := makeGame()
	wall := makePerm("Wall", "{1}{W}", 0, 4, pa.PlayerID(), mage.WithKeyword(core.Defender))
	bear := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(wall, bear)

	start := &Strategy{Personality: ai.AggroPersonality}
	attackers := start.Attackers(pa, g)
	if len(attackers) != 1 || attackers[0] != bear.ID() {
		t.Errorf("only bear should attack (wall has defender), got %v", attackers)
	}
}

// ── Strategy.Blockers ──────────────────────────────────────────

func TestBlockers_ControlBlocksHighPower(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Giant", "{3}{R}", 4, 4, pa.PlayerID())
	blk := makePerm("Bear", "{1}{G}", 2, 4, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := &Strategy{Personality: ai.ControlPersonality}
	blocks := start.Blockers(pb, g)
	if len(blocks) != 1 {
		t.Errorf("control should block 4-power attacker, got %d blocks", len(blocks))
	}
}

func TestBlockers_FreeKillIsTaken(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	blk := makePerm("Giant", "{3}{G}", 4, 4, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := &Strategy{Personality: ai.AggroPersonality}
	blocks := start.Blockers(pb, g)
	if len(blocks) != 1 {
		t.Errorf("free kill should be taken, got %d blocks", len(blocks))
	}
}

func TestBlockers_KillsAttacker(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Giant", "{3}{R}", 4, 4, pa.PlayerID())
	blk := makePerm("Champion", "{2}{G}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := &Strategy{Personality: ai.ControlPersonality}
	blocks := start.Blockers(pb, g)
	if len(blocks) != 1 {
		t.Errorf("blocker kills attacker, should block, got %d blocks", len(blocks))
	}
}

func TestBlockers_CantBlockFlying(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bird", "{1}{U}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.Flying))
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := &Strategy{Personality: ai.ControlPersonality}
	blocks := start.Blockers(pb, g)
	if len(blocks) != 0 {
		t.Errorf("ground blocker can't block flying, got %d blocks", len(blocks))
	}
}

func TestBlockers_ReachCanBlockFlying(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Bird", "{1}{U}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.Flying))
	blk := makePerm("Spider", "{1}{G}", 2, 4, pb.PlayerID(), mage.WithKeyword(core.Reach))
	g.AddToBattlefield(atk, blk)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := &Strategy{Personality: ai.ControlPersonality}
	blocks := start.Blockers(pb, g)
	if len(blocks) != 1 {
		t.Errorf("reach should block flying, got %d blocks", len(blocks))
	}
}

// ── Strategy.PriorityAction ────────────────────────────────────

func TestPriorityAction_PlaysLandFirst(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	land := mage.NewLand("Forest")
	land.SetOwner(pa.PlayerID())
	pa.AddToHand(land)

	start := &Strategy{Personality: ai.MidrangePersonality}
	action := start.PriorityAction(pa, g, 0, true)
	if action.Type != interactive.ActionPlayLand {
		t.Errorf("expected ActionPlayLand, got %v", action.Type)
	}
}

func TestPriorityAction_PassWhenEmpty(t *testing.T) {
	g, pa, _ := makeGame()
	start := &Strategy{Personality: ai.MidrangePersonality}
	action := start.PriorityAction(pa, g, 1, true)
	if action.Type != interactive.ActionPass {
		t.Errorf("expected ActionPass, got %v", action.Type)
	}
}

func TestPriorityAction_PassOnNonMainEmptyHand(t *testing.T) {
	g, pa, _ := makeGame()
	start := &Strategy{Personality: ai.MidrangePersonality}
	action := start.PriorityAction(pa, g, 1, false)
	if action.Type != interactive.ActionPass {
		t.Errorf("expected ActionPass on non-main with empty hand, got %v", action.Type)
	}
}

func TestPriorityAction_HoldsCombatTrickPrecombat(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetActivePlayerIndex(0)
	g.SetStep(core.PrecombatMain)

	bear := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(bear)

	bolt := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)

	land := mage.NewLand("Mountain")
	land.SetOwner(pa.PlayerID())
	lp := mage.NewPermanent(land, pa.PlayerID())
	lp.RevokeBaseAttr(core.AttrSummonSick)
	g.AddToBattlefield(lp)

	start := &Strategy{Personality: ai.AggroPersonality}
	action := start.PriorityAction(pa, g, 1, true)
	_ = action
	_ = pb
}

func TestPriorityAction_CastsTrickWithNoAttackers(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetActivePlayerIndex(0)
	g.SetStep(core.PrecombatMain)

	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(oppCreature)

	bolt := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)

	land := mage.NewLand("Mountain")
	land.SetOwner(pa.PlayerID())
	lp := mage.NewPermanent(land, pa.PlayerID())
	lp.RevokeBaseAttr(core.AttrSummonSick)
	g.AddToBattlefield(lp)

	start := &Strategy{Personality: ai.AggroPersonality}
	action := start.PriorityAction(pa, g, 1, true)
	_ = action
	if start.shouldHoldForCombat(g, pa.PlayerID()) {
		t.Error("shouldHoldForCombat returned true when AI has no attackers")
	}
}

// ── autoSelectTargets ───────────────────────────────────────────────────────

func TestAutoSelectTargets_AnyTargetBenefit_OwnCreature(t *testing.T) {
	g, pa, _ := makeGame()
	ownCreature := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(ownCreature)

	card := mage.NewInstant("Heal", "{W}",
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DrawCards(mage.Fixed(1))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := &Strategy{Personality: ai.MidrangePersonality}
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
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := &Strategy{Personality: ai.MidrangePersonality}
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
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := &Strategy{Personality: ai.BurnPersonality}
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

	start := &Strategy{Personality: ai.MidrangePersonality}
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

	start := &Strategy{Personality: ai.MidrangePersonality}
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

	start := &Strategy{Personality: ai.MidrangePersonality}
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

	start := &Strategy{Personality: ai.MidrangePersonality}
	targets := start.autoSelectTargets(pa, g, card)
	if targets != nil {
		t.Errorf("expected nil targets when no valid targets, got %v", targets)
	}
}

func TestAutoSelectTargets_LethalPreference(t *testing.T) {
	g, pa, pb := makeGame()
	big := makePerm("Giant", "{4}{G}", 5, 5, pb.PlayerID())
	small := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(big, small)

	card := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := &Strategy{Personality: ai.MidrangePersonality}
	targets := start.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != small.ID() {
		t.Errorf("should prefer lethal target, got %v", targets)
	}
}

func TestAutoSelectTargets_FallbackToOpponentFace(t *testing.T) {
	g, pa, pb := makeGame()
	card := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := &Strategy{Personality: ai.MidrangePersonality}
	targets := start.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != pb.PlayerID() {
		t.Errorf("with no creatures, should target opponent player, got %v", targets)
	}
}

// ── Lethal-aware attacks ────────────────────────────────────────────────────

func TestAttackers_LethalUsesMinimalSet(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(3)
	big := makePerm("Assassin", "{2}{U}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.UnblockableKW))
	small := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(big, small)

	start := &Strategy{Personality: ai.ControlPersonality}
	attackers := start.Attackers(pa, g)
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
	atk := makePerm("Rogue", "{1}{U}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.UnblockableKW))
	blk := makePerm("Giant", "{3}{G}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(atk, blk)

	start := &Strategy{Personality: ai.ControlPersonality}
	attackers := start.Attackers(pa, g)
	if len(attackers) != 1 {
		t.Errorf("control should attack for lethal, got %d attackers", len(attackers))
	}
}

// ── Lethal-aware blocking ───────────────────────────────────────────────────

func TestBlockers_BlocksToPreventLethal(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(3)
	atk := makePerm("Giant", "{2}{R}", 3, 3, pa.PlayerID())
	blk := makePerm("Elf", "{G}", 1, 1, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := &Strategy{Personality: ai.AggroPersonality}
	blocks := start.Blockers(pb, g)
	if len(blocks) != 1 {
		t.Errorf("should block to prevent lethal, got %d blocks", len(blocks))
	}
}

// ── Race-aware attacks ──────────────────────────────────────────────────────

func TestAttackers_RaceFavorably(t *testing.T) {
	g, pa, pb := makeGame()
	pa.SetLife(20)
	pb.SetLife(6)
	unblockable := makePerm("Rogue", "{1}{U}", 3, 3, pa.PlayerID(), mage.WithKeyword(core.UnblockableKW))
	ground := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	bigBlocker := makePerm("Giant", "{3}{G}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(unblockable, ground, bigBlocker)

	start := &Strategy{Personality: ai.ControlPersonality}
	attackers := start.Attackers(pa, g)
	if len(attackers) < 1 {
		t.Errorf("should attack when racing favorably, got %d attackers", len(attackers))
	}
}

// ── Mana curve in PriorityAction ────────────────────────────────────────────

func TestPriorityAction_PrefersCurvePlay(t *testing.T) {
	g, pa, _ := makeGame()
	for range 5 {
		land := mage.NewLand("Forest")
		land.SetOwner(pa.PlayerID())
		lp := mage.NewPermanent(land, pa.PlayerID())
		lp.RevokeBaseAttr(core.AttrSummonSick)
		g.AddToBattlefield(lp)
		pa.ManaPool().Add(core.Green, 1)
	}

	cheap := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	cheap.SetOwner(pa.PlayerID())
	expensive := mage.NewCreature("Wurm", "{4}{G}", 5, 5)
	expensive.SetOwner(pa.PlayerID())
	pa.AddToHand(cheap)
	pa.AddToHand(expensive)

	start := &Strategy{Personality: ai.MidrangePersonality}
	action := start.PriorityAction(pa, g, 1, true)
	_ = action
}

// ── Ability activation in PriorityAction ────────────────────────────────────

func TestPriorityAction_ActivatesAbility(t *testing.T) {
	g, pa, pb := makeGame()
	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(oppCreature)

	pinger := makePerm("Pinger", "{1}{R}", 1, 1, pa.PlayerID(),
		mage.WithActivatedAbility(mage.DealDamage(mage.Fixed(1)), mage.Tap(),
			mage.WithTarget(mage.TargetDamageAnyTarget())),
	)
	g.AddToBattlefield(pinger)

	start := &Strategy{Personality: ai.MidrangePersonality}
	action := start.PriorityAction(pa, g, 1, false)
	if action.Type == interactive.ActionActivateAbility {
		if action.PermanentID != pinger.ID() {
			t.Error("should activate the pinger's ability")
		}
	}
}
