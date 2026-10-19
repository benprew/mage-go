package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronOptionalCastPlayer struct {
	*mage.BasePlayer
	answers []bool
	offers  int
}

func (p *ronOptionalCastPlayer) ChooseMayAbility(string) bool {
	p.offers++
	if len(p.answers) == 0 {
		return false
	}
	answer := p.answers[0]
	p.answers = p.answers[1:]
	return answer
}
func (p *ronOptionalCastPlayer) ChooseCardFromHand(cards []mage.Card, _ string, _ mage.GameReader) mage.Card {
	return cards[0]
}

func TestRONOptionalFaceDownCast(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		for _, scenario := range []string{"choose-and-cast", "decline-choice", "decline-cast", "source-left", "source-phased", "source-control", "copy", "clone"} {
			t.Run(set+"/"+scenario, func(t *testing.T) {
				record := repositoryRecord(t, set, "Illusionary Mask", set+".ron")
				rule := record.Fields["Rules"].Args[0]
				branch := &rule.Args[1].Args[0].Args[1].Args[1]
				cast := branch.Args[0]
				// This projection keeps the two decisions and omits the unresolved replacement.
				if cast.Name != "MayAction" {
					t.Fatal("source lacks the independent optional cast")
				}
				branch.Args = []Node{cast, {Kind: Tuple, Name: "GainLife", Args: []Node{{Kind: Tuple, Name: "Integer", Args: []Node{{Kind: IntegerNode, Integer: 1}}}}}}
				record.Fields["Rules"] = Node{Kind: List, Args: []Node{rule}}
				factory, err := Compile(record)
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				answers := []bool{true, true}
				if scenario == "decline-choice" {
					answers = []bool{false}
				}
				if scenario == "decline-cast" {
					answers = []bool{true, false}
				}
				chooser := &ronOptionalCastPlayer{BasePlayer: a, answers: append([]bool(nil), answers...)}
				g.SetPlayerAt(0, chooser)
				source := g.PutOnBattlefield(factory(), a.PlayerID())
				g.SetStep(core.PrecombatMain)
				creature := mage.NewCreature("Private", "{U}", 5, 5)
				a.AddToHand(creature)
				a.ManaPool().Add(core.Blue, 1)
				g.SetXValue(1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if chooser.offers != 0 {
					t.Fatal("choice before resolution")
				}
				switch scenario {
				case "source-left":
					g.RemoveFromBattlefield(source)
				case "source-phased":
					g.PhaseOut(source.ID())
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					chooser.answers = []bool{true, false, true, true}
				}
				if scenario == "clone" {
					cloned := g.Clone()
					cloned.SetPlayerAt(0, &ronOptionalCastPlayer{BasePlayer: cloned.GetPlayer(a.PlayerID()).(*mage.SearchPlayer).BasePlayer, answers: []bool{true, false}})
					cloned.ResolveStack()
					if len(a.Hand()) != 1 || chooser.offers != 0 || cloned.GetPlayer(a.PlayerID()).Life() != 21 || cloned.StackSize() != 0 {
						t.Fatal("clone decision leaked")
					}
				}
				g.ResolveStack()
				wantOffers := 2
				wantLife := 21
				if scenario == "decline-choice" {
					wantOffers = 1
					wantLife = 20
				}
				if scenario == "copy" {
					wantOffers = 4
					wantLife = 22
				}
				if chooser.offers != wantOffers || a.Life() != wantLife || b.Life() != 20 {
					t.Fatalf("offers=%d life=%d/%d", chooser.offers, a.Life(), b.Life())
				}
				casted := scenario != "decline-choice" && scenario != "decline-cast"
				p := g.FindPermanent(creature.ID())
				if (p != nil) != casted || casted && (!p.FaceDown || p.CurrentPower(g) != 2) {
					t.Fatal("incorrect optional cast outcome")
				}
				if !casted && len(a.Hand()) != 1 {
					t.Fatal("decline moved card")
				}
				if len(g.PublicReveals()) != 0 {
					t.Fatal("revealed private card")
				}
			})
		}
	}
}

func TestRONOptionalFaceDownCastInvalid(t *testing.T) {
	action := `CastCardInHandWithoutPayingAsAFacedownCreatureSpell(TheCardInHandChosenThisWay,PT(2,2))`
	for _, body := range []string{`MayAction(` + action + `,` + action + `)`, `MayAction(CastCardInHandWithoutPayingAsAFacedownCreatureSpell(TheCardInHandChosenThisWay,PT(3,3)))`, `MayAction(CastCardInHandWithoutPayingAsAFacedownCreatureSpell(TheCardInHandChosenThisWay(),PT(2,2)))`, `MayAction(CastCardInHandWithoutPayingAsAFacedownCreatureSpell(TheCardInHandChosenThisWay,PT(2,2),PT(2,2)))`} {
		rule := `Activated(PayMana([]),ActionList([MayCost(ChooseACardInHand(AnyCard)),If(CostWasPaid,[` + body + `])]))`
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)); err == nil {
			t.Fatal("accepted invalid optional cast", body)
		}
	}
	for _, body := range []string{`MayAction(` + action + `)`, `MayCost(ChooseACardInHand(AnyCard)),IfElse(CostWasPaid,[GainLife(Integer(1))],[MayAction(` + action + `)])`, `MayCost(ChooseACardInHand(AnyCard)),If(CostWasPaid,[GainLife(Integer(1))]),MayAction(` + action + `)`} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`)); err == nil {
			t.Fatal("accepted unbound optional cast")
		}
	}
	record := repositoryRecord(t, "LEA", "Illusionary Mask", "LEA.ron")
	if _, err := Compile(record); err != nil {
		t.Fatalf("complete Mask record: %v", err)
	}
}
