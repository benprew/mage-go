package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func anteEffectTestGame(t *testing.T) (*Game, *BasePlayer, *BasePlayer) {
	t.Helper()
	a := NewBasePlayer("A")
	b := NewBasePlayer("B")
	g, err := NewGameWithAnte(a, b, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return g, a, b
}

func TestTargetCardYouOwnInAnteUsesSharedZoneAndCurrentOwner(t *testing.T) {
	g, a, b := anteEffectTestGame(t)
	owned := ownedTestCard(b.PlayerID(), "owned")
	notOwned := ownedTestCard(b.PlayerID(), "not owned")
	b.SetAnte([]Card{owned, notOwned})
	if err := g.ChangeOwner(owned.ID(), a.PlayerID()); err != nil {
		t.Fatal(err)
	}

	target := TargetCardYouOwnInAnte()
	possible := target.Possible(a.PlayerID(), ownedTestCard(a.PlayerID(), "source"), g)
	if !slices.Contains(possible, owned.ID()) || slices.Contains(possible, notOwned.ID()) {
		t.Fatalf("possible targets = %v, want only currently owned card", possible)
	}
	if err := target.Choose(a.PlayerID(), nil, g, []uuid.UUID{notOwned.ID()}); err == nil {
		t.Fatal("target accepted an ante card the controller does not own")
	}
}

func TestAnteLibraryTopTargetsControllerOrEachPlayer(t *testing.T) {
	t.Run("controller", func(t *testing.T) {
		g, a, b := anteEffectTestGame(t)
		aTop := ownedTestCard(a.PlayerID(), "A top")
		bTop := ownedTestCard(b.PlayerID(), "B top")
		a.SetLibrary([]Card{aTop})
		b.SetLibrary([]Card{bTop})

		if err := AnteLibraryTop().Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if g.CardZone(aTop.ID()) != core.ZoneAnte || g.CardZone(bTop.ID()) != core.ZoneLibrary {
			t.Fatal("controller selection anted the wrong library top")
		}
	})

	t.Run("each player", func(t *testing.T) {
		g, a, b := anteEffectTestGame(t)
		aTop := ownedTestCard(a.PlayerID(), "A top")
		bTop := ownedTestCard(b.PlayerID(), "B top")
		a.SetLibrary([]Card{aTop})
		b.SetLibrary([]Card{bTop})

		if err := AnteLibraryTop().Targeting(SelectEachPlayer()).Apply(&EffectContext{Game: g}); err != nil {
			t.Fatal(err)
		}
		if g.CardZone(aTop.ID()) != core.ZoneAnte || g.CardZone(bTop.ID()) != core.ZoneAnte {
			t.Fatal("each-player selection did not ante both library tops")
		}
	})

	t.Run("one player unable", func(t *testing.T) {
		g, a, b := anteEffectTestGame(t)
		aTop := ownedTestCard(b.PlayerID(), "opponent-owned A top")
		bTop := ownedTestCard(b.PlayerID(), "B top")
		a.SetLibrary([]Card{aTop})
		b.SetLibrary([]Card{bTop})

		if err := AnteLibraryTop().Targeting(SelectEachPlayer()).Apply(&EffectContext{Game: g}); err != nil {
			t.Fatal(err)
		}
		if g.CardZone(aTop.ID()) != core.ZoneLibrary || g.CardZone(bTop.ID()) != core.ZoneAnte {
			t.Fatal("one failed ante prevented another selected player from anteing")
		}
	})
}

func TestExchangeTargetAnteCardWithLibraryTop(t *testing.T) {
	g, a, b := anteEffectTestGame(t)
	anteCard := ownedTestCard(b.PlayerID(), "ante card")
	b.SetAnte([]Card{anteCard})
	if err := g.ChangeOwner(anteCard.ID(), a.PlayerID()); err != nil {
		t.Fatal(err)
	}
	top := ownedTestCard(a.PlayerID(), "library top")
	next := ownedTestCard(a.PlayerID(), "next")
	a.SetLibrary([]Card{top, next})

	effect := ExchangeTargetAnteCardWithLibraryTop()
	if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID(), Targets: []uuid.UUID{anteCard.ID()}}); err != nil {
		t.Fatal(err)
	}
	if len(a.Library()) != 2 || a.Library()[0].ID() != anteCard.ID() || a.Library()[1].ID() != next.ID() {
		t.Fatal("target ante card was not put on top of the library")
	}
	if g.CardZone(top.ID()) != core.ZoneAnte || g.CardZone(anteCard.ID()) != core.ZoneLibrary {
		t.Fatal("exchange did not preserve both card identities and destinations")
	}
}

func TestExchangeTargetAnteCardRequiresBothSides(t *testing.T) {
	g, a, b := anteEffectTestGame(t)
	anteCard := ownedTestCard(a.PlayerID(), "ante card")
	a.SetAnte([]Card{anteCard})
	opponentOwnedTop := ownedTestCard(b.PlayerID(), "opponent-owned top")
	a.SetLibrary([]Card{opponentOwnedTop})

	if err := ExchangeTargetAnteCardWithLibraryTop().Apply(&EffectContext{Game: g, Controller: a.PlayerID(), Targets: []uuid.UUID{anteCard.ID()}}); err != nil {
		t.Fatal(err)
	}
	if g.CardZone(anteCard.ID()) != core.ZoneAnte || g.CardZone(opponentOwnedTop.ID()) != core.ZoneLibrary {
		t.Fatal("partial exchange occurred when the controller could not ante the library top")
	}
}
