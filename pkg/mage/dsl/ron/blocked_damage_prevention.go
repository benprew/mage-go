package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileBlockedDamagePrevention(n Node, ctx scope) (optionFactory, error) {
	event := n.Args[0]
	filter := event.Args[0]
	if filter.Name != "And" {
		return nil, unsupported(filter, "blocked damage source filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	parts, err := list(filter.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 2 {
		return nil, problem(filter, "supply one creature type and one blocking relation")
	}
	kind, relation := parts[0], parts[1]
	if kind.Name == "IsBlockedByDefender" {
		kind, relation = relation, kind
	}
	if kind.Name != "IsCardtype" {
		return nil, unsupported(kind, "blocked damage source type")
	}
	if err := arity(kind, 1); err != nil {
		return nil, err
	}
	if kind.Args[0].Kind != Symbol || kind.Args[0].Name != "Creature" {
		return nil, unsupported(kind.Args[0], "blocked damage source type")
	}
	if relation.Name != "IsBlockedByDefender" {
		return nil, unsupported(relation, "blocked damage source relation")
	}
	if err := arity(relation, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(relation.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind != permanentSource {
		return nil, unsupported(relation.Args[0], "blocked damage source defender")
	}
	recipient := event.Args[1]
	if recipient.Name != "Permanent" {
		return nil, unsupported(recipient, "blocked damage recipient")
	}
	if err := arity(recipient, 1); err != nil {
		return nil, err
	}
	target, err := compilePermanentRef(recipient.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if target.kind != permanentSource {
		return nil, unsupported(recipient.Args[0], "blocked damage recipient")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Kind != Symbol || operations[0].Name != "PreventThatDamage" {
		return nil, problem(n.Args[1], "blocked damage prevention requires the symbol PreventThatDamage")
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.Static(mage.PreventDamageFromBlockedCreatures(ref.build(instanceBindings{}))))
	}, nil
}
