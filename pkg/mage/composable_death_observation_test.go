package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type deathCountConditionForTest int

func (deathCountConditionForTest) stateCondition() {}
func (n deathCountConditionForTest) checkState(g GameReader, _ StateContext) bool {
	return g.CreatureDeaths() >= int(n)
}
func (deathCountConditionForTest) inspectDefinition() definitionNode {
	return definitionNode{Role: "condition", Text: "a creature died this turn"}
}

func TestDeathObservationIndividual(t *testing.T) {
	for _, action := range []string{"destroy", "sacrifice", "graveyard", "token"} {
		t.Run(action, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			ability := func() Ability {
				return WhenEvent(CreatureDies(), GainLifeFor(You, CreaturesDiedThisTurn()), WithInterveningIf(deathCountConditionForTest(1)))
			}
			g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(ability())), a.PlayerID())
			victim := NewCreature("Victim", "", 1, 1, WithAbility(ability()))
			victim.SetOwner(b.PlayerID())
			g.PutOnBattlefield(victim, b.PlayerID())
			p := g.MutablePermanent(victim.ID())
			switch action {
			case "destroy":
				g.DestroyPermanent(p)
			case "sacrifice":
				g.DoSacrifice(p)
			case "graveyard":
				g.PutPermanentIntoGraveyard(p)
			case "token":
				p.IsToken = true
				g.DestroyPermanent(p)
			}
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("death trigger resolved without the stack")
			}
			g.ResolveStack()
			if a.Life() != 21 || b.Life() != 21 || g.CreatureDeaths() != 1 {
				t.Fatalf("life = %d/%d, deaths = %d; want 21/21 and 1", a.Life(), b.Life(), g.CreatureDeaths())
			}
		})
	}
}

func TestDeathObservationSacrificeEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	trigger := NewTriggered(EvtSacrifice, false, GainLifeFor(You, Fixed(1))).SetCondition(
		func(_ *GameEvent, game GameReader, _, _ uuid.UUID) bool { return game.CreatureDeaths() == 1 })
	g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(trigger)), a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DoSacrifice(g.FindPermanent(victim.ID()))
	g.ResolveStack()
	if a.Life() != 21 || g.CreatureDeaths() != 1 {
		t.Fatalf("life = %d, deaths = %d; want 21 and 1", a.Life(), g.CreatureDeaths())
	}
}

func TestDeathObservationStateTrigger(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(WhenState(deathCountConditionForTest(1), Sacrifice(This)))), a.PlayerID())
	victim := NewCreature("Victim", "", 1, 1)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	g.DestroyPermanent(g.FindPermanent(victim.ID()))
	if len(g.triggers.pending) != 1 {
		t.Fatalf("pending state triggers = %d, want 1 before another observation", len(g.triggers.pending))
	}
}

func TestDeathObservationSimultaneousAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ab := WhenEvent(CreatureDies(), GainLifeFor(You, CreaturesDiedThisTurn()), WithInterveningIf(deathCountConditionForTest(2)))
	first := NewCreature("Observer", "", 1, 1, WithAbility(ab))
	second := NewCreature("Victim", "", 1, 1)
	protected := NewCreature("Survivor", "", 1, 1, WithKeyword(Indestructible))
	state := NewEnchantment("State observer", "", WithAbility(NewStateTriggered(false, Sacrifice(This)).SetCondition(
		func(_ *GameEvent, game GameReader, _, _ uuid.UUID) bool { return game.CreatureDeaths() == 1 })))
	g.PutOnBattlefield(state, a.PlayerID())
	for _, card := range []Card{first, second, protected} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	clone := g.Clone()
	clone.DestroyPermanents([]*Permanent{clone.FindPermanent(first.ID()), clone.FindPermanent(second.ID()), clone.FindPermanent(protected.ID())})
	clone.ResolveStack()
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 24 {
		t.Fatalf("clone life = %d, want 24 from two triggers that each see two deaths", got)
	}
	if clone.CreatureDeaths() != 2 || g.CreatureDeaths() != 0 || a.Life() != 20 || g.FindPermanent(first.ID()) == nil {
		t.Fatal("simultaneous deaths changed the original game or counted prevented destruction")
	}
	if clone.FindPermanent(state.ID()) == nil {
		t.Fatal("state trigger observed a partial simultaneous group")
	}
}
