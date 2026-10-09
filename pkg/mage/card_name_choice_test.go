package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type cardNameChoiceTester struct {
	Player
	name  string
	calls int
}

func (p *cardNameChoiceTester) ChooseMode(names []string, reason string) int {
	p.calls++
	index := slices.Index(names, p.name)
	if len(names) > 0 {
		names[0] = "changed"
	}
	return index
}

func TestCardNameChoiceCatalog(t *testing.T) {
	catalog := OracleCardNameCatalog()
	for _, name := range []string{"Lightning Bolt", "Pick Your Poison", "Fire", "Ice", "Delver of Secrets", "Insectile Aberration", "Brisela, Voice of Nightmares", "Stomp", "_____"} {
		if !catalog.Contains(name) {
			t.Fatalf("missing legal name %q", name)
		}
	}
	for _, name := range []string{"", "Zombie", "Fire // Ice", "Delver of Secrets // Insectile Aberration", "invented name"} {
		if catalog.Contains(name) {
			t.Fatalf("illegal name %q", name)
		}
	}
	names := catalog.Names()
	if !slices.IsSorted(names) || len(names) < 30000 {
		t.Fatal("incomplete or unsorted catalog")
	}
	names[0] = "changed"
	if catalog.Contains("changed") || catalog.Names()[0] == "changed" {
		t.Fatal("catalog exposed mutable storage")
	}
}

func TestCardNameChoiceBindings(t *testing.T) {
	base, b := NewBasePlayer("A"), NewBasePlayer("B")
	a := &cardNameChoiceTester{Player: base, name: "Pick Your Poison"}
	g := NewGame(a, b)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	first, second := BindCardName("first"), BindCardName("second")
	choose := ChooseCardName(You, OracleCardNameCatalog(), first)
	if err := choose.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if name, ok := first.Resolve(ctx); !ok || name != a.name {
		t.Fatal("incorrect saved name")
	}
	a.name = "Ice"
	if err := ChooseCardName(You, OracleCardNameCatalog(), second).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if name, _ := first.Resolve(ctx); name != "Pick Your Poison" {
		t.Fatal("independent choice overwritten")
	}
	clone := g.Clone()
	clone.resolution.Clear()
	if _, ok := first.Resolve(&EffectContext{Game: clone}); ok {
		t.Fatal("clone retained cleared choice")
	}
	if name, _ := first.Resolve(ctx); name != "Pick Your Poison" {
		t.Fatal("clone changed original")
	}
	if err := ChooseCardName(ControllerOf(This), OracleCardNameCatalog(), first).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := first.Resolve(ctx); ok {
		t.Fatal("absent chooser kept stale result")
	}
	a.name = "invalid"
	if err := choose.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if name, ok := first.Resolve(ctx); !ok || !OracleCardNameCatalog().Contains(name) {
		t.Fatal("invalid response supplied illegal name")
	}
	if err := ApplyEffect(g, choose, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if _, ok := first.Resolve(ctx); !ok {
		t.Fatal("nested resolution erased outer choice")
	}
	g.resolution.Clear()
	if _, ok := first.Resolve(ctx); ok {
		t.Fatal("choice escaped frame")
	}
}

func TestCardNameChoiceDeclarations(t *testing.T) {
	for _, effect := range []Effect{
		ChooseCardName(nil, OracleCardNameCatalog(), BindCardName("name")),
		ChooseCardName(You, nil, BindCardName("name")),
		ChooseCardName(You, &CardNameCatalog{}, BindCardName("name")),
		ChooseCardName(You, OracleCardNameCatalog(), nil),
		ChooseCardName(You, OracleCardNameCatalog(), BindCardName("")),
	} {
		card := NewInstant("Invalid", "", Spell(effect))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	valid := NewInstant("Valid", "", Spell(ChooseCardName(You, OracleCardNameCatalog(), BindCardName("name"))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
}

func TestCardNameChoiceWaitingSource(t *testing.T) {
	base, b := NewBasePlayer("A"), NewBasePlayer("B")
	a := &cardNameChoiceTester{Player: base, name: "Ice"}
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("source", ""), a.PlayerID())
	original := g.CaptureDamageSource(p.ID())
	saved := BindCardName("choice")
	body := ChooseCardName(ControllerOfWithLKI(This), OracleCardNameCatalog(), saved)
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(p.ID())
	for range 2 {
		got := g.WaitingDelayedPermanentSources()
		if len(got) != 1 || !got[0].SameObject(original) || a.calls != 0 {
			t.Fatal("query executed choice or lost source")
		}
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(p.ID())
	g.PutOnBattlefield(p.Card, b.PlayerID())
	got := g.WaitingDelayedPermanentSources()
	if len(got) != 1 || !got[0].SameObject(original) {
		t.Fatal("return replaced retained source")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.calls != 1 || len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("delay lost LKI chooser")
	}
}

func TestCardNameChoiceLoopIsolation(t *testing.T) {
	base, b := NewBasePlayer("A"), NewBasePlayer("B")
	a := &cardNameChoiceTester{Player: base, name: "Ice"}
	g := NewGame(a, b)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	saved := BindCardName("name")
	choose := ChooseCardName(You, OracleCardNameCatalog(), saved)
	if err := choose.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	a.name = "Fire"
	g.PutOnBattlefield(NewArtifact("one", ""), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("two", ""), b.PlayerID())
	if err := ForEach(MatchingPermanents(AnyPermanent), BindPermanent("member"), choose).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if name, _ := saved.Resolve(ctx); name != "Ice" || a.calls != 3 {
		t.Fatal("loop choice escaped its frame")
	}
}
