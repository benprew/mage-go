package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// CopyPermanent replaces one present visit's copiable characteristics.
// The recipient keeps its status, counters, damage and physical card.
func CopyPermanent(recipient, subject PermanentRef, keepColors bool) ResolvingEffect {
	return &resolvingCopyEffect{recipient: recipient, subject: subject, keepColors: keepColors}
}

// MayCopyPermanent lets the saved player choose a copy at resolution.
// Both exact visits must be present before the choice is offered.
func MayCopyPermanent(player PlayerRef, recipient, subject PermanentRef, keepColors bool) ResolvingEffect {
	return &resolvingCopyEffect{recipient: recipient, subject: subject, keepColors: keepColors, optional: true, player: player}
}

type resolvingCopyEffect struct {
	recipient, subject PermanentRef
	keepColors         bool
	optional           bool
	player             PlayerRef
}

func (e *resolvingCopyEffect) Text() string {
	if e.optional {
		return "may become a copy of a permanent"
	}
	return "become a copy of a permanent"
}
func (*resolvingCopyEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *resolvingCopyEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-copy", Text: e.Text(), children: []definitionEdge{{"recipient", e.recipient}, {"subject", e.subject}}}
	if e.optional {
		n.children = append(n.children, definitionEdge{"player", e.player})
	}
	return n
}
func (e *resolvingCopyEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.recipient) || permanentRefReadsSourcePermanent(e.subject) || playerRefReadsSourcePermanent(e.player)
}
func (e *resolvingCopyEffect) Apply(ctx *EffectContext) error {
	recipient, ok := e.recipient.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	subject, ok := e.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	if e.optional {
		id, ok := e.player.resolvePlayer(ctx)
		if !ok {
			return nil
		}
		player := ctx.Game.GetPlayer(id)
		if player == nil || !player.ChooseMayAbility(e.Text()) {
			return nil
		}
	}
	return ctx.Game.BecomeCopyOfPermanent(recipient, subject, e.keepColors)
}

type resolvingCopyState struct {
	values           *CopiableValues
	intrinsicChanges []TextChange
	appliedChanges   []TextChange
	variants         []resolvingCopyTextVariant
}

// BecomeCopyOfPermanent captures the subject before it replaces recipient rules.
// No entry event or entry replacement occurs. Later copies read the saved values.
func (g *Game) BecomeCopyOfPermanent(recipient, subject *Permanent, keepColors bool) error {
	if recipient == nil || subject == nil {
		return fmt.Errorf("resolving copy requires two present visits")
	}
	current := g.FindPermanent(recipient.ID())
	if current == nil || current.incarnationID != recipient.incarnationID || current.PhasedOut {
		return fmt.Errorf("resolving copy requires the original recipient visit")
	}
	values, err := g.CaptureCopiableValues(subject)
	if err != nil {
		return err
	}
	if keepColors {
		own, err := g.CaptureCopiableValues(current)
		if err != nil {
			return err
		}
		prototype, err := own.Build()
		if err != nil {
			return err
		}
		values, err = values.WithModifications(CopyRetainColors(cardColors(prototype)...))
		if err != nil {
			return err
		}
	}
	state := &resolvingCopyState{values: values}
	if current.resolvingCopy != nil {
		state.intrinsicChanges = slices.Clone(current.resolvingCopy.intrinsicChanges)
	} else if base, ok := current.Card.(*BaseCard); ok && base.textDefinition != nil {
		state.intrinsicChanges = slices.Clone(base.textDefinition.changes)
	}
	if err := g.prepareResolvingCopyText(current, state, nil); err != nil {
		return err
	}
	changes := g.resolvingCopyTextChanges(current, state.intrinsicChanges)
	card := state.variants[0].card
	state.appliedChanges = changes
	g.replaceResolvingCopyRules(current, card, false)
	current = g.MutablePermanent(current.ID())
	current.resolvingCopy = state
	g.effects.Apply(g)
	return nil
}

func (g *Game) replaceResolvingCopyRules(subject *Permanent, card Card, retainTimestamp bool) {
	p := g.MutablePermanent(subject.ID())
	old, fresh := NewPermanent(p.Card, p.ControllerID()), NewPermanent(card, p.ControllerID())
	abilities := slices.Clone(fresh.RuntimeAbilities)
	for _, ability := range p.RuntimeAbilities {
		if _, granted := ability.(*grantedByEffect); granted {
			abilities = append(abilities, ability)
		}
	}
	for attr := range p.baseAttrs {
		p.baseAttrs[attr] += fresh.baseAttrs[attr] - old.baseAttrs[attr] + old.printedAbilityAttrs[attr] - p.printedAbilityAttrs[attr]
	}
	p.Card = card
	p.RuntimeAbilities = abilities
	if p.FaceDown {
		p.BasePTOverride = &[2]int{2, 2}
		p.RuntimeAbilities = nil
	}
	p.printedAbilityAttrs = fresh.printedAbilityAttrs
	p.revisedPrintedActions, p.layeredActivatedOrigins = nil, nil
	p.revisedPrintedTriggers, p.revisedPrintedManaBonuses = nil, nil
	p.revisedPrintedProtections, p.revisedPrintedLandwalks = nil, nil
	p.layeredLandwalkOrigins, p.layeredProtectionOrigins = nil, nil
	p.layeredManaTapOrigins, p.layeredSpellCastOrigins = nil, nil
	p.layeredUpkeepOrigins, p.layeredCombatPairOrigins = nil, nil
	p.layeredLandTapOrigins, p.layeredStaticAbilityOrigins = nil, nil
	effects := make([]ContinuousEffect, 0, len(g.effects.effects))
	position := -1
	for _, effect := range g.effects.effects {
		printed, ok := effect.(*printedAbilityContinuousEffect)
		if ok && !printed.acquired && effect.SourceID() == p.ID() {
			if position < 0 {
				position = len(effects)
			}
			continue
		}
		effects = append(effects, effect)
	}
	g.effects.effects = effects
	firstAdded := len(effects)
	g.syncAbilityContext(p)
	for _, ability := range p.RuntimeAbilities {
		if holder, ok := ability.(*StaticAbilityHolder); ok {
			for _, effect := range holder.Effects {
				effect = g.bindCharacteristicEffect(effect, p)
				g.setEffectSource(effect, p.ID())
				g.effects.Add(printedStaticEffect(effect, p, holder))
			}
		}
	}
	if retainTimestamp && position >= 0 {
		added := slices.Clone(g.effects.effects[firstAdded:])
		g.effects.effects = slices.Insert(g.effects.effects[:firstAdded], position, added...)
	}
}

// Text effects keep their object and timestamp when copy rules replace abilities.
type resolvingCopyTextEffect struct {
	effectSource
	subject  capturedPermanent
	duration core.Duration
	changes  []TextChange
}

func (*resolvingCopyTextEffect) Text() string {
	return "continue permanent text changes through copying"
}
func (*resolvingCopyTextEffect) Properties() EffectProperties { return EffectProperties{} }
func (*resolvingCopyTextEffect) GetLayer() core.Layer         { return core.LayerText }
func (e *resolvingCopyTextEffect) GetDuration() core.Duration { return e.duration }
func (e *resolvingCopyTextEffect) IsActive(g *Game) bool {
	p := g.FindPermanent(e.subject.id)
	return p != nil && p.incarnationID == e.subject.incarnation && !p.PhasedOut
}
func (*resolvingCopyTextEffect) ApplyContinuous(*EffectContext) error { return nil }

func copyTextOperands(effect ContinuousEffect) (capturedPermanent, []TextChange, bool) {
	switch e := effect.(type) {
	case *resolvingCopyTextEffect:
		return e.subject, e.changes, true
	case *printedLandwalkTextEffect:
		return e.subject, e.changes, true
	case *printedProtectionTextEffect:
		return e.subject, e.changes, true
	case *printedStaticAbilityTextEffect:
		return e.subject, e.changes, !e.copyTextOmitted
	case *printedManaTapTextEffect:
		return e.subject, e.changes, !e.copyTextOmitted
	case *printedSpellCastTextEffect:
		return e.subject, e.changes, true
	case *printedUpkeepTextEffect:
		return e.subject, e.changes, true
	case *printedCombatPairTextEffect:
		return e.subject, e.changes, true
	case *printedLandTapTextEffect:
		return e.subject, e.changes, true
	case *printedActivatedTextEffect:
		return e.subject, e.changes, true
	}
	return capturedPermanent{}, nil, false
}

func (g *Game) resolvingCopyTextChanges(p *Permanent, intrinsic []TextChange) []TextChange {
	changes := slices.Clone(intrinsic)
	for _, effect := range g.effects.effects {
		subject, words, ok := copyTextOperands(effect)
		if ok && subject.id == p.ID() && subject.incarnation == p.incarnationID {
			changes = append(changes, words...)
		}
	}
	return changes
}

func (g *Game) refreshResolvingCopyText() {
	for _, p := range g.zones.battlefield {
		if p.resolvingCopy == nil || p.PhasedOut {
			continue
		}
		changes := g.resolvingCopyTextChanges(p, p.resolvingCopy.intrinsicChanges)
		if !slices.Equal(changes, p.resolvingCopy.appliedChanges) {
			for _, variant := range p.resolvingCopy.variants {
				if !slices.Equal(variant.changes, changes) {
					continue
				}
				state := *p.resolvingCopy
				state.appliedChanges = changes
				g.replaceResolvingCopyRules(p, variant.card, true)
				p = g.MutablePermanent(p.ID())
				p.resolvingCopy = &state
				break
			}
		}
		for i, effect := range g.effects.effects {
			subject, changes, ok := copyTextOperands(effect)
			if !ok || subject.id != p.ID() || subject.incarnation != p.incarnationID {
				continue
			}
			if _, done := effect.(*resolvingCopyTextEffect); done {
				continue
			}
			replacement := &resolvingCopyTextEffect{subject: subject, duration: effect.GetDuration(), changes: slices.Clone(changes)}
			replacement.SetSourceID(effect.SourceID())
			g.effects.effects[i] = replacement
		}
	}
}

type resolvingCopyTextVariant struct {
	changes []TextChange
	card    Card
}

func (g *Game) prepareResolvingCopyText(p *Permanent, state *resolvingCopyState, added ContinuousEffect) error {
	full := g.resolvingCopyTextChanges(p, state.intrinsicChanges)
	expired := slices.Clone(state.intrinsicChanges)
	effects := slices.Clone(g.effects.effects)
	if added != nil {
		effects = append(effects, added)
		_, words, _ := copyTextOperands(added)
		full = append(full, words...)
	}
	for _, effect := range effects {
		subject, words, ok := copyTextOperands(effect)
		if !ok || subject.id != p.ID() || subject.incarnation != p.incarnationID {
			continue
		}
		if effect.GetDuration() != core.EndOfTurn && effect.GetDuration() != core.Indefinite {
			return fmt.Errorf("resolving copy text requires indefinite or end-of-turn duration")
		}
		if effect.GetDuration() == core.Indefinite {
			expired = append(expired, words...)
		}
	}
	variants := make([]resolvingCopyTextVariant, 0, 2)
	for _, changes := range [][]TextChange{full, expired} {
		card, err := BuildEntryCopyWithText(p.Card, state.values, changes...)
		if err != nil {
			return fmt.Errorf("resolving copy text definition: %w", err)
		}
		variants = append(variants, resolvingCopyTextVariant{changes: changes, card: card})
	}
	state.variants = variants
	return nil
}

func (g *Game) prepareAddedResolvingCopyText(p *Permanent, effect ContinuousEffect) error {
	if p.resolvingCopy == nil {
		return nil
	}
	state := *p.resolvingCopy
	if err := g.prepareResolvingCopyText(p, &state, effect); err != nil {
		return err
	}
	g.MutablePermanent(p.ID()).resolvingCopy = &state
	return nil
}
