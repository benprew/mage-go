package ai

import (
	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

// makePerm creates a creature permanent with cleared summoning sickness.
func makePerm(name, cost string, power, toughness int, owner uuid.UUID, opts ...mage.CardOption) *mage.Permanent {
	card := mage.NewCreature(name, cost, power, toughness, opts...)
	card.SetOwner(owner)
	perm := mage.NewPermanent(card, owner)
	perm.RevokeBaseAttr(core.AttrSummonSick)
	return perm
}

// makeGame creates a two-player game with BasePlayer instances.
func makeGame() (*mage.Game, *mage.BasePlayer, *mage.BasePlayer) {
	pa := mage.NewBasePlayer("Alice")
	pb := mage.NewBasePlayer("Bob")
	g := mage.NewGame(pa, pb)
	return g, pa, pb
}

// dummyStrategy is a no-op AIStrategy for tests that need to construct an
// AIPlayer but don't care about decisions.
type dummyStrategy struct{}

func (s *dummyStrategy) PriorityAction(_ mage.Player, _ *mage.Game, _ int, _ bool) interactive.PriorityAction {
	return interactive.PriorityAction{Type: interactive.ActionPass}
}
func (s *dummyStrategy) Attackers(_ mage.Player, _ *mage.Game) []uuid.UUID           { return nil }
func (s *dummyStrategy) Blockers(_ mage.Player, _ *mage.Game) []mage.BlockAssignment { return nil }
