package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestStatePermanentSetCapture(t *testing.T) {
	for _, scenario := range []string{"late", "returned", "phased", "cycle", "control", "source-left", "copy", "clone", "counter"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewLand("First")
			card.SetOwner(b.PlayerID())
			first := g.PutOnBattlefield(card, b.PlayerID())
			result := BindPermanentSet("detected")
			watcher := NewArtifact("Watcher", "", WithAbility(WhenPermanentsPresent(MatchingPermanents(IsLand), result, ControllersSacrificeEach(FromPermanentSet(result)))))
			if err := ValidateCardDefinition("Watcher", watcher); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(watcher, a.PlayerID())
			g.CheckStateTriggers()
			g.CheckStateTriggers()
			late := g.PutOnBattlefield(NewLand("Late"), a.PlayerID())
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("pending trigger repeated", g.StackSize())
			}
			switch scenario {
			case "returned":
				g.BouncePermanentToHand(first)
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "phased":
				g.PhaseOut(first.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(first.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: first.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "source-left":
				g.BouncePermanentToHand(source)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "counter":
				g.CounterStackObject(g.StackPeek().ID)
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
			}
			clone := g.Clone()
			if scenario == "clone" {
				clone.PhaseOut(first.ID())
			}
			g.ResolveTopOfStack()
			remains := scenario == "returned" || scenario == "phased"
			if (g.FindPermanentIncludingPhased(first.ID()) != nil) != remains {
				t.Fatal("saved visit was lost")
			}
			if (g.FindPermanent(late.ID()) != nil) != (scenario != "counter") {
				t.Fatal("capture did not use detection members")
			}
			if clone.FindPermanent(late.ID()) == nil {
				t.Fatal("original changed clone")
			}
			clone.ResolveTopOfStack()
			if (clone.FindPermanentIncludingPhased(first.ID()) != nil) != (remains || scenario == "clone") {
				t.Fatal("clone lost capture")
			}
		})
	}
}

func TestStatePermanentSetInvalid(t *testing.T) {
	result := BindPermanentSet("result")
	for _, card := range []Card{
		NewArtifact("Nil set", "", WithAbility(WhenPermanentsPresent(nil, result, ControllersSacrificeEach(FromPermanentSet(result))))),
		NewArtifact("Nil binding", "", WithAbility(WhenPermanentsPresent(MatchingPermanents(IsLand), nil, ControllersSacrificeEach(FromPermanentSet(result))))),
		NewInstant("Wrong zone", "", WithAbility(WhenPermanentsPresent(MatchingPermanents(IsLand), result, ControllersSacrificeEach(FromPermanentSet(result))))),
	} {
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted", card.Name())
		}
	}
}

func TestStatePermanentSetLifecycleAndReplacement(t *testing.T) {
	for _, scenario := range []string{"empty", "source-phased", "source-removed", "source-return", "member-type", "exile", "cancel", "batch"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			result := BindPermanentSet("members")
			factory := func() Card {
				return NewArtifact("Watcher", "", WithAbility(WhenPermanentsPresent(MatchingPermanents(IsLand), result, ControllersSacrificeEach(FromPermanentSet(result)))))
			}
			sourceCard := factory()
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("empty set triggered")
			}
			if scenario == "source-phased" {
				g.PhaseOut(source.ID())
			}
			if scenario == "source-removed" {
				g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
			}
			first := g.PutOnBattlefield(NewLand("First"), b.PlayerID())
			if scenario == "source-return" {
				g.BouncePermanentToHand(source)
				a.RemoveFromHand(sourceCard.ID())
				g.PutOnBattlefield(sourceCard, a.PlayerID())
			}
			g.CheckStateTriggers()
			g.PutTriggersOnStack()
			blocked := scenario == "source-phased" || scenario == "source-removed"
			if blocked {
				if g.StackSize() != 0 {
					t.Fatal("inactive source triggered")
				}
				return
			}
			want := 1
			if scenario == "source-return" {
				want = 2
			}
			if g.StackSize() != want {
				t.Fatal("source lifetimes shared suppression", g.StackSize())
			}
			second := g.PutOnBattlefield(NewLand("Second"), a.PlayerID())
			if scenario == "member-type" {
				g.MutablePermanent(first.ID()).RevokeBaseAttr(core.AttrIsLand)
				g.MutablePermanent(first.ID()).GrantBaseAttr(core.AttrIsArtifact)
			}
			if scenario == "exile" {
				g.AddExileIfWouldGoToGraveyardThisTurn(first.ID(), source.ID())
			}
			if scenario == "cancel" {
				g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(action Action) bool { _, ok := action.(*SacrificePermanentAction); return ok }, replaceFunc: func(Action, *Game) Action { return nil }})
			}
			if scenario == "batch" {
				observer := NewEnchantment("No lands", "", WithAbility(WhenState(BattlefieldHasNone(IsLand), Sacrifice(This))))
				watch := g.PutOnBattlefield(observer, a.PlayerID())
				g.BouncePermanentToHand(second)
				g.AddRegenerationShield(first.ID())
				g.MutablePermanent(first.ID()).GrantBaseAttr(core.Indestructible)
				g.ResolveTopOfStack()
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("batch did not detect final state")
				}
				g.ResolveTopOfStack()
				if g.FindPermanent(watch.ID()) != nil {
					t.Fatal("absence trigger failed")
				}
				return
			}
			g.ResolveTopOfStack()
			if (g.FindPermanent(first.ID()) != nil) != (scenario == "cancel") || g.FindPermanent(second.ID()) == nil {
				t.Fatal("saved set changed or replacement ignored")
			}
			if scenario == "exile" && g.FindExiledCard(first.ID()) == nil {
				t.Fatal("exile replacement ignored")
			}
			if scenario == "cancel" {
				g.CheckStateTriggers()
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("canceled sacrifice did not permit a new trigger")
				}
			}
		})
	}
}
