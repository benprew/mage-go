package mage

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedCardFrames(t *testing.T) {
	for _, zone := range []core.Zone{core.ZoneHand, core.ZoneExile} {
		for _, kind := range []string{"step", "modes", "event"} {
			t.Run(zone.String()+"/"+kind, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Saved", "{R}", 1, 1)
				card.SetOwner(a.PlayerID())
				var visit uuid.UUID
				if zone == core.ZoneHand {
					a.AddToHand(card)
					visit = g.handVisit(card.ID())
				} else {
					g.ExileCard(card, uuid.New())
					visit = g.FindExiledCard(card.ID()).Visit
				}
				one, many := BindCard("one"), BindCards("many")
				identity := capturedCard{id: card.ID(), visit: visit, zone: zone}
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
				query := func(game *Game) []DamageSource {
					if zone == core.ZoneHand {
						return game.WaitingDelayedHandSourcesFor(a.PlayerID())
					}
					return game.WaitingDelayedExileSources()
				}
				got := query(g)
				if len(got) != 1 || !got[0].HasColor(core.Red) {
					t.Fatalf("waiting sources=%v", got)
				}
				saved := got[0]
				before := g.Clone()
				card.AddType(core.TypeArtifact)
				if zone == core.ZoneHand {
					a.RemoveFromHand(card.ID())
					a.AddToHand(card)
				} else {
					g.RemoveFromExile(card.ID())
					g.ExileCard(card, uuid.New())
				}
				got = query(g)
				if len(got) != 1 || !got[0].SameObject(saved) || !got[0].HasType(core.TypeArtifact) || query(before)[0].HasType(core.TypeArtifact) {
					t.Fatal("departure or clone facts")
				}
				got[0].card.AddType(core.TypeLand)
				if query(g)[0].HasType(core.TypeLand) {
					t.Fatal("query mutation")
				}
				if zone == core.ZoneHand && query(g)[0].HandVisit() == g.handVisit(card.ID()) {
					t.Fatal("returned hand visit")
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
				var stack []DamageSource
				if zone == core.ZoneHand {
					stack = g.StackHandSourcesFor(a.PlayerID())
				} else {
					stack = g.StackBoundExileSources()
				}
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
}

func TestWaitingDelayedCardRestoration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	one, many := BindCard("one"), BindCards("many")
	identity := capturedCard{id: uuid.New(), visit: uuid.New(), zone: core.ZoneHand}
	g.resolution.cardBindings = map[*CardBinding]capturedCard{one: identity}
	g.resolution.cardsBindings = map[*CardsBinding][]capturedCard{many: {identity}}
	want := errors.New("body failure")
	body := FuncEffect("check frame", EffectProperties{}, func(game *Game, source, controller uuid.UUID, targets []uuid.UUID) error {
		if game.resolution.cardBindings[one] != identity || game.resolution.cardsBindings[many][0] != identity {
			t.Fatal("saved frame missing")
		}
		delete(game.resolution.cardBindings, one)
		game.resolution.cardsBindings[many][0] = capturedCard{}
		return want
	})
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := AtNext(EachEndStep, body).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	outer := capturedCard{id: uuid.New(), visit: uuid.New(), zone: core.ZoneExile}
	g.resolution.cardBindings[one] = outer
	g.resolution.cardsBindings[many][0] = outer
	for range 2 {
		if err := g.DelayedTriggers()[0].Effects[0].Apply(ctx); !errors.Is(err, want) {
			t.Fatal(err)
		}
		if g.resolution.cardBindings[one] != outer || g.resolution.cardsBindings[many][0] != outer {
			t.Fatal("outer frame changed")
		}
	}
}

func TestWaitingDelayedCardChoicesAndVisibility(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	chooser := &sourceChoicePlayer{BasePlayer: a, index: -1}
	g := NewGame(chooser, b)
	hand := NewArtifact("Hidden", "{R}")
	hand.SetOwner(a.PlayerID())
	a.AddToHand(hand)
	exile := NewArtifact("Exile", "{R}")
	exile.SetOwner(a.PlayerID())
	g.ExileCard(exile, uuid.New())
	h := capturedCard{id: hand.ID(), visit: g.handVisit(hand.ID()), zone: core.ZoneHand}
	x := capturedCard{id: exile.ID(), visit: g.FindExiledCard(exile.ID()).Visit, zone: core.ZoneExile}
	g.resolution.cardBindings = map[*CardBinding]capturedCard{BindCard("hand"): h, BindCard("exile"): x, nil: h, BindCard("bad"): {id: uuid.New(), visit: h.visit, zone: core.ZoneHand}}
	g.resolution.cardsBindings = map[*CardsBinding][]capturedCard{BindCards("both"): {h, x}, nil: {h}}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := AtNext(EachEndStep, GainLifeFor(You, Fixed(1))).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.resolution.Clear()
	if len(g.WaitingDelayedHandSourcesFor(uuid.Nil)) != 1 || g.WaitingDelayedHandSourcesFor(b.PlayerID())[0].HasColor(core.Red) || !g.WaitingDelayedHandSourcesFor(a.PlayerID())[0].HasColor(core.Red) {
		t.Fatal("inspection boundary")
	}
	chosen := BindDamageSource("chosen")
	choice := ChooseDamageSourceFromWaitingDelayedCardBindings(You, DamageSourceHasColor(core.Red), chosen)
	if err := choice.Apply(ctx); err != nil || chooser.calls != 1 {
		t.Fatal("filtered choice", err)
	}
	if len(chooser.seen) != 2 {
		t.Fatal("cross-zone deduplication")
	}
	g.FindExiledCard(exile.ID()).FaceDown = true
	if g.WaitingDelayedExileSources()[0].HasColor(core.Red) {
		t.Fatal("face-down facts")
	}
	a.RemoveFromHand(hand.ID())
	a.AddToHand(hand)
	if g.WaitingDelayedHandSourcesFor(b.PlayerID())[0].HasColor(core.Red) || !g.WaitingDelayedHandSourcesFor(a.PlayerID())[0].HasColor(core.Red) {
		t.Fatal("private departure")
	}
	g.RemoveFromExile(exile.ID())
	g.ExileCard(exile, uuid.New())
	if g.WaitingDelayedExileSources()[0].HasColor(core.Red) {
		t.Fatal("hidden departure changed")
	}
	g.triggers.SetDelayed(nil)
	if err := choice.Apply(ctx); err != nil || FromDamageSource(chosen).resolveDamageSource(ctx).Valid() {
		t.Fatal("empty choice retained binding")
	}
	for _, invalid := range []ResolvingEffect{ChooseDamageSourceFromWaitingDelayedCardBindings(nil, DamageSourceFilter{}, chosen), ChooseDamageSourceFromWaitingDelayedCardBindings(You, DamageSourceFilter{}, nil), ChooseDamageSourceFromWaitingDelayedCardBindings(You, DamageSourceHasColor(core.Colorless), chosen), Sequence(DealDamageFrom(FromDamageSource(chosen), You, Fixed(1)), choice)} {
		if ValidateCardDefinition("Invalid", NewInstant("Invalid", "", Spell(invalid))) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestWaitingDelayedCardRevealLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Revealed", "{R}", 1, 1)
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	paid := BindCard("reveal")
	keeper := NewArtifact("Keeper", "", WithAbility(Activate(Costs(ManaCostOf("{1}"), RevealFromHandCostWithCard(CardFilter{}, "Reveal", paid)), AtNext(EachEndStep, GainLifeFor(You, Fixed(1))))))
	p := g.PutOnBattlefield(keeper, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || len(g.DelayedTriggers()) != 0 || len(g.PublicReveals()) != 0 {
		t.Fatal("failed payment changed state")
	}
	a.ManaPool().Add(core.Red, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !g.StackHandSources()[0].HasColor(core.Red) {
		t.Fatal("active reveal hidden")
	}
	g.ResolveTopOfStack()
	if len(g.PublicReveals()) != 1 || len(g.WaitingDelayedHandSourcesFor(b.PlayerID())) != 1 || g.WaitingDelayedHandSourcesFor(b.PlayerID())[0].HasColor(core.Red) {
		t.Fatal("delay extended reveal")
	}
	if !g.WaitingDelayedHandSourcesFor(a.PlayerID())[0].HasColor(core.Red) {
		t.Fatal("owner lost inspection")
	}
	p.PhasedOut = true
	g.ApplyContinuousEffects()
	if len(g.WaitingDelayedHandSourcesFor(a.PlayerID())) != 1 {
		t.Fatal("source phasing ended delay")
	}
	a.RemoveFromHand(card.ID())
	a.AddToHand(card)
	if g.WaitingDelayedHandSourcesFor(b.PlayerID())[0].HasColor(core.Red) {
		t.Fatal("departed reveal leaked")
	}
	g.RegisterDelayedTrigger(&DelayedTrigger{EventType: core.EvtEndStep, Effects: []Effect{AtNext(EachEndStep, GainLifeFor(You, Fixed(1)))}})
	if len(g.WaitingDelayedHandSourcesFor(a.PlayerID())) != 1 {
		t.Fatal("future producer supplied references")
	}
}

func TestWaitingDelayedCardPluralPayment(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCards("payment")
	keeper := NewArtifact("Keeper", "", WithAbility(Activate(Costs(ExileFromGraveyardCostWithCards(2, paid)), AtNext(EachEndStep, GainLifeFor(You, Fixed(1))))))
	p := g.PutOnBattlefield(keeper, a.PlayerID())
	for range 2 {
		card := NewCreature("Payment", "{R}", 1, 1)
		card.SetOwner(a.PlayerID())
		a.AddToGraveyard(card)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	saved := g.StackBoundExileSources()
	g.ResolveTopOfStack()
	if len(saved) != 2 || len(g.WaitingDelayedExileSources()) != 2 {
		t.Fatal("plural payment not captured")
	}
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	if len(g.WaitingDelayedExileSources()) != 0 || len(g.StackBoundExileSources()) != 0 || len(clone.WaitingDelayedExileSources()) != 2 {
		t.Fatal("pending or clone lifetime")
	}
	g.PutTriggersOnStack()
	if len(g.StackBoundExileSources()) != 2 {
		t.Fatal("placed plural frame")
	}
	g.ResolveStack()
	if a.Life() != 21 || len(g.StackBoundExileSources()) != 0 {
		t.Fatal("plural delayed execution")
	}
}
