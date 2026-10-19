package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONReturnInstructionRealCards(t *testing.T) {
	for _, name := range []string{"Boomerang", "Unsummon", "Remove Enchantments"} {
		for _, state := range []string{"normal", "phase", "phase cycle", "shroud", "clone", "copy", "foreign owner"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				var subject mage.Card = mage.NewEnchantment("Subject", "")
				set := "LEG"
				if name == "Unsummon" {
					subject = mage.NewCreature("Subject", "", 2, 2)
					set = "3ED"
				}
				subject.SetOwner(a.PlayerID())
				if state == "foreign owner" {
					subject.SetOwner(b.PlayerID())
				}
				p := g.PutOnBattlefield(subject, a.PlayerID())
				query, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
				watched := query.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
				card := repositoryCard(t, set, name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				var obj *mage.StackObject
				if name != "Remove Enchantments" {
					obj = castRONSpell(t, g, a, card, p.ID())
				} else {
					obj = castRONSpell(t, g, a, card)
				}
				switch state {
				case "phase":
					g.PhaseOut(p.ID())
				case "phase cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "shroud":
					p.GrantBaseAttr(core.Shroud)
				case "copy":
					obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
				}
				want := name == "Remove Enchantments" && state == "foreign owner"
				for range 2 {
					if got, known := g.InspectSpellDestruction(obj, watched); !known || got != want {
						t.Fatalf("certainty=%v/%v want=%v", got, known, want)
					}
				}
				if state == "clone" {
					clone := g.Clone()
					clone.ResolveStack()
					if g.FindPermanent(p.ID()) == nil || clone.FindPermanent(p.ID()) != nil {
						t.Fatal("clone changed original")
					}
				}
				g.ResolveStack()
				absent := state != "phase" && (name == "Remove Enchantments" || state != "shroud")
				if (g.FindPermanentIncludingPhased(p.ID()) == nil) != absent {
					t.Fatal("query differs from native resolution")
				}
				owner := g.GetPlayer(subject.Owner())
				inHand, inGraveyard := false, false
				for _, card := range owner.Hand() {
					inHand = inHand || card.ID() == subject.ID()
				}
				for _, card := range owner.Graveyard() {
					inGraveyard = inGraveyard || card.ID() == subject.ID()
				}
				if absent && (inGraveyard != want || inHand == want) {
					t.Fatal("wrong owner destination")
				}
			})
		}
	}
}

func TestRONReturnInstructionBindings(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([PutPermanentIntoItsOwnersHand(Ref_TargetPermanent),DestroyPermanent(Ref_TargetPermanent)])))`)
	g, a, b := ronGame()
	one := g.PutOnBattlefield(mage.NewLand("One"), b.PlayerID())
	two := g.PutOnBattlefield(mage.NewLand("Two"), b.PlayerID())
	query, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
	visits := query.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
	first := castRONSpell(t, g, a, factory(), one.ID())
	second := castRONSpell(t, g, a, factory(), two.ID())
	for i, obj := range []*mage.StackObject{first, second} {
		if certain, known := g.InspectSpellDestruction(obj, visits[i:i+1]); certain || !known {
			t.Fatal("return did not remove exact watched visit from proof")
		}
	}
	g.ResolveStack()
	if len(b.Hand()) != 2 || g.FindPermanent(one.ID()) != nil || g.FindPermanent(two.ID()) != nil {
		t.Fatal("factory bindings crossed")
	}
}

func TestRONReturnInstructionInvalid(t *testing.T) {
	for _, action := range []string{`PutPermanentIntoItsOwnersHand()`, `PutPermanentIntoItsOwnersHand(Ref_TargetPermanent,ThisPermanent)`, `PutPermanentIntoItsOwnersHand(Ref_TargetPermanent2)`, `PutPermanentIntoItsOwnersHand(You)`, `PutEachPermanentIntoItsOwnersHand()`, `PutEachPermanentIntoItsOwnersHand(And([]))`, `PutEachPermanentIntoItsOwnersHand(Ref_TargetPermanent)`} {
		_, err := Compile(nameScopeNode(t, `Card(Name:"Invalid return",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([`+action+`])))])`))
		if err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONReturnInstructionMixedState(t *testing.T) {
	g, a, b := ronGame()
	owned := mage.NewEnchantment("Owned", "")
	owned.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(owned, a.PlayerID())
	foreign := mage.NewEnchantment("Foreign", "")
	foreign.SetOwner(b.PlayerID())
	q := g.PutOnBattlefield(foreign, a.PlayerID())
	query, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
	visits := query.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
	obj := castRONSpell(t, g, a, repositoryCard(t, "LEG", "Remove Enchantments"))
	for range 2 {
		if certain, known := g.InspectSpellDestruction(obj, visits); certain || known {
			t.Fatal("departure supplied a later state proof")
		}
	}
	g.ResolveStack()
	if g.FindPermanent(p.ID()) != nil || g.FindPermanent(q.ID()) != nil || len(a.Hand()) != 1 || a.Hand()[0].ID() != owned.ID() || len(b.Graveyard()) != 1 || b.Graveyard()[0].ID() != foreign.ID() {
		t.Fatal("return and destruction did not keep order and owners")
	}
}

func TestRONReturnInstructionCopyController(t *testing.T) {
	g, a, b := ronGame()
	first := mage.NewEnchantment("First", "")
	first.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(first, a.PlayerID())
	second := mage.NewEnchantment("Second", "")
	second.SetOwner(b.PlayerID())
	q := g.PutOnBattlefield(second, b.PlayerID())
	query, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
	visits := query.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
	obj := castRONSpell(t, g, a, repositoryCard(t, "LEG", "Remove Enchantments"))
	spellCopy := g.CopyStackObjectDirect(obj, b.PlayerID(), false)
	if spellCopy == nil {
		t.Fatal("copy failed")
	}
	for i, spell := range []*mage.StackObject{obj, spellCopy} {
		if certain, known := g.InspectSpellDestruction(spell, visits[i:i+1]); certain || !known {
			t.Fatal("copy lost resolving controller")
		}
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(p.ID()) == nil || g.FindPermanent(q.ID()) != nil || len(b.Hand()) != 1 {
		t.Fatal("copy used original controller")
	}
	g.ResolveStack()
	if g.FindPermanent(p.ID()) != nil || len(a.Hand()) != 1 {
		t.Fatal("copy changed original instance")
	}
}

func TestRONReturnInstructionLowering(t *testing.T) {
	record := repositoryRecord(t, "LEG", "Remove Enchantments", "LEG.ron")
	node := contextualQueryNode(t, record, "PutEachPermanentIntoItsOwnersHand")
	plan, err := compileDirectReturnPlan(node, scope{resolving: true})
	if err != nil {
		t.Fatal(err)
	}
	declaration := plan.build(instanceBindings{})
	instructions, ok := mage.InspectDirectReturn(declaration.Effect())
	if !ok || len(instructions) != 1 || instructions[0].Subject != nil {
		t.Fatal("lowering lost simultaneous group")
	}
	if _, ok := mage.InspectContextualPermanentCollection(instructions[0].Subjects); !ok {
		t.Fatal("lowering lost complete real collection")
	}
	for _, syntax := range []string{`Unknown(AnyPermanent)`, `PutPermanentIntoItsOwnersHand(Ref_TargetPermanent)`, `PutEachPermanentIntoItsOwnersHand(And([IsCardtype(Artifact),Unknown]))`} {
		if _, err := compileDirectReturnPlan(nameScopeNode(t, syntax), scope{resolving: true}); err == nil {
			t.Fatalf("accepted %s", syntax)
		}
	}
}
