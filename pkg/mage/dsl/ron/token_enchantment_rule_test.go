package ron

import (
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const tokenEnchantmentRule = `PermanentRuleEffect(ThisPermanent,[CantBeEnchanted])`

func tokenEnchantmentFactory(t *testing.T, rule string) func() mage.Card {
	t.Helper()
	return permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateTokens([NumberTokens(Integer(2),TokenWithPT(PT(1,1),Colorless,[],[Artifact,Creature],SubtypesList([Tetravite]),[Flying,`+rule+`]))],[])]))`)
}

func TestRONTokenEnchantmentRuleGameplay(t *testing.T) {
	for _, change := range []string{"none", "source-leaves", "remove", "phase-cycle", "control", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			factory := tokenEnchantmentFactory(t, tokenEnchantmentRule)
			maker := g.PutOnBattlefield(factory(), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), maker.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if len(g.FilterBattlefield(mage.IsToken)) != 0 {
				t.Fatal("tokens entered before resolution")
			}
			g.ResolveStack()
			tokens := g.FilterBattlefield(mage.IsToken)
			if len(tokens) != 2 {
				t.Fatal("wrong token count")
			}
			host := tokens[0]
			if !host.HasType(core.TypeArtifact) || !host.HasKeyword(core.Flying) || host.CurrentPower(g) != 1 {
				t.Fatal("wrong token characteristics")
			}
			aura := mage.NewAura("Candidate", "{W}", mage.WithCastTarget(mage.TargetPermanent(mage.AnyPermanent)))
			aura.SetOwner(a.PlayerID())
			a.AddToHand(aura)
			a.ManaPool().Add(core.White, 1)
			if err := g.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{host.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
				t.Fatal("illegal Aura cast or payment rollback")
			}
			clone := g.Clone()
			switch change {
			case "source-leaves":
				g.BouncePermanentToHand(g.FindPermanent(maker.ID()))
			case "remove":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), host.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: maker.ID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if err := mage.ApplyEffect(g, mage.CreateTokenCopiesOfPermanent(mage.You, mage.ValueAtResolution(mage.Fixed(1)), mage.This), host.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				all := g.FilterBattlefield(mage.IsToken)
				if len(all) != 3 {
					t.Fatal("copy missing")
				}
				host = all[2]
			}
			if err := g.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{tokens[1].ID()}, 0); err == nil {
				t.Fatal("sibling token lost its rule")
			}
			if err := g.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{host.ID()}, 0); (err == nil) != (change == "remove") {
				t.Fatalf("wrong live legality: %v", err)
			}
			if err := clone.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{tokens[0].ID()}, 0); err == nil {
				t.Fatal("clone restriction changed")
			}
		})
	}
}

func TestRONTokenEnchantmentRuleInvalid(t *testing.T) {
	for _, rule := range []string{`PermanentRuleEffect(ThisPermanent(),[CantBeEnchanted])`, `PermanentRuleEffect(HostPermanent,[CantBeEnchanted])`, `PermanentRuleEffect(Ref_TargetPermanent,[CantBeEnchanted])`, `PermanentRuleEffect(ThisPermanent,[])`, `PermanentRuleEffect(ThisPermanent,[CantBeEnchanted()])`, `PermanentRuleEffect(ThisPermanent,[CantBeEnchanted,CantBeEnchanted])`, `PermanentRuleEffect(ThisPermanent,[CantAttack])`, `PermanentRuleEffect(ThisPermanent,[CantBeEnchanted],UntilEndOfTurn)`} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([CreateTokens([TokenWithPT(PT(1,1),Colorless,[],[Creature],SubtypesList([Tetravite]),[`+rule+`])],[])]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONTokenEnchantmentRuleTetravusComplete(t *testing.T) {
	n := hostUntapRecord(t, "ATQ", "Tetravus")
	if _, err := Compile(n); err != nil {
		t.Fatalf("complete linked payment must compile: %v", err)
	}
}

func TestRONTokenEnchantmentRuleTetravusCreation(t *testing.T) {
	n := hostUntapRecord(t, "ATQ", "Tetravus")
	rules := n.Fields["Rules"]
	rules.Args = rules.Args[:3]
	n.Fields["Rules"] = rules
	factory, err := Compile(n)
	if err != nil {
		t.Fatal(err)
	}
	a := &ronResolutionCounterPayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A"), choice: 2}, accept: true}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast accepted")
	}
	a.ManaPool().Add(core.Colorless, 6)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(card.ID()).Counters[core.P1P1] != 3 {
		t.Fatal("missing entry counters")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("opponent upkeep triggered")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	clone := g.Clone()
	g.ResolveStack()
	if g.FindPermanent(card.ID()).Counters[core.P1P1] != 1 || len(g.FilterBattlefield(mage.IsToken)) != 2 {
		t.Fatal("counter payment or creation count")
	}
	if len(clone.FilterBattlefield(mage.IsToken)) != 0 {
		t.Fatal("pending clone shared tokens")
	}
	g.BouncePermanentToHand(g.FindPermanent(card.ID()))
	for _, token := range g.FilterBattlefield(mage.IsToken) {
		aura := mage.NewAura("Aura", "", mage.WithCastTarget(mage.TargetPermanent(mage.AnyPermanent)))
		aura.SetOwner(a.PlayerID())
		a.AddToHand(aura)
		if err := g.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{token.ID()}, 0); err == nil {
			t.Fatal("orphan token lost restriction")
		}
	}
}

func TestRONTokenEnchantmentRuleInventory(t *testing.T) {
	src := `Card(Name:"Tokens",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([CreateTokens([TokenWithPT(PT(1,1),Colorless,[],[Creature],SubtypesList([Tetravite]),[` + tokenEnchantmentRule + `])],[])]))])`
	report, err := InspectFS(fstest.MapFS{"tokens.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"PermanentRuleEffect": "token_rule", "ThisPermanent": "permanent_reference", "CantBeEnchanted": "permanent_rule_operation"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s in %s", name, role)
		}
	}
}
