package mage

import (
	"fmt"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func damageReplacementBlocked(action Action, g *Game) bool {
	if damage, ok := action.(*DamageToCreatureAction); ok {
		p := g.FindPermanent(damage.PermanentID())
		return p != nil && p.HasAttr(core.AttrDamageCantBePreventedOrRedirected)
	}
	return false
}

func damageActionAmount(action Action) int {
	switch damage := action.(type) {
	case *DamageToPlayerAction:
		return damage.Amount()
	case *DamageToCreatureAction:
		return damage.Amount()
	}
	return 0
}

func damageActionWithAmount(action Action, amount int) Action {
	if amount <= 0 {
		return nil
	}
	switch damage := action.(type) {
	case *DamageToPlayerAction:
		return damage.WithAmount(amount)
	case *DamageToCreatureAction:
		return damage.WithAmount(amount)
	}
	return action
}

type damageReplacementCandidate struct {
	index       int
	replacement ReplacementEffect
}

func (em *EffectManager) applyDamageBatchReplacements(actions []Action, g *Game) []Action {
	// Keep one use per packet so redirection cannot apply a rule twice.
	applied := make([]map[ReplacementEffect]bool, len(actions))
	for i := range applied {
		applied[i] = make(map[ReplacementEffect]bool)
	}
	for {
		var candidates []damageReplacementCandidate
		var chooser Player
		for _, player := range []Player{g.ActivePlayerObj(), g.NonActivePlayerObj(), nil} {
			priority := ReplacementPriorityDefault
			seenShields := make(map[ReplacementEffect]bool)
			for i, action := range actions {
				if action == nil || damageActionAmount(action) <= 0 || damageReplacementBlocked(action, g) {
					continue
				}
				affected := replacementAffectedPlayer(action, g)
				if (player == nil && affected != nil) || (player != nil && (affected == nil || affected.PlayerID() != player.PlayerID())) {
					continue
				}
				for _, replacement := range em.replacementCandidates(action, g, applied[i]) {
					p := getReplacementPriority(replacement)
					if p > priority {
						continue
					}
					if p < priority {
						priority = p
						candidates = nil
						clear(seenShields)
					}
					if _, shield := replacement.(*preventionShieldReplacement); shield {
						if seenShields[replacement] {
							continue
						}
						seenShields[replacement] = true
					}
					candidates = append(candidates, damageReplacementCandidate{i, replacement})
				}
			}
			if len(candidates) > 0 {
				chooser = player
				break
			}
		}
		if len(candidates) == 0 {
			break
		}
		choice := 0
		if len(candidates) > 1 && chooser != nil {
			labels := make([]string, len(candidates))
			for i, candidate := range candidates {
				action := actions[candidate.index]
				labels[i] = replacementLabel(candidate.replacement, i, g)
				if _, shield := candidate.replacement.(*preventionShieldReplacement); !shield {
					labels[i] += fmt.Sprintf("; damage from %s to %s", damageSourceLabel(action, g), damageRecipientLabel(action, g))
				}
			}
			choice = chooser.ChooseMode(labels, "Choose a replacement or prevention effect")
			if choice < 0 || choice >= len(candidates) {
				choice = 0
			}
		}
		selected := candidates[choice]
		if shield, ok := selected.replacement.(*preventionShieldReplacement); ok {
			shield.allocateBatch(actions, applied, g)
		} else {
			applied[selected.index][selected.replacement] = true
			actions[selected.index] = selected.replacement.Replace(actions[selected.index], g)
		}
	}
	return actions
}

func damageActionSource(action Action) DamageSource {
	switch damage := action.(type) {
	case *DamageToPlayerAction:
		return damage.DamageSource()
	case *DamageToCreatureAction:
		return damage.DamageSource()
	}
	return DamageSource{}
}

func damageSourceLabel(action Action, g *Game) string {
	source := damageActionSource(action)
	name := source.Name()
	if name == "" {
		name = action.ActionSource().String()
	}
	if player := g.GetPlayer(source.Controller()); player != nil {
		name += " controlled by " + player.Name()
	}
	return name
}

func damageRecipientLabel(action Action, g *Game) string {
	if player := replacementAffectedPlayer(action, g); player != nil {
		if damage, ok := action.(*DamageToCreatureAction); ok {
			if p := g.FindPermanent(damage.PermanentID()); p != nil {
				return p.Name()
			}
		}
		return player.Name()
	}
	return "unknown recipient"
}

func (r *preventionShieldReplacement) allocateBatch(actions []Action, applied []map[ReplacementEffect]bool, g *Game) {
	indices := []int{}
	sources := make(map[capturedPermanent]bool)
	for i, action := range actions {
		if action == nil || applied[i][r] || damageReplacementBlocked(action, g) || !r.Matches(action, g) {
			continue
		}
		amount := damageActionAmount(action)
		if amount <= 0 {
			continue
		}
		indices = append(indices, i)
		source := damageActionSource(action)
		sources[capturedPermanent{action.ActionSource(), source.Incarnation()}] = true
	}
	// Capacity above the shield amount cannot change a legal allocation bound.
	capacity := make([]int, len(indices)+1)
	for j := len(indices) - 1; j >= 0; j-- {
		capacity[j] = capacity[j+1] + min(damageActionAmount(actions[indices[j]]), r.remaining-capacity[j+1])
	}
	budget := capacity[0]
	player := replacementAffectedPlayer(actions[indices[0]], g)
	for j, i := range indices {
		action := actions[i]
		amount := damageActionAmount(action)
		minimum := max(0, budget-capacity[j+1])
		maximum := min(budget, amount)
		prevented := maximum
		if len(sources) > 1 && minimum < maximum && player != nil {
			prevented = player.ChooseNumber(minimum, maximum, fmt.Sprintf("Choose damage to prevent from %s (packet %d, %d damage)", damageSourceLabel(action, g), i+1, amount))
			if prevented < minimum || prevented > maximum {
				prevented = maximum
			}
		}
		applied[i][r] = true
		r.remaining -= prevented
		budget -= prevented
		actions[i] = damageActionWithAmount(action, amount-prevented)
	}
}
