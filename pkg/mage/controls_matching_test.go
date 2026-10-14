package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestControlsMatchingController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	subject := g.PutOnBattlefield(NewCreature("Subject", "", 2, 2), b.PlayerID())
	filter := ControllerControls(HasSubType("Island"))
	check := func(game *Game, want bool) {
		t.Helper()
		if got := filter.Match(game.FindPermanent(subject.ID()), game); got != want {
			t.Fatalf("controller match = %v, want %v", got, want)
		}
	}
	check(g, false)
	island := g.PutOnBattlefield(NewLand("Island", WithSubTypes("Island")), b.PlayerID())
	check(g, true)
	clone := g.Clone()
	clone.PhaseOut(island.ID())
	check(clone, false)
	check(g, true)
	g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: subject.ID(), ControllerID: a.PlayerID(), Duration: core.EndOfTurn})
	g.ApplyContinuousEffects()
	check(g, false)
	own := g.PutOnBattlefield(NewLand("Own island", WithSubTypes("Island")), a.PlayerID())
	check(g, true)
	g.PhaseOut(own.ID())
	check(g, false)
	g.PhaseIn([]uuid.UUID{own.ID()})
	check(g, true)
	g.RemoveFromBattlefield(g.FindPermanent(subject.ID()))
	if filter.Match(subject, g) {
		t.Fatal("departed subject matched")
	}
	returned := g.PutOnBattlefield(subject.Card, b.PlayerID())
	if filter.Match(subject, g) || !filter.Match(returned, g) {
		t.Fatal("returned subject replaced the saved visit")
	}
	if filter.Match(nil, g) || filter.Match(subject, nil) {
		t.Fatal("absent query matched")
	}
}

func TestControlsMatchingDefender(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetActivePlayerIndex(0)
	g.SetStep(core.DeclareAttackers)
	condition := PlayerControls(CombatDefendingPlayer(), And(IsCreature, IsUntapped, HasPowerGTE(3)))
	card := NewCreature("Attacker", "", 6, 6, WithAbility(Static(CantAttackIf(This, condition))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	attacker := g.PutOnBattlefield(card, a.PlayerID())
	attacker.RevokeBaseAttr(core.AttrSummonSick)
	defender := g.PutOnBattlefield(NewCreature("Defender", "", 2, 4), b.PlayerID())
	check := func(game *Game, want bool) {
		t.Helper()
		game.ApplyContinuousEffects()
		if got := game.FindPermanent(attacker.ID()).CanDeclareAsAttacker(game); got != want {
			t.Fatalf("attack = %v, want %v", got, want)
		}
	}
	check(g, true)
	if err := ApplyEffect(g, Until(core.EndOfTurn, SetBasePowerToughness(This, Fixed(3), Fixed(4))), defender.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	check(g, false)
	g.TapPermanent(g.FindPermanent(defender.ID()))
	check(g, true)
	g.UntapPermanent(g.FindPermanent(defender.ID()))
	check(g, false)
	clone := g.Clone()
	clone.PhaseOut(defender.ID())
	check(clone, true)
	check(g, false)
	remover := g.PutOnBattlefield(NewArtifact("Remover", ""), a.PlayerID())
	removal := RemoveAllAbilities(attacker.ID(), core.WhileOnBattlefield)
	removal.SetSourceID(remover.ID())
	g.AddContinuousEffect(removal)
	check(g, true)
	g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
	check(g, false)
	g.SetStep(core.PrecombatMain)
	if condition.checkState(g, StateContext{ControllerID: a.PlayerID()}) {
		t.Fatal("defender exists outside combat")
	}
}
