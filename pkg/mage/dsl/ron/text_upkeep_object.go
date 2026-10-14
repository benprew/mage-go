package ron

func upkeepObjectTextRules(rules []Node) bool {
	trigger, choice, chosen := false, false, false
	for _, rule := range rules {
		if rule.Kind == Symbol {
			if _, fixed := keywords[rule.Name]; fixed {
				continue
			}
			return false
		}
		if rule.Name == "AsPermanentEnters" {
			if choice || validateOpponentEntryChoice(rule) != nil || len(rule.Args[1].Args) != 1 {
				return false
			}
			choice = true
			continue
		}
		if trigger || rule.Kind != Tuple || rule.Name != "TriggerA" || len(rule.Args) != 2 {
			return false
		}
		event := rule.Args[0]
		if event.Kind != Tuple || event.Name != "AtTheBeginningOfAPlayersUpkeep" || len(event.Args) != 1 {
			return false
		}
		player := event.Args[0]
		if player.Kind == Symbol && player.Name == "AnyPlayer" {
			trigger = true
			continue
		}
		if player.Kind != Tuple || player.Name != "SinglePlayer" || len(player.Args) != 1 || player.Args[0].Kind != Symbol {
			return false
		}
		switch player.Args[0].Name {
		case "You":
		case "TheChosenPlayer":
			chosen = true
		default:
			return false
		}
		trigger = true
	}
	return trigger && chosen == choice
}
