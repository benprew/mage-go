package mage

import (
	"fmt"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedCombatReferences(t *testing.T) {
	for _, combat := range []bool{true, false} {
		for _, location := range []string{"condition", "body", "nested", "mode"} {
			for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "color", "control"} {
				t.Run(fmt.Sprintf("attacking=%t/%s/%s", combat, location, state), func(t *testing.T) {
					a, b := NewBasePlayer("A"), NewBasePlayer("B")
					g := NewGame(a, b)
					other := AttachedPermanent
					branch := IfPermanentBlocking
					if combat {
						branch = IfPermanentAttacking
					}
					body := branch(This, GainLifeFor(You, Fixed(2)))
					switch location {
					case "body":
						body = branch(other, Destroy(This))
					case "nested":
						body = branch(other, Sequence(branch(other, Destroy(This))))
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
					if a.Life() != 20 {
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
					if a.Life() != 20 || b.Life() != 20 {
						t.Fatal("false or unknown combat condition executed a body")
					}
				})
			}
		}
	}
}

func TestWaitingDelayedCombatBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	other := From(BindPermanent("other"))
	for _, body := range []ResolvingEffect{
		IfPermanentAttacking(other, GainLifeFor(You, Fixed(1))),
		IfPermanentBlocking(other, AtNext(EachEndStep, Destroy(This))),
		IfPermanentAttacking(other, Sequence(IfPermanentBlocking(other, GainLifeFor(You, Fixed(1))))),
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
		IfPermanentAttacking(nil, Destroy(This)),
		IfPermanentBlocking(This, nil),
		IfPermanentAttacking(This, IfPermanentBlocking(nil, Destroy(This))),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))) == nil {
			t.Fatal("invalid conditional declaration accepted")
		}
	}
}

func TestWaitingDelayedCombatQueryPurity(t *testing.T) {
	for _, attacking := range []bool{true, false} {
		t.Run(fmt.Sprintf("attacking=%t", attacking), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			calls := 0
			branch := IfPermanentBlocking
			if attacking {
				branch = IfPermanentAttacking
			}
			card := NewCreature("Source", "", 1, 10, WithAbility(Activate(Costs(), AtNext(EachEndStep, branch(This, GainLifeFor(You, waitingMultiAmount{&calls}))))))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			if attacking {
				g.combat.AddAttackerIncarnation(p.ID(), p.incarnationID, b.PlayerID())
			} else {
				q := g.PutOnBattlefield(NewCreature("Attacker", "", 1, 10), b.PlayerID())
				g.combat.AddAttackerIncarnation(q.ID(), q.incarnationID, a.PlayerID())
				g.combat.AddBlockerIncarnation(p.ID(), p.incarnationID, q.ID())
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			for range 2 {
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) || calls != 0 {
					t.Fatal("query executed a true combat body or lost its declaration")
				}
			}
			clone := g.Clone()
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.RemoveFromCombat(p.ID())
			g.ResolveStack()
			if calls != 0 {
				t.Fatal("branch used detection-time combat state")
			}
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			clone.PutTriggersOnStack()
			clone.ResolveStack()
			if calls != 1 {
				t.Fatal("clone lost its true combat condition")
			}
		})
	}
}
