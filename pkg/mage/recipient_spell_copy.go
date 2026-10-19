package mage

import "github.com/google/uuid"

// PlayerOrControllerOfDamageTarget reads the target player or the exact visit's controller.
// A departed permanent supplies its last known controller.
func PlayerOrControllerOfDamageTarget(target *DamageTargetBinding) PlayerRef {
	return damageTargetController{target: target}
}

type damageTargetController struct{ target *DamageTargetBinding }

func (damageTargetController) playerRef()       {}
func (damageTargetController) damageRecipient() {}
func (r damageTargetController) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	if r.target == nil {
		return uuid.Nil, false
	}
	values := ctx.targetBindings[r.target.targetDeclaration]
	value, ok := firstTargetValue(values)
	if !ok {
		return uuid.Nil, false
	}
	if ctx.Game.GetPlayer(value.id) != nil {
		return value.id, true
	}
	return ControllerOfWithLKI(PermanentFromDamageTarget(r.target)).resolvePlayer(ctx)
}
func (r damageTargetController) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (r damageTargetController) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-ref", Text: "target player or permanent controller", children: []definitionEdge{{"target", r.target}}}
}

// CopySpellFor lets the saved player copy a present spell and choose new targets.
// The resolving spell can supply its own exact stack object.
func CopySpellFor(player PlayerRef, spell StackObjectRef) ResolvingEffect {
	return &copySpellForEffect{player: player, spell: spell}
}

// MayCopySpellFor offers a separate copy choice before new targets are chosen.
func MayCopySpellFor(player PlayerRef, spell StackObjectRef) ResolvingEffect {
	return &copySpellForEffect{player: player, spell: spell, optional: true}
}

type copySpellForEffect struct {
	optional bool
	player   PlayerRef
	spell    StackObjectRef
}

func (*copySpellForEffect) Text() string                 { return "copy spell with optional new targets" }
func (*copySpellForEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *copySpellForEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"spell", e.spell}}}
}
func (e *copySpellForEffect) Apply(ctx *EffectContext) error {
	if nilDefinition(e.player) || nilDefinition(e.spell) {
		return nil
	}
	player, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	obj, ok := e.spell.resolveStackObject(ctx)
	if !ok || obj.leftStack || obj.IsAbility || obj.Card == nil {
		return nil
	}
	if e.optional && !ctx.Game.GetPlayer(player).ChooseMayAbility(e.Text()) {
		return nil
	}
	ctx.Game.CopyStackObjectDirect(obj, player, true)
	return nil
}
