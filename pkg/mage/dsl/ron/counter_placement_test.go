package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCounterPlacementRealCards(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		counter   core.CounterType
		dp, dt    int
	}{
		{"FEM", "Armor Thrull", core.P1P2, 1, 2},
		{"FEM", "Fungal Bloom", core.Spore, 0, 0},
		{"ATQ", "Dwarven Weaponsmith", core.P1P1, 1, 1},
		{"3ED", "Dwarven Weaponsmith", core.P1P1, 1, 1},
	} {
		for _, change := range []string{"none", "source-return", "target-return", "phase", "control", "noncreature"} {
			t.Run(tc.set+"/"+tc.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				c := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
					t.Fatal(err)
				}
				c.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(c, a.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				targetCard := repositoryCard(t, "LEB", "Llanowar Elves")
				if tc.name == "Fungal Bloom" {
					nodes, err := Parse(strings.NewReader(`Card(Name: "Fungus fixture", Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: [Fungus]), CardPT: (Power: 1, Toughness: 1))`), "fixture.ron")
					if err != nil {
						t.Fatal(err)
					}
					factory, err := Compile(nodes[0])
					if err != nil {
						t.Fatal(err)
					}
					targetCard = factory()
				}
				targetCard.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(targetCard, b.PlayerID())
				var payment *mage.Permanent
				if tc.name == "Dwarven Weaponsmith" {
					paymentCard := repositoryCard(t, "ATQ", "Ornithopter")
					paymentCard.SetOwner(b.PlayerID())
					payment = g.PutOnBattlefield(paymentCard, a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
						t.Fatal("wrong-step activation accepted")
					}
					if g.FindPermanent(payment.ID()) == nil || g.FindPermanent(source.ID()).Tapped {
						t.Fatal("timing rejection paid cost")
					}
					g.SetStep(core.Upkeep)
				}
				if tc.name == "Fungal Bloom" {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
						t.Fatal("unpaid activation accepted")
					}
					a.ManaPool().Add(core.Green, 2)
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
					t.Fatal(err)
				}
				if target.Counters[tc.counter] != 0 || g.StackSize() != 1 {
					t.Fatal("placement ran during payment")
				}
				if tc.name == "Armor Thrull" && (g.FindPermanent(source.ID()) != nil || len(a.Graveyard()) != 1) {
					t.Fatal("source was not sacrificed")
				}
				if payment != nil && (g.FindPermanent(payment.ID()) != nil || len(b.Graveyard()) != 1 || !g.FindPermanent(source.ID()).Tapped) {
					t.Fatal("tap and sacrifice cost missing")
				}
				if tc.name == "Fungal Bloom" && a.ManaPool().TotalMana() != 0 {
					t.Fatal("mana cost missing")
				}
				g.SetStep(core.Draw)
				switch change {
				case "source-return":
					if p := g.FindPermanent(source.ID()); p != nil {
						g.RemoveFromBattlefield(p)
					}
					a.RemoveFromGraveyard(c.ID())
					g.PutOnBattlefield(c, b.PlayerID())
				case "target-return":
					g.RemoveFromBattlefield(target)
					g.PutOnBattlefield(targetCard, b.PlayerID())
				case "phase":
					g.PhaseOut(target.ID())
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "noncreature":
					g.AddContinuousEffect(mage.FuncContinuousEffect(core.LayerType, core.EndOfTurn, func(game *mage.Game, _ uuid.UUID) error {
						game.GetEffects().RevokeAttr(target.ID(), core.AttrIsCreature)
						return nil
					}))
					g.ApplyContinuousEffects()
				}
				clone := g.Clone()
				g.ResolveStack()
				if clone.FindPermanentIncludingPhased(target.ID()).Counters[tc.counter] != 0 {
					t.Fatal("resolution changed pending clone")
				}
				clone.ResolveStack()
				want := 1
				if change == "target-return" || change == "phase" || change == "noncreature" {
					want = 0
				}
				for _, game := range []*mage.Game{g, clone} {
					p := game.FindPermanentIncludingPhased(target.ID())
					if p.Counters[tc.counter] != want {
						t.Fatalf("counter count = %d, want %d", p.Counters[tc.counter], want)
					}
				}
				g.RunStepWithPriority(core.Cleanup)
				if p := g.FindPermanent(target.ID()); p != nil && (p.CurrentPower(g) != 1+want*tc.dp || p.CurrentToughness(g) != 1+want*tc.dt) {
					t.Fatal("counter P/T or cleanup incorrect")
				}
			})
		}
	}
}

func TestRONCounterPlacementSourceAndHost(t *testing.T) {
	for _, context := range []string{"source", "host", "trigger"} {
		for _, change := range []string{"none", "return", "phase", "phase-cycle", "control", "reattach", "source-leave", "source-return"} {
			t.Run(context+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				subject := "ThisPermanent"
				types, subtypes := "Artifact", ""
				if context == "host" {
					subject, types, subtypes = "HostPermanent", "Enchantment", "Aura"
				}
				actions := fmt.Sprintf(`ActionList([PutCounters([NumberCountersOfTypeOnPermanent(Integer(3), ChargeCounter, %s)])])`, subject)
				rules := `Activated(PayMana([]), ` + actions + `)`
				if context == "host" {
					rules = `EnchantPermanent(IsCardtype(Creature)), ` + rules
				}
				if context == "trigger" {
					rules = `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)), ` + actions + `)`
				}
				text := fmt.Sprintf(`Card(Name: "Counter context", Typeline: (Supertypes: [], Cardtypes: [%s], Subtypes: [%s]), Rules: [%s])`, types, subtypes, rules)
				nodes, err := Parse(strings.NewReader(text), "context.ron")
				if err != nil {
					t.Fatal(err)
				}
				factory, err := Compile(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
				card := factory()
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				target := source
				var hostCard mage.Card
				if context == "host" {
					hostCard = repositoryCard(t, "LEB", "Llanowar Elves")
					hostCard.SetOwner(b.PlayerID())
					target = g.PutOnBattlefield(hostCard, b.PlayerID())
					g.Attach(source.ID(), target.ID())
				}
				other := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
				if context == "trigger" {
					g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
					g.PutTriggersOnStack()
				} else if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if g.StackSize() != 1 {
					t.Fatal("ability did not use stack")
				}
				want, otherWant := 3, 0
				switch change {
				case "return":
					g.RemoveFromBattlefield(target)
					returned := card
					if context == "host" {
						returned = hostCard
					}
					g.PutOnBattlefield(returned, b.PlayerID())
					want = 0
				case "phase":
					g.PhaseOut(target.ID())
					want = 0
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(target.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: target.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "reattach":
					if context == "host" {
						g.Attach(source.ID(), other.ID())
						want, otherWant = 0, 3
					}
				case "source-leave", "source-return":
					g.RemoveFromBattlefield(source)
					if context != "host" {
						want = 0
					}
					if change == "source-return" {
						g.PutOnBattlefield(card, a.PlayerID())
						if context == "host" {
							g.Attach(card.ID(), other.ID())
						}
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				clone.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					p := game.FindPermanentIncludingPhased(target.ID())
					if p != nil && p.Counters[core.Charge] != want {
						t.Fatalf("source/host counters = %d, want %d", p.Counters[core.Charge], want)
					}
					if game.FindPermanent(other.ID()).Counters[core.Charge] != otherWant {
						t.Fatal("counter followed wrong attachment")
					}
				}
			})
		}
	}
}

func TestRONCounterPlacementAmountsAndRollback(t *testing.T) {
	for _, amount := range []int{-2, 0, 3} {
		g, a, _ := ronGame()
		card := sacrificeDeclaration(t, `And([PayMana([ManaCostGeneric(1)]), TapPermanent(ThisPermanent)])`, fmt.Sprintf(`PutCounters([NumberCountersOfTypeOnPermanent(Integer(%d), ChargeCounter, ThisPermanent)])`, amount))
		p := g.PutOnBattlefield(card, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
			t.Fatal("unpaid ability accepted")
		}
		if g.FindPermanent(p.ID()).Tapped || !g.GetStack().IsEmpty() || g.FindPermanent(p.ID()).Counters[core.Charge] != 0 {
			t.Fatal("failed cost changed game")
		}
		a.ManaPool().Add(core.Colorless, 1)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if g.FindPermanent(p.ID()).Counters[core.Charge] != max(amount, 0) {
			t.Fatal("wrong literal amount")
		}
	}
	for _, x := range []int{0, 4} {
		g, a, b := ronGame()
		src := `Card(Name: "Counter X", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), ManaCost: [ManaCostX], Rules: [SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([PutCounters([NumberCountersOfTypeOnPermanent(ValueX, PTCounter(1, 1), Ref_TargetPermanent)])])))])`
		nodes, err := Parse(strings.NewReader(src), "x.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		c := factory()
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		a.ManaPool().Add(core.Colorless, x)
		target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{uuid.New()}, x); err == nil {
			t.Fatal("invalid target accepted")
		}
		if a.ManaPool().TotalMana() != x || len(a.Hand()) != 1 {
			t.Fatal("target rejection paid mana")
		}
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{target.ID()}, x); err != nil {
			t.Fatal(err)
		}
		if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
			t.Fatal("spell copy failed")
		}
		g.ResolveStack()
		if g.FindPermanent(target.ID()).Counters[core.P1P1] != 2*x {
			t.Fatal("announced X not retained by spell copy")
		}
	}
}

func TestRONCounterPlacementInventory(t *testing.T) {
	for _, tc := range []struct {
		expression, role string
		index            int
		want             string
	}{
		{`PutCounters([ACounterOfTypeOnPermanent(SporeCounter, ThisPermanent)])`, "action", 0, "counter_instruction"},
		{`ACounterOfTypeOnPermanent(PTCounter(1, 2), Ref_TargetPermanent)`, "counter_instruction", 0, "permanent_counter_type"},
		{`ACounterOfTypeOnPermanent(SporeCounter, ThisPermanent)`, "counter_instruction", 1, "permanent_reference"},
		{`NumberCountersOfTypeOnPermanent(Integer(3), TimeCounter, ThisPermanent)`, "counter_instruction", 0, "value"},
		{`NumberCountersOfTypeOnPermanent(Integer(3), TimeCounter, ThisPermanent)`, "counter_instruction", 1, "permanent_counter_type"},
		{`PTCounter(-2, -2)`, "permanent_counter_type", 0, "integer_literal"},
	} {
		nodes, err := Parse(strings.NewReader(tc.expression), "inventory.ron")
		if err != nil {
			t.Fatal(err)
		}
		if got := inventoryArgumentRole(nodes[0], tc.role, tc.index); got != tc.want {
			t.Fatalf("role = %s, want %s", got, tc.want)
		}
	}
}

func TestRONCounterPlacementPaymentAndContexts(t *testing.T) {
	for _, name := range []string{"Armor Thrull", "Dwarven Weaponsmith"} {
		g, a, b := ronGame()
		set := "FEM"
		if name == "Dwarven Weaponsmith" {
			set = "ATQ"
		}
		c := repositoryCard(t, set, name)
		c.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(c, a.PlayerID())
		g.SetStep(core.Upkeep)
		target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
			t.Fatal("unpayable tap/sacrifice accepted")
		}
		if g.FindPermanent(p.ID()) == nil || g.FindPermanent(p.ID()).Tapped || len(a.Graveyard()) != 0 || !g.GetStack().IsEmpty() {
			t.Fatal("failed cost changed source")
		}
		p.RevokeBaseAttr(core.AttrSummonSick)
		if name == "Armor Thrull" {
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
				t.Fatal(err)
			}
			a.RemoveFromGraveyard(c.ID())
			g.PutOnBattlefield(c, a.PlayerID())
			g.ResolveStack()
			if g.FindPermanent(p.ID()).Counters[core.P1P2] != 0 {
				t.Fatal("self target followed returned sacrifice")
			}
		} else {
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
				t.Fatal("missing artifact accepted")
			}
			if g.FindPermanent(p.ID()).Tapped {
				t.Fatal("missing sacrifice did not roll back tap")
			}
		}
	}
	for _, rule := range []string{
		`Activated(PutCounters([ACounterOfTypeOnPermanent(ChargeCounter, ThisPermanent)]), ActionList([GainLife(Integer(1))]))`,
		`PermanentLayerEffect(ThisPermanent, [PutCounters([ACounterOfTypeOnPermanent(ChargeCounter, ThisPermanent)])])`,
		`Activated(PayMana([]), Targeted([TargetPermanent(AnyPermanent)], ActionList([TapPermanent(Ref_TargetPermanent)]))), Activated(PayMana([]), ActionList([PutCounters([ACounterOfTypeOnPermanent(ChargeCounter, Ref_TargetPermanent)])]))`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name: "Invalid context", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid context or cross-rule binding accepted")
		}
	}
	nodes, err := Parse(strings.NewReader(`Card(Name: "Invalid source", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [SpellActions(ActionList([PutCounters([ACounterOfTypeOnPermanent(ChargeCounter, ThisPermanent)])]))])`), "invalid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("spell source used as permanent")
	}
}

type counterReplacementChooser struct {
	*mage.BasePlayer
	index   int
	choices int
}

func (p *counterReplacementChooser) ChooseMode(_ []string, _ string) int {
	p.choices++
	return p.index
}

type counterPlacementReplacement struct {
	source  uuid.UUID
	prevent bool
	counter core.CounterType
}

func (r *counterPlacementReplacement) SourceID() uuid.UUID           { return r.source }
func (*counterPlacementReplacement) GetDuration() core.Duration      { return core.Indefinite }
func (*counterPlacementReplacement) IsActive(mage.GameReader) bool   { return true }
func (r *counterPlacementReplacement) Clone() mage.ReplacementEffect { c := *r; return &c }
func (r *counterPlacementReplacement) Matches(a mage.Action, _ mage.GameReader) bool {
	placement, ok := a.(*mage.AddCountersAction)
	return ok && placement.CounterType() == r.counter && placement.Amount() > 0 && !placement.OnEntry()
}
func (r *counterPlacementReplacement) Replace(a mage.Action, _ *mage.Game) mage.Action {
	if r.prevent {
		return nil
	}
	placement := a.(*mage.AddCountersAction)
	return placement.WithAmount(placement.Amount() + 1)
}

func TestRONCounterPlacementReplacementOrder(t *testing.T) {
	for _, tc := range []struct {
		name         string
		choice, want int
		prevent      bool
	}{
		{"add-then-double", 0, 4, false}, {"double-then-add", 1, 3, false}, {"prevent", 0, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("Alice"), index: 1 - tc.choice}
			b := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("Bob"), index: tc.choice}
			g := mage.NewGame(a, b)
			card := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]), Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([PutCounters([ACounterOfTypeOnPermanent(PTCounter(1, 1), Ref_TargetPermanent)]), GainLife(Integer(1))])))`)()
			source := g.PutOnBattlefield(card, a.PlayerID())
			target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
			g.AddReplacementEffect(&counterPlacementReplacement{source: source.ID(), prevent: tc.prevent})
			g.AddCounterDoubler(source.ID(), core.P1P1, mage.AnyPermanent)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(target.ID()).Counters[core.P1P1] != tc.want {
				t.Fatal("placement ignored affected player's replacement order")
			}
			if a.choices != 0 || b.choices != 1 || a.Life() != 21 {
				t.Fatal("wrong replacement chooser or prevented placement stopped next action")
			}
		})
	}
}

func TestRONCounterPlacementSignedPT(t *testing.T) {
	for _, tc := range []struct {
		spelling         string
		counter          core.CounterType
		power, toughness int
	}{
		{`PTCounter(-2, -2)`, core.M2M2, -2, -2},
		{`PTCounter(-1, -1)`, core.M1M1, -1, -1},
		{`PTCounter(-0, -1)`, core.M0M1, 0, -1},
		{`PTCounter(-0, -2)`, core.M0M2, 0, -2},
		{`PTCounter(1, 0)`, core.P1P0, 1, 0},
		{`PTCounter(0, 1)`, core.P0P1, 0, 1},
		{`PTCounter(2, 0)`, core.P2P0, 2, 0},
		{`PTCounter(0, 2)`, core.P0P2, 0, 2},
		{`PTCounter(+2, +2)`, core.P2P2, 2, 2},
	} {
		t.Run(tc.spelling, func(t *testing.T) {
			g, a, _ := ronGame()
			factory := permanentScopeFactory(t, "Artifact", fmt.Sprintf(`Activated(PayMana([]), Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([PutCounters([ACounterOfTypeOnPermanent(%s, Ref_TargetPermanent)])])))`, tc.spelling))
			c := factory()
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(c, a.PlayerID())
			target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(target.ID())
			if p == nil || p.Counters[tc.counter] != 1 || p.CurrentPower(g) != 4+tc.power || p.CurrentToughness(g) != 4+tc.toughness {
				t.Fatal("counter spelling changed P/T semantics")
			}
		})
	}
}

func TestRONCounterPlacementInvalid(t *testing.T) {
	for _, action := range []string{
		`PutCounters`, `PutCounters()`, `PutCounters(ACounterOfTypeOnPermanent(ChargeCounter, ThisPermanent))`, `PutCounters([])`,
		`PutCounters([ACounterOfTypeOnPermanent(ChargeCounter, ThisPermanent), ACounterOfTypeOnPermanent(ChargeCounter, ThisPermanent)])`,
		`PutCounters([ACounterOfTypeOnPermanent])`, `PutCounters([ACounterOfTypeOnPermanent(ChargeCounter)])`,
		`PutCounters([ACounterOfTypeOnPermanent(UnknownCounter, ThisPermanent)])`, `PutCounters([ACounterOfTypeOnPermanent(ChargeCounter(), ThisPermanent)])`,
		`PutCounters([ACounterOfTypeOnPermanent(PTCounter(1), ThisPermanent)])`, `PutCounters([ACounterOfTypeOnPermanent(PTCounter(Integer(1), 1), ThisPermanent)])`,
		`PutCounters([ACounterOfTypeOnPermanent(PTCounter(3, 3), ThisPermanent)])`,
		`PutCounters([ACounterOfTypeOnPermanent(ChargeCounter, Ref_TargetPermanent)])`, `PutCounters([ACounterOfTypeOnPermanent(ChargeCounter, HostPermanent)])`,
		`PutCounters([ACounterOfTypeOnPermanent(ChargeCounter, Trigger_ThatPermanent)])`, `PutCounters([ACounterOfTypeOnPlayer(ChargeCounter, You)])`,
		`PutCounters([NumberCountersOfTypeOnPermanent(ValueX, ChargeCounter, ThisPermanent)])`,
		`PutCounters([NumberCountersOfTypeOnPermanent(Integer(1), ChargeCounter)])`, `PutCounters([NumberCountersOfTypeOnPermanent(ManaValueOfPermanent(Ref_TargetPlayer), ChargeCounter, ThisPermanent)])`,
	} {
		t.Run(action, func(t *testing.T) {
			src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), ActionList([%s]))])`, action)
			nodes, err := Parse(strings.NewReader(src), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid declaration accepted")
			}
		})
	}
}

func TestRONCounterPlacementCopiesAndReplacement(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]), Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([PutCounters([ACounterOfTypeOnPermanent(PTCounter(1, 1), Ref_TargetPermanent)])])))`)
	first, second := factory(), factory()
	if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("shared target declaration")
	}
	for _, c := range []mage.Card{first, second} {
		if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
			t.Fatal(err)
		}
	}
	g, a, b := ronGame()
	p := g.PutOnBattlefield(first, a.PlayerID())
	q := g.PutOnBattlefield(second, a.PlayerID())
	one := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
	two := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
	g.AddCounterDoubler(p.ID(), core.P1P1, mage.ControlledBy(a.PlayerID()))
	g.AddCounterDoubler(q.ID(), core.P1P1, mage.AnyPermanent)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{one.ID()}); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, []uuid.UUID{two.ID()}); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.ResolveStack()
	if g.FindPermanent(one.ID()).Counters[core.P1P1] != 8 || g.FindPermanent(two.ID()).Counters[core.P1P1] != 2 {
		t.Fatal("replacement recheck, copy or instance binding failed")
	}
	if clone.FindPermanent(one.ID()).Counters[core.P1P1] != 0 {
		t.Fatal("clone counters changed")
	}
	clone.ResolveStack()
	if clone.FindPermanent(one.ID()).Counters[core.P1P1] != 8 {
		t.Fatal("clone lost target or replacements")
	}
	g.MutablePermanent(one.ID()).RemoveCounter(core.P1P1, 1)
	if clone.FindPermanent(one.ID()).Counters[core.P1P1] != 8 {
		t.Fatal("counter mutation crossed clone boundary")
	}
}
