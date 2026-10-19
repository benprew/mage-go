package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestNativeMixedDelayedDeclaration(t *testing.T) {
	target := DeclarePermanentTarget("subject", AnyPermanent)
	keywords := []core.Attr{core.Trample}
	adjustments := []PTValueAdjustment{{Power: PowerOfWithLKI(target), Toughness: Fixed(0)}}
	effect := (TemporaryMixedLayerInstruction{Subject: target, Keywords: keywords, Adjustments: adjustments}).Effect()
	keywords[0] = core.Flying
	adjustments[0].Power = Fixed(99)
	got, ok := InspectTemporaryMixedLayerInstruction(effect)
	if !ok || got.Subject != target || got.Keywords[0] != core.Trample {
		t.Fatal("lost declaration")
	}
	got.Keywords[0] = core.Flying
	got.Adjustments[0].Power = Fixed(88)
	again, _ := InspectTemporaryMixedLayerInstruction(effect)
	if again.Keywords[0] != core.Trample {
		t.Fatal("aliased keyword list")
	}
	if _, ok := again.Adjustments[0].Power.(permanentInformationValue); !ok {
		t.Fatal("aliased value list")
	}
	for _, invalid := range []TemporaryMixedLayerInstruction{
		{}, {Subject: target, Keywords: []core.Attr{core.Trample}},
		{Subject: target, Adjustments: again.Adjustments},
		{Subject: (*PermanentTargetBinding)(nil), Keywords: again.Keywords, Adjustments: again.Adjustments},
		{Subject: target, Keywords: []core.Attr{core.AttrSummonSick}, Adjustments: again.Adjustments},
		{Subject: target, Keywords: again.Keywords, Adjustments: []PTValueAdjustment{{Power: Fixed(1)}}},
		{Subject: target, Keywords: again.Keywords, Adjustments: []PTValueAdjustment{{Power: (*damageInspectionValueTrap)(nil), Toughness: Fixed(0)}}},
	} {
		if invalid.Effect() != nil {
			t.Fatal("invalid mixed layer accepted")
		}
	}
	body := Destroy(target)
	delay := (NextEndStepInstruction{Body: body}).Effect()
	if got, ok := InspectNextEndStepInstruction(delay); !ok || got.Body != body {
		t.Fatal("lost delayed body")
	}
	if (NextEndStepInstruction{}).Effect() != nil || (NextEndStepInstruction{Body: (*sequenceEffect)(nil)}).Effect() != nil {
		t.Fatal("invalid delayed body")
	}
	for _, bad := range []Effect{nil, (*nextStepEffect)(nil), AtEach(EachEndStep, body), AtNextThisTurn(EachEndStep, body), AtNext(YourUpkeep, body), AtNext(EachEndStep, nil), &nextStepEffect{pattern: EachEndStep, body: body, condition: ControllerLifeAtLeastStartingLife()}} {
		if _, ok := InspectNextEndStepInstruction(bad); ok {
			t.Fatal("invalid schedule accepted")
		}
	}
	for _, bad := range []Effect{nil, Until(core.EndOfTurn, ModifyPT(target, Fixed(1), Fixed(0))), Sequence(effect)} {
		if _, ok := InspectTemporaryMixedLayerInstruction(bad); ok {
			t.Fatal("undeclared layer accepted")
		}
	}
}

func TestNativeMixedDelayedInspection(t *testing.T) {
	for _, state := range []string{"alone", "prefix", "suffix", "delay prefix", "delay suffix", "opaque value", "foreign value", "foreign condition", "foreign body", "opaque body", "opaque alternative", "opaque filter", "unknown illegal", "duplicate", "copy", "clone", "nested"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			value := PowerOfWithLKI(target)
			filter := AttackedThisTurn
			subject := PermanentRef(target)
			body := Destroy(target)
			known, want := true, false
			switch state {
			case "opaque value":
				value = damageInspectionValueTrap{}
				known = false
			case "foreign value":
				value = PowerOfWithLKI(DeclarePermanentTarget("foreign", AnyPermanent))
				known = false
			case "foreign condition":
				subject = DeclarePermanentTarget("foreign", AnyPermanent)
				known = false
			case "foreign body":
				body = Destroy(DeclarePermanentTarget("foreign", AnyPermanent))
				known = false
			case "opaque body", "unknown illegal":
				body = destructionInspectionTrap{}
				known = false
			case "opaque filter":
				filter = Or(AnyPermanent, NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter ran") }))
				known = false
			}
			branch := PermanentBranchInstruction{Subject: subject, Filter: filter, Matching: body}
			if state == "opaque alternative" {
				branch.Nonmatching = destructionInspectionTrap{}
				known = false
			}
			if state == "nested" {
				branch.Matching = (PermanentBranchInstruction{Subject: target, Filter: Not(AttackedThisTurn), Matching: body}).Effect()
			}
			delay := (NextEndStepInstruction{Body: Sequence(branch.Effect())}).Effect()
			mixed := (TemporaryMixedLayerInstruction{Subject: target, Keywords: []core.Attr{core.Trample}, Adjustments: []PTValueAdjustment{{Power: value, Toughness: Fixed(0)}}}).Effect()
			obj.Effects = []Effect{mixed, delay}
			switch state {
			case "prefix":
				obj.Effects = append(obj.Effects, Destroy(target))
				known = false
			case "suffix", "copy", "clone":
				obj.Effects = append([]Effect{Destroy(target)}, obj.Effects...)
				want = true
			case "delay prefix":
				obj.Effects = []Effect{delay, Destroy(target)}
				want = true
			case "delay suffix":
				obj.Effects = []Effect{Destroy(target), delay}
				want = true
			case "unknown illegal":
				p.GrantBaseAttr(core.Shroud)
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
				obj.Targets = append(obj.Targets, p.ID())
				known = false
			}
			if state == "copy" {
				obj = g.CopyStackObjectDirect(obj, g.GetOpponent(obj.Controller).PlayerID(), false)
			}
			if state == "clone" {
				g = g.Clone()
				obj = g.StackPeek()
			}
			for range 2 {
				certain, k := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}})
				if certain != want || k != known {
					t.Fatalf("got %v/%v want %v/%v", certain, k, want, known)
				}
			}
			if len(g.effects.effects) != 0 || len(g.DelayedTriggers()) != 0 || g.FindPermanent(p.ID()) == nil {
				t.Fatal("inspection executed instructions")
			}
		})
	}
}

func TestNativeMixedDelayedInstallation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Subject", "", 3, 6)
	p := g.PutOnBattlefield(card, a.PlayerID())
	effect := (TemporaryMixedLayerInstruction{Subject: This, Keywords: []core.Attr{core.Trample, core.FirstStrike}, Adjustments: []PTValueAdjustment{{Power: PowerOfWithLKI(This), Toughness: Fixed(0)}, {Power: PowerOfWithLKI(This), Toughness: Fixed(1)}}}).Effect()
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if p.CurrentPower(g) != 9 || p.CurrentToughness(g) != 7 || !p.HasKeyword(core.Trample) || !p.HasKeyword(core.FirstStrike) || len(g.effects.effects) != 2 {
		t.Fatal("mixed layer did not capture once in one installation")
	}
	clone := g.Clone()
	g.PhaseIn(g.PhaseOut(p.ID()))
	g.MutablePermanent(p.ID()).AddCounter(core.P1P1, 1)
	if g.FindPermanent(p.ID()).CurrentPower(g) != 10 {
		t.Fatal("value followed later power")
	}
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if returned.CurrentPower(g) != 3 || returned.HasKeyword(core.Trample) {
		t.Fatal("layer followed return")
	}
	if clone.FindPermanent(p.ID()).CurrentPower(clone) != 9 {
		t.Fatal("clone shared layer state")
	}
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if clone.FindPermanent(p.ID()).CurrentPower(clone) != 3 || clone.FindPermanent(p.ID()).HasKeyword(core.Trample) {
		t.Fatal("layer survived cleanup")
	}
}
