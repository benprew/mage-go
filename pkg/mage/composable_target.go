package mage

import (
	"fmt"
	"slices"

	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// --- from composable_target.go ---
// TargetBinding declares a target group. The declaration stores no choices.
type TargetBinding interface {
	targetBinding()
	targetSpec() Target
	targetName() string
}

type targetDeclaration struct {
	name                   string
	spec                   Target
	optionalGraveyardCount int
}

func (*targetDeclaration) targetBinding()       {}
func (b *targetDeclaration) targetSpec() Target { return b }
func (b *targetDeclaration) targetName() string { return b.name }
func (b *targetDeclaration) Possible(c uuid.UUID, card Card, g *Game) []uuid.UUID {
	return b.spec.Possible(c, card, g)
}
func (*targetDeclaration) Choose(uuid.UUID, Card, *Game, []uuid.UUID) error { return nil }
func (*targetDeclaration) Chosen() []uuid.UUID                              { return nil }
func (*targetDeclaration) IsChosen() bool                                   { return false }
func (*targetDeclaration) Reset()                                           {}
func (b *targetDeclaration) Min() int                                       { return b.spec.Min() }
func (b *targetDeclaration) Max() int                                       { return b.spec.Max() }
func (b *targetDeclaration) BoundsForX(x int) (minimum, maximum int)        { return TargetBounds(b.spec, x) }

// UnwrapTarget returns the underlying Target if wrapped in a targetDeclaration.
func UnwrapTarget(t Target) Target {
	if td, ok := t.(*targetDeclaration); ok {
		return td.spec
	}
	return t
}

// PermanentTargetBinding is a singular permanent target reference.
type PermanentTargetBinding struct{ *targetDeclaration }

// DeclarePermanentTarget declares one permanent target. DSL exports it as TargetPermanent.
func DeclarePermanentTarget(name string, filter PermanentFilter, options ...PermanentTargetOption) *PermanentTargetBinding {
	spec := TargetPermanent(filter).(*PermanentTarget)
	applyPermanentTargetOptions(spec, options)
	return &PermanentTargetBinding{&targetDeclaration{name: name, spec: spec}}
}
func (*PermanentTargetBinding) permanentRef() {}
func (b *PermanentTargetBinding) resolvePermanent(ctx *EffectContext) (*Permanent, bool) {
	values := ctx.targetBindings[b.targetDeclaration]
	if len(values) == 0 {
		return nil, false
	}
	value, _ := firstTargetValue(values)
	p := ctx.Game.FindPermanent(value.id)
	return p, p != nil && !p.PhasedOut && p.incarnationID == value.incarnation
}
func (b *PermanentTargetBinding) inspectDefinition() definitionNode {
	return targetReadNode(b.targetDeclaration, "permanent-ref")
}

// PlayerTargetBinding is a singular player target reference.
type PlayerTargetBinding struct{ *targetDeclaration }

// DeclarePlayerTarget declares one player target. DSL exports it as TargetPlayer.
func DeclarePlayerTarget(name string, options ...PlayerTargetOption) *PlayerTargetBinding {
	spec := TargetPlayer()
	for _, option := range options {
		if nilDefinition(option) {
			panic("targets: supply a non-nil player target option")
		}
		spec = option.applyPlayerTarget(spec)
	}
	return &PlayerTargetBinding{&targetDeclaration{name: name, spec: spec}}
}

// PlayerTargetOption adds a restriction to a player target declaration.
type PlayerTargetOption interface {
	playerTargetOption()
	applyPlayerTarget(Target) Target
}

type targetOpponentOnlyOption struct{}

// TargetOpponentOnly restricts the target to an opponent of the ability controller.
func TargetOpponentOnly() PlayerTargetOption         { return targetOpponentOnlyOption{} }
func (targetOpponentOnlyOption) playerTargetOption() {}
func (targetOpponentOnlyOption) applyPlayerTarget(target Target) Target {
	result := TargetOpponent().(*OpponentTarget)
	switch target := target.(type) {
	case *PlayerTarget:
		result.attackedThisTurn = target.attackedThisTurn
	case *OpponentTarget:
		result.attackedThisTurn = target.attackedThisTurn
	}
	return result
}
func (targetOpponentOnlyOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-player", Text: "opponent"}
}
func (*PlayerTargetBinding) playerRef() {}
func (b *PlayerTargetBinding) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	values := ctx.targetBindings[b.targetDeclaration]
	value, ok := firstTargetValue(values)
	return value.id, ok && ctx.Game.GetPlayer(value.id) != nil
}
func (b *PlayerTargetBinding) inspectDefinition() definitionNode {
	return targetReadNode(b.targetDeclaration, "player-ref")
}
func targetReadNode(b *targetDeclaration, role string) definitionNode {
	n := definitionNode{Role: role, targetRead: b}
	if b == nil {
		n.problem = "supply a target binding"
	} else {
		n.Text = b.name
	}
	return n
}

type targetOptions struct{ bindings []TargetBinding }

// WithTargets declares targets for a spell, activation, or trigger.
func WithTargets(bindings ...TargetBinding) targetOptions {
	return targetOptions{append([]TargetBinding(nil), bindings...)}
}
func (targetOptions) spellOption()                          {}
func (targetOptions) activationOption()                     {}
func (o targetOptions) applySpell(a *ActionDefinition)      { applyTargetOptions(a, o.bindings) }
func (o targetOptions) applyActivation(a *ActionDefinition) { applyTargetOptions(a, o.bindings) }

type spellTargetGroupsOption struct{ bindings []TargetBinding }

// WithTargetGroups declares spell targets chosen as separate groups at casting.
func WithTargetGroups(bindings ...TargetBinding) SpellOption {
	return spellTargetGroupsOption{bindings: append([]TargetBinding(nil), bindings...)}
}
func (spellTargetGroupsOption) spellOption() {}
func (o spellTargetGroupsOption) applySpell(a *ActionDefinition) {
	a.groupedTargets = true
	applyTargetOptions(a, o.bindings)
}

type activationTargetGroupsOption struct{ bindings []TargetBinding }

// WithActivationTargetGroups declares separate target groups for an activation.
func WithActivationTargetGroups(bindings ...TargetBinding) ActivationOption {
	return activationTargetGroupsOption{bindings: append([]TargetBinding(nil), bindings...)}
}
func (activationTargetGroupsOption) activationOption() {}
func (o activationTargetGroupsOption) applyActivation(a *ActionDefinition) {
	a.groupedTargets = true
	applyTargetOptions(a, o.bindings)
}
func applyTargetOptions(a *ActionDefinition, bindings []TargetBinding) {
	for _, b := range bindings {
		if nilDefinition(b) {
			panic("targets: supply a non-nil binding")
		}
		a.targets = append(a.targets, b.targetSpec())
	}
}

// SpellOption is a closed option for a resolving spell ability.
type SpellOption interface {
	spellOption()
	applySpell(*ActionDefinition)
}

// ActivationOption is a closed option for an activated ability.
type ActivationOption interface {
	activationOption()
	applyActivation(*ActionDefinition)
}

// Spell creates a spell ability with one effect body.
func Spell(effect ResolvingEffect, options ...SpellOption) *ActionDefinition {
	a := NewAction(ActionSpell, WithEffect(effect))
	for _, o := range options {
		if nilDefinition(o) {
			panic("spell: supply a non-nil option")
		}
		o.applySpell(a)
	}
	return a
}

func declaredTargetNodes(targets []Target, grouped ...bool) []definitionEdge {
	var edges []definitionEdge
	if len(targets) > 0 {
		edges = append(edges, definitionEdge{"target-groups", targetGroupContract{targets: targets, grouped: len(grouped) > 0 && grouped[0]}})
	}
	for i, t := range targets {
		if b, ok := t.(*targetDeclaration); ok {
			edges = append(edges, definitionEdge{fmt.Sprintf("targets[%d]", i), b})
		}
	}
	return edges
}
func (b *targetDeclaration) inspectDefinition() definitionNode {
	n := definitionNode{Role: "target-binding", Text: b.name, targetDeclaration: b}
	if spec, ok := b.spec.(*PermanentTarget); ok {
		for i, relation := range spec.combatRelations {
			n.children = append(n.children, definitionEdge{fmt.Sprintf("combatRelations[%d]", i), targetCombatRelationOption{relation}})
		}
		for i, player := range spec.controllerRefs {
			n.children = append(n.children, definitionEdge{fmt.Sprintf("controllerRefs[%d]", i), targetControlledByOption{player}})
		}
	}
	if b.optionalGraveyardCount > 1 {
		n.problem = "supply up to one graveyard option once"
	}
	if restriction, ok := b.spec.(*manaValueBoundGraveyardTarget); ok {
		n.children = append(n.children, definitionEdge{"restriction", restriction})
	}
	if _, ok := b.spec.(*OpponentTarget); ok {
		n.children = append(n.children, definitionEdge{"player", targetOpponentOnlyOption{}})
	}
	switch spec := b.spec.(type) {
	case *PlayerTarget:
		if spec.attackedThisTurn {
			n.children = append(n.children, definitionEdge{"history", targetPlayerAttackedThisTurnOption{}})
		}
	case *OpponentTarget:
		if spec.attackedThisTurn {
			n.children = append(n.children, definitionEdge{"history", targetPlayerAttackedThisTurnOption{}})
		}
	}
	if spec, ok := b.spec.(*PermanentTarget); ok && spec.controllerOnly {
		n.children = append(n.children, definitionEdge{"controller", targetYouControlOption{}})
	}
	if spec, ok := b.spec.(*PermanentTarget); ok && spec.opponentOnly {
		n.children = append(n.children, definitionEdge{"controller", targetOpponentControlsOption{}})
	}
	if spec, ok := b.spec.(*PermanentTarget); ok && spec.controllerOwned {
		n.children = append(n.children, definitionEdge{"owner", targetYouOwnOption{}})
	}
	if spec, ok := b.spec.(*PermanentTarget); ok && spec.attachedToControllerOnly {
		n.children = append(n.children, definitionEdge{"attachment-controller", targetAttachedToYouControlOption{}})
	}
	if isOpponentChosenTarget(b.spec) {
		n.children = append(n.children, definitionEdge{"chooser", targetChosenByOpponentOption{}})
	}
	if spec, ok := b.spec.(*PermanentTarget); ok && spec.manaValueEqualX {
		n.children = append(n.children, definitionEdge{"mana value", targetManaValueEqualXOption{}})
	}
	if spec, ok := b.spec.(*activatedAbilityTarget); ok {
		n.children = append(n.children, definitionEdge{"source type", spec})
	}
	if spec, ok := b.spec.(*typedSpellTarget); ok && spec.manaValueEqualX {
		n.children = append(n.children, definitionEdge{"mana value", targetManaValueEqualXOption{}})
	}
	emptyPermanentGroup := false
	if spec, ok := b.spec.(*PermanentTarget); ok {
		emptyPermanentGroup = spec.Min() == 0 && spec.Max() == 0
	}
	if b.name == "" {
		n.problem = "supply a target binding name"
	} else if b.spec == nil || b.spec.Min() < 0 || (b.spec.Max() < 1 && !emptyPermanentGroup) || b.spec.Min() > b.spec.Max() {
		n.problem = "supply valid target group bounds"
	}
	return n
}

// CostGroup distinguishes an explicit empty payment from a missing payment.
type CostGroup struct {
	costs   []Cost
	present bool
}

// Costs groups payment instructions for the existing total-cost transaction.
func Costs(costs ...Cost) CostGroup { return CostGroup{append([]Cost(nil), costs...), true} }

// ManaPayment declares a mana payment.
func ManaPayment(value string) Cost { return ManaCostOf(value) }

// Activate creates an activated ability with typed options and a single body.
func Activate(costs CostGroup, effect ResolvingEffect, options ...ActivationOption) *ActionDefinition {
	if !costs.present {
		panic("costs: supply an explicit Costs group")
	}
	a := NewAction(ActionActivated, WithEffect(effect))
	for _, c := range costs.costs {
		if nilDefinition(c) {
			panic("costs: supply a non-nil cost")
		}
		a.costs = append(a.costs, c)
	}
	for _, o := range options {
		if nilDefinition(o) {
			panic("activation: supply a non-nil option")
		}
		o.applyActivation(a)
	}
	return a
}

func firstTargetValue(values []capturedPermanent) (capturedPermanent, bool) {
	if len(values) == 0 {
		return capturedPermanent{}, false
	}
	return values[0], true
}

// PermanentTargetsBinding is a collection target declaration, not a singular reference.
type PermanentTargetsBinding struct{ *targetDeclaration }

// DeclarePermanentTargets declares a bounded collection of permanent targets.
func DeclarePermanentTargets(name string, minimum, maximum int, filter PermanentFilter, options ...PermanentTargetOption) *PermanentTargetsBinding {
	if minimum < 0 || maximum < minimum {
		panic("targets: require 0 <= minimum <= maximum")
	}
	spec := &PermanentTarget{BaseTarget: BaseTarget{min: minimum, max: maximum}, Filters: []PermanentFilter{filter}}
	applyPermanentTargetOptions(spec, options)
	return &PermanentTargetsBinding{&targetDeclaration{name: name, spec: spec}}
}

// DeclareXPermanentTargets requires exactly X permanent targets.
func DeclareXPermanentTargets(name string, filter PermanentFilter, options ...PermanentTargetOption) *PermanentTargetsBinding {
	return declareXBoundPermanentTargets(name, filter, false, options)
}

// DeclareUpToXPermanentTargets permits up to X permanent targets.
func DeclareUpToXPermanentTargets(name string, filter PermanentFilter, options ...PermanentTargetOption) *PermanentTargetsBinding {
	return declareXBoundPermanentTargets(name, filter, true, options)
}

func declareXBoundPermanentTargets(name string, filter PermanentFilter, upTo bool, options []PermanentTargetOption) *PermanentTargetsBinding {
	spec := TargetPermanent(filter).(*PermanentTarget)
	applyPermanentTargetOptions(spec, options)
	return &PermanentTargetsBinding{&targetDeclaration{name: name, spec: &countBoundTarget{inner: spec, usesX: !upTo, upToX: upTo}}}
}
func (*PermanentTargetsBinding) permanentSet() {}
func (b *PermanentTargetsBinding) resolvePermanents(ctx *EffectContext) []*Permanent {
	values := ctx.targetBindings[b.targetDeclaration]
	result := make([]*Permanent, 0, len(values))
	for _, value := range values {
		p := ctx.Game.FindPermanent(value.id)
		if p != nil && !p.PhasedOut && p.incarnationID == value.incarnation {
			result = append(result, p)
		}
	}
	return result
}
func (b *PermanentTargetsBinding) inspectDefinition() definitionNode {
	return targetReadNode(b.targetDeclaration, "permanent-set")
}

type targetGroupContract struct {
	targets []Target
	grouped bool
}

func (c targetGroupContract) inspectDefinition() definitionNode {
	n := definitionNode{Role: "target-groups"}
	variable, typed := 0, false
	for _, t := range c.targets {
		if _, ok := t.(*targetDeclaration); ok {
			typed = true
		}
		if !nilDefinition(t) && t.Min() != t.Max() {
			variable++
		}
	}
	if typed && variable > 1 && !c.grouped {
		n.problem = "multiple variable target groups require recorded selection boundaries"
	}
	return n
}

// PermanentTargetOption adds a restriction to a permanent target declaration.
type PermanentTargetOption interface {
	permanentTargetOption()
	applyPermanentTarget(*PermanentTarget)
}

type targetYouControlOption struct{}

// TargetYouControl restricts targets to permanents controlled by the ability controller.
func TargetYouControl() PermanentTargetOption                          { return targetYouControlOption{} }
func (targetYouControlOption) permanentTargetOption()                  {}
func (targetYouControlOption) applyPermanentTarget(t *PermanentTarget) { t.controllerOnly = true }
func applyPermanentTargetOptions(t *PermanentTarget, options []PermanentTargetOption) {
	for _, option := range options {
		if nilDefinition(option) {
			panic("targets: supply a non-nil permanent target option")
		}
		option.applyPermanentTarget(t)
	}
	if t.controllerOnly && t.opponentOnly {
		panic("targets: conflicting controller restrictions")
	}
}

func (targetYouControlOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-controller", Text: "you control"}
}

type targetOpponentControlsOption struct{}

// TargetOpponentControls restricts targets to permanents an opponent controls.
func TargetOpponentControls() PermanentTargetOption                          { return targetOpponentControlsOption{} }
func (targetOpponentControlsOption) permanentTargetOption()                  {}
func (targetOpponentControlsOption) applyPermanentTarget(t *PermanentTarget) { t.opponentOnly = true }
func (targetOpponentControlsOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-controller", Text: "an opponent controls"}
}

type targetChosenByOpponentOption struct{}

// TargetChosenByOpponent lets the opponent choose the targets at stack placement.
func TargetChosenByOpponent() PermanentTargetOption         { return targetChosenByOpponentOption{} }
func (targetChosenByOpponentOption) permanentTargetOption() {}
func (targetChosenByOpponentOption) applyPermanentTarget(t *PermanentTarget) {
	t.opponentChooses = true
}
func (targetChosenByOpponentOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-chooser", Text: "opponent"}
}

// OpponentChoosesTarget reports which player must choose the targets.
func (t *PermanentTarget) OpponentChoosesTarget() bool { return t.opponentChooses }

// OpponentChoosesTarget keeps the chooser of the enclosed target specification.
func (b *targetDeclaration) OpponentChoosesTarget() bool { return isOpponentChosenTarget(b.spec) }

type targetYouOwnOption struct{}

// TargetYouOwn restricts targets to permanents owned by the ability controller.
func TargetYouOwn() PermanentTargetOption                          { return targetYouOwnOption{} }
func (targetYouOwnOption) permanentTargetOption()                  {}
func (targetYouOwnOption) applyPermanentTarget(t *PermanentTarget) { t.controllerOwned = true }
func (targetYouOwnOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-owner", Text: "you own"}
}

// TargetOtherThanSource excludes the source card from the target choices.
func TargetOtherThanSource() PermanentTargetOption { return targetOtherThanSourceOption{} }

type targetOtherThanSourceOption struct{}

func (targetOtherThanSourceOption) permanentTargetOption()                  {}
func (targetOtherThanSourceOption) applyPermanentTarget(t *PermanentTarget) { t.excludeSource = true }
func (targetOtherThanSourceOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-exclude-source", Text: "other than this permanent"}
}

type targetAttachedToYouControlOption struct{}

// TargetAttachedToYouControl restricts targets to attachments on permanents you control.
func TargetAttachedToYouControl() PermanentTargetOption         { return targetAttachedToYouControlOption{} }
func (targetAttachedToYouControlOption) permanentTargetOption() {}
func (targetAttachedToYouControlOption) applyPermanentTarget(t *PermanentTarget) {
	t.attachedToControllerOnly = true
}
func (targetAttachedToYouControlOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-attachment-controller", Text: "you control"}
}

// --- from composable_target_count.go ---
type announcedTargetCountValue struct{ targets *PermanentTargetsBinding }

// AnnouncedTargetCount reads the chosen size of one permanent target group.
func AnnouncedTargetCount(targets *PermanentTargetsBinding) ResolutionValue {
	return announcedTargetCountValue{targets: targets}
}

func (v announcedTargetCountValue) Text() string { return "number of announced targets" }

func (v announcedTargetCountValue) resolveValue(ctx *EffectContext) int {
	if v.targets == nil || ctx == nil || ctx.Game == nil {
		return 0
	}
	count := 0
	for i, spec := range ctx.Game.resolution.targetSpecs {
		if i < len(ctx.Game.resolution.resolvingTargets) && spec == v.targets.targetDeclaration && ctx.Game.resolution.resolvingTargets[i] != uuid.Nil {
			count++
		}
	}
	return count
}

func (v announcedTargetCountValue) inspectDefinition() definitionNode {
	if v.targets == nil {
		return definitionNode{Role: "resolution-value", problem: "supply a permanent target group"}
	}
	return definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"targets", v.targets}}}
}

// --- from composable_binding.go ---
// PermanentBinding declares a saved permanent. Each resolution stores its own value.
// The name is for diagnostics. The declaration identifies the binding.
type PermanentBinding struct{ name string }

// BindPermanent declares a snapshot binding with a diagnostic name.
func BindPermanent(name string) *PermanentBinding { return &PermanentBinding{name: name} }

type capturedPermanent struct{ id, incarnation uuid.UUID }
type savedPermanentRef struct{ binding *PermanentBinding }

// From reads a saved battlefield incarnation. An absent snapshot has no subject.
func From(binding *PermanentBinding) PermanentRef { return savedPermanentRef{binding: binding} }
func (savedPermanentRef) permanentRef()           {}
func (r savedPermanentRef) resolvePermanent(ctx *EffectContext) (*Permanent, bool) {
	value, ok := ctx.Game.resolution.permanentBindings[r.binding]
	if !ok {
		return nil, false
	}
	p := ctx.Game.FindPermanent(value.id)
	if p == nil || p.PhasedOut || p.incarnationID != value.incarnation {
		return nil, false
	}
	return p, true
}

type attachedPermanentRef struct{}

// AttachedPermanent reads the attachment of the source incarnation.
// A departed source supplies its last attachment; a returned host is absent.
var AttachedPermanent = attachedPermanentRef{}

func (attachedPermanentRef) permanentRef() {}
func (attachedPermanentRef) resolvePermanent(ctx *EffectContext) (*Permanent, bool) {
	source, ok := This.resolvePermanent(ctx)
	var host capturedPermanent
	if ok {
		host = capturedPermanent{id: source.AttachedTo, incarnation: source.attachedIncarnation}
	} else if ctx.sourceCaptured {
		source = ctx.Game.FindPermanentIncludingPhased(ctx.SourceID)
		if source != nil && source.incarnationID == ctx.sourceIncarnation {
			host = capturedPermanent{id: source.AttachedTo, incarnation: source.attachedIncarnation}
		} else {
			host = ctx.Game.zones.departedAttachments[ctx.sourceIncarnation]
		}
	} else {
		return nil, false
	}
	p := ctx.Game.FindPermanent(host.id)
	if p == nil || (host.incarnation != uuid.Nil && p.incarnationID != host.incarnation) {
		return nil, false
	}
	return p, true
}

type permanentSnapshot struct {
	subject PermanentRef
	binding *PermanentBinding
}

// Snapshot records a subject before later instructions can move it.
// An absent subject records absence and replaces any earlier snapshot.
func Snapshot(subject PermanentRef, binding *PermanentBinding) ResolvingEffect {
	return &permanentSnapshot{subject: subject, binding: binding}
}
func (*permanentSnapshot) Text() string                 { return "save permanent" }
func (*permanentSnapshot) Properties() EffectProperties { return EffectProperties{} }
func (e *permanentSnapshot) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.permanentBindings == nil {
		ctx.Game.resolution.permanentBindings = make(map[*PermanentBinding]capturedPermanent)
	}
	p, ok := e.subject.resolvePermanent(ctx)
	delete(ctx.Game.resolution.permanentBindings, e.binding)
	if ok {
		ctx.Game.resolution.permanentBindings[e.binding] = capturedPermanent{p.ID(), p.incarnationID}
	}
	return nil
}

func (b *PermanentBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}
func (r savedPermanentRef) inspectDefinition() definitionNode {
	label := "saved permanent"
	if r.binding != nil {
		label = r.binding.name
	}
	return definitionNode{Role: "permanent-ref", Text: label, bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}
func (attachedPermanentRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-ref", Text: "attached permanent", battlefieldSource: true}
}
func (e *permanentSnapshot) inspectDefinition() definitionNode {
	return definitionNode{Role: "snapshot", bindingWrite: e.binding, children: []definitionEdge{{"subject", e.subject}, {"binding", e.binding}}}
}

func (b *PermanentBinding) bindingName() string { return b.name }

// --- from composable_card_binding.go ---
// CardBinding declares a card saved during one resolution.
type CardBinding struct{ name string }

// BindCard declares a card binding with a diagnostic name.
func BindCard(name string) *CardBinding { return &CardBinding{name: name} }

type capturedCard struct {
	id, visit uuid.UUID
	zone      Zone
	revealed  *revealedCardFacts
}
type savedCardRef struct{ binding *CardBinding }

// FromCard reads the saved card while its zone visit remains current.
func FromCard(binding *CardBinding) CardRef { return savedCardRef{binding: binding} }
func (savedCardRef) cardRef()               {}
func (r savedCardRef) resolveCard(ctx *EffectContext) (Card, bool) {
	value, ok := ctx.Game.resolution.cardBindings[r.binding]
	if !ok || value.visit == uuid.Nil {
		return nil, false
	}
	for _, player := range ctx.Game.players {
		if value.zone == ZoneLibrary {
			if visitor, ok := player.(interface{ libraryVisit(uuid.UUID) uuid.UUID }); ok && visitor.libraryVisit(value.id) == value.visit {
				for _, card := range player.Library() {
					if card.ID() == value.id {
						return card, true
					}
				}
			}
		}
		visitor, hasVisit := player.(interface{ handVisit(uuid.UUID) uuid.UUID })
		if value.zone == ZoneHand && hasVisit && visitor.handVisit(value.id) == value.visit {
			for _, card := range player.Hand() {
				if card.ID() == value.id {
					return card, true
				}
			}
		}
		if value.zone == ZoneGraveyard && ctx.Game.graveyardVisit(value.id) == value.visit {
			for _, card := range player.Graveyard() {
				if card.ID() == value.id {
					return card, true
				}
			}
		}
	}
	if value.zone == ZoneExile {
		if exiled := ctx.Game.FindExiledCard(value.id); exiled != nil && exiled.Visit == value.visit && exiled.VisibleTo(ctx.Controller) {
			return exiled.Card, true
		}
	}
	return nil, false
}

func (b *CardBinding) bindingName() string { return b.name }
func (b *CardBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "card-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}
func (r savedCardRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "card-ref", Text: "saved graveyard card", bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}

type eventCardCapture struct {
	binding *CardBinding
	value   capturedCard
}

func (*eventCardCapture) Text() string                 { return "remember the event card" }
func (*eventCardCapture) Properties() EffectProperties { return EffectProperties{} }
func (e *eventCardCapture) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.cardBindings == nil {
		ctx.Game.resolution.cardBindings = make(map[*CardBinding]capturedCard)
	}
	ctx.Game.resolution.cardBindings[e.binding] = e.value
	return nil
}
func (e *eventCardCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-card-capture", bindingWrite: e.binding, children: []definitionEdge{{"card", e.binding}}}
}

// --- from composable_cards_binding.go ---
// CardsBinding names a collection saved by one payment or instruction.
type CardsBinding struct{ name string }

// BindCards declares a card collection binding.
func BindCards(name string) *CardsBinding { return &CardsBinding{name: name} }

// CardSet is a sealed collection of cards in nonbattlefield zones.
type CardSet interface {
	cardSet()
	resolveCards(*EffectContext) []Card
}

type savedCardsSet struct{ binding *CardsBinding }

// FromCards reads the saved cards that remain in their captured zone visits.
func FromCards(binding *CardsBinding) CardSet { return savedCardsSet{binding: binding} }
func (savedCardsSet) cardSet()                {}
func (s savedCardsSet) resolveCards(ctx *EffectContext) []Card {
	var cards []Card
	for _, captured := range ctx.Game.resolution.cardsBindings[s.binding] {
		if captured.zone == ZoneHand {
			for _, player := range ctx.Game.players {
				visitor, ok := player.(interface{ handVisit(uuid.UUID) uuid.UUID })
				if !ok || visitor.handVisit(captured.id) != captured.visit {
					continue
				}
				for _, card := range player.Hand() {
					if card.ID() == captured.id {
						cards = append(cards, card)
						break
					}
				}
			}
			continue
		}
		if captured.zone == ZoneExile {
			if exiled := ctx.Game.FindExiledCard(captured.id); exiled != nil && exiled.Visit == captured.visit && exiled.VisibleTo(ctx.Controller) {
				cards = append(cards, exiled.Card)
			}
			continue
		}
		if captured.zone != ZoneGraveyard {
			continue
		}
		if ctx.Game.graveyardVisit(captured.id) != captured.visit {
			continue
		}
		for _, player := range ctx.Game.players {
			for _, card := range player.Graveyard() {
				if card.ID() == captured.id {
					cards = append(cards, card)
					break
				}
			}
		}
	}
	return cards
}
func (b *CardsBinding) bindingName() string { return b.name }
func (b *CardsBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "cards-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}
func (s savedCardsSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "card-set", Text: "saved cards", bindingRead: s.binding, children: []definitionEdge{{"binding", s.binding}}}
}

type cardCountValue struct{ subjects CardSet }

// CountCards counts cards still present in their captured zone visits.
func CountCards(subjects CardSet) ResolutionValue { return cardCountValue{subjects: subjects} }
func (cardCountValue) Text() string               { return "number of cards in collection" }
func (v cardCountValue) resolveValue(ctx *EffectContext) int {
	return len(v.subjects.resolveCards(ctx))
}
func (v cardCountValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "card-count", Text: v.Text(), children: []definitionEdge{{"subjects", v.subjects}}}
}

// --- from composable_damage_source_binding.go ---
// DamageSourceBinding names a source saved by one damage instruction.
type DamageSourceBinding struct{ name string }

// BindDamageSource declares a source binding for one resolution.
func BindDamageSource(name string) *DamageSourceBinding {
	return &DamageSourceBinding{name: name}
}

// FromDamageSource reads a saved source with its saved characteristics.
func FromDamageSource(binding *DamageSourceBinding) DamageSourceRef {
	return savedDamageSourceRef{binding: binding}
}

type savedDamageSourceRef struct{ binding *DamageSourceBinding }

func (savedDamageSourceRef) damageSourceRef() {}
func (r savedDamageSourceRef) resolveDamageSource(ctx *EffectContext) DamageSource {
	return ctx.Game.resolution.damageSourceBindings[r.binding]
}
func (b *DamageSourceBinding) bindingName() string { return b.name }
func (b *DamageSourceBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "damage-source-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}
func (r savedDamageSourceRef) inspectDefinition() definitionNode {
	label := "saved damage source"
	if r.binding != nil {
		label = r.binding.name
	}
	return definitionNode{Role: "damage-source-ref", Text: label, bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}

// --- from composable_number_binding.go ---
// NumberBinding declares a saved number. Each resolution has its own value.
type NumberBinding struct{ name string }

// BindNumber declares a number binding with a diagnostic name.
func BindNumber(name string) *NumberBinding { return &NumberBinding{name: name} }

// FromNumber reads a saved number. An absent binding supplies zero.
func FromNumber(binding *NumberBinding) ResolutionValue { return savedNumberValue{binding} }

type savedNumberValue struct{ binding *NumberBinding }

func (v savedNumberValue) Text() string {
	if v.binding == nil {
		return "saved number"
	}
	return v.binding.name
}
func (v savedNumberValue) resolveValue(ctx *EffectContext) int {
	return ctx.Game.resolution.numberBindings[v.binding]
}
func (v savedNumberValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), bindingRead: v.binding, children: []definitionEdge{{"binding", v.binding}}}
}
func (b *NumberBinding) bindingName() string { return b.name }
func (b *NumberBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "number-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}

type eventAmountCapture struct {
	binding *NumberBinding
	amount  int
}

func (*eventAmountCapture) Text() string                 { return "remember the event amount" }
func (*eventAmountCapture) Properties() EffectProperties { return EffectProperties{} }
func (e *eventAmountCapture) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.numberBindings == nil {
		ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
	}
	ctx.Game.resolution.numberBindings[e.binding] = e.amount
	return nil
}
func (e *eventAmountCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-amount-capture", bindingWrite: e.binding, children: []definitionEdge{{"amount", e.binding}}}
}

// --- from composable_player_binding.go ---
// PlayerBinding declares a saved player. Each resolution has its own value.
type PlayerBinding struct{ name string }

// BindPlayer declares a player binding with a diagnostic name.
func BindPlayer(name string) *PlayerBinding { return &PlayerBinding{name: name} }

// FromPlayer reads a saved player. An absent binding has no subject.
func FromPlayer(binding *PlayerBinding) PlayerRef { return savedPlayerRef{binding} }

type savedPlayerRef struct{ binding *PlayerBinding }

func (savedPlayerRef) playerRef()       {}
func (savedPlayerRef) damageRecipient() {}
func (r savedPlayerRef) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	id, ok := ctx.Game.resolution.playerBindings[r.binding]
	return id, ok && ctx.Game.GetPlayer(id) != nil
}
func (r savedPlayerRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (b *PlayerBinding) bindingName() string { return b.name }
func (b *PlayerBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "player-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}
func (r savedPlayerRef) inspectDefinition() definitionNode {
	label := "saved player"
	if r.binding != nil {
		label = r.binding.name
	}
	return definitionNode{Role: "player-ref", Text: label, bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}

// ChooseRandomPlayer saves one random member at resolution without targeting.
// An empty set clears the binding and does not use a random value.
func ChooseRandomPlayer(subjects PlayerSet, binding *PlayerBinding) ResolvingEffect {
	return &randomPlayerChoice{subjects, binding}
}

type randomPlayerChoice struct {
	subjects PlayerSet
	binding  *PlayerBinding
}

func (*randomPlayerChoice) Text() string                 { return "choose a random player" }
func (*randomPlayerChoice) Properties() EffectProperties { return EffectProperties{} }
func (e *randomPlayerChoice) Apply(ctx *EffectContext) error {
	candidates := e.subjects.resolvePlayers(ctx)
	if ctx.Game.resolution.playerBindings == nil {
		ctx.Game.resolution.playerBindings = make(map[*PlayerBinding]uuid.UUID)
	}
	delete(ctx.Game.resolution.playerBindings, e.binding)
	if len(candidates) == 0 {
		return nil
	}
	ctx.Game.resolution.playerBindings[e.binding] = candidates[ctx.Game.RandIntn(len(candidates))]
	return nil
}
func (e *randomPlayerChoice) inspectDefinition() definitionNode {
	return definitionNode{Role: "random-player-choice", bindingWrite: e.binding, children: []definitionEdge{{"subjects", e.subjects}, {"binding", e.binding}}}
}

// --- from composable_token_set.go ---
// CreatedTokensBinding names one token-entry result set.
type CreatedTokensBinding struct{ name string }

// BindCreatedTokens declares one instance-local token result.
func BindCreatedTokens(name string) *CreatedTokensBinding { return &CreatedTokensBinding{name: name} }

func (b *CreatedTokensBinding) bindingName() string { return b.name }
func (b *CreatedTokensBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "created-tokens-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}

type createdTokensSet struct{ binding *CreatedTokensBinding }

// FromCreatedTokens selects surviving, unphased incarnations from one entry batch.
func FromCreatedTokens(binding *CreatedTokensBinding) PermanentSet {
	return createdTokensSet{binding: binding}
}
func (createdTokensSet) permanentSet() {}
func (s createdTokensSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "created-tokens-set", bindingRead: s.binding, children: []definitionEdge{{"binding", s.binding}}}
}
func (s createdTokensSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	values := ctx.Game.resolution.createdTokenBindings[s.binding]
	result := make([]*Permanent, 0, len(values))
	for _, value := range values {
		p := ctx.Game.FindPermanent(value.id)
		if p != nil && !p.PhasedOut && p.incarnationID == value.incarnation {
			result = append(result, p)
		}
	}
	return result
}

func cloneCreatedTokenBindings(bindings map[*CreatedTokensBinding][]capturedPermanent) map[*CreatedTokensBinding][]capturedPermanent {
	if bindings == nil {
		return nil
	}
	cloned := make(map[*CreatedTokensBinding][]capturedPermanent, len(bindings))
	for binding, values := range bindings {
		cloned[binding] = slices.Clone(values)
	}
	return cloned
}

// --- from composable_subject.go ---
// PermanentRef identifies one battlefield incarnation.
type PermanentRef interface {
	DamageRecipientRef
	permanentRef()
	resolvePermanent(*EffectContext) (*Permanent, bool)
}
type sourceRef struct{}

func (sourceRef) permanentRef() {}

// This refers to the source in the enclosing resolution context.
var This = sourceRef{}

func (sourceRef) resolvePermanent(ctx *EffectContext) (*Permanent, bool) {
	p := ctx.Game.FindPermanent(ctx.SourceID)
	if p == nil || p.PhasedOut {
		return nil, false
	}
	if ctx.sourceCaptured && p.incarnationID != ctx.sourceIncarnation {
		return nil, false
	}
	return p, true
}

type sacrificeEffect struct {
	subject       PermanentRef
	result        *NumberBinding
	subtypeResult *NumberBinding
	subtype       string
	invalidOption bool
	optionCount   int
}

// SacrificeResultOption selects an output for one sacrifice instruction.
type SacrificeResultOption interface {
	sacrificeResultOption()
	applySacrificeResult(*sacrificeEffect)
}

type sacrificeAmountOption struct{ binding *NumberBinding }

// SacrificeAmountInto saves one if the permanent was sacrificed, or zero.
func SacrificeAmountInto(binding *NumberBinding) SacrificeResultOption {
	return sacrificeAmountOption{binding}
}

func (sacrificeAmountOption) sacrificeResultOption() {}
func (o sacrificeAmountOption) applySacrificeResult(e *sacrificeEffect) {
	e.result = o.binding
	e.optionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

type sacrificeSubtypeOption struct {
	subtype string
	binding *NumberBinding
}

// SacrificeSubtypeInto saves one if the sacrificed permanent had the subtype.
func SacrificeSubtypeInto(subtype string, binding *NumberBinding) SacrificeResultOption {
	return sacrificeSubtypeOption{subtype: subtype, binding: binding}
}

func (sacrificeSubtypeOption) sacrificeResultOption() {}
func (o sacrificeSubtypeOption) applySacrificeResult(e *sacrificeEffect) {
	e.subtype = o.subtype
	e.subtypeResult = o.binding
	e.optionCount++
	if o.binding == nil || o.subtype == "" {
		e.invalidOption = true
	}
}

// Sacrifice instructs the stored controller to sacrifice the subject.
// An absent subject or a subject controlled by another player has no effect.
func Sacrifice(subject PermanentRef, options ...SacrificeResultOption) ResolvingEffect {
	e := &sacrificeEffect{subject: subject}
	for _, option := range options {
		if nilDefinition(option) {
			e.invalidOption = true
			continue
		}
		option.applySacrificeResult(e)
	}
	return e
}
func (e *sacrificeEffect) Text() string {
	if _, ok := e.subject.(sourceRef); ok {
		return "sacrifice this permanent"
	}
	if subject, ok := e.subject.(definitionInspector); ok {
		return "sacrifice " + subject.inspectDefinition().Text
	}
	return "sacrifice permanent"
}
func (e *sacrificeEffect) Properties() EffectProperties {
	_, source := e.subject.(sourceRef)
	return EffectProperties{SacrificesSource: source}
}
func (e *sacrificeEffect) Apply(ctx *EffectContext) error {
	if e.result != nil || e.subtypeResult != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		if e.result != nil {
			ctx.Game.resolution.numberBindings[e.result] = 0
		}
		if e.subtypeResult != nil {
			ctx.Game.resolution.numberBindings[e.subtypeResult] = 0
		}
	}
	if p, ok := e.subject.resolvePermanent(ctx); ok && p.ControllerID() == ctx.Controller {
		action := ctx.Game.prepareSacrifice(p)
		if action == nil {
			return nil
		}
		current := ctx.Game.FindPermanent(action.permanentID)
		matched := current != nil && current.incarnationID == action.incarnation && e.subtypeResult != nil && current.HasSubType(e.subtype)
		if !ctx.Game.performSacrifice(action) {
			return nil
		}
		if e.result != nil {
			ctx.Game.resolution.numberBindings[e.result] = 1
		}
		if matched {
			ctx.Game.resolution.numberBindings[e.subtypeResult] = 1
		}
	}
	return nil
}

// --- from composable_value.go ---
// ResolutionValue reads a number from the current resolution context.
// It cannot be used by legacy instructions that discard binding context.
type ResolutionValue interface {
	Text() string
	resolveValue(*EffectContext) int
}

func (v fixedValue) resolveValue(_ *EffectContext) int { return v.n }

// ValueAtResolution adapts a legacy value without adding typed binding reads.
func ValueAtResolution(value ValueSource) ResolutionValue { return legacyResolutionValue{value} }

type legacyResolutionValue struct{ value ValueSource }

func (v legacyResolutionValue) Text() string {
	if nilDefinition(v.value) {
		return ""
	}
	return v.value.Text()
}
func (v legacyResolutionValue) resolveValue(ctx *EffectContext) int {
	return v.value.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
}
func (v legacyResolutionValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", children: []definitionEdge{{"value", v.value}}}
}

type permanentResolutionValue struct {
	subject   PermanentRef
	toughness bool
}

type selfCastColorsValue struct{}

// ColorsSpentToCastThisSpell reads the resolving spell's saved mana payment.
func ColorsSpentToCastThisSpell() ResolutionValue { return selfCastColorsValue{} }

func (selfCastColorsValue) Text() string { return "colors of mana spent to cast this spell" }
func (selfCastColorsValue) resolveValue(ctx *EffectContext) int {
	return ctx.Game.ResolvingCastContext().DistinctColorsSpent()
}
func (v selfCastColorsValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text()}
}

// PowerOf reads current power. An absent permanent supplies zero.
func PowerOf(subject PermanentRef) ResolutionValue { return permanentResolutionValue{subject: subject} }

// ToughnessOf reads current toughness. An absent permanent supplies zero.
func ToughnessOf(subject PermanentRef) ResolutionValue {
	return permanentResolutionValue{subject: subject, toughness: true}
}
func (v permanentResolutionValue) Text() string {
	if v.toughness {
		return "toughness of subject"
	}
	return "power of subject"
}
func (v permanentResolutionValue) resolveValue(ctx *EffectContext) int {
	p, ok := v.subject.resolvePermanent(ctx)
	if !ok {
		return 0
	}
	if v.toughness {
		return p.CurrentToughness(ctx.Game)
	}
	return p.CurrentPower(ctx.Game)
}
func (v permanentResolutionValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-value", Text: v.Text(), children: []definitionEdge{{"subject", v.subject}}}
}

// ManaValueOf reads the permanent's mana value. An absent permanent supplies zero.
func ManaValueOf(subject PermanentRef) ResolutionValue { return permanentManaValue{subject} }

type permanentManaValue struct{ subject PermanentRef }

type playerHandSizeValue struct{ subject PlayerRef }

type playerLifeTotalValue struct{ subject PlayerRef }

type playerLifeGainedThisTurnValue struct{ subject PlayerRef }

// LifeTotalOf reads the selected player's current life total at resolution.
func LifeTotalOf(subject PlayerRef) ResolutionValue { return playerLifeTotalValue{subject} }

func (v playerLifeTotalValue) Text() string { return "life total of player" }
func (v playerLifeTotalValue) resolveValue(ctx *EffectContext) int {
	id, ok := v.subject.resolvePlayer(ctx)
	if !ok {
		return 0
	}
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return 0
	}
	return player.Life()
}
func (v playerLifeTotalValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"subject", v.subject}}}
}

// LifeGainedThisTurn reads the selected player's actual life gain this turn.
func LifeGainedThisTurn(subject PlayerRef) ResolutionValue {
	return playerLifeGainedThisTurnValue{subject: subject}
}

func (v playerLifeGainedThisTurnValue) Text() string { return "life gained by player this turn" }
func (v playerLifeGainedThisTurnValue) resolveValue(ctx *EffectContext) int {
	id, ok := v.subject.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(id) == nil {
		return 0
	}
	return ctx.Game.PlayerLifeGainedThisTurn(id)
}
func (v playerLifeGainedThisTurnValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"subject", v.subject}}}
}

// HandSizeOf reads the selected player's current hand size at resolution.
func HandSizeOf(subject PlayerRef) ResolutionValue { return playerHandSizeValue{subject} }

func (v playerHandSizeValue) Text() string { return "hand size of player" }
func (v playerHandSizeValue) resolveValue(ctx *EffectContext) int {
	id, ok := v.subject.resolvePlayer(ctx)
	if !ok {
		return 0
	}
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return 0
	}
	return len(player.Hand())
}
func (v playerHandSizeValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), children: []definitionEdge{{"subject", v.subject}}}
}

func (v permanentManaValue) Text() string { return "mana value of subject" }
func (v permanentManaValue) resolveValue(ctx *EffectContext) int {
	p, ok := v.subject.resolvePermanent(ctx)
	if !ok {
		return 0
	}
	return p.Card.ManaCost().CMC()
}
func (v permanentManaValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-value", Text: v.Text(), children: []definitionEdge{{"subject", v.subject}}}
}

// --- from composable_arithmetic.go ---
type arithmeticOperation uint8

const (
	valueSubtract arithmeticOperation = iota
	valueMin
	valueMax
	valueMultiply
	valueDivideDown
	valueDivideUp
	valueAdd
)

type arithmeticValue struct {
	left, right ResolutionValue
	operation   arithmeticOperation
}

// AddValues supplies the sum of two values.
func AddValues(left, right ResolutionValue) ResolutionValue {
	return arithmeticValue{left, right, valueAdd}
}

// SubtractValues subtracts the right value from the left value.
func SubtractValues(left, right ResolutionValue) ResolutionValue {
	return arithmeticValue{left, right, valueSubtract}
}

// MultiplyValues supplies the product of two values.
func MultiplyValues(left, right ResolutionValue) ResolutionValue {
	return arithmeticValue{left, right, valueMultiply}
}

// MinValue supplies the smaller of two values.
func MinValue(left, right ResolutionValue) ResolutionValue {
	return arithmeticValue{left, right, valueMin}
}

// MaxValue supplies the larger of two values.
func MaxValue(left, right ResolutionValue) ResolutionValue {
	return arithmeticValue{left, right, valueMax}
}

// DivideRoundDown divides two values and rounds the result down.
func DivideRoundDown(numerator, denominator ResolutionValue) ResolutionValue {
	return arithmeticValue{numerator, denominator, valueDivideDown}
}

// DivideRoundUp divides two values and rounds the result up.
func DivideRoundUp(numerator, denominator ResolutionValue) ResolutionValue {
	return arithmeticValue{numerator, denominator, valueDivideUp}
}

func (v arithmeticValue) operationName() string {
	switch v.operation {
	case valueAdd:
		return "add"
	case valueMultiply:
		return "multiply"
	case valueSubtract:
		return "subtract"
	case valueMin:
		return "minimum"
	case valueMax:
		return "maximum"
	case valueDivideDown:
		return "divide rounded down"
	case valueDivideUp:
		return "divide rounded up"
	default:
		return "invalid arithmetic"
	}
}

func (v arithmeticValue) Text() string {
	left, right := "?", "?"
	if !nilDefinition(v.left) {
		left = v.left.Text()
	}
	if !nilDefinition(v.right) {
		right = v.right.Text()
	}
	return v.operationName() + "(" + left + ", " + right + ")"
}

func (v arithmeticValue) resolveValue(ctx *EffectContext) int {
	left, right := v.left.resolveValue(ctx), v.right.resolveValue(ctx)
	switch v.operation {
	case valueAdd:
		return left + right
	case valueMultiply:
		return left * right
	case valueSubtract:
		return left - right
	case valueMin:
		return min(left, right)
	case valueMax:
		return max(left, right)
	case valueDivideDown, valueDivideUp:
		if right == 0 {
			return 0
		}
		quotient, remainder := left/right, left%right
		if remainder == 0 {
			return quotient
		}
		if v.operation == valueDivideDown && (left < 0) != (right < 0) {
			return quotient - 1
		}
		if v.operation == valueDivideUp && (left < 0) == (right < 0) {
			return quotient + 1
		}
		return quotient
	default:
		return 0
	}
}

func (v arithmeticValue) inspectDefinition() definitionNode {
	n := definitionNode{
		Role: "arithmetic-value", Text: v.operationName(),
		children: []definitionEdge{{"left", v.left}, {"right", v.right}},
	}
	if (v.operation == valueDivideDown || v.operation == valueDivideUp) && !nilDefinition(v.right) {
		if fixed, ok := v.right.(fixedValue); ok && fixed.n == 0 {
			n.problem = "fixed divisor must not be zero"
		}
	}
	return n
}

// --- from composable_attachments_set.go ---
// AttachedAuras selects Auras attached to the current permanent incarnation.
func AttachedAuras(host PermanentRef) PermanentSet {
	return attachedAurasSet{host: host}
}

type attachedAurasSet struct{ host PermanentRef }

func (attachedAurasSet) permanentSet() {}

func (s attachedAurasSet) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "Auras attached to permanent"}
	if nilDefinition(s.host) {
		n.problem = "attached Auras require a permanent host"
	} else {
		n.children = []definitionEdge{{"host", s.host}}
	}
	return n
}

func (s attachedAurasSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.host) {
		return nil
	}
	host, ok := s.host.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	result := make([]*Permanent, 0, len(host.Attachments))
	for _, id := range host.Attachments {
		aura := ctx.Game.FindPermanent(id)
		if aura == nil || aura.PhasedOut || aura.AttachedTo != host.ID() || aura.attachedIncarnation != host.incarnationID {
			continue
		}
		if aura.HasSubType("Aura") && aura.HasType(TypeEnchantment) {
			result = append(result, aura)
		}
	}
	return result
}

// --- from composable_controlled_by_set.go ---
type controlledByPermanentSet struct {
	controller PlayerRef
	filter     PermanentFilter
}

// PermanentsControlledBy selects current battlefield permanents controlled by one player.
func PermanentsControlledBy(controller PlayerRef, filter PermanentFilter) PermanentSet {
	return controlledByPermanentSet{controller: controller, filter: filter}
}

func (controlledByPermanentSet) permanentSet() {}

func (s controlledByPermanentSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.controller) {
		return nil
	}
	controller, ok := s.controller.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	var result []*Permanent
	for _, permanent := range ctx.Game.FilterBattlefield(s.filter) {
		if permanent.ControllerID() == controller {
			result = append(result, permanent)
		}
	}
	return result
}

func (s controlledByPermanentSet) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "permanents controlled by player"}
	if nilDefinition(s.controller) {
		n.problem = "controlled permanent set requires a player"
	} else {
		n.children = []definitionEdge{{"controller", s.controller}}
	}
	return n
}

// --- from composable_damage_history_set.go ---
type damageHistoryPlayers struct {
	source     DamageSourceRef
	candidates PlayerSet
}

// PlayersDealtDamageBy selects candidates damaged by one permanent incarnation this game.
func PlayersDealtDamageBy(source DamageSourceRef, candidates PlayerSet) PlayerSet {
	return damageHistoryPlayers{source: source, candidates: candidates}
}

func (damageHistoryPlayers) playerSet() {}

func (s damageHistoryPlayers) resolvePlayers(ctx *EffectContext) []uuid.UUID {
	if nilDefinition(s.source) || nilDefinition(s.candidates) {
		return nil
	}
	source := s.source.resolveDamageSource(ctx)
	if source.Incarnation() == uuid.Nil {
		return nil
	}
	var selected []uuid.UUID
	for _, id := range s.candidates.resolvePlayers(ctx) {
		if ctx.Game.HasDealtDamageToPlayerFromSource(source, id) {
			selected = append(selected, id)
		}
	}
	return selected
}

func (s damageHistoryPlayers) inspectDefinition() definitionNode {
	n := definitionNode{Role: "player-set", Text: "players dealt damage by source this game"}
	if nilDefinition(s.source) || nilDefinition(s.candidates) {
		n.problem = "supply a damage source and player set"
		return n
	}
	n.children = []definitionEdge{{"source", damageSourceDefinition{s.source}}, {"candidates", s.candidates}}
	return n
}

type damageHistoryPermanents struct {
	source     DamageSourceRef
	candidates PermanentSet
}

// PermanentsDealtDamageBy selects candidates damaged by one permanent incarnation this game.
func PermanentsDealtDamageBy(source DamageSourceRef, candidates PermanentSet) PermanentSet {
	return damageHistoryPermanents{source: source, candidates: candidates}
}

func (damageHistoryPermanents) permanentSet() {}

func (s damageHistoryPermanents) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.source) || nilDefinition(s.candidates) {
		return nil
	}
	source := s.source.resolveDamageSource(ctx)
	if source.Incarnation() == uuid.Nil {
		return nil
	}
	var selected []*Permanent
	for _, permanent := range s.candidates.resolvePermanents(ctx) {
		if permanent != nil && !permanent.PhasedOut && ctx.Game.HasDealtDamageToPermanentFromSource(source, permanent) {
			selected = append(selected, permanent)
		}
	}
	return selected
}

func (s damageHistoryPermanents) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "permanents dealt damage by source this game"}
	if nilDefinition(s.source) || nilDefinition(s.candidates) {
		n.problem = "supply a damage source and permanent set"
		return n
	}
	n.children = []definitionEdge{{"source", damageSourceDefinition{s.source}}, {"candidates", s.candidates}}
	return n
}

type turnDamageHistoryPermanents struct {
	creaturesOnly bool
	source        DamageSourceRef
	candidates    PermanentSet
}

// PermanentsDealtDamageByThisTurn selects current recipients of this source's turn damage.
func PermanentsDealtDamageByThisTurn(source DamageSourceRef, candidates PermanentSet) PermanentSet {
	return turnDamageHistoryPermanents{source: source, candidates: candidates}
}

func (turnDamageHistoryPermanents) permanentSet() {}

func (s turnDamageHistoryPermanents) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.source) || nilDefinition(s.candidates) {
		return nil
	}
	source := s.source.resolveDamageSource(ctx)
	if source.Incarnation() == uuid.Nil {
		return nil
	}
	var selected []*Permanent
	for _, permanent := range s.candidates.resolvePermanents(ctx) {
		if permanent != nil && !permanent.PhasedOut && ctx.Game.damage.dealtDamageToPermanentThisTurn(source, capturedPermanent{permanent.ID(), permanent.incarnationID}) {
			if s.creaturesOnly && !ctx.Game.damage.turnDamageToCreatures[capturedPermanent{source.ID(), source.Incarnation()}][capturedPermanent{permanent.ID(), permanent.incarnationID}] {
				continue
			}
			selected = append(selected, permanent)
		}
	}
	return selected
}

func (s turnDamageHistoryPermanents) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "permanents dealt damage by source this turn"}
	if nilDefinition(s.source) || nilDefinition(s.candidates) {
		n.problem = "supply a damage source and permanent set"
		return n
	}
	n.children = []definitionEdge{{"source", damageSourceDefinition{s.source}}, {"candidates", s.candidates}}
	return n
}

type reverseTurnDamageHistoryPermanents struct {
	recipient  PermanentRef
	candidates PermanentSet
}

// PermanentsThatDealtDamageToThisTurn selects current sources of damage to one permanent.
func PermanentsThatDealtDamageToThisTurn(recipient PermanentRef, candidates PermanentSet) PermanentSet {
	return reverseTurnDamageHistoryPermanents{recipient: recipient, candidates: candidates}
}

func (reverseTurnDamageHistoryPermanents) permanentSet() {}

func (s reverseTurnDamageHistoryPermanents) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.recipient) || nilDefinition(s.candidates) {
		return nil
	}
	var recipient capturedPermanent
	if _, ok := s.recipient.(sourceRef); ok && ctx.sourceCaptured {
		recipient = capturedPermanent{ctx.SourceID, ctx.sourceIncarnation}
	} else if p, ok := s.recipient.resolvePermanent(ctx); ok {
		recipient = capturedPermanent{p.ID(), p.incarnationID}
	}
	if recipient.id == uuid.Nil || recipient.incarnation == uuid.Nil {
		return nil
	}
	var selected []*Permanent
	for _, candidate := range s.candidates.resolvePermanents(ctx) {
		if candidate == nil || candidate.PhasedOut {
			continue
		}
		if ctx.Game.damage.permanentDealtDamageToThisTurn(capturedPermanent{candidate.ID(), candidate.incarnationID}, recipient) {
			selected = append(selected, candidate)
		}
	}
	return selected
}

func (s reverseTurnDamageHistoryPermanents) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "permanents that dealt damage to recipient this turn"}
	if nilDefinition(s.recipient) || nilDefinition(s.candidates) {
		n.problem = "supply a damage recipient and permanent set"
		return n
	}
	n.children = []definitionEdge{{"recipient", s.recipient}, {"candidates", s.candidates}}
	return n
}

// --- from composable_death_count.go ---
type creatureDeathCountValue struct{}

// CreaturesDiedThisTurn reads the current turn's creature death count.
// It includes tokens and creatures controlled by any player.
func CreaturesDiedThisTurn() ResolutionValue { return creatureDeathCountValue{} }

func (creatureDeathCountValue) Text() string { return "creatures that died this turn" }

func (creatureDeathCountValue) resolveValue(ctx *EffectContext) int {
	return ctx.Game.CreatureDeaths()
}

func (v creatureDeathCountValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "creature-death-count", Text: v.Text()}
}

// --- from composable_highest_life_player.go ---
type highestLifePlayerRef struct{}

// PlayerWithUniqueHighestLife selects the sole player with the most life.
// A tie has no player to select.
func PlayerWithUniqueHighestLife() PlayerRef { return highestLifePlayerRef{} }

func (highestLifePlayerRef) playerRef()       {}
func (highestLifePlayerRef) damageRecipient() {}
func (r highestLifePlayerRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (highestLifePlayerRef) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	players := ctx.Game.AllPlayers()
	if len(players) < 2 {
		return uuid.Nil, false
	}
	best := players[0]
	tied := false
	for _, player := range players[1:] {
		switch {
		case player.Life() > best.Life():
			best = player
			tied = false
		case player.Life() == best.Life():
			tied = true
		}
	}
	if tied {
		return uuid.Nil, false
	}
	return best.PlayerID(), true
}
func (highestLifePlayerRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-ref", Text: "player with the most life"}
}

// --- from composable_mana_value_set.go ---
type manaValueAtMostSet struct {
	subjects PermanentSet
	limit    ResolutionValue
}

// ManaValueAtMost keeps set members whose mana value is at most the resolution value.
func ManaValueAtMost(subjects PermanentSet, limit ResolutionValue) PermanentSet {
	return manaValueAtMostSet{subjects: subjects, limit: limit}
}

func (manaValueAtMostSet) permanentSet() {}

func (s manaValueAtMostSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	maximum := s.limit.resolveValue(ctx)
	var result []*Permanent
	for _, p := range s.subjects.resolvePermanents(ctx) {
		if p.Card.ManaCost().CMC() <= maximum {
			result = append(result, p)
		}
	}
	return result
}

func (s manaValueAtMostSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-set", Text: "mana value at most", children: []definitionEdge{{"subjects", s.subjects}, {"limit", s.limit}}}
}

// --- from composable_number_snapshot.go ---
type numberSnapshot struct {
	value   ResolutionValue
	binding *NumberBinding
}

// SnapshotNumber saves a value when this instruction resolves.
// Later instructions can replace the value in the same binding.
func SnapshotNumber(value ResolutionValue, binding *NumberBinding) ResolvingEffect {
	return &numberSnapshot{value: value, binding: binding}
}

func (*numberSnapshot) Text() string                 { return "save number" }
func (*numberSnapshot) Properties() EffectProperties { return EffectProperties{} }
func (e *numberSnapshot) Apply(ctx *EffectContext) error {
	amount := e.value.resolveValue(ctx)
	if ctx.Game.resolution.numberBindings == nil {
		ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
	}
	ctx.Game.resolution.numberBindings[e.binding] = amount
	return nil
}
func (e *numberSnapshot) inspectDefinition() definitionNode {
	return definitionNode{Role: "number-snapshot", bindingWrite: e.binding, children: []definitionEdge{{"value", e.value}, {"binding", e.binding}}}
}

// --- from composable_owned_set.go ---
type ownedPermanentSet struct {
	owner  PlayerRef
	filter PermanentFilter
}

// PermanentsOwnedBy selects battlefield permanents owned by one player.
func PermanentsOwnedBy(owner PlayerRef, filter PermanentFilter) PermanentSet {
	return ownedPermanentSet{owner: owner, filter: filter}
}

func (ownedPermanentSet) permanentSet() {}

func (s ownedPermanentSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.owner) {
		return nil
	}
	owner, ok := s.owner.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	var result []*Permanent
	for _, permanent := range ctx.Game.FilterBattlefield(s.filter) {
		if permanent.Card.Owner() == owner {
			result = append(result, permanent)
		}
	}
	return result
}

func (s ownedPermanentSet) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "permanents owned by player"}
	if nilDefinition(s.owner) {
		n.problem = "owned permanent set requires a player"
	} else {
		n.children = []definitionEdge{{"owner", s.owner}}
	}
	return n
}

// --- from composable_permanent_count.go ---
type permanentCountValue struct {
	subjects PermanentSet
}

// CountPermanents reads collection membership when the value is used.
func CountPermanents(subjects PermanentSet) ResolutionValue {
	return permanentCountValue{subjects: subjects}
}

func (permanentCountValue) Text() string { return "number of permanents in collection" }

func (v permanentCountValue) resolveValue(ctx *EffectContext) int {
	return len(v.subjects.resolvePermanents(ctx))
}

func (v permanentCountValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-count", Text: v.Text(), children: []definitionEdge{{"subjects", v.subjects}}}
}

// --- from composable_player_snapshot.go ---
type playerSnapshot struct {
	subject PlayerRef
	binding *PlayerBinding
}

// SnapshotPlayer saves a player when this instruction resolves.
// An absent subject clears the previous value of the binding.
func SnapshotPlayer(subject PlayerRef, binding *PlayerBinding) ResolvingEffect {
	return &playerSnapshot{subject: subject, binding: binding}
}

func (*playerSnapshot) Text() string                 { return "save player" }
func (*playerSnapshot) Properties() EffectProperties { return EffectProperties{} }
func (e *playerSnapshot) Apply(ctx *EffectContext) error {
	id, ok := e.subject.resolvePlayer(ctx)
	if ctx.Game.resolution.playerBindings == nil {
		ctx.Game.resolution.playerBindings = make(map[*PlayerBinding]uuid.UUID)
	}
	delete(ctx.Game.resolution.playerBindings, e.binding)
	if ok {
		ctx.Game.resolution.playerBindings[e.binding] = id
	}
	return nil
}
func (e *playerSnapshot) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-snapshot", bindingWrite: e.binding, children: []definitionEdge{{"subject", e.subject}, {"binding", e.binding}}}
}

// --- from composable_choice.go ---
type lowestPowerSet struct{ subjects PermanentSet }

// LowestPower keeps all members tied for the least current power.
func LowestPower(subjects PermanentSet) PermanentSet { return lowestPowerSet{subjects} }
func (lowestPowerSet) permanentSet()                 {}
func (s lowestPowerSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	members := s.subjects.resolvePermanents(ctx)
	if len(members) == 0 {
		return nil
	}
	least := members[0].CurrentPower(ctx.Game)
	for _, p := range members[1:] {
		least = min(least, p.CurrentPower(ctx.Game))
	}
	result := make([]*Permanent, 0, len(members))
	for _, p := range members {
		if p.CurrentPower(ctx.Game) == least {
			result = append(result, p)
		}
	}
	return result
}
func (s lowestPowerSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "least-power-set", children: []definitionEdge{{"subjects", s.subjects}}}
}

type permanentChoice struct {
	player   PlayerRef
	subjects PermanentSet
	binding  *PermanentBinding
}

// ChoosePermanent makes a required non-target choice at resolution.
// No candidates or an absent player records absence in the binding.
func ChoosePermanent(player PlayerRef, subjects PermanentSet, binding *PermanentBinding) ResolvingEffect {
	return &permanentChoice{player, subjects, binding}
}
func (e *permanentChoice) Text() string                 { return "choose a permanent" }
func (e *permanentChoice) Properties() EffectProperties { return EffectProperties{} }
func (e *permanentChoice) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.permanentBindings == nil {
		ctx.Game.resolution.permanentBindings = make(map[*PermanentBinding]capturedPermanent)
	}
	delete(ctx.Game.resolution.permanentBindings, e.binding)
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	candidates := e.subjects.resolvePermanents(ctx)
	if len(candidates) == 0 {
		return nil
	}
	selected := candidates[0]
	if len(candidates) > 1 {
		chosen := ctx.Game.GetPlayer(id).ChoosePermanent(candidates, "Choose "+e.binding.name, ctx.Game)
		if chosen != nil {
			for _, p := range candidates {
				if p.ID() == chosen.ID() && p.incarnationID == chosen.incarnationID {
					selected = p
					break
				}
			}
		}
	}
	ctx.Game.resolution.permanentBindings[e.binding] = capturedPermanent{selected.ID(), selected.incarnationID}
	return nil
}
func (e *permanentChoice) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-choice", bindingWrite: e.binding, children: []definitionEdge{{"player", e.player}, {"subjects", e.subjects}, {"binding", e.binding}}}
}

// ChooseRandomPermanent saves one random member at resolution.
// An empty set clears the binding and does not use a random value.
func ChooseRandomPermanent(subjects PermanentSet, binding *PermanentBinding) ResolvingEffect {
	return &randomPermanentChoice{subjects, binding}
}

type randomPermanentChoice struct {
	subjects PermanentSet
	binding  *PermanentBinding
}

func (*randomPermanentChoice) Text() string                 { return "choose a random permanent" }
func (*randomPermanentChoice) Properties() EffectProperties { return EffectProperties{} }
func (e *randomPermanentChoice) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.permanentBindings == nil {
		ctx.Game.resolution.permanentBindings = make(map[*PermanentBinding]capturedPermanent)
	}
	delete(ctx.Game.resolution.permanentBindings, e.binding)
	candidates := e.subjects.resolvePermanents(ctx)
	if len(candidates) == 0 {
		return nil
	}
	selected := candidates[ctx.Game.RandIntn(len(candidates))]
	ctx.Game.resolution.permanentBindings[e.binding] = capturedPermanent{selected.ID(), selected.incarnationID}
	return nil
}
func (e *randomPermanentChoice) inspectDefinition() definitionNode {
	return definitionNode{Role: "random-permanent-choice", bindingWrite: e.binding, children: []definitionEdge{{"subjects", e.subjects}, {"binding", e.binding}}}
}

// --- from composable_mode.go ---
// ModeDefinition is an immutable mode body and its target declarations.
type ModeDefinition struct {
	mode    Mode
	grouped bool
}

// ModeOption is a closed option for a mode body.
type ModeOption interface {
	modeOption()
	applyMode(*Mode)
}

func (targetOptions) modeOption() {}
func (o targetOptions) applyMode(m *Mode) {
	for _, b := range o.bindings {
		if nilDefinition(b) {
			panic("mode targets: supply a non-nil binding")
		}
		m.Targets = append(m.Targets, b.targetSpec())
	}
}

// ModeOf declares one mode. Targets belong only to that mode.
func ModeOf(label string, effect ResolvingEffect, options ...ModeOption) ModeDefinition {
	m := Mode{Label: label, Effects: []Effect{effect}}
	for _, o := range options {
		if nilDefinition(o) {
			panic("mode: supply a non-nil option")
		}
		o.applyMode(&m)
	}
	return ModeDefinition{mode: m}
}
func modeValues(modes []ModeDefinition) []Mode {
	if len(modes) < 2 {
		panic("modes: supply at least two mode bodies")
	}
	result := make([]Mode, len(modes))
	for i, m := range modes {
		result[i] = m.mode
	}
	return result
}

// SpellModes creates a spell that chooses one mode through the existing casting path.
func SpellModes(modes ...ModeDefinition) *ModalSpellAbility { return NewModalSpell(modeValues(modes)) }

// ModalSpellOption keeps announcement rules outside mode resolution bodies.
type ModalSpellOption interface {
	modalSpellOption()
	applyModalSpell(*ModalSpellAbility)
}

// SpellModesWithOptions keeps spell costs separate from the selected mode.
func SpellModesWithOptions(options []ModalSpellOption, modes ...ModeDefinition) *ModalSpellAbility {
	m := SpellModes(modes...)
	m.dslOptions = append([]ModalSpellOption(nil), options...)
	for _, option := range m.dslOptions {
		if !nilDefinition(option) {
			option.applyModalSpell(m)
		}
	}
	return m
}

// ModalActivationOption is a mode or a restriction for a modal activation.
type ModalActivationOption interface {
	modalActivationOption()
	applyModalActivation(*ActionDefinition)
}

func (ModeDefinition) modalActivationOption() {}
func (m ModeDefinition) applyModalActivation(a *ActionDefinition) {
	a.modes = append(a.modes, m.mode)
}

// ActivateModes creates a modal activation with the existing payment transaction.
func ActivateModes(costs CostGroup, options ...ModalActivationOption) *ActionDefinition {
	a := Activate(costs, nil)
	a.effects = nil
	for _, option := range options {
		if nilDefinition(option) {
			panic("modal activation: supply a non-nil option")
		}
		option.applyModalActivation(a)
	}
	if len(a.modes) < 2 {
		panic("modes: supply at least two mode bodies")
	}
	return a
}
func (m ModeDefinition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "mode", Text: m.mode.Label, localBindingScope: true, children: declaredTargetNodes(m.mode.Targets, m.grouped)}
	if m.mode.Label == "" {
		n.problem = "supply a mode label"
	}
	for i, e := range m.mode.Effects {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("effects[%d]", i), e})
	}
	return n
}
func modeEdges(modes []Mode, grouped ...bool) []definitionEdge {
	result := make([]definitionEdge, 0, len(modes))
	for i, m := range modes {
		result = append(result, definitionEdge{fmt.Sprintf("modes[%d]", i), ModeDefinition{mode: m, grouped: len(grouped) > 0 && grouped[0]}})
	}
	return result
}

// WhenEventModes creates an event trigger with one mode chosen at stack placement.
func WhenEventModes(pattern EventPattern, modes ...ModeDefinition) TriggeredAbility {
	if nilDefinition(pattern) {
		panic("event: supply a pattern")
	}
	t := pattern.makeTrigger(nil)
	t.eventDefinition = pattern
	t.effects = nil
	return t.WithModes(modeValues(modes)...)
}

// AtModes creates a turn-step trigger with one mode chosen at stack placement.
func AtModes(pattern TurnStepPattern, modes ...ModeDefinition) TriggeredAbility {
	if nilDefinition(pattern) {
		panic("step: supply a pattern")
	}
	t := pattern.makeStepTrigger(nil)
	t.stepDefinition = pattern
	t.effects = nil
	return t.WithModes(modeValues(modes)...)
}

// --- from composable_counter_value.go ---
type counterResolutionValue struct {
	subject PermanentRef
	counter CounterType
}

// CountersOn reads the current count of one counter kind on a permanent.
// An absent permanent supplies zero. This value does not use last-known data.
func CountersOn(subject PermanentRef, counter CounterType) ResolutionValue {
	if counter < 0 || counter >= NumCounters {
		panic("CountersOn requires a valid counter kind")
	}
	return counterResolutionValue{subject: subject, counter: counter}
}

func (v counterResolutionValue) Text() string {
	return v.counter.String() + " counters on subject"
}

func (v counterResolutionValue) resolveValue(ctx *EffectContext) int {
	p, ok := v.subject.resolvePermanent(ctx)
	if !ok {
		return 0
	}
	return int(p.Counters[v.counter])
}

func (v counterResolutionValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "counter-value", Text: v.Text(), children: []definitionEdge{{"subject", v.subject}}}
}

// PermanentsDealtDamageAsCreaturesByThisTurn keeps recipient types at damage.
// Selection still requires each original recipient to be present.
func PermanentsDealtDamageAsCreaturesByThisTurn(source DamageSourceRef, candidates PermanentSet) PermanentSet {
	return turnDamageHistoryPermanents{source: source, candidates: candidates, creaturesOnly: true}
}
