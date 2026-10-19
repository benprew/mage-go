package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDeathHistoryDestination(t *testing.T) {
	for _, mode := range []string{"exile", "regeneration", "both"} {
		for _, change := range []string{"damage", "before", "none", "prevented", "zero", "other source", "source return", "recipient return", "source left", "source phase", "phase cycle", "phased", "cleanup", "not creature", "no regeneration", "hand", "cycle", "clone writes"} {
			t.Run(mode+"/"+change, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				source := g.PutOnBattlefield(NewCreature("Source", "", 1, 8), a.PlayerID())
				card := NewCreature("Recipient", "", 1, 8)
				card.SetOwner(b.PlayerID())
				p := g.PutOnBattlefield(card, b.PlayerID())
				if change == "before" {
					g.DealDamageToPermanent(p, 1, source.ID())
				}
				if mode != "regeneration" {
					if err := ApplyEffect(g, ExileDealtDamageByThisTurn(This, And(IsCreature, Not(IsArtifact))), source.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				if mode != "exile" {
					if err := ApplyEffect(g, CantRegenerateCreaturesDealtDamageByThisTurn(This), source.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				damaging := source
				switch change {
				case "prevented":
					g.AddPreventionShield(p.ID(), 1)
				case "other source":
					damaging = g.PutOnBattlefield(NewCreature("Other", "", 1, 8), a.PlayerID())
				case "source return":
					g.RemoveFromBattlefield(source)
					damaging = g.PutOnBattlefield(source.Card, a.PlayerID())
				}
				amount := 1
				if change == "zero" {
					amount = 0
				}
				if change != "none" && change != "before" && change != "clone writes" {
					g.DealDamageToPermanent(p, amount, damaging.ID())
				}
				hit := change != "none" && change != "zero" && change != "prevented" && change != "other source" && change != "source return" && change != "recipient return" && change != "cleanup" && change != "clone writes"
				switch change {
				case "recipient return":
					g.RemoveFromBattlefield(p)
					p = g.PutOnBattlefield(card, b.PlayerID())
				case "source left":
					g.RemoveFromBattlefield(source)
					g.zones.ClearLKI()
				case "source phase":
					g.PhaseOut(source.ID())
				case "phase cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "phased":
					g.PhaseOut(p.ID())
				case "cleanup":
					g.RunStepWithPriority(core.Cleanup)
				case "not creature":
					p.RevokeBaseAttr(core.AttrIsCreature)
					p.GrantBaseAttr(core.AttrIsArtifact)
				case "hand":
					p.RuntimeAbilities = append(p.RuntimeAbilities, ReturnToHandInsteadOfDyingWithPlayLock())
				case "cycle":
					g.effects.cycleReplacements = g.effects.replacements
					g.effects.replacements = nil
				}
				g.AddRegenerationShield(p.ID())
				visit := PermanentVisit{p.ID(), p.incarnationID}
				want := DestructionToGraveyard
				if change != "no regeneration" && (!hit || mode == "exile") {
					want = DestructionNoMove
				}
				if hit && mode != "regeneration" && change != "not creature" {
					if want == DestructionToGraveyard {
						want = 0
					}
					want |= DestructionToExile
				}
				if change == "hand" {
					want |= DestructionToHand
					want &^= DestructionToGraveyard
				}
				if change == "phased" {
					want = DestructionNoMove
				}
				before, random := g.Clone(), g.random.Clone()
				damage := p.Damage
				replacementCount := len(g.effects.replacements) + len(g.effects.cycleReplacements)
				shields := 0
				for _, replacement := range g.effects.replacements {
					if r, ok := replacement.(*regenerationReplacement); ok {
						shields += r.shields
					}
				}
				for _, game := range []*Game{g, g.Clone()} {
					for range 2 {
						got, known := game.InspectDestructionDestinations(visit, change == "no regeneration")
						if !known || got != want {
							t.Fatalf("got %v known %v want %v", got, known, want)
						}
					}
				}
				remainingShields := 0
				for _, replacement := range g.effects.replacements {
					if r, ok := replacement.(*regenerationReplacement); ok {
						remainingShields += r.shields
					}
				}
				after := g.Clone()
				if replacementCount != len(g.effects.replacements)+len(g.effects.cycleReplacements) || shields != remainingShields || !reflect.DeepEqual(before.damage.turnDamageToPermanents, after.damage.turnDamageToPermanents) || !reflect.DeepEqual(before.damage.turnDamageToCreatures, after.damage.turnDamageToCreatures) || !reflect.DeepEqual(random, g.random) || p.Tapped || p.Damage != damage {
					t.Fatal("query changed state")
				}
				if change == "clone writes" {
					clone := g.Clone()
					clone.DealDamageToPermanent(clone.FindPermanent(p.ID()), 1, source.ID())
					got, known := clone.InspectDestructionDestinations(visit, false)
					cloneWant := DestructionToGraveyard
					if mode == "exile" {
						cloneWant = DestructionNoMove | DestructionToExile
					}
					if mode == "both" {
						cloneWant = DestructionToExile
					}
					if !known || got != cloneWant {
						t.Fatal("clone history not read")
					}
					if got, known := g.InspectDestructionDestinations(visit, false); !known || got != want {
						t.Fatal("clone shared history")
					}
				}
				if want != DestructionNoMove && want != DestructionToGraveyard && want != DestructionToExile {
					return
				}
				for _, game := range []*Game{g.Clone(), g} {
					game.destroyPermanents([]*Permanent{game.FindPermanentIncludingPhased(p.ID())}, change == "no regeneration")
					if (game.FindPermanentIncludingPhased(p.ID()) != nil) != (want == DestructionNoMove) || (game.FindExiledCard(card.ID()) != nil) != (want == DestructionToExile) || (len(game.GetPlayer(b.PlayerID()).Graveyard()) == 1) != (want == DestructionToGraveyard) {
						t.Fatal("native destination disagrees")
					}
				}
			})
		}
	}
}

func TestDeathHistoryDestinationOpaque(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 8), a.PlayerID())
	p := g.PutOnBattlefield(NewCreature("Subject", "", 1, 8), b.PlayerID())
	g.DealDamageToPermanent(p, 1, source.ID())
	trap := NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("opaque filter") })
	for _, filter := range []PermanentFilter{trap, And(IsCreature, trap), Or(IsCreature, trap), Not(trap)} {
		g.effects.replacements = nil
		if err := ApplyEffect(g, ExileDealtDamageByThisTurn(This, filter), source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if got, known := g.InspectDestructionDestinations(PermanentVisit{p.ID(), p.incarnationID}, false); known || got != 0 {
			t.Fatal("opaque accepted")
		}
	}
	g.effects.replacements = nil
	filter := And(IsCreature, Not(IsArtifact))
	filter.fn = func(*Permanent, *Game) bool { panic("typed callback") }
	if err := ApplyEffect(g, ExileDealtDamageByThisTurn(This, filter), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got, known := g.InspectDestructionDestinations(PermanentVisit{p.ID(), p.incarnationID}, false); !known || got != DestructionToExile {
		t.Fatal("typed predicate not read")
	}
}

func TestDeathHistoryDestinationRecipientTypes(t *testing.T) {
	for _, atDamage := range []bool{false, true} {
		for _, creatureAtDamage := range []bool{false, true} {
			for _, creatureNow := range []bool{false, true} {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				source := g.PutOnBattlefield(NewCreature("Source", "", 1, 8), a.PlayerID())
				card := NewCreature("Changing", "", 1, 8)
				card.SetOwner(b.PlayerID())
				if !creatureAtDamage {
					card = NewArtifact("Changing", "")
					card.SetOwner(b.PlayerID())
				}
				p := g.PutOnBattlefield(card, b.PlayerID())
				effect := CantRegenerateDealtDamageByThisTurn(This, And(IsCreature, Not(IsLand)))
				if atDamage {
					effect = CantRegenerateCreaturesDealtDamageByThisTurn(This)
				}
				if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				g.DealDamageToPermanent(p, 1, source.ID())
				p.RevokeBaseAttr(core.AttrIsCreature)
				p.GrantBaseAttr(core.AttrIsArtifact)
				if creatureNow {
					p.RevokeBaseAttr(core.AttrIsArtifact)
					p.GrantBaseAttr(core.AttrIsCreature)
				}
				g.AddRegenerationShield(p.ID())
				prohibited := creatureNow
				if atDamage {
					prohibited = creatureAtDamage
				}
				want := DestructionNoMove
				if prohibited {
					want = DestructionToGraveyard
				}
				if got, known := g.InspectDestructionDestinations(PermanentVisit{p.ID(), p.incarnationID}, false); !known || got != want {
					t.Fatalf("at damage %v type then %v type now %v: got %v known %v", atDamage, creatureAtDamage, creatureNow, got, known)
				}
				g.DestroyPermanent(p)
				if (g.FindPermanent(p.ID()) == nil) != prohibited {
					t.Fatal("native type timing disagrees")
				}
			}
		}
	}
}

func TestDeathHistoryDestinationRegenerationUnknown(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 1, 8), a.PlayerID())
	g.AddRegenerationShield(p.ID())
	trap := NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter callback") })
	active := func(*Game, uuid.UUID) bool { panic("active callback") }
	for _, rule := range []*liveDamageRegenerationRule{
		nil,
		{},
		{funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.EndOfTurn, func(*Game, uuid.UUID) error { panic("effect callback") }, active).(*funcContinuousEffect), filter: AnyPermanent},
		{funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.EndOfTurn, func(*Game, uuid.UUID) error { panic("effect callback") }).(*funcContinuousEffect), filter: trap},
	} {
		g.effects.effects = []ContinuousEffect{rule}
		if got, known := g.InspectDestructionDestinations(PermanentVisit{p.ID(), p.incarnationID}, false); known || got != 0 {
			t.Fatal("opaque regeneration rule accepted")
		}
		if got, known := g.InspectDestructionDestinations(PermanentVisit{p.ID(), p.incarnationID}, true); !known || got != DestructionToGraveyard {
			t.Fatal("irrelevant prohibition reached")
		}
	}
}

func TestDeathHistoryDestinationOrders(t *testing.T) {
	for _, cycle := range []bool{false, true} {
		for _, reverse := range []bool{false, true} {
			for _, index := range []int{0, 1} {
				a := NewBasePlayer("A")
				b := &deathExileOrderPlayer{BasePlayer: NewBasePlayer("B"), index: index}
				g := NewGame(a, b)
				source := g.PutOnBattlefield(NewCreature("Source", "", 1, 8), a.PlayerID())
				p := g.PutOnBattlefield(NewCreature("Subject", "", 1, 8), b.PlayerID())
				p.Card.SetOwner(b.PlayerID())
				g.AddRegenerationShield(p.ID())
				if err := ApplyEffect(g, ExileDealtDamageByThisTurn(This, IsCreature), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				g.DealDamageToPermanent(p, 1, source.ID())
				if reverse {
					g.effects.replacements[0], g.effects.replacements[1] = g.effects.replacements[1], g.effects.replacements[0]
				}
				if cycle {
					g.effects.cycleReplacements = g.effects.replacements
					g.effects.replacements = nil
				}
				if got, known := g.InspectDestructionDestinations(PermanentVisit{p.ID(), p.incarnationID}, false); !known || got != DestructionNoMove|DestructionToExile || b.choices != 0 {
					t.Fatal("query selected an order")
				}
				g.DestroyPermanent(p)
				exile := (index == 1) != reverse
				if b.choices != 1 || (g.FindExiledCard(p.Card.ID()) != nil) != exile || (g.FindPermanent(p.ID()) != nil) == exile {
					t.Fatal("native order disagrees")
				}
			}
		}
	}
}
