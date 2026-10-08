package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func entryCopyPlan(t *testing.T, name string, options func() []CardOption) *CardDefinitionPlan {
	t.Helper()
	plan, err := NewCardDefinitionPlan(func() Card { return NewCreature(name, "", 2, 2, options()...) })
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestEntryCopyDeclaration(t *testing.T) {
	if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithAbility(MayCopyAsEnters(IsCreature)))); err == nil {
		t.Fatal("nonpermanent copy accepted")
	}
}

func TestEntryCopyAcquiredRules(t *testing.T) {
	a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	donorPlan := entryCopyPlan(t, "Donor", func() []CardOption {
		return []CardOption{WithKeyword(core.EntersTapped), WithAbility(EntersWithNCounters(core.P1P1, 3))}
	})
	donor, err := donorPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(donor, b.PlayerID())
	source.AddCounter(core.P1P1, 4)
	plan, err := NewCardDefinitionPlan(func() Card {
		return NewCreature("Physical", "", 0, 1, WithAbility(MayCopyAsEnters(IsCreature)), WithAbility(EntersWithNCounters(core.Javelin, 1)))
	})
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	if p == nil || p.Name() != "Donor" || !p.Tapped || p.Counters[core.P1P1] != 3 || p.Counters[core.Javelin] != 0 {
		t.Fatal("copy did not replace physical entry rules")
	}
	values, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := values.Build()
	if err != nil || rebuilt.Name() != "Donor" {
		t.Fatal("copy did not retain provenance", err)
	}
}

type entryCopyPlayer struct {
	*BasePlayer
	pick                    func([]*Permanent, *Game) *Permanent
	offers, choices, orders int
}

func (p *entryCopyPlayer) ChooseMayAbility(string) bool { p.offers++; return true }
func (p *entryCopyPlayer) ChoosePermanent(c []*Permanent, _ string, g GameReader) *Permanent {
	p.choices++
	return p.pick(c, g.(*Game))
}
func (p *entryCopyPlayer) ChooseMode([]string, string) int { p.orders++; return 1 }

func TestEntryCopyRecheck(t *testing.T) {
	for _, mode := range []string{"phase", "return", "type", "invalid", "nil"} {
		t.Run(mode, func(t *testing.T) {
			a := &entryCopyPlayer{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			donorPlan := entryCopyPlan(t, "Donor", func() []CardOption { return nil })
			donor, err := donorPlan.Build()
			if err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(donor, b.PlayerID())
			a.pick = func(c []*Permanent, g *Game) *Permanent {
				switch mode {
				case "phase":
					g.PhaseOut(source.ID())
				case "return":
					g.BouncePermanentToHand(source)
					b.RemoveFromHand(donor.ID())
					g.PutOnBattlefield(donor, b.PlayerID())
				case "type":
					source.RevokeBaseAttr(core.AttrIsCreature)
				case "invalid":
					return NewPermanent(NewCreature("Invalid", "", 4, 4), a.PlayerID())
				case "nil":
					return nil
				}
				return source
			}
			plan, err := NewCardDefinitionPlan(func() Card { return NewCreature("Physical", "", 0, 1, WithAbility(MayCopyAsEnters(IsCreature))) })
			if err != nil {
				t.Fatal(err)
			}
			card, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			want := "Physical"
			if mode == "invalid" || mode == "nil" {
				want = "Donor"
			}
			if p == nil || p.Name() != want || a.offers != 1 || a.choices != 1 {
				t.Fatal("copy failed exact choice recheck")
			}
		})
	}
}

func TestEntryCopyPriorityAndRepetition(t *testing.T) {
	a := &entryCopyPlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	donorPlan := entryCopyPlan(t, "Donor", func() []CardOption { return nil })
	donor, err := donorPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(donor, b.PlayerID())
	a.pick = func(c []*Permanent, _ *Game) *Permanent { return source }
	plan, err := NewCardDefinitionPlan(func() Card {
		return NewCreature("Physical", "", 0, 1, WithAbility(MayCopyAsEnters(IsCreature)), WithAbility(MayCopyAsEnters(IsCreature)))
	})
	if err != nil {
		t.Fatal(err)
	}
	sawCopy := false
	g.effects.AddReplacement(&testPrioritizedReplacement{replacementBase: replacementBase{priority: ReplacementPriorityDefault}, matchFunc: func(action Action) bool { _, ok := action.(*battlefieldEntryAction); return ok }, replaceFunc: func(action Action, _ *Game) Action {
		p := action.(*battlefieldEntryAction).permanent
		sawCopy = p.Name() == "Donor"
		return action
	}})
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	if p == nil || !sawCopy || a.orders != 1 || a.offers != 1 {
		t.Fatal("copy tier, affected player order or vanished rule recheck")
	}

	cyclicPlan, err := NewCardDefinitionPlan(func() Card { return NewCreature("Cycle", "", 0, 1, WithAbility(MayCopyAsEnters(IsCreature))) })
	if err != nil {
		t.Fatal(err)
	}
	cycle, err := cyclicPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	decline := &optionalEntryPlayer{BasePlayer: b, accept: false}
	g.SetPlayerAt(1, decline)
	saved := g.PutOnBattlefield(cycle, b.PlayerID())
	acquiredPlan, err := NewCardDefinitionPlan(func() Card {
		return NewCreature("Acquired copy", "", 0, 1, WithAbility(MayCopyAsEnters(IsCreature)))
	})
	if err != nil {
		t.Fatal(err)
	}
	acquired, err := acquiredPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	a.pick = func([]*Permanent, *Game) *Permanent { return saved }
	beforeAcquired := a.offers
	if result := g.PutOnBattlefield(acquired, a.PlayerID()); result == nil || result.Name() != "Cycle" || a.offers != beforeAcquired+2 {
		t.Fatal("acquired declaration did not get its own replacement opportunity")
	}
	a.pick = func([]*Permanent, *Game) *Permanent { return saved }
	copyCard, err := cyclicPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	before := a.offers
	p = g.PutOnBattlefield(copyCard, a.PlayerID())
	if p == nil || p.Name() != "Cycle" || a.offers != before+1 {
		t.Fatal("reconstructed rule repeated within one entry")
	}
	before = a.offers
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(p.ID())
	g.PutOnBattlefield(copyCard, a.PlayerID())
	if a.offers != before+1 {
		t.Fatal("new entry retained applied replacement state")
	}
}

func TestEntryCopyBatch(t *testing.T) {
	a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	donorPlan := entryCopyPlan(t, "Donor", func() []CardOption { return []CardOption{WithAbility(EntersWithNCounters(core.P1P1, 3))} })
	donor, err := donorPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(donor, b.PlayerID())
	source.AddCounter(core.P1P1, 4)
	counterDonor := g.PutOnBattlefield(NewArtifact("Counter donor", ""), b.PlayerID())
	g.AddCounterDoubler(counterDonor.ID(), core.P1P1, IsCreature)
	plan, err := NewCardDefinitionPlan(func() Card { return NewCreature("Physical", "", 0, 1, WithAbility(MayCopyAsEnters(IsCreature))) })
	if err != nil {
		t.Fatal(err)
	}
	cards := make([]Card, 0, 2)
	for range 2 {
		card, err := plan.Build()
		if err != nil {
			t.Fatal(err)
		}
		cards = append(cards, card)
	}
	entered := g.putTokensOnBattlefieldTogether(cards, a.PlayerID())
	if len(entered) != 2 || a.offers != 2 {
		t.Fatal("batch choices")
	}
	for _, p := range entered {
		if p.Name() != "Donor" || p.Counters[core.P1P1] != 6 {
			t.Fatal("copy batch or acquired counter replacement")
		}
	}
	entered[0].RemoveCounter(core.P1P1, 1)
	if entered[1].Counters[core.P1P1] != 6 || source.Counters[core.P1P1] != 7 {
		t.Fatal("batch shared counters")
	}
}
