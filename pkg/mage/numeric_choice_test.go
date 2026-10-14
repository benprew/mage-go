package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type numericChoicePlayer struct {
	*BasePlayer
	answer, calls int
	accept        bool
	during        func()
}

func (p *numericChoicePlayer) ChooseNumber(_, _ int, _ string) int {
	p.calls++
	if p.during != nil {
		p.during()
	}
	return p.answer
}
func (p *numericChoicePlayer) ChooseMayAbility(string) bool { return p.accept }

func numericChoiceCard() Card {
	condition := StateConditionFunc("live number", []StateDependency{StateBattlefield}, func(StateReader, StateContext) bool { return true })
	return NewCreature("Number", "", 0, 7,
		WithAbility(ChooseNumberAsEnters(0, 7)),
		WithAbility(Static(While(condition, SetBasePower(This, ChosenNumberOf(This)), SetBaseToughness(This, SubtractValues(Fixed(7), ChosenNumberOf(This)))))),
	)
}
func TestNumericChoiceTimingAndIdentity(t *testing.T) {
	for _, answer := range []int{-1, 0, 3, 7, 8} {
		a := &numericChoicePlayer{BasePlayer: NewBasePlayer("A"), answer: answer, accept: true}
		b := &numericChoicePlayer{BasePlayer: NewBasePlayer("B"), answer: 5, accept: true}
		g := NewGame(a, b)
		card := numericChoiceCard()
		card.SetOwner(b.PlayerID())
		a.during = func() {
			for _, live := range g.AllBattlefield() {
				if live.ID() == card.ID() {
					t.Fatal("entry choice after registration")
				}
			}
		}
		p := g.PutOnBattlefield(card, a.PlayerID())
		a.during = nil
		want := answer
		if answer < 0 || answer > 7 {
			want = 0
		}
		if p.CurrentPower(g) != want || p.CurrentToughness(g) != 7-want || a.calls != 1 || b.calls != 0 {
			t.Fatal("entry bounds, chooser or layers")
		}
		g.PhaseIn(g.PhaseOut(p.ID()))
		if p.CurrentPower(g) != want || a.calls != 1 {
			t.Fatal("phasing repeated choice")
		}
		clone := g.Clone()
		clone.MutablePermanent(p.ID()).ChosenNumber = 6
		clone.ApplyContinuousEffects()
		if clone.FindPermanent(p.ID()).CurrentPower(clone) != 6 || g.FindPermanent(p.ID()).CurrentPower(g) != want {
			t.Fatal("clone isolation")
		}
		ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
		value := ChosenNumberOf(This)
		g.BouncePermanentToHand(g.FindPermanent(p.ID()))
		if value.resolveValue(ctx) != want {
			t.Fatal("departure lost number")
		}
		returned := b.Hand()[0]
		b.RemoveFromHand(returned.ID())
		next := g.PutOnBattlefield(returned, b.PlayerID())
		if next.CurrentPower(g) != 5 || value.resolveValue(ctx) != want {
			t.Fatal("returned visit replaced old number")
		}
		calls := a.calls
		if err := MayChooseNumber(This, 0, 7).Apply(ctx); err != nil {
			t.Fatal(err)
		}
		if a.calls != calls || next.CurrentPower(g) != 5 {
			t.Fatal("update used departed source")
		}
	}
}
func TestNumericChoiceUpdate(t *testing.T) {
	a := &numericChoicePlayer{BasePlayer: NewBasePlayer("A"), answer: 2, accept: true}
	b := &numericChoicePlayer{BasePlayer: NewBasePlayer("B"), answer: 5, accept: true}
	g := NewGame(a, b)
	p := g.PutOnBattlefield(numericChoiceCard(), a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	effect := MayChooseNumber(This, 0, 7)
	a.accept = false
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.calls != 1 || p.CurrentPower(g) != 2 {
		t.Fatal("decline changed state")
	}
	a.accept = true
	a.answer = 4
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(p.ID()).CurrentPower(g) != 4 || a.calls != 2 || b.calls != 0 {
		t.Fatal("pending controller or layer refresh")
	}
	g.PhaseOut(p.ID())
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.calls != 2 {
		t.Fatal("phased subject made choice")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	a.during = func() { g.BouncePermanentToHand(g.FindPermanent(p.ID())) }
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(p.ID()) != nil {
		t.Fatal("choice recheck restored subject")
	}
}
func TestNumericChoiceInvalid(t *testing.T) {
	for _, ability := range []Ability{ChooseNumberAsEnters(-1, 7), ChooseNumberAsEnters(7, 0)} {
		if err := ValidateCardDefinition("Invalid", NewCreature("Invalid", "", 1, 1, WithAbility(ability))); err == nil {
			t.Fatal("invalid bounds accepted")
		}
	}
	if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithAbility(ChooseNumberAsEnters(0, 7)))); err == nil {
		t.Fatal("spell entry accepted")
	}
	for _, effect := range []Effect{MayChooseNumber(nil, 0, 7), MayChooseNumber(This, 7, 0)} {
		if err := ValidateCardDefinition("Invalid", NewCreature("Invalid", "", 1, 1, WithAbility(NewActivated(ManaCostOf(""), effect)))); err == nil {
			t.Fatal("invalid update accepted")
		}
	}
}

func TestNumericChoiceCopyAndBatch(t *testing.T) {
	a := &numericChoicePlayer{BasePlayer: NewBasePlayer("A"), answer: 2, accept: true}
	b := &numericChoicePlayer{BasePlayer: NewBasePlayer("B"), answer: 5, accept: true}
	g := NewGame(a, b)
	plan, err := NewCardDefinitionPlan(numericChoiceCard)
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	donor := g.PutOnBattlefield(card, b.PlayerID())
	copyPlan, err := NewCardDefinitionPlan(func() Card { return NewCreature("Copy", "", 1, 1, WithAbility(MayCopyAsEnters(IsCreature))) })
	if err != nil {
		t.Fatal(err)
	}
	copyCard, err := copyPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	copied := g.PutOnBattlefield(copyCard, a.PlayerID())
	if copied == nil || copied.Name() != "Number" || copied.CurrentPower(g) != 2 || donor.CurrentPower(g) != 5 || a.calls != 1 || b.calls != 1 {
		t.Fatal("copy inherited choice or ignored acquired entry rule")
	}
	observed := 0
	watcher := NewArtifact("Watcher", "", WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(event *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		if event.ToZone != core.ZoneBattlefield {
			return false
		}
		p := reader.FindPermanent(event.SourceID)
		if p == nil || p.Name() != "Number" {
			return false
		}
		observed++
		count := 0
		for _, candidate := range reader.AllBattlefield() {
			if candidate.Name() == "Number" {
				count++
			}
		}
		if count != 4 || a.calls != 3 {
			t.Fatal("batch observer ran before all choices and entries")
		}
		return false
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	a.answer = 3
	entries := g.putTokensOnBattlefieldTogether([]Card{numericChoiceCard(), numericChoiceCard()}, a.PlayerID())
	if len(entries) != 2 || observed != 2 {
		t.Fatal("batch entry count")
	}
	for _, p := range entries {
		if p.CurrentPower(g) != 3 {
			t.Fatal("batch choices")
		}
	}
	missing := NewPermanent(NewArtifact("Missing", ""), uuid.New())
	if !ChooseNumberAsEnters(2, 7).OnEnter(g, missing) || !missing.HasChosenNumber || missing.ChosenNumber != 2 {
		t.Fatal("missing player fallback")
	}
}

func TestNumericChoiceLayerOrderAndNoChoice(t *testing.T) {
	a := &numericChoicePlayer{BasePlayer: NewBasePlayer("A"), answer: 3, accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	p := g.PutOnBattlefield(numericChoiceCard(), a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	if err := Until(core.EndOfTurn, ModifyPT(This, Fixed(2), Fixed(1)), SwitchPowerToughness(This)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	p = g.FindPermanent(p.ID())
	p.AddCounter(core.P1P1, 1)
	g.ApplyContinuousEffects()
	if p.CurrentPower(g) != 6 || p.CurrentToughness(g) != 6 || a.calls != 1 {
		t.Fatal("base, bonus, counter and switch order")
	}
	a.answer = 1
	if err := MayChooseNumber(This, 0, 7).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	p = g.FindPermanent(p.ID())
	if p.CurrentPower(g) != 8 || p.CurrentToughness(g) != 4 {
		t.Fatal("live values lost later layers")
	}
	blank := g.PutOnBattlefield(NewArtifact("Blank", ""), a.PlayerID())
	blank.ChosenNumber = 99
	if ChosenNumberOf(This).resolveValue(&EffectContext{Game: g, SourceID: blank.ID()}) != 0 {
		t.Fatal("missing choice read as present")
	}
	g.PhaseOut(p.ID())
	if ChosenNumberOf(This).resolveValue(ctx) != 1 {
		t.Fatal("phase information lost choice")
	}
}
