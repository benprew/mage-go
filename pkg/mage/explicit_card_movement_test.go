package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestExplicitCardMovementHand(t *testing.T) {
	for _, state := range []string{"exchange", "owner-rule", "return", "wrong-zone", "missing-player"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g, _ := NewGameWithAnte(a, b, nil, nil)
			card := NewArtifact("revealed", "")
			card.SetOwner(b.PlayerID())
			b.AddToHand(card)
			binding := BindCard("reveal")
			ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
			if err := RevealRandomHandCardInto(You, binding).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if state == "exchange" {
				if err := g.ChangeOwner(card.ID(), a.PlayerID()); err != nil {
					t.Fatal(err)
				}
			}
			if state == "return" {
				b.RemoveFromHand(card.ID())
				b.AddToHand(card)
			}
			if state == "wrong-zone" {
				b.RemoveFromHand(card.ID())
				b.AddToGraveyard(card)
			}
			clone := g.Clone()
			ctx.Controller = a.PlayerID()
			player := PlayerRef(You)
			if state == "missing-player" {
				ctx.Controller = uuid.New()
			}
			if err := PutHandCardInHandFor(FromCard(binding), player).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 0
			if state == "exchange" {
				want = 1
			}
			if len(a.Hand()) != want || len(clone.GetPlayer(a.PlayerID()).Hand()) != 0 {
				t.Fatal("incorrect destination or shared clone")
			}
			if state == "exchange" {
				if len(b.Hand()) != 0 {
					t.Fatal("card remains in former hand")
				}
				if _, ok := FromCard(binding).resolveCard(ctx); ok {
					t.Fatal("movement kept the saved hand visit")
				}
			}
		})
	}
}

func TestExplicitCardMovementAnywhere(t *testing.T) {
	for _, zone := range []core.Zone{core.ZoneHand, core.ZoneLibrary, core.ZoneGraveyard, core.ZoneExile, core.ZoneBattlefield, core.ZoneStack, core.ZoneAnte, core.ZoneCommand} {
		t.Run(zone.String(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g, _ := NewGameWithAnte(a, b, nil, nil)
			card := NewCreature("source", "", 2, 2)
			card.SetOwner(a.PlayerID())
			switch zone {
			case core.ZoneHand:
				a.AddToHand(card)
			case core.ZoneLibrary:
				a.SetLibrary([]Card{card})
			case core.ZoneGraveyard:
				card.SetOwner(b.PlayerID())
				b.AddToGraveyard(card)
				if err := g.ChangeOwner(card.ID(), a.PlayerID()); err != nil {
					t.Fatal(err)
				}
			case core.ZoneExile:
				g.ExileCard(card, uuid.Nil)
			case core.ZoneBattlefield:
				g.PutOnBattlefield(card, b.PlayerID())
			case core.ZoneStack:
				g.stack.Push(&StackObject{ID: uuid.New(), Card: card, Controller: b.PlayerID()})
			case core.ZoneAnte:
				a.AddToAnte(card)
			case core.ZoneCommand:
				if _, err := g.AddCommandObject(card, false); err != nil {
					t.Fatal(err)
				}
			}
			clone := g.Clone()
			ctx := &EffectContext{Game: g, Controller: b.PlayerID(), SourceID: card.ID()}
			if err := PutSourceCardInGraveyardFromAnywhere(You).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if g.CardZone(card.ID()) != core.ZoneGraveyard || len(a.Graveyard()) != 1 || len(b.Graveyard()) != 0 {
				t.Fatal("physical source did not reach owner graveyard")
			}
			if clone.physicalCardZone(card.ID()) != zone {
				t.Fatal("movement changed clone")
			}
			visit := g.graveyardVisit(card.ID())
			if err := PutSourceCardInGraveyardFromAnywhere(You).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if visit != g.graveyardVisit(card.ID()) || len(a.Graveyard()) != 1 {
				t.Fatal("same destination made another visit")
			}
		})
	}
}

func TestExplicitCardMovementPhasingAndDeclarations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("source", "", 2, 2)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.PhaseOut(p.ID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID()}
	if err := PutSourceCardInGraveyardFromAnywhere(You).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanentIncludingPhased(p.ID()) == nil || len(a.Graveyard()) != 0 {
		t.Fatal("phased source moved")
	}
	for _, effect := range []Effect{PutHandCardInHandFor(nil, You), PutHandCardInHandFor(FromCard(nil), You), PutSourceCardInGraveyardFromAnywhere(nil)} {
		c := NewInstant("invalid", "", Spell(effect))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid movement declaration accepted")
		}
	}
}

func TestExplicitCardMovementReplacements(t *testing.T) {
	for _, state := range []string{"redirect", "cancel", "return", "battlefield"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("source", "", 2, 2)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			calls := 0
			if state == "battlefield" {
				a.RemoveFromHand(card.ID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				if err := ExileIfWouldDieThisTurn(This).Apply(&EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID()}); err != nil {
					t.Fatal(err)
				}
			} else {
				g.effects.AddReplacement(&testPrioritizedReplacement{
					replacementBase: replacementBase{sourceID: uuid.New(), priority: ReplacementPriorityDefault},
					matchFunc: func(action Action) bool {
						move, ok := action.(*CardZoneMoveAction)
						return ok && move.Destination() == core.ZoneGraveyard
					},
					replaceFunc: func(action Action, game *Game) Action {
						calls++
						move := action.(*CardZoneMoveAction)
						if move.CardID() != card.ID() || move.FromZone() != core.ZoneHand || move.PlayerID() != a.PlayerID() {
							t.Fatal("replacement lost subject facts")
						}
						if state == "cancel" {
							return nil
						}
						if state == "return" {
							a.RemoveFromHand(card.ID())
							a.AddToHand(card)
							return move
						}
						return move.WithExile(move.ActionSource())
					},
				})
			}
			ctx := &EffectContext{Game: g, SourceID: card.ID(), Controller: b.PlayerID()}
			if err := PutSourceCardInGraveyardFromAnywhere(You).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := core.ZoneHand
			if state == "redirect" || state == "battlefield" {
				want = core.ZoneExile
			}
			if g.physicalCardZone(card.ID()) != want {
				t.Fatal("replacement or exact-visit recheck failed")
			}
			if state != "battlefield" && calls != 1 {
				t.Fatal("replacement did not run exactly once")
			}
		})
	}
}

func TestExplicitCardMovementEvents(t *testing.T) {
	for _, from := range []core.Zone{core.ZoneHand, core.ZoneGraveyard, core.ZoneBattlefield} {
		t.Run(from.String(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g, _ := NewGameWithAnte(a, b, nil, nil)
			card := NewCreature("source", "", 2, 2)
			card.SetOwner(a.PlayerID())
			switch from {
			case core.ZoneHand:
				a.AddToHand(card)
			case core.ZoneGraveyard:
				card.SetOwner(b.PlayerID())
				b.AddToGraveyard(card)
				if err := g.ChangeOwner(card.ID(), a.PlayerID()); err != nil {
					t.Fatal(err)
				}
			case core.ZoneBattlefield:
				g.PutOnBattlefield(card, b.PlayerID())
			}
			zones, left, discard, sacrifice := 0, 0, 0, 0
			for _, event := range []core.EventType{core.EvtZoneChange, core.EvtCardsLeftGraveyard, core.EvtDiscard, core.EvtSacrifice} {
				watcher := NewTriggered(event, false).SetCondition(func(evt *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
					switch evt.Type {
					case core.EvtZoneChange:
						if evt.SourceID != card.ID() {
							return false
						}
						zones++
						if evt.FromZone != from || evt.ToZone != core.ZoneGraveyard || len(a.Graveyard()) != 1 || len(b.Graveyard()) != 0 || g.physicalCardZone(card.ID()) != core.ZoneGraveyard {
							t.Fatal("event precedes placement or has incorrect zones")
						}
					case core.EvtCardsLeftGraveyard:
						left++
					case core.EvtDiscard:
						discard++
					case core.EvtSacrifice:
						sacrifice++
					}
					return false
				})
				g.PutOnBattlefield(NewArtifact("watcher", "", WithAbility(watcher)), a.PlayerID())
			}
			if err := PutSourceCardInGraveyardFromAnywhere(You).Apply(&EffectContext{Game: g, Controller: a.PlayerID(), SourceID: card.ID()}); err != nil {
				t.Fatal(err)
			}
			wantLeft := 0
			if from == core.ZoneGraveyard {
				wantLeft = 1
			}
			if zones != 1 || left != wantLeft || discard != 0 || sacrifice != 0 {
				t.Fatalf("incorrect events: zones=%d left=%d discard=%d sacrifice=%d", zones, left, discard, sacrifice)
			}
			if from == core.ZoneBattlefield && (g.LKI(card.ID()) == nil || !g.LKI(card.ID()).Snapshot.HasType(core.TypeCreature)) {
				t.Fatal("battlefield departure lost LKI")
			}
		})
	}
}
