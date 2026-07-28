package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type acceptingMayPlayer struct{ *BasePlayer }

func (p *acceptingMayPlayer) ChooseMayAbility(string) bool { return true }

func TestTargetXCreaturesRejectsWrongDeclaredCount(t *testing.T) {
	g := newPriorityTestGame()
	player := g.players[0]
	spell := NewSorcery("Test X Tap", "{X}",
		NewMultiTargetSpell([]Target{TargetXCreatures()}, FuncEffect(
			"tap X target creatures", EffectProperties{},
			func(*Game, uuid.UUID, uuid.UUID, []uuid.UUID) error { return nil },
		)),
	)
	spell.SetOwner(player.PlayerID())
	player.AddToHand(spell)
	player.ManaPool().Add(Colorless, 2)

	first := NewCreature("First", "{1}", 1, 1)
	first.SetOwner(player.PlayerID())
	firstPerm := g.PutOnBattlefield(first, player.PlayerID())
	second := NewCreature("Second", "{1}", 1, 1)
	second.SetOwner(g.players[1].PlayerID())
	g.PutOnBattlefield(second, g.players[1].PlayerID())

	if err := g.CastSpellByName(player.PlayerID(), spell.Name(), []uuid.UUID{firstPerm.ID()}, 2); err == nil {
		t.Fatal("expected casting with one target for X=2 to fail")
	}
	if len(player.Hand()) != 1 {
		t.Fatal("illegal cast should leave the spell in hand")
	}
	if got := player.ManaPool().Count(Colorless); got != 2 {
		t.Fatalf("illegal cast spent mana: got %d, want 2", got)
	}
}

func TestDiscardToLibraryReplacementOnlyAppliesToEffects(t *testing.T) {
	player := &acceptingMayPlayer{NewBasePlayer("A")}
	opponent := NewBasePlayer("B")
	g := NewGame(player, opponent)
	library := NewArtifact("Test Library", "{1}", WithStaticAbility(DiscardToLibraryReplacement()))
	library.SetOwner(player.PlayerID())
	g.PutOnBattlefield(library, player.PlayerID())

	effectDiscard := NewSorcery("Effect Discard", "{1}", NewSpellAbility())
	effectDiscard.SetOwner(player.PlayerID())
	player.AddToHand(effectDiscard)
	if _, ok := g.PlayerDiscardByEffect(player, effectDiscard.ID(), library.ID()); !ok {
		t.Fatal("effect-caused discard failed")
	}
	if len(player.Library()) != 1 || player.Library()[0].ID() != effectDiscard.ID() {
		t.Fatal("effect-caused discard was not put on top of the library")
	}

	costDiscard := NewSorcery("Cost Discard", "{1}", NewSpellAbility())
	costDiscard.SetOwner(player.PlayerID())
	player.AddToHand(costDiscard)
	if _, ok := g.PlayerDiscard(player, costDiscard.ID()); !ok {
		t.Fatal("cost discard failed")
	}
	if len(player.Graveyard()) != 1 || player.Graveyard()[0].ID() != costDiscard.ID() {
		t.Fatal("non-effect discard should go to the graveyard")
	}
}

func TestTryPayManaUsesFloatingMana(t *testing.T) {
	g := newPriorityTestGame()
	player := g.players[0]
	player.ManaPool().Add(Blue, 2)
	if !g.TryPayMana(player.PlayerID(), "{2}") {
		t.Fatal("expected floating mana to pay a resolving-effect cost")
	}
	if got := player.ManaPool().TotalMana(); got != 0 {
		t.Fatalf("mana pool has %d mana after payment, want 0", got)
	}
}

func TestMayPayManaUsesAllAvailableMana(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Game, *acceptingMayPlayer)
	}{
		{
			name: "floating mana",
			setup: func(_ *Game, player *acceptingMayPlayer) {
				player.ManaPool().Add(Blue, 1)
			},
		},
		{
			name: "nonland mana source",
			setup: func(g *Game, player *acceptingMayPlayer) {
				source := NewArtifact("Test Mox", "{0}", WithManaAbility(Blue))
				source.SetOwner(player.PlayerID())
				g.PutOnBattlefield(source, player.PlayerID())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := &acceptingMayPlayer{NewBasePlayer("A")}
			opponent := NewBasePlayer("B")
			g := NewGame(player, opponent)
			tt.setup(g, player)

			resolved := false
			effect := MayPayMana("{U}", "resolve the test effect", FuncEffect(
				"resolve the test effect",
				EffectProperties{},
				func(*Game, uuid.UUID, uuid.UUID, []uuid.UUID) error {
					resolved = true
					return nil
				},
			))
			if err := ApplyEffect(g, effect, uuid.New(), player.PlayerID(), nil); err != nil {
				t.Fatalf("ApplyEffect: %v", err)
			}
			if !resolved {
				t.Fatal("effect did not resolve after the player chose and could pay")
			}
		})
	}
}
