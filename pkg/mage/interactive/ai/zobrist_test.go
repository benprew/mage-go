package ai

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

func TestZobrist_Deterministic(t *testing.T) {
	z := NewZobristTables()
	g, pa, pb := makeGame()
	pa.SetLife(20)
	pb.SetLife(18)
	addLands(g, pa, "Forest", 3)
	g.Battlefield = append(g.Battlefield, makePerm("Grizzly Bears", "{1}{G}", 2, 2, pa.PlayerID()))
	g.Step = core.PrecombatMain

	h1 := z.Hash(g)
	h2 := z.Hash(g)
	if h1 != h2 {
		t.Fatalf("Hash not deterministic: %x vs %x", h1, h2)
	}
	if h1 == 0 {
		t.Fatalf("Hash of non-empty game should not be zero")
	}
}

func TestZobrist_CloneEquivalent(t *testing.T) {
	z := NewZobristTables()
	g, pa, pb := makeGame()
	pa.SetLife(20)
	pb.SetLife(17)
	addLands(g, pa, "Mountain", 2)
	g.Battlefield = append(g.Battlefield, makePerm("Grizzly Bears", "{1}{G}", 2, 2, pa.PlayerID()))
	g.Battlefield = append(g.Battlefield, makePerm("Savannah Lions", "{W}", 2, 1, pb.PlayerID()))
	g.Step = core.PrecombatMain

	clone := g.Clone()
	if clone == nil {
		t.Fatal("Clone returned nil")
	}

	if z.Hash(g) != z.Hash(clone) {
		t.Fatalf("Hash of clone differs from original")
	}
}

func TestZobrist_TappedChangesHash(t *testing.T) {
	z := NewZobristTables()
	g, pa, _ := makeGame()
	perm := makePerm("Grizzly Bears", "{1}{G}", 2, 2, pa.PlayerID())
	g.Battlefield = append(g.Battlefield, perm)

	before := z.Hash(g)
	perm.Tapped = true
	after := z.Hash(g)
	if before == after {
		t.Fatalf("Tapping a creature should change the hash")
	}
}

func TestZobrist_LifeChangesHash(t *testing.T) {
	z := NewZobristTables()
	g, pa, _ := makeGame()
	pa.SetLife(20)
	before := z.Hash(g)
	pa.SetLife(19)
	after := z.Hash(g)
	if before == after {
		t.Fatalf("Changing life should change the hash")
	}
}

func TestZobrist_StepChangesHash(t *testing.T) {
	z := NewZobristTables()
	g, _, _ := makeGame()
	g.Step = core.PrecombatMain
	before := z.Hash(g)
	g.Step = core.PostcombatMain
	after := z.Hash(g)
	if before == after {
		t.Fatalf("Changing step should change the hash")
	}
}

func TestZobrist_DifferentPermanents(t *testing.T) {
	z := NewZobristTables()
	g1, pa1, _ := makeGame()
	g1.Battlefield = append(g1.Battlefield, makePerm("Grizzly Bears", "{1}{G}", 2, 2, pa1.PlayerID()))

	g2, pa2, _ := makeGame()
	g2.Battlefield = append(g2.Battlefield, makePerm("Savannah Lions", "{W}", 2, 1, pa2.PlayerID()))

	if z.Hash(g1) == z.Hash(g2) {
		t.Fatalf("Different creatures should produce different hashes")
	}
}

func TestZobrist_CountersChangeHash(t *testing.T) {
	z := NewZobristTables()
	g, pa, _ := makeGame()
	perm := makePerm("Grizzly Bears", "{1}{G}", 2, 2, pa.PlayerID())
	g.Battlefield = append(g.Battlefield, perm)

	before := z.Hash(g)
	perm.Counters[core.P1P1] = 2
	after := z.Hash(g)
	if before == after {
		t.Fatalf("Adding counters should change the hash")
	}
}

func TestZobrist_HandContentsChangeHash(t *testing.T) {
	z := NewZobristTables()
	g, pa, _ := makeGame()

	card1 := mage.NewCreature("Grizzly Bears", "{1}{G}", 2, 2)
	card1.SetOwner(pa.PlayerID())
	pa.AddToHand(card1)
	before := z.Hash(g)

	card2 := mage.NewCreature("Savannah Lions", "{W}", 2, 1)
	card2.SetOwner(pa.PlayerID())
	pa.AddToHand(card2)
	after := z.Hash(g)

	if before == after {
		t.Fatalf("Adding a card to hand should change the hash")
	}
}

func TestZobrist_CanPlayLandBit(t *testing.T) {
	z := NewZobristTables()
	g, _, _ := makeGame()
	g.Step = core.PrecombatMain

	// LandsPlayedThisTurn == 0: can play land → bit set.
	before := z.Hash(g)

	// LandsPlayedThisTurn == MaxLandPlays: can no longer play land → bit cleared.
	g.LandsPlayedThisTurn = g.MaxLandPlays()
	after := z.Hash(g)

	if before == after {
		t.Fatalf("Toggling canPlayLand should change the hash")
	}
}

func TestZobrist_EmptyGameStable(t *testing.T) {
	z := NewZobristTables()
	g, _, _ := makeGame()
	// Two fresh empty-state games should produce the same hash (modulo
	// per-player UUIDs, which are not hashed).
	g2, _, _ := makeGame()

	if z.Hash(g) != z.Hash(g2) {
		t.Fatalf("Two fresh games should hash identically")
	}
}

func TestSearchKey_SideToMoveDiffers(t *testing.T) {
	z := NewZobristTables()
	g, _, _ := makeGame()
	if z.SearchKey(g, true, 0) == z.SearchKey(g, false, 0) {
		t.Fatalf("maximizing vs minimizing should produce different search keys")
	}
}

func TestSearchKey_ChainCountDiffers(t *testing.T) {
	z := NewZobristTables()
	g, _, _ := makeGame()
	if z.SearchKey(g, true, 0) == z.SearchKey(g, true, 1) {
		t.Fatalf("chain count 0 vs 1 should produce different search keys")
	}
}

func TestSearchKey_ChainCountClampingStable(t *testing.T) {
	// Out-of-range chain counts must clamp, not index out of bounds.
	z := NewZobristTables()
	g, _, _ := makeGame()
	k1 := z.SearchKey(g, true, maxChainBuckets-1)
	k2 := z.SearchKey(g, true, maxChainBuckets+100)
	if k1 != k2 {
		t.Fatalf("chain counts above maxChainBuckets should clamp to the same key")
	}
}
