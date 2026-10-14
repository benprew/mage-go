package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type entryLifePlayer struct {
	*BasePlayer
	answer, calls, minimum, maximum, mode, orders int
	during                                        func()
}

type entryLifePlayers struct{ ids []uuid.UUID }

func (entryLifePlayers) playerSet()                                  {}
func (p entryLifePlayers) resolvePlayers(*EffectContext) []uuid.UUID { return p.ids }
func (entryLifePlayers) inspectDefinition() definitionNode           { return definitionNode{Role: "player-set"} }

func (p *entryLifePlayer) ChooseNumber(minimum, maximum int, _ string) int {
	p.calls++
	p.minimum, p.maximum = minimum, maximum
	if p.during != nil {
		p.during()
	}
	return p.answer
}
func (p *entryLifePlayer) ChooseMode([]string, string) int { p.orders++; return p.mode }
func (*entryLifePlayer) ChooseMayAbility(string) bool      { return true }

func entryLifeCard(maximum ResolutionValue) Card {
	return NewCreature("Entry life", "", 0, 0,
		WithAbility(PayLifeUpToAsEnters(maximum)),
		WithStaticAbility(PTEqualsEntryLifePaid()))
}

func TestEntryLifeBoundsAndTiming(t *testing.T) {
	for _, state := range []struct{ life, bound, answer, want, maximum int }{
		{20, 5, 3, 3, 5}, {20, 5, 0, 0, 5}, {3, 5, 3, 3, 3},
		{0, 5, 0, 0, 0}, {-3, 5, 0, 0, 0}, {20, -1, 0, 0, 0},
		{20, 5, -1, 0, 5}, {20, 5, 6, 0, 5},
	} {
		a := &entryLifePlayer{BasePlayer: NewBasePlayer("A"), answer: state.answer}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		a.SetLife(state.life)
		card := entryLifeCard(Fixed(state.bound))
		card.SetOwner(b.PlayerID())
		a.during = func() {
			for _, live := range g.AllBattlefield() {
				if live.ID() == card.ID() {
					t.Fatal("choice after entry")
				}
			}
			if a.Life() != state.life {
				t.Fatal("spent before choice")
			}
		}
		p := g.PutOnBattlefield(card, a.PlayerID())
		if p.EntryLifePaid != state.want || p.CurrentPower(g) != state.want || p.CurrentToughness(g) != state.want || a.Life() != state.life-state.want || b.Life() != 20 || a.calls != 1 || a.minimum != 0 || a.maximum != state.maximum {
			t.Fatalf("bounds/payment: result=%d life=%d range=%d..%d", p.EntryLifePaid, a.Life(), a.minimum, a.maximum)
		}
	}
}

func TestEntryLifeResultIdentityAndLayers(t *testing.T) {
	a := &entryLifePlayer{BasePlayer: NewBasePlayer("A"), answer: 3}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	card := entryLifeCard(Fixed(7))
	p := g.PutOnBattlefield(card, a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	value := EntryLifePaidOf(This)
	g.PhaseIn(g.PhaseOut(p.ID()))
	if a.calls != 1 || value.resolveValue(ctx) != 3 {
		t.Fatal("phase changed result")
	}
	clone := g.Clone()
	clone.MutablePermanent(p.ID()).EntryLifePaid = 6
	clone.ApplyContinuousEffects()
	if clone.FindPermanent(p.ID()).CurrentPower(clone) != 6 || value.resolveValue(ctx) != 3 {
		t.Fatal("shared clone result")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, SetBasePowerToughness(This, Fixed(1), Fixed(2))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	p = g.FindPermanent(p.ID())
	if p.CurrentPower(g) != 1 || p.CurrentToughness(g) != 2 {
		t.Fatal("entry result used layer 7b")
	}
	g.AddCountersWithReplacement(p, core.P1P1, 1, p.ID(), false)
	if p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 3 {
		t.Fatal("counter order")
	}
	g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.EndOfTurn))
	g.ApplyContinuousEffects()
	if g.FindPermanent(p.ID()).CurrentPower(g) != 2 {
		t.Fatal("ability removal changed independent base override")
	}
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if g.FindPermanent(p.ID()).CurrentPower(g) != 4 {
		t.Fatal("CDA restoration lost payment or counter order")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	if value.resolveValue(ctx) != 3 {
		t.Fatal("lost departure result")
	}
	if power, toughness := g.CardPowerToughness(a.Hand()[0]); power != 0 || toughness != 0 {
		t.Fatal("hand kept entry result")
	}
	a.RemoveFromHand(card.ID())
	a.answer = 1
	next := g.PutOnBattlefield(card, a.PlayerID())
	if next.CurrentPower(g) != 1 || value.resolveValue(ctx) != 3 || a.Life() != 16 || a.calls != 2 {
		t.Fatal("return replaced old result")
	}
}

func TestEntryLifeRecheckAndReplacementOrder(t *testing.T) {
	a := &entryLifePlayer{BasePlayer: NewBasePlayer("A"), answer: 3}
	g := NewGame(a, NewBasePlayer("B"))
	a.during = func() { a.SetLife(2) }
	p := g.PutOnBattlefield(entryLifeCard(Fixed(5)), a.PlayerID())
	if a.Life() != 2 || p.EntryLifePaid != 0 {
		t.Fatal("invalid payment spent life or saved result")
	}
	for _, mode := range []int{0, 1} {
		a := &entryLifePlayer{BasePlayer: NewBasePlayer("A"), answer: 3, mode: mode}
		g := NewGame(a, NewBasePlayer("B"))
		seen := -1
		g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(action Action) bool { _, ok := action.(*battlefieldEntryAction); return ok }, replaceFunc: func(action Action, _ *Game) Action {
			seen = action.(*battlefieldEntryAction).permanent.EntryLifePaid
			return action
		}})
		p := g.PutOnBattlefield(entryLifeCard(Fixed(5)), a.PlayerID())
		if seen != mode*3 || p.EntryLifePaid != 3 || a.calls != 1 || a.orders != 1 || a.Life() != 17 {
			t.Fatal("replacement order or repeat", seen, a.calls, a.orders)
		}
	}
}

func TestEntryLifeInvalid(t *testing.T) {
	for _, card := range []Card{
		entryLifeCard(nil), entryLifeCard(FromNumber(BindNumber("unbound"))),
		NewInstant("Bad", "", WithAbility(PayLifeUpToAsEnters(Fixed(3)))),
		NewCreature("Bad", "", 1, 1, WithAbility(PayLifeUpToAsEnters(Fixed(3))), WithAbility(PayLifeUpToAsEnters(Fixed(3)))),
		NewSorcery("Bad", "", Spell(GainLifeFor(You, EntryLifePaidOf(nil)))),
	} {
		if err := ValidateCardDefinition("Bad", card); err == nil {
			t.Fatal("invalid definition accepted")
		}
	}
}

func TestEntryLifeGraveyardPlayerSet(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.AddToGraveyard(NewCreature("Your white", "{W}", 1, 1))
	b.AddToGraveyard(NewCreature("Opponent white", "{W}", 1, 1))
	b.AddToGraveyard(NewCreature("Opponent black", "{B}", 1, 1))
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if CountGraveyardCardsAmong(OpponentsOf(You), HasColorCardFilter(core.White)).resolveValue(ctx) != 1 || CountGraveyardCardsAmong(EachPlayer, HasColorCardFilter(core.White)).resolveValue(ctx) != 2 {
		t.Fatal("graveyard player membership or filter")
	}
	if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", WithAbility(Spell(GainLifeFor(You, CountGraveyardCardsAmong(nil, CardFilter{})))))); err == nil {
		t.Fatal("nil graveyard player set accepted")
	}
	players := entryLifePlayers{[]uuid.UUID{b.PlayerID(), b.PlayerID(), uuid.New()}}
	if CountGraveyardCardsAmong(players, CardFilter{}).resolveValue(ctx) != 2 {
		t.Fatal("duplicate or missing players counted")
	}
	clone := g.Clone()
	clone.GetPlayer(b.PlayerID()).RemoveFromGraveyard(b.Graveyard()[0].ID())
	if CountGraveyardCardsAmong(players, CardFilter{}).resolveValue(&EffectContext{Game: clone, Controller: a.PlayerID()}) != 1 || CountGraveyardCardsAmong(players, CardFilter{}).resolveValue(ctx) != 2 {
		t.Fatal("graveyard count clone isolation")
	}
}

func TestEntryLifeCopyAndWaiting(t *testing.T) {
	a := &entryLifePlayer{BasePlayer: NewBasePlayer("A"), answer: 2}
	b := &entryLifePlayer{BasePlayer: NewBasePlayer("B"), answer: 3}
	g := NewGame(a, b)
	donorPlan, err := NewCardDefinitionPlan(func() Card { return entryLifeCard(Fixed(5)) })
	if err != nil {
		t.Fatal(err)
	}
	donorCard, err := donorPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	donor := g.PutOnBattlefield(donorCard, b.PlayerID())
	copyPlan, err := NewCardDefinitionPlan(func() Card { return NewCreature("Copy", "", 1, 1, WithAbility(MayCopyAsEnters(IsCreature))) })
	if err != nil {
		t.Fatal(err)
	}
	copyCard, err := copyPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(copyCard, a.PlayerID())
	if p.Name() != donor.Name() || p.CurrentPower(g) != 2 || donor.CurrentPower(g) != 3 || a.Life() != 18 || b.Life() != 17 || a.calls != 1 {
		t.Fatal("copy inherited payment instead of making its own")
	}
	saved := g.CaptureDamageSource(p.ID())
	if err := ApplyEffect(g, AtNext(EachEndStep, GainLifeFor(You, EntryLifePaidOf(This))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(p)
	clone := g.Clone()
	for range 2 {
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(saved) || a.calls != 1 {
			t.Fatal("waiting query lost identity or chose")
		}
	}
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 20 || a.Life() != 18 {
		t.Fatal("clone waiting result")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("delayed entry payment read")
	}
}

func TestEntryLifeBatchEntrantsAndCancellation(t *testing.T) {
	a := &entryLifePlayer{BasePlayer: NewBasePlayer("A"), answer: 2}
	b := &entryLifePlayer{BasePlayer: NewBasePlayer("B"), answer: 3}
	g := NewGame(a, b)
	first, second := entryLifeCard(Fixed(5)), entryLifeCard(Fixed(5))
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	a.AddToGraveyard(first)
	b.AddToGraveyard(second)
	observed := 0
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(event *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		if event.ToZone != core.ZoneBattlefield || (event.SourceID != first.ID() && event.SourceID != second.ID()) {
			return false
		}
		observed++
		p, q := reader.FindPermanent(first.ID()), reader.FindPermanent(second.ID())
		if p == nil || q == nil || p.CurrentPower(reader) != 2 || q.CurrentPower(reader) != 3 || a.Life() != 18 || b.Life() != 17 {
			t.Fatal("entry events before all payments and entries")
		}
		return false
	}))), a.PlayerID())
	if err := ApplyEffect(g, PutGraveyardCreaturesOntoBattlefieldUnderOwnersControl(), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if observed != 2 || a.calls != 1 || b.calls != 1 {
		t.Fatal("batch payment instances")
	}
	for _, mode := range []int{0, 1} {
		a := &entryLifePlayer{BasePlayer: NewBasePlayer("A"), answer: 2, mode: mode}
		g := NewGame(a, NewBasePlayer("B"))
		g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(action Action) bool { _, ok := action.(*battlefieldEntryAction); return ok }, replaceFunc: func(Action, *Game) Action { return nil }})
		p := g.PutOnBattlefield(entryLifeCard(Fixed(5)), a.PlayerID())
		if p != nil || a.Life() != 20-mode*2 || a.calls != mode {
			t.Fatal("cancelled entry order or payment rollback")
		}
	}
}

func TestEntryLifeLossEventsAndPureQueries(t *testing.T) {
	for _, answer := range []int{0, 2} {
		a := &entryLifePlayer{BasePlayer: NewBasePlayer("A"), answer: answer}
		g := NewGame(a, NewBasePlayer("B"))
		events := 0
		g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtLifeLost, false).SetCondition(func(event *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
			if reader != g {
				return false
			}
			events++
			if event.PlayerID != a.PlayerID() || event.Amount != answer || a.Life() != 20-answer {
				t.Fatal("life payment event")
			}
			return false
		}))), a.PlayerID())
		card := entryLifeCard(Fixed(5))
		for range 2 {
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			g.CardPowerToughness(card)
		}
		if a.calls != 0 || a.Life() != 20 {
			t.Fatal("definition query chose or paid")
		}
		p := g.PutOnBattlefield(card, a.PlayerID())
		if p.EntryLifePaid != answer || events != min(answer, 1) {
			t.Fatal("zero or positive life event count")
		}
	}
}

func TestEntryLifeSharedPayerBatch(t *testing.T) {
	a := &entryLifePlayer{BasePlayer: NewBasePlayer("A")}
	a.SetLife(3)
	g := NewGame(a, NewBasePlayer("B"))
	a.during = func() { a.answer = a.maximum }
	for range 2 {
		card := entryLifeCard(Fixed(5))
		card.SetOwner(a.PlayerID())
		a.AddToGraveyard(card)
	}
	if err := ApplyEffect(g, PutGraveyardCreaturesOntoBattlefieldUnderOwnersControl(), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	paid := 0
	for _, p := range g.AllBattlefield() {
		paid += p.EntryLifePaid
	}
	if paid != 3 || a.Life() != 0 || a.calls != 2 || a.maximum != 0 {
		t.Fatal("batch choices exceeded shared life")
	}
}

func TestEntryLifeGraveyardWaitingSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Source", "")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	white := NewArtifact("White", "{W}")
	white.SetOwner(b.PlayerID())
	b.AddToGraveyard(white)
	saved := g.CaptureDamageSource(p.ID())
	value := CountGraveyardCardsAmong(OpponentsOf(ControllerOfWithLKI(This)), HasColorCardFilter(core.White))
	if err := ApplyEffect(g, AtNext(EachEndStep, GainLifeFor(You, value)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(p)
	for range 2 {
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(saved) || a.Life() != 20 {
			t.Fatal("graveyard query lost source or changed life")
		}
	}
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("graveyard count followed returned source controller")
	}
}

func TestEntryLifeCloneDuringChoice(t *testing.T) {
	a := &entryLifePlayer{BasePlayer: NewBasePlayer("A"), answer: 2}
	g := NewGame(a, NewBasePlayer("B"))
	card := entryLifeCard(Fixed(5))
	var clone *Game
	a.during = func() {
		clone = g.Clone()
		clone.MutablePermanent(card.ID()).EntryLifePaid = 99
		if g.FindPermanent(card.ID()).EntryLifePaid != 0 {
			t.Fatal("entry clone shared payment result")
		}
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	if p.EntryLifePaid != 2 || clone.FindPermanent(card.ID()).EntryLifePaid != 99 || clone.GetPlayer(a.PlayerID()).Life() != 20 || a.Life() != 18 {
		t.Fatal("entry clone changed after live payment")
	}
}
