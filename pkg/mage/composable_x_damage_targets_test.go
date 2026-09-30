package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTypedXDamageTargetsAnnouncement(t *testing.T) {
	for _, upTo := range []bool{false, true} {
		t.Run(map[bool]string{false: "exact", true: "up to"}[upTo], func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			targets := DeclareXDamageTargets("recipients")
			if upTo {
				targets = DeclareUpToXDamageTargets("recipients")
			}
			spell := NewInstant("X recipients", "{X}", Spell(Sequence(DealDamageEach(This, targets, Fixed(2)), GainLifeFor(You, Fixed(1))), WithTargets(targets)))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
				t.Fatal(err)
			}
			creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 9), b.PlayerID())
			walker := g.PutOnBattlefield(NewPlaneswalker("Walker", "", 5), b.PlayerID())
			artifact := g.PutOnBattlefield(NewArtifact("Artifact", ""), b.PlayerID())
			invalid := [][]uuid.UUID{{b.PlayerID(), creature.ID(), walker.ID(), a.PlayerID()}, {creature.ID(), creature.ID()}, {artifact.ID()}, {uuid.New()}}
			if !upTo {
				invalid = append(invalid, nil, []uuid.UUID{b.PlayerID()})
			}
			for _, ids := range invalid {
				if err := g.CastSpellByName(a.PlayerID(), spell.Name(), ids, 3); err == nil {
					t.Fatalf("accepted invalid targets %v", ids)
				}
				if g.StackSize() != 0 || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 0 {
					t.Fatal("invalid announcement changed state")
				}
			}
			ids := []uuid.UUID{creature.ID(), walker.ID(), b.PlayerID()}
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), ids, 3); err == nil {
				t.Fatal("unpaid spell accepted")
			}
			if g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("failed payment changed state")
			}
			a.ManaPool().Add(Colorless, 3)
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), ids, 3); err != nil {
				t.Fatal(err)
			}
			obj := g.StackPeek()
			if obj.XValue != 3 || len(obj.TargetSpecs) != 3 {
				t.Fatal("X target count was not recorded")
			}
			g.SetXValue(99)
			g.AddPreventionShield(b.PlayerID(), 1)
			callbacks := 0
			g.damage.SetOnDamageDealt(func(string, string, int, bool) {
				callbacks++
				if creature.Damage != 2 || walker.Counters[Loyalty] != 3 || b.Life() != 19 {
					t.Error("callback observed partial batch")
				}
			})
			g.ResolveStack()
			if callbacks != 3 || a.Life() != 21 {
				t.Fatal("mixed batch or following instruction failed")
			}
		})
	}
}

func TestTypedXDamageTargetsZeroAndOptional(t *testing.T) {
	for _, upTo := range []bool{false, true} {
		for _, x := range []int{0, 2} {
			if !upTo && x != 0 {
				continue
			}
			g, a, _ := newSpellCopyGame()
			targets := DeclareXDamageTargets("recipients")
			if upTo {
				targets = DeclareUpToXDamageTargets("recipients")
			}
			spell := NewInstant("Empty recipients", "{X}", Spell(Sequence(DealDamageEach(This, targets, Fixed(2)), GainLifeFor(You, Fixed(1))), WithTargets(targets)))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(Colorless, x)
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil, x); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if a.Life() != 21 {
				t.Fatal("empty target group did not resolve")
			}
		}
	}
}

func TestTypedXDamageTargetsActivationIdentityAndCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareUpToXDamageTargets("recipients")
	card := NewCreature("X source", "", 2, 2, WithKeyword(Lifelink), WithKeyword(Haste), WithAbility(Activate(Costs(ManaPayment("{X}"), TapCost(This)), DealDamageEach(This, targets, Fixed(2)), WithTargets(targets))))
	source := g.PutOnBattlefield(card, a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 9), b.PlayerID())
	g.SetXValue(2)
	ids := []uuid.UUID{victim.ID(), b.PlayerID()}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, ids); err == nil {
		t.Fatal("unpaid activation accepted")
	}
	if source.Tapped || g.StackSize() != 0 {
		t.Fatal("payment failure tapped source")
	}
	a.ManaPool().Add(Colorless, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, ids); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(source)
	g.PutOnBattlefield(card, b.PlayerID())
	clone := g.Clone()
	copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	clone.PhaseOut(victim.ID())
	clone.ResolveStackObject(copyObj)
	if clone.GetPlayer(b.PlayerID()).Life() != 18 || clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.FindPermanentIncludingPhased(victim.ID()).Damage != 0 {
		t.Fatal("copy lost LKI or affected phased target")
	}
	if a.Life() != 20 || b.Life() != 20 || victim.Damage != 0 {
		t.Fatal("clone changed original")
	}
	g.RemoveFromBattlefield(victim)
	g.PutOnBattlefield(victim.Card, b.PlayerID())
	g.SetXValue(0)
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 18 || g.FindPermanent(victim.ID()).Damage != 0 {
		t.Fatal("activation followed returned source or target")
	}
}

func TestTypedXDamageTargetsDeclarations(t *testing.T) {
	targets := DeclareXDamageTargets("recipients")
	unnamed := DeclareUpToXDamageTargets("")
	other := DeclareUpToXDamageTargets("other")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "undeclared", Card: NewInstant("Missing", "", Spell(DealDamageEach(This, targets, Fixed(1)))), WantErr: true},
		{Name: "unnamed", Card: NewInstant("Unnamed", "", Spell(DealDamageEach(This, unnamed, Fixed(1)), WithTargets(unnamed))), WantErr: true},
		{Name: "unrecorded groups", Card: NewInstant("Unrecorded", "", Spell(DealDamageEach(This, targets, Fixed(1)), WithTargets(targets, other))), WantErr: true},
		{Name: "recorded groups", Card: NewInstant("Grouped", "", Spell(DealDamageEach(This, targets, Fixed(1)), WithTargetGroups(targets, other)))},
	})
}

func TestTypedXDamageTargetsRetargetAndLegality(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareXDamageTargets("recipients")
	spell := NewInstant("Retarget X", "{X}{R}", Spell(Sequence(DealDamageEach(This, targets, Fixed(2)), GainLifeFor(You, Fixed(1))), WithTargets(targets)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 9), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 9), b.PlayerID())
	a.ManaPool().Add(Red, 3)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{first.ID(), b.PlayerID()}, 2); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	a.chooseQueue = [][]uuid.UUID{{second.ID(), a.PlayerID()}}
	g.SetXValue(99)
	copied := g.CopySpellOnStack(spell.ID(), a.PlayerID(), true)
	if copied.XValue != 2 || len(copied.Targets) != 2 || original.Targets[0] != first.ID() {
		t.Fatal("retargeting changed X, count, or original")
	}
	g.ResolveStackObject(copied)
	if second.Damage != 2 || first.Damage != 0 || a.Life() != 19 || b.Life() != 20 {
		t.Fatal("copy did not use new mixed targets")
	}
	first.RuntimeAbilities = append(first.RuntimeAbilities, ProtectionFromColor(Red))
	g.ResolveStackObject(original)
	if first.Damage != 0 || b.Life() != 18 || a.Life() != 20 {
		t.Fatal("protection invalidated legal player or failed to exclude creature")
	}
}

func TestTypedXDamageTargetsRecordedGroups(t *testing.T) {
	for _, activation := range []bool{false, true} {
		g, a, b := newSpellCopyGame()
		empty := DeclareUpToXDamageTargets("empty")
		targets := DeclareXDamageTargets("recipients")
		body := Sequence(DealDamageEach(This, empty, Fixed(7)), DealDamageEach(This, targets, Fixed(2)))
		groups := [][]uuid.UUID{nil, {b.PlayerID()}}
		a.ManaPool().Add(Colorless, 1)
		if activation {
			source := g.PutOnBattlefield(NewArtifact("Grouped X", "", WithAbility(Activate(Costs(ManaPayment("{X}")), body, WithActivationTargetGroups(empty, targets)))), a.PlayerID())
			g.SetXValue(1)
			if err := g.ActivateAbilityByIndexWithTargetGroups(a.PlayerID(), source.ID(), 0, groups); err != nil {
				t.Fatal(err)
			}
		} else {
			spell := NewInstant("Grouped X", "{X}", Spell(body, WithTargetGroups(empty, targets)))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByNameWithTargetGroups(a.PlayerID(), spell.Name(), groups, 1); err != nil {
				t.Fatal(err)
			}
		}
		original := g.StackPeek()
		clone := g.Clone()
		copied := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
		clone.ResolveStackObject(copied)
		if clone.GetPlayer(b.PlayerID()).Life() != 18 || b.Life() != 20 {
			t.Fatal("copy lost empty group boundary or crossed clone")
		}
		g.ResolveStackObject(original)
		if b.Life() != 18 {
			t.Fatal("target shifted into empty group")
		}
	}
}

func TestTypedXDamageTargetsAllIllegal(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclareXDamageTargets("recipients")
	spell := NewInstant("All illegal", "{X}", Spell(Sequence(DealDamageEach(This, targets, Fixed(2)), GainLifeFor(You, Fixed(1))), WithTargets(targets)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 9), b.PlayerID())
	a.ManaPool().Add(Colorless, 1)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{victim.ID()}, 1); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(victim.ID())
	g.ResolveStack()
	if a.Life() != 20 || g.FindPermanentIncludingPhased(victim.ID()).Damage != 0 {
		t.Fatal("all-illegal spell resolved")
	}
}
