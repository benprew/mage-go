package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONEligiblePlayerPaymentGameplay(t *testing.T) {
	for _, eligible := range []string{"Opponent", "AnyPlayer"} {
		for _, state := range []string{"paid", "decline", "short", "nested"} {
			t.Run(eligible+"/"+state, func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
				b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: state != "decline"}
				g := mage.NewGame(a, b)
				body := `GainLife(Integer(3))`
				if state == "nested" {
					body = `AnyPlayerMayCost(AnyPlayer,PayMana([ManaCostGeneric(9)])),Unless(CostWasPaid,[GainLife(Integer(3))])`
				}
				card := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([AnyPlayerMayCost(`+eligible+`,PayMana([ManaCostGeneric(2)])),If(CostWasPaid,[`+body+`])]))`)()
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				if state != "short" {
					b.ManaPool().Add(core.Colorless, 2)
				}
				a.ManaPool().Add(core.Colorless, 2)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if a.offers != 0 || b.offers != 0 {
					t.Fatal("offered before resolution")
				}
				g.ResolveStack()
				want := 20
				if state == "paid" || state == "nested" {
					want = 23
				}
				if g.GetPlayer(a.PlayerID()).Life() != want {
					t.Fatal("wrong local payment branch")
				}
				if eligible == "Opponent" && a.offers != 0 {
					t.Fatal("controller was eligible")
				}
				if state == "short" && b.offers != 0 {
					t.Fatal("offered unaffordable payment")
				}
				wantMana := 2
				if state == "short" || state == "paid" || state == "nested" {
					wantMana = 0
				}
				if g.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != wantMana {
					t.Fatal("wrong payer or payment")
				}
			})
		}
	}
}

func TestRONEligiblePlayerPaymentPolarityAndContexts(t *testing.T) {
	for _, clause := range []string{"If(CostWasPaid", "Unless(CostWasntPaid", "If(CostWasntPaid", "Unless(CostWasPaid", "IfElse(CostWasPaid", "IfElse(CostWasntPaid"} {
		for _, accept := range []bool{false, true} {
			for _, context := range []string{"spell", "trigger", "delay"} {
				t.Run(clause+"/"+context+"/"+fmt.Sprint(accept), func(t *testing.T) {
					a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
					b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: accept}
					g := mage.NewGame(a, b)
					branch := clause + `,[GainLife(Integer(3))])`
					if strings.HasPrefix(clause, "IfElse") {
						branch = clause + `,[GainLife(Integer(3))],[GainLife(Integer(7))])`
					}
					body := `ActionList([AnyPlayerMayCost(Opponent,PayMana([ManaCostGeneric(2)])),` + branch + `])`
					typ, rule := "Instant", `SpellActions(`+body+`)`
					if context == "trigger" {
						typ, rule = "Artifact", `TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),`+body+`)`
					}
					if context == "delay" {
						typ, rule = "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,`+body+`)]))`
					}
					card := permanentScopeFactory(t, typ, rule)()
					card.SetOwner(a.PlayerID())
					b.ManaPool().Add(core.Colorless, 2)
					switch context {
					case "spell":
						a.AddToHand(card)
						if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
							t.Fatal(err)
						}
					case "trigger":
						g.PutOnBattlefield(card, a.PlayerID())
						g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
					case "delay":
						source := g.PutOnBattlefield(card, a.PlayerID())
						if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
							t.Fatal(err)
						}
						g.ResolveStack()
						g.RemoveFromBattlefield(source)
						for range 3 {
							g.WaitingDelayedPermanentSources()
						}
						g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
					}
					if a.offers+b.offers != 0 {
						t.Fatal("payment executed before body")
					}
					g.ResolveStack()
					positive := clause == "If(CostWasPaid" || clause == "Unless(CostWasntPaid" || clause == "IfElse(CostWasPaid"
					want := 20
					if accept == positive {
						want = 23
					} else if strings.HasPrefix(clause, "IfElse") {
						want = 27
					}
					if g.GetPlayer(a.PlayerID()).Life() != want || b.offers != 1 || a.offers != 0 {
						t.Fatalf("wrong context/polarity: life=%d offers=%d/%d", g.GetPlayer(a.PlayerID()).Life(), a.offers, b.offers)
					}
				})
			}
		}
	}
}

func TestRONEligiblePlayerPaymentTargetsCopiesAndFactories(t *testing.T) {
	for _, state := range []string{"live", "source-return", "source-phase", "source-control", "target-return", "target-phase", "target-cycle", "shroud", "copy", "clone", "mana-source"} {
		t.Run(state, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
			g := mage.NewGame(a, b)
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Artifact))],ActionList([AnyPlayerMayCost(Opponent,PayMana([ManaCostGeneric(2)])),Unless(CostWasPaid,[TapPermanent(Ref_TargetPermanent)])])))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			victimCard := mage.NewArtifact("Victim", "")
			victimCard.SetOwner(b.PlayerID())
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			other := g.PutOnBattlefield(factory(), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), other.ID(), 0, []uuid.UUID{source.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveTopOfStack()
			if !g.FindPermanent(source.ID()).Tapped || victim.Tapped {
				t.Fatal("factory target declarations share state")
			}
			b.offers = 0
			if state != "mana-source" {
				b.ManaPool().Add(core.Colorless, 2)
			}
			var manaSource uuid.UUID
			switch state {
			case "source-return":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(card, b.PlayerID())
			case "source-phase":
				g.PhaseOut(source.ID())
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "target-return":
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(victimCard, b.PlayerID())
			case "target-phase":
				g.PhaseOut(victim.ID())
			case "target-cycle":
				phased := g.PhaseOut(victim.ID())
				g.PhaseIn(phased)
			case "shroud":
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
			case "copy":
				g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
			case "mana-source":
				b.accept = true
				land := repositoryCard(t, "LEB", "Forest")
				land.SetOwner(b.PlayerID())
				g.PutOnBattlefield(land, b.PlayerID())
				land = repositoryCard(t, "LEB", "Forest")
				land.SetOwner(b.PlayerID())
				manaSource = g.PutOnBattlefield(land, b.PlayerID()).ID()
			}
			clone := g.Clone()
			clone.ResolveStack()
			if state == "clone" && victim.Tapped {
				t.Fatal("clone resolved original target")
			}
			g.ResolveStack()
			legal := state != "target-return" && state != "target-phase" && state != "shroud"
			wantTap := legal && state != "mana-source"
			if g.FindPermanentIncludingPhased(victim.ID()).Tapped != wantTap || clone.FindPermanentIncludingPhased(victim.ID()).Tapped != legal {
				t.Fatal("wrong target visit or payment result")
			}
			if !legal && b.offers != 0 {
				t.Fatal("illegal target offered payment")
			}
			if legal && state != "mana-source" && b.offers != 1 && state != "copy" {
				t.Fatal("wrong eligible player after source change", b.offers)
			}
			if state == "mana-source" && !g.FindPermanent(manaSource).Tapped {
				t.Fatal("resolution payment did not use mana abilities")
			}
		})
	}
}

func TestRONEligiblePlayerPaymentInventoryAndAtomicLoading(t *testing.T) {
	src := `Card(Name:"Offer",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([AnyPlayerMayCost(Opponent,PayMana([ManaCostGeneric(2)])),Unless(CostWasPaid,[GainLife(Integer(1))])]))])`
	files := fstest.MapFS{"a.ron": {Data: []byte(src)}}
	if _, err := LoadFS(files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]bool{}
	for _, entry := range report.Signatures {
		if entry.Name == "Opponent" || entry.Name == "PayMana" {
			roles[entry.Role] = true
		}
	}
	if !roles["player_collection"] || !roles["resolution_cost"] {
		t.Fatal("missing payment inventory roles", roles)
	}
	files["b.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(src, "Name:\"Offer\"", "Name:\"Invalid\"", 1))}
	files["b.ron"].Data = []byte(strings.Replace(string(files["b.ron"].Data), "Opponent", "You", 1))
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("published partial registry")
	}
}

func TestRONEligiblePlayerPaymentInvalid(t *testing.T) {
	for _, body := range []string{
		`AnyPlayerMayCost(Opponent,PayMana([]))`,
		`AnyPlayerMayCost(Opponent(),PayMana([])),If(CostWasPaid,[GainLife(Integer(1))])`,
		`AnyPlayerMayCost(You,PayMana([])),If(CostWasPaid,[GainLife(Integer(1))])`,
		`AnyPlayerMayCost(Opponent),If(CostWasPaid,[GainLife(Integer(1))])`,
		`AnyPlayerMayCost(Opponent,PayManaX([ManaCostX],Integer(1))),If(CostWasPaid,[GainLife(Integer(1))])`,
		`AnyPlayerMayCost(Opponent,PayMana([ManaCostX])),If(CostWasPaid,[GainLife(Integer(1))])`,
		`AnyPlayerMayCost(Opponent,PayMana([ManaCostGeneric(-1)])),If(CostWasPaid,[GainLife(Integer(1))])`,
		`AnyPlayerMayCost(Opponent,PayLife(Integer(1))),If(CostWasPaid,[GainLife(Integer(1))])`,
		`AnyPlayerMayCost(Opponent,PayMana([])),If(CostWasPaid(),[GainLife(Integer(1))])`,
		`AnyPlayerMayCost(Opponent,PayMana([])),If(CostWasPaid,[])`,
		`AnyPlayerMayCost(Opponent,PayMana([])),IfElse(CostWasPaid,[GainLife(Integer(1))],[])`,
		`AnyPlayerMayCost(Opponent,PayMana([])),GainLife(Integer(1)),If(CostWasPaid,[GainLife(Integer(1))])`,
		`AnyPlayerMayCost(Opponent,PayMana([])),If(CostWasPaid,[GainLife(Integer(1))]),If(CostWasPaid,[GainLife(Integer(1))])`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + body + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid declaration", body)
		}
	}
	nodes, err := Parse(strings.NewReader(`AnyPlayerMayCost(Opponent,PayMana([])) Unless(CostWasPaid,[GainLife(Integer(1))])`), "contexts.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{drawReplacementBody: true}, {enteringSource: true}} {
		if _, err := compileEligiblePlayerPayment(nodes[0], nodes[1], ctx); err == nil {
			t.Fatal("accepted unsupported execution context")
		}
	}
	nodes, err = Parse(strings.NewReader(`AnyPlayerMayCost(Opponent,PayMana([])) If(CostWasPaid,[PlayerAction(ActionPlayer,GainLife(Integer(1)))])`), "scope.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := compileEligiblePlayerPayment(nodes[0], nodes[1], scope{actionPlayer: &playerActionsPlan{name: "outer"}}); err != nil {
		t.Fatal("eligible payment did not supply its local ActionPlayer", err)
	}
}

func TestRONEligiblePlayerPaymentRealRecords(t *testing.T) {
	for _, name := range []string{"Worms of the Earth"} {
		record := staticBaseRecord(t, "DRK", name)
		if _, err := Compile(record); err == nil {
			t.Fatal("accepted enclosing record with remaining blockers")
		}
	}
	record := staticBaseRecord(t, "DRK", "Scarwood Bandits")
	rules, _ := list(record.Fields["Rules"])
	payment := rules[1].Args[1].Args[1].Args[0].Args[0]
	if payment.Name != "AnyPlayerMayCost" {
		t.Fatal("wrong real-record payment", payment)
	}
	continuation, err := Parse(strings.NewReader(`Unless(CostWasPaid,[GainLife(Integer(3))])`), "continuation.ron")
	if err != nil {
		t.Fatal(err)
	}
	build, err := compileActionSequence([]Node{payment, continuation[0]}, scope{})
	if err != nil {
		t.Fatal(err)
	}
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
	g := mage.NewGame(a, b)
	b.ManaPool().Add(core.Colorless, 2)
	if err := mage.ApplyEffect(g, build[0](instanceBindings{}), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 23 || a.offers != 0 || b.offers != 1 {
		t.Fatal("real payment did not use opponent")
	}
}
