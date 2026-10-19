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

func TestRONPermanentStateTargetLowering(t *testing.T) {
	for _, syntax := range []string{`IsTapped`, `IsUntapped`, `IsAttacking`, `IsntAttacking`, `IsBlocking`, `Or([IsTapped,IsBlocking])`, `And([IsCardtype(Creature),Or([IsUntapped,IsntAttacking]),IsNonColor(Black)])`} {
		filter, handled, err := compileInspectableTargetFilter(nameScopeNode(t, syntax))
		if err != nil || !handled {
			t.Fatalf("lowering %s: %v/%v", syntax, handled, err)
		}
		if _, ok := mage.InspectPermanentTarget(mage.DeclarePermanentTarget("state", filter)); !ok {
			t.Fatal("native state predicate lost", syntax)
		}
	}
	for _, syntax := range []string{`IsTapped()`, `IsUntapped(Unknown)`, `IsAttacking()`, `IsntAttacking(IsBlocking)`, `IsBlocking()`, `And([IsTapped,Or([])])`, `Or([IsBlocking,IsUntapped()])`} {
		if _, _, err := compileInspectableTargetFilter(nameScopeNode(t, syntax)); err == nil {
			t.Fatal("invalid predicate accepted", syntax)
		}
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(`+syntax+`)],ActionList([DestroyPermanent(Ref_TargetPermanent)])))])`)
		if _, err := Compile(node); err == nil {
			t.Fatal("invalid full card accepted", syntax)
		}
	}
	for _, syntax := range []string{`Or([IsTapped,Unknown])`, `And([IsBlocking,AttackedThisTurn])`, `PowerIs(LessThanOrEqualTo(PowerOfPermanent(ThisPermanent)))`} {
		if _, handled, _ := compileInspectableTargetFilter(nameScopeNode(t, syntax)); handled {
			t.Fatal("partial state tree accepted", syntax)
		}
	}
}

func TestRONPermanentStateTargetDestruction(t *testing.T) {
	for _, tc := range []struct{ set, name string }{
		{"LEA", "Royal Assassin"}, {"LEB", "Royal Assassin"}, {"2ED", "Royal Assassin"}, {"3ED", "Royal Assassin"}, {"CED", "Royal Assassin"}, {"CEI", "Royal Assassin"}, {"LEG", "Tetsuo Umezawa"},
	} {
		for _, state := range []string{"legal", "untap", "blocking", "remove", "phase cycle", "return", "copy", "clone", "source leaves", "source control", "shroud"} {
			t.Run(tc.set+"/"+tc.name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				victimCard := mage.NewCreature("Subject", "", 2, 5)
				victimCard.SetOwner(b.PlayerID())
				victim := g.PutOnBattlefield(victimCard, b.PlayerID())
				if state == "blocking" {
					attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 5), a.PlayerID())
					g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
					g.GetCombat().AddBlocker(victim.ID(), attacker.ID())
				} else {
					g.TapPermanent(victim)
				}
				if tc.name == "Tetsuo Umezawa" {
					a.ManaPool().Add(core.Blue, 1)
					a.ManaPool().Add(core.Black, 2)
					a.ManaPool().Add(core.Red, 1)
				}
				index := 0
				if tc.name == "Tetsuo Umezawa" {
					index = 1
				}
				err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, []uuid.UUID{victim.ID()})
				if state == "blocking" && tc.name == "Royal Assassin" {
					if err == nil || g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped {
						t.Fatal("illegal target changed state")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				obj := g.StackPeek()
				instructions, ok := mage.InspectDirectDestruction(mage.Sequence(obj.Effects...))
				if !ok || len(instructions) != 1 {
					t.Fatal("native destruction lost")
				}
				plan, ok := mage.InspectPermanentTarget(instructions[0].Subject)
				if !ok {
					t.Fatal("complete real target is opaque")
				}
				want := true
				switch state {
				case "untap":
					g.UntapPermanent(victim)
					want = false
				case "remove":
					g.RemoveFromCombat(victim.ID())
				case "phase cycle":
					g.PhaseIn(g.PhaseOut(victim.ID()))
				case "return":
					g.RemoveFromBattlefield(victim)
					g.PutOnBattlefield(victimCard, b.PlayerID())
					want = false
				case "copy":
					obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
				case "clone":
					clone := g.Clone()
					clone.UntapPermanent(clone.FindPermanent(victim.ID()))
					if got, known := plan.Select(clone, clone.StackPeek()); !known || len(got) != 0 {
						t.Fatal("clone query ignored untap")
					}
					clone.ResolveStack()
					if clone.FindPermanent(victim.ID()) == nil {
						t.Fatal("clone destroyed illegal target")
					}
				case "source leaves":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				case "source control":
					g.AddControlEffect(mage.ControlEffectSpec{TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "shroud":
					g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
					want = false
				}
				if got, known := plan.Select(g, obj); !known || (len(got) == 1) != want {
					t.Fatalf("query %v/%v want %v", got, known, want)
				}
				g.ResolveStack()
				if (g.FindPermanentIncludingPhased(victim.ID()) == nil) != want {
					t.Fatal("native destruction and query differ")
				}
			})
		}
	}
}

func TestRONPermanentStateTargetRecords(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil || len(files) == 0 {
		t.Fatal("missing collection", err)
	}
	names := map[string]bool{}
	records := 0
	var hasState func(Node) bool
	hasState = func(n Node) bool {
		return slices.Contains([]string{"IsTapped", "IsUntapped", "IsAttacking", "IsntAttacking", "IsBlocking"}, n.Name) || slices.ContainsFunc(n.Args, hasState)
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
				if n.Name == "TargetPermanent" && len(n.Args) == 1 && hasState(n.Args[0]) {
					filter, options, err := compilePermanentTargetFilter(n.Args[0], scope{})
					if err != nil {
						t.Fatal(err)
					}
					if _, ok := mage.InspectPermanentTarget(mage.DeclarePermanentTarget("audit", filter, options...)); !ok {
						t.Fatal("state target opaque", node.Fields["Name"].String)
					}
					found = true
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
				names[node.Fields["Name"].String] = true
				records++
			}
		}
	}
	list := make([]string, 0, len(names))
	for name := range names {
		list = append(list, name)
	}
	slices.Sort(list)
	t.Logf("state targets %d names / %d records: %v", len(names), records, list)
	if len(names) != 19 || records != 30 {
		t.Fatal("state target coverage changed")
	}
}

func TestRONPermanentStateTargetBindings(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(And([IsCardtype(Creature),IsTapped])),TargetPermanent(And([IsCardtype(Creature),IsUntapped]))],ActionList([DestroyPermanent(Ref_TargetPermanent1),DestroyPermanentNoRegen(Ref_TargetPermanent2)])))`)
	g, a, b := ronGame()
	one := g.PutOnBattlefield(mage.NewCreature("One", "", 1, 3), b.PlayerID())
	two := g.PutOnBattlefield(mage.NewCreature("Two", "", 1, 3), b.PlayerID())
	g.TapPermanent(one)
	first := castRONSpell(t, g, a, factory(), one.ID(), two.ID())
	second := castRONSpell(t, g, a, factory(), one.ID(), two.ID())
	instructions, ok := mage.InspectDirectDestruction(mage.Sequence(first.Effects...))
	if !ok || len(instructions) != 2 {
		t.Fatal("groups lost")
	}
	tapped, ok := mage.InspectPermanentTarget(instructions[0].Subject)
	if !ok {
		t.Fatal("tapped declaration lost")
	}
	untapped, ok := mage.InspectPermanentTarget(instructions[1].Subject)
	if !ok {
		t.Fatal("untapped declaration lost")
	}
	if _, known := tapped.Select(g, second); known {
		t.Fatal("factory declarations share identity")
	}
	g.UntapPermanent(one)
	if got, known := tapped.Select(g, first); !known || len(got) != 0 {
		t.Fatal("illegal group selected")
	}
	if got, known := untapped.Select(g, first); !known || len(got) != 1 || got[0].ID != two.ID() {
		t.Fatal("independent group lost")
	}
	g.AddRegenerationShield(two.ID())
	g.ResolveStack()
	if g.FindPermanent(one.ID()) == nil || g.FindPermanent(two.ID()) != nil {
		t.Fatal("illegal group blocked independent destruction")
	}
}
