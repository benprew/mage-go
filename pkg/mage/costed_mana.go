package mage

import (
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type manaSourceAbility struct {
	AbilityIndex    int
	Productions     []ManaProduction
	ManaCost        ManaCost
	Interchangeable bool
}

func manaSourceAbilityForPlanning(a Ability, sourceID uuid.UUID, abilityIndex int, g *Game) (manaSourceAbility, bool) {
	producer, ok := UnwrapAbility(a).(ManaProducer)
	if !ok {
		return manaSourceAbility{}, false
	}
	profile := producer.ManaProfile(g, sourceID)
	if len(profile.Productions) == 0 {
		return manaSourceAbility{}, false
	}
	if permission, ok := producer.(interface {
		manaActivationAllowed(AbilityContext, *Game) bool
	}); ok {
		permanent := g.FindPermanent(sourceID)
		if permanent == nil || !permission.manaActivationAllowed(AbilityContext{SourceID: sourceID, Controller: permanent.ControllerID(), AbilityIndex: abilityIndex}, g) {
			return manaSourceAbility{}, false
		}
	}
	var mana ManaCost
	hasTap := false
	for _, cost := range profile.Costs {
		metadata, ok := cost.(manaPlanningCost)
		if !ok {
			return manaSourceAbility{}, false
		}
		component, tap := metadata.manaPlanCost(sourceID, g)
		if tap && hasTap {
			return manaSourceAbility{}, false
		}
		hasTap = hasTap || tap
		addActionManaCost(&mana, component)
	}
	if !hasTap {
		return manaSourceAbility{}, false
	}
	return manaSourceAbility{AbilityIndex: abilityIndex, Productions: profile.Productions, ManaCost: mana, Interchangeable: profile.Interchangeable}, true
}

func manaSourceActivationAllowed(ability *SimpleActivatedAbility, sourceID uuid.UUID, abilityIndex int, g *Game) bool {
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
	if ability.timing == TimingStepOnly && ability.stepOnly != 0 && g.turns.Step() != ability.stepOnly {
		return false
	}
	if ability.limits.MaxActivationsPerTurn > 0 && g.activationCount(sourceID, ability.AbilityID(), abilityIndex) >= ability.limits.MaxActivationsPerTurn {
		return false
	}
	if ability.limits.OncePerTurn && (g.activationCount(sourceID, ability.AbilityID(), abilityIndex) > 0) {
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
	var productions []ManaProduction
	for _, effect := range effects {
		producer, ok := effect.(ManaProductionEffect)
		if !ok {
			return nil, false
		}
		productions = append(productions, producer.ManaProductions()...)
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
