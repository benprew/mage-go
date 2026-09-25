package mage

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ContinuousModification describes a layer operation before installation.
type ContinuousModification interface {
	continuousModification()
	modification() *layerModification
}
type layerModification struct {
	subject          PermanentRef
	subjects         PermanentSet
	power, toughness ResolutionValue
	keyword          Attr
	cantBeBlocked    bool
	setBasePT        bool
	setPowerOnly     bool
	layer            Layer
	colors           []Color
	family           SubtypeFamily
	subtypes         []string
	addSubtypes      bool
	ability          *ActionDefinition
}

func (*layerModification) continuousModification()            {}
func (m *layerModification) modification() *layerModification { return m }

// ModifyPT adds power and toughness to one permanent.
func ModifyPT(subject PermanentRef, power, toughness ResolutionValue) ContinuousModification {
	return &layerModification{subject: subject, power: power, toughness: toughness, layer: LayerPT}
}

// SetBasePowerToughness sets both base values of one permanent.
func SetBasePowerToughness(subject PermanentRef, power, toughness ResolutionValue) ContinuousModification {
	return &layerModification{subject: subject, power: power, toughness: toughness, layer: LayerPT, setBasePT: true}
}

// SetBasePower sets base power and preserves base toughness.
func SetBasePower(subject PermanentRef, power ResolutionValue) ContinuousModification {
	return &layerModification{subject: subject, power: power, toughness: Fixed(0), layer: LayerPT, setBasePT: true, setPowerOnly: true}
}

// ModifyEachPT applies a power and toughness modification to a permanent set.
func ModifyEachPT(subjects PermanentSet, power, toughness ResolutionValue) ContinuousModification {
	return &layerModification{subjects: subjects, power: power, toughness: toughness, layer: LayerPT}
}

// GrantKeywordModification grants a keyword. DSL exports it as GrantKeyword.
func GrantKeywordModification(subject PermanentRef, keyword Attr) ContinuousModification {
	return &layerModification{subject: subject, keyword: keyword, layer: LayerAbility}
}

// CantBeBlocked prevents all creatures from blocking the subject.
// Install this rules restriction with a duration; it does not grant an ability.
func CantBeBlocked(subject PermanentRef) ContinuousModification {
	return &layerModification{subject: subject, cantBeBlocked: true, layer: LayerAbility}
}

// GrantEachKeyword grants a keyword to each member of a permanent set.
func GrantEachKeyword(subjects PermanentSet, keyword Attr) ContinuousModification {
	return &layerModification{subjects: subjects, keyword: keyword, layer: LayerAbility}
}

// ActivatedAbilityModification gives a permanent an activated ability in layer 6.
// Install it with Until, UntilSourceLeaves, or While.
func ActivatedAbilityModification(subject PermanentRef, costs CostGroup, body ResolvingEffect, options ...GrantedActivationOption) ContinuousModification {
	rule := GrantActivatedAbility(subject, costs, body, options...).(*grantedActivationRule)
	return &layerModification{subject: subject, ability: rule.ability, layer: LayerAbility}
}

// ActivatedModesModification gives a permanent a modal activated ability in layer 6.
// Install it with Until, UntilSourceLeaves, or While. Each mode declares its targets.
func ActivatedModesModification(subject PermanentRef, costs CostGroup, options ...ModalActivationOption) ContinuousModification {
	rule := GrantActivatedModes(subject, costs, options...).(*grantedActivationRule)
	return &layerModification{subject: subject, ability: rule.ability, layer: LayerAbility}
}

// EachActivatedAbilityModification gives each set member an activated ability.
// Until captures the set at resolution. While evaluates the set in layer 6.
func EachActivatedAbilityModification(subjects PermanentSet, costs CostGroup, body ResolvingEffect, options ...GrantedActivationOption) ContinuousModification {
	rule := GrantEachActivatedAbility(subjects, costs, body, options...).(*grantedActivationRule)
	return &layerModification{subjects: subjects, ability: rule.ability, layer: LayerAbility}
}

// EachActivatedModesModification gives each set member a modal activated ability.
// Each recipient has separate targets, costs, and activation counts.
func EachActivatedModesModification(subjects PermanentSet, costs CostGroup, options ...ModalActivationOption) ContinuousModification {
	rule := GrantEachActivatedModes(subjects, costs, options...).(*grantedActivationRule)
	return &layerModification{subjects: subjects, ability: rule.ability, layer: LayerAbility}
}

// SetColors replaces a permanent's colors. An empty list makes it colorless.
func SetColors(subject PermanentRef, colors ...Color) ContinuousModification {
	return &layerModification{subject: subject, colors: modificationColors(colors), layer: LayerColor}
}

// SetEachColors replaces the colors of each member of a permanent set.
func SetEachColors(subjects PermanentSet, colors ...Color) ContinuousModification {
	return &layerModification{subjects: subjects, colors: modificationColors(colors), layer: LayerColor}
}

func modificationColors(colors []Color) []Color {
	for _, color := range colors {
		if color < White || color > Green {
			panic("colors: supply actual colors or an empty list for colorless")
		}
	}
	return append([]Color(nil), colors...)
}

// SetSubtypeFamily replaces subtypes in one family and retains the other families.
func SetSubtypeFamily(subject PermanentRef, family SubtypeFamily, subtypes ...string) ContinuousModification {
	validateSubtypeChange("SetSubtypeFamily", family, subtypes)
	return &layerModification{subject: subject, family: family, subtypes: append([]string(nil), subtypes...), layer: LayerType}
}

// AddSubtypeFamily adds subtypes without removing existing types or abilities.
func AddSubtypeFamily(subject PermanentRef, family SubtypeFamily, subtypes ...string) ContinuousModification {
	validateSubtypeChange("AddSubtypeFamily", family, subtypes)
	return &layerModification{subject: subject, family: family, subtypes: append([]string(nil), subtypes...), addSubtypes: true, layer: LayerType}
}

func (m *layerModification) inspectDefinition() definitionNode {
	subject := any(m.subject)
	if m.subjects != nil {
		subject = m.subjects
	}
	n := definitionNode{Role: "continuous-modification", Text: m.description(), children: []definitionEdge{{"subject", subject}}}
	if m.ability != nil {
		n.children = append(n.children, definitionEdge{"ability", grantedActivationDefinition{m.ability}})
	}
	if m.layer == LayerPT {
		n.children = append(n.children, definitionEdge{"power", m.power}, definitionEdge{"toughness", m.toughness})
	}
	return n
}
func (m *layerModification) description() string {
	switch m.layer {
	case LayerType:
		if m.addSubtypes {
			return fmt.Sprintf("add subtypes %v in family %v", m.subtypes, m.family)
		}
		return fmt.Sprintf("set subtypes %v in family %v", m.subtypes, m.family)
	case LayerColor:
		return fmt.Sprintf("set colors %v", m.colors)
	case LayerAbility:
		if m.cantBeBlocked {
			return "can’t be blocked"
		}
		if m.ability != nil {
			return "grant activated ability"
		}
		return fmt.Sprintf("grant %v", m.keyword)
	default:
		if m.setPowerOnly {
			return "set base power"
		}
		if m.setBasePT {
			return "set base power and toughness"
		}
		return "modify power and toughness"
	}
}

func (m *layerModification) members(ctx *EffectContext) []*Permanent {
	if m.subjects != nil {
		return m.subjects.resolvePermanents(ctx)
	}
	if p, ok := m.subject.resolvePermanent(ctx); ok {
		return []*Permanent{p}
	}
	return nil
}

type capturedModification struct {
	permanent        capturedPermanent
	power, toughness int
	keyword          Attr
	cantBeBlocked    bool
	setBasePT        bool
	setPowerOnly     bool
	colors           []Color
	family           SubtypeFamily
	subtypes         []string
	addSubtypes      bool
	ability          *ActionDefinition
	grantID          uuid.UUID
}

func (m *layerModification) capture(ctx *EffectContext) []capturedModification {
	members := m.members(ctx)
	result := make([]capturedModification, 0, len(members))
	for _, p := range members {
		v := capturedModification{permanent: capturedPermanent{p.ID(), p.incarnationID}, keyword: m.keyword, cantBeBlocked: m.cantBeBlocked, setBasePT: m.setBasePT, setPowerOnly: m.setPowerOnly, colors: m.colors, family: m.family, subtypes: m.subtypes, addSubtypes: m.addSubtypes}
		if m.ability != nil {
			v.ability, v.grantID = m.ability, uuid.New()
		}
		if m.layer == LayerPT {
			v.power = m.power.resolveValue(ctx)
			v.toughness = m.toughness.resolveValue(ctx)
		}
		result = append(result, v)
	}
	return result
}
func (v capturedModification) apply(g *Game, layer Layer) {
	p := g.FindPermanent(v.permanent.id)
	if p == nil || p.PhasedOut || p.incarnationID != v.permanent.incarnation {
		return
	}
	p = g.MutablePermanent(p.ID())
	switch layer {
	case LayerPT:
		if v.setBasePT {
			toughness := v.toughness
			if v.setPowerOnly {
				toughness = p.Card.Toughness()
				if p.BasePTOverride != nil {
					toughness = p.BasePTOverride[1]
				}
			}
			p.BasePTOverride = &[2]int{v.power, toughness}
		} else {
			p.powerBonus += v.power
			p.toughBonus += v.toughness
		}
	case LayerAbility:
		if v.cantBeBlocked {
			g.effects.AddCantBeBlockedExceptBy(p.ID(), Not(AnyPermanent))
		} else if v.ability != nil {
			installGrantedActivation(p, v.ability, v.grantID)
		} else {
			g.effects.GrantAttr(p.ID(), v.keyword)
		}
	case LayerColor:
		colors := append([]Color(nil), v.colors...)
		p.ColorOverride = &colors
	case LayerType:
		if v.addSubtypes {
			applyAddSubtypes(p, v.family, v.subtypes)
		} else {
			applySetSubtypes(g, p, v.family, v.subtypes)
		}
	}
}

// StaticRule can be installed as a static ability; resolving effects cannot.
type StaticRule interface {
	staticRule()
	staticEffects() []ContinuousEffect
}
type continuousRule struct {
	condition     StateCondition
	modifications []ContinuousModification
}

func (*continuousRule) staticRule() {}

// While applies modifications dynamically while the state condition is true.
func While(condition StateCondition, modifications ...ContinuousModification) StaticRule {
	return &continuousRule{condition, append([]ContinuousModification(nil), modifications...)}
}
func (r *continuousRule) inspectDefinition() definitionNode {
	n := definitionNode{Role: "continuous-rule", children: []definitionEdge{{"condition", r.condition}}}
	if len(r.modifications) == 0 {
		n.problem = "supply at least one modification"
	}
	for i, m := range r.modifications {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("modifications[%d]", i), m})
	}
	return n
}
func (r *continuousRule) staticEffects() []ContinuousEffect {
	if nilDefinition(r.condition) {
		panic("static: supply a condition")
	}
	if len(r.modifications) == 0 {
		panic("static: supply at least one modification")
	}
	var effects []ContinuousEffect
	for _, layer := range []Layer{LayerType, LayerColor, LayerAbility, LayerPT} {
		var mods []*layerModification
		for _, value := range r.modifications {
			if nilDefinition(value) {
				panic("static: supply a non-nil modification")
			}
			m := value.modification()
			if m.layer == layer {
				mods = append(mods, m)
			}
		}
		if len(mods) > 0 {
			effects = append(effects, &composedLayerEffect{layer: layer, duration: WhileOnBattlefield, rule: r, modifications: mods, grantID: uuid.New()})
		}
	}
	return effects
}

// Static installs only static rules in the existing static ability holder.
func Static(rules ...StaticRule) *StaticAbilityHolder {
	if len(rules) == 0 {
		panic("static: supply at least one rule")
	}
	var effects []ContinuousEffect
	for _, r := range rules {
		if nilDefinition(r) {
			panic("static: supply a non-nil rule")
		}
		effects = append(effects, r.staticEffects()...)
	}
	return StaticAbility(effects...)
}

type composedLayerEffect struct {
	effectSource
	layer             Layer
	duration          Duration
	rule              *continuousRule
	modifications     []*layerModification
	captured          []capturedModification
	sourceBound       bool
	sourceIncarnation uuid.UUID
	grantID           uuid.UUID
}

func (e *composedLayerEffect) Text() string                 { return "continuous modification" }
func (e *composedLayerEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *composedLayerEffect) GetLayer() Layer              { return e.layer }
func (e *composedLayerEffect) GetDuration() Duration        { return e.duration }
func (e *composedLayerEffect) survivesSourceExit() bool {
	return e.rule == nil && !e.sourceBound
}
func (e *composedLayerEffect) IsActive(g *Game) bool {
	if e.sourceBound {
		p := g.FindPermanentIncludingPhased(e.sourceID)
		return p != nil && p.incarnationID == e.sourceIncarnation
	}
	if e.rule == nil {
		return true
	}
	p := g.FindPermanent(e.sourceID)
	return p != nil && !p.PhasedOut
}
func (e *composedLayerEffect) ApplyContinuous(ctx *EffectContext) error {
	if e.rule == nil {
		for _, v := range e.captured {
			v.apply(ctx.Game, e.layer)
		}
		return nil
	}
	source := ctx.Game.FindPermanent(e.sourceID)
	if source == nil {
		return nil
	}
	child := *ctx
	child.SourceID = e.sourceID
	child.Controller = source.ControllerID()
	child.sourceCaptured = true
	child.sourceIncarnation = source.incarnationID
	if e.rule.condition.checkState(ctx.Game, StateContext{SourceID: e.sourceID, ControllerID: child.Controller}) {
		for i, m := range e.modifications {
			for _, v := range m.capture(&child) {
				if v.ability != nil {
					identity := fmt.Sprintf("%s/%s/%d", source.incarnationID, v.permanent.incarnation, i)
					v.grantID = uuid.NewSHA1(e.grantID, []byte(identity))
				}
				v.apply(ctx.Game, e.layer)
			}
		}
	}
	return nil
}
func (e *composedLayerEffect) cloneEffect() ContinuousEffect { cloned := *e; return &cloned }
func (e *composedLayerEffect) inspectDefinition() definitionNode {
	if e.rule != nil {
		return e.rule.inspectDefinition()
	}
	return definitionNode{Role: "installed-continuous-effect"}
}

type untilEffect struct {
	duration      Duration
	modifications []ContinuousModification
	sourceBound   bool
}

// Until captures subjects and values at resolution and installs all layers together.
func Until(duration Duration, modifications ...ContinuousModification) ResolvingEffect {
	return &untilEffect{duration: duration, modifications: append([]ContinuousModification(nil), modifications...)}
}

// UntilSourceLeaves captures modifications until this source leaves the battlefield.
// Phasing does not end this duration. An absent or returned source has no effect.
func UntilSourceLeaves(_ sourceRef, modifications ...ContinuousModification) ResolvingEffect {
	return &untilEffect{duration: Indefinite, modifications: append([]ContinuousModification(nil), modifications...), sourceBound: true}
}

func (e *untilEffect) Text() string {
	var descriptions []string
	for _, m := range e.modifications {
		if !nilDefinition(m) {
			descriptions = append(descriptions, m.modification().description())
		}
	}
	body := strings.Join(descriptions, "; ")
	if e.sourceBound {
		return body + " until this source leaves the battlefield"
	}
	switch e.duration {
	case EndOfTurn:
		return body + " until end of turn"
	case EndOfCombat:
		return body + " until end of combat"
	default:
		return body + " indefinitely"
	}
}
func (e *untilEffect) Properties() EffectProperties {
	p := EffectProperties{Outcome: OutcomeBenefit}
	for _, value := range e.modifications {
		if nilDefinition(value) {
			continue
		}
		m := value.modification()
		if m.setBasePT {
			p.Outcome = OutcomeDetriment
		}
		if v, ok := m.power.(fixedValue); ok && !m.setBasePT {
			p.PowerBoost += v.n
		}
		if v, ok := m.toughness.(fixedValue); ok && !m.setBasePT {
			p.ToughnessBoost += v.n
		}
		if m.keyword != 0 {
			p.GrantedKeyword = m.keyword
			if m.keyword == AttrMustBeBlockedIfAble || m.keyword == CantRegenerate {
				p.Outcome = OutcomeDetriment
			}
		}
		p.Mass = p.Mass || m.subjects != nil
	}
	if p.PowerBoost < 0 || p.ToughnessBoost < 0 {
		p.Outcome = OutcomeDetriment
	}
	return p
}

// KeywordGrantTargetIndex identifies one pure target keyword grant for AI checks.
// Unknown target boundaries and compound modifications require full evaluation.
func (e *untilEffect) KeywordGrantTargetIndex(targets []Target) (int, bool) {
	if e.sourceBound || e.duration != EndOfTurn || len(e.modifications) != 1 || nilDefinition(e.modifications[0]) {
		return 0, false
	}
	m := e.modifications[0].modification()
	if m.layer != LayerAbility || m.keyword == 0 || m.subjects != nil {
		return 0, false
	}
	subject, ok := m.subject.(*PermanentTargetBinding)
	if !ok || subject == nil {
		return 0, false
	}
	index := 0
	for _, target := range targets {
		if target == subject.targetDeclaration {
			return index, true
		}
		if target.Min() < 0 || target.Min() != target.Max() {
			return 0, false
		}
		index += target.Min()
	}
	return 0, false
}

func (e *untilEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "until", Text: e.Text()}
	if e.sourceBound {
		n.Role = "until-source-leaves"
		n.children = append(n.children, definitionEdge{"source", This})
	}
	if len(e.modifications) == 0 {
		n.problem = "supply at least one modification"
	}
	if e.duration != EndOfTurn && e.duration != EndOfCombat && e.duration != Indefinite {
		n.problem = "duration requires an adapter with an explicit lifetime policy"
	}
	for i, m := range e.modifications {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("modifications[%d]", i), m})
	}
	return n
}
func (e *untilEffect) Apply(ctx *EffectContext) error {
	var sourceIncarnation uuid.UUID
	if e.sourceBound {
		source := ctx.Game.FindPermanentIncludingPhased(ctx.SourceID)
		if source == nil || (ctx.sourceCaptured && source.incarnationID != ctx.sourceIncarnation) {
			return nil
		}
		sourceIncarnation = source.incarnationID
	}
	var effects []*composedLayerEffect
	for _, layer := range []Layer{LayerType, LayerColor, LayerAbility, LayerPT} {
		entry := &composedLayerEffect{layer: layer, duration: e.duration, sourceBound: e.sourceBound, sourceIncarnation: sourceIncarnation}
		entry.SetSourceID(ctx.SourceID)
		for _, value := range e.modifications {
			m := value.modification()
			if m.layer == layer {
				entry.captured = append(entry.captured, m.capture(ctx)...)
			}
		}
		if len(entry.captured) > 0 {
			effects = append(effects, entry)
		}
	}
	for _, entry := range effects {
		ctx.Game.effects.Add(entry)
	}
	ctx.Game.ApplyContinuousEffects()
	return nil
}
