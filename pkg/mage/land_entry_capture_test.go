package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestLandEntryPermanentDeclaration(t *testing.T) {
	binding := BindPermanent("entering land")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "valid", Card: NewArtifact("Good", "", WithAbility(WhenEvent(LandEnters().WithPermanent(binding), TapSubject(From(binding)))))},
		{Name: "nil", Card: NewArtifact("Bad", "", WithAbility(WhenEvent(LandEnters().WithPermanent(nil), GainLifeFor(You, Fixed(1))))), WantErr: true},
		{Name: "empty", Card: NewArtifact("Bad", "", WithAbility(WhenEvent(LandEnters().WithPermanent(BindPermanent("")), GainLifeFor(You, Fixed(1))))), WantErr: true},
		{Name: "duplicate", Card: NewArtifact("Bad", "", WithAbility(WhenEvent(LandEnters().WithPermanent(binding).WithPermanent(binding), TapSubject(From(binding))))), WantErr: true},
		{Name: "unbound", Card: NewArtifact("Bad", "", WithAbility(WhenEvent(LandEnters(), TapSubject(From(binding))))), WantErr: true},
	})
}

func TestLandEntryPermanentIdentity(t *testing.T) {
	for _, change := range []string{"present", "departed", "returned", "phased", "phase cycle"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			binding := BindPermanent("entering land")
			watcher := g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(WhenEvent(LandEnters().WithPermanent(binding), TapSubject(From(binding))))), a.PlayerID())
			land := NewLand("Land")
			g.PutOnBattlefield(land, b.PlayerID())
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("entry did not trigger")
			}
			if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			switch change {
			case "departed", "returned":
				g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
				if change == "returned" {
					g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
					g.PutOnBattlefield(land, a.PlayerID())
				}
			case "phased", "phase cycle":
				g.PhaseOut(land.ID())
				if change == "phase cycle" {
					g.PhaseIn([]uuid.UUID{land.ID()})
				}
			}
			clone := g.Clone()
			clone.ResolveStack()
			if p := g.FindPermanent(land.ID()); p != nil && p.Tapped {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			want := change == "present" || change == "phase cycle"
			if p := g.FindPermanent(land.ID()); p != nil && p.Tapped != want {
				t.Fatalf("tapped=%v want=%v", p.Tapped, want)
			}
		})
	}
}

func TestLandEntryPermanentCombinedCaptures(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land, actor, controller := BindPermanent("land"), BindPlayer("actor"), BindPlayer("controller")
	card := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		LandEnters().WithPermanent(land).WithActor(actor).WithController(controller),
		Sequence(TapSubject(From(land)), GainLifeFor(FromPlayer(actor), Fixed(2)), GainLifeFor(FromPlayer(controller), Fixed(1))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, a.PlayerID())
	first := g.PutOnBattlefieldBy(NewLand("First"), b.PlayerID(), a.PlayerID())
	second := g.PutOnBattlefieldBy(NewLand("Second"), a.PlayerID(), b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("independent entries")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if g.FindPermanent(first.ID()).Tapped || g.FindPermanent(second.ID()).Tapped || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed captures")
	}
	g.ResolveStack()
	if !g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped || a.Life() != 23 || b.Life() != 23 {
		t.Fatal("capture combination lost a binding")
	}
}
