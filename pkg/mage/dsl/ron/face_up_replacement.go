package ron

import "github.com/benprew/mage-go/pkg/mage"

type faceUpReplacementPlan struct{ castResult *castResultPlan }

func (plan faceUpReplacementPlan) build(bindings instanceBindings) mage.Effect {
	return mage.TurnResolvedPermanentFaceUpBeforeAction(mage.FromDamageSource(bindings.castResults[plan.castResult]))
}

func compileFaceUpReplacement(n Node, ctx scope) (effectFactory, error) {
	plan, err := compileFaceUpReplacementPlan(n, ctx)
	if err != nil {
		return nil, err
	}
	return plan.build, nil
}

func compileFaceUpReplacementPlan(n Node, ctx scope) (faceUpReplacementPlan, error) {
	if n.Kind != Tuple || n.Name != "CreateReplaceWouldMaskUntil" {
		return faceUpReplacementPlan{}, unsupported(n, "face-up replacement action")
	}
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return faceUpReplacementPlan{}, unsupported(n, "face-up replacement context")
	}
	if err := arity(n, 3); err != nil {
		return faceUpReplacementPlan{}, err
	}
	event, operations, duration := n.Args[0], n.Args[1], n.Args[2]
	if event.Kind != Tuple || event.Name != "PermanentWouldAssignDamageDealDamageBeDealDamageOrBecomeTapped" || len(event.Args) != 1 {
		return faceUpReplacementPlan{}, unsupported(event, "face-up replacement event")
	}
	if operations.Kind != List || len(operations.Args) != 1 || (operations.Args[0].Kind != Symbol || operations.Args[0].Name != "TurnItFaceUpInstead") {
		return faceUpReplacementPlan{}, unsupported(operations, "face-up replacement operations")
	}
	if duration.Kind != Tuple || duration.Name != "UntilPermanentIsTurnedFaceUp" || len(duration.Args) != 1 {
		return faceUpReplacementPlan{}, unsupported(duration, "face-up replacement duration")
	}
	subject, err := compilePermanentRef(event.Args[0], ctx)
	if err != nil {
		return faceUpReplacementPlan{}, err
	}
	end, err := compilePermanentRef(duration.Args[0], ctx)
	if err != nil {
		return faceUpReplacementPlan{}, err
	}
	if subject.kind != permanentCastResult || end.kind != permanentCastResult || subject.castResult != end.castResult {
		return faceUpReplacementPlan{}, unsupported(n, "face-up replacement requires the same saved cast result")
	}
	return faceUpReplacementPlan{castResult: subject.castResult}, nil
}
