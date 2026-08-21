package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type manaSourceAbility struct {
	AbilityIndex int
	Productions  []ManaProduction
	ManaCost     ManaCost
}

func manaSourceAbilityForPlanning(a Ability, sourceID uuid.UUID, abilityIndex int, g *Game) (manaSourceAbility, bool) {
	inner := UnwrapAbility(a)
	if manaAbility, ok := inner.(*ManaAbility); ok {
		productions := manaAbility.currentProductions(g, sourceID)
		if len(productions) == 0 {
			return manaSourceAbility{}, false
		}
		return manaSourceAbility{
			AbilityIndex: abilityIndex,
			Productions:  append([]ManaProduction(nil), productions...),
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
		AbilityIndex: abilityIndex,
		Productions:  productions,
		ManaCost:     manaCost,
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

func (g *Game) applyManaSolution(playerID uuid.UUID, solution *ManaSolution) error {
	if solution == nil {
		return nil
	}
	for _, action := range solution.SourcesToTap {
		if err := g.activatePlannedManaSource(playerID, action); err != nil {
			return err
		}
	}
	return nil
}

func (g *Game) activatePlannedManaSource(playerID uuid.UUID, action ManaTap) error {
	perm := g.FindPermanent(action.PermanentID)
	if perm == nil || action.AbilityIndex < 0 || action.AbilityIndex >= len(perm.RuntimeAbilities) {
		return ErrPermanentNotFound
	}
	if perm.ControllerID() != playerID {
		return fmt.Errorf("you don't control that permanent")
	}
	if perm.Tapped {
		return fmt.Errorf("permanent is already tapped")
	}
	if perm.HasAttr(AttrCantActivate) || !perm.CanTapForEffect(g) {
		return fmt.Errorf("cannot activate mana ability of %s", perm.Name())
	}

	ability, ok := manaSourceAbilityForPlanning(perm.RuntimeAbilities[action.AbilityIndex], perm.ID(), action.AbilityIndex, g)
	if !ok {
		return fmt.Errorf("ability is not a supported mana ability")
	}
	if !plannedManaChoiceAvailable(ability.Productions, g.manaBonuses(perm.ID()), action) {
		return fmt.Errorf("planned mana production is no longer available")
	}
	player := g.GetPlayer(playerID)
	if player == nil {
		return ErrPlayerNotFound
	}
	if !player.ManaPool().CanPay(ability.ManaCost, nil) {
		return fmt.Errorf("cannot pay mana ability activation cost %s", ability.ManaCost)
	}
	if err := player.ManaPool().Pay(ability.ManaCost, nil); err != nil {
		return err
	}

	g.TapPermanent(perm)
	addConcreteMana(player.ManaPool(), action.Productions, action.BonusColors)

	inner := UnwrapAbility(perm.RuntimeAbilities[action.AbilityIndex])
	switch activated := inner.(type) {
	case *ManaAbility:
		if err := g.runManaPostProduction(activated, player, perm); err != nil {
			return err
		}
	case *SimpleActivatedAbility:
		activated.MarkActivated()
		g.FireEvent(GameEvent{Type: EvtAbilityActivated, SourceID: perm.ID(), PlayerID: playerID})
	}
	return nil
}

func plannedManaChoiceAvailable(productions []ManaProduction, bonuses []ManaBonusColor, action ManaTap) bool {
	for _, choice := range manaProductionChoices(productions, bonuses) {
		if slices.Equal(choice.Productions, action.Productions) && slices.Equal(choice.BonusColors, action.BonusColors) {
			return true
		}
	}
	return false
}
