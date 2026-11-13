package heuristic

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

func creatureAbilityValue(g *mage.Game, p mage.Player, card *mage.Card) int {
	if !card.HasType(core.TypeCreature) {
		return 0
	}
	perm := mage.NewPermanent(card, p.PlayerID())
	stats := card.Power()*eval.PowerWeight + card.Toughness()*eval.ToughnessWeight
	if perm.HasAttr(core.AttrSummonSick) && !perm.HasKeyword(core.Haste) {
		stats /= 2
	}
	value := eval.EvalCreatureInGameNoTapPenalty(perm, g) - stats
	return value + unansweredEvasionValue(g, perm, card.Power())
}

func unansweredEvasionValue(g *mage.Game, perm *mage.Permanent, power int) int {
	defender := g.GetOpponent(perm.ControllerID())
	if defender == nil || perm.HasKeyword(core.Defender) || !perm.HasAttr(core.AttrCanAttack) {
		return 0
	}
	blockers := 0
	for _, blocker := range g.AllBattlefield() {
		if blocker.ControllerID() == defender.PlayerID() && blocker.HasType(core.TypeCreature) && blocker.HasAttr(core.AttrCanBlock) && mage.CanBlock(blocker, perm, g) {
			blockers++
		}
	}
	if blockers == 0 || (blockers < 2 && perm.HasKeyword(core.Menace)) || mage.HasLandwalkEvasion(perm, defender.PlayerID(), g) {
		return max(power, 0)
	}
	return 0
}
