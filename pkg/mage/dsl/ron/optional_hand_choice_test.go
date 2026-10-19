package ron

import (
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronHandChoicePlayer struct {
	*mage.BasePlayer
	accept          bool
	offers, choices int
}

func (p *ronHandChoicePlayer) ChooseMayAbility(string) bool { p.offers++; return p.accept }
func (p *ronHandChoicePlayer) ChooseCardFromHand(cards []mage.Card, _ string, _ mage.GameReader) mage.Card {
	p.choices++
	return cards[len(cards)-1]
}

func TestRONOptionalHandChoice(t *testing.T) {
	for _, condition := range []string{"If(CostWasPaid", "Unless(CostWasntPaid"} {
		for _, scenario := range []string{"paid", "declined", "empty", "source-left", "source-return", "source-control", "source-phase", "copy", "clone", "failed-activation", "phased-activation"} {
			t.Run(condition+"/"+scenario, func(t *testing.T) {
				g, a, b := ronGame()
				chooser := &ronHandChoicePlayer{BasePlayer: a, accept: scenario != "declined"}
				g.SetPlayerAt(0, chooser)
				rule := `Activated(PayMana([ManaCostGeneric(1)]),ActionList([MayCost(ChooseACardInHand(IsCardtype(Creature))),` + condition + `,[GainLife(Integer(2))]),GainLife(Integer(1))]))`
				card := permanentScopeFactory(t, "Artifact", rule)()
				source := g.PutOnBattlefield(card, a.PlayerID())
				if scenario != "empty" {
					a.AddToHand(mage.NewCreature("Hand creature", "", 1, 1))
				}
				a.AddToHand(mage.NewLand("Not a creature"))
				if scenario != "failed-activation" {
					a.ManaPool().Add(core.Colorless, 1)
				}
				if scenario == "phased-activation" {
					g.PhaseOut(source.ID())
				}
				err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
				if scenario == "failed-activation" || scenario == "phased-activation" {
					if err == nil || chooser.offers != 0 || chooser.choices != 0 {
						t.Fatal("failed activation made a choice")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if chooser.offers != 0 || chooser.choices != 0 || a.Life() != 20 {
					t.Fatal("selection happened before resolution")
				}
				switch scenario {
				case "source-left":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				case "source-return":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					g.PutOnBattlefield(card, a.PlayerID())
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "source-phase":
					g.PhaseOut(source.ID())
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				clone.SetPlayerAt(0, &ronHandChoicePlayer{BasePlayer: clone.GetPlayer(a.PlayerID()).(*mage.SearchPlayer).BasePlayer, accept: scenario != "declined"})
				clone.ResolveStack()
				if a.Life() != 20 || chooser.offers != 0 {
					t.Fatal("clone changed original")
				}
				g.ResolveStack()
				want := 23
				if scenario == "declined" || scenario == "empty" {
					want = 21
				}
				if scenario == "copy" {
					want = 26
				}
				if a.Life() != want || clone.GetPlayer(a.PlayerID()).Life() != want {
					t.Fatalf("life=%d/%d want=%d", a.Life(), clone.GetPlayer(a.PlayerID()).Life(), want)
				}
				if b.Life() != 20 || len(g.PublicReveals()) != 0 || len(a.Hand()) != 2 && scenario != "empty" {
					t.Fatal("selection moved or revealed a card")
				}
			})
		}
	}
}

func TestRONOptionalHandChoiceMaskPrerequisite(t *testing.T) {
	record := repositoryRecord(t, "LEB", "Illusionary Mask", "LEB.ron")
	_, err := Compile(record)
	if err != nil {
		t.Fatalf("complete Mask record: %v", err)
	}
	rule := record.Fields["Rules"].Args[0]
	actions := &rule.Args[1].Args[0]
	actions.Args[0].Args[0].Args[0] = actions.Args[0].Args[0].Args[0].Args[0].Args[0]
	actions.Args[1].Args[1].Args = []Node{nameScopeNode(t, `GainLife(Integer(4))`)}
	record.Fields["Rules"] = Node{Kind: List, Args: []Node{rule}}
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	chooser := &ronHandChoicePlayer{BasePlayer: a, accept: true}
	g.SetPlayerAt(0, chooser)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	a.AddToHand(mage.NewCreature("Candidate", "", 1, 1))
	g.SetStep(core.PrecombatMain)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 24 || chooser.choices != 1 {
		t.Fatal("real Mask choice projection did not resolve")
	}
}

func TestRONOptionalHandChoiceInvalid(t *testing.T) {
	for _, body := range []string{
		`MayCost(ChooseACardInHand),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(ChooseACardInHand()),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(ChooseACardInHand(AnyCard,AnyCard)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(ChooseACardInHand(And([]))),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(ChooseACardInHand(IsTapped)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`MayCost(ChooseACardInHand(AnyCard))`,
		`MayCost(ChooseACardInHand(AnyCard)),If(CostWasPaid,[])`,
		`MayCost(ChooseACardInHand(AnyCard)),If(CostWasPaid(),[GainLife(Integer(1))])`,
		`MayCost(ChooseACardInHand(AnyCard)),If(CostWasPaid,[ExchangeOwnershipOfTwoCards(TheCardInHandChosenThisWay(),TheCardInHandChosenThisWay)])`,
		`MayCost(ChooseACardInHand(AnyCard)),If(CostWasntPaid,[ExchangeOwnershipOfTwoCards(TheCardInHandChosenThisWay,TheCardInHandChosenThisWay)])`,
		`MayCost(ChooseACardInHand(AnyCard)),If(CostWasPaid,[GainLife(Integer(1))]),ExchangeOwnershipOfTwoCards(TheCardInHandChosenThisWay,TheCardInHandChosenThisWay)`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + body + `]))])`
		if _, err := Compile(nameScopeNode(t, src)); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONOptionalHandChoiceBranches(t *testing.T) {
	for _, accept := range []bool{true, false} {
		for _, branch := range []string{
			`IfElse(CostWasPaid,[GainLife(Integer(2))],[GainLife(Integer(3))])`,
			`IfElse(CostWasntPaid,[GainLife(Integer(3))],[GainLife(Integer(2))])`,
		} {
			g, a, b := ronGame()
			chooser := &ronHandChoicePlayer{BasePlayer: b, accept: accept}
			g.SetPlayerAt(1, chooser)
			b.AddToHand(mage.NewLand("Other player's card"))
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(Opponent)],ActionList([PlayerMayCost(Ref_TargetPlayer,ChooseACardInHand(AnyCard)),`+branch+`])))`)
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			want := 22
			if !accept {
				want = 23
			}
			if a.Life() != want || b.Life() != 20 || chooser.offers != 1 {
				t.Fatalf("branch or explicit payer: %d/%d/%d", a.Life(), b.Life(), chooser.offers)
			}
		}
	}
}

func TestRONOptionalHandChoiceReference(t *testing.T) {
	_, a, b := ronGame()
	g, err := mage.NewGameWithAnte(a, b, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	chooser := &ronHandChoicePlayer{BasePlayer: b, accept: true}
	g.SetPlayerAt(1, chooser)
	first, second := mage.NewLand("Own"), mage.NewLand("Opponent")
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	a.AddToHand(first)
	b.AddToHand(second)
	rule := `Activated(PayMana([]),Targeted([TargetPlayer(Opponent)],ActionList([PlayerAction(You,RevealACardFromHandAtRandom),PlayerMayCost(Ref_TargetPlayer,ChooseACardInHand(AnyCard)),If(CostWasPaid,[ExchangeOwnershipOfTwoCards(TheCardInHandChosenThisWay,TheCardRevealedFromHandThisWay)])])))`
	source := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindCardAnywhere(first.ID()).Owner() != b.PlayerID() || g.FindCardAnywhere(second.ID()).Owner() != a.PlayerID() {
		t.Fatal("chosen reference did not retain the actual hand card")
	}
}

func TestRONOptionalHandChoiceInventory(t *testing.T) {
	source := `Card(Name:"Choice",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([MayCost(ChooseACardInHand(And([IsCardtype(Creature),IsNonColor(Blue)]))),If(CostWasPaid,[ExchangeOwnershipOfTwoCards(TheCardInHandChosenThisWay,TheCardInHandChosenThisWay)])]))])`
	report, err := InspectFS(fstest.MapFS{"choice.ron": &fstest.MapFile{Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{"ChooseACardInHand": "resolution_cost", "And": "card_filter", "IsCardtype": "card_filter", "IsNonColor": "card_filter", "TheCardInHandChosenThisWay": "card_reference"}
	for _, sig := range report.Signatures {
		if role, ok := roles[sig.Name]; ok && role == sig.Role {
			delete(roles, sig.Name)
		}
	}
	if len(roles) > 0 || report.Compiled != 1 {
		t.Fatalf("missing roles=%v compiled=%d issues=%v", roles, report.Compiled, report.Issues)
	}
}

func TestRONOptionalHandChoiceContexts(t *testing.T) {
	body := `MayCost(ChooseACardInHand(AnyCard)),If(CostWasPaid,[GainLife(Integer(1))])`
	for _, rule := range []string{
		`Activated(ChooseACardInHand(AnyCard),ActionList([GainLife(Integer(1))]))`,
		`AsPermanentEnters(ThisPermanent,[` + body + `])`,
		`PermanentLayerEffect(ThisPermanent,[` + body + `])`,
		`Activated(PayMana([]),ActionList([MustCost(ChooseACardInHand(AnyCard)),If(CostWasPaid,[GainLife(Integer(1))])]))`,
		`Activated(PayMana([]),ActionList([AnyPlayerMayCost(ChooseACardInHand(AnyCard)),If(CostWasPaid,[GainLife(Integer(1))])]))`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	g, a, _ := ronGame()
	chooser := &ronHandChoicePlayer{BasePlayer: a, accept: true}
	g.SetPlayerAt(0, chooser)
	a.AddToHand(mage.NewLand("Selection"))
	factory, err := Compile(nameScopeNode(t, `Card(Name:"Choice spell",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`))
	if err != nil {
		t.Fatal(err)
	}
	castRONSpell(t, g, a, factory())
	g.ResolveStack()
	if a.Life() != 21 || chooser.choices != 1 {
		t.Fatal("spell choice did not resolve")
	}
}
