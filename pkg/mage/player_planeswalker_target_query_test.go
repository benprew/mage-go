package mage

import (
	"reflect"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPlayerPlaneswalkerTargetQuery(t *testing.T) {
	for _, opponentOnly := range []bool{false, true} {
		for _, state := range []string{"opponent", "self", "own walker", "other walker", "creature", "phase", "cycle", "return", "shroud", "hexproof", "protection", "control", "type change", "missing visit", "missing zone", "foreign", "duplicate", "copy", "clone", "opaque protection"} {
			t.Run(map[bool]string{false: "any", true: "opponent"}[opponentOnly]+"/"+state, func(t *testing.T) {
				g, _, obj, _ := damageQueryFixture(t)
				a, b := g.players[0], g.players[1]
				target := DeclarePlayerOrPlaneswalkerTarget("recipient")
				if opponentOnly {
					target = DeclareOpponentOrPlaneswalkerTarget("recipient")
				}
				walker := g.PutOnBattlefield(NewPlaneswalker("Walker", "", 10), b.PlayerID())
				id := walker.ID()
				legal, known := true, true
				switch state {
				case "opponent", "copy":
					id = b.PlayerID()
				case "self":
					id = a.PlayerID()
					legal = !opponentOnly
				case "own walker":
					g.AddControlEffect(ControlEffectSpec{SourceID: walker.ID(), TargetID: walker.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "creature":
					id = g.PutOnBattlefield(NewCreature("Creature", "", 1, 10), b.PlayerID()).ID()
					legal = false
				}
				g.stack.Pop()
				obj.Targets = []uuid.UUID{id}
				obj.TargetSpecs = []Target{target.targetSpec()}
				obj.Effects = []Effect{DealDamageFrom(This, target, Fixed(2))}
				g.PushStack(obj)
				switch state {
				case "phase":
					g.PhaseOut(walker.ID())
					legal = false
				case "cycle":
					g.PhaseIn(g.PhaseOut(walker.ID()))
				case "return":
					g.RemoveFromBattlefield(walker)
					g.PutOnBattlefield(walker.Card, b.PlayerID())
					legal = false
				case "shroud":
					walker.GrantBaseAttr(core.Shroud)
					legal = false
				case "hexproof":
					walker.GrantBaseAttr(core.Hexproof)
					legal = false
				case "protection":
					walker.RuntimeAbilities = append(walker.RuntimeAbilities, ProtectionFromColor(core.Red))
					legal = false
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: walker.ID(), TargetID: walker.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "type change":
					walker.Card = NewCreature("Changed", "", 1, 10)
					legal = false
				case "missing visit":
					delete(obj.targetIncarnations, id)
					known = false
				case "missing zone":
					delete(obj.TargetZones, id)
					known = false
				case "foreign":
					obj.TargetSpecs[0] = DeclarePlayerOrPlaneswalkerTarget("recipient").targetSpec()
					known = false
				case "duplicate":
					obj.TargetSpecs = append(obj.TargetSpecs, obj.TargetSpecs[0])
					obj.Targets = append(obj.Targets, id)
					known = false
				case "copy":
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
					legal = !opponentOnly
				case "opaque protection":
					walker.RuntimeAbilities = append(walker.RuntimeAbilities, &ProtectionAbility{Filter: NewCardFilter("trap", func(Card) bool { panic("protection callback") })})
					known = false
				}
				predicate, ok := InspectDamageTarget(target)
				if !ok {
					t.Fatal("native player-or-planeswalker declaration rejected")
				}
				if state == "clone" {
					g = g.Clone()
					obj = g.StackPeek()
					g.PhaseOut(walker.ID())
					legal = false
				}
				random := g.random.Clone()
				for range 2 {
					selection, gotKnown := predicate.Select(g, obj)
					if gotKnown != known || known && !selection.Empty() != legal {
						t.Fatalf("selection=%+v known=%v want legal=%v known=%v", selection, gotKnown, legal, known)
					}
					if known && legal {
						if id == a.PlayerID() || id == b.PlayerID() {
							if selection.Player != id || selection.Permanent.ID != uuid.Nil {
								t.Fatal("player identity became a permanent")
							}
						} else if selection.Permanent.ID != id || selection.Player != uuid.Nil {
							t.Fatal("permanent visit became a player")
						}
					}
				}
				if !reflect.DeepEqual(random, g.random) {
					t.Fatal("query changed RNG")
				}
				if known && state != "copy" && state != "return" && state != "type change" && state != "foreign" && state != "duplicate" && state != "clone" {
					possible := target.spec.Possible(obj.Controller, obj.Card, g)
					if slices.Contains(possible, id) != legal {
						t.Fatal("query differs from native target legality")
					}
				}
				if walker.Counters[core.Loyalty] != 10 || a.Life() != 20 || b.Life() != 20 {
					t.Fatal("query applied damage")
				}
			})
		}
	}
}

func TestPlayerPlaneswalkerTargetQueryInvalid(t *testing.T) {
	var absent *DamageTargetBinding
	var absentSpec *PlayerOrPlaneswalkerTarget
	for _, target := range []*DamageTargetBinding{
		nil, absent, {}, {&targetDeclaration{spec: absentSpec}},
		{&targetDeclaration{spec: &PlayerOrPlaneswalkerTarget{BaseTarget: BaseTarget{min: 0, max: 1}}}},
		{&targetDeclaration{spec: &PlayerOrPlaneswalkerTarget{BaseTarget: BaseTarget{min: 1, max: 2}}}},
		DeclareOpponentChosenDamageTarget("chooser"),
	} {
		if _, ok := InspectDamageTarget(target); ok {
			t.Fatal("incomplete or opaque declaration accepted")
		}
	}
	g, _, obj, _ := damageQueryFixture(t)
	predicate, _ := InspectDamageTarget(DeclarePlayerOrPlaneswalkerTarget("foreign"))
	for _, stack := range []*StackObject{nil, obj} {
		if _, known := predicate.Select(g, stack); known {
			t.Fatal("missing declaration accepted")
		}
	}
	if _, known := predicate.Select(nil, obj); known {
		t.Fatal("nil game accepted")
	}
	if _, known := (DamageTargetPredicate{}).Select(g, obj); known {
		t.Fatal("zero predicate accepted")
	}
}

func TestPlayerPlaneswalkerTargetQuerySpell(t *testing.T) {
	for _, state := range []string{"singular", "packets", "prevention", "prefix", "suffix", "partial", "all illegal", "missing", "opaque amount", "opaque suffix", "opaque all illegal", "opaque child", "zero"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, _ := damageQueryFixture(t)
			target := DeclareOpponentOrPlaneswalkerTarget("recipient")
			land := DeclarePermanentTarget("land", IsLand)
			g.stack.Pop()
			obj.TargetSpecs = []Target{target.targetSpec(), land.targetSpec()}
			obj.Targets = []uuid.UUID{g.players[1].PlayerID(), p.ID()}
			g.PushStack(obj)
			effect := DealDamageFrom(This, target, Fixed(2))
			want, known := false, true
			switch state {
			case "packets":
				effect = DealDamagePackets(This, DamagePacketTo(target, Fixed(2)), DamagePacketTo(You, Fixed(1)))
			case "prevention":
				effect = (FuturePreventionInstruction{Recipient: target, Amount: Fixed(2)}).Effect()
			case "zero":
				effect = DealDamagePackets(This, DamagePacketTo(target, Fixed(0)))
			case "partial", "all illegal":
				obj.Targets[0] = obj.Controller
			case "missing":
				obj.TargetSpecs[0] = DeclareOpponentOrPlaneswalkerTarget("recipient").targetSpec()
				known = false
			case "opaque amount", "opaque suffix", "opaque all illegal":
				effect = DealDamageFrom(This, target, damageInspectionValueTrap{})
				known = false
			}
			obj.Effects = []Effect{effect}
			switch state {
			case "prefix", "zero":
				obj.Effects = []Effect{Sequence(effect, Destroy(land))}
				known = false
			case "suffix", "partial":
				obj.Effects = []Effect{Sequence(Destroy(land), effect)}
				want = true
			case "all illegal":
				p.GrantBaseAttr(core.Shroud)
				obj.Effects = []Effect{Sequence(effect, Destroy(land))}
			case "opaque suffix":
				obj.Effects = []Effect{Sequence(Destroy(land), effect)}
			case "opaque all illegal":
				obj.Targets[0] = obj.Controller
				p.GrantBaseAttr(core.Shroud)
			case "opaque child":
				obj.Effects = []Effect{Sequence(Destroy(land), effect, destructionInspectionTrap{})}
				known = false
			}
			for _, game := range []*Game{g, g.Clone()} {
				for range 2 {
					if got, gotKnown := game.InspectSpellDestruction(game.StackPeek(), []PermanentVisit{{p.ID(), p.incarnationID}}); got != want || gotKnown != known {
						t.Fatalf("proof=%v/%v want=%v/%v", got, gotKnown, want, known)
					}
				}
			}
			if p.Damage != 0 || g.players[0].Life() != 20 || g.players[1].Life() != 20 {
				t.Fatal("query changed game state")
			}
		})
	}
}
