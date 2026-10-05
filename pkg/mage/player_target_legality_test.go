package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestPlayerTargetResolutionChecksSpecification(t *testing.T) {
	for _, typed := range []bool{false, true} {
		for _, opponentOnly := range []bool{false, true} {
			for _, copyController := range []bool{false, true} {
				g, a, b := newSpellCopyGame()
				var target = TargetPlayer()
				if opponentOnly {
					target = TargetOpponent()
				}
				if typed {
					var options []PlayerTargetOption
					if opponentOnly {
						options = append(options, TargetOpponentOnly())
					}
					target = DeclarePlayerTarget("recipient", options...)
				}
				controller := a.PlayerID()
				if copyController {
					controller = b.PlayerID()
				}
				card := NewInstant("Life loss", "", WithAbility(NewTargetedSpell(target, TargetPlayerLoseLife(Fixed(2)), DrawFor(You, Fixed(1)))))
				obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: controller, Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: []Target{target}, Effects: []Effect{TargetPlayerLoseLife(Fixed(2)), DrawFor(You, Fixed(1))}}
				g.PushStack(obj)
				clone := g.Clone()
				g.ResolveTopOfStack()
				illegal := opponentOnly && copyController
				wantLife, wantHand := 18, 1
				if illegal {
					wantLife, wantHand = 20, 0
				}
				if b.Life() != wantLife || len(g.GetPlayer(controller).Hand()) != wantHand {
					t.Fatalf("typed=%v opponent=%v copy=%v: life=%d hand=%d", typed, opponentOnly, copyController, b.Life(), len(g.GetPlayer(controller).Hand()))
				}
				if clone.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("clone changed")
				}
				clone.ResolveTopOfStack()
				if clone.GetPlayer(b.PlayerID()).Life() != wantLife || len(clone.GetPlayer(controller).Hand()) != wantHand {
					t.Fatal("clone target legality")
				}
			}
		}
	}
}
