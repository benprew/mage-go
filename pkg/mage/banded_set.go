package mage

type bandedWithSet struct{ subject PermanentRef }

// BandedWith selects present members of the subject's current attacking band.
func BandedWith(subject PermanentRef) PermanentSet {
	return bandedWithSet{subject: subject}
}

func (bandedWithSet) permanentSet() {}

func (s bandedWithSet) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "current attacking band"}
	if nilDefinition(s.subject) {
		n.problem = "band membership requires a permanent"
	} else {
		n.children = []definitionEdge{{"subject", s.subject}}
	}
	return n
}

func (s bandedWithSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.subject) || ctx == nil || ctx.Game == nil || ctx.Game.combat == nil {
		return nil
	}
	subject, ok := s.subject.resolvePermanent(ctx)
	combat := ctx.Game.combat
	if !ok || subject.PhasedOut || !combat.IsAttackingIncarnation(subject.ID(), subject.incarnationID) {
		return nil
	}
	var result []*Permanent
	for _, p := range ctx.Game.FilterBattlefield(AnyPermanent) {
		if !p.PhasedOut && combat.IsAttackingIncarnation(p.ID(), p.incarnationID) && combat.IsBandedWith(subject.ID(), p.ID()) {
			result = append(result, p)
		}
	}
	return result
}
