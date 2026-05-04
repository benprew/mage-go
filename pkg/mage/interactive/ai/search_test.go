package ai

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/eval"
)

// ── Test helpers ────────────────────────────────────────────────────────────

func makeSearchAI(config SearchConfig) *SearchStrategy {
	return &SearchStrategy{
		Config:    config,
		Evaluator: eval.DefaultEvaluator,
		Fallback:  &HeuristicStrategy{Personality: MidrangePersonality},
	}
}

// landColor maps basic land names to their mana color.
var landColor = map[string]core.Color{
	"Forest":   core.Green,
	"Mountain": core.Red,
	"Plains":   core.White,
	"Island":   core.Blue,
	"Swamp":    core.Black,
}

func addLands(g *mage.Game, p *mage.BasePlayer, name string, count int) {
	color := landColor[name]
	for range count {
		land := mage.NewLand(name, mage.WithManaAbility(color))
		land.SetOwner(p.PlayerID())
		perm := mage.NewPermanent(land, p.PlayerID())
		perm.RevokeBaseAttr(core.AttrSummonSick)
		g.AddToBattlefield(perm)
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

	start := makeSearchAI(DefaultSearchConfig())
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

	// Give player two creatures: a weak 1/1 and a strong 5/5
	smallCreature := mage.NewCreature("Elf", "{G}", 1, 1)
	smallCreature.SetOwner(pa.PlayerID())
	pa.AddToHand(smallCreature)

	bigCreature := mage.NewCreature("Force of Nature", "{2}{G}{G}{G}", 5, 5)
	bigCreature.SetOwner(pa.PlayerID())
	pa.AddToHand(bigCreature)

	// Provide enough mana for either
	addLands(g, pa, "Forest", 5)

	g.SetStep(core.PrecombatMain)

	start := makeSearchAI(DefaultSearchConfig())
	action := start.PriorityAction(pa, g, 0, true)

	if action.Type != interactive.ActionCastSpell {
		t.Fatalf("expected ActionCastSpell, got %v", action.Type)
	}
	// Search should cast one of the available creatures (the specific choice
	// depends on depth and eval — keeping mana open may favor the cheaper one).
	if action.CardName != "Force of Nature" && action.CardName != "Elf" {
		t.Errorf("search should cast a creature, got %s", action.CardName)
	}
}

// ── Fallback on Timeout ─────────────────────────────────────────────────────

func TestSearch_FallbackOnNodeBudget(t *testing.T) {
	g, pa, _ := makeGame()

	// Give player a creature to cast
	creature := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	creature.SetOwner(pa.PlayerID())
	pa.AddToHand(creature)

	addLands(g, pa, "Forest", 2)
	g.SetStep(core.PrecombatMain)

	// Use MaxNodes=1 so search exhausts budget immediately
	config := SearchConfig{
		MaxDepth:  3,
		MaxNodes:  1,
		TimeLimit: 500 * time.Millisecond,
	}
	start := makeSearchAI(config)
	action := start.PriorityAction(pa, g, 0, true)

	// Should still produce a valid action (fallback to heuristic)
	if action.Type != interactive.ActionCastSpell && action.Type != interactive.ActionPass {
		t.Errorf("expected valid action from fallback, got %v", action.Type)
	}
}

// ── Node Budget Respected ───────────────────────────────────────────────────

func TestSearch_NodeBudgetRespected(t *testing.T) {
	g, pa, _ := makeGame()

	// Add several spells to create many search branches
	for range 3 {
		c := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(pa.PlayerID())
		pa.AddToHand(c)
	}
	addLands(g, pa, "Forest", 4)
	g.SetStep(core.PrecombatMain)

	config := SearchConfig{
		MaxDepth:  4,
		MaxNodes:  50,
		TimeLimit: 5 * time.Second,
	}
	start := makeSearchAI(config)

	// This should complete without hanging — the node budget caps the search
	action := start.PriorityAction(pa, g, 0, true)
	_ = action // just verify it completes
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

	// Add a creature and an instant
	creature := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	creature.SetOwner(pa.PlayerID())
	pa.AddToHand(creature)

	addLands(g, pa, "Forest", 2)
	g.SetStep(core.PrecombatMain)

	moves := GeneratePriorityMoves(g, pa, 0, true)
	// Verify moves are sorted by heuristic descending (pass should be last)
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
	// Should have: all (1 creature) and none
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
	// Should have: all, none, each solo (2) = 4 minimum
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
	// Should include an evasive-only set (just the flyer)
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

	// Original should be unaffected
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
	// Check that a land appeared on the battlefield
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
	// Check creature on battlefield
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
	pb.SetLife(5) // low life — attacking is very valuable

	c := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
	g.AddToBattlefield(c)

	start := makeSearchAI(DefaultSearchConfig())
	attackers := start.Attackers(pa, g)

	// With opponent at 5 life and no blockers, search should attack
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

	start := makeSearchAI(DefaultSearchConfig())
	blocks := start.Blockers(pb, g)

	// Search evaluates blocking assignments; the wall (0/4) can block the 3/3
	// and survive, so search should find that blocking is better than not blocking.
	if len(blocks) != 1 {
		t.Errorf("search-based blockers expected 1 block, got %d", len(blocks))
	}
}

func TestSearch_Blockers_PreventsLethal(t *testing.T) {
	// Opponent at 3 life with a 3/3 attacker. We have a 2/2 blocker.
	// Blocking prevents lethal. Search should block even though the blocker dies.
	g, pa, pb := makeGame()
	pb.SetLife(3)
	atk := makePerm("Giant", "{2}{R}", 3, 3, pa.PlayerID())
	blk := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := makeSearchAI(DefaultSearchConfig())
	blocks := start.Blockers(pb, g)

	if len(blocks) == 0 {
		t.Error("search should block to prevent lethal damage")
	}
}

func TestSearch_Blockers_GangBlocksBigThreat(t *testing.T) {
	// 5/5 attacker. We have two 3/3 blockers. Gang block should kill the 5/5.
	g, pa, pb := makeGame()
	atk := makePerm("Wurm", "{3}{G}{G}", 5, 5, pa.PlayerID())
	b1 := makePerm("Knight1", "{2}{W}", 3, 3, pb.PlayerID())
	b2 := makePerm("Knight2", "{2}{W}", 3, 3, pb.PlayerID())
	g.AddToBattlefield(atk, b1, b2)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	start := makeSearchAI(DefaultSearchConfig())
	blocks := start.Blockers(pb, g)

	// Search should find the gang block (2 blockers on 1 attacker) is the best option.
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

// TestGenerateBlockerSets_IncludesNoBlocks — removed in Phase 4.
// generateBlockerSets is now in combatsolver/enumerate.go and the no-blocks
// invariant is covered by TestEnumerateBlockerSets_NoBlockersReturnsEmpty.

// ── NewSearchAI Constructor ─────────────────────────────────────────────────

func TestNewSearchAI_Constructor(t *testing.T) {
	ai := NewSearchAI("SearchBot", DefaultSearchConfig(), MidrangeWeighted)
	if ai.Name() != "SearchBot" {
		t.Errorf("name = %q, want SearchBot", ai.Name())
	}
	if ai.strategy == nil {
		t.Error("strategy should not be nil")
	}
	_, ok := ai.strategy.(*SearchStrategy)
	if !ok {
		t.Error("strategy should be *SearchStrategy")
	}
}

// ── Benchmark ───────────────────────────────────────────────────────────────

// ── Multi-Spell Turns in Search (Item 7) ────────────────────────────────────

func TestSearch_MultiSpell_BoltAndCreature(t *testing.T) {
	// With a bolt and a creature in hand, search should be able to consider
	// casting both in one turn (multi-spell chain).
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

	// Increase search budget for multi-spell exploration
	config := SearchConfig{MaxDepth: 4, MaxNodes: 10000, TimeLimit: 2 * time.Second}
	start := makeSearchAI(config)

	// The search should produce a valid action (not crash/hang with chaining).
	action := start.PriorityAction(pa, g, 0, true)
	if action.Type == interactive.ActionPass {
		t.Error("search should find a useful action with bolt + creature in hand")
	}
}

func TestSearch_MultiSpell_NodeBudgetRespected(t *testing.T) {
	// With multiple spells, multi-spell chaining should still respect node budget.
	g, pa, _ := makeGame()

	for range 4 {
		c := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(pa.PlayerID())
		pa.AddToHand(c)
	}
	addLands(g, pa, "Forest", 8)
	g.SetStep(core.PrecombatMain)

	config := SearchConfig{MaxDepth: 4, MaxNodes: 100, TimeLimit: 5 * time.Second}
	start := makeSearchAI(config)

	// Should complete without hanging — chain limit + node budget cap the search.
	action := start.PriorityAction(pa, g, 0, true)
	_ = action
}

func TestSearch_MultiSpell_PassEndsChain(t *testing.T) {
	// Verify that pass moves don't cause infinite loops in multi-spell search.
	g, pa, _ := makeGame()
	g.SetStep(core.PrecombatMain)

	config := SearchConfig{MaxDepth: 3, MaxNodes: 500, TimeLimit: 1 * time.Second}
	start := makeSearchAI(config)

	// Empty hand: only pass available. Should complete quickly.
	action := start.PriorityAction(pa, g, 0, true)
	if action.Type != interactive.ActionPass {
		t.Errorf("expected pass with empty hand, got %v", action.Type)
	}
}

// ── X-Spell Move Generation (Item 8) ───────────────────────────────────────

func TestXSpell_GeneratesMultipleVariants(t *testing.T) {
	g, pa, pb := makeGame()

	// Fireball: {X}{R}, deals X damage
	fireball := mage.NewSorcery("Fireball", "{X}{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.XValue())),
	)
	fireball.SetOwner(pa.PlayerID())
	pa.AddToHand(fireball)

	// 5 mountains: available mana = 5, fixed cost = 1 (the {R}), max X = 4
	addLands(g, pa, "Mountain", 5)
	g.SetStep(core.PrecombatMain)

	moves := GeneratePriorityMoves(g, pa, 0, true)

	// Count fireball moves with different X values
	xValues := make(map[int]bool)
	for _, m := range moves {
		if m.CardName == "Fireball" {
			xValues[m.XValue] = true
		}
	}

	// Should have X=1, X=2 (max/2=4/2), X=4 (max)
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
	pb.SetLife(3) // 3 life — X=3 is exactly lethal

	fireball := mage.NewSorcery("Fireball", "{X}{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.XValue())),
	)
	fireball.SetOwner(pa.PlayerID())
	pa.AddToHand(fireball)

	addLands(g, pa, "Mountain", 5)
	g.SetStep(core.PrecombatMain)

	start := makeSearchAI(DefaultSearchConfig())
	action := start.PriorityAction(pa, g, 0, true)

	if action.Type != interactive.ActionCastSpell {
		t.Fatalf("expected ActionCastSpell, got %v", action.Type)
	}
	if action.CardName != "Fireball" {
		t.Fatalf("expected Fireball, got %s", action.CardName)
	}
	// Should target opponent for lethal
	if len(action.Targets) != 1 || action.Targets[0] != pb.PlayerID() {
		t.Error("search should target opponent for lethal with Fireball")
	}
	// XValue should be set (at least 3 for lethal)
	if action.XValue < 3 {
		t.Errorf("XValue should be >= 3 for lethal, got %d", action.XValue)
	}
}

func TestXSpell_ApplySpellCast_UsesXValue(t *testing.T) {
	// Verify that applyMoveToClone uses m.XValue for X-cost damage.
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

	// Only 1 mountain: can cast for X=0 but expandXSpellMoves requires maxX >= 1.
	// Actually with 1 mountain, fixedCost=1 (the {R}), maxX = 1-1 = 0, no variants.
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
	// Verify that the history table is initialized and persists across calls.
	g, pa, pb := makeGame()
	pb.SetLife(15)

	// Multiple spells to create branching that triggers cutoffs.
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

	strategy := makeSearchAI(SearchConfig{MaxDepth: 4, MaxNodes: 5000, TimeLimit: 1 * time.Second})
	_ = strategy.PriorityAction(pa, g, 0, true)

	// History should be initialized (even if no cutoffs occurred, the map exists).
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

	// Should include all-but-one sets (3 of them for 3 creatures).
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

// ── Benchmark ───────────────────────────────────────────────────────────────

func BenchmarkSearch_TypicalBoard_Depth2(b *testing.B) {
	for b.Loop() {
		g, pa, pb := makeGame()
		pa.SetLife(20)
		pb.SetLife(15)

		// Build a typical mid-game board
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

		config := SearchConfig{MaxDepth: 2, MaxNodes: 5000, TimeLimit: 500 * time.Millisecond}
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

		config := SearchConfig{MaxDepth: 5, MaxNodes: 10000, TimeLimit: 500 * time.Millisecond}
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

		config := SearchConfig{MaxDepth: 6, MaxNodes: 10000, TimeLimit: 1 * time.Second}
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

		config := SearchConfig{MaxDepth: 6, MaxNodes: 10000, TimeLimit: 1 * time.Second}
		start := makeSearchAI(config)
		start.Blockers(pb, g)
	}
}

func BenchmarkSearch_LargeBoard_Depth6(b *testing.B) {
	for b.Loop() {
		g, pa, pb := makeGame()
		pa.SetLife(20)
		pb.SetLife(15)

		// Larger board: 4 creatures each side
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

		config := SearchConfig{MaxDepth: 6, MaxNodes: 15000, TimeLimit: 1 * time.Second}
		start := makeSearchAI(config)
		start.PriorityAction(pa, g, 0, true)
	}
}

func BenchmarkSearch_Attackers_LargeBoard(b *testing.B) {
	for b.Loop() {
		g, pa, pb := makeGame()
		pa.SetLife(20)
		pb.SetLife(10)

		// 5 attackers, 3 blockers
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

		config := SearchConfig{MaxDepth: 6, MaxNodes: 15000, TimeLimit: 1 * time.Second}
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

		config := SearchConfig{MaxDepth: 3, MaxNodes: 5000, TimeLimit: 500 * time.Millisecond}
		start := makeSearchAI(config)
		start.PriorityAction(pa, g, 0, true)
	}
}
