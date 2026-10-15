package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// LimitDamageLifeLoss limits damage's life loss to the specified life total.
// Damage amounts, damage history and lifelink keep their full values.
func LimitDamageLifeLoss(player PlayerRef, floor int) StaticRule {
	return damageLifeFloorRule{player: player, floor: floor}
}

type damageLifeFloorRule struct {
	player PlayerRef
	floor  int
}

func (damageLifeFloorRule) staticRule() {}
func (r damageLifeFloorRule) inspectDefinition() definitionNode {
	n := definitionNode{Role: "static-rule", Text: "limit damage life loss", battlefieldSource: true, children: []definitionEdge{{"player", r.player}}}
	if r.floor <= 0 {
		n.problem = "supply a positive life total"
	}
	return n
}
func (r damageLifeFloorRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&damageLifeFloorEffect{funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect), rule: r}}
}

type damageLifeFloorEffect struct {
	*funcContinuousEffect
	rule damageLifeFloorRule
}

func (e *damageLifeFloorEffect) inspectDefinition() definitionNode { return e.rule.inspectDefinition() }
func (e *damageLifeFloorEffect) cloneEffect() ContinuousEffect {
	c := *e
	inner := *e.funcContinuousEffect
	c.funcContinuousEffect = &inner
	return &c
}
func (e *damageLifeFloorEffect) ApplyContinuous(ctx *EffectContext) error {
	source := ctx.Game.FindPermanent(e.SourceID())
	if source != nil {
		ctx.Game.effects.AddCycleReplacement(&damageLifeFloorReplacement{replacementBase: replacementBase{sourceID: source.ID()}, incarnation: source.incarnationID, rule: e.rule})
	}
	return nil
}

type damageLifeLossAction struct {
	playerID uuid.UUID
	amount   int
	life     int
}

func (*damageLifeLossAction) ActionSource() uuid.UUID { return uuid.Nil }
func (a *damageLifeLossAction) PlayerID() uuid.UUID   { return a.playerID }

type damageLifeFloorReplacement struct {
	replacementBase
	incarnation uuid.UUID
	rule        damageLifeFloorRule
}

func (r *damageLifeFloorReplacement) Matches(action Action, reader GameReader) bool {
	loss, ok := action.(*damageLifeLossAction)
	if !ok || loss.amount <= 0 || loss.life > r.rule.floor && loss.amount <= loss.life-r.rule.floor {
		return false
	}
	g := reader.(*Game)
	source := g.FindPermanent(r.sourceID)
	if source == nil || source.PhasedOut || source.incarnationID != r.incarnation {
		return false
	}
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: source.ControllerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	player, ok := r.rule.player.resolvePlayer(ctx)
	return ok && player == loss.playerID
}
func (r *damageLifeFloorReplacement) Replace(action Action, _ *Game) Action {
	c := *action.(*damageLifeLossAction)
	if c.life <= r.rule.floor {
		c.amount = 0
	} else {
		c.amount = min(c.amount, c.life-r.rule.floor)
	}
	return &c
}
func (*damageLifeFloorReplacement) IsActive(GameReader) bool   { return true }
func (r *damageLifeFloorReplacement) Clone() ReplacementEffect { c := *r; return &c }

type damageLifeResult struct {
	player Player
	amount int
}

func (g *Game) damageLifeGain(player Player, amount int) {
	if g.damageBatch == nil || !g.damageBatch.stageLifeResults {
		g.gainLifeWithEvent(player, amount)
		return
	}
	g.damageBatch.gains = append(g.damageBatch.gains, damageLifeResult{player, amount})
}
func (g *Game) damageLifeLoss(player Player, amount int) {
	if g.damageBatch != nil && g.damageBatch.stageLifeResults {
		g.damageBatch.losses = append(g.damageBatch.losses, damageLifeResult{player, amount})
		return
	}
	batch := &damageBatch{losses: []damageLifeResult{{player, amount}}}
	g.applyDamageLifeResults(batch)
}

func (g *Game) applyDamageLifeResults(batch *damageBatch) {
	players := []Player{g.ActivePlayerObj(), g.NonActivePlayerObj()}
	gains := make(map[uuid.UUID]int)
	for _, player := range players {
		if player == nil {
			continue
		}
		for i, result := range batch.gains {
			if result.player.PlayerID() != player.PlayerID() {
				continue
			}
			actual := g.prepareLifeGain(result.player, result.amount)
			batch.gains[i].amount = actual
			gains[player.PlayerID()] += actual
		}
	}
	losses := make(map[uuid.UUID]int)
	for _, result := range batch.losses {
		losses[result.player.PlayerID()] += result.amount
	}
	for _, player := range players {
		if player == nil {
			continue
		}
		id := player.PlayerID()
		if _, ok := losses[id]; !ok {
			continue
		}
		action := &damageLifeLossAction{playerID: id, amount: losses[id], life: player.Life() + gains[id]}
		losses[id] = 0
		if !g.CanLoseLife(id) {
			continue
		}
		if result, ok := g.effects.ApplyReplacements(action, g).(*damageLifeLossAction); ok && g.CanLoseLife(id) {
			losses[id] = result.amount
		}
	}
	for _, result := range batch.gains {
		if result.amount > 0 {
			result.player.GainLife(result.amount)
		}
	}
	var events []core.GameEvent
	for _, result := range batch.losses {
		id := result.player.PlayerID()
		actual := min(result.amount, losses[id])
		losses[id] -= actual
		if actual > 0 {
			result.player.LoseLife(actual)
			events = append(events, core.GameEvent{Type: core.EvtLifeLost, PlayerID: id, Amount: actual})
		}
	}
	for _, event := range events {
		g.FireEvent(event)
	}
	for _, result := range batch.gains {
		if result.amount > 0 {
			g.FireEvent(core.GameEvent{Type: core.EvtLifeGained, PlayerID: result.player.PlayerID(), Amount: result.amount})
		}
	}
}

func (g *Game) hasDamageLifeFloor() bool {
	for _, list := range [][]ReplacementEffect{g.effects.replacements, g.effects.cycleReplacements} {
		for _, replacement := range list {
			if _, ok := replacement.(*damageLifeFloorReplacement); ok && replacement.IsActive(g) {
				return true
			}
		}
	}
	return false
}
