package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLinkedExileIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	sourceCard := NewArtifact("Source", "")
	sourceCard.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: source.ID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	link, other := NewExileLink(), NewExileLink()
	first, second := NewLand("First"), NewLand("Second")
	first.SetOwner(b.PlayerID())
	second.SetOwner(a.PlayerID())
	a.SetLibrary([]Card{first, second})
	if err := ExileLibraryTopLinked(You, link, true).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.CardsPutIntoExileThisTurn() != 1 {
		t.Fatal("library exile counted twice")
	}
	if err := ExileLibraryTopLinked(You, other, false).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	set := CardsExiledWith(link, This)
	if got := set.resolveExiledCards(ctx); len(got) != 1 || got[0].id != first.ID() || got[0].visit != g.FindExiledCard(first.ID()).Visit || got[0].zone != core.ZoneExile {
		t.Fatal("links mixed")
	}
	ec := g.FindExiledCard(first.ID())
	if ec == nil || ec.VisibleTo(a.PlayerID()) || ec.VisibleTo(b.PlayerID()) || ec.Visit == uuid.Nil || ec.ExiledBy != source.ID() {
		t.Fatal("face-down identity or visibility")
	}
	clone := g.Clone()
	cloneCtx := *ctx
	cloneCtx.Game = clone
	g.DestroyPermanent(source)
	g.zones.ClearEndOfTurn()
	a.RemoveFromGraveyard(source.ID())
	returned := g.PutOnBattlefield(source.Card, b.PlayerID())
	if got := set.resolveExiledCards(ctx); len(got) != 1 {
		t.Fatal("source departure lost link")
	}
	newCtx := &EffectContext{Game: g, Controller: b.PlayerID(), SourceID: returned.ID(), sourceCaptured: true, sourceIncarnation: returned.incarnationID}
	if len(set.resolveExiledCards(newCtx)) != 0 {
		t.Fatal("returned source inherited link")
	}
	card, ok := g.RemoveFromExile(first.ID())
	if !ok {
		t.Fatal("missing exile")
	}
	g.ExileCardFaceDown(card, source.ID())
	if len(set.resolveExiledCards(ctx)) != 0 || len(set.resolveExiledCards(&cloneCtx)) != 1 {
		t.Fatal("returned exile or clone crossed link")
	}
	if err := ExileLibraryTopLinked(You, link, true).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.Library()) != 0 || a.drewFromEmpty {
		t.Fatal("empty exile drew a card")
	}
}

func TestLinkedExilePermanentAndEvents(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	sourceCard := NewArtifact("Source", "")
	sourceCard.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	link := NewExileLink()
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: source.ID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	target := g.PutOnBattlefield(NewCreature("Target", "", 2, 2), b.PlayerID())
	bound := BindPermanent("subject")
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{bound: {target.ID(), target.incarnationID}}
	seen := 0
	observer := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if e.SourceID == target.ID() {
			seen++
			if len(CardsExiledWith(link, This).resolveExiledCards(ctx)) != 1 || g.FindPermanent(target.ID()) != nil {
				t.Error("event preceded linked move")
			}
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer)), a.PlayerID())
	g.PhaseOut(target.ID())
	if err := ExilePermanentLinked(From(bound), link).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if seen != 0 || len(g.GetExile()) != 0 {
		t.Fatal("phased target moved")
	}
	g.PhaseIn([]uuid.UUID{target.ID()})
	g.PhaseOut(source.ID())
	if err := ExilePermanentLinked(From(bound), link).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if seen != 1 || len(CardsExiledWith(link, This).resolveExiledCards(ctx)) != 1 {
		t.Fatal("phased source lost announced link")
	}
}

func TestLinkedExileDeclarations(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		ExileLibraryTopLinked(nil, NewExileLink(), true),
		ExileLibraryTopLinked(You, nil, true),
		ExilePermanentLinked(nil, NewExileLink()),
		ExilePermanentLinked(This, nil),
		GainLifeFor(You, CountExiledCards(CardsExiledWith(nil, This))),
		GainLifeFor(You, CountExiledCards(CardsExiledWith(NewExileLink(), nil))),
		GainLifeFor(You, CountExiledCards(nil)),
		ExileLibraryTopLinked(You, &ExileLink{}, true),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(ManaPayment("")), effect)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid linked declaration accepted")
		}
	}
}

func TestLinkedExileWaiting(t *testing.T) {
	for _, body := range []ResolvingEffect{
		ExileLibraryTopLinked(You, NewExileLink(), true),
		ExilePermanentLinked(This, NewExileLink()),
		GainLifeFor(You, CountExiledCards(CardsExiledWith(NewExileLink(), This))),
	} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		card := NewArtifact("Source", "")
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		saved := g.CaptureDamageSource(p.ID())
		top := NewLand("Top")
		top.SetOwner(a.PlayerID())
		a.AddToLibrary(top)
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.BouncePermanentToHand(p)
		a.RemoveFromHand(card.ID())
		g.PutOnBattlefield(card, b.PlayerID())
		for range 2 {
			got := g.WaitingDelayedPermanentSources()
			if len(got) != 1 || !got[0].SameObject(saved) || len(g.GetExile()) != 0 || len(a.Library()) != 1 {
				t.Fatal("waiting query lost source or executed linked exile")
			}
		}
	}
}

func TestLinkedExileRepeatedAndTokens(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: source.ID()}
	link := NewExileLink()
	set := CardsExiledWith(link, This)
	for range 2 {
		card := NewLand("Top")
		card.SetOwner(a.PlayerID())
		a.AddToLibrary(card)
		if err := ExileLibraryTopLinked(You, link, false).Apply(ctx); err != nil {
			t.Fatal(err)
		}
	}
	foreign := NewLand("Unlinked")
	g.ExileCard(foreign, source.ID())
	if len(set.resolveExiledCards(ctx)) != 2 || len(CardsExiledWith(NewExileLink(), This).resolveExiledCards(ctx)) != 0 {
		t.Fatal("repeat or unrelated exile mixed")
	}
	clone := g.Clone()
	cloneCtx := *ctx
	cloneCtx.Game = clone
	clone.FindExiledCard(set.resolveExiledCards(&cloneCtx)[0].id).Card.AddType(core.TypeCreature)
	if g.FindExiledCard(set.resolveExiledCards(ctx)[0].id).Card.HasType(core.TypeCreature) {
		t.Fatal("clone shared card characteristics")
	}
	token := g.PutOnBattlefield(NewCreature("Token", "", 1, 1), a.PlayerID())
	token.IsToken = true
	bound := BindPermanent("token")
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{bound: {token.ID(), token.incarnationID}}
	if err := ExilePermanentLinked(From(bound), link).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(token.ID()) != nil || g.FindExiledCard(token.ID()) != nil || len(set.resolveExiledCards(ctx)) != 2 {
		t.Fatal("token supplied linked card")
	}
}
