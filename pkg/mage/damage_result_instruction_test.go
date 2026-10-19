package mage

import (
	"reflect"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageResultInstructionDeclaration(t *testing.T) {
	total, results := BindNumber("total"), BindDamageResults("outcomes")
	target := DeclareDamageTarget("target")
	single := DamageResultInstruction{Source: This, Recipient: target, Amount: Fixed(3), Total: total, Results: results}
	group := single
	group.Recipient, group.Recipients = nil, PermanentDamageRecipients(MatchingPermanents(IsCreature))
	for _, instruction := range []DamageResultInstruction{single, group} {
		got, ok := InspectDamageResultInstruction(instruction.Effect())
		if !ok || got.Total != total || got.Results != results || got.Recipient != instruction.Recipient || (got.Recipients == nil) != (instruction.Recipients == nil) {
			t.Fatal("lost native result declarations")
		}
		got.Total = nil
		again, _ := InspectDamageResultInstruction(instruction.Effect())
		if again.Total != total {
			t.Fatal("inspection changed declaration")
		}
	}
	for _, state := range []string{"source", "recipient", "amount", "total", "results", "unnamed total", "unnamed results", "both"} {
		bad := single
		switch state {
		case "source":
			bad.Source = nil
		case "recipient":
			bad.Recipient = nil
		case "amount":
			bad.Amount = nil
		case "total":
			bad.Total = nil
		case "results":
			bad.Results = nil
		case "unnamed total":
			bad.Total = BindNumber("")
		case "unnamed results":
			bad.Results = BindDamageResults("")
		case "both":
			bad.Recipients = group.Recipients
		}
		if bad.Effect() != nil {
			t.Fatal("invalid descriptor", state)
		}
	}
	var absent *damageFromEffect
	for _, bad := range []Effect{
		nil, absent, Sequence(single.Effect()),
		DealDamageFrom(This, target, Fixed(1), DamageAmountInto(total)),
		DealDamageFrom(This, target, Fixed(1), DamageAmountInto(total), DamageAmountInto(total), DamageOutcomesInto(results)),
		DealDamageFrom(This, target, Fixed(1), DamageAmountInto(total), DamageOutcomesInto(results), DamagePlayerInto(BindPlayer("player"))),
		DealDamageEach(This, group.Recipients, Fixed(1), DamageTotalInto(total), DamageResultsInto(nil)),
		DealDamageEach(This, group.Recipients, Fixed(1), DamageTotalInto(total), DamageResultsInto(results), DamageResultsInto(results)),
	} {
		if _, ok := InspectDamageResultInstruction(bad); ok {
			t.Fatal("invalid result effect accepted")
		}
	}
}

func TestDamageResultInstructionSpell(t *testing.T) {
	for _, clamp := range []bool{false, true} {
		for _, state := range []string{"normal", "suffix", "prefix", "forward", "missing", "duplicate", "foreign", "foreign total", "opaque", "illegal opaque", "phase", "cycle", "returned", "missing visit", "copy"} {
			t.Run(map[bool]string{false: "total", true: "clamped"}[clamp]+"/"+state, func(t *testing.T) {
				g, p, obj, target := damageQueryFixture(t)
				total, results := BindNumber("total"), BindDamageResults("outcomes")
				producer := (DamageResultInstruction{Source: This, Recipient: target, Amount: Fixed(3), Total: total, Results: results}).Effect()
				amount := FromNumber(total)
				if clamp {
					amount = ClampedDamageAmount(results)
				}
				if state == "foreign" {
					amount = ClampedDamageAmount(BindDamageResults("outcomes"))
				}
				if state == "foreign total" {
					amount = FromNumber(BindNumber("total"))
				}
				consumer := GainLifeFor(You, amount)
				obj.Effects = []Effect{Sequence(producer, consumer)}
				known, certain := true, false
				switch state {
				case "suffix":
					obj.Effects = []Effect{Sequence(DestroyEach(MatchingPermanents(IsLand)), producer, consumer)}
					certain = true
				case "prefix":
					obj.Effects = []Effect{Sequence(producer, consumer, DestroyEach(MatchingPermanents(IsLand)))}
					known = false
				case "forward":
					obj.Effects = []Effect{Sequence(consumer, producer)}
					known = false
				case "missing":
					obj.Effects = []Effect{consumer}
					known = false
				case "duplicate":
					obj.Effects = []Effect{Sequence(producer, producer, consumer)}
					known = false
				case "foreign", "foreign total":
					known = false
				case "opaque", "illegal opaque":
					obj.Effects = append(obj.Effects, destructionInspectionTrap{})
					known = false
					if state == "illegal opaque" {
						p.GrantBaseAttr(core.Shroud)
					}
				case "phase":
					g.PhaseOut(p.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "returned":
					g.BouncePermanentToHand(p)
					g.PutOnBattlefield(p.Card, obj.Controller)
				case "missing visit":
					obj.targetIncarnations = nil
					known = false
				case "copy":
					obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
				}
				rng := g.random.Clone()
				for _, game := range []*Game{g, g.Clone()} {
					for range 2 {
						got, k := game.InspectSpellDestruction(game.StackPeek(), []PermanentVisit{{p.ID(), p.incarnationID}})
						if got != certain || k != known {
							t.Fatalf("got %v/%v want %v/%v", got, k, certain, known)
						}
					}
				}
				if g.GetPlayer(obj.Controller).Life() != 20 || p.Damage != 0 || !reflect.DeepEqual(g.random, rng) {
					t.Fatal("query changed state")
				}
			})
		}
	}
}

func TestDamageResultInstructionBindings(t *testing.T) {
	for _, state := range []string{"collection total", "collection clamp", "duplicate total", "duplicate outcomes", "forward total", "forward outcomes", "unknown amount", "unknown recipient", "unknown source", "illegal unbound", "unknown all illegal", "empty collection", "partial illegal"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := damageQueryFixture(t)
			total, results := BindNumber("total"), BindDamageResults("outcomes")
			instruction := DamageResultInstruction{Source: This, Recipient: target, Amount: Fixed(0), Total: total, Results: results}
			life := GainLifeFor(You, FromNumber(total))
			known := true
			switch state {
			case "collection total", "collection clamp", "empty collection":
				instruction.Recipient = nil
				instruction.Recipients = PermanentDamageRecipients(MatchingPermanents(IsLand))
				if state == "empty collection" {
					instruction.Recipients = PermanentDamageRecipients(MatchingPermanents(IsArtifact))
				}
				if state == "collection clamp" {
					life = GainLifeFor(You, ClampedDamageAmount(results))
					known = false
				}
			case "unknown amount":
				instruction.Amount = damageInspectionValueTrap{}
				known = false
			case "unknown recipient":
				instruction.Recipient = packetInspectionRecipientTrap{}
				known = false
			case "unknown source":
				instruction.Source = DamageSourceOf(This)
				known = false
			}
			producer := instruction.Effect()
			obj.Effects = []Effect{Sequence(producer, life)}
			switch state {
			case "duplicate total":
				instruction.Results = BindDamageResults("other")
				obj.Effects = []Effect{Sequence(producer, instruction.Effect(), life)}
				known = false
			case "duplicate outcomes":
				instruction.Total = BindNumber("other")
				obj.Effects = []Effect{Sequence(producer, instruction.Effect(), life)}
				known = false
			case "forward total":
				obj.Effects = []Effect{Sequence(life, producer)}
				known = false
			case "forward outcomes":
				obj.Effects = []Effect{Sequence(GainLifeFor(You, ClampedDamageAmount(results)), producer)}
				known = false
			case "illegal unbound":
				p.GrantBaseAttr(core.Shroud)
				obj.Effects = []Effect{life}
				known = false
			case "unknown all illegal":
				p.GrantBaseAttr(core.Shroud)
				obj.Effects = append(obj.Effects, GainLifeFor(You, ClampedDamageAmount(BindDamageResults("foreign"))))
				known = false
			case "partial illegal":
				p.GrantBaseAttr(core.Shroud)
				other := DeclarePlayerTarget("other")
				obj.TargetSpecs = append(obj.TargetSpecs, other.targetSpec())
				obj.Targets = append(obj.Targets, obj.Controller)
			}
			for _, game := range []*Game{g, g.Clone()} {
				got, k := game.InspectSpellDestruction(game.StackPeek(), nil)
				if got || k != known {
					t.Fatalf("got %v/%v want false/%v", got, k, known)
				}
				if len(game.resolution.numberBindings) != 0 || len(game.resolution.damageResultsBindings) != 0 {
					t.Fatal("query wrote result bindings")
				}
			}
		})
	}
}
