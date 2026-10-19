package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTargetCharacteristicValueCard(t *testing.T) {
	for _, state := range []string{"normal", "indestructible", "regeneration", "phase", "phase cycle", "returned", "shroud", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			a := mage.NewBasePlayer("A")
			b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B")}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			p := g.PutOnBattlefield(mage.NewArtifact("subject", "{4}"), b.PlayerID())
			q := g.PutOnBattlefield(mage.NewArtifact("copy target", "{2}"), a.PlayerID())
			card := repositoryCard(t, "LEG", "Divine Offering")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("unpaid cast changed state")
			}
			a.ManaPool().Add(core.White, 1)
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			gain, receiver, proof := 4, a.PlayerID(), true
			switch state {
			case "indestructible":
				p.GrantBaseAttr(core.Indestructible)
				proof = false
			case "regeneration":
				g.AddRegenerationShield(p.ID())
				proof = false
			case "phase":
				g.PhaseOut(p.ID())
				gain = 0
				proof = false
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "returned":
				g.BouncePermanentToHand(p)
				g.PutOnBattlefield(p.Card, b.PlayerID())
				gain = 0
				proof = false
			case "shroud":
				p.GrantBaseAttr(core.Shroud)
				gain = 0
				proof = false
			case "copy":
				b.targets = []uuid.UUID{q.ID()}
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true) == nil {
					t.Fatal("copy failed")
				}
				gain = 2
				receiver = b.PlayerID()
				p = q
			}
			predicate, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
			watched := predicate.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
			for _, game := range []*mage.Game{g, g.Clone()} {
				for range 2 {
					if got, known := game.InspectSpellDestruction(game.StackPeek(), watched); got != proof || !known {
						t.Fatalf("query=%v/%v want=%v/true", got, known, proof)
					}
				}
				if game.GetPlayer(receiver).Life() != 20 {
					t.Fatal("query gained life")
				}
				if state == "regeneration" && !game.HasRegenerationShield(p.ID()) {
					t.Fatal("query used shield")
				}
				if game != g {
					game.ResolveTopOfStack()
					if game.GetPlayer(receiver).Life() != 20+gain || g.GetPlayer(receiver).Life() != 20 {
						t.Fatal("clone crossed state")
					}
				}
			}
			g.ResolveTopOfStack()
			if g.GetPlayer(receiver).Life() != 20+gain {
				t.Fatal("wrong life gain")
			}
			removed := state == "normal" || state == "phase cycle" || state == "copy" || state == "clone"
			if (g.FindPermanentIncludingPhased(p.ID()) == nil) != removed {
				t.Fatal("wrong destruction result")
			}
		})
	}
}

func TestRONTargetCharacteristicValueInvalid(t *testing.T) {
	for _, value := range []string{`ManaValueOfPermanent()`, `ManaValueOfPermanent(Ref_TargetPlayer)`, `PowerOfPermanent(Ref_TargetPermanent())`, `ManaValueOfPermanent(Ref_TargetPermanent,Integer(1))`, `PowerOfPermanent(Ref_TargetPermanent2)`} {
		source := `Card(Name:"Invalid value",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([GainLife(` + value + `)])))])`
		if _, err := Compile(nameScopeNode(t, source)); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
}

func TestRONTargetCharacteristicValueAdapter(t *testing.T) {
	if _, err := compilePermanentInformationValue(nameScopeNode(t, `UnknownValue(ThisPermanent)`), scope{permanent: true}); err == nil {
		t.Fatal("unknown constructor accepted")
	}
}

func TestRONTargetCharacteristicValueBindings(t *testing.T) {
	for _, constructor := range []string{"PowerOfPermanent", "ManaValueOfPermanent"} {
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([GainLife(`+constructor+`(Ref_TargetPermanent))])))`)
		g, a, b := ronGame()
		p := g.PutOnBattlefield(mage.NewCreature("first", "{3}", 2, 5), b.PlayerID())
		q := g.PutOnBattlefield(mage.NewCreature("second", "{5}", 4, 5), a.PlayerID())
		one := castRONSpell(t, g, a, factory(), p.ID())
		two := castRONSpell(t, g, a, factory(), q.ID())
		instructions, ok := mage.InspectLifeInstructions(mage.Sequence(one.Effects...))
		if !ok || len(instructions) != 1 {
			t.Fatal("lost native life")
		}
		if _, ok := mage.InspectTargetCharacteristicValue(instructions[0].Amount); !ok {
			t.Fatal("lost native amount")
		}
		original := two.Effects
		two.Effects = one.Effects
		if _, known := g.InspectSpellDestruction(two, nil); known {
			t.Fatal("foreign factory target accepted")
		}
		two.Effects = original
		for _, obj := range []*mage.StackObject{one, two} {
			if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
				t.Fatal("instance query failed")
			}
		}
		clone := g.Clone()
		clone.ResolveStack()
		want := 26
		if constructor == "ManaValueOfPermanent" {
			want = 28
		}
		if clone.GetPlayer(a.PlayerID()).Life() != want || a.Life() != 20 {
			t.Fatal("clone state crossed")
		}
		g.ResolveStack()
		if a.Life() != want || b.Life() != 20 {
			t.Fatal("wrong target value or player")
		}
	}
}
