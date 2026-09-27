package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentDiesNotSacrificedOutcomeAndController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	former := BindPlayer("former controller")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		PermanentDies().OfType(core.TypeArtifact).YouControlled().NotSacrificed().WithController(former),
		GainLifeFor(FromPlayer(former), Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(watcher.Abilities()[0]).Children[0].Text; !strings.Contains(got, "not sacrificed") {
		t.Fatalf("inspection = %q", got)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := func(name string) *Permanent {
		card := NewArtifact(name, "")
		card.SetOwner(a.PlayerID())
		return g.PutOnBattlefield(card, a.PlayerID())
	}
	g.DoSacrifice(artifact("Sacrificed"))
	ownedByB := NewArtifact("Controlled by A", "")
	ownedByB.SetOwner(b.PlayerID())
	g.DestroyPermanent(g.PutOnBattlefield(ownedByB, a.PlayerID()))
	opponent := NewArtifact("Controlled by B", "")
	opponent.SetOwner(a.PlayerID())
	g.DestroyPermanent(g.PutOnBattlefield(opponent, b.PlayerID()))
	g.DestroyPermanent(artifact("Destroyed"))
	replaced := artifact("Exiled")
	g.AddExileIfWouldGoToGraveyardThisTurn(replaced.ID(), watcher.ID())
	g.DestroyPermanent(replaced)
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 2 {
		t.Fatalf("triggers = %d, want 2", got)
	}
	if len(g.StackPeek().Targets) != 0 {
		t.Fatal("event controller became a target")
	}
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 20 {
		t.Fatalf("life = (%d, %d), want (23, 20)", a.Life(), b.Life())
	}
}

func TestPermanentDiesNotSacrificedSourcePresence(t *testing.T) {
	for _, tc := range []struct {
		name string
		prep func(*Game, *Permanent)
		want int
	}{
		{"present", nil, 1},
		{"phased", func(g *Game, p *Permanent) { g.PhaseOut(p.ID()) }, 0},
		{"absent", func(g *Game, p *Permanent) { g.RemoveFromBattlefield(p) }, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
				PermanentDies().OfType(core.TypeArtifact).NotSacrificed(), GainLifeFor(You, Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			g.PutOnBattlefield(watcher, a.PlayerID())
			victim := NewArtifact("Victim", "")
			victim.SetOwner(b.PlayerID())
			g.PutOnBattlefield(victim, b.PlayerID())
			if tc.prep != nil {
				tc.prep(g, g.FindPermanent(watcher.ID()))
			}
			g.DestroyPermanent(g.FindPermanent(victim.ID()))
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.want {
				t.Fatalf("triggers = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestPermanentDiesNotSacrificedSimultaneousSourceDeath(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		PermanentDies().OfType(core.TypeArtifact).YouControlled().NotSacrificed(),
		GainLifeFor(You, Fixed(1)))))
	watcher.SetOwner(a.PlayerID())
	victim := NewArtifact("Victim", "")
	victim.SetOwner(a.PlayerID())
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutOnBattlefield(victim, a.PlayerID())
	g.DestroyPermanents([]*Permanent{g.FindPermanent(watcher.ID()), g.FindPermanent(victim.ID())})
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 2 {
		t.Fatalf("triggers = %d, want 2", got)
	}
	g.ResolveStack()
	if got := a.Life(); got != 22 {
		t.Fatalf("life = %d, want 22", got)
	}
}

func TestPermanentDiesNotSacrificedDeclaration(t *testing.T) {
	good := NewArtifact("Good", "", WithAbility(WhenEvent(
		PermanentDies().NotSacrificed(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	bad := NewArtifact("Bad", "", WithAbility(WhenEvent(
		PermanentDies().NotSacrificed().NotSacrificed(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
		t.Fatal("duplicate filter accepted")
	}
	bad = NewArtifact("Bad", "", WithAbility(WhenEvent(
		PermanentDies().YouControlled().YouControlled(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
		t.Fatal("duplicate controller filter accepted")
	}
}
