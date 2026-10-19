package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTapInstructionRealCards(t *testing.T) {
	for _, name := range []string{"Twiddle", "Riptide", "Reset"} {
		for _, state := range []string{"normal", "phase", "phase cycle", "shroud", "clone", "copy"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				a := &ronTapChoicePlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetStep(core.Upkeep)
				g.SetStep(core.Draw)
				var card mage.Card = mage.NewCreature("Subject", "{U}", 2, 2)
				if name == "Reset" {
					card = mage.NewLand("Subject")
				}
				card.SetOwner(b.PlayerID())
				p := g.PutOnBattlefield(card, b.PlayerID())
				p.Tapped = name == "Reset"
				query, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
				visits := query.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
				set := "LEB"
				if name == "Riptide" {
					set = "DRK"
				}
				if name == "Reset" {
					set = "LEG"
				}
				spell := repositoryCard(t, set, name)
				var obj *mage.StackObject
				switch name {
				case "Reset":
					obj = castRONSpell(t, g, b, spell)
				case "Twiddle":
					obj = castRONSpell(t, g, a.BasePlayer, spell, p.ID())
				default:
					obj = castRONSpell(t, g, a.BasePlayer, spell)
				}
				switch state {
				case "phase":
					g.PhaseOut(p.ID())
				case "phase cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "shroud":
					p.GrantBaseAttr(core.Shroud)
				case "copy":
					obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
				}
				for _, effect := range obj.Effects {
					instructions, ok := mage.InspectTapInstructions(effect)
					if !ok || len(instructions) != 1 || (instructions[0].Subject != nil) != (name == "Twiddle") {
						t.Fatal("real lowering lost singular or simultaneous instruction")
					}
				}
				beforeTapped := p.Tapped
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, visits); certain || !known {
						t.Fatal("native tap tree did not prove no direct destruction")
					}
				}
				if p.Tapped != beforeTapped || len(a.prompts) != 0 {
					t.Fatal("inspection executed tap or choice")
				}
				if state == "clone" {
					cloned := g.Clone()
					if certain, known := cloned.InspectSpellDestruction(cloned.StackPeek(), visits); certain || !known {
						t.Fatal("clone lost declaration")
					}
				}
				g.ResolveTopOfStack()
				changed := state != "phase" && (name != "Twiddle" || state != "shroud")
				if g.FindPermanentIncludingPhased(p.ID()).Tapped != (beforeTapped != changed) {
					t.Fatal("native tap or untap differs")
				}
				wantChoices := 0
				if name == "Twiddle" && changed {
					wantChoices = 1
				}
				if len(a.prompts) != wantChoices {
					t.Fatal("resolution choice count differs")
				}
			})
		}
	}
}

func TestRONTapInstructionInvalid(t *testing.T) {
	for _, action := range []string{`TapPermanent()`, `UntapPermanent(You)`, `TapPermanent(Ref_TargetPermanent2)`, `TapEachPermanent(Ref_TargetPermanent)`, `UntapEachPermanent(And([]))`, `MayAction(TapOrUntapPermanent(You))`, `MayAction(TapOrUntapPermanent(Ref_TargetPermanent,ThisPermanent))`} {
		_, err := Compile(nameScopeNode(t, `Card(Name:"Invalid tap",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([`+action+`])))])`))
		if err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONTapInstructionBindingsAndOrder(t *testing.T) {
	for _, action := range []string{"TapPermanent", "UntapPermanent"} {
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Land)),TargetPermanent(IsCardtype(Land))],ActionList([`+action+`(Ref_TargetPermanent1),DestroyPermanent(Ref_TargetPermanent2)])))`)
		g, a, b := ronGame()
		one := g.PutOnBattlefield(mage.NewLand("One"), b.PlayerID())
		two := g.PutOnBattlefield(mage.NewLand("Two"), b.PlayerID())
		one.Tapped = action == "UntapPermanent"
		two.Tapped = action == "UntapPermanent"
		query, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
		visits := query.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
		first := castRONSpell(t, g, a, factory(), one.ID(), two.ID())
		second := castRONSpell(t, g, a, factory(), two.ID(), one.ID())
		for _, obj := range []*mage.StackObject{first, second} {
			if certain, known := g.InspectSpellDestruction(obj, visits); certain || known {
				t.Fatal("tap prefix supplied a changed-state proof")
			}
		}
		one.GrantBaseAttr(core.Shroud)
		if certain, known := g.InspectSpellDestruction(first, visits); !certain || !known {
			t.Fatal("illegal tap blocked the other legal target")
		}
		if certain, known := g.InspectSpellDestruction(second, visits); certain || known {
			t.Fatal("second factory lost independent declaration")
		}
		g.ResolveTopOfStack()
		if g.FindPermanent(one.ID()) == nil || g.FindPermanent(two.ID()).Tapped != (action == "TapPermanent") {
			t.Fatal("second factory resolved wrong bindings")
		}
		g.ResolveStack()
		if g.FindPermanent(one.ID()) == nil || g.FindPermanent(two.ID()) != nil {
			t.Fatal("partial legality differs from native destruction")
		}
	}
}

func TestRONTapInstructionLowering(t *testing.T) {
	for _, tc := range []struct {
		set, name, syntax string
		operation         mage.TapOperationKind
	}{
		{"DRK", "Riptide", "TapEachPermanent", mage.TapOperation},
		{"LEG", "Reset", "UntapEachPermanent", mage.UntapOperation},
	} {
		record := repositoryRecord(t, tc.set, tc.name, tc.set+".ron")
		node := contextualQueryNode(t, record, tc.syntax)
		plan, err := compileTapInstructionPlan(node, scope{resolving: true})
		if err != nil {
			t.Fatal(err)
		}
		instructions, ok := mage.InspectTapInstructions(plan.build(instanceBindings{}).Effect())
		if !ok || len(instructions) != 1 || instructions[0].Operation != tc.operation || instructions[0].Subject != nil {
			t.Fatal("lowering lost collection instruction")
		}
		if _, ok := mage.InspectContextualPermanentCollection(instructions[0].Subjects); !ok {
			t.Fatal("lowering lost pure collection")
		}
	}
	if _, err := compileTapInstructionPlan(nameScopeNode(t, `Unknown(AnyPermanent)`), scope{resolving: true}); err == nil {
		t.Fatal("accepted unknown instruction")
	}
}
