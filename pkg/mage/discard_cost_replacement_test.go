package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type allDiscardToLibraryReplacement struct{ replacementBase }

func (*allDiscardToLibraryReplacement) Matches(action Action, _ GameReader) bool {
	_, ok := action.(*DiscardAction)
	return ok
}
func (*allDiscardToLibraryReplacement) Replace(action Action, _ *Game) Action {
	return action.(*DiscardAction).withDestination(core.ZoneLibrary)
}
func (*allDiscardToLibraryReplacement) IsActive(GameReader) bool { return true }
func (*allDiscardToLibraryReplacement) Clone() ReplacementEffect {
	return &allDiscardToLibraryReplacement{}
}

type discardToHandReplacement struct{ replacementBase }

func (*discardToHandReplacement) Matches(action Action, _ GameReader) bool {
	discard, ok := action.(*DiscardAction)
	return ok && discard.Destination() == core.ZoneLibrary
}
func (*discardToHandReplacement) Replace(_ Action, _ *Game) Action { return nil }
func (*discardToHandReplacement) IsActive(GameReader) bool         { return true }
func (*discardToHandReplacement) Clone() ReplacementEffect {
	return &discardToHandReplacement{}
}

type preventCostDiscardReplacement struct{ replacementBase }

func (*preventCostDiscardReplacement) Matches(action Action, _ GameReader) bool {
	discard, ok := action.(*DiscardAction)
	return ok && !discard.CausedByEffect()
}
func (*preventCostDiscardReplacement) Replace(_ Action, _ *Game) Action { return nil }
func (*preventCostDiscardReplacement) IsActive(GameReader) bool         { return true }
func (*preventCostDiscardReplacement) Clone() ReplacementEffect {
	return &preventCostDiscardReplacement{}
}

func TestDiscardPaymentUsesSharedReplacementAction(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewLand("Paid")
	a.AddToHand(card)
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{DiscardCost(1)}})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 0 || len(a.Graveyard()) != 0 || len(a.Library()) != 1 || a.Library()[0].ID() != card.ID() {
		t.Fatal("cost discard did not use its replacement destination")
	}
	if got := g.PlayerDiscardCountThisTurn(a.PlayerID()); got != 1 {
		t.Fatalf("discard events = %d; want 1", got)
	}
}

func TestDiscardPaymentEffectOnlyReplacementDoesNotApply(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	g.PutOnBattlefield(NewArtifact("Library", "", WithStaticAbility(DiscardToLibraryReplacement())), a.PlayerID())
	card := NewLand("Paid")
	a.AddToHand(card)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{DiscardCost(1)}})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != card.ID() || len(a.Library()) != 0 {
		t.Fatal("effect-only replacement changed a cost discard")
	}
}

func TestDiscardPaymentRechecksReplacementResult(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	a.AddToHand(NewLand("Paid"))
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	g.effects.AddReplacement(&discardToHandReplacement{})
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{DiscardCost(1)}}); err == nil {
		t.Fatal("fully replaced discard paid a cost")
	}
	if len(a.Hand()) != 1 || len(a.Library()) != 0 || len(a.Graveyard()) != 0 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 {
		t.Fatal("failed validation changed the live game")
	}
}

func TestDiscardPaymentPreventionRollsBackOtherCosts(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	a.AddToHand(NewLand("Paid"))
	a.ManaPool().Add(core.Red, 1)
	g.effects.AddReplacement(&preventCostDiscardReplacement{})
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller: a.PlayerID(), Costs: []Cost{ManaPayment("{R}"), DiscardCost(1)},
	}); err == nil {
		t.Fatal("prevented cost discard paid the total cost")
	}
	if a.ManaPool().Count(core.Red) != 1 || len(a.Hand()) != 1 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 {
		t.Fatal("failed total payment changed mana, hand, or discard history")
	}
}

func TestDiscardPaymentReplacementCloneIsolation(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewLand("Paid")
	a.AddToHand(card)
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	clone := g.Clone()
	clonePlayer := clone.GetPlayer(a.PlayerID())
	tx, err := clone.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{DiscardCost(1)}})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(clonePlayer.Library()) != 1 || len(a.Hand()) != 1 || len(a.Library()) != 0 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 {
		t.Fatal("clone payment changed original game")
	}
}
