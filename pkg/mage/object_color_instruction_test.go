package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestObjectColorInstructionDeclaration(t *testing.T) {
	target := DeclareSpellOrPermanentTarget("object")
	effect := (ObjectColorInstruction{Subject: target, Color: core.Red}).Effect()
	got, ok := InspectObjectColorInstruction(effect)
	if !ok || got.Subject != target || got.Color != core.Red {
		t.Fatal("lost native operands")
	}
	for _, instruction := range []ObjectColorInstruction{{}, {Subject: target, Color: core.Colorless}, {Subject: target, Color: core.AnyColor}, {Subject: DeclareSpellOrPermanentTarget(""), Color: core.Red}} {
		if instruction.Effect() != nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	for _, effect := range []Effect{nil, (*objectTargetColorEffect)(nil), &objectTargetColorEffect{}, SetObjectColor(target, core.Colorless), SetObjectColor(nil, core.Red), Sequence(effect, destructionInspectionTrap{})} {
		if _, ok := InspectObjectColorInstruction(effect); ok {
			t.Fatal("unknown declaration accepted")
		}
	}
}

func TestObjectColorInstructionQuery(t *testing.T) {
	for _, category := range []string{"permanent", "spell"} {
		for _, state := range []string{"alone", "prefix", "suffix", "illegal prefix", "unknown suffix", "unknown illegal", "missing zone", "missing visit", "foreign", "duplicate", "clone", "copy", "phase", "cycle", "return", "gone", "recast", "ability", "wrong zone"} {
			t.Run(category+"/"+state, func(t *testing.T) {
				g, p, obj, land := targetQueryFixture(t)
				union := DeclareSpellOrPermanentTarget("color")
				id := p.ID()
				var spell *StackObject
				if category == "spell" {
					spell = &StackObject{ID: uuid.New(), Card: NewCreature("Spell", "{G}", 2, 2), Controller: obj.Controller, SourceID: uuid.New()}
					g.stack.Pop()
					g.pushStack(spell)
					g.stack.Push(obj)
					id = spell.ID
				}
				obj.Targets = append(obj.Targets, id)
				obj.TargetSpecs = append(obj.TargetSpecs, union.targetSpec())
				obj.TargetZones[id] = core.ZoneBattlefield
				obj.targetIncarnations[id] = p.incarnationID
				if category == "spell" {
					obj.TargetZones[id] = core.ZoneStack
					delete(obj.targetIncarnations, id)
				}
				effect := SetObjectColor(union, core.Red)
				obj.Effects = []Effect{effect}
				want, known := false, true
				switch state {
				case "prefix":
					obj.Effects = []Effect{Sequence(effect, Destroy(land))}
					known = false
				case "suffix", "clone", "copy":
					obj.Effects = []Effect{Sequence(Destroy(land), effect)}
					want = true
				case "illegal prefix":
					obj.Effects = []Effect{Sequence(effect, DestroyEach(MatchingPermanents(IsLand)))}
					if category == "permanent" {
						p.GrantBaseAttr(core.Shroud)
					} else {
						g.CounterStackObject(id)
					}
					want = category == "spell"
				case "unknown suffix", "unknown illegal":
					obj.Effects = []Effect{Sequence(effect, destructionInspectionTrap{})}
					known = false
					if state == "unknown illegal" {
						p.GrantBaseAttr(core.Shroud)
						if spell != nil {
							g.CounterStackObject(id)
						}
					}
				case "missing zone":
					delete(obj.TargetZones, id)
					known = false
				case "missing visit":
					if category == "permanent" {
						delete(obj.targetIncarnations, id)
						known = false
					}
				case "foreign":
					obj.Effects = []Effect{SetObjectColor(DeclareSpellOrPermanentTarget("foreign"), core.Red)}
					known = false
				case "duplicate":
					obj.Targets = append(obj.Targets, id)
					obj.TargetSpecs = append(obj.TargetSpecs, union.targetSpec())
					known = false
				case "phase":
					g.PhaseOut(p.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "return":
					g.RemoveFromBattlefield(p)
					g.PutOnBattlefield(p.Card, p.ControllerID())
				case "gone", "recast":
					if spell != nil {
						g.CounterStackObject(id)
						if state == "recast" {
							replacement := &StackObject{ID: uuid.New(), Card: spell.Card, SourceID: spell.SourceID, Controller: obj.Controller}
							g.stack.Pop()
							g.pushStack(replacement)
							g.stack.Push(obj)
						}
					}
				case "ability":
					if spell != nil {
						spell.IsAbility = true
					}
				case "wrong zone":
					obj.TargetZones[id] = core.ZoneHand
					known = false
				}
				if state == "clone" {
					g = g.Clone()
					obj = g.StackPeek()
				}
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, obj.Controller, false)
				}
				before := len(g.effects.effects)
				for range 2 {
					got, k := g.InspectSpellDestruction(obj, []PermanentVisit{{p.ID(), p.incarnationID}})
					if got != want || k != known {
						t.Fatalf("got %v/%v want %v/%v", got, k, want, known)
					}
				}
				if len(g.effects.effects) != before || len(p.Colors()) != 0 {
					t.Fatal("query changed state")
				}
			})
		}
	}
}

func TestObjectColorInstructionTargetDeclarations(t *testing.T) {
	for _, binding := range []*SpellOrPermanentTargetBinding{nil, {}, DeclareSpellOrPermanentTarget(""), {&targetDeclaration{name: "foreign", spec: TargetPermanent()}}, {&targetDeclaration{name: "count", spec: &typedSpellOrPermanentTarget{BaseTarget: BaseTarget{min: 0, max: 1}}}}} {
		if _, ok := InspectObjectTarget(binding); ok {
			t.Fatal("incomplete target accepted")
		}
	}
	g, p, obj, _ := targetQueryFixture(t)
	target := DeclareSpellOrPermanentTarget("object")
	obj.TargetSpecs = []Target{target.targetSpec()}
	predicate, ok := InspectObjectTarget(target)
	if !ok {
		t.Fatal("native declaration rejected")
	}
	got, known := predicate.Select(g, obj)
	if !known || got.Permanent != (PermanentVisit{p.ID(), p.incarnationID}) || got.Spell != uuid.Nil {
		t.Fatal("lost permanent category or visit")
	}
	for _, state := range []string{"zero", "missing", "nil id", "wrong count", "opaque protection", "shroud", "protection"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, _ := targetQueryFixture(t)
			target := DeclareSpellOrPermanentTarget("object")
			obj.TargetSpecs = []Target{target.targetSpec()}
			query, _ := InspectObjectTarget(target)
			wantKnown := false
			switch state {
			case "zero":
				query = ObjectTargetPredicate{}
			case "missing":
				query, _ = InspectObjectTarget(DeclareSpellOrPermanentTarget("other"))
			case "nil id":
				obj.Targets[0] = uuid.Nil
			case "wrong count":
				obj.Targets = nil
			case "opaque protection":
				g.FindPermanent(p.ID()).RuntimeAbilities = append(g.FindPermanent(p.ID()).RuntimeAbilities, &ProtectionAbility{Filter: NewCardFilter("trap", func(Card) bool { panic("protection ran") })})
			case "shroud":
				g.FindPermanent(p.ID()).GrantBaseAttr(core.Shroud)
				wantKnown = true
			case "protection":
				g.FindPermanent(p.ID()).RuntimeAbilities = append(g.FindPermanent(p.ID()).RuntimeAbilities, ProtectionFromColor(core.Red))
				wantKnown = true
			}
			got, known := query.Select(g, obj)
			if known != wantKnown || !got.Empty() {
				t.Fatalf("selection %v/%v", got, known)
			}
		})
	}
}

func TestObjectColorInstructionSpellSelection(t *testing.T) {
	for _, state := range []string{"normal", "gone", "recast", "ability", "source id", "missing zone", "stack visit", "duplicate"} {
		t.Run(state, func(t *testing.T) {
			g, _, obj, _ := targetQueryFixture(t)
			spell := &StackObject{ID: uuid.New(), Card: NewCreature("Spell", "{G}", 2, 2), SourceID: uuid.New(), Controller: obj.Controller}
			g.stack.Pop()
			g.pushStack(spell)
			g.stack.Push(obj)
			target := DeclareSpellOrPermanentTarget("spell or permanent")
			obj.Targets = []uuid.UUID{spell.ID}
			obj.TargetSpecs = []Target{target.targetSpec()}
			obj.TargetZones = map[uuid.UUID]core.Zone{spell.ID: core.ZoneStack}
			obj.targetIncarnations = nil
			known, legal := true, true
			switch state {
			case "gone", "recast":
				g.CounterStackObject(spell.ID)
				legal = false
				if state == "recast" {
					replacement := &StackObject{ID: uuid.New(), Card: spell.Card, SourceID: spell.SourceID, Controller: obj.Controller}
					g.stack.Pop()
					g.pushStack(replacement)
					g.stack.Push(obj)
				}
			case "ability":
				spell.IsAbility = true
				legal = false
			case "source id":
				obj.Targets[0] = spell.SourceID
				obj.TargetZones[spell.SourceID] = core.ZoneStack
				legal = false
			case "missing zone":
				obj.TargetZones = nil
				known = false
				legal = false
			case "stack visit":
				obj.targetIncarnations = map[uuid.UUID]uuid.UUID{spell.ID: uuid.New()}
				known = false
				legal = false
			case "duplicate":
				obj.Targets = append(obj.Targets, spell.ID)
				obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
				known = false
				legal = false
			}
			predicate, ok := InspectObjectTarget(target)
			if !ok {
				t.Fatal("native target rejected")
			}
			got, k := predicate.Select(g, obj)
			if k != known || got.Empty() == legal || legal && got.Spell != spell.ID || got.Permanent.ID != uuid.Nil {
				t.Fatalf("selection %v/%v", got, k)
			}
		})
	}
}
