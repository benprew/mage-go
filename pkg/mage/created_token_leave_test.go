package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCreatedTokenLeaveLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	link, other := NewCreatedTokenLink(), NewCreatedTokenLink()
	maker := func() *Permanent {
		card := NewArtifact("Maker", "", WithAbility(WhenEvent(CreatedTokenLeaves(link), GainLifeFor(You, Fixed(1)))))
		card.SetOwner(a.PlayerID())
		return g.PutOnBattlefield(card, a.PlayerID())
	}
	first, second := maker(), maker()
	create := func(source *Permanent, declaration *CreatedTokenLink, count int) []*Permanent {
		ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: source.ID(), sourceCaptured: true, sourceIncarnation: source.incarnationID}
		err := CreateTokenCopiesLinked(You, Fixed(count), DefineToken("Member", TokenPT(1, 1), TokenTypes(core.TypeCreature)), declaration).Apply(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return TokensCreatedWith(declaration, This).resolvePermanents(ctx)
	}
	members := create(first, link, 3)
	unrelated := create(second, link, 1)
	otherMembers := create(first, other, 1)
	g.BouncePermanentToHand(otherMembers[0])
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("another declaration matched")
	}
	phase := g.PhaseOut(members[0].ID())
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("phasing caused departure")
	}
	g.PhaseIn(phase)
	g.AddControlEffect(ControlEffectSpec{SourceID: first.ID(), TargetID: members[0].ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.BouncePermanentToHand(members[0])
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 21 || game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatal("departure lost event controller or cloned pending trigger")
		}
	}
	g.BouncePermanentsToHand([]*Permanent{first, members[1], members[2]})
	a.RemoveFromHand(first.ID())
	returned := g.PutOnBattlefield(first.Card, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 20 {
		t.Fatal("simultaneous departure lost source visit")
	}
	g.BouncePermanentToHand(unrelated[0])
	g.ResolveStack()
	if a.Life() != 24 || b.Life() != 20 {
		t.Fatal("source instances mixed")
	}
	stale := create(returned, link, 1)[0]
	g.BouncePermanentToHand(returned)
	a.RemoveFromHand(returned.ID())
	newSource := g.PutOnBattlefield(returned.Card, a.PlayerID())
	g.BouncePermanentToHand(stale)
	g.ResolveStack()
	if a.Life() != 24 || b.Life() != 20 {
		t.Fatal("returned source inherited old members")
	}
	fresh := create(newSource, link, 3)
	phase = g.PhaseOut(newSource.ID())
	g.BouncePermanentToHand(fresh[0])
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatal("phased source listened")
	}
	g.PhaseIn(phase)
	removal := RemoveAllAbilities(newSource.ID(), core.EndOfTurn)
	g.AddContinuousEffect(removal)
	g.BouncePermanentToHand(fresh[1])
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatal("removed ability listened")
	}
	g.effects.RemoveEndOfTurn()
	g.effects.Apply(g)
	g.BouncePermanentToHand(fresh[2])
	g.ResolveStack()
	if a.Life() != 25 {
		t.Fatal("restored ability lost declaration")
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 21 {
		t.Fatal("clone mutated")
	}
}

func TestCreatedTokenLeaveInvalid(t *testing.T) {
	for _, link := range []*CreatedTokenLink{nil, {}} {
		card := NewArtifact("Bad", "", WithAbility(WhenEvent(CreatedTokenLeaves(link), GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid link")
		}
	}
}

func TestCreatedTokenLeaveDestinations(t *testing.T) {
	for _, destination := range []string{"destroy", "exile", "sacrifice", "bounce", "replacement", "regenerate"} {
		t.Run(destination, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			link := NewCreatedTokenLink()
			source := g.PutOnBattlefield(NewArtifact("Maker", "", WithAbility(WhenEvent(CreatedTokenLeaves(link), GainLifeFor(You, Fixed(1))))), a.PlayerID())
			ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: source.ID()}
			if err := CreateTokenCopiesLinked(You, Fixed(1), DefineToken("Member", TokenPT(1, 1), TokenTypes(core.TypeCreature)), link).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			member := g.FilterBattlefield(IsToken)[0]
			g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			switch destination {
			case "destroy":
				g.DestroyPermanent(member)
			case "exile":
				g.ExilePermanent(member)
			case "sacrifice":
				g.DoSacrifice(member)
			case "bounce":
				g.BouncePermanentToHand(member)
			case "replacement":
				g.zones.AddExileInstead(member.ID(), source.ID())
				g.DestroyPermanent(member)
			case "regenerate":
				g.AddRegenerationShield(member.ID())
				g.DestroyPermanent(member)
			}
			g.ResolveStack()
			want := 21
			if destination == "regenerate" {
				want = 20
			}
			if b.Life() != want || a.Life() != 20 {
				t.Fatalf("life A=%d B=%d, want B=%d", a.Life(), b.Life(), want)
			}
			if destination == "regenerate" && g.FindPermanent(member.ID()) == nil {
				t.Fatal("regenerated token left")
			}
		})
	}
}

func TestCreatedTokenLeaveExactEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Maker", ""), a.PlayerID())
	link := NewCreatedTokenLink()
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: source.ID()}
	if err := CreateTokenCopiesLinked(You, Fixed(1), DefineToken("Member", TokenPT(1, 1), TokenTypes(core.TypeCreature)), link).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	member := g.FilterBattlefield(IsToken)[0]
	event := core.GameEvent{Type: core.EvtZoneChange, FromZone: core.ZoneBattlefield, ToZone: core.ZoneExile, SourceID: member.ID(), SourceIncarnation: member.incarnationID}
	pattern := createdTokenLeavePattern{link: link}
	before := g.Clone()
	g.ExilePermanent(member)
	returned := g.PutOnBattlefield(member.Card, a.PlayerID())
	if returned.incarnationID == member.incarnationID {
		t.Fatal("test did not create a new visit")
	}
	for _, game := range []*Game{g, before, g.Clone()} {
		if !pattern.CheckTriggerCond(&event, game, source.ID(), a.PlayerID()) {
			t.Fatal("exact departure facts lost")
		}
	}
	for _, mutate := range []func(*core.GameEvent){
		func(e *core.GameEvent) { e.SourceIncarnation = returned.incarnationID },
		func(e *core.GameEvent) { e.SourceIncarnation = uuid.Nil },
		func(e *core.GameEvent) { e.SourceID = source.ID() },
		func(e *core.GameEvent) { e.Type = core.EvtSacrifice },
		func(e *core.GameEvent) { e.FromZone = core.ZoneGraveyard },
		func(e *core.GameEvent) { e.ToZone = core.ZoneBattlefield },
	} {
		bad := event
		mutate(&bad)
		if pattern.CheckTriggerCond(&bad, g, source.ID(), a.PlayerID()) {
			t.Fatal("matched different visit or event")
		}
	}
}
