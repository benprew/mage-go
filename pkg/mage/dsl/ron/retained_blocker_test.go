package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONRetainedBlockerEligibility(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			ca, cb := &ronPartitionPlayer{Player: a}, &ronPartitionPlayer{Player: b}
			g.SetPlayerAt(0, ca)
			g.SetPlayerAt(1, cb)
			card := repositoryCard(t, set, "Raging River")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Red, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 8), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 8), a.PlayerID())
			first.RevokeBaseAttr(core.AttrSummonSick)
			second.RevokeBaseAttr(core.AttrSummonSick)
			right := g.PutOnBattlefield(mage.NewCreature("Right", "", 1, 8), b.PlayerID())
			left := g.PutOnBattlefield(mage.NewCreature("Left", "", 1, 8), b.PlayerID())
			flying := g.PutOnBattlefield(mage.NewCreature("Flying", "", 1, 8, mage.WithKeyword(core.Flying)), b.PlayerID())
			dormant := g.PutOnBattlefield(mage.NewArtifact("Dormant", ""), b.PlayerID())
			phased := g.PutOnBattlefield(mage.NewCreature("Phased", "", 1, 8, mage.WithKeyword(core.Flying)), b.PlayerID())
			g.PhaseOut(phased.ID())
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID(), second.ID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || ca.calls+cb.calls != 0 {
				t.Fatal("wrong trigger count or early choice")
			}
			g.BouncePermanentToHand(source)
			clone := g.Clone()
			clone.SetPlayerAt(0, &ronPartitionPlayer{Player: clone.GetPlayer(a.PlayerID())})
			clone.SetPlayerAt(1, &ronPartitionPlayer{Player: clone.GetPlayer(b.PlayerID())})
			g.ResolveStack()
			if ca.calls != 2 || cb.calls != 2 {
				t.Fatal("wrong choice player or count", ca.calls, cb.calls)
			}
			check := func(game *mage.Game, blocker, attacker uuid.UUID, want bool) {
				t.Helper()
				if got := mage.CanBlock(game.FindPermanent(blocker), game.FindPermanent(attacker), game); got != want {
					t.Fatalf("%s blocks %s: got %v want %v", game.FindPermanent(blocker).Name(), game.FindPermanent(attacker).Name(), got, want)
				}
			}
			check(g, right.ID(), first.ID(), true)
			check(g, left.ID(), first.ID(), false)
			check(g, right.ID(), second.ID(), false)
			check(g, left.ID(), second.ID(), true)
			check(g, flying.ID(), first.ID(), true)
			check(g, flying.ID(), second.ID(), true)
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: left.ID(), AttackerID: first.ID()}})
			if mage.IsBlocking.Match(g.FindPermanent(left.ID()), g) {
				t.Fatal("illegal pile block was committed")
			}
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: right.ID(), AttackerID: first.ID()}, {BlockerID: left.ID(), AttackerID: second.ID()}})
			if !mage.IsBlocking.Match(g.FindPermanent(right.ID()), g) || !mage.IsBlocking.Match(g.FindPermanent(left.ID()), g) {
				t.Fatal("legal pile blocks were not committed")
			}

			late := g.PutOnBattlefield(mage.NewCreature("Late flyer", "", 1, 8, mage.WithKeyword(core.Flying)), b.PlayerID())
			check(g, late.ID(), first.ID(), false)
			g.PhaseIn([]uuid.UUID{phased.ID()})
			check(g, phased.ID(), first.ID(), false)
			g.MutablePermanent(dormant.ID()).GrantBaseAttr(core.AttrIsCreature)
			g.MutablePermanent(dormant.ID()).GrantBaseAttr(core.AttrCanBlock)
			g.ApplyContinuousEffects()
			check(g, dormant.ID(), first.ID(), false)
			g.MutablePermanent(flying.ID()).RevokeBaseAttr(core.Flying)
			g.MutablePermanent(left.ID()).GrantBaseAttr(core.Flying)
			g.ApplyContinuousEffects()
			check(g, flying.ID(), first.ID(), false)
			check(g, left.ID(), first.ID(), true)
			entered := g.PutOnBattlefieldAttacking(mage.NewCreature("Entered attacking", "", 1, 8), a.PlayerID(), b.PlayerID())
			check(g, late.ID(), entered.ID(), true)
			g.PhaseIn(g.PhaseOut(right.ID()))
			check(g, right.ID(), first.ID(), true)
			g.RemoveFromBattlefield(right)
			returned := g.PutOnBattlefield(right.Card, b.PlayerID())
			check(g, returned.ID(), first.ID(), false)
			clone.ResolveStack()
			check(clone, right.ID(), first.ID(), true)
			check(clone, left.ID(), first.ID(), false)
			calls := 0
			g.SetOnPriority(func(game *mage.Game, _ int, _ bool) mage.PriorityAction {
				calls++
				check(game, returned.ID(), first.ID(), false)
				return mage.PriorityAction{Type: mage.PriorityPass}
			})
			g.RunStepWithPriority(core.EndCombat)
			if calls < 2 {
				t.Fatal("end-combat priority not observed")
			}

			check(g, returned.ID(), first.ID(), true)
			check(clone, left.ID(), first.ID(), false)
		})
	}
}

func TestRONRetainedBlockerEligibilityInvalid(t *testing.T) {
	for _, op := range []string{
		`CantBeBlockedExceptByDefenders`, `CantBeBlockedExceptByDefenders()`,
		`CantBeBlockedExceptByDefenders(IsCardtype(Creature),AnyPermanent)`,
		`CantBeBlockedExceptByDefenders(InThePileChosenForPermanent(ThisPermanent))`,
		`CantBeBlockedExceptByDefenders(Or([]))`,
	} {
		source := pileSelectionSource(ronPartitionAction + `,ActionForEachPermanent(IsAttacking,[` + ronPileChoice + `,CreatePermanentRuleEffectUntil(EachablePermanent,[` + op + `],UntilEndOfCombat)])`)
		nodes, err := Parse(strings.NewReader(source), "invalid-blockers.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", op)
		}
	}
}

func TestRONRetainedBlockerEligibilityScope(t *testing.T) {
	rule := `CreatePermanentRuleEffectUntil(EachablePermanent,[CantBeBlockedExceptByDefenders(InThePileChosenForPermanent(EachablePermanent))],UntilEndOfCombat)`
	for _, actions := range []string{
		ronPartitionAction + `,ActionForEachPermanent(IsAttacking,[` + rule + `])`,
		ronPartitionAction + `,ActionForEachPermanent(IsAttacking,[` + ronPileChoice + `]),` + rule,
		ronPartitionAction + `,ActionForEachPermanent(IsAttacking,[` + ronPileChoice + `,ActionForEachPermanent(IsCardtype(Creature),[` + rule + `])])`,
		ronPartitionAction + `,ActionForEachPermanent(IsAttacking,[If(PermanentPassesFilter(EachablePermanent,IsAttacking),[` + ronPileChoice + `]),` + rule + `])`,
	} {
		assertPileSelectionRejected(t, actions)
	}
	for _, invalid := range []string{
		strings.Replace(rule, "UntilEndOfCombat", "UntilEndOfTurn", 1),
		strings.Replace(rule, "UntilEndOfCombat", "UntilEndOfCombat()", 1),
		strings.Replace(rule, "EachablePermanent", "ThisPermanent", 1),
		strings.Replace(rule, "EachablePermanent", "EachablePermanent()", 1),
		strings.Replace(rule, "InThePileChosenForPermanent(EachablePermanent)", "InThePileChosenForPermanent(EachablePermanent())", 1),
	} {
		assertPileSelectionRejected(t, ronPartitionAction+`,ActionForEachPermanent(IsAttacking,[`+ronPileChoice+`,`+invalid+`])`)
	}
}

func TestRONRetainedBlockerEligibilityInstances(t *testing.T) {
	factory, err := Compile(hostUntapRecord(t, "LEA", "Raging River"))
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	ca, cb := &ronPartitionPlayer{Player: a}, &ronPartitionPlayer{Player: b}
	g.SetPlayerAt(0, ca)
	g.SetPlayerAt(1, cb)
	g.PutOnBattlefield(factory(), a.PlayerID())
	g.PutOnBattlefield(factory(), a.PlayerID())
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 8), a.PlayerID())
	attacker.RevokeBaseAttr(core.AttrSummonSick)
	right := g.PutOnBattlefield(mage.NewCreature("Right", "", 1, 8), b.PlayerID())
	left := g.PutOnBattlefield(mage.NewCreature("Left", "", 1, 8), b.PlayerID())
	flying := g.PutOnBattlefield(mage.NewCreature("Flying", "", 1, 8, mage.WithKeyword(core.Flying)), b.PlayerID())
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("expected two independent triggers")
	}
	g.ResolveStack()
	if ca.calls != 2 || cb.calls != 4 {
		t.Fatal("instances shared resolution choices")
	}
	if mage.CanBlock(right, attacker, g) || mage.CanBlock(left, attacker, g) || !mage.CanBlock(flying, attacker, g) {
		t.Fatal("independent restrictions did not intersect")
	}
	if err = mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), attacker.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if mage.CanBlock(right, attacker, g) || mage.CanBlock(left, attacker, g) {
		t.Fatal("ability removal erased rules")
	}
}

func TestRONRetainedBlockerEligibilityDelayedFrames(t *testing.T) {
	record := hostUntapRecord(t, "LEA", "Raging River")
	loop := record.Fields["Rules"].Args[0].Args[1].Args[0].Args[1]
	rule := loop.Args[1].Args[1]
	nodes, err := Parse(strings.NewReader(`CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainLife(Integer(1))]))`), "delayed-blockers.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes[0].Args[1].Args[0].Args[0] = rule
	loop.Args[1].Args[1] = nodes[0]
	record.Fields["Rules"].Args[0].Args[1].Args[0].Args[1] = loop
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	g.SetPlayerAt(0, &ronPartitionPlayer{Player: a})
	g.SetPlayerAt(1, &ronPartitionPlayer{Player: b})
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 8), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 8), a.PlayerID())
	first.RevokeBaseAttr(core.AttrSummonSick)
	second.RevokeBaseAttr(core.AttrSummonSick)
	right := g.PutOnBattlefield(mage.NewCreature("Right", "", 1, 8), b.PlayerID())
	left := g.PutOnBattlefield(mage.NewCreature("Left", "", 1, 8), b.PlayerID())
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID(), second.ID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if !mage.CanBlock(left, first, g) {
		t.Fatal("restriction installed before delay")
	}
	g.BouncePermanentToHand(source)
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if !mage.CanBlock(game.FindPermanent(right.ID()), game.FindPermanent(first.ID()), game) || mage.CanBlock(game.FindPermanent(left.ID()), game.FindPermanent(first.ID()), game) || !mage.CanBlock(game.FindPermanent(left.ID()), game.FindPermanent(second.ID()), game) || mage.CanBlock(game.FindPermanent(right.ID()), game.FindPermanent(second.ID()), game) {
			t.Fatal("delayed loop frames shared selected piles")
		}
	}
}
