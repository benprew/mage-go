package mage

import (
	. "github.com/benprew/mage-go/pkg/mage/core"
)

type manaSpendingPermissions [AnyColor + 1][AnyColor + 1]bool

func (p manaSpendingPermissions) permits(from, to Color) bool { return p[from][to] }

func (p manaSpendingPermissions) withLegacy(conversions map[Color]Color) manaSpendingPermissions {
	if len(conversions) == 0 && p == (manaSpendingPermissions{}) {
		return p
	}
	for from, to := range conversions {
		p[from][to] = true
	}
	for _, via := range manaPoolColors {
		for _, from := range manaPoolColors {
			for _, to := range manaPoolColors {
				p[from][to] = p[from][to] || (p[from][via] && p[via][to])
			}
		}
	}
	return p
}

func (mp *ManaPool) effectiveSpendingPermissions() manaSpendingPermissions {
	return mp.spendingPermissions.withLegacy(mp.ManaConversions)
}

func (in ManaSolverInputs) effectiveSpendingPermissions() manaSpendingPermissions {
	var permissions manaSpendingPermissions
	if in.Pool != nil {
		permissions = in.Pool.spendingPermissions
	}
	return permissions.withLegacy(in.Conversions)
}

// ControllerManaSpending permits payment with one color as another color.
// The permission keeps the actual color and restrictions of each mana unit.
func ControllerManaSpending(from, to Color) ContinuousEffect {
	return &controllerManaSpendingEffect{from: from, to: to}
}

type controllerManaSpendingEffect struct {
	effectSource
	from, to Color
}

func (*controllerManaSpendingEffect) GetLayer() Layer       { return LayerAbility }
func (*controllerManaSpendingEffect) GetDuration() Duration { return WhileOnBattlefield }
func (e *controllerManaSpendingEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.SourceID()) != nil
}
func (*controllerManaSpendingEffect) Text() string                 { return "spend colored mana as another color" }
func (*controllerManaSpendingEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *controllerManaSpendingEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "mana-spending", Text: e.Text(), battlefieldSource: true}
	if !isSpendingColor(e.from) || !isSpendingColor(e.to) {
		n.problem = "supply two mana colors"
	}
	return n
}
func isSpendingColor(c Color) bool {
	return c == White || c == Blue || c == Black || c == Red || c == Green
}
func (*controllerManaSpendingEffect) ApplyContinuous(*EffectContext) error { return nil }

func (g *Game) syncManaSpendingPermissions() {
	for _, player := range g.players {
		player.ManaPool().spendingPermissions = manaSpendingPermissions{}
	}
	for _, effect := range g.effects.effects {
		inner := effect
		if printed, ok := inner.(*printedAbilityContinuousEffect); ok {
			inner = printed.ContinuousEffect
		}
		rule, ok := inner.(*controllerManaSpendingEffect)
		if !ok || !effect.IsActive(g) {
			continue
		}
		source := g.FindPermanent(rule.SourceID())
		if source == nil {
			continue
		}
		player := g.GetPlayer(source.ControllerID())
		if player != nil {
			player.ManaPool().spendingPermissions[rule.from][rule.to] = true
		}
	}
}

func (e *controllerManaSpendingEffect) cloneEffect() ContinuousEffect { clone := *e; return &clone }
