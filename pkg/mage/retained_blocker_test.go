package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRetainedBlockerEligibility(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 1, 8), a.PlayerID())
	ground := g.PutOnBattlefield(NewCreature("Ground", "", 1, 8), b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 8), b.PlayerID())
	flying := g.PutOnBattlefield(NewCreature("Flying", "", 1, 8, WithKeyword(core.Flying)), b.PlayerID())
	phased := g.PutOnBattlefield(NewCreature("Phased", "", 1, 8, WithKeyword(core.Flying)), b.PlayerID())
	g.PhaseOut(phased.ID())
	binding := BindPermanentSet("selected")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: attacker.ID()}
	producer := SeparatePermanentsIntoNamedPiles(You, MatchingPermanents(Named("Ground")), NamedPermanentPile{"Selected", binding}, NamedPermanentPile{"Empty", BindPermanentSet("empty")})
	subject := DeclarePermanentTarget("attacker", IsCreature)
	effect := RetainBlockerEligibility(core.EndOfCombat, subject, MatchingPermanents(IsCreature), AnyRetainedBlockers(CurrentBlockers(HasKeywordFilter(core.Flying)), SavedBlockers(FromPermanentSet(binding))))
	g.resolution.targetSpecs = []Target{subject.targetSpec()}
	if err := ApplyEffect(g, Sequence(producer, effect), attacker.ID(), a.PlayerID(), []uuid.UUID{attacker.ID()}); err != nil {
		t.Fatal(err)
	}
	check := func(game *Game, blockerID, attackerID uuid.UUID, want bool) {
		t.Helper()
		if got := CanBlock(game.FindPermanent(blockerID), game.FindPermanent(attackerID), game); got != want {
			t.Fatalf("CanBlock %s: got %v want %v", game.FindPermanent(blockerID).Name(), got, want)
		}
	}
	check(g, ground.ID(), attacker.ID(), true)
	check(g, other.ID(), attacker.ID(), false)
	check(g, flying.ID(), attacker.ID(), true)
	clone := g.Clone()
	if err := SeparatePermanentsIntoNamedPiles(You, MatchingPermanents(IsLand), NamedPermanentPile{"Selected", binding}, NamedPermanentPile{"Empty", BindPermanentSet("empty")}).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.PhaseIn([]uuid.UUID{phased.ID()})
	check(g, phased.ID(), attacker.ID(), false)
	late := g.PutOnBattlefield(NewCreature("Late", "", 1, 8, WithKeyword(core.Flying)), b.PlayerID())
	check(g, late.ID(), attacker.ID(), false)
	g.MutablePermanent(other.ID()).GrantBaseAttr(core.Flying)
	g.ApplyContinuousEffects()
	check(g, other.ID(), attacker.ID(), true)
	check(clone, other.ID(), attacker.ID(), false)
	g.MutablePermanent(flying.ID()).RevokeBaseAttr(core.Flying)
	g.ApplyContinuousEffects()
	check(g, flying.ID(), attacker.ID(), false)
	check(clone, flying.ID(), attacker.ID(), true)
	g.PhaseIn(g.PhaseOut(ground.ID()))
	check(g, ground.ID(), attacker.ID(), true)
	g.RemoveFromBattlefield(ground)
	returned := g.PutOnBattlefield(ground.Card, b.PlayerID())
	check(g, returned.ID(), attacker.ID(), false)
	g.PhaseIn(g.PhaseOut(attacker.ID()))
	check(g, returned.ID(), attacker.ID(), false)
	g.GetEffects().RemoveEndOfCombat()
	g.ApplyContinuousEffects()
	check(g, returned.ID(), attacker.ID(), true)
	check(clone, other.ID(), attacker.ID(), false)
}

func TestRetainedBlockerEligibilityDefinition(t *testing.T) {
	binding := BindPermanentSet("missing")
	target := DeclarePermanentTarget("target", IsCreature)
	for _, effect := range []ResolvingEffect{
		RetainBlockerEligibility(core.EndOfCombat, nil, MatchingPermanents(IsCreature), CurrentBlockers(AnyPermanent)),
		RetainBlockerEligibility(core.EndOfCombat, This, nil, CurrentBlockers(AnyPermanent)),
		RetainBlockerEligibility(core.WhileOnBattlefield, This, MatchingPermanents(IsCreature), CurrentBlockers(AnyPermanent)),
		RetainBlockerEligibility(core.EndOfCombat, This, MatchingPermanents(IsCreature), RetainedBlockerFilter{}),
		RetainBlockerEligibility(core.EndOfCombat, This, MatchingPermanents(IsCreature), AnyRetainedBlockers()),
		RetainBlockerEligibility(core.EndOfCombat, This, MatchingPermanents(IsCreature), AllRetainedBlockers()),
		RetainBlockerEligibility(core.EndOfCombat, This, MatchingPermanents(IsCreature), SavedBlockers(nil)),
		RetainBlockerEligibility(core.EndOfCombat, This, MatchingPermanents(IsCreature), SavedBlockers(FromPermanentSet(binding))),
		RetainBlockerEligibility(core.EndOfCombat, target, MatchingPermanents(IsCreature), CurrentBlockers(AnyPermanent)),
	} {
		c := NewEnchantment("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("accepted invalid retained blocker definition", effect)
		}
	}
	valid := NewEnchantment("Valid", "", WithAbility(Activate(Costs(), Sequence(SeparatePermanentsIntoNamedPiles(You, MatchingPermanents(IsCreature), NamedPermanentPile{"Selected", binding}, NamedPermanentPile{"Empty", BindPermanentSet("empty")}), RetainBlockerEligibility(core.EndOfTurn, This, MatchingPermanents(IsCreature), AllRetainedBlockers(CurrentBlockers(AnyPermanent), SavedBlockers(FromPermanentSet(binding))))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
}

func TestRetainedBlockerEligibilityLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(1)
	source := g.PutOnBattlefield(NewEnchantment("Source", ""), a.PlayerID())
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 1, 8), a.PlayerID())
	blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
	target := DeclarePermanentTarget("subject", IsCreature)
	g.resolution.targetSpecs = []Target{target.targetSpec()}
	effect := RetainBlockerEligibility(core.EndOfCombat, target, MatchingPermanents(IsCreature), CurrentBlockers(Not(AnyPermanent)))
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{attacker.ID()}); err != nil {
		t.Fatal(err)
	}
	check := func(game *Game, want bool) {
		t.Helper()
		if CanBlock(game.FindPermanent(blocker.ID()), game.FindPermanent(attacker.ID()), game) != want {
			t.Fatal("wrong restriction lifetime")
		}
	}
	g.PhaseOut(source.ID())
	check(g, false)
	g.PhaseIn([]uuid.UUID{source.ID()})
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	check(g, false)
	g.PhaseIn(g.PhaseOut(attacker.ID()))
	check(g, false)
	clone := g.Clone()
	calls := 0
	g.SetOnPriority(func(game *Game, _ int, _ bool) PriorityAction {
		calls++
		check(game, false)
		return PriorityAction{Type: PriorityPass}
	})
	g.RunStepWithPriority(core.EndCombat)
	if calls < 2 {
		t.Fatal("end-combat priority not observed")
	}
	check(g, true)
	check(clone, false)
	g.MutablePermanent(attacker.ID()).RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
	check(g, true)
	clone.RemoveFromBattlefield(clone.FindPermanent(attacker.ID()))
	clone.PutOnBattlefield(attacker.Card, a.PlayerID())
	check(clone, true)
}

func TestRetainedBlockerEligibilityUnavailableSubject(t *testing.T) {
	for _, missing := range []string{"phased", "returned", "unbound"} {
		t.Run(missing, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			subject := g.PutOnBattlefield(NewCreature("Subject", "", 1, 8), a.PlayerID())
			blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
			binding := BindPermanent("subject")
			g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{binding: {subject.ID(), subject.incarnationID}}
			switch missing {
			case "phased":
				g.PhaseOut(subject.ID())
			case "returned":
				g.RemoveFromBattlefield(subject)
				subject = g.PutOnBattlefield(subject.Card, a.PlayerID())
			case "unbound":
				delete(g.resolution.permanentBindings, binding)
			}
			effect := RetainBlockerEligibility(core.EndOfCombat, From(binding), MatchingPermanents(IsCreature), CurrentBlockers(Not(AnyPermanent)))
			if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			g.PhaseIn([]uuid.UUID{subject.ID()})
			if !CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(subject.ID()), g) {
				t.Fatal("unavailable subject acquired a rule")
			}
		})
	}
}
