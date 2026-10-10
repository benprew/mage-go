package ron

func attachedObjectTextRules(rules []Node) bool {
	if len(rules) != 2 {
		return false
	}
	enchant, body := rules[0], rules[1]
	if body.Name == "EnchantPermanent" {
		enchant, body = body, enchant
	}
	if enchant.Name != "EnchantPermanent" || len(enchant.Args) != 1 || enchant.Args[0].Name != "IsCardtype" || len(enchant.Args[0].Args) != 1 || enchant.Args[0].Args[0].Kind != Symbol || enchant.Args[0].Args[0].Name != "Creature" {
		return false
	}
	switch body.Name {
	case "PermanentLayerEffect":
		if len(body.Args) != 2 || body.Args[0].Kind != Symbol || body.Args[0].Name != "HostPermanent" || body.Args[1].Kind != List || len(body.Args[1].Args) != 1 {
			return false
		}
		grant := body.Args[1].Args[0]
		if grant.Name != "AddAbility" || len(grant.Args) != 1 || grant.Args[0].Kind != List || len(grant.Args[0].Args) == 0 {
			return false
		}
		for _, ability := range grant.Args[0].Args {
			if _, ok := keywords[ability.Name]; ok && ability.Kind == Symbol {
				continue
			}
			if ability.Name != "Landwalk" && ability.Name != "ProtectionAndDoesntRemovePermanents" {
				return false
			}
		}
		return true
	case "Activated":
		if len(body.Args) != 2 || body.Args[0].Name != "PayMana" || len(body.Args[0].Args) != 1 || body.Args[0].Args[0].Kind != List || len(body.Args[0].Args[0].Args) == 0 {
			return false
		}
		for _, symbol := range body.Args[0].Args[0].Args {
			if symbol.Kind != Symbol || symbol.Name == "ManaCostX" {
				return false
			}
		}
		actions := body.Args[1]
		if actions.Name != "ActionList" || len(actions.Args) != 1 || actions.Args[0].Kind != List || len(actions.Args[0].Args) != 1 {
			return false
		}
		action := actions.Args[0].Args[0]
		if action.Name == "RegeneratePermanent" {
			return len(action.Args) == 1 && action.Args[0].Kind == Symbol && action.Args[0].Name == "HostPermanent"
		}
		if action.Name != "CreatePermanentLayerEffectUntil" || len(action.Args) != 3 || action.Args[0].Kind != Symbol || action.Args[0].Name != "HostPermanent" || action.Args[1].Kind != List || len(action.Args[1].Args) != 1 || action.Args[1].Args[0].Name != "AdjustPT" || action.Args[2].Kind != Symbol || action.Args[2].Name != "UntilEndOfTurn" {
			return false
		}
		return true
	}
	return false
}
