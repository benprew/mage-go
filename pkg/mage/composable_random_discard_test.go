package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTypedRandomDiscardContextAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	cards := []Card{NewArtifact("One", ""), NewArtifact("Two", ""), NewArtifact("Three", "")}
	for _, c := range cards {
		a.AddToHand(c)
	}
	source := NewCreature("Amount", "{2}", 2, 2)
	g.PutOnBattlefield(source, b.PlayerID())
	g.SetRandomResults([]int{2, 0, 7})
	clone := g.Clone()
	body := DiscardAtRandom(You, ValueAtResolution(TargetPermanentManaValue()))
	if err := ApplyEffect(clone, body, source.ID(), a.PlayerID(), []uuid.UUID{source.ID()}); err != nil {
		t.Fatal(err)
	}
	p := clone.GetPlayer(a.PlayerID())
	if len(p.Hand()) != 1 || p.Hand()[0].ID() != cards[1].ID() || len(p.Graveyard()) != 2 || p.Graveyard()[0].ID() != cards[2].ID() {
		t.Fatal("wrong random selection or dynamic amount")
	}
	if len(a.Hand()) != 3 || g.RandIntn(10) != 2 || clone.RandIntn(10) != 7 {
		t.Fatal("clone changed original state or random queue")
	}
}

func TestTypedRandomDiscardReplacementAndAbsence(t *testing.T) {
	a := NewBasePlayer("A")
	b := &discardChoicePlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	for range 2 {
		b.AddToHand(NewArtifact("Card", ""))
	}
	chosen := b.Hand()[1].ID()
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
	g.SetActivePlayerIndex(1)
	g.SetRandomResults([]int{1, 7})
	calls := 0
	for _, body := range []ResolvingEffect{DiscardAtRandom(ActivePlayer, Fixed(0)), DiscardAtRandom(ActivePlayer, Fixed(-1)), DiscardAtRandom(ControllerOf(This), countedLifeAmount{&calls})} {
		if err := ApplyEffect(g, body, uuid.New(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 0 {
		t.Fatal("absent subject evaluated amount")
	}
	if err := ApplyEffect(g, DiscardAtRandom(ActivePlayer, MinValue(Fixed(1), countedLifeAmount{&calls})), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(b.requests) != 0 || len(b.Library()) != 1 || b.Library()[0].ID() != chosen || events != 1 || g.RandIntn(10) != 7 {
		t.Fatal("choice, amount, replacement, event, or random queue changed")
	}
}

func TestTypedRandomDiscardShortHandAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewArtifact("Only", ""))
	g.SetRandomResults([]int{0, 7})
	body := Sequence(DiscardAtRandom(You, Fixed(5)), GainLifeFor(You, Fixed(1)))
	for range 2 {
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(a.Graveyard()) != 1 || a.Life() != 22 || g.RandIntn(10) != 7 {
		t.Fatal("short hand changed continuation or random consumption")
	}
	for _, body := range []ResolvingEffect{DiscardAtRandom(nil, Fixed(1)), DiscardAtRandom(You, nil), DiscardAtRandom(You, PowerOf(From(BindPermanent("missing"))))} {
		card := NewSorcery("Invalid", "", WithAbility(Spell(body)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid definition accepted")
		}
	}
	info := InspectDefinition(DiscardAtRandom(You, Fixed(2)))
	if len(info.Children) != 2 || info.Properties.Outcome != OutcomeDetriment {
		t.Fatal("missing metadata")
	}
}

func TestTypedRandomDiscardCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 3 {
		a.AddToHand(NewArtifact("A card", ""))
		b.AddToHand(NewArtifact("B card", ""))
	}
	source := NewCreature("Amount", "", 2, 3)
	g.PutOnBattlefield(source, a.PlayerID())
	target := DeclarePermanentTarget("amount", IsCreature)
	ability := Spell(DiscardAtRandom(You, ValueAtResolution(XValue())), WithTargets(target))
	card := NewSorcery("Discard", "", WithAbility(ability))
	original := &StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), XValue: 2, Targets: []uuid.UUID{source.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
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
