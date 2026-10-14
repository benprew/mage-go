package mage

import (
	"fmt"
	"slices"
	"strings"

	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// --- from composable_activation.go ---
type activationRestriction struct {
	kind      string
	timing    TimingRule
	step      PhaseStep
	moreSteps []PhaseStep
	limit     int
	condition StateCondition
}

func (activationRestriction) activationOption() {}
func (o activationRestriction) applyActivation(a *ActionDefinition) {
	combinedTiming := false
	step := o.step
	moreSteps := o.moreSteps
	for _, prior := range a.dslRestrictions {
		if prior.kind == o.kind && o.kind != "condition" {
			if o.kind == "timing" && ((prior.timing == TimingStepOnly && o.timing == YourTurnOnly) || (prior.timing == YourTurnOnly && o.timing == TimingStepOnly)) {
				combinedTiming = true
				if prior.timing == TimingStepOnly {
					step = prior.step
					moreSteps = prior.moreSteps
				}
				continue
			}
			panic("activation: conflicting " + o.kind + " options")
		}
	}
	switch o.kind {
	case "permission":
		a.permission.AnyPlayerMayUse = true
	case "timing":
		if o.timing == TimingStepOnly && (o.step < Untap || o.step > Cleanup) {
			panic("activation: supply a valid turn step")
		}
		a.timing, a.stepOnly = o.timing, o.step
		if combinedTiming {
			a.timing, a.stepOnly = TimingStepOnly, step
			a.activationConds = append(a.activationConds, func(g *Game, _ *Permanent, controller uuid.UUID) bool {
				return g.ActivePlayerObj().PlayerID() == controller
			})
		}
	case "limit":
		if o.limit <= 0 {
			panic("activation: supply a positive limit")
		}
		a.limits.MaxActivationsPerTurn = o.limit
	case "condition":
		a.activationConds = append(a.activationConds, func(g *Game, source *Permanent, controller uuid.UUID) bool {
			if nilDefinition(o.condition) {
				return false
			}
			var id uuid.UUID
			if source != nil {
				id = source.ID()
			}
			return o.condition.checkState(g, StateContext{SourceID: id, ControllerID: controller})
		})
	}
	if o.kind == "timing" && a.timing == TimingStepOnly {
		a.stepMask = 0
		for _, extra := range moreSteps {
			if extra < Untap || extra > Cleanup {
				panic("activation: supply a valid turn step")
			}
			a.stepMask |= 1 << uint(extra)
		}
	}
	a.dslRestrictions = append(a.dslRestrictions, o)
}

// AnyPlayerMayActivate permits each player to activate and pay for the ability.
func AnyPlayerMayActivate() ActivationOption {
	return activationRestriction{kind: "permission"}
}

// ActivationTimingOption sets timing for an activation or a static grant.
type ActivationTimingOption interface {
	ActivationOption
	GrantedActivationOption
	ModalActivationOption
}

type activationTiming struct{ activationRestriction }

func (activationTiming) grantedActivationOption() {}
func (o activationTiming) applyGrantedActivation(a *ActionDefinition) {
	o.applyActivation(a)
}

// OnlyAsSorcery requires a main phase, the player's turn, and an empty stack.
func OnlyAsSorcery() ActivationTimingOption {
	return activationTiming{activationRestriction{kind: "timing", timing: TimingSorcery}}
}

// StepTimingOption restricts a spell or activation to the specified turn steps.
type StepTimingOption interface {
	ActivationTimingOption
	SpellOption
}

type stepTiming struct{ activationTiming }

func (stepTiming) spellOption() {}
func (o stepTiming) applySpell(a *ActionDefinition) {
	o.applyActivation(a)
}

// OnlyDuring restricts casting or activation to any of the specified turn steps.
func OnlyDuring(step PhaseStep, moreSteps ...PhaseStep) StepTimingOption {
	return stepTiming{activationTiming{activationRestriction{kind: "timing", timing: TimingStepOnly, step: step, moreSteps: append([]PhaseStep(nil), moreSteps...)}}}
}

// OnlyOnYourTurn restricts activation to the controller's turn.
func OnlyOnYourTurn() ActivationTimingOption {
	return activationTiming{activationRestriction{kind: "timing", timing: YourTurnOnly}}
}

// ActivationLimitOption sets a per-turn limit for an activation or a static grant.
type ActivationLimitOption interface {
	ActivationOption
	GrantedActivationOption
	ModalActivationOption
}

type activationLimit struct{ activationRestriction }

func (activationLimit) grantedActivationOption() {}
func (o activationLimit) applyGrantedActivation(a *ActionDefinition) {
	o.applyActivation(a)
}

// ActivationLimit restricts the number of activations each turn.
func ActivationLimit(maximum int) ActivationLimitOption {
	return activationLimit{activationRestriction{kind: "limit", limit: maximum}}
}

// ActivationConditionOption sets a condition for an activation or a static grant.
type ActivationConditionOption interface {
	ActivationOption
	GrantedActivationOption
	ModalActivationOption
}

type activationStateCondition struct{ activationRestriction }

func (activationStateCondition) grantedActivationOption() {}
func (o activationStateCondition) applyGrantedActivation(a *ActionDefinition) {
	o.applyActivation(a)
}

// ActivationIf checks a pure condition before payment through the activation path.
func ActivationIf(condition StateCondition) ActivationConditionOption {
	return activationStateCondition{activationRestriction{kind: "condition", condition: condition}}
}

// WithActivationStateCondition checks a typed condition before activation payment.
func WithActivationStateCondition(condition StateCondition) ActionOption {
	return func(a *ActionDefinition) {
		activationRestriction{kind: "condition", condition: condition}.applyActivation(a)
	}
}

func (o activationRestriction) inspectDefinition() definitionNode {
	n := definitionNode{Role: "activation-" + o.kind}
	if o.kind == "condition" {
		n.children = []definitionEdge{{"condition", o.condition}}
	}
	return n
}

func (activationTiming) modalActivationOption()                     {}
func (o activationTiming) applyModalActivation(a *ActionDefinition) { o.applyActivation(a) }

func (activationLimit) modalActivationOption()                     {}
func (o activationLimit) applyModalActivation(a *ActionDefinition) { o.applyActivation(a) }

func (activationStateCondition) modalActivationOption()                     {}
func (o activationStateCondition) applyModalActivation(a *ActionDefinition) { o.applyActivation(a) }

type spellStepDefinition struct {
	step      PhaseStep
	moreSteps []PhaseStep
}

func (s spellStepDefinition) inspectDefinition() definitionNode {
	steps := make([]string, 1, 1+len(s.moreSteps))
	steps[0] = s.step.String()
	for _, step := range s.moreSteps {
		steps = append(steps, step.String())
	}
	return definitionNode{Role: "spell-step", Text: strings.Join(steps, " or ")}
}

// --- from composable_activation_capture.go ---
// ActivationCaptureOption saves a value before activation payment.
type ActivationCaptureOption interface {
	ActivationOption
	GrantedActivationOption
	ModalActivationOption
}

type activationCapture interface {
	definitionInspector
	capture(*EffectContext)
}

type activationNumberCapture struct {
	value   ResolutionValue
	binding *NumberBinding
}

// CaptureNumberBeforePayment saves a value after target selection and cost
// validation, before mana production and payment. Copies retain that value.
func CaptureNumberBeforePayment(value ResolutionValue, binding *NumberBinding) ActivationCaptureOption {
	return activationNumberCapture{value, binding}
}

func (activationNumberCapture) activationOption()        {}
func (activationNumberCapture) grantedActivationOption() {}
func (activationNumberCapture) modalActivationOption()   {}
func (o activationNumberCapture) applyActivation(a *ActionDefinition) {
	a.dslCaptures = append(a.dslCaptures, o)
}
func (o activationNumberCapture) applyGrantedActivation(a *ActionDefinition) { o.applyActivation(a) }
func (o activationNumberCapture) applyModalActivation(a *ActionDefinition)   { o.applyActivation(a) }
func (o activationNumberCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "activation-number-capture", bindingWrite: o.binding,
		children: []definitionEdge{{"value", o.value}, {"binding", o.binding}}}
}

func (a *ActionDefinition) captureValues(g *Game, source *Permanent, controller uuid.UUID, specs []Target, targets []uuid.UUID) (numbers map[*NumberBinding]int, permanents map[*PermanentBinding]capturedPermanent) {
	if len(a.dslCaptures) == 0 {
		return nil, nil
	}
	outer := g.resolution.numberBindings
	outerPermanents := g.resolution.permanentBindings
	permanents = make(map[*PermanentBinding]capturedPermanent)
	g.resolution.permanentBindings = permanents
	defer func() { g.resolution.permanentBindings = outerPermanents }()
	values := make(map[*NumberBinding]int)
	g.resolution.numberBindings = values
	defer func() { g.resolution.numberBindings = outer }()
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: controller, Targets: targets,
		sourceCaptured: true, sourceIncarnation: source.incarnationID,
		targetBindings: make(map[*targetDeclaration][]capturedPermanent)}
	if len(specs) != len(targets) {
		specs = expandTargetSpecs(specs, targets, g.resolution.X())
	}
	for i, spec := range specs {
		if binding, ok := spec.(*targetDeclaration); ok && targets[i] != uuid.Nil {
			value := capturedPermanent{id: targets[i]}
			if p := g.FindPermanent(targets[i]); p != nil {
				value.incarnation = p.incarnationID
			}
			ctx.targetBindings[binding] = append(ctx.targetBindings[binding], value)
		}
	}
	for _, capture := range a.dslCaptures {
		capture.capture(ctx)
	}
	return values, permanents
}

func (o activationNumberCapture) capture(ctx *EffectContext) {
	ctx.Game.resolution.numberBindings[o.binding] = o.value.resolveValue(ctx)
}

type activationPermanentCapture struct {
	subject PermanentRef
	binding *PermanentBinding
}

// CapturePermanentBeforePayment saves a permanent after target and cost
// validation, before mana production and payment. Copies retain its identity.
func CapturePermanentBeforePayment(subject PermanentRef, binding *PermanentBinding) ActivationCaptureOption {
	return activationPermanentCapture{subject, binding}
}

func (activationPermanentCapture) activationOption()        {}
func (activationPermanentCapture) grantedActivationOption() {}
func (activationPermanentCapture) modalActivationOption()   {}
func (o activationPermanentCapture) applyActivation(a *ActionDefinition) {
	a.dslCaptures = append(a.dslCaptures, o)
}
func (o activationPermanentCapture) applyGrantedActivation(a *ActionDefinition) { o.applyActivation(a) }
func (o activationPermanentCapture) applyModalActivation(a *ActionDefinition)   { o.applyActivation(a) }
func (o activationPermanentCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "activation-permanent-capture", bindingWrite: o.binding,
		children: []definitionEdge{{"subject", o.subject}, {"binding", o.binding}}}
}
func (o activationPermanentCapture) capture(ctx *EffectContext) {
	snapshot := permanentSnapshot(o)
	_ = snapshot.Apply(ctx)
}

// --- from composable_granted_activation.go ---
type excludedPermanentSet struct {
	subjects PermanentSet
	excluded PermanentRef
}

// WithoutPermanent removes one permanent reference from a set.
func WithoutPermanent(subjects PermanentSet, excluded PermanentRef) PermanentSet {
	return excludedPermanentSet{subjects, excluded}
}
func (excludedPermanentSet) permanentSet() {}
func (s excludedPermanentSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	excluded, ok := s.excluded.resolvePermanent(ctx)
	var result []*Permanent
	for _, p := range s.subjects.resolvePermanents(ctx) {
		if !ok || p.ID() != excluded.ID() {
			result = append(result, p)
		}
	}
	return result
}
func (s excludedPermanentSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-set", Text: "exclude permanent", children: []definitionEdge{{"subjects", s.subjects}, {"excluded", s.excluded}}}
}

type grantedActivationRule struct {
	subjects PermanentSet
	ability  *ActionDefinition
}

type grantRecipientSet struct{ subject PermanentRef }

func (grantRecipientSet) permanentSet() {}
func (s grantRecipientSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.subject) {
		return nil
	}
	if p, ok := s.subject.resolvePermanent(ctx); ok {
		return []*Permanent{p}
	}
	return nil
}
func (s grantRecipientSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-set", children: []definitionEdge{{"subject", s.subject}}}
}

// SingletonPermanent keeps the identity and legality checks of its reference.
func SingletonPermanent(subject PermanentRef) PermanentSet {
	return grantRecipientSet{subject}
}

func grantRecipient(subject PermanentRef) PermanentSet {
	if nilDefinition(subject) {
		panic("grant: supply a permanent reference")
	}
	return grantRecipientSet{subject}
}

// GrantActivatedAbility gives one current permanent an activated ability.
// The subject uses the donor context. The body uses the recipient context.
func GrantActivatedAbility(subject PermanentRef, costs CostGroup, body ResolvingEffect, options ...GrantedActivationOption) StaticRule {
	return GrantEachActivatedAbility(grantRecipient(subject), costs, body, options...)
}

// GrantActivatedModes gives one current permanent an ability with a mode choice.
// Each mode declares its own targets.
func GrantActivatedModes(subject PermanentRef, costs CostGroup, options ...ModalActivationOption) StaticRule {
	return GrantEachActivatedModes(grantRecipient(subject), costs, options...)
}

// GrantedActivationOption is a closed option for a static activation grant.
type GrantedActivationOption interface {
	grantedActivationOption()
	applyGrantedActivation(*ActionDefinition)
}

func (targetOptions) grantedActivationOption() {}
func (o targetOptions) applyGrantedActivation(a *ActionDefinition) {
	applyTargetOptions(a, o.bindings)
}

// GrantEachActivatedAbility gives each set member an activated ability.
// The set is evaluated in layer 6. The body uses the recipient as its source.
func GrantEachActivatedAbility(subjects PermanentSet, costs CostGroup, body ResolvingEffect, options ...GrantedActivationOption) StaticRule {
	if nilDefinition(subjects) {
		panic("grant: supply a permanent set")
	}
	ability := Activate(costs, body)
	for _, option := range options {
		if nilDefinition(option) {
			panic("grant: supply a non-nil option")
		}
		option.applyGrantedActivation(ability)
	}
	return &grantedActivationRule{subjects: subjects, ability: ability}
}

// GrantEachActivatedModes gives each set member an ability that chooses one mode.
// Each mode declares its own targets. Costs are paid by the recipient's controller.
func GrantEachActivatedModes(subjects PermanentSet, costs CostGroup, options ...ModalActivationOption) StaticRule {
	if nilDefinition(subjects) {
		panic("grant: supply a permanent set")
	}
	return &grantedActivationRule{subjects: subjects, ability: ActivateModes(costs, options...)}
}
func (*grantedActivationRule) staticRule() {}
func (r *grantedActivationRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&grantedActivationEffect{rule: r, grantID: uuid.New()}}
}
func (r *grantedActivationRule) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-ability-grant", children: []definitionEdge{{"subjects", r.subjects}, {"ability", grantedActivationDefinition{r.ability}}}}
}

type grantedActivationDefinition struct{ ability *ActionDefinition }

func (d grantedActivationDefinition) inspectDefinition() definitionNode {
	return definitionNode{Role: "granted-activation", bindingScope: true, children: []definitionEdge{{"definition", d.ability}}}
}

type grantedActivationEffect struct {
	effectSource
	rule    *grantedActivationRule
	grantID uuid.UUID
}

func (*grantedActivationEffect) Text() string                 { return "grant activated ability" }
func (*grantedActivationEffect) Properties() EffectProperties { return EffectProperties{} }
func (*grantedActivationEffect) GetLayer() Layer              { return LayerAbility }
func (*grantedActivationEffect) GetDuration() Duration        { return WhileOnBattlefield }
func (e *grantedActivationEffect) IsActive(g *Game) bool {
	p := g.FindPermanent(e.sourceID)
	return p != nil && !p.PhasedOut
}
func (e *grantedActivationEffect) ApplyContinuous(ctx *EffectContext) error {
	source := ctx.Game.FindPermanent(e.sourceID)
	if source == nil || source.PhasedOut {
		return nil
	}
	child := *ctx
	child.SourceID, child.Controller = source.ID(), source.ControllerID()
	child.sourceCaptured, child.sourceIncarnation = true, source.incarnationID
	for _, member := range e.rule.subjects.resolvePermanents(&child) {
		p := ctx.Game.MutablePermanent(member.ID())
		if p == nil || p.PhasedOut {
			continue
		}
		// A grant keeps its limit through layer evaluations and control changes.
		var identity [32]byte
		copy(identity[:16], source.incarnationID[:])
		copy(identity[16:], p.incarnationID[:])
		installGrantedActivation(p, e.rule.ability, uuid.NewSHA1(e.grantID, identity[:]))
	}
	return nil
}
func (e *grantedActivationEffect) cloneEffect() ContinuousEffect { cloned := *e; return &cloned }
func (e *grantedActivationEffect) inspectDefinition() definitionNode {
	return e.rule.inspectDefinition()
}

func installGrantedActivation(p *Permanent, definition *ActionDefinition, id uuid.UUID) {
	ability := *definition
	ability.id = id
	ability.staticGrant = true
	ability.source, ability.controller = p.ID(), p.ControllerID()
	p.RuntimeAbilities = append(p.RuntimeAbilities, WrapGrantedAbility(&ability))
}

// --- from composable_mana_amount.go ---
type manaAmountEffect struct {
	color  Color
	amount ResolutionValue
}

type manaAnyOneColorEffect struct {
	amount      ResolutionValue
	restriction ManaRestriction
	restricted  bool
}

type manaToPlayerEffect struct {
	player PlayerRef
	color  Color
	amount ResolutionValue
}

// AddManaToPlayer puts mana into the selected player's pool.
func AddManaToPlayer(player PlayerRef, color Color, amount ResolutionValue) ResolvingEffect {
	return &manaToPlayerEffect{player: player, color: color, amount: amount}
}

func (e *manaToPlayerEffect) Text() string                 { return "add mana to player" }
func (e *manaToPlayerEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *manaToPlayerEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || resolutionValueReadsSourcePermanent(e.amount)
}
func (e *manaToPlayerEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"amount", e.amount}}}
	if e.color < Colorless || e.color >= AnyColor {
		n.problem = "supply a concrete mana color"
	}
	return n
}
func (e *manaToPlayerEffect) Apply(ctx *EffectContext) error {
	player, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	child := *ctx
	child.Controller = player
	return AddMana(e.color, amount).Apply(&child)
}

// AddManaOfAnyOneColor adds the full amount in one color chosen at resolution.
func AddManaOfAnyOneColor(amount ResolutionValue) ResolvingEffect {
	return &manaAnyOneColorEffect{amount: amount}
}

// AddRestrictedManaOfAnyOneColor applies one spending rule to the full amount.
// The controller chooses one color when the effect resolves.
func AddRestrictedManaOfAnyOneColor(amount ResolutionValue, restriction ManaRestriction) ResolvingEffect {
	return &manaAnyOneColorEffect{amount: amount, restriction: restriction, restricted: true}
}

func (e *manaAnyOneColorEffect) Text() string {
	if nilDefinition(e.amount) {
		return "add mana of any one color"
	}
	return fmt.Sprintf("add %s mana of any one color", e.amount.Text())
}
func (e *manaAnyOneColorEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *manaAnyOneColorEffect) refersToSourcePermanent() bool {
	return resolutionValueReadsSourcePermanent(e.amount)
}
func (e *manaAnyOneColorEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"amount", e.amount}}}
	if e.restricted && nilDefinition(e.restriction) {
		n.problem = "supply a mana spending restriction"
	}
	return n
}
func (e *manaAnyOneColorEffect) Apply(ctx *EffectContext) error {
	if ctx.Game.GetPlayer(ctx.Controller) == nil {
		return ErrPlayerNotFound
	}
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	return (&addAnyManaEffect{amount: amount, restriction: e.restriction}).Apply(ctx)
}

// AddManaAmount adds mana to the resolving controller's pool.
// It reads the amount once. A nonpositive amount adds no mana.
func AddManaAmount(color Color, amount ResolutionValue) ResolvingEffect {
	return &manaAmountEffect{color: color, amount: amount}
}

func (e *manaAmountEffect) Text() string {
	if nilDefinition(e.amount) {
		return fmt.Sprintf("add %s mana", e.color)
	}
	return fmt.Sprintf("add %s %s mana", e.amount.Text(), e.color)
}
func (e *manaAmountEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *manaAmountEffect) refersToSourcePermanent() bool {
	return resolutionValueReadsSourcePermanent(e.amount)
}
func (e *manaAmountEffect) inspectDefinition() definitionNode {
	node := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"amount", e.amount}}}
	if e.color < Colorless || e.color >= AnyColor {
		node.problem = "supply a concrete mana color"
	}
	return node
}
func (e *manaAmountEffect) Apply(ctx *EffectContext) error {
	if ctx.Game.GetPlayer(ctx.Controller) == nil {
		return ErrPlayerNotFound
	}
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	return AddMana(e.color, amount).Apply(ctx)
}

// IsActivatedManaAbility uses the same classification as immediate resolution.
func IsActivatedManaAbility(ability Ability) bool {
	inner := UnwrapAbility(ability)
	if _, ok := inner.(*ManaAbility); ok {
		return true
	}
	activated, ok := inner.(ActivatedAbility)
	return ok && inner.Type() == AbilityActivated && canProduceMana(activated.Effects(), activated.Targets())
}

func canProduceMana(effects []Effect, targets []Target) bool {
	if len(targets) != 0 {
		return false
	}
	for _, effect := range effects {
		switch e := effect.(type) {
		case *manaAmountEffect:
			return true
		case *manaToPlayerEffect:
			return true
		case *manaFromProducedTypesEffect:
			return true
		case *manaAnyOneColorEffect:
			return true
		case *restrictedFixedManaEffect:
			return true
		case *addManaEffect:
			if e.amount > 0 {
				return true
			}
		case *addAnyManaEffect:
			if e.amount > 0 {
				return true
			}
		case *sequenceEffect:
			if canProduceMana(e.effects, nil) {
				return true
			}
		case *compositeEffect:
			if canProduceMana(e.effects, nil) {
				return true
			}
		}
	}
	return false
}

// --- from composable_mana_types.go ---
// ManaTypesBinding names the mana types saved for one trigger instance.
type ManaTypesBinding struct{ name string }

func cloneManaTypesBindings(src map[*ManaTypesBinding][]Color) map[*ManaTypesBinding][]Color {
	if src == nil {
		return nil
	}
	dst := make(map[*ManaTypesBinding][]Color, len(src))
	for binding, types := range src {
		dst[binding] = slices.Clone(types)
	}
	return dst
}

// BindManaTypes declares an event-time set of mana types.
func BindManaTypes(name string) *ManaTypesBinding { return &ManaTypesBinding{name: name} }

func (b *ManaTypesBinding) bindingName() string { return b.name }
func (b *ManaTypesBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "mana-types-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}

type eventManaTypesCapture struct {
	binding *ManaTypesBinding
	types   []Color
}

func (*eventManaTypesCapture) Text() string                 { return "remember produced mana types" }
func (*eventManaTypesCapture) Properties() EffectProperties { return EffectProperties{} }
func (e *eventManaTypesCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-mana-types-capture", bindingWrite: e.binding, children: []definitionEdge{{"types", e.binding}}}
}
func (e *eventManaTypesCapture) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.manaTypesBindings == nil {
		ctx.Game.resolution.manaTypesBindings = make(map[*ManaTypesBinding][]Color)
	}
	ctx.Game.resolution.manaTypesBindings[e.binding] = slices.Clone(e.types)
	return nil
}

type manaFromProducedTypesEffect struct {
	player PlayerRef
	types  *ManaTypesBinding
	amount ResolutionValue
}

// AddManaOfProducedTypeToPlayer adds mana of one event-time produced type.
func AddManaOfProducedTypeToPlayer(player PlayerRef, types *ManaTypesBinding, amount ResolutionValue) ResolvingEffect {
	return &manaFromProducedTypesEffect{player: player, types: types, amount: amount}
}

func (*manaFromProducedTypesEffect) Text() string                 { return "add mana of a produced type to player" }
func (*manaFromProducedTypesEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *manaFromProducedTypesEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || resolutionValueReadsSourcePermanent(e.amount)
}
func (e *manaFromProducedTypesEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), bindingRead: e.types,
		children: []definitionEdge{{"player", e.player}, {"types", e.types}, {"amount", e.amount}}}
}
func (e *manaFromProducedTypesEffect) Apply(ctx *EffectContext) error {
	playerID, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	types := ctx.Game.resolution.manaTypesBindings[e.types]
	if len(types) == 0 {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	if amount <= 0 {
		return nil
	}
	player := ctx.Game.GetPlayer(playerID)
	selected := types[0]
	if len(types) > 1 {
		if choice := player.ChooseManaColor("add mana of a produced type"); slices.Contains(types, choice) {
			selected = choice
		}
	}
	child := *ctx
	child.Controller = playerID
	return AddMana(selected, amount).Apply(&child)
}

// --- from composable_payment.go ---
type permanentControllerRef struct{ subject PermanentRef }
type permanentOwnerRef struct{ subject PermanentRef }

// ControllerOf reads the current controller of the selected battlefield incarnation.
func ControllerOf(subject PermanentRef) PlayerRef { return permanentControllerRef{subject} }
func (permanentControllerRef) playerRef()         {}
func (permanentControllerRef) damageRecipient()   {}
func (r permanentControllerRef) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	p, ok := r.subject.resolvePermanent(ctx)
	if !ok {
		return uuid.Nil, false
	}
	return p.ControllerID(), ctx.Game.GetPlayer(p.ControllerID()) != nil
}
func (r permanentControllerRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (r permanentControllerRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-ref", Text: "controller", children: []definitionEdge{{"subject", r.subject}}}
}

// OwnerOf reads the owner of the selected battlefield incarnation.
func OwnerOf(subject PermanentRef) PlayerRef { return permanentOwnerRef{subject} }
func (permanentOwnerRef) playerRef()         {}
func (permanentOwnerRef) damageRecipient()   {}
func (r permanentOwnerRef) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	if nilDefinition(r.subject) {
		return uuid.Nil, false
	}
	p, ok := r.subject.resolvePermanent(ctx)
	if !ok {
		return uuid.Nil, false
	}
	owner := p.Card.Owner()
	return owner, ctx.Game.GetPlayer(owner) != nil
}
func (r permanentOwnerRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (r permanentOwnerRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-ref", Text: "owner", children: []definitionEdge{{"subject", r.subject}}}
}

type unlessPaidEffect struct {
	player      PlayerRef
	costs       CostGroup
	effect      ResolvingEffect
	otherwise   ResolvingEffect
	twoBranches bool
	ifPaid      bool
	required    bool
}

// UnlessPaid offers one player a payment through the total-cost transaction.
// A declined or failed payment runs the effect. An absent player has no effect.
func UnlessPaid(player PlayerRef, costs CostGroup, effect ResolvingEffect) ResolvingEffect {
	return &unlessPaidEffect{player: player, costs: costs, effect: effect}
}

// IfPaid runs the body only after the player accepts and pays the complete cost.
func IfPaid(player PlayerRef, costs CostGroup, effect ResolvingEffect) ResolvingEffect {
	return &unlessPaidEffect{player: player, costs: costs, effect: effect, ifPaid: true}
}

// PayOrElse uses one payment result to select one resolution branch.
func PayOrElse(player PlayerRef, costs CostGroup, paid, unpaid ResolvingEffect) ResolvingEffect {
	return &unlessPaidEffect{player: player, costs: costs, effect: paid, otherwise: unpaid, ifPaid: true, twoBranches: true}
}

// IfRequiredPaid runs the body after a mandatory resolution payment succeeds.
// The player does not receive an optional payment choice.
func IfRequiredPaid(player PlayerRef, costs CostGroup, effect ResolvingEffect) ResolvingEffect {
	return &unlessPaidEffect{player: player, costs: costs, effect: effect, ifPaid: true, required: true}
}

// UnlessRequiredPaid runs the body when a mandatory resolution payment fails.
func UnlessRequiredPaid(player PlayerRef, costs CostGroup, effect ResolvingEffect) ResolvingEffect {
	return &unlessPaidEffect{player: player, costs: costs, effect: effect, required: true}
}

func (e *unlessPaidEffect) Text() string {
	if e.twoBranches {
		return "pay, then resolve one branch"
	}
	if e.ifPaid {
		return "if the player pays"
	}
	return "unless the player pays"
}
func (e *unlessPaidEffect) Properties() EffectProperties {
	if nilDefinition(e.effect) {
		return EffectProperties{}
	}
	props := e.effect.Properties()
	if !nilDefinition(e.otherwise) {
		mergeEffectProperties(&props, e.otherwise.Properties())
	}
	return props
}
func (e *unlessPaidEffect) Apply(ctx *EffectContext) error {
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	costs := e.costs.costs
	if e.required {
		costs = make([]Cost, len(e.costs.costs))
		for i, cost := range e.costs.costs {
			costs[i] = cost
			if sacrifice, ok := cost.(*sacrificeMatchingCost); ok && !sacrifice.includeSource {
				copied := *sacrifice
				original := sacrifice.filter
				sourceID, incarnation := ctx.SourceID, ctx.sourceIncarnation
				copied.includeSource = true
				// A returned card is a new object and can pay the old ability.
				copied.filter = NewPermanentFilter(original.Text(), func(p *Permanent, g *Game) bool {
					return (p.ID() != sourceID || p.incarnationID != incarnation) && original.Match(p, g)
				})
				costs[i] = &copied
			}
		}
	}
	paid := tryResolutionCostPayment(ctx, id, costs, e.required)
	if paid == e.ifPaid {
		return e.effect.Apply(ctx)
	}
	if !nilDefinition(e.otherwise) {
		return e.otherwise.Apply(ctx)
	}
	return nil
}

func tryResolutionCostPayment(ctx *EffectContext, id uuid.UUID, costs []Cost, required bool) bool {
	return tryResolutionCostPaymentFor(ctx, id, costs, required, "")
}

func tryResolutionCostPaymentFor(ctx *EffectContext, id uuid.UUID, costs []Cost, required bool, subject string) bool {
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return false
	}
	texts := make([]string, len(costs))
	for i, cost := range costs {
		texts[i] = cost.Text()
	}
	paid := false
	available := true
	for _, cost := range costs {
		if !canPayWithManaPolicy(cost, ctx.SourceID, id, ctx.Game, false) {
			available = false
			break
		}
	}
	prompt := "pay " + strings.Join(texts, ", ")
	if subject != "" {
		prompt += " for " + subject
	}
	if available && (required || player.ChooseMayAbility(prompt)) {
		tx, err := ctx.Game.prepareActionPaymentTransaction(actionPaymentSpec{
			SourceID: ctx.SourceID, Controller: id, Costs: costs,
			Targets: ctx.Targets, XValue: ctx.Game.XValue(),
		})
		paid = err == nil && tx.Commit() == nil
	}
	return paid
}
func (e *unlessPaidEffect) inspectDefinition() definitionNode {
	role, branch := "unless-paid", "unpaid"
	if e.ifPaid {
		role, branch = "if-paid", "paid"
	}
	if e.required {
		role = "required-" + role
	}
	n := definitionNode{Role: role, Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"costs", e.costs}, {branch, e.effect}}}
	if e.twoBranches {
		n.children = append(n.children, definitionEdge{"unpaid", e.otherwise})
	}
	if e.required {
		if len(e.costs.costs) == 0 {
			n.problem = "supply at least one mandatory cost"
		}
		for _, cost := range e.costs.costs {
			if _, ok := cost.(*discardCost); ok {
				continue
			}
			if _, ok := cost.(*sacrificeMatchingCost); !ok {
				n.problem = "mandatory resolution payment requires discard or matching sacrifice costs"
			}
		}
	}
	return n
}
func (c CostGroup) inspectDefinition() definitionNode {
	n := definitionNode{Role: "cost-group"}
	if !c.present {
		n.problem = "supply an explicit Costs group"
	}
	for i, cost := range c.costs {
		if _, ok := cost.(*sourceCardCost); ok {
			n.problem = "paid source card requires an activated ability cost"
		}
		if mill, ok := cost.(*millCost); ok && mill.capture {
			n.problem = "paid mill count requires an activated ability cost or spell additional cost"
		}
		if optional, ok := cost.(*optionalCost); ok && optional.capture {
			n.problem = "optional payment result requires an activated ability cost or spell additional cost"
		}
		if exile, ok := cost.(*exileMatchingCardFromGraveyardCost); ok && (exile.capture || exile.captureCard) {
			n.problem = "paid graveyard exile value requires an activated ability cost or spell additional cost"
		}
		if discard, ok := cost.(*discardCost); ok && (discard.captureCards || discard.captureCard) {
			n.problem = "paid discard cards require an activated ability cost or spell additional cost"
		}
		if exile, ok := cost.(*exileFromGraveyardCost); ok && exile.paidRequired {
			n.problem = "paid graveyard exile cards require an activated ability cost or spell additional cost"
		}
		if _, ok := cost.(*xLifePayCost); ok {
			n.problem = "paid life value requires an activated ability cost or spell additional cost"
		}
		if life, ok := cost.(*lifePayCost); ok && life.capture {
			n.problem = "paid life value requires an activated ability cost or spell additional cost"
		}
		if _, ok := cost.(*removeAnyCountersCost); ok {
			n.problem = "paid counter value requires an activated ability cost"
		}
		if _, ok := cost.(*capturedXCountersCost); ok {
			n.problem = "paid counter value requires an activated ability cost"
		}
		if counter, ok := cost.(*removeCountersCost); ok && counter.capture {
			n.problem = "paid counter value requires an activated ability cost"
		}
		if sacrifice, ok := cost.(*sacrificeMatchingCost); ok && (sacrifice.captureStats || sacrifice.captureManaValue || sacrifice.captureSubtype || sacrifice.capturePermanent || sacrifice.captureTotalPower) {
			n.problem = "paid sacrifice value requires an activated ability cost or spell additional cost"
		}
		if exile, ok := cost.(*matchingPermanentExileCost); ok && len(exile.facts) != 0 {
			n.problem = "paid permanent exile facts require an activated ability cost or spell additional cost"
		}
		if taps, ok := cost.(*tapNOtherMatchingCost); ok && taps.capture {
			n.problem = "paid tap permanent requires an activated ability cost or spell additional cost"
		}
		n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
	}
	return n
}
