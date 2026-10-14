package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombatPairSubjectDeclarations(t *testing.T) {
	subject, opponent := BindPermanent("subject"), BindPermanent("opponent")
	for _, build := range []func(PermanentRef, PermanentFilter) filteredCombatPairPattern{
		BlocksCreatureMatching, BecomesBlockedByCreatureMatching, BlocksOrBecomesBlockedByCreatureMatching,
	} {
		RunDefinitionValidationTable(t, []DefinitionValidationCase{
			{Name: "subject only", Card: NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(build(This, IsCreature).WithSubject(subject), TapSubject(From(subject)))))},
			{Name: "both", Card: NewAura("Valid", "", WithAbility(WhenEvent(build(AttachedPermanent, IsCreature).WithSubject(subject).WithOpponent(opponent), Sequence(TapSubject(From(subject)), TapSubject(From(opponent))))))},
			{Name: "nil", Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(build(This, IsCreature).WithSubject(nil), GainLifeFor(You, Fixed(1))))), WantErrSubstr: "subject binding"},
			{Name: "empty", Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(build(This, IsCreature).WithSubject(BindPermanent("")), GainLifeFor(You, Fixed(1))))), WantErrSubstr: "binding"},
			{Name: "duplicate", Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(build(This, IsCreature).WithSubject(subject).WithSubject(subject), GainLifeFor(You, Fixed(1))))), WantErrSubstr: "one subject"},
			{Name: "alias collision", Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(build(This, IsCreature).WithSubject(subject).WithOpponent(subject), TapSubject(From(subject))))), WantErrSubstr: "separate"},
			{Name: "unbound", Card: NewCreature("Bad", "", 1, 1, WithAbility(WhenEvent(build(This, IsCreature).WithOpponent(opponent), TapSubject(From(subject))))), WantErrSubstr: "binding"},
		})
	}
}

func TestCombatPairSubjectCapture(t *testing.T) {
	for _, attached := range []bool{false, true} {
		for _, kind := range []EventType{EvtDeclaredBlocker, EvtCreatureBlockedBy, EvtCombatPair} {
			for _, change := range []string{"none", "reattach", "source-leaves", "return", "phase", "phase-cycle", "copy", "control"} {
				t.Run(boolLabel(attached)+"/"+kind.String()+"/"+change, func(t *testing.T) {
					a, b := NewBasePlayer("A"), NewBasePlayer("B")
					g := NewGame(a, b)
					saved, other := BindPermanent("subject"), BindPermanent("opponent")
					var ref PermanentRef = This
					if attached {
						ref = AttachedPermanent
					}
					pattern := BlocksOrBecomesBlockedByCreatureMatching(ref, IsCreature)
					if kind == EvtDeclaredBlocker {
						pattern = BlocksCreatureMatching(ref, IsCreature)
					}
					if kind == EvtCreatureBlockedBy {
						pattern = BecomesBlockedByCreatureMatching(ref, IsCreature)
					}
					body := AtNext(EachEndStep, Sequence(AddCountersTo(From(saved), P1P1, Fixed(1)), TapSubject(From(other)), GainLifeFor(You, Fixed(1))))
					card := NewCreature("Source", "", 0, 8, WithAbility(WhenEvent(pattern.WithSubject(saved).WithOpponent(other), body)))
					if attached {
						card = NewAura("Source", "", WithAbility(WhenEvent(pattern.WithSubject(saved).WithOpponent(other), body)))
					}
					if err := ValidateCardDefinition(card.Name(), card); err != nil {
						t.Fatal(err)
					}
					p := g.PutOnBattlefield(card, a.PlayerID())
					hostCard := card
					host := p
					if attached {
						hostCard = NewCreature("Host", "", 0, 8)
						host = g.PutOnBattlefield(hostCard, b.PlayerID())
						g.Attach(p.ID(), host.ID())
					}
					q := g.PutOnBattlefield(NewCreature("Opponent", "", 0, 8), b.PlayerID())
					g.FireEvent(GameEvent{Type: kind, SourceID: host.ID(), TargetID: q.ID()})
					if change == "reattach" && attached {
						g.Attach(p.ID(), q.ID())
					}
					if change == "source-leaves" && attached {
						g.RemoveFromBattlefield(p)
					}
					g.PutTriggersOnStack()
					if g.StackSize() != 1 {
						t.Fatalf("stack=%d", g.StackSize())
					}
					if change == "copy" {
						g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
					}
					if change == "return" || change == "source-leaves" && !attached {
						g.RemoveFromBattlefield(host)
						g.PutOnBattlefield(hostCard, b.PlayerID())
					}
					if change == "phase" {
						g.PhaseOut(host.ID())
					}
					if change == "phase-cycle" {
						g.PhaseIn(g.PhaseOut(host.ID()))
					}
					if change == "control" {
						g.AddControlEffect(ControlEffectSpec{SourceID: p.ID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
					}
					clone := g.Clone()
					for _, game := range []*Game{clone, g} {
						game.ResolveStack()
						game.RunStepWithPriority(EndStep)
						if change == "phase" {
							game.PhaseIn([]uuid.UUID{host.ID()})
						}
						want := 1
						if change == "copy" {
							want = 2
						}
						if change == "return" || change == "phase" || change == "source-leaves" && !attached {
							want = 0
						}
						if game.FindPermanent(host.ID()).Counters[P1P1] != want {
							t.Fatalf("counters=%d want=%d", game.FindPermanent(host.ID()).Counters[P1P1], want)
						}
						if !game.FindPermanent(q.ID()).Tapped {
							t.Fatal("opponent capture lost")
						}
						if game == clone && g.StackSize() == 0 {
							t.Fatal("clone changed original stack")
						}
					}
				})
			}
		}
	}
}

func TestCombatPairSubjectIndependentFacts(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	subject, opponent := BindPermanent("subject"), BindPermanent("opponent")
	pattern := BlocksOrBecomesBlockedByCreatureMatching(AttachedPermanent, IsCreature).WithSubject(subject).WithOpponent(opponent)
	body := AtNext(EachEndStep, Sequence(AddCountersTo(From(subject), P1P1, Fixed(1)), GainLifeFor(You, PowerOfWithLKI(From(subject))), TapSubject(From(opponent))))
	aura := g.PutOnBattlefield(NewAura("Source", "", WithAbility(WhenEvent(pattern, body))), a.PlayerID())
	firstCard := NewCreature("First", "", 2, 8)
	first := g.PutOnBattlefield(firstCard, b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 4, 8), b.PlayerID())
	victims := []*Permanent{g.PutOnBattlefield(NewCreature("One", "", 0, 8), b.PlayerID()), g.PutOnBattlefield(NewCreature("Two", "", 0, 8), b.PlayerID())}
	for i, host := range []*Permanent{first, second} {
		g.Attach(aura.ID(), host.ID())
		g.FireEvent(GameEvent{Type: EvtCombatPair, SourceID: host.ID(), TargetID: victims[i].ID()})
	}
	g.RemoveFromBattlefield(aura)
	g.RemoveFromBattlefield(first)
	g.PutOnBattlefield(firstCard, b.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack=%d", g.StackSize())
	}
	g.ResolveStack()
	clone := g.Clone()
	clone.RunStepWithPriority(EndStep)
	if a.Life() != 20 || g.FindPermanent(second.ID()).Counters[P1P1] != 0 {
		t.Fatal("clone changed original facts")
	}
	g.RunStepWithPriority(EndStep)
	for _, game := range []*Game{clone, g} {
		if game.FindPermanent(first.ID()).Counters[P1P1] != 0 || game.FindPermanent(second.ID()).Counters[P1P1] != 1 {
			t.Fatal("pending subjects overwrote one another")
		}
		if game.GetPlayer(a.PlayerID()).Life() != 27 || game.GetPlayer(b.PlayerID()).Life() != 20 {
			t.Fatal("saved departure power or controller lost")
		}
		for _, victim := range victims {
			if !game.FindPermanent(victim.ID()).Tapped {
				t.Fatal("pending opponent lost")
			}
		}
	}
}
