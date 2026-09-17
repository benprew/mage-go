package mage

import (
	"fmt"

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
	power, toughness ValueSource
	keyword          Attr
	layer            Layer
	colors           []Color
	family           SubtypeFamily
	subtypes         []string
	addSubtypes      bool
}

func (*layerModification) continuousModification()            {}
func (m *layerModification) modification() *layerModification { return m }

// ModifyPT adds power and toughness to one permanent.
func ModifyPT(subject PermanentRef, power, toughness ValueSource) ContinuousModification {
	return &layerModification{subject: subject, power: power, toughness: toughness, layer: LayerPT}
}

// ModifyEachPT applies a power and toughness modification to a permanent set.
func ModifyEachPT(subjects PermanentSet, power, toughness ValueSource) ContinuousModification {
	return &layerModification{subjects: subjects, power: power, toughness: toughness, layer: LayerPT}
}

// GrantKeywordModification grants a keyword. DSL exports it as GrantKeyword.
func GrantKeywordModification(subject PermanentRef, keyword Attr) ContinuousModification {
	return &layerModification{subject: subject, keyword: keyword, layer: LayerAbility}
}

// GrantEachKeyword grants a keyword to each member of a permanent set.
func GrantEachKeyword(subjects PermanentSet, keyword Attr) ContinuousModification {
	return &layerModification{subjects: subjects, keyword: keyword, layer: LayerAbility}
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
		return fmt.Sprintf("grant %v", m.keyword)
	default:
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
	colors           []Color
	family           SubtypeFamily
	subtypes         []string
	addSubtypes      bool
}

func (m *layerModification) capture(ctx *EffectContext) []capturedModification {
	members := m.members(ctx)
	result := make([]capturedModification, 0, len(members))
	for _, p := range members {
		v := capturedModification{permanent: capturedPermanent{p.ID(), p.incarnationID}, keyword: m.keyword, colors: m.colors, family: m.family, subtypes: m.subtypes, addSubtypes: m.addSubtypes}
		if m.layer == LayerPT {
			v.power = m.power.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
			v.toughness = m.toughness.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
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
		p.powerBonus += v.power
		p.toughBonus += v.toughness
	case LayerAbility:
		g.effects.GrantAttr(p.ID(), v.keyword)
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
			effects = append(effects, &composedLayerEffect{layer: layer, duration: WhileOnBattlefield, rule: r, modifications: mods})
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
	layer         Layer
	duration      Duration
	rule          *continuousRule
	modifications []*layerModification
	captured      []capturedModification
}

func (e *composedLayerEffect) Text() string                 { return "continuous modification" }
func (e *composedLayerEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *composedLayerEffect) GetLayer() Layer              { return e.layer }
func (e *composedLayerEffect) GetDuration() Duration        { return e.duration }
func (e *composedLayerEffect) IsActive(g *Game) bool {
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
		for _, m := range e.modifications {
			for _, v := range m.capture(&child) {
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
}

// Until captures subjects and values at resolution and installs all layers together.
func Until(duration Duration, modifications ...ContinuousModification) ResolvingEffect {
	return &untilEffect{duration, append([]ContinuousModification(nil), modifications...)}
}
func (e *untilEffect) Text() string { return "apply temporary modifications" }
func (e *untilEffect) Properties() EffectProperties {
	p := EffectProperties{Outcome: OutcomeBenefit}
	for _, value := range e.modifications {
		if nilDefinition(value) {
			continue
		}
		m := value.modification()
		if v, ok := m.power.(fixedValue); ok {
			p.PowerBoost += v.n
		}
		if v, ok := m.toughness.(fixedValue); ok {
			p.ToughnessBoost += v.n
		}
		if m.keyword != 0 {
			p.GrantedKeyword = m.keyword
		}
		p.Mass = p.Mass || m.subjects != nil
	}
	return p
}
func (e *untilEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "until", Text: e.Text()}
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
	var effects []*composedLayerEffect
	for _, layer := range []Layer{LayerType, LayerColor, LayerAbility, LayerPT} {
		entry := &composedLayerEffect{layer: layer, duration: e.duration}
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
