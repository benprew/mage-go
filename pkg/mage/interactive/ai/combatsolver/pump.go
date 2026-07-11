package combatsolver

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
)

type selfPumpInfo struct {
	power     int
	toughness int
	cost      int
}

func applyCombatSelfPump(g *mage.Game, playerID uuid.UUID, blocks []mage.BlockAssignment) {
	manaBudget := g.HypotheticalMana(playerID)
	if manaBudget <= 0 {
		return
	}
	for _, block := range blocks {
		if manaBudget <= 0 {
			return
		}
		blocker := g.FindPermanent(block.BlockerID)
		attacker := g.FindPermanent(block.AttackerID)
		if blocker == nil || attacker == nil || blocker.ControllerID() != playerID {
			continue
		}
		info, ok := selfPumpAbility(blocker)
		if !ok || info.cost <= 0 {
			continue
		}
		uses := usefulPumpActivations(g, blocker, attacker, info, manaBudget/info.cost)
		if uses <= 0 {
			continue
		}
		eff := mage.TemporaryBoost(blocker.ID(), info.power*uses, info.toughness*uses)
		eff.SetSourceID(blocker.ID())
		g.AddContinuousEffect(eff)
		g.ApplyContinuousEffects()
		manaBudget -= uses * info.cost
	}
}

func selfPumpAbility(perm *mage.Permanent) (selfPumpInfo, bool) {
	for _, ability := range perm.RuntimeAbilities {
		ab, ok := mage.UnwrapAbility(ability).(mage.ActivatedAbility)
		if !ok || len(ab.Targets()) > 0 {
			continue
		}
		info := selfPumpInfo{cost: 0}
		for _, cost := range ab.Costs() {
			if mc, ok := cost.(*mage.ManaCostPayment); ok {
				info.cost += mc.MC.CMC()
			} else {
				info.cost = 0
				break
			}
		}
		if info.cost <= 0 {
			continue
		}
		for _, effect := range ab.Effects() {
			props := effect.Properties()
			if props.Outcome != mage.OutcomeBenefit {
				continue
			}
			info.power += max(props.PowerBoost, 0)
			info.toughness += max(props.ToughnessBoost, 0)
		}
		if info.power > 0 || info.toughness > 0 {
			return info, true
		}
	}
	return selfPumpInfo{}, false
}

func usefulPumpActivations(g *mage.Game, blocker, attacker *mage.Permanent, info selfPumpInfo, maxUses int) int {
	if maxUses <= 0 {
		return 0
	}
	uses := 0
	for uses < maxUses {
		power := blocker.CurrentPower(g) + info.power*uses
		toughness := blocker.CurrentToughness(g) + info.toughness*uses
		killsAttacker := power >= attacker.CurrentToughness(g)
		survivesAttacker := toughness > attacker.CurrentPower(g)
		if killsAttacker && survivesAttacker {
			return uses
		}
		uses++
	}
	return maxUses
}
