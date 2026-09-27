package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestReturnEachToHandOwnerAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewArtifact("First", "")
	first.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(first, b.PlayerID())
	second := NewArtifact("Second", "")
	second.SetOwner(a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	other := NewArtifact("Other", "")
	other.SetOwner(b.PlayerID())
	g.PutOnBattlefield(other, a.PlayerID())
	phased := NewArtifact("Phased", "")
	phased.SetOwner(a.PlayerID())
	q := g.PutOnBattlefield(phased, a.PlayerID())
	g.PhaseOut(q.ID())
	clone := g.Clone()
	if err := ReturnEachToHand(PermanentsOwnedBy(You, IsArtifact)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.FindPermanent(p.ID()) != nil || clone.FindPermanent(second.ID()) != nil || clone.FindPermanent(other.ID()) == nil || clone.FindPermanentIncludingPhased(q.ID()) == nil {
		t.Fatal("wrong battlefield membership")
	}
	if len(clone.GetPlayer(a.PlayerID()).Hand()) != 2 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 0 {
		t.Fatal("returned cards did not reach their owner's hand")
	}
	if g.FindPermanent(p.ID()) == nil || len(a.Hand()) != 0 {
		t.Fatal("batch changed original game")
	}
	clone.PutOnBattlefield(first, b.PlayerID())
	if clone.FindPermanent(first.ID()) == nil || clone.FindPermanent(first.ID()).incarnationID == p.incarnationID {
		t.Fatal("returned card reused its old incarnation")
	}
}

func TestReturnEachToHandIsOneBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewArtifact("First", "")
	second := NewArtifact("Second", "")
	observer := NewLand("Observer", WithAbility(NewStateTriggered(false, Sacrifice(This)).SetCondition(
		func(_ *core.GameEvent, game GameReader, _, _ uuid.UUID) bool {
			return game.FindPermanent(first.ID()) == nil && game.FindPermanent(second.ID()) != nil
		})))
	for _, card := range []Card{first, second, observer} {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	if err := ReturnEachToHand(MatchingPermanents(IsArtifact)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(observer.ID()) == nil {
		t.Fatal("state trigger observed an intermediate return")
	}
}

func TestReturnEachToHandDeclaration(t *testing.T) {
	player := DeclarePlayerTarget("player")
	valid := NewInstant("Valid", "", Spell(ReturnEachToHand(PermanentsOwnedBy(player, IsArtifact)), WithTargets(player)))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, set := range []PermanentSet{nil, PermanentsOwnedBy(player, IsArtifact)} {
		bad := NewInstant("Bad", "", Spell(ReturnEachToHand(set)))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestBouncePermanentsToHandSkipsDuplicateAndStaleObjects(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Card", "")
	card.SetOwner(a.PlayerID())
	old := g.PutOnBattlefield(card, a.PlayerID())
	g.BouncePermanentToHand(old)
	a.RemoveFromHand(card.ID())
	current := g.PutOnBattlefield(card, a.PlayerID())
	token := NewArtifact("Token", "")
	token.SetOwner(a.PlayerID())
	tokenPerm := g.PutOnBattlefield(token, a.PlayerID())
	tokenPerm.IsToken = true
	g.BouncePermanentsToHand([]*Permanent{old, current, current, tokenPerm})
	if g.FindPermanent(card.ID()) != nil || g.FindPermanent(token.ID()) != nil {
		t.Fatal("current objects did not leave")
	}
	if got := len(a.Hand()); got != 1 {
		t.Fatalf("hand size = %d, want one card and no token", got)
	}
}
