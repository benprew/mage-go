package ron

type damageTargetPlan struct{ name string }

func compileDamageTargetRef(n Node, ctx scope) (*damageTargetPlan, error) {
	if n.Kind == Symbol && n.Name == "Ref_AnyTarget" && ctx.damageTarget != nil && ctx.damageTarget.name == n.Name {
		return ctx.damageTarget, nil
	}
	return nil, problem(n, "unbound or incompatible damage recipient reference")
}
