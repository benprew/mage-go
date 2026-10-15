package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCounterSnapshotEntry(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phased", "phase-cycle", "replacement", "copy-entry"} {
		t.Run(state, func(t *testing.T) {
			a, b := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}, &optionalEntryPlayer{BasePlayer: NewBasePlayer("B"), accept: true}
			g := NewGame(a, b)
			sourcePlan, err := NewCardDefinitionPlan(func() Card { return NewCreature("Subject", "", 2, 8) })
			if err != nil {
				t.Fatal(err)
			}
			card, err := sourcePlan.Build()
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			p.AddCounter(core.Charge, 300)
			p.AddCounter(core.P1P1, 2)
			ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
			if state == "departed" || state == "returned" {
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID()).AddCounter(core.Charge, 9)
				}
			}
			if state == "phased" || state == "phase-cycle" {
				phased := g.PhaseOut(p.ID())
				if state == "phase-cycle" {
					g.PhaseIn(phased)
				}
			}
			saved := BindCounterSnapshot("saved")
			if err := SnapshotCounters(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			var entrant Card = NewCreature("Entrant", "", 2, 8, WithAbility(EntersWithNCounters(core.Charge, 1)))
			if state == "copy-entry" {
				plan, err := NewCardDefinitionPlan(func() Card {
					return NewCreature("Entrant", "", 2, 8, WithAbility(EntersWithNCounters(core.Charge, 1)), WithAbility(MayCopyAsEnters(IsCreature)))
				})
				if err != nil {
					t.Fatal(err)
				}
				entrant, err = plan.Build()
				if err != nil {
					t.Fatal(err)
				}
			}
			entrant.SetOwner(b.PlayerID())
			g.ExileCard(entrant, uuid.Nil)
			body := PutExiledCardsOntoBattlefieldUnderOwnersControl(savedTransportVisits{{id: entrant.ID(), visit: g.FindExiledCard(entrant.ID()).Visit, zone: core.ZoneExile}}, EnterWithSavedCounters(saved), EnterTapped())
			if state == "replacement" {
				g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(action Action) bool {
					add, ok := action.(*AddCountersAction)
					return ok && add.PermanentID() == entrant.ID() && add.CounterType() == core.Charge
				}, replaceFunc: func(action Action, _ *Game) Action {
					add := action.(*AddCountersAction)
					return add.WithAmount(add.Amount() + 1)
				}})
			}
			clone := g.Clone()
			if state == "copy-entry" {
				clone.SetPlayerAt(1, &counterSnapshotPlayer{Player: clone.GetPlayer(b.PlayerID())})
			}
			if err := SnapshotCounters(This, saved).Apply(&EffectContext{Game: clone, SourceID: uuid.New()}); err != nil {
				t.Fatal(err)
			}
			for _, game := range []*Game{g, clone} {
				local := *ctx
				local.Game = game
				if err := body.Apply(&local); err != nil {
					t.Fatal(err)
				}
				got := game.FindPermanent(entrant.ID())
				want, pt := 301, 2
				if state == "copy-entry" {
					want = 300
				}
				if state == "replacement" {
					want++
				}
				if game == clone {
					want, pt = 1, 0
					if state == "copy-entry" {
						want = 0
					}
					if state == "replacement" {
						want++
					}
				}
				if got == nil {
					t.Fatal("entry failed")
				}
				if got.Counters[core.Charge] != want || got.Counters[core.P1P1] != pt || !got.Tapped || got.ControllerID() != b.PlayerID() {
					t.Fatalf("wrong restored entry: charge=%d pt=%d", got.Counters[core.Charge], got.Counters[core.P1P1])
				}
			}
		})
	}
}

func TestCounterSnapshotDeclarations(t *testing.T) {
	saved := BindCounterSnapshot("saved")
	cards := BindCards("cards")
	producer := ExilePermanentGroupsInto(PermanentExileGroup{Subjects: SingletonPermanent(This), Result: cards})
	entry := func(options ...BattlefieldEntryOption) ResolvingEffect {
		return PutExiledCardsOntoBattlefieldUnderOwnersControl(ExiledCardsFrom(cards), options...)
	}
	valid := Sequence(SnapshotCounters(This, saved), producer, entry(EnterWithSavedCounters(saved)))
	if err := ValidateCardDefinition("Valid", NewArtifact("Valid", "", WithAbility(Activate(Costs(), valid)))); err != nil {
		t.Fatal(err)
	}
	for _, body := range []ResolvingEffect{
		SnapshotCounters(nil, saved), SnapshotCounters(This, nil), SnapshotCounters(This, &CounterSnapshotBinding{}),
		SnapshotCounters(From(BindPermanent("missing")), saved),
		Sequence(producer, entry(EnterWithSavedCounters(saved))),
		Sequence(SnapshotCounters(This, saved), producer, entry(EnterWithSavedCounters(nil))),
		Sequence(SnapshotCounters(This, saved), producer, entry(EnterWithSavedCounters(saved), EnterWithSavedCounters(saved))),
		Sequence(IfPositive(Fixed(1), SnapshotCounters(This, saved)), producer, entry(EnterWithSavedCounters(saved))),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid counter snapshot accepted")
		}
	}
}

func TestCounterSnapshotAllKinds(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Subject", ""), a.PlayerID())
	for counter := range core.NumCounters {
		p.Counters[counter] = 300 + int(counter)
	}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	first, second := BindCounterSnapshot("first"), BindCounterSnapshot("second")
	if err := SnapshotCounters(This, first).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	for counter := range core.NumCounters {
		g.MutablePermanent(p.ID()).Counters[counter] = 0
	}
	if err := SnapshotCounters(This, second).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	for counter := range core.NumCounters {
		if FromNumber(first.counts[counter]).resolveValue(ctx) != 300+int(counter) || FromNumber(second.counts[counter]).resolveValue(ctx) != 0 {
			t.Fatal("snapshot was shared or missed a counter kind")
		}
	}
	g.resolution.Clear()
	for counter := range core.NumCounters {
		if FromNumber(first.counts[counter]).resolveValue(ctx) != 0 {
			t.Fatal("snapshot escaped resolution cleanup")
		}
	}
}

func TestCounterSnapshotGraveyardAndWaitingLKI(t *testing.T) {
	for _, state := range []string{"entry", "cancel", "waiting"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			sourceCard := NewArtifact("Source", "")
			sourceCard.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			source.AddCounter(core.Charge, 3)
			ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: source.ID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
			saved := BindCounterSnapshot("saved")
			card := NewCreature("Entrant", "", 2, 8, WithAbility(EntersWithNCounters(core.Charge, 1)), WithAbility(WhenEvent(Enters(This), GainLifeFor(You, CountersOn(This, core.Charge)))))
			card.SetOwner(b.PlayerID())
			b.AddToGraveyard(card)
			bound := BindCard("entrant")
			g.resolution.cardBindings = map[*CardBinding]capturedCard{bound: {id: card.ID(), visit: g.graveyardVisit(card.ID()), zone: core.ZoneGraveyard}}
			body := Sequence(SnapshotCounters(This, saved), ReturnCardToBattlefield(FromCard(bound), You, EnterWithSavedCounters(saved)))
			if state == "cancel" {
				g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(action Action) bool {
					entry, ok := action.(*battlefieldEntryAction)
					return ok && entry.permanent.ID() == card.ID()
				}, replaceFunc: func(Action, *Game) Action { return nil }})
			}
			if state == "waiting" {
				if err := AtNext(EachEndStep, body).Apply(ctx); err != nil {
					t.Fatal(err)
				}
				g.BouncePermanentToHand(source)
				for range 2 {
					if len(g.WaitingDelayedPermanentSources()) != 1 || len(g.resolution.numberBindings) != 0 || g.FindPermanent(card.ID()) != nil {
						t.Fatal("waiting query changed snapshot or lost LKI")
					}
				}
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			} else if err := body.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if state == "cancel" {
				if g.FindPermanent(card.ID()) != nil || len(b.Graveyard()) != 1 || a.Life() != 20 {
					t.Fatal("cancelled entry restored counters")
				}
			} else if p := g.FindPermanent(card.ID()); p == nil || p.Counters[core.Charge] != 4 || a.Life() != 24 {
				t.Fatal("entry observation or waiting source snapshot failed")
			}
		})
	}
}

type counterSnapshotPlayer struct{ Player }

func (*counterSnapshotPlayer) ChooseMayAbility(string) bool { return true }
