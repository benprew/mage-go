package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDiscardEntireHandActualCount(t *testing.T) {
	for _, replaced := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		for range 3 {
			a.AddToHand(NewLand("Card"))
		}
		if replaced {
			g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
		}
		count := BindNumber("discarded")
		body := Sequence(DiscardEntireHand(You, DiscardAmountInto(count)), GainLifeFor(You, FromNumber(count)))
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 23 || len(a.Hand()) != 0 || len(a.Library()) != map[bool]int{false: 0, true: 3}[replaced] {
			t.Fatalf("replacement %v: count or destination is wrong", replaced)
		}
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 23 {
			t.Fatal("empty hand kept an old count")
		}
	}
}

func TestDiscardEntireHandCountPreventionAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewLand("Card"))
	g.effects.AddReplacement(&discardToHandReplacement{})
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	count := BindNumber("discarded")
	g.resolution.numberBindings = map[*NumberBinding]int{count: 9}
	clone := g.Clone()
	if err := DiscardEntireHand(You, DiscardAmountInto(count)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[count] != 0 || g.resolution.numberBindings[count] != 9 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 {
		t.Fatal("prevented discard or clone frame is wrong")
	}
	if err := DiscardEntireHand(FromPlayer(BindPlayer("absent")), DiscardAmountInto(count)).Apply(&EffectContext{Game: clone, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if clone.resolution.numberBindings[count] != 0 {
		t.Fatal("absent recipient kept an old count")
	}
}

func TestDiscardEntireHandCountDeclaration(t *testing.T) {
	count := BindNumber("discarded")
	valid := NewSorcery("Valid", "", WithAbility(Spell(Sequence(DiscardEntireHand(You, DiscardAmountInto(count)), DrawFor(You, FromNumber(count))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		body ResolvingEffect
		want string
	}{
		{DiscardEntireHand(You, DiscardAmountInto(nil)), "binding"},
		{DiscardEntireHand(You, DiscardAmountInto(BindNumber(""))), "name"},
		{DiscardEntireHand(You, DiscardAmountInto(count), DiscardAmountInto(count)), "once"},
		{DrawFor(You, FromNumber(count)), "producer"},
		{Sequence(DrawFor(You, FromNumber(count)), DiscardEntireHand(You, DiscardAmountInto(count))), "producer"},
	} {
		card := NewSorcery("Invalid", "", WithAbility(Spell(tc.body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
			t.Fatalf("error %v does not contain %q", err, tc.want)
		}
	}
}

func TestDiscardEntireHandCountStackCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	count := BindNumber("discarded")
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(Dies(This), Sequence(
		DiscardEntireHand(You, DiscardAmountInto(count)),
		GainLifeFor(You, FromNumber(count)),
	))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	a.AddToHand(NewLand("First"))
	a.AddToHand(NewLand("Second"))
	g.DestroyPermanent(g.FindPermanent(source.ID()))
	g.PutTriggersOnStack()
	original := g.StackPeek()
	if original == nil {
		t.Fatal("missing trigger")
	}
	stackCopy := g.CopyStackObjectDirect(original, a.PlayerID(), false)
	if stackCopy == nil {
		t.Fatal("missing copy")
	}
	g.ResolveStackObject(stackCopy)
	if a.Life() != 22 {
		t.Fatal("copy did not use its discard count")
	}
	a.AddToHand(NewLand("Third"))
	g.ResolveStackObject(original)
	if a.Life() != 23 || len(a.Hand()) != 0 {
		t.Fatal("original reused the copy's count")
	}
}
