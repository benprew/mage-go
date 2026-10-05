package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"
)

// DamagePacket keeps one recipient and its amount in the same instruction.
type DamagePacket struct {
	recipient DamageRecipientRef
	amount    ResolutionValue
}

// DamagePacketTo permits a separate amount for each recipient.
func DamagePacketTo(recipient DamageRecipientRef, amount ResolutionValue) DamagePacket {
	return DamagePacket{recipient: recipient, amount: amount}
}

func (p DamagePacket) inspectDefinition() definitionNode {
	return definitionNode{Role: "damage-packet", children: []definitionEdge{{"recipient", p.recipient}, {"amount", p.amount}}}
}

type damagePacketsEffect struct {
	source  DamageSourceRef
	packets []DamagePacket
}

// DealDamagePackets applies all packets before it sends damage events.
func DealDamagePackets(source DamageSourceRef, packets ...DamagePacket) ResolvingEffect {
	return &damagePacketsEffect{source: source, packets: slices.Clone(packets)}
}

func (*damagePacketsEffect) Text() string { return "deal simultaneous damage" }
func (*damagePacketsEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *damagePacketsEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"source", damageSourceDefinition{e.source}}}}
	if len(e.packets) == 0 {
		n.problem = "supply at least one damage packet"
	}
	for i, packet := range e.packets {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("packets[%d]", i), packet})
	}
	return n
}
func (e *damagePacketsEffect) Apply(ctx *EffectContext) error {
	source := e.source.resolveDamageSource(ctx)
	if !source.Valid() {
		return nil
	}
	assignments := make([]DamageAssignment, 0, len(e.packets))
	indices := make(map[uuid.UUID]int, len(e.packets))
	for _, packet := range e.packets {
		id, ok := packet.recipient.resolveRecipient(ctx)
		if !ok {
			continue
		}
		amount := packet.amount.resolveValue(ctx)
		if amount <= 0 {
			continue
		}
		if index, found := indices[id]; found {
			assignments[index].request.amount += amount
			continue
		}
		var assignment DamageAssignment
		if player := ctx.Game.GetPlayer(id); player != nil {
			assignment = PlayerDamageAssignment(player, amount, source)
		} else if permanent := ctx.Game.FindPermanent(id); permanent != nil && !permanent.PhasedOut {
			assignment = PermanentDamageAssignment(permanent, amount, source)
		} else {
			continue
		}
		indices[id] = len(assignments)
		assignments = append(assignments, assignment)
	}
	ctx.Game.DealSimultaneousDamage(assignments...)
	return nil
}
