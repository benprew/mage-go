package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTapStateChangeUntappedStun(t *testing.T) {
	g, p, _, _ := targetQueryFixture(t)
	p.AddCounter(core.Stun, 2)
	if g.UntapPermanent(p) || p.Tapped || p.Counters[core.Stun] != 2 {
		t.Fatal("untapping an untapped permanent removed a stun counter")
	}
}

func TestTapStateChangeRead(t *testing.T) {
	for _, operation := range []TapOperationKind{TapOperation, UntapOperation, MayTapOrUntapOperation} {
		for _, tapped := range []bool{false, true} {
			g, p, _, _ := targetQueryFixture(t)
			p.Tapped = tapped
			p.AddCounter(core.Stun, 2)
			visit := PermanentVisit{p.ID(), p.incarnationID}
			before := g.Clone()
			want := operation == MayTapOrUntapOperation || (operation == TapOperation && !tapped) || (operation == UntapOperation && tapped)
			for range 2 {
				mayChange, known := g.InspectTapStateChange(operation, []PermanentVisit{visit})
				if mayChange != want || !known {
					t.Fatalf("operation=%d tapped=%v change=%v/%v want=%v", operation, tapped, mayChange, known, want)
				}
			}
			if p.Tapped != tapped || p.Counters[core.Stun] != 2 || !reflect.DeepEqual(g.random, before.random) || !reflect.DeepEqual(g.resolution, before.resolution) {
				t.Fatal("inspection changed state")
			}
			clone := g.Clone()
			clone.MutablePermanent(p.ID()).Tapped = !tapped
			if changed, known := clone.InspectTapStateChange(operation, []PermanentVisit{visit}); !known || (operation != MayTapOrUntapOperation && changed == want) {
				t.Fatal("inspection did not read clone state")
			}
		}
	}
}

func TestTapStateChangeInvalid(t *testing.T) {
	g, p, _, _ := targetQueryFixture(t)
	good := PermanentVisit{p.ID(), p.incarnationID}
	for _, operation := range []TapOperationKind{0, 99} {
		if _, known := g.InspectTapStateChange(operation, nil); known {
			t.Fatal("invalid operation accepted")
		}
	}
	if _, known := (*Game)(nil).InspectTapStateChange(TapOperation, nil); known {
		t.Fatal("nil game accepted")
	}
	for _, bad := range []PermanentVisit{{}, {p.ID(), uuid.New()}, {uuid.New(), uuid.New()}} {
		if _, known := g.InspectTapStateChange(TapOperation, []PermanentVisit{good, bad}); known {
			t.Fatal("invalid suffix hidden by transition")
		}
	}
	g.PhaseOut(p.ID())
	if _, known := g.InspectTapStateChange(TapOperation, []PermanentVisit{good}); known {
		t.Fatal("phased visit accepted")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	if changed, known := g.InspectTapStateChange(TapOperation, []PermanentVisit{good, good}); !changed || !known {
		t.Fatal("phase cycle or repeated visit rejected")
	}
	g.BouncePermanentToHand(p)
	g.PutOnBattlefield(p.Card, p.ControllerID())
	if _, known := g.InspectTapStateChange(TapOperation, []PermanentVisit{good}); known {
		t.Fatal("old visit followed returned object")
	}
	for _, operation := range []TapOperationKind{TapOperation, UntapOperation, MayTapOrUntapOperation} {
		if changed, known := g.InspectTapStateChange(operation, nil); changed || !known {
			t.Fatal("empty operation changes state")
		}
	}
}

func TestTapStateChangeSpell(t *testing.T) {
	for _, operation := range []TapOperationKind{TapOperation, UntapOperation, MayTapOrUntapOperation} {
		for _, group := range []bool{false, true} {
			if group && operation == MayTapOrUntapOperation {
				continue
			}
			for _, changes := range []bool{false, true} {
				g, p, obj, binding := targetQueryFixture(t)
				p.Tapped = (operation == TapOperation) != changes
				p.AddCounter(core.Stun, 2)
				instruction := TapInstruction{Subject: binding, Operation: operation}
				if group {
					instruction.Subject = nil
					instruction.Subjects = MatchingPermanents(IsLand)
				}
				obj.Effects = []Effect{Sequence(instruction.Effect(), Destroy(binding))}
				visits := []PermanentVisit{{p.ID(), p.incarnationID}}
				want := !changes && operation != MayTapOrUntapOperation
				for _, game := range []*Game{g, g.Clone()} {
					if certain, known := game.InspectSpellDestruction(game.StackPeek(), visits); certain != want || known != want {
						t.Fatalf("operation=%d group=%v changes=%v certainty=%v/%v want=%v", operation, group, changes, certain, known, want)
					}
				}
				g.ResolveStack()
				if g.FindPermanent(p.ID()) != nil {
					t.Fatal("native execution failed to destroy")
				}
				if operation == UntapOperation {
					wantStun := 2
					if changes {
						wantStun--
					}
					lki := g.zones.LKI(p.ID())
					if lki == nil || lki.Snapshot == nil || lki.Snapshot.Counters[core.Stun] != wantStun {
						t.Fatal("stun execution differs from state proof")
					}
				}
			}
		}
	}
}

func TestTapStateChangeGroupBarrier(t *testing.T) {
	for _, operation := range []TapOperationKind{TapOperation, UntapOperation} {
		for _, mixed := range []bool{false, true} {
			g, p, obj, _ := targetQueryFixture(t)
			other := g.PutOnBattlefield(NewLand("Other"), obj.Controller)
			p.Tapped = operation == TapOperation
			other.Tapped = p.Tapped != mixed
			p.AddCounter(core.Stun, 2)
			other.AddCounter(core.Stun, 2)
			obj.Effects = []Effect{Sequence((TapInstruction{Subjects: MatchingPermanents(IsLand), Operation: operation}).Effect(), DestroyEach(MatchingPermanents(IsLand)))}
			watched := []PermanentVisit{{p.ID(), p.incarnationID}}
			if certain, known := g.InspectSpellDestruction(obj, watched); certain != !mixed || known != !mixed {
				t.Fatal("group ignored one possible change")
			}
		}
	}
}

func TestTapStateChangeNoCallbacks(t *testing.T) {
	g, p, obj, binding := targetQueryFixture(t)
	p.Tapped = true
	g.effects.AddReplacement(&proposalTrap{})
	visit := PermanentVisit{p.ID(), p.incarnationID}
	if changed, known := g.InspectTapStateChange(TapOperation, []PermanentVisit{visit}); changed || !known {
		t.Fatal("no-op depended on replacement callback")
	}
	obj.Effects = []Effect{Sequence((TapInstruction{Subject: binding, Operation: TapOperation}).Effect(), Destroy(binding))}
	if certain, known := g.InspectSpellDestruction(obj, []PermanentVisit{visit}); certain || known {
		t.Fatal("unknown destruction replacement supplied proof")
	}
	obj.Effects = []Effect{Sequence((TapInstruction{Subject: binding, Operation: TapOperation}).Effect(), destructionInspectionTrap{})}
	if certain, known := g.InspectSpellDestruction(obj, []PermanentVisit{visit}); certain || known {
		t.Fatal("no-op hid unknown suffix")
	}
}
