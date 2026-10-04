package ron

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSacrificeValueRealCards(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		value     int
	}{{"ARN", "Diamond Valley", 4}, {"LEG", "Life Chisel", 4}, {"FEM", "Homarid Spawning Bed", 3}} {
		for _, change := range []string{"none", "leave", "return", "control", "phase"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
				b := mage.NewBasePlayer("Bob")
				g := mage.NewGame(a, b)
				g.SetStep(core.Upkeep)
				c := repositoryCard(t, tc.set, tc.name)
				c.SetOwner(a.PlayerID())
				if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
					t.Fatal(err)
				}
				source := g.PutOnBattlefield(c, a.PlayerID())
				victimCard := mage.NewCreature("Payment", "{2}{U}", 2, 2)
				victimCard.SetOwner(b.PlayerID())
				victim := g.PutOnBattlefield(victimCard, a.PlayerID())
				a.choice = victim.ID()
				victim.AddCounter(core.P1P1, 2)
				a.ManaPool().Add(core.Blue, 3)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if tc.name == "Diamond Valley" && !g.FindPermanent(source.ID()).Tapped {
					t.Fatal("tap cost missing")
				}
				if tc.name == "Homarid Spawning Bed" && a.ManaPool().TotalMana() != 0 {
					t.Fatal("mana cost missing")
				}
				if a.Life() != 20 || len(b.Graveyard()) != 1 || g.GetStack().Size() != 1 {
					t.Fatal("payment and resolution timing")
				}
				b.RemoveFromGraveyard(victimCard.ID())
				returned := g.PutOnBattlefield(victimCard, b.PlayerID())
				returned.AddCounter(core.P1P1, 8)
				switch change {
				case "leave", "return":
					g.BouncePermanentToHand(g.FindPermanent(source.ID()))
					if change == "return" {
						a.RemoveFromHand(c.ID())
						g.PutOnBattlefield(c, b.PlayerID())
					}
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "phase":
					g.PhaseOut(source.ID())
				}
				g.SetStep(core.Draw)
				g.SetActivePlayerIndex(1)
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				clone := g.Clone()
				g.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetStack().Size() != 2 {
					t.Fatal("resolution changed clone")
				}
				clone.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					if tc.name == "Homarid Spawning Bed" {
						n := 0
						for _, p := range game.FilterBattlefield(mage.IsToken) {
							if p.IsToken && p.ControllerID() == a.PlayerID() {
								n++
								if p.CurrentPower(game) != 1 || p.CurrentToughness(game) != 1 || !slices.Contains(p.Colors(), core.Blue) || !p.HasSubType("Camarid") {
									t.Fatal("token characteristics")
								}
							}
						}
						if n != 6 {
							t.Fatalf("tokens=%d", n)
						}
					} else if game.GetPlayer(a.PlayerID()).Life() != 28 || game.GetPlayer(b.PlayerID()).Life() != 20 {
						t.Fatal("saved toughness or controller lost")
					}
				}
			})
		}
	}
}

func TestRONSacrificeValuePendingAndRollback(t *testing.T) {
	a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
	b := mage.NewBasePlayer("Bob")
	g := mage.NewGame(a, b)
	c := sacrificeDeclaration(t, `SacrificeAPermanent(IsCardtype(Creature))`, `GainLife(ToughnessOfTheSacrificedCreature)`)
	source := g.PutOnBattlefield(c, a.PlayerID())
	for _, toughness := range []int{2, 5} {
		p := g.PutOnBattlefield(mage.NewCreature("Payment", "", 1, toughness), a.PlayerID())
		a.choice = p.ID()
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	if a.Life() != 27 {
		t.Fatal("pending payments shared a value")
	}
	for _, scenario := range []string{"mana", "phased", "opponent", "invalid-choice"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
			b := mage.NewBasePlayer("Bob")
			g := mage.NewGame(a, b)
			c := sacrificeDeclaration(t, `And([TapPermanent(ThisPermanent), SacrificeAPermanent(IsCardtype(Creature)), PayMana([ManaCostGeneric(1)])])`, `GainLife(ToughnessOfTheSacrificedCreature)`)
			s := g.PutOnBattlefield(c, a.PlayerID())
			controller := a.PlayerID()
			if scenario == "opponent" {
				controller = b.PlayerID()
			}
			v := g.PutOnBattlefield(mage.NewCreature("Payment", "", 1, 5), controller)
			a.choice = v.ID()
			if scenario == "phased" {
				g.PhaseOut(v.ID())
			}
			if scenario == "invalid-choice" {
				a.invalid = s
			}
			if scenario != "mana" {
				a.ManaPool().Add(core.Colorless, 1)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), s.ID(), 0, nil); err == nil {
				t.Fatal("invalid payment accepted")
			}
			if g.FindPermanentIncludingPhased(v.ID()) == nil || g.FindPermanent(s.ID()).Tapped || !g.GetStack().IsEmpty() || a.Life() != 20 {
				t.Fatal("failed cost changed state")
			}
		})
	}
}

func TestRONSacrificeValueInvalid(t *testing.T) {
	for _, rule := range []string{
		`Activated(TapPermanent(ThisPermanent), ActionList([GainLife(ToughnessOfTheSacrificedCreature)]))`,
		`Activated(SacrificeAPermanent(IsCardtype(Artifact)), ActionList([GainLife(ToughnessOfTheSacrificedCreature)]))`,
		`Activated(And([SacrificeAPermanent(IsCardtype(Creature)),SacrificeAPermanent(IsCardtype(Creature))]), ActionList([GainLife(ToughnessOfTheSacrificedCreature)]))`,
		`Activated(SacrificeAPermanent(IsCardtype(Creature)), ActionList([GainLife(ToughnessOfTheSacrificedCreature()),GainLife(ManaValueOfTheSacrificedPermanent)]))`,
		`SpellActions(ActionList([GainLife(ToughnessOfTheSacrificedCreature)]))`,
		`Activated(And([SacrificePermanent(ThisPermanent),SacrificeAPermanent(IsCardtype(Creature))]), ActionList([GainLife(ToughnessOfTheSacrificedCreature)]))`,
		`Activated(SacrificeAPermanent(IsCardtype(Creature)),ActionList([GainLife(ManaValueOfTheSacrificedPermanent),GainLife(ToughnessOfTheSacrificedCreature)]))`,
		`Activated(SacrificeAPermanent(IsCardtype(Creature)), ActionList([GainLife(Integer(1))])), Activated(TapPermanent(ThisPermanent), ActionList([GainLife(ManaValueOfTheSacrificedPermanent)]))`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONSacrificeValueFactoriesAndSelfPayment(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Offering",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:2),Rules:[Activated(SacrificeAPermanent(IsCardtype(Creature)),ActionList([GainLife(ToughnessOfTheSacrificedCreature)]))])`), "offering.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"independent", "self", "zero", "negative"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
			b := mage.NewBasePlayer("Bob")
			g := mage.NewGame(a, b)
			first := g.PutOnBattlefield(factory(), a.PlayerID())
			second := g.PutOnBattlefield(factory(), a.PlayerID())
			a.choice = first.ID()
			first.AddCounter(core.P1P1, 2)
			if scenario == "zero" {
				first.AddCounter(core.M1M1, 4)
			}
			if scenario == "negative" {
				first.AddCounter(core.M1M1, 5)
			}
			source := second
			if scenario == "self" {
				source = first
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if scenario == "independent" {
				v := g.PutOnBattlefield(mage.NewCreature("Other payment", "", 1, 7), a.PlayerID())
				a.choice = v.ID()
				if err := g.ActivateAbilityByIndex(a.PlayerID(), second.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			want := 24
			if scenario == "independent" {
				want = 31
			}
			if scenario == "zero" || scenario == "negative" {
				want = 20
			}
			if a.Life() != want {
				t.Fatalf("life %d want %d", a.Life(), want)
			}
		})
	}
}

func TestRONSacrificeValueRealRestrictions(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "LEG", "Life Chisel")
	source := g.PutOnBattlefield(c, a.PlayerID())
	v := g.PutOnBattlefield(mage.NewCreature("Payment", "", 1, 3), a.PlayerID())
	for _, scenario := range []string{"step", "turn", "permission", "phased"} {
		g.SetStep(core.Upkeep)
		g.SetActivePlayerIndex(0)
		who := a.PlayerID()
		switch scenario {
		case "step":
			g.SetStep(core.Draw)
		case "turn":
			g.SetActivePlayerIndex(1)
		case "permission":
			who = b.PlayerID()
		case "phased":
			g.PhaseOut(source.ID())
		}
		if err := g.ActivateAbilityByIndex(who, source.ID(), 0, nil); err == nil {
			t.Fatal("illegal activation accepted")
		}
		if g.FindPermanent(v.ID()) == nil || !g.GetStack().IsEmpty() {
			t.Fatal("illegal activation paid")
		}
	}
}

func TestRONSacrificeValueCopyController(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "ARN", "Diamond Valley")
	source := g.PutOnBattlefield(c, a.PlayerID())
	g.PutOnBattlefield(mage.NewCreature("Payment", "", 1, 5), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if a.Life() != 25 || b.Life() != 25 {
		t.Fatal("copy lost saved value or used original controller")
	}
}

func TestRONSacrificeValueConsumers(t *testing.T) {
	for _, value := range []string{"ToughnessOfTheSacrificedCreature", "ManaValueOfTheSacrificedPermanent"} {
		for _, action := range []string{"DrawNumberCards", "GainLife", "LoseLife", "tokens"} {
			t.Run(value+"/"+action, func(t *testing.T) {
				g, a, _ := ronGame()
				fillRONLibrary(t, a, 4)
				text := fmt.Sprintf("%s(%s)", action, value)
				if action == "tokens" {
					text = fmt.Sprintf("CreateTokens([NumberTokens(%s, %s)], [])", value, saprolingRON)
				}
				c := sacrificeDeclaration(t, `SacrificeAPermanent(IsCardtype(Creature))`, text)
				source := g.PutOnBattlefield(c, a.PlayerID())
				g.PutOnBattlefield(mage.NewCreature("Payment", "{3}", 1, 3), a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				switch action {
				case "DrawNumberCards":
					if len(a.Hand()) != 3 {
						t.Fatal("draw count")
					}
				case "GainLife":
					if a.Life() != 23 {
						t.Fatal("life gain")
					}
				case "LoseLife":
					if a.Life() != 17 {
						t.Fatal("life loss")
					}
				case "tokens":
					if len(g.FilterBattlefield(mage.IsToken)) != 3 {
						t.Fatal("token count")
					}
				}
			})
		}
	}
}

func TestRONSacrificeValueTargetLegality(t *testing.T) {
	for _, change := range []string{"none", "leave", "return", "phase"} {
		t.Run(change, func(t *testing.T) {
			a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
			b := mage.NewBasePlayer("Bob")
			g := mage.NewGame(a, b)
			nodes, err := Parse(strings.NewReader(`Card(Name:"Offering",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(SacrificeAPermanent(IsCardtype(Creature)),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([GainLife(ToughnessOfTheSacrificedCreature)])))])`), "offering.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			c := factory()
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(c, a.PlayerID())
			victim := g.PutOnBattlefield(mage.NewCreature("Payment", "", 1, 4), a.PlayerID())
			a.choice = victim.ID()
			targetCard := mage.NewCreature("Target", "", 1, 2)
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{uuid.New()}); err == nil || g.FindPermanent(victim.ID()) == nil {
				t.Fatal("invalid target paid cost")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "leave", "return":
				g.RemoveFromBattlefield(target)
				if change == "return" {
					g.PutOnBattlefield(targetCard, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(target.ID())
			}
			g.ResolveStack()
			want := 20
			if change == "none" {
				want = 24
			}
			if a.Life() != want {
				t.Fatal("target legality lost")
			}
		})
	}
}
