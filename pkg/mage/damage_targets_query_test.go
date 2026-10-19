package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestEvenDamageTargetsQuery(t *testing.T) {
	for _, state := range []string{"normal", "phase", "cycle", "return", "shroud", "player", "empty", "foreign", "duplicate", "missing-visit", "missing-zone", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, _ := targetQueryFixture(t)
			p.GrantBaseAttr(core.AttrIsCreature)
			recipients := DeclareAnyNumberDamageTargets("recipients")
			query, ok := InspectDamageTargets(recipients)
			if !ok {
				t.Fatal("lost native group")
			}
			obj.Card = NewInstant("Division", "", Spell(DealDamageEvenly(This, recipients, Fixed(5)), WithTargets(recipients)))
			obj.Targets = []uuid.UUID{p.ID()}
			obj.TargetSpecs = []Target{recipients.targetDeclaration}
			want, known := 1, true
			switch state {
			case "phase":
				g.PhaseOut(p.ID())
				want = 0
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, p.ControllerID())
				want = 0
			case "shroud":
				p.GrantBaseAttr(core.Shroud)
				want = 0
			case "player":
				obj.Targets = []uuid.UUID{obj.Controller}
			case "empty":
				obj.Targets = nil
				obj.TargetSpecs = nil
				want = 0
			case "foreign":
				foreign := DeclareAnyNumberDamageTargets("recipients")
				obj.TargetSpecs = []Target{foreign.targetDeclaration}
				obj.Card = NewInstant("Foreign", "", Spell(DealDamageEvenly(This, foreign, Fixed(5)), WithTargets(foreign)))
				known = false
			case "duplicate":
				obj.Targets = append(obj.Targets, p.ID())
				obj.TargetSpecs = append(obj.TargetSpecs, recipients.targetDeclaration)
				known = false
			case "missing-visit":
				delete(obj.targetIncarnations, p.ID())
				known = false
			case "missing-zone":
				delete(obj.TargetZones, p.ID())
				known = false
			case "clone":
				g = g.Clone()
			}
			selected, gotKnown := query.Select(g, obj)
			if gotKnown != known || known && len(selected) != want {
				t.Fatalf("selection=%v/%v want=%d/%v", selected, gotKnown, want, known)
			}
			if known && len(selected) > 0 {
				selected[0] = DamageTargetSelection{}
				again, _ := query.Select(g, obj)
				if again[0].Empty() {
					t.Fatal("query aliases state")
				}
			}
			if p.Damage != 0 {
				t.Fatal("query dealt damage")
			}
		})
	}
	var absent *DamageTargetsBinding
	for _, bad := range []*DamageTargetsBinding{nil, absent, {}, DeclareXDamageTargets("x"), DeclareDamageTargets("bad", -1, 2)} {
		if _, ok := InspectDamageTargets(bad); ok {
			t.Fatal("unsupported group accepted")
		}
	}
}

func TestEvenDamageTargetFrames(t *testing.T) {
	g, p, obj, _ := targetQueryFixture(t)
	recipients := DeclareAnyNumberDamageTargets("recipients")
	query, ok := InspectDamageRecipientCollection(recipients)
	if !ok {
		t.Fatal("native group rejected")
	}
	ctx := &EffectContext{Game: g, Controller: obj.Controller}
	for _, values := range [][]capturedPermanent{{{id: uuid.Nil}}, {{id: p.ID()}}, {{id: obj.Controller, incarnation: uuid.New()}}, {{id: p.ID(), incarnation: p.incarnationID}, {id: p.ID(), incarnation: p.incarnationID}}} {
		ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{recipients.targetDeclaration: values}
		if _, known := query.Select(ctx); known {
			t.Fatal("malformed frame accepted")
		}
	}
	ctx.targetBindings = nil
	if _, known := query.Select(ctx); known {
		t.Fatal("missing frame accepted")
	}
	for _, values := range [][]capturedPermanent{nil, {{id: p.ID(), incarnation: p.incarnationID}}, {{id: obj.Controller}}} {
		ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{recipients.targetDeclaration: values}
		got, known := query.Select(ctx)
		if !known || len(got) != len(values) {
			t.Fatal("valid frame rejected")
		}
	}
	foreign := DeclareAnyNumberDamageTargets("recipients")
	ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{foreign.targetDeclaration: nil}
	if _, known := query.Select(ctx); known {
		t.Fatal("foreign frame accepted")
	}
}
