package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceTappedStateWithLKILifecycle(t *testing.T) {
	for _, tapped := range []bool{false, true} {
		for _, change := range []string{"live", "invert", "leave", "return", "second-departure", "phase", "phase-cycle", "unknown", "clear-transient-LKI"} {
			t.Run(change+map[bool]string{false: "/untapped", true: "/tapped"}[tapped], func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				c := NewArtifact("Source", "")
				c.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
				p.Tapped = tapped
				ctx := StateContext{SourceID: p.ID(), ControllerID: a.PlayerID(), SourceIncarnation: p.incarnationID}
				condition := SourceTappedStateWithLKI(tapped)
				switch change {
				case "invert":
					g.MutablePermanent(p.ID()).Tapped = !tapped
				case "leave", "return", "second-departure", "clear-transient-LKI":
					g.BouncePermanentToHand(p)
					if change == "return" || change == "second-departure" {
						a.RemoveFromHand(c.ID())
						q := g.PutOnBattlefield(c, a.PlayerID())
						q.Tapped = !tapped
						if change == "second-departure" {
							g.BouncePermanentToHand(q)
						}
					}
					if change == "clear-transient-LKI" {
						g.zones.ClearLKI()
					}
				case "phase":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "unknown":
					ctx.SourceIncarnation = uuid.New()
				}
				want := change != "invert" && change != "unknown"
				if condition.checkState(g, ctx) != want || condition.checkState(g.Clone(), ctx) != want {
					t.Fatal("wrong source state or incarnation")
				}
				if want && SourceTappedStateWithLKI(!tapped).checkState(g, ctx) {
					t.Fatal("inverse state also true")
				}
				if change == "phase" {
					ctx.SourceIncarnation = uuid.Nil
					if condition.checkState(g, ctx) {
						t.Fatal("phased source satisfied event-time check")
					}
				}
				if change == "leave" {
					ctx.SourceIncarnation = uuid.Nil
					if condition.checkState(g, ctx) {
						t.Fatal("unbound read used old LKI")
					}
				}
			})
		}
	}
}

func TestSourceTappedStateWithLKITrigger(t *testing.T) {
	for _, change := range []string{"none", "tap", "leave", "tap-leave", "return", "copy", "phase"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewArtifact("Watcher", "", WithAbility(WhenEvent(TurnStepBegins(EachDrawStep, BindPlayer("event player")), GainLifeFor(You, Fixed(2)), WithInterveningIf(SourceTappedStateWithLKI(false)))))
			if err := ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.FireEvent(GameEvent{Type: EvtDrawStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("missing detection")
			}
			switch change {
			case "tap", "tap-leave":
				g.TapPermanent(p)
				if change == "tap-leave" {
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				}
			case "leave", "return":
				g.BouncePermanentToHand(p)
				if change == "return" {
					a.RemoveFromHand(c.ID())
					g.TapPermanent(g.PutOnBattlefield(c, a.PlayerID()))
				}
			case "phase":
				g.PhaseOut(p.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed live game")
			}
			g.ResolveStack()
			want := 22
			if change == "tap" || change == "tap-leave" {
				want = 20
			}
			wantB := 20
			if change == "copy" {
				wantB = 22
			}
			if a.Life() != want || b.Life() != wantB || clone.GetPlayer(a.PlayerID()).Life() != want {
				t.Fatal("condition recheck, LKI, or copy failed")
			}
		})
	}
	invalid := NewInstant("Invalid", "", WithEffect(IfState(SourceTappedStateWithLKI(false), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
		t.Fatal("spell context accepted")
	}
}

func TestDrawStepDetectionBeforeNormalDraw(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(2)
	watcher := g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(At(EachDrawStep, GainLifeFor(You, Fixed(2)), WithInterveningIf(SourceTappedStateWithLKI(false))))), a.PlayerID())
	g.TapPermanent(watcher)
	replacement := g.PutOnBattlefield(NewArtifact("Replacement", ""), a.PlayerID())
	g.AddEmptyLibraryDrawReplacement(replacement.ID(), a.PlayerID(), func(game *Game, _ uuid.UUID) { game.UntapPermanent(game.FindPermanent(watcher.ID())) })
	g.RunStepWithPriority(Draw)
	if a.Life() != 20 {
		t.Fatal("normal draw created a beginning-of-step trigger")
	}
}

func TestDrawStepPriorityAfterNormalDraw(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetTurn(2)
	for range 3 {
		a.AddToLibrary(NewLand("Library"))
	}
	g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(At(EachDrawStep, DrawFor(You, Fixed(1))))), a.PlayerID())
	first := true
	g.SetOnPriority(func(game *Game, _ int, _ bool) PriorityAction {
		if first {
			first = false
			if len(a.Hand()) != 1 || game.StackSize() != 1 {
				t.Fatal("priority before normal draw or trigger placement")
			}
		}
		return PriorityAction{Type: PriorityPass}
	})
	g.RunStepWithPriority(Draw)
	if first || len(a.Hand()) != 2 {
		t.Fatal("draw timing or trigger resolution")
	}
}
