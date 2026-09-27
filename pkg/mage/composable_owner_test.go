package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestOwnerOfPermanentIdentityAndSnapshot(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Owned", "", 2, 2)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, b.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: b.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	owner := OwnerOf(This)
	if id, ok := owner.resolvePlayer(ctx); !ok || id != a.PlayerID() {
		t.Fatal("owner followed controller")
	}
	saved := BindPlayer("owner")
	if err := SnapshotPlayer(owner, saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(p.ID())
	if _, ok := owner.resolvePlayer(ctx); ok {
		t.Fatal("phased permanent supplied owner")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	if id, ok := owner.resolvePlayer(ctx); !ok || id != a.PlayerID() {
		t.Fatal("phase-in lost owner")
	}
	clone := g.Clone()
	cloneCtx := &EffectContext{Game: clone, SourceID: p.ID(), Controller: b.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
	clone.RemoveFromBattlefield(clone.FindPermanent(p.ID()))
	clone.PutOnBattlefield(card, b.PlayerID())
	if _, ok := owner.resolvePlayer(cloneCtx); ok {
		t.Fatal("returned source reused old incarnation")
	}
	if id, ok := FromPlayer(saved).resolvePlayer(ctx); !ok || id != a.PlayerID() {
		t.Fatal("snapshot changed with clone")
	}
	if err := SnapshotPlayer(owner, saved).Apply(cloneCtx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromPlayer(saved).resolvePlayer(cloneCtx); ok {
		t.Fatal("absent owner retained snapshot in clone")
	}
	if id, ok := FromPlayer(saved).resolvePlayer(ctx); !ok || id != a.PlayerID() {
		t.Fatal("clone changed original snapshot")
	}
}

func TestOwnerOfTargetAndRegistration(t *testing.T) {
	target := DeclarePermanentTarget("permanent", AnyPermanent)
	saved := BindPlayer("owner")
	ability := Spell(
		Sequence(SnapshotPlayer(OwnerOf(target), saved), GainLifeFor(FromPlayer(saved), Fixed(1))),
		WithTargets(target))
	valid := NewInstant("Owner", "", WithAbility(ability))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		subject PermanentRef
		want    string
	}{
		{"nil", nil, "non-nil"},
		{"undeclared", target, "target"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewInstant("Bad", "", WithAbility(Spell(GainLifeFor(OwnerOf(tc.subject), Fixed(1)))))
			if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validation = %v, want %s", err, tc.want)
			}
		})
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Target", "", 1, 1)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: valid.ID(), Card: valid, Controller: b.PlayerID(), Targets: []uuid.UUID{p.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	phased := g.Clone()
	phased.PhaseOut(p.ID())
	phased.ResolveStack()
	if phased.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("phased target supplied owner")
	}
	returned := g.Clone()
	returned.RemoveFromBattlefield(returned.FindPermanent(p.ID()))
	returned.PutOnBattlefield(card, b.PlayerID())
	returned.ResolveStack()
	if returned.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("returned target supplied owner")
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("target owner did not gain life")
	}
}
