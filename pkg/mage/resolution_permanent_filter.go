package mage

// IfPermanentMatches uses current information or the saved information for the same object.
// An information read does not permit actions on an absent object.
func IfPermanentMatches(subject PermanentRef, filter PermanentFilter, body ResolvingEffect) ResolvingEffect {
	return &permanentInformationBranch{subject: subject, filter: filter, body: body}
}

type permanentInformationBranch struct {
	subject PermanentRef
	filter  PermanentFilter
	body    ResolvingEffect
}

func (e *permanentInformationBranch) Text() string { return "if permanent matches " + e.filter.Text() }
func (e *permanentInformationBranch) Properties() EffectProperties {
	if e.body == nil {
		return EffectProperties{}
	}
	return e.body.Properties()
}
func (e *permanentInformationBranch) Apply(ctx *EffectContext) error {
	p, lki := readPermanentInformation(e.subject, ctx)
	if lki != nil {
		p = lki.Snapshot
	}
	if p != nil && e.filter.Match(p, ctx.Game) {
		return e.body.Apply(ctx)
	}
	return nil
}
func (e *permanentInformationBranch) inspectDefinition() definitionNode {
	n := permanentInformationNode(e.subject, "resolution-branch", e.Text())
	n.children = append(n.children, definitionEdge{"then", e.body})
	return n
}
