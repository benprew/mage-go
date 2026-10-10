package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func cloneTextNode(n Node) Node {
	result := n
	if n.Args != nil {
		result.Args = make([]Node, len(n.Args))
		for i, arg := range n.Args {
			result.Args[i] = cloneTextNode(arg)
		}
	}
	if n.Fields != nil {
		result.Fields = make(map[string]Node, len(n.Fields))
		for key, field := range n.Fields {
			result.Fields[key] = cloneTextNode(field)
		}
	}
	return result
}

func transformTextDefinition(n Node, changes []mage.TextChange) (Node, error) {
	result := cloneTextNode(n)
	for _, change := range changes {
		if rules, ok := result.Fields["Rules"]; ok {
			changed, err := transformTextWords(rules, "rule", change)
			if err != nil {
				return Node{}, err
			}
			result.Fields["Rules"] = changed
		}
		typeline := result.Fields["Typeline"]
		subtypes, err := transformTextWords(typeline.Fields["Subtypes"], "land_subtype", change)
		if err != nil {
			return Node{}, err
		}
		unique := make([]Node, 0, len(subtypes.Args))
		seen := make(map[string]bool)
		for _, subtype := range subtypes.Args {
			if !seen[subtype.Name] {
				unique = append(unique, subtype)
				seen[subtype.Name] = true
			}
		}
		subtypes.Args = unique
		typeline.Fields["Subtypes"] = subtypes
	}
	return result, nil
}

func transformTextWords(n Node, role string, change mage.TextChange) (Node, error) {
	if n.Kind == Symbol {
		color, colorOK := map[string]core.Color{"White": core.White, "Blue": core.Blue, "Black": core.Black, "Red": core.Red, "Green": core.Green}[n.Name]
		land, landOK := map[string]core.BasicLandType{"Plains": core.PlainsLandType, "Island": core.IslandLandType, "Swamp": core.SwampLandType, "Mountain": core.MountainLandType, "Forest": core.ForestLandType}[n.Name]
		if role == "unknown" && (colorOK && change.ApplyColor(color) != color || landOK && change.ApplyBasicLand(land) != land) {
			return Node{}, problem(n, "text word has no validated semantic role")
		}
		if colorOK && (role == "color" || role == "token_color") {
			n.Name = change.ApplyColor(color).String()
		}
		if landOK && (role == "land_subtype" || role == "subtype" || role == "token_subtype") {
			n.Name = change.ApplyBasicLand(land).String()
		}
	}
	for i, arg := range n.Args {
		childRole := textArgumentRole(n, role, i)
		if n.Kind == List {
			childRole = role
		}
		changed, err := transformTextWords(arg, childRole, change)
		if err != nil {
			return Node{}, err
		}
		n.Args[i] = changed
	}
	return n, nil
}

func textArgumentRole(n Node, role string, index int) string {
	if role == "permanent_collection" && n.Kind != List {
		switch n.Name {
		case "Ref_TargetPermanents":
			role = "permanent_collection_reference"
		case "And", "Or", "AttachedToAPermanent", "IsBlockingAttacker", "IsBlockedByDefender", "WasBlockedByDefenderThisTurn", "SinglePermanent":
		default:
			role = "permanent_filter"
		}
	}
	if role == "damage_permanent_collection" && n.Kind != List {
		if n.Name == "Ref_TargetPermanents" {
			role = "permanent_collection_reference"
		} else if n.Name != "And" {
			role = "permanent_filter"
		}
	}
	if role == "spell_filter" && n.Kind == Tuple {
		if (n.Name == "IsColor" || n.Name == "IsNonColor") && len(n.Args) == 1 {
			return "color"
		}
		if (n.Name == "And" || n.Name == "Or") && len(n.Args) == 1 {
			return role
		}
	}
	if (role == "rule" || role == "keyword") && n.Kind == Tuple {
		if n.Name == "Protection" && len(n.Args) == 1 {
			return "protection_filter"
		}
		if n.Name == "ProtectionAndDoesntRemovePermanents" && len(n.Args) == 2 {
			return []string{"protection_filter", "permanent_collection_reference"}[index]
		}
	}
	return inventoryArgumentRole(n, role, index)
}

type textSubstitutionFactory func(instanceBindings) mage.TextChangeRef

func compileTextSubstitution(n Node, ctx scope) (textSubstitutionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if n.Args[0].Name == n.Args[1].Name {
		return nil, problem(n, "text substitution requires different word references")
	}
	switch n.Name {
	case "ReplaceColorWordVariableWithNewColorWordVariable":
		from, err := compileColorWordRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		to, err := compileColorWordRef(n.Args[1], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.TextChangeRef {
			return mage.ColorWordSubstitution(from(bindings), to(bindings))
		}, nil
	case "ReplaceLandTypeVariableWithNewLandTypeVariable":
		from, err := compileLandWordRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		to, err := compileLandWordRef(n.Args[1], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.TextChangeRef {
			return mage.BasicLandWordSubstitution(from(bindings), to(bindings))
		}, nil
	default:
		return nil, unsupported(n, "text substitution")
	}
}

func definitionAbilityGroupOption(group int, build optionFactory) optionFactory {
	return func() mage.CardOption { return mage.WithDefinitionAbilityGroup(group, build()) }
}
