package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONAntePaymentPermanent(t *testing.T) {
	for _, cost := range []string{"MustCost", "MayCost"} {
		for _, state := range []string{"pay", "decline", "phased", "return", "departed", "foreign-owner", "clone"} {
			t.Run(cost+"/"+state, func(t *testing.T) {
				a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: state != "decline"}
				b := mage.NewBasePlayer("B")
				g, _ := mage.NewGameWithAnte(a, b, nil, nil)
				g.SetTurn(2)
				factory := permanentScopeFactory(t, "Artifact", `Activated(TapPermanent(ThisPermanent),ActionList([`+cost+`(AnteAPermanent(SinglePermanent(ThisPermanent))),IfElse(CostWasPaid,[GainLife(Integer(3))],[LoseLife(Integer(2))])]))`)
				card := factory()
				card.SetOwner(a.PlayerID())
				if state == "foreign-owner" {
					card.SetOwner(b.PlayerID())
				}
				p := g.PutOnBattlefield(card, a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if a.offers != 0 || !p.Tapped || len(a.Ante()) != 0 {
					t.Fatal("payment before resolution")
				}
				if state == "phased" {
					g.PhaseOut(p.ID())
				}
				if state == "return" || state == "departed" {
					g.BouncePermanentToHand(p)
				}
				if state == "return" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
				if state == "clone" {
					clone := g.Clone()
					clone.ResolveStack()
					if len(a.Ante()) != 0 || a.Life() != 20 || g.StackSize() != 1 {
						t.Fatal("clone changed original")
					}
				}
				g.ResolveStack()
				paid := state == "pay" || state == "clone" || state == "decline" && cost == "MustCost"
				want := 18
				if paid {
					want = 23
				}
				if a.Life() != want || (len(a.Ante()) == 1) != paid {
					t.Fatalf("life=%d ante=%d", a.Life(), len(a.Ante()))
				}
				if cost == "MustCost" && a.offers != 0 {
					t.Fatal("mandatory offer")
				}
			})
		}
	}
}

func TestRONAntePaymentRecordSubrules(t *testing.T) {
	record := staticBaseRecord(t, "ARN", "Jeweled Bird")
	if _, err := Compile(record); err != nil {
		t.Fatalf("complete Jeweled Bird declaration: %v", err)
	}
	rules := record.Fields["Rules"]
	activation := rules.Args[1]
	actions := activation.Args[1].Args[0]
	branch := actions.Args[1]
	branch.Args[1] = Node{Kind: List, Args: []Node{{Kind: Symbol, Name: "DrawACard"}}}
	actions.Args[1] = branch
	activation.Args[1].Args[0] = actions
	rules.Args[1] = activation
	record.Fields["Rules"] = rules
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g, _ := mage.NewGameWithAnte(a, b, nil, nil)
	fillRONLibrary(t, a, 3)
	card := factory()
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(a.Ante()) != 1 || a.Ante()[0].ID() != card.ID() || len(a.Hand()) != 1 || len(a.Graveyard()) != 0 {
		t.Fatal("real source payment subrule")
	}
	rebirth := staticBaseRecord(t, "LEG", "Rebirth")
	if _, err := Compile(rebirth); err == nil || !strings.Contains(err.Error(), "EachPlayerMayCost") {
		t.Fatalf("expected paid-player gap: %v", err)
	}
	cost := rebirth.Fields["Rules"].Args[1].Args[0].Args[0].Args[0].Args[1]
	if cost.Kind != Symbol || cost.Name != "AnteTopCardOfLibrary" {
		t.Fatal("real Rebirth payment shape changed")
	}
}

func TestRONAntePaymentPayerAndSubject(t *testing.T) {
	for _, permanent := range []bool{false, true} {
		a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: true}
		b := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("B")}, accept: true}
		g, _ := mage.NewGameWithAnte(a, b, nil, nil)
		g.SetStep(core.PrecombatMain)
		g.SetTurn(1)
		fillRONLibrary(t, b.BasePlayer, 3)
		cost, targets := "AnteTopCardOfLibrary", "TargetPlayer(AnyPlayer)"
		payer := "Ref_TargetPlayer"
		chosen := []uuid.UUID{b.PlayerID()}
		if permanent {
			card := mage.NewArtifact("owned by B", "")
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			cost = "AnteAPermanent(SinglePermanent(Ref_TargetPermanent))"
			targets = "TargetPermanent(AnyPermanent)"
			payer = "ControllerOfPermanent(Ref_TargetPermanent)"
			chosen = []uuid.UUID{p.ID()}
		}
		card := permanentScopeFactory(t, "Sorcery", `SpellActions(Targeted([`+targets+`],ActionList([PlayerMayCost(`+payer+`,`+cost+`),If(CostWasPaid,[PlayerAction(`+payer+`,GainLife(Integer(3)))])])))`)()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), chosen, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if a.Life() != 20 || b.Life() != 23 || a.offers != 0 || b.offers != 1 || len(b.Ante()) != 1 {
			t.Fatal("wrong payer or subject")
		}
	}
}

func TestRONAntePaymentTop(t *testing.T) {
	for _, accept := range []bool{false, true} {
		a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: accept}
		b := mage.NewBasePlayer("B")
		g, _ := mage.NewGameWithAnte(a, b, nil, nil)
		g.SetStep(core.PrecombatMain)
		g.SetTurn(1)
		fillRONLibrary(t, a.BasePlayer, 3)
		top := a.Library()[0].ID()
		card := permanentScopeFactory(t, "Sorcery", `SpellActions(ActionList([MayCost(AnteTopCardOfLibrary),If(CostWasPaid,[GainLife(Integer(3)),MayCost(AnteTopCardOfLibrary),Unless(CostWasPaid,[LoseLife(Integer(1))])])]))`)()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if accept {
			if a.Life() != 23 || len(a.Ante()) != 2 || a.Ante()[0].ID() != top || a.offers != 2 {
				t.Fatal("nested payments")
			}
		} else if a.Life() != 20 || len(a.Ante()) != 0 || a.offers != 1 {
			t.Fatal("declined payment")
		}
	}
}

func TestRONAntePaymentInvalid(t *testing.T) {
	for _, body := range []string{
		`MayCost(AnteTopCardOfLibrary())`,
		`MustCost(AnteAPermanent(ThisPermanent)),If(CostWasPaid,[DrawACard])`,
		`MustCost(AnteAPermanent(SinglePermanent(Ref_TargetPermanent))),If(CostWasPaid,[DrawACard])`,
		`MayCost(AnteAPermanent(SinglePermanent(ThisPermanent))),If(AnyPlayer,[DrawACard])`,
		`MayCost(AnteTopCardOfLibrary),DrawACard`,
		`If(CostWasPaid,[DrawACard])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(TapPermanent(ThisPermanent),ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONAntePaymentDelayedIdentity(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phased", "cycle", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g, _ := mage.NewGameWithAnte(a, b, nil, nil)
			card := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([MustCost(AnteAPermanent(SinglePermanent(ThisPermanent))),If(CostWasPaid,[GainLife(Integer(3))])]))]))`)()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			original := g.CaptureDamageSource(p.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if state == "departed" || state == "returned" {
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			}
			if state == "phased" {
				g.PhaseOut(p.ID())
			}
			if state == "cycle" {
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			for range 2 {
				refs := g.WaitingDelayedPermanentSources()
				if len(refs) != 1 || !refs[0].SameObject(original) || len(a.Ante()) != 0 {
					t.Fatal("waiting identity or query purity")
				}
			}
			clone := g.Clone()
			if state == "clone" {
				clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				clone.PutTriggersOnStack()
				clone.ResolveStack()
				if len(a.Ante()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
					t.Fatal("clone changed original")
				}
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if len(g.WaitingDelayedPermanentSources()) != 0 {
				t.Fatal("delay was not consumed")
			}
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			want := 20
			if state == "live" || state == "cycle" || state == "copy" || state == "clone" {
				want = 23
			}
			if a.Life() != want || b.Life() != 20 {
				t.Fatal("saved source or copied payer")
			}
		})
	}
}

func TestRONAntePaymentContextsAndInventory(t *testing.T) {
	for _, rule := range []string{
		`Activated(AnteTopCardOfLibrary,ActionList([DrawACard]))`,
		`Activated(AnteAPermanent(SinglePermanent(ThisPermanent)),ActionList([DrawACard]))`,
		`AsPermanentEnters(ThisPermanent,[MustCost(AnteAPermanent(SinglePermanent(ThisPermanent)))])`,
		`PermanentLayerEffect(ThisPermanent,MayCost(AnteTopCardOfLibrary))`,
		`Activated(PayMana([]),ActionList([EachPlayerMayCost(AnyPlayer,AnteTopCardOfLibrary),EachPlayerAction(PaidCost,DrawACard)]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	nodes, err := Parse(strings.NewReader(`Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([MustCost(AnteAPermanent(SinglePermanent(ThisPermanent))),If(CostWasPaid,[DrawACard])]))])`), "inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Inventory", "Card", "card", "collection", true, entries)
	roles := map[string]string{}
	for _, entry := range entries {
		roles[entry.Name] = entry.Role
	}
	if roles["AnteAPermanent"] != "resolution_cost" || roles["SinglePermanent"] != "ante_payment_subject" || roles["ThisPermanent"] != "permanent_reference" {
		t.Fatal("incorrect semantic roles", roles)
	}
}
