package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONHistoryManaCards(t *testing.T) {
	for _, name := range []string{"Farrelite Priest", "Initiates of the Ebon Hand"} {
		t.Run(name, func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, "FEM", name)
			p := g.PutOnBattlefield(card, a.PlayerID())
			color := core.Black
			if name == "Farrelite Priest" {
				color = core.White
			}

			for i := range 5 {
				a.ManaPool().Clear()
				a.ManaPool().Add(core.Red, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				delays := max(0, i-2)
				if g.StackSize() != 0 || a.ManaPool().Count(color) != 1 || len(g.DelayedTriggers()) != delays {
					t.Fatal("mana timing, amount or threshold")
				}
			}
			g.RunStepWithPriority(core.EndStep)
			if g.FindPermanent(p.ID()) != nil {
				t.Fatal("missing sacrifice")
			}
		})
	}
}

func TestRONHistoryManaPaymentAndLifecycle(t *testing.T) {
	for _, name := range []string{"Farrelite Priest", "Initiates of the Ebon Hand"} {
		for _, change := range []string{"none", "clone", "return", "phase", "phase-cycle", "control", "remove", "after-end", "reset", "two"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "FEM", name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				mana := core.Black
				amount := 1
				if name == "Farrelite Priest" {
					mana = core.White
					amount = 3
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
					t.Fatal("unpaid cast")
				}
				a.ManaPool().Add(mana, amount)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				p := g.FindPermanent(card.ID())
				if p == nil {
					t.Fatal("casting entry")
				}
				if p.CurrentPower(g) != 1 || p.CurrentToughness(g) != map[string]int{"Farrelite Priest": 3, "Initiates of the Ebon Hand": 1}[name] {
					t.Fatal("metadata")
				}
				if change == "after-end" {
					g.RunStepWithPriority(core.EndStep)
				}
				a.ManaPool().Clear()
				a.ManaPool().Add(core.Red, 3)
				cost := core.ManaCost{Black: 3}
				if mana == core.White {
					cost = core.ManaCost{White: 3}
				}
				if !g.CanAfford(a.PlayerID(), cost, nil) || len(g.DelayedTriggers()) != 0 {
					t.Fatal("pure query")
				}
				if err := g.AutoTapForCost(a.PlayerID(), cost); err != nil {
					t.Fatal(err)
				}
				if len(g.DelayedTriggers()) != 0 || g.StackSize() != 0 || g.FindPermanent(p.ID()).Tapped {
					t.Fatal("first three conversions")
				}
				if change == "reset" {
					g.RunStepWithPriority(core.Cleanup)
					a.ManaPool().Add(core.Red, 1)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if len(g.DelayedTriggers()) != 0 {
						t.Fatal("cleanup retained history")
					}
					return
				}
				if change == "two" {
					q := g.PutOnBattlefield(repositoryCard(t, "FEM", name), a.PlayerID())
					a.ManaPool().Clear()
					a.ManaPool().Add(core.Red, 1)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if len(g.DelayedTriggers()) != 0 {
						t.Fatal("instances share history")
					}
				}
				a.ManaPool().Clear()
				a.ManaPool().Add(core.Red, 1)
				clone := g.Clone()
				one := core.ManaCost{Black: 1}
				if mana == core.White {
					one = core.ManaCost{White: 1}
				}
				if err := g.AutoTapForCost(a.PlayerID(), one); err != nil {
					t.Fatal(err)
				}
				if len(g.DelayedTriggers()) != 1 || len(clone.DelayedTriggers()) != 0 {
					t.Fatal("fourth activation or clone")
				}
				survives := false
				switch change {
				case "clone":
					if err := clone.AutoTapForCost(a.PlayerID(), one); err != nil {
						t.Fatal(err)
					}
					if len(clone.DelayedTriggers()) != 1 {
						t.Fatal("clone lost history")
					}
				case "return":
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
					survives = true
				case "phase":
					g.PhaseOut(p.ID())
					survives = true
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					survives = true
				case "remove":
					g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				case "after-end":
					g.RunStepWithPriority(core.Cleanup)
					if g.FindPermanent(p.ID()) == nil {
						t.Fatal("delay fired in old end step")
					}
				}
				g.RunStepWithPriority(core.EndStep)
				if (g.FindPermanentIncludingPhased(p.ID()) != nil) != survives {
					t.Fatal("delayed sacrifice identity")
				}
				if clone.FindPermanent(p.ID()) == nil {
					t.Fatal("clone shared departure")
				}
			})
		}
	}
}

func TestRONHistoryManaInvalid(t *testing.T) {
	history := `If(TimesThisAbilityHasBeenActivatedThisTurnIs(GreaterThanOrEqualTo(Integer(4))),[CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([SacrificePermanent(ThisPermanent)]))])`
	for _, body := range []string{
		`AddMana(AnyManaColor),` + history,
		`AddMana(ManaProduceB),If(TimesThisAbilityHasBeenActivatedThisTurnIs(GreaterThanOrEqualTo(Integer(4))),[SacrificePermanent(ThisPermanent)])`,
		`AddMana(ManaProduceB),` + history + `,DrawACard`,
		`AddMana(ManaProduceB),If(TimesThisAbilityHasBeenActivatedThisTurnIs(GreaterThanOrEqualTo(Integer(-1))),[CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([SacrificePermanent(ThisPermanent)]))])`,
		`AddMana(ManaProduceB),If(TimesThisAbilityHasBeenActivatedThisTurnIs(GreaterThanOrEqualTo(Integer(4))),[CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([SacrificePermanent(Ref_TargetPermanent)]))])`,
		`AddMana(ManaProduceB),If(TimesThisAbilityHasBeenActivatedThisTurnIs(GreaterThanOrEqualTo(Integer(4))),[CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + history + `]))])`,
	} {
		text := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([ManaCostGeneric(1)]),ActionList([` + body + `]))])`
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", body)
		}
	}
}

func TestRONHistoryManaCostRollback(t *testing.T) {
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "FEM", "Initiates of the Ebon Hand"), a.PlayerID())
	for range 3 {
		a.ManaPool().Clear()
		a.ManaPool().Add(core.Red, 1)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	a.ManaPool().Clear()
	a.ManaPool().Add(core.Red, 1)
	spell := mage.NewSorcery("Payment rollback", "{B}", mage.WithAdditionalCost(mage.LifePayCost(100)), mage.WithAbility(mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1)))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err == nil {
		t.Fatal("unpaid total cost accepted")
	}
	if a.ManaPool().Count(core.Red) != 1 || len(g.DelayedTriggers()) != 0 || g.StackSize() != 0 || len(a.Hand()) != 1 {
		t.Fatal("failed total cost changed live state")
	}
	if err := g.AutoTapForCost(a.PlayerID(), core.ManaCost{Black: 1}); err != nil {
		t.Fatal(err)
	}
	if len(g.DelayedTriggers()) != 1 {
		t.Fatal("failed payment changed history")
	}
}

func TestRONHistoryManaNativeSpellPayment(t *testing.T) {
	for _, tc := range []struct {
		name, spell string
		prior, red  int
	}{{"Farrelite Priest", "Armageddon", 3, 4}, {"Initiates of the Ebon Hand", "Sinkhole", 2, 2}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefield(repositoryCard(t, "FEM", tc.name), a.PlayerID())
			for range tc.prior {
				a.ManaPool().Clear()
				a.ManaPool().Add(core.Red, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			a.ManaPool().Clear()
			a.ManaPool().Add(core.Red, tc.red)
			land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
			spell := repositoryCard(t, "LEA", tc.spell)
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			var targets []uuid.UUID
			if tc.spell == "Sinkhole" {
				targets = []uuid.UUID{land.ID()}
			}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), targets, 0); err != nil {
				t.Fatal(err)
			}
			if g.StackSize() != 1 || a.ManaPool().TotalMana() != 0 || len(g.DelayedTriggers()) != 1 || g.FindPermanent(land.ID()) == nil {
				t.Fatal("casting payment timing")
			}
			g.ResolveStack()
			if g.FindPermanent(land.ID()) != nil || g.FindPermanent(p.ID()) == nil {
				t.Fatal("spell resolution")
			}
			g.RunStepWithPriority(core.EndStep)
			if g.FindPermanent(p.ID()) != nil {
				t.Fatal("payment omitted sacrifice delay")
			}
		})
	}
}

func TestRONHistoryManaInventoryAndStrictLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/FEM.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var records []string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Farrelite Priest"`) || strings.Contains(line, `Name: "Initiates of the Ebon Hand"`) {
			records = append(records, line)
		}
	}
	if len(records) != 2 {
		t.Fatal("complete records missing")
	}
	files := fstest.MapFS{"cards.ron": {Data: []byte(strings.Join(records, "\n"))}}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 2 || report.UniqueCards != 2 || len(report.Issues) != 0 {
		t.Fatal("complete records rejected")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || registry == nil {
		t.Fatal("test collection", err)
	}
	for _, name := range []string{"Farrelite Priest", "Initiates of the Ebon Hand"} {
		card := repositoryCard(t, "FEM", name)
		if err := mage.ValidateCardDefinition(name, card); err != nil {
			t.Fatal(err)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnknownRule])`)}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial registry published")
	}
}

func TestRONHistoryManaHistoryIdentity(t *testing.T) {
	for _, change := range []string{"control", "phase-cycle", "return", "phased", "removed", "forbidden"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "FEM", "Initiates of the Ebon Hand")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			for range 3 {
				a.ManaPool().Clear()
				a.ManaPool().Add(core.Red, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			actor := a
			want := 1
			illegal := false
			switch change {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				actor = b
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(card.ID())
				p = g.PutOnBattlefield(card, a.PlayerID())
				want = 0
			case "phased":
				g.PhaseOut(p.ID())
				illegal = true
			case "removed":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				illegal = true
			case "forbidden":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.AttrCantActivate)
				illegal = true
			}
			actor.ManaPool().Clear()
			actor.ManaPool().Add(core.Red, 1)
			err := g.AutoTapForCost(actor.PlayerID(), core.ManaCost{Black: 1})
			if illegal {
				if err == nil || actor.ManaPool().Count(core.Red) != 1 || len(g.DelayedTriggers()) != 0 {
					t.Fatal("illegal payment changed resources or history")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(g.DelayedTriggers()) != want || actor.ManaPool().Count(core.Black) != 1 {
				t.Fatal("history followed wrong incarnation or controller")
			}
			if want == 1 {
				g.RunStepWithPriority(core.EndStep)
				if g.FindPermanent(p.ID()) != nil {
					t.Fatal("new controller delay")
				}
			}
		})
	}
}
