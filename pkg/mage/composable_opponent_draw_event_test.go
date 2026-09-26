package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestOpponentDrawEventDamageAndReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	drawer := BindPlayer("drawer")
	source := NewEnchantment("Draw watcher", "", WithAbility(WhenEvent(
		OpponentDrawsCard(drawer), DealDamageFrom(This, FromPlayer(drawer), Fixed(1)))))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	a.AddToLibrary(NewArtifact("A card", ""))
	for range 3 {
		b.AddToLibrary(NewArtifact("B card", ""))
	}
	g.PlayerDrawCard(a)
	for range 2 {
		g.PlayerDrawCard(b)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("draw triggers = %d", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 18 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
	replaced := 0
	g.AddEmptyLibraryDrawReplacement(source.ID(), b.PlayerID(), func(*Game, uuid.UUID) { replaced++ })
	g.PlayerDrawCard(b)
	g.PlayerDrawCard(b)
	g.PutTriggersOnStack()
	if replaced != 1 || g.StackSize() != 1 {
		t.Fatalf("replacement = %d, triggers = %d", replaced, g.StackSize())
	}
	g.ResolveStack()
	if b.Life() != 17 {
		t.Fatalf("opponent life = %d", b.Life())
	}
}

func TestOpponentDrawEventControlAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 4 {
		a.AddToLibrary(NewArtifact("A card", ""))
		b.AddToLibrary(NewArtifact("B card", ""))
	}
	drawer := BindPlayer("drawer")
	source := NewEnchantment("Draw watcher", "", WithAbility(WhenEvent(
		OpponentDrawsCard(drawer), GainLifeFor(FromPlayer(drawer), Fixed(2)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	phase := g.PhaseOut(source.ID())
	g.PlayerDrawCard(b)
	g.PhaseIn(phase)
	g.PlayerDrawCard(b)
	g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	g.PlayerDrawCard(b)
	g.PlayerDrawCard(a)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("draw triggers = %d", g.StackSize())
	}
	var first *StackObject
	for _, obj := range g.StackObjects() {
		if len(obj.Targets) != 0 {
			t.Fatal("event drawer became a target")
		}
		if obj.Controller == a.PlayerID() {
			first = obj
		}
	}
	if first == nil {
		t.Fatal("missing first controller")
	}
	g.CopyStackObjectDirect(first, b.PlayerID(), false)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 22 || game.GetPlayer(b.PlayerID()).Life() != 24 {
			t.Fatalf("life = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestOpponentDrawEventValidation(t *testing.T) {
	drawer := BindPlayer("drawer")
	for _, tc := range []struct {
		pattern EventPattern
		want    string
	}{
		{OpponentDrawsCard(drawer), ""},
		{OpponentDrawsCard(nil), "non-nil"},
		{OpponentDrawsCard(BindPlayer("")), "name"},
		{OpponentDrawsCard(BindPlayer("different")), "producer"},
	} {
		card := NewEnchantment("Watcher", "", WithAbility(WhenEvent(tc.pattern, GainLifeFor(FromPlayer(drawer), Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("want %q: %v", tc.want, err)
		}
	}
	card := NewEnchantment("Scope", "", WithAbility(WhenEvent(OpponentDrawsCard(drawer), GainLifeFor(FromPlayer(drawer), Fixed(1)))), WithAbility(At(YourUpkeep, GainLifeFor(FromPlayer(drawer), Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("scope: %v", err)
	}
}
