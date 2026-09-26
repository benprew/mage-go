package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentDiesTypeAndFormerController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	former := BindPlayer("former controller")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		PermanentDies().OfType(core.TypeLand).WithController(former),
		GainLifeFor(FromPlayer(former), Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	first := NewLand("First")
	first.SetOwner(a.PlayerID())
	second := NewLand("Second")
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	creature := NewCreature("Creature", "", 1, 1)
	creature.SetOwner(a.PlayerID())
	g.PutOnBattlefield(creature, a.PlayerID())
	g.DestroyPermanent(g.FindPermanent(creature.ID()))
	g.DestroyPermanents([]*Permanent{g.FindPermanent(first.ID()), g.FindPermanent(second.ID())})
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 2 {
		t.Fatalf("land triggers = %d, want 2", got)
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatalf("life = (%d, %d), want (21, 21)", a.Life(), b.Life())
	}
}

func TestPermanentDiesTypeExcludesOtherEvents(t *testing.T) {
	for _, tc := range []struct {
		name    string
		prepare func(*Game, *Permanent)
		want    int
	}{
		{"artifact death", nil, 1},
		{"phased source", func(g *Game, source *Permanent) { g.PhaseOut(source.ID()) }, 0},
		{"absent source", func(g *Game, source *Permanent) { g.RemoveFromBattlefield(source) }, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				PermanentDies().OfType(core.TypeArtifact), GainLifeFor(You, Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			g.PutOnBattlefield(watcher, a.PlayerID())
			victim := NewArtifact("Victim", "")
			victim.SetOwner(b.PlayerID())
			g.PutOnBattlefield(victim, b.PlayerID())
			if tc.prepare != nil {
				tc.prepare(g, g.FindPermanent(watcher.ID()))
			}
			g.DestroyPermanent(g.FindPermanent(victim.ID()))
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.want {
				t.Fatalf("triggers = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestPermanentDiesSimultaneousSourceAndReplacement(t *testing.T) {
	for _, replaced := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		former := BindPlayer("former")
		watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
			PermanentDies().OfType(core.TypeLand).WithController(former),
			GainLifeFor(FromPlayer(former), Fixed(1)))))
		watcher.SetOwner(a.PlayerID())
		land := NewLand("Land")
		land.SetOwner(b.PlayerID())
		g.PutOnBattlefield(watcher, a.PlayerID())
		g.PutOnBattlefield(land, b.PlayerID())
		if replaced {
			g.AddExileIfWouldGoToGraveyardThisTurn(land.ID(), watcher.ID())
		}
		g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(land.ID())})
		g.PutTriggersOnStack()
		want := 1
		if replaced {
			want = 0
		}
		if got := g.StackSize(); got != want {
			t.Fatalf("replaced=%v: triggers=%d, want %d", replaced, got, want)
		}
		if want == 1 {
			if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
		}
		g.ResolveStack()
		if got := b.Life(); got != 20+2*want {
			t.Fatalf("replaced=%v: former controller life=%d", replaced, got)
		}
	}
}

func TestPermanentDiesAllTypesAndInspection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		PermanentDies(), GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(watcher.Abilities()[0]).Children[0].Text; got != "a permanent dies" {
		t.Fatalf("inspection=%q", got)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Land")
	land.SetOwner(b.PlayerID())
	artifact := NewArtifact("Artifact", "")
	artifact.SetOwner(b.PlayerID())
	g.PutOnBattlefield(land, b.PlayerID())
	g.PutOnBattlefield(artifact, b.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(land.ID()), g.FindPermanent(artifact.ID())})
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 2 {
		t.Fatalf("triggers=%d, want 2", got)
	}
}

func TestPermanentDiesTypeRegistration(t *testing.T) {
	for _, typ := range []core.CardType{core.TypeInstant, core.TypeSorcery, core.CardType(-1)} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(
			PermanentDies().OfType(typ), GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("type %v accepted", typ)
		}
	}
	for _, pattern := range []EventPattern{
		PermanentDies().OfType(core.TypeLand).OfType(core.TypeArtifact),
		CreatureDies().OfType(core.TypeLand),
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("repeated or creature-only type filter accepted")
		}
	}
}
