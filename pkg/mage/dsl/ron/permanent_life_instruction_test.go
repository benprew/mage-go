package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPermanentLifeInstructionRealCards(t *testing.T) {
	for _, name := range []string{"Crumble", "Swords to Plowshares"} {
		sets, color := []string{"ATQ", "3ED"}, core.Green
		if name != "Crumble" {
			sets, color = []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"}, core.White
		}
		for _, set := range sets {
			for _, state := range []string{"normal", "control", "power", "negative", "indestructible", "regeneration", "phase", "phase cycle", "returned", "shroud", "copy", "retarget", "unpaid", "replacement"} {
				t.Run(name+"/"+set+"/"+state, func(t *testing.T) {
					a := mage.NewBasePlayer("A")
					b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B")}
					g := mage.NewGame(a, b)
					g.SetStep(core.PrecombatMain)
					g.SetTurn(1)
					makeSubject := func(label string) *mage.Permanent {
						c := mage.NewCreature(label, "{3}{G}", 3, 8, mage.WithCardType(core.TypeArtifact))
						c.SetOwner(a.PlayerID())
						return g.PutOnBattlefield(c, b.PlayerID())
					}
					p, q := makeSubject("subject"), makeSubject("other")
					p.AddCounter(core.P1P1, 2)
					card := repositoryCard(t, set, name)
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					if state != "unpaid" {
						a.ManaPool().Add(color, 1)
					}
					err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0)
					if state == "unpaid" {
						if err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 || b.Life() != 20 || g.FindPermanent(p.ID()) == nil {
							t.Fatal("unpaid casting changed state")
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					obj := g.StackPeek()
					want, receiver, removed, proof := 5, b.PlayerID(), true, name == "Crumble"
					knownWant := true
					if name == "Crumble" {
						want = 4
					}
					switch state {
					case "control":
						g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
						receiver = a.PlayerID()
					case "power":
						p.AddCounter(core.P1P1, 3)
						if name != "Crumble" {
							want = 8
						}
					case "negative":
						if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(-9), mage.Fixed(0))), p.ID(), b.PlayerID(), nil); err != nil {
							t.Fatal(err)
						}
						if name != "Crumble" {
							want = 0
						}
					case "indestructible":
						p.GrantBaseAttr(core.Indestructible)
						if name == "Crumble" {
							removed, proof = false, false
						}
					case "regeneration":
						g.AddRegenerationShield(p.ID())
					case "phase":
						g.PhaseOut(p.ID())
						want, removed, proof = 0, false, false
					case "phase cycle":
						g.PhaseIn(g.PhaseOut(p.ID()))
					case "returned":
						g.BouncePermanentToHand(p)
						a.RemoveFromHand(p.Card.ID())
						g.PutOnBattlefield(p.Card, a.PlayerID())
						want, removed, proof = 0, false, false
					case "shroud":
						p.GrantBaseAttr(core.Shroud)
						want, removed, proof = 0, false, false
					case "copy", "retarget":
						b.targets = []uuid.UUID{q.ID()}
						obj = g.CopyStackObjectDirect(obj, b.PlayerID(), state == "retarget")
						if obj == nil {
							t.Fatal("copy failed")
						}
						if state == "retarget" {
							p = q
							if name != "Crumble" {
								want = 3
							}
						}
					case "replacement":
						source := g.PutOnBattlefield(mage.NewArtifact("Double", ""), a.PlayerID())
						g.AddReplacementEffect(&informationLifeReplacement{source: source.ID(), double: true})
						want *= 2
						if name == "Crumble" {
							knownWant = false
							proof = false
						}
					}
					instructions, ok := mage.InspectLifeInstructions(mage.Sequence(obj.Effects...))
					if ok || len(instructions) != 0 {
						t.Fatal("mixed tree admitted as life-only")
					}
					query, _ := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
					visits := query.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
					for range 2 {
						certain, known := g.InspectSpellDestruction(obj, visits)
						if certain != proof || known != knownWant {
							t.Fatalf("proof=%v/%v want=%v/%v", certain, known, proof, knownWant)
						}
					}
					if a.Life() != 20 || b.Life() != 20 {
						t.Fatal("query gained life")
					}
					if state == "regeneration" && !g.HasRegenerationShield(p.ID()) {
						t.Fatal("query consumed shield")
					}
					clone := g.Clone()
					clone.ResolveTopOfStack()
					if clone.GetPlayer(receiver).Life() != 20+want || a.Life() != 20 || b.Life() != 20 {
						t.Fatal("clone life state crossed")
					}
					g.ResolveTopOfStack()
					if g.GetPlayer(receiver).Life() != 20+want || (g.FindPermanentIncludingPhased(p.ID()) == nil) != removed {
						t.Fatalf("life=%d want=%d removed=%v", g.GetPlayer(receiver).Life(), 20+want, removed)
					}
				})
			}
		}
	}
}

func TestRONPermanentLifeInstructionInvalid(t *testing.T) {
	for _, body := range []string{
		`PlayerAction(ControllerOfPermanent(Ref_TargetPermanent),GainLife(PowerOfPermanent(Ref_TargetPermanent_2)))`,
		`PlayerAction(ControllerOfPermanent(Ref_TargetPlayer),GainLife(PowerOfPermanent(Ref_TargetPermanent)))`,
		`PlayerAction(ControllerOfPermanent(Ref_TargetPermanent()),GainLife(PowerOfPermanent(Ref_TargetPermanent)))`,
		`PlayerAction(ControllerOfPermanent(Ref_TargetPermanent),GainLife(ManaValueOfPermanent()))`,
		`PlayerAction(ControllerOfPermanent(Ref_TargetPermanent),GainLife(PowerOfPermanent(Ref_TargetPermanent,Integer(1))))`,
	} {
		source := `Card(Name:"Invalid information life",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([` + body + `])))])`
		if _, err := Compile(nameScopeNode(t, source)); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONPermanentLifeInstructionBindings(t *testing.T) {
	for _, value := range []string{"PowerOfPermanent", "ManaValueOfPermanent"} {
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([PlayerAction(ControllerOfPermanent(Ref_TargetPermanent),GainLife(`+value+`(Ref_TargetPermanent)))])))`)
		first, second := factory(), factory()
		inspect := func(obj *mage.StackObject) mage.PermanentLifeInstruction {
			t.Helper()
			instruction, ok := mage.InspectPermanentLifeInstruction(mage.Sequence(obj.Effects...))
			if !ok || instruction.ManaValue != (value == "ManaValueOfPermanent") {
				t.Fatal("lost native information life declaration")
			}
			return instruction
		}
		g, a, b := ronGame()
		p := g.PutOnBattlefield(mage.NewCreature("first", "{3}", 2, 5), b.PlayerID())
		q := g.PutOnBattlefield(mage.NewCreature("second", "{5}", 4, 5), a.PlayerID())
		one, two := inspect(castRONSpell(t, g, a, first, p.ID())), inspect(castRONSpell(t, g, a, second, q.ID()))
		if one.Subject == two.Subject {
			t.Fatal("factories share target bindings")
		}
		g.ResolveStack()
		wantA, wantB := 24, 22
		if value == "ManaValueOfPermanent" {
			wantA, wantB = 25, 23
		}
		if a.Life() != wantA || b.Life() != wantB {
			t.Fatal("target instance life reads crossed")
		}
	}
}
