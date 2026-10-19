package mage

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// RetainedBlockerFilter combines current properties and saved permanent visits.
// Its zero value is not a valid declaration.
type RetainedBlockerFilter struct {
	kind    string
	filter  PermanentFilter
	members PermanentSet
	parts   []RetainedBlockerFilter
}

// CurrentBlockers checks current properties when block legality is checked.
func CurrentBlockers(filter PermanentFilter) RetainedBlockerFilter {
	return RetainedBlockerFilter{kind: "current", filter: filter}
}

// SavedBlockers saves exact visits when the restriction starts.
func SavedBlockers(members PermanentSet) RetainedBlockerFilter {
	return RetainedBlockerFilter{kind: "saved", members: members}
}

// AllRetainedBlockers requires every supplied condition.
func AllRetainedBlockers(parts ...RetainedBlockerFilter) RetainedBlockerFilter {
	return RetainedBlockerFilter{kind: "and", parts: slices.Clone(parts)}
}

// AnyRetainedBlockers requires at least one supplied condition.
func AnyRetainedBlockers(parts ...RetainedBlockerFilter) RetainedBlockerFilter {
	return RetainedBlockerFilter{kind: "or", parts: slices.Clone(parts)}
}

func (f RetainedBlockerFilter) inspectDefinition() definitionNode {
	n := definitionNode{Role: "retained-blocker-filter", Text: f.kind}
	switch f.kind {
	case "current":
	case "saved":
		n.children = []definitionEdge{{"members", f.members}}
	case "and", "or":
		if len(f.parts) == 0 {
			n.problem = "supply at least one blocker condition"
		}
		for _, part := range f.parts {
			n.children = append(n.children, definitionEdge{"condition", part})
		}
	default:
		n.problem = "supply a blocker condition"
	}
	return n
}

func (f RetainedBlockerFilter) capture(ctx *EffectContext) PermanentFilter {
	switch f.kind {
	case "current":
		return f.filter
	case "saved":
		visits := make(map[capturedPermanent]bool)
		for _, p := range f.members.resolvePermanents(ctx) {
			if p != nil && !p.PhasedOut {
				visits[capturedPermanent{p.ID(), p.incarnationID}] = true
			}
		}
		return NewPermanentFilter("saved blocker visits", func(p *Permanent, _ *Game) bool {
			return p != nil && !p.PhasedOut && visits[capturedPermanent{p.ID(), p.incarnationID}]
		})
	default:
		parts := make([]PermanentFilter, len(f.parts))
		for i, part := range f.parts {
			parts[i] = part.capture(ctx)
		}
		if f.kind == "and" {
			return And(parts...)
		}
		return Or(parts...)
	}
}

func (f RetainedBlockerFilter) readsSourcePermanent() bool {
	if f.kind == "saved" && permanentSetReadsSourcePermanent(f.members) {
		return true
	}
	return slices.ContainsFunc(f.parts, func(part RetainedBlockerFilter) bool { return part.readsSourcePermanent() })
}

// RetainBlockerEligibility saves eligible visits and installs a blocking rule.
// Later entries cannot pass the rule through a current-property condition.
func RetainBlockerEligibility(duration core.Duration, subject PermanentRef, eligible PermanentSet, allowed RetainedBlockerFilter) ResolvingEffect {
	return &retainedBlockerEffect{duration, subject, eligible, allowed}
}

type retainedBlockerEffect struct {
	duration core.Duration
	subject  PermanentRef
	eligible PermanentSet
	allowed  RetainedBlockerFilter
}

func (*retainedBlockerEffect) Text() string { return "retain blocker eligibility" }
func (*retainedBlockerEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *retainedBlockerEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "retained-blocker-eligibility", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"eligible", e.eligible}, {"allowed", e.allowed}}}
	if e.duration != core.EndOfCombat && e.duration != core.EndOfTurn {
		n.problem = "supply EndOfCombat or EndOfTurn"
	}
	return n
}
func (e *retainedBlockerEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject) || permanentSetReadsSourcePermanent(e.eligible) || e.allowed.readsSourcePermanent()
}
func (e *retainedBlockerEffect) Apply(ctx *EffectContext) error {
	subject, ok := e.subject.resolvePermanent(ctx)
	if !ok || subject.PhasedOut {
		return nil
	}
	filter := And(SavedBlockers(e.eligible).capture(ctx), e.allowed.capture(ctx))
	return Until(e.duration, CantBeBlockedExceptBy(e.subject, filter)).Apply(ctx)
}
