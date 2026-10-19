package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type cantAttackEachRule struct {
	subjects  PermanentSet
	condition StateCondition
}

// CantAttackEach checks current collection membership at attack declaration.
// Changes to the rule do not remove creatures already in combat.
func CantAttackEach(subjects PermanentSet) StaticRule {
	return cantAttackEachRule{subjects: subjects, condition: StateConditionFunc("always", []StateDependency{StateBattlefield}, func(StateReader, StateContext) bool { return true })}
}

// CantAttackEachIf checks a live condition before each attack declaration.
func CantAttackEachIf(subjects PermanentSet, condition StateCondition) StaticRule {
	return cantAttackEachRule{subjects: subjects, condition: condition}
}

// CantAttack prevents the current subject from attacking.
// The rule remains separate from the subject's abilities.
func CantAttack(subject PermanentRef) StaticRule {
	return CantAttackEach(grantRecipientSet{subject: subject})
}

// CantAttackIf prevents attacks while the current condition is true.
// A later condition change does not remove declared attackers.
func CantAttackIf(subject PermanentRef, condition StateCondition) StaticRule {
	return cantAttackEachRule{subjects: grantRecipientSet{subject: subject}, condition: condition}
}

func (cantAttackEachRule) staticRule() {}

func (r cantAttackEachRule) inspectDefinition() definitionNode {
	n := definitionNode{Role: "static-rule", Text: "selected permanents cannot attack", battlefieldSource: true}
	if nilDefinition(r.subjects) {
		n.problem = "attack restriction requires a permanent set"
	} else {
		n.children = []definitionEdge{{"subjects", r.subjects}, {"condition", r.condition}}
	}
	return n
}

func (r cantAttackEachRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&matchingAttackRestriction{
		funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect),
		subjects:             r.subjects,
		condition:            r.condition,
	}}
}

type matchingAttackRestriction struct {
	*funcContinuousEffect
	subjects  PermanentSet
	condition StateCondition
}

func (e *matchingAttackRestriction) inspectDefinition() definitionNode {
	return cantAttackEachRule{subjects: e.subjects, condition: e.condition}.inspectDefinition()
}

func (e *matchingAttackRestriction) cloneEffect() ContinuousEffect {
	clone := *e
	inner := *e.funcContinuousEffect
	clone.funcContinuousEffect = &inner
	return &clone
}

func (g *Game) restrictedFromAttacking(p *Permanent) bool {
	if g.hasTurnAttackRule(p, false) || g.playerAttackRestricted(p) {
		return true
	}
	for _, effect := range g.effects.effects {
		if !effect.IsActive(g) {
			continue
		}
		inner := effect
		if printed, ok := inner.(*printedAbilityContinuousEffect); ok {
			inner = printed.ContinuousEffect
		}
		rule, ok := inner.(*matchingAttackRestriction)
		if !ok || nilDefinition(rule.subjects) {
			continue
		}
		source := g.FindPermanent(rule.SourceID())
		if source == nil {
			continue
		}
		if nilDefinition(rule.condition) || !rule.condition.checkState(g, StateContext{SourceID: source.ID(), ControllerID: source.ControllerID(), SourceIncarnation: source.incarnationID}) {
			continue
		}
		ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: source.ControllerID(), sourceIncarnation: source.incarnationID}
		for _, subject := range rule.subjects.resolvePermanents(ctx) {
			if subject.ID() == p.ID() && subject.incarnationID == p.incarnationID {
				return true
			}
		}
	}
	return false
}
