package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func pluralQueryFixture(t *testing.T, count int) (*Game, *Permanent, *Permanent, *StackObject, *PermanentTargetsBinding) {
	t.Helper()
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	first := g.PutOnBattlefield(NewCreature("Flyer", "", 1, 8, WithKeyword(core.Flying)), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Ground", "", 1, 8), b.PlayerID())
	targets := DeclareXPermanentTargets("subjects", IsCreature)
	body := CollectionDamageInstruction{This, PermanentDamageRecipients(FilterPermanentSet(targets, HasKeywordFilter(core.Flying))), Fixed(2)}.Effect()
	card := NewSorcery("Plural", "{X}", Spell(body, WithTargets(targets)))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Colorless, count)
	ids := []uuid.UUID{first.ID(), second.ID()}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids[:count], count); err != nil {
		t.Fatal(err)
	}
	return g, first, second, g.StackPeek(), targets
}

func TestPluralTargetQuery(t *testing.T) {
	for _, state := range []string{"normal", "zero", "phase", "cycle", "return", "shroud", "hexproof", "protection", "type", "control", "clone", "copy", "copy hexproof"} {
		t.Run(state, func(t *testing.T) {
			count := 2
			if state == "zero" {
				count = 0
			}
			g, p, q, obj, binding := pluralQueryFixture(t, count)
			plan, ok := InspectPermanentTargets(binding)
			if !ok {
				t.Fatal("native plural declaration rejected")
			}
			want := count
			switch state {
			case "phase":
				g.PhaseOut(p.ID())
				want--
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, p.ControllerID())
				want--
			case "shroud":
				p.GrantBaseAttr(core.Shroud)
				want--
			case "hexproof":
				p.GrantBaseAttr(core.Hexproof)
				want--
			case "protection":
				p.RuntimeAbilities = append(p.RuntimeAbilities, ProtectionFromColor(core.Green))
				colors := []core.Color{core.Green}
				obj.ColorOverride = &colors
				want--
			case "type":
				p.Card = NewLand("Changed")
				want--
			case "control":
				p.GrantBaseAttr(core.Hexproof)
				p.computedController = obj.Controller
			case "clone":
				g = g.Clone()
				obj = g.StackPeek()
			case "copy":
				obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
			case "copy hexproof":
				p.GrantBaseAttr(core.Hexproof)
				obj = g.CopyStackObjectDirect(obj, p.ControllerID(), false)
			}
			rng := g.random.Clone()
			got, known := plan.Select(g, obj)
			if !known || len(got) != want {
				t.Fatalf("got=%v/%v want %d", got, known, want)
			}
			if len(got) > 0 {
				got[0] = PermanentVisit{}
				again, _ := plan.Select(g, obj)
				if again[0].ID == uuid.Nil {
					t.Fatal("result aliases query")
				}
			}
			if !reflect.DeepEqual(rng, g.random) || q.Damage != 0 || q.Tapped {
				t.Fatal("query changed state")
			}
		})
	}
}

func TestPluralTargetQueryUnknown(t *testing.T) {
	trap := NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("filter callback") })
	for _, binding := range []*PermanentTargetsBinding{nil, {}, DeclareXPermanentTargets("opaque", trap), DeclareXPermanentTargets("partial", Or(IsCreature, trap)), DeclareUpToXPermanentTargets("optional", IsCreature)} {
		if _, ok := InspectPermanentTargets(binding); ok {
			t.Fatal("unsupported declaration accepted")
		}
	}
	for _, state := range []string{"missing visit", "missing zone", "duplicate identity", "wrong count", "missing declaration", "foreign declaration", "opaque suffix", "opaque all illegal"} {
		t.Run(state, func(t *testing.T) {
			g, p, _, obj, binding := pluralQueryFixture(t, 2)
			plan, _ := InspectPermanentTargets(binding)
			switch state {
			case "missing visit":
				delete(obj.targetIncarnations, p.ID())
			case "missing zone":
				delete(obj.TargetZones, p.ID())
			case "duplicate identity":
				obj.Targets[1] = obj.Targets[0]
			case "wrong count":
				obj.XValue = 3
			case "missing declaration":
				obj.TargetSpecs = nil
			case "foreign declaration":
				foreign := DeclareXPermanentTargets("subjects", IsCreature)
				plan, _ = InspectPermanentTargets(foreign)
				obj.Effects = []Effect{DealDamageEach(This, PermanentDamageRecipients(foreign), Fixed(1))}
			case "opaque suffix", "opaque all illegal":
				obj.Effects = append(obj.Effects, DealDamageEach(This, PermanentDamageRecipients(collectionQueryTrap{}), Fixed(1)))
				if state == "opaque all illegal" {
					for _, id := range obj.Targets {
						g.FindPermanent(id).GrantBaseAttr(core.Shroud)
					}
				}
			}
			if state != "opaque suffix" && state != "opaque all illegal" {
				if _, known := plan.Select(g, obj); known {
					t.Fatal("incomplete group known")
				}
			}
			if certain, known := g.InspectSpellDestruction(obj, nil); certain || known {
				t.Fatal("incomplete spell tree known")
			}
		})
	}
}

func TestPluralTargetQuerySpell(t *testing.T) {
	for _, state := range []string{"only", "prefix", "suffix", "empty prefix", "all illegal", "missing binding", "fixed", "fixed zero"} {
		t.Run(state, func(t *testing.T) {
			g, p, q, obj, binding := pluralQueryFixture(t, 2)
			visit := PermanentVisit{p.ID(), p.incarnationID}
			known, certain := true, false
			destroy := DestroyEach(binding)
			damage := obj.Effects[0]
			switch state {
			case "prefix":
				obj.Effects = []Effect{Sequence(damage, destroy)}
				known = false
			case "suffix":
				obj.Effects = []Effect{Sequence(destroy, damage)}
				certain = true
			case "empty prefix":
				p.RevokeBaseAttr(core.Flying)
				obj.Effects = []Effect{Sequence(damage, destroy)}
				certain = true
			case "all illegal":
				p.GrantBaseAttr(core.Shroud)
				q.GrantBaseAttr(core.Shroud)
			case "missing binding":
				other := DeclareXPermanentTargets("subjects", IsCreature)
				obj.Effects = []Effect{DealDamageEach(This, PermanentDamageRecipients(other), Fixed(1))}
				known = false
			case "fixed":
				binding.spec = &PermanentTarget{BaseTarget: BaseTarget{min: 2, max: 2}, Filters: []PermanentFilter{IsCreature}}
			case "fixed zero":
				binding.spec = &PermanentTarget{BaseTarget: BaseTarget{min: 0, max: 0}, Filters: []PermanentFilter{IsCreature}}
				obj.Targets = nil
				obj.TargetSpecs = nil
			}
			rng := g.random.Clone()
			for range 2 {
				if got, k := g.InspectSpellDestruction(obj, []PermanentVisit{visit}); got != certain || k != known {
					t.Fatalf("got=%v/%v want=%v/%v", got, k, certain, known)
				}
			}
			if p.Damage != 0 || !reflect.DeepEqual(rng, g.random) {
				t.Fatal("query changed state")
			}
		})
	}
}

func TestPluralTargetQueryCollection(t *testing.T) {
	g, p, q, obj, binding := pluralQueryFixture(t, 2)
	filter := HasKeywordFilter(core.Flying)
	filter.fn = func(*Permanent, *Game) bool { panic("filter callback") }
	plan, ok := InspectContextualPermanentCollection(FilterPermanentSet(binding, filter))
	if !ok {
		t.Fatal("bound collection rejected")
	}
	ctx := &EffectContext{Game: g, Controller: obj.Controller}
	if _, known := plan.selectKnown(ctx); known {
		t.Fatal("missing binding supplied empty proof")
	}
	ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{binding.targetDeclaration: {{p.ID(), p.incarnationID}, {q.ID(), q.incarnationID}}}
	got, known := plan.selectKnown(ctx)
	if !known || len(got) != 1 || got[0].ID != p.ID() {
		t.Fatal("bound flying collection differs")
	}
	ctx.targetBindings[binding.targetDeclaration] = nil
	if got, known := plan.selectKnown(ctx); !known || len(got) != 0 {
		t.Fatal("declared empty group unknown")
	}
}

func TestPluralTargetQueryGroups(t *testing.T) {
	for _, state := range []string{"normal", "copy", "clone", "missing frame", "unknown suffix", "unknown all illegal", "zero copy"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			p := g.PutOnBattlefield(NewCreature("Shared", "", 1, 8), b.PlayerID())
			q := g.PutOnBattlefield(NewCreature("Other", "", 1, 8), a.PlayerID())
			first := DeclareXPermanentTargets("first", IsCreature)
			second := DeclarePermanentTargets("second", 1, 1, IsCreature, TargetYouControl())
			body := Sequence(DestroyEach(first), DealDamageEach(This, PermanentDamageRecipients(second), Fixed(2)))
			if state == "missing frame" || state == "unknown all illegal" {
				foreign := DeclareXPermanentTargets("first", IsCreature)
				body = Sequence(DestroyEach(first), DealDamageEach(This, PermanentDamageRecipients(foreign), Fixed(2)))
			}
			if state == "unknown suffix" {
				body = Sequence(body, DealDamageEach(This, PermanentDamageRecipients(collectionQueryTrap{}), Fixed(2)))
			}
			card := NewSorcery("Groups", "{X}", Spell(body, WithTargetGroups(first, second)))
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Colorless, 2)
			x := 2
			ids := []uuid.UUID{p.ID(), q.ID()}
			if state == "zero copy" {
				x = 0
				ids = nil
			}
			if err := g.CastSpellByNameWithTargetGroups(a.PlayerID(), card.Name(), [][]uuid.UUID{ids, {q.ID()}}, x); err != nil {
				t.Fatal(err)
			}
			obj := g.StackPeek()
			if state == "copy" || state == "zero copy" {
				obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
			}
			if state == "clone" {
				g = g.Clone()
				obj = g.StackPeek()
				g.PhaseOut(q.ID())
			}
			if state == "unknown all illegal" {
				p.GrantBaseAttr(core.Shroud)
				q.GrantBaseAttr(core.Shroud)
			}
			known := state != "missing frame" && state != "unknown suffix" && state != "unknown all illegal"
			want := known && x != 0
			for range 2 {
				if certain, k := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}}); certain != want || k != known {
					t.Fatalf("got=%v/%v want=%v/%v", certain, k, want, known)
				}
			}
			if p.Damage != 0 || q.Damage != 0 {
				t.Fatal("query executed an instruction")
			}
		})
	}
}
