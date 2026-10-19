package ron

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSpellColorTargetGameplay(t *testing.T) {
	for _, name := range []string{"Blue Elemental Blast", "Red Elemental Blast"} {
		for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
			for mode := range 2 {
				for _, state := range []string{"normal", "multicolor", "color changed", "left", "copy", "clone", "uncounterable or indestructible", "phased"} {
					if mode == 0 && state == "phased" {
						continue
					}
					t.Run(fmt.Sprintf("%s/%s/%d/%s", set, name, mode, state), func(t *testing.T) {
						g, a, b := ronModalGame(mode)
						color, cost := core.Blue, "{U}"
						if name == "Blue Elemental Blast" {
							color, cost = core.Red, "{R}"
						}
						victim := mage.NewInstant("Victim", cost, mage.WithAbility(mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(2)))))
						if mode == 1 {
							victim = mage.NewCreature("Victim", cost, 2, 2)
						}
						if state == "multicolor" {
							victim.SetColorOverride([]core.Color{core.Blue, core.Red})
						}
						if state == "uncounterable or indestructible" {
							if mode == 0 {
								mage.WithUncounterable()(victim)
							} else {
								mage.WithKeyword(core.Indestructible)(victim)
							}
						}
						var target uuid.UUID
						var permanent *mage.Permanent
						if mode == 0 {
							target = castRONSpell(t, g, b, victim).ID
						} else {
							victim.SetOwner(b.PlayerID())
							permanent = g.PutOnBattlefield(victim, b.PlayerID())
							target = permanent.ID()
						}
						watch, _ := mage.InspectPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
						watched := watch.Select(g)
						obj := castRONSpell(t, g, a.BasePlayer, repositoryCard(t, set, name), target)
						if obj.ModeChoice != mode || len(obj.TargetSpecs) != 1 || len(obj.ModalTargets[1-mode]) != 0 {
							t.Fatal("selected-mode frame differs")
						}
						switch state {
						case "color changed":
							g.ChangeObjectColor(target, core.Green)
						case "left":
							if mode == 0 {
								g.CounterStackObject(target)
							} else {
								g.BouncePermanentToHand(permanent)
							}
						case "phased":
							permanent.PhasedOut = true
						case "copy":
							obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
						case "clone":
							g = g.Clone()
							obj = g.StackPeek()
						}
						legal := state != "color changed" && state != "left" && state != "phased"
						if mode == 0 {
							instructions, ok := mage.InspectCounterInstructions(obj.Effects[0])
							if !ok || len(instructions) != 1 {
								t.Fatal("counter descriptor lost")
							}
							predicate, ok := mage.InspectSpellTarget(instructions[0].Subject)
							if !ok {
								t.Fatal("color target is unknown")
							}
							got, known := predicate.Select(g, obj)
							if !known || (got == target) != legal {
								t.Fatal("color target selection differs")
							}
						}
						wantProof := mode == 1 && legal && state != "uncounterable or indestructible"
						before := g.StackSize()
						for range 2 {
							certain, known := g.InspectSpellDestruction(obj, watched)
							if !known || certain != wantProof {
								t.Fatalf("inspection=%v/%v want %v/true (color %v)", certain, known, wantProof, color)
							}
						}
						if g.StackSize() != before {
							t.Fatal("inspection changed stack")
						}
						g.ResolveTopOfStack()
						survives := !legal || state == "uncounterable or indestructible"
						if state == "left" {
							survives = false
						}
						if mode == 0 {
							if (g.GetStack().FindByID(target) != nil) != survives {
								t.Fatal("counter outcome differs")
							}
						} else if (g.FindPermanentIncludingPhased(target) != nil) != survives {
							t.Fatal("destruction outcome differs")
						}
						if a.prompts != 1 {
							t.Fatal("mode chosen again")
						}
					})
				}
			}
		}
	}
}

func TestRONSpellColorTargetInvalid(t *testing.T) {
	for _, filter := range []string{`IsColor()`, `IsColor(Blue,Red)`, `IsColor(Purple)`, `IsColor("Blue")`, `And([])`, `Or([])`} {
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetSpell(`+filter+`)],ActionList([CounterSpell(Ref_TargetSpell)])))])`)
		if _, err := Compile(node); err == nil {
			t.Fatalf("accepted %s", filter)
		}
	}
}

func TestRONSpellColorTargetBooleanBindings(t *testing.T) {
	node := nameScopeNode(t, `Card(Name:"Native color",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetSpell(And([IsCardtype(Instant),Or([IsColor(Red),IsColor(Blue)])]))],ActionList([CounterSpell(Ref_TargetSpell)])))])`)
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	victim := castRONSpell(t, g, a, mage.NewInstant("Victim", "{R}"))
	first := castRONSpell(t, g, b, factory(), victim.ID)
	g.ChangeObjectColor(first.ID, core.Blue)
	second := castRONSpell(t, g, a, factory(), first.ID)
	one, ok := mage.InspectCounterInstructions(first.Effects[0])
	if !ok {
		t.Fatal("first descriptor missing")
	}
	two, ok := mage.InspectCounterInstructions(second.Effects[0])
	if !ok {
		t.Fatal("second descriptor missing")
	}
	p, ok := mage.InspectSpellTarget(one[0].Subject)
	if !ok {
		t.Fatal("Boolean color tree rejected")
	}
	if _, known := p.Select(g, second); known {
		t.Fatal("foreign declaration accepted")
	}
	p, ok = mage.InspectSpellTarget(two[0].Subject)
	if !ok {
		t.Fatal("second color tree rejected")
	}
	if got, known := p.Select(g, second); !known || got != first.ID {
		t.Fatal("independent target lost")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if g.StackSize() != 3 {
		t.Fatal("clone changed live stack")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("unrelated state changed")
	}
}
