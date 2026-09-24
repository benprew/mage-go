package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestControllerDrawEventActualDraws(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("drawer")
	source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerDrawsCard(player), GainLifeFor(FromPlayer(player), Fixed(1)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	for range 2 {
		a.AddToLibrary(NewArtifact("Card", ""))
	}
	b.AddToLibrary(NewArtifact("Other", ""))
	g.PlayerDrawCard(b)
	if err := ApplyEffect(g, DrawFor(You, Fixed(3)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 || !a.DrewFromEmpty() {
		t.Fatalf("stack = %d, empty = %v", g.StackSize(), a.DrewFromEmpty())
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatal("incorrect draw recipients")
	}
	replacements := 0
	g.AddEmptyLibraryDrawReplacement(source.ID(), a.PlayerID(), func(_ *Game, _ uuid.UUID) { replacements++ })
	g.PlayerDrawCard(a)
	g.PutTriggersOnStack()
	if replacements != 1 || g.StackSize() != 0 {
		t.Fatal("replaced draw triggered")
	}
}

func TestControllerDrawEventLifecycle(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		for _, modal := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			for range 3 {
				a.AddToLibrary(NewArtifact("A", ""))
				b.AddToLibrary(NewArtifact("B", ""))
			}
			player := BindPlayer("drawer")
			body := Sequence(GainLifeFor(FromPlayer(player), Fixed(2)), GainLifeFor(You, Fixed(1)))
			if delayed {
				body = AtNext(EachEndStep, body)
			}
			pattern := ControllerDrawsCard(player)
			trigger := WhenEvent(pattern, body)
			if modal {
				trigger = WhenEventModes(pattern, ModeOf("Gain", body), ModeOf("Other", GainLifeFor(You, Fixed(9))))
			}
			source := NewEnchantment("Watcher", "", WithAbility(trigger))
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.PlayerDrawCard(a)
			g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			g.PlayerDrawCard(a)
			g.PlayerDrawCard(b)
			pendingClone := g.Clone()
			for _, game := range []*Game{g, pendingClone} {
				game.PutTriggersOnStack()
				if game.StackSize() != 2 {
					t.Fatalf("stack = %d", game.StackSize())
				}
				for _, obj := range game.StackObjects() {
					if len(obj.Targets) != 0 {
						t.Fatal("event player became a target")
					}
					if obj.Controller == a.PlayerID() {
						game.CopyStackObjectDirect(obj, b.PlayerID(), false)
						break
					}
				}
				game.RemoveFromBattlefield(game.FindPermanent(source.ID()))
				game.PutOnBattlefield(source, a.PlayerID())
				cloned := game.Clone()
				for _, resolved := range []*Game{game, cloned} {
					resolved.ResolveStack()
					if delayed {
						resolved.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
						resolved.PutTriggersOnStack()
						resolved.ResolveStack()
					}
					if resolved.GetPlayer(a.PlayerID()).Life() != 25 || resolved.GetPlayer(b.PlayerID()).Life() != 24 {
						t.Fatalf("life = %d, %d", resolved.GetPlayer(a.PlayerID()).Life(), resolved.GetPlayer(b.PlayerID()).Life())
					}
				}
			}
		}
	}
}

func TestControllerDrawEventSourceIdentity(t *testing.T) {
	for _, returned := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		for range 3 {
			a.AddToLibrary(NewArtifact("Card", ""))
		}
		source := NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerDrawsCard(BindPlayer("drawer")), Destroy(This))))
		source.SetOwner(a.PlayerID())
		g.PutOnBattlefield(source, a.PlayerID())
		phased := g.PhaseOut(source.ID())
		g.PlayerDrawCard(a)
		g.PutTriggersOnStack()
		if g.StackSize() != 0 {
			t.Fatal("phased source triggered")
		}
		g.PhaseIn(phased)
		g.PlayerDrawCard(a)
		if returned {
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
		} else {
			phased = g.PhaseOut(source.ID())
			g.PhaseIn(phased)
		}
		g.PutTriggersOnStack()
		if g.StackSize() != 1 {
			t.Fatal("pending trigger lost")
		}
		g.ResolveStack()
		if (g.FindPermanent(source.ID()) != nil) != returned {
			t.Fatal("incorrect source incarnation")
		}
	}
}

func TestControllerDrawEventValidation(t *testing.T) {
	p := BindPlayer("drawer")
	for _, tt := range []struct {
		pattern EventPattern
		want    string
	}{
		{ControllerDrawsCard(p), ""}, {ControllerDrawsCard(nil), "non-nil"},
		{ControllerDrawsCard(BindPlayer("")), "name"}, {ControllerDrawsCard(BindPlayer("other")), "producer"},
	} {
		card := NewEnchantment("Watcher", "", WithAbility(WhenEvent(tt.pattern, GainLifeFor(FromPlayer(p), Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tt.want == "" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("want %q: %v", tt.want, err)
		}
	}
	card := NewEnchantment("Scope", "", WithAbility(WhenEvent(ControllerDrawsCard(p), GainLifeFor(FromPlayer(p), Fixed(1)))), WithAbility(At(YourUpkeep, GainLifeFor(FromPlayer(p), Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("scope: %v", err)
	}
}
