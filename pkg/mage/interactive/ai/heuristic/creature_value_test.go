package heuristic

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
)

func TestCreatureAbilityValuePrefersKeywords(t *testing.T) {
	for _, keyword := range []core.Attr{core.Flying, core.FirstStrike, core.Deathtouch, core.Vigilance} {
		t.Run(keyword.String(), func(t *testing.T) {
			g, p, _ := makeGame()
			g.SetStep(core.PostcombatMain)
			addLands(g, p, "Forest", 2)
			plain := mage.NewCreature("Plain", "{1}{G}", 2, 2)
			better := mage.NewCreature("Keyword", "{1}{G}", 2, 2, mage.WithKeyword(keyword))
			for _, card := range []*mage.Card{plain, better} {
				card.SetOwner(p.PlayerID())
				p.AddToHand(card)
			}
			action := New(ai.MidrangeWeighted).PriorityAction(p, g, 1, true)
			if action.Type != interactive.ActionCastSpell || action.CardID != better.ID() {
				t.Fatalf("expected keyword creature, got %+v", action)
			}
		})
	}
}

func TestCreatureAbilityValueRewardsUnansweredFlying(t *testing.T) {
	values := make([]int, 2)
	for i, keyword := range []core.Attr{core.FirstStrike, core.Reach} {
		g, p, opponent := makeGame()
		g.SetStep(core.PostcombatMain)
		addLands(g, p, "Forest", 2)
		g.AddToBattlefield(makePerm("Blocker", "{1}{G}", 2, 3, opponent.PlayerID(), mage.WithKeyword(keyword)))
		card := mage.NewCreature("Flyer", "{1}{G}", 2, 2, mage.WithKeyword(core.Flying))
		card.SetOwner(p.PlayerID())
		p.AddToHand(card)
		values[i] = creatureAbilityValue(g, p, card)
	}
	if values[0] <= values[1] {
		t.Fatalf("expected unanswered flying to be worth more: ground %d, reach %d", values[0], values[1])
	}
}
