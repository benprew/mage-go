package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCastingManaInspectionRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, state := range []string{"normal", "zero", "exile", "unpaid", "no creature", "opponent", "phased", "wrong type", "return", "countered", "copy", "clone", "text"} {
			t.Run(set+"/"+state, func(t *testing.T) {
				a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				card := repositoryCard(t, set, "Sacrifice")
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				cost, want := "{X}{3}{U}", 4
				if state == "zero" {
					cost, want = "", 0
				}
				payment := mage.NewCreature("Payment", cost, 1, 1, mage.WithKeyword(core.Indestructible))
				if state == "wrong type" {
					payment = mage.NewArtifact("Payment", cost)
				}
				payment.SetOwner(b.PlayerID())
				controller := a.PlayerID()
				if state == "opponent" {
					controller = b.PlayerID()
				}
				paid := g.PutOnBattlefield(payment, controller)
				a.choice = paid.ID()
				if state == "phased" {
					g.PhaseOut(paid.ID())
				}
				if state == "no creature" {
					g.RemoveFromBattlefield(paid)
				}
				if state == "exile" {
					g.AddExileIfWouldGoToGraveyardThisTurn(paid.ID(), paid.ID())
				}
				if state != "unpaid" {
					a.ManaPool().Add(core.Black, 1)
				}
				before := a.ManaPool().TotalMana()
				err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0)
				switch state {
				case "unpaid", "no creature", "opponent", "phased", "wrong type":
					if err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 || a.ManaPool().TotalMana() != before || len(b.Graveyard()) != 0 {
						t.Fatal("invalid payment changed state")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				graveyard := 1
				if state == "exile" {
					graveyard = 0
					if g.FindExiledCard(payment.ID()) == nil {
						t.Fatal("sacrifice destination was not replaced")
					}
				}
				if a.ManaPool().TotalMana() != 0 || len(b.Graveyard()) != graveyard || g.StackSize() != 1 {
					t.Fatal("payment or timing differs")
				}
				obj := g.StackPeek()
				if state == "return" {
					b.RemoveFromGraveyard(payment.ID())
					returned := g.PutOnBattlefield(payment, b.PlayerID())
					g.PhaseOut(returned.ID())
				}
				if state == "text" {
					change, _ := mage.ReplaceColorWord(core.Black, core.Red)
					if err := g.RevisePrintedSpellText(obj.ID, change); err != nil {
						t.Fatal(err)
					}
					obj = g.StackPeek()
				}
				if state == "copy" {
					obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
					if obj == nil {
						t.Fatal("copy failed")
					}
				}
				for range 2 {
					if certain, known := g.InspectSpellDestruction(obj, nil); certain || !known {
						t.Fatal("paid production inspection unknown")
					}
				}
				if a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != 0 {
					t.Fatal("inspection produced mana")
				}
				if state == "countered" {
					g.CounterStackObject(obj.ID)
					if a.ManaPool().TotalMana() != 0 || len(b.Graveyard()) != 1 {
						t.Fatal("counter changed payment")
					}
					return
				}
				if state == "clone" {
					clone := g.Clone()
					clone.ResolveStack()
					if clone.GetPlayer(a.PlayerID()).ManaPool().Count(core.Black) != want || a.ManaPool().TotalMana() != 0 {
						t.Fatal("clone shares pool")
					}
				}
				g.ResolveStack()
				if a.ManaPool().Count(core.Black) != want {
					t.Fatalf("mana = %d, want %d", a.ManaPool().Count(core.Black), want)
				}
				if state == "copy" && b.ManaPool().Count(core.Black) != want {
					t.Fatal("copy lost saved payment")
				}
			})
		}
	}
}

func TestRONCastingManaInspectionInvalid(t *testing.T) {
	for _, rule := range []string{
		`SpellActions(ActionList([AddManaRepeated(ManaProduceB,ManaValueOfTheSacrificedPermanent)]))`,
		`CastEffect(AdditionalCastingCost(SacrificeAPermanent(IsCardtype(Creature)))),SpellActions(ActionList([AddManaRepeated(ManaProduceB,ManaValueOfTheSacrificedPermanent())]))`,
		`CastEffect(AdditionalCastingCost(SacrificeAPermanent(IsCardtype(Creature)))),SpellActions(ActionList([AddManaRepeated(ManaProduceB,ToughnessOfTheSacrificedCreature),AddManaRepeated(ManaProduceB,ManaValueOfTheSacrificedPermanent)]))`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+rule+`])`)); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONCastingManaInspectionInvalidArithmetic(t *testing.T) {
	for _, amount := range []string{`Plus(Integer(1))`, `Plus(Integer(-1),ManaValueOfTheSacrificedPermanent)`, `Plus(Integer(1),ManaValueOfTheSacrificedPermanent())`, `Plus(Integer(1),TheNumberOfCountersRemovedThisWay)`, `Twice()`, `Twice(ManaValueOfTheSacrificedPermanent,Integer(1))`, `Plus(Integer(1),LifeTotalOfPlayer(You))`} {
		rule := `CastEffect(AdditionalCastingCost(SacrificeAPermanent(IsCardtype(Creature)))),SpellActions(ActionList([AddManaRepeated(ManaProduceB,` + amount + `)]))`
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+rule+`])`)); err == nil {
			t.Fatalf("accepted %s", amount)
		}
	}
}

func TestRONCastingManaInspectionArithmetic(t *testing.T) {
	for _, test := range []struct {
		amount string
		want   int
	}{
		{`Plus(Integer(1),ManaValueOfTheSacrificedPermanent)`, 4},
		{`Twice(Plus(Integer(1),ManaValueOfTheSacrificedPermanent))`, 8},
	} {
		factory := permanentScopeFactory(t, "Instant", `CastEffect(AdditionalCastingCost(SacrificeAPermanent(IsCardtype(Creature)))),SpellActions(ActionList([AddManaRepeated(ManaProduceR,`+test.amount+`)]))`)
		for range 2 {
			a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}
			g := mage.NewGame(a, mage.NewBasePlayer("B"))
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			paid := g.PutOnBattlefield(mage.NewCreature("Payment", "{3}", 1, 1), a.PlayerID())
			a.choice = paid.ID()
			a.ManaPool().Add(core.Black, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if certain, known := g.InspectSpellDestruction(g.StackPeek(), nil); certain || !known {
				t.Fatal("native arithmetic unknown")
			}
			g.ResolveStack()
			if a.ManaPool().Count(core.Red) != test.want {
				t.Fatal("wrong saved arithmetic")
			}
		}
	}
}

func TestRONCastingManaInspectionPending(t *testing.T) {
	a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	for _, cost := range []string{"{2}", "{5}"} {
		card := repositoryCard(t, "LEA", "Sacrifice")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		p := g.PutOnBattlefield(mage.NewCreature("Payment", cost, 1, 1), a.PlayerID())
		a.choice = p.ID()
		a.ManaPool().Add(core.Black, 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		if certain, known := g.InspectSpellDestruction(g.StackPeek(), nil); certain || !known {
			t.Fatal("pending payment unknown")
		}
	}
	clone := g.Clone()
	g.ResolveTopOfStack()
	if a.ManaPool().Count(core.Black) != 5 {
		t.Fatal("second spell used first payment")
	}
	if certain, known := g.InspectSpellDestruction(g.StackPeek(), nil); certain || !known {
		t.Fatal("first payment lost after another resolution")
	}
	g.ResolveStack()
	clone.ResolveStack()
	if a.ManaPool().Count(core.Black) != 7 || clone.GetPlayer(a.PlayerID()).ManaPool().Count(core.Black) != 7 {
		t.Fatal("pending spells shared values")
	}
}
