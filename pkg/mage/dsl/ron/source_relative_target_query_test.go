package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSourceRelativeTargetQueryLowering(t *testing.T) {
	for _, syntax := range []string{`PowerIs(LessThanOrEqualTo(PowerOfPermanent(ThisPermanent)))`, `ToughnessIs(LessThan(PowerOfPermanent(ThisPermanent)))`} {
		filter, options, err := compilePermanentTargetFilter(nameScopeNode(t, `And([IsCardtype(Creature),`+syntax+`])`), scope{permanent: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := mage.InspectPermanentTarget(mage.DeclarePermanentTarget("source-relative", filter, options...)); !ok {
			t.Fatal("lowered source-relative target is opaque")
		}
		if _, _, err := compilePermanentTargetFilter(nameScopeNode(t, syntax), scope{}); err == nil {
			t.Fatal("missing source scope accepted")
		}
	}
	for _, syntax := range []string{`PowerIs(LessThanOrEqualTo(PowerOfPermanent(ThisPermanent())))`, `PowerIs(LessThanOrEqualTo(PowerOfPermanent(Ref_TargetPermanent)))`, `PowerIs(LessThanOrEqualTo(PowerOfPermanent(ThisPermanent,ThisPermanent)))`, `ToughnessIs(LessThan(PowerOfPermanent(ThisPermanent())))`, `ToughnessIs(LessThan(PowerOfPermanent(HostPermanent)))`, `ToughnessIs(LessThan())`, `Or([IsTapped,PowerIs(LessThanOrEqualTo(PowerOfPermanent(ThisPermanent)))])`, `And([Unknown,ToughnessIs(LessThan(PowerOfPermanent(ThisPermanent)))])`} {
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:3,Toughness:4),Rules:[Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(`+syntax+`)],ActionList([DestroyPermanent(Ref_TargetPermanent)])))])`)
		if _, err := Compile(node); err == nil {
			t.Fatal("invalid target accepted", syntax)
		}
	}
}

func TestRONSourceRelativeTargetQueryRecords(t *testing.T) {
	for _, set := range []string{"ARN", "LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		name, index := "Stone Giant", 0
		if set == "ARN" {
			name, index = "Old Man of the Sea", 1
		}
		t.Run(set, func(t *testing.T) {
			record := repositoryRecord(t, set, name, "source-relative.ron")
			var declaration mage.PermanentRef
			count := 0
			var walk func(Node)
			walk = func(n Node) {
				if n.Name == "TargetPermanent" {
					filter, options, err := compilePermanentTargetFilter(n.Args[0], scope{permanent: true})
					if err != nil {
						t.Fatal(err)
					}
					declaration = mage.DeclarePermanentTarget("record target", filter, options...)
					if _, ok := mage.InspectPermanentTarget(declaration); !ok {
						t.Fatal("complete record target opaque")
					}
					count++
				}
				for _, child := range n.Args {
					walk(child)
				}
				for _, child := range n.Fields {
					walk(child)
				}
			}
			walk(record)
			if count != 1 {
				t.Fatal("target count changed", count)
			}
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			controller := a.PlayerID()
			if set == "ARN" {
				controller = b.PlayerID()
			}
			victim := g.PutOnBattlefield(mage.NewCreature("Subject", "", 2, 2), controller)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			probe, _ := mage.InspectPermanentTarget(declaration)
			if _, known := probe.Select(g, g.StackPeek()); known {
				t.Fatal("record probe shared factory declaration")
			}
			g.ResolveStack()
			if set == "ARN" {
				if g.FindPermanent(victim.ID()).ControllerID() != a.PlayerID() {
					t.Fatal("control failed")
				}
			} else {
				if !g.FindPermanent(victim.ID()).HasKeyword(core.Flying) {
					t.Fatal("flying failed")
				}
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.RunStepWithPriority(core.EndStep)
				if g.FindPermanent(victim.ID()) != nil {
					t.Fatal("delayed destruction failed")
				}
			}
		})
	}
}

func TestRONSourceRelativeTargetQueryBindings(t *testing.T) {
	for _, comparison := range []string{`PowerIs(LessThanOrEqualTo(PowerOfPermanent(ThisPermanent)))`, `ToughnessIs(LessThan(PowerOfPermanent(ThisPermanent)))`} {
		factory, err := Compile(nameScopeNode(t, `Card(Name:"Source",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:3,Toughness:4),Rules:[Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(And([IsCardtype(Creature),`+comparison+`]))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))])`))
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		one, two := factory(), factory()
		one.SetOwner(a.PlayerID())
		two.SetOwner(a.PlayerID())
		first := g.PutOnBattlefield(one, a.PlayerID())
		second := g.PutOnBattlefield(two, a.PlayerID())
		first.RevokeBaseAttr(core.AttrSummonSick)
		second.RevokeBaseAttr(core.AttrSummonSick)
		tooLarge := g.PutOnBattlefield(mage.NewCreature("Too large", "", 4, 4), b.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, []uuid.UUID{tooLarge.ID()}); err == nil || g.StackSize() != 0 || g.FindPermanent(first.ID()).Tapped {
			t.Fatal("illegal comparison paid tap cost")
		}
		victim := g.PutOnBattlefield(mage.NewCreature("Subject", "", 2, 2), b.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
			t.Fatal(err)
		}
		obj := g.StackPeek()
		instructions, ok := mage.InspectDirectDestruction(mage.Sequence(obj.Effects...))
		if !ok || len(instructions) != 1 {
			t.Fatal("destruction lost")
		}
		plan, ok := mage.InspectPermanentTarget(instructions[0].Subject)
		if !ok {
			t.Fatal("actual RON binding is opaque")
		}
		if got, known := plan.Select(g, obj); !known || len(got) != 1 {
			t.Fatal("actual target not selected")
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), second.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
			t.Fatal(err)
		}
		if _, known := plan.Select(g, g.StackPeek()); known {
			t.Fatal("factories share declaration identity")
		}
		clone := g.Clone()
		clone.MutablePermanent(first.ID()).AddCounter(core.M1M1, 2)
		if got, known := plan.Select(clone, clone.GetStack().Objects()[0]); !known || len(got) != 0 {
			t.Fatal("clone source power ignored")
		}
		copied := g.CopyStackObjectDirect(obj, b.PlayerID(), false)
		if got, known := plan.Select(g, copied); !known || len(got) != 1 {
			t.Fatal("copy lost source or declaration")
		}
		g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
		g.PutOnBattlefield(one, a.PlayerID()).AddCounter(core.M1M1, 2)
		if got, known := plan.Select(g, obj); !known || len(got) != 1 {
			t.Fatal("returned source replaced saved power")
		}
		g.ResolveStack()
		if g.FindPermanent(victim.ID()) != nil {
			t.Fatal("native destruction differs")
		}
	}
}
