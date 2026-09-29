package mage

import "github.com/google/uuid"

type announcedTargetCountValue struct{ targets *PermanentTargetsBinding }

// AnnouncedTargetCount reads the chosen size of one permanent target group.
func AnnouncedTargetCount(targets *PermanentTargetsBinding) ResolutionValue {
	return announcedTargetCountValue{targets: targets}
}

func (v announcedTargetCountValue) Text() string { return "number of announced targets" }

func (v announcedTargetCountValue) resolveValue(ctx *EffectContext) int {
	if v.targets == nil || ctx == nil || ctx.Game == nil {
		return 0
	}
	count := 0
	for i, spec := range ctx.Game.resolution.targetSpecs {
		if i < len(ctx.Game.resolution.resolvingTargets) && spec == v.targets.targetDeclaration && ctx.Game.resolution.resolvingTargets[i] != uuid.Nil {
			count++
		}
	}
	return count
}

func (v announcedTargetCountValue) inspectDefinition() definitionNode {
	if v.targets == nil {
		return definitionNode{Role: "resolution-value", problem: "supply a permanent target group"}
	}
	return definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"targets", v.targets}}}
}
