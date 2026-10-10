package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONRequiredPlayerPaymentGameplay(t *testing.T) {
	for _, branch := range []struct {
		text                 string
		paidLife, unpaidLife int
	}{
		{`If(CostWasPaid,[GainLife(Integer(3))])`, 23, 20},
		{`Unless(CostWasntPaid,[GainLife(Integer(3))])`, 23, 20},
		{`If(CostWasntPaid,[GainLife(Integer(3))])`, 20, 23},
		{`Unless(CostWasPaid,[GainLife(Integer(3))])`, 20, 23},
		{`IfElse(CostWasPaid,[GainLife(Integer(3))],[GainLife(Integer(7))])`, 23, 27},
		{`IfElse(CostWasntPaid,[GainLife(Integer(7))],[GainLife(Integer(3))])`, 23, 27},
	} {
		for _, mode := range []string{"paid", "empty", "invalid", "clone", "departed", "phased", "control"} {
			t.Run(branch.text+"/"+mode, func(t *testing.T) {
				a := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("A")}
				b := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("B"), invalid: mode == "invalid"}
				g := mage.NewGame(a, b)
				g.SetTurn(3)
				g.SetStep(core.PrecombatMain)
				factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerMustCost(Ref_TargetPlayer,DiscardACard),`+branch.text+`])))`)
				card := factory()
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				if mode != "empty" {
					c := mage.NewLand("Payment")
					c.SetOwner(b.PlayerID())
					b.AddToHand(c)
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil {
					t.Fatal("unpaid activation succeeded")
				}
				a.ManaPool().Add(core.Colorless, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				if b.choices != 0 || a.offers != 0 || b.offers != 0 || a.Life() != 20 {
					t.Fatal("payment ran at announcement")
				}
				switch mode {
				case "departed":
					g.DestroyPermanent(p)
				case "phased":
					g.PhaseOut(p.ID())
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})

				}
				run := g
				if mode == "clone" {
					run = g.Clone()
				}
				run.ResolveStack()
				paid := mode != "empty" && mode != "invalid"
				want := branch.unpaidLife
				if paid {
					want = branch.paidLife
				}
				if run.GetPlayer(a.PlayerID()).Life() != want || a.offers != 0 || b.offers != 0 {
					t.Fatalf("life=%d want=%d offers=%d/%d", run.GetPlayer(a.PlayerID()).Life(), want, a.offers, b.offers)
				}
				if paid && len(run.GetPlayer(b.PlayerID()).Hand()) != 0 {
					t.Fatal("wrong payer hand")
				}
				if !paid && mode == "invalid" && len(b.Hand()) != 1 {
					t.Fatal("invalid choice changed hand")
				}
				if mode == "clone" && (a.Life() != 20 || len(b.Hand()) != 1 || g.StackSize() != 1) {
					t.Fatal("clone changed original")
				}
			})
		}
	}
}

func TestRONRequiredPlayerPaymentNested(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([PlayerMustCost(You,DiscardACard),IfElse(CostWasPaid,[PlayerMustCost(You,DiscardACard),IfElse(CostWasPaid,[GainLife(Integer(100))],[GainLife(Integer(2))]),GainLife(Integer(3))],[GainLife(Integer(100))]),PlayerMustCost(You,DiscardACard),IfElse(CostWasPaid,[GainLife(Integer(100))],[GainLife(Integer(7))])]))`)
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	a.AddToHand(mage.NewLand("Only"))
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 32 || len(a.Graveyard()) != 1 {
		t.Fatal("nested or adjacent result escaped")
	}
}

func TestRONRequiredPlayerPaymentChainsClause(t *testing.T) {
	for _, empty := range []bool{false, true} {
		record := staticBaseRecord(t, "LEG", "Chains of Mephistopheles")
		if _, err := Compile(record); err == nil {
			t.Fatal("unsupported mill must remain rejected")
		}
		rule := record.Fields["Rules"].Args[0]
		body, err := Parse(strings.NewReader(`PlayerAction(WouldDrawACard_ThatPlayer,GainLife(Integer(7)))`), "unpaid.ron")
		if err != nil {
			t.Fatal(err)
		}
		rule.Args[1].Args[1].Args[2] = Node{Kind: List, Args: body}
		record.Fields["Rules"] = Node{Kind: List, Args: []Node{rule}}
		factory, err := Compile(record)
		if err != nil {
			t.Fatal(err)
		}
		a := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("A")}
		b := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("B")}
		g := mage.NewGame(a, b)
		g.SetTurn(3)
		g.SetStep(core.Upkeep)
		g.PutOnBattlefield(factory(), a.PlayerID())
		seedRONDrawLibrary(b.BasePlayer)
		if !empty {
			c := mage.NewLand("Discard")
			c.SetOwner(b.PlayerID())
			b.AddToHand(c)
		}
		g.PlayerDrawCard(b)
		if a.Life() != 20 || a.choices != 0 || a.offers != 0 || b.offers != 0 || g.StackSize() != 0 {
			t.Fatal("drawing player or immediate timing")
		}
		if empty {
			if b.Life() != 27 || len(b.Hand()) != 0 || len(b.Graveyard()) != 0 {
				t.Fatal("unpaid branch")
			}
		} else if b.Life() != 20 || len(b.Hand()) != 1 || len(b.Graveyard()) != 1 {
			t.Fatal("discard and nonrecursive replacement draw")
		}
	}
}

func TestRONRequiredPlayerPaymentInvalid(t *testing.T) {
	pair := `PlayerMustCost(You,DiscardACard),IfElse(CostWasPaid,[GainLife(Integer(1))],[GainLife(Integer(2))])`
	for _, body := range []string{
		`PlayerMustCost`, `PlayerMustCost()`, `PlayerMustCost(You)`, `PlayerMustCost(You,DiscardACard)`,
		strings.Replace(pair, "You,", "Ref_TargetPermanent,", 1), strings.Replace(pair, "You,", "WouldDrawACard_ThatPlayer,", 1),
		strings.Replace(pair, "DiscardACard", "DiscardACard()", 1), strings.Replace(pair, "DiscardACard", "ChooseAPermanent(IsCardtype(Creature))", 1),
		strings.Replace(pair, "CostWasPaid", "CostWasPaid()", 1), strings.Replace(pair, "CostWasPaid", "You", 1),
		strings.Replace(pair, "[GainLife(Integer(1))]", "[]", 1), strings.Replace(pair, "[GainLife(Integer(2))]", "[]", 1),
		strings.Replace(pair, "IfElse", "GainLife(Integer(1)),IfElse", 1), pair + `,If(CostWasPaid,[GainLife(Integer(100))])`,
		strings.Replace(pair, "[GainLife(Integer(2))]", "[PlayerAction(You,DiscardTheCardDrawnThisWay)]", 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", body)
		}
	}
}

func TestRONRequiredPlayerPaymentInventory(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Scope",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([PlayerMustCost(You,DiscardACard),If(CostWasPaid,[DrawACard])]))])`), "inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := make(map[string]*InventorySignature)
	walkInventory(nodes[0], "Scope", "card", "card", "card", true, entries)
	for name, role := range map[string]string{"PlayerMustCost": "action", "You": "player_reference", "DiscardACard": "resolution_cost"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name {
				found = true
				if entry.Role != role {
					t.Fatalf("%s role=%s", name, entry.Role)
				}
			}
		}
		if !found {
			t.Fatal("missing", name)
		}
	}
}

func TestRONRequiredPlayerPaymentTakklemaggotContext(t *testing.T) {
	for _, mode := range []string{"normal", "return-host", "return-aura", "copy", "clone", "empty"} {
		t.Run(mode, func(t *testing.T) {
			record := staticBaseRecord(t, "LEG", "Takklemaggot")
			if _, err := Compile(record); err == nil {
				t.Fatal("unsupported choice and Aura return must remain rejected")
			}
			rule := record.Fields["Rules"].Args[2]
			actions, err := Parse(strings.NewReader(`[PlayerMustCost(ControllerOfDeadPermanent,DiscardACard),IfElse(CostWasPaid,[PlayerAction(ControllerOfDeadPermanent,GainLife(Integer(3)))],[PlayerAction(ControllerOfDeadPermanent,GainLife(Integer(7)))]) ]`), "supported-payment.ron")
			if err != nil {
				t.Fatal(err)
			}
			rule.Args[1] = Node{Kind: Tuple, Name: "ActionList", Args: []Node{actions[0]}}
			record.Fields["Rules"] = Node{Kind: List, Args: []Node{record.Fields["Rules"].Args[0], rule}}
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			a := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("B")}
			g := mage.NewGame(a, b)
			g.SetTurn(3)
			host := mage.NewCreature("Host", "", 2, 3)
			host.SetOwner(a.PlayerID())
			hp := g.PutOnBattlefield(host, b.PlayerID())
			aura := factory()
			aura.SetOwner(a.PlayerID())
			ap := g.PutOnBattlefield(aura, a.PlayerID())
			g.Attach(ap.ID(), hp.ID())
			if mode != "empty" {
				c := mage.NewLand("Payment")
				c.SetOwner(b.PlayerID())
				b.AddToHand(c)
			}
			g.DestroyPermanent(hp)
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || b.choices != 0 {
				t.Fatal("event timing")
			}
			if mode == "return-host" {
				a.RemoveFromGraveyard(host.ID())
				g.PutOnBattlefield(host, a.PlayerID())
			}
			if mode == "return-aura" {
				g.DestroyPermanent(g.FindPermanent(ap.ID()))
				a.RemoveFromGraveyard(aura.ID())
				g.PutOnBattlefield(aura, b.PlayerID())
			}
			if mode == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy")
			}
			run := g
			if mode == "clone" {
				run = g.Clone()
			}
			run.ResolveStack()
			want := 23
			if mode == "empty" {
				want = 27
			}
			if mode == "copy" {
				want = 30
			}
			if run.GetPlayer(a.PlayerID()).Life() != 20 || run.GetPlayer(b.PlayerID()).Life() != want || a.offers != 0 || b.offers != 0 {
				t.Fatal("saved death controller or payment result")
			}
			if mode == "clone" && (b.Life() != 20 || len(b.Hand()) != 1 || g.StackSize() != 1) {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestRONRequiredPlayerPaymentCopyAndLKI(t *testing.T) {
	for _, mode := range []string{"copy", "return", "departed", "phase-cycle"} {
		t.Run(mode, func(t *testing.T) {
			nodes, err := Parse(strings.NewReader(`Card(Name:"Saved source",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),ManaCost:[],CardPT:(Power:3,Toughness:4),Rules:[Activated(PayMana([]),ActionList([PlayerMustCost(You,DiscardACard),IfElse(CostWasPaid,[GainLife(PowerOfPermanent(ThisPermanent))],[GainLife(Integer(7))])]))])`), "source.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			a := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("B")}
			g := mage.NewGame(a, b)
			g.SetTurn(3)
			c := factory()
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			pay := mage.NewLand("Pay")
			pay.SetOwner(a.PlayerID())
			a.AddToHand(pay)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "copy":
				pay := mage.NewLand("Other pay")
				pay.SetOwner(b.PlayerID())
				b.AddToHand(pay)
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			case "departed", "return":
				g.DestroyPermanent(p)
				if mode == "return" {
					a.RemoveFromGraveyard(c.ID())
					q := g.PutOnBattlefield(c, b.PlayerID())
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(20), mage.Fixed(0))), q.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			g.ResolveStack()
			wantB := 20
			if mode == "copy" {
				wantB = 23
			}
			if a.Life() != 23 || b.Life() != wantB || a.offers != 0 || b.offers != 0 {
				t.Fatal("copy controller or exact source LKI")
			}
		})
	}
}

func TestRONRequiredPlayerPaymentReplacementDiscardIsEffect(t *testing.T) {
	record := staticBaseRecord(t, "LEG", "Chains of Mephistopheles")
	rule := record.Fields["Rules"].Args[0]
	body, err := Parse(strings.NewReader(`PlayerAction(WouldDrawACard_ThatPlayer,GainLife(Integer(7)))`), "unpaid.ron")
	if err != nil {
		t.Fatal(err)
	}
	rule.Args[1].Args[1].Args[2] = Node{Kind: List, Args: body}
	record.Fields["Rules"] = Node{Kind: List, Args: []Node{rule}}
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	a := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
	g := mage.NewGame(a, b)
	g.SetTurn(3)
	g.SetStep(core.Upkeep)
	g.PutOnBattlefield(factory(), a.PlayerID())
	g.PutOnBattlefield(mage.NewArtifact("Effect discard replacement", "", mage.WithStaticAbility(mage.DiscardToLibraryReplacement())), b.PlayerID())
	seedRONDrawLibrary(b.BasePlayer)
	pay := mage.NewLand("Saved")
	pay.SetOwner(b.PlayerID())
	b.AddToHand(pay)
	g.PlayerDrawCard(b)
	if len(b.Hand()) != 1 || b.Hand()[0].ID() != pay.ID() || len(b.Graveyard()) != 0 || b.offers != 1 || a.offers != 0 || b.Life() != 20 {
		t.Fatal("replacement discard must permit Library of Leng and redraw its card")
	}
}

func TestRONRequiredPlayerPaymentFutureAndDelayedContext(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		body := `PlayerMustCost(You,DiscardACard),IfElse(CostWasPaid,[GainLife(Integer(3))],[GainLife(Integer(7))])`
		if delayed {
			body = `CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + body + `]))`
		}
		factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureReplaceWouldDraw(NextTimePlayerWouldDrawACardThisTurn(You),[`+body+`])]))`)
		a := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
		g := mage.NewGame(a, mage.NewBasePlayer("B"))
		g.SetTurn(3)
		source := g.PutOnBattlefield(factory(), a.PlayerID())
		g.PutOnBattlefield(mage.NewArtifact("Effect discard replacement", "", mage.WithStaticAbility(mage.DiscardToLibraryReplacement())), a.PlayerID())
		c := mage.NewLand("Pay")
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		g.PlayerDrawCard(a)
		if delayed {
			if a.Life() != 20 || len(a.Hand()) != 1 {
				t.Fatal("delayed body ran early")
			}
			g.RunStepWithPriority(core.EndStep)
			g.ResolveStack()
		}
		if a.Life() != 23 {
			t.Fatal("payment branch")
		}
		if delayed {
			if len(a.Graveyard()) != 1 || a.offers != 0 {
				t.Fatal("delayed trigger inherited replacement action context")
			}
		} else if len(a.Library()) != 1 || a.offers != 1 {
			t.Fatal("future replacement lost effect discard context")
		}
	}
}
