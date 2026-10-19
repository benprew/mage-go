package ron

import "github.com/benprew/mage-go/pkg/mage"

type damageRecipientPlayerPlan struct{ name string }

func compilePlayerRecipientDamage(n Node, ctx scope, source damageSourceFactory) (effectFactory, error) {
	recipient := n.Args[2]
	if err := arity(recipient, 1); err != nil {
		return nil, err
	}
	if recipient.Name == "Player" {
		player, err := compilePlayerRef(recipient.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		ctx.damageRecipient = &player
		amount, err := compileInstanceValue(n.Args[1], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.Effect {
			return (mage.SingularDamageInstruction{Source: source(bindings), Recipient: player.build(bindings), Amount: amount(bindings)}).Effect()
		}, nil
	}
	players := recipient.Args[0]
	if players.Kind != Symbol || (players.Name != "AnyPlayer" && players.Name != "Opponent") {
		return nil, unsupported(players, "damage player collection")
	}
	plan := &damageRecipientPlayerPlan{name: "damage recipient " + n.Position.String()}
	player := playerReferencePlan{damageRecipient: plan}
	ctx.damageRecipient = &player
	amount, err := compileInstanceValue(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		member := mage.BindPlayer(plan.name)
		bindings.damageRecipients = map[*damageRecipientPlayerPlan]*mage.PlayerBinding{plan: member}
		var subjects mage.PlayerSet = mage.EachPlayer
		if players.Name == "Opponent" {
			subjects = mage.OpponentsOf(mage.You)
		}
		return (mage.PlayerCollectionDamageInstruction{Source: source(bindings), Subjects: subjects, Member: member, Amount: amount(bindings)}).Effect()
	}, nil
}
