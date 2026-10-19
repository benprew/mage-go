package ron

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPermanentSubtypeTargetLowering(t *testing.T) {
	for _, syntax := range []string{`IsCreatureType(Wall)`, `IsNonCreatureType(Elf)`, `IsLandType(Island)`, `IsNonLandType(Mountain)`, `IsEnchantmentType(Aura)`, `Or([IsCreatureType(Wall),And([IsCardtype(Land),IsLandType(Forest),IsNonColor(Black)])])`} {
		filter, handled, err := compileInspectableTargetFilter(nameScopeNode(t, syntax))
		if err != nil || !handled {
			t.Fatalf("lowering %s: %v/%v", syntax, handled, err)
		}
		if _, ok := mage.InspectPermanentTarget(mage.DeclarePermanentTarget("subtype", filter)); !ok {
			t.Fatal("subtype plan lost")
		}
	}
	for _, syntax := range []string{`IsCreatureType()`, `IsLandType`, `IsLandType(Island,Forest)`, `IsCreatureType("Wall")`, `IsCreatureType(Wall())`, `IsCreatureType(Island)`, `IsLandType(Wall)`, `IsNonLandType(Goblin)`, `IsEnchantmentType(Forest)`, `And([IsColor(Red),IsCreatureType(Island)])`} {
		if _, _, err := compileInspectableTargetFilter(nameScopeNode(t, syntax)); err == nil {
			t.Fatalf("invalid subtype accepted: %s", syntax)
		}
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(`+syntax+`)],ActionList([DestroyPermanent(Ref_TargetPermanent)])))])`)
		if _, err := Compile(node); err == nil {
			t.Fatalf("invalid card accepted: %s", syntax)
		}
	}
	for _, syntax := range []string{`And([IsCreatureType(Wall),IsEnchanted])`, `Or([IsLandType(Forest),Unknown])`} {
		if _, handled, _ := compileInspectableTargetFilter(nameScopeNode(t, syntax)); handled {
			t.Fatal("partial subtype tree claimed")
		}
	}
}

func TestRONPermanentSubtypeTargetTunnel(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, state := range []string{"normal", "changed subtype", "phase", "phase cycle", "return", "clone", "copy", "shroud", "shield"} {
			t.Run(set+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := mage.NewCreature("Subject", "", 0, 5, mage.WithSubTypes("Wall"))
				card.SetOwner(b.PlayerID())
				p := g.PutOnBattlefield(card, b.PlayerID())
				obj := castRONSpell(t, g, a, repositoryCard(t, set, "Tunnel"), p.ID())
				instructions, ok := mage.InspectDirectDestruction(mage.Sequence(obj.Effects...))
				if !ok || len(instructions) != 1 {
					t.Fatal("instruction lost")
				}
				plan, ok := mage.InspectPermanentTarget(instructions[0].Subject)
				if !ok {
					t.Fatal("Tunnel target rejected")
				}
				want := true
				switch state {
				case "changed subtype":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetSubtypeFamily(mage.This, mage.SubtypeCreature, "Elf")), p.ID(), p.ControllerID(), nil); err != nil {
						t.Fatal(err)
					}
					want = false
				case "phase", "phase cycle":
					g.PhaseOut(p.ID())
					want = state == "phase cycle"
					if want {
						g.PhaseIn([]uuid.UUID{p.ID()})
					}
				case "return":
					g.RemoveFromBattlefield(p)
					g.PutOnBattlefield(card, b.PlayerID())
					want = false
				case "shroud":
					p.GrantBaseAttr(core.Shroud)
					want = false
				case "shield":
					g.AddRegenerationShield(p.ID())
				case "copy":
					obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
				case "clone":
					clone := g.Clone()
					clone.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
					if got, known := plan.Select(clone, clone.StackPeek()); !known || len(got) != 0 {
						t.Fatal("clone shared target state")
					}
					clone.ResolveStack()
				}
				for range 2 {
					if got, known := plan.Select(g, obj); !known || (len(got) == 1) != want {
						t.Fatalf("selection=%v/%v want=%v", got, known, want)
					}
				}
				g.ResolveStack()
				if (g.FindPermanentIncludingPhased(p.ID()) == nil) != want {
					t.Fatal("native resolution differs")
				}
			})
		}
	}
}

func TestRONPermanentSubtypeTargetReturnModes(t *testing.T) {
	for _, tc := range []struct{ name, subtype string }{{"Active Volcano", "Island"}, {"Flash Flood", "Mountain"}} {
		for _, state := range []string{"normal", "changed subtype", "phase", "return", "copy"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				g, a, b := ronModalGame(1)
				card := mage.NewLand("Subject", mage.WithSubTypes(tc.subtype))
				card.SetOwner(b.PlayerID())
				p := g.PutOnBattlefield(card, b.PlayerID())
				castRONSpell(t, g, a.BasePlayer, repositoryCard(t, "LEG", tc.name), p.ID())
				want := state == "normal" || state == "copy"
				switch state {
				case "changed subtype":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetSubtypeFamily(mage.This, mage.SubtypeLand, "Forest")), p.ID(), p.ControllerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "phase":
					g.PhaseOut(p.ID())
				case "return":
					g.RemoveFromBattlefield(p)
					g.PutOnBattlefield(card, b.PlayerID())
				case "copy":
					g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
				}
				g.ResolveStack()
				if (g.FindPermanentIncludingPhased(p.ID()) == nil) != want {
					t.Fatal("return mode ignored subtype or visit")
				}
				if (len(b.Hand()) == 1) != want {
					t.Fatal("return mode used wrong destination")
				}
				if a.prompts != 1 {
					t.Fatal("mode was not retained")
				}
			})
		}
	}
}

func TestRONPermanentSubtypeTargetWizard(t *testing.T) {
	for _, state := range []string{"normal", "changed subtype", "source departure", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "DRK", "Goblin Wizard"), a.PlayerID())
			p := g.PutOnBattlefield(mage.NewCreature("Subject", "", 2, 2, mage.WithSubTypes("Goblin")), b.PlayerID())
			a.ManaPool().Add(core.Red, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{p.ID()}); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "changed subtype":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetSubtypeFamily(mage.This, mage.SubtypeCreature, "Elf")), p.ID(), p.ControllerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "source departure":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "copy":
				g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
			}
			g.ResolveStack()
			if g.FindPermanent(p.ID()).HasProtectionFrom(textColorCard(core.White)) != (state != "changed subtype") {
				t.Fatal("protection ignored target subtype")
			}
		})
	}
}

func TestRONPermanentSubtypeTargetDestructionAbilities(t *testing.T) {
	for _, tc := range []struct{ set, name, subtype string }{{"LEB", "Dwarven Demolition Team", "Wall"}, {"ARN", "King Suleiman", "Djinn"}, {"DRK", "Goblin Digging Team", "Wall"}} {
		for _, state := range []string{"normal", "changed subtype", "departed source", "protection LKI", "copy"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				p := g.PutOnBattlefield(mage.NewCreature("Subject", "", 2, 3, mage.WithSubTypes(tc.subtype)), b.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
					t.Fatal(err)
				}
				obj := g.StackPeek()
				instructions, ok := mage.InspectDirectDestruction(mage.Sequence(obj.Effects...))
				if !ok || len(instructions) != 1 {
					t.Fatal("native instruction lost")
				}
				plan, ok := mage.InspectPermanentTarget(instructions[0].Subject)
				if !ok {
					t.Fatal("native subtype declaration lost")
				}
				want := true
				switch state {
				case "changed subtype":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetSubtypeFamily(mage.This, mage.SubtypeCreature, "Elf")), p.ID(), p.ControllerID(), nil); err != nil {
						t.Fatal(err)
					}
					want = false
				case "departed source", "protection LKI":
					if live := g.FindPermanent(source.ID()); live != nil {
						g.RemoveFromBattlefield(live)
					}
					if state == "protection LKI" {
						p.RuntimeAbilities = append(p.RuntimeAbilities, mage.ProtectionFromColors(source.Card.ManaCost().Colors()...))
						want = false
					}
				case "copy":
					obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
				}
				if got, known := plan.Select(g, obj); !known || (len(got) == 1) != want {
					t.Fatalf("ability selection=%v/%v want=%v", got, known, want)
				}
				g.ResolveStack()
				if (g.FindPermanent(p.ID()) == nil) != want {
					t.Fatal("ability resolution differs")
				}
			})
		}
	}
}

func TestRONPermanentSubtypeTargetBindings(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsLandType(Forest)),TargetPermanent(IsCardtype(Land))],ActionList([DestroyPermanent(Ref_TargetPermanent1),DestroyPermanentNoRegen(Ref_TargetPermanent2)])))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(mage.NewLand("Subject", mage.WithSubTypes("Forest")), b.PlayerID())
	first := castRONSpell(t, g, a, factory(), p.ID(), p.ID())
	second := castRONSpell(t, g, a, factory(), p.ID(), p.ID())
	instructions, _ := mage.InspectDirectDestruction(mage.Sequence(first.Effects...))
	plan, ok := mage.InspectPermanentTarget(instructions[0].Subject)
	if !ok {
		t.Fatal("subtype lost")
	}
	if _, known := plan.Select(g, second); known {
		t.Fatal("factories shared declaration")
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetSubtypeFamily(mage.This, mage.SubtypeLand, "Island")), p.ID(), p.ControllerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got, known := plan.Select(g, first); !known || len(got) != 0 {
		t.Fatal("first group ignored changed subtype")
	}
	g.AddRegenerationShield(p.ID())
	g.ResolveStack()
	if g.FindPermanent(p.ID()) != nil {
		t.Fatal("illegal subtype group blocked legal independent group")
	}
}

func TestRONPermanentSubtypeTargetRecords(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil || len(files) == 0 {
		t.Fatal("missing RON collection", err)
	}
	names := map[string]bool{}
	records := 0
	var hasSubtype func(Node) bool
	hasSubtype = func(n Node) bool {
		switch n.Name {
		case "IsCreatureType", "IsNonCreatureType", "IsLandType", "IsNonLandType", "IsEnchantmentType":
			return true
		}
		return slices.ContainsFunc(n.Args, hasSubtype)
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			found := false
			var walk func(Node)
			walk = func(n Node) {
				if n.Name == "TargetPermanent" && len(n.Args) == 1 && hasSubtype(n.Args[0]) {
					_, handled, err := compileInspectableTargetFilter(n.Args[0])
					if err != nil {
						t.Fatal(err)
					}
					if handled {
						filter, options, err := compilePermanentTargetFilter(n.Args[0], scope{})
						if err != nil {
							t.Fatal(err)
						}
						if _, ok := mage.InspectPermanentTarget(mage.DeclarePermanentTarget("audit", filter, options...)); !ok {
							t.Fatalf("complete target rejected: %s", node.Fields["Name"].String)
						}
						found = true
					}
				}
				for _, child := range n.Args {
					walk(child)
				}
				for _, child := range n.Fields {
					walk(child)
				}
			}
			walk(node)
			if found {
				factory, err := Compile(node)
				if err != nil {
					t.Fatal(err)
				}
				if err := mage.ValidateCardDefinition(factory().Name(), factory()); err != nil {
					t.Fatal(err)
				}
				records++
				names[node.Fields["Name"].String] = true
			}
		}
	}
	list := make([]string, 0, len(names))
	for name := range names {
		list = append(list, name)
	}
	slices.Sort(list)
	t.Logf("pure subtype targets: %d names / %d records: %v", len(names), records, list)
	if len(names) == 0 {
		t.Fatal("no subtype target coverage")
	}
}
