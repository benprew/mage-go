package mage

import (
	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// PlayableAction describes one legal action the given player could take at
// the current priority point. Used by oracle-replay to compare mage-go's
// available actions against XMage's recorded snapshot. Loose match: only
// Kind + SourceName are compared (text/cost/target candidates ignored).
type PlayableAction struct {
	Kind       string // "land" | "cast" | "activate"
	SourceName string
}

// Playable enumerates the actions the given player could take if they had
// priority right now. Scope of the v1 stub:
//   - lands in hand (if it's their main phase and they have a land drop left)
//   - non-land cards in hand whose mana cost the player can auto-tap-pay
//
// Activated abilities on the battlefield are NOT enumerated yet — XMage's
// recorder includes them so the cross-val diff will under-report on the
// mage-go side. That's still useful: a missing land/cast surfaces clearly,
// activated-ability mismatches are deferred until needed.
//
// Returns nil when the player isn't legally able to act (wrong phase,
// can't afford anything, etc.). Callers that need a stable empty slice
// should treat nil as equivalent.
func (g *Game) Playable(playerIdx int) []PlayableAction {
	if playerIdx < 0 || playerIdx >= len(g.players) {
		return nil
	}
	p := g.players[playerIdx]

	var out []PlayableAction

	mainPhase := g.step.IsMainPhase()
	isActive := g.ActivePlayerObj().PlayerID() == p.PlayerID()
	canPlayLand := mainPhase && isActive && g.landsPlayedThisTurn < g.MaxLandPlays()

	for _, c := range p.Hand() {
		if c.HasType(TypeLand) {
			if canPlayLand {
				out = append(out, PlayableAction{Kind: "land", SourceName: c.Name()})
			}
			continue
		}
		// Non-land: castable iff main phase + we can pay the mana cost.
		// (Instants would also be castable outside main, but XMage's
		// SelfPlayPlayer only casts in main phases too — keeping parity.)
		if !mainPhase {
			continue
		}
		// Affordability: an X spell with X=0 still needs to cover any fixed
		// generic + colored portion. We treat affordability the same way for
		// both X and non-X spells — X effectively just bumps the generic
		// portion, and we don't enumerate X choices here.
		mc := c.ManaCost()
		if mc.IsZero() || p.ManaPool().CanPay(mc, SpellContextForCard(c)) || g.canAutoTapForCost(p.PlayerID(), mc) {
			out = append(out, PlayableAction{Kind: "cast", SourceName: c.Name()})
		}
	}
	return out
}

// canAutoTapForCost reports whether the player has enough untapped mana
// sources to pay the given cost via AutoTapForCost. Read-only check —
// doesn't actually tap anything. Conservative approximation: we just
// count untapped lands the player controls vs. total mana required.
// False negatives are OK (we'll miss some legal casts that need careful
// color matching); false positives would inflate the playable set, so
// avoid them.
func (g *Game) canAutoTapForCost(playerID uuid.UUID, cost ManaCost) bool {
	required := cost.CMC()
	if required == 0 {
		return true
	}
	available := 0
	for _, perm := range g.battlefield {
		if perm.Controller != playerID {
			continue
		}
		if perm.Tapped {
			continue
		}
		if perm.HasType(TypeLand) {
			available++
		}
	}
	return available >= required
}
