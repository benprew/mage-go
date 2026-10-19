package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPermanentTargetQuery(t *testing.T) {
	for _, syntax := range []string{`AnyPermanent`, `IsCardtype(Land)`, `IsNonCardtype(Creature)`, `And([IsCardtype(Land),IsNonCardtype(Creature)])`, `Or([IsCardtype(Land),IsCardtype(Artifact)])`} {
		node := nameScopeNode(t, syntax)
		filter, handled, err := compileInspectableTargetFilter(node)
		if err != nil || !handled {
			t.Fatalf("type lowering %s: %v", syntax, err)
		}
		ref := mage.DeclarePermanentTarget("target", filter)
		if _, ok := mage.InspectPermanentTarget(ref); !ok {
			t.Fatalf("opaque target %s", syntax)
		}
	}
	for _, syntax := range []string{`IsCardtype()`, `IsCardtype(Unknown)`, `IsCardtype(Land,Creature)`, `IsNonCardtype`, `And(IsCardtype(Land))`, `Or([])`} {
		if _, _, err := compileInspectableTargetFilter(nameScopeNode(t, syntax)); err == nil {
			t.Fatalf("accepted %s", syntax)
		}
	}
	for _, syntax := range []string{`IsEnchanted`, `And([IsCardtype(Land),IsEnchanted])`} {
		if _, handled, _ := compileInspectableTargetFilter(nameScopeNode(t, syntax)); handled {
			t.Fatalf("claimed opaque %s", syntax)
		}
	}
}

func TestRONPermanentTargetQueryGameplay(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Stone Rain", "Ice Storm", "Sinkhole", "Shatter"} {
			if set == "3ED" && (name == "Ice Storm" || name == "Sinkhole") {
				continue
			}
			t.Run(set+"/"+name, func(t *testing.T) {
				for _, state := range []string{"legal", "shroud", "protection", "phase", "return"} {
					t.Run(state, func(t *testing.T) {
						g, a, b := ronGame()
						card := mage.NewLand("Subject")
						if name == "Shatter" {
							card = mage.NewArtifact("Subject", "")
						}
						card.SetOwner(b.PlayerID())
						victim := g.PutOnBattlefield(card, b.PlayerID())
						spell := repositoryCard(t, set, name)
						obj := castRONSpell(t, g, a, spell, victim.ID())
						instructions, ok := mage.InspectDirectDestruction(obj.Effects[0])
						if !ok || len(instructions) != 1 {
							t.Fatal("destruction lowering lost")
						}
						plan, ok := mage.InspectPermanentTarget(instructions[0].Subject)
						if !ok {
							t.Fatal("target lowering lost")
						}
						switch state {
						case "shroud":
							g.FindPermanent(victim.ID()).GrantBaseAttr(core.Shroud)
						case "protection":
							g.FindPermanent(victim.ID()).RuntimeAbilities = append(g.FindPermanent(victim.ID()).RuntimeAbilities, mage.ProtectionFromColors(spell.ManaCost().Colors()...))
						case "phase":
							g.PhaseOut(victim.ID())
						case "return":
							g.RemoveFromBattlefield(victim)
							victim = g.PutOnBattlefield(card, b.PlayerID())
						}
						got, known := plan.Select(g, obj)
						if !known || (len(got) == 1) != (state == "legal") {
							t.Fatalf("query %v known %v", got, known)
						}
						clone := g.Clone()
						clone.ResolveStack()
						if g.StackSize() != 1 {
							t.Fatal("clone changed original")
						}
						g.ResolveStack()
						for _, game := range []*mage.Game{g, clone} {
							if (game.FindPermanentIncludingPhased(victim.ID()) == nil) != (state == "legal") {
								t.Fatal("query and gameplay differ")
							}
						}
					})
				}
			})
		}
	}
}

func TestRONPermanentTargetQueryBindings(t *testing.T) {
	for _, filter := range []string{`AnyPermanent`, `And([IsCardtype(Land),IsNonCardtype(Creature)])`, `Or([IsCardtype(Land),IsCardtype(Artifact)])`} {
		t.Run(filter, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(`+filter+`)],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`)
			g, a, b := ronGame()
			one := g.PutOnBattlefield(mage.NewLand("One"), b.PlayerID())
			two := g.PutOnBattlefield(mage.NewLand("Two"), b.PlayerID())
			first := castRONSpell(t, g, a, factory(), one.ID())
			second := castRONSpell(t, g, a, factory(), two.ID())
			firstInstructions, _ := mage.InspectDirectDestruction(first.Effects[0])
			secondInstructions, _ := mage.InspectDirectDestruction(second.Effects[0])
			firstPlan, ok := mage.InspectPermanentTarget(firstInstructions[0].Subject)
			secondPlan, otherOK := mage.InspectPermanentTarget(secondInstructions[0].Subject)
			if !ok || !otherOK {
				t.Fatal("composite target lost pure lowering")
			}
			if _, known := firstPlan.Select(g, second); known {
				t.Fatal("factory declarations share identity")
			}
			if got, known := secondPlan.Select(g, second); !known || len(got) != 1 || got[0].ID != two.ID() {
				t.Fatal("second factory lost target")
			}
			copied := g.CopyStackObjectDirect(first, a.PlayerID(), false)
			if got, known := firstPlan.Select(g, copied); !known || len(got) != 1 || got[0].ID != one.ID() {
				t.Fatal("copy lost target binding")
			}
			g.ResolveStack()
			if g.FindPermanent(one.ID()) != nil || g.FindPermanent(two.ID()) != nil {
				t.Fatal("independent spells failed")
			}
		})
	}
	for _, body := range []string{
		`SpellActions(Targeted([TargetPermanent(IsCardtype(Unknown))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`,
		`SpellActions(Targeted([TargetPermanent(And([]))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`,
		`SpellActions(Targeted([TargetPermanent(IsCardtype(Land))],ActionList([DestroyPermanent(Ref_TargetPermanent2)])))`,
		`SpellActions(ActionList([DestroyPermanent(Ref_TargetPermanent)]))`,
	} {
		node := nameScopeNode(t, `Card(Name:"Invalid query",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+body+`])`)
		if _, err := Compile(node); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}
