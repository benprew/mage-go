package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// LimitPlayerDamageFromTypes replaces matching damage at the threshold.
// Source types use the current or retained information for the exact object.
// Simultaneous packets from one source to the player share one amount.
func LimitPlayerDamageFromTypes(subject PlayerRef, threshold, amount int, kinds ...core.CardType) StaticRule {
	return staticPlayerDamageCap{subject: subject, threshold: threshold, amount: amount, kinds: slices.Clone(kinds)}
}

type staticPlayerDamageCap struct {
	subject   PlayerRef
	threshold int
	amount    int
	kinds     []core.CardType
}

func (staticPlayerDamageCap) staticRule() {}
func (r staticPlayerDamageCap) inspectDefinition() definitionNode {
	node := definitionNode{Role: "static-rule", Text: "limit player damage from source types", battlefieldSource: true, children: []definitionEdge{{"subject", r.subject}}}
	if r.threshold <= 0 || r.amount < 0 || r.amount >= r.threshold {
		node.problem = "supply a positive threshold and a smaller nonnegative damage amount"
	}
	if len(r.kinds) == 0 {
		node.problem = "supply at least one source card type"
	}
	seen := make(map[core.CardType]bool)
	for _, kind := range r.kinds {
		if kind < core.TypeCreature || kind > core.TypePlaneswalker || seen[kind] {
			node.problem = "supply distinct valid source card types"
		}
		seen[kind] = true
	}
	return node
}
func (r staticPlayerDamageCap) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&staticPlayerDamageCapEffect{
		funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect),
		rule:                 r,
	}}
}

type staticPlayerDamageCapEffect struct {
	*funcContinuousEffect
	rule staticPlayerDamageCap
}

func (e *staticPlayerDamageCapEffect) inspectDefinition() definitionNode {
	return e.rule.inspectDefinition()
}
func (e *staticPlayerDamageCapEffect) cloneEffect() ContinuousEffect {
	clone := *e
	inner := *e.funcContinuousEffect
	clone.funcContinuousEffect = &inner
	return &clone
}
func (e *staticPlayerDamageCapEffect) ApplyContinuous(ctx *EffectContext) error {
	source := ctx.Game.FindPermanent(e.SourceID())
	if source != nil {
		ctx.Game.effects.AddCycleReplacement(&staticPlayerDamageCapReplacement{
			replacementBase: replacementBase{sourceID: source.ID()},
			incarnation:     source.incarnationID,
			rule:            e.rule,
		})
	}
	return nil
}

type staticPlayerDamageCapReplacement struct {
	replacementBase
	incarnation uuid.UUID
	rule        staticPlayerDamageCap
}

func (r *staticPlayerDamageCapReplacement) Matches(action Action, reader GameReader) bool {
	return damageActionAmount(action) >= r.rule.threshold && r.matchesRecipientAndSource(action, reader.(*Game))
}

func (r *staticPlayerDamageCapReplacement) matchesRecipientAndSource(action Action, g *Game) bool {
	damage, ok := action.(*DamageToPlayerAction)
	if !ok || damage.Amount() <= 0 {
		return false
	}
	source := g.FindPermanent(r.sourceID)
	if source == nil || source.PhasedOut || source.incarnationID != r.incarnation {
		return false
	}
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: source.ControllerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	player, ok := r.rule.subject.resolvePlayer(ctx)
	if !ok || player != damage.PlayerID() {
		return false
	}
	return slices.ContainsFunc(r.rule.kinds, func(kind core.CardType) bool {
		return damageSourceHasCardType(action, g, kind)
	})
}

func (r *staticPlayerDamageCapReplacement) batchIndices(actions []Action, applied []map[ReplacementEffect]bool, index int, g *Game) []int {
	selected, ok := actions[index].(*DamageToPlayerAction)
	if !ok || !r.matchesRecipientAndSource(selected, g) {
		return nil
	}
	source := selected.DamageSource()
	var indices []int
	for i, action := range actions {
		damage, ok := action.(*DamageToPlayerAction)
		if !ok || applied[i][r] || !r.matchesRecipientAndSource(action, g) || damage.IsCombatDamage() != selected.IsCombatDamage() {
			continue
		}
		other := damage.DamageSource()
		if i == index || source.SameObject(other) {
			indices = append(indices, i)
		}
	}
	return indices
}

func (r *staticPlayerDamageCapReplacement) matchesDamageBatch(actions []Action, applied []map[ReplacementEffect]bool, index int, g *Game) bool {
	indices := r.batchIndices(actions, applied, index, g)
	if len(indices) == 0 || indices[0] != index {
		return false
	}
	total := 0
	for _, i := range indices {
		total += min(damageActionAmount(actions[i]), r.rule.threshold-total)
	}
	return total >= r.rule.threshold
}

func (r *staticPlayerDamageCapReplacement) replaceDamageBatch(actions []Action, applied []map[ReplacementEffect]bool, index int, g *Game) {
	indices := r.batchIndices(actions, applied, index, g)
	remaining := r.rule.amount
	for _, i := range indices {
		applied[i][r] = true
		amount := min(damageActionAmount(actions[i]), remaining)
		remaining -= amount
		actions[i] = keepDamageCause(actions[i], damageActionWithAmount(actions[i], amount))
	}
}
func (r *staticPlayerDamageCapReplacement) Replace(action Action, _ *Game) Action {
	return damageActionWithAmount(action, r.rule.amount)
}
func (*staticPlayerDamageCapReplacement) IsActive(GameReader) bool { return true }
func (r *staticPlayerDamageCapReplacement) Clone() ReplacementEffect {
	clone := *r
	return &clone
}
