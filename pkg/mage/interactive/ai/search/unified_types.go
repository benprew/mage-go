package search

import (
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

var (
	negInf = math.Inf(-1)
	posInf = math.Inf(1)
)

type ChainStep struct {
	Phase  core.PhaseStep
	Player string

	Move      *Move
	Attackers []uuid.UUID
	Blocks    []mage.BlockAssignment
}

func (c ChainStep) String() string {
	switch {
	case c.Move != nil:
		return fmt.Sprintf("[%s] %s: %s", c.Phase, c.Player, describeMove(c.Move))
	case c.Attackers != nil:
		return fmt.Sprintf("[%s] %s attacks with %s", c.Phase, c.Player, idsToString(c.Attackers))
	case c.Blocks != nil:
		return fmt.Sprintf("[%s] %s blocks: %s", c.Phase, c.Player, blocksToString(c.Blocks))
	}
	return fmt.Sprintf("[%s] %s: <empty>", c.Phase, c.Player)
}

func describeMove(m *Move) string {
	name := m.CardName
	if name == "" {
		name = "<unknown>"
	}
	switch {
	case m.PermanentID != (uuid.UUID{}):
		return "activate " + name
	case m.Type == 0:
		return "pass"
	default:
		return "cast " + name
	}
}

func idsToString(ids []uuid.UUID) string {
	if len(ids) == 0 {
		return "(none)"
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = id.String()[:8]
	}
	return strings.Join(parts, ", ")
}

func blocksToString(bs []mage.BlockAssignment) string {
	if len(bs) == 0 {
		return "(none)"
	}
	parts := make([]string, len(bs))
	for i, b := range bs {
		parts[i] = fmt.Sprintf("%s->%s", b.BlockerID.String()[:8], b.AttackerID.String()[:8])
	}
	return strings.Join(parts, ", ")
}

type Result struct {
	Chain    []ChainStep
	Score    float64
	Nodes    int
	TTHits   int
	TTStores int
}
