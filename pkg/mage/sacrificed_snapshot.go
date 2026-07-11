package mage

import "github.com/google/uuid"

// SacrificedSnapshot captures the last-known information about a permanent
// sacrificed as a cost. Effects in the same spell/ability resolution can read
// this via Game.LastSacrificed() to access the sacrificed creature's power,
// toughness, name, and ID — values that would otherwise be lost when the
// permanent moves to the graveyard before the effect resolves.
//
// Used by Fling, Momentous Fall, Lena Selfless Champion, Ghoulcaller Gisa,
// and any other card whose effect depends on the sacrificed creature's P/T.
type SacrificedSnapshot struct {
	ID         uuid.UUID
	Name       string
	Power      int
	Toughness  int
	Controller uuid.UUID
}

// CaptureSacrificed records a snapshot of a permanent immediately before it
// is sacrificed. Called by sacrifice-cost implementations during cost
// payment. Overwrites any previous snapshot — only the most recent
// sacrificed permanent is exposed.
func (g *Game) CaptureSacrificed(p *Permanent) {
	if p == nil {
		return
	}
	g.lastSacrificed = &SacrificedSnapshot{
		ID:         p.ID(),
		Name:       p.Name(),
		Power:      p.CurrentPower(g),
		Toughness:  p.CurrentToughness(g),
		Controller: p.ControllerID(),
	}
}

// LastSacrificed returns the most recently captured sacrifice snapshot, or
// nil if none has been recorded for the current resolution. Effects should
// nil-check.
func (g *Game) LastSacrificed() *SacrificedSnapshot {
	return g.lastSacrificed
}

// ClearSacrificed wipes the saved snapshot. Called after stack-object
// resolution completes.
func (g *Game) ClearSacrificed() {
	g.lastSacrificed = nil
}
