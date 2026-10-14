package mage

import "github.com/google/uuid"

// NextTurnExpiration supplies a saved player and the turn of installation.
// A zero player means that the effect has no next-turn endpoint.
// Installed effects must keep these values separate in each cloned game.
type NextTurnExpiration interface {
	NextTurnExpiry() (player uuid.UUID, afterTurn int)
}

func (e *composedLayerEffect) NextTurnExpiry() (player uuid.UUID, afterTurn int) {
	return e.nextTurnPlayer, e.nextTurnAfter
}

func (em *EffectManager) removeUntilNextTurn(player uuid.UUID, turn int) bool {
	expires := func(effect NextTurnExpiration) bool {
		saved, after := effect.NextTurnExpiry()
		return saved != uuid.Nil && saved == player && turn > after
	}
	changed := false
	effects := em.effects[:0]
	for _, effect := range em.effects {
		if lifetime, ok := effect.(NextTurnExpiration); ok && expires(lifetime) {
			changed = true
			continue
		}
		effects = append(effects, effect)
	}
	em.effects = effects
	replacements := em.replacements[:0]
	for _, effect := range em.replacements {
		if lifetime, ok := effect.(NextTurnExpiration); ok && expires(lifetime) {
			changed = true
			continue
		}
		replacements = append(replacements, effect)
	}
	em.replacements = replacements
	return changed
}

// BeginActiveTurn expires saved lifetimes before the active turn's actions.
// Drivers must check turn skips first. Repeated calls in one turn are safe.
func (g *Game) BeginActiveTurn() {
	g.recordActiveTurnStart()
	started := g.effects.beginNextTurnWindows(g.ActivePlayerObj().PlayerID(), g.CurrentTurn())
	if g.effects.removeUntilNextTurn(g.ActivePlayerObj().PlayerID(), g.CurrentTurn()) || started {
		g.ApplyContinuousEffects()
	}
	g.captureTurnStartUntapped()
}

// Each installed rule keeps its own window so clones can enter different turns.
type nextTurnWindow struct {
	player     uuid.UUID
	afterTurn  int
	activeTurn int
}

func (w *nextTurnWindow) active(turn int) bool {
	return w.player == uuid.Nil || w.activeTurn == turn && turn > w.afterTurn
}

type nextTurnWindowEffect interface {
	nextTurnLifetime() *nextTurnWindow
}

func (em *EffectManager) beginNextTurnWindows(player uuid.UUID, turn int) bool {
	changed := false
	for _, effect := range em.effects {
		if rule, ok := effect.(nextTurnWindowEffect); ok {
			window := rule.nextTurnLifetime()
			if window.player == player && turn > window.afterTurn && window.activeTurn == 0 {
				window.activeTurn = turn
				changed = true
			}
		}
	}
	return changed
}

func (em *EffectManager) removeNextTurnWindows(turn int) {
	effects := em.effects[:0]
	for _, effect := range em.effects {
		if rule, ok := effect.(nextTurnWindowEffect); ok && rule.nextTurnLifetime().player != uuid.Nil && rule.nextTurnLifetime().activeTurn == turn {
			continue
		}
		effects = append(effects, effect)
	}
	em.effects = effects
}
