package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONFaceUpReplacement(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		for _, scenario := range []string{"tap", "outgoing", "incoming", "assignment", "source-left", "phase", "clone", "return", "decline", "counter", "ability-copy", "pending-clone"} {
			t.Run(set+"/"+scenario, func(t *testing.T) {
				factory, err := Compile(repositoryRecord(t, set, "Illusionary Mask", set+".ron"))
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				answers := []bool{true, true}
				if scenario == "decline" {
					answers = []bool{true, false}
				}
				if scenario == "ability-copy" {
					answers = []bool{true, true, false}
				}
				g.SetPlayerAt(0, &ronOptionalCastPlayer{BasePlayer: a, answers: answers})
				source := g.PutOnBattlefield(factory(), a.PlayerID())
				g.SetStep(core.PrecombatMain)
				card := mage.NewCreature("Hidden", "{U}", 5, 6, mage.WithKeyword(core.Lifelink), mage.WithAbility(mage.WhenEvent(mage.BecomesTapped(mage.This), mage.GainLifeFor(mage.You, mage.Fixed(3)))))
				a.AddToHand(card)
				a.ManaPool().Add(core.Blue, 1)
				g.SetXValue(1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if scenario == "ability-copy" {
					g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
				}
				g.ResolveTopOfStack()
				if scenario == "decline" {
					if g.StackSize() != 0 || len(a.Hand()) != 1 {
						t.Fatal("decline changed the card")
					}
					return
				}
				if scenario == "counter" {
					g.CounterStackObject(g.StackPeek().ID)
					g.ResolveStack()
					if g.FindPermanent(card.ID()) != nil {
						t.Fatal("counter produced a permanent")
					}
					return
				}
				if scenario == "pending-clone" {
					clone := g.Clone()
					clone.ResolveStack()
					clone.TapPermanent(clone.FindPermanent(card.ID()))
					if clone.FindPermanent(card.ID()).FaceDown || g.FindPermanent(card.ID()) != nil {
						t.Fatal("pending clone lost replacement")
					}
				}
				g.ResolveStack()
				p := g.FindPermanent(card.ID())
				if p == nil || !p.FaceDown {
					t.Fatal("missing face-down entry")
				}
				switch scenario {
				case "outgoing":
					g.DealDamageToPlayer(b, 2, p.ID())
					if a.Life() != 22 {
						t.Fatal("damage did not use face-up lifelink")
					}
				case "incoming":
					enemy := g.PutOnBattlefield(mage.NewCreature("Enemy", "", 1, 1), b.PlayerID())
					g.DealDamageToPermanent(p, 2, enemy.ID())
					if g.FindPermanent(p.ID()).Damage != 2 {
						t.Fatal("incoming damage missing")
					}
				case "assignment":
					g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
					g.ExecuteCombatDamage()
					if b.Life() != 15 || a.Life() != 25 {
						t.Fatalf("assignment used hidden power: %d/%d", a.Life(), b.Life())
					}
				case "source-left":
					g.BouncePermanentToHand(source)
					g.TapPermanent(p)
				case "phase":
					ids := g.PhaseOut(p.ID())
					g.TapPermanent(p)
					if !g.FindPermanentIncludingPhased(p.ID()).FaceDown {
						t.Fatal("phased subject changed")
					}
					g.PhaseIn(ids)
					g.TapPermanent(g.FindPermanent(p.ID()))
				case "clone":
					clone := g.Clone()
					clone.TapPermanent(clone.FindPermanent(p.ID()))
					if clone.FindPermanent(p.ID()).FaceDown || !g.FindPermanent(p.ID()).FaceDown {
						t.Fatal("clone isolation")
					}
					g.TapPermanent(p)
				case "return":
					g.BouncePermanentToHand(p)
					a.RemoveFromHand(card.ID())
					returned := g.PutOnBattlefield(card, a.PlayerID())
					returned.FaceDown = true
					g.TapPermanent(returned)
					if !g.FindPermanent(card.ID()).FaceDown {
						t.Fatal("returned object inherited replacement")
					}
					return
				default:
					g.TapPermanent(p)
				}
				if g.FindPermanent(card.ID()).FaceDown {
					t.Fatal("replacement did not turn subject face up")
				}
				if scenario == "tap" {
					g.PutTriggersOnStack()
					g.ResolveStack()
					if a.Life() != 23 {
						t.Fatal("tap event preceded face-up ability")
					}
				}
			})
		}
	}
}

func TestRONFaceUpReplacementInvalid(t *testing.T) {
	const event = `PermanentWouldAssignDamageDealDamageBeDealDamageOrBecomeTapped(ThePermanentSpellResolvedThisWay)`
	const duration = `UntilPermanentIsTurnedFaceUp(ThePermanentSpellResolvedThisWay)`
	for _, action := range []string{
		`CreateReplaceWouldMaskUntil()`,
		`CreateReplaceWouldMaskUntil(` + event + `,[],` + duration + `)`,
		`CreateReplaceWouldMaskUntil(` + event + `,[TurnItFaceUpInstead()],` + duration + `)`,
		`CreateReplaceWouldMaskUntil(` + event + `,[TurnItFaceUpInstead,TurnItFaceUpInstead],` + duration + `)`,
		`CreateReplaceWouldMaskUntil(` + event + `,[TurnItFaceUpInstead],UntilEndOfTurn)`,
		`CreateReplaceWouldMaskUntil(PermanentWouldAssignDamageDealDamageBeDealDamageOrBecomeTapped(ThisPermanent),[TurnItFaceUpInstead],` + duration + `)`,
		`CreateReplaceWouldMaskUntil(` + event + `,[TurnItFaceUpInstead],UntilPermanentIsTurnedFaceUp(ThisPermanent))`,
		`CreateReplaceWouldMaskUntil(PermanentWouldAssignDamageDealDamageBeDealDamageOrBecomeTapped(ThePermanentSpellResolvedThisWay()),[TurnItFaceUpInstead],` + duration + `)`,
	} {
		body := `MayCost(ChooseACardInHand(AnyCard)),If(CostWasPaid,[` + castResultAction + `,` + action + `])`
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`)
		if _, err := Compile(node); err == nil {
			t.Fatal("accepted malformed replacement", action)
		}
	}
	valid := `CreateReplaceWouldMaskUntil(` + event + `,[TurnItFaceUpInstead],` + duration + `)`
	for _, body := range []string{valid, valid + `,` + castResultAction, `MayCost(ChooseACardInHand(AnyCard)),If(CostWasPaid,[` + castResultAction + `]),` + valid} {
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`)
		if _, err := Compile(node); err == nil {
			t.Fatal("accepted escaped result", body)
		}
	}
}
