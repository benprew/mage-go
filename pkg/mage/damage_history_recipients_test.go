package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageHistoryRecipientsCurrentMembership(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 2, 3), a.PlayerID())
	walker := g.PutOnBattlefield(resolvingCopyCard(t, func() Card { return NewPlaneswalker("Walker", "", 10) }), b.PlayerID())
	laterWalker := g.PutOnBattlefield(resolvingCopyCard(t, func() Card { return NewCreature("Later walker", "", 1, 10) }), b.PlayerID())
	for _, p := range []*Permanent{walker, laterWalker} {
		g.DealDamageToPermanent(p, 1, source.ID())
	}
	g.DealDamageToPlayer(a, 1, source.ID())
	g.DealDamageToPlayer(b, 1, source.ID())
	g.RunStepWithPriority(core.Cleanup)
	if err := g.BecomeCopyOfPermanent(g.FindPermanent(laterWalker.ID()), g.FindPermanent(walker.ID()), false); err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(laterWalker.ID()).Counters[core.Loyalty] = 10
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	recipients := DamageRecipients(PlayersDealtDamageBy(This, OpponentsOf(You)), PermanentsDealtDamageBy(This, MatchingPermanents(IsPlaneswalker)))
	selected := recipients.resolveDamageRecipients(ctx, 1, g.CaptureDamageSource(source.ID()))
	if len(selected) != 3 {
		t.Fatalf("current type selection: got %d recipients", len(selected))
	}
	if err := DealDamageEach(This, recipients, Fixed(1)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 19 || b.Life() != 18 || g.FindPermanent(walker.ID()).Counters[core.Loyalty] != 8 || g.FindPermanent(laterWalker.ID()).Counters[core.Loyalty] != 9 {
		t.Fatal("controller, current type, or durable history changed")
	}
}
