package mage

import (
	"slices"

	"github.com/google/uuid"
)

// StackObjectRef selects one stack object during resolution.
type StackObjectRef interface {
	stackObjectRef()
	resolveStackObject(*EffectContext) (*StackObject, bool)
}

// SpellTargetBinding declares one spell target on the stack.
type SpellTargetBinding struct{ *targetDeclaration }

// DeclareSpellTarget declares a spell target with optional card filters.
func DeclareSpellTarget(name string, filters ...CardFilter) *SpellTargetBinding {
	spec := &typedSpellTarget{BaseTarget: BaseTarget{min: 1, max: 1}, filters: slices.Clone(filters)}
	return &SpellTargetBinding{&targetDeclaration{name: name, spec: spec}}
}

func (*SpellTargetBinding) stackObjectRef() {}
func (b *SpellTargetBinding) resolveStackObject(ctx *EffectContext) (*StackObject, bool) {
	value, ok := firstTargetValue(ctx.targetBindings[b.targetDeclaration])
	if !ok {
		return nil, false
	}
	obj := ctx.Game.stack.FindByID(value.id)
	return obj, obj != nil && !obj.IsAbility && obj.Card != nil
}
func (b *SpellTargetBinding) inspectDefinition() definitionNode {
	return targetReadNode(b.targetDeclaration, "stack-object-ref")
}

type typedSpellTarget struct {
	BaseTarget
	filters []CardFilter
}

func (t *typedSpellTarget) Possible(_ uuid.UUID, _ Card, g *Game) []uuid.UUID {
	var result []uuid.UUID
	for _, obj := range g.stack.Objects() {
		if obj.IsAbility || obj.Card == nil {
			continue
		}
		matches := true
		for _, filter := range t.filters {
			if !filter.Match(obj.Card) {
				matches = false
				break
			}
		}
		if matches {
			result = append(result, obj.ID)
		}
	}
	return result
}

func (t *typedSpellTarget) Choose(uuid.UUID, Card, *Game, []uuid.UUID) error { return nil }

type counterReferencedEffect struct{ spell StackObjectRef }

// Counter counters the referenced spell if it is still on the stack.
func Counter(spell StackObjectRef) ResolvingEffect { return &counterReferencedEffect{spell: spell} }
func (*counterReferencedEffect) Text() string      { return "counter target spell" }
func (*counterReferencedEffect) Properties() EffectProperties {
	return (&counterSpellEffect{}).Properties()
}
func (e *counterReferencedEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"spell", e.spell}}}
}
func (e *counterReferencedEffect) Apply(ctx *EffectContext) error {
	obj, ok := e.spell.resolveStackObject(ctx)
	if ok {
		ctx.Game.CounterStackObject(obj.ID)
	}
	return nil
}
