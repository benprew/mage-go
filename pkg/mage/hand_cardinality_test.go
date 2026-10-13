package mage

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type handCardinalityPlayer struct {
	*BasePlayer
	choose func(int, string, GameReader) []Card
}

func (p *handCardinalityPlayer) ChooseCardsFromHand(n int, reason string, g GameReader) []Card {
	return p.choose(n, reason, g)
}

func TestHandCardinalityChoicesAndBatch(t *testing.T) {
	for _, active := range []int{0, 1} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		g.SetActivePlayerIndex(active)
		var order []string
		for index, p := range []Player{a, b} {
			for i := range 3 {
				p.AddToHand(NewLand(p.Name() + string(rune('a'+i))))
			}
			g.SetPlayerAt(index, &handCardinalityPlayer{BasePlayer: p.(*BasePlayer), choose: func(n int, reason string, _ GameReader) []Card {
				order = append(order, p.Name())
				if n != 2 || len(a.Hand()) != 3 || len(b.Hand()) != 3 {
					t.Fatal("choices observed partial movement or wrong count")
				}
				if strings.Contains(reason, "Previous choices") || strings.Contains(reason, "Aa") || strings.Contains(reason, "Ba") {
					t.Fatal("hidden card names escaped through prompt")
				}
				return []Card{p.Hand()[2], p.Hand()[0]}
			}})
		}
		events := 0
		g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtDiscard, false).SetCondition(func(_ *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
			events++
			if len(a.Hand()) != 1 || len(b.Hand()) != 1 || a.Hand()[0].Name() != "Ab" || b.Hand()[0].Name() != "Bb" {
				t.Fatal("discard event saw partial batch or wrong choices")
			}
			return false
		}))), a.PlayerID())
		if err := ApplyEffect(g, DiscardAllButEach(EachPlayer, Fixed(1)), uuid.Nil, b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if events != 4 || len(order) != 2 || order[0] != g.ActivePlayerObj().Name() {
			t.Fatal("APNAP choices or batch events", order, events)
		}
	}
}

func TestHandCardinalityBoundsLeastAndDeclarations(t *testing.T) {
	for _, keep := range []int{-1, 0, 1, 3, 20} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		for range 3 {
			a.AddToHand(NewLand("A"))
		}
		b.AddToHand(NewLand("B"))
		players := playerDamageTestSet{[]uuid.UUID{a.PlayerID(), b.PlayerID(), a.PlayerID(), uuid.New()}}
		ctx := &EffectContext{Game: g}
		if got := LeastHandSize(players).resolveValue(ctx); got != 1 || LeastHandSize(playerDamageTestSet{}).resolveValue(ctx) != 0 {
			t.Fatal("least hand size", got)
		}
		g.SetPlayerAt(0, &handCardinalityPlayer{BasePlayer: a, choose: func(_ int, _ string, _ GameReader) []Card {
			return []Card{nil, b.Hand()[0], a.Hand()[2], a.Hand()[2]}
		}})
		if err := ApplyEffect(g, DiscardAllButEach(players, Fixed(keep)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(a.Hand()) != min(max(keep, 0), 3) || len(b.Hand()) != min(max(keep, 0), 1) {
			t.Fatal("bounds or distinct mandatory fallback", keep)
		}
	}
	for _, effect := range []ResolvingEffect{
		DiscardAllButEach(nil, Fixed(1)), DiscardAllButEach(EachPlayer, nil),
		DiscardAllButEach(EachPlayer, LeastHandSize(nil)),
	} {
		if err := ValidateCardDefinition("Invalid", NewSorcery("Invalid", "", Spell(effect))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewLand("A"))
	if LeastHandSize(EachPlayer).resolveValue(&EffectContext{Game: g}) != 0 {
		t.Fatal("empty hand excluded from minimum")
	}
}

func TestHandCardinalityReplacementsAndVisits(t *testing.T) {
	for _, replacement := range []string{"library", "cancel", "return"} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		first, second := NewLand("First"), NewLand("Second")
		a.AddToHand(first)
		a.AddToHand(second)
		b.AddToHand(NewLand("B"))
		switch replacement {
		case "library":
			g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
		case "cancel":
			g.effects.AddReplacement(&preventRandomDiscardReplacement{card: first.ID()})
		case "return":
			g.effects.AddReplacement(&playerDiscardReturningReplacement{card: first.ID()})
		}
		if err := ApplyEffect(g, Sequence(DiscardAllButEach(EachPlayer, LeastHandSize(EachPlayer)), GainLifeFor(You, Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 21 || len(b.Hand()) != 1 || len(a.Hand()) != map[string]int{"library": 1, "cancel": 2, "return": 2}[replacement] {
			t.Fatal("replacement triggered new choices or followed returned visit", replacement)
		}
		if replacement == "library" && len(a.Library()) != 1 {
			t.Fatal("discard destination replacement lost")
		}
	}
}

func TestHandCardinalityReadsOnceAndWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToHand(NewLand("A"))
	b.AddToHand(NewLand("B"))
	calls := 0
	value := countedLifeAmount{&calls}
	if err := ApplyEffect(g, DiscardAllButEach(EachPlayer, value), uuid.Nil, a.PlayerID(), nil); err != nil || calls != 1 {
		t.Fatal("kept count was not read once", err, calls)
	}
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	players := OpponentsOf(ControllerOfWithLKI(This))
	body := DiscardAllButEach(EachPlayer, LeastHandSize(players))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	saved := g.CaptureDamageSource(source.ID())
	g.PhaseOut(source.ID())
	before := slices.Clone(b.Hand())
	for range 2 {
		got := g.WaitingDelayedPermanentSources()
		if len(got) != 1 || !got[0].SameObject(saved) || !slices.Equal(before, b.Hand()) {
			t.Fatal("waiting query lost source or executed body")
		}
	}
}

func TestHandCardinalityChoiceVisitsAndSnapshot(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := NewLand("First"), NewLand("Second")
	a.AddToHand(first)
	b.AddToHand(second)
	g.SetPlayerAt(0, &handCardinalityPlayer{BasePlayer: a, choose: func(_ int, _ string, _ GameReader) []Card {
		c, _ := b.RemoveFromHand(second.ID())
		b.AddToHand(c)
		b.AddToHand(NewLand("New"))
		return []Card{first}
	}})
	requested := 0
	g.SetPlayerAt(1, &handCardinalityPlayer{BasePlayer: b, choose: func(n int, _ string, _ GameReader) []Card {
		requested = n
		return []Card{second}
	}})
	if err := ApplyEffect(g, DiscardAllButEach(EachPlayer, Fixed(0)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if requested != 1 || len(a.Hand()) != 0 || len(b.Hand()) != 2 {
		t.Fatal("later choice read new candidates or moved a returned visit")
	}
}
