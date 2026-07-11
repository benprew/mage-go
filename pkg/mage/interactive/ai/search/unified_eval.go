package search

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const (
	wPressure    = 1.0
	wBoardDiff   = 2.0
	wSpellInHand = 0.5
	wLandInHand  = 0.1
	wManaDiff    = 0.3

	handPermanentEffectScale = 0.85
	handRemovalEffectScale   = 0.45
	threatCeiling            = 4.0

	evFlyingNoAnswer = 2.0
	evMenaceFew      = 1.5

	terminalScore = 1_000_000.0
)

func evaluate(g *mage.Game, rootPlayer uuid.UUID) float64 {
	me := g.GetPlayer(rootPlayer)
	opp := g.GetOpponent(rootPlayer)
	if me == nil || opp == nil {
		return 0
	}

	myBoard, oppBoard := boardPower(g, rootPlayer)
	myMana, oppMana := untappedLands(g, rootPlayer)

	score := 0.0
	score += wPressure * (lifePressure(opp.Life()) - lifePressure(me.Life()))
	score += wBoardDiff * (myBoard - oppBoard)
	score += handValue(g, me) - handValue(g, opp)
	score += wManaDiff * float64(myMana-oppMana)
	score -= incomingDamagePenalty(g, me, opp)
	score += incomingDamagePenalty(g, opp, me)
	return score
}

func lifePressure(life int) float64 {
	if life >= 20 {
		return 0
	}
	delta := float64(20 - life)
	return delta * delta / 20.0
}

func boardPower(g *mage.Game, rootPlayer uuid.UUID) (float64, float64) {
	opp := g.GetOpponent(rootPlayer)
	if opp == nil {
		return 0, 0
	}

	var mine, theirs float64
	oppID := opp.PlayerID()
	for _, p := range g.AllBattlefield() {
		if !p.HasType(core.TypeCreature) {
			continue
		}
		base := float64(p.CurrentPower(g) + p.CurrentToughness(g))
		defender := rootPlayer
		if p.ControllerID() == rootPlayer {
			defender = oppID
		}
		v := base * evasionFactor(g, p, defender)
		if p.ControllerID() == rootPlayer {
			mine += v
		} else {
			theirs += v
		}
	}
	return mine, theirs
}

func evasionFactor(g *mage.Game, p *mage.Permanent, defenderID uuid.UUID) float64 {
	if p.HasAttr(core.Flying) && !defenderHasAnyAttr(g, defenderID, core.Flying, core.Reach) {
		return evFlyingNoAnswer
	}
	if p.HasAttr(core.Menace) && countDefenderCreatures(g, defenderID) <= 1 {
		return evMenaceFew
	}
	return 1.0
}

func defenderHasAnyAttr(g *mage.Game, defenderID uuid.UUID, attrs ...core.Attr) bool {
	for _, p := range g.AllBattlefield() {
		if p.ControllerID() != defenderID || !p.HasType(core.TypeCreature) {
			continue
		}
		if slices.ContainsFunc(attrs, p.HasAttr) {
			return true
		}
	}
	return false
}

func countDefenderCreatures(g *mage.Game, defenderID uuid.UUID) int {
	n := 0
	for _, p := range g.AllBattlefield() {
		if p.ControllerID() == defenderID && p.HasType(core.TypeCreature) {
			n++
		}
	}
	return n
}

func untappedLands(g *mage.Game, rootPlayer uuid.UUID) (int, int) {
	var mine, theirs int
	for _, p := range g.AllBattlefield() {
		if !p.HasType(core.TypeLand) || p.Tapped {
			continue
		}
		if p.ControllerID() == rootPlayer {
			mine++
		} else {
			theirs++
		}
	}
	return mine, theirs
}

func terminalEval(g *mage.Game, rootPlayer uuid.UUID) float64 {
	me := g.GetPlayer(rootPlayer)
	opp := g.GetOpponent(rootPlayer)
	if me == nil || opp == nil {
		return 0
	}
	if !opp.IsAlive() && me.IsAlive() {
		return terminalScore
	}
	if !me.IsAlive() && opp.IsAlive() {
		return -terminalScore
	}
	return 0
}
