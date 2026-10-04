package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

var sacrificeCards = []struct {
	set, name, payment, target string
	tap                        bool
}{
	{"3ED", "Atog", "Ornithopter", "", false},
	{"LEG", "Fallen Angel", "Llanowar Elves", "", false},
	{"DRK", "Dark Heart of the Wood", "Forest", "", false},
	{"FEM", "Goblin Chirurgeon", "Goblin Balloon Brigade", "Llanowar Elves", false},
	{"LEG", "Horror of Horrors", "Swamp", "Black Knight", false},
	{"ATQ", "Gate to Phyrexia", "Llanowar Elves", "Ornithopter", false},
	{"ATQ", "Orcish Mechanics", "Ornithopter", "player", true},
	{"ATQ", "Sage of Lat-Nam", "Ornithopter", "", true},
	{"ATQ", "Ashnod's Altar", "Llanowar Elves", "", false},
}

func TestRONSacrificeCostRealCards(t *testing.T) {
	for _, tc := range sacrificeCards {
		for _, change := range []string{"none", "leave", "return", "control", "phase"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
				b := mage.NewBasePlayer("Bob")
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				fillRONLibrary(t, a.BasePlayer, 3)
				c := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
					t.Fatal(err)
				}
				c.SetOwner(b.PlayerID())
				source := g.PutOnBattlefield(c, a.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				g.SetStep(core.Upkeep)
				pay := sacrificeFixture(t, tc.payment)
				pay.SetOwner(b.PlayerID())
				victim := g.PutOnBattlefield(pay, a.PlayerID())
				a.choice = victim.ID()
				g.AddRegenerationShield(victim.ID())
				var targets []uuid.UUID
				var target *mage.Permanent
				if tc.target == "player" {
					targets = []uuid.UUID{b.PlayerID()}
				} else if tc.target != "" {
					card := sacrificeFixture(t, tc.target)
					card.SetOwner(b.PlayerID())
					target = g.PutOnBattlefield(card, b.PlayerID())
					targets = []uuid.UUID{target.ID()}
				}
				power, toughness := source.CurrentPower(g), source.CurrentToughness(g)
				before := g.Clone()
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(victim.ID()) != nil || len(b.Graveyard()) != 1 || len(a.Graveyard()) != 0 {
					t.Fatal("cost did not sacrifice to owner before resolution")
				}
				if before.FindPermanent(victim.ID()) == nil || len(before.GetPlayer(b.PlayerID()).Graveyard()) != 0 {
					t.Fatal("payment changed prior clone")
				}
				if tc.name == "Ashnod's Altar" {
					if !g.GetStack().IsEmpty() || a.ManaPool().TotalMana() != 2 {
						t.Fatal("mana ability was not immediate")
					}
					return
				}
				if g.GetStack().Size() != 1 || a.Life() != 20 || len(a.Hand()) != 0 || b.Life() != 20 {
					t.Fatal("effect ran at payment")
				}
				if tc.tap && !g.FindPermanent(source.ID()).Tapped {
					t.Fatal("tap cost missing")
				}
				switch change {
				case "leave", "return":
					g.BouncePermanentToHand(g.FindPermanent(source.ID()))
					if change == "return" {
						b.RemoveFromHand(c.ID())
						g.PutOnBattlefield(c, a.PlayerID())
					}
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "phase":
					g.PhaseOut(source.ID())
				}
				otherHand := len(b.Hand())
				clone := g.Clone()
				g.ResolveStack()
				clone.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					switch tc.name {
					case "Atog", "Fallen Angel":
						p := game.FindPermanent(source.ID())
						if p == nil {
							continue
						}
						dp, dt := 2, 2
						if tc.name == "Fallen Angel" {
							dt = 1
							if !p.HasAttr(core.Flying) {
								t.Fatal("flying missing")
							}
						}
						if change == "return" || change == "phase" {
							dp, dt = 0, 0
						}
						if p.CurrentPower(game) != power+dp || p.CurrentToughness(game) != toughness+dt {
							t.Fatal("bonus used wrong incarnation")
						}
					case "Dark Heart of the Wood":
						if game.GetPlayer(a.PlayerID()).Life() != 23 || game.GetPlayer(b.PlayerID()).Life() != 20 {
							t.Fatal("life followed owner or changed controller")
						}
					case "Goblin Chirurgeon", "Horror of Horrors":
						if !game.HasRegenerationShield(target.ID()) {
							t.Fatal("shield missing")
						}
						game.DestroyPermanent(game.FindPermanent(target.ID()))
						if game.FindPermanent(target.ID()) == nil {
							t.Fatal("shield failed")
						}
					case "Gate to Phyrexia":
						if game.FindPermanent(target.ID()) != nil {
							t.Fatal("artifact survived")
						}
					case "Orcish Mechanics":
						if game.GetPlayer(b.PlayerID()).Life() != 18 {
							t.Fatal("damage missing after source change")
						}
					case "Sage of Lat-Nam":
						if len(game.GetPlayer(a.PlayerID()).Hand()) != 1 || len(game.GetPlayer(b.PlayerID()).Hand()) != otherHand {
							t.Fatal("draw used wrong player")
						}
					}
				}
				g.RunStepWithPriority(core.Cleanup)
				if p := g.FindPermanent(source.ID()); p != nil && change != "phase" && (tc.name == "Atog" || tc.name == "Fallen Angel") && (p.CurrentPower(g) != power || p.CurrentToughness(g) != toughness) {
					t.Fatal("bonus survived cleanup")
				}
			})
		}
	}
}

func TestRONSacrificeCostSourceAndTargets(t *testing.T) {
	for _, name := range []string{"Fallen Angel", "Goblin Chirurgeon"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			set := "LEG"
			if name == "Goblin Chirurgeon" {
				set = "FEM"
			}
			c := repositoryCard(t, set, name)
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			var targets []uuid.UUID
			if name == "Goblin Chirurgeon" {
				targets = []uuid.UUID{p.ID()}
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, targets); err != nil {
				t.Fatal("source cannot pay", err)
			}
			if g.FindPermanent(p.ID()) != nil || len(a.Graveyard()) != 1 {
				t.Fatal("source sacrifice missing")
			}
			g.PutOnBattlefield(c, b.PlayerID())
			g.ResolveStack()
			returned := g.FindPermanent(c.ID())
			if returned.CurrentPower(g) != c.Power() || g.HasRegenerationShield(returned.ID()) {
				t.Fatal("effect followed returned source or target")
			}
		})
	}
	for _, change := range []string{"leave", "return", "phase", "noncreature"} {
		t.Run("target/"+change, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "FEM", "Goblin Chirurgeon")
			p := g.PutOnBattlefield(c, a.PlayerID())
			targetCard := repositoryCard(t, "LEB", "Llanowar Elves")
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
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
			case "noncreature":
				g.AddContinuousEffect(mage.FuncContinuousEffect(core.LayerType, core.EndOfTurn, func(game *mage.Game, _ uuid.UUID) error {
					game.GetEffects().RevokeAttr(target.ID(), core.AttrIsCreature)
					return nil
				}))
				g.ApplyContinuousEffects()
			}
			g.ResolveStack()
			if g.HasRegenerationShield(target.ID()) {
				t.Fatal("illegal target got shield")
			}
		})
	}
}

func TestRONSacrificeCostInvalid(t *testing.T) {
	for _, cost := range []string{"SacrificeAPermanent", "SacrificeAPermanent()", "SacrificeAPermanent(AnyPermanent, AnyPermanent)", "SacrificeAPermanent(ThisPermanent)", "SacrificeAPermanent(Ref_TargetPermanent)", "SacrificeAPermanent(And([]))", "SacrificeAPermanent(Or([]))", "SacrificeAPermanent(And([IsCardtype(Creature), Or([])]))", "SacrificeAPermanent(Other(ThisPermanent))"} {
		t.Run(cost, func(t *testing.T) {
			text := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(%s, ActionList([GainLife(Integer(1))]))])`, cost)
			nodes, err := Parse(strings.NewReader(text), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Compile(nodes[0]); err == nil {
				t.Fatal("invalid cost accepted")
			}
		})
	}
	for _, rule := range []string{`SpellActions(ActionList([SacrificeAPermanent(IsCardtype(Creature))]))`, `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)), ActionList([MayCost(SacrificeAPermanent(IsCardtype(Creature))), If(CostWasPaid, [GainLife(Integer(1))])]))`, `Activated(SacrificeAPermanent(IsCardtype(Artifact)), ActionList([GainLife(ToughnessOfTheSacrificedCreature)]))`} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("unsupported action/payment/result accepted")
		}
	}
}

type ronSacrificeChooser struct {
	*mage.BasePlayer
	choice  uuid.UUID
	choices int
	invalid *mage.Permanent
}

func (p *ronSacrificeChooser) ChoosePermanent(candidates []*mage.Permanent, _ string, _ mage.GameReader) *mage.Permanent {
	p.choices++
	if p.invalid != nil {
		return p.invalid
	}
	for _, c := range candidates {
		if c.ID() == p.choice {
			return c
		}
	}
	return candidates[0]
}

func sacrificeDeclaration(t *testing.T, cost, action string) mage.Card {
	t.Helper()
	text := fmt.Sprintf(`Card(Name: "Offering", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(%s, ActionList([%s]))])`, cost, action)
	nodes, err := Parse(strings.NewReader(text), "offering.ron")
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
	return c
}

func TestRONSacrificeCostRollbackAndChoice(t *testing.T) {
	for _, scenario := range []string{"success", "phased", "opponent", "wrong-type", "invalid-choice", "mana", "double"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
			b := mage.NewBasePlayer("Bob")
			g := mage.NewGame(a, b)
			cost := `And([PayMana([ManaCostGeneric(1)]), TapPermanent(ThisPermanent), SacrificeAPermanent(IsCardtype(Creature))])`
			if scenario == "double" {
				cost = `And([PayMana([ManaCostGeneric(1)]), TapPermanent(ThisPermanent), SacrificeAPermanent(IsCardtype(Creature)), SacrificeAPermanent(IsCardtype(Creature))])`
			}
			c := sacrificeDeclaration(t, cost, `GainLife(Integer(1))`)
			source := g.PutOnBattlefield(c, a.PlayerID())
			card := repositoryCard(t, "LEB", "Llanowar Elves")
			if scenario == "wrong-type" {
				card = repositoryCard(t, "LEB", "Forest")
			}
			card.SetOwner(b.PlayerID())
			controller := a.PlayerID()
			if scenario == "opponent" {
				controller = b.PlayerID()
			}
			victim := g.PutOnBattlefield(card, controller)
			a.choice = victim.ID()
			if scenario == "phased" {
				g.PhaseOut(victim.ID())
			}
			if scenario == "invalid-choice" {
				a.invalid = source
			}
			if scenario != "mana" {
				a.ManaPool().Add(core.Colorless, 1)
			}
			before := g.Clone()
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			if scenario == "success" {
				if err != nil {
					t.Fatal(err)
				}
				if a.choices != 1 || g.FindPermanent(victim.ID()) != nil || !g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
					t.Fatal("payment did not keep the choice")
				}
				g.ResolveStack()
				if a.Life() != 21 {
					t.Fatal("paid effect missing")
				}
			} else {
				if err == nil {
					t.Fatal("invalid total payment accepted")
				}
				if g.FindPermanentIncludingPhased(victim.ID()) == nil || g.FindPermanent(source.ID()).Tapped || a.Life() != 20 || !g.GetStack().IsEmpty() || a.ManaPool().TotalMana() != before.GetPlayer(a.PlayerID()).ManaPool().TotalMana() || len(b.Graveyard()) != 0 {
					t.Fatal("failed payment changed live state")
				}
			}
			if before.FindPermanentIncludingPhased(victim.ID()) == nil || before.FindPermanent(source.ID()).Tapped {
				t.Fatal("payment changed clone")
			}
		})
	}
}

func TestRONSacrificeCostGateTimingAndLimits(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "ATQ", "Gate to Phyrexia")
	p := g.PutOnBattlefield(c, a.PlayerID())
	pay := repositoryCard(t, "LEB", "Llanowar Elves")
	victim := g.PutOnBattlefield(pay, a.PlayerID())
	card := repositoryCard(t, "ATQ", "Ornithopter")
	target := g.PutOnBattlefield(card, b.PlayerID())
	for _, scenario := range []string{"step", "turn", "target"} {
		g.SetStep(core.Upkeep)
		g.SetActivePlayerIndex(0)
		id := target.ID()
		switch scenario {
		case "step":
			g.SetStep(core.Draw)
		case "turn":
			g.SetActivePlayerIndex(1)
		case "target":
			id = uuid.New()
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{id}); err == nil {
			t.Fatal("illegal activation accepted")
		}
		if g.FindPermanent(victim.ID()) == nil {
			t.Fatal("rejection sacrificed creature")
		}
	}
	g.SetStep(core.Upkeep)
	g.SetActivePlayerIndex(0)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	second := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{target.ID()}); err == nil || g.FindPermanent(second.ID()) == nil {
		t.Fatal("limit consumed another creature")
	}
	g.SetStep(core.Draw)
	g.SetActivePlayerIndex(1)
	g.ResolveStack()
	if g.FindPermanent(target.ID()) != nil {
		t.Fatal("resolution repeated timing check")
	}
}

func TestRONSacrificeCostCompositionAndInventory(t *testing.T) {
	cost := `SacrificeAPermanent(And([IsCardtype(Creature), Or([IsColor(Green), IsCreatureType(Goblin)])]))`
	for range 2 {
		g, a, _ := ronGame()
		c := sacrificeDeclaration(t, cost, `GainLife(Integer(1))`)
		p := g.PutOnBattlefield(c, a.PlayerID())
		g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if a.Life() != 21 {
			t.Fatal("composed filter failed")
		}
	}
	n, err := Parse(strings.NewReader(cost), "cost.ron")
	if err != nil {
		t.Fatal(err)
	}
	if got := inventoryArgumentRole(n[0], "cost", 0); got != "permanent_filter" {
		t.Fatal(got)
	}
}

func sacrificeFixture(t *testing.T, name string) mage.Card {
	t.Helper()
	set := "LEB"
	if name == "Ornithopter" {
		set = "ATQ"
	}
	return repositoryCard(t, set, name)
}

func TestRONSacrificeCostManualManaAndQueries(t *testing.T) {
	a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("Alice")}
	b := mage.NewBasePlayer("Bob")
	g := mage.NewGame(a, b)
	c := repositoryCard(t, "ATQ", "Ashnod's Altar")
	p := g.PutOnBattlefield(c, a.PlayerID())
	victim := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
	a.choice = victim.ID()
	for range 2 {
		if g.CanAfford(a.PlayerID(), core.ParseManaCost("{2}"), nil) {
			t.Fatal("tap-source planner advertised sacrifice mana")
		}
		if a.choices != 0 || g.FindPermanent(victim.ID()) == nil || a.ManaPool().TotalMana() != 0 || !g.GetStack().IsEmpty() {
			t.Fatal("query paid or chose a sacrifice")
		}
	}
	clone := g.Clone()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if clone.FindPermanent(victim.ID()) == nil || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
		t.Fatal("mana payment changed clone")
	}
	if !g.CanAfford(a.PlayerID(), core.ParseManaCost("{2}"), nil) {
		t.Fatal("produced mana cannot pay a cost")
	}
	if g.CanAfford(a.PlayerID(), core.ParseManaCost("{3}"), nil) {
		t.Fatal("planner used already sacrificed creature")
	}
}

func TestRONSacrificeCostDamageReplacement(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "ATQ", "Orcish Mechanics")
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	artifact := repositoryCard(t, "ATQ", "Ornithopter")
	g.PutOnBattlefield(artifact, a.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.DestroyPermanent(g.FindPermanent(p.ID()))
	g.ResolveStack()
	if b.Life() != 19 {
		t.Fatal("damage bypassed prevention or lost source LKI")
	}
}

func TestRONSacrificeCostInstancesAndCopy(t *testing.T) {
	g, a, _ := ronGame()
	first := repositoryCard(t, "LEG", "Fallen Angel")
	first.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(first, a.PlayerID())
	second := repositoryCard(t, "LEG", "Fallen Angel")
	second.SetOwner(a.PlayerID())
	q := g.PutOnBattlefield(second, a.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(p.ID()) != nil {
		t.Fatal("default choice did not sacrifice first Angel")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("ability copy failed")
	}
	clone := g.Clone()
	g.ResolveStack()
	if g.FindPermanent(q.ID()).CurrentPower(g) != 7 || g.FindPermanent(q.ID()).CurrentToughness(g) != 5 || len(a.Graveyard()) != 1 {
		t.Fatal("copy paid again or used wrong instance")
	}
	if clone.FindPermanent(q.ID()).CurrentPower(clone) != 3 {
		t.Fatal("original resolution changed pending clone")
	}
	clone.ResolveStack()
	if clone.FindPermanent(q.ID()).CurrentPower(clone) != 7 {
		t.Fatal("clone lost source binding")
	}
	g.RunStepWithPriority(core.Cleanup)
	if g.FindPermanent(q.ID()).CurrentPower(g) != 3 || clone.FindPermanent(q.ID()).CurrentPower(clone) != 7 {
		t.Fatal("cleanup crossed clone boundary")
	}
}
