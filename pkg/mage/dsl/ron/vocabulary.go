package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// Actions keeps target declarations with their action list.
type Actions struct {
	targetBindings []mage.TargetBinding
	options        []mage.ActionOption
	effects        []mage.Effect
	targets        []mage.Target
}

// ActionList keeps actions in resolution order.
func ActionList(effects ...mage.Effect) Actions {
	return Actions{effects: append([]mage.Effect(nil), effects...)}
}

// Targeted adds target declarations to an action list.
func Targeted(targets []mage.Target, actions Actions) Actions {
	actions.targets = append([]mage.Target(nil), targets...)
	return actions
}

// SpellActions creates a spell action from a RON action list.
func SpellActions(actions Actions) *mage.ActionDefinition {
	return mage.NewAction(mage.ActionSpell, actionOptions(actions)...)
}

// Activated creates an activated action with its declared costs.
func Activated(costs []mage.Cost, actions Actions) *mage.ActionDefinition {
	opts := actionOptions(actions)
	for _, cost := range costs {
		opts = append(opts, mage.WithCost(cost))
	}
	return mage.NewAction(mage.ActionActivated, opts...)
}

func actionOptions(actions Actions) []mage.ActionOption {
	opts := make([]mage.ActionOption, 0, 1+len(actions.targets))
	opts = append(opts, mage.WithEffects(mage.Sequence(actions.effects...)))
	for _, target := range actions.targets {
		opts = append(opts, mage.WithTarget(target))
	}
	return append(opts, actions.options...)
}

// Integer supplies a fixed value to the engine.
func Integer(n int) mage.ValueSource { return mage.Fixed(n) }

// PayMana preserves the declared mana symbols as an activation cost.
func PayMana(cost core.ManaCost) mage.Cost { return &mage.ManaCostPayment{MC: cost} }

// IsCardtype tests the current type of a permanent.
func IsCardtype(kind core.CardType) mage.PermanentFilter {
	return mage.HasCardTypeFilter(kind)
}

// IsCreatureType tests a creature subtype.
func IsCreatureType(subtype string) mage.PermanentFilter { return mage.HasSubType(subtype) }

// SpellDealsDamage uses the current spell as the damage source.
func SpellDealsDamage(amount mage.ValueSource) mage.Effect { return mage.DealDamage(amount) }

// PermanentDealsDamage uses the current ability source as the damage source.
func PermanentDealsDamage(amount mage.ValueSource) mage.Effect { return mage.DealDamage(amount) }

// DrawNumberCards uses an explicit player selector to avoid an implicit target.
func DrawNumberCards(amount mage.ValueSource, who mage.PlayerSelector) mage.Effect {
	return mage.DrawCards(amount).Targeting(who)
}

// DestroyPermanent destroys the bound permanent target.
func DestroyPermanent() mage.Effect { return mage.DestroyTarget() }

// DestroyPermanentNoRegen destroys the bound target without regeneration.
func DestroyPermanentNoRegen() mage.Effect { return mage.DestroyTargetNoRegen() }

// DestroyEachPermanent uses the engine's batch destruction action.
func DestroyEachPermanent(filter mage.PermanentFilter) mage.Effect {
	return mage.DestroyEach(mage.MatchingPermanents(filter))
}

// DestroyEachPermanentNoRegen uses batch destruction without regeneration.
func DestroyEachPermanentNoRegen(filter mage.PermanentFilter) mage.Effect {
	return mage.DestroyEachWithoutRegeneration(mage.MatchingPermanents(filter))
}
