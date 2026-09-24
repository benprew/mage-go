package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTypedDiscardAmountContext(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 3 {
		a.AddToHand(NewArtifact("A card", ""))
		b.AddToHand(NewArtifact("B card", ""))
	}
	source := NewCreature("Amount", "{2}", 2, 3)
	g.PutOnBattlefield(source, a.PlayerID())
	saved := BindPermanent("amount")
	body := Sequence(Snapshot(This, saved), Discard(You, PowerOf(From(saved))))
	card := NewCreature("Discard", "", 1, 1, WithAbility(Activate(Costs(), body)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := ApplyEffect(clone, body, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 3 || len(a.Hand()) != 3 {
		t.Fatal("saved amount, recipient, or clone isolation changed")
	}
	if err := ApplyEffect(g, Discard(You, ValueAtResolution(TargetPermanentManaValue())), uuid.Nil, b.PlayerID(), []uuid.UUID{source.ID()}); err != nil {
		t.Fatal(err)
	}
	if len(b.Hand()) != 1 || len(a.Hand()) != 3 {
		t.Fatal("amount lost outer targets")
	}
}

type discardChoicePlayer struct {
	*BasePlayer
	requests []int
}

func (p *discardChoicePlayer) ChooseMayAbility(string) bool { return true }
func (p *discardChoicePlayer) ChooseCardsFromHand(n int, _ string, _ GameReader) []Card {
	p.requests = append(p.requests, n)
	hand := p.Hand()
	n = min(n, len(hand))
	return append([]Card(nil), hand[len(hand)-n:]...)
}

func TestTypedDiscardAmountChoiceAndReplacement(t *testing.T) {
	a := NewBasePlayer("A")
	b := &discardChoicePlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	for range 3 {
		b.AddToHand(NewArtifact("Card", ""))
	}
	chosen := b.Hand()[2].ID()
	events := 0
	watcher := NewTriggered(core.EvtDiscard, false, DrawFor(You, Fixed(1))).SetCondition(func(evt *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		events++
		if evt.PlayerID != b.PlayerID() || evt.SourceID != chosen {
			t.Error("wrong discard event")
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Library", "", WithStaticAbility(DiscardToLibraryReplacement())), b.PlayerID())
	if err := ApplyEffect(g, Discard(ActivePlayer, Fixed(0)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.SetActivePlayerIndex(1)
	for _, n := range []int{-2, 0} {
		if err := ApplyEffect(g, Discard(ActivePlayer, Fixed(n)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(b.requests) != 0 {
		t.Fatal("nonpositive amount requested a choice")
	}
	calls := 0
	amount := MinValue(Fixed(1), countedLifeAmount{&calls})
	if err := ApplyEffect(g, Discard(ControllerOf(This), amount), uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("absent recipient evaluated amount")
	}
	if err := ApplyEffect(g, Discard(ActivePlayer, amount), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(b.requests) != 1 || b.requests[0] != 1 || len(b.Hand()) != 2 || len(b.Library()) != 1 || b.Library()[0].ID() != chosen || len(b.Graveyard()) != 0 || events != 1 {
		t.Fatal("amount capture, affected-player choice, replacement, or event changed")
	}
}

func TestTypedDiscardAmountShortHandAndContinuation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("Only card", ""))
	for range 2 {
		body := Sequence(Discard(You, SubtractValues(Fixed(5), Fixed(1))), GainLifeFor(You, Fixed(1)))
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(a.Hand()) != 0 || len(a.Graveyard()) != 1 || a.Life() != 22 {
		t.Fatal("short or empty hand stopped resolution")
	}
}

func TestTypedDiscardAmountValidation(t *testing.T) {
	for _, amount := range []ResolutionValue{nil, PowerOf(nil), PowerOf(From(BindPermanent("missing"))), PowerOf(DeclarePermanentTarget("undeclared", IsCreature)), ValueAtResolution(nil)} {
		card := NewSorcery("Invalid discard", "", WithAbility(Spell(Discard(You, amount))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid amount accepted")
		}
	}
	calls := 0
	info := InspectDefinition(Discard(You, countedLifeAmount{&calls}))
	if info.Properties.Outcome != OutcomeDetriment || calls != 0 {
		t.Fatal("inspection changed metadata or evaluated amount")
	}
}

func TestTypedDiscardAmountCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 3 {
		a.AddToHand(NewArtifact("A card", ""))
		b.AddToHand(NewArtifact("B card", ""))
	}
	source := NewCreature("Amount", "", 2, 3)
	g.PutOnBattlefield(source, a.PlayerID())
	target := DeclarePermanentTarget("amount", IsCreature)
	ability := Spell(Discard(You, PowerOf(target)), WithTargets(target))
	card := NewSorcery("Discard", "", WithAbility(ability))
	original := &StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{source.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(original)
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	g.ResolveStackObject(copied)
	if len(a.Hand()) != 3 || len(b.Hand()) != 1 {
		t.Fatal("copy lost controller or amount binding")
	}
	g.ResolveStackObject(original)
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 {
		t.Fatal("original lost its amount")
	}
}
