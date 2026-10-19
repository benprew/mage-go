package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSavedSetLeaveLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	result := BindCreatedTokens("batch")
	token := DefineToken("Member", TokenPT(1, 1), TokenTypes(core.TypeCreature))
	source := g.PutOnBattlefield(NewArtifact("Maker", ""), a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}
	effect := Sequence(CreateTokenCopiesInto(You, Fixed(2), token, result), WhenAnyLeaves(FromCreatedTokens(result), GainLifeFor(You, Fixed(1))))
	if err := ValidateCardDefinition("Probe", NewArtifact("Probe", "", WithAbility(Activate(Costs(), effect)))); err != nil {
		t.Fatal(err)
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	members := g.FilterBattlefield(IsToken)
	g.triggers.ClearEndOfTurn()
	g.FireEvent(core.GameEvent{Type: core.EvtZoneChange, SourceID: members[0].ID(), SourceIncarnation: uuid.New(), FromZone: core.ZoneBattlefield, ToZone: core.ZoneExile})
	g.ResolveStack()
	if a.Life() != 20 || len(g.triggers.Delayed()) != 1 {
		t.Fatal("wrong visit matched or cleanup removed delayed event")
	}
	phase := g.PhaseOut(members[0].ID())
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("phasing triggered departure")
	}
	g.PhaseIn(phase)
	g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: members[0].ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.BouncePermanentToHand(source)
	if len(g.WaitingDelayedPermanentSources()) < 2 {
		t.Fatal("waiting query omitted saved members")
	}
	clone := g.Clone()
	clone.triggers.Delayed()[0].leaveVisits[0] = capturedPermanent{}
	if g.triggers.Delayed()[0].leaveVisits[0].incarnation == uuid.Nil {
		t.Fatal("clone shares subscription members")
	}
	clone = g.Clone()
	for _, game := range []*Game{g, clone} {
		game.BouncePermanentsToHand([]*Permanent{game.FindPermanent(members[0].ID()), game.FindPermanent(members[1].ID())})
		game.PutTriggersOnStack()
		if game.StackSize() != 1 {
			t.Fatalf("one-shot group produced %d triggers", game.StackSize())
		}
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 21 || game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatal("saved controller or clone lost")
		}
	}
}

func TestSavedSetLeaveIdentityAndDefinitions(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Maker", ""), a.PlayerID())
	card := NewCreature("Subject", "", 1, 1)
	card.SetOwner(a.PlayerID())
	subject := g.PutOnBattlefield(card, a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}
	if err := WhenAnyLeaves(MatchingPermanents(IsCreature), GainLifeFor(You, Fixed(1))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(subject)
	g.ResolveStack()
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, a.PlayerID())
	g.BouncePermanentToHand(returned)
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("returned visit retriggered")
	}
	for _, e := range []Effect{WhenAnyLeaves(nil, GainLifeFor(You, Fixed(1))), WhenAnyLeaves(MatchingPermanents(AnyPermanent), nil), WhenAnyLeaves(FromCreatedTokens(BindCreatedTokens("missing")), GainLifeFor(You, Fixed(1)))} {
		if ValidateCardDefinition("Bad", NewArtifact("Bad", "", WithAbility(Activate(Costs(), e)))) == nil {
			t.Fatal("invalid delayed declaration accepted")
		}
	}
	if err := WhenAnyLeaves(MatchingPermanents(IsCreature), GainLifeFor(You, Fixed(1))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.triggers.Delayed()) != 0 {
		t.Fatal("empty set installed a subscription")
	}
}

func TestSavedSetLeavePhasedSourceRegistration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	phase := g.PhaseOut(source.ID())
	if err := WhenLeaves(This, GainLifeFor(You, Fixed(1))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.triggers.Delayed()) != 1 {
		t.Fatal("phased visit lost delayed source reference")
	}
	g.PhaseIn(phase)
	g.BouncePermanentToHand(source)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("phase-in departure did not trigger")
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 21 {
			t.Fatal("pending clone lost delayed body")
		}
	}
}
