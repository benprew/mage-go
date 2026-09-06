package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSkipNextUntapFollowsPermanentAcrossControlChanges(t *testing.T) {
	g, _, b, perm := controllerTestGame(t)
	perm.Tapped = true
	g.SkipNextUntap(perm.ID())
	g.AddControlEffect(ControlEffectSpec{
		SourceID: b.PlayerID(), TargetID: perm.ID(), ControllerID: b.PlayerID(), Duration: Indefinite,
	})

	g.SetActivePlayerIndex(0)
	g.doUntap()
	if !g.FindPermanent(perm.ID()).Tapped {
		t.Fatal("noncontroller's untap step consumed skip")
	}
	g.SetActivePlayerIndex(1)
	g.doUntap()
	if !g.FindPermanent(perm.ID()).Tapped {
		t.Fatal("permanent did not skip its new controller's untap")
	}
	g.doUntap()
	if g.FindPermanent(perm.ID()).Tapped {
		t.Fatal("skip was not consumed after one attempted untap")
	}
}

func TestSkipNextUntapRequiresAnUntapAttemptAndIsCloneSafe(t *testing.T) {
	g, _, _, perm := controllerTestGame(t)
	g.SetActivePlayerIndex(0)
	g.SkipNextUntap(perm.ID())
	g.doUntap()
	perm = g.MutablePermanent(perm.ID())
	perm.Tapped = true

	clone := g.Clone()
	clone.doUntap()
	if !clone.FindPermanent(perm.ID()).Tapped {
		t.Fatal("clone did not consume its skip")
	}
	clone.doUntap()
	if clone.FindPermanent(perm.ID()).Tapped {
		t.Fatal("clone skip was not consumed")
	}
	if !g.FindPermanent(perm.ID()).Tapped {
		t.Fatal("clone untap leaked to original")
	}
	g.doUntap()
	if !g.FindPermanent(perm.ID()).Tapped {
		t.Fatal("untapped permanent incorrectly consumed original skip")
	}
	g.doUntap()
	if g.FindPermanent(perm.ID()).Tapped {
		t.Fatal("original skip was not consumed")
	}
}

func TestStunCreatureUsesSkipNextUntap(t *testing.T) {
	g, a, _, perm := controllerTestGame(t)
	perm.Tapped = true
	if err := ApplyEffect(g, StunCreature(), uuid.New(), a.PlayerID(), []uuid.UUID{perm.ID()}); err != nil {
		t.Fatal(err)
	}
	g.doUntap()
	if !g.FindPermanent(perm.ID()).Tapped {
		t.Fatal("stunned creature untapped")
	}
	g.doUntap()
	if g.FindPermanent(perm.ID()).Tapped {
		t.Fatal("stun did not expire after one attempted untap")
	}
}

func TestExchangeControlIsAtomicAndUsesSnapshotControllers(t *testing.T) {
	g, a, b, first := controllerTestGame(t)
	secondCard := NewArtifact("Second", "{1}")
	secondCard.SetOwner(b.PlayerID())
	second := g.PutOnBattlefield(secondCard, b.PlayerID())
	first.RevokeBaseAttr(AttrSummonSick)
	second.RevokeBaseAttr(AttrSummonSick)

	if !g.ExchangeControl(first.ID(), second.ID(), uuid.New()) {
		t.Fatal("valid exchange failed")
	}
	if first.ControllerID() != b.PlayerID() || second.ControllerID() != a.PlayerID() {
		t.Fatalf("controllers after exchange = %s/%s", first.ControllerID(), second.ControllerID())
	}
	if !first.HasAttr(AttrSummonSick) || !second.HasAttr(AttrSummonSick) {
		t.Fatal("exchange did not update continuous-control timing")
	}

	clone := g.Clone()
	cloneFirst := clone.FindPermanent(first.ID())
	cloneSecond := clone.FindPermanent(second.ID())
	if !clone.ExchangeControl(cloneFirst.ID(), cloneSecond.ID(), uuid.New()) {
		t.Fatal("clone exchange failed")
	}
	cloneFirst = clone.FindPermanent(first.ID())
	cloneSecond = clone.FindPermanent(second.ID())
	if cloneFirst.ControllerID() != a.PlayerID() || cloneSecond.ControllerID() != b.PlayerID() {
		t.Fatal("later exchange did not win by timestamp")
	}
	if first.ControllerID() != b.PlayerID() || second.ControllerID() != a.PlayerID() {
		t.Fatal("clone exchange leaked to original")
	}
}

func TestExchangeControlMissingPermanentDoesNothing(t *testing.T) {
	g, a, b, first := controllerTestGame(t)
	secondCard := NewArtifact("Second", "{1}")
	secondCard.SetOwner(b.PlayerID())
	second := g.PutOnBattlefield(secondCard, b.PlayerID())
	g.RemoveFromBattlefield(second)

	if g.ExchangeControl(first.ID(), second.ID(), uuid.New()) {
		t.Fatal("exchange with missing permanent succeeded")
	}
	if first.ControllerID() != a.PlayerID() {
		t.Fatal("failed exchange partially changed control")
	}
}

func TestExchangeControlOfTargetsSharingPermanentTypeRevalidatesPair(t *testing.T) {
	g, a, b, creature := controllerTestGame(t)
	artifactCard := NewArtifact("Artifact", "{1}")
	artifactCard.SetOwner(b.PlayerID())
	artifact := g.PutOnBattlefield(artifactCard, b.PlayerID())

	if err := ApplyEffect(g, ExchangeControlOfTargetsSharingPermanentType(), uuid.New(), a.PlayerID(), []uuid.UUID{creature.ID(), artifact.ID()}); err != nil {
		t.Fatal(err)
	}
	if creature.ControllerID() != a.PlayerID() || artifact.ControllerID() != b.PlayerID() {
		t.Fatal("nonmatching permanent types were exchanged")
	}
}
