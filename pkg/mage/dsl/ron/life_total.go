package ron

import "github.com/benprew/mage-go/pkg/mage"

type lifeSubjectPlan struct {
	name       string
	referenced bool
}

func compileLifeTotal(n Node, ctx scope) (valueFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if !ctx.resolving || ctx.drawReplacementBody {
		return nil, problem(n, "life total requires an ordinary resolving body")
	}
	subject, err := compilePlayerRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.ResolutionValue {
		return mage.LifeTotalOf(subject.build(bindings))
	}, nil
}
