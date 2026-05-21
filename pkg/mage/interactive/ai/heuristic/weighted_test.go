package heuristic

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/ai"
)

// ── Backward compatibility: old Personality struct literals ──────────────────

func TestStrategy_OldPersonalityBackwardCompat(t *testing.T) {
	start := &Strategy{Personality: ai.AggroPersonality}
	g, pa, _ := makeGame()
	c := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(c)

	attackers := start.Attackers(pa, g)
	if len(attackers) != 1 {
		t.Errorf("old-style aggro should attack, got %d attackers", len(attackers))
	}
}

func TestStrategy_OldControlHoldsInstants(t *testing.T) {
	start := &Strategy{Personality: ai.ControlPersonality}
	g, pa, _ := makeGame()
	card := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)
	land := mage.NewLand("Mountain")
	land.SetOwner(pa.PlayerID())
	lp := mage.NewPermanent(land, pa.PlayerID())
	lp.RevokeBaseAttr(core.AttrSummonSick)
	g.AddToBattlefield(lp)

	action := start.PriorityAction(pa, g, 1, true)
	if action.Type != interactive.ActionPass {
		t.Errorf("old-style control should pass (hold instant), got %v", action.Type)
	}
}

// ── Weighted presets match old behavior ─────────────────────────────────────

func TestWeightedPresets_AggroAttacksFlyer(t *testing.T) {
	g, pa, pb := makeGame()
	flier := makePerm("Bird", "{1}{U}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.Flying))
	ground := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(flier, ground, blk)

	start := New(ai.AggroWeighted)
	attackers := start.Attackers(pa, g)
	hasFlier := false
	for _, id := range attackers {
		if id == flier.ID() {
			hasFlier = true
		}
	}
	if !hasFlier {
		t.Errorf("AggroWeighted should attack with the flier, got %v", attackers)
	}
}

func TestWeightedPresets_ControlOnlyProfitable(t *testing.T) {
	g, pa, pb := makeGame()
	smallAtk := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	blk := makePerm("Bear", "{1}{G}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(smallAtk, blk)

	start := New(ai.ControlWeighted)
	attackers := start.Attackers(pa, g)
	if len(attackers) != 0 {
		t.Errorf("ControlWeighted should not attack unprofitably, got %d", len(attackers))
	}
}

func TestWeightedPresets_BurnKillsCreatureBeforeFace(t *testing.T) {
	g, pa, pb := makeGame()
	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(oppCreature)

	card := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	start := New(ai.BurnWeighted)
	targets := start.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != oppCreature.ID() {
		t.Errorf("BurnWeighted should kill a valuable creature before going face, got %v", targets)
	}
}

func TestWeightedPresets_ControlHoldsInstants(t *testing.T) {
	g, pa, pb := makeGame()
	card := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))),
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

	start := New(ai.ControlWeighted)
	action := start.PriorityAction(pa, g, 1, true)
	if action.Type != interactive.ActionPass {
		t.Errorf("ControlWeighted should hold instants in main phase, got %v", action.Type)
	}
}

func TestWeightedPresets_BurnSkipsBadBlock(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Giant", "{3}{R}", 5, 5, pa.PlayerID())
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := New(ai.BurnWeighted)
	blocks := start.Blockers(pb, g)
	if len(blocks) > 1 {
		t.Errorf("BurnWeighted should produce at most 1 block, got %d", len(blocks))
	}
}

func TestWeightedPresets_ControlBlocksToSaveLife(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	blk := makePerm("Wall", "{W}", 0, 4, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := New(ai.ControlWeighted)
	blocks := start.Blockers(pb, g)
	if len(blocks) != 1 {
		t.Errorf("ControlWeighted should block 1/1 with 0/4 wall (free life-save), got %d", len(blocks))
	}
}

// ── Intermediate weight behavior ────────────────────────────────────────────

func TestIntermediateTargetFace(t *testing.T) {
	g, pa, pb := makeGame()
	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(oppCreature)

	card := mage.NewInstant("Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	creatFace := New(ai.WeightedPersonality{TargetFace: 0.0})
	targets := creatFace.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != oppCreature.ID() {
		t.Error("TargetFace 0.0 should target creature")
	}

	faceFace := New(ai.WeightedPersonality{TargetFace: 1.0})
	targets = faceFace.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != oppCreature.ID() {
		t.Error("TargetFace 1.0 should still target a valuable creature when face is not lethal or race-positive")
	}

	midFace := New(ai.WeightedPersonality{TargetFace: 0.4})
	targets = midFace.autoSelectTargets(pa, g, card)
	if len(targets) != 1 || targets[0] != oppCreature.ID() {
		t.Error("TargetFace 0.4 should target creature (below threshold)")
	}
}

// ── shouldAttack / shouldBlock direct tests ─────────────────────────────────

func TestShouldAttack_MaxAggression(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	blk := makePerm("Giant", "{3}{G}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
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
