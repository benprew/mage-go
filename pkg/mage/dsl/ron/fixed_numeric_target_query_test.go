package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONFixedNumericTargetQueryLowering(t *testing.T) {
	for _, syntax := range []string{
		`PowerAndToughnessIsEqualTo(PT(1,1))`,
		`ToughnessIs(EqualTo(Integer(-1)))`,
		`ToughnessIs(LessThanOrEqualTo(Integer(2)))`,
		`ToughnessIs(GreaterThanOrEqualTo(Integer(0)))`,
		`Or([IsTapped,PowerAndToughnessIsEqualTo(PT(1,1))])`,
		`And([IsCardtype(Creature),Or([ToughnessIs(LessThanOrEqualTo(Integer(2))),IsBlocking])])`,
	} {
		filter, handled, err := compileInspectableTargetFilter(nameScopeNode(t, syntax))
		if err != nil || !handled {
			t.Fatalf("fixed target %s: %v/%v", syntax, handled, err)
		}
		if _, ok := mage.InspectPermanentTarget(mage.DeclarePermanentTarget("fixed", filter)); !ok {
			t.Fatal("lowered numeric tree opaque", syntax)
		}
	}
	for _, syntax := range []string{
		`PowerAndToughnessIsEqualTo(PT(1))`, `PowerAndToughnessIsEqualTo(PT(1,"1"))`,
		`PowerAndToughnessIsEqualTo(PT(1,1),Unknown)`,
		`ToughnessIs(EqualTo(Integer(1,2)))`, `ToughnessIs(LessThanOrEqualTo(Integer()))`,
		`ToughnessIs(GreaterThanOrEqualTo(Integer("2")))`, `ToughnessIs(EqualTo(ValueX))`,
		`ToughnessIs(LessThan(Integer(2)))`, `Or([PowerAndToughnessIsEqualTo(PT(1,1)),Unknown])`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(`+syntax+`)],ActionList([DestroyPermanent(Ref_TargetPermanent)])))])`)); err == nil {
			t.Fatal("invalid declaration accepted", syntax)
		}
	}
}

func TestRONFixedNumericTargetQueryRecords(t *testing.T) {
	for _, tc := range []struct {
		set, name               string
		index, power, toughness int
	}{
		{"LEG", "Pendelhaven", 1, 1, 1}, {"FEM", "Goblin Kites", 0, 3, 2},
	} {
		for _, state := range []string{"legal", "counter", "control", "return", "phase", "phase cycle", "source leaves", "copy", "clone", "shroud"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				record := repositoryRecord(t, tc.set, tc.name, "fixed-target.ron")
				count := 0
				var walk func(Node)
				walk = func(n Node) {
					if n.Name == "TargetPermanent" {
						f, opts, err := compilePermanentTargetFilter(n.Args[0], scope{permanent: true})
						if err != nil {
							t.Fatal(err)
						}
						if _, ok := mage.InspectPermanentTarget(mage.DeclarePermanentTarget("record", f, opts...)); !ok {
							t.Fatal("full record target opaque")
						}
						count++
					}
					for _, v := range n.Args {
						walk(v)
					}
					for _, v := range n.Fields {
						walk(v)
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
				card.SetOwner(a.PlayerID())
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				source := g.PutOnBattlefield(card, a.PlayerID())
				victimCard := mage.NewCreature("Subject", "", tc.power, tc.toughness)
				victimCard.SetOwner(a.PlayerID())
				victim := g.PutOnBattlefield(victimCard, a.PlayerID())
				tooLarge := g.PutOnBattlefield(mage.NewCreature("Too large", "", 3, 3), a.PlayerID())
				a.ManaPool().Add(core.Red, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), tc.index, []uuid.UUID{tooLarge.ID()}); err == nil || g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped || a.ManaPool().Count(core.Red) != 1 {
					t.Fatal("illegal numeric target spent cost")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), tc.index, []uuid.UUID{victim.ID()}); err != nil {
					t.Fatal(err)
				}
				legal := true
				switch state {
				case "counter":
					victim.AddCounter(core.P1P1, 1)
					legal = false
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{TargetID: victim.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					legal = tc.name == "Pendelhaven"
				case "return":
					g.RemoveFromBattlefield(victim)
					victim = g.PutOnBattlefield(victimCard, a.PlayerID())
					legal = false
				case "phase":
					g.PhaseOut(victim.ID())
					legal = false
				case "phase cycle":
					g.PhaseIn(g.PhaseOut(victim.ID()))
				case "source leaves":
					g.RemoveFromBattlefield(source)
				case "copy":
					g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
				case "clone":
					clone := g.Clone()
					clone.MutablePermanent(victim.ID()).AddCounter(core.P1P1, 1)
					clone.ResolveStack()
					if clone.FindPermanent(victim.ID()).HasKeyword(core.Flying) || len(clone.DelayedTriggers()) != 0 || clone.FindPermanent(victim.ID()).CurrentPower(clone) != tc.power+1 {
						t.Fatal("clone ignored numeric legality")
					}
				case "shroud":
					victim.GrantBaseAttr(core.Shroud)
					legal = false
				}
				g.ResolveStack()
				p := g.FindPermanentIncludingPhased(victim.ID())
				if tc.name == "Pendelhaven" {
					want := tc.power
					if state == "counter" {
						want++
					}
					if legal {
						want++
					}
					if p.CurrentPower(g) != want {
						t.Fatal("wrong current power", p.CurrentPower(g), want)
					}
				} else {
					if p.HasKeyword(core.Flying) != legal || (len(g.DelayedTriggers()) > 0) != legal {
						t.Fatal("wrong flying or delay legality")
					}
					if legal {
						p.AddCounter(core.P1P1, 3)
						g.SetCoinFlipResults([]bool{false, false})
						g.RunStepWithPriority(core.EndStep)
						if g.FindPermanent(p.ID()) != nil {
							t.Fatal("delayed sacrifice repeated numeric targeting restriction")
						}
					}
				}
			})
		}
	}
}

func TestRONFixedNumericTargetQueryBindings(t *testing.T) {
	factory, err := Compile(nameScopeNode(t, `Card(Name:"Source",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(PowerAndToughnessIsEqualTo(PT(1,1)))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))])`))
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	first := g.PutOnBattlefield(factory(), a.PlayerID())
	second := g.PutOnBattlefield(factory(), a.PlayerID())
	victim := g.PutOnBattlefield(mage.NewCreature("Subject", "", 1, 1), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	obj := g.StackPeek()
	instructions, ok := mage.InspectDirectDestruction(mage.Sequence(obj.Effects...))
	if !ok || len(instructions) != 1 {
		t.Fatal("destruction missing")
	}
	plan, ok := mage.InspectPermanentTarget(instructions[0].Subject)
	if !ok {
		t.Fatal("actual lowered numeric binding opaque")
	}
	if got, known := plan.Select(g, obj); !known || len(got) != 1 {
		t.Fatal("actual target lost")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), second.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	if _, known := plan.Select(g, g.StackPeek()); known {
		t.Fatal("factory instances share declaration")
	}
	copied := g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	if got, known := plan.Select(g, copied); !known || len(got) != 1 {
		t.Fatal("copy lost declaration")
	}
	g.MutablePermanent(victim.ID()).AddCounter(core.P1P1, 1)
	if got, known := plan.Select(g, copied); !known || len(got) != 0 {
		t.Fatal("copy ignored current values")
	}
	g.ResolveStack()
	if g.FindPermanent(victim.ID()) == nil {
		t.Fatal("illegal target destroyed")
	}
}
