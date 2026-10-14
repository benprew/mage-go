package mage

import (
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type permanentManaColorsPlan struct{ filter PermanentFilter }

// NewManaAbilityOfColorsOpponentPermanentsCouldProduce creates a tap ability.
// It adds one mana of a color that a matching opponent permanent could produce.
// Payment and tapped status do not limit the opponent's production options.
func NewManaAbilityOfColorsOpponentPermanentsCouldProduce(filter PermanentFilter) *ManaAbility {
	ability := NewManaAbility(Colorless)
	ability.Productions = nil
	ability.permanentColors = &permanentManaColorsPlan{filter: filter}
	return ability
}

type manaCouldProduceKey struct{ source, ability uuid.UUID }

type manaCouldProduceQuery struct {
	game            *Game
	typesByAbility  map[manaCouldProduceKey][]Color
	solving, solved bool
}

func newManaCouldProduceQuery(g *Game) *manaCouldProduceQuery {
	return &manaCouldProduceQuery{game: g, typesByAbility: make(map[manaCouldProduceKey][]Color)}
}

func (p *permanentManaColorsPlan) productions(g GameReader, sourceID uuid.UUID) []ManaProduction {
	game, ok := g.(*Game)
	if !ok {
		return nil
	}
	return newManaCouldProduceQuery(game).opponentProductions(p, sourceID)
}

func (q *manaCouldProduceQuery) opponentProductions(plan *permanentManaColorsPlan, sourceID uuid.UUID) []ManaProduction {
	source := q.game.FindPermanent(sourceID)
	if source == nil || source.PhasedOut {
		return nil
	}
	opponent := q.game.GetOpponent(source.ControllerID())
	if opponent == nil {
		return nil
	}
	production := ManaProduction{Color: AnyColor, Amount: 1}
	for _, permanent := range q.game.zones.battlefield {
		if permanent.PhasedOut || permanent.ControllerID() != opponent.PlayerID() || !plan.filter.Match(permanent, q.game) {
			continue
		}
		for _, ability := range permanent.RuntimeAbilities {
			for _, color := range q.types(ability, permanent.ID()) {
				if color > Colorless && color < AnyColor {
					production.colorOptions[color] = true
				}
			}
		}
	}
	if production.colorOptions == ([AnyColor]bool{}) {
		return []ManaProduction{}
	}
	return []ManaProduction{production}
}

func (q *manaCouldProduceQuery) types(ability Ability, sourceID uuid.UUID) []Color {
	p := q.game.FindPermanent(sourceID)
	if p == nil || p.PhasedOut || nilDefinition(ability) {
		return nil
	}
	ability = UnwrapAbility(ability)
	if nilDefinition(ability) {
		return nil
	}
	key := manaCouldProduceKey{source: p.incarnationID, ability: ability.AbilityID()}
	if !q.solving && !q.solved {
		q.solve(ability, sourceID)
	}
	return q.typesByAbility[key]
}

func (q *manaCouldProduceQuery) solve(root Ability, sourceID uuid.UUID) {
	type querySubject struct {
		ability  Ability
		sourceID uuid.UUID
	}
	queue := []querySubject{{root, sourceID}}
	for _, p := range q.game.zones.battlefield {
		if p.PhasedOut {
			continue
		}
		for _, ability := range p.RuntimeAbilities {
			queue = append(queue, querySubject{ability, p.ID()})
		}
	}
	q.solving = true
	defer func() { q.solving = false; q.solved = true }()
	// Defined colors can supply a cycle. A cycle cannot supply its first color.
	for {
		changed := false
		for _, subject := range queue {
			p := q.game.FindPermanent(subject.sourceID)
			if p == nil || nilDefinition(subject.ability) {
				continue
			}
			ability := UnwrapAbility(subject.ability)
			if nilDefinition(ability) {
				continue
			}
			key := manaCouldProduceKey{source: p.incarnationID, ability: ability.AbilityID()}
			for _, color := range q.computeTypes(ability, p) {
				if !slices.Contains(q.typesByAbility[key], color) {
					q.typesByAbility[key] = append(q.typesByAbility[key], color)
					changed = true
				}
			}
		}
		if !changed {
			return
		}
	}
}

func (q *manaCouldProduceQuery) computeTypes(ability Ability, p *Permanent) []Color {
	sourceID := p.ID()
	var productions []ManaProduction
	tapping := false
	switch a := ability.(type) {
	case *ManaAbility:
		tapping = true
		if a.permanentColors != nil {
			productions = q.opponentProductions(a.permanentColors, sourceID)
		} else {
			productions = a.currentProductions(q.game, sourceID)
		}
	case *SimpleActivatedAbility:
		if !IsActivatedManaAbility(a) {
			return nil
		}
		productions = couldProduceManaEffects(a.effects, &EffectContext{Game: q.game, SourceID: sourceID, Controller: p.ControllerID(), sourceIncarnation: p.incarnationID}, a.costs)
		for _, cost := range a.costs {
			if _, ok := cost.(*tap); ok {
				tapping = true
			}
		}
	case *GenericTriggered:
		if !isTriggeredManaAbility(a) {
			return nil
		}
		productions = couldProduceManaEffects(a.Effects(), &EffectContext{Game: q.game, SourceID: sourceID, Controller: p.ControllerID(), sourceIncarnation: p.incarnationID}, nil)
	}
	if len(productions) == 0 {
		return nil
	}
	outcomes := []manaReplacementOutcome{{productions: productions}}
	if tapping {
		outcomes = q.game.manaProductionVariants(&manaProductionAction{source: sourceID, incarnation: p.incarnationID, player: p.ControllerID(), tapping: true, productions: productions})
	}
	var types []Color
	for _, outcome := range outcomes {
		for _, prod := range outcome.productions {
			for _, color := range manaProductionColors(prod) {
				types = appendProducedColor(types, color)
			}
		}
	}
	return types
}

func couldProduceManaAmount(value ResolutionValue, ctx *EffectContext, costs []Cost) int {
	if saved, ok := value.(savedNumberValue); ok {
		for _, cost := range costs {
			if payment, ok := cost.(*removeAnyCountersCost); ok && payment.paid == saved.binding {
				// One hypothetical removal defines the color without testing payment.
				return 1
			}
		}
		return 0
	}
	return value.resolveValue(ctx)
}

func couldProduceManaEffects(effects []Effect, ctx *EffectContext, costs []Cost) []ManaProduction {
	var result []ManaProduction
	for _, effect := range effects {
		switch e := effect.(type) {
		case *manaAmountEffect:
			if amount := couldProduceManaAmount(e.amount, ctx, costs); amount > 0 {
				result = append(result, ManaProduction{Color: e.color, Amount: amount})
			}
		case *manaAnyOneColorEffect:
			if amount := couldProduceManaAmount(e.amount, ctx, costs); amount > 0 {
				result = append(result, ManaProduction{Color: AnyColor, Amount: amount, Restriction: e.restriction})
			}
		case *addManaEffect:
			if e.amount > 0 {
				result = append(result, ManaProduction{Color: e.color, Amount: e.amount})
			}
		case *addAnyManaEffect:
			if e.amount > 0 {
				result = append(result, ManaProduction{Color: AnyColor, Amount: e.amount, Restriction: e.restriction})
			}
		case *restrictedFixedManaEffect:
			result = append(result, e.productions...)
		case *sequenceEffect:
			result = append(result, couldProduceManaEffects(e.effects, ctx, costs)...)
		case *compositeEffect:
			result = append(result, couldProduceManaEffects(e.effects, ctx, costs)...)
		}
	}
	return result
}

func manaProductionColors(p ManaProduction) []Color {
	if p.Color != AnyColor || p.colorOptions == ([AnyColor]bool{}) {
		return expandProductionColor(p.Color)
	}
	var colors []Color
	for color := White; color <= Green; color++ {
		if p.colorOptions[color] {
			colors = append(colors, color)
		}
	}
	return colors
}

func chooseManaProductionColor(p Player, production ManaProduction, preferred Color) Color {
	colors := manaProductionColors(production)
	if slices.Contains(colors, preferred) {
		return preferred
	}
	if production.colorOptions == ([AnyColor]bool{}) {
		return p.ChooseManaColor("add mana")
	}
	if len(colors) == 1 {
		return colors[0]
	}
	options := make([]string, len(colors))
	for i, color := range colors {
		options[i] = color.String()
	}
	choice := p.ChooseString(options, "add mana")
	for i, option := range options {
		if option == choice {
			return colors[i]
		}
	}
	return colors[0]
}
