package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileCounterPlacement(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 1 {
		return nil, problem(n, "counter placement requires one instruction")
	}
	instruction := parts[0]
	if instruction.Name == "ACounterOfTypeOnPlayer" || instruction.Name == "NumberCountersOfTypeOnPlayer" {
		return compilePlayerCounterPlacement(instruction, ctx)
	}
	var counterNode, subjectNode Node
	amount := valueFactory(func(instanceBindings) mage.ResolutionValue { return mage.Fixed(1) })
	switch instruction.Name {
	case "ACounterOfTypeOnPermanent":
		if err := arity(instruction, 2); err != nil {
			return nil, err
		}
		counterNode, subjectNode = instruction.Args[0], instruction.Args[1]
	case "NumberCountersOfTypeOnPermanent":
		if err := arity(instruction, 3); err != nil {
			return nil, err
		}
		amount, err = compileInstanceCounterAmount(instruction.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		counterNode, subjectNode = instruction.Args[1], instruction.Args[2]
	default:
		return nil, unsupported(instruction, "permanent counter instruction")
	}
	counter, err := compilePermanentCounterType(counterNode)
	if err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(subjectNode, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.AddCountersTo(subject.build(bindings), counter, amount(bindings))
	}, nil
}

func compilePermanentCounterType(n Node) (core.CounterType, error) {
	if n.Kind == Symbol {
		counter, ok := map[string]core.CounterType{
			"CarrionCounter": core.Carrion, "ChargeCounter": core.Charge,
			"CorpseCounter": core.Corpse, "CreditCounter": core.Credit,
			"CubeCounter": core.Cube, "DoomCounter": core.Doom,
			"DreamCounter": core.Dream, "GlyphCounter": core.Glyph,
			"HatchlingCounter": core.Hatchling, "HungerCounter": core.Hunger,
			"InterventionCounter": core.Intervention, "JavelinCounter": core.Javelin,
			"MatrixCounter": core.Matrix, "MireCounter": core.Mire,
			"NetCounter": core.Net, "PinCounter": core.Pin,
			"PupaCounter": core.Pupa, "ScreamCounter": core.Scream,
			"SleepCounter": core.Sleep, "SporeCounter": core.Spore,
			"StorageCounter": core.Storage, "TideCounter": core.Tide,
			"TimeCounter": core.Time, "VitalityCounter": core.Vitality,
			"WindCounter": core.Wind,
		}[n.Name]
		if ok {
			return counter, nil
		}
	}
	if n.Name == "PTCounter" {
		if err := arity(n, 2); err != nil {
			return 0, err
		}
		if n.Args[0].Kind != IntegerNode || n.Args[1].Kind != IntegerNode {
			return 0, problem(n, "counter coordinates require integer literals")
		}
		counter, ok := map[[2]int]core.CounterType{
			{1, 1}: core.P1P1, {-1, -1}: core.M1M1, {1, 0}: core.P1P0,
			{1, 2}: core.P1P2, {-2, -2}: core.M2M2, {0, -1}: core.M0M1,
			{0, -2}: core.M0M2, {0, 1}: core.P0P1, {2, 0}: core.P2P0,
			{0, 2}: core.P0P2, {2, 2}: core.P2P2,
		}[[2]int{n.Args[0].Integer, n.Args[1].Integer}]
		if ok {
			return counter, nil
		}
	}
	return 0, unsupported(n, "permanent counter type")
}

func compileCounterAmount(n Node, ctx scope) (mage.ResolutionValue, error) {
	if n.Name == "NumCreaturesOrPlaneswalkersThatDiedThisTurn" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		filter := n.Args[0]
		if filter.Name != "IsCardtype" {
			return nil, unsupported(filter, "creature death history filter")
		}
		if err := arity(filter, 1); err != nil {
			return nil, err
		}
		if filter.Args[0].Kind != Symbol || filter.Args[0].Name != "Creature" {
			return nil, unsupported(filter.Args[0], "creature death history type")
		}
		return mage.CreaturesDiedThisTurn(), nil
	}
	value, err := compileValue(n, ctx)
	if err != nil {
		return nil, err
	}
	return mage.ValueAtResolution(value), nil
}

func compileInstanceCounterAmount(n Node, ctx scope) (valueFactory, error) {
	if n.Name == "NumCreaturesOrPlaneswalkersThatDiedThisTurn" {
		value, err := compileCounterAmount(n, ctx)
		return func(instanceBindings) mage.ResolutionValue { return value }, err
	}
	return compileInstanceValue(n, ctx)
}
