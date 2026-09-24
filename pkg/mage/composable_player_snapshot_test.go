package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestPlayerSnapshotLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), b.PlayerID())
	saved := BindPlayer("controller")
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}
	if err := SnapshotPlayer(ControllerOf(This), saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(source)
	g.PutOnBattlefield(source.Card, a.PlayerID())
	if err := PreventNextDamage(FromPlayer(saved), Fixed(3)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.DealDamageToPlayer(clone.GetPlayer(b.PlayerID()), 5, uuid.Nil)
	if clone.GetPlayer(b.PlayerID()).Life() != 18 || b.Life() != 20 {
		t.Fatal("captured player or clone shield changed")
	}
	g.DealDamageToPlayer(a, 1, uuid.Nil)
	g.DealDamageToPlayer(b, 2, uuid.Nil)
	g.DealDamageToPlayer(b, 3, uuid.Nil)
	if a.Life() != 19 || b.Life() != 18 {
		t.Fatal("shield did not retain the captured controller")
	}
	if err := SnapshotPlayer(FromPlayer(saved), saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if id, ok := FromPlayer(saved).resolvePlayer(ctx); !ok || id != b.PlayerID() {
		t.Fatal("self capture lost its input")
	}
	cloneCtx := &EffectContext{Game: clone, Controller: a.PlayerID()}
	if err := SnapshotPlayer(ControllerOf(This), saved).Apply(cloneCtx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromPlayer(saved).resolvePlayer(cloneCtx); ok {
		t.Fatal("absent subject retained the old player")
	}
	if id, ok := FromPlayer(saved).resolvePlayer(ctx); !ok || id != b.PlayerID() {
		t.Fatal("clone changed the original binding")
	}
}

func TestPlayerSnapshotDelayedCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindPlayer("player")
	target := DeclarePlayerTarget("target")
	body := Sequence(SnapshotPlayer(target, saved), AtNext(EachEndStep, GainLifeFor(FromPlayer(saved), Fixed(3))), SnapshotPlayer(You, saved))
	ability := Spell(body, WithTargets(target))
	card := NewInstant("Snapshot", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.pushStack(&StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	g.ResolveStack()
	if _, ok := FromPlayer(saved).resolvePlayer(&EffectContext{Game: g}); ok {
		t.Fatal("binding escaped resolution")
	}
	clone := g.Clone()
	for _, game := range []*Game{clone, g} {
		game.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 23 {
			t.Fatal("delayed trigger lost its snapshot")
		}
		if game == clone && b.Life() != 20 {
			t.Fatal("clone changed the original player")
		}
	}
}

func TestPlayerSnapshotValidation(t *testing.T) {
	saved := BindPlayer("saved")
	for _, tt := range []struct {
		name string
		body ResolvingEffect
		want string
	}{
		{"valid", Sequence(SnapshotPlayer(You, saved), GainLifeFor(FromPlayer(saved), Fixed(1))), ""},
		{"nil subject", SnapshotPlayer(nil, saved), "non-nil"},
		{"nil binding", SnapshotPlayer(You, nil), "non-nil"},
		{"empty name", SnapshotPlayer(You, BindPlayer("")), "name"},
		{"read before write", SnapshotPlayer(FromPlayer(saved), saved), "producer"},
		{"undeclared target", SnapshotPlayer(DeclarePlayerTarget("target"), saved), "target"},
		{"optional producer", Sequence(IfState(Controls(IsCreature), SnapshotPlayer(You, saved)), GainLifeFor(FromPlayer(saved), Fixed(1))), "producer"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			card := NewInstant("Snapshot", "", WithAbility(Spell(tt.body)))
			err := ValidateCardDefinition(card.Name(), card)
			if tt.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %s", err, tt.want)
			}
		})
	}
}

func TestPlayerSnapshotSpellCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindPlayer("controller")
	ability := Spell(Sequence(SnapshotPlayer(You, saved), GainLifeFor(FromPlayer(saved), Fixed(2))))
	card := NewInstant("Snapshot", "", WithAbility(ability))
	original := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Effects: ability.Effects()}
	g.pushStack(original)
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	g.ResolveStackObject(copied)
	if a.Life() != 20 || b.Life() != 22 {
		t.Fatal("copy captured the wrong controller")
	}
	g.ResolveStackObject(original)
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("original reused the copy binding")
	}
}

func TestPlayerSnapshotFrameIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewCreature("Member", "", 1, 1), b.PlayerID())
	saved := BindPlayer("player")
	member := BindPermanent("member")
	body := Sequence(SnapshotPlayer(You, saved),
		ForEach(MatchingPermanents(IsCreature), member, Sequence(SnapshotPlayer(ControllerOf(From(member)), saved), GainLifeFor(FromPlayer(saved), Fixed(1)))),
		FuncEffect("nested resolution", EffectProperties{}, func(g *Game, source, controller uuid.UUID, _ []uuid.UUID) error {
			return ApplyEffect(g, Sequence(SnapshotPlayer(You, saved), GainLifeFor(FromPlayer(saved), Fixed(2))), source, b.PlayerID(), nil)
		}), GainLifeFor(FromPlayer(saved), Fixed(3)))
	g.SetRandomResults([]int{1})
	for range 2 {
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if a.Life() != 26 || b.Life() != 26 {
		t.Fatalf("frame leaked: life = %d/%d", a.Life(), b.Life())
	}
	if g.RandIntn(2) != 1 {
		t.Fatal("snapshot consumed randomness")
	}
}
