package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetCharacteristicValueRead(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phase", "phase cycle", "shroud", "negative"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewCreature("subject", "{3}{G}", 3, 8), b.PlayerID())
			p.AddCounter(core.P1P1, 2)
			target := DeclarePermanentTarget("value", AnyPermanent)
			ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{p.ID(), p.incarnationID}}}}
			wantPower := 5
			switch state {
			case "departed", "returned":
				g.BouncePermanentToHand(p)
				if state == "returned" {
					b.RemoveFromHand(p.ID())
					q := g.PutOnBattlefield(p.Card, a.PlayerID())
					q.AddCounter(core.P1P1, 9)
				}
			case "phase":
				g.PhaseOut(p.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "shroud":
				p.GrantBaseAttr(core.Shroud)
			case "negative":
				if err := ApplyEffect(g, Until(core.EndOfTurn, ModifyPT(This, Fixed(-9), Fixed(0))), p.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				wantPower = -4
			}
			for i, value := range []ResolutionValue{PowerOfWithLKI(target), ManaValueOfWithLKI(target)} {
				predicate, ok := InspectTargetCharacteristicValue(value)
				if !ok {
					t.Fatal("native target value rejected")
				}
				want := wantPower
				if i == 1 {
					want = 4
				}
				rng := g.random.Clone()
				for _, game := range []*Game{g, g.Clone()} {
					local := *ctx
					local.Game = game
					for range 2 {
						if got, known := predicate.Read(&local); !known || got != want || got != value.resolveValue(&local) {
							t.Fatalf("read=%d/%v want=%d", got, known, want)
						}
					}
				}
				if !reflect.DeepEqual(rng, g.random) || a.Life() != 20 || b.Life() != 20 {
					t.Fatal("query changed state")
				}
			}
		})
	}
}

func TestTargetCharacteristicValueBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("subject", "{3}", 2, 4), a.PlayerID())
	target, foreign := DeclarePermanentTarget("same", AnyPermanent), DeclarePermanentTarget("same", AnyPermanent)
	predicate, _ := InspectTargetCharacteristicValue(PowerOfWithLKI(target))
	for _, frame := range []map[*targetDeclaration][]capturedPermanent{
		nil, {foreign.targetDeclaration: {{p.ID(), p.incarnationID}}},
		{target.targetDeclaration: {{id: p.ID()}}}, {target.targetDeclaration: {{incarnation: p.incarnationID}}},
		{target.targetDeclaration: {{p.ID(), p.incarnationID}, {p.ID(), p.incarnationID}}},
	} {
		if _, known := predicate.Read(&EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: frame}); known {
			t.Fatal("invalid frame accepted")
		}
	}
	for _, ctx := range []*EffectContext{nil, {}, {Game: g, Controller: uuid.New()}} {
		if _, known := predicate.Read(ctx); known {
			t.Fatal("invalid context accepted")
		}
	}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: nil}}
	if got, known := predicate.Read(ctx); !known || got != 0 {
		t.Fatal("empty legal frame is unknown")
	}
	if _, known := (TargetCharacteristicValuePredicate{}).Read(ctx); known {
		t.Fatal("zero predicate accepted")
	}
	ctx.targetBindings[target.targetDeclaration] = []capturedPermanent{{id: uuid.New(), incarnation: uuid.New()}}
	if _, known := predicate.Read(ctx); known {
		t.Fatal("missing information accepted")
	}
	var absent *PermanentTargetBinding
	for _, value := range []ResolutionValue{nil, damageInspectionValueTrap{}, Fixed(1), PowerOfWithLKI(absent), PowerOfWithLKI(This), PowerOfWithLKI(savedPermanentRef{BindPermanent("saved")}), ToughnessOfWithLKI(target), PowerOfWithLKI(DeclarePermanentTarget("opaque", NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter called") }))), AddValues(Fixed(1), PowerOfWithLKI(target))} {
		if _, ok := InspectTargetCharacteristicValue(value); ok {
			t.Fatal("unsupported value accepted")
		}
	}
}

func TestTargetCharacteristicValueSpell(t *testing.T) {
	for _, mana := range []bool{false, true} {
		for _, state := range []string{"only", "loss", "prefix", "suffix", "phase", "phase cycle", "returned", "shroud", "missing visit", "foreign", "duplicate", "opaque suffix", "opaque all illegal", "partial illegal"} {
			t.Run(map[bool]string{true: "mana", false: "power"}[mana]+"/"+state, func(t *testing.T) {
				g, p, obj, target := targetQueryFixture(t)
				value := PowerOfWithLKI(target)
				if mana {
					value = ManaValueOfWithLKI(target)
				}
				life := (LifeInstruction{Subject: You, Amount: value, Gain: true}).Effect()
				obj.Effects = []Effect{life}
				want, known := false, true
				watched := []PermanentVisit{{p.ID(), p.incarnationID}}
				switch state {
				case "loss":
					obj.Effects = []Effect{(LifeInstruction{Subject: You, Amount: value}).Effect()}
				case "prefix":
					obj.Effects = []Effect{Sequence(life, Destroy(target))}
					known = false
				case "suffix":
					obj.Effects = []Effect{Sequence(Destroy(target), life)}
					want = true
				case "phase":
					g.PhaseOut(p.ID())
				case "phase cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "returned":
					g.BouncePermanentToHand(p)
					g.PutOnBattlefield(p.Card, obj.Controller)
				case "shroud":
					p.GrantBaseAttr(core.Shroud)
				case "missing visit":
					obj.targetIncarnations = nil
					known = false
				case "foreign":
					obj.Effects = []Effect{GainLifeFor(You, PowerOfWithLKI(DeclarePermanentTarget("query", AnyPermanent)))}
					known = false
				case "duplicate":
					obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
					obj.Targets = append(obj.Targets, p.ID())
					known = false
				case "opaque suffix":
					obj.Effects = []Effect{Sequence(Destroy(target), life, destructionInspectionTrap{})}
					known = false
				case "opaque all illegal":
					p.GrantBaseAttr(core.Shroud)
					obj.Effects = []Effect{Sequence(life, destructionInspectionTrap{})}
					known = false
				case "partial illegal":
					p.GrantBaseAttr(core.Shroud)
					q := g.PutOnBattlefield(NewLand("other"), obj.Controller)
					other := DeclarePermanentTarget("other", IsLand)
					obj.TargetSpecs = append(obj.TargetSpecs, other.targetSpec())
					obj.Targets = append(obj.Targets, q.ID())
					obj.TargetZones[q.ID()] = core.ZoneBattlefield
					obj.targetIncarnations[q.ID()] = q.incarnationID
					obj.Effects = []Effect{Sequence(life, Destroy(other))}
					watched = []PermanentVisit{{q.ID(), q.incarnationID}}
					known = false
				}
				rng := g.random.Clone()
				for range 2 {
					if got, k := g.InspectSpellDestruction(obj, watched); got != want || k != known {
						t.Fatalf("query=%v/%v want=%v/%v", got, k, want, known)
					}
				}
				if !reflect.DeepEqual(rng, g.random) || g.GetPlayer(obj.Controller).Life() != 20 {
					t.Fatal("query changed state")
				}
			})
		}
	}
}
