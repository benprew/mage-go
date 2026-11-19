package mage

import (
	"slices"
	"testing"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestAlternateCastValidatesAnnouncement(t *testing.T) {
	for _, tc := range []struct {
		name        string
		alternate   string
		x           int
		targetValue int
		omitTarget  bool
		wantMana    int
	}{
		{name: "free cast with a mismatched target", targetValue: 1},
		{name: "free cast with nonzero X", x: 1, targetValue: 1},
		{name: "fixed alternative with nonzero X", alternate: "{U}", x: 1, targetValue: 1},
		{name: "alternative with X and a mismatched target", alternate: "{X}{U}", x: 2, targetValue: 1},
		{name: "alternative with X and a missing target", alternate: "{X}{U}", x: 1, targetValue: 1, omitTarget: true},
		{name: "negative X", alternate: "{X}{U}", x: -1, targetValue: 1},
		{name: "free cast with X zero", wantMana: 1},
		{name: "alternative pays for X", alternate: "{X}{U}", x: 1, targetValue: 1, wantMana: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := newModalGame()
			a.ManaPool().Add(Blue, 8)
			cost := "{0}"
			if tc.targetValue == 1 {
				cost = "{R}"
				b.ManaPool().Add(Red, 1)
			}
			target := NewInstant("Stack target", cost, NewSpell(GainLife(1)))
			b.AddToHand(target)
			if err := g.CastSpellByID(b.PlayerID(), target.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			spell := NewInstant("Counter", "{X}{U}",
				NewTargetedSpell(TargetSpellOnStack(ManaValueIs(EqualTo(XValue()))), CounterSpell()),
				WithAdditionalCost(ManaCostOf("{1}")))
			spell.SetOwner(a.PlayerID())
			a.AddToGraveyard(spell)
			g.SetXValue(7)
			var targets []uuid.UUID
			if !tc.omitTarget {
				targets = []uuid.UUID{target.ID()}
			}
			var err error
			if tc.alternate == "" {
				err = g.CastCardFromZoneWithoutPaying(a.PlayerID(), spell.ID(), ZoneGraveyard, targets, tc.x)
			} else {
				err = g.CastCardFromZoneWithAlternateCost(a.PlayerID(), spell.ID(), ZoneGraveyard, ParseManaCost(tc.alternate), targets, tc.x)
			}
			if tc.wantMana == 0 {
				if err == nil {
					t.Fatal("accepted an illegal spell announcement")
				}
				if g.Stack().Size() != 1 || a.ManaPool().TotalMana() != 8 ||
					!slices.Contains(a.Graveyard(), spell) || g.XValue() != 7 {
					t.Fatal("rejected announcement changed game state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if g.Stack().Peek().XValue != tc.x || a.ManaPool().TotalMana() != 8-tc.wantMana {
				t.Fatal("cast did not keep the announced X or pay the total cost")
			}
			g.ResolveStack()
			if b.Life() != 20 {
				t.Fatal("counterspell did not resolve")
			}
		})
	}
}

func TestFreeModalCastRejectsModeWithoutLegalTargets(t *testing.T) {
	g, a, b := newModalGame()
	g.PutOnBattlefield(NewArtifact("Three", "{3}"), b.PlayerID())
	spell := NewSorcery("Removal", "{X}{R}", NewModalSpell([]Mode{
		{Label: "Destroy", Targets: []Target{TargetArtifactWithManaValueX()}, Effects: []Effect{DestroyTarget()}},
		{Label: "Gain life", Effects: []Effect{GainLife(1)}},
	}))
	a.AddToGraveyard(spell)
	if err := g.CastCardFromZoneWithoutPaying(a.PlayerID(), spell.ID(), ZoneGraveyard, nil, 0); err == nil {
		t.Fatal("cast succeeded with no legal targets for the chosen mode")
	}
	if !g.Stack().IsEmpty() || !slices.Contains(a.Graveyard(), spell) {
		t.Fatal("rejected modal cast changed game state")
	}
}

func TestAlternateCostKeepsAnnouncedZero(t *testing.T) {
	g, a, b := newModalGame()
	a.ManaPool().Add(Blue, 1)
	target := NewInstant("Stack target", "{0}")
	b.AddToHand(target)
	if err := g.CastSpellByID(b.PlayerID(), target.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	spell := NewInstant("Counter", "{U}",
		NewTargetedSpell(TargetSpellOnStack(ManaValueIs(EqualTo(XValue()))), CounterSpell()))
	a.AddToGraveyard(spell)
	g.SetXValue(7)
	if err := g.CastCardFromZoneWithAlternateCost(a.PlayerID(), spell.ID(), ZoneGraveyard, ParseManaCost("{X}{U}"), []uuid.UUID{target.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	if got := g.Stack().Peek().XValue; got != 0 || g.XValue() != 7 {
		t.Fatalf("stack and resolution X = %d, %d, want 0, 7", got, g.XValue())
	}
}

func TestSpellTargetOptionsUseCastCosts(t *testing.T) {
	for _, tc := range []struct {
		name       string
		blue       int
		life       int
		channel    bool
		increase   int
		reduction  int
		additional Cost
		wantCast   bool
		wantLife   int
	}{
		{name: "Channel", blue: 1, life: 20, channel: true, wantCast: true, wantLife: 17},
		{name: "Channel with insufficient life", blue: 4, life: 2, channel: true},
		{name: "tax", blue: 4, life: 20, increase: 2},
		{name: "X cost reduction", blue: 2, life: 20, reduction: 2, wantCast: true, wantLife: 20},
		{name: "additional mana", blue: 4, life: 20, additional: ManaCostOf("{U}")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := newModalGame()
			a.ManaPool().Add(Blue, tc.blue)
			a.SetLife(tc.life)
			if tc.channel {
				g.SetChannelActive(a.PlayerID())
			}
			g.effects.Rules.SpellCostIncreases[Blue] = tc.increase
			g.AddSpellTypeCostReduction(TypeInstant, tc.reduction)
			target := NewInstant("Stack target", "{3}")
			b.AddToHand(target)
			b.ManaPool().Add(Red, 3)
			if err := g.CastSpellByID(b.PlayerID(), target.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			spell := NewInstant("Counter", "{X}{U}", NewTargetedSpell(TargetSpellOnStack(ManaValueIs(EqualTo(XValue()))), CounterSpell()))
			if tc.additional != nil {
				WithAdditionalCost(tc.additional)(spell)
			}
			a.AddToHand(spell)
			g.SetXValue(7)
			options := g.SpellTargetOptions(a.PlayerID(), spell)
			if a.Life() != tc.life || a.ManaPool().TotalMana() != tc.blue || g.XValue() != 7 {
				t.Fatal("discovery changed game state")
			}
			if !tc.wantCast {
				if len(options) != 0 {
					t.Fatalf("discovery offered an unaffordable cast: %+v", options)
				}
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 3); err == nil {
					t.Fatal("cast succeeded with an unaffordable cost")
				}
				return
			}
			targets := []uuid.UUID{target.ID()}
			if len(options) != 1 || options[0].XValue != 3 || !slices.Equal(options[0].Candidates[0], targets) {
				t.Fatalf("missing legal X=3 cast: %+v", options)
			}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), targets, 3); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || a.Life() != tc.wantLife {
				t.Fatalf("after payment got %d mana and %d life", a.ManaPool().TotalMana(), a.Life())
			}
		})
	}
}

func TestSpellTargetOptionsChannelWithXExpression(t *testing.T) {
	g, a, b := newModalGame()
	a.ManaPool().Add(Blue, 1)
	b.ManaPool().Add(Red, 2)
	g.SetChannelActive(a.PlayerID())
	target := NewInstant("Stack target", "{2}")
	b.AddToHand(target)
	if err := g.CastSpellByID(b.PlayerID(), target.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	spell := NewInstant("Counter", "{X}{U}",
		NewTargetedSpell(TargetSpellOnStack(ManaValueIs(EqualTo(HalfXRoundedDown()))), CounterSpell()))
	options := g.SpellTargetOptions(a.PlayerID(), spell)
	values := make([]int, 0, len(options))
	for _, option := range options {
		values = append(values, option.XValue)
	}
	if !slices.Equal(values, []int{4, 5}) {
		t.Fatalf("affordable X values = %v, want [4 5]", values)
	}
}

func TestSpellTargetOptionsUseTargetDependentReduction(t *testing.T) {
	g, a, b := newModalGame()
	discounted := NewInstant("Discounted target", "{3}")
	fullPrice := NewInstant("Full price target", "{3}")
	b.ManaPool().Add(Red, 6)
	for _, target := range []*Card{discounted, fullPrice} {
		b.AddToHand(target)
		if err := g.CastSpellByID(b.PlayerID(), target.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
	}
	spell := NewInstant("Counter", "{X}{U}",
		NewTargetedSpell(TargetSpellOnStack(ManaValueIs(EqualTo(XValue()))), CounterSpell()),
		WithTargetConditionalCostReduction(1, func(_ *Game, _ uuid.UUID, _ *Card, targets []uuid.UUID) bool {
			return slices.Contains(targets, discounted.ID())
		}))
	a.ManaPool().Add(Blue, 3)
	a.AddToHand(spell)
	options := g.SpellTargetOptions(a.PlayerID(), spell)
	if len(options) != 1 || options[0].XValue != 3 || !slices.Equal(options[0].Candidates[0], []uuid.UUID{discounted.ID()}) {
		t.Fatalf("expected only the discounted target, got %+v", options)
	}
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), options[0].Candidates[0], options[0].XValue); err != nil {
		t.Fatal(err)
	}
}
