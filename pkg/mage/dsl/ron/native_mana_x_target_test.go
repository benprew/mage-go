package ron

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNativeManaXTargetLowering(t *testing.T) {
	for _, set := range []string{"ATQ", "LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		name := "Spell Blast"
		if set == "ATQ" {
			name = "Detonate"
		}
		record := repositoryRecord(t, set, name, "native-x.ron")
		target := record.Fields["Rules"].Args[0].Args[0].Args[0].Args[0]
		if name == "Detonate" {
			filter, options, err := compilePermanentTargetFilter(target.Args[0], scope{x: true})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := mage.InspectPermanentTarget(mage.DeclarePermanentTarget("native", filter, options...)); !ok {
				t.Fatal("full record target opaque")
			}
		} else {
			build, _, err := compileSpellTarget(target.Args[0], scope{x: true})
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := mage.InspectSpellTarget(build(instanceBindings{}).(*mage.SpellTargetBinding)); !ok {
				t.Fatal("full record target opaque")
			}
		}
	}
}

func TestRONNativeManaXTargetDetonate(t *testing.T) {
	for _, state := range []string{"normal", "zero", "mana change", "shroud", "phase", "phase cycle", "return", "control", "copy", "clone", "regeneration", "indestructible", "prevention"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			x := 2
			if state == "zero" {
				x = 0
			}
			card := mage.NewArtifact("Artifact land", fmt.Sprintf("{%d}", x), mage.WithCardType(core.TypeLand))
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			spell := repositoryCard(t, "ATQ", "Detonate")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.Red, 1)
			a.ManaPool().Add(core.Colorless, x)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, x); err != nil {
				t.Fatal(err)
			}
			obj := g.StackPeek()
			predicate, ok := mage.InspectPermanentTargetDeclaration(obj.TargetSpecs[0])
			if !ok {
				t.Fatal("native full-record target rejected")
			}
			watched, known := predicate.Select(g, obj)
			if !known || len(watched) != 1 {
				t.Fatal("initial target visit missing")
			}
			legal := true
			switch state {
			case "mana change":
				g.MutablePermanent(p.ID()).Card = mage.NewArtifact("Changed", "{3}")
				legal = false
			case "shroud":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
				legal = false
			case "phase":
				g.PhaseOut(p.ID())
				legal = false
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(card, b.PlayerID())
				legal = false
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "copy":
				obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
			case "regeneration":
				g.AddRegenerationShield(p.ID())
			case "indestructible":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Indestructible)
			case "prevention":
				g.AddPreventionShield(b.PlayerID(), 1)
			}
			g.ApplyContinuousEffects()
			for range 2 {
				visits, known := predicate.Select(g, obj)
				if !known || (len(visits) == 1) != legal {
					t.Fatal("target query differs from legality")
				}
			}
			clone := g.Clone()
			if state == "clone" {
				clone.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
				clone.ApplyContinuousEffects()
				if visits, k := predicate.Select(clone, clone.StackPeek()); !k || len(visits) != 0 {
					t.Fatal("clone target query read original")
				}
				if certain, k := clone.InspectSpellDestruction(clone.StackPeek(), watched); !k || certain {
					t.Fatal("clone destruction query read original")
				}
				clone.ResolveStack()
				if clone.GetPlayer(b.PlayerID()).Life() != 20 || clone.FindPermanent(p.ID()) == nil {
					t.Fatal("clone resolved an illegal target")
				}
			}
			for range 2 {
				certain, known := g.InspectSpellDestruction(obj, watched)
				if !known || certain != (legal && state != "indestructible") {
					t.Fatalf("complete Detonate query = %v/%v", certain, known)
				}
			}
			g.ResolveStack()
			if state == "phase" {
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			if (g.FindPermanent(p.ID()) == nil) != (legal && state != "indestructible") {
				t.Fatal("wrong native destruction outcome")
			}
			controller := b.PlayerID()
			if state == "control" {
				controller = a.PlayerID()
			}
			wantLife := 20
			if legal {
				wantLife -= x
			}
			if state == "prevention" {
				wantLife++
			}
			if g.GetPlayer(controller).Life() != wantLife {
				t.Fatal("wrong damage after legality recheck")
			}
		})
	}
}

func TestRONNativeManaXTargetSpellBlast(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, state := range []string{"normal", "zero", "left", "recast", "mana change", "copy", "clone"} {
			t.Run(set+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				x, enemyX := 5, 2
				cost := "{X}{X}{U}"
				if state == "zero" {
					x, enemyX, cost = 0, 0, "{X}"
				}
				victim := mage.NewInstant("Variable", cost, mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1))))
				victim.SetOwner(b.PlayerID())
				b.AddToHand(victim)
				b.ManaPool().Add(core.Blue, 1)
				b.ManaPool().Add(core.Colorless, 10)
				if err := g.CastSpellByID(b.PlayerID(), victim.ID(), nil, enemyX); err != nil {
					t.Fatal(err)
				}
				enemy := g.StackPeek()
				card := repositoryCard(t, set, "Spell Blast")
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(core.Blue, 1)
				a.ManaPool().Add(core.Colorless, x)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{enemy.ID}, x); err != nil {
					t.Fatal(err)
				}
				obj := g.StackPeek()
				parts, ok := mage.InspectCounterInstructions(obj.Effects[0])
				if !ok || len(parts) != 1 {
					t.Fatal("counter declaration missing")
				}
				predicate, ok := mage.InspectSpellTarget(parts[0].Subject)
				if !ok {
					t.Fatal("native spell target rejected")
				}
				legal := true
				switch state {
				case "left", "recast":
					g.CounterStackObject(enemy.ID)
					legal = false
					if state == "recast" {
						b.RemoveFromGraveyard(victim.ID())
						b.AddToHand(victim)
						b.ManaPool().Add(core.Blue, 1)
						if err := g.CastSpellByID(b.PlayerID(), victim.ID(), nil, enemyX); err != nil {
							t.Fatal(err)
						}
						g.ResolveTopOfStack()
					}
				case "mana change":
					enemy.Card = mage.NewInstant("Changed", "{U}")
					legal = false
				case "copy":
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
				}
				g.ApplyContinuousEffects()
				for range 2 {
					id, known := predicate.Select(g, obj)
					if !known || (id == enemy.ID) != legal {
						t.Fatal("native stack query differs")
					}
					if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
						t.Fatal("complete native spell tree rejected")
					}
				}
				clone := g.Clone()
				if state == "clone" {
					clone.GetStack().FindByID(enemy.ID).Card = mage.NewInstant("Changed", "{U}")
					if id, k := predicate.Select(clone, clone.StackPeek()); !k || id != uuid.Nil {
						t.Fatal("clone read original spell")
					}
				}
				g.ResolveTopOfStack()
				if (g.GetStack().FindByID(enemy.ID) == nil) != (legal || state == "left" || state == "recast") {
					t.Fatal("counter used wrong target or X")
				}
			})
		}
	}
}
