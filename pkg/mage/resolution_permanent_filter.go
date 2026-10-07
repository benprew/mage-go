package mage

// IfPermanentMatches uses current information or the saved information for the same object.
// An information read does not permit actions on an absent object.
func IfPermanentMatches(subject PermanentRef, filter PermanentFilter, body ResolvingEffect) ResolvingEffect {
	return &permanentInformationBranch{subject: subject, filter: filter, body: body}
}

// IfPermanentMatchesElse selects one body from current or retained object information.
// Missing information cannot select either body.
func IfPermanentMatchesElse(subject PermanentRef, filter PermanentFilter, matching, nonmatching ResolvingEffect) ResolvingEffect {
	return &permanentInformationBranch{subject: subject, filter: filter, body: matching, otherwise: nonmatching, hasElse: true}
}

// UnlessPermanentMatches runs the body only when known object information does not match.
func UnlessPermanentMatches(subject PermanentRef, filter PermanentFilter, body ResolvingEffect) ResolvingEffect {
	return &permanentInformationBranch{subject: subject, filter: filter, body: body, negate: true}
}

// UnlessPermanentCameUnderControlThisTurn uses the original object's control history.
// Missing object or player information cannot satisfy the negative condition.
func UnlessPermanentCameUnderControlThisTurn(subject PermanentRef, player PlayerRef, body ResolvingEffect) ResolvingEffect {
	return &permanentInformationBranch{subject: subject, controlPlayer: player, controlHistory: true, body: body, negate: true}
}

type permanentInformationBranch struct {
	subject        PermanentRef
	filter         PermanentFilter
	body           ResolvingEffect
	otherwise      ResolvingEffect
	hasElse        bool
	negate         bool
	controlPlayer  PlayerRef
	controlHistory bool
}

func (e *permanentInformationBranch) Text() string {
	if e.controlHistory {
		return "unless permanent came under player control this turn"
	}
	if e.negate {
		return "unless permanent matches " + e.filter.Text()
	}
	return "if permanent matches " + e.filter.Text()
}
func (e *permanentInformationBranch) Properties() EffectProperties {
	if e.hasElse || e.negate {
		// Separate branches must not supply a combined shortcut to the AI.
		return EffectProperties{}
	}
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
	if p == nil {
		return nil
	}
	var matches bool
	if e.controlHistory {
		player, ok := e.controlPlayer.resolvePlayer(ctx)
		if !ok || ctx.Game.GetPlayer(player) == nil {
			return nil
		}
		turn, recorded := p.controlGainedTurns[player]
		matches = recorded && turn == ctx.Game.CurrentTurn()
	} else {
		matches = e.filter.Match(p, ctx.Game)
	}
	if matches != e.negate {
		return e.body.Apply(ctx)
	}
	if e.hasElse {
		return e.otherwise.Apply(ctx)
	}
	return nil
}
func (e *permanentInformationBranch) inspectDefinition() definitionNode {
	n := permanentInformationNode(e.subject, "resolution-branch", e.Text())
	if e.controlHistory {
		n.children = append(n.children, definitionEdge{"player", e.controlPlayer})
	}
	n.children = append(n.children, definitionEdge{"then", e.body})
	if e.hasElse {
		n.children = append(n.children, definitionEdge{"else", e.otherwise})
	}
	return n
}
