package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// PreventDamageByThisTurn prevents damage from one exact permanent until cleanup.
// The saved visit can supply damage after departure. Return creates a new visit.
func PreventDamageByThisTurn(source PermanentRef) ResolvingEffect {
	return &sourceDamagePreventionInstallation{source: source}
}

type sourceDamagePreventionInstallation struct{ source PermanentRef }

func (*sourceDamagePreventionInstallation) Text() string {
	return "prevent damage by saved permanent this turn"
}
func (*sourceDamagePreventionInstallation) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *sourceDamagePreventionInstallation) inspectDefinition() definitionNode {
	n := definitionNode{Role: "install-source-damage-prevention", Text: e.Text(), children: []definitionEdge{{"source", e.source}}}
	if _, ok := e.source.(permanentInformationReference); !ok {
		n.problem = "source requires an exact permanent identity"
	}
	return n
}
func (e *sourceDamagePreventionInstallation) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.source)
}
func (e *sourceDamagePreventionInstallation) Apply(ctx *EffectContext) error {
	ref, ok := e.source.(permanentInformationReference)
	if !ok {
		return nil
	}
	source, ok := ref.informationIdentity(ctx)
	if !ok {
		return nil
	}
	ctx.Game.effects.AddReplacement(&combatSourcePreventionReplacement{
		replacementBase: replacementBase{duration: core.EndOfTurn},
		preventedID:     source.id, incarnation: source.incarnation, allDamage: true,
	})
	return nil
}
