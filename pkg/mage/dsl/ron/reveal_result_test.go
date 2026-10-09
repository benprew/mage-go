package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONRevealResultGameplay(t *testing.T) {
	for _, state := range []string{"present", "departed", "returned", "phased", "empty"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,RevealACardFromHandAtRandom),If(ACardWasRevealedThisWay(IsCardtype(Land)),[GainLife(Integer(2))]),Unless(ACardWasRevealedThisWay(IsCardtype(Land)),[GainLife(Integer(3))]),IfElse(ACardWasRevealedThisWay(IsNonCardtype(Land)),[GainLife(Integer(5))],[GainLife(Integer(7))])])) )`)
			c := factory()
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			if state != "empty" {
				land := mage.NewLand("Land")
				land.SetOwner(b.PlayerID())
				b.AddToHand(land)
				artifact := mage.NewArtifact("Artifact", "")
				artifact.SetOwner(b.PlayerID())
				b.AddToHand(artifact)
			}
			before := a.Life()
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil || len(g.PublicReveals()) != 0 {
				t.Fatal("failed payment executed reveal")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if a.Life() != before || len(g.PublicReveals()) != 0 {
				t.Fatal("condition ran at announcement")
			}
			switch state {
			case "phased":
				g.PhaseOut(p.ID())
			case "departed", "returned":
				g.DestroyPermanent(p)
				if state == "returned" {
					g.PutOnBattlefield(p.Card, a.PlayerID())
				}
			}
			clone := g.Clone()
			g.SetRandomResults([]int{0})
			clone.SetRandomResults([]int{1})
			g.ResolveStack()
			clone.ResolveStack()
			want, cwant := 9, 8
			if state == "empty" {
				want, cwant = 10, 10
			}
			if a.Life() != before+want || clone.GetPlayer(a.PlayerID()).Life() != before+cwant {
				t.Fatalf("wrong branch: original=%d clone=%d", a.Life()-before, clone.GetPlayer(a.PlayerID()).Life()-before)
			}
			if len(b.Hand()) != 2 && state != "empty" {
				t.Fatal("predicate moved a card")
			}
		})
	}
}

func TestRONRevealResultInvalid(t *testing.T) {
	for _, actions := range []string{
		`If(ACardWasRevealedThisWay(IsCardtype(Land)),[GainLife(Integer(1))])`,
		`If(ACardWasRevealedThisWay(IsCardtype(Land)),[GainLife(Integer(1))]),PlayerAction(You,RevealACardFromHandAtRandom)`,
		`PlayerAction(You,RevealACardFromHandAtRandom),If(ACardWasRevealedThisWay,[GainLife(Integer(1))])`,
		`PlayerAction(You,RevealACardFromHandAtRandom),If(ACardWasRevealedThisWay(IsCardtype(Land),IsCardtype(Land)),[GainLife(Integer(1))])`,
		`PlayerAction(You,RevealACardFromHandAtRandom),If(ACardWasRevealedThisWay(IsCardtype(You)),[GainLife(Integer(1))])`,
		`PlayerAction(You,RevealACardFromHandAtRandom),If(ACardWasRevealedThisWay(IsNamed(TheChosenName)),[GainLife(Integer(1))])`,
		`PlayerAction(You,RevealACardFromHandAtRandom),If(ACardWasRevealedThisWay(IsColor(Red)),[GainLife(Integer(1))])`,
		`PlayerAction(You,RevealACardFromHandAtRandom),If(ACardWasRevealedThisWay(IsCardtype(Land)),[])`,
		`PlayerMayCost(You,PayLife(Integer(1))),Unless(CostWasPaid,[PlayerAction(You,RevealACardFromHandAtRandom)]),If(ACardWasRevealedThisWay(IsCardtype(Land)),[GainLife(Integer(1))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+actions+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", actions)
		}
	}
}

func TestRONRevealResultRealOperands(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"DRK", "Wand of Ith"}, {"LEG", "Petra Sphinx"}, {"ARN", "Sindbad"}} {
		t.Run(tc.name, func(t *testing.T) {
			record := staticBaseRecord(t, tc.set, tc.name)
			var conditions []Node
			var choice Node
			var walk func(Node)
			walk = func(n Node) {
				if n.Name == "ACardWasRevealedThisWay" {
					conditions = append(conditions, n)
				}
				if n.Name == "ChooseACardName" {
					choice = n
				}
				for _, child := range n.Args {
					walk(child)
				}
				for _, child := range n.Fields {
					walk(child)
				}
			}
			walk(record)
			if len(conditions) == 0 {
				t.Fatal("real record has no reveal predicate")
			}
			if tc.name == "Wand of Ith" && (len(conditions) != 2 || conditions[1].Args[0].Name != "IsNonCardtype") {
				t.Fatal("nonland branch differs from Oracle")
			}
			for i, condition := range conditions {
				g, base, b := ronGame()
				a := &ronCardNameTester{Player: base, name: "Eleven, the Mage"}
				g.SetPlayerAt(0, a)
				land := mage.NewLand("Cecily, Haunted Mage")
				land.SetOwner(b.PlayerID())
				b.AddToHand(land)
				parts := []Node{}
				if choice.Name != "" {
					parts = append(parts, choice)
				}
				parts = append(parts, nameScopeNode(t, `PlayerAction(Ref_TargetPlayer,RevealACardFromHandAtRandom)`), Node{Kind: Tuple, Name: "IfElse", Args: []Node{condition, nameScopeNode(t, `[GainLife(Integer(2))]`), nameScopeNode(t, `[GainLife(Integer(1))]`)}})
				target := &playerTargetPlan{name: "Ref_TargetPlayer"}
				declaration := mage.DeclarePlayerTarget(target.name)
				builds, err := compileActionSequence(parts, scope{playerTarget: target})
				if err != nil {
					t.Fatal(err)
				}
				bindings := instanceBindings{chosenNames: make(map[*cardNameChoicePlan]*mage.CardNameBinding), players: map[*playerTargetPlan]*mage.PlayerTargetBinding{target: declaration}}
				effects := make([]mage.Effect, len(builds))
				for j, build := range builds {
					effects[j] = build(bindings)
				}
				card := mage.NewArtifact("Predicate", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.Sequence(effects...), mage.WithTargets(declaration))))
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				before := a.Life()
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				want := 2
				if tc.name == "Wand of Ith" && i == 1 {
					want = 1
				}
				if a.Life() != before+want || len(g.PublicReveals()) != 1 {
					t.Fatal("real predicate selected the wrong branch")
				}
			}
			if _, err := Compile(record); err == nil {
				t.Fatal("incomplete full card accepted")
			}
		})
	}
}

func TestRONRevealResultContexts(t *testing.T) {
	for _, kind := range []string{"spell", "trigger", "delayed"} {
		t.Run(kind, func(t *testing.T) {
			g, a, b := ronGame()
			land := mage.NewLand("Land")
			land.SetOwner(b.PlayerID())
			b.AddToHand(land)
			actions := `PlayerAction(Ref_TargetPlayer,RevealACardFromHandAtRandom),If(ACardWasRevealedThisWay(IsCardtype(Land)),[GainLife(Integer(2))])`
			typ, rule := "Instant", `SpellActions(Targeted([TargetPlayer(Opponent)],ActionList([`+actions+`])))`
			if kind == "trigger" {
				typ, rule = "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),Targeted([TargetPlayer(Opponent)],ActionList([`+actions+`])))`
			}
			if kind == "delayed" {
				typ, rule = "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(Opponent)],ActionList([PlayerAction(Ref_TargetPlayer,RevealACardFromHandAtRandom),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([If(ACardWasRevealedThisWay(IsCardtype(Land)),[GainLife(Integer(2))])]))])))`
			}
			factory := permanentScopeFactory(t, typ, rule)
			card := factory()
			card.SetOwner(a.PlayerID())
			before := a.Life()
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
			if len(g.PublicReveals()) != 0 || a.Life() != before {
				t.Fatal("actions executed before resolution")
			}
			g.ResolveStack()
			if kind == "delayed" {
				if a.Life() != before {
					t.Fatal("delayed predicate ran early")
				}
				b.RemoveFromHand(land.ID())
				b.AddToHand(land)
				clone := g.Clone()
				for _, game := range []*mage.Game{g, clone} {
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					game.PutTriggersOnStack()
					game.ResolveStack()
				}
				if clone.GetPlayer(a.PlayerID()).Life() != before+2 {
					t.Fatal("clone lost captured reveal facts")
				}
			}
			if a.Life() != before+2 || len(g.PublicReveals()) != 1 {
				t.Fatal("context lost reveal result")
			}
		})
	}
}

func TestRONRevealResultDistinctProducers(t *testing.T) {
	g, a, b := ronGame()
	land := mage.NewLand("Land")
	land.SetOwner(a.PlayerID())
	a.AddToHand(land)
	artifact := mage.NewArtifact("Artifact", "")
	artifact.SetOwner(b.PlayerID())
	b.AddToHand(artifact)
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(Opponent)],ActionList([PlayerAction(You,RevealACardFromHandAtRandom),If(ACardWasRevealedThisWay(IsCardtype(Land)),[GainLife(Integer(2))]),PlayerAction(Ref_TargetPlayer,RevealACardFromHandAtRandom),IfElse(ACardWasRevealedThisWay(IsNonCardtype(Land)),[GainLife(Integer(3))],[GainLife(Integer(7))])])) )`)
	before := a.Life()
	for range 2 {
		card := factory()
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	if a.Life() != before+10 || len(g.PublicReveals()) != 4 {
		t.Fatal("producers or instances shared result state")
	}
}

func TestRONRevealResultWandLandBranch(t *testing.T) {
	record := staticBaseRecord(t, "DRK", "Wand of Ith")
	var producer, branch Node
	var walk func(Node)
	walk = func(n Node) {
		if isRandomHandReveal(n) {
			producer = n
		}
		if n.Name == "If" && len(n.Args) > 0 && n.Args[0].Name == "ACardWasRevealedThisWay" && n.Args[0].Args[0].Name == "IsCardtype" {
			branch = n
		}
		for _, child := range n.Args {
			walk(child)
		}
		for _, child := range n.Fields {
			walk(child)
		}
	}
	walk(record)
	if producer.Name == "" || branch.Name == "" {
		t.Fatal("missing real Wand land branch")
	}
	for _, pay := range []bool{true, false} {
		t.Run(map[bool]string{true: "pay", false: "decline"}[pay], func(t *testing.T) {
			a := mage.NewBasePlayer("A")
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: pay}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			land := mage.NewLand("Land")
			land.SetOwner(b.PlayerID())
			b.AddToHand(land)
			target := &playerTargetPlan{name: "Ref_TargetPlayer"}
			declaration := mage.DeclarePlayerTarget(target.name)
			build, err := compileRandomHandReveal(producer, []Node{branch}, scope{playerTarget: target})
			if err != nil {
				t.Fatal(err)
			}
			body := build(instanceBindings{players: map[*playerTargetPlan]*mage.PlayerTargetBinding{target: declaration}})
			card := mage.NewArtifact("Land branch", "", mage.WithAbility(mage.Activate(mage.Costs(), body, mage.WithTargets(declaration))))
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			before := b.Life()
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if b.offers != 0 || len(g.PublicReveals()) != 0 {
				t.Fatal("resolution payment or reveal ran at announcement")
			}
			g.ResolveStack()
			if b.offers != 1 || len(g.PublicReveals()) != 1 {
				t.Fatal("real branch did not reveal and offer payment")
			}
			if pay {
				if b.Life() != before-1 || len(b.Hand()) != 1 || len(b.Graveyard()) != 0 {
					t.Fatal("paid branch discarded or charged the wrong player")
				}
			} else if b.Life() != before || len(b.Hand()) != 0 || len(b.Graveyard()) != 1 || b.Graveyard()[0].ID() != land.ID() {
				t.Fatal("unpaid branch lost the exact revealed card")
			}
		})
	}
}
