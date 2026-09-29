package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDiscardCostBatchEventsObserveCompletePayment(t *testing.T) {
	for _, replaced := range []bool{false, true} {
		t.Run(map[bool]string{false: "graveyard", true: "library"}[replaced], func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			if replaced {
				g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
			}
			source := NewArtifact("Synthetic discard payment", "", WithAbility(Activate(Costs(DiscardCost(2)), GainLifeFor(You, Fixed(1)))))
			g.PutOnBattlefield(source, a.PlayerID())
			for _, name := range []string{"First", "Second"} {
				a.AddToHand(NewLand(name))
			}
			seen := 0
			watcher := NewTriggered(EvtDiscard, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *GameEvent, game GameReader, _, _ uuid.UUID) bool {
				p := game.GetPlayer(evt.PlayerID)
				if p == a {
					seen++
				}
				if len(p.Hand()) != 0 || len(p.Graveyard())+len(p.Library()) != 2 {
					t.Errorf("discard event saw partial payment: hand=%d grave=%d library=%d", len(p.Hand()), len(p.Graveyard()), len(p.Library()))
				}
				if !replaced && evt.DiscardedCardVisit == uuid.Nil {
					t.Error("discard event lost graveyard visit")
				}
				if replaced && evt.DiscardedCardVisit != uuid.Nil {
					t.Error("hidden destination exposed graveyard visit")
				}
				return false
			})
			g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(watcher)), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if seen != 2 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 2 {
				t.Fatalf("discard events = %d", seen)
			}
			g.ResolveStack()
			if a.Life() != 21 {
				t.Fatal("payment did not resolve")
			}
		})
	}
}

type discardBatchHandReplacement struct {
	replacementBase
	card uuid.UUID
}

func (r *discardBatchHandReplacement) Matches(action Action, game GameReader) bool {
	d, ok := action.(*DiscardAction)
	return ok && !d.CausedByEffect() && d.CardID() == r.card && len(game.GetPlayer(d.PlayerID()).Hand()) == 2
}
func (*discardBatchHandReplacement) Replace(action Action, _ *Game) Action {
	return action.(*DiscardAction).withDestination(ZoneLibrary)
}
func (*discardBatchHandReplacement) IsActive(GameReader) bool { return true }
func (r *discardBatchHandReplacement) Clone() ReplacementEffect {
	return &discardBatchHandReplacement{card: r.card}
}

func TestDiscardCostBatchPreparesReplacementsBeforeMoves(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewLand("First"), NewLand("Second")
	a.AddToHand(first)
	a.AddToHand(second)
	g.effects.AddReplacement(&discardBatchHandReplacement{card: second.ID()})
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{DiscardCost(2)}})
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(a.Graveyard()) != 1 || len(a.Library()) != 1 || a.Library()[0].ID() != second.ID() {
		t.Fatal("replacement observed an intermediate hand")
	}
}

type preventOneCostDiscardReplacement struct {
	replacementBase
	card uuid.UUID
}

func (r *preventOneCostDiscardReplacement) Matches(action Action, _ GameReader) bool {
	d, ok := action.(*DiscardAction)
	return ok && !d.CausedByEffect() && d.CardID() == r.card
}
func (*preventOneCostDiscardReplacement) Replace(Action, *Game) Action { return nil }
func (*preventOneCostDiscardReplacement) IsActive(GameReader) bool     { return true }
func (r *preventOneCostDiscardReplacement) Clone() ReplacementEffect {
	return &preventOneCostDiscardReplacement{card: r.card}
}

func TestDiscardCostBatchFailureAndClone(t *testing.T) {
	for _, mode := range []string{"prevented", "returned visit", "clone"} {
		t.Run(mode, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			first, second := NewLand("First"), NewLand("Second")
			a.AddToHand(first)
			a.AddToHand(second)
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), DiscardCost(2)}})
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "prevented":
				g.effects.AddReplacement(&preventOneCostDiscardReplacement{card: second.ID()})
				if err := DiscardCost(2).Pay(uuid.Nil, a.PlayerID(), g); err == nil {
					t.Fatal("prevented discard paid cost")
				}
			case "returned visit":
				card, _ := a.RemoveFromHand(second.ID())
				a.AddToHand(card)
				if err := tx.Commit(); err == nil {
					t.Fatal("returned hand visit paid cost")
				}
			case "clone":
				clone := g.Clone()
				clonePlayer := clone.GetPlayer(a.PlayerID())
				if err := DiscardCost(2).Pay(uuid.Nil, a.PlayerID(), clone); err != nil {
					t.Fatal(err)
				}
				if len(clonePlayer.Graveyard()) != 2 || clone.PlayerDiscardCountThisTurn(a.PlayerID()) != 2 {
					t.Fatal("clone lost batch")
				}
			}
			if len(a.Hand()) != 2 || len(a.Graveyard()) != 0 || a.Life() != 20 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 || len(tx.paidCards) != 0 {
				t.Fatal("failed or cloned payment changed live state")
			}
		})
	}
}

func TestDiscardCostBatchReplacementRecheckRejectsWholePayment(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	a.AddToHand(NewLand("First"))
	a.AddToHand(NewLand("Second"))
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	g.effects.AddReplacement(&discardToHandReplacement{})
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), DiscardCost(2)}}); err == nil {
		t.Fatal("prevented batch paid cost")
	}
	if len(a.Hand()) != 2 || len(a.Library()) != 0 || a.Life() != 20 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 {
		t.Fatal("failed preparation changed live state")
	}
}

func TestDiscardCostBatchDirectChoicePreflight(t *testing.T) {
	for _, mode := range []string{"duplicate", "too few", "outside", "nil", "zero"} {
		t.Run(mode, func(t *testing.T) {
			a := &fixedDiscardChoicesPlayer{BasePlayer: NewBasePlayer("A")}
			g := NewGame(a, NewBasePlayer("B"))
			first, second := NewLand("First"), NewLand("Second")
			a.AddToHand(first)
			a.AddToHand(second)
			count := 2
			switch mode {
			case "duplicate":
				a.selected = []Card{first, first}
			case "too few":
				a.selected = []Card{first}
			case "outside":
				a.selected = []Card{first, NewLand("Outside")}
			case "nil":
				a.selected = []Card{first, nil}
			case "zero":
				count = 0
			}
			err := DiscardCost(count).Pay(uuid.Nil, a.PlayerID(), g)
			if (err == nil) != (mode == "zero") {
				t.Fatalf("payment error = %v", err)
			}
			if len(a.Hand()) != 2 || len(a.Graveyard()) != 0 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 0 {
				t.Fatal("choice preflight moved cards")
			}
		})
	}
}
