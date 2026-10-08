package ron

import "github.com/benprew/mage-go/pkg/mage"

type damageRecipientFactory func(instanceBindings) mage.DamageRecipientRef

func compileDamagePacketRecipient(n Node, ctx scope) (damageRecipientFactory, error) {
	switch n.Name {
	case "Permanent":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		ref, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.DamageRecipientRef { return ref.build(bindings) }, nil
	case "Player":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		ref, err := compilePlayerRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.DamageRecipientRef { return ref.build(bindings) }, nil
	case "Ref_AnyTarget", "Ref_TargetPlayerOrPermanent":
		ref, err := compileDamageTargetRef(n, ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.DamageRecipientRef { return bindings.recipients[ref] }, nil
	default:
		return nil, unsupported(n, "damage packet recipient")
	}
}

func compileDamagePackets(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	source, err := compileSelfDamageSource(n.Args[0], ctx, n.Name == "PermanentDealsMultipleDamage")
	if err != nil {
		return nil, err
	}
	packets, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(packets) == 0 {
		return nil, problem(n.Args[1], "supply at least one damage packet")
	}
	type packetPlan struct {
		recipient damageRecipientFactory
		amount    valueFactory
	}
	plans := make([]packetPlan, len(packets))
	for i, packet := range packets {
		if packet.Name != "DamageToRecipients" {
			return nil, unsupported(packet, "damage packet")
		}
		if err := arity(packet, 2); err != nil {
			return nil, err
		}
		amount, err := compileInstanceValue(packet.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		recipient, err := compileDamagePacketRecipient(packet.Args[1], ctx)
		if err != nil {
			return nil, err
		}
		plans[i] = packetPlan{recipient: recipient, amount: amount}
	}
	return func(bindings instanceBindings) mage.Effect {
		packets := make([]mage.DamagePacket, len(plans))
		for i, plan := range plans {
			packets[i] = mage.DamagePacketTo(plan.recipient(bindings), plan.amount(bindings))
		}
		return mage.DealDamagePackets(source(bindings), packets...)
	}, nil
}
