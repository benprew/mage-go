package mage

import (
	"testing"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// =============================================================================
// Prepared
// =============================================================================

// TestPrepared_ETBSetsPreparedAttr: a creature built with WithPreparedSpell
// becomes Prepared as it enters the battlefield.
func TestPrepared_ETBSetsPreparedAttr(t *testing.T) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B")
	g := NewGame(a, b)

	spellFactory := func() Card {
		return NewInstant("Rejoinder", "{1}{W}",
			NewSpellAbility(DrawCards(Fixed(1))))
	}
	c := NewCreature("Elite Interceptor", "{W}", 1, 2,
		WithSubTypes("Human", "Wizard"),
		WithPreparedSpell(spellFactory),
	)
	c.SetOwner(a.PlayerID())
	perm := g.PutOnBattlefield(c, a.PlayerID())
	g.ResolveStack()

	if !g.IsPrepared(perm.Card.ID()) {
		t.Errorf("expected creature to be prepared on ETB")
	}
}

// TestPrepared_ActivateCastsCopyAndUnprepares: the Prepared activated
// ability casts a copy of the spell side and clears the prepared flag.
func TestPrepared_ActivateCastsCopyAndUnprepares(t *testing.T) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B")
	g := NewGame(a, b)
	for _, p := range g.players {
		for range 10 {
			p.AddToLibrary(NewLand("Plains"))
		}
	}

	spellFactory := func() Card {
		return NewInstant("Rejoinder", "{1}{W}",
			NewSpellAbility(DrawCards(Fixed(1))))
	}
	c := NewCreature("Elite Interceptor", "{W}", 1, 2,
		WithPreparedSpell(spellFactory),
	)
	c.SetOwner(a.PlayerID())
	perm := g.PutOnBattlefield(c, a.PlayerID())
	g.ResolveStack()

	// Run the Prepared cast-copy directly.
	if err := g.CastPreparedSpellCopy(a.PlayerID(), perm.Card.ID(), spellFactory); err != nil {
		t.Fatalf("CastPreparedSpellCopy: %v", err)
	}

	// A copy should be sitting on the stack.
	if g.stack.IsEmpty() {
		t.Fatalf("expected stack non-empty after casting copy")
	}
	top := g.stack.Peek()
	if !top.IsCopy {
		t.Errorf("expected the cast spell to be a copy")
	}

	g.ResolveStack()

	// The creature is no longer prepared.
	if g.IsPrepared(perm.Card.ID()) {
		t.Errorf("expected creature to be unprepared after casting copy")
	}

	// The copy should have drawn a card.
	if len(a.Hand()) != 1 {
		t.Errorf("expected 1 card drawn, got %d", len(a.Hand()))
	}
}

// TestPrepared_CannotCastWhenUnprepared: once unprepared, the activated
// ability is not activatable (returns an error from the engine helper).
func TestPrepared_CannotCastWhenUnprepared(t *testing.T) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B")
	g := NewGame(a, b)

	spellFactory := func() Card {
		return NewInstant("Rejoinder", "{1}{W}",
			NewSpellAbility(DrawCards(Fixed(1))))
	}
	c := NewCreature("Elite Interceptor", "{W}", 1, 2,
		WithPreparedSpell(spellFactory),
	)
	c.SetOwner(a.PlayerID())
	perm := g.PutOnBattlefield(c, a.PlayerID())
	g.ResolveStack()

	g.SetPrepared(perm.Card.ID(), false)

	if err := g.CastPreparedSpellCopy(a.PlayerID(), perm.Card.ID(), spellFactory); err == nil {
		t.Errorf("expected error casting prepared copy when unprepared")
	}
}

// =============================================================================
// Repartee
// =============================================================================

// TestRepartee_TriggersWhenInstantTargetsCreature: when controller casts an
// instant targeting a creature on the battlefield, the trigger fires.
func TestRepartee_TriggersWhenInstantTargetsCreature(t *testing.T) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B")
	g := NewGame(a, b)

	// A creature on the battlefield to target.
	target := NewCreature("Bear", "{1}{G}", 2, 2)
	target.SetOwner(a.PlayerID())
	targetPerm := g.PutOnBattlefield(target, a.PlayerID())

	// The Repartee source — count fires via a side-effect closure.
	fires := 0
	srcCard := NewCreature("Inkwright", "{2}{W}", 2, 2,
		WithAbility(WheneverYouCastInstantOrSorceryTargetingCreatureTrigger(
			FuncEffect("repartee fires", EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					fires++
					return nil
				}),
			false,
		)),
	)
	srcCard.SetOwner(a.PlayerID())
	g.PutOnBattlefield(srcCard, a.PlayerID())
	g.ResolveStack()

	// Cast a fake "instant targeting a creature" by pushing a stack object.
	bolt := NewInstant("Bolt", "{R}",
		NewTargetedSpell(TargetAnyTarget(), DealDamage(Fixed(3))),
	)
	bolt.SetOwner(a.PlayerID())
	obj := &StackObject{
		ID:         uuid.New(),
		Card:       bolt,
		Controller: a.PlayerID(),
		SourceID:   bolt.ID(),
		Effects:    []Effect{DealDamage(Fixed(3))},
		Targets:    []uuid.UUID{targetPerm.Card.ID()},
	}
	g.stack.Push(obj)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: bolt.ID(), PlayerID: a.PlayerID()})

	g.ResolveStack()

	if fires != 1 {
		t.Errorf("expected Repartee to fire once, got %d", fires)
	}
}

// TestRepartee_DoesNotTriggerWhenSpellTargetsPlayer: the trigger should not
// fire when the targeted object is a player rather than a creature.
func TestRepartee_DoesNotTriggerWhenSpellTargetsPlayer(t *testing.T) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B")
	g := NewGame(a, b)

	fires := 0
	srcCard := NewCreature("Inkwright", "{2}{W}", 2, 2,
		WithAbility(WheneverYouCastInstantOrSorceryTargetingCreatureTrigger(
			FuncEffect("repartee fires", EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					fires++
					return nil
				}),
			false,
		)),
	)
	srcCard.SetOwner(a.PlayerID())
	g.PutOnBattlefield(srcCard, a.PlayerID())
	g.ResolveStack()

	bolt := NewInstant("Bolt", "{R}",
		NewTargetedSpell(TargetAnyTarget(), DealDamage(Fixed(3))),
	)
	bolt.SetOwner(a.PlayerID())
	obj := &StackObject{
		ID:         uuid.New(),
		Card:       bolt,
		Controller: a.PlayerID(),
		SourceID:   bolt.ID(),
		Effects:    []Effect{DealDamage(Fixed(3))},
		Targets:    []uuid.UUID{b.PlayerID()}, // player target
	}
	g.stack.Push(obj)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: bolt.ID(), PlayerID: a.PlayerID()})
	g.ResolveStack()

	if fires != 0 {
		t.Errorf("expected Repartee not to fire on player target, got %d", fires)
	}
}

// =============================================================================
// Opus / Increment — mana spent to cast
// =============================================================================

// TestManaSpentToCast_FromCastContext: the helper sums ColorsSpent.
func TestManaSpentToCast_FromCastContext(t *testing.T) {
	obj := &StackObject{
		CastContext: &CastContext{
			ColorsSpent: map[Color]int{Red: 1, Colorless: 3},
		},
	}
	if got := ManaSpentToCast(obj); got != 4 {
		t.Errorf("ManaSpentToCast: got %d, want 4", got)
	}
	if got := ManaSpentToCast(nil); got != 0 {
		t.Errorf("ManaSpentToCast(nil) = %d, want 0", got)
	}
}

// TestIncrement_PutsCounterWhenManaSpentExceedsPower: a 1/1 with the
// Increment trigger gains a +1/+1 counter when its controller casts a spell
// that drained 2 mana from the pool.
func TestIncrement_PutsCounterWhenManaSpentExceedsPower(t *testing.T) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B")
	g := NewGame(a, b)

	src := NewCreature("Pensive Professor", "{1}{U}", 1, 1,
		WithAbility(IncrementTrigger()),
	)
	src.SetOwner(a.PlayerID())
	srcPerm := g.PutOnBattlefield(src, a.PlayerID())
	g.ResolveStack()

	// Push a fake spell with ColorsSpent total = 2 (greater than 1/1 power).
	bolt := NewInstant("Bolt", "{R}", NewSpellAbility(DealDamage(Fixed(2))))
	bolt.SetOwner(a.PlayerID())
	obj := &StackObject{
		ID:         uuid.New(),
		Card:       bolt,
		Controller: a.PlayerID(),
		SourceID:   bolt.ID(),
		Effects:    []Effect{DealDamage(Fixed(2))},
		CastContext: &CastContext{
			ColorsSpent: map[Color]int{Red: 1, Colorless: 1},
		},
	}
	g.stack.Push(obj)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: bolt.ID(), PlayerID: a.PlayerID()})
	g.ResolveStack()

	if got := int(srcPerm.Counters[P1P1]); got != 1 {
		t.Errorf("expected 1 +1/+1 counter, got %d", got)
	}
}

// TestIncrement_NoCounterWhenManaSpentEqualsPower: equal does not exceed.
func TestIncrement_NoCounterWhenManaSpentEqualsPower(t *testing.T) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B")
	g := NewGame(a, b)

	src := NewCreature("Pensive Professor", "{1}{U}", 2, 2,
		WithAbility(IncrementTrigger()),
	)
	src.SetOwner(a.PlayerID())
	srcPerm := g.PutOnBattlefield(src, a.PlayerID())
	g.ResolveStack()

	bolt := NewInstant("Bolt", "{R}", NewSpellAbility(DealDamage(Fixed(0))))
	bolt.SetOwner(a.PlayerID())
	obj := &StackObject{
		ID:          uuid.New(),
		Card:        bolt,
		Controller:  a.PlayerID(),
		SourceID:    bolt.ID(),
		CastContext: &CastContext{ColorsSpent: map[Color]int{Red: 1, Colorless: 1}},
	}
	g.stack.Push(obj)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: bolt.ID(), PlayerID: a.PlayerID()})
	g.ResolveStack()

	if got := int(srcPerm.Counters[P1P1]); got != 0 {
		t.Errorf("expected 0 +1/+1 counters, got %d", got)
	}
}

// TestOpusEffect_RunsWithManaSpent: an OpusEffect closure sees the total
// mana spent from CastContext when its trigger resolves.
func TestOpusEffect_RunsWithManaSpent(t *testing.T) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B")
	g := NewGame(a, b)

	var seen int
	src := NewCreature("Deluge Virtuoso", "{2}{U}", 2, 2,
		WithAbility(WheneverYouCastSpellTrigger(
			OpusEffect("opus", func(g *Game, sourceID, controller uuid.UUID, manaSpent int) error {
				seen = manaSpent
				return nil
			}), false, IsInstantOrSorceryCard,
		)),
	)
	src.SetOwner(a.PlayerID())
	g.PutOnBattlefield(src, a.PlayerID())
	g.ResolveStack()

	bolt := NewInstant("Bolt", "{R}", NewSpellAbility(DealDamage(Fixed(2))))
	bolt.SetOwner(a.PlayerID())
	obj := &StackObject{
		ID: uuid.New(), Card: bolt, Controller: a.PlayerID(), SourceID: bolt.ID(),
		Effects:     []Effect{DealDamage(Fixed(2))},
		CastContext: &CastContext{ColorsSpent: map[Color]int{Red: 1, Colorless: 4}},
	}
	g.stack.Push(obj)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: bolt.ID(), PlayerID: a.PlayerID()})
	g.ResolveStack()

	if seen != 5 {
		t.Errorf("OpusEffect saw mana=%d, want 5", seen)
	}
}

// =============================================================================
// Infusion
// =============================================================================

// TestInfusionEffect_RunsOnlyIfLifeGainedThisTurn:
func TestInfusionEffect_RunsOnlyIfLifeGainedThisTurn(t *testing.T) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B")
	g := NewGame(a, b)

	ran := 0
	inner := FuncEffect("hit", EffectProperties{},
		func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
			ran++
			return nil
		})
	eff := InfusionEffect("infusion", inner)

	// Without life gained: should not run.
	if err := eff.Apply(g, uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if ran != 0 {
		t.Errorf("infusion ran without life gain")
	}

	// Fire EvtLifeGained directly (life-gain effects normally do this; the
	// per-turn tracker keys off the event, not the raw GainLife call).
	g.PlayerGainLife(a, 2)
	g.FireEvent(GameEvent{Type: EvtLifeGained, PlayerID: a.PlayerID(), Amount: 2})
	if !IfControllerGainedLifeThisTurn(g, a.PlayerID()) {
		t.Fatalf("expected lifeGainedThisTurn>0")
	}
	if err := eff.Apply(g, uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if ran != 1 {
		t.Errorf("infusion did not run after life gain")
	}
	if got := LifeGainedThisTurnFor(g, a.PlayerID()); got != 2 {
		t.Errorf("LifeGainedThisTurnFor = %d, want 2", got)
	}
}

// =============================================================================
// Grandeur
// =============================================================================

// TestDiscardAnotherCardNamedSelfCost_PayableOnlyWithMatchingNameInHand:
func TestDiscardAnotherCardNamedSelfCost_PayableOnlyWithMatchingNameInHand(t *testing.T) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B")
	g := NewGame(a, b)

	src := NewCreature("Page, Loose Leaf", "{2}", 2, 2,
		WithSuperTypes(SuperLegendary),
	)
	src.SetOwner(a.PlayerID())
	g.PutOnBattlefield(src, a.PlayerID())

	cost := DiscardAnotherCardNamedSelfCost()

	// No matching named card in hand: cost is unpayable.
	if cost.CanPay(src.ID(), a.PlayerID(), g) {
		t.Errorf("cost should be unpayable with no matching named card")
	}

	// Add another Page, Loose Leaf to hand.
	other := NewCreature("Page, Loose Leaf", "{2}", 2, 2)
	other.SetOwner(a.PlayerID())
	a.AddToHand(other)

	if !cost.CanPay(src.ID(), a.PlayerID(), g) {
		t.Errorf("cost should be payable with another Page, Loose Leaf in hand")
	}

	if err := cost.Pay(src.ID(), a.PlayerID(), g); err != nil {
		t.Fatalf("pay: %v", err)
	}
	// Card should be in graveyard.
	found := false
	for _, c := range a.Graveyard() {
		if c.ID() == other.ID() {
			found = true
		}
	}
	if !found {
		t.Errorf("expected discarded copy in graveyard")
	}
}

// =============================================================================
// Paradigm
// =============================================================================

// TestParadigm_RecordResolutionAndExiledCopy: round-trip the per-game
// trackers used by the Paradigm helpers.
func TestParadigm_RecordResolutionAndExiledCopy(t *testing.T) {
	a := newRecordingPlayer("A")
	b := newRecordingPlayer("B")
	g := NewGame(a, b)

	if g.HasResolvedParadigmSpell(a.PlayerID(), "Restoration Seminar") {
		t.Errorf("expected no resolution recorded initially")
	}
	g.RecordParadigmResolution(a.PlayerID(), "Restoration Seminar")
	if !g.HasResolvedParadigmSpell(a.PlayerID(), "Restoration Seminar") {
		t.Errorf("expected resolution to be recorded")
	}
	if g.HasResolvedParadigmSpell(b.PlayerID(), "Restoration Seminar") {
		t.Errorf("resolution should be per-player")
	}

	cardID := uuid.New()
	g.RegisterParadigmExiledCopy(a.PlayerID(), "Restoration Seminar", cardID)
	got, ok := g.ParadigmExiledCopy(a.PlayerID(), "Restoration Seminar")
	if !ok || got != cardID {
		t.Errorf("ParadigmExiledCopy: got (%v,%v), want (%v,true)", got, ok, cardID)
	}
}
