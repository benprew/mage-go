package ron

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLibraryTopRevealGameplay(t *testing.T) {
	for _, action := range []string{`PlayerAction(Ref_TargetPlayer,RevealTopCardOfLibrary)`, `PlayerActions(Ref_TargetPlayer,[RevealTopCardOfLibrary])`, `PlayerActions(Ref_TargetPlayer,[PlayerActions(ActionPlayer,[RevealTopCardOfLibrary])])`} {
		for _, state := range []string{"present", "departed", "returned", "phased", "empty"} {
			t.Run(action+state, func(t *testing.T) {
				g, a, b := ronGame()
				factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetPlayer(AnyPlayer)],ActionList([`+action+`,IfElse(ACardWasRevealedThisWay(IsCardtype(Land)),[GainLife(Integer(2))],[GainLife(Integer(3))])])))`)
				source := factory()
				source.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(source, a.PlayerID())
				if state != "empty" {
					c := mage.NewLand("Plains")
					c.SetOwner(b.PlayerID())
					b.AddToLibrary(c)
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil || len(g.PublicReveals()) != 0 {
					t.Fatal("failed payment executed reveal")
				}
				a.ManaPool().Add(core.Colorless, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				if len(g.PublicReveals()) != 0 {
					t.Fatal("announcement executed reveal")
				}
				switch state {
				case "departed", "returned":
					g.DestroyPermanent(p)
					if state == "returned" {
						g.PutOnBattlefield(p.Card, a.PlayerID())
					}
				case "phased":
					g.PhaseOut(p.ID())
				}
				clone := g.Clone()
				clone.GetPlayer(b.PlayerID()).SetLibrary(nil)
				g.ResolveStack()
				clone.ResolveStack()
				want := 22
				if state == "empty" {
					want = 23
				}
				if a.Life() != want || clone.GetPlayer(a.PlayerID()).Life() != 23 || b.Life() != 20 {
					t.Fatal("reveal result or player scope crossed frames")
				}
				if len(b.Library()) != len(g.PublicReveals()) {
					t.Fatal("reveal moved a card or revealed an empty library")
				}
			})
		}
	}
}

func TestRONLibraryTopRevealPetraPrerequisite(t *testing.T) {
	record := staticBaseRecord(t, "LEG", "Petra Sphinx")
	body := record.Fields["Rules"].Args[0].Args[1].Args[1].Args[0].Args
	producer := body[0]
	condition := body[1].Args[0]
	targetPlan := &playerTargetPlan{name: "Ref_TargetPlayer"}
	builds, err := compileActionSequence([]Node{producer, {Kind: Tuple, Name: "IfElse", Args: []Node{condition, nameScopeNode(t, `[GainLife(Integer(2))]`), nameScopeNode(t, `[GainLife(Integer(3))]`)}}}, scope{playerTarget: targetPlan})
	if err != nil {
		t.Fatal(err)
	}
	g, a, base := ronGame()
	b := &libraryNameTester{BasePlayer: base, name: "Plains"}
	g.SetPlayerAt(1, b)
	card := mage.NewLand("Plains")
	card.SetOwner(b.PlayerID())
	b.AddToLibrary(card)
	target := mage.DeclarePlayerTarget("Ref_TargetPlayer")
	bindings := instanceBindings{players: map[*playerTargetPlan]*mage.PlayerTargetBinding{targetPlan: target}, chosenNames: make(map[*cardNameChoicePlan]*mage.CardNameBinding), libraryReveals: make(map[*libraryRevealPlan]*mage.CardBinding)}
	effects := make([]mage.Effect, len(builds))
	for i, build := range builds {
		effects[i] = build(bindings)
	}
	c := mage.NewArtifact("Prerequisite", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.Sequence(effects...), mage.WithTargets(target))))
	if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(c, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if b.calls != 1 || a.Life() != 22 || len(g.PublicReveals()) != 1 {
		t.Fatal("actual Petra producer or predicate failed")
	}
	if _, err := Compile(record); err == nil {
		t.Fatal("incomplete Petra record accepted")
	}
}

func TestRONLibraryTopRevealInvalid(t *testing.T) {
	for _, actions := range []string{`RevealTopCardOfLibrary()`, `RevealTopCardOfLibrary(You)`, `PlayerAction(Ref_TargetPlayer,RevealTopCardOfLibrary)`, `If(ACardWasRevealedThisWay(IsCardtype(Land)),[RevealTopCardOfLibrary])`, `PlayerActions(You,[RevealTopCardOfLibrary()])`, `PlayerMayCost(You,PayLife(Integer(1))),Unless(CostWasPaid,[RevealTopCardOfLibrary]),If(ACardWasRevealedThisWay(IsCardtype(Land)),[GainLife(Integer(1))])`, `RevealTopCardOfLibrary,PlayerAction(You,DiscardTheCardRevealedThisWay)`} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+actions+`]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", actions)
		}
	}
}

type libraryNameTester struct {
	*mage.BasePlayer
	name  string
	calls int
}

func (p *libraryNameTester) ChooseMode(names []string, _ string) int {
	p.calls++
	return slices.Index(names, p.name)
}

func TestRONLibraryTopRevealContexts(t *testing.T) {
	for _, kind := range []string{"spell", "trigger", "delayed"} {
		t.Run(kind, func(t *testing.T) {
			g, a, b := ronGame()
			land := mage.NewLand("Plains")
			land.SetOwner(b.PlayerID())
			b.AddToLibrary(land)
			actions := `PlayerAction(Ref_TargetPlayer,RevealTopCardOfLibrary),If(ACardWasRevealedThisWay(IsCardtype(Land)),[GainLife(Integer(2))])`
			typ, rule := "Instant", `SpellActions(Targeted([TargetPlayer(Opponent)],ActionList([`+actions+`])))`
			if kind == "trigger" {
				typ, rule = "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),Targeted([TargetPlayer(Opponent)],ActionList([`+actions+`])))`
			}
			if kind == "delayed" {
				typ, rule = "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(Opponent)],ActionList([PlayerAction(Ref_TargetPlayer,RevealTopCardOfLibrary),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([If(ACardWasRevealedThisWay(IsCardtype(Land)),[GainLife(Integer(2))])]))])))`
			}
			card := permanentScopeFactory(t, typ, rule)()
			card.SetOwner(a.PlayerID())
			switch kind {
			case "spell":
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
			case "trigger":
				g.PutOnBattlefield(card, a.PlayerID())
				g.PutTriggersOnStack()
			case "delayed":
				p := g.PutOnBattlefield(card, a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
			}
			if len(g.PublicReveals()) != 0 {
				t.Fatal("reveal ran before resolution")
			}
			g.ResolveStack()
			if kind == "delayed" {
				if a.Life() != 20 {
					t.Fatal("delayed body ran early")
				}
				b.SetLibrary(nil)
				b.AddToLibrary(land)
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					game.PutTriggersOnStack()
					game.ResolveStack()
				}
				if clone.GetPlayer(a.PlayerID()).Life() != 22 {
					t.Fatal("clone lost saved reveal facts")
				}
			}
			if a.Life() != 22 || len(g.PublicReveals()) != 1 {
				t.Fatal("resolution context lost reveal result")
			}
		})
	}
}

func TestRONLibraryTopRevealDistinctResults(t *testing.T) {
	g, a, b := ronGame()
	land := mage.NewLand("Plains")
	land.SetOwner(a.PlayerID())
	a.AddToLibrary(land)
	artifact := mage.NewArtifact("Other", "")
	artifact.SetOwner(b.PlayerID())
	b.AddToLibrary(artifact)
	card := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(Opponent)],ActionList([RevealTopCardOfLibrary,If(ACardWasRevealedThisWay(IsCardtype(Land)),[GainLife(Integer(2))]),PlayerActions(Ref_TargetPlayer,[RevealTopCardOfLibrary]),IfElse(ACardWasRevealedThisWay(IsCardtype(Land)),[GainLife(Integer(5))],[GainLife(Integer(3))])])) )`)()
	p := g.PutOnBattlefield(card, a.PlayerID())
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	if a.Life() != 30 || len(g.PublicReveals()) != 4 {
		t.Fatal("pending activations or reveal producers shared results")
	}
}

func TestRONLibraryTopRevealInvalidContexts(t *testing.T) {
	for _, rule := range []string{
		`Activated(RevealTopCardOfLibrary,ActionList([GainLife(Integer(1))]))`,
		`PermanentLayerEffect(ThisPermanent,[RevealTopCardOfLibrary])`,
		`AsPermanentEnters(ThisPermanent,[RevealTopCardOfLibrary])`,
		`SpellActions(ActionList([PlayerAction(You,RevealACardFromHandAtRandom),PlayerActions(You,[RevealTopCardOfLibrary]),PlayerAction(You,DiscardTheCardRevealedThisWay)]))`,
	} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}
