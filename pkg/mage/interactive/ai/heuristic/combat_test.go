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

// CombatScore summarizes the outcome of a combat step. It is a test-only
// scoring view over the production simulateCombatDamage simulator, used to
// assert that damage, deaths, and life exchange resolve correctly across
// first strike, deathtouch, trample, and lifelink.
type CombatScore struct {
	DamageToOpponent   int
	OurCreaturesLost   int
	TheirCreaturesLost int
	LifeGained         int // life gained by our lifelink attackers
	OpponentLifeGained int // life gained by opponent's lifelink blockers
	Score              int
}

// evaluateCombatOutcome simulates the given attack/block and scores it. Kept as
// a test helper around the live simulateCombatDamage after the standalone
// heuristic combat evaluator was superseded by the combatsolver.
func evaluateCombatOutcome(g *mage.Game, playerID uuid.UUID, attackers []uuid.UUID, blocks []mage.BlockAssignment) CombatScore {
	opponent := g.GetOpponent(playerID)
	if opponent == nil {
		return CombatScore{}
	}

	blockerMap := make(map[uuid.UUID][]uuid.UUID)
	for _, b := range blocks {
		blockerMap[b.AttackerID] = append(blockerMap[b.AttackerID], b.BlockerID)
	}

	res := simulateCombatDamage(g, attackers, blockerMap, nil)

	cs := CombatScore{
		DamageToOpponent:   res.damageToOpponent,
		LifeGained:         res.lifeGained,
		OpponentLifeGained: res.opponentLifeGained,
	}

	// Tally kills and lost value from accumulated damage (including deathtouch marks).
	var ourLostValue, theirLostValue int
	for _, atkID := range attackers {
		atk := g.FindPermanent(atkID)
		if atk == nil {
			continue
		}
		blockerIDs := blockerMap[atkID]
		if len(blockerIDs) == 0 {
			continue
		}
		if res.isDead(atkID, atk.CurrentToughness(g)) {
			cs.OurCreaturesLost++
			ourLostValue += eval.EvalCreatureInGame(atk, g)
		}
		for _, blkID := range blockerIDs {
			blk := g.FindPermanent(blkID)
			if blk == nil {
				continue
			}
			if res.isDead(blkID, blk.CurrentToughness(g)) {
				cs.TheirCreaturesLost++
				theirLostValue += eval.EvalCreatureInGame(blk, g)
			}
		}
	}

	cs.Score = cs.DamageToOpponent + theirLostValue - ourLostValue + cs.LifeGained - cs.OpponentLifeGained

	return cs
}

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

// Holding an instant should reserve only its mana, not freeze the whole main
// phase: when the AI can afford both a board-developing spell and the held
// instant, it develops and keeps the instant's mana up.
func TestPriorityAction_DevelopsBoardWhileHoldingInstant(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetActivePlayerIndex(0)
	g.SetStep(core.PrecombatMain)

	dragon := makePerm("Dragon", "{4}{R}{R}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(dragon)

	terror := mage.NewInstant("Terror", "{1}{B}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()))
	terror.SetOwner(pa.PlayerID())
	pa.AddToHand(terror)

	bears := mage.NewCreature("Grizzly Bears", "{1}{G}", 2, 2)
	bears.SetOwner(pa.PlayerID())
	pa.AddToHand(bears)

	// Enough mana for both Terror ({1}{B}) and Grizzly Bears ({1}{G}).
	addLands(g, pa, "Swamp", 2)
	addLands(g, pa, "Forest", 2)

	// Precondition: the AI genuinely wants to hold the removal instant.
	if holdBackValue(pa, g, ai.ControlWeighted) <= 0 {
		t.Fatalf("test setup: expected the AI to want to hold Terror")
	}

	s := New(ai.ControlWeighted)
	action := s.PriorityAction(pa, g, 0, true)
	if action.Type != interactive.ActionCastSpell || action.CardName != "Grizzly Bears" {
		t.Fatalf("expected the AI to develop Grizzly Bears while holding Terror, got %+v", action)
	}
}

// When the AI cannot afford both the held instant and a development spell, it
// keeps the instant's mana up and passes rather than tapping out.
func TestPriorityAction_PassesWhenCannotAffordBothHeldInstantAndSpell(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetActivePlayerIndex(0)
	g.SetStep(core.PrecombatMain)

	dragon := makePerm("Dragon", "{4}{R}{R}", 5, 5, pb.PlayerID())
	g.AddToBattlefield(dragon)

	terror := mage.NewInstant("Terror", "{1}{B}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()))
	terror.SetOwner(pa.PlayerID())
	pa.AddToHand(terror)

	bears := mage.NewCreature("Grizzly Bears", "{1}{G}", 2, 2)
	bears.SetOwner(pa.PlayerID())
	pa.AddToHand(bears)

	// Only enough mana for one of the two spells.
	addLands(g, pa, "Swamp", 1)
	addLands(g, pa, "Forest", 1)

	if holdBackValue(pa, g, ai.ControlWeighted) <= 0 {
		t.Fatalf("test setup: expected the AI to want to hold Terror")
	}

	s := New(ai.ControlWeighted)
	action := s.PriorityAction(pa, g, 0, true)
	if action.Type != interactive.ActionPass {
		t.Fatalf("expected the AI to hold mana and pass, got %+v", action)
	}
}

// End-to-end: a held pump trick should make the Strategy's attack solver
// declare an attack it would otherwise refuse (2/2 into a 2/3).
func TestAttackers_AttacksWhenHoldingPumpTrick(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetActivePlayerIndex(0)

	attacker := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	blocker := makePerm("Ogre", "{1}{R}", 2, 3, pb.PlayerID())
	g.AddToBattlefield(attacker, blocker)
	addLands(g, pa, "Forest", 1)

	s := New(ai.MidrangeWeighted)

	if atks := s.Attackers(pa, g); len(atks) != 0 {
		t.Fatalf("baseline: 2/2 should not attack into 2/3 without a trick, got %v", atks)
	}

	pa.AddToHand(createCard(t, "Giant Growth", pa.PlayerID()))
	atks := s.Attackers(pa, g)
	if len(atks) != 1 || atks[0] != attacker.ID() {
		t.Fatalf("with Giant Growth in hand the 2/2 should attack, got %v", atks)
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

func TestPriorityAction_CastsRegenerationSpellOnDoomedCreature(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.DeclareBlockers)
	addLands(g, pa, "Plains", 1)

	ward := mage.NewInstant("Death Ward", "{W}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.RegenerateTarget()),
	)
	ward.SetOwner(pa.PlayerID())
	pa.AddToHand(ward)

	doomed := makePerm("Doomed Guard", "{W}", 1, 1, pa.PlayerID())
	safe := makePerm("Safe Giant", "{3}{W}", 5, 5, pa.PlayerID())
	attacker := makePerm("Grizzly Bears", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(doomed, safe, attacker)

	g.GetCombat().AddAttacker(attacker.ID(), pa.PlayerID())
	g.GetCombat().AddBlocker(doomed.ID(), attacker.ID())

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, false)

	if action.Type != interactive.ActionCastSpell {
		t.Fatalf("expected regeneration spell, got %+v", action)
	}
	if action.CardName != "Death Ward" {
		t.Fatalf("expected Death Ward, got %s", action.CardName)
	}
	if len(action.Targets) != 1 || action.Targets[0] != doomed.ID() {
		t.Fatalf("expected Death Ward to target doomed creature, got %+v", action.Targets)
	}
}

func TestPriorityAction_ActivatesAttachedRegenerationOnDoomedHost(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.DeclareBlockers)

	host := makePerm("Retained Guard", "{1}{W}", 2, 2, pa.PlayerID())
	retainerCard := mage.NewAura("Test Retainer", "{B}",
		mage.WithActivatedAbility(
			mage.Pipeline("Regenerate enchanted creature",
				mage.EffectProperties{},
				mage.SnapshotAttached("attached"),
				mage.RegenerateGathered("attached"),
			),
			mage.SacrificeSourceCost(),
		),
	)
	retainerCard.SetOwner(pa.PlayerID())
	attacker := makePerm("Hill Giant", "{3}{R}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(host, attacker)
	retainer := g.PutOnBattlefield(retainerCard, pa.PlayerID())
	g.Attach(retainer.ID(), host.ID())

	g.GetCombat().AddAttacker(attacker.ID(), pa.PlayerID())
	g.GetCombat().AddBlocker(host.ID(), attacker.ID())

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, false)

	if action.Type != interactive.ActionActivateAbility {
		t.Fatalf("expected attached regeneration activation, got %+v", action)
	}
	if action.PermanentID != retainer.ID() {
		t.Fatalf("expected activation from retainer aura, got %v", action.PermanentID)
	}
	if len(action.Targets) != 0 {
		t.Fatalf("expected no targets for attached regeneration ability, got %+v", action.Targets)
	}
}

// A creature targeted by a destroy-removal spell on the stack should be saved
// with a regeneration shield in response, outside of combat.
func TestPriorityAction_RegeneratesAgainstRemovalSpell(t *testing.T) {
	g, pa, pb := makeGame()
	addLands(g, pa, "Swamp", 1)

	skeleton := regenerator("Drudge Skeletons", 1, 1, pa.PlayerID())
	g.AddToBattlefield(skeleton)

	// Opponent's "destroy target creature" spell targets our skeleton.
	g.PushStack(&mage.StackObject{
		ID:         uuid.New(),
		Controller: pb.PlayerID(),
		Effects:    []mage.Effect{mage.DestroyTarget()},
		Targets:    []uuid.UUID{skeleton.ID()},
	})

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, false)

	if action.Type != interactive.ActionActivateAbility || action.PermanentID != skeleton.ID() {
		t.Fatalf("expected regeneration in response to removal, got %+v", action)
	}
}

// A creature targeted by lethal burn on the stack should be regenerated, since
// regeneration also replaces destruction from lethal damage.
func TestPriorityAction_RegeneratesAgainstLethalBurn(t *testing.T) {
	g, pa, pb := makeGame()
	addLands(g, pa, "Swamp", 1)

	skeleton := regenerator("Drudge Skeletons", 1, 1, pa.PlayerID())
	g.AddToBattlefield(skeleton)

	bolt := createCard(t, "Lightning Bolt", pb.PlayerID())
	var boltEffects []mage.Effect
	for _, a := range bolt.Abilities() {
		if sa, ok := a.(*mage.SpellAbility); ok && sa.Kind() == mage.ActionSpell {
			boltEffects = sa.Effects()
		}
	}
	g.PushStack(&mage.StackObject{
		ID:         uuid.New(),
		Card:       bolt,
		Controller: pb.PlayerID(),
		Effects:    boltEffects,
		Targets:    []uuid.UUID{skeleton.ID()},
	})

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, false)

	if action.Type != interactive.ActionActivateAbility || action.PermanentID != skeleton.ID() {
		t.Fatalf("expected regeneration in response to lethal burn, got %+v", action)
	}
}

// Regeneration does not replace exile, so the AI should not waste its shield
// when the removal on the stack exiles the creature.
func TestPriorityAction_DoesNotRegenerateAgainstExile(t *testing.T) {
	g, pa, pb := makeGame()
	addLands(g, pa, "Swamp", 1)

	skeleton := regenerator("Drudge Skeletons", 1, 1, pa.PlayerID())
	g.AddToBattlefield(skeleton)

	g.PushStack(&mage.StackObject{
		ID:         uuid.New(),
		Controller: pb.PlayerID(),
		Effects:    []mage.Effect{mage.ExileTarget()},
		Targets:    []uuid.UUID{skeleton.ID()},
	})

	start := New(ai.MidrangeWeighted)
	action := start.PriorityAction(pa, g, 0, false)

	if action.Type == interactive.ActionActivateAbility {
		t.Fatalf("regeneration cannot stop exile; expected no activation, got %+v", action)
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
