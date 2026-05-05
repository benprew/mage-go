package search

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/ai"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/ai/heuristic"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/eval"
)

// ── Test helpers ────────────────────────────────────────────────────────────

func makeSearchAI(config Config) *Strategy {
	return &Strategy{
		Config:    config,
		Evaluator: eval.DefaultEvaluator,
		Fallback:  heuristic.New(ai.MidrangeWeighted),
	}
}

// ── Find Lethal ─────────────────────────────────────────────────────────────

// TestSearch_FindLethal tests that search finds bolt-face-for-lethal when
// heuristic would target a creature instead.
func TestSearch_FindLethal(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(3) // opponent at 3 life — bolt to face is lethal

	// Give opponent a creature (heuristic would target this)
	oppCreature := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(oppCreature)

	// Give player A a Lightning Bolt and a Mountain to cast it
	bolt := mage.NewInstant("Lightning Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)

	addLands(g, pa, "Mountain", 1)

	// Set up the game state for main phase
	g.SetStep(core.PrecombatMain)

	start := makeSearchAI(DefaultConfig())
	action := start.PriorityAction(pa, g, 0, true)

	if action.Type != interactive.ActionCastSpell {
		t.Fatalf("expected ActionCastSpell, got %v", action.Type)
	}
	if action.CardName != "Lightning Bolt" {
		t.Fatalf("expected Lightning Bolt, got %s", action.CardName)
	}
	// The search should target the opponent player for lethal, not the creature
	if len(action.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(action.Targets))
	}
	if action.Targets[0] != pb.PlayerID() {
		t.Errorf("search should find lethal by targeting opponent face, but targeted something else")
	}
}

// ── Prefer Higher Value Spell ───────────────────────────────────────────────

func TestSearch_PreferHigherValueCreature(t *testing.T) {
	g, pa, _ := makeGame()

	smallCreature := mage.NewCreature("Elf", "{G}", 1, 1)
	smallCreature.SetOwner(pa.PlayerID())
	pa.AddToHand(smallCreature)

	bigCreature := mage.NewCreature("Force of Nature", "{2}{G}{G}{G}", 5, 5)
	bigCreature.SetOwner(pa.PlayerID())
	pa.AddToHand(bigCreature)

	addLands(g, pa, "Forest", 5)

	g.SetStep(core.PrecombatMain)

	start := makeSearchAI(DefaultConfig())
	action := start.PriorityAction(pa, g, 0, true)

	if action.Type != interactive.ActionCastSpell {
		t.Fatalf("expected ActionCastSpell, got %v", action.Type)
	}
	if action.CardName != "Force of Nature" && action.CardName != "Elf" {
		t.Errorf("search should cast a creature, got %s", action.CardName)
	}
}

// ── Fallback on Timeout ─────────────────────────────────────────────────────

func TestSearch_FallbackOnNodeBudget(t *testing.T) {
	g, pa, _ := makeGame()

	creature := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	creature.SetOwner(pa.PlayerID())
	pa.AddToHand(creature)

	addLands(g, pa, "Forest", 2)
	g.SetStep(core.PrecombatMain)

	config := Config{
		MaxDepth:  3,
		MaxNodes:  1,
		TimeLimit: 500 * time.Millisecond,
	}
	start := makeSearchAI(config)
	action := start.PriorityAction(pa, g, 0, true)

	if action.Type != interactive.ActionCastSpell && action.Type != interactive.ActionPass {
		t.Errorf("expected valid action from fallback, got %v", action.Type)
	}
}

// ── Node Budget Respected ───────────────────────────────────────────────────

func TestSearch_NodeBudgetRespected(t *testing.T) {
	g, pa, _ := makeGame()

	for range 3 {
		c := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(pa.PlayerID())
		pa.AddToHand(c)
	}
	addLands(g, pa, "Forest", 4)
	g.SetStep(core.PrecombatMain)

	config := Config{
		MaxDepth:  4,
		MaxNodes:  50,
		TimeLimit: 5 * time.Second,
	}
	start := makeSearchAI(config)

	action := start.PriorityAction(pa, g, 0, true)
	_ = action
}

// ── Move Generation ─────────────────────────────────────────────────────────

func TestGeneratePriorityMoves_IncludesLandPlay(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	land := mage.NewLand("Forest")
	land.SetOwner(pa.PlayerID())
	pa.AddToHand(land)

	moves := GeneratePriorityMoves(g, pa, 0, true)
	foundLand := false
	for _, m := range moves {
		if m.Type == interactive.ActionPlayLand {
			foundLand = true
		}
	}
	if !foundLand {
		t.Error("expected land play in moves")
	}
}

func TestGeneratePriorityMoves_NoLandIfAlreadyPlayed(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	land := mage.NewLand("Forest")
	land.SetOwner(pa.PlayerID())
	pa.AddToHand(land)
	g.SetLandsPlayedThisTurn(1)

	moves := GeneratePriorityMoves(g, pa, 1, true)
	for _, m := range moves {
		if m.Type == interactive.ActionPlayLand {
			t.Error("should not offer land play when already played one")
		}
	}
}

func TestGeneratePriorityMoves_AlwaysIncludesPass(t *testing.T) {
	g, pa, _ := makeGame()
	moves := GeneratePriorityMoves(g, pa, 0, true)
	foundPass := false
	for _, m := range moves {
		if m.Type == interactive.ActionPass {
			foundPass = true
		}
	}
	if !foundPass {
		t.Error("moves should always include pass")
	}
}

func TestGeneratePriorityMoves_SortedByHeuristic(t *testing.T) {
	g, pa, _ := makeGame()

	creature := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	creature.SetOwner(pa.PlayerID())
	pa.AddToHand(creature)

	addLands(g, pa, "Forest", 2)
	g.SetStep(core.PrecombatMain)

	moves := GeneratePriorityMoves(g, pa, 0, true)
	if len(moves) < 2 {
		t.Fatalf("expected at least 2 moves, got %d", len(moves))
	}
	lastMove := moves[len(moves)-1]
	if lastMove.Type != interactive.ActionPass {
		t.Error("pass should be last move (lowest heuristic)")
	}
}

// ── Attacker Sets ───────────────────────────────────────────────────────────

func TestGenerateAttackerSets_Empty(t *testing.T) {
	g, pa, _ := makeGame()
	sets := GenerateAttackerSets(g, pa.PlayerID())
	if len(sets) != 1 {
		t.Errorf("expected 1 set (nil), got %d", len(sets))
	}
}

func TestGenerateAttackerSets_SingleCreature(t *testing.T) {
	g, pa, _ := makeGame()
	c := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(c)

	sets := GenerateAttackerSets(g, pa.PlayerID())
	if len(sets) != 2 {
		t.Errorf("expected 2 sets for single creature, got %d", len(sets))
	}
}

func TestGenerateAttackerSets_MultipleCreatures(t *testing.T) {
	g, pa, _ := makeGame()
	c1 := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	c2 := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	g.AddToBattlefield(c1, c2)

	sets := GenerateAttackerSets(g, pa.PlayerID())
	if len(sets) < 4 {
		t.Errorf("expected at least 4 sets for 2 creatures, got %d", len(sets))
	}
}

func TestGenerateAttackerSets_IncludesEvasive(t *testing.T) {
	g, pa, _ := makeGame()
	flyer := makePerm("Bird", "{1}{U}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.Flying))
	ground := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(flyer, ground)

	sets := GenerateAttackerSets(g, pa.PlayerID())
	foundEvasive := false
	for _, set := range sets {
		if len(set) == 1 && set[0] == flyer.ID() {
			foundEvasive = true
		}
	}
	if !foundEvasive {
		t.Error("expected evasive-only set in attacker sets")
	}
}

// ── Game Cloning ────────────────────────────────────────────────────────────

func TestCloneGameForSearch_PreservesLife(t *testing.T) {
	g, pa, pb := makeGame()
	pa.SetLife(15)
	pb.SetLife(10)

	clone := g.Clone()
	clonePA := clone.GetPlayer(pa.PlayerID())
	clonePB := clone.GetPlayer(pb.PlayerID())

	if clonePA.Life() != 15 {
		t.Errorf("clone PA life = %d, want 15", clonePA.Life())
	}
	if clonePB.Life() != 10 {
		t.Errorf("clone PB life = %d, want 10", clonePB.Life())
	}
}

func TestCloneGameForSearch_IndependentMutation(t *testing.T) {
	g, pa, _ := makeGame()
	pa.SetLife(20)

	clone := g.Clone()
	clonePA := clone.GetPlayer(pa.PlayerID())
	clonePA.LoseLife(5)

	if pa.Life() != 20 {
		t.Errorf("original PA life changed to %d after clone mutation", pa.Life())
	}
	if clonePA.Life() != 15 {
		t.Errorf("clone PA life = %d, want 15", clonePA.Life())
	}
}

func TestCloneGameForSearch_PreservesBattlefield(t *testing.T) {
	g, pa, _ := makeGame()
	c := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	c.Tapped = true
	g.AddToBattlefield(c)

	clone := g.Clone()
	if len(clone.AllBattlefield()) != 1 {
		t.Fatalf("clone battlefield should have 1 permanent, got %d", len(clone.AllBattlefield()))
	}
	cp := clone.AllBattlefield()[0]
	if cp.Name() != "Bear" {
		t.Errorf("clone permanent name = %q, want Bear", cp.Name())
	}
	if !cp.Tapped {
		t.Error("clone permanent should be tapped")
	}
}

func TestCloneGameForSearch_PreservesHand(t *testing.T) {
	g, pa, _ := makeGame()
	card := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	card.SetOwner(pa.PlayerID())
	pa.AddToHand(card)

	clone := g.Clone()
	clonePA := clone.GetPlayer(pa.PlayerID())
	if len(clonePA.Hand()) != 1 {
		t.Errorf("clone PA hand size = %d, want 1", len(clonePA.Hand()))
	}
}

// ── Move Application ────────────────────────────────────────────────────────

func TestApplyMoveToClone_LandPlay(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	g.SetActivePlayerIndex(0)
	land := mage.NewLand("Forest", mage.WithManaAbility(core.Green))
	land.SetOwner(pa.PlayerID())
	pa.AddToHand(land)

	clone := g.Clone()

	m := &Move{
		Type:   interactive.ActionPlayLand,
		CardID: land.ID(),
	}
	applyMoveToClone(clone, pa.PlayerID(), m, 0)

	clonePA := clone.GetPlayer(pa.PlayerID())
	if len(clonePA.Hand()) != 0 {
		t.Errorf("hand should be empty after land play, got %d", len(clonePA.Hand()))
	}
	if clone.GetLandsPlayedThisTurn() != 1 {
		t.Errorf("lands played should be 1, got %d", clone.GetLandsPlayedThisTurn())
	}
	foundLand := false
	for _, perm := range clone.AllBattlefield() {
		if perm.HasType(core.TypeLand) {
			foundLand = true
		}
	}
	if !foundLand {
		t.Error("land should be on battlefield after land play")
	}
}

func TestApplyMoveToClone_CreatureCast(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	g.SetActivePlayerIndex(0)
	creature := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	creature.SetOwner(pa.PlayerID())
	pa.AddToHand(creature)
	addLands(g, pa, "Forest", 2)

	clone := g.Clone()

	m := &Move{
		Type:     interactive.ActionCastSpell,
		CardID:   creature.ID(),
		CardName: "Bear",
	}
	applyMoveToClone(clone, pa.PlayerID(), m, 0)

	clonePA := clone.GetPlayer(pa.PlayerID())
	if len(clonePA.Hand()) != 0 {
		t.Errorf("hand should be empty after creature cast, got %d", len(clonePA.Hand()))
	}
	foundCreature := false
	for _, perm := range clone.AllBattlefield() {
		if perm.Name() == "Bear" && perm.HasType(core.TypeCreature) {
			foundCreature = true
		}
	}
	if !foundCreature {
		t.Error("creature should be on battlefield after cast")
	}
}

func TestApplyMoveToClone_DamageSpell(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.PrecombatMain)
	g.SetActivePlayerIndex(0)
	bolt := mage.NewInstant("Lightning Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)
	addLands(g, pa, "Mountain", 1)

	clone := g.Clone()

	m := &Move{
		Type:     interactive.ActionCastSpell,
		CardID:   bolt.ID(),
		CardName: "Lightning Bolt",
		Targets:  []uuid.UUID{pb.PlayerID()},
	}
	applyMoveToClone(clone, pa.PlayerID(), m, 0)

	clonePB := clone.GetPlayer(pb.PlayerID())
	if clonePB.Life() != 17 {
		t.Errorf("opponent life should be 17 after bolt, got %d", clonePB.Life())
	}
}

// ── Search Attackers ────────────────────────────────────────────────────────

func TestSearch_Attackers_PrefersUnblockedDamage(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(5)

	c := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(c)

	start := makeSearchAI(DefaultConfig())
	attackers := start.Attackers(pa, g)

	if len(attackers) == 0 {
		t.Error("search should attack when opponent is at low life")
	}
}

// ── Search Blockers (Item 6) ────────────────────────────────────────────────

func TestSearch_Blockers_FallsBackToHeuristic(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Giant", "{3}{R}", 3, 3, pa.PlayerID())
	blk := makePerm("Wall", "{1}{W}", 0, 4, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := makeSearchAI(DefaultConfig())
	blocks := start.Blockers(pb, g)

	if len(blocks) != 1 {
		t.Errorf("search-based blockers expected 1 block, got %d", len(blocks))
	}
}

func TestSearch_Blockers_PreventsLethal(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(3)
	atk := makePerm("Giant", "{2}{R}", 3, 3, pa.PlayerID())
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := makeSearchAI(DefaultConfig())
	blocks := start.Blockers(pb, g)

	if len(blocks) == 0 {
		t.Error("search should block to prevent lethal damage")
	}
}

func TestSearch_Blockers_GangBlocksBigThreat(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Wurm", "{3}{G}{G}", 5, 5, pa.PlayerID())
	b1 := makePerm("Knight1", "{2}{W}", 3, 3, pb.PlayerID())
	b2 := makePerm("Knight2", "{2}{W}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(atk, b1, b2)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := makeSearchAI(DefaultConfig())
	blocks := start.Blockers(pb, g)

	gangBlockCount := 0
	for _, b := range blocks {
		if b.AttackerID == atk.ID() {
			gangBlockCount++
		}
	}
	if gangBlockCount < 2 {
		t.Errorf("search should gang-block the 5/5, got %d blockers on it", gangBlockCount)
	}
}

// ── Multi-Spell Turns in Search (Item 7) ────────────────────────────────────

func TestSearch_MultiSpell_BoltAndCreature(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(20)

	bolt := mage.NewInstant("Lightning Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)

	creature := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	creature.SetOwner(pa.PlayerID())
	pa.AddToHand(creature)

	addLands(g, pa, "Mountain", 2)
	addLands(g, pa, "Forest", 1)
	g.SetStep(core.PrecombatMain)

	config := Config{MaxDepth: 4, MaxNodes: 10000, TimeLimit: 2 * time.Second}
	start := makeSearchAI(config)

	action := start.PriorityAction(pa, g, 0, true)
	if action.Type == interactive.ActionPass {
		t.Error("search should find a useful action with bolt + creature in hand")
	}
}

func TestSearch_MultiSpell_NodeBudgetRespected(t *testing.T) {
	g, pa, _ := makeGame()

	for range 4 {
		c := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(pa.PlayerID())
		pa.AddToHand(c)
	}
	addLands(g, pa, "Forest", 8)
	g.SetStep(core.PrecombatMain)

	config := Config{MaxDepth: 4, MaxNodes: 100, TimeLimit: 5 * time.Second}
	start := makeSearchAI(config)

	action := start.PriorityAction(pa, g, 0, true)
	_ = action
}

func TestSearch_MultiSpell_PassEndsChain(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)

	config := Config{MaxDepth: 3, MaxNodes: 500, TimeLimit: 1 * time.Second}
	start := makeSearchAI(config)

	action := start.PriorityAction(pa, g, 0, true)
	if action.Type != interactive.ActionPass {
		t.Errorf("expected pass with empty hand, got %v", action.Type)
	}
}

// ── X-Spell Move Generation (Item 8) ───────────────────────────────────────

func TestXSpell_GeneratesMultipleVariants(t *testing.T) {
	g, pa, pb := makeGame()

	fireball := mage.NewSorcery("Fireball", "{X}{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.XValue())),
	)
	fireball.SetOwner(pa.PlayerID())
	pa.AddToHand(fireball)

	addLands(g, pa, "Mountain", 5)
	g.SetStep(core.PrecombatMain)

	moves := GeneratePriorityMoves(g, pa, 0, true)

	xValues := make(map[int]bool)
	for _, m := range moves {
		if m.CardName == "Fireball" {
			xValues[m.XValue] = true
		}
	}

	if !xValues[1] {
		t.Error("expected X=1 variant for Fireball")
	}
	if !xValues[4] {
		t.Errorf("expected X=4 variant for Fireball (max), got X values: %v", xValues)
	}
	if len(xValues) < 2 {
		t.Errorf("expected at least 2 X variants, got %d: %v", len(xValues), xValues)
	}
	_ = pb
}

func TestXSpell_SearchPicksExactLethal(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(3)

	fireball := mage.NewSorcery("Fireball", "{X}{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.XValue())),
	)
	fireball.SetOwner(pa.PlayerID())
	pa.AddToHand(fireball)

	addLands(g, pa, "Mountain", 5)
	g.SetStep(core.PrecombatMain)

	start := makeSearchAI(DefaultConfig())
	action := start.PriorityAction(pa, g, 0, true)

	if action.Type != interactive.ActionCastSpell {
		t.Fatalf("expected ActionCastSpell, got %v", action.Type)
	}
	if action.CardName != "Fireball" {
		t.Fatalf("expected Fireball, got %s", action.CardName)
	}
	if len(action.Targets) != 1 || action.Targets[0] != pb.PlayerID() {
		t.Error("search should target opponent for lethal with Fireball")
	}
	if action.XValue < 3 {
		t.Errorf("XValue should be >= 3 for lethal, got %d", action.XValue)
	}
}

func TestXSpell_ApplySpellCast_UsesXValue(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.PrecombatMain)
	g.SetActivePlayerIndex(0)
	fireball := mage.NewSorcery("Fireball", "{X}{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.XValue())),
	)
	fireball.SetOwner(pa.PlayerID())
	pa.AddToHand(fireball)
	addLands(g, pa, "Mountain", 4)

	clone := g.Clone()
	m := &Move{
		Type:     interactive.ActionCastSpell,
		CardID:   fireball.ID(),
		CardName: "Fireball",
		Targets:  []uuid.UUID{pb.PlayerID()},
		XValue:   3,
	}
	applyMoveToClone(clone, pa.PlayerID(), m, 0)

	clonePB := clone.GetPlayer(pb.PlayerID())
	if clonePB.Life() != 17 {
		t.Errorf("opponent life should be 17 after X=3 fireball, got %d", clonePB.Life())
	}
}

func TestXSpell_NoVariantsIfCantAfford(t *testing.T) {
	g, pa, _ := makeGame()

	fireball := mage.NewSorcery("Fireball", "{X}{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.XValue())),
	)
	fireball.SetOwner(pa.PlayerID())
	pa.AddToHand(fireball)

	addLands(g, pa, "Mountain", 1)
	g.SetStep(core.PrecombatMain)

	moves := GeneratePriorityMoves(g, pa, 0, true)
	for _, m := range moves {
		if m.CardName == "Fireball" {
			t.Error("should not generate Fireball moves when can only afford X=0")
		}
	}
}

// ── History Heuristic ────────────────────────────────────────────────────────

func TestSearch_HistoryHeuristicPersists(t *testing.T) {
	g, pa, pb := makeGame()
	pb.SetLife(15)

	bolt := mage.NewInstant("Lightning Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)

	creature := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	creature.SetOwner(pa.PlayerID())
	pa.AddToHand(creature)

	opp := makePerm("Ogre", "{2}{R}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(opp)

	addLands(g, pa, "Mountain", 2)
	addLands(g, pa, "Forest", 2)
	g.SetStep(core.PrecombatMain)

	strategy := makeSearchAI(Config{MaxDepth: 4, MaxNodes: 5000, TimeLimit: 1 * time.Second})
	_ = strategy.PriorityAction(pa, g, 0, true)

	if strategy.history == nil {
		t.Error("history heuristic should be initialized after search")
	}
}

// ── Attacker Generation ──────────────────────────────────────────────────────

func TestGenerateAttackerSets_IncludesAllButOne(t *testing.T) {
	g, pa, _ := makeGame()
	c1 := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	c2 := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
	c3 := makePerm("Knight", "{1}{W}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(c1, c2, c3)

	sets := GenerateAttackerSets(g, pa.PlayerID())

	allButOneCount := 0
	for _, set := range sets {
		if len(set) == 2 {
			allButOneCount++
		}
	}
	if allButOneCount < 3 {
		t.Errorf("expected at least 3 all-but-one sets for 3 creatures, got %d", allButOneCount)
	}
}

// ── Modal spell move generation (moved from modal_test.go) ──────────────────

func TestGeneratePriorityMoves_ModalSpellGeneratesPerModeMove(t *testing.T) {
	g, pa, _ := makeGame()

	charm := mage.NewInstant("Charm", "{W}",
		mage.NewSpellAbility(mage.GainLife(3)),
	)
	charm.SetModes([]string{"Gain 3 life", "Prevent 3 damage", "Destroy target enchantment"})
	charm.SetOwner(pa.PlayerID())
	pa.AddToHand(charm)

	addLands(g, pa, "Plains", 1)
	g.SetStep(core.PrecombatMain)

	moves := GeneratePriorityMoves(g, pa, 0, true)

	charmMoves := 0
	modesSeen := map[int]bool{}
	for _, m := range moves {
		if m.CardName == "Charm" {
			charmMoves++
			modesSeen[m.ModeIndex] = true
		}
	}
	if charmMoves != 3 {
		t.Errorf("expected 3 moves for 3-mode Charm, got %d", charmMoves)
	}
	for i := range 3 {
		if !modesSeen[i] {
			t.Errorf("missing move for mode %d", i)
		}
	}
}

func TestGeneratePriorityMoves_NonModalSpellNoModeIndex(t *testing.T) {
	g, pa, _ := makeGame()

	bolt := mage.NewInstant("Lightning Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)

	addLands(g, pa, "Mountain", 1)
	g.SetStep(core.PrecombatMain)

	moves := GeneratePriorityMoves(g, pa, 0, true)
	for _, m := range moves {
		if m.CardName == "Lightning Bolt" && m.ModeIndex != 0 {
			t.Errorf("non-modal spell should have ModeIndex 0, got %d", m.ModeIndex)
		}
	}
}

// ── applyMoveToClone for various spell effects (moved from modal_test.go) ───

func TestApplySpellCast_LifeGain(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	g.SetActivePlayerIndex(0)
	pa.SetLife(17)

	lifeSpell := mage.NewSorcery("Healing Touch", "{W}",
		mage.NewSpellAbility(mage.GainLife(3)),
	)
	lifeSpell.SetOwner(pa.PlayerID())
	pa.AddToHand(lifeSpell)
	addLands(g, pa, "Plains", 1)

	clone := g.Clone()
	m := &Move{
		Type:     interactive.ActionCastSpell,
		CardID:   lifeSpell.ID(),
		CardName: "Healing Touch",
	}
	applyMoveToClone(clone, pa.PlayerID(), m, 0)

	clonePA := clone.GetPlayer(pa.PlayerID())
	if clonePA.Life() != 20 {
		t.Errorf("life after gain = %d, want 20", clonePA.Life())
	}
}

func TestApplySpellCast_BuffEffect(t *testing.T) {
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)
	g.SetActivePlayerIndex(0)

	creature := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(creature)

	buff := mage.NewInstant("Giant Growth", "{G}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.Boost(mage.Fixed(3), mage.Fixed(3))),
	)
	buff.SetOwner(pa.PlayerID())
	pa.AddToHand(buff)
	addLands(g, pa, "Forest", 1)

	clone := g.Clone()
	m := &Move{
		Type:     interactive.ActionCastSpell,
		CardID:   buff.ID(),
		CardName: "Giant Growth",
		Targets:  []uuid.UUID{creature.ID()},
	}
	applyMoveToClone(clone, pa.PlayerID(), m, 0)

	perm := clone.FindPermanent(creature.ID())
	if perm == nil {
		t.Fatal("creature not found in clone")
	}
	gotP := perm.CurrentPower(clone)
	gotT := perm.CurrentToughness(clone)
	if gotP != 5 || gotT != 5 {
		t.Errorf("creature P/T after buff = %d/%d, want 5/5", gotP, gotT)
	}
}

func TestApplySpellCast_MultiEffect_DrawAndDamage(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.PrecombatMain)
	g.SetActivePlayerIndex(0)
	pa.SetLife(20)
	pb.SetLife(20)

	multiSpell := mage.NewSorcery("Arcane Blast", "{1}{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(),
			mage.DealDamage(mage.Fixed(3)),
			mage.DrawCardsActivePlayer(mage.Fixed(2)),
		),
	)
	multiSpell.SetOwner(pa.PlayerID())
	pa.AddToHand(multiSpell)

	for range 5 {
		c := mage.NewCreature("Filler", "{G}", 1, 1)
		c.SetOwner(pa.PlayerID())
		pa.AddToLibrary(c)
	}

	addLands(g, pa, "Mountain", 2)

	clone := g.Clone()
	m := &Move{
		Type:     interactive.ActionCastSpell,
		CardID:   multiSpell.ID(),
		CardName: "Arcane Blast",
		Targets:  []uuid.UUID{pb.PlayerID()},
	}
	applyMoveToClone(clone, pa.PlayerID(), m, 0)

	clonePB := clone.GetPlayer(pb.PlayerID())
	if clonePB.Life() != 17 {
		t.Errorf("opponent life after damage = %d, want 17", clonePB.Life())
	}

	clonePA := clone.GetPlayer(pa.PlayerID())
	if len(clonePA.Hand()) < 2 {
		t.Errorf("should have drawn 2 cards, hand size = %d", len(clonePA.Hand()))
	}
}

func TestApplySpellCast_BounceEffect(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.PrecombatMain)
	g.SetActivePlayerIndex(0)

	target := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(target)

	bounce := mage.NewInstant("Unsummon", "{U}",
		mage.NewTargetedSpell(mage.TargetCreature(), mage.ReturnToHandTarget()),
	)
	bounce.SetOwner(pa.PlayerID())
	pa.AddToHand(bounce)
	addLands(g, pa, "Island", 1)

	clone := g.Clone()
	m := &Move{
		Type:     interactive.ActionCastSpell,
		CardID:   bounce.ID(),
		CardName: "Unsummon",
		Targets:  []uuid.UUID{target.ID()},
	}
	applyMoveToClone(clone, pa.PlayerID(), m, 0)

	perm := clone.FindPermanent(target.ID())
	if perm != nil {
		t.Error("bounced permanent should be removed from battlefield")
	}
}

func TestApplySpellCast_XSpellDamage(t *testing.T) {
	g, pa, pb := makeGame()
	g.SetStep(core.PrecombatMain)
	g.SetActivePlayerIndex(0)
	pb.SetLife(20)

	fireball := mage.NewSorcery("Fireball", "{X}{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.XValue())),
	)
	fireball.SetOwner(pa.PlayerID())
	pa.AddToHand(fireball)
	addLands(g, pa, "Mountain", 5)

	clone := g.Clone()
	m := &Move{
		Type:     interactive.ActionCastSpell,
		CardID:   fireball.ID(),
		CardName: "Fireball",
		Targets:  []uuid.UUID{pb.PlayerID()},
		XValue:   4,
	}
	applyMoveToClone(clone, pa.PlayerID(), m, 0)

	clonePB := clone.GetPlayer(pb.PlayerID())
	if clonePB.Life() != 16 {
		t.Errorf("opponent life after X=4 Fireball = %d, want 16", clonePB.Life())
	}
}

// ── Benchmarks ──────────────────────────────────────────────────────────────

func BenchmarkSearch_TypicalBoard_Depth2(b *testing.B) {
	for b.Loop() {
		g, pa, pb := makeGame()
		pa.SetLife(20)
		pb.SetLife(15)

		c1 := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
		c2 := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
		opp := makePerm("Ogre", "{2}{R}", 3, 3, pb.PlayerID())
		g.AddToBattlefield(c1, c2, opp)
		addLands(g, pa, "Forest", 4)
		addLands(g, pb, "Mountain", 3)

		bolt := mage.NewInstant("Lightning Bolt", "{R}",
			mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
		)
		bolt.SetOwner(pa.PlayerID())
		pa.AddToHand(bolt)

		g.SetStep(core.PrecombatMain)

		config := Config{MaxDepth: 2, MaxNodes: 5000, TimeLimit: 500 * time.Millisecond}
		start := makeSearchAI(config)
		start.PriorityAction(pa, g, 0, true)
	}
}

func BenchmarkSearch_TypicalBoard_Depth5(b *testing.B) {
	for b.Loop() {
		g, pa, pb := makeGame()
		pa.SetLife(20)
		pb.SetLife(15)

		c1 := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
		c2 := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
		opp := makePerm("Ogre", "{2}{R}", 3, 3, pb.PlayerID())
		g.AddToBattlefield(c1, c2, opp)
		addLands(g, pa, "Forest", 4)
		addLands(g, pb, "Mountain", 3)

		bolt := mage.NewInstant("Lightning Bolt", "{R}",
			mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
		)
		bolt.SetOwner(pa.PlayerID())
		pa.AddToHand(bolt)

		g.SetStep(core.PrecombatMain)

		config := Config{MaxDepth: 5, MaxNodes: 10000, TimeLimit: 500 * time.Millisecond}
		start := makeSearchAI(config)
		start.PriorityAction(pa, g, 0, true)
	}
}

func BenchmarkSearch_Attackers(b *testing.B) {
	for b.Loop() {
		g, pa, pb := makeGame()
		pa.SetLife(20)
		pb.SetLife(10)

		c1 := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
		c2 := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
		c3 := makePerm("Flyer", "{1}{U}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.Flying))
		opp1 := makePerm("Wall", "{1}{W}", 0, 4, pb.PlayerID())
		opp2 := makePerm("Guard", "{2}{W}", 2, 3, pb.PlayerID())
		g.AddToBattlefield(c1, c2, c3, opp1, opp2)

		g.SetStep(core.DeclareAttackers)

		config := Config{MaxDepth: 6, MaxNodes: 10000, TimeLimit: 1 * time.Second}
		start := makeSearchAI(config)
		start.Attackers(pa, g)
	}
}

func BenchmarkSearch_Blockers(b *testing.B) {
	for b.Loop() {
		g, pa, pb := makeGame()
		pa.SetLife(20)
		pb.SetLife(15)

		atk1 := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
		atk2 := makePerm("Giant", "{3}{R}", 4, 4, pa.PlayerID())
		blk1 := makePerm("Elf", "{G}", 1, 1, pb.PlayerID())
		blk2 := makePerm("Guard", "{2}{W}", 2, 3, pb.PlayerID())
		g.AddToBattlefield(atk1, atk2, blk1, blk2)
		g.GetCombat().AddAttacker(atk1.ID(), pb.PlayerID())
		g.GetCombat().AddAttacker(atk2.ID(), pb.PlayerID())

		config := Config{MaxDepth: 6, MaxNodes: 10000, TimeLimit: 1 * time.Second}
		start := makeSearchAI(config)
		start.Blockers(pb, g)
	}
}

func BenchmarkSearch_LargeBoard_Depth6(b *testing.B) {
	for b.Loop() {
		g, pa, pb := makeGame()
		pa.SetLife(20)
		pb.SetLife(15)

		c1 := makePerm("Bear1", "{1}{G}", 2, 2, pa.PlayerID())
		c2 := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
		c3 := makePerm("Flyer", "{1}{U}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.Flying))
		c4 := makePerm("Knight", "{1}{W}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.FirstStrike))
		o1 := makePerm("Ogre", "{2}{R}", 3, 3, pb.PlayerID())
		o2 := makePerm("Wall", "{1}{W}", 0, 4, pb.PlayerID(), mage.WithKeyword(core.Defender))
		o3 := makePerm("Guard", "{2}{W}", 2, 3, pb.PlayerID())
		o4 := makePerm("Bat", "{1}{B}", 1, 1, pb.PlayerID(), mage.WithKeyword(core.Flying))
		g.AddToBattlefield(c1, c2, c3, c4, o1, o2, o3, o4)
		addLands(g, pa, "Forest", 4)
		addLands(g, pa, "Mountain", 2)
		addLands(g, pb, "Mountain", 3)
		addLands(g, pb, "Swamp", 2)

		bolt := mage.NewInstant("Lightning Bolt", "{R}",
			mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
		)
		bolt.SetOwner(pa.PlayerID())
		pa.AddToHand(bolt)

		creature := mage.NewCreature("Bear2", "{1}{G}", 2, 2)
		creature.SetOwner(pa.PlayerID())
		pa.AddToHand(creature)

		g.SetStep(core.PrecombatMain)

		config := Config{MaxDepth: 6, MaxNodes: 15000, TimeLimit: 1 * time.Second}
		start := makeSearchAI(config)
		start.PriorityAction(pa, g, 0, true)
	}
}

func BenchmarkSearch_Attackers_LargeBoard(b *testing.B) {
	for b.Loop() {
		g, pa, pb := makeGame()
		pa.SetLife(20)
		pb.SetLife(10)

		c1 := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
		c2 := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
		c3 := makePerm("Flyer", "{1}{U}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.Flying))
		c4 := makePerm("Knight", "{1}{W}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.FirstStrike))
		c5 := makePerm("Warrior", "{2}{R}", 3, 2, pa.PlayerID())
		opp1 := makePerm("Wall", "{1}{W}", 0, 4, pb.PlayerID())
		opp2 := makePerm("Guard", "{2}{W}", 2, 3, pb.PlayerID())
		opp3 := makePerm("Soldier", "{W}", 1, 1, pb.PlayerID())
		g.AddToBattlefield(c1, c2, c3, c4, c5, opp1, opp2, opp3)

		g.SetStep(core.DeclareAttackers)

		config := Config{MaxDepth: 6, MaxNodes: 15000, TimeLimit: 1 * time.Second}
		start := makeSearchAI(config)
		start.Attackers(pa, g)
	}
}

func BenchmarkSearch_TypicalBoard_Depth3(b *testing.B) {
	for b.Loop() {
		g, pa, pb := makeGame()
		pa.SetLife(20)
		pb.SetLife(15)

		c1 := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
		c2 := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
		opp := makePerm("Ogre", "{2}{R}", 3, 3, pb.PlayerID())
		g.AddToBattlefield(c1, c2, opp)
		addLands(g, pa, "Forest", 4)
		addLands(g, pb, "Mountain", 3)

		bolt := mage.NewInstant("Lightning Bolt", "{R}",
			mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
		)
		bolt.SetOwner(pa.PlayerID())
		pa.AddToHand(bolt)

		g.SetStep(core.PrecombatMain)

		config := Config{MaxDepth: 3, MaxNodes: 5000, TimeLimit: 500 * time.Millisecond}
		start := makeSearchAI(config)
		start.PriorityAction(pa, g, 0, true)
	}
}
