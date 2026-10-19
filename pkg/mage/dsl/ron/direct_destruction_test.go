package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDirectDestruction(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Stone Rain", "Ice Storm", "Sinkhole", "Armageddon", "Shatter", "Wrath of God"} {
			if set == "3ED" && (name == "Ice Storm" || name == "Sinkhole") {
				continue
			}
			t.Run(set+"/"+name, func(t *testing.T) {
				for _, state := range []string{"normal", "regenerate", "indestructible", "phase", "return"} {
					t.Run(state, func(t *testing.T) {
						g, a, b := ronGame()
						card := mage.NewLand("Victim")
						if name == "Shatter" {
							card = mage.NewArtifact("Victim", "")
						}
						if name == "Wrath of God" {
							card = mage.NewCreature("Victim", "", 2, 2)
						}
						card.SetOwner(b.PlayerID())
						victim := g.PutOnBattlefield(card, b.PlayerID())
						mass := name == "Armageddon" || name == "Wrath of God"
						spellCard := repositoryCard(t, set, name)
						var obj *mage.StackObject
						if mass {
							obj = castRONSpell(t, g, a, spellCard)
						} else {
							obj = castRONSpell(t, g, a, spellCard, victim.ID())
						}
						if len(obj.Effects) != 1 {
							t.Fatalf("effect count = %d", len(obj.Effects))
						}
						instructions, ok := mage.InspectDirectDestruction(obj.Effects[0])
						if !ok || len(instructions) != 1 {
							t.Fatal("RON did not lower to a complete typed destruction instruction")
						}
						instruction := instructions[0]
						if (instruction.Subjects != nil) != mass || (instruction.Subject != nil) == mass || instruction.CannotRegenerate != (name == "Wrath of God") {
							t.Fatalf("wrong instruction: %+v", instruction)
						}
						if !mass && mage.UnwrapTarget(instruction.Subject.(*mage.PermanentTargetBinding)) != mage.UnwrapTarget(obj.TargetSpecs[0]) {
							t.Fatal("instruction lost its exact target declaration")
						}
						switch state {
						case "regenerate":
							g.AddRegenerationShield(victim.ID())
						case "indestructible":
							victim.GrantBaseAttr(core.Indestructible)
						case "phase":
							victim.PhasedOut = true
						case "return":
							g.RemoveFromBattlefield(victim)
							victim = g.PutOnBattlefield(card, b.PlayerID())
						}
						clone := g.Clone()
						clone.ResolveStack()
						if g.StackSize() != 1 || g.FindPermanentIncludingPhased(victim.ID()) == nil {
							t.Fatal("clone changed original game")
						}
						g.ResolveStack()
						wantDestroyed := state == "normal" || state == "regenerate" && name == "Wrath of God" || state == "return" && mass
						for _, game := range []*mage.Game{g, clone} {
							if (game.FindPermanentIncludingPhased(victim.ID()) == nil) != wantDestroyed {
								t.Fatalf("state %s: wrong destruction outcome", state)
							}
						}
					})
				}
			})
		}
	}
}

func TestRONDirectDestructionInvalid(t *testing.T) {
	for _, action := range []string{
		`DestroyPermanent`, `DestroyPermanent()`, `DestroyPermanent(ThisPermanent,ThisPermanent)`,
		`DestroyPermanent(Ref_TargetPermanent)`, `DestroyPermanent(You)`,
		`DestroyPermanentNoRegen(Ref_TargetPermanent)`, `DestroyEachPermanent()`,
		`DestroyEachPermanent(Ref_TargetPermanent)`, `DestroyEachPermanentNoRegen(You)`,
	} {
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([`+action+`]))])`)
		if _, err := Compile(node); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONDirectDestructionBindings(t *testing.T) {
	for _, action := range []string{"DestroyPermanent", "DestroyPermanentNoRegen"} {
		t.Run(action, func(t *testing.T) {
			node := nameScopeNode(t, `Card(Name:"Destruction fixture",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Land))],ActionList([`+action+`(Ref_TargetPermanent)])))])`)
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			first := g.PutOnBattlefield(mage.NewLand("First"), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewLand("Second"), b.PlayerID())
			one := castRONSpell(t, g, a, factory(), first.ID())
			two := castRONSpell(t, g, a, factory(), second.ID())
			firstPlan, ok := mage.InspectDirectDestruction(one.Effects[0])
			secondPlan, otherOK := mage.InspectDirectDestruction(two.Effects[0])
			if !ok || !otherOK || len(firstPlan) != 1 || len(secondPlan) != 1 || firstPlan[0].Subject == secondPlan[0].Subject {
				t.Fatal("factory instances share a target declaration")
			}
			if firstPlan[0].CannotRegenerate != (action == "DestroyPermanentNoRegen") {
				t.Fatal("singular regeneration restriction changed")
			}
			g.AddRegenerationShield(first.ID())
			g.AddRegenerationShield(second.ID())
			g.ResolveStack()
			for _, subject := range []*mage.Permanent{first, second} {
				if (g.FindPermanent(subject.ID()) == nil) != (action == "DestroyPermanentNoRegen") {
					t.Fatal("one spell used another instance's target")
				}
			}
		})
	}
}
