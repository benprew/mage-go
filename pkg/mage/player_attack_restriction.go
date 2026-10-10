package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// CantBeAttackedExceptByUntilNextTurn saves player protection at resolution.
// The filter reads current attackers until the saved player's next turn.
func CantBeAttackedExceptByUntilNextTurn(player PlayerRef, filter PermanentFilter, until PlayerRef) ResolvingEffect {
	return &playerAttackRestrictionEffect{player: player, filter: filter, until: until}
}

type playerAttackRestrictionEffect struct {
	player PlayerRef
	filter PermanentFilter
	until  PlayerRef
}

func (*playerAttackRestrictionEffect) Text() string {
	return "restrict attacks against a player until the next turn"
}
func (*playerAttackRestrictionEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *playerAttackRestrictionEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"duration player", e.until}}}
}
func (e *playerAttackRestrictionEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || playerRefReadsSourcePermanent(e.until)
}
func (e *playerAttackRestrictionEffect) Apply(ctx *EffectContext) error {
	player, ok := e.player.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(player) == nil {
		return nil
	}
	until, ok := e.until.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(until) == nil {
		return nil
	}
	rule := &playerAttackRestriction{
		funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.Indefinite, func(*Game, uuid.UUID) error { return nil }, func(*Game, uuid.UUID) bool { return true }).(*funcContinuousEffect),
		player:               player, filter: e.filter, until: until, after: ctx.Game.CurrentTurn(),
	}
	rule.SetSourceID(ctx.SourceID)
	ctx.Game.AddContinuousEffect(rule)
	return nil
}

type playerAttackRestriction struct {
	*funcContinuousEffect
	player uuid.UUID
	filter PermanentFilter
	until  uuid.UUID
	after  int
}

func (*playerAttackRestriction) survivesSourceExit() bool { return true }
func (e *playerAttackRestriction) NextTurnExpiry() (player uuid.UUID, afterTurn int) {
	return e.until, e.after
}
func (e *playerAttackRestriction) cloneEffect() ContinuousEffect {
	clone := *e
	inner := *e.funcContinuousEffect
	clone.funcContinuousEffect = &inner
	return &clone
}
func (g *Game) playerAttackRestricted(p *Permanent) bool {
	defender := g.GetOpponent(p.ControllerID())
	if defender == nil {
		return false
	}
	for _, effect := range g.effects.effects {
		if rule, ok := effect.(*playerAttackRestriction); ok && rule.player == defender.PlayerID() && rule.IsActive(g) && !rule.filter.Match(p, g) {
			return true
		}
	}
	return false
}
