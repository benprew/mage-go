package search

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type simpleTTFlag uint8

const (
	simpleTTExact simpleTTFlag = iota
	simpleTTLowerBound
	simpleTTUpperBound
)

type simpleTTEntry struct {
	key   uint64
	score float64
	flag  simpleTTFlag
}

const (
	ttPrevPassKey  uint64 = 0x9e3779b97f4a7c15
	ttPriorityP0   uint64 = 0xbf58476d1ce4e5b9
	ttPriorityP1   uint64 = 0x94d049bb133111eb
	ttPriorityNone uint64 = 0xd6e8feb86659fd93
)

func (s *searcher) ttKey(g *mage.Game, priorityID uuid.UUID, prevPass bool) uint64 {
	if g == nil {
		return 0
	}
	z := s.zobrist
	if z == nil {
		z = DefaultZobrist
	}
	h := z.Hash(g)
	h ^= ttSupplementalKey(g)
	switch {
	case g.PlayerCount() > 0 && priorityID == g.PlayerAt(0).PlayerID():
		h ^= ttPriorityP0
	case g.PlayerCount() > 1 && priorityID == g.PlayerAt(1).PlayerID():
		h ^= ttPriorityP1
	default:
		h ^= ttPriorityNone
	}
	if prevPass {
		h ^= ttPrevPassKey
	}
	return h
}

func ttSupplementalKey(g *mage.Game) uint64 {
	h := uint64(1469598103934665603)
	mix := func(v byte) {
		h ^= uint64(v)
		h *= 1099511628211
	}
	mixString := func(s string) {
		for i := 0; i < len(s); i++ {
			mix(s[i])
		}
		mix(0)
	}
	mixUUID := func(id uuid.UUID) {
		for _, b := range id {
			mix(b)
		}
	}
	mixInt := func(n int) {
		for range 8 {
			mix(byte(n))
			n >>= 8
		}
	}

	mixInt(int(g.GetStep()))
	mixInt(g.ActivePlayerIndex())
	mixInt(g.GetLandsPlayedThisTurn())

	for i, perm := range g.AllBattlefield() {
		if perm == nil || perm.PhasedOut {
			continue
		}
		mixInt(i)
		mixString(perm.Name())
		mixUUID(perm.ID())
		mixUUID(perm.Controller)
		if perm.Tapped {
			mix(1)
		} else {
			mix(0)
		}
		if perm.HasAttr(core.AttrSummonSick) {
			mix(1)
		} else {
			mix(0)
		}
		mixUUID(perm.AttachedTo)
		for _, id := range perm.Attachments {
			mixUUID(id)
		}
		for ct := range core.NumCounters {
			if perm.Counters[ct] == 0 {
				continue
			}
			mixInt(int(ct))
			mixInt(int(perm.Counters[ct]))
		}
	}

	for _, group := range g.CombatGroups() {
		mixUUID(group.AttackerID)
		mixUUID(group.DefenderID)
		for _, id := range group.BlockerIDs {
			mixUUID(id)
		}
	}

	return h
}

func (s *searcher) probeTT(key uint64, alpha, beta float64) (simpleTTEntry, bool) {
	if s.tt == nil {
		return simpleTTEntry{}, false
	}
	s.ttProbes++
	entry, ok := s.tt[key]
	if !ok || entry.key != key {
		return simpleTTEntry{}, false
	}
	switch entry.flag {
	case simpleTTExact:
		return entry, true
	case simpleTTLowerBound:
		if entry.score >= beta {
			return entry, true
		}
	case simpleTTUpperBound:
		if entry.score <= alpha {
			return entry, true
		}
	}
	return simpleTTEntry{}, false
}

func (s *searcher) storeTT(key uint64, _ []ChainStep, score, alphaOrig, betaOrig float64) {
	if s.tt == nil {
		return
	}
	flag := simpleTTExact
	if score <= alphaOrig {
		flag = simpleTTUpperBound
	} else if score >= betaOrig {
		flag = simpleTTLowerBound
	}
	s.tt[key] = simpleTTEntry{key: key, score: score, flag: flag}
	s.ttStores++
}
