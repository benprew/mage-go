package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// DamageSourceIsAttacking reads combat status for a damage source filter.
// Absent sources use the status saved before departure or phasing.
func DamageSourceIsAttacking() PermanentFilter {
	return NewPermanentFilter("attacking damage source", func(p *Permanent, g *Game) bool {
		if current := g.FindPermanentIncludingPhased(p.ID()); current != nil && current.incarnationID == p.incarnationID {
			if !current.PhasedOut {
				return g.IsAttackingIncarnation(p.ID(), p.incarnationID)
			}
			if information := g.zones.phaseInformation[p.incarnationID]; information != nil {
				return information.Attacking
			}
			return false
		}
		information := g.zones.LKIByIncarnation(p.incarnationID)
		return information != nil && information.ID == p.ID() && information.Attacking
	})
}

// PreventDamageThisTurn prevents damage to the recipient until cleanup.
// A zero source filter permits damage from any source.
func PreventDamageThisTurn(recipient DamageRecipientRef, from PermanentFilter) ResolvingEffect {
	return &recipientPreventionInstallation{recipient: recipient, from: from}
}

type recipientPreventionInstallation struct {
	recipient DamageRecipientRef
	from      PermanentFilter
}

func (*recipientPreventionInstallation) Text() string {
	return "prevent damage to the recipient this turn"
}
func (*recipientPreventionInstallation) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *recipientPreventionInstallation) inspectDefinition() definitionNode {
	return definitionNode{Role: "install-recipient-prevention", Text: e.Text(), children: []definitionEdge{{"recipient", e.recipient}}}
}
func (e *recipientPreventionInstallation) Apply(ctx *EffectContext) error {
	id, ok := e.recipient.resolveRecipient(ctx)
	if !ok {
		return nil
	}
	rule := &damagePreventionRuleReplacement{
		replacementBase: replacementBase{duration: core.EndOfTurn},
		from:            e.from,
	}
	if ctx.Game.GetPlayer(id) != nil {
		rule.toPlayerID = id
		rule.playerOnly = true
	} else {
		p := ctx.Game.FindPermanent(id)
		if p == nil {
			return nil
		}
		incarnation := p.incarnationID
		rule.to = NewPermanentFilter("captured recipient", func(current *Permanent, _ *Game) bool {
			return current.ID() == id && current.incarnationID == incarnation
		})
	}
	ctx.Game.effects.AddReplacement(rule)
	return nil
}
