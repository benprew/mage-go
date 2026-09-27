package mage

import (
	"strings"
	"testing"
)

func TestPlayerDiscardsCardBothPlayersAndCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	discarder := BindPlayer("discarder")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PlayerDiscardsCard(discarder), GainLifeFor(FromPlayer(discarder), Fixed(2)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	for _, player := range []Player{a, b} {
		card := NewArtifact("Discard", "")
		player.AddToHand(card)
		g.PlayerDiscard(player, card.ID())
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("discard triggers = %d", g.StackSize())
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 22 || game.GetPlayer(b.PlayerID()).Life() != 22 {
			t.Fatalf("life = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestPlayerDiscardsCardDeclaration(t *testing.T) {
	player := BindPlayer("discarder")
	if info := InspectDefinition(PlayerDiscardsCard(player)); info.Role != "event-pattern" || info.Text != "a player discards a card" {
		t.Fatalf("inspection = %+v", info)
	}
	for _, tc := range []struct {
		pattern EventPattern
		want    string
	}{
		{PlayerDiscardsCard(player), ""},
		{PlayerDiscardsCard(nil), "non-nil"},
		{PlayerDiscardsCard(BindPlayer("")), "name"},
		{PlayerDiscardsCard(BindPlayer("other")), "producer"},
	} {
		card := NewEnchantment("Watcher", "", WithAbility(WhenEvent(tc.pattern, GainLifeFor(FromPlayer(player), Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("want %q: %v", tc.want, err)
		}
	}
	scope := NewEnchantment("Scope", "",
		WithAbility(WhenEvent(PlayerDiscardsCard(player), GainLifeFor(FromPlayer(player), Fixed(1)))),
		WithAbility(At(YourUpkeep, GainLifeFor(FromPlayer(player), Fixed(1)))))
	if err := ValidateCardDefinition(scope.Name(), scope); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("scope: %v", err)
	}
}

func TestPlayerDiscardsCardReplacementPhasingAndDeparture(t *testing.T) {
	a, b := NewBasePlayer("A"), &acceptingMayPlayer{NewBasePlayer("B")}
	g := NewGame(a, b)
	discarder := BindPlayer("discarder")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		PlayerDiscardsCard(discarder), GainLifeFor(FromPlayer(discarder), Fixed(1)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	card := NewArtifact("Ignored", "")
	a.AddToHand(card)
	phase := g.PhaseOut(source.ID())
	g.PlayerDiscard(a, card.ID())
	g.PhaseIn(phase)
	card = NewArtifact("Replaced", "")
	b.AddToHand(card)
	g.PutOnBattlefield(NewArtifact("Replacement", "", WithStaticAbility(DiscardToLibraryReplacement())), b.PlayerID())
	if _, ok := g.PlayerDiscardByEffect(b, card.ID(), source.ID()); !ok {
		t.Fatal("replacement prevented the discard")
	}
	if _, ok := g.PlayerDiscardByEffect(b, card.ID(), source.ID()); ok {
		t.Fatal("one card was discarded twice")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("discard triggers = %d", g.StackSize())
	}
	stackObject := g.StackObjects()[0]
	if len(stackObject.Targets) != 0 {
		t.Fatal("saved player became a target")
	}
	g.CopyStackObjectDirect(stackObject, a.PlayerID(), false)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.ResolveStack()
	if b.Life() != 22 || a.Life() != 20 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
	found := false
	for _, candidate := range b.Library() {
		found = found || candidate.ID() == card.ID()
	}
	if !found || len(b.Graveyard()) != 0 {
		t.Fatal("discard replacement did not move the card to the library")
	}
}
