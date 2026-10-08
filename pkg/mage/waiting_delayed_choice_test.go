package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedChoiceOperands(t *testing.T) {
	noSource := GainLifeFor(You, Fixed(0))
	for name, body := range map[string]ResolvingEffect{
		"chooser":     ChooseAtResolution(ControllerOfWithLKI(This), ResolutionOption{Label: "zero", Body: noSource}),
		"unselected":  ChooseAtResolution(You, ResolutionOption{Label: "zero", Body: noSource}, ResolutionOption{Label: "source", Body: GainLifeFor(You, PowerOfWithLKI(This))}),
		"unavailable": ChooseAtResolution(You, ResolutionOption{Label: "source", Body: noSource, When: PermanentIsTapped(This)}),
		"nested":      ChooseAtResolution(You, ResolutionOption{Label: "nested", Body: ChooseAtResolution(You, ResolutionOption{Label: "source", Body: Destroy(This)})}),
	} {
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "{U}", 2, 4)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			delay := AtNext(EachEndStep, body)
			if err := ValidateCardDefinition("Source", NewArtifact("Source", "", WithAbility(Activate(Costs(), delay)))); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, delay, p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			check := func() {
				t.Helper()
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) || got[0].HasColor(core.Red) {
					t.Fatal("choice lost exact source reference")
				}
			}
			check()
			phased := g.PhaseOut(p.ID())
			check()
			g.PhaseIn(phased)
			g.BouncePermanentToHand(p)
			a.RemoveFromHand(card.ID())
			g.PutOnBattlefield(card, b.PlayerID())
			g.ChangeObjectColor(card.ID(), core.Red)
			check()
			clone := g.Clone()
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("clone shares waiting lifetime")
			}
		})
	}
}

func TestWaitingDelayedChoicePurity(t *testing.T) {
	a := &resolutionChooser{BasePlayer: NewBasePlayer("A"), choice: 0}
	g := NewGame(a, NewBasePlayer("B"))
	p := g.PutOnBattlefield(NewCreature("Source", "", 2, 4), a.PlayerID())
	reads := 0
	condition := StateConditionFunc("opaque", []StateDependency{StateBattlefield}, func(StateReader, StateContext) bool { reads++; return true })
	body := ChooseAtResolution(You, ResolutionOption{Label: "one", Body: GainLifeFor(You, Fixed(1))}, ResolutionOption{Label: "source", Body: GainLifeFor(You, PowerOfWithLKI(This)), When: condition})
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || a.prompts != 0 || a.Life() != 20 {
			t.Fatal("query evaluated choice")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if reads != 1 || a.prompts != 1 || a.Life() != 21 {
		t.Fatal("choice did not run once at resolution")
	}
	g.triggers.SetDelayed(nil)
	for _, body := range []ResolvingEffect{
		ChooseAtResolution(You, ResolutionOption{Label: "implicit", Body: GainLifeFor(ControllerOf(AttachedPermanent), Fixed(0)), When: condition}),
		ChooseAtResolution(You, ResolutionOption{Label: "future", Body: AtNext(EachEndStep, Destroy(This))}),
	} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 || reads != 1 {
		t.Fatal("query inferred implicit or future source")
	}
}
