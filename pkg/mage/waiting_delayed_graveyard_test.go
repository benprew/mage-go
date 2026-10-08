package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedGraveyardFrames(t *testing.T) {
	zone := core.ZoneGraveyard
	for _, kind := range []string{"step", "modes", "event"} {
		t.Run(zone.String()+"/"+kind, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Saved", "{R}", 1, 1)
			card.SetOwner(a.PlayerID())
			a.AddToGraveyard(card)
			visit := g.graveyardVisit(card.ID())
			one, many := BindCard("one"), BindCards("many")
			identity := capturedCard{card.ID(), visit, zone}
			g.resolution.cardBindings = map[*CardBinding]capturedCard{one: identity}
			g.resolution.cardsBindings = map[*CardsBinding][]capturedCard{many: {identity}}
			calls := 0
			body := FuncEffect("check frame", EffectProperties{}, func(game *Game, source, controller uuid.UUID, targets []uuid.UUID) error {
				calls++
				ctx := &EffectContext{Game: game, Controller: controller}
				if _, ok := FromCard(one).resolveCard(ctx); ok {
					t.Fatal("returned visit resolved as original")
				}
				if len(FromCards(many).resolveCards(ctx)) != 0 {
					t.Fatal("returned collection resolved as original")
				}
				if game.resolution.cardBindings[one] != identity || game.resolution.cardsBindings[many][0] != identity {
					t.Fatal("saved bindings lost")
				}
				delete(game.resolution.cardBindings, one)
				game.resolution.cardsBindings[many][0] = capturedCard{}
				return nil
			})
			subject := g.PutOnBattlefield(NewCreature("Watch", "", 1, 1), a.PlayerID())
			delay := AtNext(EachEndStep, body)
			if kind == "modes" {
				delay = AtNextModes(EachEndStep, ModeOf("one", body), ModeOf("two", body))
			}
			if kind == "event" {
				delay = WhenLeavesThisTurn(This, body)
			}
			ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: subject.ID()}
			if err := delay.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			g.resolution.cardsBindings[many][0] = capturedCard{}
			g.resolution.Clear()
			query := (*Game).WaitingDelayedGraveyardSources
			got := query(g)
			if len(got) != 1 || !got[0].HasColor(core.Red) {
				t.Fatalf("waiting sources=%v", got)
			}
			saved := got[0]
			before := g.Clone()
			card.AddType(core.TypeArtifact)
			a.RemoveFromGraveyard(card.ID())
			a.AddToGraveyard(card)
			got = query(g)
			if len(got) != 1 || !got[0].SameObject(saved) || !got[0].HasType(core.TypeArtifact) || query(before)[0].HasType(core.TypeArtifact) {
				t.Fatal("departure or clone facts")
			}
			got[0].card.AddType(core.TypeLand)
			if query(g)[0].HasType(core.TypeLand) {
				t.Fatal("query mutation")
			}
			if query(g)[0].GraveyardVisit() == g.graveyardVisit(card.ID()) {
				t.Fatal("returned graveyard visit")
			}
			if kind == "event" {
				g.BouncePermanentToHand(subject)
			} else {
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			}
			if kind != "event" && len(query(g)) != 0 {
				t.Fatal("one-shot still waiting")
			}
			g.PutTriggersOnStack()
			stack := g.StackGraveyardSources()
			if len(stack) != 1 || !stack[0].SameObject(saved) {
				t.Fatal("placed frame lost reference")
			}
			if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			if calls != 4 {
				t.Fatalf("body calls=%d", calls)
			}
		})
	}
}

func TestWaitingDelayedGraveyardBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	chooser := &sourceChoicePlayer{BasePlayer: b, index: -1}
	g := NewGame(a, chooser)
	card := NewArtifact("Public source", "{R}")
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	identity := capturedCard{card.ID(), g.graveyardVisit(card.ID()), core.ZoneGraveyard}
	one, many := BindCard("one"), BindCards("many")
	g.resolution.cardBindings = map[*CardBinding]capturedCard{one: identity, nil: identity, BindCard("wrong ID"): {uuid.New(), identity.visit, identity.zone}, BindCard("no visit"): {card.ID(), uuid.Nil, identity.zone}, BindCard("wrong zone"): {card.ID(), identity.visit, core.ZoneHand}}
	g.resolution.cardsBindings = map[*CardsBinding][]capturedCard{many: {identity, identity}, nil: {identity}}
	source := g.PutOnBattlefield(NewArtifact("Keeper", ""), a.PlayerID())
	ctx := &EffectContext{Game: g, Controller: b.PlayerID(), SourceID: source.ID()}
	if err := AtNext(EachEndStep, GainLifeFor(You, Fixed(1))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.resolution.Clear()
	if len(g.WaitingDelayedGraveyardSources()) != 1 {
		t.Fatal("invalid identities or duplicate visits supplied choices")
	}
	source.PhasedOut = true
	g.ApplyContinuousEffects()
	if len(g.WaitingDelayedGraveyardSources()) != 1 {
		t.Fatal("source phasing ended independent delay")
	}
	source.PhasedOut = false
	g.BouncePermanentToHand(source)
	if len(g.WaitingDelayedGraveyardSources()) != 1 {
		t.Fatal("source departure ended independent delay")
	}
	g.RegisterDelayedTrigger(&DelayedTrigger{EventType: core.EvtEndStep, Effects: []Effect{AtNext(EachEndStep, GainLifeFor(You, Fixed(1)))}})
	binding := BindDamageSource("chosen")
	choice := ChooseDamageSourceFromWaitingDelayedGraveyardBindings(You, DamageSourceHasColor(core.Red), binding)
	if err := choice.Apply(ctx); err != nil || chooser.calls != 1 || len(chooser.seen) != 1 {
		t.Fatal("public opponent graveyard choice", err)
	}
	saved := FromDamageSource(binding).resolveDamageSource(ctx)
	a.RemoveFromGraveyard(card.ID())
	card.AddType(core.TypeLand)
	a.AddToGraveyard(card)
	if got := g.WaitingDelayedGraveyardSources(); len(got) != 1 || !got[0].SameObject(saved) || got[0].HasType(core.TypeLand) {
		t.Fatal("returned card changed departure information")
	}
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	if len(g.WaitingDelayedGraveyardSources()) != 0 || len(g.StackGraveyardSources()) != 0 || len(clone.WaitingDelayedGraveyardSources()) != 1 {
		t.Fatal("pending or clone lifetime")
	}
	g.PutTriggersOnStack()
	if len(g.StackGraveyardSources()) != 1 {
		t.Fatal("stack transfer")
	}
	g.ResolveStack()
	if len(g.StackGraveyardSources()) != 0 || b.Life() != 21 {
		t.Fatal("stack departure or delayed controller")
	}
	if err := choice.Apply(ctx); err != nil || FromDamageSource(binding).resolveDamageSource(ctx).Valid() {
		t.Fatal("empty choice kept stale result")
	}
	ctx.Controller = uuid.New()
	if err := choice.Apply(ctx); err != nil || FromDamageSource(binding).resolveDamageSource(ctx).Valid() {
		t.Fatal("absent player kept stale result")
	}
	for _, invalid := range []ResolvingEffect{
		ChooseDamageSourceFromWaitingDelayedGraveyardBindings(nil, DamageSourceFilter{}, binding),
		ChooseDamageSourceFromWaitingDelayedGraveyardBindings(You, DamageSourceFilter{}, nil),
		ChooseDamageSourceFromWaitingDelayedGraveyardBindings(You, DamageSourceHasColor(core.Colorless), binding),
		Sequence(DealDamageFrom(FromDamageSource(binding), You, Fixed(1)), choice),
	} {
		if ValidateCardDefinition("Invalid", NewInstant("Invalid", "", Spell(invalid))) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestWaitingDelayedGraveyardDiscardEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	binding := BindCard("discarded")
	keeper := NewArtifact("Keeper", "", WithAbility(WhenEvent(PlayerDiscardsCard(BindPlayer("discarder")).WithCard(binding), AtNext(EachEndStep, GainLifeFor(You, Fixed(1))))))
	g.PutOnBattlefield(keeper, a.PlayerID())
	card := NewCreature("Discarded", "{R}", 1, 1)
	card.SetOwner(b.PlayerID())
	b.AddToHand(card)
	if err := ApplyEffect(g, Discard(You, Fixed(1)), keeper.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	g.ResolveStack()
	got := g.WaitingDelayedGraveyardSources()
	if len(got) != 1 || got[0].ID() != card.ID() {
		t.Fatal("singular event capture", got)
	}
}
