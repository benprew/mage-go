package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONEligiblePayerScopeGameplay(t *testing.T) {
	for _, state := range []string{"paid", "decline", "nested", "copy", "clone", "source-return", "source-phase", "source-control"} {
		t.Run(state, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: state != "decline"}
			g := mage.NewGame(a, b)
			paid := `PlayerAction(ActionPlayer,GainLife(Integer(3)))`
			if state == "nested" {
				paid = `AnyPlayerMayCost(Opponent,PayMana([ManaCostGeneric(9)])),Unless(CostWasPaid,[GainLife(Integer(1))]),PlayerAction(ActionPlayer,GainLife(Integer(3)))`
			}
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([AnyPlayerMayCost(Opponent,PayMana([ManaCostGeneric(2)])),IfElse(CostWasPaid,[`+paid+`],[GainLife(Integer(7))])]))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			b.ManaPool().Add(core.Colorless, 4)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "source-return":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(card, b.PlayerID())
			case "source-phase":
				g.PhaseOut(source.ID())
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			cloneLife := 27
			if state == "copy" {
				cloneLife = 34
			}
			if clone.GetPlayer(a.PlayerID()).Life() != cloneLife || clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("clone unpaid branch inherited payer")
			}
			g.ResolveStack()
			wantA, wantB := 20, 23
			if state == "decline" {
				wantA, wantB = 27, 20
			}
			if state == "nested" {
				wantA = 21
			}
			if state == "copy" {
				wantB = 26
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life=%d/%d want %d/%d", a.Life(), b.Life(), wantA, wantB)
			}
		})
	}
}

func TestRONEligiblePayerScopeInvalid(t *testing.T) {
	for _, body := range []string{
		`AnyPlayerMayCost(Opponent,PayMana([])),Unless(CostWasPaid,[PlayerAction(ActionPlayer,GainLife(Integer(1)))])`,
		`AnyPlayerMayCost(Opponent,PayMana([])),If(CostWasntPaid,[PlayerAction(ActionPlayer,GainLife(Integer(1)))])`,
		`AnyPlayerMayCost(Opponent,PayMana([])),IfElse(CostWasPaid,[GainLife(Integer(1))],[PlayerAction(ActionPlayer,GainLife(Integer(1)))])`,
		`AnyPlayerMayCost(Opponent,PayMana([])),If(CostWasPaid,[GainLife(Integer(1))]),PlayerAction(ActionPlayer,GainLife(Integer(1)))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + body + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted unpaid or escaped payer", body)
		}
	}
}

func TestRONEligiblePayerScopePolarityAndContexts(t *testing.T) {
	for _, clause := range []string{"If(CostWasPaid", "Unless(CostWasntPaid", "IfElse(CostWasPaid", "IfElse(CostWasntPaid"} {
		for _, context := range []string{"spell", "trigger", "delay-before", "delay-after"} {
			t.Run(clause+"/"+context, func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
				b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
				g := mage.NewGame(a, b)
				paid := `PlayerAction(ActionPlayer,GainLife(Integer(3)))`
				if context == "delay-after" {
					paid = `CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerAction(ActionPlayer,GainLife(Integer(3)))]))`
				}
				branch := clause + `,[` + paid + `])`
				if clause == "IfElse(CostWasPaid" {
					branch = clause + `,[` + paid + `],[GainLife(Integer(7))])`
				}
				if clause == "IfElse(CostWasntPaid" {
					branch = clause + `,[GainLife(Integer(7))],[` + paid + `])`
				}
				body := `ActionList([AnyPlayerMayCost(Opponent,PayMana([ManaCostGeneric(2)])),` + branch + `])`
				typ, rule := "Artifact", `Activated(PayMana([]),`+body+`)`
				if context == "spell" {
					typ, rule = "Instant", `SpellActions(`+body+`)`
				}
				if context == "trigger" {
					rule = `TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),` + body + `)`
				}
				if context == "delay-before" {
					rule = `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,` + body + `)]))`
				}
				card := permanentScopeFactory(t, typ, rule)()
				card.SetOwner(a.PlayerID())
				b.ManaPool().Add(core.Colorless, 2)
				if context == "spell" {
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
						t.Fatal(err)
					}
				} else {
					source := g.PutOnBattlefield(card, a.PlayerID())
					if context == "trigger" {
						g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
					} else if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if context == "delay-before" {
						g.ResolveStack()
						g.RemoveFromBattlefield(source)
						for range 3 {
							g.WaitingDelayedPermanentSources()
						}
						if b.offers != 0 {
							t.Fatal("waiting query offered payment")
						}
						g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
					}
				}
				if b.offers != 0 {
					t.Fatal("payment happened before resolution")
				}
				g.ResolveStack()
				if context == "delay-after" {
					if b.Life() != 20 {
						t.Fatal("delayed paid body ran early")
					}
					for _, p := range g.AllBattlefield() {
						g.RemoveFromBattlefield(p)
					}
					clone := g.Clone()
					clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
					clone.ResolveStack()
					if clone.GetPlayer(b.PlayerID()).Life() != 23 || b.Life() != 20 {
						t.Fatal("delayed clone lost payer or changed original")
					}
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
					g.ResolveStack()
				}
				if a.Life() != 20 || b.Life() != 23 || b.offers != 1 || a.offers != 0 {
					t.Fatal("wrong paid path, payer or controller", a.Life(), b.Life())
				}
			})
		}
	}
}

func TestRONEligiblePayerScopeOrderAndLoading(t *testing.T) {
	src := `Card(Name:"Payer",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([AnyPlayerMayCost(AnyPlayer,PayMana([])),If(CostWasPaid,[PlayerAction(ActionPlayer,GainLife(Integer(3)))])]))])`
	files := fstest.MapFS{"payer.ron": {Data: []byte(src)}}
	if _, err := LoadFS(files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(src), "payer.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, active := range []int{0, 1} {
		t.Run(fmt.Sprint(active), func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			g.SetActivePlayerIndex(active)
			first, second := factory(), factory()
			first.SetOwner(a.PlayerID())
			second.SetOwner(b.PlayerID())
			one, two := g.PutOnBattlefield(first, a.PlayerID()), g.PutOnBattlefield(second, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), one.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(b.PlayerID(), two.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveTopOfStack()
			g.SetActivePlayerIndex(1 - active)
			g.ResolveStack()
			if a.Life() != 23 || b.Life() != 23 || a.offers != 1 || b.offers != 1 {
				t.Fatal("payer followed controller, shared instance or escaped resolution", a.Life(), b.Life())
			}
		})
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.Replace(src, `Name:"Payer"`, `Name:"Invalid"`, 1), "If(CostWasPaid", "Unless(CostWasPaid"))}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("published partial registry with invalid unpaid payer")
	}
}
