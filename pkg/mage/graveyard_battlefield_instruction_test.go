package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestGraveyardBattlefieldInstructionDeclaration(t *testing.T) {
	target := DeclareGraveyardCardTarget("card")
	instruction := GraveyardBattlefieldInstruction{Subject: target, Controller: You}
	got, ok := InspectGraveyardBattlefieldInstruction(instruction.Effect())
	if !ok || got.Subject != target || got.Controller != You {
		t.Fatal("lost return declaration")
	}
	var missing *CardTargetBinding
	for _, subject := range []CardRef{nil, missing} {
		if (GraveyardBattlefieldInstruction{Subject: subject, Controller: You}).Effect() != nil {
			t.Fatal("accepted missing subject")
		}
	}
	var missingController *PlayerTargetBinding
	if (GraveyardBattlefieldInstruction{Subject: target, Controller: missingController}).Effect() != nil {
		t.Fatal("accepted typed nil controller")
	}
	if (GraveyardBattlefieldInstruction{Subject: target}).Effect() != nil {
		t.Fatal("accepted missing controller")
	}
	for _, effect := range []ResolvingEffect{
		nil, (*returnCardToBattlefieldEffect)(nil), ReturnCardToBattlefield(nil, You),
		ReturnCardToBattlefield(target, nil), ReturnCardToBattlefield(target, You, nil),
		ReturnCardToBattlefield(target, You, EnterTapped()),
		ReturnCardToBattlefield(target, You, EnterAttachedTo(This)),
		ReturnCardToBattlefield(target, You, RecordEntryFrom(This)),
		ReturnCardToBattlefieldInto(target, You, BindPermanent("entrant")),
		MayReturnNonAuraCardToBattlefield(target, You),
		Sequence(instruction.Effect(), destructionInspectionTrap{}),
	} {
		if _, ok := InspectGraveyardBattlefieldInstruction(effect); ok {
			t.Fatal("accepted incomplete or different return shape")
		}
	}
}

func TestGraveyardBattlefieldInstructionInspection(t *testing.T) {
	for _, state := range []string{"normal", "absent", "returned", "foreign", "missing zone", "missing visit", "opaque", "duplicate", "other controller", "clone", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, target, victim, obj := graveyardHandInspectionFixture(t)
			obj.Effects = []Effect{ReturnCardToBattlefield(target, You)}
			known := true
			switch state {
			case "absent", "returned":
				card, _ := a.RemoveFromGraveyard(victim.ID())
				if state == "returned" {
					a.AddToGraveyard(card)
				} else {
					a.AddToHand(card)
				}
			case "foreign":
				obj.Effects = []Effect{ReturnCardToBattlefield(DeclareGraveyardCardTarget("card"), You)}
				known = false
			case "missing zone":
				obj.TargetZones = nil
				known = false
			case "missing visit":
				obj.targetIncarnations = nil
				known = false
			case "opaque":
				target.spec.(*GraveyardCardTarget).Filters = []CardFilter{NewCardFilter("trap", func(Card) bool { panic("inspection ran callback") })}
				known = false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, obj.TargetSpecs[0])
				obj.Targets = append(obj.Targets, victim.ID())
				known = false
			case "other controller":
				obj.Effects = []Effect{ReturnCardToBattlefield(target, ActivePlayer)}
				known = false
			case "clone":
				g = g.Clone()
				obj = g.StackPeek()
			case "copy":
				obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
			}
			for range 2 {
				if certain, gotKnown := g.InspectSpellDestruction(obj, nil); certain || gotKnown != known {
					t.Fatalf("certainty=%v/%v want=false/%v", certain, gotKnown, known)
				}
			}
			if g.FindPermanent(victim.ID()) != nil {
				t.Fatal("inspection entered card")
			}
		})
	}
}

func TestGraveyardBattlefieldInstructionOrder(t *testing.T) {
	for _, state := range []string{"prefix", "suffix", "illegal prefix", "all illegal", "foreign suffix", "opaque suffix", "all illegal opaque suffix", "changed-controller copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, a, target, victim, obj := graveyardHandInspectionFixture(t)
			land := g.PutOnBattlefield(NewLand("Land"), a.PlayerID())
			destroy := DestroyEach(MatchingPermanents(IsLand))
			entry := ReturnCardToBattlefield(target, You)
			obj.Effects = []Effect{Sequence(entry, destroy)}
			want, known := false, false
			switch state {
			case "suffix", "clone":
				obj.Effects = []Effect{Sequence(destroy, entry)}
				want, known = true, true
			case "illegal prefix", "changed-controller copy":
				if state == "illegal prefix" {
					a.RemoveFromGraveyard(victim.ID())
				}
				player := DeclarePlayerTarget("player")
				obj.TargetSpecs = append(obj.TargetSpecs, player.targetSpec())
				obj.Targets = append(obj.Targets, a.PlayerID())
				obj.TargetZones[a.PlayerID()] = core.ZoneAny
				obj.Effects = append(obj.Effects, GainLifeFor(player, Fixed(1)))
				if state == "changed-controller copy" {
					obj = g.CopyStackObjectDirect(obj, g.players[1].PlayerID(), false)
				}
				want, known = true, true
			case "all illegal":
				a.RemoveFromGraveyard(victim.ID())
				known = true
			case "foreign suffix":
				obj.Effects = []Effect{Sequence(destroy, ReturnCardToBattlefield(DeclareGraveyardCardTarget("card"), You))}
			case "all illegal opaque suffix":
				a.RemoveFromGraveyard(victim.ID())
				obj.Effects = []Effect{Sequence(destroy, entry, destructionInspectionTrap{})}
			case "opaque suffix":
				obj.Effects = []Effect{Sequence(destroy, entry, destructionInspectionTrap{})}
			}
			if state == "clone" {
				g = g.Clone()
				obj = g.StackPeek()
			}
			for range 2 {
				if got, gotKnown := g.InspectSpellDestruction(obj, []PermanentVisit{{land.ID(), land.incarnationID}}); got != want || gotKnown != known {
					t.Fatalf("certainty=%v/%v want=%v/%v", got, gotKnown, want, known)
				}
			}
			if g.FindPermanent(victim.ID()) != nil || g.FindPermanent(land.ID()) == nil {
				t.Fatal("inspection changed battlefield")
			}
		})
	}
}

func TestGraveyardBattlefieldInstructionPurity(t *testing.T) {
	g, a, target, victim, obj := graveyardHandInspectionFixture(t)
	obj.Effects = []Effect{ReturnCardToBattlefield(target, You)}
	target.spec.(*GraveyardCardTarget).Filters[0].fn = func(Card) bool { panic("inspection ran native filter callback") }
	calls := 0
	g.effects.AddReplacement(&landEntryObserverReplacement{replacementBase: replacementBase{priority: ReplacementPriorityDefault}, calls: &calls})
	before := g.graveyardVisit(victim.ID())
	for range 2 {
		if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
			t.Fatal("native entry alone is unknown")
		}
	}
	if calls != 0 || g.graveyardVisit(victim.ID()) != before || len(a.Graveyard()) != 1 || g.FindPermanent(victim.ID()) != nil {
		t.Fatal("inspection ran replacements or changed the graveyard visit")
	}
	target.spec.(*GraveyardCardTarget).Filters[0] = HasCardTypeCardFilter(core.TypeCreature)
	g.ResolveTopOfStack()
	if calls != 1 || g.FindPermanent(victim.ID()) == nil {
		t.Fatal("execution did not use entry replacement service")
	}
}
