package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedLibraryOperands(t *testing.T) {
	for name, body := range map[string]ResolvingEffect{
		"viewer":          LookAtLibraryTop(ControllerOfWithLKI(This), You, 0, false),
		"library":         LookAtLibraryTop(You, OwnerOf(This), 0, false),
		"count":           LookAtLibraryTopNumber(You, You, PowerOfWithLKI(This), false),
		"chooser":         MayShuffleLibrary(ControllerOfWithLKI(This), You),
		"shuffle library": MayShuffleLibrary(You, OwnerOf(This)),
	} {
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Source", "{U}", 2, 4)
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			check := func() {
				t.Helper()
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) || len(g.LibraryLooks(a.PlayerID())) != 0 {
					t.Fatal("lost reference or ran inspection", got)
				}
			}
			check()
			phased := g.PhaseOut(p.ID())
			check()
			g.PhaseIn(phased)
			check()
			g.BouncePermanentToHand(p)
			g.zones.ClearLKI()
			check()
			a.RemoveFromHand(c.ID())
			g.PutOnBattlefield(c, b.PlayerID())
			check()
			clone := g.Clone()
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("shared wait lifetime")
			}
		})
	}
}

func TestWaitingDelayedLibraryPurity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 2, 4), a.PlayerID())
	for range 4 {
		a.AddToLibrary(NewLand("Land"))
	}
	reads := 0
	body := LookAtLibraryTopNumber(You, You, AddValues(PowerOfWithLKI(This), ValueAtResolution(waitingTokenCount{&reads})), false)
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || len(g.LibraryLooks(a.PlayerID())) != 0 {
			t.Fatal("query ran count")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if reads != 1 || len(g.LibraryLooks(a.PlayerID())) != 4 || len(a.Library()) != 4 || len(g.LibraryLooks(b.PlayerID())) != 0 {
		t.Fatal("inspection timing or privacy", reads)
	}
	for _, body := range []ResolvingEffect{LookAtLibraryTopNumber(You, You, nil, false), LookAtLibraryTopNumber(You, You, Fixed(-1), false)} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), body)))) == nil {
			t.Fatal("invalid count accepted")
		}
	}
	for _, body := range []ResolvingEffect{LookAtLibraryTop(You, You, 0, false), LookAtLibraryTop(You, ControllerOf(AttachedPermanent), 0, false), AtNext(EachEndStep, MayShuffleLibrary(OwnerOf(This), You))} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("inferred implicit or future reference")
	}
}
