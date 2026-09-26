package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestOpponentDiscardsCardPerCardAndReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	discarder := BindPlayer("discarder")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OpponentDiscardsCard(discarder), LoseLifeFor(FromPlayer(discarder), Fixed(2)))))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	own := NewArtifact("Own", "")
	a.AddToHand(own)
	g.PlayerDiscard(a, own.ID())
	for range 2 {
		card := NewArtifact("Other", "")
		b.AddToHand(card)
		g.PlayerDiscard(b, card.ID())
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("discard triggers = %d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 16 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
	card := NewArtifact("Replaced", "")
	b.AddToHand(card)
	g.PutOnBattlefield(NewArtifact("Replacement", "", WithStaticAbility(DiscardToLibraryReplacement())), b.PlayerID())
	g.PlayerDiscardByEffect(b, card.ID(), source.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("replaced discard triggers = %d", g.StackSize())
	}
	g.ResolveStack()
	if b.Life() != 14 {
		t.Fatalf("opponent life = %d", b.Life())
	}
	if _, ok := g.PlayerDiscardByEffect(b, card.ID(), source.ID()); ok {
		t.Fatal("discarded card was removed twice")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("failed discard caused a trigger")
	}
}

func TestOpponentDiscardsCardIdentityAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	discarder := BindPlayer("discarder")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OpponentDiscardsCard(discarder), GainLifeFor(FromPlayer(discarder), Fixed(2)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	card := NewArtifact("Discard", "")
	b.AddToHand(card)
	phase := g.PhaseOut(source.ID())
	g.PlayerDiscard(b, card.ID())
	g.PhaseIn(phase)
	card = NewArtifact("Discard", "")
	b.AddToHand(card)
	g.PlayerDiscard(b, card.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("discard triggers = %d", g.StackSize())
	}
	obj := g.StackObjects()[0]
	if len(obj.Targets) != 0 {
		t.Fatal("discarder became a target")
	}
	g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(b.PlayerID()).Life() != 24 {
			t.Fatalf("opponent life = %d", game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestOpponentDiscardsCardDeclaration(t *testing.T) {
	discarder := BindPlayer("discarder")
	if info := InspectDefinition(OpponentDiscardsCard(discarder)); info.Role != "event-pattern" || info.Text != "an opponent discards a card" {
		t.Fatalf("inspection = %+v", info)
	}
	for _, tc := range []struct {
		pattern EventPattern
		want    string
	}{
		{OpponentDiscardsCard(discarder), ""},
		{OpponentDiscardsCard(nil), "non-nil"},
		{OpponentDiscardsCard(BindPlayer("")), "name"},
		{OpponentDiscardsCard(BindPlayer("other")), "producer"},
	} {
		card := NewEnchantment("Watcher", "", WithAbility(WhenEvent(tc.pattern, GainLifeFor(FromPlayer(discarder), Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("want %q: %v", tc.want, err)
		}
	}
	card := NewEnchantment("Scope", "", WithAbility(WhenEvent(OpponentDiscardsCard(discarder), GainLifeFor(FromPlayer(discarder), Fixed(1)))), WithAbility(At(YourUpkeep, GainLifeFor(FromPlayer(discarder), Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("scope: %v", err)
	}
}

func TestOpponentDiscardsCardEventTimeControl(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	discarder := BindPlayer("discarder")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		OpponentDiscardsCard(discarder), GainLifeFor(FromPlayer(discarder), Fixed(1)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	for _, p := range []Player{a, b} {
		card := NewArtifact("Discard", "")
		p.AddToHand(card)
		g.PlayerDiscard(p, card.ID())
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.StackObjects()[0].Controller != b.PlayerID() {
		t.Fatal("discard matched the wrong controller")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
}
