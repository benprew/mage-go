package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTypedDrawAmountContext(t *testing.T) {
	for _, each := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		for range 3 {
			a.AddToLibrary(NewArtifact("A card", ""))
			b.AddToLibrary(NewArtifact("B card", ""))
		}
		source := NewCreature("Amount", "{2}", 2, 3)
		g.PutOnBattlefield(source, a.PlayerID())
		saved := BindPermanent("amount")
		var draw = DrawFor(You, PowerOf(From(saved)))
		if each {
			draw = DrawEach(EachPlayer, PowerOf(From(saved)))
		}
		body := Sequence(Snapshot(This, saved), draw)
		card := NewCreature("Draw", "", 1, 1, WithAbility(Activate(Costs(), body)))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		clone := g.Clone()
		if err := ApplyEffect(clone, body, source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		wantB := 0
		if each {
			wantB = 2
		}
		if len(clone.GetPlayer(a.PlayerID()).Hand()) != 2 || len(clone.GetPlayer(b.PlayerID()).Hand()) != wantB {
			t.Fatal("incorrect saved amount or recipient")
		}
		if len(a.Hand()) != 0 || len(b.Hand()) != 0 {
			t.Fatal("draw changed original game")
		}
		if err := ApplyEffect(g, DrawFor(You, ValueAtResolution(TargetPermanentManaValue())), uuid.Nil, b.PlayerID(), []uuid.UUID{source.ID()}); err != nil {
			t.Fatal(err)
		}
		if len(b.Hand()) != 2 {
			t.Fatal("amount lost the outer target context")
		}
	}
}

func TestTypedDrawAmountCaptureAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	calls := 0
	amount := countedLifeAmount{&calls}
	for _, body := range []ResolvingEffect{DrawFor(ControllerOf(This), amount), DrawEach(OpponentsOf(ControllerOf(This)), amount)} {
		if err := ApplyEffect(g, body, uuid.New(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 0 {
		t.Fatal("absent subjects evaluated amount")
	}
	if err := ApplyEffect(g, DrawEach(EachPlayer, amount), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || !a.DrewFromEmpty() || !b.DrewFromEmpty() {
		t.Fatal("collection did not capture once or retain empty-library behavior")
	}
}

func TestTypedDrawAmountValidation(t *testing.T) {
	for _, amount := range []ResolutionValue{nil, PowerOf(nil), PowerOf(From(BindPermanent("missing"))), PowerOf(DeclarePermanentTarget("undeclared", IsCreature)), ValueAtResolution(nil)} {
		for _, body := range []ResolvingEffect{DrawFor(You, amount), DrawEach(EachPlayer, amount)} {
			card := NewSorcery("Invalid draw", "", WithAbility(Spell(body)))
			if err := ValidateCardDefinition(card.Name(), card); err == nil {
				t.Fatal("invalid amount accepted")
			}
		}
	}
	for _, each := range []bool{false, true} {
		var body = DrawFor(You, Fixed(2))
		if each {
			body = DrawEach(EachPlayer, Fixed(2))
		}
		info := InspectDefinition(body)
		if info.Properties.DrawCount != 2 || info.Properties.Outcome != OutcomeBenefit || info.Properties.Mass != each {
			t.Fatalf("metadata = %+v", info)
		}
	}
}

func TestTypedDrawAmountOrderAndReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetActivePlayerIndex(1)
	for range 2 {
		a.AddToLibrary(NewArtifact("A card", ""))
		b.AddToLibrary(NewArtifact("B card", ""))
	}
	var order []uuid.UUID
	watcher := NewTriggered(core.EvtCardDrawn, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		order = append(order, evt.PlayerID)
		return false
	})
	g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(watcher)), a.PlayerID())
	if err := ApplyEffect(g, DrawEach(EachPlayer, SubtractValues(Fixed(3), Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(order) != 4 || order[0] != b.PlayerID() || order[1] != b.PlayerID() || order[2] != a.PlayerID() || order[3] != a.PlayerID() {
		t.Fatalf("draw order = %v", order)
	}
	replacements := 0
	source := NewArtifact("Replacement", "")
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddEmptyLibraryDrawReplacement(source.ID(), a.PlayerID(), func(_ *Game, _ uuid.UUID) { replacements++ })
	if err := ApplyEffect(g, DrawFor(You, SubtractValues(Fixed(3), Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if replacements != 2 || a.DrewFromEmpty() || len(order) != 4 {
		t.Fatal("individual replacements or draw events changed")
	}
	for _, n := range []int{-1, 0} {
		if err := ApplyEffect(g, DrawEach(EachPlayer, Fixed(n)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if replacements != 2 || b.DrewFromEmpty() {
		t.Fatal("nonpositive amount drew cards")
	}
}

func TestTypedDrawAmountCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 3 {
		a.AddToLibrary(NewArtifact("A card", ""))
		b.AddToLibrary(NewArtifact("B card", ""))
	}
	source := NewCreature("Amount", "", 2, 3)
	g.PutOnBattlefield(source, a.PlayerID())
	target := DeclarePermanentTarget("amount", IsCreature)
	ability := Spell(DrawFor(You, PowerOf(target)), WithTargets(target))
	card := NewSorcery("Draw", "", WithAbility(ability))
	original := &StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{source.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(original)
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	g.ResolveStackObject(copied)
	if len(a.Hand()) != 0 || len(b.Hand()) != 2 {
		t.Fatal("copy lost controller or target amount")
	}
	g.ResolveStackObject(original)
	if len(a.Hand()) != 2 || len(b.Hand()) != 2 {
		t.Fatal("original lost its amount")
	}
}
