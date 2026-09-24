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

func TestSkipNextUntapExpiresWithoutUntapAttempt(t *testing.T) {
	for _, blocked := range []string{"untapped", "attribute", "limit"} {
		t.Run(blocked, func(t *testing.T) {
			g, _, _, perm := controllerTestGame(t)
			g.SetActivePlayerIndex(0)
			perm.Tapped = blocked != "untapped"
			if blocked == "attribute" {
				perm.GrantBaseAttr(AttrDoesNotUntap)
			}
			if blocked == "limit" {
				g.effects.Rules.CreatureUntapMax = 0
			}
			g.SkipNextUntap(perm.ID())
			g.doUntap()
			if g.turns.HasSkipNextUntap(perm.ID()) {
				t.Fatal("skip survived the specified untap step")
			}
			perm = g.MutablePermanent(perm.ID())
			perm.RevokeBaseAttr(AttrDoesNotUntap)
			g.effects.Rules.CreatureUntapMax = -1
			perm.Tapped = true
			g.doUntap()
			if g.FindPermanent(perm.ID()).Tapped {
				t.Fatal("expired skip blocked a later untap step")
			}
		})
	}
}

func TestSkipNextUntapRepeatedApplicationsShareOneStep(t *testing.T) {
	g, _, _, perm := controllerTestGame(t)
	perm.Tapped = true
	g.SkipNextUntap(perm.ID())
	g.SkipNextUntap(perm.ID())
	g.doUntap()
	if !g.FindPermanent(perm.ID()).Tapped {
		t.Fatal("permanent untapped during the blocked step")
	}
	g.doUntap()
	if g.FindPermanent(perm.ID()).Tapped {
		t.Fatal("repeated applications blocked a second step")
	}
}

func TestSkipNextUntapIsCloneSafe(t *testing.T) {
	g, _, _, perm := controllerTestGame(t)
	perm.Tapped = true
	g.SkipNextUntap(perm.ID())
	clone := g.Clone()
	clone.doUntap()
	clone.doUntap()
	if clone.FindPermanent(perm.ID()).Tapped {
		t.Fatal("clone skip did not expire")
	}
	if !g.FindPermanent(perm.ID()).Tapped || !g.turns.HasSkipNextUntap(perm.ID()) {
		t.Fatal("clone consumed the original skip or untapped its permanent")
	}
	g.doUntap()
	if !g.FindPermanent(perm.ID()).Tapped {
		t.Fatal("original lost its skip")
	}
	g.doUntap()
	if g.FindPermanent(perm.ID()).Tapped {
		t.Fatal("original skip did not expire")
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

func TestSkipNextUntapOtherUntapsAndStunCounters(t *testing.T) {
	g, _, _, p := controllerTestGame(t)
	p.Tapped = true
	g.SkipNextUntap(p.ID())
	if !g.UntapPermanent(p) || !g.turns.HasSkipNextUntap(p.ID()) {
		t.Fatal("restriction blocked an effect untap or expired outside the step")
	}
	p = g.MutablePermanent(p.ID())
	p.Tapped = true
	p.AddCounter(Stun, 1)
	g.doUntap()
	p = g.FindPermanent(p.ID())
	if !p.Tapped || p.Counters[Stun] != 1 {
		t.Fatal("blocked untap consumed a stun counter")
	}
	g.doUntap()
	p = g.FindPermanent(p.ID())
	if !p.Tapped || p.Counters[Stun] != 0 {
		t.Fatal("next untap did not use the stun counter")
	}
	g.doUntap()
	if g.FindPermanent(p.ID()).Tapped {
		t.Fatal("permanent did not untap after both restrictions ended")
	}
}

func TestSkipNextUntapDepartureAndPhasing(t *testing.T) {
	for _, change := range []string{"return", "phase cycle", "phased at step"} {
		t.Run(change, func(t *testing.T) {
			g, a, _, p := controllerTestGame(t)
			id := p.ID()
			p.Tapped = true
			g.SkipNextUntap(id)
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, a.PlayerID()).Tapped = true
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(id))
			case "phased at step":
				g.PhaseOut(id)
			}
			g.doUntap()
			if change == "phased at step" {
				g.PhaseIn([]uuid.UUID{id})
			}
			if got := g.FindPermanent(id).Tapped; got != (change != "return") {
				t.Fatalf("tapped = %v after %s", got, change)
			}
			if g.turns.HasSkipNextUntap(id) {
				t.Fatal("restriction survived the step")
			}
			g.doUntap()
			if g.FindPermanent(id).Tapped {
				t.Fatal("restriction applied to a later step")
			}
		})
	}
}
