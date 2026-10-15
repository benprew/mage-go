package mage

type exileEntryContinuation struct {
	entry               *exileEntryEffect
	entered, notEntered ResolvingEffect
}

// PutExiledCardOntoBattlefieldOrElse selects a branch after one exact exile visit.
// Failed entry clears the result. Only the success body can read the new result.
func PutExiledCardOntoBattlefieldOrElse(card CardRef, result *PermanentBinding, entered, notEntered ResolvingEffect, options ...BattlefieldEntryOption) ResolvingEffect {
	return &exileEntryContinuation{
		entry:   &exileEntryEffect{subjects: ExiledCardFrom(card), entry: newBattlefieldEntryOptions(options), result: result, saveResult: true},
		entered: entered, notEntered: notEntered,
	}
}
func (*exileEntryContinuation) Text() string {
	return "put exiled card onto battlefield and select a branch"
}
func (e *exileEntryContinuation) Properties() EffectProperties {
	props := EffectProperties{Outcome: OutcomeBenefit}
	for _, body := range []ResolvingEffect{e.entered, e.notEntered} {
		if !nilDefinition(body) {
			mergeEffectProperties(&props, body.Properties())
		}
	}
	return props
}
func (e *exileEntryContinuation) inspectDefinition() definitionNode {
	success := []Effect{e.entry}
	if !nilDefinition(e.entered) {
		success = append(success, e.entered)
	}
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"success", Sequence(success...)}}}
	if !nilDefinition(e.notEntered) {
		n.children = append(n.children, definitionEdge{"failure", e.notEntered})
	}
	if nilDefinition(e.entered) && nilDefinition(e.notEntered) {
		n.problem = "supply at least one entry branch"
	}
	return n
}
func (e *exileEntryContinuation) refersToSourcePermanent() bool {
	return e.entry.refersToSourcePermanent() || effectReadsSourcePermanent(e.entered) || effectReadsSourcePermanent(e.notEntered)
}
func (e *exileEntryContinuation) Apply(ctx *EffectContext) error {
	entered := e.entry.apply(ctx)
	body := e.notEntered
	if entered {
		body = e.entered
	}
	if !nilDefinition(body) {
		return body.Apply(ctx)
	}
	return nil
}
