package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatureYouSacrificeEventFiltering(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(CreatureYouSacrifice(), GainLifeFor(You, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	own := NewCreature("Own", "", 1, 1)
	g.PutOnBattlefield(own, a.PlayerID())
	other := NewCreature("Other", "", 1, 1)
	g.PutOnBattlefield(other, b.PlayerID())
	land := NewLand("Land")
	g.PutOnBattlefield(land, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(other.ID()))
	g.DoSacrifice(g.FindPermanent(land.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("opponent creature or own land matched")
	}
	g.DoSacrifice(g.FindPermanent(own.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 {
		t.Fatal("own creature must create one untargeted trigger")
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("life = %d, want 21", a.Life())
	}
	g.DoSacrifice(g.FindPermanent(source.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("source sacrifice did not trigger from LKI")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("life = %d, want 22", a.Life())
	}
}

func TestCreatureYouSacrificeEventIdentityAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(CreatureYouSacrifice(), GainLifeFor(You, Fixed(1)))))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	first := NewCreature("First", "", 1, 1)
	g.PutOnBattlefield(first, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(first.ID()))
	clone := g.Clone()
	for _, game := range []*Game{clone, g} {
		game.PutTriggersOnStack()
		if game.StackSize() != 1 {
			t.Fatal("pending trigger lost in clone")
		}
		if game.CopyStackObjectDirect(game.StackPeek(), b.PlayerID(), true) == nil {
			t.Fatal("trigger copy failed")
		}
		game.RemoveFromBattlefield(game.FindPermanent(source.ID()))
		game.PutOnBattlefield(source, b.PlayerID())
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 21 || game.GetPlayer(b.PlayerID()).Life() != 21 {
			t.Fatal("return changed pending trigger or copied controller")
		}
		if game == clone && a.Life() != 20 {
			t.Fatal("clone changed original game")
		}
	}
}

func TestCreatureYouSacrificeEventPhasedSourceAndInspection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(CreatureYouSacrifice(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.PhaseOut(source.ID())
	creature := NewCreature("Creature", "", 1, 1)
	g.PutOnBattlefield(creature, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(creature.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source observed sacrifice")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	creature = NewCreature("Second", "", 1, 1)
	g.PutOnBattlefield(creature, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(creature.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("phased-in source missed sacrifice")
	}
	g.ResolveStack()
	g.ExilePermanent(g.FindPermanent(source.ID()))
	creature = NewCreature("Third", "", 1, 1)
	g.PutOnBattlefield(creature, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(creature.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("absent source observed sacrifice")
	}
}

func TestCreatureYouSacrificeEventControllerAndToken(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "", 1, 1, WithAbility(WhenEvent(CreatureYouSacrifice(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(source, a.PlayerID())
	stolen := NewCreature("Stolen", "", 1, 1)
	stolen.SetOwner(b.PlayerID())
	g.PutOnBattlefield(stolen, b.PlayerID())
	g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: stolen.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
	g.DoSacrifice(g.FindPermanent(stolen.ID()))
	token := NewToken("Token", 1, 1, []CardType{TypeCreature}, nil)
	g.PutOnBattlefield(token, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(token.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("trigger count = %d, want 2", g.StackSize())
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("life = %d, want 22", a.Life())
	}
}
