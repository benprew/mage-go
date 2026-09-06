package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type manaSourceAbility struct {
	AbilityIndex    int
	Productions     []ManaProduction
	ManaCost        ManaCost
	Interchangeable bool
}

func manaSourceAbilityForPlanning(a Ability, sourceID uuid.UUID, abilityIndex int, g *Game) (manaSourceAbility, bool) {
	inner := UnwrapAbility(a)
	if manaAbility, ok := inner.(*ManaAbility); ok {
		productions := manaAbility.currentProductions(g, sourceID)
		if len(productions) == 0 {
			return manaSourceAbility{}, false
		}
		return manaSourceAbility{
			AbilityIndex:    abilityIndex,
			Productions:     append([]ManaProduction(nil), productions...),
			Interchangeable: manaAbility.dynamicProductions == nil && len(manaAbility.postProduction) == 0,
		}, true
	}

	activated, ok := inner.(*SimpleActivatedAbility)
	if !ok || len(activated.targets) != 0 || len(activated.effects) == 0 {
		return manaSourceAbility{}, false
	}
	if !manaSourceActivationAllowed(activated, sourceID, g) {
		return manaSourceAbility{}, false
	}
	productions, ok := activatedManaEffects(activated.effects)
	if !ok {
		return manaSourceAbility{}, false
	}

	var manaCost ManaCost
	hasTapCost := false
	for _, cost := range activated.costs {
		switch payment := cost.(type) {
		case *ManaCostPayment:
			addActionManaCost(&manaCost, payment.reducedCost(sourceID, g))
		case *tap:
			if hasTapCost {
				return manaSourceAbility{}, false
			}
			hasTapCost = true
		default:
			return manaSourceAbility{}, false
		}
	}
	if !hasTapCost {
		return manaSourceAbility{}, false
	}
	return manaSourceAbility{
		AbilityIndex:    abilityIndex,
		Productions:     productions,
		ManaCost:        manaCost,
		Interchangeable: true,
	}, true
}

func manaSourceActivationAllowed(ability *SimpleActivatedAbility, sourceID uuid.UUID, g *Game) bool {
	perm := g.FindPermanent(sourceID)
	if perm == nil {
		return false
	}
	controller := perm.ControllerID()
	if ability.timing == TimingUpkeepOnly && g.step != Upkeep {
		return false
	}
	if ability.timing == YourTurnOnly && g.ActivePlayerObj().PlayerID() != controller {
		return false
	}
	if ability.timing == TimingStepOnly && ability.stepOnly != 0 && g.step != ability.stepOnly {
		return false
	}
	if ability.limits.MaxActivationsPerTurn > 0 && ability.activationsThisTurn >= ability.limits.MaxActivationsPerTurn {
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
	if ability.SorcerySpeed() && (!g.step.IsMainPhase() || g.ActivePlayerObj().PlayerID() != controller || !g.stack.IsEmpty()) {
		return false
	}
	return true
}

func activatedManaEffects(effects []Effect) ([]ManaProduction, bool) {
	productions := make([]ManaProduction, 0, len(effects))
	for _, effect := range effects {
		switch add := effect.(type) {
		case *addManaEffect:
			productions = append(productions, ManaProduction{Color: add.color, Amount: add.amount})
		case *addAnyManaEffect:
			productions = append(productions, ManaProduction{Color: AnyColor, Amount: add.amount})
		default:
			return nil, false
		}
	}
	return productions, len(productions) > 0
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
