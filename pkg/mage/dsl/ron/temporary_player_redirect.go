package ron

func compileTemporaryPlayerRedirectPlan(n Node, ctx scope) (*temporaryDamageRedirectPlan, error) {
	if ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "temporary redirection requires an ordinary resolving body")
	}
	duration := n.Args[2]
	if duration.Kind != Symbol || duration.Name != "UntilEndOfTurn" {
		return nil, unsupported(duration, "temporary player redirection duration")
	}
	event := n.Args[0]
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "supply one player redirection operation")
	}
	operation := operations[0]
	if event.Name == "DamageWouldBeDealtBySpell" {
		spell, err := compileSpellRef(event.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		if operation.Name != "DealToPlayerInstead" {
			return nil, unsupported(operation, "spell damage redirection operation")
		}
		if err := arity(operation, 1); err != nil {
			return nil, err
		}
		if operation.Args[0].Kind != Symbol || operation.Args[0].Name != "WouldDealDamage_ControllerOfDamageSource" {
			return nil, unsupported(operation.Args[0], "damage source controller")
		}
		return &temporaryDamageRedirectPlan{spell: &spell}, nil
	}
	recipient := event.Args[0]
	if recipient.Name != "APermanent" {
		return nil, unsupported(recipient, "optional player redirection recipient")
	}
	if err := arity(recipient, 1); err != nil {
		return nil, err
	}
	filter, err := compileFilter(recipient.Args[0])
	if err != nil {
		return nil, err
	}
	if filter.IsZero() {
		return nil, problem(recipient, "supply a recipient filter")
	}
	if operation.Name != "MayActions" {
		return nil, unsupported(operation, "optional player redirection operation")
	}
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	optional, err := list(operation.Args[0])
	if err != nil {
		return nil, err
	}
	if len(optional) != 1 {
		return nil, problem(operation, "supply one optional redirect")
	}
	redirect := optional[0]
	if redirect.Name != "DealToPlayerInstead" {
		return nil, unsupported(redirect, "player redirection operation")
	}
	if err := arity(redirect, 1); err != nil {
		return nil, err
	}
	if redirect.Args[0].Kind != Symbol || redirect.Args[0].Name != "You" {
		return nil, unsupported(redirect.Args[0], "optional redirection player")
	}
	return &temporaryDamageRedirectPlan{recipients: &filter}, nil
}
