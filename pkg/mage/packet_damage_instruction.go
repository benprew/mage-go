package mage

import "slices"

// PacketDamageInstruction retains one simultaneous damage group.
// Each packet keeps its own recipient and amount.
type PacketDamageInstruction struct {
	Source  DamageSourceRef
	Packets []DamagePacket
}

// Effect builds the native batch. Missing operands return nil.
func (instruction PacketDamageInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Source) || len(instruction.Packets) == 0 {
		return nil
	}
	for _, packet := range instruction.Packets {
		if nilDefinition(packet.recipient) || nilDefinition(packet.amount) {
			return nil
		}
	}
	return DealDamagePackets(instruction.Source, instruction.Packets...)
}

// InspectPacketDamageInstructions reads complete native trees without callbacks.
// Returned packet lists do not share mutable storage with the declarations.
func InspectPacketDamageInstructions(effect ResolvingEffect) ([]PacketDamageInstruction, bool) {
	var children []ResolvingEffect
	switch effect := effect.(type) {
	case *sequenceEffect:
		if effect == nil {
			return nil, false
		}
		children = effect.effects
	case *compositeEffect:
		if effect == nil {
			return nil, false
		}
		children = effect.effects
	case *damagePacketsEffect:
		if effect != nil {
			instruction := PacketDamageInstruction{Source: effect.source, Packets: slices.Clone(effect.packets)}
			if instruction.Effect() != nil {
				return []PacketDamageInstruction{instruction}, true
			}
		}
		return nil, false
	default:
		return nil, false
	}
	var out []PacketDamageInstruction
	for _, child := range children {
		instructions, ok := InspectPacketDamageInstructions(child)
		if !ok {
			return nil, false
		}
		out = append(out, instructions...)
	}
	return out, true
}

// DamageRecipientPredicate retains a native singular damage recipient.
// It stores no chosen state and does not execute reference callbacks.
type DamageRecipientPredicate struct {
	permanent           *PermanentTargetPredicate
	permanentController *PermanentTargetPredicate
	damage              *DamageTargetPredicate
	player              *PlayerTargetPredicate
	controller          bool
	active              bool
}

// InspectDamageRecipient accepts native damage targets and pure player references.
// Singular permanent targets use the shared visit and legality contract.
// Their controller references read current control after the same legality check.
// Saved and chooser references need separate contracts.
func InspectDamageRecipient(reference DamageRecipientRef) (DamageRecipientPredicate, bool) {
	switch reference := reference.(type) {
	case permanentInformationController:
		predicate, ok := InspectPermanentTarget(reference.subject)
		if ok {
			return DamageRecipientPredicate{permanentController: &predicate}, true
		}
	case *PermanentTargetBinding:
		predicate, ok := InspectPermanentTarget(reference)
		if ok {
			return DamageRecipientPredicate{permanent: &predicate}, true
		}
	case *DamageTargetBinding:
		predicate, ok := InspectDamageTarget(reference)
		if ok {
			return DamageRecipientPredicate{damage: &predicate}, true
		}
	case *PlayerTargetBinding:
		predicate, ok := InspectPlayerTarget(reference)
		if ok {
			return DamageRecipientPredicate{player: &predicate}, true
		}
	case controllerRef:
		return DamageRecipientPredicate{controller: true}, true
	case activePlayerRef:
		return DamageRecipientPredicate{active: true}, true
	}
	return DamageRecipientPredicate{}, false
}

// Select reads exact target legality or a present native player without callbacks.
// An empty known selection is illegal. Missing declarations return unknown.
func (predicate DamageRecipientPredicate) Select(g *Game, obj *StackObject) (DamageTargetSelection, bool) {
	if g == nil || obj == nil || g.GetPlayer(obj.Controller) == nil {
		return DamageTargetSelection{}, false
	}
	switch {
	case predicate.permanentController != nil:
		visits, known := predicate.permanentController.Select(g, obj)
		if !known || len(visits) == 0 {
			return DamageTargetSelection{}, known
		}
		permanent := g.FindPermanent(visits[0].ID)
		if permanent == nil || g.GetPlayer(permanent.ControllerID()) == nil {
			return DamageTargetSelection{}, false
		}
		return DamageTargetSelection{Player: permanent.ControllerID()}, true
	case predicate.permanent != nil:
		visits, known := predicate.permanent.Select(g, obj)
		if !known || len(visits) == 0 {
			return DamageTargetSelection{}, known
		}
		return DamageTargetSelection{Permanent: visits[0]}, true
	case predicate.damage != nil:
		return predicate.damage.Select(g, obj)
	case predicate.player != nil:
		id, known := predicate.player.Select(g, obj)
		return DamageTargetSelection{Player: id}, known
	case predicate.controller:
		return DamageTargetSelection{Player: obj.Controller}, true
	case predicate.active:
		if player := g.ActivePlayerObj(); player != nil {
			return DamageTargetSelection{Player: player.PlayerID()}, true
		}
	}
	return DamageTargetSelection{}, false
}
