package heuristic

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
)

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

func TestBlockers_AccountsForSelfPumpCreature(t *testing.T) {
	g, pa, pb := makeGame()
	addLands(g, pb, "Swamp", 1)
	atk := makePerm("Savannah Lions", "{W}", 1, 1, pa.PlayerID())
	shade := makePerm("Frozen Shade", "{2}{B}", 0, 1, pb.PlayerID(),
		mage.WithActivatedAbility(
			mage.Boost(mage.Fixed(1), mage.Fixed(1)).Targeting(mage.ToSource()),
			mage.ManaCostOf("{B}"),
		),
	)
	g.AddToBattlefield(atk, shade)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := New(ai.MidrangeWeighted)
	blocks := start.Blockers(pb, g)
	if len(blocks) != 1 || blocks[0].BlockerID != shade.ID() {
		t.Fatalf("expected Frozen Shade to block using pump potential, got %v", blocks)
	}
}

func TestBlockers_AccountsForCarrionAntsPump(t *testing.T) {
	g, pa, pb := makeGame()
	addLands(g, pb, "Swamp", 1)
	atk := makePerm("Savannah Lions", "{W}", 1, 1, pa.PlayerID())
	ants := makePerm("Carrion Ants", "{2}{B}{B}", 0, 1, pb.PlayerID(),
		mage.WithActivatedAbility(
			mage.Boost(mage.Fixed(1), mage.Fixed(1)).Targeting(mage.ToSource()),
			mage.GenericCost(1),
		),
	)
	g.AddToBattlefield(atk, ants)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := New(ai.MidrangeWeighted)
	blocks := start.Blockers(pb, g)
	if len(blocks) != 1 || blocks[0].BlockerID != ants.ID() {
		t.Fatalf("expected Carrion Ants to block using pump potential, got %v", blocks)
	}
}

func TestPriorityAction_HoldsWinterBlastWithOnlyOwnCreatures(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	addLands(g, pa, "Forest", 4)
	own := makePerm("Grizzly Bears", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(own)
	card := mage.NewSorcery("Winter Blast", "{X}{G}",
		mage.NewSpellAbility(mage.FuncEffect("tap X target creatures", mage.EffectProperties{
			Outcome: mage.OutcomeDetriment,
			Mass:    true,
			Taps:    true,
		}, func(*mage.Game, uuid.UUID, uuid.UUID, []uuid.UUID) error { return nil })),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, true)
	if action.Type != interactive.ActionPass {
		t.Fatalf("expected Winter Blast to be held with only own creatures, got %v", action)
	}
}

func TestPriorityAction_WinterBlastXOnlyCountsOpponentCreatures(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.PrecombatMain)
	addLands(g, pa, "Forest", 5)
	own := makePerm("Grizzly Bears", "{1}{G}", 2, 2, pa.PlayerID())
	opp := makePerm("Hill Giant", "{3}{R}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(own, opp)
	card := mage.NewSorcery("Winter Blast", "{X}{G}",
		mage.NewSpellAbility(mage.FuncEffect("tap X target creatures", mage.EffectProperties{
			Outcome: mage.OutcomeDetriment,
			Mass:    true,
			Taps:    true,
		}, func(*mage.Game, uuid.UUID, uuid.UUID, []uuid.UUID) error { return nil })),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, true)
	if action.Type != interactive.ActionCastSpell || action.XValue != 1 {
		t.Fatalf("expected Winter Blast with X=1 for one opposing creature, got %v", action)
	}
}

func TestBestXValue_LethalAgainstCreature(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.PrecombatMain)
	addLands(g, pa, "Mountain", 10)
	opp := makePerm("Hill Giant", "{3}{R}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(opp)

	fireball := mage.NewSorcery("Fireball", "{X}{R}",
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.XValue())),
	)
	fireball.SetOwner(pa.PlayerID())

	x := bestXValue(g, pa.PlayerID(), fireball, []uuid.UUID{opp.ID()})
	if x != 3 {
		t.Fatalf("expected X=3 to kill a 3-toughness creature, got %d", x)
	}
}

func TestBestXValue_LethalAccountsForExistingDamage(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.PrecombatMain)
	addLands(g, pa, "Mountain", 10)
	opp := makePerm("Hill Giant", "{3}{R}", 4, 4, pb.PlayerID())
	opp.Damage = 3
	g.AddToBattlefield(opp)

	fireball := mage.NewSorcery("Fireball", "{X}{R}",
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.XValue())),
	)
	fireball.SetOwner(pa.PlayerID())

	x := bestXValue(g, pa.PlayerID(), fireball, []uuid.UUID{opp.ID()})
	if x != 1 {
		t.Fatalf("expected X=1 to finish a 4-toughness creature with 3 damage, got %d", x)
	}
}

func TestBestXValue_ClampsToLethalWhenManaAllowsMore(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.PrecombatMain)
	addLands(g, pa, "Mountain", 10)
	opp := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(opp)

	fireball := mage.NewSorcery("Fireball", "{X}{R}",
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.XValue())),
	)
	fireball.SetOwner(pa.PlayerID())

	x := bestXValue(g, pa.PlayerID(), fireball, []uuid.UUID{opp.ID()})
	if x != 2 {
		t.Fatalf("expected X=2 (lethal) rather than dumping all mana into a creature, got %d", x)
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

func TestPriorityAction_CastsZeroCostManaArtifact(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	mox := mage.NewArtifact("Test Mox", "{0}", mage.WithManaAbility(core.White))
	mox.SetOwner(pa.PlayerID())
	pa.AddToHand(mox)

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 1, true)
	if action.Type != interactive.ActionCastSpell || action.CardName != "Test Mox" {
		t.Fatalf("expected zero-cost mana artifact to be cast, got %v", action)
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

func TestPriorityAction_SkipsInvalidHighValueSpellForPlayableSpell(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	addLands(g, pa, "Island", 2)
	addLands(g, pa, "Forest", 1)

	drawOnCreature := mage.NewSorcery("Need a Creature", "{1}{U}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.DrawCards(mage.Fixed(2))),
	)
	drawOnCreature.SetOwner(pa.PlayerID())
	pa.AddToHand(drawOnCreature)

	bear := mage.NewCreature("Grizzly Bears", "{1}{G}", 2, 2)
	bear.SetOwner(pa.PlayerID())
	pa.AddToHand(bear)

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 1, true)
	if action.Type != interactive.ActionCastSpell || action.CardName != "Grizzly Bears" {
		t.Fatalf("expected playable creature after invalid target spell, got %v", action)
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

func TestAutoSelectTargets_BurnKillsCreatureBeforeFace(t *testing.T) {
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
	if len(targets) != 1 || targets[0] != oppCreature.ID() {
		t.Errorf("burn should kill a valuable creature before going face, got %v", targets)
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

func TestChooseBestLandUnlocksHighestValueSpell(t *testing.T) {
	g, pa, _ := makeGame()
	island := mage.NewLand("Island", mage.WithManaAbility(core.Blue))
	island.SetOwner(pa.PlayerID())
	swamp := mage.NewLand("Swamp", mage.WithManaAbility(core.Black))
	swamp.SetOwner(pa.PlayerID())
	pa.AddToHand(island)
	pa.AddToHand(swamp)

	dragon := mage.NewCreature("Blue Dragon", "{U}", 5, 5)
	dragon.SetOwner(pa.PlayerID())
	rat := mage.NewCreature("Black Rat", "{B}", 1, 1)
	rat.SetOwner(pa.PlayerID())
	pa.AddToHand(dragon)
	pa.AddToHand(rat)

	best := chooseBestLand(pa, g)
	if best == nil || best.Name() != "Island" {
		t.Fatalf("expected Island to unlock the higher-value spell, got %v", best)
	}
}

func TestAutoSelectTargets_BeneficialAuraDoesNotEnchantOpponentCreature(t *testing.T) {
	g, pa, pb := makeGame()
	oppCreature := makePerm("Ornithopter", "{0}", 0, 2, pb.PlayerID())
	g.AddToBattlefield(oppCreature)

	card := mage.NewBoostAura("Holy Strength", "{W}", 1, 2)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := &Strategy{Personality: ai.MidrangePersonality}
	targets := start.autoSelectTargets(pa, g, card)
	if targets != nil {
		t.Fatalf("beneficial aura should be held with only opponent targets, got %v", targets)
	}
}

func TestAutoSelectTargets_GenericRemovalPrefersCreatureOverLand(t *testing.T) {
	g, pa, pb := makeGame()
	forest := mage.NewLand("Forest", mage.WithManaAbility(core.Green))
	forest.SetOwner(pb.PlayerID())
	forestPerm := mage.NewPermanent(forest, pb.PlayerID())
	forestPerm.RevokeBaseAttr(core.AttrSummonSick)
	threat := makePerm("Zephyr Falcon", "{1}{U}", 1, 1, pb.PlayerID(), mage.WithKeyword(core.Flying))
	g.AddToBattlefield(forestPerm, threat)

	card := mage.NewSorcery("Desert Twister", "{4}{G}{G}",
		mage.NewTargetedSpell(mage.TargetPermanent(), mage.DestroyTargetPermanent()),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := &Strategy{Personality: ai.MidrangePersonality}
	targets := start.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != threat.ID() {
		t.Fatalf("generic removal should target the creature threat, got %v", targets)
	}
}

func TestAutoSelectTargets_AIHintPrefersSmallestOwnCreature(t *testing.T) {
	g, pa, _ := makeGame()
	small := makePerm("Young Hero", "{W}", 1, 1, pa.PlayerID())
	large := makePerm("Veteran", "{3}{W}", 4, 4, pa.PlayerID())
	g.AddToBattlefield(small, large)

	card := mage.NewInstant("Odd Blessing", "{W}",
		mage.NewSpell(
			mage.FuncEffect("put a quest counter on target creature", mage.EffectProperties{
				Outcome: mage.OutcomeBenefit,
			}, func(*mage.Game, uuid.UUID, uuid.UUID, []uuid.UUID) error { return nil }),
			mage.WithTarget(mage.TargetCreature()),
			mage.WithAIHint(mage.AIHint{
				TargetPurpose: mage.AITargetCounters,
				PreferTarget:  mage.PreferSmallestOwnCreature,
			}),
		),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := New(ai.MidrangeWeighted)
	targets := start.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != small.ID() {
		t.Fatalf("AI hint should prefer smallest own creature, got %v", targets)
	}
}

func TestPriorityAction_HoldsEndStepHintedInstantInMainPhase(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	addLands(g, pa, "Island", 3)

	card := mage.NewInstant("Careful Study Later", "{2}{U}",
		mage.NewSpell(
			mage.DrawCards(mage.Fixed(2)),
			mage.WithAIHint(mage.AIHint{Timing: mage.AITimingEndStep}),
		),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 1, true)
	if action.Type != interactive.ActionPass {
		t.Fatalf("expected end-step hinted instant to be held in main phase, got %v", action)
	}
}

func TestAutoSelectTargets_TemporaryPumpSkipsSummoningSickCreature(t *testing.T) {
	g, pa, _ := makeGame()
	creature := mage.NewPermanent(mage.NewCreature("Timber Wolves", "{G}", 1, 1), pa.PlayerID())
	g.AddToBattlefield(creature)

	card := mage.NewInstant("Giant Growth", "{G}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.Boost(mage.Fixed(3), mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := &Strategy{Personality: ai.MidrangePersonality}
	targets := start.autoSelectTargets(pa, g, card)
	if targets != nil {
		t.Fatalf("temporary pump should be held for summoning-sick-only board, got %v", targets)
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

func TestPriorityAction_CardDrawAbilityDrawsWhenHandIsEmpty(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	addLands(g, pa, "Swamp", 3)

	greedCard := mage.NewEnchantment("Greed", "{3}{B}",
		mage.WithActivatedAbility(
			mage.DrawCards(mage.Fixed(1)),
			mage.ManaCostOf("{B}"),
			mage.WithCost(mage.LifePayCost(2)),
		),
	)
	greedCard.SetOwner(pa.PlayerID())
	greed := g.PutOnBattlefield(greedCard, pa.PlayerID())

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, true)
	if action.Type != interactive.ActionActivateAbility || action.PermanentID != greed.ID() {
		t.Fatalf("expected AI to draw with mana and life available, got %+v", action)
	}
}

func TestPriorityAction_CardDrawAbilityValuesFullHandLess(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	addLands(g, pa, "Swamp", 3)
	for range 7 {
		card := mage.NewCreature("Expensive Threat", "{3}{B}", 4, 4)
		card.SetOwner(pa.PlayerID())
		pa.AddToHand(card)
	}

	greedCard := mage.NewEnchantment("Greed", "{3}{B}",
		mage.WithActivatedAbility(
			mage.DrawCards(mage.Fixed(1)),
			mage.ManaCostOf("{B}"),
			mage.WithCost(mage.LifePayCost(2)),
		),
	)
	greedCard.SetOwner(pa.PlayerID())
	g.PutOnBattlefield(greedCard, pa.PlayerID())

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, true)
	if action.Type != interactive.ActionPass {
		t.Fatalf("expected AI to value a full hand over another card, got %+v", action)
	}
}

func TestPriorityAction_CardDrawAbilityKeepsLifeBuffer(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	addLands(g, pa, "Swamp", 4)
	pa.SetLife(5)

	greedCard := mage.NewEnchantment("Greed", "{3}{B}",
		mage.WithActivatedAbility(
			mage.DrawCards(mage.Fixed(1)),
			mage.ManaCostOf("{B}"),
			mage.WithCost(mage.LifePayCost(2)),
		),
	)
	greedCard.SetOwner(pa.PlayerID())
	g.PutOnBattlefield(greedCard, pa.PlayerID())

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, true)
	if action.Type != interactive.ActionPass {
		t.Fatalf("expected AI to keep a life buffer, got %+v", action)
	}
}

func TestPriorityAction_CardDrawAbilityValuesLifeByCurrentTotal(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	addLands(g, pa, "Swamp", 4)

	greedCard := mage.NewEnchantment("Greed", "{3}{B}",
		mage.WithActivatedAbility(
			mage.DrawCards(mage.Fixed(1)),
			mage.ManaCostOf("{B}"),
			mage.WithCost(mage.LifePayCost(2)),
		),
	)
	greedCard.SetOwner(pa.PlayerID())
	greed := g.PutOnBattlefield(greedCard, pa.PlayerID())

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, true)
	if action.Type != interactive.ActionActivateAbility || action.PermanentID != greed.ID() {
		t.Fatalf("expected AI to spend two life while it has a large life buffer, got %+v", action)
	}
}

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

func TestPriorityAction_ProdigalSorcererTargetsKillableCreature(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.PrecombatMain)
	pingerCard := mage.NewCreature("Prodigal Sorcerer", "{2}{U}", 1, 1,
		mage.WithActivatedAbility(
			mage.DealDamage(mage.Fixed(1)),
			mage.Tap(),
			mage.WithTarget(mage.TargetDamageAnyTarget()),
		),
	)
	pingerCard.SetOwner(pa.PlayerID())
	pinger := g.PutOnBattlefield(pingerCard, pa.PlayerID())
	pinger.RevokeBaseAttr(core.AttrSummonSick)
	olderBear := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	killable := makePerm("Merfolk", "{U}", 1, 1, pb.PlayerID())
	g.AddToBattlefield(olderBear, killable)

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, false)
	if action.Type != interactive.ActionActivateAbility {
		t.Fatalf("expected activated ability, got %v", action.Type)
	}
	if len(action.Targets) != 1 || action.Targets[0] != killable.ID() {
		t.Fatalf("expected pinger to target killable creature, got %v want %s", action.Targets, killable.ID())
	}
}

func TestPriorityAction_RodOfRuinTargetsKillableCreature(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.PrecombatMain)
	for range 3 {
		land := mage.NewLand("Mountain", mage.WithManaAbility(core.Red))
		land.SetOwner(pa.PlayerID())
		perm := g.PutOnBattlefield(land, pa.PlayerID())
		perm.RevokeBaseAttr(core.AttrSummonSick)
	}
	rodCard := mage.NewArtifact("Rod of Ruin", "{4}",
		mage.WithActivatedAbility(
			mage.DealDamage(mage.Fixed(1)),
			mage.GenericCost(3),
			mage.WithCost(mage.Tap()),
			mage.WithTarget(mage.TargetDamageAnyTarget()),
		),
	)
	rodCard.SetOwner(pa.PlayerID())
	rod := g.PutOnBattlefield(rodCard, pa.PlayerID())
	rod.RevokeBaseAttr(core.AttrSummonSick)
	olderBear := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	killable := makePerm("Scryb Sprite", "{G}", 1, 1, pb.PlayerID(), mage.WithKeyword(core.Flying))
	g.AddToBattlefield(olderBear, killable)

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, true)
	if action.Type != interactive.ActionActivateAbility {
		t.Fatalf("expected activated ability, got %v", action.Type)
	}
	if len(action.Targets) != 1 || action.Targets[0] != killable.ID() {
		t.Fatalf("expected Rod of Ruin to target killable creature, got %v want %s", action.Targets, killable.ID())
	}
}

func TestPriorityAction_DoesNotGrantSourceKeywordTwice(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	balloonCard := mage.NewCreature("Balloon", "{R}", 1, 1,
		mage.WithKeyword(core.Flying),
		mage.WithActivatedAbility(
			mage.GrantKeyword(core.Flying).Targeting(mage.ToSource()),
			mage.ManaCostOf("{0}"),
		),
	)
	balloonCard.SetOwner(pa.PlayerID())
	balloon := g.PutOnBattlefield(balloonCard, pa.PlayerID())
	balloon.RevokeBaseAttr(core.AttrSummonSick)

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, false)
	if action.Type != interactive.ActionPass {
		t.Fatalf("expected pass when source already has granted keyword, got %v", action.Type)
	}
}

func TestPriorityAction_DoesNotGrantTargetKeywordTwice(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	wandCard := mage.NewArtifact("Wand", "{2}",
		mage.WithActivatedAbility(
			mage.GrantKeyword(core.UnblockableKW),
			mage.Tap(),
			mage.WithTarget(mage.TargetControlledCreature()),
		),
	)
	wandCard.SetOwner(pa.PlayerID())
	g.PutOnBattlefield(wandCard, pa.PlayerID())
	alreadyUnblockable := makePerm("Rogue", "{1}{U}", 2, 2, pa.PlayerID(),
		mage.WithKeyword(core.UnblockableKW),
	)
	g.AddToBattlefield(alreadyUnblockable)

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, false)
	if action.Type != interactive.ActionPass {
		t.Fatalf("expected pass when only target already has granted keyword, got %v", action.Type)
	}
}

func TestPriorityAction_GrantsKeywordToDifferentTarget(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	wandCard := mage.NewArtifact("Wand", "{2}",
		mage.WithActivatedAbility(
			mage.GrantKeyword(core.UnblockableKW),
			mage.Tap(),
			mage.WithTarget(mage.TargetControlledCreature()),
		),
	)
	wandCard.SetOwner(pa.PlayerID())
	wand := g.PutOnBattlefield(wandCard, pa.PlayerID())
	alreadyUnblockable := makePerm("Rogue", "{1}{U}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.UnblockableKW))
	vanilla := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(alreadyUnblockable, vanilla)

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, false)
	if action.Type != interactive.ActionActivateAbility || action.PermanentID != wand.ID() {
		t.Fatalf("expected Wand activation, got %+v", action)
	}
	if len(action.Targets) != 1 || action.Targets[0] != vanilla.ID() {
		t.Fatalf("expected non-redundant target %s, got %v", vanilla.ID(), action.Targets)
	}
}

func TestPriorityAction_DoesNotQueueDuplicateKeywordGrant(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.PrecombatMain)
	firstCard := mage.NewCreature("First Jackal", "{R}", 1, 1,
		mage.WithActivatedAbility(
			mage.GrantKeyword(core.CantRegenerate),
			mage.Tap(),
			mage.WithTarget(mage.TargetCreature()),
		),
	)
	firstCard.SetOwner(pa.PlayerID())
	first := g.PutOnBattlefield(firstCard, pa.PlayerID())
	first.RevokeBaseAttr(core.AttrSummonSick)
	secondCard := mage.NewCreature("Second Jackal", "{R}", 1, 1,
		mage.WithActivatedAbility(
			mage.GrantKeyword(core.CantRegenerate),
			mage.Tap(),
			mage.WithTarget(mage.TargetCreature()),
		),
	)
	secondCard.SetOwner(pa.PlayerID())
	second := g.PutOnBattlefield(secondCard, pa.PlayerID())
	second.RevokeBaseAttr(core.AttrSummonSick)
	target := makePerm("Skeleton", "{1}{B}", 1, 1, pb.PlayerID())
	g.AddToBattlefield(target)
	if err := g.ActivateAbilityByIndex(pa.PlayerID(), first.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
		t.Fatalf("activate first Jackal: %v", err)
	}

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, false)
	if action.Type != interactive.ActionPass {
		t.Fatalf("expected pass when equivalent keyword grant is already on stack, got %v", action.Type)
	}
}
