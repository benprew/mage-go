package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestGroupedExileIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Host", "", 2, 2)
	card.SetOwner(b.PlayerID())
	host := g.PutOnBattlefield(card, a.PlayerID())
	auraCard := NewAura("Aura", "")
	auraCard.SetOwner(a.PlayerID())
	aura := g.PutOnBattlefield(auraCard, a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	first, other := BindCards("first"), BindCards("other")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: host.ID(), sourceCaptured: true, sourceIncarnation: host.incarnationID}
	effect := ExilePermanentGroupsInto(PermanentExileGroup{SingletonPermanent(This), first}, PermanentExileGroup{AttachmentsOn(SingletonPermanent(This)), other})
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.GetExile()) != 2 || len(ExiledCardsFrom(first).resolveExiledCards(ctx)) != 1 || len(ExiledCardsFrom(other).resolveExiledCards(ctx)) != 1 {
		t.Fatal("groups did not select before movement")
	}
	if c, ok := SingleExiledCardFrom(first).resolveCard(ctx); !ok || c.ID() != card.ID() || c.Owner() != b.PlayerID() {
		t.Fatal("single result lost identity or owner")
	}
	clone := g.Clone()
	cloneCtx := *ctx
	cloneCtx.Game = clone
	g.RemoveFromExile(card.ID())
	g.ExileCard(card, host.ID())
	if _, ok := SingleExiledCardFrom(first).resolveCard(ctx); ok {
		t.Fatal("result followed a new exile visit")
	}
	if len(ExiledCardsFrom(first).resolveExiledCards(&cloneCtx)) != 1 {
		t.Fatal("clone lost saved visit")
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(ExiledCardsFrom(first).resolveExiledCards(ctx)) != 0 || len(ExiledCardsFrom(other).resolveExiledCards(ctx)) != 0 || len(ExiledCardsFrom(other).resolveExiledCards(&cloneCtx)) != 1 {
		t.Fatal("empty production did not clear isolated results")
	}
}

func TestGroupedExileDeclarations(t *testing.T) {
	saved := BindCards("saved")
	for _, effect := range []ResolvingEffect{
		ExilePermanentGroupsInto(),
		ExilePermanentGroupsInto(PermanentExileGroup{nil, saved}),
		ExilePermanentGroupsInto(PermanentExileGroup{SingletonPermanent(This), nil}),
		ExilePermanentGroupsInto(PermanentExileGroup{SingletonPermanent(This), BindCards("")}),
		ExilePermanentGroupsInto(PermanentExileGroup{SingletonPermanent(This), saved}, PermanentExileGroup{MatchingPermanents(AnyPermanent), saved}),
		PutExiledCardsInOwnersGraveyards(ExiledCardsFrom(saved)),
		Sequence(PutExiledCardsInOwnersGraveyards(ExiledCardsFrom(saved)), ExilePermanentGroupsInto(PermanentExileGroup{SingletonPermanent(This), saved})),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	valid := NewArtifact("Valid", "", WithAbility(Activate(Costs(), Sequence(ExilePermanentGroupsInto(PermanentExileGroup{SingletonPermanent(This), saved}), PutExiledCardsInOwnersGraveyards(ExiledCardsFrom(saved))))))
	if err := ValidateCardDefinition("Valid", valid); err != nil {
		t.Fatal(err)
	}
}

func TestGroupedExileWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	reads := 0
	subjects := FilterPermanentSet(SingletonPermanent(This), NewPermanentFilter("query must not select", func(*Permanent, *Game) bool { reads++; return false }))
	effect := ExilePermanentGroupsInto(PermanentExileGroup{subjects, BindCards("result")})
	if err := ApplyEffect(g, AtNext(EachEndStep, effect), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	saved := g.CaptureDamageSource(p.ID())
	g.BouncePermanentToHand(p)
	for range 2 {
		if got := g.WaitingDelayedPermanentSources(); len(got) != 1 || !got[0].SameObject(saved) || reads != 0 || len(g.GetExile()) != 0 {
			t.Fatal("waiting query executed or lost source")
		}
	}
	clone := g.Clone()
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	if len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("clone shared delayed lifetime")
	}
}

func TestGroupedExileOverlapAndLegality(t *testing.T) {
	for _, state := range []string{"present", "phased", "returned", "token"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 2, 2)
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			saved := BindPermanent("subject")
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			g.resolution.permanentBindings = make(map[*PermanentBinding]capturedPermanent)
			g.resolution.permanentBindings[saved] = capturedPermanent{id: p.ID(), incarnation: p.incarnationID}
			switch state {
			case "phased":
				g.PhaseOut(p.ID())
			case "returned":
				g.BouncePermanentToHand(p)
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "token":
				p.IsToken = true
			}
			first, second := BindCards("first"), BindCards("second")
			subjects := SingletonPermanent(From(saved))
			effect := ExilePermanentGroupsInto(PermanentExileGroup{UnionPermanents(subjects, subjects), first}, PermanentExileGroup{subjects, second})
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 0
			if state == "present" {
				want = 1
			}
			if len(g.GetExile()) != want || len(ExiledCardsFrom(first).resolveExiledCards(ctx)) != want || len(ExiledCardsFrom(second).resolveExiledCards(ctx)) != want {
				t.Fatal("overlap, token or incarnation legality failed")
			}
			moves := want
			if state == "token" {
				moves = 1
			}
			if g.CardsPutIntoExileThisTurn() != moves {
				t.Fatal("overlap counted a move twice")
			}
			if state == "present" && g.resolution.cardsBindings[first][0] != g.resolution.cardsBindings[second][0] {
				t.Fatal("overlap did not retain the same visit")
			}
		})
	}
}

func TestGroupedExileSingularDoesNotPromote(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	cards := []Card{NewArtifact("First", ""), NewArtifact("Second", "")}
	for _, card := range cards {
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
	}
	result := BindCards("both")
	if err := ExilePermanentGroupsInto(PermanentExileGroup{MatchingPermanents(AnyPermanent), result}).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromExile(cards[0].ID())
	if len(ExiledCardsFrom(result).resolveExiledCards(ctx)) != 1 {
		t.Fatal("remaining group visit lost")
	}
	if _, ok := SingleExiledCardFrom(result).resolveCard(ctx); ok {
		t.Fatal("a plural result became a singular reference")
	}
}

func TestGroupedExileSimultaneousEvents(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := BindCards("first"), BindCards("second")
	checks := 0
	trigger := NewTriggered(core.EvtZoneChange, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *core.GameEvent, reader GameReader, source, controller uuid.UUID) bool {
		if evt.FromZone != core.ZoneBattlefield || evt.ToZone != core.ZoneExile {
			return false
		}
		checks++
		if len(g.GetExile()) != 2 || len(g.resolution.cardsBindings[first]) != 1 || len(g.resolution.cardsBindings[second]) != 1 {
			t.Error("event observed incomplete batch or results")
		}
		return true
	})
	watcherCard := NewCreature("Watcher", "", 2, 2, WithAbility(trigger))
	watcherCard.SetOwner(a.PlayerID())
	watcher := g.PutOnBattlefield(watcherCard, a.PlayerID())
	otherCard := NewCreature("Other", "", 2, 2)
	otherCard.SetOwner(b.PlayerID())
	other := g.PutOnBattlefield(otherCard, b.PlayerID())
	s1, s2 := BindPermanent("watcher"), BindPermanent("other")
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{s1: {id: watcher.ID(), incarnation: watcher.incarnationID}, s2: {id: other.ID(), incarnation: other.incarnationID}}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := ExilePermanentGroupsInto(PermanentExileGroup{SingletonPermanent(From(s1)), first}, PermanentExileGroup{SingletonPermanent(From(s2)), second}).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if checks != 2 || len(g.triggers.Pending()) != 2 {
		t.Fatalf("departure LKI lost simultaneous watcher: checks=%d pending=%d", checks, len(g.triggers.Pending()))
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("departure triggers did not resolve")
	}
}

func TestGroupedExileFrames(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Source", "")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	result := BindCards("result")
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	producer := ExilePermanentGroupsInto(PermanentExileGroup{SingletonPermanent(This), result})
	if err := producer.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, producer, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(ExiledCardsFrom(result).resolveExiledCards(ctx)) != 1 {
		t.Fatal("nested frame cleared outer result")
	}
	if err := AtNext(EachEndStep, ReturnExiledCardsToOwnersHands(ExiledCardsFrom(result))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.resolution.Clear()
	moved, _ := g.RemoveFromExile(card.ID())
	g.ExileCard(moved, p.ID())
	for _, game := range []*Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.ResolveStack()
	}
	if len(a.Hand()) != 0 || len(g.GetExile()) != 1 || len(clone.GetExile()) != 0 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 {
		t.Fatal("delayed frame or clone followed a new visit")
	}
}
