package ron

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRONPoisonCards(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		amount    int
	}{{"DRK", "Marsh Viper", 2}, {"LEG", "Pit Scorpion", 1}, {"LEG", "Serpent Generator", 1}} {
		for _, state := range []string{"normal", "self", "other", "zero", "prevented", "partial", "twice", "leave", "return", "phase-after", "phase-before", "control", "removed", "copy"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				c := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
					t.Fatal(err)
				}
				c.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
				if tc.name == "Serpent Generator" {
					a.ManaPool().Add(core.Colorless, 4)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
					tokens := g.FilterBattlefield(mage.IsToken)
					if len(tokens) != 1 {
						t.Fatal("missing token")
					}
					p = tokens[0]
					if p.Name() != "Snake Token" || !p.HasType(core.TypeArtifact) || !p.HasType(core.TypeCreature) || !p.HasSubType("Snake") || p.CurrentPower(g) != 1 || p.CurrentToughness(g) != 1 || len(p.Colors()) != 0 || p.Card.Owner() != a.PlayerID() {
						t.Fatal("wrong token characteristics")
					}
				}
				if state == "phase-before" {
					g.PhaseOut(p.ID())
				}
				if state == "removed" {
					g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				}
				if state == "control" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				if state == "prevented" {
					g.AddPreventionShield(b.PlayerID(), 3)
				}
				if state == "partial" {
					g.AddPreventionShield(b.PlayerID(), 2)
				}
				recipient := b
				if state == "self" {
					recipient = a
				}
				amount := 3
				if state == "zero" {
					amount = 0
				}
				damageSource := p.ID()
				if state == "other" {
					damageSource = g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 9), a.PlayerID()).ID()
				}
				g.DealDamageToPlayer(recipient, amount, damageSource)
				want := tc.amount
				if state == "other" || state == "phase-before" || state == "removed" || state == "prevented" || state == "zero" {
					want = 0
				}
				if state == "twice" {
					g.DealDamageToPlayer(recipient, 1, p.ID())
					want *= 2
				}
				if recipient.PoisonCounters() != 0 {
					t.Fatal("poison before resolution")
				}
				if state == "leave" || state == "return" {
					g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
					if state == "return" && tc.name != "Serpent Generator" {
						g.PutOnBattlefield(c, b.PlayerID())
					}
				}
				if state == "phase-after" {
					g.PhaseOut(p.ID())
				}
				clone := g.Clone()
				g.PutTriggersOnStack()
				if state == "copy" {
					g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
					want *= 2
				}
				g.ResolveStack()
				if recipient.PoisonCounters() != want {
					t.Fatalf("poison %d, want %d", recipient.PoisonCounters(), want)
				}
				other := a
				if recipient == a {
					other = b
				}
				if other.PoisonCounters() != 0 {
					t.Fatal("wrong player")
				}
				if clone.GetPlayer(recipient.PlayerID()).PoisonCounters() != 0 {
					t.Fatal("clone changed")
				}
				clone.PutTriggersOnStack()
				clone.ResolveStack()
				if state == "copy" {
					want /= 2
				}
				if clone.GetPlayer(recipient.PlayerID()).PoisonCounters() != want {
					t.Fatal("clone lost pending player")
				}
			})
		}
	}
}

func TestRONPoisonInvalid(t *testing.T) {
	for _, action := range []string{`PutCounters([ACounterOfTypeOnPlayer(PoisonCounter(),Trigger_ThatPlayer)])`, `PutCounters([ACounterOfTypeOnPlayer(ChargeCounter,Trigger_ThatPlayer)])`, `PutCounters([ACounterOfTypeOnPlayer(PoisonCounter,ThisPermanent)])`, `PutCounters([ACounterOfTypeOnPlayer(PoisonCounter,Trigger_ThatPlayer())])`, `PutCounters([NumberCountersOfTypeOnPlayer(Integer(-1),PoisonCounter,Trigger_ThatPlayer)])`, `PutCounters([NumberCountersOfTypeOnPlayer(ValueX,PoisonCounter,Trigger_ThatPlayer)])`, `PutCounters([ACounterOfTypeOnPlayer(PoisonCounter)])`, `PutCounters([ACounterOfTypeOnPlayer(PoisonCounter,Trigger_ThatPlayer),ACounterOfTypeOnPlayer(PoisonCounter,You)])`} {
		text := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[TriggerA(WhenAPermanentDealsDamageToAPlayer(SinglePermanent(ThisPermanent),AnyPlayer),ActionList([` + action + `]))])`
		nodes, err := Parse(strings.NewReader(text), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", action)
		}
	}
	for _, filter := range []string{`AnyPlayer()`, `SinglePlayer(You)`, `You`} {
		text := fmt.Sprintf(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[TriggerA(WhenAPermanentDealsDamageToAPlayer(SinglePermanent(ThisPermanent),%s),ActionList([PutCounters([ACounterOfTypeOnPlayer(PoisonCounter,Trigger_ThatPlayer)])]))])`, filter)
		nodes, err := Parse(strings.NewReader(text), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", filter)
		}
	}
}

func TestRONPoisonCastingAndCombat(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		colors    []core.Color
		poison    int
	}{
		{"DRK", "Marsh Viper", []core.Color{core.Colorless, core.Colorless, core.Colorless, core.Green}, 2},
		{"LEG", "Pit Scorpion", []core.Color{core.Colorless, core.Colorless, core.Black}, 1},
		{"LEG", "Serpent Generator", []core.Color{core.Colorless, core.Colorless, core.Colorless, core.Colorless, core.Colorless, core.Colorless}, 1},
	} {
		for _, blocked := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%v", tc.name, blocked), func(t *testing.T) {
				a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				g.SetStep(core.PrecombatMain)
				c := repositoryCard(t, tc.set, tc.name)
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
				for _, color := range tc.colors[:len(tc.colors)-1] {
					a.ManaPool().Add(color, 1)
				}
				before := a.ManaPool().TotalMana()
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 || a.ManaPool().TotalMana() != before {
					t.Fatal("cast rollback")
				}
				a.ManaPool().Add(tc.colors[len(tc.colors)-1], 1)
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				p := g.FindPermanent(c.ID())
				if p == nil || a.ManaPool().TotalMana() != 0 {
					t.Fatal("cast or mana payment")
				}
				if tc.name == "Serpent Generator" {
					a.ManaPool().Add(core.Colorless, 4)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
					p = g.FilterBattlefield(mage.IsToken)[0]
				}
				p.RevokeBaseAttr(core.AttrSummonSick)
				g.PutOnBattlefield(mage.NewCreature("Blocker", "", 0, 9), b.PlayerID())
				a.SetAttackers(1, []string{p.Name()})
				if blocked {
					b.SetBlockers(1, map[string]string{"Blocker": p.Name()})
				}
				g.RunStepWithPriority(core.DeclareAttackers)
				g.RunStepWithPriority(core.DeclareBlockers)
				g.RunStepWithPriority(core.CombatDamage)
				want := tc.poison
				life := 19
				if blocked {
					want = 0
					life = 20
				}
				if b.PoisonCounters() != want || b.Life() != life {
					t.Fatalf("combat poison/life %d/%d, want %d/%d", b.PoisonCounters(), b.Life(), want, life)
				}
			})
		}
	}
}

func TestRONPoisonGeneratorPaymentAndCopies(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "LEG", "Serpent Generator")
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 3 || g.StackSize() != 0 {
		t.Fatal("activation rollback")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 0 || len(g.FilterBattlefield(mage.IsToken)) != 0 {
		t.Fatal("payment or early token")
	}
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || a.ManaPool().TotalMana() != 4 || g.StackSize() != 1 {
		t.Fatal("tapped source accepted")
	}
	g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.ResolveStack()
		tokens := game.FilterBattlefield(mage.IsToken)
		if len(tokens) != 2 {
			t.Fatal("copied token creation")
		}
		var first, second *mage.Permanent
		for _, token := range tokens {
			if token.ControllerID() == a.PlayerID() {
				first = token
			} else {
				second = token
			}
			if token.Card.Owner() != token.ControllerID() {
				t.Fatal("token owner")
			}
		}
		if first == nil || second == nil {
			t.Fatal("creation controller")
		}
		game.DealSimultaneousDamage(mage.PlayerDamageAssignment(game.GetPlayer(b.PlayerID()), 1, game.CaptureDamageSource(first.ID())), mage.PlayerDamageAssignment(game.GetPlayer(a.PlayerID()), 1, game.CaptureDamageSource(second.ID())))
		game.DealDamageToPlayer(game.GetPlayer(a.PlayerID()), 1, first.ID())
		game.PutTriggersOnStack()
		if game.StackSize() != 3 {
			t.Fatal("shared token source or event binding")
		}
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).PoisonCounters() != 2 || game.GetPlayer(b.PlayerID()).PoisonCounters() != 1 {
			t.Fatal("shared token recipients")
		}
	}
}

func TestRONPoisonTokenScope(t *testing.T) {
	for _, action := range []string{`PutCounters([ACounterOfTypeOnPlayer(PoisonCounter,Ref_TargetPlayer)])`, `GainLife(ValueX)`, `GainLife(Trigger_AmountOfDamageDealt)`} {
		text := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),ManaCost:[ManaCostX],Rules:[SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([CreateTokens([TokenWithPT(PT(1,1),Colorless,[],[Creature],SubtypesList([Snake]),[TriggerA(WhenAPermanentDealsDamageToAPlayer(SinglePermanent(ThisPermanent),AnyPlayer),ActionList([` + action + `]))])],[])])))])`
		nodes, err := Parse(strings.NewReader(text), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("token borrowed outer binding", action)
		}
	}
	for _, rule := range []string{`TriggerA`, `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([GainLife(Integer(1))]))`, `TriggerA(WhenAPermanentDealsDamageToAPlayer(SinglePermanent(HostPermanent),AnyPlayer),ActionList([GainLife(Integer(1))]))`} {
		text := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([CreateTokens([TokenWithPT(PT(1,1),Colorless,[],[Creature],SubtypesList([Snake]),[` + rule + `])],[])]))])`
		nodes, err := Parse(strings.NewReader(text), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("unsupported token trigger accepted", rule)
		}
	}
}

func TestRONPoisonPlacementAndLoss(t *testing.T) {
	for _, amount := range []int{0, 1, 2} {
		text := fmt.Sprintf(`Card(Name:"Poison",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([PutCounters([NumberCountersOfTypeOnPlayer(Integer(%d),PoisonCounter,You)])]))])`, amount)
		if amount == 1 {
			text = strings.Replace(text, "NumberCountersOfTypeOnPlayer(Integer(1),PoisonCounter,You)", "ACounterOfTypeOnPlayer(PoisonCounter,You)", 1)
		}
		reg, err := LoadFS(fstest.MapFS{"a.ron": {Data: []byte(text)}}, "*.ron")
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		c, err := reg.CreateCard("Poison")
		if err != nil {
			t.Fatal(err)
		}
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if a.PoisonCounters() != amount || b.PoisonCounters() != 0 || a.Life() != 20 {
			t.Fatal("explicit player or zero amount")
		}
	}
	g, a, b := ronGame()
	c := repositoryCard(t, "DRK", "Marsh Viper")
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	b.AddPoisonCounters(8)
	g.DealDamageToPlayer(b, 1, p.ID())
	g.CheckStateBasedActions()
	if !b.IsAlive() {
		t.Fatal("loss before poison resolution")
	}
	g.PutTriggersOnStack()
	g.ResolveStack()
	g.CheckStateBasedActions()
	if b.PoisonCounters() != 10 || b.IsAlive() || !g.IsGameOver() {
		t.Fatal("ten poison did not lose")
	}
}

func TestRONPoisonFactoryInstances(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		amount    int
	}{{"DRK", "Marsh Viper", 2}, {"LEG", "Pit Scorpion", 1}, {"LEG", "Serpent Generator", 1}} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
			if err != nil {
				t.Fatal(err)
			}
			nodes, err := Parse(strings.NewReader(string(data)), tc.set+".ron")
			if err != nil {
				t.Fatal(err)
			}
			var factory mage.CardFactory
			for _, n := range nodes {
				if n.Fields["Name"].String == tc.name {
					factory, err = Compile(n)
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			if factory == nil {
				t.Fatal("missing factory")
			}
			g, a, b := ronGame()
			first, second := factory(), factory()
			first.SetOwner(a.PlayerID())
			second.SetOwner(b.PlayerID())
			p, q := g.PutOnBattlefield(first, a.PlayerID()), g.PutOnBattlefield(second, b.PlayerID())
			if tc.name == "Serpent Generator" {
				for _, pair := range []struct {
					player mage.Player
					source *mage.Permanent
				}{{a, p}, {b, q}} {
					pair.player.ManaPool().Add(core.Colorless, 4)
					if err := g.ActivateAbilityByIndex(pair.player.PlayerID(), pair.source.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
				}
				g.ResolveStack()
				tokens := g.FilterBattlefield(mage.IsToken)
				if len(tokens) != 2 {
					t.Fatal("independent factory token count")
				}
				for _, token := range tokens {
					if token.ControllerID() == a.PlayerID() {
						p = token
					} else {
						q = token
					}
				}
				if p.RuntimeAbilities[0] == q.RuntimeAbilities[0] || p.RuntimeAbilities[0].AbilityID() == q.RuntimeAbilities[0].AbilityID() {
					t.Fatal("shared runtime ability")
				}
			}
			g.PhaseOut(p.ID())
			g.PhaseIn([]uuid.UUID{p.ID()})
			g.DealSimultaneousDamage(mage.PlayerDamageAssignment(b, 1, g.CaptureDamageSource(p.ID())), mage.PlayerDamageAssignment(a, 1, g.CaptureDamageSource(q.ID())))
			g.DealDamageToPermanent(q, 1, p.ID())
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			g.PutTriggersOnStack()
			if g.StackSize() != 2 {
				t.Fatal("wrong source, permanent damage, or phase identity")
			}
			g.ResolveStack()
			if a.PoisonCounters() != tc.amount || b.PoisonCounters() != tc.amount {
				t.Fatal("factory instances shared recipient")
			}
		})
	}
}

func TestRONPoisonInventory(t *testing.T) {
	report, err := InspectFS(os.DirFS("../../../../cards/mtgish"), "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"TriggerA": "token_rule", "PoisonCounter": "player_counter_type", "Trigger_ThatPlayer": "player_reference", "ACounterOfTypeOnPlayer": "counter_instruction"} {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role && slices.Contains(sig.Cards, "Serpent Generator") {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role %s", name, role)
		}
	}
}
