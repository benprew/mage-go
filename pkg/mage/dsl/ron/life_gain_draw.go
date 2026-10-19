package ron

import "github.com/benprew/mage-go/pkg/mage"

type lifeGainDrawPlan struct{ subject playerReferencePlan }

func (p lifeGainDrawPlan) build() mage.CardOption {
	return mage.WithStaticAbility(mage.DrawInsteadOfLifeGain(p.subject.build(instanceBindings{})))
}

func compileLifeGainDrawReplacement(n Node) (optionFactory, error) {
	plan, err := compileLifeGainDrawPlan(n)
	if err != nil {
		return nil, err
	}
	return plan.build, nil
}

func compileLifeGainDrawPlan(n Node) (*lifeGainDrawPlan, error) {
	if n.Name != "ReplaceWouldGainLife" {
		return nil, unsupported(n, "life gain replacement")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name != "PlayerWouldGainLife" {
		return nil, unsupported(event, "life gain replacement event")
	}
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	if event.Args[0].Kind != Symbol || event.Args[0].Name != "You" {
		return nil, unsupported(event.Args[0], "life gain replacement player")
	}
	ops, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(ops) != 1 || ops[0].Name != "DrawNumberCards" {
		return nil, problem(n.Args[1], "supply one card draw operation")
	}
	if err := arity(ops[0], 1); err != nil {
		return nil, err
	}
	amount := ops[0].Args[0]
	if amount.Kind != Symbol || amount.Name != "WouldGainLife_LifeAmount" {
		return nil, unsupported(amount, "life gain replacement amount")
	}
	subject, err := compilePlayerRef(event.Args[0], scope{})
	if err != nil {
		return nil, err
	}
	return &lifeGainDrawPlan{subject: subject}, nil
}
