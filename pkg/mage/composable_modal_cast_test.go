package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func modalValidationSpell(typed bool) *BaseCard {
	var ability Ability
	if typed {
		red := DeclarePermanentTarget("red", HasColorFilter(Red))
		mountain := DeclarePermanentTarget("mountain", HasSubType("Mountain"))
		ability = SpellModes(ModeOf("Destroy red", Destroy(red), WithTargets(red)), ModeOf("Return Mountain", ReturnToHand(mountain), WithTargets(mountain)))
	} else {
		ability = NewModalSpell([]Mode{{Label: "Destroy red", Targets: []Target{TargetPermanent(HasColorFilter(Red))}, Effects: []Effect{DestroyTarget()}}, {Label: "Return Mountain", Targets: []Target{TargetPermanent(HasSubType("Mountain"))}, Effects: []Effect{ReturnToHandTarget()}}})
	}
	return NewInstant("Modal validation", "{1}", WithAbility(ability))
}

func TestModalCastRejectsTargetsBeforePayment(t *testing.T) {
	for _, typed := range []bool{false, true} {
		for _, alternate := range []bool{false, true} {
			for _, mode := range []int{0, 1} {
				for _, prompted := range []bool{false, true} {
					g, a, b := newModalGame()
					card := modalValidationSpell(typed)
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					a.ManaPool().Add(Colorless, 1)
					victim := NewCreature("Wrong target", "{U}", 2, 2)
					victim.SetOwner(b.PlayerID())
					g.PutOnBattlefield(victim, b.PlayerID())
					a.modes = []int{mode}
					targets := []uuid.UUID{victim.ID()}
					if prompted {
						a.targets = [][]uuid.UUID{targets}
						targets = nil
					}
					var err error
					if alternate {
						err = g.CastCardFromZoneWithAlternateCost(a.PlayerID(), card.ID(), ZoneHand, ManaCost{Generic: 1}, targets, 0)
					} else {
						err = g.CastSpellByName(a.PlayerID(), card.Name(), targets)
					}
					if err == nil {
						t.Errorf("typed=%v alternate=%v mode=%d prompted=%v: illegal cast accepted", typed, alternate, mode, prompted)
					}
					if a.ManaPool().TotalMana() != 1 || len(a.Hand()) != 1 || !g.stack.IsEmpty() {
						t.Errorf("illegal cast changed resources or stack")
					}
				}
			}
		}
	}
}

func TestModalCastKeepsModeAndTargets(t *testing.T) {
	for _, typed := range []bool{false, true} {
		for _, mode := range []int{0, 1} {
			g, a, b := newModalGame()
			card := modalValidationSpell(typed)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(Colorless, 1)
			var victim Card = NewCreature("Red target", "{R}", 2, 2)
			if mode == 1 {
				victim = NewLand("Mountain", WithSubTypes("Mountain"))
			}
			victim.SetOwner(b.PlayerID())
			g.PutOnBattlefield(victim, b.PlayerID())
			a.modes = []int{mode, 1 - mode}
			if err := g.CastSpellByName(a.PlayerID(), card.Name(), []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			cp := g.CopySpellOnStack(card.ID(), a.PlayerID(), false)
			if cp == nil || cp.ModeChoice != mode || len(cp.TargetSpecs) != 1 || len(cp.ModalTargets[mode]) != 1 {
				t.Fatal("copy lost selected mode or targets")
			}
			if len(a.modes) != 1 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("mode prompted twice or payment incorrect")
			}
			g.ResolveStack()
			if g.FindPermanent(victim.ID()) != nil {
				t.Fatal("selected effect did not resolve")
			}
			if mode == 1 && len(b.Hand()) != 1 {
				t.Fatal("return mode did not return target")
			}
			if mode == 0 && len(b.Graveyard()) != 1 {
				t.Fatal("destroy mode did not destroy target")
			}
		}
	}
}

func TestModalCastTargetFailuresLeaveManaSources(t *testing.T) {
	for _, scenario := range []string{"missing", "duplicate", "extra", "shroud", "phased", "invalid mode", "wrong land subtype"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := newModalGame()
			land := addLand(t, g, a.PlayerID(), "Plains", White)
			card := modalValidationSpell(true)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			victim := NewCreature("Red target", "{R}", 2, 2)
			victim.SetOwner(b.PlayerID())
			perm := g.PutOnBattlefield(victim, b.PlayerID())
			targets := make([]uuid.UUID, 1, 2)
			targets[0] = victim.ID()
			switch scenario {
			case "missing":
				g.PutPermanentIntoGraveyard(perm)
				targets = targets[:0]
			case "duplicate":
				targets = append(targets, victim.ID())
			case "extra":
				targets = append(targets, b.PlayerID())
			case "shroud":
				perm.GrantBaseAttr(Shroud)
			case "phased":
				g.PhaseOut(perm.ID())
			case "invalid mode":
				a.modes = []int{2}
			case "wrong land subtype":
				island := NewLand("Island", WithSubTypes("Island"))
				island.SetOwner(b.PlayerID())
				g.PutOnBattlefield(island, b.PlayerID())
				targets = []uuid.UUID{island.ID()}
				a.modes = []int{1}
			}
			if err := g.CastSpellByName(a.PlayerID(), card.Name(), targets); err == nil {
				t.Fatal("illegal cast accepted")
			}
			if land.Tapped || a.ManaPool().TotalMana() != 0 || len(a.Hand()) != 1 || !g.stack.IsEmpty() {
				t.Fatal("illegal cast changed resources")
			}
		})
	}
}

func TestModalCastTargetlessMode(t *testing.T) {
	for _, extra := range []bool{false, true} {
		g, a, b := newModalGame()
		target := DeclarePermanentTarget("creature", IsCreature)
		card := NewInstant("Targetless mode", "{1}", WithAbility(SpellModes(ModeOf("Destroy", Destroy(target), WithTargets(target)), ModeOf("Gain", GainLifeFor(You, Fixed(3))))))
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(Colorless, 1)
		a.modes = []int{1}
		var targets []uuid.UUID
		if extra {
			targets = []uuid.UUID{b.PlayerID()}
		}
		err := g.CastSpellByName(a.PlayerID(), card.Name(), targets)
		if extra {
			if err == nil || a.ManaPool().TotalMana() != 1 {
				t.Fatal("targetless mode accepted target")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		before := a.Life()
		g.ResolveStack()
		if a.Life() != before+3 {
			t.Fatal("targetless mode did not resolve")
		}
	}
}

func TestModalCastVariableTargetPrompt(t *testing.T) {
	g, a, b := newModalGame()
	card := NewInstant("Variable mode", "{X}", WithAbility(NewModalSpell([]Mode{{Label: "Destroy X", Targets: []Target{TargetXCreatures()}, Effects: []Effect{DestroyTarget()}}, {Label: "Gain", Effects: []Effect{GainLifeFor(You, Fixed(1))}}})))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(Colorless, 2)
	for range 2 {
		c := NewCreature("Target", "", 2, 2)
		c.SetOwner(b.PlayerID())
		g.PutOnBattlefield(c, b.PlayerID())
	}
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil, 2); err != nil {
		t.Fatal(err)
	}
	obj := g.stack.Peek()
	if len(obj.Targets) != 2 || obj.XValue != 2 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("modal prompt did not use declared X")
	}
}
