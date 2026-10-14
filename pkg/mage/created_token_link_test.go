package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatedTokenLinkLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Maker", "")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: source.ID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
	link, other := NewCreatedTokenLink(), NewCreatedTokenLink()
	set := TokensCreatedWith(link, This)
	token := DefineToken("Member", TokenPT(1, 1), TokenTypes(core.TypeCreature))
	makeTokens := CreateTokenCopiesLinked(You, Fixed(2), token, link)
	var observed []int
	watcher := NewArtifact("Watcher", "", WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if p := g.FindPermanent(e.SourceID); e.ToZone == core.ZoneBattlefield && p != nil && p.IsToken {
			observed = append(observed, len(set.resolvePermanents(ctx)))
		}
		return false
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	if err := makeTokens.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(observed) != 2 || observed[0] != 2 || observed[1] != 2 {
		t.Fatalf("link not visible to entry events: %v", observed)
	}
	if err := CreateTokenCopiesLinked(You, Fixed(1), token, other).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(set.resolvePermanents(ctx)) != 2 || len(TokensCreatedWith(other, This).resolvePermanents(ctx)) != 1 {
		t.Fatal("declarations mixed")
	}
	if err := makeTokens.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := CreateTokenCopiesLinked(You, Fixed(0), token, link).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(set.resolvePermanents(ctx)) != 4 {
		t.Fatal("repeated or zero creation lost links")
	}
	members := set.resolvePermanents(ctx)
	g.PhaseOut(members[0].ID())
	if len(set.resolvePermanents(ctx)) != 3 {
		t.Fatal("phased token selected")
	}
	g.PhaseIn([]uuid.UUID{members[0].ID()})
	if len(set.resolvePermanents(ctx)) != 4 {
		t.Fatal("phase cycle lost link")
	}
	sourcePhase := g.PhaseOut(source.ID())
	if len(set.resolvePermanents(ctx)) != 4 {
		t.Fatal("source phasing removed links")
	}
	g.PhaseIn(sourcePhase)
	clone := g.Clone()
	cloneCtx := *ctx
	cloneCtx.Game = clone
	g.BouncePermanentToHand(source)
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, b.PlayerID())
	newCtx := &EffectContext{Game: g, Controller: b.PlayerID(), SourceID: returned.ID(), sourceCaptured: true, sourceIncarnation: returned.incarnationID}
	if len(set.resolvePermanents(newCtx)) != 0 || len(set.resolvePermanents(ctx)) != 4 {
		t.Fatal("source visits mixed")
	}
	if err := makeTokens.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(set.resolvePermanents(ctx)) != 6 || len(set.resolvePermanents(&cloneCtx)) != 4 {
		t.Fatal("departed creation or clone isolation")
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: returned.ID(), TargetID: members[0].ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if len(set.resolvePermanents(ctx)) != 6 {
		t.Fatalf("control changed link: %d tokens, %d linked", len(g.FilterBattlefield(IsToken)), len(set.resolvePermanents(ctx)))
	}
	g.BouncePermanentToHand(members[0])
	if len(set.resolvePermanents(ctx)) != 5 || len(set.resolvePermanents(&cloneCtx)) != 4 {
		t.Fatal("token departure or clone isolation")
	}
	copied := g.PutOnBattlefield(members[1].Card.Copy(), a.PlayerID())
	if copied == nil || len(set.resolvePermanents(ctx)) != 5 {
		t.Fatal("copy inherited creation link")
	}
}

func TestCreatedTokenLinkInvalid(t *testing.T) {
	token := DefineToken("Member", TokenPT(1, 1), TokenTypes(core.TypeCreature))
	for _, effect := range []ResolvingEffect{
		CreateTokenCopiesLinked(You, Fixed(1), token, nil),
		CreateTokenCopiesLinked(You, Fixed(1), token, &CreatedTokenLink{}),
		CreateTokenCopiesLinked(nil, Fixed(1), token, NewCreatedTokenLink()),
		CreateTokenCopiesLinked(You, nil, token, NewCreatedTokenLink()),
		CreateTokenCopiesLinked(You, Fixed(1), nil, NewCreatedTokenLink()),
		ExileEach(TokensCreatedWith(nil, This)),
		ExileEach(TokensCreatedWith(NewCreatedTokenLink(), nil)),
	} {
		card := NewArtifact("Bad", "", WithAbility(NewActivated(GenericCost(0), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid link declaration accepted")
		}
	}
}

func TestCreatedTokenLinkEntryProhibition(t *testing.T) {
	for _, planned := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
		g.PutOnBattlefield(NewEnchantment("Block", "", WithAbility(Static(PreventLandEntry()))), a.PlayerID())
		link := NewCreatedTokenLink()
		build := func() TokenDefinition { return DefineToken("Land", TokenTypes(core.TypeLand)) }
		var token TokenDefinition
		if planned {
			factory, err := NewTokenDefinitionFactory(build)
			if err != nil {
				t.Fatal(err)
			}
			token = factory()
		} else {
			token = build()
		}
		ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: source.ID()}
		if err := CreateTokenCopiesLinked(You, Fixed(2), token, link).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if len(TokensCreatedWith(link, This).resolvePermanents(ctx)) != 0 || len(g.FilterBattlefield(IsToken)) != 0 {
			t.Fatal("prohibited entry supplied a link")
		}
	}
}

func TestCreatedTokenLinkWaiting(t *testing.T) {
	for _, produce := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		card := NewArtifact("Source", "")
		card.SetOwner(a.PlayerID())
		source := g.PutOnBattlefield(card, a.PlayerID())
		link := NewCreatedTokenLink()
		token := DefineToken("Member", TokenPT(1, 1), TokenTypes(core.TypeCreature))
		create := CreateTokenCopiesLinked(You, Fixed(2), token, link)
		var body = ExileEach(TokensCreatedWith(link, This))
		if produce {
			body = create
		} else if err := ApplyEffect(g, create, source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		saved := g.CaptureDamageSource(source.ID())
		delay := AtNext(EachEndStep, body)
		if err := ValidateCardDefinition("Source", NewArtifact("Source", "", WithAbility(Activate(Costs(), delay)))); err != nil {
			t.Fatal(err)
		}
		if err := ApplyEffect(g, delay, source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		phased := g.PhaseOut(source.ID())
		for range 2 {
			got := g.WaitingDelayedPermanentSources()
			if len(got) != 1 || !got[0].SameObject(saved) {
				t.Fatal("waiting query lost exact source")
			}
		}
		g.PhaseIn(phased)
		g.BouncePermanentToHand(source)
		a.RemoveFromHand(card.ID())
		g.PutOnBattlefield(card, b.PlayerID())
		clone := g.Clone()
		for _, game := range []*Game{g, clone} {
			if len(game.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("departure lost waiting link")
			}
			game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			game.PutTriggersOnStack()
			game.ResolveStack()
			want := 0
			if produce {
				want = 2
			}
			if len(game.FilterBattlefield(IsToken)) != want {
				t.Fatal("delayed link body resolved against new visit")
			}
		}
	}
}
