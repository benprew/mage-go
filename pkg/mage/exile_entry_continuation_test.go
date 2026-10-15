package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestExileEntryContinuation(t *testing.T) {
	for _, state := range []string{"normal", "stale", "token", "nonpermanent", "illegal-host"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 2, 3)
			if state == "token" {
				card = NewToken("Token", 2, 3, []core.CardType{core.TypeCreature}, nil)
			}
			if state == "nonpermanent" {
				card = NewInstant("Spell", "")
			}
			if state == "illegal-host" {
				card = NewAura("Aura", "", WithCastTarget(TargetCreature()))
			}
			card.SetOwner(b.PlayerID())
			g.ExileCard(card, card.ID())
			binding := BindCard("card")
			result := BindPermanent("entrant")
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			g.resolution.cardBindings = map[*CardBinding]capturedCard{binding: {id: card.ID(), visit: g.FindExiledCard(card.ID()).Visit, zone: core.ZoneExile}}
			old := g.PutOnBattlefield(NewArtifact("Old", ""), a.PlayerID())
			g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{result: {old.ID(), old.incarnationID}}
			if state == "stale" {
				g.RemoveFromExile(card.ID())
				g.ExileCard(card, card.ID())
			}
			body := PutExiledCardOntoBattlefieldOrElse(FromCard(binding), result, Sequence(GainLifeFor(You, Fixed(2)), TapSubject(From(result))), GainLifeFor(You, Fixed(5)), EnterTapped())
			if err := body.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if state == "normal" {
				p := g.FindPermanent(card.ID())
				if p == nil || !p.Tapped || p.ControllerID() != b.PlayerID() || a.Life() != 22 {
					t.Fatal("entry result, owner or success branch")
				}
			} else if g.FindPermanent(card.ID()) != nil || a.Life() != 25 {
				t.Fatal("failed entry selected success")
			}
			if state != "normal" {
				if _, ok := From(result).resolvePermanent(ctx); ok {
					t.Fatal("failed entry retained prior result")
				}
			}
		})
	}
}

func TestExileEntryContinuationDeclarations(t *testing.T) {
	card := DeclareGraveyardCardTarget("card")
	result := BindPermanent("entrant")
	for _, body := range []ResolvingEffect{
		PutExiledCardOntoBattlefieldOrElse(nil, result, GainLifeFor(You, Fixed(1)), nil),
		PutExiledCardOntoBattlefieldOrElse(card, nil, GainLifeFor(You, Fixed(1)), nil),
		PutExiledCardOntoBattlefieldOrElse(card, result, nil, nil),
		PutExiledCardOntoBattlefieldOrElse(card, result, nil, TapSubject(From(result))),
		Sequence(PutExiledCardOntoBattlefieldOrElse(card, result, TapSubject(From(result)), nil), TapSubject(From(result))),
	} {
		c := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), body, WithTargets(card))))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatalf("accepted invalid %T", body)
		}
	}
	valid := NewArtifact("Valid", "", WithAbility(Activate(Costs(), PutExiledCardOntoBattlefieldOrElse(card, result, TapSubject(From(result)), nil), WithTargets(card))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
}

func TestExileEntryContinuationObservations(t *testing.T) {
	for _, state := range []string{"replacement", "failed-entry", "departs-on-entry"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 2, 4, WithAbility(EntersWithNCounters(core.Charge, 1)))
			if state == "failed-entry" {
				card.AddAbility(&graveyardRejectEntry{})
			}
			card.SetOwner(b.PlayerID())
			g.ExileCard(card, card.ID())
			binding := BindCard("card")
			result := BindPermanent("entrant")
			g.resolution.cardBindings = map[*CardBinding]capturedCard{binding: {id: card.ID(), visit: g.FindExiledCard(card.ID()).Visit, zone: core.ZoneExile}}
			if state == "replacement" {
				g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(action Action) bool {
					add, ok := action.(*AddCountersAction)
					return ok && add.PermanentID() == card.ID()
				}, replaceFunc: func(action Action, _ *Game) Action {
					add := action.(*AddCountersAction)
					return add.WithAmount(add.Amount() + 1)
				}})
			}
			observed := 0
			watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(event *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				if event.SourceID == card.ID() && event.ToZone == core.ZoneBattlefield {
					observed++
					p := g.FindPermanent(card.ID())
					if p == nil || !p.Tapped {
						t.Error("entry observation preceded tapped entry")
					}
					if state == "replacement" && p.Counters[core.Charge] != 2 {
						t.Error("entry observation preceded counter replacement")
					}
					if state == "departs-on-entry" {
						g.DestroyPermanent(p)
					}
				}
				return false
			})
			g.PutOnBattlefield(NewArtifact("Watcher", "", WithAbility(watcher)), a.PlayerID())
			body := PutExiledCardOntoBattlefieldOrElse(FromCard(binding), result, GainLifeFor(You, Fixed(2)), GainLifeFor(You, Fixed(5)), EnterTapped())
			if err := body.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if state == "failed-entry" {
				if observed != 0 || a.Life() != 25 || len(b.Graveyard()) != 1 {
					t.Fatal("cancelled entry selected success")
				}
			} else if observed != 1 || a.Life() != 22 {
				t.Fatal("successful entry selected wrong continuation")
			}
		})
	}
}
