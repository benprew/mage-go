package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSavedHandRevealIdentityAndHistory(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	card := NewLand("Forest")
	a.AddToLibrary(card)
	drawn, revealed := BindCard("drawn"), BindCard("revealed")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := DrawFor(You, Fixed(1), DrawCardInto(drawn)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	reveal := RevealHandCardInto(FromCard(drawn), revealed)
	if err := reveal.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.PublicReveals()) != 1 || g.PublicReveals()[0].CardID != card.ID() || len(a.Hand()) != 1 {
		t.Fatal("reveal moved or lost the drawn card")
	}
	clone := g.Clone()
	a.RemoveFromHand(card.ID())
	a.AddToHand(card)
	card.AddType(core.TypeArtifact)
	branch := IfRevealedCardMatches(revealed, RevealedCardLacksType(core.TypeArtifact), GainLifeFor(You, Fixed(2)), GainLifeFor(You, Fixed(7)))
	before := a.Life()
	if err := branch.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.Life() != before+2 {
		t.Fatal("saved facts changed after departure")
	}
	if _, ok := FromCard(revealed).resolveCard(ctx); ok {
		t.Fatal("result followed a returned visit")
	}
	if err := reveal.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.PublicReveals()) != 1 || g.resolution.cardBindings[revealed].revealed != nil {
		t.Fatal("invalid reveal retained facts")
	}
	cc := &EffectContext{Game: clone, Controller: a.PlayerID()}
	if _, ok := FromCard(revealed).resolveCard(cc); !ok {
		t.Fatal("clone lost independent visit")
	}
	if err := branch.Apply(cc); err != nil {
		t.Fatal(err)
	}
	if clone.GetPlayer(a.PlayerID()).Life() != before+2 {
		t.Fatal("clone shared cleared result")
	}
}

func TestSavedHandRevealDeclarations(t *testing.T) {
	drawn, revealed := BindCard("drawn"), BindCard("revealed")
	valid := Sequence(DrawFor(You, Fixed(1), DrawCardInto(drawn)), RevealHandCardInto(FromCard(drawn), revealed), IfRevealedCardMatches(revealed, RevealedCardHasType(core.TypeLand), GainLifeFor(You, Fixed(1)), nil))
	for i, effect := range []Effect{valid, RevealHandCardInto(nil, revealed), RevealHandCardInto(FromCard(drawn), revealed), Sequence(DrawFor(You, Fixed(1), DrawCardInto(drawn)), RevealHandCardInto(FromCard(drawn), nil)), Sequence(DrawFor(You, Fixed(1), DrawCardInto(drawn)), RevealHandCardInto(FromCard(drawn), BindCard("")))} {
		err := ValidateCardDefinition("Definition", NewInstant("Definition", "", Spell(effect)))
		if (err != nil) != (i != 0) {
			t.Fatalf("case %d: %v", i, err)
		}
	}
}

func TestSavedHandRevealReplacementDraw(t *testing.T) {
	for _, replacementDraw := range []bool{false, true} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		card := NewLand("Forest")
		a.AddToLibrary(card)
		g.effects.AddReplacement(&savedRevealDrawReplacement{player: a.PlayerID(), draw: replacementDraw})
		drawn, revealed := BindCard("drawn"), BindCard("revealed")
		ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
		effect := Sequence(DrawFor(You, Fixed(1), DrawCardInto(drawn)), RevealHandCardInto(FromCard(drawn), revealed), DiscardCardFromHandFor(You, FromCard(drawn)))
		if err := effect.Apply(ctx); err != nil {
			t.Fatal(err)
		}
		want := 0
		if replacementDraw {
			want = 1
		}
		if len(a.Hand()) != want || len(g.PublicReveals()) != 0 || len(a.Graveyard()) != 0 || g.resolution.cardBindings[revealed].revealed != nil {
			t.Fatal("original draw captured replacement output")
		}
	}
}

type savedRevealDrawReplacement struct {
	replacementBase
	player uuid.UUID
	draw   bool
}

func (r *savedRevealDrawReplacement) Matches(a Action, _ GameReader) bool {
	d, ok := a.(*DrawCardAction)
	return ok && d.PlayerID() == r.player
}
func (r *savedRevealDrawReplacement) Replace(_ Action, g *Game) Action {
	if r.draw {
		g.drawCardRaw(g.GetPlayer(r.player))
	}
	return nil
}
func (*savedRevealDrawReplacement) IsActive(GameReader) bool   { return true }
func (r *savedRevealDrawReplacement) Clone() ReplacementEffect { cloned := *r; return &cloned }

func TestSavedHandRevealNestedFrames(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewLand("First"), NewArtifact("Second", "")
	a.AddToLibrary(first)
	a.AddToLibrary(second)
	drawn, revealed := BindCard("drawn"), BindCard("revealed")
	effect := Sequence(DrawFor(You, Fixed(1), DrawCardInto(drawn)), RevealHandCardInto(FromCard(drawn), revealed))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if card, ok := FromCard(revealed).resolveCard(ctx); !ok || card.ID() != first.ID() {
		t.Fatal("nested result replaced outer result")
	}
	if len(g.PublicReveals()) != 2 || g.PublicReveals()[1].CardID != second.ID() {
		t.Fatal("nested reveal did not execute")
	}
}
