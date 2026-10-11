package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// PermanentMatchesFilterWithLKI reads the source or attachment's exact information.
// Saved information does not permit an action on an absent object.
func PermanentMatchesFilterWithLKI(subject PermanentRef, filter PermanentFilter) StateCondition {
	return permanentFilterLKICondition{subject: subject, filter: filter}
}

type permanentFilterLKICondition struct {
	subject PermanentRef
	filter  PermanentFilter
}

func (permanentFilterLKICondition) stateCondition() {}
func (c permanentFilterLKICondition) checkState(reader GameReader, state StateContext) bool {
	game, ok := reader.(*Game)
	if !ok {
		return false
	}
	ctx := &EffectContext{Game: game, SourceID: state.SourceID, Controller: state.ControllerID, sourceCaptured: state.SourceIncarnation != uuid.Nil, sourceIncarnation: state.SourceIncarnation}
	p, lki := readPermanentInformation(c.subject, ctx)
	if p == nil && lki != nil {
		p = lki.Snapshot
	}
	return p != nil && c.filter.Match(p, game)
}
func (c permanentFilterLKICondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", Text: "permanent matches with saved information", battlefieldSource: true}
	switch c.subject.(type) {
	case sourceRef, attachedPermanentRef:
	default:
		n.problem = "condition requires This or AttachedPermanent"
	}
	return n
}

type deathNotSacrificedOption struct{}

// WithDeathNotSacrificed checks the death cause at detection and resolution.
// Each pending trigger keeps its own event facts.
func WithDeathNotSacrificed() deathNotSacrificedOption { return deathNotSacrificedOption{} }
func (deathNotSacrificedOption) triggerOption()        {}
func (deathNotSacrificedOption) applyTrigger(t *GenericTriggered) {
	if t.deathNotSacrificed {
		t.duplicateDeathCause = true
	}
	t.deathNotSacrificed = true
	t.AndConditionData(nonSacrificeDeathCondition{})
	t.effects = []Effect{&deathNotSacrificedBranch{body: Sequence(t.effects...)}}
}

type nonSacrificeDeathCondition struct{}

func (nonSacrificeDeathCondition) CheckTriggerCond(event *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
	return event != nil && event.Type == EvtZoneChange && event.FromZone == ZoneBattlefield && event.ToZone == ZoneGraveyard && !event.Flag
}

type deathNotSacrificedBranch struct {
	body     ResolvingEffect
	event    GameEvent
	captured bool
}

func (*deathNotSacrificedBranch) Text() string                   { return "if death was not a sacrifice" }
func (e *deathNotSacrificedBranch) Properties() EffectProperties { return e.body.Properties() }
func (e *deathNotSacrificedBranch) Apply(ctx *EffectContext) error {
	if e.captured && (nonSacrificeDeathCondition{}).CheckTriggerCond(&e.event, ctx.Game, ctx.SourceID, ctx.Controller) {
		return e.body.Apply(ctx)
	}
	return nil
}
func (e *deathNotSacrificedBranch) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-branch", Text: e.Text(), children: []definitionEdge{{"body", e.body}}}
}

func captureDeathCause(effects []Effect, event *GameEvent) []Effect {
	result := effects
	for i, effect := range result {
		branch, ok := effect.(*deathNotSacrificedBranch)
		if !ok {
			continue
		}
		result = append([]Effect(nil), effects...)
		captured := *branch
		captured.event = *event
		captured.event.RelatedIDs = append([]uuid.UUID(nil), event.RelatedIDs...)
		captured.captured = true
		result[i] = &captured
	}
	return result
}

func validDeathCauseContext(t *GenericTriggered) bool {
	if t.duplicateDeathCause || t.interveningDefinition != nil || len(t.modes) != 0 || len(t.effects) != 1 {
		return false
	}
	switch p := t.eventDefinition.(type) {
	case creatureDiesPattern:
		return true
	case sourceEventPattern:
		return p.dies
	default:
		return false
	}
}
