package mage

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestArtifactBecomesTappedCapturesParticipants(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		artifact, opponent    bool
		phaseSource, phaseObj bool
		returnObj             bool
		wantTriggers          int
		wantCounters          uint8
	}{
		{"own artifact", true, false, false, false, false, 1, 1},
		{"opponent artifact", true, true, false, false, false, 1, 1},
		{"nonartifact", false, true, false, false, false, 0, 0},
		{"phased source", true, true, true, false, false, 0, 0},
		{"phased artifact", true, true, false, true, false, 0, 0},
		{"returned artifact", true, true, false, false, true, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			object := BindPermanent("tapped artifact")
			controller := BindPlayer("tap controller")
			watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
				ArtifactBecomesTapped().WithArtifact(object).WithController(controller),
				Sequence(AddCountersTo(From(object), core.Charge, Fixed(1)), GainLifeFor(FromPlayer(controller), Fixed(1))))))
			if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(watcher, a.PlayerID())
			var tapped Card
			if tc.artifact {
				tapped = NewArtifact("Artifact", "")
			} else {
				tapped = NewLand("Land")
			}
			owner := a.PlayerID()
			if tc.opponent {
				owner = b.PlayerID()
			}
			g.PutOnBattlefield(tapped, owner)
			if tc.phaseSource {
				g.PhaseOut(watcher.ID())
			}
			if tc.phaseObj {
				g.PhaseOut(tapped.ID())
			}
			g.TapPermanent(g.FindPermanent(tapped.ID()))
			g.TapPermanent(g.FindPermanent(tapped.ID()))
			g.PutTriggersOnStack()
			if got := g.StackSize(); got != tc.wantTriggers {
				t.Fatalf("stack=%d, want %d", got, tc.wantTriggers)
			}
			if tc.returnObj {
				g.RemoveFromBattlefield(g.FindPermanent(tapped.ID()))
				g.PutOnBattlefield(tapped, a.PlayerID())
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original life")
			}
			g.ResolveStack()
			if got := g.FindPermanentIncludingPhased(tapped.ID()).Counters[core.Charge]; got != tc.wantCounters {
				t.Fatalf("counters=%d, want %d", got, tc.wantCounters)
			}
			wantA, wantB := 20, 20
			if tc.wantTriggers == 1 {
				if tc.opponent {
					wantB++
				} else {
					wantA++
				}
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life A=%d B=%d, want %d %d", a.Life(), b.Life(), wantA, wantB)
			}
		})
	}
}

func TestArtifactBecomesTappedPendingCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	controller := BindPlayer("controller")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ArtifactBecomesTapped().WithController(controller), GainLifeFor(FromPlayer(controller), Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Artifact", "")
	g.PutOnBattlefield(artifact, b.PlayerID())
	g.TapPermanent(g.FindPermanent(artifact.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("pending trigger or copy is absent")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 22 {
		t.Fatalf("life A=%d B=%d", a.Life(), b.Life())
	}
}

func TestArtifactBecomesTappedDeclaration(t *testing.T) {
	artifact := BindPermanent("artifact")
	controller := BindPlayer("controller")
	valid := NewArtifact("Valid", "", WithAbility(WhenEvent(
		ArtifactBecomesTapped().WithArtifact(artifact).WithController(controller),
		GainLifeFor(FromPlayer(controller), Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(ArtifactBecomesTapped()); got.Role != "event-pattern" || got.Text != "artifact becomes tapped" {
		t.Fatalf("inspection=%+v", got)
	}
	for _, pattern := range []EventPattern{
		ArtifactBecomesTapped().WithArtifact(nil),
		ArtifactBecomesTapped().WithArtifact(BindPermanent("")),
		ArtifactBecomesTapped().WithArtifact(artifact).WithArtifact(artifact),
		ArtifactBecomesTapped().WithController(nil),
		ArtifactBecomesTapped().WithController(BindPlayer("")),
		ArtifactBecomesTapped().WithController(controller).WithController(controller),
		ArtifactBecomesTapped().WithArtifact(artifact).WithController(BindPlayer("artifact")),
	} {
		bad := NewArtifact("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("accepted invalid artifact tap pattern")
		}
	}
	for _, body := range []ResolvingEffect{
		AddCountersTo(From(artifact), core.Charge, Fixed(1)),
		GainLifeFor(FromPlayer(controller), Fixed(1)),
	} {
		bad := NewArtifact("Unbound", "", WithAbility(WhenEvent(ArtifactBecomesTapped(), body)))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "producer") {
			t.Fatalf("unbound validation=%v", err)
		}
	}
	badSource := NewInstant("Bad source", "", WithAbility(WhenEvent(ArtifactBecomesTapped(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(badSource.Name(), badSource); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("source validation=%v", err)
	}
}
