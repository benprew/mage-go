package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLibraryTopRevealVisits(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	first, second, third := NewLand("First"), NewArtifact("Second", ""), NewLand("Third")
	for _, card := range []Card{first, second, third} {
		a.AddToLibrary(card)
	}
	result := BindCard("top")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	reveal := RevealLibraryTopInto(You, result)
	if err := reveal.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	visit := a.libraryVisit(first.ID())
	if visit == uuid.Nil || len(g.PublicReveals()) != 1 || len(a.Library()) != 3 {
		t.Fatal("reveal did not save a present library visit")
	}
	clone := g.Clone()
	a.SetLibrary([]Card{second, first, third})
	if _, ok := FromCard(result).resolveCard(ctx); ok {
		t.Fatal("reordered revealed card kept its object identity")
	}
	if card, ok := FromCard(result).resolveCard(&EffectContext{Game: clone}); !ok || card.ID() != first.ID() {
		t.Fatal("reordering changed clone identity")
	}
	if err := IfRevealedCardMatches(result, RevealedCardHasType(core.TypeLand), GainLifeFor(You, Fixed(2)), nil).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 {
		t.Fatal("reordering erased historical reveal facts")
	}
	if err := reveal.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	secondVisit := a.libraryVisit(second.ID())
	a.SetLibrary([]Card{first, third})
	a.SetLibrary([]Card{second, first, third})
	if a.libraryVisit(second.ID()) == secondVisit {
		t.Fatal("returned card reused its visit")
	}
	if _, ok := FromCard(result).resolveCard(ctx); ok {
		t.Fatal("saved reference followed a returned card")
	}
	if err := reveal.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	a.ShuffleLibrary()
	if _, ok := FromCard(result).resolveCard(ctx); ok {
		t.Fatal("shuffle kept revealed object identity")
	}
	a.SetLibrary(nil)
	if err := reveal.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.resolution.cardBindings[result].revealed != nil {
		t.Fatal("empty library retained reveal result")
	}
}

func TestLibraryTopRevealRemovalAndClone(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewLand("First"), NewLand("Second")
	a.AddToLibrary(first)
	a.AddToLibrary(second)
	secondVisit := a.libraryVisit(second.ID())
	clone := g.Clone()
	a.DrawCard()
	if a.libraryVisit(first.ID()) != uuid.Nil || a.libraryVisit(second.ID()) != secondVisit {
		t.Fatal("draw lost remaining library identity")
	}
	cp := clone.GetPlayer(a.PlayerID()).(*SearchPlayer)
	if cp.libraryVisit(first.ID()) == uuid.Nil {
		t.Fatal("draw changed clone visits")
	}
	result := BindCard("top")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := RevealLibraryTopInto(You, result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := RevealLibraryTopInto(FromPlayer(BindPlayer("absent")), result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(result).resolveCard(ctx); ok {
		t.Fatal("absent player retained a result")
	}
}

func TestLibraryTopRevealDeclarations(t *testing.T) {
	result := BindCard("top")
	for _, effect := range []Effect{RevealLibraryTopInto(nil, result), RevealLibraryTopInto(You, nil), RevealLibraryTopInto(You, BindCard("")), IfRevealedCardMatches(result, RevealedCardHasType(core.TypeLand), GainLifeFor(You, Fixed(1)), nil)} {
		card := NewInstant("Invalid", "", Spell(effect))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	valid := NewInstant("Valid", "", Spell(Sequence(RevealLibraryTopInto(You, result), IfRevealedCardMatches(result, RevealedCardHasType(core.TypeLand), GainLifeFor(You, Fixed(1)), nil))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
}

func TestLibraryTopRevealWaitingAndFrames(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	land := NewLand("Land")
	a.AddToLibrary(land)
	result := BindCard("top")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	reveal := RevealLibraryTopInto(You, result)
	if err := reveal.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, RevealLibraryTopInto(FromPlayer(BindPlayer("absent")), result), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if card, ok := FromCard(result).resolveCard(ctx); !ok || card.ID() != land.ID() {
		t.Fatal("nested resolution cleared the outer result")
	}
	source := NewArtifact("Source", "")
	source.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(source, a.PlayerID())
	body := RevealLibraryTopInto(ControllerOfWithLKI(This), result)
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.DestroyPermanent(p)
	g.PutOnBattlefield(source, g.GetOpponent(a.PlayerID()).PlayerID())
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || len(g.PublicReveals()) != 1 {
			t.Fatal("waiting query executed reveal or lost exact source")
		}
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if len(game.PublicReveals()) != 2 {
			t.Fatal("delayed source LKI lost its library player")
		}
	}
}

func TestLibraryTopRevealMissingIdentity(t *testing.T) {
	base := NewBasePlayer("A")
	a := &revealPlayerWithoutVisits{Player: base}
	g := NewGame(a, NewBasePlayer("B"))
	a.AddToLibrary(NewLand("Land"))
	if err := RevealLibraryTopInto(You, BindCard("top")).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err == nil {
		t.Fatal("missing library identity was accepted")
	}
	if len(g.PublicReveals()) != 0 || len(a.Library()) != 1 {
		t.Fatal("failed reveal changed zones or public history")
	}
}

func TestLibraryTopRevealReorderRegion(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	middle := NewLand("Middle")
	a.AddToLibrary(middle)
	result := BindCard("middle")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := RevealLibraryTopInto(You, result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	first, last, tail := NewLand("First"), NewLand("Last"), NewLand("Tail")
	a.SetLibrary([]Card{first, middle, last, tail})
	if _, ok := FromCard(result).resolveCard(ctx); !ok {
		t.Fatal("insertion changed a retained object")
	}
	tailVisit := a.libraryVisit(tail.ID())
	a.SetLibrary([]Card{last, middle, first, tail})
	if _, ok := FromCard(result).resolveCard(ctx); ok {
		t.Fatal("reordered region retained a revealed object at the same index")
	}
	if a.libraryVisit(tail.ID()) != tailVisit {
		t.Fatal("reordering changed an untouched suffix")
	}
}

func TestLibraryTopRevealContinuousVisibility(t *testing.T) {
	for _, change := range []string{"cover", "depart", "phase", "multiple"} {
		t.Run(change, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			top := NewLand("Top")
			a.AddToLibrary(top)
			source := g.PutOnBattlefield(NewEnchantment("Visibility", "", WithAbility(Static(RevealAllLibraryTops()))), a.PlayerID())
			g.ApplyContinuousEffects()
			result := BindCard("top")
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			if err := RevealLibraryTopInto(You, result).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			switch change {
			case "cover":
				a.SetLibrary([]Card{NewLand("Cover"), top})
			case "depart":
				g.RemoveFromBattlefield(source)
			case "phase":
				g.PhaseOut(source.ID())
			case "multiple":
				g.PutOnBattlefield(NewEnchantment("Second", "", WithAbility(Static(RevealAllLibraryTops()))), a.PlayerID())
				g.RemoveFromBattlefield(source)
			}
			g.ApplyContinuousEffects()
			_, ok := FromCard(result).resolveCard(ctx)
			if ok != (change == "multiple") {
				t.Fatal("continuous top visibility did not control the saved visit")
			}
			if _, ok := FromCard(result).resolveCard(&EffectContext{Game: clone}); !ok {
				t.Fatal("visibility change crossed clones")
			}
		})
	}
}

func TestLibraryTopRevealMutableCallerOrder(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewLand("First"), NewLand("Second")
	a.AddToLibrary(first)
	a.AddToLibrary(second)
	result := BindCard("top")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := RevealLibraryTopInto(You, result).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	library := a.Library()
	library[0], library[1] = library[1], library[0]
	a.SetLibrary(library)
	if _, ok := FromCard(result).resolveCard(ctx); ok {
		t.Fatal("mutable caller hid its library reorder")
	}
}
