package mage

import (
	. "github.com/mage/mage/pkg/mage/core"
	"github.com/google/uuid"
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
		g.Effects.GrantAttr(target.ID(), kw)
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
		g.Effects.RevokeAttr(target.ID(), kw)
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

// GrantActivatedAbilityToAttached grants an activated ability to the attached creature.
func GrantActivatedAbilityToAttached(effect Effect, cost Cost, at AttachType) ContinuousEffect {
	return AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
		ab := NewActivatedAbility(effect, cost)
		ab.source = target.ID()
		ab.controller = target.Controller
		target.RuntimeAbilities = append(target.RuntimeAbilities, &grantedByEffect{ab})
		return nil
	})
}

// PreventAttachedFromUntapping creates a continuous effect preventing the attached creature from untapping.
func PreventAttachedFromUntapping(at AttachType) ContinuousEffect {
	return AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
		g.Effects.GrantAttr(target.ID(), AttrDoesNotUntap)
		return nil
	})
}

// PreventAttachedFromAttacking creates a continuous effect preventing the attached creature from attacking.
func PreventAttachedFromAttacking(at AttachType) ContinuousEffect {
	return AttachedEffect(LayerAbility, func(g *Game, source, target *Permanent) error {
		g.Effects.RevokeAttr(target.ID(), AttrCanAttack)
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

// BoostAttachedByForestCount boosts the attached creature by Forests controlled.
func BoostAttachedByForestCount() ContinuousEffect {
	return AttachedEffect(LayerPT, func(g *Game, source, target *Permanent) error {
		forests := g.CountBattlefield(And(ControlledBy(source.Controller), IsLand, HasSubType("Forest")))
		target.powerBonus += forests / 2
		target.toughBonus += (forests + 1) / 2
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
		g.Effects.GrantAttr(target.ID(), kw)
		return nil
	})
}

// KeywordReplacement replaces one keyword with another on a target permanently
// (e.g. swampwalk -> forestwalk via Sleight of Mind / Magical Hack).
func KeywordReplacement(targetID uuid.UUID, from, to Keyword) ContinuousEffect {
	return TargetEffect(LayerAbility, Indefinite, targetID, func(g *Game, target *Permanent) error {
		g.Effects.RevokeAttr(target.ID(), from)
		g.Effects.GrantAttr(target.ID(), to)
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
		g.Effects.GrantAttr(target.ID(), AttrIsCreature)
		g.Effects.GrantAttr(target.ID(), AttrCanAttack)
		g.Effects.GrantAttr(target.ID(), AttrCanBlock)
		g.Effects.GrantAttr(target.ID(), AttrHasPowerToughness)
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

// PreventBlockingUntilEndOfCombat creates an EndOfCombat-scoped continuous effect
// that revokes AttrCanBlock from a specific creature. Re-fires on each Apply() cycle
// (surviving grantedAttrs reset) and expires at EndCombat via RemoveEndOfCombat().
// Use this instead of the imperative preventBlock map so block-prevention goes through
// the attr system like every other capability restriction.
func PreventBlockingUntilEndOfCombat(permID uuid.UUID) ContinuousEffect {
	return TargetEffect(LayerAbility, EndOfCombat, permID, func(g *Game, target *Permanent) error {
		g.Effects.RevokeAttr(target.ID(), AttrCanBlock)
		return nil
	})
}

// ---------------------------------------------------------------------------
// FuncContinuousEffect-based effects (source on battlefield)
// ---------------------------------------------------------------------------

// GrantActivatedAbilityToAll grants an activated ability to all creatures matching filter.
func GrantActivatedAbilityToAll(effect Effect, cost Cost, filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		for _, p := range g.Battlefield {
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
		g.Effects.RevokeAttr(sourceID, AttrCanAttack)
		return nil
	})
}

// BoostAllCreatures creates a continuous effect that boosts all matching creatures
// except the source (typical lord behavior).
func BoostAllCreatures(power, toughness int, filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		for _, p := range g.Battlefield {
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
		for _, p := range g.Battlefield {
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

// PowerEqualsCount creates a continuous effect where the source creature gets
// P/T bonus equal to count of permanents matching countFilter.
func PowerEqualsCount(countFilter PermanentFilter) ContinuousEffect {
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

// GrantKeywordToAll grants a keyword ability to all matching creatures (excluding source).
func GrantKeywordToAll(kw Keyword, filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		for _, p := range g.Battlefield {
			if !p.HasType(TypeCreature) || p.ID() == sourceID {
				continue
			}
			if !filter.Match(p, g) {
				continue
			}
			g.Effects.GrantAttr(p.ID(), kw)
		}
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
		for _, p := range g.Battlefield {
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
			g.Effects.GrantAttr(p.ID(), AttrDoesNotUntap)
		}
		return nil
	})
}

// IncreaseSpellCostForColor is a continuous effect that increases the cost of
// spells of a given color (e.g. Gloom makes white spells cost {3} more).
func IncreaseSpellCostForColor(color Color, amount int) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		g.Effects.rules.spellCostIncrease[color] += amount
		return nil
	})
}

// ReduceSpellCostForColor is a continuous effect that reduces the cost of
// spells of a given color (e.g. "Blue spells cost {1} less to cast").
func ReduceSpellCostForColor(color Color, amount int) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		g.Effects.rules.spellCostReduction[color] += amount
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
		for _, p := range g.Battlefield {
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
		for _, p := range g.Battlefield {
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
		for _, p := range g.Battlefield {
			g.Effects.GrantAttr(p.ID(), AttrDoesNotUntap)
		}
		return nil
	})
}

// BoostSelf creates a ContinuousEffect that boosts the source P/T while the SourceCondition passes.
func BoostSelf(power, toughness int, condition SourceCondition) ContinuousEffect {
	return FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		src := g.FindPermanent(sourceID)
		if src == nil {
			return nil
		}
		if condition != nil && !condition(src, g) {
			return nil
		}
		src.powerBonus += power
		src.toughBonus += toughness
		return nil
	})
}

// LimitLandUntaps creates a continuous effect that limits land untaps per turn
// (e.g. Winter Orb). Only active while the source permanent is untapped.
func LimitLandUntaps(limit int) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		if g.Effects.rules.landUntapLimit < 0 || limit < g.Effects.rules.landUntapLimit {
			g.Effects.rules.landUntapLimit = limit
		}
		return nil
	}, SourceUntapped)
}

// AnimateLands creates a continuous effect that turns matching lands into creatures.
// Used by Living Lands (Forests become 1/1 creatures).
func AnimateLands(filter PermanentFilter, power, toughness int) ContinuousEffect {
	return FuncContinuousEffect(LayerType, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		for _, p := range g.FilterBattlefield(filter) {
			g.Effects.GrantAttr(p.ID(), AttrIsCreature)
			g.Effects.GrantAttr(p.ID(), AttrCanAttack)
			g.Effects.GrantAttr(p.ID(), AttrCanBlock)
			g.Effects.GrantAttr(p.ID(), AttrHasPowerToughness)
			p.BasePTOverride = &[2]int{power, toughness}
		}
		return nil
	})
}

// AllowUnlimitedLandPlays creates a continuous effect that removes the land play limit.
// Used by Fastbond.
func AllowUnlimitedLandPlays() ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		g.Effects.rules.unlimitedLandPlays = true
		return nil
	})
}

// ManaConversion creates a continuous effect that allows spending one color as another
// (e.g. Sunglasses of Urza: red→white).
func ManaConversion(from, to Color) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
		g.Effects.SetManaConversion(from, to)
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
		g.Effects.Damage.SetBodyguard(src.Controller, src.ID())
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
		g.Effects.Damage.SetPlayerDamageRedirect(src.Controller, src.ID())
		return nil
	})
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
		g.Effects.GrantAttr(perm.ID(), kw)
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
	seen := make(map[Keyword]bool)
	var keywords []Keyword
	add := func(a Attr) {
		if IsKeywordAttr(a) && !seen[a] && p.HasAttr(a) {
			seen[a] = true
			keywords = append(keywords, a)
		}
	}
	for a := range p.baseAttrs {
		add(a)
	}
	for a := range p.grantedAttrs {
		add(a)
	}
	return keywords
}
