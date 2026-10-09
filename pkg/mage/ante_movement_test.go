package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAnteMovementOwnedOtherBatch(t *testing.T) {
	a := &millOrderPlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g, _ := NewGameWithAnte(a, b, nil, nil)
	source := ownedTestCard(a.PlayerID(), "source")
	first, second := ownedTestCard(a.PlayerID(), "first"), ownedTestCard(b.PlayerID(), "second")
	foreign := ownedTestCard(b.PlayerID(), "foreign")
	prior := ownedTestCard(a.PlayerID(), "prior")
	a.AddToGraveyard(prior)
	a.AddToAnte(source)
	a.AddToAnte(first)
	b.AddToAnte(second)
	b.AddToAnte(foreign)
	if err := g.ChangeOwner(second.ID(), a.PlayerID()); err != nil {
		t.Fatal(err)
	}
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}
	set := OtherAnteCards(AnteCardsOwnedBy(You))
	for range 3 {
		if len(set.resolveAnteCards(ctx)) != 2 || a.calls != 0 || len(a.Graveyard()) != 1 {
			t.Fatal("selection made a choice or moved a card")
		}
	}
	events := 0
	watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if e.FromZone == core.ZoneAnte {
			events++
			if e.ToZone != core.ZoneGraveyard || e.PlayerID != a.PlayerID() || len(a.Graveyard()) != 3 || len(a.Ante()) != 1 || len(b.Ante()) != 1 {
				t.Error("event observed an incomplete or wrong-owner batch")
			}
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("observer", "", WithAbility(watcher)), a.PlayerID())
	clone := g.Clone()
	if err := PutAnteCardsInGraveyards(set).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if events != 2 || a.calls != 1 || a.Graveyard()[0].ID() != prior.ID() || a.Graveyard()[1].ID() != second.ID() || a.Graveyard()[2].ID() != first.ID() || a.Ante()[0].ID() != source.ID() || b.Ante()[0].ID() != foreign.ID() {
		t.Fatal("wrong selection, source exclusion or owner order")
	}
	if g.anteVisit(first.ID()) != uuid.Nil || g.graveyardVisit(first.ID()) == uuid.Nil || len(clone.GetPlayer(a.PlayerID()).Ante()) != 2 || len(clone.GetPlayer(b.PlayerID()).Ante()) != 2 {
		t.Fatal("wrong visits or clone storage")
	}
}

type anteMovementSnapshot []capturedCard

func (anteMovementSnapshot) anteCardSet()                                     {}
func (s anteMovementSnapshot) resolveAnteCards(*EffectContext) []capturedCard { return s }

func TestAnteMovementExactVisits(t *testing.T) {
	g, a, b := anteEffectTestGame(t)
	first, second := ownedTestCard(a.PlayerID(), "first"), ownedTestCard(b.PlayerID(), "second")
	a.AddToAnte(first)
	b.AddToAnte(second)
	saved := anteMovementSnapshot{
		{id: first.ID(), visit: g.anteVisit(first.ID()), zone: core.ZoneAnte},
		{id: second.ID(), visit: g.anteVisit(second.ID()), zone: core.ZoneAnte},
		{id: second.ID(), visit: g.anteVisit(second.ID()), zone: core.ZoneAnte},
	}
	a.RemoveFromAnte(first.ID())
	a.AddToAnte(first)
	if err := g.ChangeOwner(second.ID(), a.PlayerID()); err != nil {
		t.Fatal(err)
	}
	if err := PutAnteCardsInGraveyards(saved).Apply(&EffectContext{Game: g}); err != nil {
		t.Fatal(err)
	}
	if len(a.Ante()) != 1 || a.Ante()[0].ID() != first.ID() || len(b.Ante()) != 0 || len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != second.ID() || len(b.Graveyard()) != 0 {
		t.Fatal("saved movement followed a returned card, duplicated a card or used an old owner")
	}
}

type anteMovementOrderPlayer struct {
	*BasePlayer
	onChoice func()
	invalid  bool
}

func (p *anteMovementOrderPlayer) ChooseCardFromLibrary(cards []Card, _ string, _ GameReader) Card {
	if p.onChoice != nil {
		p.onChoice()
	}
	if p.invalid {
		return NewArtifact("invalid", "")
	}
	return cards[len(cards)-1]
}

func TestAnteMovementOrderRecheck(t *testing.T) {
	for _, state := range []string{"valid", "invalid", "return"} {
		t.Run(state, func(t *testing.T) {
			a := &anteMovementOrderPlayer{BasePlayer: NewBasePlayer("A"), invalid: state == "invalid"}
			b := NewBasePlayer("B")
			g, _ := NewGameWithAnte(a, b, nil, nil)
			first, second := ownedTestCard(a.PlayerID(), "first"), ownedTestCard(a.PlayerID(), "second")
			a.AddToAnte(first)
			a.AddToAnte(second)
			a.onChoice = func() {
				if len(a.Ante()) != 2 || len(a.Graveyard()) != 0 {
					t.Error("choice saw partial movement")
				}
				if state == "return" {
					a.RemoveFromAnte(second.ID())
					a.AddToAnte(second)
				}
			}
			if err := PutAnteCardsInGraveyards(AnteCardsOwnedBy(You)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			want := []Card{second, first}
			if state == "invalid" {
				want = []Card{first, second}
			}
			if state == "return" {
				want = []Card{first}
			}
			if len(a.Graveyard()) != len(want) {
				t.Fatal("stale visit or invalid choice changed the moved set")
			}
			for i, card := range want {
				if a.Graveyard()[i].ID() != card.ID() {
					t.Fatal("wrong graveyard order")
				}
			}
		})
	}
}

func TestAnteMovementMultipleOwners(t *testing.T) {
	a, b := &anteMovementOrderPlayer{BasePlayer: NewBasePlayer("A")}, &anteMovementOrderPlayer{BasePlayer: NewBasePlayer("B")}
	g, _ := NewGameWithAnte(a, b, nil, nil)
	g.SetActivePlayerIndex(1)
	saved := make(anteMovementSnapshot, 0, 4)
	var choices []uuid.UUID
	for _, p := range []*anteMovementOrderPlayer{a, b} {
		for range 2 {
			card := ownedTestCard(p.PlayerID(), "ante")
			p.AddToAnte(card)
			saved = append(saved, capturedCard{id: card.ID(), visit: g.anteVisit(card.ID()), zone: core.ZoneAnte})
		}
		p.onChoice = func() {
			choices = append(choices, p.PlayerID())
			if len(a.Ante()) != 2 || len(b.Ante()) != 2 || len(a.Graveyard()) != 0 || len(b.Graveyard()) != 0 {
				t.Error("choice saw partial batch")
			}
		}
	}
	events := 0
	watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if e.FromZone == core.ZoneAnte {
			events++
			if len(a.Ante()) != 0 || len(b.Ante()) != 0 || len(a.Graveyard()) != 2 || len(b.Graveyard()) != 2 {
				t.Error("event saw partial batch")
			}
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("observer", "", WithAbility(watcher)), a.PlayerID())
	if err := PutAnteCardsInGraveyards(saved).Apply(&EffectContext{Game: g}); err != nil {
		t.Fatal(err)
	}
	if events != 4 || len(choices) != 2 || choices[0] != b.PlayerID() || choices[1] != a.PlayerID() {
		t.Fatal("wrong event count or APNAP decision order")
	}
}

func TestAnteMovementLegalityAndDefinition(t *testing.T) {
	for _, state := range []string{"disabled", "settled", "missing-player", "empty"} {
		t.Run(state, func(t *testing.T) {
			g, a, _ := anteEffectTestGame(t)
			card := ownedTestCard(a.PlayerID(), "ante")
			if state != "empty" {
				a.AddToAnte(card)
			}
			controller := a.PlayerID()
			if state == "disabled" {
				g.anteEnabled = false
			}
			if state == "settled" {
				g.anteSettled = true
			}
			if state == "missing-player" {
				controller = uuid.New()
			}
			before := len(a.Ante())
			if err := PutAnteCardsInGraveyards(AnteCardsOwnedBy(You)).Apply(&EffectContext{Game: g, Controller: controller}); err != nil {
				t.Fatal(err)
			}
			if len(a.Ante()) != before || len(a.Graveyard()) != 0 {
				t.Fatal("illegal movement")
			}
		})
	}
	for _, set := range []AnteCardSet{nil, AnteCardsOwnedBy(nil), OtherAnteCards(nil)} {
		card := NewInstant("invalid", "", WithAbility(Spell(PutAnteCardsInGraveyards(set))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("nil operand accepted")
		}
	}
}
