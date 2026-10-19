package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileResolutionSacrificeValue(payment, continuation Node, ctx scope) (effectFactory, error) {
	if ctx.enteringSource || ctx.drawReplacementBody {
		return nil, unsupported(payment, "resolving sacrifice value context")
	}
	if err := arity(payment, 1); err != nil {
		return nil, err
	}
	cost := payment.Args[0]
	if cost.Name != "SacrificeAPermanent" {
		return nil, unsupported(cost, "resolving sacrifice value cost")
	}
	if err := arity(cost, 1); err != nil {
		return nil, err
	}
	filter, err := compileSacrificeCostFilter(cost.Args[0])
	if err != nil {
		return nil, err
	}
	if continuation.Name != "If" {
		return nil, unsupported(continuation, "paid sacrifice value continuation")
	}
	if err := arity(continuation, 2); err != nil {
		return nil, err
	}
	if continuation.Args[0].Kind != Symbol || continuation.Args[0].Name != "CostWasPaid" {
		return nil, unsupported(continuation.Args[0], "paid sacrifice value condition")
	}
	parts, err := list(continuation.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(continuation, "supply a paid continuation")
	}
	plan := &sacrificeValuePlan{name: "ManaValueOfTheSacrificedPermanent"}
	ctx.sacrificeValue = plan
	body, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.sacrifices = maps.Clone(bindings.sacrifices)
		if bindings.sacrifices == nil {
			bindings.sacrifices = make(map[*sacrificeValuePlan]sacrificeValueBindings)
		}
		result := mage.BindNumber("resolving sacrifice mana value " + payment.Position.String())
		bindings.sacrifices[plan] = sacrificeValueBindings{value: result}
		effects := make([]mage.Effect, len(body))
		for i, build := range body {
			effects[i] = build(bindings)
		}
		return mage.IfSacrificeWithManaValue(ctx.player.build(bindings), filter, result, mage.Sequence(effects...))
	}, nil
}

func searchShape(n Node, kind Kind, name string, count int) error {
	if n.Kind != kind || n.Name != name || len(n.Args) != count {
		return unsupported(n, "mana-difference search instruction")
	}
	return nil
}

func normalSearchEntry(n Node) error {
	if err := searchShape(n, Tuple, "PutFoundCardsOntoBattlefield", 1); err != nil {
		return err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return err
	}
	if len(parts) != 1 {
		return unsupported(n, "normal search entry")
	}
	return searchShape(parts[0], Symbol, "EntersNormally", 0)
}

func compileLibraryManaDifference(n Node, steps []Node, ctx scope) (effectFactory, error) {
	if ctx.enteringSource || ctx.drawReplacementBody || ctx.sacrificeValue == nil || ctx.sacrificeValue.name != "ManaValueOfTheSacrificedPermanent" {
		return nil, problem(n, "mana-difference search requires a paid sacrifice value")
	}
	find := steps[0]
	if err := searchShape(find, Tuple, "FindACardOfType", 1); err != nil {
		return nil, err
	}
	filter := find.Args[0]
	if err := searchShape(filter, Tuple, "IsCardtype", 1); err != nil {
		return nil, err
	}
	if err := searchShape(filter.Args[0], Symbol, "Artifact", 0); err != nil {
		return nil, err
	}
	branch := steps[1]
	if err := searchShape(branch, Tuple, "IfElse", 3); err != nil {
		return nil, err
	}
	condition := branch.Args[0]
	for _, name := range []string{"ACardOfTypeWasFoundThisWay", "ManaValueIs", "LessThanOrEqualTo"} {
		if err := searchShape(condition, Tuple, name, 1); err != nil {
			return nil, err
		}
		condition = condition.Args[0]
	}
	if err := searchShape(condition, Symbol, "ManaValueOfTheSacrificedPermanent", 0); err != nil {
		return nil, err
	}
	lower, err := list(branch.Args[1])
	if err != nil {
		return nil, err
	}
	if len(lower) != 1 {
		return nil, unsupported(branch.Args[1], "search cheaper-card branch")
	}
	if err := normalSearchEntry(lower[0]); err != nil {
		return nil, err
	}
	greater, err := list(branch.Args[2])
	if err != nil {
		return nil, err
	}
	if len(greater) != 2 {
		return nil, unsupported(branch.Args[2], "search greater-card branch")
	}
	payment := greater[0]
	if err := searchShape(payment, Tuple, "MayCost", 1); err != nil {
		return nil, err
	}
	payment = payment.Args[0]
	if err := searchShape(payment, Tuple, "PayManaX", 2); err != nil {
		return nil, err
	}
	mana, err := list(payment.Args[0])
	if err != nil {
		return nil, err
	}
	if len(mana) != 1 {
		return nil, unsupported(payment.Args[0], "search difference mana")
	}
	if err := searchShape(mana[0], Symbol, "ManaCostX", 0); err != nil {
		return nil, err
	}
	difference := payment.Args[1]
	if err := searchShape(difference, Tuple, "Minus", 2); err != nil {
		return nil, err
	}
	for i, name := range []string{"ManaValueOfTheFoundCard", "ManaValueOfTheSacrificedPermanent"} {
		if err := searchShape(difference.Args[i], Symbol, name, 0); err != nil {
			return nil, err
		}
	}
	paid := greater[1]
	if err := searchShape(paid, Tuple, "IfElse", 3); err != nil {
		return nil, err
	}
	if err := searchShape(paid.Args[0], Symbol, "CostWasPaid", 0); err != nil {
		return nil, err
	}
	yes, err := list(paid.Args[1])
	if err != nil {
		return nil, err
	}
	if len(yes) != 1 {
		return nil, unsupported(paid.Args[1], "paid search branch")
	}
	if err := normalSearchEntry(yes[0]); err != nil {
		return nil, err
	}
	no, err := list(paid.Args[2])
	if err != nil {
		return nil, err
	}
	if len(no) != 1 {
		return nil, unsupported(paid.Args[2], "unpaid search branch")
	}
	if err := searchShape(no[0], Symbol, "PutFoundCardsIntoGraveyard", 0); err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.SearchLibraryWithManaDifference(ctx.player.build(bindings), mage.NewCardFilter("artifact", func(c mage.Card) bool { return c.HasType(core.TypeArtifact) }), mage.FromNumber(bindings.sacrifices[ctx.sacrificeValue].value))
	}, nil
}
