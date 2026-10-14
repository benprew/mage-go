package ron

import "github.com/benprew/mage-go/pkg/mage"

func numericChoiceBounds(n Node) (minimum, maximum int, err error) {
	if err := arity(n, 2); err != nil {
		return 0, 0, err
	}
	if n.Args[0].Kind != IntegerNode || n.Args[1].Kind != IntegerNode || n.Args[0].Integer < 0 || n.Args[1].Integer < n.Args[0].Integer {
		return 0, 0, problem(n, "number choice requires nonnegative integer bounds in ascending order")
	}
	return n.Args[0].Integer, n.Args[1].Integer, nil
}
func compileNumericEntryDeclaration(rules []Node, permanent bool) (bool, error) {
	found := false
	entries := 0
	var declaration Node
	for _, rule := range rules {
		if rule.Name != "AsPermanentEnters" {
			continue
		}
		entries++
		if rule.Kind != Tuple || len(rule.Args) != 2 || rule.Args[1].Kind != List {
			continue
		}
		for _, operation := range rule.Args[1].Args {
			if operation.Name != "ChooseANumberBetween" {
				continue
			}
			if !permanent || rule.Args[0].Kind != Symbol || rule.Args[0].Name != "ThisPermanent" || len(rule.Args[1].Args) != 1 {
				return false, problem(rule, "number entry choice requires one source permanent operation")
			}
			if _, _, err := numericChoiceBounds(operation); err != nil {
				return false, err
			}
			if found {
				return false, problem(rule, "duplicate number entry choice")
			}
			found = true
			declaration = rule
		}
	}
	if found && entries != 1 {
		return false, problem(declaration, "number choice requires one entry declaration")
	}
	return found, nil
}
func compileNumericEntry(n Node) (optionFactory, error) {
	minimum, maximum, err := numericChoiceBounds(n)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption { return mage.WithAbility(mage.ChooseNumberAsEnters(minimum, maximum)) }, nil
}
func compileOptionalNumericChoice(n Node, ctx scope) (effectFactory, error) {
	if !ctx.permanent || !ctx.numberEntryChoice || !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody || ctx.sourceZone != nil {
		return nil, problem(n, "number update requires a resolving battlefield source with an entry choice")
	}
	minimum, maximum, err := numericChoiceBounds(n)
	if err != nil {
		return nil, err
	}
	return func(instanceBindings) mage.Effect { return mage.MayChooseNumber(mage.This, minimum, maximum) }, nil
}
func compileSeparateStaticPT(n Node, parts []Node, ctx scope) (optionFactory, error) {
	if n.Name != "PermanentLayerEffect" || !ctx.permanent || !ctx.numberEntryChoice || n.Args[0].Kind != Symbol || n.Args[0].Name != "ThisPermanent" {
		return nil, problem(n, "separate static power and toughness require a source entry number choice")
	}
	subject, err := compileConditionalSubject(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	var builds []func() mage.ContinuousModification
	power, toughness := false, false
	for _, part := range parts {
		if part.Name != "SetPower" && part.Name != "SetToughness" {
			return nil, unsupported(part, "separate static power or toughness")
		}
		if err := arity(part, 1); err != nil {
			return nil, err
		}
		value, err := compileStaticPTValue(part.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		if part.Name == "SetPower" {
			if power {
				return nil, problem(part, "duplicate static power")
			}
			power = true
			builds = append(builds, func() mage.ContinuousModification { return mage.SetBasePower(subject, value(instanceBindings{})) })
		} else {
			if toughness {
				return nil, problem(part, "duplicate static toughness")
			}
			toughness = true
			builds = append(builds, func() mage.ContinuousModification { return mage.SetBaseToughness(subject, value(instanceBindings{})) })
		}
	}
	return func() mage.CardOption {
		modifications := make([]mage.ContinuousModification, len(builds))
		for i, build := range builds {
			modifications[i] = build()
		}
		condition := mage.StateConditionFunc("live static power and toughness", []mage.StateDependency{mage.StateBattlefield}, func(mage.StateReader, mage.StateContext) bool { return true })
		return mage.WithAbility(mage.Static(mage.While(condition, modifications...)))
	}, nil
}
