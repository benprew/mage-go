package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func waitingRecipientBody(kind string, recipient DamageRecipientRef, amount ResolutionValue) ResolvingEffect {
	source := DamageSourceOf(AttachedPermanent)
	if kind == "single" {
		return DealDamageFrom(source, recipient, amount)
	}
	return DealDamagePackets(source, DamagePacketTo(You, Fixed(0)), DamagePacketTo(recipient, amount), DamagePacketTo(recipient, Fixed(0)))
}

func TestWaitingDelayedRecipientReferences(t *testing.T) {
	for _, kind := range []string{"single", "packets"} {
		for _, shape := range []string{"direct", "sequence", "mode"} {
			for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "color", "control"} {
				t.Run(kind+"/"+shape+"/"+state, func(t *testing.T) {
					a, b := NewBasePlayer("A"), NewBasePlayer("B")
					g := NewGame(a, b)
					calls := 0
					body := waitingRecipientBody(kind, This, waitingMultiAmount{&calls})
					delay := AtNext(EachEndStep, body)
					if shape == "sequence" {
						delay = AtNext(EachEndStep, Sequence(Sequence(body)))
					}
					if shape == "mode" {
						delay = AtNextModes(EachEndStep, ModeOf("First", body), ModeOf("Second", body))
					}
					card := NewArtifact("Recipient", "{U}", WithAbility(Activate(Costs(), delay)))
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
						t.Fatal("recipient declaration lost source incarnation")
					}
					if calls != 0 || a.Life() != 20 || b.Life() != 20 {
						t.Fatal("query evaluated damage")
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
					if calls != 0 {
						t.Fatal("absent damage source evaluated amount")
					}
				})
			}
		}
	}
}

func TestWaitingDelayedRecipientBoundaries(t *testing.T) {
	for _, kind := range []string{"single", "packets"} {
		t.Run(kind, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			for _, recipient := range []DamageRecipientRef{You, AttachedPermanent, From(BindPermanent("other"))} {
				if err := ApplyEffect(g, AtNext(EachEndStep, waitingRecipientBody(kind, recipient, Fixed(0))), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if err := ApplyEffect(g, AtNext(EachEndStep, AtNext(EachEndStep, waitingRecipientBody(kind, This, Fixed(0)))), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if len(g.WaitingDelayedPermanentSources()) != 0 {
				t.Fatal("nonself or nested future recipient implied self")
			}
			for _, recipient := range []DamageRecipientRef{nil, From(nil)} {
				card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, waitingRecipientBody(kind, recipient, Fixed(1))))))
				if ValidateCardDefinition(card.Name(), card) == nil {
					t.Fatal("invalid recipient accepted")
				}
			}
		})
	}
}

func TestWaitingDelayedRecipientPacketGameplay(t *testing.T) {
	for _, state := range []string{"live", "returned", "phase", "cycle", "source-departed"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			target := DeclarePermanentTarget("damage source", IsCreature)
			body := DealDamagePackets(DamageSourceOf(target), DamagePacketTo(This, Fixed(1)), DamagePacketTo(You, Fixed(2)), DamagePacketTo(This, Fixed(1)))
			card := NewCreature("Recipient", "", 1, 10, WithAbility(Activate(Costs(), AtNext(EachEndStep, body), WithTargets(target))))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			q := g.PutOnBattlefield(NewCreature("Damage source", "", 1, 10), b.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{q.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if got := g.WaitingDelayedPermanentSources(); len(got) != 2 || !got[0].SameObject(saved) && !got[1].SameObject(saved) {
				t.Fatal("packet recipient missing beside bound source")
			}
			switch state {
			case "returned":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "source-departed":
				g.BouncePermanentToHand(q)
			}
			clone := g.Clone()
			wantDamage := 2
			if state == "returned" || state == "phase" {
				wantDamage = 0
			}
			notifications := 0
			g.SetOnDamageDealt(func(_, _ string, _ int, _ bool) {
				notifications++
				if a.Life() != 18 || g.FindPermanentIncludingPhased(p.ID()).Damage != wantDamage {
					t.Error("notification saw an incomplete packet batch")
				}
			})
			for _, game := range []*Game{g, clone} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				if game.GetPlayer(a.PlayerID()).Life() != 18 || game.GetPlayer(b.PlayerID()).Life() != 20 || game.FindPermanentIncludingPhased(p.ID()).Damage != wantDamage {
					t.Fatal("packet lost recipient identity, LKI source or saved controller")
				}
			}
			wantNotifications := 2
			if wantDamage == 0 {
				wantNotifications = 1
			}
			if notifications != wantNotifications {
				t.Fatal("duplicate recipient packets did not combine")
			}
		})
	}
}
