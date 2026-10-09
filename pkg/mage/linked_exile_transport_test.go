package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLinkedExileTransport(t *testing.T) {
	for _, destination := range []core.Zone{core.ZoneHand, core.ZoneGraveyard} {
		t.Run(fmt.Sprint(destination), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewArtifact("Source", "")
			source.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(source, a.PlayerID())
			ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
			link := NewExileLink()
			first, second := NewLand("First"), NewLand("Second")
			first.SetOwner(a.PlayerID())
			second.SetOwner(b.PlayerID())
			a.SetLibrary([]Card{first, second})
			for range 2 {
				if err := ExileLibraryTopLinked(You, link, true).Apply(ctx); err != nil {
					t.Fatal(err)
				}
			}
			unlinked := NewLand("Unlinked")
			g.ExileCard(unlinked, p.ID())

			events := 0
			observer := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				if e.FromZone == core.ZoneExile {
					events++
					if e.ToZone != destination || g.FindExiledCard(first.ID()) != nil || g.FindExiledCard(second.ID()) != nil {
						t.Error("event preceded the complete transport batch")
					}
				}
				return false
			})
			g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer)), a.PlayerID())
			clone := g.Clone()
			body := ReturnExiledCardsToOwnersHands(CardsExiledWith(link, This))
			if destination == core.ZoneGraveyard {
				body = PutExiledCardsInOwnersGraveyards(CardsExiledWith(link, This))
			}
			g.DestroyPermanent(p)
			if err := body.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if g.FindExiledCard(first.ID()) != nil || g.FindExiledCard(second.ID()) != nil || g.FindExiledCard(unlinked.ID()) == nil {
				t.Fatal("wrong linked visits moved")
			}
			if events != 2 {
				t.Fatal("wrong zone event count")
			}
			if len(clone.GetExile()) != 3 {
				t.Fatal("transport changed clone")
			}
			for _, player := range []Player{a, b} {
				if destination == core.ZoneHand && len(player.Hand()) != 1 {
					t.Fatal("wrong hand owner")
				}
				if destination == core.ZoneGraveyard && len(player.Graveyard()) == 0 {
					t.Fatal("missing graveyard arrival")
				}
			}
		})
	}
}

func TestLinkedExileTransportDefinition(t *testing.T) {
	for _, effect := range []ResolvingEffect{ReturnExiledCardsToOwnersHands(nil), PutExiledCardsInOwnersGraveyards(nil), ReturnExiledCardsToOwnersHands(CardsExiledWith(nil, This)), PutExiledCardsInOwnersGraveyards(CardsExiledWith(NewExileLink(), nil))} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(ManaPayment("")), effect)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid transport declaration accepted")
		}
	}
}

type savedTransportVisits []capturedCard

func (savedTransportVisits) exiledCardSet()                                     {}
func (s savedTransportVisits) resolveExiledCards(*EffectContext) []capturedCard { return s }

func TestLinkedExileTransportExactVisitsAndOwners(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g, err := NewGameWithAnte(a, b, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	first, second := NewLand("First"), NewLand("Second")
	first.SetOwner(a.PlayerID())
	second.SetOwner(a.PlayerID())
	g.ExileCardFaceDown(first, uuid.New())
	g.ExileCardFaceDown(second, uuid.New())
	saved := make(savedTransportVisits, 0, 3)
	saved = append(saved, savedTransportVisits{
		{id: first.ID(), visit: g.FindExiledCard(first.ID()).Visit, zone: core.ZoneExile},
		{id: second.ID(), visit: g.FindExiledCard(second.ID()).Visit, zone: core.ZoneExile},
	}...)
	saved = append(saved, saved[1])
	removed, _ := g.RemoveFromExile(first.ID())
	g.ExileCard(removed, uuid.New())
	if err := g.ChangeOwner(second.ID(), b.PlayerID()); err != nil {
		t.Fatal(err)
	}
	if err := ReturnExiledCardsToOwnersHands(saved).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.FindExiledCard(first.ID()) == nil || g.FindExiledCard(second.ID()) != nil || len(a.Hand()) != 0 || len(b.Hand()) != 1 || b.Hand()[0].ID() != second.ID() {
		t.Fatal("transport followed a new visit, duplicated a card, or ignored current ownership")
	}
}

func TestLinkedExileTransportOrderAndWaiting(t *testing.T) {
	a := &millOrderPlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewArtifact("Source", "")
	source.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(source, a.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	link := NewExileLink()
	first, second := NewLand("First"), NewLand("Second")
	first.SetOwner(a.PlayerID())
	second.SetOwner(a.PlayerID())
	a.SetLibrary([]Card{first, second})
	for range 2 {
		if err := ExileLibraryTopLinked(You, link, true).Apply(ctx); err != nil {
			t.Fatal(err)
		}
	}
	body := PutExiledCardsInOwnersGraveyards(CardsExiledWith(link, This))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(p)
	for range 2 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || len(g.GetExile()) != 2 || a.calls != 0 {
			t.Fatal("waiting query executed transport or lost source")
		}
	}
	if err := body.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.calls != 1 || len(a.Graveyard()) != 2 || a.Graveyard()[0].ID() != second.ID() || a.Graveyard()[1].ID() != first.ID() {
		t.Fatal("owner order was not used")
	}
}

type exileTransportOrderPlayer struct {
	*BasePlayer
	choose func([]Card) Card
}

func (p *exileTransportOrderPlayer) ChooseCardFromLibrary(cards []Card, _ string, _ GameReader) Card {
	return p.choose(cards)
}

func TestLinkedExileTransportOrderRechecks(t *testing.T) {
	for _, reexile := range []bool{false, true} {
		t.Run(fmt.Sprint(reexile), func(t *testing.T) {
			a := &exileTransportOrderPlayer{BasePlayer: NewBasePlayer("A")}
			b := &exileTransportOrderPlayer{BasePlayer: NewBasePlayer("B")}
			g := NewGame(a, b)
			saved := make(savedTransportVisits, 0, 4)
			cards := make([]Card, 0, 4)
			for i := range 4 {
				card := NewLand(fmt.Sprint(i))
				owner := a.PlayerID()
				if i >= 2 {
					owner = b.PlayerID()
				}
				card.SetOwner(owner)
				g.ExileCardFaceDown(card, uuid.New())
				cards = append(cards, card)
				saved = append(saved, capturedCard{id: card.ID(), visit: g.FindExiledCard(card.ID()).Visit, zone: core.ZoneExile})
			}
			var order []uuid.UUID
			a.choose = func(options []Card) Card {
				order = append(order, a.PlayerID())
				if len(g.GetExile()) != 4 || len(a.Graveyard())+len(b.Graveyard()) != 0 {
					t.Error("movement preceded order choices")
				}
				return options[len(options)-1]
			}
			b.choose = func(options []Card) Card {
				order = append(order, b.PlayerID())
				if len(g.GetExile()) != 4 || len(a.Graveyard())+len(b.Graveyard()) != 0 {
					t.Error("movement preceded all owners' choices")
				}
				if reexile {
					card, _ := g.RemoveFromExile(cards[1].ID())
					g.ExileCard(card, uuid.New())
				}
				return options[len(options)-1]
			}
			if err := PutExiledCardsInOwnersGraveyards(saved).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if len(order) != 2 || order[0] != a.PlayerID() || order[1] != b.PlayerID() {
				t.Fatal("choices were not in active-player order")
			}
			if len(b.Graveyard()) != 2 || b.Graveyard()[0].ID() != cards[3].ID() || g.graveyardVisit(cards[3].ID()) == uuid.Nil {
				t.Fatal("wrong order or graveyard visit")
			}
			if reexile {
				if len(a.Graveyard()) != 1 || g.FindExiledCard(cards[1].ID()) == nil {
					t.Fatal("choice replaced an exact saved visit")
				}
			} else if len(a.Graveyard()) != 2 || a.Graveyard()[0].ID() != cards[1].ID() {
				t.Fatal("wrong first owner order")
			}
		})
	}
}
