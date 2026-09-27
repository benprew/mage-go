package mage

import (
	"fmt"
)

// CastCondition reads facts saved when a spell was cast.
type CastCondition interface {
	castCondition()
	checkCast(*CastContext) bool
	inspectDefinition() definitionNode
}

type controlledCreatureSubtypeAtCast struct{ subtype string }

// ControlledCreatureSubtypeAtCast checks the caster's creature control at cast time.
func ControlledCreatureSubtypeAtCast(subtype string) CastCondition {
	return controlledCreatureSubtypeAtCast{subtype: subtype}
}

func (controlledCreatureSubtypeAtCast) castCondition() {}
func (c controlledCreatureSubtypeAtCast) checkCast(ctx *CastContext) bool {
	return ctx != nil && ctx.ControllerCreatureSubtypesAtCast[c.subtype]
}
func (c controlledCreatureSubtypeAtCast) inspectDefinition() definitionNode {
	n := definitionNode{Role: "cast-condition", Text: "controller controlled a " + c.subtype + " creature as this spell was cast"}
	if c.subtype == "" {
		n.problem = "cast condition requires a creature subtype"
	} else if family, known := knownSubtypeFamily(c.subtype); known && family != SubtypeCreature {
		n.problem = "cast condition requires a creature subtype"
	}
	return n
}

type revealedSubtypeAtCast struct{ subtype string }

// RevealedSubtypeAtCast checks cards revealed while the spell was cast.
func RevealedSubtypeAtCast(subtype string) CastCondition {
	return revealedSubtypeAtCast{subtype: subtype}
}

func (revealedSubtypeAtCast) castCondition() {}
func (c revealedSubtypeAtCast) checkCast(ctx *CastContext) bool {
	return ctx.RevealedSubtypeAtCast(c.subtype)
}
func (c revealedSubtypeAtCast) inspectDefinition() definitionNode {
	n := definitionNode{Role: "cast-condition", Text: "revealed a " + c.subtype + " card as this spell was cast"}
	if c.subtype == "" {
		n.problem = "cast condition requires a subtype"
	}
	return n
}

type castBranch struct {
	condition CastCondition
	effect    ResolvingEffect
}

// IfCast runs its effect if the spell's saved cast condition is true.
func IfCast(condition CastCondition, effect ResolvingEffect) ResolvingEffect {
	return &castBranch{condition: condition, effect: effect}
}

func (e *castBranch) Text() string                 { return fmt.Sprintf("if cast condition, %s", e.effect.Text()) }
func (e *castBranch) Properties() EffectProperties { return e.effect.Properties() }
func (e *castBranch) Apply(ctx *EffectContext) error {
	if e.condition.checkCast(ctx.Game.ResolvingCastContext()) {
		return e.effect.Apply(ctx)
	}
	return nil
}
func (e *castBranch) inspectDefinition() definitionNode {
	return definitionNode{Role: "cast-branch", children: []definitionEdge{{"condition", e.condition}, {"then", e.effect}}}
}
