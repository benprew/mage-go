package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestFlipCopyCloneAndZoneChange(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	card := NewCreature("Front", "{U}", 2, 3, WithFlipFace(NewCreature("Back", "{R}", 4, 5, WithKeyword(Flying))))
	card.SetOwner(owner)
	source := g.PutOnBattlefield(card, owner)
	source.Damage = 1
	branch := g.Clone()
	branch.FlipPermanent(source.ID())
	flipped := branch.FindPermanent(source.ID())
	if !flipped.Flipped || flipped.Name() != "Back" || flipped.CurrentPower(branch) != 4 || flipped.Damage != 1 || !flipped.HasKeyword(Flying) {
		t.Fatal("flip lost status or characteristics")
	}
	if source.Flipped || source.Name() != "Front" || source.HasKeyword(Flying) {
		t.Fatal("flipping a clone changed the original permanent")
	}
	if !slices.Contains(flipped.Colors(), Blue) || slices.Contains(flipped.Colors(), Red) || flipped.Card.ManaCost().String() != "{U}" {
		t.Fatal("flipping changed mana cost or color")
	}
	branch.FlipPermanent(flipped.ID())
	if !flipped.Flipped || flipped.Name() != "Back" {
		t.Fatal("repeated flip changed a flipped permanent")
	}
	branch.BouncePermanentToHand(flipped)
	returnedCard, ok := branch.players[0].RemoveFromHand(flipped.ID())
	if !ok || returnedCard.Name() != "Front" {
		t.Fatal("departing flip card did not regain its normal characteristics")
	}
	returned := branch.PutOnBattlefield(returnedCard, owner)
	if returned.Flipped || returned.Name() != "Front" || returned.Damage != 0 {
		t.Fatal("returned flip card retained its previous status")
	}
}

func TestCopyOptionsKeepColorless(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	source := g.PutOnBattlefield(NewCreature("Colorless", "{0}", 1, 1), owner)
	model := g.PutOnBattlefield(NewCreature("Green", "{G}", 3, 4), owner)
	g.UpdateCopyEffect(source.ID(), model, CopyOptions{KeepColor: true})
	if len(source.Colors()) != 0 || source.Card.ManaCost().Green != 1 {
		t.Fatal("copy did not preserve colorlessness while copying mana cost")
	}
	copiedCard := source.Card.NewCopy()
	if len(copiedCard.Colors()) != 0 {
		t.Fatal("copy of a colorless copy acquired its mana cost's color")
	}
}

func TestCloneStackPreservesPermanentIncarnations(t *testing.T) {
	g := newPriorityTestGame()
	owner := g.players[0].PlayerID()
	card := NewCreature("Target", "{0}", 3, 4)
	card.SetOwner(owner)
	target := g.PutOnBattlefield(card, owner)
	g.PushStack(&StackObject{ID: uuid.New(), IsAbility: true, SourceID: target.ID(), Controller: owner, Targets: []uuid.UUID{target.ID()}, Effects: []Effect{GainLife(1)}})
	branch := g.Clone()
	original := g.Stack().Peek()
	copied := branch.Stack().Peek()
	if copied.SourceIncarnation != original.SourceIncarnation || copied.TargetIncarnations[target.ID()] != target.incarnationID {
		t.Fatal("stack clone lost source or target incarnation")
	}
	branch.BouncePermanentToHand(branch.FindPermanent(target.ID()))
	returnedCard, ok := branch.players[0].RemoveFromHand(target.ID())
	if !ok {
		t.Fatal("bounced target is not in hand")
	}
	branch.PutOnBattlefield(returnedCard, owner)
	life := branch.players[0].Life()
	branch.ResolveStack()
	if branch.players[0].Life() != life {
		t.Fatal("cloned stack ability resolved against a new target incarnation")
	}
	g.ResolveStack()
	if g.players[0].Life() != life+1 {
		t.Fatal("clone changed the original stack target")
	}
}
