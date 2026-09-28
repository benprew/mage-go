package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDefendingPlayerControlAttackRestriction(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetActivePlayerIndex(0)
	card := NewCreature("Attacker", "", 2, 2, WithAbility(Static(CanAttackOnlyIfDefendingPlayerControls(HasSubType("Island")))))
	attacker := g.PutOnBattlefield(card, a.PlayerID())
	land := NewLand("Island", WithSubTypes("Island"))
	land.SetOwner(a.PlayerID())
	island := g.PutOnBattlefield(land, a.PlayerID())
	check := func(want bool) {
		t.Helper()
		g.effects.Apply(g)
		if got := g.FindPermanent(attacker.ID()).HasAttr(AttrCanAttack); got != want {
			t.Fatalf("can attack = %v, want %v", got, want)
		}
	}
	check(false)
	g.RemoveFromBattlefield(island)
	island = g.PutOnBattlefield(land, b.PlayerID())
	check(true)
	g.PhaseOut(island.ID())
	check(false)
	g.PhaseIn([]uuid.UUID{island.ID()})
	check(true)
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(island.ID()))
	clone.effects.Apply(clone)
	if clone.FindPermanent(attacker.ID()).HasAttr(AttrCanAttack) {
		t.Fatal("clone kept permission after defender lost Island")
	}
	check(true)
	g.RemoveFromBattlefield(g.FindPermanent(island.ID()))
	check(false)
	g.PutOnBattlefield(land, b.PlayerID())
	check(true)
}

func TestDefendingPlayerControlAttackRestrictionDeclaration(t *testing.T) {
	rule := CanAttackOnlyIfDefendingPlayerControls(HasSubType("Island"))
	card := NewCreature("Attacker", "", 2, 2, WithAbility(Static(rule)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if meta := InspectDefinition(rule); meta.Role == "" || meta.Text == "" {
		t.Fatalf("inspection = %+v", meta)
	}
}
