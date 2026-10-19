package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSingularDamageInstructionRealCards(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		amount    int
	}{{"LEB", "Lightning Bolt", 3}, {"LEG", "Psychic Purge", 1}} {
		for _, state := range []string{"normal", "player", "phase", "cycle", "shroud", "protection", "prevention", "lethal", "clone", "copy"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				toughness := 5
				if state == "lethal" {
					toughness = 1
				}
				p := g.PutOnBattlefield(mage.NewCreature("Animated land", "", 1, toughness), b.PlayerID())
				p.GrantBaseAttr(core.AttrIsLand)
				query, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
				watched := query.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
				target := p.ID()
				if state == "player" {
					target = b.PlayerID()
				}
				obj := castRONSpell(t, g, a, repositoryCard(t, tc.set, tc.name), target)
				switch state {
				case "phase":
					g.PhaseOut(p.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "shroud":
					p.GrantBaseAttr(core.Shroud)
				case "protection":
					p.RuntimeAbilities = append(p.RuntimeAbilities, mage.ProtectionFromColor(core.Red), mage.ProtectionFromColor(core.Blue))
				case "prevention":
					g.AddPreventionShield(p.ID(), 2)
				case "copy":
					obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
				}
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, watched); certain || !known {
						t.Fatal("damage spell has no pure no-destruction result")
					}
				}
				if p.Damage != 0 || b.Life() != 20 {
					t.Fatal("inspection dealt damage")
				}
				if state == "clone" {
					clone := g.Clone()
					if certain, known := clone.InspectSpellDestruction(clone.StackPeek(), watched); certain || !known {
						t.Fatal("clone lost declaration")
					}
					clone.ResolveTopOfStack()
					if clone.FindPermanent(p.ID()).Damage != tc.amount || p.Damage != 0 {
						t.Fatal("clone crossed damage state")
					}
				}
				g.ResolveTopOfStack()
				switch state {
				case "player":
					if b.Life() != 20-tc.amount {
						t.Fatal("wrong player damage")
					}
				case "lethal":
					if g.FindPermanent(p.ID()) != nil {
						t.Fatal("lethal damage did not cause state-based death")
					}
				default:
					want := tc.amount
					if state == "phase" || state == "shroud" || state == "protection" {
						want = 0
					}
					if state == "prevention" {
						want = max(0, want-2)
					}
					if g.FindPermanentIncludingPhased(p.ID()).Damage != want {
						t.Fatal("wrong permanent damage")
					}
				}
			})
		}
	}
}

func TestRONSingularDamageInstructionInvalid(t *testing.T) {
	for _, action := range []string{`SpellDealsDamage(ThisSpell,Integer(3))`, `SpellDealsDamage(ThisPermanent,Integer(3),Ref_AnyTarget)`, `SpellDealsDamage(ThisSpell,Integer(3),Ref_AnyTarget2)`, `SpellDealsDamage(ThisSpell,Integer(3),You)`, `SpellDealsDamage(ThisSpell,Integer(),Ref_AnyTarget)`} {
		_, err := Compile(nameScopeNode(t, `Card(Name:"Invalid damage",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([AnyTarget],ActionList([`+action+`])))])`))
		if err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONSingularDamageInstructionBindings(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([AnyTarget,AnyTarget],ActionList([SpellDealsDamage(ThisSpell,Integer(2),Ref_AnyTarget1),SpellDealsDamage(ThisSpell,Integer(1),Ref_AnyTarget2)])))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 8), b.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 8), b.PlayerID())
	first := castRONSpell(t, g, a, factory(), p.ID(), q.ID())
	second := castRONSpell(t, g, a, factory(), q.ID(), p.ID())
	for _, obj := range []*mage.StackObject{first, second} {
		instructions, ok := mage.InspectSingularDamageInstructions(mage.Sequence(obj.Effects...))
		if !ok || len(instructions) != 2 || instructions[0].Recipient == instructions[1].Recipient {
			t.Fatal("lowering lost separate damage declarations")
		}
		if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
			t.Fatal("numbered recipient tree is unknown")
		}
	}
	p.GrantBaseAttr(core.Shroud)
	g.ResolveTopOfStack()
	if p.Damage != 0 || q.Damage != 2 {
		t.Fatal("second factory or partial legality differs")
	}
	g.ResolveStack()
	if p.Damage != 0 || q.Damage != 3 {
		t.Fatal("first factory lost independent bindings")
	}
}
