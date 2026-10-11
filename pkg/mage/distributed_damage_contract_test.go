package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDistributedDamageUnionBatchContract(t *testing.T) {
	g, a, b := newActivationDivisionGame()
	g.SetActivePlayerIndex(0)
	g.SetTurn(1)
	g.SetStep(core.PrecombatMain)
	targets := DeclareAnyNumberDamageTargets("recipients")
	spell := NewSorcery("Distribution contract", "{4}{R}", Spell(DealAllocatedDamage(This, targets, 4), WithTargets(targets)))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Red, 1)
	a.ManaPool().Add(core.Colorless, 4)
	creature := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 9), b.PlayerID())
	walker := g.PutOnBattlefield(NewPlaneswalker("Walker", "", 9), b.PlayerID())
	a.choose = func([]uuid.UUID, int, *Game) map[uuid.UUID]int {
		return map[uuid.UUID]int{creature.ID(): 2, walker.ID(): 1, b.PlayerID(): 1}
	}
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{creature.ID(), walker.ID(), b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.AddPreventionShield(creature.ID(), 1)
	callbacks := 0
	g.damage.SetOnDamageDealt(func(string, string, int, bool) {
		callbacks++
		if creature.Damage != 1 || walker.Counters[core.Loyalty] != 8 || b.Life() != 19 {
			t.Error("observer saw a partial union damage batch")
		}
	})
	g.ResolveStack()
	if callbacks != 3 || a.divisions != 1 {
		t.Fatalf("callbacks=%d divisions=%d", callbacks, a.divisions)
	}
}
