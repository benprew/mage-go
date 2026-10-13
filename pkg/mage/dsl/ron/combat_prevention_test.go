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

const globalCombatPrevention = `CreatePreventDamageUntil(CombatDamageWouldBeDealt, [PreventThatDamage], UntilEndOfTurn)`

func TestRONGlobalCombatPreventionRealSpells(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
	}{
		{"LEA", "Fog", core.Green}, {"LEB", "Fog", core.Green},
		{"2ED", "Fog", core.Green}, {"3ED", "Fog", core.Green},
		{"CED", "Fog", core.Green}, {"CEI", "Fog", core.Green},
		{"LEG", "Darkness", core.Black}, {"LEG", "Holy Day", core.White},
	} {
		t.Run(tc.set+"/"+tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid spell accepted")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed cast changed hand or stack")
			}
			a.ManaPool().Add(tc.color, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			before := g.GetEffects().ApplyReplacements(mage.NewDamageToPlayerAction(uuid.Nil, b.PlayerID(), 2, true), g)
			if before == nil || a.ManaPool().TotalMana() != 0 || len(g.StackPeek().Targets) != 0 {
				t.Fatal("prevention started before resolution or cast used targets")
			}
			g.ResolveStack()
			if len(a.Graveyard()) != 1 {
				t.Fatal("resolved spell did not leave the stack")
			}
			assertRONGlobalCombatPrevention(t, g, a.PlayerID(), b.PlayerID())
		})
	}
}

func assertRONGlobalCombatPrevention(t *testing.T, g *mage.Game, a, b uuid.UUID) {
	t.Helper()
	attackerCard := mage.NewCreature("Attacker", "", 4, 8, mage.WithKeyword(core.FirstStrike), mage.WithKeyword(core.Trample), mage.WithKeyword(core.Lifelink))
	attackerCard.SetOwner(a)
	blockerCard := mage.NewCreature("Blocker", "", 3, 3)
	blockerCard.SetOwner(b)
	unblockedCard := mage.NewCreature("Unblocked", "", 2, 8)
	unblockedCard.SetOwner(a)
	attacker := g.PutOnBattlefield(attackerCard, a)
	blocker := g.PutOnBattlefield(blockerCard, b)
	unblocked := g.PutOnBattlefield(unblockedCard, a)
	for range 2 {
		g.GetCombat().Reset()
		g.GetCombat().AddAttacker(attacker.ID(), b)
		g.GetCombat().AddAttacker(unblocked.ID(), b)
		g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: attacker.ID()}})
		g.ExecuteCombatDamage()
		if g.GetPlayer(a).Life() != 20 || g.GetPlayer(b).Life() != 20 || g.FindPermanent(attacker.ID()).Damage != 0 || g.FindPermanent(blocker.ID()).Damage != 0 {
			t.Fatal("combat damage or lifelink passed through prevention")
		}
	}
	g.DealDamageToPlayer(g.GetPlayer(b), 1, attacker.ID())
	g.DealDamageToPermanent(g.FindPermanent(blocker.ID()), 1, unblocked.ID())
	if g.GetPlayer(b).Life() != 19 || g.FindPermanent(blocker.ID()).Damage != 1 {
		t.Fatal("prevention stopped noncombat damage")
	}
	clone := g.Clone()
	clone.RunStepWithPriority(core.Cleanup)
	if clone.GetEffects().ApplyReplacements(mage.NewDamageToPlayerAction(uuid.Nil, b, 2, true), clone) == nil {
		t.Fatal("prevention survived cleanup")
	}
	if g.GetEffects().ApplyReplacements(mage.NewDamageToPlayerAction(uuid.Nil, b, 2, true), g) != nil {
		t.Fatal("clone cleanup changed original prevention")
	}
	g.RunStepWithPriority(core.Cleanup)
	g.GetCombat().Reset()
	g.GetCombat().AddAttacker(unblocked.ID(), b)
	g.ExecuteCombatDamage()
	if g.GetPlayer(b).Life() != 17 {
		t.Fatal("combat damage did not resume after cleanup")
	}
}

func TestRONGlobalCombatPreventionSporeFlower(t *testing.T) {
	for _, change := range []string{"none", "leave-return", "phase", "control", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "FEM", "Spore Flower")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep placed counter")
			}
			for count := 1; count <= 3; count++ {
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
				if g.FindPermanent(source.ID()).Counters[core.Spore] != (count - 1) {
					t.Fatal("counter placed before resolution")
				}
				g.ResolveStack()
				if g.FindPermanent(source.ID()).Counters[core.Spore] != count {
					t.Fatal("wrong upkeep counter amount")
				}
				if count < 3 {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err == nil {
						t.Fatal("insufficient counters accepted")
					}
					if g.StackSize() != 0 || g.FindPermanent(source.ID()).Counters[core.Spore] != count {
						t.Fatal("failed payment changed counters")
					}
				}
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(source.ID()).Counters[core.Spore] != 0 || g.FindPermanent(source.ID()).Tapped {
				t.Fatal("wrong counter or tap payment")
			}
			switch change {
			case "leave-return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			if clone.GetEffects().ApplyReplacements(mage.NewDamageToPlayerAction(uuid.Nil, b.PlayerID(), 2, true), clone) == nil {
				t.Fatal("resolution changed pending clone")
			}
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				assertRONGlobalCombatPrevention(t, game, a.PlayerID(), b.PlayerID())
			}
		})
	}
}

func TestRONGlobalCombatPreventionInvalidAndInventory(t *testing.T) {
	for _, action := range []string{
		`CreatePreventDamageUntil`, `CreatePreventDamageUntil()`,
		`CreatePreventDamageUntil(CombatDamageWouldBeDealt, [PreventThatDamage])`,
		`CreatePreventDamageUntil(CombatDamageWouldBeDealt, [PreventThatDamage], UntilEndOfTurn, UntilEndOfTurn)`,
		strings.Replace(globalCombatPrevention, "CombatDamageWouldBeDealt", "CombatDamageWouldBeDealt()", 1),
		strings.Replace(globalCombatPrevention, "CombatDamageWouldBeDealt", "DamageWouldBeDealt", 1),
		strings.Replace(globalCombatPrevention, "[PreventThatDamage]", "PreventThatDamage", 1),
		strings.Replace(globalCombatPrevention, "[PreventThatDamage]", "[]", 1),
		strings.Replace(globalCombatPrevention, "[PreventThatDamage]", "[PreventThatDamage()]", 1),
		strings.Replace(globalCombatPrevention, "[PreventThatDamage]", "[PreventThatDamage, PreventThatDamage]", 1),
		strings.Replace(globalCombatPrevention, "[PreventThatDamage]", "[GainLife(Integer(1))]", 1),
		strings.Replace(globalCombatPrevention, "UntilEndOfTurn", "UntilEndOfTurn()", 1),
		strings.Replace(globalCombatPrevention, "UntilEndOfTurn", "UntilEndOfCombat", 1),
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([%s]))])`, action)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
	for _, rule := range []string{globalCombatPrevention, `Activated(` + globalCombatPrevention + `, ActionList([]))`, `PermanentLayerEffect(HostPermanent, [` + globalCombatPrevention + `])`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted context %s", rule)
		}
	}
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + globalCombatPrevention + `]))])`
	report, err := InspectFS(fstest.MapFS{"prevention.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"CombatDamageWouldBeDealt": "prevention_event", "PreventThatDamage": "prevention_operation", "UntilEndOfTurn": "duration"} {
		found := false
		for _, entry := range report.Signatures {
			if entry.Name == name {
				found = true
				if entry.Role != role {
					t.Fatalf("%s role=%s want=%s", name, entry.Role, role)
				}
			}
		}
		if !found {
			t.Fatalf("missing %s", name)
		}
	}
}

func TestRONGlobalCombatPreventionReplacementOrder(t *testing.T) {
	for mode := range 2 {
		a := mage.NewBasePlayer("A")
		b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B"), mode: mode}
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		g.SetTurn(1)
		card := repositoryCard(t, "LEG", "Darkness")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Black, 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 8), b.PlayerID())
		g.SetCreatureDamageRedirect(victim.ID(), b.PlayerID())
		if result := g.GetEffects().ApplyReplacements(mage.NewDamageToCreatureAction(uuid.Nil, victim.ID(), 3, true), g); result != nil {
			t.Fatal("redirected combat damage escaped prevention")
		}
		if b.prompts != 1 {
			t.Fatalf("replacement choices=%d", b.prompts)
		}
		victim.GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
		if result := g.GetEffects().ApplyReplacements(mage.NewDamageToCreatureAction(uuid.Nil, victim.ID(), 3, true), g); result == nil {
			t.Fatal("unpreventable damage was prevented")
		}
	}
}

func TestRONGlobalCombatPreventionLateCopyAndCounter(t *testing.T) {
	g, a, b := ronGame()
	g.SetActivePlayerIndex(1)
	g.SetStep(core.PostcombatMain)
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 8), b.PlayerID())
	g.GetCombat().AddAttacker(attacker.ID(), a.PlayerID())
	g.ExecuteCombatDamage()
	if a.Life() != 18 {
		t.Fatal("initial combat did not deal damage")
	}
	card := repositoryCard(t, "LEG", "Holy Day")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	if g.CopyStackObjectDirect(original, b.PlayerID(), false) == nil {
		t.Fatal("spell copy failed")
	}
	g.CounterStackObject(original.ID)
	clone := g.Clone()
	g.ResolveStack()
	if a.Life() != 18 || g.GetEffects().ApplyReplacements(mage.NewDamageToPlayerAction(attacker.ID(), a.PlayerID(), 2, true), g) != nil {
		t.Fatal("copy failed or prevention changed past damage")
	}
	if clone.GetEffects().ApplyReplacements(mage.NewDamageToPlayerAction(attacker.ID(), a.PlayerID(), 2, true), clone) == nil {
		t.Fatal("original resolution changed clone")
	}
	clone.CounterStackObject(clone.StackPeek().ID)
	clone.ResolveStack()
	if clone.GetEffects().ApplyReplacements(mage.NewDamageToPlayerAction(attacker.ID(), a.PlayerID(), 2, true), clone) == nil {
		t.Fatal("countered spell installed prevention")
	}
}
