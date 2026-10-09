package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileHostPTRevision(n Node) optionFactory {
	if n.Name != "If" {
		return nil
	}
	condition := n.Args[0]
	if condition.Name != "PermanentPassesFilter" || len(condition.Args) != 2 || condition.Args[0].Kind != Symbol || condition.Args[0].Name != "HostPermanent" {
		return nil
	}
	basic, land, ok := hostPTConjunction(condition.Args[1], "IsSupertype", "IsLandType")
	if !ok || basic != "Basic" {
		return nil
	}
	switch land {
	case "Plains", "Island", "Swamp", "Mountain", "Forest":
	default:
		return nil
	}
	body := n.Args[1]
	if body.Kind != List || len(body.Args) != 1 {
		return nil
	}
	rule := body.Args[0]
	if rule.Name != "EachPermanentLayerEffect" || len(rule.Args) != 2 {
		return nil
	}
	kind, subtype, ok := hostPTConjunction(rule.Args[0], "IsCardtype", "IsCreatureType")
	if !ok || kind != "Creature" {
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
		return mage.WithAbility(mage.StaticAbility(mage.CreaturesPTWhileAttachedToBasicLand(land, subtype, power, toughness)).WithPTTextRevision())
	}
}

func hostPTConjunction(n Node, first, second string) (firstValue, secondValue string, matched bool) {
	if n.Name != "And" || len(n.Args) != 1 || n.Args[0].Kind != List || len(n.Args[0].Args) != 2 {
		return "", "", false
	}
	parts := n.Args[0].Args
	a, b := parts[0], parts[1]
	if a.Name == second {
		a, b = b, a
	}
	if a.Name != first || b.Name != second || len(a.Args) != 1 || len(b.Args) != 1 || a.Args[0].Kind != Symbol || b.Args[0].Kind != Symbol {
		return "", "", false
	}
	return a.Args[0].Name, b.Args[0].Name, true
}
