package mage

import (
	"slices"

	"github.com/google/uuid"
)

// CaptureSpellDamageSource reads a live spell by its stack identity.
// A card ID cannot select one of its separate casts.
func (g *Game) CaptureSpellDamageSource(stackID uuid.UUID) DamageSource {
	obj := g.stack.FindByID(stackID)
	if obj == nil || obj.IsAbility || obj.Card == nil || stackID == uuid.Nil {
		return DamageSource{}
	}
	return damageSourceFromSpellObject(obj)
}

func damageSourceFromSpellObject(obj *StackObject) DamageSource {
	if obj == nil || obj.IsAbility || obj.Card == nil || obj.ID == uuid.Nil {
		return DamageSource{}
	}
	card := obj.Card.Copy()
	card.SetID(obj.Card.ID())
	colors := cardColors(obj.Card)
	if obj.ColorOverride != nil {
		colors = *obj.ColorOverride
	}
	return DamageSource{id: obj.Card.ID(), stackID: obj.ID, controller: obj.Controller,
		owner: obj.Card.Owner(), name: obj.Card.Name(), card: card, colors: slices.Clone(colors)}
}

type spellDamageSource struct{ spell StackObjectRef }

// DamageSourceOfSpell reads a live spell through its typed reference.
// Use a snapshot before departure when later actions need its information.
func DamageSourceOfSpell(spell StackObjectRef) DamageSourceRef {
	return spellDamageSource{spell: spell}
}

func (spellDamageSource) damageSourceRef() {}
func (s spellDamageSource) resolveDamageSource(ctx *EffectContext) DamageSource {
	if nilDefinition(s.spell) {
		return DamageSource{}
	}
	obj, ok := s.spell.resolveStackObject(ctx)
	if !ok {
		return DamageSource{}
	}
	return ctx.Game.CaptureSpellDamageSource(obj.ID)
}
func (s spellDamageSource) inspectDefinition() definitionNode {
	n := definitionNode{Role: "damage-source-ref", Text: "spell", children: []definitionEdge{{"spell", s.spell}}}
	switch s.spell.(type) {
	case *SpellTargetBinding, savedSpellRef:
	default:
		n.problem = "supply a typed spell reference"
	}
	return n
}

type damageSourceSnapshot struct {
	source  DamageSourceRef
	binding *DamageSourceBinding
}

// SnapshotDamageSource saves information before later actions change its source.
func SnapshotDamageSource(source DamageSourceRef, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceSnapshot{source: source, binding: binding}
}

func (*damageSourceSnapshot) Text() string                 { return "save damage source" }
func (*damageSourceSnapshot) Properties() EffectProperties { return EffectProperties{} }
func (s *damageSourceSnapshot) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.damageSourceBindings == nil {
		ctx.Game.resolution.damageSourceBindings = make(map[*DamageSourceBinding]DamageSource)
	}
	ctx.Game.resolution.damageSourceBindings[s.binding] = cloneDamageSource(s.source.resolveDamageSource(ctx))
	return nil
}
func (s *damageSourceSnapshot) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: s.Text(), bindingWrite: s.binding,
		children: []definitionEdge{{"source", damageSourceDefinition{s.source}}, {"binding", s.binding}}}
}
