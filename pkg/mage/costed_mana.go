package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type manaSourceAbility struct {
	replacementOrder    []uuid.UUID
	originalProductions []ManaProduction
	AbilityIndex        int
	Productions         []ManaProduction
	ManaCost            ManaCost
	Interchangeable     bool
	Repeatable          bool
	ResolveBody         bool
}

func manaSourceAbilityForPlanning(a Ability, sourceID uuid.UUID, abilityIndex int, g *Game) (manaSourceAbility, bool) {
	inner := UnwrapAbility(a)
	if manaAbility, ok := inner.(*ManaAbility); ok {
		productions := manaAbility.currentProductions(g, sourceID)
		if len(productions) == 0 {
			return manaSourceAbility{}, false
		}
		manaCost := g.modifyActivationManaCost(sourceID, ManaCost{})
		return manaSourceAbility{
			AbilityIndex:    abilityIndex,
			ManaCost:        manaCost,
			Productions:     append([]ManaProduction(nil), productions...),
			Interchangeable: manaAbility.dynamicProductions == nil && manaAbility.permanentColors == nil && len(manaAbility.postProduction) == 0 && manaCost.IsZero(),
		}, true
	}

	activated, ok := inner.(*SimpleActivatedAbility)
	if !ok || len(activated.targets) != 0 || len(activated.effects) == 0 {
		return manaSourceAbility{}, false
	}
	if !manaSourceActivationAllowed(activated, sourceID, g) {
		return manaSourceAbility{}, false
	}
	productions, resolveBody, ok := plannedActivatedManaEffects(activated)
	if !ok {
		return manaSourceAbility{}, false
	}

	var manaCost ManaCost
	hasTapCost := false
	hasX := false
	for _, cost := range activated.costs {
		switch payment := cost.(type) {
		case *ManaCostPayment:
			hasX = hasX || payment.MC.HasX
			addActionManaCost(&manaCost, payment.MC)
		case *tap:
			if hasTapCost {
				return manaSourceAbility{}, false
			}
			hasTapCost = true
		default:
			return manaSourceAbility{}, false
		}
	}
	manaCost = g.modifyActivationManaCost(sourceID, manaCost)
	if !hasTapCost && (hasX || manaCostUnits(manaCost) < productionsTotalAmount(productions) ||
		activated.limits.OncePerTurn || activated.limits.MaxActivationsPerTurn > 0 ||
		len(activated.activationConds) != 0) {
		return manaSourceAbility{}, false
	}
	return manaSourceAbility{
		AbilityIndex:    abilityIndex,
		Productions:     productions,
		ManaCost:        manaCost,
		Interchangeable: true,
		Repeatable:      !hasTapCost,
		ResolveBody:     resolveBody,
	}, true
}

func manaSourceActivationAllowed(ability *SimpleActivatedAbility, sourceID uuid.UUID, g *Game) bool {
	perm := g.FindPermanent(sourceID)
	if perm == nil {
		return false
	}
	controller := perm.ControllerID()
	if ability.timing == TimingUpkeepOnly && g.turns.Step() != Upkeep {
		return false
	}
	if ability.timing == YourTurnOnly && g.ActivePlayerObj().PlayerID() != controller {
		return false
	}
	if ability.timing == TimingStepOnly && !ability.allowsStep(g.turns.Step()) {
		return false
	}
	if ability.limits.MaxActivationsPerTurn > 0 && ability.activationCount(g) >= ability.limits.MaxActivationsPerTurn {
		return false
	}
	if ability.limits.OncePerTurn && ability.activatedThisTurn {
		return false
	}
	if ability.limits.ControlledSinceTurnStart && !perm.ControlledSinceTurnStart(g) {
		return false
	}
	for _, condition := range ability.activationConds {
		if !condition(g, perm, controller) {
			return false
		}
	}
	if ability.SorcerySpeed() && (!g.turns.Step().IsMainPhase() || g.ActivePlayerObj().PlayerID() != controller || !g.stack.IsEmpty()) {
		return false
	}
	return true
}

func activatedManaEffects(effects []Effect) ([]ManaProduction, bool) {
	productions, onlyMana := sequentialManaProductions(effects)
	if !onlyMana {
		return nil, false
	}
	return productions, len(productions) > 0
}

// sequentialManaProductions reads only instructions that execute unconditionally.
// The planner must reject a body with instructions that it cannot execute.
func sequentialManaProductions(effects []Effect) ([]ManaProduction, bool) {
	var productions []ManaProduction
	onlyMana := len(effects) > 0
	for _, effect := range effects {
		switch add := effect.(type) {
		case *manaAmountEffect:
			if amount, ok := add.amount.(fixedValue); ok && amount.n > 0 {
				productions = append(productions, ManaProduction{Color: add.color, Amount: amount.n})
			} else {
				onlyMana = false
			}
		case *manaAnyOneColorEffect:
			if amount, ok := add.amount.(fixedValue); ok && amount.n > 0 {
				productions = append(productions, ManaProduction{Color: AnyColor, Amount: amount.n, Restriction: add.restriction})
			} else {
				onlyMana = false
			}
		case *restrictedFixedManaEffect:
			productions = append(productions, add.productions...)
		case *addManaEffect:
			productions = append(productions, ManaProduction{Color: add.color, Amount: add.amount})
		case *addAnyManaEffect:
			productions = append(productions, ManaProduction{Color: AnyColor, Amount: add.amount, Restriction: add.restriction})
		case *sequenceEffect:
			nested, pure := sequentialManaProductions(add.effects)
			productions = append(productions, nested...)
			onlyMana = onlyMana && pure
		case *compositeEffect:
			nested, pure := sequentialManaProductions(add.effects)
			productions = append(productions, nested...)
			onlyMana = onlyMana && pure
		default:
			onlyMana = false
		}
	}
	return productions, onlyMana
}

func plannedManaChoiceAvailable(productions []ManaProduction, bonuses []ManaBonusColor, action ManaTap) bool {
	if !plannedProductionsAvailable(productions, action.Productions) {
		return false
	}
	return plannedBonusColorsAvailable(bonuses, action.Productions, action.BonusColors)
}

func plannedProductionsAvailable(productions, planned []ManaProduction) bool {
	var caps manaChoiceCaps
	for _, production := range planned {
		if production.Color == AnyColor || production.Amount <= 0 {
			return false
		}
		caps[production.Color] += production.Amount
	}
	for _, choice := range concreteManaProductionsForDemand(productions, caps) {
		if concreteManaProductionsEqual(choice, planned) {
			return true
		}
	}
	return false
}

type concreteManaProductionKey struct {
	color       Color
	restriction string
}

func concreteManaProductionsEqual(a, b []ManaProduction) bool {
	amounts := make(map[concreteManaProductionKey]int, len(a))
	for _, production := range a {
		key := concreteManaProductionKey{
			color:       production.Color,
			restriction: manaRestrictionIdentity(production.Restriction),
		}
		amounts[key] += normalizedManaAmount(production.Amount)
	}
	for _, production := range b {
		key := concreteManaProductionKey{
			color:       production.Color,
			restriction: manaRestrictionIdentity(production.Restriction),
		}
		amounts[key] -= normalizedManaAmount(production.Amount)
	}
	for _, amount := range amounts {
		if amount != 0 {
			return false
		}
	}
	return true
}

func plannedBonusColorsAvailable(bonuses []ManaBonusColor, plannedProductions []ManaProduction, planned []Color) bool {
	var remaining [AnyColor + 1]int
	for _, color := range planned {
		if color == AnyColor {
			return false
		}
		remaining[color]++
	}
	matchCount := 0
	for _, bonus := range bonuses {
		if bonus == MatchProduced {
			matchCount++
			continue
		}
		color := Color(bonus)
		remaining[color]--
		if remaining[color] < 0 {
			return false
		}
	}
	produced := [AnyColor + 1]bool{}
	for _, color := range productionColors(plannedProductions) {
		produced[color] = true
	}
	if matchCount > 0 && len(productionColors(plannedProductions)) == 0 {
		matchCount = 0
	}
	total := 0
	for _, color := range manaPoolColors {
		if remaining[color] > 0 && !produced[color] {
			return false
		}
		total += remaining[color]
	}
	return total == matchCount
}

// plannedActivatedManaEffects permits delays that do not change current payment.
func plannedActivatedManaEffects(a *SimpleActivatedAbility) ([]ManaProduction, bool, bool) {
	if productions, ok := activatedManaEffects(a.effects); ok {
		return productions, false, true
	}
	var productions []ManaProduction
	var walk func([]Effect) bool
	walk = func(effects []Effect) bool {
		if len(effects) == 0 {
			return false
		}
		for _, effect := range effects {
			switch e := effect.(type) {
			case *sequenceEffect:
				if !walk(e.effects) {
					return false
				}
			case *compositeEffect:
				if !walk(e.effects) {
					return false
				}
			case *activationHistoryBranch:
				if e.binding == nil || e.binding != a.activationHistory || !plannedDeferredManaBody(e.body) {
					return false
				}
			default:
				values, ok := sequentialManaProductions([]Effect{effect})
				if !ok {
					return false
				}
				for _, value := range values {
					if value.Color == AnyColor || value.Amount <= 0 {
						return false
					}
				}
				productions = append(productions, values...)
			}
		}
		return true
	}
	ok := walk(a.effects)
	return productions, true, ok && len(productions) > 0
}

func plannedDeferredManaBody(effect Effect) bool {
	switch e := effect.(type) {
	case *nextStepEffect:
		return e.pattern != nil && e.body != nil && len(e.targets) == 0 && len(e.modes) == 0
	case *sequenceEffect:
		if len(e.effects) == 0 {
			return false
		}
		for _, part := range e.effects {
			if !plannedDeferredManaBody(part) {
				return false
			}
		}
		return true
	case *compositeEffect:
		if len(e.effects) == 0 {
			return false
		}
		for _, part := range e.effects {
			if !plannedDeferredManaBody(part) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
