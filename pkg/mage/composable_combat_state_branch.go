package mage

type combatStateBranch struct {
	subject   PermanentRef
	attacking bool
	body      ResolvingEffect
}

// IfPermanentAttacking checks combat membership when the effect resolves.
func IfPermanentAttacking(subject PermanentRef, body ResolvingEffect) ResolvingEffect {
	return &combatStateBranch{subject: subject, attacking: true, body: body}
}

// IfPermanentBlocking checks combat membership when the effect resolves.
func IfPermanentBlocking(subject PermanentRef, body ResolvingEffect) ResolvingEffect {
	return &combatStateBranch{subject: subject, body: body}
}

func (e *combatStateBranch) Text() string {
	if e.attacking {
		return "if permanent is attacking"
	}
	return "if permanent is blocking"
}

func (e *combatStateBranch) Properties() EffectProperties {
	if e.body == nil {
		return EffectProperties{}
	}
	return e.body.Properties()
}

func (e *combatStateBranch) Apply(ctx *EffectContext) error {
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok || p.PhasedOut {
		return nil
	}
	match := ctx.Game.IsBlockingIncarnation(p.ID(), p.incarnationID)
	if e.attacking {
		match = ctx.Game.IsAttackingIncarnation(p.ID(), p.incarnationID)
	}
	if match {
		return e.body.Apply(ctx)
	}
	return nil
}

func (e *combatStateBranch) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-branch", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"then", e.body}}}
}
