package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileOnlyPlayerActivation(n Node, ctx scope) ([]mage.ActionOption, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	player := n.Args[0]
	if player.Name != "OwnerOfPermanent" && player.Name != "ControllerOfPermanent" {
		return nil, unsupported(player, "activation permission player reference")
	}
	if err := arity(player, 1); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(player.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if subject.kind != permanentSource && subject.kind != permanentHost {
		return nil, unsupported(player.Args[0], "activation permission permanent reference")
	}
	ref := subject.build(instanceBindings{})
	condition := mage.PlayerIsPermanentController(ref)
	if player.Name == "OwnerOfPermanent" {
		condition = mage.PlayerIsPermanentOwner(ref)
	}
	return []mage.ActionOption{mage.WithAnyPlayerMay(), mage.WithActivationStateCondition(condition)}, nil
}
