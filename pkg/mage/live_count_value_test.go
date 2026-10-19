package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLiveCountValue(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	filter := IsLand
	filter.fn = func(*Permanent, *Game) bool { panic("filter callback") }
	count := CountPermanents(PermanentsControlledBy(You, filter))
	for _, tc := range []struct {
		value ResolutionValue
		want  []int
	}{
		{count, []int{0, 1, 2, 3, 4, 5, 6}},
		{DivideRoundUp(count, Fixed(2)), []int{0, 1, 1, 2, 2, 3, 3}},
		{DivideRoundDown(count, Fixed(2)), []int{0, 0, 1, 1, 2, 2, 3}},
		{DivideRoundUp(DivideRoundDown(count, Fixed(2)), Fixed(2)), []int{0, 0, 1, 1, 1, 1, 2}},
	} {
		plan, ok := InspectLivePermanentCount(tc.value)
		if !ok {
			t.Fatal("native count tree is unknown")
		}
		lands := []*Permanent{}
		for _, want := range tc.want[:6] {
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			if got, known := plan.Read(ctx); !known || got != want {
				t.Fatalf("count=%d known=%v want=%d", got, known, want)
			}
			lands = append(lands, g.PutOnBattlefield(NewLand("Land"), a.PlayerID()))
		}
		clone := g.Clone()
		clone.PhaseOut(lands[0].ID())
		before := g.random.Clone()
		ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
		old, _ := plan.Read(ctx)
		if got, known := plan.Read(&EffectContext{Game: clone, Controller: a.PlayerID()}); !known || got != tc.want[5] {
			t.Fatal("clone count includes phased land")
		}
		if got, _ := plan.Read(ctx); got != old || !reflect.DeepEqual(before, g.random) {
			t.Fatal("query changed state")
		}
		for _, land := range lands {
			g.RemoveFromBattlefield(land)
		}
	}
	for _, bad := range []ResolutionValue{nil, Fixed(2), damageInspectionValueTrap{}, CountPermanents(collectionQueryTrap{}), DivideRoundUp(count, Fixed(3)), DivideRoundUp(count, damageInspectionValueTrap{}), AddValues(count, Fixed(1)), DivideRoundUp(nil, Fixed(2))} {
		if _, known := InspectLivePermanentCount(bad); known {
			t.Fatal("unsupported tree accepted")
		}
	}
	var zero LivePermanentCountPredicate
	if _, known := zero.Read(&EffectContext{Game: g, Controller: a.PlayerID()}); known {
		t.Fatal("zero predicate known")
	}
	plan, _ := InspectLivePermanentCount(count)
	for _, ctx := range []*EffectContext{nil, {}, {Game: g, Controller: uuid.New()}} {
		if _, known := plan.Read(ctx); known {
			t.Fatal("missing context accepted")
		}
	}
}

func TestLiveCountValueSpell(t *testing.T) {
	for _, state := range []string{"only", "prefix", "suffix", "zero", "unknown", "missing binding", "illegal missing binding", "bound player", "illegal bound player", "unknown suffix", "unknown all illegal", "foreign source"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			count := CountPermanents(PermanentsControlledBy(You, IsLand))
			player := DeclarePlayerTarget("count player")
			amount := DivideRoundUp(count, Fixed(2))
			known, want := true, false
			if state == "missing binding" || state == "illegal missing binding" || state == "bound player" || state == "illegal bound player" {
				amount = CountPermanents(PermanentsControlledBy(player, IsLand))
				if state == "bound player" || state == "illegal bound player" {
					obj.TargetSpecs = append(obj.TargetSpecs, player.targetSpec())
					obj.Targets = append(obj.Targets, obj.Controller)
					if state == "illegal bound player" {
						obj.Targets[len(obj.Targets)-1] = uuid.New()
					}
				} else {
					known = false
				}
			}
			if state == "illegal missing binding" {
				p.GrantBaseAttr(core.Shroud)
			}
			if state == "unknown" || state == "unknown suffix" || state == "unknown all illegal" {
				amount = DivideRoundUp(count, damageInspectionValueTrap{})
				known = false
			}
			if state == "zero" {
				amount = CountPermanents(MatchingPermanents(IsArtifact))
			}
			source := DamageSourceRef(This)
			if state == "foreign source" {
				source = DamageSourceOf(This)
				known = false
			}
			damage := DealDamagePackets(source, DamagePacketTo(You, amount))
			if state == "unknown all illegal" {
				p.GrantBaseAttr(core.Shroud)
			}
			obj.Effects = []Effect{damage}
			if state == "prefix" || state == "zero" {
				obj.Effects = []Effect{Sequence(damage, Destroy(target))}
				known = false
			}
			if state == "suffix" || state == "unknown suffix" {
				obj.Effects = []Effect{Sequence(Destroy(target), damage)}
				want = state == "suffix"
			}
			for range 2 {
				certain, gotKnown := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}})
				if certain != want || gotKnown != known {
					t.Fatalf("got=%v/%v want=%v/%v", certain, gotKnown, want, known)
				}
			}
		})
	}
}
