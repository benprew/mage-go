package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestAnyNumberDamageTargetsAnnouncement(t *testing.T) {
	for _, activation := range []bool{false, true} {
		t.Run(map[bool]string{false: "spell", true: "activation"}[activation], func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			targets := DeclareAnyNumberDamageTargets("recipients")
			body := Sequence(DealDamageEach(This, targets, Fixed(1)), GainLifeFor(You, Fixed(1)))
			ids := make([]uuid.UUID, 1, 103)
			ids[0] = b.PlayerID()
			for range 101 {
				p := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 5), b.PlayerID())
				ids = append(ids, p.ID())
			}
			walker := g.PutOnBattlefield(NewPlaneswalker("Walker", "", 5), b.PlayerID())
			ids = append(ids, walker.ID())
			artifact := g.PutOnBattlefield(NewArtifact("Artifact", ""), b.PlayerID())
			var cast func([]uuid.UUID) error
			var source *Permanent
			if activation {
				source = g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(ManaPayment("{1}"), TapCost(This)), body, WithTargets(targets)))), a.PlayerID())
				cast = func(chosen []uuid.UUID) error { return g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, chosen) }
			} else {
				spell := NewInstant("Any number", "{1}", Spell(body, WithTargets(targets)))
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				cast = func(chosen []uuid.UUID) error { return g.CastSpellByName(a.PlayerID(), spell.Name(), chosen, 0) }
			}
			for _, chosen := range [][]uuid.UUID{{b.PlayerID(), b.PlayerID()}, {artifact.ID()}, {uuid.New()}, ids} {
				if err := cast(chosen); err == nil {
					t.Fatal("invalid or unpaid announcement accepted")
				}
				if g.StackSize() != 0 || a.ManaPool().TotalMana() != 0 || (source != nil && source.Tapped) {
					t.Fatal("failed announcement spent payment")
				}
			}
			a.ManaPool().Add(Colorless, 1)
			phased := g.PutOnBattlefield(NewCreature("Phased", "", 1, 5), b.PlayerID())
			g.PhaseOut(phased.ID())
			for _, chosen := range [][]uuid.UUID{{b.PlayerID(), b.PlayerID()}, {artifact.ID()}, {uuid.New()}, {phased.ID()}} {
				if err := cast(chosen); err == nil {
					t.Fatal("funded invalid announcement accepted")
				}
				if g.StackSize() != 0 || a.ManaPool().TotalMana() != 1 || (source != nil && source.Tapped) {
					t.Fatal("invalid targets spent payment")
				}
			}
			if err := cast(ids); err != nil {
				t.Fatal(err)
			}
			if len(g.StackPeek().TargetSpecs) != len(ids) {
				t.Fatal("target count lost")
			}
			callbacks := 0
			g.damage.SetOnDamageDealt(func(string, string, int, bool) {
				callbacks++
				if b.Life() != 19 || walker.Counters[Loyalty] != 4 {
					t.Error("partial batch visible")
				}
				for _, id := range ids[1 : len(ids)-1] {
					if g.FindPermanent(id).Damage != 1 {
						t.Error("partial creature batch visible")
					}
				}
			})
			g.ResolveStack()
			if callbacks != len(ids) || a.Life() != 21 {
				t.Fatal("batch or following instruction failed")
			}
		})
	}
}

func TestAnyNumberDamageTargetsEmptyAndGroups(t *testing.T) {
	for _, grouped := range []bool{false, true} {
		g, a, b := newSpellCopyGame()
		empty := DeclareAnyNumberDamageTargets("empty")
		other := DeclareAnyNumberDamageTargets("other")
		body := Sequence(DealDamageEach(This, empty, Fixed(7)), GainLifeFor(You, Fixed(1)))
		spell := NewInstant("Empty", "", Spell(body, WithTargets(empty)))
		if grouped {
			spell = NewInstant("Empty", "", Spell(Sequence(body, DealDamageEach(This, other, Fixed(2))), WithTargetGroups(empty, other)))
		}
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		var err error
		if grouped {
			err = g.CastSpellByNameWithTargetGroups(a.PlayerID(), spell.Name(), [][]uuid.UUID{nil, {b.PlayerID()}}, 0)
		} else {
			err = g.CastSpellByName(a.PlayerID(), spell.Name(), nil, 0)
		}
		if err != nil {
			t.Fatal(err)
		}
		clone := g.Clone()
		copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
		clone.ResolveStackObject(copyObj)
		want := 20
		if grouped {
			want = 18
		}
		if clone.GetPlayer(a.PlayerID()).Life() != 21 || clone.GetPlayer(b.PlayerID()).Life() != want || a.Life() != 20 || b.Life() != 20 {
			t.Fatal("empty boundary or clone isolation failed")
		}
		g.ResolveStack()
		if a.Life() != 21 || b.Life() != want {
			t.Fatal("empty group failed")
		}
	}
}

func TestAnyNumberDamageTargetsAllocationAdmission(t *testing.T) {
	for _, xTotal := range []bool{false, true} {
		g, a, b := newActivationDivisionGame()
		targets := DeclareAnyNumberDamageTargets("recipients")
		body := DealAllocatedDamage(This, targets, 1)
		if xTotal {
			body = DealAllocatedDamageX(This, targets)
		}
		spell := NewInstant("Allocation", "{1}", Spell(body, WithTargets(targets)), WithAdditionalCost(LifePayCost(2)))
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		a.ManaPool().Add(Colorless, 1)
		victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 5), b.PlayerID())
		if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{victim.ID(), b.PlayerID()}, 1); err == nil {
			t.Fatal("more targets than positive shares accepted")
		}
		if a.divisions != 0 || a.Life() != 20 || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 || len(a.Hand()) != 1 {
			t.Fatal("rejected allocation prompted or paid")
		}
		if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil, 1); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if a.divisions != 0 || a.Life() != 18 || b.Life() != 20 {
			t.Fatal("empty allocation prompted or dealt damage")
		}
	}
}

func TestAnyNumberDamageTargetsDeclarations(t *testing.T) {
	targets := DeclareAnyNumberDamageTargets("recipients")
	unnamed := DeclareAnyNumberDamageTargets("")
	other := DeclareAnyNumberDamageTargets("other")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "valid", Card: NewInstant("Valid", "", Spell(DealDamageEach(This, targets, Fixed(1)), WithTargets(targets)))},
		{Name: "undeclared", Card: NewInstant("Invalid", "", Spell(DealDamageEach(This, targets, Fixed(1)))), WantErr: true},
		{Name: "unnamed", Card: NewInstant("Invalid", "", Spell(DealDamageEach(This, unnamed, Fixed(1)), WithTargets(unnamed))), WantErr: true},
		{Name: "unrecorded groups", Card: NewInstant("Invalid", "", Spell(DealDamageEach(This, targets, Fixed(1)), WithTargets(targets, other))), WantErr: true},
		{Name: "recorded groups", Card: NewInstant("Valid", "", Spell(DealDamageEach(This, targets, Fixed(1)), WithTargetGroups(targets, other)))},
	})
}

func TestAnyNumberDamageTargetsIdentityAndCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareAnyNumberDamageTargets("recipients")
	card := NewCreature("Source", "", 2, 2, WithKeyword(Lifelink), WithAbility(Activate(Costs(SacrificeSourceCost()), DealDamageEach(This, targets, Fixed(2)), WithTargets(targets))))
	source := g.PutOnBattlefield(card, a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 9), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID(), b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, b.PlayerID())
	clone := g.Clone()
	copied := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	clone.PhaseOut(victim.ID())
	clone.AddPreventionShield(b.PlayerID(), 1)
	clone.ResolveStackObject(copied)
	if clone.GetPlayer(a.PlayerID()).Life() != 21 || clone.GetPlayer(b.PlayerID()).Life() != 19 || clone.FindPermanentIncludingPhased(victim.ID()).Damage != 0 {
		t.Fatal("copy lost source LKI or target identity")
	}
	if a.Life() != 20 || b.Life() != 20 || victim.Damage != 0 {
		t.Fatal("clone changed original")
	}
	g.RemoveFromBattlefield(victim)
	returned := g.PutOnBattlefield(victim.Card, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 18 || returned.Damage != 0 {
		t.Fatal("damage followed returned incarnation")
	}
}

func TestAnyNumberDamageTargetsRetargetAndProtection(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareAnyNumberDamageTargets("recipients")
	spell := NewInstant("Retarget", "{R}", Spell(Sequence(DealDamageEach(This, targets, Fixed(2)), GainLifeFor(You, Fixed(1))), WithTargets(targets)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(Red, 1)
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 9), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 9), b.PlayerID())
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{first.ID(), b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	a.chooseQueue = [][]uuid.UUID{{second.ID(), a.PlayerID()}}
	copied := g.CopySpellOnStack(spell.ID(), a.PlayerID(), true)
	if len(copied.Targets) != 2 || original.Targets[0] != first.ID() {
		t.Fatal("copy changed original count or choices")
	}
	g.ResolveStackObject(copied)
	if second.Damage != 2 || first.Damage != 0 || a.Life() != 19 || b.Life() != 20 {
		t.Fatal("retargeted copy failed")
	}
	first.RuntimeAbilities = append(first.RuntimeAbilities, ProtectionFromColor(Red))
	g.ResolveStackObject(original)
	if first.Damage != 0 || b.Life() != 18 || a.Life() != 20 {
		t.Fatal("protection removed independent legal player")
	}
}

func TestAnyNumberDamageTargetsAllIllegal(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareAnyNumberDamageTargets("recipients")
	spell := NewInstant("All illegal", "", Spell(Sequence(DealDamageEach(This, targets, Fixed(2)), GainLifeFor(You, Fixed(1))), WithTargets(targets)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 9), b.PlayerID())
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{victim.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(victim.ID())
	g.ResolveStack()
	if a.Life() != 20 || victim.Damage != 0 {
		t.Fatal("all-illegal targets resolved")
	}
}
