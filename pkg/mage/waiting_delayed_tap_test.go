package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedTapReferences(t *testing.T) {
	for _, untap := range []bool{false, true} {
		for _, shape := range []string{"direct", "sequence", "mode"} {
			for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "control", "color", "unchanged"} {
				name := "tap"
				if untap {
					name = "untap"
				}
				t.Run(name+"/"+shape+"/"+state, func(t *testing.T) {
					a, b := NewBasePlayer("A"), NewBasePlayer("B")
					g := NewGame(a, b)
					var body = TapSubject(This)
					if untap {
						body = UntapPermanent(This)
					}
					delay := AtNext(EachEndStep, body)
					if shape == "sequence" {
						delay = AtNext(EachEndStep, Sequence(Sequence(body, body)))
					}
					if shape == "mode" {
						delay = AtNextModes(EachEndStep, ModeOf("First", body), ModeOf("Second", body))
					}
					card := NewArtifact("Source", "{U}", WithAbility(Activate(Costs(), delay)))
					if err := ValidateCardDefinition(card.Name(), card); err != nil {
						t.Fatal(err)
					}
					card.SetOwner(a.PlayerID())
					p := g.PutOnBattlefield(card, a.PlayerID())
					p.Tapped = untap
					if state == "unchanged" {
						p.Tapped = !untap
					}
					saved := g.CaptureDamageSource(p.ID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if len(g.WaitingDelayedPermanentSources()) != 0 {
						t.Fatal("unresolved delay supplies references")
					}
					g.ResolveStack()
					switch state {
					case "departed", "returned":
						g.BouncePermanentToHand(p)
						if state == "returned" {
							a.RemoveFromHand(card.ID())
							p = g.PutOnBattlefield(card, b.PlayerID())
							p.Tapped = untap
							g.ChangeObjectColor(p.ID(), core.Red)
						}
					case "phased":
						g.PhaseOut(p.ID())
					case "cycle":
						g.PhaseIn(g.PhaseOut(p.ID()))
					case "control":
						g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					case "color":
						g.ChangeObjectColor(p.ID(), core.Red)
					}
					before := p.Tapped
					g.SetRandomResults([]int{7})
					for range 3 {
						got := g.WaitingDelayedPermanentSources()
						if len(got) != 1 || !got[0].SameObject(saved) {
							t.Fatal("tap declaration lost saved source")
						}
						if got[0].HasColor(core.Red) != (state == "color") {
							t.Fatal("wrong live or departure facts")
						}
						if state == "control" && got[0].Controller() != b.PlayerID() {
							t.Fatal("wrong live controller")
						}
						if p.Tapped != before || g.StackSize() != 0 {
							t.Fatal("query executed tap action")
						}
						got[0].card.AddType(core.TypeEnchantment)
						if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeEnchantment) {
							t.Fatal("query shares mutable facts")
						}
					}
					if g.RandIntn(10) != 7 {
						t.Fatal("query consumed randomness")
					}
					clone := g.Clone()
					for _, game := range []*Game{g, clone} {
						game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
						if len(game.WaitingDelayedPermanentSources()) != 0 {
							t.Fatal("detected delay still waits")
						}
						game.PutTriggersOnStack()
						game.ResolveStack()
						live := game.FindPermanentIncludingPhased(p.ID())
						if live != nil {
							want := !untap
							if state == "returned" || state == "phased" {
								want = before
							}
							if live.Tapped != want {
								t.Fatal("tap action followed wrong incarnation or phasing state")
							}
						}
						if game == g && len(clone.WaitingDelayedPermanentSources()) != 1 {
							t.Fatal("clone shares delayed lifetime")
						}
					}
				})
			}
		}
	}
}

func TestWaitingDelayedTapBoundaries(t *testing.T) {
	for _, untap := range []bool{false, true} {
		action := func(ref PermanentRef) ResolvingEffect {
			if untap {
				return UntapPermanent(ref)
			}
			return TapSubject(ref)
		}
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
		for _, ref := range []PermanentRef{AttachedPermanent, From(BindPermanent("other"))} {
			if err := ApplyEffect(g, AtNext(EachEndStep, action(ref)), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
		}
		if err := ApplyEffect(g, AtNext(EachEndStep, AtNext(EachEndStep, action(This))), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(g.WaitingDelayedPermanentSources()) != 0 {
			t.Fatal("nonself or future body implies source")
		}
		for _, ref := range []PermanentRef{nil, From(nil)} {
			card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, action(ref)))))
			if ValidateCardDefinition(card.Name(), card) == nil {
				t.Fatal("invalid tap declaration accepted")
			}
		}
	}
}
