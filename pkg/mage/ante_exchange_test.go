package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAnteExchangeAtomicAndOwnership(t *testing.T) {
	for _, state := range []string{"normal", "empty", "foreign", "token", "returned", "disabled", "settled", "missing"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := anteEffectTestGame(t)
			card := ownedTestCard(b.PlayerID(), "ante")
			b.AddToAnte(card)
			visit := g.anteVisit(card.ID())
			top, next := ownedTestCard(a.PlayerID(), "top"), ownedTestCard(a.PlayerID(), "next")
			a.SetLibrary([]Card{top, next})
			target := DeclareAnteCardTarget("ante")
			ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: card.ID(), incarnation: visit}}}}
			switch state {
			case "empty":
				a.SetLibrary(nil)
			case "foreign":
				top.SetOwner(b.PlayerID())
			case "token":
				top = NewToken("token", 1, 1, []core.CardType{core.TypeCreature}, nil)
				top.SetOwner(a.PlayerID())
				a.SetLibrary([]Card{top, next})
			case "returned":
				b.RemoveFromAnte(card.ID())
				b.AddToAnte(card)
			case "disabled":
				g.anteEnabled = false
			case "settled":
				g.anteSettled = true
			case "missing":
				ctx.Controller = uuid.New()
			}
			clone := g.Clone()
			if err := AcquireCardOwnership(target, You).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			acquired := state != "returned" && state != "disabled" && state != "settled" && state != "missing"
			if (g.FindCardAnywhere(card.ID()).Owner() == a.PlayerID()) != acquired {
				t.Fatal("ownership result")
			}
			if clone.FindCardAnywhere(card.ID()).Owner() != b.PlayerID() || g.anteVisit(card.ID()) != clone.anteVisit(card.ID()) {
				t.Fatal("ownership changed clone or visit")
			}
			events := 0
			watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				if e.SourceID == card.ID() || e.SourceID == top.ID() {
					events++
					if len(a.Library()) == 0 || a.Library()[0].ID() != card.ID() || g.anteVisit(card.ID()) != uuid.Nil {
						t.Error("partial exchange")
					}
					if state != "empty" && (len(a.Ante()) != 1 || a.Ante()[0].ID() != top.ID()) {
						t.Error("missing other half")
					}
				}
				return false
			})
			g.PutOnBattlefield(NewArtifact("observer", "", WithAbility(watcher)), a.PlayerID())
			if err := ExchangeAnteCardWithLibraryTop(target, You).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			moved := state == "normal" || state == "empty"
			if moved {
				if a.Library()[0].ID() != card.ID() || len(b.Ante()) != 0 {
					t.Fatal("exchange result")
				}
				want := 2
				if state == "empty" {
					want = 1
				} else if len(a.Library()) != 2 || a.Library()[1].ID() != next.ID() || g.anteVisit(top.ID()) == uuid.Nil {
					t.Fatal("top and tail")
				}
				if events != want {
					t.Fatalf("events=%d want=%d", events, want)
				}
			} else if events != 0 || g.anteVisit(card.ID()) == uuid.Nil {
				t.Fatal("incomplete exchange moved cards")
			}
		})
	}
}

func TestAnteExchangeRequiresSameOwner(t *testing.T) {
	g, a, b := anteEffectTestGame(t)
	card := ownedTestCard(b.PlayerID(), "ante")
	b.AddToAnte(card)
	top := ownedTestCard(a.PlayerID(), "top")
	a.SetLibrary([]Card{top})
	target := DeclareAnteCardTarget("ante")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: card.ID(), incarnation: g.anteVisit(card.ID())}}}}
	if err := ExchangeAnteCardWithLibraryTop(target, You).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(b.Ante()) != 1 || a.Library()[0].ID() != top.ID() || card.Owner() != b.PlayerID() {
		t.Fatal("exchange acquired ownership")
	}
}

func TestAnteExchangeDefinition(t *testing.T) {
	for _, effect := range []ResolvingEffect{AcquireCardOwnership(nil, You), AcquireCardOwnership(DeclareAnteCardTarget("ante"), nil), ExchangeAnteCardWithLibraryTop(nil, You), ExchangeAnteCardWithLibraryTop(DeclareAnteCardTarget("ante"), nil)} {
		if err := ValidateCardDefinition("invalid", NewSorcery("invalid", "", WithAbility(Spell(effect)))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestAnteExchangeOwnershipOtherZonesAndSettlement(t *testing.T) {
	for _, zone := range []core.Zone{core.ZoneGraveyard, core.ZoneExile, core.ZoneHand} {
		t.Run(zone.String(), func(t *testing.T) {
			g, a, b := anteEffectTestGame(t)
			card := ownedTestCard(b.PlayerID(), "owned card")
			binding := BindCard("card")
			var visit uuid.UUID
			switch zone {
			case core.ZoneGraveyard:
				b.AddToGraveyard(ownedTestCard(b.PlayerID(), "neighbor"))
				b.AddToGraveyard(card)
				visit = g.graveyardVisit(card.ID())
			case core.ZoneExile:
				g.ExileCard(card, b.PlayerID())
				visit = g.FindExiledCard(card.ID()).Visit
			case core.ZoneHand:
				b.AddToHand(card)
				visit = b.handVisit(card.ID())
			}
			g.resolution.cardBindings = map[*CardBinding]capturedCard{binding: {id: card.ID(), visit: visit, zone: zone}}
			clone := g.Clone()
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			if err := AcquireCardOwnership(FromCard(binding), You).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if g.CardZone(card.ID()) != zone || g.FindCardAnywhere(card.ID()).Owner() != a.PlayerID() || clone.FindCardAnywhere(card.ID()).Owner() != b.PlayerID() {
				t.Fatal("ownership changed zone or clone")
			}
			if zone == core.ZoneGraveyard && g.graveyardVisit(b.Graveyard()[0].ID()) != clone.graveyardVisit(b.Graveyard()[0].ID()) {
				t.Fatal("ownership changed neighbor visit")
			}
			if _, ok := FromCard(binding).resolveCard(ctx); !ok {
				t.Fatal("ownership lost visit")
			}
			b.SetLife(0)
			changes, err := g.AnteResult()
			if err != nil || len(changes) != 1 || changes[0].CardID != card.ID() || changes[0].OriginalOwner != b.PlayerID() || changes[0].FinalOwner != a.PlayerID() {
				t.Fatal("ownership settlement", changes, err)
			}
		})
	}
}

func TestAnteExchangeReadsCurrentTop(t *testing.T) {
	g, a, b := anteEffectTestGame(t)
	card := ownedTestCard(a.PlayerID(), "ante")
	b.AddToAnte(card)
	target := DeclareAnteCardTarget("ante")
	top := ownedTestCard(a.PlayerID(), "old top")
	a.SetLibrary([]Card{top})
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	spell := NewSorcery("exchange", "", WithAbility(Spell(ExchangeAnteCardWithLibraryTop(target, You), WithTargets(target))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{card.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	newTop := ownedTestCard(a.PlayerID(), "new top")
	a.SetLibrary([]Card{newTop, top})
	clone := g.Clone()
	clone.ResolveStack()
	if len(a.Ante()) != 0 || a.Library()[0].ID() != newTop.ID() || len(b.Ante()) != 1 {
		t.Fatal("clone movement leaked")
	}
	g.ResolveStack()
	if a.Ante()[0].ID() != newTop.ID() || a.Library()[0].ID() != card.ID() || a.Library()[1].ID() != top.ID() {
		t.Fatal("exchange used announcement top")
	}
}

func TestAnteExchangeWaitingPlayerReferences(t *testing.T) {
	for _, ownership := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g, _ := NewGameWithAnte(a, b, nil, nil)
		source := NewCreature("source", "", 1, 1)
		source.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(source, a.PlayerID())
		saved := g.CaptureDamageSource(p.ID())
		target := DeclareAnteCardTarget("ante")
		var body = ExchangeAnteCardWithLibraryTop(target, ControllerOfWithLKI(This))
		if ownership {
			body = AcquireCardOwnership(target, OwnerOf(This))
		}
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		check := func() {
			t.Helper()
			refs := g.WaitingDelayedPermanentSources()
			if len(refs) != 1 || !refs[0].SameObject(saved) || len(a.Library()) != 0 || len(a.Ante()) != 0 || len(g.originalOwners) != 0 {
				t.Fatal("waiting query changed state or lost identity")
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
		a.RemoveFromHand(source.ID())
		g.PutOnBattlefield(source, b.PlayerID())
		check()
		clone := g.Clone()
		g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
			t.Fatal("waiting lifetime shared")
		}
	}
}
