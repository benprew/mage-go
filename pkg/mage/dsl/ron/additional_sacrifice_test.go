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

func TestRONAdditionalSacrificeRealCards(t *testing.T) {
	for _, name := range []string{"Goblin Grenade", "Sacrifice"} {
		for _, change := range []string{"none", "return", "clone-copy"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
				b := mage.NewBasePlayer("Bob")
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				set, color := "FEM", core.Red
				if name == "Sacrifice" {
					set, color = "LEA", core.Black
				}
				spell := repositoryCard(t, set, name)
				spell.SetOwner(a.PlayerID())
				if err := mage.ValidateCardDefinition(name, spell); err != nil {
					t.Fatal(err)
				}
				a.AddToHand(spell)
				paid := mage.NewCreature("Borrowed Goblin", "{X}{2}{G}", 2, 2, mage.WithSubTypes("Goblin"), mage.WithKeyword(core.Indestructible))
				paid.SetOwner(b.PlayerID())
				p := g.PutOnBattlefield(paid, a.PlayerID())
				p.AddCounter(core.P1P1, 4)
				a.choice = p.ID()
				a.ManaPool().Add(color, 1)
				var targets []uuid.UUID
				if name == "Goblin Grenade" {
					targets = []uuid.UUID{b.PlayerID()}
				}
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), targets, 0); err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().TotalMana() != 0 || g.FindPermanent(p.ID()) != nil || len(b.Graveyard()) != 1 || g.GetStack().Size() != 1 || b.Life() != 20 {
					t.Fatal("casting did not pay before resolution")
				}
				if change == "return" {
					b.RemoveFromGraveyard(paid.ID())
					g.PutOnBattlefield(paid, b.PlayerID())
				}
				copies := 1
				if change == "clone-copy" {
					if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					copies = 2
				}
				clone := g.Clone()
				g.ResolveStack()
				if clone.GetStack().Size() != copies || clone.GetPlayer(b.PlayerID()).Life() != 20 || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
					t.Fatal("resolution changed clone")
				}
				clone.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					if name == "Sacrifice" {
						if got := game.GetPlayer(a.PlayerID()).ManaPool().TotalMana(); got != 3*copies {
							t.Fatalf("saved mana value: got %d, want %d", got, 3*copies)
						}
					} else if game.GetPlayer(b.PlayerID()).Life() != 20-5*copies {
						t.Fatal("damage amount")
					}
				}
			})
		}
	}
}

func TestRONAdditionalSacrificeRollback(t *testing.T) {
	for _, scenario := range []string{"mana", "opponent", "phase", "wrong-type", "invalid-choice", "illegal-target", "sorcery-timing"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
			b := mage.NewBasePlayer("Bob")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			spell := repositoryCard(t, "FEM", "Goblin Grenade")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			paid := mage.NewCreature("Payment", "", 1, 2, mage.WithSubTypes("Goblin"))
			paid.SetOwner(a.PlayerID())
			controller := a.PlayerID()
			if scenario == "opponent" {
				controller = b.PlayerID()
			}
			if scenario == "wrong-type" {
				paid = mage.NewCreature("Elf", "", 1, 2, mage.WithSubTypes("Elf"))
				paid.SetOwner(a.PlayerID())
			}
			p := g.PutOnBattlefield(paid, controller)
			a.choice = p.ID()
			if scenario == "phase" {
				g.PhaseOut(p.ID())
			}
			if scenario == "invalid-choice" {
				a.invalid = g.PutOnBattlefield(mage.NewArtifact("Invalid", ""), a.PlayerID())
			}
			if scenario != "mana" {
				a.ManaPool().Add(core.Red, 1)
			}
			target := b.PlayerID()
			if scenario == "illegal-target" {
				target = g.PutOnBattlefield(mage.NewArtifact("Target", ""), b.PlayerID()).ID()
			}
			if scenario == "sorcery-timing" {
				g.SetStep(core.Upkeep)
			}
			mana := a.ManaPool().TotalMana()
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target}, 0); err == nil {
				t.Fatal("invalid cast accepted")
			}
			if g.FindPermanentIncludingPhased(p.ID()) == nil || a.ManaPool().TotalMana() != mana || len(a.Hand()) != 1 || len(a.Graveyard()) != 0 || !g.GetStack().IsEmpty() || b.Life() != 20 {
				t.Fatal("failed cast changed state")
			}
		})
	}
}

func TestRONAdditionalSacrificeDeclarations(t *testing.T) {
	for _, rules := range []string{
		`CastEffect(AdditionalCastingCost()), SpellActions(ActionList([GainLife(Integer(1))]))`,
		`CastEffect(AdditionalCastingCost(SacrificeAPermanent(And([])))), SpellActions(ActionList([GainLife(Integer(1))]))`,
		`CastEffect(AdditionalCastingCost(SacrificeAPermanent(Ref_TargetPermanent))), SpellActions(ActionList([GainLife(Integer(1))]))`,
		`CastEffect(AdditionalCastingCost(SacrificeAPermanent(IsCardtype(Creature)))), CastEffect(AdditionalCastingCost(SacrificeAPermanent(IsCardtype(Creature)))), SpellActions(ActionList([GainLife(Integer(1))]))`,
		`CastEffect(AdditionalCastingCost(TapPermanent(ThisPermanent))), SpellActions(ActionList([GainLife(Integer(1))]))`,
		`SpellActions(ActionList([AddManaRepeated(ManaProduceB,ManaValueOfTheSacrificedPermanent)]))`,
		`CastEffect(AdditionalCastingCost(SacrificeAPermanent(IsCardtype(Creature)))), SpellActions(ActionList([AddManaRepeated(ManaProduceB,ManaValueOfTheSacrificedPermanent())]))`,
		`CastEffect(AdditionalCastingCost(SacrificeAPermanent(IsCardtype(Artifact)))), SpellActions(ActionList([GainLife(ToughnessOfTheSacrificedCreature)]))`,
		`CastEffect(AdditionalCastingCost(SacrificeAPermanent(IsCardtype(Creature)),PayMana([]))), SpellActions(ActionList([GainLife(Integer(1))]))`,
		`CastEffect(AdditionalCastingCost(SacrificeAPermanent(IsCardtype(Creature)))), SpellActions(Modal_ChooseOne([ActionList([GainLife(Integer(1))]),ActionList([GainLife(Integer(2))])]))`,
		`CastEffect(AdditionalCastingCost(SacrificeAPermanent(IsCardtype(Creature)))), SpellActions(ActionList([AddManaRepeated(AnyManaColor,ManaValueOfTheSacrificedPermanent)]))`,
		`CastEffect(AdditionalCastingCost(SacrificeAPermanent(IsCardtype(Creature)))), SpellActions(ActionList([AddManaRepeated(And([ManaProduceB]),ManaValueOfTheSacrificedPermanent)]))`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[%s])`, rules)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rules)
		}
	}
}

func TestRONAdditionalSacrificePendingValues(t *testing.T) {
	a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
	b := mage.NewBasePlayer("Bob")
	g := mage.NewGame(a, b)
	for _, value := range []int{2, 5} {
		spell := repositoryCard(t, "LEA", "Sacrifice")
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		card := mage.NewCreature("Payment", fmt.Sprintf("{%d}", value), 1, 1)
		card.SetOwner(a.PlayerID())
		paid := g.PutOnBattlefield(card, a.PlayerID())
		a.choice = paid.ID()
		a.ManaPool().Add(core.Black, 1)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveTopOfStack()
	if b.ManaPool().TotalMana() != 5 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("copy did not keep payment and new controller")
	}
	g.ResolveTopOfStack()
	if a.ManaPool().TotalMana() != 5 {
		t.Fatal("second payment changed")
	}
	g.ResolveTopOfStack()
	if a.ManaPool().TotalMana() != 7 || len(a.Graveyard()) != 4 {
		t.Fatal("pending values or copy payment changed")
	}
}

func TestRONAdditionalSacrificeGrenadeTargets(t *testing.T) {
	for _, change := range []string{"creature", "planeswalker", "self-payment", "return", "phase", "phase-cycle", "shroud", "control", "prevent", "counter"} {
		t.Run(change, func(t *testing.T) {
			a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
			b := mage.NewBasePlayer("Bob")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			spell := repositoryCard(t, "FEM", "Goblin Grenade")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			payment := mage.NewCreature("Payment", "", 1, 2, mage.WithSubTypes("Goblin"))
			payment.SetOwner(a.PlayerID())
			paid := g.PutOnBattlefield(payment, a.PlayerID())
			a.choice = paid.ID()
			card := mage.NewCreature("Target", "", 2, 9)
			if change == "planeswalker" {
				card = mage.NewPlaneswalker("Target", "", 9)
			}
			card.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(card, b.PlayerID())
			targetID := target.ID()
			if change == "self-payment" {
				targetID = paid.ID()
			}
			a.ManaPool().Add(core.Red, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{targetID}, 0); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.BouncePermanentToHand(target)
				b.RemoveFromHand(card.ID())
				target = g.PutOnBattlefield(card, b.PlayerID())
			case "phase", "phase-cycle":
				phased := g.PhaseOut(target.ID())
				if change == "phase-cycle" {
					g.PhaseIn(phased)
				}
			case "shroud":
				g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: paid.ID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "prevent":
				if err := mage.ApplyEffect(g, mage.PreventNextDamage(mage.This, mage.Fixed(3)), target.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "counter":
				g.CounterStackObject(g.StackPeek().ID)
			}
			g.ResolveStack()
			if len(a.Graveyard()) != 2 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("resolution refunded payment")
			}
			if change == "planeswalker" {
				if g.FindPermanent(target.ID()).Counters[core.Loyalty] != 4 {
					t.Fatal("planeswalker damage")
				}
				return
			}
			want := 5
			if change == "prevent" {
				want = 2
			}
			if change == "self-payment" || change == "return" || change == "phase" || change == "shroud" || change == "counter" {
				want = 0
			}
			if got := g.FindPermanentIncludingPhased(target.ID()).Damage; got != want {
				t.Fatalf("damage=%d, want %d", got, want)
			}
		})
	}
}

func TestRONAdditionalSacrificeZeroAndRecast(t *testing.T) {
	for _, value := range []int{0, 3} {
		t.Run(fmt.Sprint(value), func(t *testing.T) {
			a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
			b := mage.NewBasePlayer("Bob")
			g := mage.NewGame(a, b)
			spell := repositoryCard(t, "LEA", "Sacrifice")
			spell.SetOwner(a.PlayerID())
			for i := range 2 {
				a.AddToHand(spell)
				card := mage.NewCreature("Payment", fmt.Sprintf("{%d}", value+i), 1, 2)
				card.SetOwner(a.PlayerID())
				paid := g.PutOnBattlefield(card, a.PlayerID())
				a.choice = paid.ID()
				a.ManaPool().Add(core.Black, 1)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				a.RemoveFromGraveyard(spell.ID())
			}
			if a.ManaPool().TotalMana() != 2*value+1 {
				t.Fatal("recast reused prior payment")
			}
		})
	}
}

func TestRONAdditionalSacrificeStrictLoadAndInventory(t *testing.T) {
	source := `Card(Name:"Offering",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[CastEffect(AdditionalCastingCost(SacrificeAPermanent(IsCardtype(Creature)))),SpellActions(ActionList([AddManaRepeated(ManaProduceB,ManaValueOfTheSacrificedPermanent)]))])`
	files := fstest.MapFS{"offering.ron": {Data: []byte(source)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || registry == nil {
		t.Fatalf("load: %v", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"AdditionalCastingCost": "casting_restriction", "SacrificeAPermanent": "cost", "IsCardtype": "permanent_filter", "ManaValueOfTheSacrificedPermanent": "value"} {
		found := false
		for _, entry := range report.Signatures {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing role %s/%s", name, role)
		}
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(source, "Offering", "Invalid") + ` Card(Name:"Rejected",Rules:[Unknown])`)}
	if registry, err = LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("invalid collection published a registry")
	}
}

func TestRONAdditionalSacrificeFactoriesAndQueries(t *testing.T) {
	source := `Card(Name:"Offering",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:[ManaCostB],Rules:[CastEffect(AdditionalCastingCost(SacrificeAPermanent(IsCardtype(Creature)))),SpellActions(ActionList([GainLife(ToughnessOfTheSacrificedCreature)]))])`
	nodes, err := Parse(strings.NewReader(source), "offering.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	first, second := factory(), factory()
	if first.(*mage.BaseCard).AdditionalCosts()[0] == second.(*mage.BaseCard).AdditionalCosts()[0] || first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("factories shared cost or ability declarations")
	}
	a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
	b := mage.NewBasePlayer("Bob")
	g := mage.NewGame(a, b)
	for i, spell := range []mage.Card{first, second} {
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		card := mage.NewCreature("Payment", "", 1, 2+i)
		card.SetOwner(a.PlayerID())
		paid := g.PutOnBattlefield(card, a.PlayerID())
		a.choice = paid.ID()
		a.ManaPool().Add(core.Black, 1)
		choices := a.choices
		for range 3 {
			g.GetCastableSpells(a.PlayerID())
		}
		if a.choices != choices || a.ManaPool().TotalMana() != 1 || g.FindPermanent(paid.ID()) == nil {
			t.Fatal("query changed payment state")
		}
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	if a.Life() != 25 {
		t.Fatal("factories shared captured toughness")
	}
	for _, color := range []string{"W", "U", "B", "R", "G", "C"} {
		text := strings.Replace(source, "GainLife(ToughnessOfTheSacrificedCreature)", "AddManaRepeated(ManaProduce"+color+",ManaValueOfTheSacrificedPermanent)", 1)
		nodes, err := Parse(strings.NewReader(text), "color.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		if err := mage.ValidateCardDefinition("Offering", factory()); err != nil {
			t.Fatal(err)
		}
	}
	permanent := strings.Replace(source, "Cardtypes:[Instant]", "Cardtypes:[Artifact]", 1)
	nodes, err = Parse(strings.NewReader(permanent), "permanent.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("additional sacrifice on permanent accepted")
	}
}
