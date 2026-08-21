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

func testSolverSource(id uuid.UUID, color Color, amount int) manaSourceInfo {
	return manaSourceInfo{
		PermanentID: id,
		Colors:      []Color{color},
		Abilities: []manaSourceAbility{{
			AbilityIndex: 0,
			Productions:  []ManaProduction{{Color: color, Amount: amount}},
		}},
	}
}
