package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileConditionalAnimation(parts []Node, subject mage.PermanentRef, ctx scope) ([]func() mage.ContinuousModification, core.Layer, error) {
	var builds []func() mage.ContinuousModification
	var hasBase, hasCreature bool
	for _, part := range parts {
		switch part.Name {
		case "AddCardtype":
			if err := arity(part, 1); err != nil {
				return nil, 0, err
			}
			kind, err := cardType(part.Args[0])
			if err != nil {
				return nil, 0, err
			}
			if kind != core.TypeCreature && kind != core.TypeArtifact {
				return nil, 0, unsupported(part, "conditional animation type")
			}
			hasCreature = hasCreature || kind == core.TypeCreature
			builds = append(builds, func() mage.ContinuousModification { return mage.AddCardType(subject, kind) })
		case "SetPowerAndToughnessBoth":
			if hasBase {
				return nil, 0, problem(part, "duplicate conditional animation base values")
			}
			if err := arity(part, 1); err != nil {
				return nil, 0, err
			}
			value, err := compileStaticPTValue(part.Args[0], ctx)
			if err != nil {
				return nil, 0, err
			}
			hasBase = true
			builds = append(builds, func() mage.ContinuousModification {
				return mage.SetBasePowerToughness(subject, value(instanceBindings{}), value(instanceBindings{}))
			})
		default:
			return nil, 0, unsupported(part, "conditional animation modification")
		}
	}
	if !hasBase || !hasCreature {
		return nil, 0, problem(parts[0], "conditional animation requires creature type and base values")
	}
	return builds, core.LayerType, nil
}
