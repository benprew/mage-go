package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentSetConditionIntervening(t *testing.T) {
	for _, change := range []string{"none", "remove", "subtype", "return", "phase", "copy", "clone", "source-left"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			subject, opponent := BindPermanent("subject"), BindPermanent("opponent")
			walls := IntersectPermanents(BlockersOf(From(opponent)), MatchingPermanents(HasSubType("Wall")))
			others := WithoutPermanents(walls, SingletonPermanent(From(subject)))
			condition := AllConditions(PermanentSetCountAtLeast(others, 1), PermanentSetIsEmpty(IntersectPermanents(BlockersOf(From(opponent)), MatchingPermanents(Not(HasSubType("Wall"))))))
			source := NewCreature("Source", "", 0, 8, WithSubTypes("Wall"), WithAbility(WhenEvent(BlocksCreatureMatching(This, IsCreature).WithSubject(subject).WithOpponent(opponent), GainLifeFor(You, Fixed(1)), WithInterveningIf(condition))))
			if err := ValidateCardDefinition(source.Name(), source); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(source, a.PlayerID())
			q := g.PutOnBattlefield(NewCreature("Attacker", "", 0, 8), b.PlayerID())
			otherCard := NewCreature("Other wall", "", 0, 8, WithSubTypes("Wall"))
			other := g.PutOnBattlefield(otherCard, a.PlayerID())
			g.GetCombat().AddAttackerIncarnation(q.ID(), q.incarnationID, a.PlayerID())
			g.GetCombat().AddBlockerIncarnation(p.ID(), p.incarnationID, q.ID())
			g.GetCombat().AddBlockerIncarnation(other.ID(), other.incarnationID, q.ID())
			g.FireEvent(GameEvent{Type: EvtDeclaredBlocker, SourceID: p.ID(), TargetID: q.ID(), PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("condition did not see event bindings", g.StackSize())
			}
			switch change {
			case "remove":
				g.RemoveFromCombat(other.ID())
			case "subtype":
				g.AddContinuousEffect(SetSubtypes(other.ID(), SubtypeCreature, Indefinite, "Goblin"))
				g.ApplyContinuousEffects()
			case "return":
				g.RemoveFromBattlefield(other)
				g.PutOnBattlefield(otherCard, a.PlayerID())
			case "phase":
				g.PhaseIn(g.PhaseOut(other.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "source-left":
				g.RemoveFromBattlefield(p)
			}
			if change == "clone" {
				clone := g.Clone()
				clone.RemoveFromCombat(other.ID())
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 20 || g.StackSize() != 1 {
					t.Fatal("clone shared frame")
				}
			}
			g.ResolveStack()
			want := 20
			if change == "none" || change == "clone" || change == "source-left" {
				want = 21
			}
			if change == "copy" {
				want = 22
			}
			if a.Life() != want {
				t.Fatalf("life=%d want=%d", a.Life(), want)
			}
		})
	}
}

func TestPermanentSetConditionDeclarations(t *testing.T) {
	other := BindPermanent("other")
	for _, tc := range []struct {
		name      string
		condition StateCondition
		valid     bool
	}{
		{"present", PermanentSetCountAtLeast(MatchingPermanents(IsCreature), 1), true},
		{"zero", PermanentSetCountAtLeast(MatchingPermanents(IsCreature), 0), true},
		{"empty", PermanentSetIsEmpty(MatchingPermanents(IsCreature)), true},
		{"nil", PermanentSetIsEmpty(nil), false},
		{"negative", PermanentSetCountAtLeast(MatchingPermanents(IsCreature), -1), false},
		{"unbound", PermanentSetCountAtLeast(BlockersOf(From(other)), 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewArtifact("Condition", "", WithAbility(WhenEvent(Enters(This), GainLifeFor(You, Fixed(1)), WithInterveningIf(tc.condition))))
			err := ValidateCardDefinition(card.Name(), card)
			if (err == nil) != tc.valid {
				t.Fatal(err)
			}
		})
	}
}

func TestPermanentSetConditionQueryFrame(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 0, 8), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 0, 8), b.PlayerID())
	binding := BindPermanent("saved")
	outer := capturedPermanent{first.ID(), first.incarnationID}
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{binding: outer}
	state := StateContext{SourceID: first.ID(), ControllerID: a.PlayerID(), permanents: map[*PermanentBinding]capturedPermanent{binding: {second.ID(), second.incarnationID}}}
	condition := PermanentSetCountAtLeast(IntersectPermanents(SingletonPermanent(From(binding)), MatchingPermanents(ControlledBy(b.PlayerID()))), 1)
	if !condition.checkState(g, state) || g.resolution.permanentBindings[binding] != outer {
		t.Fatal("query changed or used the live frame")
	}
	delete(state.permanents, binding)
	if condition.checkState(g, state) {
		t.Fatal("unbound query inherited the live frame")
	}
	for _, minimum := range []int{0, 1, 2, 3} {
		if PermanentSetCountAtLeast(MatchingPermanents(IsCreature), minimum).checkState(g, state) != (minimum <= 2) {
			t.Fatal("minimum boundary", minimum)
		}
	}
	if PermanentSetIsEmpty(MatchingPermanents(IsCreature)).checkState(g, state) {
		t.Fatal("nonempty set reported empty")
	}
	g.PhaseOut(second.ID())
	if PermanentSetCountAtLeast(MatchingPermanents(IsCreature), 2).checkState(g, state) {
		t.Fatal("phased member counted")
	}
}

func TestPermanentSetConditionWaitingSource(t *testing.T) {
	for _, change := range []string{"none", "left", "returned", "phased"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Source", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, IfState(PermanentSetCountAtLeast(SingletonPermanent(This), 1), GainLifeFor(You, Fixed(1)))))))
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			switch change {
			case "left", "returned":
				g.RemoveFromBattlefield(p)
				if change == "returned" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			}
			sources := g.WaitingDelayedPermanentSources()
			if len(sources) != 1 || !sources[0].SameObject(saved) || a.Life() != 20 {
				t.Fatal("waiting source or query mutation")
			}
			g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			want := 20
			if change == "none" {
				want = 21
			}
			if a.Life() != want {
				t.Fatal("LKI supplied actionable membership")
			}
		})
	}
}

func TestPermanentSetConditionIndependentEvents(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	binding := BindPermanent("opponent")
	source := NewCreature("Source", "", 0, 8, WithAbility(WhenEvent(BlocksCreatureMatching(This, IsCreature).WithOpponent(binding), GainLifeFor(You, Fixed(1)), WithInterveningIf(PermanentSetCountAtLeast(BlockersOf(From(binding)), 2)))))
	p := g.PutOnBattlefield(source, a.PlayerID())
	others := make([]*Permanent, 0, 2)
	for range 2 {
		attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 0, 8), b.PlayerID())
		other := g.PutOnBattlefield(NewCreature("Other", "", 0, 8), a.PlayerID())
		g.GetCombat().AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, a.PlayerID())
		g.GetCombat().AddBlockerIncarnation(p.ID(), p.incarnationID, attacker.ID())
		g.GetCombat().AddBlockerIncarnation(other.ID(), other.incarnationID, attacker.ID())
		others = append(others, other)
		g.FireEvent(GameEvent{Type: EvtDeclaredBlocker, SourceID: p.ID(), TargetID: attacker.ID(), PlayerID: a.PlayerID()})
	}
	g.RemoveFromCombat(others[0].ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("pending events lost")
	}
	clone := g.Clone()
	clone.RemoveFromCombat(others[1].ID())
	clone.ResolveStack()
	g.ResolveStack()
	if a.Life() != 21 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("events from one source shared saved facts")
	}
}
