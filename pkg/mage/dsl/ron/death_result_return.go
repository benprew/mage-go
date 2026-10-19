package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileDeathResultReturn(producer, loop Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(loop, "death-result iteration requires an ordinary resolving body")
	}
	if err := arity(producer, 1); err != nil {
		return nil, err
	}
	subjects, err := compileContextualCollection(producer.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if err := arity(loop, 1); err != nil {
		return nil, err
	}
	parts, err := list(loop.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 1 || parts[0].Name != "ReturnACardFromPlayersGraveyardToBattlefield" {
		return nil, problem(loop, "supply one creature-card return under its owner's control")
	}
	action := parts[0]
	plan := &permanentLoopPlan{name: "death member at " + loop.Position.String(), deathResult: true}
	ctx.permanentLoop = plan
	if _, err := compileChosenGraveyardReturn(action, ctx); err != nil {
		return nil, err
	}
	entry := action.Args[2].Args[0]
	if entry.Kind != Symbol || entry.Name != "EntersUnderOwnersControl" {
		return nil, unsupported(entry, "death-result return entry operation")
	}
	graveyard, err := compilePlayerRef(action.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		result := mage.BindPermanentSet("creature deaths at " + producer.Position.String())
		member := mage.BindPermanent(plan.name)
		bindings.permanentLoops = maps.Clone(bindings.permanentLoops)
		if bindings.permanentLoops == nil {
			bindings.permanentLoops = make(map[*permanentLoopPlan]*mage.PermanentBinding)
		}
		bindings.permanentLoops[plan] = member
		destroy := (mage.CreatureDeathInstruction{Subjects: subjects(bindings), Result: result, CannotRegenerate: producer.Name == "DestroyEachPermanentNoRegen"}).Effect()
		ret := (mage.CreatureDeathReturnInstruction{Result: result, Member: member, Graveyard: graveyard.build(bindings)}).Effect()
		return mage.Sequence(destroy, ret)
	}, nil
}
