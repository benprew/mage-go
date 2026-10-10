package ron

import "github.com/benprew/mage-go/pkg/mage/core"

func intrinsicCreatureSpellTextRules(rules []Node) bool {
	intrinsic := false
	for _, rule := range rules {
		if rule.Kind == Symbol {
			if _, ok := keywords[rule.Name]; !ok {
				return false
			}
			continue
		}
		switch rule.Name {
		case "Protection":
			intrinsic = true
		case "Landwalk":
			kw, err := compileLandwalkKeyword(rule)
			if err != nil || kw == core.Desertwalk || kw == core.LegendaryLandwalk {
				return false
			}
			if len(rule.Args) != 1 || rule.Args[0].Name != "IsLandType" {
				return false
			}
			intrinsic = true
		default:
			return false
		}
	}
	return intrinsic
}
