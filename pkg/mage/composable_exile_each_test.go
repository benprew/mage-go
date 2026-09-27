package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestExileEachUsesOneZoneChangeBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewArtifact("First", ""), a.PlayerID())
	second := g.PutOnBattlefield(NewArtifact("Second", ""), b.PlayerID())
	observer := NewLand("Observer", WithAbility(NewStateTriggered(false, Sacrifice(This)).SetCondition(
		func(_ *core.GameEvent, game GameReader, _, _ uuid.UUID) bool {
			return game.FindPermanent(first.ID()) == nil && game.FindPermanent(second.ID()) != nil
		})))
	g.PutOnBattlefield(observer, a.PlayerID())
	if err := ExileEach(MatchingPermanents(IsArtifact)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(g.GetExile()) != 2 || g.FindPermanent(observer.ID()) == nil {
		t.Fatal("an intermediate exile was visible")
	}
}

func TestExileEachIdentityAndRegistration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Artifact", "")
	old := g.PutOnBattlefield(card, a.PlayerID())
	g.RemoveFromBattlefield(old)
	current := g.PutOnBattlefield(card, b.PlayerID())
	clone := g.Clone()
	if err := ExileEach(fixedTapSet{[]*Permanent{old, current, current}}).Apply(&EffectContext{Game: clone}); err != nil {
		t.Fatal(err)
	}
	if len(clone.GetExile()) != 1 || g.FindPermanent(current.ID()) == nil {
		t.Fatal("exile changed the wrong incarnation or shared clone state")
	}
	good := NewSorcery("Good", "", Spell(ExileEach(MatchingPermanents(IsArtifact))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	bad := NewSorcery("Bad", "", Spell(ExileEach(nil)))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
		t.Fatal("nil set accepted at registration")
	}
}

func TestExileEachSkipsPhasedAndReturnedTargets(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Returned", "", 1, 1)
	old := g.PutOnBattlefield(card, b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Phased", "", 1, 1), b.PlayerID())
	targets := DeclarePermanentTargets("creatures", 1, 2, IsCreature)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{
		targets.targetDeclaration: {{id: old.ID(), incarnation: old.incarnationID}, {id: other.ID(), incarnation: other.incarnationID}},
	}}
	g.RemoveFromBattlefield(old)
	current := g.PutOnBattlefield(card, b.PlayerID())
	g.PhaseOut(other.ID())
	if err := ExileEach(targets).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(current.ID()) == nil || g.FindPermanentIncludingPhased(other.ID()) == nil || len(g.GetExile()) != 0 {
		t.Fatal("an absent or phased target was exiled")
	}
}

func TestExileEachQueuesSourceLeaveTrigger(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewArtifact("First", "", WithAbility(WhenEvent(LeavesBattlefield(This), GainLifeFor(You, Fixed(2)))))
	second := NewArtifact("Second", "", WithAbility(WhenEvent(LeavesBattlefield(This), GainLifeFor(You, Fixed(3)))))
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	if err := ExileEach(ControlledPermanents(IsArtifact)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 25 {
		t.Fatalf("life after both leave triggers = %d", a.Life())
	}
}

func TestExileEachTokenDoesNotRemainInExile(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	token := g.PutOnBattlefield(NewArtifact("Token", ""), a.PlayerID())
	token.IsToken = true
	card := g.PutOnBattlefield(NewArtifact("Card", ""), b.PlayerID())
	if err := ExileEach(MatchingPermanents(IsArtifact)).Apply(&EffectContext{Game: g}); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(token.ID()) != nil || g.FindPermanent(card.ID()) != nil || len(g.GetExile()) != 1 || g.GetExile()[0].Card.ID() != card.ID() {
		t.Fatal("token stayed in exile or nontoken card was lost")
	}
}
