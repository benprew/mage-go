package ron

func mixedStaticEventSpellTextRules(rules []Node) bool {
	if len(rules) != 2 {
		return false
	}
	for i, rule := range rules {
		if !globalStaticSpellTextRules([]Node{rule}) {
			continue
		}
		event := rules[1-i]
		if event.Kind != Tuple || event.Name != "TriggerA" || len(event.Args) != 2 {
			return false
		}
		trigger := event.Args[0]
		switch trigger.Name {
		case "WhenAPermanentIsTappedForMana":
			return trigger.Kind == Tuple && len(trigger.Args) == 1 && trigger.Args[0].Name == "IsLandType"
		case "AtTheBeginningOfAPlayersUpkeep":
			if trigger.Kind != Tuple || len(trigger.Args) != 1 {
				return false
			}
			player := trigger.Args[0]
			if player.Kind != Tuple || player.Name != "SinglePlayer" || len(player.Args) != 1 || player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
				return false
			}
			body := event.Args[1]
			if body.Kind != Tuple || body.Name != "ActionList" || len(body.Args) != 1 || body.Args[0].Kind != List || len(body.Args[0].Args) != 2 {
				return false
			}
			payment, continuation := body.Args[0].Args[0], body.Args[0].Args[1]
			if payment.Kind != Tuple || payment.Name != "MayCost" || len(payment.Args) != 1 || payment.Args[0].Name != "PayMana" {
				return false
			}
			if continuation.Kind != Tuple || continuation.Name != "Unless" || len(continuation.Args) != 2 || continuation.Args[0].Kind != Symbol || continuation.Args[0].Name != "CostWasPaid" || continuation.Args[1].Kind != List || len(continuation.Args[1].Args) != 1 {
				return false
			}
			sacrifice := continuation.Args[1].Args[0]
			return sacrifice.Kind == Tuple && sacrifice.Name == "SacrificePermanent" && len(sacrifice.Args) == 1 && sacrifice.Args[0].Kind == Symbol && sacrifice.Args[0].Name == "ThisPermanent"
		}
	}
	return false
}
