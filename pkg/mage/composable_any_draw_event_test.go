package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPlayerDrawsCardEitherPlayerAndCompletedDraws(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	drawer := BindPlayer("drawer")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDrawsCard(drawer), GainLifeFor(FromPlayer(drawer), Fixed(2)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	a.AddToLibrary(NewArtifact("A card", ""))
	for range 2 {
		b.AddToLibrary(NewArtifact("B card", ""))
	}
	g.PlayerDrawCard(a)
	g.PlayerDrawCard(b)
	g.PlayerDrawCard(b)
	g.PlayerDrawCard(a)
	if g.StackSize() != 0 {
		t.Fatal("triggers entered stack before placement")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("stack = %d", g.StackSize())
	}
	for _, obj := range g.StackObjects() {
		if len(obj.Targets) != 0 {
			t.Fatal("captured drawer became a target")
		}
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 24 {
		t.Fatalf("life = %d, %d", a.Life(), b.Life())
	}
	replaced := 0
	g.AddEmptyLibraryDrawReplacement(source.ID(), a.PlayerID(), func(*Game, uuid.UUID) { replaced++ })
	g.PlayerDrawCard(a)
	g.PutTriggersOnStack()
	if replaced != 1 || g.StackSize() != 0 {
		t.Fatalf("replacement = %d, stack = %d", replaced, g.StackSize())
	}
}

func TestPlayerDrawsCardSourceLifecycleAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 3 {
		a.AddToLibrary(NewArtifact("A card", ""))
		b.AddToLibrary(NewArtifact("B card", ""))
	}
	drawer := BindPlayer("drawer")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(PlayerDrawsCard(drawer), GainLifeFor(FromPlayer(drawer), Fixed(1)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	phased := g.PhaseOut(source.ID())
	g.PlayerDrawCard(a)
	g.PhaseIn(phased)
	g.PlayerDrawCard(a)
	g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	g.PlayerDrawCard(b)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d", g.StackSize())
	}
	for _, obj := range g.StackObjects() {
		if obj.Controller == a.PlayerID() {
			g.CopyStackObjectDirect(obj, b.PlayerID(), false)
			break
		}
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 22 || game.GetPlayer(b.PlayerID()).Life() != 21 {
			t.Fatalf("life = %d, %d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life())
		}
	}
}

func TestPlayerDrawsCardDeclaration(t *testing.T) {
	drawer := BindPlayer("drawer")
	if info := InspectDefinition(PlayerDrawsCard(drawer)); info.Role != "event-pattern" || info.Text != "a player draws a card" || len(info.Children) != 1 {
		t.Fatalf("inspection = %+v", info)
	}
	for _, tc := range []struct {
		pattern EventPattern
		want    string
	}{
		{PlayerDrawsCard(drawer), ""},
		{PlayerDrawsCard(nil), "non-nil"},
		{PlayerDrawsCard(BindPlayer("")), "name"},
		{PlayerDrawsCard(BindPlayer("other")), "producer"},
	} {
		card := NewEnchantment("Watcher", "", WithAbility(WhenEvent(tc.pattern, GainLifeFor(FromPlayer(drawer), Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("want %q: %v", tc.want, err)
		}
	}
	card := NewEnchantment("Scope", "", WithAbility(WhenEvent(PlayerDrawsCard(drawer), GainLifeFor(FromPlayer(drawer), Fixed(1)))), WithAbility(At(YourUpkeep, GainLifeFor(FromPlayer(drawer), Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("scope: %v", err)
	}
}
