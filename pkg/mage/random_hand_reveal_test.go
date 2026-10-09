package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRandomHandRevealIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	for _, card := range []Card{first, second} {
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
	}
	binding := BindCard("revealed")
	reveal := RevealRandomHandCardInto(You, binding)
	g.SetRandomResults([]int{1, 7})
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("result exists before reveal")
	}
	if err := reveal.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	card, ok := FromCard(binding).resolveCard(ctx)
	if !ok || card.ID() != second.ID() || len(a.Hand()) != 2 || len(g.PublicReveals()) != 1 || g.RandIntn(10) != 7 {
		t.Fatal("reveal identity, movement, visibility or randomness")
	}
	clone := g.Clone()
	if err := DiscardCardFromHandFor(You, FromCard(binding)).Apply(&EffectContext{Game: g, Controller: b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 2 || len(b.Graveyard()) != 0 {
		t.Fatal("discard ignored the explicit player")
	}
	a.RemoveFromHand(second.ID())
	a.AddToHand(second)
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("result followed a returned card")
	}
	if err := DiscardCardFromHandFor(You, FromCard(binding)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.Hand()) != 2 {
		t.Fatal("discard moved the returned visit")
	}
	cctx := &EffectContext{Game: clone, Controller: a.PlayerID()}
	if err := DiscardCardFromHandFor(You, FromCard(binding)).Apply(cctx); err != nil {
		t.Fatal(err)
	}
	if len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 || len(a.Hand()) != 2 {
		t.Fatal("clone lost the saved visit or changed the original")
	}
	reveals := clone.PublicReveals()
	reveals[0].Card.SetOwner(uuid.Nil)
	if g.PublicReveals()[0].Name != "Second" || clone.PublicReveals()[0].Card.Owner() != a.PlayerID() {
		t.Fatal("reveal history shares mutable cards")
	}
	a.RemoveFromHand(first.ID())
	a.RemoveFromHand(second.ID())
	g.SetRandomResults([]int{8})
	if err := reveal.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(binding).resolveCard(ctx); ok || g.RandIntn(10) != 8 || len(g.PublicReveals()) != 1 {
		t.Fatal("empty reveal did not clear result or consumed randomness")
	}
}

func TestRandomHandRevealDeclarations(t *testing.T) {
	binding := BindCard("reveal")
	for _, effect := range []Effect{
		RevealRandomHandCardInto(nil, binding), RevealRandomHandCardInto(You, nil),
		RevealRandomHandCardInto(You, BindCard("")),
		DiscardCardFromHandFor(nil, FromCard(binding)),
		DiscardCardFromHandFor(You, nil),
		Sequence(DiscardCardFromHandFor(You, FromCard(binding)), RevealRandomHandCardInto(You, binding)),
	} {
		card := NewInstant("Invalid", "", Spell(effect))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	card := NewInstant("Valid", "", Spell(Sequence(RevealRandomHandCardInto(You, binding), DiscardCardFromHandFor(You, FromCard(binding)))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestRandomHandRevealFrames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	for _, card := range []Card{first, second} {
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
	}
	binding := BindCard("reveal")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	g.SetRandomResults([]int{0, 1, 1})
	reveal := RevealRandomHandCardInto(You, binding)
	if err := reveal.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, reveal, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(NewArtifact("Member", ""), a.PlayerID())
	if err := ForEach(MatchingPermanents(AnyPermanent), BindPermanent("member"), reveal).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	card, ok := FromCard(binding).resolveCard(ctx)
	if !ok || card.ID() != first.ID() {
		t.Fatal("nested or loop result escaped its frame")
	}
	g.resolution.Clear()
	if _, ok := FromCard(binding).resolveCard(ctx); ok {
		t.Fatal("result survived frame cleanup")
	}
}

func TestRandomHandRevealReplacements(t *testing.T) {
	for _, mode := range []string{"normal", "prevent", "library"} {
		t.Run(mode, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first, second := NewArtifact("First", ""), NewArtifact("Second", "")
			for _, card := range []Card{first, second} {
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
			}
			if mode != "normal" {
				g.effects.AddReplacement(&handOperationReplacement{card: second.ID(), prevent: mode == "prevent"})
			}
			seen := 0
			watcher := NewTriggered(core.EvtDiscard, false).SetCondition(func(_ *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				seen++
				return false
			})
			g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(watcher)), a.PlayerID())
			binding := BindCard("reveal")
			g.SetRandomResults([]int{1, 8})
			if err := ApplyEffect(g, Sequence(RevealRandomHandCardInto(You, binding), DiscardCardFromHandFor(You, FromCard(binding))), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if len(g.PublicReveals()) != 1 || g.PublicReveals()[0].CardID != second.ID() || g.RandIntn(10) != 8 {
				t.Fatal("replacement changed reveal or used another random result")
			}
			switch mode {
			case "normal":
				if len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != second.ID() || seen != 1 {
					t.Fatal("discard or event missing")
				}
			case "library":
				if len(a.Library()) != 1 || a.Library()[0].ID() != second.ID() || len(a.Graveyard()) != 0 || seen != 1 {
					t.Fatal("discard redirection or event missing")
				}
			case "prevent":
				if len(a.Hand()) != 2 || len(a.Graveyard()) != 0 || seen != 0 {
					t.Fatal("discard prevention failed")
				}
			}
		})
	}
}

func TestRandomHandRevealAbsentAndWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Held", "")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	binding := BindCard("reveal")
	if err := RevealRandomHandCardInto(You, binding).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.SetRandomResults([]int{9})
	if err := RevealRandomHandCardInto(FromPlayer(BindPlayer("absent")), binding).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(binding).resolveCard(ctx); ok || g.RandIntn(10) != 9 || len(g.PublicReveals()) != 1 {
		t.Fatal("absent player retained result or used randomness")
	}
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	body := Sequence(RevealRandomHandCardInto(ControllerOfWithLKI(This), binding), DiscardCardFromHandFor(ControllerOfWithLKI(This), FromCard(binding)))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.SetRandomResults([]int{7})
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || len(g.PublicReveals()) != 1 || len(a.Hand()) != 1 {
			t.Fatal("waiting query lost source or ran the body")
		}
	}
	if g.RandIntn(10) != 7 {
		t.Fatal("waiting query used randomness")
	}
}

type revealPlayerWithoutVisits struct{ Player }

func TestRandomHandRevealMissingIdentity(t *testing.T) {
	base, b := NewBasePlayer("A"), NewBasePlayer("B")
	a := &revealPlayerWithoutVisits{Player: base}
	g := NewGame(a, b)
	card := NewArtifact("Held", "")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := RevealRandomHandCardInto(You, BindCard("reveal")).Apply(ctx); err == nil {
		t.Fatal("missing hand identity silently skipped the reveal")
	}
	if len(g.PublicReveals()) != 0 || len(a.Hand()) != 1 {
		t.Fatal("unsupported identity published a reveal or moved the card")
	}
}
