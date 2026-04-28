package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// PermanentLKI captures Last-Known-Information about a permanent at the
// moment it left the battlefield. Triggered abilities that reference the
// dying/leaving permanent (e.g. "whenever a creature an opponent controls
// dies") need its controller, types, and P/T after it has been moved to
// the graveyard. Per CR 603.10 and CR 113.7a, such triggers see the
// permanent's last existence on the battlefield.
//
// LKI is recorded by RemoveFromBattlefield just before the permanent leaves
// g.battlefield, and is read via Game.LKI(permID). Entries persist for the
// remainder of the current step so death-trigger conditions and effects can
// use them; they are cleared at end-of-turn cleanup.
type PermanentLKI struct {
	ID         uuid.UUID
	Name       string
	Controller uuid.UUID
	Owner      uuid.UUID
	Power      int
	Toughness  int
	Types      []CardType
	SubTypes   []string
	IsToken    bool
}

// HasType reports whether the LKI snapshot had the given card type.
func (l *PermanentLKI) HasType(t CardType) bool {
	for _, x := range l.Types {
		if x == t {
			return true
		}
	}
	return false
}

// HasSubType reports whether the LKI snapshot had the given subtype.
func (l *PermanentLKI) HasSubType(s string) bool {
	for _, x := range l.SubTypes {
		if x == s {
			return true
		}
	}
	return false
}

// captureLKI records an LKI snapshot for a permanent that is about to leave
// the battlefield. Idempotent — re-snapshotting overwrites.
func (g *Game) captureLKI(p *Permanent) {
	if p == nil {
		return
	}
	if g.lki == nil {
		g.lki = make(map[uuid.UUID]*PermanentLKI)
	}
	types := make([]CardType, 0, 4)
	for _, t := range []CardType{TypeCreature, TypeArtifact, TypeEnchantment, TypeLand, TypeInstant, TypeSorcery} {
		if p.HasType(t) {
			types = append(types, t)
		}
	}
	subTypes := make([]string, len(p.Card.SubTypes()))
	copy(subTypes, p.Card.SubTypes())
	isToken := false
	if bc, ok := p.Card.(*BaseCard); ok {
		isToken = bc.IsToken()
	}
	owner := p.Card.Owner()
	g.lki[p.ID()] = &PermanentLKI{
		ID:         p.ID(),
		Name:       p.Name(),
		Controller: p.Controller,
		Owner:      owner,
		Power:      p.CurrentPower(g),
		Toughness:  p.CurrentToughness(g),
		Types:      types,
		SubTypes:   subTypes,
		IsToken:    isToken,
	}
}

// LKI returns the last-known-information snapshot for a permanent that has
// left the battlefield, or nil if none is recorded. Triggered abilities that
// reference the dying permanent's controller, types, or P/T (e.g. "whenever
// a creature an opponent controls dies") should consult LKI when
// FindPermanent returns nil.
func (g *Game) LKI(id uuid.UUID) *PermanentLKI {
	if g.lki == nil {
		return nil
	}
	return g.lki[id]
}

// clearLKI wipes LKI entries. Called at the end of each turn's cleanup.
func (g *Game) clearLKI() {
	g.lki = nil
}
