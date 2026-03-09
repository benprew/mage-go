package interactive

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// ── Test helpers ────────────────────────────────────────────────────────────

func makeSearchAI(config SearchConfig) *SearchStrategy {
	return &SearchStrategy{
		Config:    config,
		Evaluator: DefaultEvaluator,
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
	for i := 0; i < count; i++ {
		land := mage.NewLand(name, mage.WithManaAbility(color))
		land.SetOwner(p.PlayerID())
		perm := mage.NewPermanent(land, p.PlayerID())
		perm.RevokeBaseAttr(core.AttrSummonSick)
		g.Battlefield = append(g.Battlefield, perm)
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
	g.Battlefield = append(g.Battlefield, oppCreature)

	// Give player A a Lightning Bolt and a Mountain to cast it
	bolt := mage.NewInstant("Lightning Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)

	addLands(g, pa, "Mountain", 1)

	// Set up the game state for main phase
	g.Step = core.PrecombatMain

	strat := makeSearchAI(DefaultSearchConfig())
	action := strat.PriorityAction(pa, g, 0, true)

	if action.Type != ActionCastSpell {
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

	g.Step = core.PrecombatMain

	strat := makeSearchAI(DefaultSearchConfig())
	action := strat.PriorityAction(pa, g, 0, true)

	if action.Type != ActionCastSpell {
		t.Fatalf("expected ActionCastSpell, got %v", action.Type)
	}
	// Search should prefer the 5/5 (higher board impact)
	if action.CardName != "Force of Nature" {
		t.Errorf("search should prefer higher-value creature, got %s", action.CardName)
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
	g.Step = core.PrecombatMain

	// Use MaxNodes=1 so search exhausts budget immediately
	config := SearchConfig{
		MaxDepth:  3,
		MaxNodes:  1,
		TimeLimit: 500 * time.Millisecond,
	}
	strat := makeSearchAI(config)
	action := strat.PriorityAction(pa, g, 0, true)

	// Should still produce a valid action (fallback to heuristic)
	if action.Type != ActionCastSpell && action.Type != ActionPass {
		t.Errorf("expected valid action from fallback, got %v", action.Type)
	}
}

// ── Node Budget Respected ───────────────────────────────────────────────────

func TestSearch_NodeBudgetRespected(t *testing.T) {
	g, pa, _ := makeGame()

	// Add several spells to create many search branches
	for i := 0; i < 3; i++ {
		c := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		c.SetOwner(pa.PlayerID())
		pa.AddToHand(c)
	}
	addLands(g, pa, "Forest", 4)
	g.Step = core.PrecombatMain

	config := SearchConfig{
		MaxDepth:  4,
		MaxNodes:  50,
		TimeLimit: 5 * time.Second,
	}
	strat := makeSearchAI(config)

	// This should complete without hanging — the node budget caps the search
	action := strat.PriorityAction(pa, g, 0, true)
	_ = action // just verify it completes
}

// ── Move Generation ─────────────────────────────────────────────────────────

func TestGeneratePriorityMoves_IncludesLandPlay(t *testing.T) {
	g, pa, _ := makeGame()
	land := mage.NewLand("Forest")
	land.SetOwner(pa.PlayerID())
	pa.AddToHand(land)

	moves := GeneratePriorityMoves(g, pa, 0, true)
	foundLand := false
	for _, m := range moves {
		if m.Type == ActionPlayLand {
			foundLand = true
		}
	}
	if !foundLand {
		t.Error("expected land play in moves")
	}
}

func TestGeneratePriorityMoves_NoLandIfAlreadyPlayed(t *testing.T) {
	g, pa, _ := makeGame()
	land := mage.NewLand("Forest")
	land.SetOwner(pa.PlayerID())
	pa.AddToHand(land)

	moves := GeneratePriorityMoves(g, pa, 1, true)
	for _, m := range moves {
		if m.Type == ActionPlayLand {
			t.Error("should not offer land play when already played one")
		}
	}
}

func TestGeneratePriorityMoves_AlwaysIncludesPass(t *testing.T) {
	g, pa, _ := makeGame()
	moves := GeneratePriorityMoves(g, pa, 0, true)
	foundPass := false
	for _, m := range moves {
		if m.Type == ActionPass {
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
	g.Step = core.PrecombatMain

	moves := GeneratePriorityMoves(g, pa, 0, true)
	// Verify moves are sorted by heuristic descending (pass should be last)
	if len(moves) < 2 {
		t.Fatalf("expected at least 2 moves, got %d", len(moves))
	}
	lastMove := moves[len(moves)-1]
	if lastMove.Type != ActionPass {
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
	g.Battlefield = append(g.Battlefield, c)

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
	g.Battlefield = append(g.Battlefield, c1, c2)

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
	g.Battlefield = append(g.Battlefield, flyer, ground)

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

	clone := cloneGameForSearch(g)
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

	clone := cloneGameForSearch(g)
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
	g.Battlefield = append(g.Battlefield, c)

	clone := cloneGameForSearch(g)
	if len(clone.Battlefield) != 1 {
		t.Fatalf("clone battlefield should have 1 permanent, got %d", len(clone.Battlefield))
	}
	cp := clone.Battlefield[0]
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

	clone := cloneGameForSearch(g)
	clonePA := clone.GetPlayer(pa.PlayerID())
	if len(clonePA.Hand()) != 1 {
		t.Errorf("clone PA hand size = %d, want 1", len(clonePA.Hand()))
	}
}

// ── Move Application ────────────────────────────────────────────────────────

func TestApplyMoveToClone_LandPlay(t *testing.T) {
	g, pa, _ := makeGame()
	land := mage.NewLand("Forest")
	land.SetOwner(pa.PlayerID())
	pa.AddToHand(land)

	clone := cloneGameForSearch(g)

	m := &Move{
		Type:   ActionPlayLand,
		CardID: land.ID(),
	}
	applyMoveToClone(clone, pa.PlayerID(), m, 0)

	clonePA := clone.GetPlayer(pa.PlayerID())
	if len(clonePA.Hand()) != 0 {
		t.Errorf("hand should be empty after land play, got %d", len(clonePA.Hand()))
	}
	if clone.LandsPlayedThisTurn != 1 {
		t.Errorf("lands played should be 1, got %d", clone.LandsPlayedThisTurn)
	}
	// Check that a land appeared on the battlefield
	foundLand := false
	for _, perm := range clone.Battlefield {
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
	creature := mage.NewCreature("Bear", "{1}{G}", 2, 2)
	creature.SetOwner(pa.PlayerID())
	pa.AddToHand(creature)
	addLands(g, pa, "Forest", 2)

	clone := cloneGameForSearch(g)

	m := &Move{
		Type:     ActionCastSpell,
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
	for _, perm := range clone.Battlefield {
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
	bolt := mage.NewInstant("Lightning Bolt", "{R}",
		mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
	)
	bolt.SetOwner(pa.PlayerID())
	pa.AddToHand(bolt)
	addLands(g, pa, "Mountain", 1)

	clone := cloneGameForSearch(g)

	m := &Move{
		Type:     ActionCastSpell,
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
	g.Battlefield = append(g.Battlefield, c)

	strat := makeSearchAI(DefaultSearchConfig())
	attackers := strat.Attackers(pa, g)

	// With opponent at 5 life and no blockers, search should attack
	if len(attackers) == 0 {
		t.Error("search should attack when opponent is at low life")
	}
}

// ── Search Blockers ─────────────────────────────────────────────────────────

func TestSearch_Blockers_FallsBackToHeuristic(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Giant", "{3}{R}", 3, 3, pa.PlayerID())
	blk := makePerm("Wall", "{1}{W}", 0, 4, pb.PlayerID())
	g.Battlefield = append(g.Battlefield, atk, blk)
	g.Combat.AddAttacker(atk.ID(), pb.PlayerID())

	strat := makeSearchAI(DefaultSearchConfig())
	blocks := strat.Blockers(pb, g)

	// Should delegate to heuristic (ControlPersonality blocks power >= 3, MidrangePersonality blocks >=3)
	// MidrangePersonality has BlockPowerThreshold=3, so atkPow=3 triggers blocking
	if len(blocks) != 1 {
		t.Errorf("blockers should fall back to heuristic, expected 1 block, got %d", len(blocks))
	}
}

// ── NewSearchAI Constructor ─────────────────────────────────────────────────

func TestNewSearchAI_Constructor(t *testing.T) {
	ai := NewSearchAI("SearchBot", DefaultSearchConfig(), MidrangePersonality)
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

func BenchmarkSearch_TypicalBoard_Depth2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		g, pa, pb := makeGame()
		pa.SetLife(20)
		pb.SetLife(15)

		// Build a typical mid-game board
		c1 := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
		c2 := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
		opp := makePerm("Ogre", "{2}{R}", 3, 3, pb.PlayerID())
		g.Battlefield = append(g.Battlefield, c1, c2, opp)
		addLands(g, pa, "Forest", 4)
		addLands(g, pb, "Mountain", 3)

		bolt := mage.NewInstant("Lightning Bolt", "{R}",
			mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
		)
		bolt.SetOwner(pa.PlayerID())
		pa.AddToHand(bolt)

		g.Step = core.PrecombatMain

		config := SearchConfig{MaxDepth: 2, MaxNodes: 5000, TimeLimit: 500 * time.Millisecond}
		strat := makeSearchAI(config)
		strat.PriorityAction(pa, g, 0, true)
	}
}

func BenchmarkSearch_TypicalBoard_Depth3(b *testing.B) {
	for i := 0; i < b.N; i++ {
		g, pa, pb := makeGame()
		pa.SetLife(20)
		pb.SetLife(15)

		c1 := makePerm("Bear", "{1}{G}", 2, 2, pa.PlayerID())
		c2 := makePerm("Elf", "{G}", 1, 1, pa.PlayerID())
		opp := makePerm("Ogre", "{2}{R}", 3, 3, pb.PlayerID())
		g.Battlefield = append(g.Battlefield, c1, c2, opp)
		addLands(g, pa, "Forest", 4)
		addLands(g, pb, "Mountain", 3)

		bolt := mage.NewInstant("Lightning Bolt", "{R}",
			mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.DealDamage(mage.Fixed(3))),
		)
		bolt.SetOwner(pa.PlayerID())
		pa.AddToHand(bolt)

		g.Step = core.PrecombatMain

		config := SearchConfig{MaxDepth: 3, MaxNodes: 5000, TimeLimit: 500 * time.Millisecond}
		strat := makeSearchAI(config)
		strat.PriorityAction(pa, g, 0, true)
	}
}
