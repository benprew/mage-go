package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// RemoveGraveyardCreatureEnchant removes the fixed graveyard enchant declaration.
// The saved attachment stays available to later resolving instructions.
func RemoveGraveyardCreatureEnchant(subject PermanentRef) ContinuousModification {
	return &layerModification{subject: subject, removeGraveyardEnchant: true, layer: core.LayerAbility}
}

// GrantPermanentEnchant gives one exact recipient a live enchant restriction.
// Hosts use the recipient as This and its current controller as You.
func GrantPermanentEnchant(subject PermanentRef, hosts PermanentSet) ContinuousModification {
	return &layerModification{subject: subject, enchantHosts: hosts, enchantGrant: true, layer: core.LayerAbility}
}

type grantedEnchantDefinition struct{ hosts PermanentSet }

func (d grantedEnchantDefinition) inspectDefinition() definitionNode {
	return definitionNode{Role: "granted-enchant", bindingScope: true, children: []definitionEdge{{"hosts", d.hosts}}}
}

func isGraveyardCreatureEnchant(target Target) bool {
	if binding, ok := target.(TargetBinding); ok {
		target = binding.targetSpec()
	}
	declaration, ok := UnwrapTarget(target).(*AnyGraveyardCardTarget)
	return ok && declaration.graveyardEnchant
}
