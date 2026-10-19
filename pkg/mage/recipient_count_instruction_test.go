package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRecipientCountInstructionRead(t *testing.T) {
	g, a, b := newSpellCopyGame()
	member := BindPlayer("recipient")
	filter := IsLand
	filter.fn = func(*Permanent, *Game) bool { panic("filter callback") }
	count := CountPermanents(IntersectPermanents(PermanentsControlledBy(FromPlayer(member), filter), PermanentsControlledBy(You, IsLand)))
	predicate, ok := InspectRecipientPermanentCount(count, member)
	if !ok {
		t.Fatal("local count rejected")
	}
	p := g.PutOnBattlefield(NewLand("land"), a.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	outer := map[*PlayerBinding]uuid.UUID{member: b.PlayerID()}
	g.resolution.playerBindings = outer
	before := g.random.Clone()
	for range 2 {
		if got, known := predicate.Read(ctx, a.PlayerID()); !known || got != 1 {
			t.Fatalf("count=%d/%v", got, known)
		}
		if got, known := predicate.Read(ctx, b.PlayerID()); !known || got != 0 {
			t.Fatal("local player replaced You")
		}
	}
	clone := g.Clone()
	clone.PhaseOut(p.ID())
	if got, known := predicate.Read(&EffectContext{Game: clone, Controller: a.PlayerID()}, a.PlayerID()); !known || got != 0 {
		t.Fatal("phased count")
	}
	g.PhaseIn(g.PhaseOut(p.ID()))
	if got, _ := predicate.Read(ctx, a.PlayerID()); got != 1 {
		t.Fatal("phase cycle")
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if got, _ := predicate.Read(ctx, a.PlayerID()); got != 0 {
		t.Fatal("stale control")
	}
	if g.resolution.playerBindings[member] != b.PlayerID() || !reflect.DeepEqual(before, g.random) {
		t.Fatal("query changed frame or RNG")
	}
	for _, bad := range []ResolutionValue{nil, Fixed(1), damageInspectionValueTrap{}, CountPermanents(collectionQueryTrap{}), CountPermanents(PermanentsControlledBy(FromPlayer(BindPlayer("recipient")), IsLand)), AddValues(count, Fixed(1))} {
		if _, known := InspectRecipientPermanentCount(bad, member); known {
			t.Fatal("unknown count accepted")
		}
	}
	for _, binding := range []*PlayerBinding{nil, BindPlayer("")} {
		if _, known := InspectRecipientPermanentCount(count, binding); known {
			t.Fatal("invalid member")
		}
	}
	for _, context := range []*EffectContext{nil, {}, {Game: g, Controller: uuid.New()}} {
		if _, known := predicate.Read(context, a.PlayerID()); known {
			t.Fatal("invalid context")
		}
	}
	if _, known := predicate.Read(ctx, uuid.New()); known {
		t.Fatal("absent recipient")
	}
	var zero RecipientPermanentCountPredicate
	if _, known := zero.Read(ctx, a.PlayerID()); known {
		t.Fatal("zero predicate")
	}
}

func TestRecipientCountInstructionCollections(t *testing.T) {
	g, a, b := newSpellCopyGame()
	member := BindPlayer("local")
	ref := FromPlayer(member)
	p := g.PutOnBattlefield(NewLand("land"), b.PlayerID())
	p.Card.SetOwner(a.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	for _, tc := range []struct {
		subjects PermanentSet
		a, b     int
	}{
		{PermanentsControlledBy(ref, IsLand), 0, 1},
		{PermanentsOwnedBy(ref, IsLand), 1, 0},
		{PermanentsControlledByPlayers(OpponentsOf(ref), IsLand), 1, 0},
		{UnionPermanents(PermanentsControlledBy(ref, IsLand), PermanentsOwnedBy(ref, IsLand)), 1, 1},
	} {
		value := CountPermanents(tc.subjects)
		for _, rounded := range []bool{false, true} {
			if rounded {
				value = DivideRoundUp(DivideRoundDown(value, Fixed(2)), Fixed(2))
			}
			predicate, ok := InspectRecipientPermanentCount(value, member)
			if !ok {
				t.Fatal("native collection rejected")
			}
			for i, id := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
				want := tc.a
				if i == 1 {
					want = tc.b
				}
				if rounded {
					want = 0
				}
				if got, known := predicate.Read(ctx, id); !known || got != want {
					t.Fatalf("count=%d/%v want=%d", got, known, want)
				}
			}
		}
	}
	if _, ok := InspectLivePermanentCount(CountPermanents(PermanentsControlledBy(ref, IsLand))); ok {
		t.Fatal("local scope escaped")
	}
	if _, ok := InspectRecipientPermanentCount(CountPermanents(PermanentsControlledBy(FromPlayer(nil), IsLand)), member); ok {
		t.Fatal("nil saved player accepted")
	}
}

func TestRecipientCountInstructionSpell(t *testing.T) {
	for _, state := range []string{"only", "prefix", "suffix", "unknown suffix", "foreign member", "missing target", "illegal missing target", "bound target", "empty recipients", "unknown source", "unknown players", "unknown after proof", "unknown illegal"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			member := BindPlayer("recipient")
			ref := FromPlayer(member)
			if state == "foreign member" {
				ref = FromPlayer(BindPlayer("recipient"))
			}
			subjects := PermanentsControlledBy(ref, IsLand)
			player := DeclarePlayerTarget("count player")
			if state == "missing target" || state == "illegal missing target" || state == "bound target" {
				subjects = IntersectPermanents(subjects, PermanentsControlledBy(player, IsLand))
				if state == "bound target" {
					obj.TargetSpecs = append(obj.TargetSpecs, player.targetSpec())
					obj.Targets = append(obj.Targets, obj.Controller)
				}
			}
			if state == "illegal missing target" {
				p.GrantBaseAttr(core.Shroud)
			}
			amount := CountPermanents(subjects)
			if state == "unknown suffix" || state == "unknown after proof" || state == "unknown illegal" {
				amount = damageInspectionValueTrap{}
			}
			players := OpponentsOf(You)
			if state == "empty recipients" {
				players = noDamagePlayers{}
			}
			source := DamageSourceRef(This)
			if state == "unknown source" {
				source = damageSourceQueryTrap{}
			}
			if state == "unknown players" {
				players = collectionDamagePlayerTrap{}
			}
			if state == "unknown illegal" {
				p.GrantBaseAttr(core.Shroud)
			}
			effect := PlayerCollectionDamageInstruction{Source: source, Subjects: players, Member: member, Amount: amount}.Effect()
			obj.Effects = []Effect{effect}
			want, known := false, true
			switch state {
			case "prefix":
				obj.Effects = []Effect{Sequence(effect, Destroy(target))}
				known = false
			case "suffix", "unknown suffix", "unknown after proof":
				obj.Effects = []Effect{Sequence(Destroy(target), effect)}
				want = state == "suffix"
			case "empty recipients":
				obj.Effects = []Effect{Sequence(effect, Destroy(target))}
				want = true
			}
			if state == "unknown suffix" || state == "unknown source" || state == "unknown players" || state == "unknown after proof" || state == "unknown illegal" || state == "foreign member" || state == "missing target" || state == "illegal missing target" {
				known = false
			}
			before := g.random.Clone()
			for range 2 {
				if got, k := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}}); got != want || k != known {
					t.Fatalf("query=%v/%v want=%v/%v", got, k, want, known)
				}
			}
			if !reflect.DeepEqual(before, g.random) || len(g.resolution.playerBindings) != 0 {
				t.Fatal("query changed state")
			}
		})
	}
}

func TestRecipientCountInstructionDescriptor(t *testing.T) {
	member := BindPlayer("member")
	good := PlayerCollectionDamageInstruction{This, EachPlayer, member, CountPermanents(PermanentsControlledBy(FromPlayer(member), IsLand))}
	effect := good.Effect()
	got, ok := InspectPlayerCollectionDamageInstruction(effect)
	if !ok || got.Source != good.Source || got.Subjects != good.Subjects || got.Member != member {
		t.Fatal("lost operands")
	}
	if _, known := InspectRecipientPermanentCount(got.Amount, member); !known {
		t.Fatal("lost amount binding")
	}
	var typedNil *playerDamageEachEffect
	for _, bad := range []ResolvingEffect{nil, typedNil, Sequence(effect), DealDamageEach(This, PlayerDamageRecipients(EachPlayer), Fixed(1))} {
		if _, ok := InspectPlayerCollectionDamageInstruction(bad); ok {
			t.Fatal("wrong instruction")
		}
	}
	for _, bad := range []PlayerCollectionDamageInstruction{{}, {nil, EachPlayer, member, Fixed(1)}, {This, nil, member, Fixed(1)}, {This, EachPlayer, nil, Fixed(1)}, {This, EachPlayer, BindPlayer(""), Fixed(1)}, {This, EachPlayer, member, nil}} {
		if bad.Effect() != nil {
			t.Fatal("invalid operands")
		}
	}
}
