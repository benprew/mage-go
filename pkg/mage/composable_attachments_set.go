package mage

import . "github.com/benprew/mage-go/pkg/mage/core"

// AttachedAuras selects Auras attached to the current permanent incarnation.
func AttachedAuras(host PermanentRef) PermanentSet {
	return attachedAurasSet{host: host}
}

type attachedAurasSet struct{ host PermanentRef }

func (attachedAurasSet) permanentSet() {}

func (s attachedAurasSet) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "Auras attached to permanent"}
	if nilDefinition(s.host) {
		n.problem = "attached Auras require a permanent host"
	} else {
		n.children = []definitionEdge{{"host", s.host}}
	}
	return n
}

func (s attachedAurasSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.host) {
		return nil
	}
	host, ok := s.host.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	result := make([]*Permanent, 0, len(host.Attachments))
	for _, id := range host.Attachments {
		aura := ctx.Game.FindPermanent(id)
		if aura == nil || aura.PhasedOut || aura.AttachedTo != host.ID() || aura.attachedIncarnation != host.incarnationID {
			continue
		}
		if aura.HasSubType("Aura") && aura.HasType(TypeEnchantment) {
			result = append(result, aura)
		}
	}
	return result
}
