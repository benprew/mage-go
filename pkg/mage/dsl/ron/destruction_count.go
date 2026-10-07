package ron

import (
	"maps"
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
)

type destructionCountPlan struct {
	name      string
	selection Node
}

func isDestructionCount(n Node) bool {
	return n.Name == "TheNumberOfPermanentsThatDiedThisWay" || n.Name == "TheNumberOfPermanentsPutIntoAPlayersGraveyardThisWay"
}

func hasDestructionCount(n Node) bool {
	if isDestructionCount(n) {
		return true
	}
	return slices.ContainsFunc(n.Args, hasDestructionCount)
}

func compileDestructionCount(producer, continuation Node, ctx scope) (effectFactory, error) {
	if err := arity(producer, 1); err != nil {
		return nil, err
	}
	set, err := compileContextualCollection(producer.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	selection := producer.Args[0]
	if selection.Kind == Symbol && selection.Name == "Ref_TargetPermanents" && ctx.permanentTargets != nil {
		selection = ctx.permanentTargets.filter
	}
	plan := &destructionCountPlan{selection: selection}
	var find func(Node) error
	find = func(n Node) error {
		if isDestructionCount(n) {
			if plan.name != "" && plan.name != n.Name {
				return problem(n, "mixed destruction count results are not supported")
			}
			plan.name = n.Name
		}
		for _, child := range n.Args {
			if err := find(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := find(continuation); err != nil {
		return nil, err
	}
	ctx.destructionCount = plan
	body, err := compileEffect(continuation, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.destructionCounts = maps.Clone(bindings.destructionCounts)
		if bindings.destructionCounts == nil {
			bindings.destructionCounts = make(map[*destructionCountPlan]*mage.NumberBinding)
		}
		result := mage.BindNumber("destruction count at " + producer.Position.String())
		bindings.destructionCounts[plan] = result
		option := mage.CreaturesDiedThisWayInto(result)
		if plan.name == "TheNumberOfPermanentsPutIntoAPlayersGraveyardThisWay" {
			option = mage.PermanentsPutIntoGraveyardThisWayInto(result)
		}
		var destroy = mage.DestroyEach(set(bindings), option)
		if producer.Name == "DestroyEachPermanentNoRegen" {
			destroy = mage.DestroyEachWithoutRegeneration(set(bindings), option)
		}
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, destroy)
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}

func compileDestructionCountValue(n Node, ctx scope) (valueFactory, error) {
	plan := ctx.destructionCount
	if plan == nil || plan.name != n.Name || ctx.delayedStep {
		return nil, problem(n, "unbound or incompatible destruction count")
	}
	if n.Name == "TheNumberOfPermanentsThatDiedThisWay" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		if !sameDeclaration(n.Args[0], Node{Kind: Tuple, Name: "IsCardtype", Args: []Node{{Kind: Symbol, Name: "Creature"}}}) {
			return nil, unsupported(n.Args[0], "creature death count filter")
		}
	} else {
		if err := arity(n, 2); err != nil {
			return nil, err
		}
		if n.Args[1].Kind != Symbol || n.Args[1].Name != "AnyPlayer" {
			return nil, unsupported(n.Args[1], "destruction graveyard player filter")
		}
		if _, err := compileFilter(n.Args[0]); err != nil {
			return nil, err
		}
		if !selectionGuarantees(plan.selection, n.Args[0]) {
			return nil, problem(n.Args[0], "graveyard count filter must be guaranteed by the destruction selection")
		}
	}
	return func(bindings instanceBindings) mage.ResolutionValue {
		return mage.FromNumber(bindings.destructionCounts[plan])
	}, nil
}

func selectionGuarantees(selection, filter Node) bool {
	if sameDeclaration(selection, filter) {
		return true
	}
	if selection.Name == "And" && len(selection.Args) == 1 && selection.Args[0].Kind == List {
		for _, child := range selection.Args[0].Args {
			if selectionGuarantees(child, filter) {
				return true
			}
		}
	}
	return false
}

func sameDeclaration(left, right Node) bool {
	if left.Kind != right.Kind || left.Name != right.Name || left.String != right.String || left.Integer != right.Integer || len(left.Fields) != 0 || len(right.Fields) != 0 || len(left.Args) != len(right.Args) {
		return false
	}
	for i := range left.Args {
		if !sameDeclaration(left.Args[i], right.Args[i]) {
			return false
		}
	}
	return true
}
