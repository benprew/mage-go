package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCollectionDamageInstructionDeclaration(t *testing.T) {
	recipients := DamageRecipients(EachPlayer, MatchingPermanents(And(IsCreature, NotHasKeywordFilter(core.Flying))))
	instruction := CollectionDamageInstruction{Source: This, Recipients: recipients, Amount: Fixed(3)}
	effect := instruction.Effect()
	instructions, ok := InspectCollectionDamageInstructions(Sequence(effect, Sequence(effect)))
	if !ok || len(instructions) != 2 {
		t.Fatal("native collection instruction lost")
	}
	instructions[0].Source = nil
	again, _ := InspectCollectionDamageInstructions(effect)
	if again[0].Source == nil {
		t.Fatal("inspection aliases declaration")
	}
	var absent *DamageTargetsBinding
	for _, bad := range []CollectionDamageInstruction{{}, {Source: This, Recipients: absent, Amount: Fixed(1)}, {Recipients: recipients, Amount: Fixed(1)}, {Source: This, Recipients: recipients}, {Source: This, Amount: Fixed(1)}} {
		if bad.Effect() != nil {
			t.Fatal("incomplete instruction built an effect")
		}
	}
	var absentEffect *damageEachEffect
	for _, bad := range []Effect{nil, absentEffect, DealDamageEach(nil, recipients, Fixed(1)), DealDamageEach(This, nil, Fixed(1)), DealDamageEach(This, recipients, nil), DealDamageEach(This, recipients, Fixed(1), nil), DealDamageEach(This, recipients, Fixed(1), DamageTotalInto(&NumberBinding{})), DealDamageEach(This, recipients, Fixed(1), DamageResultsInto(&DamageResultsBinding{})), DealDamageEvenlyToRecipients(This, recipients, Fixed(1)), Sequence(effect, destructionInspectionTrap{})} {
		if _, ok := InspectCollectionDamageInstructions(bad); ok {
			t.Fatal("unsupported collection declaration accepted")
		}
	}
	for _, bad := range []DamageRecipientSet{nil, absent, DamageRecipients(EachPlayer, nil), DamageRecipients(nil, MatchingPermanents(IsCreature)), DamageRecipients(collectionDamagePlayerTrap{}, MatchingPermanents(IsCreature)), PermanentDamageRecipients(collectionDamageSetTrap{}), PermanentDamageRecipients(MatchingPermanents(NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter callback") })))} {
		if _, ok := InspectDamageRecipientCollection(bad); ok {
			t.Fatal("incomplete recipient set accepted")
		}
	}
}

type collectionDamagePlayerTrap struct{ PlayerSet }

func (collectionDamagePlayerTrap) resolvePlayers(*EffectContext) []uuid.UUID {
	panic("player callback")
}

type collectionDamageSetTrap struct{ PermanentSet }

func (collectionDamageSetTrap) resolvePermanents(*EffectContext) []*Permanent {
	panic("collection callback")
}

func TestCollectionDamageInstructionRecipients(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Ground", "", 1, 8), a.PlayerID())
	q := g.PutOnBattlefield(NewCreature("Flyer", "", 1, 8, WithKeyword(core.Flying)), b.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	for _, players := range []PlayerSet{EachPlayer, noDamagePlayers{}} {
		for _, filter := range []PermanentFilter{HasKeywordFilter(core.Flying), NotHasKeywordFilter(core.Flying), And(IsCreature, Not(HasKeywordFilter(core.Flying))), Or(HasKeywordFilter(core.Flying), IsTapped)} {
			set := DamageRecipients(players, MatchingPermanents(filter))
			predicate, ok := InspectDamageRecipientCollection(set)
			if !ok {
				t.Fatal("native recipients rejected")
			}
			for _, state := range []string{"normal", "grant", "remove", "phase", "cycle", "return", "clone"} {
				game := g.Clone()
				child := *ctx
				child.Game = game
				current := game.FindPermanent(p.ID())
				switch state {
				case "grant":
					if err := ApplyEffect(game, Until(core.EndOfTurn, GrantKeywordModification(This, core.Flying)), p.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "remove":
					if err := ApplyEffect(game, Until(core.EndOfTurn, RemoveKeyword(This, core.Flying)), q.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "phase":
					game.PhaseOut(p.ID())
				case "cycle":
					game.PhaseIn(game.PhaseOut(p.ID()))
				case "return":
					game.RemoveFromBattlefield(current)
					current = game.PutOnBattlefield(current.Card, b.PlayerID())
					if current.incarnationID == p.incarnationID {
						t.Fatal("return kept old visit")
					}
				}
				before := game.Clone()
				got, known := predicate.Select(&child)
				if !known {
					t.Fatal("known collection rejected at selection")
				}
				var want []DamageTargetSelection
				for _, id := range players.resolvePlayers(&child) {
					want = append(want, DamageTargetSelection{Player: id})
				}
				for _, member := range MatchingPermanents(filter).resolvePermanents(&child) {
					want = append(want, DamageTargetSelection{Permanent: PermanentVisit{member.ID(), member.incarnationID}})
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("%s: got %v want %v", state, got, want)
				}
				if len(got) > 0 {
					got[0] = DamageTargetSelection{}
				}
				again, _ := predicate.Select(&child)
				if !reflect.DeepEqual(again, want) || !reflect.DeepEqual(game.random, before.random) || game.FindPermanentIncludingPhased(p.ID()).Damage != 0 {
					t.Fatal("query shares or changes state")
				}
			}
		}
	}
	var zero DamageRecipientCollectionPredicate
	if _, known := zero.Select(ctx); known {
		t.Fatal("zero query accepted")
	}
	query, _ := InspectDamageRecipientCollection(PermanentDamageRecipients(MatchingPermanents(IsCreature)))
	for _, bad := range []*EffectContext{nil, {}, {Game: g}, {Game: g, Controller: uuid.New()}} {
		if _, known := query.Select(bad); known {
			t.Fatal("invalid context accepted")
		}
	}
}

func TestCollectionDamageInstructionOrder(t *testing.T) {
	for _, state := range []string{"only", "prefix", "empty prefix", "zero prefix", "player prefix", "suffix", "all illegal", "opaque amount", "opaque source", "opaque recipients", "opaque child", "opaque replacement"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			obj.Targets = nil
			obj.TargetSpecs = nil
			recipients := PermanentDamageRecipients(MatchingPermanents(IsCreature))
			p.GrantBaseAttr(core.AttrIsCreature)
			damage := DealDamageEach(This, recipients, Fixed(2))
			obj.Effects = []Effect{damage}
			want, known := false, true
			switch state {
			case "prefix":
				obj.Effects = []Effect{damage, DestroyEach(MatchingPermanents(IsLand))}
				known = false
			case "empty prefix":
				p.RevokeBaseAttr(core.AttrIsCreature)
				obj.Effects = []Effect{damage, DestroyEach(MatchingPermanents(IsLand))}
				want = true
			case "zero prefix":
				obj.Effects = []Effect{DealDamageEach(This, recipients, Fixed(0)), DestroyEach(MatchingPermanents(IsLand))}
				known = false
			case "player prefix":
				obj.Effects = []Effect{DealDamageEach(This, DamageRecipients(EachPlayer, MatchingPermanents(IsArtifact)), Fixed(1)), DestroyEach(MatchingPermanents(IsLand))}
				known = false
			case "suffix":
				obj.Effects = []Effect{DestroyEach(MatchingPermanents(IsLand)), damage}
				want = true
			case "all illegal":
				obj.Targets = []uuid.UUID{p.ID()}
				obj.TargetSpecs = []Target{target.targetSpec()}
				g.stack.Pop()
				g.PushStack(obj)
				p.GrantBaseAttr(core.Shroud)
				obj.Effects = []Effect{damage, Destroy(target)}
			case "opaque amount":
				obj.Effects = []Effect{DealDamageEach(This, recipients, damageInspectionValueTrap{})}
				known = false
			case "opaque source":
				obj.Effects = []Effect{DealDamageEach(DamageSourceOf(This), recipients, Fixed(2))}
				known = false
			case "opaque recipients":
				obj.Effects = []Effect{DealDamageEach(This, PermanentDamageRecipients(collectionDamageSetTrap{}), Fixed(2))}
				known = false
			case "opaque child":
				obj.Effects = []Effect{Sequence(damage, destructionInspectionTrap{})}
				known = false
			case "opaque replacement":
				g.effects.AddReplacement(&proposalTrap{})
			}
			for _, game := range []*Game{g, g.Clone()} {
				for range 2 {
					if got, gotKnown := game.InspectSpellDestruction(game.StackPeek(), []PermanentVisit{{p.ID(), p.incarnationID}}); got != want || gotKnown != known {
						t.Fatalf("proof=%v/%v want=%v/%v", got, gotKnown, want, known)
					}
				}
			}
			if p.Damage != 0 || g.GetPlayer(obj.Controller).Life() != 20 {
				t.Fatal("inspection dealt damage")
			}
		})
	}
}

func TestCollectionDamageInstructionPureKeyword(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Flyer", "", 1, 8, WithKeyword(core.Flying)), a.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	filter := HasKeywordFilter(core.Flying)
	filter.fn = func(*Permanent, *Game) bool { panic("native filter callback") }
	query, ok := InspectDamageRecipientCollection(PermanentDamageRecipients(MatchingPermanents(And(IsCreature, filter))))
	if !ok {
		t.Fatal("native keyword rejected")
	}
	got, known := query.Select(ctx)
	if !known || len(got) != 1 || got[0].Permanent.Incarnation != p.incarnationID {
		t.Fatal("pure keyword query lost member")
	}
	p.FaceDown = true
	got, known = query.Select(ctx)
	if !known || len(got) != 0 {
		t.Fatal("face-down object kept printed flying")
	}
	p.FaceDown = false
	p.RevokeBaseAttr(core.Flying)
	got, known = query.Select(ctx)
	if !known || len(got) != 0 {
		t.Fatal("query kept old keyword")
	}
	opaque := NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("opaque filter callback") })
	for _, filter := range []PermanentFilter{And(filter, opaque), Or(filter, opaque), Not(opaque)} {
		if _, ok := InspectDamageRecipientCollection(PermanentDamageRecipients(MatchingPermanents(filter))); ok {
			t.Fatal("unknown keyword tree accepted")
		}
	}
}
