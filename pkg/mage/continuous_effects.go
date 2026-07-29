package mage

import (
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ---------------------------------------------------------------------------
// Attached effects (aura/equipment patterns)
// ---------------------------------------------------------------------------

// BoostAttached creates a continuous effect that boosts the attached creature's P/T.
func BoostAttached(power, toughness int, at AttachType) ContinuousEffect {
	return AttachedEffect(LayerPT, func(g *Game, source, target *Permanent) error {
		target.powerBonus += power
		target.toughBonus += toughness
		return nil
	})
}

// GrantAbilityToAttached creates a continuous effect granting a keyword to the attached creature.
func GrantAbilityToAttached(kw Keyword, at AttachType) ContinuousEffect {
	return AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
		g.effects.GrantAttr(target.ID(), kw)
		return nil
	})
}

// GrantProtectionToAttached creates a continuous effect granting protection from
// a color to the attached creature (e.g. Black Ward, Blue Ward).
func GrantProtectionToAttached(color Color, at AttachType) ContinuousEffect {
	return AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
		target.RuntimeAbilities = append(target.RuntimeAbilities, ProtectionFromColor(color))
		return nil
	})
}

// PreventAttachedFromActivatingNonManaAbilities creates a continuous effect that
// stops the attached permanent's non-mana activated abilities from being
// activated. CR 605 mana abilities are unaffected (they bypass priority and the
// stack, and aren't checked through SimpleActivatedAbility.CanActivate).
// Used by Lawmage's Binding and similar "and its activated abilities can't be
// activated" auras.
func PreventAttachedFromActivatingNonManaAbilities(at AttachType) ContinuousEffect {
	return AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
		g.effects.GrantAttr(target.ID(), AttrCantActivateNonManaAbilities)
		return nil
	})
}

// AssignsDamageEqualToToughnessForCreaturesYouControl grants
// AttrAssignsDamageEqualToToughness to each creature controlled by the
// source's controller. Damage assignment in combat then uses the creature's
// toughness instead of power. Used by Assault Formation
// ("Each creature you control assigns combat damage equal to its toughness
// rather than its power"). Does not change the creature's power.
func AssignsDamageEqualToToughnessForCreaturesYouControl() ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		src := g.FindPermanent(sourceID)
		if src == nil {
			return nil
		}
		for _, p := range g.battlefield {
			if !p.HasType(TypeCreature) || p.ControllerID() != src.ControllerID() {
				continue
			}
			g.effects.GrantAttr(p.ID(), AttrAssignsDamageEqualToToughness)
		}
		return nil
	})
}

// AssignsDamageEqualToToughnessSelf grants AttrAssignsDamageEqualToToughness
// to the source permanent itself. Used by Doran the Siege Tower-style cards
// where only the source assigns toughness instead of power.
func AssignsDamageEqualToToughnessSelf() ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		g.effects.GrantAttr(sourceID, AttrAssignsDamageEqualToToughness)
		return nil
	})
}

// PreventActivationsOfMatching creates a continuous effect that stops non-mana
// activated abilities from being activated for permanents matching filter.
// Used by static effects like "Activated abilities of artifacts your opponents
// control can't be activated" — pass an opponent-scoped filter.
func PreventActivationsOfMatching(filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		for _, p := range g.battlefield {
			if !filter.Match(p, g) {
				continue
			}
			g.effects.GrantAttr(p.ID(), AttrCantActivateNonManaAbilities)
		}
		return nil
	})
}

// RemoveKeywordFromAttached creates a continuous effect removing a keyword from the attached creature.
func RemoveKeywordFromAttached(kw Keyword, at AttachType) ContinuousEffect {
	return AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
		g.effects.RevokeAttr(target.ID(), kw)
		return nil
	})
}

// ChangeAttachedSubTypes replaces the subtypes of the attached permanent (e.g. Evil Presence
// makes enchanted land a Swamp, Phantasmal Terrain makes it a chosen type).
func ChangeAttachedSubTypes(newSubTypes []string) ContinuousEffect {
	return AttachedEffect(LayerType, func(g *Game, source, target *Permanent) error {
		target.SubTypeOverride = make([]string, len(newSubTypes))
		copy(target.SubTypeOverride, newSubTypes)
		return nil
	})
}

// ChangeAttachedSubTypesByChosenColor replaces the subtypes of the attached permanent
// based on the source permanent's ChosenColor (e.g. Phantasmal Terrain: "choose a
// basic land type, enchanted land is the chosen type").
func ChangeAttachedSubTypesByChosenColor() ContinuousEffect {
	return AttachedEffect(LayerType, func(g *Game, source, target *Permanent) error {
		landType := ColorToBasicLandType(source.ChosenColor)
		if landType != "" {
			target.SubTypeOverride = []string{landType}
		}
		return nil
	})
}

// ColorToBasicLandType maps a Color to its corresponding basic land type name.
func ColorToBasicLandType(c Color) string {
	switch c {
	case White:
		return "Plains"
	case Blue:
		return "Island"
	case Black:
		return "Swamp"
	case Red:
		return "Mountain"
	case Green:
		return "Forest"
	default:
		return ""
	}
}

// GrantTriggeredAbilityToAttached grants a triggered ability to the permanent
// the source aura/equipment is attached to. The trigger fires from the
// attached permanent (its source/controller), not from the aura — so e.g.
// Curse of Bloodletting's "If a source would deal damage to enchanted player,
// it deals double that damage instead" is registered with the enchanted
// player's controller as the trigger source.
//
// Use this for auras like the Curse cycle and Stalwart Aven, where Oracle
// text grants an ability "to enchanted creature" or "to enchanted player".
func GrantTriggeredAbilityToAttached(eventType EventType, optional bool, cond TriggerConditionData, effects ...Effect) ContinuousEffect {
	return AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
		trig := NewTriggered(eventType, optional, effects...)
		trig.source = target.ID()
		trig.controller = target.ControllerID()
		if cond != nil {
			trig.SetConditionData(cond)
		}
		target.RuntimeAbilities = append(target.RuntimeAbilities, &grantedByEffect{trig})
		return nil
	})
}

// GrantActivatedAbilityToAttached grants an activated ability to the attached creature.
// The ability instance is persisted across continuous-effect reapplications so that
// per-turn activation tracking (OncePerTurn, MaxActivationsPerTurn) survives.
func GrantActivatedAbilityToAttached(effect Effect, cost Cost, at AttachType, opts ...AbilityOption) ContinuousEffect {
	abByTarget := map[uuid.UUID]*SimpleActivatedAbility{}
	return AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
		ab, ok := abByTarget[target.ID()]
		if !ok {
			ab = NewActivatedAbility(effect, cost, opts...)
			abByTarget[target.ID()] = ab
		}
		ab.source = target.ID()
		ab.controller = target.ControllerID()
		target.RuntimeAbilities = append(target.RuntimeAbilities, &grantedByEffect{ab})
		return nil
	})
}

// PreventAttachedFromUntapping creates a continuous effect preventing the attached creature from untapping.
func PreventAttachedFromUntapping(at AttachType) ContinuousEffect {
	return AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
		g.effects.GrantAttr(target.ID(), AttrDoesNotUntap)
		return nil
	})
}

// PreventAttachedFromAttacking creates a continuous effect preventing the attached creature from attacking.
func PreventAttachedFromAttacking(at AttachType) ContinuousEffect {
	return AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
		g.effects.RevokeAttr(target.ID(), AttrCanAttack)
		return nil
	})
}

// ControlChangeContinuous creates a continuous control change effect (e.g., Control Magic).
func ControlChangeContinuous() ContinuousEffect {
	return ControlAttached()
}

// BoostAttachedByCount boosts the attached creature based on the count of
// controlled permanents matching filter. powerFn and toughFn convert the count
// to P/T bonuses (e.g. for Aspect of Wolf: count/2 and (count+1)/2).
func BoostAttachedByCount(filter PermanentFilter, powerFn, toughFn func(int) int) ContinuousEffect {
	return AttachedEffect(LayerPT, func(g *Game, source, target *Permanent) error {
		count := g.CountBattlefield(And(ControlledBy(source.ControllerID()), filter))
		target.powerBonus += powerFn(count)
		target.toughBonus += toughFn(count)
		return nil
	})
}

// ---------------------------------------------------------------------------
// Target effects (specific permanent by ID)
// ---------------------------------------------------------------------------

// TemporaryBoost creates a continuous effect that boosts a specific creature until end of turn.
func TemporaryBoost(targetID uuid.UUID, power, toughness int) ContinuousEffect {
	return TargetEffect(LayerPT, EndOfTurn, targetID, func(g *Game, target *Permanent) error {
		target.powerBonus += power
		target.toughBonus += toughness
		return nil
	})
}

// TemporaryKeyword creates a continuous effect granting a keyword to a creature until end of turn.
func TemporaryKeyword(targetID uuid.UUID, kw Keyword) ContinuousEffect {
	return TargetEffect(LayerAbility, EndOfTurn, targetID, func(g *Game, target *Permanent) error {
		g.effects.GrantAttr(target.ID(), kw)
		return nil
	})
}

// KeywordReplacement replaces one keyword with another on a target permanently
// (e.g. swampwalk -> forestwalk via Sleight of Mind / Magical Hack).
func KeywordReplacement(targetID uuid.UUID, from, to Keyword) ContinuousEffect {
	return TargetEffect(LayerAbility, Indefinite, targetID, func(g *Game, target *Permanent) error {
		g.effects.RevokeAttr(target.ID(), from)
		g.effects.GrantAttr(target.ID(), to)
		return nil
	})
}

// SetBasePT creates a continuous effect that sets a creature's base P/T until end of turn.
// Used by Sorceress Queen ({T}: target creature has base P/T 0/2 until EOT).
func SetBasePT(targetID uuid.UUID, power, toughness int) ContinuousEffect {
	return TargetEffect(LayerPT, EndOfTurn, targetID, func(g *Game, target *Permanent) error {
		target.BasePTOverride = &[2]int{power, toughness}
		return nil
	})
}

// SetBasePower creates a continuous effect that sets a creature's base power until end of turn,
// leaving toughness unchanged. Used by Singing Tree and Island of Wak-Wak.
func SetBasePower(targetID uuid.UUID, power int) ContinuousEffect {
	return TargetEffect(LayerPT, EndOfTurn, targetID, func(g *Game, target *Permanent) error {
		currentToughness := target.Card.Toughness()
		if target.BasePTOverride != nil {
			currentToughness = target.BasePTOverride[1]
		}
		target.BasePTOverride = &[2]int{power, currentToughness}
		return nil
	})
}

// ColorOverride permanently changes a target permanent's color (Lace cycle).
func ColorOverride(targetID uuid.UUID, color Color) ContinuousEffect {
	return TargetEffect(LayerColor, Indefinite, targetID, func(g *Game, target *Permanent) error {
		colors := []Color{color}
		target.ColorOverride = &colors
		return nil
	})
}

// temporaryAnimate creates a target effect that animates a permanent into a creature.
func temporaryAnimate(targetID uuid.UUID, power, toughness int, duration Duration) ContinuousEffect {
	return TargetEffect(LayerType, duration, targetID, func(g *Game, target *Permanent) error {
		g.effects.GrantAttr(target.ID(), AttrIsCreature)
		g.effects.GrantAttr(target.ID(), AttrCanAttack)
		g.effects.GrantAttr(target.ID(), AttrCanBlock)
		g.effects.GrantAttr(target.ID(), AttrHasPowerToughness)
		target.BasePTOverride = &[2]int{power, toughness}
		return nil
	})
}

// TemporaryAnimate creates an end-of-turn effect that animates a permanent.
func TemporaryAnimate(targetID uuid.UUID, power, toughness int) ContinuousEffect {
	return temporaryAnimate(targetID, power, toughness, EndOfTurn)
}

// TemporaryAnimateUntilEndOfCombat creates an end-of-combat effect that animates a permanent.
func TemporaryAnimateUntilEndOfCombat(targetID uuid.UUID, power, toughness int) ContinuousEffect {
	return temporaryAnimate(targetID, power, toughness, EndOfCombat)
}

// PreventAttackingUntilEndOfTurn creates an EndOfTurn-scoped continuous effect
// that revokes AttrCanAttack from a specific creature. Used for instant-speed
// "target creature can't attack this turn" effects (CR 506.4a). Per CR 506.4a,
// applying this effect to a creature that has already been declared as an
// attacker does not remove it from combat — the engine layer handles that
// because the AttrCanAttack check only runs at declaration time. The effect
// keeps the attribute revoked for the rest of the turn, so the creature can't
// be re-declared in any later combat phase this turn.
func PreventAttackingUntilEndOfTurn(permID uuid.UUID) ContinuousEffect {
	return TargetEffect(LayerAbility, EndOfTurn, permID, func(g *Game, target *Permanent) error {
		g.effects.RevokeAttr(target.ID(), AttrCanAttack)
		return nil
	})
}

// PreventBlockingUntilEndOfCombat creates an EndOfCombat-scoped continuous effect
// that revokes AttrCanBlock from a specific creature. Re-fires on each Apply() cycle
// (surviving grantedAttrs reset) and expires at EndCombat via RemoveEndOfCombat().
// Use this instead of the imperative preventBlock map so block-prevention goes through
// the attr system like every other capability restriction.
func PreventBlockingUntilEndOfCombat(permID uuid.UUID) ContinuousEffect {
	return TargetEffect(LayerAbility, EndOfCombat, permID, func(g *Game, target *Permanent) error {
		g.effects.RevokeAttr(target.ID(), AttrCanBlock)
		return nil
	})
}

// PreventBlockingUntilEndOfTurn is the EndOfTurn-scoped sibling of
// PreventBlockingUntilEndOfCombat: revokes AttrCanBlock from a specific
// creature and persists across all combat phases this turn (CR 514 cleanup).
// Use for "target creature can't block this turn" effects (Volcanic Hammer
// variants, Conduit of Storms, etc.) where the restriction must outlive a
// single combat phase.
func PreventBlockingUntilEndOfTurn(permID uuid.UUID) ContinuousEffect {
	return TargetEffect(LayerAbility, EndOfTurn, permID, func(g *Game, target *Permanent) error {
		g.effects.RevokeAttr(target.ID(), AttrCanBlock)
		return nil
	})
}

// ---------------------------------------------------------------------------
// FuncContinuousEffect-based effects (source on battlefield)
// ---------------------------------------------------------------------------

// GrantActivatedAbilityToAll grants an activated ability to all creatures matching filter.
func GrantActivatedAbilityToAll(effect Effect, cost Cost, filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		for _, p := range g.battlefield {
			if !p.HasType(TypeCreature) {
				continue
			}
			if p.ID() == sourceID {
				continue // typically "other" creatures
			}
			if !filter.Match(p, g) {
				continue
			}
			ab := NewActivatedAbility(effect, cost)
			ab.source = p.ID()
			ab.controller = p.ControllerID()
			p = g.MutablePermanent(p.ID())
			if p == nil {
				continue
			}
			p.RuntimeAbilities = append(p.RuntimeAbilities, &grantedByEffect{ab})
		}
		return nil
	})
}

// GrantTriggeredAbilityToAll grants a triggered ability to all permanents matching filter.
// Each permanent gets its own copy of the triggered ability, allowing individual trigger
// ordering (unlike a single batch trigger). The trigger is constructed from the provided
// event type, optional flag, condition, and effects. The source permanent itself is
// excluded (typical "other creatures you control" lord behavior); use
// GrantTriggeredAbilityToAllIncludingSource for "creatures you control" effects that
// include the source itself (e.g. Kira, Great Glass-Spinner).
func GrantTriggeredAbilityToAll(eventType EventType, optional bool, cond TriggerConditionData, filter PermanentFilter, effects ...Effect) ContinuousEffect {
	return grantTriggeredAbilityToAll(eventType, optional, cond, filter, false, effects...)
}

// GrantTriggeredAbilityToAllIncludingSource is like GrantTriggeredAbilityToAll but
// matches the source permanent as well, when filter accepts it. Used by
// "creatures you control" lord-style triggers where Oracle text clearly includes
// the source itself.
func GrantTriggeredAbilityToAllIncludingSource(eventType EventType, optional bool, cond TriggerConditionData, filter PermanentFilter, effects ...Effect) ContinuousEffect {
	return grantTriggeredAbilityToAll(eventType, optional, cond, filter, true, effects...)
}

func grantTriggeredAbilityToAll(eventType EventType, optional bool, cond TriggerConditionData, filter PermanentFilter, includeSource bool, effects ...Effect) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		for _, p := range g.battlefield {
			if !includeSource && p.ID() == sourceID {
				continue
			}
			if !filter.Match(p, g) {
				continue
			}
			trig := NewTriggered(eventType, optional, effects...)
			trig.source = p.ID()
			trig.controller = p.ControllerID()
			if cond != nil {
				trig.SetConditionData(cond)
			}
			p = g.MutablePermanent(p.ID())
			if p == nil {
				continue
			}
			p.RuntimeAbilities = append(p.RuntimeAbilities, &grantedByEffect{trig})
		}
		return nil
	})
}

// PreventFromAttackingIfDefendingPlayerControls creates a continuous effect preventing the creature from
// attacking if the defender controls a certain type of card.
// In a 2-player game the defending player is always the non-active player.
// Multiplayer will need a PlayerSelector parameter here.
func PreventFromAttackingIfDefendingPlayerControls(filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		who := g.NonActivePlayerObj()
		whoID := who.PlayerID()
		if g.AnyBattlefield(And(ControlledBy(whoID), filter)) {
			return nil
		}
		g.effects.RevokeAttr(sourceID, AttrCanAttack)
		return nil
	})
}

// BoostAllCreatures creates a continuous effect that boosts all matching creatures
// except the source (typical lord behavior).
func BoostAllCreatures(power, toughness int, filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		for _, p := range g.battlefield {
			if !p.HasType(TypeCreature) || p.ID() == sourceID {
				continue
			}
			if !filter.Match(p, g) {
				continue
			}
			p = g.MutablePermanent(p.ID())
			if p == nil {
				continue
			}
			p.powerBonus += power
			p.toughBonus += toughness
		}
		return nil
	})
}

// BoostAllCreaturesIncludingSelf creates a continuous effect that boosts all
// matching creatures including the source.
func BoostAllCreaturesIncludingSelf(power, toughness int, filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		for _, p := range g.battlefield {
			if !p.HasType(TypeCreature) {
				continue
			}
			if !filter.Match(p, g) {
				continue
			}
			p = g.MutablePermanent(p.ID())
			if p == nil {
				continue
			}
			p.powerBonus += power
			p.toughBonus += toughness
		}
		return nil
	})
}

// PTEqualsCount creates a continuous effect where the source creature gets
// +N/+N where N is the count of permanents matching countFilter (on whole battlefield).
// Used for Plague Rats, etc.
func PTEqualsCount(countFilter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		src := g.MutablePermanent(sourceID)
		if src == nil {
			return nil
		}
		count := g.CountBattlefield(countFilter)
		src.powerBonus += count
		src.toughBonus += count
		return nil
	})
}

// PTEqualsControlledCount creates a continuous effect where the source creature gets
// +N/+N where N is the count of permanents matching countFilter that you control.
func PTEqualsControlledCount(countFilter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		src := g.MutablePermanent(sourceID)
		if src == nil {
			return nil
		}
		count := g.CountBattlefield(And(ControlledBy(src.ControllerID()), countFilter))
		src.powerBonus += count
		src.toughBonus += count
		return nil
	})
}

// GrantKeywordToAll grants a keyword ability to all matching creatures (excluding source).
func GrantKeywordToAll(kw Keyword, filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		for _, p := range g.battlefield {
			if !p.HasType(TypeCreature) || p.ID() == sourceID {
				continue
			}
			if !filter.Match(p, g) {
				continue
			}
			g.effects.GrantAttr(p.ID(), kw)
		}
		return nil
	})
}

// GrantKeywordToControlled grants a keyword ability to creatures controlled by
// the source's controller that match an optional filter (e.g. Goblin War Drums
// gives menace to your creatures).
func GrantKeywordToControlled(kw Keyword, filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		src := g.FindPermanent(sourceID)
		if src == nil {
			return nil
		}
		for _, p := range g.battlefield {
			if !p.HasType(TypeCreature) || p.ControllerID() != src.ControllerID() {
				continue
			}
			if !filter.Match(p, g) {
				continue
			}
			g.effects.GrantAttr(p.ID(), kw)
		}
		return nil
	})
}

// GrantKeywordToOtherControlled grants a keyword ability to OTHER creatures
// controlled by the source's controller that match an optional filter
// (e.g. Kobold Overlord gives first strike to other Kobolds you control).
func GrantKeywordToOtherControlled(kw Keyword, filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		src := g.FindPermanent(sourceID)
		if src == nil {
			return nil
		}
		for _, p := range g.battlefield {
			if !p.HasType(TypeCreature) || p.ID() == sourceID || p.ControllerID() != src.ControllerID() {
				continue
			}
			if !filter.Match(p, g) {
				continue
			}
			g.effects.GrantAttr(p.ID(), kw)
		}
		return nil
	})
}

// BoostOtherControlledCreatures boosts OTHER creatures controlled by the
// source's controller that match an optional filter
// (e.g. Kobold Taskmaster gives +1/+0 to other Kobolds you control).
func BoostOtherControlledCreatures(power, toughness int, filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		src := g.FindPermanent(sourceID)
		if src == nil {
			return nil
		}
		for _, p := range g.battlefield {
			if !p.HasType(TypeCreature) || p.ID() == sourceID || p.ControllerID() != src.ControllerID() {
				continue
			}
			if !filter.Match(p, g) {
				continue
			}
			p = g.MutablePermanent(p.ID())
			if p == nil {
				continue
			}
			p.powerBonus += power
			p.toughBonus += toughness
		}
		return nil
	})
}

// RevokeKeywordFromAll revokes a keyword ability from all creatures matching
// the given filter (e.g. Gravity Sphere revokes Flying from all creatures).
func RevokeKeywordFromAll(kw Keyword, filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		for _, p := range g.FilterBattlefield(And(IsCreature, filter)) {
			g.effects.RevokeAttr(p.ID(), kw)
		}
		return nil
	})
}

// RevokeAttrFromControlled revokes an attr from all creatures controlled by
// the source's controller that match the given filter (e.g. Akron Legionnaire
// revokes AttrCanAttack from non-artifact, non-self creatures you control).
func RevokeAttrFromControlled(attr Attr, filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		src := g.FindPermanent(sourceID)
		if src == nil {
			return nil
		}
		for _, p := range g.battlefield {
			if !p.HasType(TypeCreature) || p.ControllerID() != src.ControllerID() {
				continue
			}
			if !filter.Match(p, g) {
				continue
			}
			g.effects.RevokeAttr(p.ID(), attr)
		}
		return nil
	})
}

// NullifyLandwalkEffect creates a continuous effect that nullifies a specific
// landwalk ability (e.g. Great Wall nullifies plainswalk). While the source is
// on the battlefield, creatures with the specified landwalk can be blocked as
// though they didn't have it.
func NullifyLandwalkEffect(kw Attr) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		g.effects.Rules.NullifyLandwalk(kw)
		return nil
	})
}

// BoostControlledCreatures boosts creatures controlled by the source's controller
// that match an optional filter.
func BoostControlledCreatures(power, toughness int, filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		src := g.FindPermanent(sourceID)
		if src == nil {
			return nil
		}
		for _, p := range g.battlefield {
			if !p.HasType(TypeCreature) || p.ControllerID() != src.ControllerID() {
				continue
			}
			if !filter.Match(p, g) {
				continue
			}
			p = g.MutablePermanent(p.ID())
			if p == nil {
				continue
			}
			p.powerBonus += power
			p.toughBonus += toughness
		}
		return nil
	})
}

// PreventUntapForMatching creates a continuous effect that sets DoesNotUntap
// on all permanents matching the given filter (e.g. Meekstone).
func PreventUntapForMatching(filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		for _, p := range g.FilterBattlefield(filter) {
			g.effects.GrantAttr(p.ID(), AttrDoesNotUntap)
		}
		return nil
	})
}

// IncreaseSpellCostForColor is a continuous effect that increases the cost of
// spells of a given color (e.g. Gloom makes white spells cost {3} more).
func IncreaseSpellCostForColor(color Color, amount int) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		g.effects.Rules.SpellCostIncreases[color] += amount
		return nil
	})
}

// ReduceSpellCostForColor is a continuous effect that reduces the cost of
// spells of a given color (e.g. "Blue spells cost {1} less to cast").
func ReduceSpellCostForColor(color Color, amount int) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		g.effects.Rules.SpellCostReductions[color] += amount
		return nil
	})
}

// ChangeSubTypesForAll changes subtypes of all permanents matching fromSubTypes
// to toSubTypes (e.g. Conversion: all Mountains become Plains).
func ChangeSubTypesForAll(fromSubTypes, toSubTypes []string) ContinuousEffect {
	return FuncContinuousEffect(LayerType, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		// Map basic land subtypes to mana colors
		colorMap := map[string]Color{
			"Plains": White, "Island": Blue, "Swamp": Black,
			"Mountain": Red, "Forest": Green,
		}
		var newColor Color
		hasNewColor := false
		for _, st := range toSubTypes {
			if c, ok := colorMap[st]; ok {
				newColor = c
				hasNewColor = true
				break
			}
		}
		for _, p := range g.battlefield {
			if slices.ContainsFunc(fromSubTypes, p.HasSubType) {
				p = g.MutablePermanent(p.ID())
				if p == nil {
					continue
				}
				p.SubTypeOverride = toSubTypes
				if hasNewColor && p.HasType(TypeLand) {
					var filtered []Ability
					for _, a := range p.RuntimeAbilities {
						inner := UnwrapAbility(a)
						if _, ok := inner.(*ManaAbility); !ok {
							filtered = append(filtered, a)
						}
					}
					p.RuntimeAbilities = filtered
					p.RuntimeAbilities = append(p.RuntimeAbilities, &grantedByEffect{NewManaAbility(newColor)})
				}
			}
		}
		return nil
	})
}

// CyclopeanTombEffect overrides subtypes of all permanents with Mire counters to Swamp.
func CyclopeanTombEffect() ContinuousEffect {
	return FuncContinuousEffect(LayerType, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		for _, p := range g.battlefield {
			if !p.HasType(TypeLand) || p.Counters[Mire] == 0 {
				continue
			}
			p = g.MutablePermanent(p.ID())
			if p == nil {
				continue
			}
			p.SubTypeOverride = []string{"Swamp"}
			var filtered []Ability
			for _, a := range p.RuntimeAbilities {
				inner := UnwrapAbility(a)
				if _, ok := inner.(*ManaAbility); !ok {
					filtered = append(filtered, a)
				}
			}
			p.RuntimeAbilities = filtered
			p.RuntimeAbilities = append(p.RuntimeAbilities, &grantedByEffect{NewManaAbility(Black)})
		}
		return nil
	})
}

// PreventAllUntaps prevents ALL permanents from untapping during untap steps (Stasis).
func PreventAllUntaps() ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		for _, p := range g.battlefield {
			g.effects.GrantAttr(p.ID(), AttrDoesNotUntap)
		}
		return nil
	})
}

// BoostSelf creates a ContinuousEffect that boosts the source P/T while the SourceCondition passes.
func BoostSelf(power, toughness int, condition SourceCondition) ContinuousEffect {
	var opts []ActiveCondition
	if condition != nil {
		opts = append(opts, WithSourceCondition(condition))
	}
	return FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		src := g.MutablePermanent(sourceID)
		if src == nil {
			return nil
		}
		src.powerBonus += power
		src.toughBonus += toughness
		return nil
	}, opts...)
}

// LimitLandUntaps creates a continuous effect that limits land untaps per turn
// (e.g. Winter Orb). Only active while the source permanent is untapped.
func LimitLandUntaps(limit int) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		if g.effects.Rules.LandUntapMax < 0 || limit < g.effects.Rules.LandUntapMax {
			g.effects.Rules.LandUntapMax = limit
		}
		return nil
	}, SourceUntapped)
}

// LimitCreatureUntaps creates a continuous effect that limits creature untaps per turn
// (e.g. Smoke). Unlike Winter Orb, Smoke does not have a "while untapped" condition.
func LimitCreatureUntaps(limit int) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		if g.effects.Rules.CreatureUntapMax < 0 || limit < g.effects.Rules.CreatureUntapMax {
			g.effects.Rules.CreatureUntapMax = limit
		}
		return nil
	})
}

// AnimateLands creates a continuous effect that turns matching lands into creatures.
// Used by Living Lands (Forests become 1/1 creatures).
func AnimateLands(filter PermanentFilter, power, toughness int) ContinuousEffect {
	return FuncContinuousEffect(LayerType, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		for _, p := range g.FilterBattlefield(filter) {
			g.effects.GrantAttr(p.ID(), AttrIsCreature)
			g.effects.GrantAttr(p.ID(), AttrCanAttack)
			g.effects.GrantAttr(p.ID(), AttrCanBlock)
			g.effects.GrantAttr(p.ID(), AttrHasPowerToughness)
			p = g.MutablePermanent(p.ID())
			if p == nil {
				continue
			}
			p.BasePTOverride = &[2]int{power, toughness}
		}
		return nil
	})
}

// AnimateLandOptions configures a "becomes a creature" effect on a land.
// Per CR 305.7 / 614 layers, an animate effect makes the land also be a
// creature with the specified P/T while keeping its land types/abilities
// intact. SubTypes are added on top of existing subtypes (so a Forest that
// becomes an Elemental is still a Forest), Colors are added on top of
// existing colors, and Keywords are granted at LayerAbility.
type AnimateLandOptions struct {
	Power     int
	Toughness int
	SubTypes  []string
	Colors    []Color
	Keywords  []Attr
}

// applyAnimateLand applies the animate-land mutation to a single permanent.
// The caller is responsible for choosing the layer; this function performs
// the layer-4 (type), layer-5 (color), layer-6 (keyword), and layer-7b
// (P/T) mutations together. Each Apply() cycle resets BasePTOverride,
// SubTypeOverride, ColorOverride, granted keyword attrs, and granted
// runtime abilities, so we recompute them from the card baseline here.
func applyAnimateLand(g *Game, target *Permanent, opts AnimateLandOptions) {
	g.effects.GrantAttr(target.ID(), AttrIsCreature)
	g.effects.GrantAttr(target.ID(), AttrCanAttack)
	g.effects.GrantAttr(target.ID(), AttrCanBlock)
	g.effects.GrantAttr(target.ID(), AttrHasPowerToughness)
	target.BasePTOverride = &[2]int{opts.Power, opts.Toughness}

	if len(opts.SubTypes) > 0 {
		base := target.Card.SubTypes()
		merged := make([]string, 0, len(base)+len(opts.SubTypes))
		seen := map[string]bool{}
		for _, s := range base {
			if !seen[s] {
				merged = append(merged, s)
				seen[s] = true
			}
		}
		if len(target.SubTypeOverride) > 0 {
			for _, s := range target.SubTypeOverride {
				if !seen[s] {
					merged = append(merged, s)
					seen[s] = true
				}
			}
		}
		for _, s := range opts.SubTypes {
			if !seen[s] {
				merged = append(merged, s)
				seen[s] = true
			}
		}
		target.SubTypeOverride = merged
	}

	if len(opts.Colors) > 0 {
		var existing []Color
		if target.ColorOverride != nil {
			existing = *target.ColorOverride
		} else {
			existing = target.Card.ManaCost().Colors()
		}
		merged := make([]Color, 0, len(existing)+len(opts.Colors))
		seen := map[Color]bool{}
		for _, c := range existing {
			if !seen[c] {
				merged = append(merged, c)
				seen[c] = true
			}
		}
		for _, c := range opts.Colors {
			if !seen[c] {
				merged = append(merged, c)
				seen[c] = true
			}
		}
		target.ColorOverride = &merged
	}

	for _, kw := range opts.Keywords {
		g.effects.GrantAttr(target.ID(), kw)
	}
}

// AnimateTargetLand animates a specific land into a creature for the given
// duration (e.g. EndOfTurn for Elemental Uprising). The land remains a land
// (its land subtypes and mana abilities are preserved). Implemented at
// LayerType so type, subtype, color, keyword, and P/T mutations are all
// established in one effect-manager pass.
func AnimateTargetLand(targetID uuid.UUID, opts AnimateLandOptions, duration Duration) ContinuousEffect {
	return TargetEffect(LayerType, duration, targetID, func(g *Game, target *Permanent) error {
		applyAnimateLand(g, target, opts)
		return nil
	})
}

// AnimateLandWhileSourceOnBattlefield animates a specific land for as long as
// the source permanent (the registered source of the effect) remains on the
// battlefield. Used for Awakener Druid: "Target Forest becomes a 4/5 green
// Treefolk creature for as long as Awakener Druid remains on the battlefield."
//
// The source ID is supplied by the effect manager when the effect is
// registered (via SetSourceID); the target ID is the captured land ID at
// the time the ETB effect resolves.
func AnimateLandWhileSourceOnBattlefield(targetID uuid.UUID, opts AnimateLandOptions) ContinuousEffect {
	return FuncContinuousEffect(LayerType, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		target := g.FindPermanent(targetID)
		if target == nil {
			return nil
		}
		applyAnimateLand(g, target, opts)
		return nil
	})
}

// AnimateAttachedLand animates the permanent this aura is attached to into a
// creature for as long as the aura remains attached. Used by Vastwood
// Zendikon ("Enchant land. Enchanted land is a 6/4 green Elemental creature
// with trample. It's still a land."). The effect is gated by SourceAttached
// so that detaching/destroying the aura immediately reverts the land.
func AnimateAttachedLand(opts AnimateLandOptions) ContinuousEffect {
	return AttachedEffect(LayerType, func(g *Game, source, target *Permanent) error {
		applyAnimateLand(g, target, opts)
		return nil
	})
}

// GrantManaAbilityToAttached grants an additional mana ability to the
// permanent this aura is attached to. Used by New Horizons ("Enchanted land
// has '{T}: Add {G}' as additional mana ability."). The granted ability is
// wrapped via WrapGrantedAbility so it is cleared and re-installed on each
// Apply() cycle.
func GrantManaAbilityToAttached(productions ...ManaProduction) ContinuousEffect {
	return AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
		ma := NewMultiManaAbility(productions...)
		ma.SetSource(target.ID())
		ma.SetController(target.ControllerID())
		target.RuntimeAbilities = append(target.RuntimeAbilities, WrapGrantedAbility(ma))
		return nil
	})
}

// SourceHasManaAbilitiesOpponentLandsCouldProduce grants the source one
// tap-for-mana ability for each color an opponent's land could produce.
func SourceHasManaAbilitiesOpponentLandsCouldProduce() ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		source := g.MutablePermanent(sourceID)
		if source == nil {
			return nil
		}
		colors := make(map[Color]bool)
		for _, land := range g.battlefield {
			if land.ControllerID() == source.ControllerID() || !land.HasType(TypeLand) {
				continue
			}
			for _, ability := range land.RuntimeAbilities {
				for _, production := range abilityManaProductions(UnwrapAbility(ability)) {
					if production.Color == AnyColor {
						for _, color := range []Color{White, Blue, Black, Red, Green} {
							colors[color] = true
						}
						continue
					}
					if production.Color != Colorless {
						colors[production.Color] = true
					}
				}
			}
		}
		for _, color := range []Color{White, Blue, Black, Red, Green} {
			if !colors[color] {
				continue
			}
			ability := NewManaAbility(color)
			ability.SetSource(source.ID())
			ability.SetController(source.ControllerID())
			source.RuntimeAbilities = append(source.RuntimeAbilities, WrapGrantedAbility(ability))
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// Animate artifact effects (P/T = mana value)
// ---------------------------------------------------------------------------

// AnimateArtifactScope controls which artifacts are animated and how.
type AnimateArtifactScope struct {
	mode     int // 0=attached, 1=all, 2=target
	targetID uuid.UUID
	duration Duration
}

// Attached is the scope for aura-based artifact animation (Animate Artifact).
var Attached = AnimateArtifactScope{mode: 0}

// ForAll returns a scope that animates all noncreature artifacts globally.
// Used by Titania's Song.
func ForAll(d Duration) AnimateArtifactScope {
	return AnimateArtifactScope{mode: 1, duration: d}
}

// ForTarget returns a scope that animates a specific artifact by ID.
// Used by Xenic Poltergeist.
func ForTarget(id uuid.UUID, d Duration) AnimateArtifactScope {
	return AnimateArtifactScope{mode: 2, targetID: id, duration: d}
}

// AnimateArtifact returns continuous effects that turn noncreature artifacts into
// artifact creatures with P/T equal to their mana value. Returns separate effects
// for LayerType and LayerPT so other effects (e.g. ability stripping) can layer
// between them correctly.
//
// Scope controls the targeting:
//
//	AnimateArtifact(Attached)                          // aura (Animate Artifact)
//	AnimateArtifact(ForAll(WhileOnBattlefield))        // global (Titania's Song)
//	AnimateArtifact(ForTarget(permID, UntilYourNextTurn)) // targeted (Xenic Poltergeist)
func AnimateArtifact(scope AnimateArtifactScope) []ContinuousEffect {
	switch scope.mode {
	case 0: // attached
		return []ContinuousEffect{
			AttachedEffect(LayerType, func(g *Game, source, target *Permanent) error {
				if target.Card.HasType(TypeCreature) {
					return nil
				}
				g.effects.GrantAttr(target.ID(), AttrIsCreature)
				g.effects.GrantAttr(target.ID(), AttrCanAttack)
				g.effects.GrantAttr(target.ID(), AttrCanBlock)
				g.effects.GrantAttr(target.ID(), AttrHasPowerToughness)
				return nil
			}),
			AttachedEffect(LayerPT, func(g *Game, source, target *Permanent) error {
				if target.Card.HasType(TypeCreature) {
					return nil
				}
				cmc := target.Card.ManaCost().CMC()
				target.BasePTOverride = &[2]int{cmc, cmc}
				return nil
			}),
		}
	case 1: // all
		isNoncreatureArtifactByPrint := func(perm *Permanent) bool {
			return perm.HasType(TypeArtifact) && !perm.Card.HasType(TypeCreature)
		}
		return []ContinuousEffect{
			FuncContinuousEffect(LayerType, scope.duration, func(g *Game, _ uuid.UUID) error {
				for _, perm := range g.AllBattlefield() {
					if isNoncreatureArtifactByPrint(perm) {
						g.GrantAttr(perm.ID(), AttrIsCreature)
						g.GrantAttr(perm.ID(), AttrCanAttack)
						g.GrantAttr(perm.ID(), AttrCanBlock)
						g.GrantAttr(perm.ID(), AttrHasPowerToughness)
					}
				}
				return nil
			}),
			FuncContinuousEffect(LayerPT, scope.duration, func(g *Game, _ uuid.UUID) error {
				for _, perm := range g.AllBattlefield() {
					if isNoncreatureArtifactByPrint(perm) {
						cmc := perm.Card.ManaCost().CMC()
						perm = g.MutablePermanent(perm.ID())
						if perm == nil {
							continue
						}
						perm.BasePTOverride = &[2]int{cmc, cmc}
					}
				}
				return nil
			}),
		}
	default: // target
		return []ContinuousEffect{
			TargetEffect(LayerType, scope.duration, scope.targetID, func(g *Game, target *Permanent) error {
				if target.Card.HasType(TypeCreature) {
					return nil
				}
				g.effects.GrantAttr(target.ID(), AttrIsCreature)
				g.effects.GrantAttr(target.ID(), AttrCanAttack)
				g.effects.GrantAttr(target.ID(), AttrCanBlock)
				g.effects.GrantAttr(target.ID(), AttrHasPowerToughness)
				return nil
			}),
			TargetEffect(LayerPT, scope.duration, scope.targetID, func(g *Game, target *Permanent) error {
				if target.Card.HasType(TypeCreature) {
					return nil
				}
				cmc := target.Card.ManaCost().CMC()
				target.BasePTOverride = &[2]int{cmc, cmc}
				return nil
			}),
		}
	}
}

// GrantColorToAll sets the color of all permanents matching the filter.
func GrantColorToAll(color Color, filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerColor, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		colors := []Color{color}
		for _, p := range g.FilterBattlefield(filter) {
			p = g.MutablePermanent(p.ID())
			if p == nil {
				continue
			}
			p.ColorOverride = &colors
		}
		return nil
	})
}

// AllowUnlimitedLandPlays creates a continuous effect that removes the land play limit.
// Used by Fastbond.
func AllowUnlimitedLandPlays() ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		g.effects.Rules.UnlimitedLandPlays = true
		return nil
	})
}

// ManaConversion creates a continuous effect that allows spending one color as another
// (e.g. Sunglasses of Urza: red→white).
func ManaConversion(from, to Color) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		g.effects.Rules.SetManaConversion(from, to)
		return nil
	})
}

// BodyguardContinuous creates a continuous effect for Veteran Bodyguard.
// Only active while the source is untapped.
func BodyguardContinuous() ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		src := g.FindPermanent(sourceID)
		if src == nil {
			return nil
		}
		g.effects.AddCycleReplacement(&bodyguardReplacement{
			replacementBase: replacementBase{sourceID: sourceID},
			controllerID:    src.ControllerID(),
			bodyguardPermID: src.ID(),
		})
		return nil
	}, SourceUntapped)
}

// PersonalIncarnationRedirect creates a continuous effect for Personal Incarnation.
// Redirects ALL damage from a player to Personal Incarnation (unlike Veteran Bodyguard
// which only redirects combat damage).
func PersonalIncarnationRedirect() ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		src := g.FindPermanent(sourceID)
		if src == nil {
			return nil
		}
		g.effects.AddCycleReplacement(&playerDamageRedirectReplacement{
			replacementBase: replacementBase{sourceID: sourceID},
			controllerID:    src.ControllerID(),
			redirectPermID:  src.ID(),
		})
		return nil
	})
}

// ---------------------------------------------------------------------------
// Source conditions (predicates for conditional continuous effects)
// ---------------------------------------------------------------------------

// SourceCondition is a predicate checked by continuous effects to
// decide whether the effect is active. It receives the source permanent and the game.
type SourceCondition func(source *Permanent, g *Game) bool

// WhileSourceAttacking is a SourceCondition that is true only while the source
// permanent is declared as an attacker.
func WhileSourceAttacking(source *Permanent, g *Game) bool {
	return g.combat != nil && g.combat.IsAttacking(source.ID())
}

// WhileSourceUntapped is a SourceCondition true only when the source permanent
// is untapped.
func WhileSourceUntapped(source *Permanent, g *Game) bool {
	return !source.Tapped
}

// WhileControlling is a SourceCondition factory, it creates a SourceCondition that
// ensures the source's controller controls a permanent that matches the filter.
func WhileControlling(filter PermanentFilter) SourceCondition {
	return func(source *Permanent, g *Game) bool {
		return g.AnyBattlefield(And(ControlledBy(source.ControllerID()), filter))
	}
}

// ---------------------------------------------------------------------------
// Damage prevention continuous effect
// ---------------------------------------------------------------------------

// preventDamageRuleContinuous registers a damage prevention rule each Apply()
// cycle. The toFactory receives the source permanent's ID so filters can
// reference "self" dynamically.
type preventDamageRuleContinuous struct {
	from      PermanentFilter
	toFactory func(sourceID uuid.UUID) PermanentFilter
	condition SourceCondition // optional; nil means always active while on battlefield
	effectSource
}

// PreventDamageFromTo creates a continuous effect that prevents all damage
// from permanents matching `from` to permanents matching the filter produced
// by `toFactory(sourceID)`. The toFactory pattern lets filters like
// IsBandedWith reference the source permanent's ID. An optional SourceCondition
// controls when the rule is active (e.g. WhileSourceAttacking for Camel).
func PreventDamageFromTo(from PermanentFilter, toFactory func(uuid.UUID) PermanentFilter, condition ...SourceCondition) ContinuousEffect {
	var cond SourceCondition
	if len(condition) > 0 {
		cond = condition[0]
	}
	return &preventDamageRuleContinuous{
		from:      from,
		toFactory: toFactory,
		condition: cond,
	}
}

func (e *preventDamageRuleContinuous) GetLayer() Layer       { return LayerAbility }
func (e *preventDamageRuleContinuous) GetDuration() Duration { return WhileOnBattlefield }

func (e *preventDamageRuleContinuous) IsActive(g *Game) bool {
	src := g.FindPermanent(e.sourceID)
	if src == nil {
		return false
	}
	if e.condition != nil {
		return e.condition(src, g)
	}
	return true
}

func (e *preventDamageRuleContinuous) Apply(g *Game) error {
	toFilter := e.toFactory(e.sourceID)
	g.effects.AddCycleReplacement(&damagePreventionRuleReplacement{
		replacementBase: replacementBase{sourceID: e.sourceID},
		from:            e.from,
		to:              toFilter,
	})
	return nil
}

// PreventDamageToSourceByRemovingCounters creates a static prevention effect
// that removes one counter from its source for each 1 damage prevented.
func PreventDamageToSourceByRemovingCounters(counterType CounterType) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		g.effects.AddCycleReplacement(&counterDamagePreventionReplacement{
			replacementBase: replacementBase{sourceID: sourceID},
			counterType:     counterType,
		})
		return nil
	})
}

// preventNoncombatDamageToControllerContinuous prevents all noncombat damage
// dealt to the source's controller and to creatures the controller controls.
// Used by Blessed Sanctuary.
type preventNoncombatDamageToControllerContinuous struct {
	effectSource
}

// PreventNoncombatDamageToControllerAndCreatures creates a continuous effect
// that prevents all noncombat damage that would be dealt to the source's
// controller and to creatures that controller controls.
func PreventNoncombatDamageToControllerAndCreatures() ContinuousEffect {
	return &preventNoncombatDamageToControllerContinuous{}
}

func (e *preventNoncombatDamageToControllerContinuous) GetLayer() Layer {
	return LayerAbility
}

func (e *preventNoncombatDamageToControllerContinuous) GetDuration() Duration {
	return WhileOnBattlefield
}

func (e *preventNoncombatDamageToControllerContinuous) IsActive(g *Game) bool {
	return g.FindPermanent(e.sourceID) != nil
}

func (e *preventNoncombatDamageToControllerContinuous) Apply(g *Game) error {
	src := g.FindPermanent(e.sourceID)
	if src == nil {
		return nil
	}
	controller := src.ControllerID()
	g.effects.AddCycleReplacement(&damagePreventionRuleReplacement{
		replacementBase: replacementBase{sourceID: e.sourceID},
		to:              And(IsCreature, ControlledBy(controller)),
		noncombatOnly:   true,
	})
	g.effects.AddCycleReplacement(&damagePreventionRuleReplacement{
		replacementBase: replacementBase{sourceID: e.sourceID},
		toPlayerID:      controller,
		noncombatOnly:   true,
	})
	return nil
}

// ---------------------------------------------------------------------------
// Doppelganger copy effect (mutable state, not convertible to primitives)
// ---------------------------------------------------------------------------

// doppelgangerCopyEffect copies another creature's P/T and keyword abilities
// onto the Doppelganger. Operates at LayerCopy (layer 1).
type doppelgangerCopyEffect struct {
	effectSource
	doppelgangerID uuid.UUID
	copiedName     string // name of the creature being copied
	power          int
	toughness      int
	keywords       []Keyword
}

func (e *doppelgangerCopyEffect) GetLayer() Layer       { return LayerCopy }
func (e *doppelgangerCopyEffect) GetDuration() Duration { return Indefinite }

func (e *doppelgangerCopyEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.doppelgangerID) != nil
}

func (e *doppelgangerCopyEffect) Apply(g *Game) error {
	perm := g.MutablePermanent(e.doppelgangerID)
	if perm == nil {
		return nil
	}
	perm.BasePTOverride = &[2]int{e.power, e.toughness}
	for _, kw := range e.keywords {
		g.effects.GrantAttr(perm.ID(), kw)
	}
	return nil
}

// AddCopyEffect creates a doppelganger copy effect from the target creature.
func (em *EffectManager) AddCopyEffect(doppelgangerID uuid.UUID, target *Permanent) {
	keywords := extractKeywords(target)
	ce := &doppelgangerCopyEffect{
		doppelgangerID: doppelgangerID,
		copiedName:     target.Name(),
		power:          target.Card.Power(),
		toughness:      target.Card.Toughness(),
		keywords:       keywords,
	}
	ce.sourceID = doppelgangerID
	em.effects = append(em.effects, ce)
}

// UpdateCopyEffect updates the copy effect for a doppelganger to copy a new target.
func (em *EffectManager) UpdateCopyEffect(doppelgangerID uuid.UUID, target *Permanent) {
	keywords := extractKeywords(target)
	for _, e := range em.effects {
		ce, ok := e.(*doppelgangerCopyEffect)
		if !ok || ce.doppelgangerID != doppelgangerID {
			continue
		}
		ce.copiedName = target.Name()
		ce.power = target.Card.Power()
		ce.toughness = target.Card.Toughness()
		ce.keywords = keywords
		return
	}
	// No existing effect found, create a new one
	em.AddCopyEffect(doppelgangerID, target)
}

// CopyEffectCurrentName returns the name of the creature currently being copied
// by the doppelganger, or "" if no copy effect exists.
func (em *EffectManager) CopyEffectCurrentName(doppelgangerID uuid.UUID) string {
	for _, e := range em.effects {
		if ce, ok := e.(*doppelgangerCopyEffect); ok && ce.doppelgangerID == doppelgangerID {
			return ce.copiedName
		}
	}
	return ""
}

// extractKeywords returns the keyword attrs currently active on a permanent,
// reading from both baseAttrs and grantedAttrs so that effect-granted keywords
// (e.g. Flying from an equipment) are included in the Doppelganger copy snapshot.
func extractKeywords(p *Permanent) []Keyword {
	var keywords []Keyword
	for a := Attr(1); a < NumAttrs; a++ {
		if IsKeywordAttr(a) && p.HasAttr(a) {
			keywords = append(keywords, a)
		}
	}
	return keywords
}

// ---------------------------------------------------------------------------
// Type-granting continuous effects (CR 614 layer 4) and color-changing
// continuous effects (CR 614 layer 5).
// ---------------------------------------------------------------------------

// addSubTypeAddition appends a subtype to a permanent's additive subtype slice
// without duplicating one already present (in either base subtypes, an
// override, or another addition).
func addSubTypeAddition(p *Permanent, subtype string) {
	if subtype == "" {
		return
	}
	if p.HasSubType(subtype) {
		return
	}
	p.SubTypeAdditions = append(p.SubTypeAdditions, subtype)
}

// GrantSubTypeToControlled grants a subtype to all permanents controlled by
// the source's controller that match the given filter (e.g. Allosaurus
// Shepherd's static "All creatures you control that are Elves are also
// Dinosaurs in addition to their other types"). Operates at LayerType.
func GrantSubTypeToControlled(subtype string, filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerType, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		src := g.FindPermanent(sourceID)
		if src == nil {
			return nil
		}
		for _, p := range g.battlefield {
			if p.ControllerID() != src.ControllerID() {
				continue
			}
			if !filter.Match(p, g) {
				continue
			}
			p = g.MutablePermanent(p.ID())
			if p == nil {
				continue
			}
			addSubTypeAddition(p, subtype)
		}
		return nil
	})
}

// GrantSubTypeToAll grants a subtype to all permanents matching the filter
// (no controller restriction). Operates at LayerType.
func GrantSubTypeToAll(subtype string, filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerType, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		for _, p := range g.FilterBattlefield(filter) {
			p = g.MutablePermanent(p.ID())
			if p == nil {
				continue
			}
			addSubTypeAddition(p, subtype)
		}
		return nil
	})
}

// GrantSubTypeToTarget grants a subtype to a specific permanent for the given
// duration ("in addition to its other types"). Operates at LayerType.
func GrantSubTypeToTarget(targetID uuid.UUID, subtype string, duration Duration) ContinuousEffect {
	return TargetEffect(LayerType, duration, targetID, func(g *Game, target *Permanent) error {
		addSubTypeAddition(target, subtype)
		return nil
	})
}

// BecomesSubType replaces the (creature) subtypes of a specific permanent with
// the given subtype until the given duration expires (e.g. Wishful Merfolk
// "becomes a Human until end of turn"). The card retains all its types
// (creature/etc.) but its printed subtypes are overridden. Operates at
// LayerType.
func BecomesSubType(targetID uuid.UUID, subtype string, duration Duration) ContinuousEffect {
	return TargetEffect(LayerType, duration, targetID, func(g *Game, target *Permanent) error {
		target.SubTypeOverride = []string{subtype}
		target.SubTypeAdditions = nil
		return nil
	})
}

// BecomesColor replaces the colors of a specific permanent with the given
// color until the given duration expires (e.g. Scuttlemutt "Target creature
// becomes the chosen color until end of turn"). Operates at LayerColor.
func BecomesColor(targetID uuid.UUID, color Color, duration Duration) ContinuousEffect {
	return TargetEffect(LayerColor, duration, targetID, func(g *Game, target *Permanent) error {
		colors := []Color{color}
		target.ColorOverride = &colors
		return nil
	})
}

// BecomesColors is the multi-color variant of BecomesColor.
func BecomesColors(targetID uuid.UUID, colors []Color, duration Duration) ContinuousEffect {
	cs := make([]Color, len(colors))
	copy(cs, colors)
	return TargetEffect(LayerColor, duration, targetID, func(g *Game, target *Permanent) error {
		out := make([]Color, len(cs))
		copy(out, cs)
		target.ColorOverride = &out
		return nil
	})
}
