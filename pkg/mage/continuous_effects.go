package mage

import (
	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
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
		ab.controller = target.Controller
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
	return AttachedEffect(LayerControl, func(g *Game, source, target *Permanent) error {
		target.Controller = source.Controller
		return nil
	})
}

// BoostAttachedByCount boosts the attached creature based on the count of
// controlled permanents matching filter. powerFn and toughFn convert the count
// to P/T bonuses (e.g. for Aspect of Wolf: count/2 and (count+1)/2).
func BoostAttachedByCount(filter PermanentFilter, powerFn, toughFn func(int) int) ContinuousEffect {
	return AttachedEffect(LayerPT, func(g *Game, source, target *Permanent) error {
		count := g.CountBattlefield(And(ControlledBy(source.Controller), filter))
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
			ab.controller = p.Controller
			p.RuntimeAbilities = append(p.RuntimeAbilities, &grantedByEffect{ab})
		}
		return nil
	})
}

// GrantTriggeredAbilityToAll grants a triggered ability to all permanents matching filter.
// Each permanent gets its own copy of the triggered ability, allowing individual trigger
// ordering (unlike a single batch trigger). The trigger is constructed from the provided
// event type, optional flag, condition, and effects.
func GrantTriggeredAbilityToAll(eventType EventType, optional bool, cond TriggerConditionData, filter PermanentFilter, effects ...Effect) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		for _, p := range g.battlefield {
			if p.ID() == sourceID {
				continue
			}
			if !filter.Match(p, g) {
				continue
			}
			trig := NewTriggered(eventType, optional, effects...)
			trig.source = p.ID()
			trig.controller = p.Controller
			if cond != nil {
				trig.SetConditionData(cond)
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
		src := g.FindPermanent(sourceID)
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
		src := g.FindPermanent(sourceID)
		if src == nil {
			return nil
		}
		count := g.CountBattlefield(And(ControlledBy(src.Controller), countFilter))
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
			if !p.HasType(TypeCreature) || p.Controller != src.Controller {
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
			if !p.HasType(TypeCreature) || p.ID() == sourceID || p.Controller != src.Controller {
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
			if !p.HasType(TypeCreature) || p.ID() == sourceID || p.Controller != src.Controller {
				continue
			}
			if !filter.Match(p, g) {
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
			if !p.HasType(TypeCreature) || p.Controller != src.Controller {
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
			if !p.HasType(TypeCreature) || p.Controller != src.Controller {
				continue
			}
			if !filter.Match(p, g) {
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
			for _, from := range fromSubTypes {
				if p.HasSubType(from) {
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
					break
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
			if p.HasType(TypeLand) && p.Counters[Mire] > 0 {
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
		src := g.FindPermanent(sourceID)
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
			p.BasePTOverride = &[2]int{power, toughness}
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
			controllerID:    src.Controller,
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
			controllerID:    src.Controller,
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
		return g.AnyBattlefield(And(ControlledBy(source.Controller), filter))
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
	perm := g.FindPermanent(e.doppelgangerID)
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
		if ce, ok := e.(*doppelgangerCopyEffect); ok && ce.doppelgangerID == doppelgangerID {
			ce.copiedName = target.Name()
			ce.power = target.Card.Power()
			ce.toughness = target.Card.Toughness()
			ce.keywords = keywords
			return
		}
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
