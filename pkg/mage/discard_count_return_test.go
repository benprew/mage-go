package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDiscardCountReturnOutcomes(t *testing.T) {
	for _, mode := range []string{"normal", "library", "cancel", "returned", "short", "zero"} {
		t.Run(mode, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first, second := NewLand("First"), NewLand("Second")
			a.AddToHand(first)
			a.AddToHand(second)
			switch mode {
			case "library":
				g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
			case "cancel":
				g.effects.AddReplacement(&preventRandomDiscardReplacement{card: first.ID()})
			case "returned":
				g.effects.AddReplacement(&playerDiscardReturningReplacement{card: first.ID()})
			}
			count := BindNumber("completed discards")
			amount := 2
			if mode == "short" {
				amount = 5
			}
			if mode == "zero" {
				amount = 0
			}
			events := 0
			observer := NewTriggered(core.EvtDiscard, false).SetCondition(func(_ *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				events++
				want := 0
				if mode == "cancel" || mode == "returned" {
					want = 1
				}
				if len(a.Hand()) != want {
					t.Error("event saw partial discard batch")
				}
				return false
			})
			g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer)), b.PlayerID())
			effect := Sequence(Discard(You, Fixed(amount), DiscardCountInto(count)), GainLifeFor(You, FromNumber(count)))
			if err := ValidateCardDefinition("Count", NewSorcery("Count", "", Spell(effect))); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			want := 2
			if mode == "cancel" || mode == "returned" {
				want = 1
			}
			if mode == "zero" {
				want = 0
			}
			if a.Life() != 20+want || events != want {
				t.Fatalf("completed count: life %d events %d", a.Life(), events)
			}
		})
	}
}

type countReturnChooser struct {
	*BasePlayer
	choose func([]Card) Card
}

func (p *countReturnChooser) ChooseCardFromGraveyard(cards []Card, _ string, _ GameReader) Card {
	return p.choose(cards)
}

func TestDiscardCountReturnChoicesAndVisits(t *testing.T) {
	for _, mode := range []string{"normal", "returned", "new", "invalid", "short", "zero", "last"} {
		t.Run(mode, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first, second, third := NewLand("First"), NewLand("Second"), NewLand("Third")
			for _, c := range []Card{first, second, third} {
				c.SetOwner(b.PlayerID())
				b.AddToGraveyard(c)
			}
			calls := 0
			g.SetPlayerAt(0, &countReturnChooser{BasePlayer: a, choose: func(cards []Card) Card {
				calls++
				if len(b.Graveyard()) != 3 && mode == "normal" {
					t.Fatal("choice saw partial return")
				}
				if calls == 1 {
					switch mode {
					case "returned":
						b.RemoveFromGraveyard(first.ID())
						b.AddToGraveyard(first)
					case "new":
						c := NewLand("New")
						c.SetOwner(b.PlayerID())
						b.AddToGraveyard(c)
					case "invalid":
						return NewLand("Foreign")
					}
				}
				if mode == "last" {
					return cards[len(cards)-1]
				}
				return cards[0]
			}})
			left := 0
			g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtCardsLeftGraveyard, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				left += e.Amount
				if len(b.Hand()) != e.Amount {
					t.Error("departure observed incomplete destination")
				}
				return false
			}))), a.PlayerID())
			amount := 2
			if mode == "short" {
				amount = 8
			}
			if mode == "zero" {
				amount = 0
			}
			g.SetActivePlayerIndex(1)
			target := ActivePlayer
			effect := ReturnChosenGraveyardCardsToHand(target, Fixed(amount), NewCardFilter("any card", func(Card) bool { return true }))
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), []uuid.UUID{b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			want := 2
			if mode == "returned" {
				want = 1
			}
			if mode == "short" {
				want = 3
			}
			if mode == "zero" {
				want = 0
			}
			if len(a.Hand()) != 0 || len(b.Hand()) != want || left != want {
				t.Fatalf("return destination/count %d/%d/%d", len(a.Hand()), len(b.Hand()), left)
			}
			if mode == "last" && (b.Hand()[0].ID() != third.ID() || b.Hand()[1].ID() != second.ID()) {
				t.Fatal("controller choice was ignored")
			}
			if mode == "new" && b.Hand()[0].Name() == "New" {
				t.Fatal("new visit joined selection")
			}
		})
	}
}

func TestDiscardCountReturnDefinition(t *testing.T) {
	count := BindNumber("count")
	for _, effect := range []ResolvingEffect{
		Discard(You, Fixed(1), DiscardCountInto(nil)),
		Discard(You, Fixed(1), DiscardCountInto(count), DiscardCountInto(count)),
		ReturnChosenGraveyardCardsToHand(nil, Fixed(1), NewCardFilter("any card", func(Card) bool { return true })),
		ReturnChosenGraveyardCardsToHand(You, nil, NewCardFilter("any card", func(Card) bool { return true })),
		ReturnChosenGraveyardCardsToHand(You, Fixed(1), CardFilter{}),
		GainLifeFor(You, FromNumber(count)),
	} {
		if err := ValidateCardDefinition("Bad", NewSorcery("Bad", "", Spell(effect))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestDiscardCountReturnSavedFramesAndReset(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	count := BindNumber("discard count")
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	a.AddToHand(NewLand("Held"))
	body := Sequence(Discard(You, Fixed(4), DiscardCountInto(count)), AtNext(EachEndStep, GainLifeFor(You, FromNumber(count))))
	if err := ValidateCardDefinition("Frames", NewSorcery("Frames", "", Spell(body))); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.ResolveStack()
	if a.Life() != 21 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("saved count or clone isolation")
	}
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 21 {
		t.Fatal("clone lost saved count")
	}
	g.resolution.numberBindings = map[*NumberBinding]int{count: 99}
	if err := Discard(You, Fixed(0), DiscardCountInto(count)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.resolution.numberBindings[count] != 0 {
		t.Fatal("zero reused stale count")
	}
	missing := FromPlayer(BindPlayer("missing"))
	g.resolution.numberBindings[count] = 99
	if err := Discard(missing, Fixed(1), DiscardCountInto(count)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.resolution.numberBindings[count] != 0 {
		t.Fatal("missing player reused stale count")
	}
}

func TestDiscardCountReturnMandatorySelection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := NewLand("First"), NewLand("Second")
	a.AddToHand(first)
	a.AddToHand(second)
	b.AddToHand(NewLand("Foreign"))
	g.SetPlayerAt(0, &handCardinalityPlayer{BasePlayer: a, choose: func(n int, _ string, _ GameReader) []Card {
		if n != 2 {
			t.Fatal("choice was not capped to hand")
		}
		return []Card{nil, b.Hand()[0], first, first}
	}})
	count := BindNumber("actual")
	if err := ApplyEffect(g, Sequence(Discard(You, Fixed(3), DiscardCountInto(count)), GainLifeFor(You, FromNumber(count))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || len(a.Hand()) != 0 || len(a.Graveyard()) != 2 || len(b.Hand()) != 1 {
		t.Fatal("mandatory distinct selection failed")
	}
}
