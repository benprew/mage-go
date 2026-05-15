package mage

import (
	"testing"

	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
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
		{PermanentID: mountainID, Name: "Mountain", Colors: []Color{Red}, Amount: 1},
		{PermanentID: forestID, Name: "Forest", Colors: []Color{Green}, Amount: 1},
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
	for _, id := range sol.SourcesToTap {
		tappedSet[id] = true
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
