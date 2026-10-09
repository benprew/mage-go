package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileConditionalSourcePTRevision(n Node) optionFactory {
	if n.Name != "If" {
		return nil
	}
	condition := n.Args[0]
	if condition.Name != "PlayerPassesFilter" || len(condition.Args) != 2 || condition.Args[0].Kind != Symbol || condition.Args[0].Name != "You" {
		return nil
	}
	selector := condition.Args[1]
	if selector.Name != "ControlsA" || len(selector.Args) != 1 {
		return nil
	}
	filter := selector.Args[0]
	if filter.Name != "IsLandType" || len(filter.Args) != 1 || filter.Args[0].Kind != Symbol {
		return nil
	}
	landType := filter.Args[0].Name
	switch landType {
	case "Plains", "Island", "Swamp", "Mountain", "Forest":
	default:
		return nil
	}
	body := n.Args[1]
	if body.Kind != List || len(body.Args) != 1 {
		return nil
	}
	rule := body.Args[0]
	if rule.Name != "PermanentLayerEffect" || len(rule.Args) != 2 || rule.Args[0].Kind != Symbol || rule.Args[0].Name != "ThisPermanent" {
		return nil
	}
	parts := rule.Args[1]
	if parts.Kind != List || len(parts.Args) != 1 || parts.Args[0].Name != "AdjustPT" {
		return nil
	}
	power, toughness, err := compilePT(parts.Args[0])
	if err != nil {
		return nil
	}
	return func() mage.CardOption {
		effect := mage.SourcePTWhileControllerControlsBasicLand(landType, power, toughness)
		return mage.WithAbility(mage.StaticAbility(effect).WithPTTextRevision())
	}
}
