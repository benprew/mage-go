package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestOpponentOrPlaneswalkerTarget(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	walkers := []*Permanent{
		g.PutOnBattlefield(NewPlaneswalker("Own", "", 10), a.PlayerID()),
		g.PutOnBattlefield(NewPlaneswalker("Other", "", 10), b.PlayerID()),
	}
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 1, 10), b.PlayerID())
	target := DeclareOpponentOrPlaneswalkerTarget("recipient")
	card := NewInstant("Test", "", WithAbility(Spell(DealDamageFrom(This, target, Fixed(2)), WithTargets(target))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	for _, controller := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
		possible := target.spec.Possible(controller, card, g)
		if slices.Contains(possible, controller) || slices.Contains(possible, creature.ID()) || len(possible) != 3 {
			t.Fatalf("wrong candidates: %v", possible)
		}
		for _, walker := range walkers {
			if !slices.Contains(possible, walker.ID()) {
				t.Fatal("planeswalker control restricted")
			}
		}
	}
	g.MutablePermanent(walkers[0].ID()).GrantBaseAttr(core.Shroud)
	g.PhaseOut(walkers[1].ID())
	if got := target.spec.Possible(a.PlayerID(), card, g); len(got) != 1 || got[0] != b.PlayerID() {
		t.Fatalf("illegal planeswalkers included: %v", got)
	}
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "missing declaration", Card: NewInstant("Invalid", "", WithAbility(Spell(DealDamageFrom(This, target, Fixed(2))))), WantErr: true},
		{Name: "empty label", Card: NewInstant("Invalid", "", WithAbility(Spell(DealDamageFrom(This, DeclareOpponentOrPlaneswalkerTarget(""), Fixed(2))))), WantErr: true},
	})
}

func TestOpponentOrPlaneswalkerTargetCastingAndCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	target := DeclareOpponentOrPlaneswalkerTarget("recipient")
	card := NewInstant("Test", "{R}", WithAbility(Spell(DealDamageFrom(This, target, Fixed(2)), WithTargets(target))))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{a.PlayerID()}, 0); err == nil || a.ManaPool().TotalMana() != 1 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("invalid target did not roll back")
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	g.ResolveTopOfStack()
	if b.Life() != 20 {
		t.Fatal("copy kept a target that is now its controller")
	}
	copied = g.CopySpellOnStack(card.ID(), b.PlayerID(), true)
	if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != a.PlayerID() {
		t.Fatal("copy did not select its opponent")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 || clone.GetPlayer(a.PlayerID()).Life() != 18 || clone.GetPlayer(b.PlayerID()).Life() != 18 {
		t.Fatal("clone or target identity")
	}
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 18 {
		t.Fatal("copy changed original target")
	}
}
