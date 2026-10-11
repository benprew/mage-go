package ron

import "github.com/benprew/mage-go/pkg/mage"

func isRepeatedCoinDamage(n Node) bool {
	return len(n.Args) == 1 && n.Args[0].Kind == List && len(n.Args[0].Args) > 0 && n.Args[0].Args[0].Name == "EachPlayerAction"
}

func compileRepeatedCoinDamage(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if !ctx.spellTargets || ctx.permanent || ctx.activation || ctx.delayedStep || ctx.drawReplacementBody || ctx.targetNamespace != "" {
		return nil, unsupported(n, "coin rounds outside an ordinary spell")
	}
	if len(parts) != 3 {
		return nil, problem(n, "coin rounds require a producer, tails damage and an adjacent repeat condition")
	}
	producer, damage, continuation := parts[0], parts[1], parts[2]
	if producer.Name != "EachPlayerAction" {
		return nil, unsupported(producer, "coin round producer")
	}
	if err := arity(producer, 2); err != nil {
		return nil, err
	}
	if _, err := symbol(producer.Args[1]); err != nil {
		return nil, err
	}
	if producer.Args[1].Name != "FlipACoin" {
		return nil, unsupported(producer.Args[1], "heads or tails action")
	}
	collection := producer.Args[0]
	if collection.Name != "Or" {
		return nil, unsupported(collection, "coin round player union")
	}
	if err := arity(collection, 1); err != nil {
		return nil, err
	}
	operands, err := list(collection.Args[0])
	if err != nil {
		return nil, err
	}
	if len(operands) == 0 {
		return nil, problem(collection, "coin round requires player references")
	}
	players := make([]playerReferencePlan, len(operands))
	for i, operand := range operands {
		if operand.Name != "SinglePlayer" {
			return nil, unsupported(operand, "coin round player operand")
		}
		if err := arity(operand, 1); err != nil {
			return nil, err
		}
		players[i], err = compilePlayerRef(operand.Args[0], ctx)
		if err != nil {
			return nil, err
		}
	}
	if damage.Name != "SpellDealsDamage" {
		return nil, unsupported(damage, "coin round damage")
	}
	if err := arity(damage, 3); err != nil {
		return nil, err
	}
	source, err := compileSelfDamageSource(damage.Args[0], ctx, false)
	if err != nil {
		return nil, err
	}
	if damage.Args[1].Name != "Integer" {
		return nil, unsupported(damage.Args[1], "fixed coin round damage amount")
	}
	amount, err := compileInstanceValue(damage.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	recipients := damage.Args[2]
	if recipients.Name != "EachPlayer" {
		return nil, unsupported(recipients, "coin round damage recipients")
	}
	if err := arity(recipients, 1); err != nil {
		return nil, err
	}
	if name, err := symbol(recipients.Args[0]); err != nil || name != "CoinCameUpTails" {
		return nil, problem(recipients.Args[0], "coin round damage requires bare CoinCameUpTails")
	}
	if continuation.Name != "Unless" {
		return nil, unsupported(continuation, "coin round continuation")
	}
	if err := arity(continuation, 2); err != nil {
		return nil, err
	}
	if name, err := symbol(continuation.Args[0]); err != nil || name != "AllCoinsCameUpHeads" {
		return nil, problem(continuation.Args[0], "coin round continuation requires bare AllCoinsCameUpHeads")
	}
	again, err := list(continuation.Args[1])
	if err != nil {
		return nil, err
	}
	if len(again) != 1 || again[0].Kind != Symbol || again[0].Name != "RepeatThisProcess" {
		return nil, problem(continuation, "coin continuation requires one bare RepeatThisProcess")
	}
	return func(bindings instanceBindings) mage.Effect {
		refs := make([]mage.PlayerRef, len(players))
		for i, player := range players {
			refs[i] = player.build(bindings)
		}
		return mage.RepeatHeadsOrTailsDamage(source(bindings), amount(bindings), refs...)
	}, nil
}
