package mage

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposableStateSacrifice(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Watcher", "", 1, 1, WithSubTypes("Dwarf"), WithAbility(WhenState(Controls(HasSubType("Dwarf")), Sacrifice(This))))
	g.PutOnBattlefield(c, a.PlayerID())
	g.CheckStateBasedActions()
	if g.FindPermanent(c.ID()) == nil {
		t.Fatal("state trigger bypassed stack")
	}
	g.CheckStateTriggers()
	g.PutTriggersOnStack()
	if g.stack.Size() != 1 {
		t.Fatalf("stack size = %d", g.stack.Size())
	}
	g.ResolveStackObject(g.stack.Pop())
	if g.FindPermanent(c.ID()) != nil {
		t.Fatal("source was not sacrificed")
	}
}

func TestComposableSourceReentry(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Watcher", "", 1, 1, WithAbility(WhenState(ControlsNone(IsLand), Sacrifice(This))))
	g.PutOnBattlefield(c, a.PlayerID())
	g.CheckStateTriggers()
	g.PutTriggersOnStack()
	old := g.stack.Pop()
	g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
	g.PutOnBattlefield(c, a.PlayerID())
	g.ResolveStackObject(old)
	if g.FindPermanent(c.ID()) == nil {
		t.Fatal("old trigger sacrificed new incarnation")
	}
}

func TestComposableConditions(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Dwarf", "", 1, 1, WithSubTypes("Dwarf"))
	g.PutOnBattlefield(c, a.PlayerID())
	conditions := []StateCondition{Controls(HasSubType("Dwarf")), ControlsNone(IsLand), BattlefieldHasNone(IsLand), AllConditions(Controls(IsCreature), NotCondition(Controls(IsLand))), AnyCondition(Controls(IsLand), Controls(IsCreature))}
	for _, condition := range conditions {
		ability := WhenState(condition, Sacrifice(This))
		ability.SetSource(c.ID())
		ability.SetController(a.PlayerID())
		for range 3 {
			if !ability.CheckTrigger(nil, g) {
				t.Fatal("condition false")
			}
		}
		if !ability.CheckTrigger(nil, g.Clone()) {
			t.Fatal("condition differs in clone")
		}
	}
}

func TestComposableDefinitionValidation(t *testing.T) {
	tests := []struct {
		name      string
		effect    ResolvingEffect
		condition StateCondition
		want      string
	}{
		{"empty sequence", Sequence(), ControlsNone(IsLand), "abilities[0].effects[0]"},
		{"nil subject", Sacrifice(nil), ControlsNone(IsLand), "effects[0].subject"},
		{"nil condition", Sacrifice(This), nil, "condition"},
		{"empty conditions", Sacrifice(This), AllConditions(), "condition"},
		{"nil child", Sacrifice(This), AnyCondition(nil), "conditions[0]"},
		{"nil callback", Sacrifice(This), StateConditionFunc("callback", []StateDependency{StateBattlefield}, nil), "callback"},
		{"missing dependencies", Sacrifice(This), StateConditionFunc("callback", nil, func(StateReader, StateContext) bool { return true }), "dependencies"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCreature("Invalid", "", 1, 1, WithAbility(WhenState(tt.condition, tt.effect)))
			err := ValidateCardDefinition("Invalid", c)
			if err == nil || !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), "Invalid") {
				t.Fatalf("error = %v", err)
			}
		})
	}
	c := NewInstant("Invalid spell", "", WithEffect(Sacrifice(This)))
	if err := ValidateCardDefinition("Invalid spell", c); err == nil {
		t.Fatal("accepted battlefield action from spell")
	}
}

func TestComposableMetadata(t *testing.T) {
	effect := Sequence(GainLifeFor(You, Fixed(2)), IfState(ControlsNone(IsLand), Sacrifice(This)))
	metadata := InspectDefinition(effect)
	if metadata.Role != "sequence" || len(metadata.Children) != 2 || metadata.Children[1].Role != "resolution-branch" {
		t.Fatalf("metadata = %+v", metadata)
	}
	if metadata.Properties.LifeGain != 2 || effect.Text() == "" {
		t.Fatal("lost effect metadata")
	}
}

func TestComposableSourceLifetimes(t *testing.T) {
	for _, kind := range []string{"event", "activated", "delayed"} {
		for _, roundtrip := range []string{"reentry", "phasing", "clone"} {
			t.Run(kind+"/"+roundtrip, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				c := NewCreature("Source", "", 1, 1)
				switch kind {
				case "event":
					c.AddAbility(NewTriggered(EvtTapped, false, Sacrifice(This)).SetConditionData(EventSourceIsSelf{}))
				case "activated":
					WithActivatedAbility(Sacrifice(This), ManaCostOf("{0}"))(c)
				}
				g.PutOnBattlefield(c, a.PlayerID())
				switch kind {
				case "event":
					g.FireEvent(GameEvent{Type: EvtTapped, SourceID: c.ID()})
					g.PutTriggersOnStack()
				case "activated":
					if err := g.ActivateAbilityByIndex(a.PlayerID(), c.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
				case "delayed":
					g.RegisterDelayedTrigger(&DelayedTrigger{EventType: EvtTapped, SourceID: c.ID(), Controller: a.PlayerID(), Effects: []Effect{Sacrifice(This)}})
				}
				switch roundtrip {
				case "reentry":
					g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
					g.PutOnBattlefield(c, a.PlayerID())
				case "phasing":
					ids := g.PhaseOut(c.ID())
					g.PhaseIn(ids)
				case "clone":
					g = g.Clone()
				}
				if kind == "delayed" {
					g.FireEvent(GameEvent{Type: EvtTapped, SourceID: c.ID()})
					g.PutTriggersOnStack()
				}
				if g.stack.Size() != 1 {
					t.Fatalf("stack size = %d", g.stack.Size())
				}
				g.ResolveStackObject(g.stack.Pop())
				exists := g.FindPermanent(c.ID()) != nil
				if exists != (roundtrip == "reentry") {
					t.Fatalf("source exists = %v", exists)
				}
			})
		}
	}
}

func TestComposableSequenceSourceIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 1, 1)
	g.PutOnBattlefield(c, a.PlayerID())
	effect := Sequence(FuncEffect("return source", EffectProperties{}, func(g *Game, _, controller uuid.UUID, _ []uuid.UUID) error {
		g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
		g.PutOnBattlefield(c, controller)
		return nil
	}), Sacrifice(This))
	if err := ApplyEffect(g, effect, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) == nil {
		t.Fatal("sequence followed source through zone change")
	}
}

func TestComposableStateReaderSnapshot(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Green", "{G}", 1, 1)
	g.PutOnBattlefield(c, a.PlayerID())
	reader := snapshotState(g)
	p, ok := reader.Permanent(c.ID())
	if !ok || !p.HasColor(Green) {
		t.Fatal("missing snapshot")
	}
	first := reader.Battlefield()
	first[0] = StatePermanent{}
	clone := g.Clone()
	clone.ChangeObjectColor(c.ID(), Blue)
	after, _ := reader.Permanent(c.ID())
	if !after.HasColor(Green) {
		t.Fatal("snapshot changed")
	}
	original, _ := snapshotState(g).Permanent(c.ID())
	if !original.HasColor(Green) {
		t.Fatal("clone changed original")
	}
}

func TestComposableRegistrationBoundary(t *testing.T) {
	r := NewRegistry()
	defer func() {
		p := recover()
		if p == nil || !strings.Contains(fmt.Sprint(p), "Broken: abilities[0].condition") {
			t.Fatalf("panic = %v", p)
		}
	}()
	r.Register("Broken", func() Card { return NewCreature("Broken", "", 1, 1, WithAbility(WhenState(nil, Sacrifice(This)))) })
}

func TestComposableCloneOutstandingState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Watcher", "", 1, 1, WithAbility(WhenState(ControlsNone(IsLand), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(c, a.PlayerID())
	g.CheckStateTriggers()
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.CheckStateTriggers()
	if len(clone.triggers.Pending()) != 0 {
		t.Fatal("clone duplicated outstanding state trigger")
	}
	clone.ResolveStackObject(clone.stack.Pop())
	if g.stack.Size() != 1 || a.Life() != 20 {
		t.Fatal("clone changed original resolution")
	}
	if len(clone.triggers.Pending())+clone.stack.Size() != 1 {
		t.Fatal("clone did not rearm after resolution")
	}
}

func TestComposableConditionTruthTable(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewCreature("Opponent Dwarf", "", 1, 1, WithSubTypes("Dwarf")), b.PlayerID())
	for _, test := range []struct {
		condition StateCondition
		want      bool
	}{
		{Controls(HasSubType("Dwarf")), false}, {ControlsNone(HasSubType("Dwarf")), true},
		{BattlefieldHasNone(IsCreature), false}, {NotCondition(ControlsNone(IsLand)), false},
		{AllConditions(ControlsNone(IsLand), Controls(IsCreature)), false},
		{AnyCondition(Controls(IsLand), Controls(IsCreature)), false},
	} {
		if got := test.condition.checkState(g, StateContext{ControllerID: a.PlayerID()}); got != test.want {
			t.Fatalf("%s = %v", test.condition.inspectDefinition().Text, got)
		}
	}
}

func TestComposableTransientCharacteristics(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Watcher", "{G}", 1, 1, WithAbility(WhenState(Controls(HasColorFilter(Blue)), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(c, a.PlayerID())
	g.ChangeObjectColor(c.ID(), Blue)
	g.ChangeObjectColor(c.ID(), Green)
	g.CheckStateTriggers()
	if len(g.triggers.Pending()) != 1 {
		t.Fatal("missed transient derived characteristic")
	}
}

func TestComposableDefinitionCyclesAndTypedNil(t *testing.T) {
	cyclic := &statePredicate{kind: "not"}
	cyclic.children = []StateCondition{cyclic}
	c := NewCreature("Cycle", "", 1, 1, WithAbility(WhenState(cyclic, Sacrifice(This))))
	if err := ValidateCardDefinition("Cycle", c); err == nil || !strings.Contains(err.Error(), "cyclic") {
		t.Fatalf("error = %v", err)
	}
	var nilEffect *sacrificeEffect
	c = NewCreature("Nil", "", 1, 1, WithAbility(WhenState(ControlsNone(IsLand), nilEffect)))
	if err := ValidateCardDefinition("Nil", c); err == nil || !strings.Contains(err.Error(), "effects[0]") {
		t.Fatalf("error = %v", err)
	}
}
