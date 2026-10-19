package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttachmentCollectionQuery(t *testing.T) {
	for _, state := range []string{"normal", "move", "aura-phase", "host-phase", "cycle", "host-return", "aura-return", "control", "stale", "missing-host"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			host := g.PutOnBattlefield(NewCreature("Host", "", 2, 4), a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 2, 4), b.PlayerID())
			aura := g.PutOnBattlefield(NewAura("Aura", ""), b.PlayerID())
			g.Attach(aura.ID(), host.ID())
			equipment := g.PutOnBattlefield(NewArtifact("Equipment", "", WithSubTypes("Equipment")), b.PlayerID())
			g.Attach(equipment.ID(), host.ID())
			outer := g.PutOnBattlefield(NewAura("Outer", "", WithCastTarget(TargetPermanent())), a.PlayerID())
			g.Attach(outer.ID(), aura.ID())
			g.combat.AddAttackerIncarnation(host.ID(), host.incarnationID, b.PlayerID())
			hosts := PermanentsControlledBy(You, IsCreature)
			sets := []PermanentSet{
				AttachmentsOn(hosts),
				IntersectPermanents(AttachmentsOn(hosts), MatchingPermanents(HasSubType("Aura"))),
				AttachmentsOn(AttachmentsOn(hosts)),
				UnionPermanents(AttachmentsOn(hosts), AttachmentsOn(hosts)),
				AttachmentsOn(PermanentsControlledByPlayers(OpponentsOf(You), IsCreature)),
				AttachmentsOn(MatchingPermanents(IsAttacking)),
			}
			plans := make([]ContextualPermanentCollectionPredicate, len(sets))
			for i, set := range sets {
				var ok bool
				plans[i], ok = InspectContextualPermanentCollection(set)
				if !ok {
					t.Fatalf("missing attachment declaration %d", i)
				}
				if _, ok := InspectPermanentCollection(set); ok {
					t.Fatal("context-free boundary changed")
				}
			}
			switch state {
			case "move":
				g.Attach(aura.ID(), other.ID())
			case "aura-phase":
				g.PhaseOut(aura.ID())
			case "host-phase":
				g.PhaseOut(host.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "host-return":
				g.RemoveFromBattlefield(host)
				g.PutOnBattlefield(host.Card, a.PlayerID())
			case "aura-return":
				g.RemoveFromBattlefield(aura)
				g.PutOnBattlefield(aura.Card, b.PlayerID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "stale":
				g.MutablePermanent(aura.ID()).attachedIncarnation = uuid.New()
			case "missing-host":
				g.MutablePermanent(aura.ID()).AttachedTo = uuid.New()
			}
			for _, game := range []*Game{g, g.Clone()} {
				for _, controller := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
					ctx := &EffectContext{Game: game, Controller: controller}
					for i, set := range sets {
						var want []PermanentVisit
						for _, p := range set.resolvePermanents(ctx) {
							want = append(want, PermanentVisit{p.ID(), p.incarnationID})
						}
						if got := plans[i].Select(ctx); !reflect.DeepEqual(got, want) {
							t.Fatalf("set %d: got %v want %v", i, got, want)
						}
					}
				}
			}
			if state == "normal" {
				ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
				if got := plans[0].Select(ctx); len(got) != 2 || got[0].ID != aura.ID() || got[1].ID != equipment.ID() {
					t.Fatal("wrong attachment visits")
				}
				if got := plans[2].Select(ctx); len(got) != 1 || got[0].ID != outer.ID() {
					t.Fatal("wrong nested host")
				}
				clone := g.Clone()
				clone.Attach(aura.ID(), other.ID())
				if len(plans[0].Select(&EffectContext{Game: clone, Controller: a.PlayerID()})) != 1 || len(plans[0].Select(ctx)) != 2 {
					t.Fatal("clone changed original links")
				}
			}
		})
	}
}

func TestAttachmentCollectionQueryNoCallbacks(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewLand("Host"), a.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Aura", "", WithCastTarget(TargetLand())), b.PlayerID())
	g.Attach(aura.ID(), host.ID())
	filter := IsLand
	filter.fn = func(*Permanent, *Game) bool { panic("filter callback") }
	members := []PermanentSet{PermanentsControlledBy(You, filter), PermanentsOwnedBy(ActivePlayer, filter)}
	set := AttachmentsOn(UnionPermanents(members...))
	plan, ok := InspectContextualPermanentCollection(set)
	if !ok {
		t.Fatal("missing attachment query")
	}
	members[0] = collectionQueryTrap{}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	random := g.random.Clone()
	for range 3 {
		got := plan.Select(ctx)
		if len(got) != 1 || got[0].ID != aura.ID() {
			t.Fatal("lost attachment or duplicated member")
		}
		got[0] = PermanentVisit{}
	}
	if !reflect.DeepEqual(random, g.random) || g.StackSize() != 0 || g.FindPermanent(aura.ID()).AttachedTo != host.ID() {
		t.Fatal("query changed game")
	}
	for _, invalid := range []PermanentSet{
		nil, collectionQueryTrap{}, SingletonPermanent(This), SingletonPermanent(From(BindPermanent("saved"))),
		UnionPermanents(), PermanentsControlledBy(contextualPlayerTrap{}, IsLand),
		MatchingPermanents(Or(AnyPermanent, NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("opaque callback") }))),
		UnionPermanents(MatchingPermanents(AnyPermanent), collectionQueryTrap{}),
	} {
		for _, wrapped := range []PermanentSet{AttachmentsOn(invalid), AttachmentsOn(AttachmentsOn(invalid)), UnionPermanents(MatchingPermanents(AnyPermanent), AttachmentsOn(invalid))} {
			if _, ok := InspectContextualPermanentCollection(wrapped); ok {
				t.Fatal("unknown host supplied partial query")
			}
		}
	}
	if plan.Select(nil) != nil || plan.Select(&EffectContext{}) != nil || plan.Select(&EffectContext{Game: g}) != nil {
		t.Fatal("invalid context supplied attachments")
	}
}
