package mage

import (
	"maps"

	"github.com/google/uuid"
)

// DamageResultsBinding names the final actions of one damage batch.
type DamageResultsBinding struct{ name string }

// DamageResultBinding names one final action during result iteration.
type DamageResultBinding struct{ name string }

func BindDamageResults(name string) *DamageResultsBinding { return &DamageResultsBinding{name} }
func BindDamageResult(name string) *DamageResultBinding   { return &DamageResultBinding{name} }

func (b *DamageResultsBinding) bindingName() string { return b.name }
func (b *DamageResultBinding) bindingName() string  { return b.name }
func (b *DamageResultsBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "damage-results-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}
func (b *DamageResultBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "damage-result-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}

type damageResultsOption struct{ binding *DamageResultsBinding }

// DamageResultsInto saves final positive damage actions for later instructions.
func DamageResultsInto(binding *DamageResultsBinding) DamageEachResultOption {
	return damageResultsOption{binding}
}
func (damageResultsOption) damageEachResultOption() {}
func (o damageResultsOption) applyDamageEachResult(e *damageEachEffect) {
	e.results = o.binding
	e.resultsCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

func cloneDamageOutcome(outcome damageOutcome) damageOutcome {
	outcome.source = cloneDamageSource(outcome.source)
	return outcome
}
func cloneDamageOutcomes(outcomes []damageOutcome) []damageOutcome {
	if len(outcomes) == 0 {
		return nil
	}
	cloned := make([]damageOutcome, len(outcomes))
	for i, outcome := range outcomes {
		cloned[i] = cloneDamageOutcome(outcome)
	}
	return cloned
}
func cloneDamageResultsBindings(bindings map[*DamageResultsBinding][]damageOutcome) map[*DamageResultsBinding][]damageOutcome {
	if bindings == nil {
		return nil
	}
	cloned := make(map[*DamageResultsBinding][]damageOutcome, len(bindings))
	for binding, outcomes := range bindings {
		cloned[binding] = cloneDamageOutcomes(outcomes)
	}
	return cloned
}
func cloneDamageSourceBindings(bindings map[*DamageSourceBinding]DamageSource) map[*DamageSourceBinding]DamageSource {
	if bindings == nil {
		return nil
	}
	cloned := make(map[*DamageSourceBinding]DamageSource, len(bindings))
	for binding, source := range bindings {
		cloned[binding] = cloneDamageSource(source)
	}
	return cloned
}

type damageResultLoop struct {
	results *DamageResultsBinding
	member  *DamageResultBinding
	body    ResolvingEffect
}

// ForEachDamageResult runs its body after the complete damage batch.
func ForEachDamageResult(results *DamageResultsBinding, member *DamageResultBinding, body ResolvingEffect) ResolvingEffect {
	return &damageResultLoop{results, member, body}
}
func (*damageResultLoop) Text() string { return "for each damage result" }
func (e *damageResultLoop) Properties() EffectProperties {
	var props EffectProperties
	if !nilDefinition(e.body) {
		props = e.body.Properties()
	}
	props.Mass = true
	return props
}
func (e *damageResultLoop) inspectDefinition() definitionNode {
	n := definitionNode{Role: "damage-result-loop", isolatedBindings: true, children: []definitionEdge{
		{"results", damageResultsRead{e.results}}, {"member", damageResultMember{e.member}}, {"body", e.body},
	}}
	if e.member == nil {
		n.problem = "supply a damage result member binding"
	}
	return n
}
func (e *damageResultLoop) Apply(ctx *EffectContext) error {
	outcomes := cloneDamageOutcomes(ctx.Game.resolution.damageResultsBindings[e.results])
	state := &ctx.Game.resolution
	outerNumbers, outerPlayers := state.numberBindings, state.playerBindings
	outerPermanents, outerSources := state.permanentBindings, state.damageSourceBindings
	outerResults, outerItems := state.damageResultsBindings, state.damageResultBindings
	defer func() {
		state.numberBindings, state.playerBindings = outerNumbers, outerPlayers
		state.permanentBindings, state.damageSourceBindings = outerPermanents, outerSources
		state.damageResultsBindings, state.damageResultBindings = outerResults, outerItems
	}()
	for _, outcome := range outcomes {
		state.numberBindings, state.playerBindings = maps.Clone(outerNumbers), maps.Clone(outerPlayers)
		state.permanentBindings, state.damageSourceBindings = maps.Clone(outerPermanents), cloneDamageSourceBindings(outerSources)
		state.damageResultsBindings = cloneDamageResultsBindings(outerResults)
		state.damageResultBindings = maps.Clone(outerItems)
		if state.damageResultBindings == nil {
			state.damageResultBindings = make(map[*DamageResultBinding]damageOutcome)
		}
		state.damageResultBindings[e.member] = outcome
		if err := e.body.Apply(ctx); err != nil {
			return err
		}
	}
	return nil
}

type damageResultsRead struct{ binding *DamageResultsBinding }

func (r damageResultsRead) inspectDefinition() definitionNode {
	return definitionNode{Role: "damage-results-ref", bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}

type damageResultMember struct{ binding *DamageResultBinding }

func (m damageResultMember) inspectDefinition() definitionNode {
	return definitionNode{Role: "damage-result-member", bindingWrite: m.binding, children: []definitionEdge{{"binding", m.binding}}}
}

type damageResultAmount struct{ binding *DamageResultBinding }

func DamageResultAmount(binding *DamageResultBinding) ResolutionValue {
	return damageResultAmount{binding}
}
func (r damageResultAmount) Text() string { return "damage result amount" }
func (r damageResultAmount) resolveValue(ctx *EffectContext) int {
	return ctx.Game.resolution.damageResultBindings[r.binding].amount
}
func (r damageResultAmount) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: r.Text(), bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}

type damageResultPlayer struct{ binding *DamageResultBinding }

func DamageResultPlayer(binding *DamageResultBinding) PlayerRef { return damageResultPlayer{binding} }
func (damageResultPlayer) playerRef()                           {}
func (damageResultPlayer) damageRecipient()                     {}
func (r damageResultPlayer) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	id := ctx.Game.resolution.damageResultBindings[r.binding].playerID
	return id, id != uuid.Nil && ctx.Game.GetPlayer(id) != nil
}
func (r damageResultPlayer) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (r damageResultPlayer) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-ref", Text: "damage result player", bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}

type damageResultPermanent struct{ binding *DamageResultBinding }

func DamageResultPermanent(binding *DamageResultBinding) PermanentRef {
	return damageResultPermanent{binding}
}
func (damageResultPermanent) permanentRef()    {}
func (damageResultPermanent) damageRecipient() {}
func (r damageResultPermanent) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return permanentRecipient(r, ctx)
}
func (r damageResultPermanent) resolvePermanent(ctx *EffectContext) (*Permanent, bool) {
	value := ctx.Game.resolution.damageResultBindings[r.binding].permanent
	permanent := ctx.Game.FindPermanent(value.id)
	if permanent == nil || permanent.PhasedOut || value.incarnation == uuid.Nil || permanent.incarnationID != value.incarnation {
		return nil, false
	}
	return permanent, true
}
func (r damageResultPermanent) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-ref", Text: "damage result permanent", bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}

type damageResultSource struct{ binding *DamageResultBinding }

func DamageResultSource(binding *DamageResultBinding) DamageSourceRef {
	return damageResultSource{binding}
}
func (damageResultSource) damageSourceRef() {}
func (r damageResultSource) resolveDamageSource(ctx *EffectContext) DamageSource {
	return ctx.Game.resolution.damageResultBindings[r.binding].source
}
func (r damageResultSource) inspectDefinition() definitionNode {
	return definitionNode{Role: "damage-source-ref", Text: "damage result source", bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}
