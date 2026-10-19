package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPermanentSourceExclusionLowering(t *testing.T) {
	for _, syntax := range []string{`Other(ThisPermanent)`, `And([IsCardtype(Creature),Other(ThisPermanent)])`, `And([Other(ThisPermanent),Or([IsTapped,IsNonColor(Red)])])`} {
		filter, options, err := compilePermanentTargetFilter(nameScopeNode(t, syntax), scope{permanent: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := mage.InspectPermanentTarget(mage.DeclarePermanentTarget("other", filter, options...)); !ok {
			t.Fatal("lowered exclusion is opaque", syntax)
		}
	}
	for _, syntax := range []string{`Other`, `Other()`, `Other(ThisPermanent,ThisPermanent)`, `Other(ThisPermanent())`, `Other(HostPermanent)`, `Other(Ref_TargetPermanent)`, `Or([Other(ThisPermanent),IsTapped])`, `And([Other(ThisPermanent),Unknown])`} {
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(`+syntax+`)],ActionList([DestroyPermanent(Ref_TargetPermanent)])))])`)
		if _, err := Compile(node); err == nil {
			t.Fatal("invalid exclusion accepted", syntax)
		}
	}
	if _, _, err := compilePermanentTargetFilter(nameScopeNode(t, `Other(ThisPermanent)`), scope{}); err == nil {
		t.Fatal("source exclusion outside permanent scope")
	}
}

func TestRONPermanentSourceExclusionGameplay(t *testing.T) {
	for _, set := range []string{"ARN", "3ED"} {
		g, a, b := ronGame()
		card := repositoryCard(t, set, "Sorceress Queen")
		card.SetOwner(a.PlayerID())
		source := g.PutOnBattlefield(card, a.PlayerID())
		source.RevokeBaseAttr(core.AttrSummonSick)
		victim := g.PutOnBattlefield(mage.NewCreature("Subject", "", 4, 5), b.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{source.ID()}); err == nil || g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped {
			t.Fatal("self target paid tap cost")
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
			t.Fatal(err)
		}
		g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
		g.ResolveStack()
		if p := g.FindPermanent(victim.ID()); p.CurrentPower(g) != 0 || p.CurrentToughness(g) != 2 {
			t.Fatal("departed source lost other target")
		}
	}
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Halfdane")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("Halfdane targeted itself")
	}
	victim := g.PutOnBattlefield(mage.NewCreature("Subject", "", 4, 5), b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.StackPeek().Targets[0] != victim.ID() {
		t.Fatal("other target missing")
	}
	g.ResolveStack()
	if p := g.FindPermanent(source.ID()); p.CurrentPower(g) != 4 || p.CurrentToughness(g) != 5 {
		t.Fatal("Halfdane values differ")
	}
}

func TestRONPermanentSourceExclusionBindings(t *testing.T) {
	factory, err := Compile(nameScopeNode(t, `Card(Name:"Source",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:3),Rules:[Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(And([Other(ThisPermanent),IsCardtype(Creature)]))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))])`))
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
	victim := g.PutOnBattlefield(mage.NewCreature("Subject", "", 2, 3), b.PlayerID())
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
		t.Fatal("RON target opaque")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), second.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	if _, known := plan.Select(g, g.StackPeek()); known {
		t.Fatal("factory declarations share identity")
	}
	clone := g.Clone()
	clone.PhaseOut(victim.ID())
	if got, known := plan.Select(clone, clone.GetStack().Objects()[0]); !known || len(got) != 0 {
		t.Fatal("clone state ignored")
	}
	copied := g.CopyStackObjectDirect(obj, b.PlayerID(), false)
	if got, known := plan.Select(g, copied); !known || len(got) != 1 {
		t.Fatal("copy lost original source binding")
	}
	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	if got, known := plan.Select(g, obj); !known || len(got) != 1 {
		t.Fatal("source LKI lost")
	}
	g.ResolveStack()
	if g.FindPermanent(victim.ID()) != nil {
		t.Fatal("native resolution differs")
	}
}

func TestRONPermanentSourceExclusionRecords(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"ARN", "Sorceress Queen"}, {"3ED", "Sorceress Queen"}, {"LEG", "Halfdane"}} {
		card := repositoryCard(t, tc.set, tc.name)
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		node := repositoryRecord(t, tc.set, tc.name, "source-exclusion.ron")
		count := 0
		var walk func(Node)
		walk = func(n Node) {
			if n.Name == "TargetPermanent" {
				filter, options, err := compilePermanentTargetFilter(n.Args[0], scope{permanent: true})
				if err != nil {
					t.Fatal(err)
				}
				if _, ok := mage.InspectPermanentTarget(mage.DeclarePermanentTarget("real target", filter, options...)); !ok {
					t.Fatal("complete target is opaque", tc.name)
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
		walk(node)
		if count != 1 {
			t.Fatal("source-exclusion target count changed", tc.name, count)
		}
	}
}
