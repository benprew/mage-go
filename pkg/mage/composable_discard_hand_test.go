package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTypedDiscardEntireHandReplacementAndContinuation(t *testing.T) {
	a := NewBasePlayer("A")
	b := &discardChoicePlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("Keep", ""))
	for range 3 {
		b.AddToHand(NewArtifact("Discard", ""))
	}
	events := 0
	watcher := NewTriggered(core.EvtDiscard, false, DrawFor(You, Fixed(1))).SetCondition(func(evt *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		events++
		if evt.PlayerID != b.PlayerID() {
			t.Error("wrong discard recipient")
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Library", "", WithStaticAbility(DiscardToLibraryReplacement())), b.PlayerID())
	g.SetActivePlayerIndex(1)
	body := Sequence(DiscardEntireHand(ActivePlayer), GainLifeFor(You, Fixed(1)))
	for range 2 {
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(b.Hand()) != 0 || len(b.Library()) != 3 || len(b.Graveyard()) != 0 || events != 3 || len(a.Hand()) != 1 || a.Life() != 22 || len(b.requests) != 0 {
		t.Fatal("hand discard lost replacement, events, recipient, or continuation")
	}
}

func TestTypedDiscardEntireHandEventCopyAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("damaged")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(DealsDamageToOpponent(This, player), DiscardEntireHand(FromPlayer(player)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.DealDamageToPlayer(b, 1, source.ID())
	g.PutTriggersOnStack()
	original := g.StackPeek()
	if original == nil || len(original.Targets) != 0 {
		t.Fatal("missing untargeted trigger")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	copied := g.CopyStackObjectDirect(original, b.PlayerID(), false)
	if copied == nil {
		t.Fatal("missing copy")
	}
	a.AddToHand(NewArtifact("Keep", ""))
	b.AddToHand(NewArtifact("Late card", ""))
	clone := g.Clone()
	clone.ResolveStack()
	if len(clone.GetPlayer(b.PlayerID()).Hand()) != 0 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 || len(b.Hand()) != 1 {
		t.Fatal("clone or captured recipient changed")
	}
	g.ResolveStackObject(copied)
	if len(b.Hand()) != 0 || len(a.Hand()) != 1 {
		t.Fatal("copy lost event recipient")
	}
	b.AddToHand(NewArtifact("Later card", ""))
	g.ResolveStackObject(original)
	if len(b.Hand()) != 0 || len(b.Graveyard()) != 2 {
		t.Fatal("hand was captured before resolution")
	}
}

func TestTypedDiscardEntireHandValidationAndAbsence(t *testing.T) {
	for _, subject := range []PlayerRef{nil, FromPlayer(BindPlayer("missing")), DeclarePlayerTarget("undeclared")} {
		card := NewSorcery("Invalid", "", WithAbility(Spell(DiscardEntireHand(subject))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid subject accepted")
		}
	}
	info := InspectDefinition(DiscardEntireHand(You))
	if info.Properties.Outcome != OutcomeDetriment {
		t.Fatal("missing detriment metadata")
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("Keep", ""))
	if err := ApplyEffect(g, Sequence(DiscardEntireHand(ControllerOf(This)), GainLifeFor(You, Fixed(1))), uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 || a.Life() != 21 {
		t.Fatal("absent subject changed hand or stopped sequence")
	}
}
