package heuristic

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
)

func controls(g *mage.Game, ids []uuid.UUID, playerID uuid.UUID) bool {
	for _, id := range ids {
		perm := g.FindPermanent(id)
		if perm == nil || perm.ControllerID() != playerID {
			return false
		}
	}
	return true
}

// upToTwoBoost mimics "Up to two target creatures each get +2/+2" (Dauntless
// Onslaught): a beneficial multi-target instant.
func upToTwoBoost(owner uuid.UUID) mage.Card {
	card := mage.NewInstant("Test Onslaught", "{2}{W}",
		mage.NewMultiTargetSpell(
			[]mage.Target{mage.TargetUpToNCreatures(2)},
			mage.Boost(mage.Fixed(2), mage.Fixed(2)).Targeting(mage.ToAllTargets()),
		),
	)
	card.SetOwner(owner)
	return card
}

// A beneficial "up to two target creatures" pump should buff two of the AI's
// own creatures rather than stopping at a single target.
func TestAutoSelectTargets_MultiTargetPump_BuffsTwoOwnCreatures(t *testing.T) {
	g, pa, pb := makeGame()

	for _, name := range []string{"Alpha Bear", "Beta Bear", "Gamma Bear"} {
		g.AddToBattlefield(makePerm(name, "{1}{G}", 2, 2, pa.PlayerID()))
	}
	g.AddToBattlefield(makePerm("Enemy", "{1}{G}", 3, 3, pb.PlayerID()))

	card := upToTwoBoost(pa.PlayerID())
	pa.AddToHand(card)

	s := New(ai.MidrangeWeighted)
	targets := s.autoSelectTargets(pa, g, card)
	if len(targets) != 2 {
		t.Fatalf("multi-target pump should pick two targets, got %d: %v", len(targets), targets)
	}
	if !controls(g, targets, pa.PlayerID()) {
		t.Fatalf("beneficial pump should target own creatures, got %v", targets)
	}
	if targets[0] == targets[1] {
		t.Fatalf("multi-target should pick distinct creatures, got %v", targets)
	}
}

// With only one creature available, an "up to two" spell still resolves with a
// single target (Min is 0, so partial fills are legal).
func TestAutoSelectTargets_MultiTargetPump_SingleAvailable(t *testing.T) {
	g, pa, _ := makeGame()
	g.AddToBattlefield(makePerm("Lone Bear", "{1}{G}", 2, 2, pa.PlayerID()))

	card := upToTwoBoost(pa.PlayerID())
	pa.AddToHand(card)

	s := New(ai.MidrangeWeighted)
	targets := s.autoSelectTargets(pa, g, card)
	if len(targets) != 1 {
		t.Fatalf("with one creature, should pick exactly one target, got %v", targets)
	}
}
