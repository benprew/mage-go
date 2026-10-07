package mage

import (
	"fmt"
	"strings"

	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// --- from composable_continuous.go ---
// ContinuousModification describes a layer operation before installation.
type ContinuousModification interface {
	continuousModification()
	modification() *layerModification
}
type layerModification struct {
	subject               PermanentRef
	subjects              PermanentSet
	controller            PlayerRef
	power, toughness      ResolutionValue
	keyword               Attr
	removeKeyword         bool
	removeLandwalk        bool
	removeAbilities       bool
	cantBeBlocked         bool
	cantBlock             bool
	assignsNoCombatDamage bool
	blockerFilter         *PermanentFilter
	requiredBlockerFilter *PermanentFilter
	blockEachAttacker     bool
	blockAnyNumber        bool
	setBasePT             bool
	setPowerOnly          bool
	setToughnessOnly      bool
	isSwitch              bool
	layer                 Layer
	colors                []Color
	family                SubtypeFamily
	subtypes              []string
	addSubtypes           bool
	addType               Attr
	typeAddition          bool
	ability               *ActionDefinition
	trigger               *GenericTriggered
	triggerGrant          bool
}

func (*layerModification) continuousModification()            {}
func (m *layerModification) modification() *layerModification { return m }

// ModifyPT adds power and toughness to one permanent.
func ModifyPT(subject PermanentRef, power, toughness ResolutionValue) ContinuousModification {
	return &layerModification{subject: subject, power: power, toughness: toughness, layer: LayerPT}
}

// ChangeController gives one permanent to the selected player until end of turn.
func ChangeController(subject PermanentRef, controller PlayerRef) ContinuousModification {
	return &layerModification{subject: subject, controller: controller, layer: LayerControl}
}

// SetBasePowerToughness sets both base values of one permanent.
func SetBasePowerToughness(subject PermanentRef, power, toughness ResolutionValue) ContinuousModification {
	return &layerModification{subject: subject, power: power, toughness: toughness, layer: LayerPT, setBasePT: true}
}

// SetEachBasePowerToughness sets base values for each member of a permanent set.
// Until saves the members and values when the effect begins.
func SetEachBasePowerToughness(subjects PermanentSet, power, toughness ResolutionValue) ContinuousModification {
	return &layerModification{subjects: subjects, power: power, toughness: toughness, layer: LayerPT, setBasePT: true}
}

// SetBasePower sets base power and preserves base toughness.
func SetBasePower(subject PermanentRef, power ResolutionValue) ContinuousModification {
	return &layerModification{subject: subject, power: power, toughness: Fixed(0), layer: LayerPT, setBasePT: true, setPowerOnly: true}
}

// SetBaseToughness sets base toughness and preserves base power.
func SetBaseToughness(subject PermanentRef, toughness ResolutionValue) ContinuousModification {
	return &layerModification{subject: subject, power: Fixed(0), toughness: toughness, layer: LayerPT, setBasePT: true, setToughnessOnly: true}
}

// SwitchPowerToughness switches power and toughness in Layer 7d (CR 613.4d).
func SwitchPowerToughness(subject PermanentRef) ContinuousModification {
	return &layerModification{subject: subject, layer: LayerPT, isSwitch: true}
}

// ModifyEachPT applies a power and toughness modification to a permanent set.
func ModifyEachPT(subjects PermanentSet, power, toughness ResolutionValue) ContinuousModification {
	return &layerModification{subjects: subjects, power: power, toughness: toughness, layer: LayerPT}
}

// GrantKeywordModification grants a keyword. DSL exports it as GrantKeyword.
func GrantKeywordModification(subject PermanentRef, keyword Attr) ContinuousModification {
	return &layerModification{subject: subject, keyword: keyword, layer: LayerAbility}
}

// AttackAsThoughHaste permits attacks without changing tap cost legality.
func AttackAsThoughHaste(subject PermanentRef) ContinuousModification {
	return &layerModification{subject: subject, keyword: AttrAttackAsThoughHaste, layer: LayerAbility}
}

// RemoveKeyword removes all instances of one keyword from a permanent in layer 6.
func RemoveKeyword(subject PermanentRef, keyword Attr) ContinuousModification {
	return &layerModification{subject: subject, keyword: keyword, removeKeyword: true, layer: LayerAbility}
}

// RemoveEachKeyword removes all instances of one keyword from each selected permanent.
// Until saves the set at resolution. While reads the current set in layer 6.
func RemoveEachKeyword(subjects PermanentSet, keyword Attr) ContinuousModification {
	return &layerModification{subjects: subjects, keyword: keyword, removeKeyword: true, layer: LayerAbility}
}

// RemoveLandwalk removes all landwalk abilities from a permanent in layer 6.
func RemoveLandwalk(subject PermanentRef) ContinuousModification {
	return &layerModification{subject: subject, removeLandwalk: true, layer: LayerAbility}
}

// RemoveAbilities removes all abilities from one permanent in layer 6.
func RemoveAbilities(subject PermanentRef) ContinuousModification {
	return &layerModification{subject: subject, removeAbilities: true, layer: LayerAbility}
}

// RemoveAbilitiesEach removes all abilities from each selected permanent in layer 6.
func RemoveAbilitiesEach(subjects PermanentSet) ContinuousModification {
	return &layerModification{subjects: subjects, removeAbilities: true, layer: LayerAbility}
}

// AssignsNoCombatDamage stops combat assignment without adding an ability.
// Use a duration to keep the rule after ability removal.
func AssignsNoCombatDamage(subject PermanentRef) ContinuousModification {
	return &layerModification{subject: subject, assignsNoCombatDamage: true, layer: LayerAbility}
}

// CantBlock restricts blocking without adding an ability to the subject.
func CantBlock(subject PermanentRef) ContinuousModification {
	return &layerModification{subject: subject, cantBlock: true, layer: LayerAbility}
}

// CantBeBlocked prevents all creatures from blocking the subject.
// Install this rules restriction with a duration; it does not grant an ability.
func CantBeBlocked(subject PermanentRef) ContinuousModification {
	return &layerModification{subject: subject, cantBeBlocked: true, layer: LayerAbility}
}

// CantBeBlockedExceptBy requires each blocker to pass the filter.
// Use this rule with a duration; it does not give the subject an ability.
func CantBeBlockedExceptBy(subject PermanentRef, filter PermanentFilter) ContinuousModification {
	return &layerModification{subject: subject, blockerFilter: &filter, layer: LayerAbility}
}

// AllCreaturesMustBlock requires each matching creature to block the subject if able.
// Install the rule with a duration. It does not give the subject an ability.
func AllCreaturesMustBlock(subject PermanentRef, filter PermanentFilter) ContinuousModification {
	return &layerModification{subject: subject, requiredBlockerFilter: &filter, layer: LayerAbility}
}

// BlockAnyNumber permits multiple blocks without granting an ability.
func BlockAnyNumber(subject PermanentRef) ContinuousModification {
	return &layerModification{subject: subject, blockAnyNumber: true, layer: LayerAbility}
}

// BlockEachAttacker requires the subject to block each attacker if able.
// Use a duration to keep the rule after ability removal.
func BlockEachAttacker(subject PermanentRef) ContinuousModification {
	return &layerModification{subject: subject, blockEachAttacker: true, layer: LayerAbility}
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

// TriggeredAbilityModification gives the recipient its own trigger context.
// Install it with a duration to retain the recipient's original identity.
func TriggeredAbilityModification(subject PermanentRef, ability *GenericTriggered) ContinuousModification {
	return &layerModification{subject: subject, trigger: ability, triggerGrant: true, layer: LayerAbility}
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

// AddCardType keeps existing types and abilities during a type addition.
func AddCardType(subject PermanentRef, kind CardType) ContinuousModification {
	return &layerModification{subject: subject, addType: CardTypeAttr(kind), typeAddition: true, layer: LayerType}
}

// AddEachCardType keeps prior types and abilities for each member of a set.
// Until saves the members before the type change.
func AddEachCardType(subjects PermanentSet, kind CardType) ContinuousModification {
	return &layerModification{subjects: subjects, addType: CardTypeAttr(kind), typeAddition: true, layer: LayerType}
}

func (m *layerModification) inspectDefinition() definitionNode {
	subject := any(m.subject)
	if m.subjects != nil {
		subject = m.subjects
	}
	n := definitionNode{Role: "continuous-modification", Text: m.description(), children: []definitionEdge{{"subject", subject}}}
	if m.typeAddition && m.addType == 0 {
		n.problem = "supply a supported permanent card type"
	}
	if m.triggerGrant {
		n.children = append(n.children, definitionEdge{"trigger", grantedTriggerDefinition{m.trigger}})
	}
	if m.ability != nil {
		n.children = append(n.children, definitionEdge{"ability", grantedActivationDefinition{m.ability}})
	}
	if m.layer == LayerPT {
		n.children = append(n.children, definitionEdge{"power", m.power}, definitionEdge{"toughness", m.toughness})
	}
	if m.layer == LayerControl {
		n.children = append(n.children, definitionEdge{"controller", m.controller})
	}
	return n
}
func (m *layerModification) description() string {
	switch m.layer {
	case LayerControl:
		return "change controller"
	case LayerType:
		if m.addType != 0 {
			return fmt.Sprintf("add card type %v", m.addType)
		}
		if m.addSubtypes {
			return fmt.Sprintf("add subtypes %v in family %v", m.subtypes, m.family)
		}
		return fmt.Sprintf("set subtypes %v in family %v", m.subtypes, m.family)
	case LayerColor:
		return fmt.Sprintf("set colors %v", m.colors)
	case LayerAbility:
		if m.assignsNoCombatDamage {
			return "assigns no combat damage"
		}
		if m.blockAnyNumber {
			return "can block any number of creatures"
		}
		if m.blockEachAttacker {
			return "blocks each attacker if able"
		}
		if m.requiredBlockerFilter != nil {
			return "must be blocked by each " + m.requiredBlockerFilter.Text()
		}
		if m.blockerFilter != nil {
			return "can’t be blocked except by " + m.blockerFilter.Text()
		}
		if m.cantBlock {
			return "can’t block"
		}
		if m.cantBeBlocked {
			return "can’t be blocked"
		}
		if m.trigger != nil {
			return "grant triggered ability"
		}
		if m.ability != nil {
			return "grant activated ability"
		}
		if m.removeLandwalk {
			return "remove all landwalk"
		}
		if m.removeAbilities {
			return "remove all abilities"
		}
		if m.removeKeyword {
			return fmt.Sprintf("remove %v", m.keyword)
		}
		return fmt.Sprintf("grant %v", m.keyword)
	default:
		if m.setToughnessOnly {
			return "set base toughness"
		}
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
	permanent             capturedPermanent
	controller            uuid.UUID
	power, toughness      int
	keyword               Attr
	removeKeyword         bool
	removeLandwalk        bool
	removeAbilities       bool
	cantBeBlocked         bool
	cantBlock             bool
	assignsNoCombatDamage bool
	blockerFilter         *PermanentFilter
	requiredBlockerFilter *PermanentFilter
	blockEachAttacker     bool
	blockAnyNumber        bool
	setBasePT             bool
	setPowerOnly          bool
	setToughnessOnly      bool
	isSwitch              bool
	colors                []Color
	family                SubtypeFamily
	subtypes              []string
	addSubtypes           bool
	addType               Attr
	ability               *ActionDefinition
	trigger               *GenericTriggered
	grantID               uuid.UUID
}

func (m *layerModification) capture(ctx *EffectContext) []capturedModification {
	members := m.members(ctx)
	result := make([]capturedModification, 0, len(members))
	for _, p := range members {
		v := capturedModification{permanent: capturedPermanent{p.ID(), p.incarnationID}, keyword: m.keyword, removeKeyword: m.removeKeyword, removeLandwalk: m.removeLandwalk, removeAbilities: m.removeAbilities, cantBeBlocked: m.cantBeBlocked, cantBlock: m.cantBlock, assignsNoCombatDamage: m.assignsNoCombatDamage, blockerFilter: m.blockerFilter, requiredBlockerFilter: m.requiredBlockerFilter, blockEachAttacker: m.blockEachAttacker, blockAnyNumber: m.blockAnyNumber, setBasePT: m.setBasePT, setPowerOnly: m.setPowerOnly, setToughnessOnly: m.setToughnessOnly, isSwitch: m.isSwitch, colors: m.colors, family: m.family, subtypes: m.subtypes, addSubtypes: m.addSubtypes, addType: m.addType}
		if m.layer == LayerControl {
			controller, ok := m.controller.resolvePlayer(ctx)
			if !ok {
				continue
			}
			v.controller = controller
		}
		if m.ability != nil || m.trigger != nil {
			v.ability, v.trigger, v.grantID = m.ability, m.trigger, uuid.New()
		}
		if m.layer == LayerPT {
			if !m.isSwitch {
				v.power = m.power.resolveValue(ctx)
				v.toughness = m.toughness.resolveValue(ctx)
			}
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
	case LayerControl:
		setComputedController(p, v.controller)
	case LayerPT:
		if v.setBasePT {
			power, toughness := v.power, v.toughness
			if v.setPowerOnly {
				toughness = p.Card.Toughness()
				if p.cdaToughness != nil {
					toughness = *p.cdaToughness
				}
				if p.BasePTOverride != nil {
					toughness = p.BasePTOverride[1]
				}
			}
			if v.setToughnessOnly {
				power = p.Card.Power()
				if p.cdaPower != nil {
					power = *p.cdaPower
				}
				if p.BasePTOverride != nil {
					power = p.BasePTOverride[0]
				}
			}
			p.BasePTOverride = &[2]int{power, toughness}
		} else if v.isSwitch {
			p.SwitchPowerToughness()
		} else {
			p.powerBonus += v.power
			p.toughBonus += v.toughness
		}
	case LayerAbility:
		if v.assignsNoCombatDamage {
			if g.effects.noCombatAssignmentRules == nil {
				g.effects.noCombatAssignmentRules = make(map[uuid.UUID]uuid.UUID)
			}
			g.effects.noCombatAssignmentRules[p.ID()] = p.incarnationID
		} else if v.blockAnyNumber {
			if g.effects.blockAnyNumberRules == nil {
				g.effects.blockAnyNumberRules = make(map[uuid.UUID]bool)
			}
			g.effects.blockAnyNumberRules[p.ID()] = true
		} else if v.blockEachAttacker {
			if g.effects.blockEachAttackerRules == nil {
				g.effects.blockEachAttackerRules = make(map[uuid.UUID]int)
			}
			g.effects.blockEachAttackerRules[p.ID()]++
		} else if v.requiredBlockerFilter != nil {
			g.effects.AddAllCreaturesMustBlock(p.ID(), *v.requiredBlockerFilter)
		} else if v.cantBlock {
			g.effects.AddCanBlockOnly(p.ID(), Not(AnyPermanent))
		} else if v.blockerFilter != nil {
			g.effects.AddCantBeBlockedExceptBy(p.ID(), *v.blockerFilter)
		} else if v.cantBeBlocked {
			g.effects.AddCantBeBlockedExceptBy(p.ID(), Not(AnyPermanent))
		} else if v.trigger != nil {
			installGrantedTrigger(p, v.trigger, v.grantID)
		} else if v.ability != nil {
			installGrantedActivation(p, v.ability, v.grantID)
		} else if v.removeLandwalk {
			for _, keyword := range landwalkKeywords() {
				g.effects.RevokeAttr(p.ID(), keyword)
			}
		} else if v.removeAbilities {
			removeAllAbilitiesAtCurrentLayer(g, p)
		} else if v.removeKeyword {
			g.effects.RevokeAttr(p.ID(), v.keyword)
		} else {
			g.effects.GrantAttr(p.ID(), v.keyword)
		}
	case LayerColor:
		colors := append([]Color(nil), v.colors...)
		p.ColorOverride = &colors
	case LayerType:
		if v.addType != 0 {
			g.effects.GrantAttr(p.ID(), v.addType)
			if v.addType == AttrIsCreature {
				g.effects.GrantAttr(p.ID(), AttrHasPowerToughness)
			}
		} else if v.addSubtypes {
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
		if !nilDefinition(m) && m.modification().layer == LayerControl {
			n.problem = "control modification requires Until(EndOfTurn)"
		}
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
	for _, layer := range []Layer{LayerControl, LayerText, LayerType, LayerColor, LayerAbility, LayerPT} {
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
		if layer == LayerPT {
			sublayerMods := make(map[SublayerPT][]*layerModification)
			for _, m := range mods {
				sub := SublayerPT_Modify
				if m.setBasePT {
					sub = SublayerPT_Set
				} else if m.isSwitch {
					sub = SublayerPT_Switch
				}
				sublayerMods[sub] = append(sublayerMods[sub], m)
			}
			for _, sub := range []SublayerPT{SublayerPT_CDA, SublayerPT_Set, SublayerPT_Modify, SublayerPT_Switch} {
				if sm := sublayerMods[sub]; len(sm) > 0 {
					effects = append(effects, &composedLayerEffect{layer: LayerPT, sublayer: sub, duration: WhileOnBattlefield, rule: r, modifications: sm, grantID: uuid.New()})
				}
			}
		} else if len(mods) > 0 {
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
	sublayer          SublayerPT
	duration          Duration
	rule              *continuousRule
	modifications     []*layerModification
	captured          []capturedModification
	sourceBound       bool
	sourceTapped      bool
	expired           bool
	sourceIncarnation uuid.UUID
	grantID           uuid.UUID
	upkeepPlayer      uuid.UUID
	endNextUpkeep     bool
	upkeepReady       bool
}

func (e *composedLayerEffect) Text() string { return "continuous modification" }
func (e *composedLayerEffect) Properties() EffectProperties {
	return propertiesFromModifications(e.modifications)
}
func (e *composedLayerEffect) GetLayer() Layer { return e.layer }
func (e *composedLayerEffect) GetSublayer() SublayerPT {
	if e.sublayer == 0 {
		return SublayerPT_Modify
	}
	return e.sublayer
}
func (e *composedLayerEffect) GetDuration() Duration { return e.duration }
func (e *composedLayerEffect) survivesSourceExit() bool {
	return e.rule == nil && !e.sourceBound
}
func (e *composedLayerEffect) IsActive(g *Game) bool {
	if e.sourceTapped {
		p := g.FindPermanent(e.sourceID)
		if p == nil || p.incarnationID != e.sourceIncarnation || !p.Tapped {
			e.expired = true
		}
		return !e.expired
	}
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
func (e *composedLayerEffect) removesAbilities() bool {
	for _, m := range e.modifications {
		if m.removeAbilities {
			return true
		}
	}
	for _, m := range e.captured {
		if m.removeAbilities {
			return true
		}
	}
	return false
}

func (e *composedLayerEffect) ApplyContinuous(ctx *EffectContext) error {
	if e.rule == nil {
		if e.layer == LayerType {
			// Subtype checks need the added card types from the same effect.
			for _, v := range e.captured {
				if v.addType != 0 {
					v.apply(ctx.Game, e.layer)
				}
			}
			ctx.Game.effects.syncAttrDeltas(ctx.Game)
		}
		for _, v := range e.captured {
			if e.layer != LayerType || v.addType == 0 {
				v.apply(ctx.Game, e.layer)
			}
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
	if e.rule.condition.checkState(ctx.Game, StateContext{SourceID: e.sourceID, ControllerID: child.Controller, SourceIncarnation: source.incarnationID}) {
		for i, m := range e.modifications {
			for _, v := range m.capture(&child) {
				if v.ability != nil || v.trigger != nil {
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
	sourceTapped  bool
	upkeepPlayer  PlayerRef
	endNextUpkeep bool
}

// UntilNextUpkeep saves the player so source changes cannot change the duration.
func UntilNextUpkeep(player PlayerRef, modifications ...ContinuousModification) ResolvingEffect {
	return &untilEffect{duration: UntilYourNextTurn, upkeepPlayer: player, modifications: append([]ContinuousModification(nil), modifications...)}
}

// UntilEndOfNextUpkeep keeps the saved values through the next upkeep's priority.
func UntilEndOfNextUpkeep(player PlayerRef, modifications ...ContinuousModification) ResolvingEffect {
	return &untilEffect{duration: UntilYourNextTurn, upkeepPlayer: player, endNextUpkeep: true, modifications: append([]ContinuousModification(nil), modifications...)}
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

// UntilSourceUntaps captures modifications while this source remains tapped.
// Untapping, source phasing, and source departure end the duration permanently.
func UntilSourceUntaps(_ sourceRef, modifications ...ContinuousModification) ResolvingEffect {
	return &untilEffect{duration: Indefinite, modifications: append([]ContinuousModification(nil), modifications...), sourceBound: true, sourceTapped: true}
}

func (e *untilEffect) Text() string {
	var descriptions []string
	for _, m := range e.modifications {
		if !nilDefinition(m) {
			descriptions = append(descriptions, m.modification().description())
		}
	}
	body := strings.Join(descriptions, "; ")
	if e.sourceTapped {
		return body + " for as long as this source remains tapped"
	}
	if e.sourceBound {
		return body + " until this source leaves the battlefield"
	}
	switch e.duration {
	case EndOfTurn:
		return body + " until end of turn"
	case EndOfCombat:
		return body + " until end of combat"
	case UntilYourNextTurn:
		if e.endNextUpkeep {
			return body + " until the end of the selected player's next upkeep"
		}
		return body + " until the selected player's next upkeep"
	default:
		return body + " indefinitely"
	}
}
func propertiesFromModifications(modifications []*layerModification) EffectProperties {
	p := EffectProperties{Outcome: OutcomeBenefit}
	for _, m := range modifications {
		if m == nil {
			continue
		}
		if m.layer == LayerControl {
			p.Outcome = OutcomeDetriment
		}
		if m.setBasePT {
			p.Outcome = OutcomeDetriment
		}
		if v, ok := m.power.(fixedValue); ok && !m.setBasePT {
			p.PowerBoost += v.n
		}
		if v, ok := m.toughness.(fixedValue); ok && !m.setBasePT {
			p.ToughnessBoost += v.n
		}
		if m.removeKeyword || m.removeLandwalk || m.removeAbilities {
			p.Outcome = OutcomeDetriment
		} else if m.keyword != 0 {
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

func (e *untilEffect) Properties() EffectProperties {
	var mods []*layerModification
	for _, value := range e.modifications {
		if !nilDefinition(value) {
			mods = append(mods, value.modification())
		}
	}
	return propertiesFromModifications(mods)
}

// KeywordGrantTargetIndex identifies one pure target keyword grant for AI checks.
// Unknown target boundaries and compound modifications require full evaluation.
func (e *untilEffect) KeywordGrantTargetIndex(targets []Target) (int, bool) {
	if e.sourceBound || e.duration != EndOfTurn || len(e.modifications) != 1 || nilDefinition(e.modifications[0]) {
		return 0, false
	}
	m := e.modifications[0].modification()
	if m.layer != LayerAbility || m.keyword == 0 || m.removeKeyword || m.subjects != nil {
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
		if e.sourceTapped {
			n.Role = "until-source-untaps"
		}
		n.children = append(n.children, definitionEdge{"source", This})
	}
	if len(e.modifications) == 0 {
		n.problem = "supply at least one modification"
	}
	if e.duration != EndOfTurn && e.duration != EndOfCombat && e.duration != Indefinite {
		if e.duration == UntilYourNextTurn && !nilDefinition(e.upkeepPlayer) {
			n.children = append(n.children, definitionEdge{"upkeep-player", e.upkeepPlayer})
		} else {
			n.problem = "duration requires an adapter with an explicit lifetime policy"
		}
	}
	for i, m := range e.modifications {
		n.children = append(n.children, definitionEdge{fmt.Sprintf("modifications[%d]", i), m})
		if !nilDefinition(m) && m.modification().layer == LayerControl && (e.sourceBound || e.duration != EndOfTurn) {
			n.problem = "control modification requires Until(EndOfTurn)"
		}
	}
	return n
}
func (e *untilEffect) Apply(ctx *EffectContext) error {
	var upkeepPlayer uuid.UUID
	if e.duration == UntilYourNextTurn {
		var ok bool
		upkeepPlayer, ok = e.upkeepPlayer.resolvePlayer(ctx)
		if !ok {
			return nil
		}
	}
	var sourceIncarnation uuid.UUID
	if e.sourceBound {
		source := ctx.Game.FindPermanentIncludingPhased(ctx.SourceID)
		if source == nil || (ctx.sourceCaptured && source.incarnationID != ctx.sourceIncarnation) {
			return nil
		}
		if e.sourceTapped && (source.PhasedOut || !source.Tapped) {
			return nil
		}
		sourceIncarnation = source.incarnationID
	}
	var effects []*composedLayerEffect
	for _, layer := range []Layer{LayerControl, LayerText, LayerType, LayerColor, LayerAbility, LayerPT} {
		if layer == LayerPT {
			sublayerCaps := make(map[SublayerPT][]capturedModification)
			for _, value := range e.modifications {
				m := value.modification()
				if m.layer == LayerPT {
					sub := SublayerPT_Modify
					if m.setBasePT {
						sub = SublayerPT_Set
					} else if m.isSwitch {
						sub = SublayerPT_Switch
					}
					sublayerCaps[sub] = append(sublayerCaps[sub], m.capture(ctx)...)
				}
			}
			for _, sub := range []SublayerPT{SublayerPT_CDA, SublayerPT_Set, SublayerPT_Modify, SublayerPT_Switch} {
				if caps := sublayerCaps[sub]; len(caps) > 0 {
					entry := &composedLayerEffect{layer: LayerPT, sublayer: sub, duration: e.duration, sourceBound: e.sourceBound, sourceTapped: e.sourceTapped, sourceIncarnation: sourceIncarnation, captured: caps}
					entry.SetSourceID(ctx.SourceID)
					effects = append(effects, entry)
				}
			}
			continue
		}
		entry := &composedLayerEffect{layer: layer, duration: e.duration, sourceBound: e.sourceBound, sourceTapped: e.sourceTapped, sourceIncarnation: sourceIncarnation}
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
		entry.upkeepPlayer = upkeepPlayer
		entry.endNextUpkeep = e.endNextUpkeep
		ctx.Game.effects.Add(entry)
	}
	ctx.Game.ApplyContinuousEffects()
	return nil
}

// --- from composable_land_plays.go ---
type grantLandPlaysEffect struct {
	subject PlayerRef
	amount  ResolutionValue
}

// GrantLandPlays adds to the selected player's land allowance for this turn.
// It does not play a land or change when the player can play lands.
// An absent player or a nonpositive amount has no effect.
func GrantLandPlays(subject PlayerRef, amount ResolutionValue) ResolvingEffect {
	return &grantLandPlaysEffect{subject, amount}
}

func (e *grantLandPlaysEffect) Text() string {
	return "grant additional land plays this turn to " + InspectDefinition(e.subject).Text
}
func (*grantLandPlaysEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *grantLandPlaysEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
}
func (e *grantLandPlaysEffect) Apply(ctx *EffectContext) error {
	id, ok := e.subject.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	ctx.Game.GrantExtraLandPlay(id, e.amount.resolveValue(ctx))
	return nil
}

// --- from composable_skip_untap.go ---
// SkipNextUntap prevents untapping during the next untap step of the subject's
// current controller. It does not tap the subject or prevent other untap effects.
func SkipNextUntap(subject PermanentRef) ResolvingEffect {
	return &skipNextUntapEffect{subject}
}

type skipNextUntapEffect struct{ subject PermanentRef }

func (e *skipNextUntapEffect) Text() string {
	return InspectDefinition(e.subject).Text + " doesn't untap during its controller's next untap step"
}
func (*skipNextUntapEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *skipNextUntapEffect) Apply(ctx *EffectContext) error {
	if p, ok := e.subject.resolvePermanent(ctx); ok {
		ctx.Game.SkipNextUntap(p.ID())
	}
	return nil
}
func (e *skipNextUntapEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
}

// SkipNextUntapOf prevents untapping during the selected player's next untap
// step. It captures the player at resolution, even if control changes later.
func SkipNextUntapOf(subject PermanentRef, player PlayerRef) ResolvingEffect {
	return &capturedUntapEffect{subject: subject, player: player}
}

type capturedUntapEffect struct {
	subject PermanentRef
	player  PlayerRef
}

func (e *capturedUntapEffect) Text() string {
	return InspectDefinition(e.subject).Text + " doesn't untap during the next untap step of " + InspectDefinition(e.player).Text
}
func (*capturedUntapEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *capturedUntapEffect) Apply(ctx *EffectContext) error {
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	player, ok := e.player.resolvePlayer(ctx)
	if ok {
		ctx.Game.turns.skipNextUntapOf(p.ID(), player)
	}
	return nil
}
func (e *capturedUntapEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"player", e.player}}}
}
