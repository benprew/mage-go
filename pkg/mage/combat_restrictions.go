package mage

import (
	"strings"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// Combat-restriction storage on the EffectManager. These maps are reset each
// Apply() cycle and re-populated by continuous effects, mirroring how
// blockPairRestrictions work.
//
// Semantics:
//   - cantBeBlockedExceptByRules[attackerID]: ALL filters in the slice must
//     match a candidate blocker for the block to be legal (CR 509.1b).
//   - canBlockOnlyRules[blockerID]: ALL filters must match a candidate
//     attacker for this blocker to be allowed to block it.
//   - cantBeBlockedByFewerThan[attackerID]: minimum number of blockers
//     required (e.g. Goblin Goon "can't be blocked except by three or more
//     creatures"). Enforced post-declaration: if fewer than N creatures are
//     declared as blockers, ALL of them are removed from this attacker.
//   - mustBeBlockedIfAble[attackerID]: CR 509.1c — defender must designate
//     at least one able blocker for this attacker.

// AddCantBeBlockedExceptBy registers a per-attacker block restriction.
// A potential blocker must satisfy filter (and all other filters registered
// for this attacker) for the block to be legal.
func (em *EffectManager) AddCantBeBlockedExceptBy(attackerID uuid.UUID, filter PermanentFilter) {
	if em.cantBeBlockedExceptByRules == nil {
		em.cantBeBlockedExceptByRules = make(map[uuid.UUID][]PermanentFilter)
	}
	em.cantBeBlockedExceptByRules[attackerID] = append(em.cantBeBlockedExceptByRules[attackerID], filter)
}

// AddCanBlockOnly registers a per-blocker restriction. The blocker can only
// block attackers that satisfy filter (and any other filters registered for
// this blocker).
func (em *EffectManager) AddCanBlockOnly(blockerID uuid.UUID, filter PermanentFilter) {
	if em.canBlockOnlyRules == nil {
		em.canBlockOnlyRules = make(map[uuid.UUID][]PermanentFilter)
	}
	em.canBlockOnlyRules[blockerID] = append(em.canBlockOnlyRules[blockerID], filter)
}

// AddMinBlockers raises the minimum number of blockers required for attackerID
// to the maximum of the existing minimum and n. CR equivalent: "can't be
// blocked except by N or more creatures" (e.g. Goblin Goon).
func (em *EffectManager) AddMinBlockers(attackerID uuid.UUID, n int) {
	if em.minBlockers == nil {
		em.minBlockers = make(map[uuid.UUID]int)
	}
	if cur, ok := em.minBlockers[attackerID]; !ok || n > cur {
		em.minBlockers[attackerID] = n
	}
}

// MinBlockers returns the minimum number of blockers required to block
// attackerID, or 0 if no minimum has been registered.
func (em *EffectManager) MinBlockers(attackerID uuid.UUID) int {
	return em.minBlockers[attackerID]
}

// AddAttackCost registers a cost that permID's controller must pay to declare
// it as an attacker (CR 508.1e, "can't attack unless you pay/sacrifice ...").
func (em *EffectManager) AddAttackCost(permID uuid.UUID, cost Cost) {
	if em.attackCosts == nil {
		em.attackCosts = make(map[uuid.UUID][]Cost)
	}
	em.attackCosts[permID] = append(em.attackCosts[permID], cost)
}

// AttackCosts returns the costs required to declare permID as an attacker,
// or nil if attacking it is free.
func (em *EffectManager) AttackCosts(permID uuid.UUID) []Cost {
	return em.attackCosts[permID]
}

// resetCombatRestrictions clears all per-cycle restriction maps. Called from
// EffectManager.Apply alongside blockPairRestrictions.
func (em *EffectManager) resetCombatRestrictions() {
	clear(em.cantBeBlockedExceptByRules)
	clear(em.canBlockOnlyRules)
	clear(em.minBlockers)
	clear(em.attackCosts)
}

// passesCombatRestrictions returns true if blocker is allowed to block
// attacker under the registered filter-based restrictions. (Numeric
// minimum-blocker checks are enforced separately, post-declaration.)
func (em *EffectManager) passesCombatRestrictions(blocker, attacker *Permanent, g *Game) bool {
	if filters, ok := em.cantBeBlockedExceptByRules[attacker.ID()]; ok {
		for _, f := range filters {
			if !f.Match(blocker, g) {
				return false
			}
		}
	}
	if filters, ok := em.canBlockOnlyRules[blocker.ID()]; ok {
		for _, f := range filters {
			if !f.Match(attacker, g) {
				return false
			}
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Continuous-effect constructors
// ---------------------------------------------------------------------------

// SourceCantBeBlockedExceptBy creates a continuous effect that restricts which
// creatures can block the source (CR 509.1b). E.g. Gingerbrute: "can't be
// blocked except by creatures with haste."
func SourceCantBeBlockedExceptBy(filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		g.effects.AddCantBeBlockedExceptBy(sourceID, filter)
		return nil
	})
}

// SourceCanBlockOnly creates a continuous effect that restricts the source so
// that it can only block attackers matching filter (e.g. Rishadan Airship
// "can block only creatures with flying").
func SourceCanBlockOnly(filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		g.effects.AddCanBlockOnly(sourceID, filter)
		return nil
	})
}

// SourceCantBeBlockedByFewerThan creates a continuous effect that requires at
// least n blockers to block the source (e.g. Goblin Goon "can't be blocked
// except by three or more creatures"). If fewer than n creatures block it
// during declaration, all of them are removed from the block.
func SourceCantBeBlockedByFewerThan(n int) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		g.effects.AddMinBlockers(sourceID, n)
		return nil
	})
}

// SourceCantAttackUnlessPays creates a continuous effect registering an attack
// cost on the source (CR 508.1e): the source can't attack unless its controller
// pays cost as attackers are declared. E.g. Leviathan: "Leviathan can't attack
// unless you sacrifice two Islands."
func SourceCantAttackUnlessPays(cost Cost) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		g.effects.AddAttackCost(sourceID, cost)
		return nil
	})
}

// AttachedCantAttackUnlessPays creates a continuous effect for an Aura whose
// enchanted creature has a cost to attack.
func AttachedCantAttackUnlessPays(cost Cost) ContinuousEffect {
	return AttachedEffect(LayerAbility, func(g *Game, _, target *Permanent) error {
		g.effects.AddAttackCost(target.ID(), cost)
		return nil
	})
}

// PayAttackCosts pays the CR 508.1e costs registered for atk. When prompt is
// true the controller is first asked whether to pay (declining is legal — the
// creature simply isn't declared as an attacker); search clones pass false
// because passing the attacker to ExecuteAttackers already encodes that
// decision. Returns false if any cost was declined or could not be paid, in
// which case the creature must not be declared.
func (g *Game) PayAttackCosts(atk *Permanent, controller uuid.UUID, prompt bool) bool {
	costs := g.effects.AttackCosts(atk.ID())
	if len(costs) == 0 {
		return true
	}
	preparedCosts := g.prepareActionCosts(costs, nil, 0)
	payment, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller: controller,
		SourceID:   atk.ID(),
		Costs:      preparedCosts,
		Hint: AutoTapHint{
			ReservedSources: []uuid.UUID{atk.ID()},
		},
	})
	if err != nil {
		return false
	}
	if prompt {
		p := g.GetPlayer(controller)
		if p == nil {
			return false
		}
		texts := make([]string, len(preparedCosts))
		for i, c := range preparedCosts {
			texts[i] = c.Text()
		}
		if !p.ChooseMayAbility(strings.Join(texts, " and ") + " for " + atk.Name() + " to attack") {
			return false
		}
	}
	return payment.Commit() == nil
}

// TargetCantBeBlockedExceptBy creates a target-scoped continuous effect that
// applies a "can't be blocked except by X" restriction to a specific
// permanent for the given duration (e.g. Ghirapur Guide).
func TargetCantBeBlockedExceptBy(targetID uuid.UUID, filter PermanentFilter, duration Duration) ContinuousEffect {
	return TargetEffect(LayerAbility, duration, targetID, func(g *Game, target *Permanent) error {
		g.effects.AddCantBeBlockedExceptBy(target.ID(), filter)
		return nil
	})
}

// TargetMustBeBlockedIfAble creates a target-scoped continuous effect granting
// AttrMustBeBlockedIfAble (CR 509.1c) until the given duration expires. Used
// by "Target creature must be blocked this turn if able" cards (Enlarge,
// Irresistible Prey).
func TargetMustBeBlockedIfAble(targetID uuid.UUID, duration Duration) ContinuousEffect {
	return TargetEffect(LayerAbility, duration, targetID, func(g *Game, target *Permanent) error {
		g.effects.GrantAttr(target.ID(), AttrMustBeBlockedIfAble)
		return nil
	})
}

// PreventBlockByPowerLessThanSource creates a continuous effect: any creature
// (matching blockerFilter) whose power is less than the source's power can't
// block any creature whose controller matches attackerOwnerSelf (true means
// "you control"). Used by Champion of Lambholt: "Creatures with power less
// than ~'s power can't block creatures you control."
func PreventBlockByPowerLessThanSource(blockerFilter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		src := g.FindPermanent(sourceID)
		if src == nil {
			return nil
		}
		threshold := src.CurrentPower(g)
		controllerID := src.ControllerID()
		for _, blocker := range g.zones.battlefield {
			if !blocker.HasType(TypeCreature) {
				continue
			}
			if !blockerFilter.Match(blocker, g) {
				continue
			}
			if blocker.CurrentPower(g) >= threshold {
				continue
			}
			for _, atk := range g.zones.battlefield {
				if atk.ControllerID() != controllerID {
					continue
				}
				if !atk.HasType(TypeCreature) {
					continue
				}
				g.effects.PreventBlockPair(blocker.ID(), atk.ID())
			}
		}
		return nil
	})
}

// PreventAttackingIfDefenderControlsMore creates a continuous effect that
// revokes AttrCanAttack from the source while the defending player controls
// strictly more permanents matching filter than the source's controller does.
// Used by Goblin Goon: "Goblin Goon can't attack if defending player controls
// as many or more creatures than you do." (Goblin Goon's exact text is
// "more"; this constructor expresses the more-strict variant.)
func PreventAttackingIfDefenderControlsMore(filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		src := g.FindPermanent(sourceID)
		if src == nil {
			return nil
		}
		mine := g.CountBattlefield(And(ControlledBy(src.ControllerID()), filter))
		other := g.NonActivePlayerObj()
		// Source's defender (the opposing player from the source controller's POV).
		var defID uuid.UUID
		for _, pl := range g.players {
			if pl.PlayerID() != src.ControllerID() {
				defID = pl.PlayerID()
				break
			}
		}
		_ = other
		theirs := g.CountBattlefield(And(ControlledBy(defID), filter))
		if theirs > mine {
			g.effects.RevokeAttr(sourceID, AttrCanAttack)
		}
		return nil
	})
}

// PreventAttackingIfDefenderControlsAsManyOrMore is the strict variant of
// PreventAttackingIfDefenderControlsMore. It revokes AttrCanAttack from
// the source unless the source's controller controls strictly more
// permanents matching filter than the defending player does. Used by
// Goblin Goon: "Goblin Goon can't attack unless you control more
// creatures than defending player." (Equality also revokes; Oracle
// requires "more" from your side.)
func PreventAttackingIfDefenderControlsAsManyOrMore(filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		src := g.FindPermanent(sourceID)
		if src == nil {
			return nil
		}
		mine := g.CountBattlefield(And(ControlledBy(src.ControllerID()), filter))
		var defID uuid.UUID
		for _, pl := range g.players {
			if pl.PlayerID() != src.ControllerID() {
				defID = pl.PlayerID()
				break
			}
		}
		theirs := g.CountBattlefield(And(ControlledBy(defID), filter))
		if mine <= theirs {
			g.effects.RevokeAttr(sourceID, AttrCanAttack)
		}
		return nil
	})
}

// PreventBlockingIfAttackerControlsAsManyOrMore is the block-side analogue
// of PreventAttackingIfDefenderControlsAsManyOrMore. It revokes
// AttrCanBlock from the source unless the source's controller controls
// strictly more permanents matching filter than the attacking player
// does. Used by Goblin Goon: "Goblin Goon can't block unless you
// control more creatures than attacking player." Treats the opposing
// 2-player seat as the attacking-player.
func PreventBlockingIfAttackerControlsAsManyOrMore(filter PermanentFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		src := g.FindPermanent(sourceID)
		if src == nil {
			return nil
		}
		mine := g.CountBattlefield(And(ControlledBy(src.ControllerID()), filter))
		var attID uuid.UUID
		for _, pl := range g.players {
			if pl.PlayerID() != src.ControllerID() {
				attID = pl.PlayerID()
				break
			}
		}
		theirs := g.CountBattlefield(And(ControlledBy(attID), filter))
		if mine <= theirs {
			g.effects.RevokeAttr(sourceID, AttrCanBlock)
		}
		return nil
	})
}

// PowerLessOrEqual returns a PermanentFilter matching creatures whose current
// power is <= n. Useful for "can't be blocked by creatures with power N or
// less" (Ghirapur Guide).
func PowerLessOrEqual(n int) PermanentFilter {
	return NewPermanentFilter("power "+itoa(n)+" or less", func(p *Permanent, g *Game) bool {
		if !p.HasType(TypeCreature) {
			return false
		}
		return p.CurrentPower(g) <= n
	})
}

// PowerGreaterThan returns a PermanentFilter matching creatures whose current
// power is > n. The natural complement for "can't be blocked except by
// creatures with power N+1 or greater" expressions.
func PowerGreaterThan(n int) PermanentFilter {
	return NewPermanentFilter("power greater than "+itoa(n), func(p *Permanent, g *Game) bool {
		if !p.HasType(TypeCreature) {
			return false
		}
		return p.CurrentPower(g) > n
	})
}

// itoa converts a small int to a decimal string without importing strconv
// into this hot-path file. Handles negatives.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
