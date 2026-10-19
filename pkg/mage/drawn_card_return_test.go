package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type drawnReturnPlayer struct {
	*BasePlayer
	pay               bool
	choices, payments int
	onChoose          func()
}

func (p *drawnReturnPlayer) ChooseCardFromHand(cards []Card, _ string, _ GameReader) Card {
	p.choices++
	if p.payments != 0 {
		panic("payment before all card choices")
	}
	if p.onChoose != nil {
		p.onChoose()
	}
	return cards[len(cards)-1]
}
func (p *drawnReturnPlayer) ChooseMayAbility(string) bool { p.payments++; return p.pay }

func TestDrawnCardReturnHistory(t *testing.T) {
	for _, state := range []string{"drawn", "returned", "reset", "clone", "added", "other-player"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewLand("Card")
			switch state {
			case "added":
				a.AddToHand(c)
			case "other-player":
				b.AddToLibrary(c)
				g.PlayerDrawCard(b)
			default:
				a.AddToLibrary(c)
				g.PlayerDrawCard(a)
			}
			if state == "returned" {
				a.RemoveFromHand(c.ID())
				a.AddToHand(c)
			}
			if state == "reset" {
				g.trackers.Turn.ResetForNewTurn(a.PlayerID())
			}
			if state == "clone" {
				g = g.Clone()
			}
			if err := ApplyEffect(g, PayLifeOrReturnDrawnCards(You, 2, 4), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			want := 0
			if state == "drawn" || state == "clone" {
				want = 1
			}
			if len(g.GetPlayer(a.PlayerID()).Library()) != want {
				t.Fatal("wrong draw visit", state)
			}
			if state == "clone" && len(a.Hand()) != 1 {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestDrawnCardReturnChoicesAndLife(t *testing.T) {
	for _, life := range []int{3, 4, 7, 8, 20} {
		t.Run(fmt.Sprint(life), func(t *testing.T) {
			a := &drawnReturnPlayer{BasePlayer: NewBasePlayer("A"), pay: true}
			g := NewGame(a, NewBasePlayer("B"))
			a.SetLife(life)
			a.AddToHand(NewLand("Old hand"))
			for range 3 {
				a.AddToLibrary(NewLand("Draw"))
				g.PlayerDrawCard(a)
			}
			if err := ApplyEffect(g, PayLifeOrReturnDrawnCards(You, 2, 4), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			paid := min(2, life/4)
			if a.choices != 2 || a.payments != paid || a.Life() != life-4*paid || len(a.Library()) != 2-paid || len(a.Hand()) != 2+paid {
				t.Fatalf("choices %d payments %d life %d hand %d library %d", a.choices, a.payments, a.Life(), len(a.Hand()), len(a.Library()))
			}
		})
	}
}

func TestDrawnCardReturnReplacements(t *testing.T) {
	for _, state := range []string{"draw-cancel", "generated-draw", "move-cancel", "move-exile", "move-return", "cant-pay"} {
		t.Run(state, func(t *testing.T) {
			a := &drawnReturnPlayer{BasePlayer: NewBasePlayer("A"), pay: state == "cant-pay"}
			g := NewGame(a, NewBasePlayer("B"))
			c := NewLand("Draw")
			a.AddToLibrary(c)
			if state == "draw-cancel" || state == "generated-draw" {
				g.effects.AddReplacement(&testPrioritizedReplacement{replacementBase: replacementBase{sourceID: uuid.New(), priority: ReplacementPriorityDefault}, matchFunc: func(action Action) bool { _, ok := action.(*DrawCardAction); return ok }, replaceFunc: func(_ Action, game *Game) Action {
					if state == "generated-draw" {
						game.drawCardRaw(a)
					}
					return nil
				}})
			} else if state != "cant-pay" {
				g.effects.AddReplacement(&testPrioritizedReplacement{replacementBase: replacementBase{sourceID: uuid.New(), priority: ReplacementPriorityDefault}, matchFunc: func(action Action) bool { _, ok := action.(*CardZoneMoveAction); return ok }, replaceFunc: func(action Action, _ *Game) Action {
					move := action.(*CardZoneMoveAction)
					if move.FromZone() != core.ZoneHand || move.Destination() != core.ZoneLibrary {
						t.Fatal("lost movement facts")
					}
					if state == "move-cancel" {
						return nil
					}
					if state == "move-return" {
						a.RemoveFromHand(c.ID())
						a.AddToHand(c)
						return move
					}
					return move.WithExile(uuid.Nil)
				}})
			}
			g.PlayerDrawCard(a)
			if state == "cant-pay" {
				g.effects.Rules.AddCantLoseLife(a.PlayerID())
			}
			if err := ApplyEffect(g, PayLifeOrReturnDrawnCards(You, 2, 4), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			wantPayments := 1
			if state == "draw-cancel" || state == "cant-pay" {
				wantPayments = 0
			}
			if a.Life() != 20 || a.payments != wantPayments {
				t.Fatal("unexpected life payment")
			}
			switch state {
			case "draw-cancel":
				if a.choices != 0 || len(a.Hand()) != 0 {
					t.Fatal("replaced draw counted")
				}
			case "generated-draw", "cant-pay":
				if a.choices != 1 || len(a.Library()) != 1 || len(a.Hand()) != 0 {
					t.Fatal("actual draw not returned")
				}
			case "move-cancel", "move-return":
				if len(a.Hand()) != 1 || len(a.Library()) != 0 {
					t.Fatal("replacement ignored")
				}
			case "move-exile":
				if g.FindExiledCard(c.ID()) == nil || len(a.Hand()) != 0 {
					t.Fatal("redirect ignored")
				}
			}
		})
	}
}

func TestDrawnCardReturnOptionalContinuation(t *testing.T) {
	for _, accept := range []bool{false, true} {
		a := &drawnReturnPlayer{BasePlayer: NewBasePlayer("A"), pay: accept}
		g := NewGame(a, NewBasePlayer("B"))
		source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
		g.AddEmptyLibraryDrawReplacement(source.ID(), a.PlayerID(), func(*Game, uuid.UUID) {})
		if err := ApplyEffect(g, MayDrawCardsThen(You, 2, GainLifeFor(You, Fixed(3))), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := 20
		if accept {
			want = 23
		}
		if a.Life() != want || a.payments != 1 {
			t.Fatal("continuation used successful draw count")
		}
	}
}

func TestDrawnCardReturnChoiceVisit(t *testing.T) {
	a := &drawnReturnPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	c := NewLand("Returned")
	a.AddToLibrary(c)
	g.PlayerDrawCard(a)
	a.onChoose = func() { a.RemoveFromHand(c.ID()); a.AddToHand(c) }
	if err := ApplyEffect(g, PayLifeOrReturnDrawnCards(You, 2, 4), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 1 || a.payments != 0 {
		t.Fatal("returned visit supplied an old choice")
	}
}

func TestDrawnCardReturnDefinitions(t *testing.T) {
	for _, effect := range []ResolvingEffect{PayLifeOrReturnDrawnCards(nil, 2, 4), PayLifeOrReturnDrawnCards(You, -1, 4), PayLifeOrReturnDrawnCards(You, 2, -1), MayDrawCardsThen(nil, 2, GainLifeFor(You, Fixed(1))), MayDrawCardsThen(You, -1, GainLifeFor(You, Fixed(1))), MayDrawCardsThen(You, 2, nil)} {
		if ValidateCardDefinition("Bad", NewSorcery("Bad", "", Spell(effect))) == nil {
			t.Fatal("invalid definition accepted")
		}
	}
}

func TestDrawnCardReturnOrderAndEvents(t *testing.T) {
	a := &drawnReturnPlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	first, second, top := NewLand("First"), NewLand("Second"), NewLand("Top")
	a.SetLibrary([]Card{first, second, top})
	g.PlayerDrawCard(a)
	g.PlayerDrawCard(a)
	moves, discards := 0, 0
	watch := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if e.FromZone == core.ZoneHand && e.ToZone == core.ZoneLibrary {
			moves++
			if a.Library()[0].ID() != e.SourceID || g.handVisit(e.SourceID) != uuid.Nil {
				t.Error("zone event preceded the actual library return")
			}
		}
		return false
	})
	discard := NewTriggered(core.EvtDiscard, false).SetCondition(func(*core.GameEvent, GameReader, uuid.UUID, uuid.UUID) bool {
		discards++
		return false
	})
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(watch), WithAbility(discard)), b.PlayerID())
	if err := ApplyEffect(g, PayLifeOrReturnDrawnCards(You, 2, 4), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.choices != 2 || a.payments != 2 || moves != 2 || discards != 0 || len(g.PublicReveals()) != 0 || len(a.Hand()) != 0 || len(a.Library()) != 3 {
		t.Fatal("choice, payment or movement events")
	}
	if a.Library()[0].ID() != first.ID() || a.Library()[1].ID() != second.ID() || a.Library()[2].ID() != top.ID() {
		t.Fatal("sequential put-back order")
	}
}
