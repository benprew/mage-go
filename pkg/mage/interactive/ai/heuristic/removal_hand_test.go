package heuristic

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
)

func TestEarlyRemovalUsesBoardAndHandValue(t *testing.T) {
	for _, personality := range []ai.WeightedPersonality{ai.AggroWeighted, ai.MidrangeWeighted, ai.BurnWeighted} {
		for _, step := range []core.PhaseStep{core.Upkeep, core.PostcombatMain} {
			for _, name := range []string{"Swords to Plowshares", "Lightning Bolt"} {
				g, p, opponent := makeGame()
				g.SetTurn(4)
				g.SetStep(step)
				addLands(g, p, "Plains", 1)
				addLands(g, p, "Mountain", 1)
				bird := mage.NewPermanent(createCard(t, "Birds of Paradise", opponent.PlayerID()), opponent.PlayerID())
				bird.RevokeBaseAttr(core.AttrSummonSick)
				g.AddToBattlefield(bird)
				p.AddToHand(createCard(t, name, p.PlayerID()))
				action := New(personality).PriorityAction(p, g, 1, step.IsMainPhase())
				want := interactive.ActionPass
				if name == "Lightning Bolt" {
					want = interactive.ActionCastSpell
				}
				if action.Type != want {
					t.Fatalf("%s, %s, %s: got %+v, want %v", personality.Name, step, name, action, want)
				}
			}
		}
	}
}

func TestEarlyRemovalPrefersBoltWhenBothAreHeld(t *testing.T) {
	g, p, opponent := makeGame()
	g.SetTurn(4)
	g.SetStep(core.PostcombatMain)
	addLands(g, p, "Plains", 1)
	addLands(g, p, "Mountain", 1)
	g.AddToBattlefield(makePerm("Bird", "{G}", 0, 1, opponent.PlayerID(), mage.WithKeyword(core.Flying), mage.WithManaAbility(core.AnyColor)))
	for _, name := range []string{"Swords to Plowshares", "Lightning Bolt"} {
		p.AddToHand(createCard(t, name, p.PlayerID()))
	}
	action := New(ai.AggroWeighted).PriorityAction(p, g, 1, true)
	if action.Type != interactive.ActionCastSpell || action.CardName != "Lightning Bolt" {
		t.Fatalf("expected Bolt and retained Swords, got %+v", action)
	}
}
