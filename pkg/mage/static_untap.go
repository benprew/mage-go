package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type beginUntapStepAction struct{ playerID uuid.UUID }

func (a *beginUntapStepAction) ActionSource() uuid.UUID { return uuid.Nil }
func (a *beginUntapStepAction) PlayerID() uuid.UUID     { return a.playerID }

type untapStepSkipRule struct{}

// SkipUntapSteps replaces each player's untap step before any step action.
// Use this rule with Static so source departure stops the replacement.
func SkipUntapSteps() StaticRule      { return untapStepSkipRule{} }
func (untapStepSkipRule) staticRule() {}
func (untapStepSkipRule) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-rule", Text: "players skip their untap steps", battlefieldSource: true}
}
func (r untapStepSkipRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&untapStepSkipEffect{
		funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect),
		rule:                 r, ruleID: uuid.New(),
	}}
}

type untapStepSkipEffect struct {
	*funcContinuousEffect
	rule   untapStepSkipRule
	ruleID uuid.UUID
}

func (e *untapStepSkipEffect) inspectDefinition() definitionNode { return e.rule.inspectDefinition() }
func (e *untapStepSkipEffect) cloneEffect() ContinuousEffect {
	clone := *e
	inner := *e.funcContinuousEffect
	clone.funcContinuousEffect = &inner
	return &clone
}
func (e *untapStepSkipEffect) ApplyContinuous(ctx *EffectContext) error {
	source := ctx.Game.FindPermanent(e.SourceID())
	if source != nil {
		ctx.Game.effects.AddCycleReplacement(&untapStepSkipReplacement{
			replacementBase: replacementBase{sourceID: source.ID(), duration: core.WhileOnBattlefield},
			incarnation:     source.incarnationID, ruleID: e.ruleID,
		})
	}
	return nil
}

type untapStepSkipReplacement struct {
	replacementBase
	incarnation uuid.UUID
	ruleID      uuid.UUID
}

func (r *untapStepSkipReplacement) Matches(a Action, _ GameReader) bool {
	_, ok := a.(*beginUntapStepAction)
	return ok
}
func (*untapStepSkipReplacement) Replace(_ Action, _ *Game) Action { return nil }
func (r *untapStepSkipReplacement) IsActive(g GameReader) bool {
	source := g.FindPermanent(r.sourceID)
	if source == nil || source.PhasedOut || source.incarnationID != r.incarnation {
		return false
	}
	for _, ability := range source.RuntimeAbilities {
		if holder, ok := UnwrapAbility(ability).(*StaticAbilityHolder); ok {
			for _, effect := range holder.Effects {
				if skip, ok := effect.(*untapStepSkipEffect); ok && skip.ruleID == r.ruleID {
					return true
				}
			}
		}
	}
	return false
}
func (r *untapStepSkipReplacement) Clone() ReplacementEffect { clone := *r; return &clone }
