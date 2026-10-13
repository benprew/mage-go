package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestEntryAnnouncedXDirect(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		g, a, _ := newSpellCopyGame()
		ability := EntersWithCounters(core.P1P1, AnnouncedXForEnteringPermanent())
		if legacy {
			ability = EntersWithXCounters(core.P1P1)
		}
		g.SetXValue(19)
		p := g.PutOnBattlefield(NewCreature("Entry", "{X}", 0, 1, WithAbility(ability)), a.PlayerID())
		if p.Counters[core.P1P1] != 0 || g.XValue() != 19 {
			t.Fatal("direct entry inherited or changed ambient X")
		}
	}
}

func TestEntryAnnouncedXSpellAndReturn(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		g, a, b := newSpellCopyGame()
		g.SetStep(core.PrecombatMain)
		g.SetTurn(1)
		ability := EntersWithCounters(core.P1P1, AnnouncedXForEnteringPermanent())
		if legacy {
			ability = EntersWithXCounters(core.P1P1)
		}
		c := NewCreature("Entry", "{X}", 0, 1, WithAbility(ability))
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		a.ManaPool().Add(core.Colorless, 3)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 4); err == nil {
			t.Fatal("unpaid X accepted")
		}
		if len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 3 || g.StackSize() != 0 {
			t.Fatal("failed X payment changed state")
		}
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 3); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(c.ID()) != nil {
			t.Fatal("entered before resolution")
		}
		cp := g.CopySpellOnStack(c.ID(), b.PlayerID(), false)
		if cp == nil || cp.XValue != 3 {
			t.Fatal("copy lost X")
		}
		clone := g.Clone()
		g.SetXValue(29)
		g.ResolveStack()
		if g.FindPermanent(c.ID()).Counters[core.P1P1] != 3 || g.FindPermanent(cp.SourceID).Counters[core.P1P1] != 3 {
			t.Fatal("entry lost announced X")
		}
		p := g.FindPermanent(c.ID())
		visit := p.incarnationID
		g.PhaseIn(g.PhaseOut(p.ID()))
		if g.FindPermanent(p.ID()).Counters[core.P1P1] != 3 {
			t.Fatal("phasing repeated entry")
		}
		g.BouncePermanentToHand(g.FindPermanent(p.ID()))
		if g.zones.departedCounters[visit][core.P1P1] != 3 {
			t.Fatal("departure lost counters")
		}
		a.RemoveFromHand(c.ID())
		g.SetXValue(11)
		returned := g.PutOnBattlefield(c, a.PlayerID())
		if returned.Counters[core.P1P1] != 0 || returned.incarnationID == visit || g.XValue() != 11 {
			t.Fatal("return inherited X or identity")
		}
		clone.ResolveStack()
		if clone.FindPermanent(c.ID()).Counters[core.P1P1] != 3 || clone.FindPermanent(cp.SourceID).Counters[core.P1P1] != 3 {
			t.Fatal("clone lost entry X")
		}
	}
}

func TestEntryAnnouncedXContext(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	g.SetXValue(99)
	c := NewCreature("Entry facts", "{X}", 0, 1,
		WithAbility(EntersWithCounters(core.P1P1, AnnouncedXForEnteringPermanent())),
		WithAbility(EntersWithCounters(core.Charge, ColorsSpentToCastEnteringPermanent())))
	p := g.prepareBattlefieldEntryWithContext(c, a.PlayerID(), nil, battlefieldEntryContext{announcedX: 3, castColorsSpent: 2}, false)
	if p == nil || p.Counters[core.P1P1] != 3 || p.Counters[core.Charge] != 2 || g.XValue() != 99 {
		t.Fatal("entry facts were confused with each other or ambient X")
	}
	if len(g.AllBattlefield()) != 0 {
		t.Fatal("preparation installed a permanent")
	}
}

func TestEntryAnnouncedXAcquiredCopyRules(t *testing.T) {
	a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	donorPlan := entryCopyPlan(t, "Donor", func() []CardOption {
		return []CardOption{WithAbility(EntersWithCounters(core.P1P1, AnnouncedXForEnteringPermanent()))}
	})
	donor, err := donorPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(donor, b.PlayerID())
	source.AddCounter(core.P1P1, 17)
	plan, err := NewCardDefinitionPlan(func() Card { return NewCreature("Physical", "{X}", 0, 1, WithAbility(MayCopyAsEnters(IsCreature))) })
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 4); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if p := g.FindPermanent(card.ID()); p == nil || p.Name() != "Donor" || p.Counters[core.P1P1] != 4 {
		t.Fatal("acquired entry rule used donor counters or lost entering X")
	}
	if source.Counters[core.P1P1] != 17 || a.offers != 1 {
		t.Fatal("entry mutated the donor or repeated a choice")
	}
}

func TestEntryAnnouncedXDefinition(t *testing.T) {
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "valid entry X", Card: NewCreature("Valid", "{X}", 0, 1, WithAbility(EntersWithCounters(core.P1P1, AnnouncedXForEnteringPermanent())))},
		{Name: "invalid entry X counter", Card: NewCreature("Invalid", "{X}", 0, 1, WithAbility(EntersWithCounters(core.CounterType(999), AnnouncedXForEnteringPermanent()))), WantErrSubstr: "counter type"},
	})
}
