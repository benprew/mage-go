package ai

import (
	"math/rand"
	"sync"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// ZobristTables holds the per-component random uint64 values used to compute a
// Zobrist hash of a game state. Tables keyed by card name grow on demand as
// new card names are encountered.
//
// All public methods are safe for concurrent use. The hot path (Hash) takes
// only a read lock when every card name it sees is already registered; the
// write lock is taken only to introduce a new name, which happens at most
// once per distinct card name per process.
type ZobristTables struct {
	Step         [13]uint64
	ActivePlayer [2]uint64
	CanPlayLand  uint64

	Life     [2][maxLifeTable]uint64
	HandSize [2][maxHandSize]uint64
	LibSize  [2][maxLibSize]uint64

	// Without SideToMove, two nodes sharing a board but different movers
	// would collide and serve each other wrong-perspective scores.
	SideToMove [2]uint64
	// Without ChainCount, two nodes on the same board at different points
	// in a multi-spell chain would collide despite having different
	// remaining move budgets before priority switches.
	ChainCount [maxChainBuckets]uint64

	mu          sync.RWMutex
	nameIndex   map[string]int
	handCard    [2][]uint64                          // [player][nameIdx]
	graveCard   [2][]uint64                          // [player][nameIdx]
	permCard    [][2][2][2]uint64                    // [nameIdx][ctrl][tapped][sick]
	permCounter [][core.NumCounters][maxCount]uint64 // [nameIdx][counterType][count]
	stackItem   [][2]uint64                          // [nameIdx][controller]
	rng         *rand.Rand
}

const (
	maxLifeTable    = 64
	maxHandSize     = 64
	maxLibSize      = 128
	maxCount        = 16 // 0..15 clamped
	maxChainBuckets = 8  // chain position 0..7
)

// zobristSeed is fixed so that hashes are reproducible across runs.
const zobristSeed = 0xC0FFEE

// DefaultZobrist is the package-level ZobristTables shared by SearchStrategy
// instances. Using a shared instance avoids redundant table initialization and
// name-index churn across multiple AI players within the same process.
var DefaultZobrist = NewZobristTables()

// NewZobristTables creates a fresh ZobristTables with random uint64 values
// initialized from a fixed seed. Tables keyed by card name start empty and
// grow the first time a new name is hashed.
func NewZobristTables() *ZobristTables {
	z := &ZobristTables{
		nameIndex: make(map[string]int),
		rng:       rand.New(rand.NewSource(zobristSeed)),
	}

	for i := range z.Step {
		z.Step[i] = z.rng.Uint64()
	}
	for i := range z.ActivePlayer {
		z.ActivePlayer[i] = z.rng.Uint64()
	}
	z.CanPlayLand = z.rng.Uint64()

	for p := range 2 {
		for i := range maxLifeTable {
			z.Life[p][i] = z.rng.Uint64()
		}
		for i := range maxHandSize {
			z.HandSize[p][i] = z.rng.Uint64()
		}
		for i := range maxLibSize {
			z.LibSize[p][i] = z.rng.Uint64()
		}
	}

	for i := range z.SideToMove {
		z.SideToMove[i] = z.rng.Uint64()
	}
	for i := range z.ChainCount {
		z.ChainCount[i] = z.rng.Uint64()
	}

	return z
}

// SearchKey returns a TT key for a position from the perspective of a search
// node. It XORs the state hash with side-to-move and chain-count keys so that
// two nodes on the same game state but different mover/chain position produce
// distinct TT keys.
func (z *ZobristTables) SearchKey(g *mage.Game, maximizing bool, chainCount int) uint64 {
	h := z.Hash(g)
	idx := 0
	if maximizing {
		idx = 1
	}
	h ^= z.SideToMove[idx]
	if chainCount < 0 {
		chainCount = 0
	}
	if chainCount >= maxChainBuckets {
		chainCount = maxChainBuckets - 1
	}
	h ^= z.ChainCount[chainCount]
	return h
}

// nameIdx returns the index for a card name, allocating new table entries
// (under the write lock) if the name is seen for the first time.
func (z *ZobristTables) nameIdx(name string) int {
	z.mu.RLock()
	if idx, ok := z.nameIndex[name]; ok {
		z.mu.RUnlock()
		return idx
	}
	z.mu.RUnlock()

	z.mu.Lock()
	defer z.mu.Unlock()
	if idx, ok := z.nameIndex[name]; ok {
		return idx
	}
	idx := len(z.nameIndex)
	z.nameIndex[name] = idx

	z.handCard[0] = append(z.handCard[0], z.rng.Uint64())
	z.handCard[1] = append(z.handCard[1], z.rng.Uint64())
	z.graveCard[0] = append(z.graveCard[0], z.rng.Uint64())
	z.graveCard[1] = append(z.graveCard[1], z.rng.Uint64())

	var pc [2][2][2]uint64
	for c := range 2 {
		for t := range 2 {
			for s := range 2 {
				pc[c][t][s] = z.rng.Uint64()
			}
		}
	}
	z.permCard = append(z.permCard, pc)

	var cc [core.NumCounters][maxCount]uint64
	for ct := range core.NumCounters {
		for n := range maxCount {
			cc[ct][n] = z.rng.Uint64()
		}
	}
	z.permCounter = append(z.permCounter, cc)

	var si [2]uint64
	si[0] = z.rng.Uint64()
	si[1] = z.rng.Uint64()
	z.stackItem = append(z.stackItem, si)

	return idx
}

// Hash computes a 64-bit Zobrist fingerprint of the given game state.
// Positions that differ in any hashed component produce different hashes
// (barring rare collisions inherent to 64-bit hashing).
//
// The hash covers: step, active player, whether the active player can still
// play a land, each player's life / hand size / library size, bag-of-names
// for each player's hand and graveyard, and every battlefield permanent's
// name/controller/tapped/summoning-sick state plus its counters. Stack
// objects contribute their card name and controller.
//
// XXX: intentional scope omissions. Adding any of these requires also adding
// them to the hash, or else the TT will return wrong scores when two
// positions differ only in an un-hashed field:
//   - g.Exile (no current card reads/writes exile in a way that changes
//     search value within a single decision)
//   - g.InstantsCastThisTurn (matters for cards like Ichneumon Druid —
//     unimplemented)
//   - g.CreatureDeathsThisTurn (matters for death-counting triggers —
//     unimplemented)
//   - Turn number, mana pool, combat tracking maps, targets on stack objects
func (z *ZobristTables) Hash(g *mage.Game) uint64 {
	if g == nil {
		return 0
	}

	var h uint64

	step := int(g.GetStep())
	if step >= 0 && step < len(z.Step) {
		h ^= z.Step[step]
	}

	if g.ActivePlayerIndex() >= 0 && g.ActivePlayerIndex() < len(z.ActivePlayer) {
		h ^= z.ActivePlayer[g.ActivePlayerIndex()]
	}

	if g.MaxLandPlays()-g.GetLandsPlayedThisTurn() > 0 {
		h ^= z.CanPlayLand
	}

	for pIdx, p := range g.AllPlayers() {
		if pIdx >= 2 {
			break
		}

		life := clampIdx(p.Life(), maxLifeTable)
		h ^= z.Life[pIdx][life]

		handSize := clampIdx(len(p.Hand()), maxHandSize)
		h ^= z.HandSize[pIdx][handSize]

		libSize := clampIdx(len(p.Library()), maxLibSize)
		h ^= z.LibSize[pIdx][libSize]

		for _, c := range p.Hand() {
			idx := z.nameIdx(c.Name())
			h ^= z.handCard[pIdx][idx]
		}

		for _, c := range p.Graveyard() {
			idx := z.nameIdx(c.Name())
			h ^= z.graveCard[pIdx][idx]
		}
	}

	for _, perm := range g.AllBattlefield() {
		if perm == nil || perm.PhasedOut {
			continue
		}
		idx := z.nameIdx(perm.Name())

		ctrl := 0
		if g.PlayerCount() >= 2 && perm.Controller == g.PlayerAt(1).PlayerID() {
			ctrl = 1
		}
		tapped := 0
		if perm.Tapped {
			tapped = 1
		}
		sick := 0
		if perm.HasAttr(core.AttrSummonSick) {
			sick = 1
		}
		h ^= z.permCard[idx][ctrl][tapped][sick]

		for ct := range core.NumCounters {
			n := perm.Counters[ct]
			if n == 0 {
				continue
			}
			h ^= z.permCounter[idx][ct][clampIdx(int(n), maxCount)]
		}
	}

	if g.GetStack() != nil {
		for _, obj := range g.StackObjects() {
			if obj == nil || obj.Card == nil {
				continue
			}
			idx := z.nameIdx(obj.Card.Name())
			ctrl := 0
			if g.PlayerCount() >= 2 && obj.Controller == g.PlayerAt(1).PlayerID() {
				ctrl = 1
			}
			h ^= z.stackItem[idx][ctrl]
		}
	}

	return h
}

func clampIdx(v, max int) int {
	if v < 0 {
		return 0
	}
	if v >= max {
		return max - 1
	}
	return v
}
