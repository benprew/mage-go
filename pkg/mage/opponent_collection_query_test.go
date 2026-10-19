package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestOpponentCollectionQuery(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ownCard := NewArtifact("Owned", "")
	ownCard.SetOwner(a.PlayerID())
	own := g.PutOnBattlefield(ownCard, b.PlayerID())
	foreignCard := NewArtifact("Foreign", "")
	foreignCard.SetOwner(b.PlayerID())
	foreign := g.PutOnBattlefield(foreignCard, a.PlayerID())
	g.PutOnBattlefield(NewLand("Land"), b.PlayerID())
	sets := []PermanentSet{
		PermanentsControlledByPlayers(OpponentsOf(You), IsArtifact),
		PermanentsControlledByPlayers(OpponentsOf(ActivePlayer), IsArtifact),
		IntersectPermanents(MatchingPermanents(IsArtifact), PermanentsControlledByPlayers(OpponentsOf(You), AnyPermanent)),
		UnionPermanents(PermanentsOwnedBy(You, IsArtifact), PermanentsControlledByPlayers(OpponentsOf(You), IsArtifact)),
	}
	for _, set := range sets {
		plan, ok := InspectContextualPermanentCollection(set)
		if !ok {
			t.Fatal("missing opponent declaration")
		}
		for _, controller := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
			for _, active := range []int{0, 1} {
				g.SetActivePlayerIndex(active)
				ctx := &EffectContext{Game: g, Controller: controller}
				var want []PermanentVisit
				for _, p := range set.resolvePermanents(ctx) {
					want = append(want, PermanentVisit{p.ID(), p.incarnationID})
				}
				if got := plan.Select(ctx); !reflect.DeepEqual(got, want) {
					t.Fatalf("got %v, want %v", got, want)
				}
			}
		}
	}
	plan, _ := InspectContextualPermanentCollection(sets[0])
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	before := plan.Select(ctx)
	if len(before) != 1 || before[0].ID != own.ID() {
		t.Fatal("used owner instead of controller")
	}
	clone := g.Clone()
	clone.AddControlEffect(ControlEffectSpec{SourceID: foreign.ID(), TargetID: own.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	if len(plan.Select(&EffectContext{Game: clone, Controller: a.PlayerID()})) != 0 || !reflect.DeepEqual(plan.Select(ctx), before) {
		t.Fatal("clone control leaked")
	}
	g.PhaseOut(own.ID())
	if len(plan.Select(ctx)) != 0 {
		t.Fatal("phased member")
	}
	g.PhaseIn([]uuid.UUID{own.ID()})
	if !reflect.DeepEqual(plan.Select(ctx), before) {
		t.Fatal("phase changed visit")
	}
	g.RemoveFromBattlefield(own)
	if len(plan.Select(ctx)) != 0 {
		t.Fatal("departure supplied LKI")
	}
	returned := g.PutOnBattlefield(ownCard, b.PlayerID())
	after := plan.Select(ctx)
	if len(after) != 1 || after[0].ID != returned.ID() || after[0].Incarnation == before[0].Incarnation {
		t.Fatal("return reused visit")
	}
}

type opponentSetTrap struct{ PlayerSet }

func (opponentSetTrap) resolvePlayers(*EffectContext) []uuid.UUID { panic("player set callback") }

func TestOpponentCollectionQueryNoCallbacks(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Artifact", ""), b.PlayerID())
	filter := IsArtifact
	filter.fn = func(*Permanent, *Game) bool { panic("filter callback") }
	players := OpponentsOf(You).(*opponentsRef)
	set := PermanentsControlledByPlayers(players, filter)
	plan, ok := InspectContextualPermanentCollection(UnionPermanents(set, set))
	if !ok {
		t.Fatal("missing opponent plan")
	}
	players.player = contextualPlayerTrap{}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	random := g.random.Clone()
	for range 3 {
		got := plan.Select(ctx)
		if len(got) != 1 || got[0].ID != p.ID() {
			t.Fatal("declaration changed or duplicates")
		}
		got[0] = PermanentVisit{}
	}
	if !reflect.DeepEqual(random, g.random) || g.StackSize() != 0 || !p.HasType(core.TypeArtifact) {
		t.Fatal("inspection changed state")
	}
	opaque := NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("opaque filter") })
	var nilOpponents *opponentsRef
	for _, unknown := range []PermanentSet{
		PermanentsControlledByPlayers(nil, IsArtifact),
		PermanentsControlledByPlayers(nilOpponents, IsArtifact),
		PermanentsControlledByPlayers(opponentSetTrap{}, IsArtifact),
		PermanentsControlledByPlayers(OpponentsOf(nil), IsArtifact),
		PermanentsControlledByPlayers(OpponentsOf(contextualPlayerTrap{}), IsArtifact),
		PermanentsControlledByPlayers(OpponentsOf(FromPlayer(BindPlayer("saved"))), IsArtifact),
		PermanentsControlledByPlayers(OpponentsOf(You), opaque),
		UnionPermanents(MatchingPermanents(AnyPermanent), PermanentsControlledByPlayers(opponentSetTrap{}, IsArtifact)),
	} {
		if _, ok := InspectContextualPermanentCollection(unknown); ok {
			t.Fatal("partial opponent query")
		}
	}
	if _, ok := InspectPermanentCollection(set); ok {
		t.Fatal("context-free boundary changed")
	}
	if plan.Select(nil) != nil || plan.Select(&EffectContext{Game: g, Controller: uuid.New()}) != nil {
		t.Fatal("invalid context selected members")
	}
}
