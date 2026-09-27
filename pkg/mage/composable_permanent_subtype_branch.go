package mage

import "github.com/google/uuid"

// IfPermanentHasEnchantmentSubtype checks an enchantment subtype during resolution.
func IfPermanentHasEnchantmentSubtype(subject PermanentRef, subtype string, body ResolvingEffect) ResolvingEffect {
	return &permanentEnchantmentSubtypeBranch{subject: subject, subtype: subtype, body: body}
}

type permanentEnchantmentSubtypeBranch struct {
	subject PermanentRef
	subtype string
	body    ResolvingEffect
}

func (e *permanentEnchantmentSubtypeBranch) Text() string {
	return "if permanent has enchantment subtype " + e.subtype
}

func (e *permanentEnchantmentSubtypeBranch) Properties() EffectProperties {
	if e.body == nil {
		return EffectProperties{}
	}
	return e.body.Properties()
}

func (e *permanentEnchantmentSubtypeBranch) Apply(ctx *EffectContext) error {
	if p, ok := e.subject.resolvePermanent(ctx); ok {
		if p.HasSubType(e.subtype) {
			return e.body.Apply(ctx)
		}
		return nil
	}
	var captured capturedPermanent
	switch ref := e.subject.(type) {
	case savedPermanentRef:
		captured = ctx.Game.resolution.permanentBindings[ref.binding]
	case sourceRef:
		if ctx.sourceCaptured {
			captured = capturedPermanent{id: ctx.SourceID, incarnation: ctx.sourceIncarnation}
		}
	}
	if captured.incarnation != uuid.Nil {
		if lki := ctx.Game.zones.LKIByIncarnation(captured.incarnation); lki != nil && lki.ID == captured.id && lki.HasSubType(e.subtype) {
			return e.body.Apply(ctx)
		}
	}
	return nil
}

func (e *permanentEnchantmentSubtypeBranch) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolution-branch", Text: e.Text(), children: []definitionEdge{{"permanent", e.subject}, {"then", e.body}}}
	if !enchantmentSubtypes[e.subtype] {
		n.problem = "supply an enchantment subtype"
	}
	return n
}
