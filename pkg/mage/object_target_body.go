package mage

import "github.com/google/uuid"

// PermanentFromObjectTarget reads the permanent branch of one union target.
func PermanentFromObjectTarget(target *SpellOrPermanentTargetBinding) *PermanentTargetBinding {
	if target == nil {
		return nil
	}
	return &PermanentTargetBinding{target.targetDeclaration}
}

// SpellFromObjectTarget reads the spell branch of one union target.
func SpellFromObjectTarget(target *SpellOrPermanentTargetBinding) *SpellTargetBinding {
	if target == nil {
		return nil
	}
	return &SpellTargetBinding{target.targetDeclaration}
}

// ResolveObjectTarget runs the body for the saved target category.
func ResolveObjectTarget(target *SpellOrPermanentTargetBinding, spell, permanent ResolvingEffect) ResolvingEffect {
	return &objectTargetBody{target, spell, permanent}
}

type objectTargetBody struct {
	target           *SpellOrPermanentTargetBinding
	spell, permanent ResolvingEffect
}

func (*objectTargetBody) Text() string { return "resolve spell or permanent target" }
func (e *objectTargetBody) Properties() EffectProperties {
	var properties EffectProperties
	for _, body := range []ResolvingEffect{e.spell, e.permanent} {
		if !nilDefinition(body) {
			mergeEffectProperties(&properties, body.Properties())
		}
	}
	return properties
}
func (e *objectTargetBody) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-branch", children: []definitionEdge{{"target", e.target}, {"spell", e.spell}, {"permanent", e.permanent}}}
}
func (e *objectTargetBody) Apply(ctx *EffectContext) error {
	value, ok := firstTargetValue(ctx.targetBindings[e.target.targetDeclaration])
	if !ok {
		return nil
	}
	if value.incarnation != uuid.Nil {
		if _, ok := PermanentFromObjectTarget(e.target).resolvePermanent(ctx); ok {
			return e.permanent.Apply(ctx)
		}
	} else {
		obj := ctx.Game.stack.FindByID(value.id)
		if obj != nil && !obj.IsAbility && obj.Card != nil {
			return e.spell.Apply(ctx)
		}
	}
	return nil
}

func (e *objectTargetBody) refersToSourcePermanent() bool {
	return effectReadsSourcePermanent(e.spell) || effectReadsSourcePermanent(e.permanent)
}
