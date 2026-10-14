package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type entrySacrificePlayer struct {
	*BasePlayer
	count, calls int
	mode, orders int
	during       func([]*Permanent)
}

func (p *entrySacrificePlayer) ChooseNumber(_, _ int, _ string) int { p.calls++; return p.count }
func (*entrySacrificePlayer) ChooseMayAbility(string) bool          { return true }
func (p *entrySacrificePlayer) ChooseMode([]string, string) int     { p.orders++; return p.mode }
func (p *entrySacrificePlayer) ChoosePermanent(candidates []*Permanent, _ string, _ GameReader) *Permanent {
	if p.during != nil {
		p.during(candidates)
	}
	return candidates[0]
}

func entrySacrificeCard() Card {
	return NewCreature("Entry sacrifice", "", 0, 0,
		WithAbility(SacrificeAnyNumberAsEnters(And(IsLand, IsUntapped), IsLand)),
		WithStaticAbility(PTEqualsEntrySacrificeCount()))
}

func TestEntrySacrificeBatchAndHistory(t *testing.T) {
	a := &entrySacrificePlayer{BasePlayer: NewBasePlayer("A"), count: 2}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	lands := make([]*Permanent, 0, 3)
	for range 3 {
		lands = append(lands, g.PutOnBattlefield(NewLand("Land"), a.PlayerID()))
	}
	lands[0].Card.SetOwner(b.PlayerID())
	g.AddExileIfWouldGoToGraveyardThisTurn(lands[1].ID(), uuid.New())
	card := entrySacrificeCard()
	a.during = func(_ []*Permanent) {
		if len(g.AllBattlefield()) != 3 {
			t.Fatal("choice after entry or movement")
		}
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	a.during = nil
	if p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 2 || len(b.Graveyard()) != 1 || len(g.GetExile()) != 1 || a.calls != 1 {
		t.Fatal("completed count, owner or replacement", p.CurrentPower(g))
	}
	g.PhaseIn(g.PhaseOut(p.ID()))
	if p.CurrentPower(g) != 2 || a.calls != 1 {
		t.Fatal("phase changed history")
	}
	clone := g.Clone()
	if clone.FindPermanent(p.ID()).CurrentPower(clone) != 2 {
		t.Fatal("clone lost history")
	}
	clone.MutablePermanent(p.ID()).EntrySacrificeCount = 7
	clone.ApplyContinuousEffects()
	if clone.FindPermanent(p.ID()).CurrentPower(clone) != 7 || p.CurrentPower(g) != 2 {
		t.Fatal("shared history")
	}
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	g.BouncePermanentToHand(p)
	if EntrySacrificeCountOf(This).resolveValue(ctx) != 2 {
		t.Fatal("lost departure history")
	}
	if power, toughness := g.CardPowerToughness(a.Hand()[0]); power != 0 || toughness != 0 {
		t.Fatal("new zone kept history")
	}
	a.RemoveFromHand(card.ID())
	a.count = 1
	next := g.PutOnBattlefield(card, a.PlayerID())
	if next.CurrentPower(g) != 1 || EntrySacrificeCountOf(This).resolveValue(ctx) != 2 {
		t.Fatal("return changed saved visit")
	}
}

func TestEntrySacrificeChoiceRechecks(t *testing.T) {
	for _, change := range []string{"return", "phase", "control", "tap", "cancel"} {
		t.Run(change, func(t *testing.T) {
			a := &entrySacrificePlayer{BasePlayer: NewBasePlayer("A"), count: 1}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			land := g.PutOnBattlefield(NewLand("Land"), a.PlayerID())
			g.PutOnBattlefield(NewLand("Spare"), a.PlayerID())
			a.during = func(_ []*Permanent) {
				switch change {
				case "return":
					g.RemoveFromBattlefield(land)
					g.PutOnBattlefield(land.Card, a.PlayerID())
				case "phase":
					g.PhaseOut(land.ID())
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: land.ID(), TargetID: land.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "tap":
					g.TapPermanent(land)
				case "cancel":
					g.effects.AddReplacement(&entrySacrificeCancel{replacementBase: replacementBase{sourceID: land.ID()}})
				}
			}
			p := g.PutOnBattlefield(entrySacrificeCard(), a.PlayerID())
			if p.CurrentPower(g) != 0 || g.FindPermanentIncludingPhased(land.ID()) == nil {
				t.Fatal("invalid choice counted or moved")
			}
		})
	}
}

type entrySacrificeCancel struct{ replacementBase }

func (*entrySacrificeCancel) IsActive(GameReader) bool { return true }
func (*entrySacrificeCancel) Matches(a Action, _ GameReader) bool {
	_, ok := a.(*SacrificePermanentAction)
	return ok
}
func (*entrySacrificeCancel) Replace(Action, *Game) Action { return nil }
func (r *entrySacrificeCancel) Clone() ReplacementEffect   { c := *r; return &c }

func TestEntrySacrificeInvalid(t *testing.T) {
	if err := ValidateCardDefinition("Bad", NewCreature("Bad", "", 1, 1, WithAbility(SacrificeAnyNumberAsEnters(IsLand, IsLand)), WithAbility(SacrificeAnyNumberAsEnters(IsLand, IsLand)))); err == nil {
		t.Fatal("duplicate entry results accepted")
	}
	if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", WithAbility(SacrificeAnyNumberAsEnters(IsLand, IsLand)))); err == nil {
		t.Fatal("spell entry accepted")
	}
	if err := ValidateCardDefinition("Bad", NewSorcery("Bad", "", Spell(GainLifeFor(You, EntrySacrificeCountOf(nil))))); err == nil {
		t.Fatal("nil subject accepted")
	}
}

func TestEntrySacrificeCountedFilterAndLayers(t *testing.T) {
	a := &entrySacrificePlayer{BasePlayer: NewBasePlayer("A"), count: 2}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	forest := g.PutOnBattlefield(NewLand("Forest", WithSubTypes("Forest")), a.PlayerID())
	g.PutOnBattlefield(NewLand("Plains", WithSubTypes("Plains")), a.PlayerID())
	card := NewCreature("Counted", "", 0, 0, WithAbility(SacrificeAnyNumberAsEnters(IsLand, HasSubType("Forest"))), WithStaticAbility(PTEqualsEntrySacrificeCount()))
	observed := 0
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtSacrifice, false).SetCondition(func(_ *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		observed++
		if len(g.AllBattlefield()) != 1 || reader.FindPermanent(forest.ID()) == nil {
			t.Fatal("partial batch or missing former battlefield")
		}
		return false
	}))), a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	if p.CurrentPower(g) != 1 || observed != 2 {
		t.Fatal("filtered history or batch observations")
	}
	g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.EndOfTurn))
	g.ApplyContinuousEffects()
	if p.CurrentPower(g) != 0 {
		t.Fatal("removed CDA stayed active")
	}
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if p.CurrentPower(g) != 1 {
		t.Fatal("CDA restoration lost result")
	}
	g.AddCountersWithReplacement(p, core.P1P1, 2, p.ID(), false)
	if p.CurrentPower(g) != 3 {
		t.Fatal("CDA used the wrong layer")
	}
}

func TestEntrySacrificeReplacementOrder(t *testing.T) {
	for _, mode := range []int{0, 1} {
		a := &entrySacrificePlayer{BasePlayer: NewBasePlayer("A"), count: 2, mode: mode}
		g := NewGame(a, NewBasePlayer("B"))
		for range 2 {
			g.PutOnBattlefield(NewLand("Land"), a.PlayerID())
		}
		seen := -1
		g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(action Action) bool { _, ok := action.(*battlefieldEntryAction); return ok }, replaceFunc: func(action Action, _ *Game) Action {
			seen = action.(*battlefieldEntryAction).permanent.EntrySacrificeCount
			return action
		}})
		p := g.PutOnBattlefield(entrySacrificeCard(), a.PlayerID())
		if seen != mode*2 || p.CurrentPower(g) != 2 || a.calls != 1 || a.orders != 1 {
			t.Fatal("entry ordering or repeated application", seen, a.calls, a.orders)
		}
	}
}

func TestEntrySacrificeCopyAcquisition(t *testing.T) {
	a := &entrySacrificePlayer{BasePlayer: NewBasePlayer("A"), count: 1}
	b := &entrySacrificePlayer{BasePlayer: NewBasePlayer("B"), count: 2}
	g := NewGame(a, b)
	for range 2 {
		g.PutOnBattlefield(NewLand("B land"), b.PlayerID())
	}
	donorPlan, err := NewCardDefinitionPlan(entrySacrificeCard)
	if err != nil {
		t.Fatal(err)
	}
	card, err := donorPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	donor := g.PutOnBattlefield(card, b.PlayerID())
	g.PutOnBattlefield(NewLand("A land"), a.PlayerID())
	copyPlan, err := NewCardDefinitionPlan(func() Card { return NewCreature("Copy", "", 1, 1, WithAbility(MayCopyAsEnters(IsCreature))) })
	if err != nil {
		t.Fatal(err)
	}
	copyCard, err := copyPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(copyCard, a.PlayerID())
	if p.Name() != donor.Name() || p.CurrentPower(g) != 1 || donor.CurrentPower(g) != 2 || a.calls != 1 {
		t.Fatal("copied entry result instead of a fresh choice")
	}
}

func TestEntrySacrificeBatchEntrants(t *testing.T) {
	a := &entrySacrificePlayer{BasePlayer: NewBasePlayer("A"), count: 2}
	b := &entrySacrificePlayer{BasePlayer: NewBasePlayer("B"), count: 1}
	g := NewGame(a, b)
	for range 2 {
		g.PutOnBattlefield(NewLand("A land"), a.PlayerID())
	}
	g.PutOnBattlefield(NewLand("B land"), b.PlayerID())
	first, second := entrySacrificeCard(), entrySacrificeCard()
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	a.AddToGraveyard(first)
	b.AddToGraveyard(second)
	observed := 0
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(event *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		if event.ToZone != core.ZoneBattlefield {
			return false
		}
		if event.SourceID != first.ID() && event.SourceID != second.ID() {
			return false
		}
		observed++
		p, q := reader.FindPermanent(first.ID()), reader.FindPermanent(second.ID())
		if p == nil || q == nil || p.CurrentPower(reader) != 2 || q.CurrentPower(reader) != 1 {
			t.Fatal("entry observation before all results")
		}
		return false
	}))), a.PlayerID())
	if err := ApplyEffect(g, PutGraveyardCreaturesOntoBattlefieldUnderOwnersControl(), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if observed != 2 || a.calls != 1 || b.calls != 1 {
		t.Fatal("batch entry choices")
	}
}

func TestEntrySacrificeWaitingRead(t *testing.T) {
	a := &entrySacrificePlayer{BasePlayer: NewBasePlayer("A"), count: 1}
	g := NewGame(a, NewBasePlayer("B"))
	g.PutOnBattlefield(NewLand("Land"), a.PlayerID())
	p := g.PutOnBattlefield(entrySacrificeCard(), a.PlayerID())
	saved := g.CaptureDamageSource(p.ID())
	if err := ApplyEffect(g, AtNext(EachEndStep, GainLifeFor(You, EntrySacrificeCountOf(This))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(p)
	for range 2 {
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(saved) || a.calls != 1 {
			t.Fatal("waiting query lost source or made a choice")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("delayed read lost entry history")
	}
}
