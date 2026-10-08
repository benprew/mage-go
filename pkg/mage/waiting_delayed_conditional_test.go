package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedConditionalReferences(t *testing.T) {
	for _, location := range []string{"condition", "then", "else", "nested", "mode"} {
		for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "color", "control"} {
			t.Run(location+"/"+state, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				other := AttachedPermanent
				calls := 0
				filter := PermanentFilter{fn: func(*Permanent, *Game) bool { calls++; return false }}
				var body = IfPermanentMatches(This, filter, GainLifeFor(You, Fixed(2)))
				switch location {
				case "then":
					body = IfPermanentMatches(other, filter, Destroy(This))
				case "else":
					body = IfPermanentMatchesElse(other, filter, GainLifeFor(You, Fixed(2)), Destroy(This))
				case "nested":
					body = UnlessPermanentMatches(other, filter, Sequence(IfPermanentMatchesElse(other, filter, Destroy(This), Destroy(This))))
				case "mode":
					body = IfPermanentMatches(This, filter, GainLifeFor(You, Fixed(2)))
				}
				delay := AtNext(EachEndStep, Sequence(body))
				if location == "mode" {
					delay = AtNextModes(EachEndStep, ModeOf("Condition", body), ModeOf("Duplicate", body))
				}
				card := NewArtifact("Source", "{U}", WithAbility(Activate(Costs(), delay)))
				if err := ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				saved := g.CaptureDamageSource(p.ID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if len(g.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("unresolved delay supplied references")
				}
				g.ResolveStack()
				switch state {
				case "departed", "returned":
					g.BouncePermanentToHand(p)
					if state == "returned" {
						a.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, b.PlayerID())
						g.ChangeObjectColor(card.ID(), core.Red)
					}
				case "phased":
					g.PhaseOut(p.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "color":
					g.ChangeObjectColor(p.ID(), core.Red)
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) {
					t.Fatal("conditional reference lost source incarnation")
				}
				if calls != 0 || a.Life() != 20 {
					t.Fatal("query evaluated a condition or body")
				}
				if got[0].HasColor(core.Red) != (state == "color") {
					t.Fatal("wrong current or departure facts")
				}
				if state == "control" && got[0].Controller() != b.PlayerID() {
					t.Fatal("wrong current controller")
				}
				got[0].card.AddType(core.TypeEnchantment)
				if g.WaitingDelayedPermanentSources()[0].HasType(core.TypeEnchantment) {
					t.Fatal("query shares mutable facts")
				}
				clone := g.Clone()
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
					t.Fatal("waiting lifetime or clone isolation")
				}
				g.PutTriggersOnStack()
				g.ResolveStack()
				wantCalls := 0
				if location == "condition" || location == "mode" {
					wantCalls = 1
				}
				if calls != wantCalls || a.Life() != 20 || b.Life() != 20 {
					t.Fatal("false or unknown condition executed a branch")
				}
			})
		}
	}
}

func TestWaitingDelayedConditionalBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	other := From(BindPermanent("other"))
	for _, body := range []ResolvingEffect{
		IfPermanentMatches(other, IsArtifact, GainLifeFor(You, Fixed(1))),
		UnlessPermanentMatches(other, IsArtifact, AtNext(EachEndStep, Destroy(This))),
		IfPermanentMatchesElse(other, IsArtifact, AtNext(EachEndStep, Destroy(This)), GainLifeFor(You, Fixed(1))),
	} {
		g.triggers.SetDelayed(nil)
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(g.WaitingDelayedPermanentSources()) != 0 {
			t.Fatal("bound or nested future reference implied self")
		}
	}
	for _, body := range []ResolvingEffect{
		IfPermanentMatches(nil, IsArtifact, Destroy(This)),
		IfPermanentMatches(This, IsArtifact, nil),
		IfPermanentMatchesElse(This, IsArtifact, Destroy(This), nil),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))) == nil {
			t.Fatal("invalid conditional declaration accepted")
		}
	}
}
