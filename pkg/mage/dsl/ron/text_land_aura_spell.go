package ron

func landAuraSpellTextRules(rules []Node) bool {
	enchant, layer, choice := 0, 0, 0
	chosen := false
	for _, rule := range rules {
		switch rule.Name {
		case "EnchantPermanent":
			enchant++
		case "PermanentLayerEffect":
			if len(rule.Args) != 2 || rule.Args[0].Kind != Symbol || rule.Args[0].Name != "HostPermanent" || rule.Args[1].Kind != List || len(rule.Args[1].Args) != 1 {
				return false
			}
			operation := rule.Args[1].Args[0]
			if operation.Name != "SetLandType" && operation.Name != "SetLandTypeVariable" {
				return false
			}
			chosen = operation.Name == "SetLandTypeVariable"
			layer++
		case "AsPermanentEnters":
			if len(rule.Args) != 2 || rule.Args[0].Kind != Symbol || rule.Args[0].Name != "ThisPermanent" || rule.Args[1].Kind != List || len(rule.Args[1].Args) != 1 || rule.Args[1].Args[0].Kind != Symbol || rule.Args[1].Args[0].Name != "ChooseABasicLandType" {
				return false
			}
			choice++
		default:
			return false
		}
	}
	return enchant == 1 && layer == 1 && (chosen && choice == 1 || !chosen && choice == 0)
}
