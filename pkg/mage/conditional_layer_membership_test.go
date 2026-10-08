package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestConditionalLayerMembership(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := PermanentMatchesFilter(This, Not(IsCreature))
	card := NewArtifact("Subject", "{3}", WithAbility(Static(WhileWithSharedSubjects(condition, AddCardType(This, core.TypeCreature), SetBasePowerToughness(This, ManaValueOf(This), ManaValueOf(This))))))
	if err := ValidateCardDefinition("Subject", card); err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	check := func(game *Game, want int) {
		t.Helper()
		live := game.FindPermanent(p.ID())
		if !live.HasType(core.TypeCreature) || live.CurrentPower(game) != want || live.CurrentToughness(game) != want {
			t.Fatalf("later layer lost membership: %d/%d", live.CurrentPower(game), live.CurrentToughness(game))
		}
	}
	check(g, 3)
	g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.EndOfTurn))
	check(g, 3)
	clone := g.Clone()
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	check(clone, 3)
	check(g, 3)
	g.ApplyContinuousEffects()
	check(g, 3)
	phased := g.PhaseOut(p.ID())
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	check(g, 3)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	check(g, 3)
}

func TestConditionalLayerMembershipSet(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	set := MatchingPermanents(And(IsArtifact, Not(IsCreature)))
	condition := StateConditionFunc("active", []StateDependency{StateBattlefield}, func(StateReader, StateContext) bool { return true })
	g.PutOnBattlefield(NewEnchantment("Effect", "", WithAbility(Static(WhileWithSharedSubjects(condition, AddEachCardType(set, core.TypeCreature), SetEachBasePowerToughness(set, Fixed(4), Fixed(5)))))), a.PlayerID())
	p := g.PutOnBattlefield(NewArtifact("Artifact", ""), b.PlayerID())
	if p.CurrentPower(g) != 4 || p.CurrentToughness(g) != 5 {
		t.Fatal("set membership was read again after the type change")
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	clone.ApplyContinuousEffects()
	if clone.FindPermanent(p.ID()).CurrentPower(clone) != 4 {
		t.Fatal("clone lost its own membership")
	}
}

func TestConditionalLayerMembershipDependency(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("Land", WithSubTypes("Mountain")), a.PlayerID())
	set := MatchingPermanents(HasSubType("Forest"))
	condition := StateConditionFunc("active", []StateDependency{StateBattlefield}, func(StateReader, StateContext) bool { return true })
	source := g.PutOnBattlefield(NewEnchantment("Effect", "", WithAbility(Static(WhileWithSharedSubjects(condition, AddEachCardType(set, core.TypeCreature), SetEachBasePowerToughness(set, Fixed(2), Fixed(3)))))), a.PlayerID())
	change := BecomesBasicLandTargetEffect(land.ID(), core.EndOfTurn, "Forest")
	change.SetSourceID(source.ID())
	g.AddContinuousEffect(change)
	p := g.FindPermanent(land.ID())
	if !p.HasType(core.TypeCreature) || p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 3 {
		t.Fatal("type dependency did not run before selection")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if g.FindPermanent(land.ID()).HasType(core.TypeCreature) {
		t.Fatal("later pass kept former membership")
	}
}

func TestConditionalLayerMembershipFreshConditionAndValue(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := PermanentIsUntapped(This)
	value := CountPermanents(MatchingPermanents(IsCreature))
	p := g.PutOnBattlefield(NewArtifact("Subject", "", WithAbility(Static(WhileWithSharedSubjects(condition, AddCardType(This, core.TypeCreature), SetBasePowerToughness(This, value, value))))), a.PlayerID())
	if p.CurrentPower(g) != 1 {
		t.Fatal("value was read before the type layer")
	}
	g.PutOnBattlefield(NewCreature("Other", "", 1, 1), b.PlayerID())
	if g.FindPermanent(p.ID()).CurrentPower(g) != 2 {
		t.Fatal("value did not update on a new pass")
	}
	g.MutablePermanent(p.ID()).Tapped = true
	g.ApplyContinuousEffects()
	if g.FindPermanent(p.ID()).HasType(core.TypeCreature) || g.FindPermanent(p.ID()).BasePTOverride != nil {
		t.Fatal("false condition kept an earlier pass")
	}
	g.MutablePermanent(p.ID()).Tapped = false
	g.ApplyContinuousEffects()
	if g.FindPermanent(p.ID()).CurrentPower(g) != 2 {
		t.Fatal("condition did not restart the effect")
	}
}
