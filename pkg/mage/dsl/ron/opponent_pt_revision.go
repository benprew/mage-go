package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileOpponentPTRevision(n Node) optionFactory {
	if n.Name != "If" {
		return nil
	}
	condition := n.Args[0]
	if condition.Name != "APlayerPassesFilter" || len(condition.Args) != 2 || condition.Args[0].Kind != Symbol || condition.Args[0].Name != "Opponent" {
		return nil
	}
	selector := condition.Args[1]
	if selector.Name != "ControlsA" || len(selector.Args) != 1 {
		return nil
	}
	filter := selector.Args[0]
	landType := ""
	var chosenColor core.Color
	if filter.Name == "IsLandType" && len(filter.Args) == 1 && filter.Args[0].Kind == Symbol {
		landType = filter.Args[0].Name
		switch landType {
		case "Plains", "Island", "Swamp", "Mountain", "Forest":
		default:
			return nil
		}
	} else {
		if filter.Name != "And" || len(filter.Args) != 1 || filter.Args[0].Kind != List || len(filter.Args[0].Args) != 2 {
			return nil
		}
		parts := filter.Args[0].Args
		if parts[1].Kind == Symbol && parts[1].Name == "IsNonToken" {
			parts = []Node{parts[1], parts[0]}
		}
		if parts[0].Kind != Symbol || parts[0].Name != "IsNonToken" || parts[1].Name != "IsColor" || len(parts[1].Args) != 1 {
			return nil
		}
		var err error
		chosenColor, err = color(parts[1].Args[0])
		if err != nil || chosenColor < core.White || chosenColor > core.Green {
			return nil
		}
	}
	body := n.Args[1]
	if body.Kind != List || len(body.Args) != 1 {
		return nil
	}
	rule := body.Args[0]
	if len(rule.Args) != 2 {
		return nil
	}
	name := ""
	switch rule.Name {
	case "PermanentLayerEffect":
		if rule.Args[0].Kind != Symbol || rule.Args[0].Name != "ThisPermanent" {
			return nil
		}
	case "EachPermanentLayerEffect":
		if landType != "" {
			return nil
		}
		filter := rule.Args[0]
		if filter.Name != "And" || len(filter.Args) != 1 || filter.Args[0].Kind != List || len(filter.Args[0].Args) != 2 {
			return nil
		}
		parts := filter.Args[0].Args
		if parts[0].Name == "IsNamed" {
			parts = []Node{parts[1], parts[0]}
		}
		if parts[0].Name != "IsCardtype" || len(parts[0].Args) != 1 || parts[0].Args[0].Kind != Symbol || parts[0].Args[0].Name != "Creature" || parts[1].Name != "IsNamed" || len(parts[1].Args) != 1 {
			return nil
		}
		named := parts[1].Args[0]
		if named.Name != "NamedCard" || len(named.Args) != 1 || named.Args[0].Kind != String || named.Args[0].String == "" {
			return nil
		}
		name = named.Args[0].String
	default:
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
		var effect mage.ContinuousEffect
		switch {
		case landType != "":
			effect = mage.SourcePTWhileOpponentControlsBasicLand(landType, power, toughness)
		case name != "":
			effect = mage.NamedCreaturesPTWhileOpponentControlsNontokenColor(name, chosenColor, power, toughness)
		default:
			effect = mage.SourcePTWhileOpponentControlsNontokenColor(chosenColor, power, toughness)
		}
		return mage.WithAbility(mage.StaticAbility(effect).WithLayeredPTTextRevision())
	}
}
