package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestFilteredBlockingCapturedRestriction(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Attacker", "", 2, 2)
	card.SetOwner(a.PlayerID())
	attacker := g.PutOnBattlefield(card, a.PlayerID())
	wall := g.PutOnBattlefield(NewCreature("Wall", "", 0, 4, WithSubTypes("Wall")), b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 2, 2), b.PlayerID())
	if err := ApplyEffect(g, Until(EndOfTurn, CantBeBlockedExceptBy(This, HasSubType("Wall"))), attacker.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !CanBlock(wall, attacker, g) || CanBlock(other, attacker, g) {
		t.Fatal("filter restriction")
	}
	if err := ApplyEffect(g, Until(EndOfTurn, RemoveAbilities(This)), attacker.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if CanBlock(other, attacker, g) {
		t.Fatal("ability removal erased the rule")
	}
	g.PhaseIn(g.PhaseOut(attacker.ID()))
	if CanBlock(g.FindPermanent(other.ID()), g.FindPermanent(attacker.ID()), g) {
		t.Fatal("phase cycle erased the rule")
	}
	clone := g.Clone()
	if CanBlock(clone.FindPermanent(other.ID()), clone.FindPermanent(attacker.ID()), clone) {
		t.Fatal("clone lost the active filter")
	}
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if !CanBlock(clone.FindPermanent(other.ID()), clone.FindPermanent(attacker.ID()), clone) || CanBlock(g.FindPermanent(other.ID()), g.FindPermanent(attacker.ID()), g) {
		t.Fatal("cleanup changed another game")
	}
	g.RemoveFromBattlefield(g.FindPermanent(attacker.ID()))
	attacker = g.PutOnBattlefield(card, a.PlayerID())
	if !CanBlock(g.FindPermanent(other.ID()), attacker, g) {
		t.Fatal("returned card received old rule")
	}
}

func TestFilteredBlockingDefinitionValidation(t *testing.T) {
	for _, subject := range []PermanentRef{nil, From(BindPermanent("unwritten"))} {
		card := NewInstant("Invalid", "", WithAbility(Spell(Until(EndOfTurn, CantBeBlockedExceptBy(subject, AnyPermanent)))))
		if err := ValidateCardDefinition("Invalid", card); err == nil {
			t.Fatal("invalid subject accepted")
		}
	}
	target := DeclarePermanentTarget("subject", IsCreature)
	card := NewInstant("Valid", "", WithAbility(Spell(Until(EndOfTurn, CantBeBlockedExceptBy(target, HasSubType("Wall"))), WithTargets(target))))
	if err := ValidateCardDefinition("Valid", card); err != nil {
		t.Fatal(err)
	}
}
