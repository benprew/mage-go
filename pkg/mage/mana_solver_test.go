package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// Dual land (Tundra: W/U) should be usable for either color requirement.
// Before multi-color support the solver only saw the first declared color.
func TestAutoTapForCost_DualLandPaysEitherColor(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	// Mimics cards/limited/lands.go construction: two single-color mana
	// abilities on the same land.
	tundra := NewLand("Tundra",
		WithSubTypes("Plains", "Island"),
		WithManaAbility(White),
		WithManaAbility(Blue),
	)
	tundra.SetOwner(pid)
	perm := g.PutOnBattlefield(tundra, pid)
	perm.RevokeBaseAttr(AttrSummonSick)

	// Pay {U} from the dual — used to fail because the source was indexed
	// only by its first color (White).
	if err := g.AutoTapForCost(pid, ManaCost{Blue: 1}); err != nil {
		t.Fatalf("AutoTapForCost({U}) failed for dual: %v", err)
	}
	if !perm.Tapped {
		t.Error("expected Tundra to be tapped for {U}")
	}
}

// 4 Mountains + 1 Sol Ring, pay {5}: solver should use Sol Ring (2 mana)
// alongside 3 Mountains rather than tapping all 5 sources. The multi-mana
// efficiency switch kicks in when single-mana sources alone can't cover the
// remaining generic.
func TestAutoTapForCost_MultiManaSwitchAvoidsExtraTaps(t *testing.T) {
	g := newPriorityTestGame()
	pid := g.players[0].PlayerID()

	mountains := make([]*Permanent, 0, 4)
	for range 4 {
		m := NewLand("Mountain", WithManaAbility(Red))
		m.SetOwner(pid)
		p := g.PutOnBattlefield(m, pid)
		p.RevokeBaseAttr(AttrSummonSick)
		mountains = append(mountains, p)
	}
	ring := NewArtifact("Sol Ring", "{1}", WithMultiManaAbility(ManaProduction{Color: Colorless, Amount: 2}))
	ring.SetOwner(pid)
	rperm := g.PutOnBattlefield(ring, pid)
	rperm.RevokeBaseAttr(AttrSummonSick)

	if err := g.AutoTapForCost(pid, ManaCost{Generic: 5}); err != nil {
		t.Fatalf("AutoTapForCost({5}) failed: %v", err)
	}
	if !rperm.Tapped {
		t.Error("expected Sol Ring tapped (multi-mana for efficiency)")
	}
	tappedMountains := 0
	for _, m := range mountains {
		if m.Tapped {
			tappedMountains++
		}
	}
	if tappedMountains != 3 {
		t.Errorf("expected 3 Mountains tapped (Sol Ring covers 2), got %d", tappedMountains)
	}
}

// SolveMana is a pure function — exercise it directly without touching Game.
func TestSolveMana_PureWithoutGame(t *testing.T) {
	mountainID := uuid.New()
	forestID := uuid.New()
	sources := []manaSourceInfo{
		testSolverSource(mountainID, Red, 1),
		testSolverSource(forestID, Green, 1),
	}
	scores := []int{1, 1}
	pool := NewManaPool()

	sol, err := SolveMana(ManaSolverInputs{
		Pool:    pool,
		Cost:    ManaCost{Red: 1, Generic: 1},
		Sources: sources,
		Scores:  scores,
	})
	if err != nil {
		t.Fatalf("SolveMana failed: %v", err)
	}
	if len(sol.SourcesToTap) != 2 {
		t.Fatalf("expected 2 sources to tap, got %d", len(sol.SourcesToTap))
	}
	tappedSet := map[uuid.UUID]bool{}
	for _, tap := range sol.SourcesToTap {
		tappedSet[tap.PermanentID] = true
	}
	if !tappedSet[mountainID] {
		t.Error("expected Mountain in tap set (only R source)")
	}
	if !tappedSet[forestID] {
		t.Error("expected Forest in tap set (only generic source left)")
	}
}

// SolveMana returns an error when the cost is unpayable. Verifies it doesn't
// panic and stays a pure function.
func TestSolveMana_UnpayableReturnsError(t *testing.T) {
	sol, err := SolveMana(ManaSolverInputs{
		Pool:    NewManaPool(),
		Cost:    ManaCost{Red: 1},
		Sources: []manaSourceInfo{},
		Scores:  []int{},
	})
	if err == nil {
		t.Fatalf("expected error, got solution %+v", sol)
	}
}

func TestSolveMana_PreservationScoresDoNotMakeFeasibleCostFail(t *testing.T) {
	dualID := uuid.New()
	plainsID := uuid.New()

	sol, err := SolveMana(ManaSolverInputs{
		Pool: NewManaPool(),
		Cost: ManaCost{White: 1, Blue: 1},
		Sources: []manaSourceInfo{
			{
				PermanentID: dualID,
				Colors:      []Color{White, Blue},
				Abilities: []manaSourceAbility{
					{AbilityIndex: 0, Productions: []ManaProduction{{Color: White, Amount: 1}}},
					{AbilityIndex: 1, Productions: []ManaProduction{{Color: Blue, Amount: 1}}},
				},
			},
			testSolverSource(plainsID, White, 1),
		},
		Scores: []int{2, 11},
	})
	if err != nil {
		t.Fatalf("SolveMana rejected a feasible {W}{U} payment: %v", err)
	}
	if len(sol.SourcesToTap) != 2 {
		t.Fatalf("expected both sources in the plan, got %+v", sol.SourcesToTap)
	}
}

func TestSolveMana_CostedManaSourceReturnsExecutablePlan(t *testing.T) {
	landIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	prismID := uuid.New()
	pool := NewManaPool()

	sol, err := SolveMana(ManaSolverInputs{
		Pool: pool,
		Cost: ManaCost{Generic: 1, Blue: 1},
		Sources: []manaSourceInfo{
			testSolverSource(landIDs[0], Red, 1),
			testSolverSource(landIDs[1], Black, 1),
			testSolverSource(landIDs[2], Black, 1),
			{
				PermanentID: prismID,
				Colors:      []Color{White, Blue, Black, Red, Green},
				Abilities: []manaSourceAbility{{
					AbilityIndex: 7,
					Productions:  []ManaProduction{{Color: AnyColor, Amount: 1}},
					ManaCost:     ManaCost{Generic: 2},
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("SolveMana failed: %v", err)
	}
	if !CanSolveMana(ManaSolverInputs{
		Pool: pool,
		Cost: ManaCost{Generic: 1, Blue: 1},
		Sources: []manaSourceInfo{
			testSolverSource(landIDs[0], Red, 1),
			testSolverSource(landIDs[1], Black, 1),
			testSolverSource(landIDs[2], Black, 1),
			{
				PermanentID: prismID,
				Colors:      []Color{White, Blue, Black, Red, Green},
				Abilities: []manaSourceAbility{{
					AbilityIndex: 7,
					Productions:  []ManaProduction{{Color: AnyColor, Amount: 1}},
					ManaCost:     ManaCost{Generic: 2},
				}},
			},
		},
	}) {
		t.Fatal("CanSolveMana should use the same costed-source planning")
	}

	foundPrism := false
	for i, action := range sol.SourcesToTap {
		if action.PermanentID != prismID {
			continue
		}
		foundPrism = true
		if action.AbilityIndex != 7 || len(action.Productions) != 1 || action.Productions[0].Color != Blue {
			t.Fatalf("unexpected Prism action: %+v", action)
		}
		if i < 2 {
			t.Fatalf("Prism was planned before its {2} activation cost was funded: %+v", sol.SourcesToTap)
		}
	}
	if !foundPrism {
		t.Fatal("solution did not include the costed mana ability")
	}
}

func TestSolveMana_CanonicalizesEquivalentSources(t *testing.T) {
	const sourceCount = 30
	sources := make([]manaSourceInfo, sourceCount)
	for i := range sources {
		sources[i] = testSolverSource(uuid.New(), Red, 1)
	}

	stats := &manaSearchStats{}
	plan, ok := searchManaWithStats(ManaSolverInputs{
		Pool:    NewManaPool(),
		Cost:    ManaCost{Generic: sourceCount / 2},
		Sources: sources,
	}, stats)
	if !ok {
		t.Fatal("equivalent sources should pay the generic cost")
	}
	if len(plan) != sourceCount/2 {
		t.Fatalf("plan taps %d sources, want %d", len(plan), sourceCount/2)
	}
	if stats.ExpandedNodes > sourceCount+1 {
		t.Fatalf("search expanded %d nodes for %d equivalent sources", stats.ExpandedNodes, sourceCount)
	}
}

func TestSolveMana_DoesNotCanonicalizeDifferentPreservationScores(t *testing.T) {
	expensiveID := uuid.New()
	preferredID := uuid.New()
	plan, err := SolveMana(ManaSolverInputs{
		Pool: NewManaPool(),
		Cost: ManaCost{Generic: 1},
		Sources: []manaSourceInfo{
			testSolverSource(expensiveID, Red, 1),
			testSolverSource(preferredID, Red, 1),
		},
		Scores: []int{100, 0},
	})
	if err != nil {
		t.Fatalf("SolveMana failed: %v", err)
	}
	if len(plan.SourcesToTap) != 1 || plan.SourcesToTap[0].PermanentID != preferredID {
		t.Fatalf("solver ignored preservation scores: %+v", plan.SourcesToTap)
	}
}

func TestSolveMana_CanonicalizesEquivalentSourcesWhenCostIsImpossible(t *testing.T) {
	const sourceCount = 30
	sources := make([]manaSourceInfo, sourceCount)
	for i := range sources {
		sources[i] = testSolverSource(uuid.New(), Red, 1)
	}

	stats := &manaSearchStats{}
	if _, ok := searchManaWithStats(ManaSolverInputs{
		Pool:    NewManaPool(),
		Cost:    ManaCost{White: 1},
		Sources: sources,
	}, stats); ok {
		t.Fatal("Mountains should not pay a white mana cost")
	}
	if stats.ExpandedNodes > sourceCount+1 {
		t.Fatalf("impossible search expanded %d nodes for %d equivalent sources", stats.ExpandedNodes, sourceCount)
	}
}

func TestSolveMana_PrunesImpossibleColoredCostBeforeSubsetSearch(t *testing.T) {
	const sourceCount = 15
	sources := make([]manaSourceInfo, sourceCount)
	scores := make([]int, sourceCount)
	for i := range sources {
		sources[i] = testSolverSource(uuid.New(), Red, 1)
		scores[i] = i + 1
	}

	stats := &manaSearchStats{}
	if _, ok := searchManaWithStats(ManaSolverInputs{
		Pool:    NewManaPool(),
		Cost:    ManaCost{White: 1},
		Sources: sources,
		Scores:  scores,
	}, stats); ok {
		t.Fatal("Mountains should not pay a white mana cost")
	}
	if stats.ExpandedNodes != 1 {
		t.Fatalf("impossible colored cost expanded %d nodes, want only the root", stats.ExpandedNodes)
	}
}

func TestSolveMana_RestrictedProductionUsesSpellContext(t *testing.T) {
	workshopID := uuid.New()
	inputs := ManaSolverInputs{
		Pool: NewManaPool(),
		Cost: ManaCost{Generic: 3},
		Sources: []manaSourceInfo{{
			PermanentID: workshopID,
			Abilities: []manaSourceAbility{{
				AbilityIndex: 0,
				Productions: []ManaProduction{{
					Color:       Colorless,
					Amount:      3,
					Restriction: ArtifactSpellsOnly{},
				}},
			}},
		}},
	}

	artifactContext := SpellContextForCard(NewArtifact("Test Artifact", "{3}"))
	inputs.SpellContext = artifactContext
	if _, err := SolveMana(inputs); err != nil {
		t.Fatalf("restricted source should pay for an artifact spell: %v", err)
	}

	creatureContext := SpellContextForCard(NewCreature("Test Creature", "{3}", 3, 3))
	inputs.SpellContext = creatureContext
	if _, err := SolveMana(inputs); err == nil {
		t.Fatal("artifact-only source should not pay for a creature spell")
	}
}

func TestSolveMana_RestrictedProductionCannotPayManaAbilityCost(t *testing.T) {
	artifactContext := SpellContextForCard(NewArtifact("White Artifact", "{W}"))
	_, err := SolveMana(ManaSolverInputs{
		Pool:         NewManaPool(),
		Cost:         ManaCost{White: 1},
		SpellContext: artifactContext,
		Sources: []manaSourceInfo{
			{
				PermanentID: uuid.New(),
				Abilities: []manaSourceAbility{{
					Productions: []ManaProduction{{
						Color:       Colorless,
						Amount:      1,
						Restriction: ArtifactSpellsOnly{},
					}},
				}},
			},
			{
				PermanentID: uuid.New(),
				Abilities: []manaSourceAbility{{
					ManaCost:    ManaCost{Generic: 1},
					Productions: []ManaProduction{{Color: White, Amount: 1}},
				}},
			},
		},
	})
	if err == nil {
		t.Fatal("spell-only mana must not pay a mana ability activation cost")
	}
}

func TestPlannedProductionsAvailable_PreservesRestriction(t *testing.T) {
	original := []ManaProduction{{
		Color:       Colorless,
		Amount:      3,
		Restriction: ArtifactSpellsOnly{},
	}}
	if !plannedProductionsAvailable(original, original) {
		t.Fatal("matching restricted production should be available")
	}
	if plannedProductionsAvailable(original, []ManaProduction{{Color: Colorless, Amount: 3}}) {
		t.Fatal("planned production must not discard the original restriction")
	}
}

func TestManaProductionChoicesForDemand_PreservesRestriction(t *testing.T) {
	restriction := CreatureSpellsOnly{}
	choices := manaProductionChoicesForDemand([]ManaProduction{{
		Color:       AnyColor,
		Amount:      1,
		Restriction: restriction,
	}}, nil, manaChoiceCapsForCost(ManaCost{Blue: 1}, nil))
	for _, choice := range choices {
		for _, production := range choice.Productions {
			if production.Color == Blue && manaRestrictionIdentity(production.Restriction) == manaRestrictionIdentity(restriction) {
				return
			}
		}
	}
	t.Fatalf("restricted any-color choices omitted restricted blue mana: %+v", choices)
}

func TestManaProductionChoicesForDemand_BoundsIrrelevantSurplus(t *testing.T) {
	caps := manaChoiceCapsForCost(ManaCost{White: 1}, nil)
	choices := manaProductionChoicesForDemand([]ManaProduction{{
		Color:          AnyColor,
		Amount:         20,
		AnyCombination: true,
	}}, nil, caps)

	if len(choices) != 2 {
		t.Fatalf("20 mana in any combination for a one-white demand produced %d choices, want 2", len(choices))
	}
	foundWhite := false
	foundNoWhite := false
	for _, choice := range choices {
		var total, white int
		for _, production := range choice.Productions {
			total += production.Amount
			if production.Color == White {
				white += production.Amount
			}
		}
		if total != 20 {
			t.Fatalf("choice produced %d mana, want 20: %+v", total, choice)
		}
		foundWhite = foundWhite || white > 0
		foundNoWhite = foundNoWhite || white == 0
	}
	if !foundWhite || !foundNoWhite {
		t.Fatalf("choices must represent both paying and not paying white: %+v", choices)
	}
}

func TestManaProductionChoicesForDemand_DeduplicatesEquivalentAnyColorEntries(t *testing.T) {
	productions := make([]ManaProduction, 6)
	for i := range productions {
		productions[i] = ManaProduction{Color: AnyColor, Amount: 1}
	}
	choices := manaProductionChoicesForDemand(productions, nil, manaChoiceCaps{})
	if len(choices) != 1 {
		t.Fatalf("generic-only demand produced %d equivalent any-color choices, want 1", len(choices))
	}
	if got := productionsTotalAmount(choices[0].Productions); got != 6 {
		t.Fatalf("choice produces %d mana, want 6: %+v", got, choices[0])
	}
}

func TestManaChoiceCapsIncludeColoredActivationDemand(t *testing.T) {
	inputs := ManaSolverInputs{
		Pool: NewManaPool(),
		Cost: ManaCost{Generic: 1},
		Sources: []manaSourceInfo{
			{
				PermanentID: uuid.New(),
				Abilities: []manaSourceAbility{{
					AbilityIndex: 0,
					Productions: []ManaProduction{{
						Color:          AnyColor,
						Amount:         20,
						AnyCombination: true,
					}},
				}},
			},
			{
				PermanentID: uuid.New(),
				Abilities: []manaSourceAbility{{
					AbilityIndex: 1,
					ManaCost:     ManaCost{Blue: 1},
					Productions:  []ManaProduction{{Color: Colorless, Amount: 1}},
				}},
			},
		},
	}
	caps := manaChoiceCapsForInputs(inputs)
	if caps[Blue] != 1 {
		t.Fatalf("blue demand cap is %d, want 1", caps[Blue])
	}
	choices := manaProductionChoicesForDemand(inputs.Sources[0].Abilities[0].Productions, nil, caps)
	foundBlue := false
	for _, choice := range choices {
		for _, production := range choice.Productions {
			foundBlue = foundBlue || production.Color == Blue
		}
	}
	if !foundBlue {
		t.Fatalf("bounded choices omitted mana needed for a source activation: %+v", choices)
	}
}

func BenchmarkSolveMana_LargeAnyCombination(b *testing.B) {
	inputs := ManaSolverInputs{
		Pool: NewManaPool(),
		Cost: ManaCost{Generic: 3, White: 1},
		Sources: []manaSourceInfo{{
			PermanentID: uuid.New(),
			Abilities: []manaSourceAbility{{
				Productions: []ManaProduction{{
					Color:          AnyColor,
					Amount:         20,
					AnyCombination: true,
				}},
			}},
		}},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := SolveMana(inputs); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSolveMana_ManyEquivalentSources(b *testing.B) {
	const sourceCount = 30
	sources := make([]manaSourceInfo, sourceCount)
	for i := range sources {
		sources[i] = testSolverSource(uuid.New(), Red, 1)
	}
	inputs := ManaSolverInputs{
		Pool:    NewManaPool(),
		Cost:    ManaCost{Generic: sourceCount / 2},
		Sources: sources,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := SolveMana(inputs); err != nil {
			b.Fatal(err)
		}
	}
}

func testSolverSource(id uuid.UUID, color Color, amount int) manaSourceInfo {
	return manaSourceInfo{
		PermanentID: id,
		Colors:      []Color{color},
		Abilities: []manaSourceAbility{{
			AbilityIndex:    0,
			Productions:     []ManaProduction{{Color: color, Amount: amount}},
			Interchangeable: true,
		}},
	}
}
