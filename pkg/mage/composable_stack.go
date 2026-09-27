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

// SpellBinding declares a spell seen by an event.
type SpellBinding struct{ name string }

// BindSpell declares a spell binding for a cast event.
func BindSpell(name string) *SpellBinding { return &SpellBinding{name: name} }

type savedSpellRef struct{ binding *SpellBinding }

// FromSpell reads the same stack object seen by the cast event.
func FromSpell(binding *SpellBinding) StackObjectRef { return savedSpellRef{binding: binding} }

func (savedSpellRef) stackObjectRef() {}
func (r savedSpellRef) resolveStackObject(ctx *EffectContext) (*StackObject, bool) {
	id := ctx.Game.resolution.spellBindings[r.binding]
	obj := ctx.Game.stack.FindByID(id)
	return obj, obj != nil && !obj.IsAbility && obj.Card != nil
}
func (b *SpellBinding) bindingName() string { return b.name }
func (b *SpellBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "spell-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}
func (r savedSpellRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "stack-object-ref", Text: "cast spell", bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
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
	if obj == nil {
		obj = ctx.Game.stack.FindBySourceID(value.id)
	}
	return obj, obj != nil && !obj.IsAbility && obj.Card != nil
}
func (b *SpellTargetBinding) inspectDefinition() definitionNode {
	return targetReadNode(b.targetDeclaration, "stack-object-ref")
}

type spellControllerRef struct{ spell StackObjectRef }

type spellManaValue struct{ spell StackObjectRef }

// ManaValueOfSpell reads the referenced spell's mana value on the stack.
func ManaValueOfSpell(spell StackObjectRef) ResolutionValue { return spellManaValue{spell: spell} }
func (v spellManaValue) Text() string                       { return "mana value of spell" }
func (v spellManaValue) resolveValue(ctx *EffectContext) int {
	spell, ok := v.spell.resolveStackObject(ctx)
	if !ok {
		return 0
	}
	cost := spell.Card.ManaCost()
	return cost.CMC() + cost.XCount*spell.XValue
}
func (v spellManaValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"spell", v.spell}}}
}

// ControllerOfSpell reads the current controller of the referenced spell.
func ControllerOfSpell(spell StackObjectRef) PlayerRef { return spellControllerRef{spell: spell} }
func (spellControllerRef) playerRef()                  {}
func (spellControllerRef) damageRecipient()            {}
func (r spellControllerRef) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	spell, ok := r.spell.resolveStackObject(ctx)
	if !ok {
		return uuid.Nil, false
	}
	return spell.Controller, ctx.Game.GetPlayer(spell.Controller) != nil
}
func (r spellControllerRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (r spellControllerRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-ref", Text: "spell controller", children: []definitionEdge{{"spell", r.spell}}}
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
