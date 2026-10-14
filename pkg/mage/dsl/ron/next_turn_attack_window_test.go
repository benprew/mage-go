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

func TestRONNextTurnAttackWindowWallOfDust(t *testing.T) {
	g, a, b := ronGame()
	wall := g.PutOnBattlefield(repositoryCard(t, "LEG", "Wall of Dust"), a.PlayerID())
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 6), b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: wall.ID(), TargetID: attacker.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("block did not trigger")
	}
	g.ResolveStack()
	attacker.RevokeBaseAttr(core.AttrSummonSick)
	if !attacker.CanDeclareAsAttacker(g) {
		t.Fatal("blocked before next turn")
	}
	g.RunStepWithPriority(core.Cleanup)
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(core.PrecombatMain)
	if g.FindPermanent(attacker.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("allowed attack in next turn")
	}
	g.RunStepWithPriority(core.Cleanup)
	if !g.FindPermanent(attacker.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("restriction survived cleanup")
	}
}

func TestRONNextTurnAttackWindowNativeCombat(t *testing.T) {
	g, a, b := ronGame()
	wall := g.PutOnBattlefield(repositoryCard(t, "LEG", "Wall of Dust"), b.PlayerID())
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 6, mage.WithKeyword(core.Shroud)), a.PlayerID())
	attacker.RevokeBaseAttr(core.AttrSummonSick)
	if wall.CanDeclareAsAttacker(g) {
		t.Fatal("printed defender missing")
	}
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: wall.ID(), AttackerID: attacker.ID()}})
	g.ExecuteCombatDamage()
	if g.FindPermanent(wall.ID()).Damage != 2 || g.FindPermanent(attacker.ID()).Damage != 1 {
		t.Fatal("printed power or toughness changed")
	}
	g.RunStepWithPriority(core.EndCombat)
	g.RunStepWithPriority(core.Cleanup)
	g.MutablePermanent(attacker.ID()).Tapped = false
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(core.PrecombatMain)
	if !g.FindPermanent(attacker.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("other player's turn started window")
	}
	g.RunStepWithPriority(core.Cleanup)
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(core.PrecombatMain)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
	if g.IsAttackingInCombat(attacker.ID()) {
		t.Fatal("native declaration ignored restriction")
	}
	g.RunStepWithPriority(core.Cleanup)
	if !g.FindPermanent(attacker.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("native restriction did not expire")
	}
}

func TestRONNextTurnAttackWindowIdentity(t *testing.T) {
	for _, change := range []string{"none", "source-left", "source-phase", "return-before", "return-after", "phase-before", "phase-after", "phase-cycle", "control-before", "control-after", "abilities", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			wall := g.PutOnBattlefield(repositoryCard(t, "LEG", "Wall of Dust"), a.PlayerID())
			card := mage.NewCreature("Attacker", "", 2, 6)
			attacker := g.PutOnBattlefield(card, b.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: wall.ID(), TargetID: attacker.ID(), PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("missing pending event")
			}
			switch change {
			case "source-left":
				g.RemoveFromBattlefield(wall)
			case "source-phase":
				g.PhaseOut(wall.ID())
			case "return-before":
				g.RemoveFromBattlefield(attacker)
				attacker = g.PutOnBattlefield(card, b.PlayerID())
			case "phase-before":
				g.PhaseOut(attacker.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(attacker.ID()))
			case "control-before":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: wall.ID(), TargetID: attacker.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			pendingClone := g.Clone()
			for _, game := range []*mage.Game{g, pendingClone} {
				game.ResolveStack()
				current := game.FindPermanentIncludingPhased(attacker.ID())
				switch change {
				case "return-after":
					game.RemoveFromBattlefield(current)
					current = game.PutOnBattlefield(card, b.PlayerID())
				case "phase-after":
					game.PhaseOut(current.ID())
				case "control-after":
					game.AddControlEffect(mage.ControlEffectSpec{SourceID: wall.ID(), TargetID: current.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "abilities":
					if err := mage.ApplyEffect(game, mage.Until(core.Indefinite, mage.RemoveAbilities(mage.This)), current.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				game.RunStepWithPriority(core.Cleanup)
				game.AdvanceToNextTurn()
				game.RunStepWithPriority(core.PrecombatMain)
				if change == "phase-before" || change == "phase-after" {
					game.PhaseIn([]uuid.UUID{current.ID()})
				}
				current = game.MutablePermanent(current.ID())
				current.RevokeBaseAttr(core.AttrSummonSick)
				wantBlocked := change != "return-before" && change != "return-after" && change != "phase-before" && change != "control-before"
				if current.CanDeclareAsAttacker(game) == wantBlocked {
					t.Fatal("wrong first turn or incarnation")
				}
				game.RunStepWithPriority(core.Cleanup)
				game.AdvanceToNextTurn()
				game.RunStepWithPriority(core.PrecombatMain)
				if game.FindPermanent(current.ID()).CanDeclareAsAttacker(game) == (change == "control-before") {
					t.Fatal("wrong second turn or expiry")
				}
			}
		})
	}
}

func TestRONNextTurnAttackWindowInvalid(t *testing.T) {
	for _, body := range []string{
		`CreatePermanentRuleEffectUntil(ThisPermanent,[CantAttack()],DuringPlayersNextTurn(You))`,
		`CreatePermanentRuleEffectUntil(ThisPermanent,[CantAttack],DuringPlayersNextTurn)`,
		`CreatePermanentRuleEffectUntil(ThisPermanent,[CantAttack],DuringPlayersNextTurn())`,
		`CreatePermanentRuleEffectUntil(ThisPermanent,[CantAttack],DuringPlayersNextTurn(You,You))`,
		`CreatePermanentRuleEffectUntil(ThisPermanent,[CantAttack],DuringPlayersNextTurn(Ref_TargetPlayer))`,
		`CreatePermanentRuleEffectUntil(Trigger_ThatOtherPermanent,[CantAttack],DuringPlayersNextTurn(You))`,
		`CreatePermanentRuleEffectUntil(ThisPermanent,[CantAttack],UntilPlayersNextTurn(You))`,
		`CreatePermanentRuleEffectUntil(ThisPermanent,[CantAttack,CantBlock],DuringPlayersNextTurn(You))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:4),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONNextTurnAttackWindowCastingAndLoad(t *testing.T) {
	g, a, _ := ronGame()
	card := repositoryCard(t, "LEG", "Wall of Dust")
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("failed cast did not roll back")
	}
	a.ManaPool().Add(core.Red, 1)
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(card.ID()) == nil || a.ManaPool().TotalMana() != 0 {
		t.Fatal("printed casting payment or entry failed")
	}
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	record := strings.Split(string(data), "\n")[hostUntapRecord(t, "LEG", "Wall of Dust").Position.Line-1]
	files := fstest.MapFS{"wall.ron": {Data: []byte(record)}}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([CreatePermanentRuleEffectUntil(Trigger_ThatOtherPermanent,[CantAttack],DuringPlayersNextTurn(You))]))])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Wall of Dust") || registry.CardRegistered("Invalid") {
		t.Fatal("invalid collection replaced registry")
	}
}

func TestRONNextTurnAttackWindowIndependentBindings(t *testing.T) {
	g, a, b := ronGame()
	opponents := make([]*mage.Permanent, 0, 4)
	for range 2 {
		wall := g.PutOnBattlefield(repositoryCard(t, "LEG", "Wall of Dust"), a.PlayerID())
		for range 2 {
			other := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 8), b.PlayerID())
			opponents = append(opponents, other)
			g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: wall.ID(), TargetID: other.ID(), PlayerID: a.PlayerID()})
		}
		g.RemoveFromBattlefield(wall)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatal("independent events lost")
	}
	g.ResolveStack()
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(core.PrecombatMain)
	for _, other := range opponents {
		if g.FindPermanent(other.ID()).CanDeclareAsAttacker(g) {
			t.Fatal("factory or event overwrote subject")
		}
	}
	g.RunStepWithPriority(core.Cleanup)
	for _, other := range opponents {
		if !g.FindPermanent(other.ID()).CanDeclareAsAttacker(g) {
			t.Fatal("independent window did not expire")
		}
	}
}

func TestRONNextTurnAttackWindowContexts(t *testing.T) {
	for _, subject := range []string{"ThisPermanent", "HostPermanent", "Ref_TargetPermanent"} {
		body := `ActionList([CreatePermanentRuleEffectUntil(` + subject + `,[CantAttack],DuringPlayersNextTurn(ControllerOfPermanent(` + subject + `)))])`
		kind, rule := "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),`+body+`)`
		if subject == "HostPermanent" {
			kind, rule = "Enchantment", `EnchantPermanent(IsCardtype(Creature)),`+rule
		}
		if subject == "Ref_TargetPermanent" {
			kind, rule = "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],`+body+`))`
		}
		subtypes := "[]"
		if subject == "HostPermanent" {
			subtypes = "[Aura]"
		}
		nodes, err := Parse(strings.NewReader(`Card(Name:"Context",Typeline:(Supertypes:[],Cardtypes:[`+kind+`],Subtypes:`+subtypes+`),Rules:[`+rule+`])`), "context.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 8), b.PlayerID())
		var p *mage.Permanent
		if subject == "Ref_TargetPermanent" {
			card := factory()
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{other.ID()}, 0); err != nil {
				t.Fatal(err)
			}
		} else {
			p = g.PutOnBattlefield(factory(), a.PlayerID())
			if subject == "HostPermanent" {
				g.Attach(p.ID(), other.ID())
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || g.StackSize() != 0 {
				t.Fatal("unpaid activation installed rule")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
		}
		g.ResolveStack()
		if p != nil {
			g.RemoveFromBattlefield(p)
		}
		g.AdvanceToNextTurn()
		g.RunStepWithPriority(core.PrecombatMain)
		if subject != "ThisPermanent" && g.FindPermanent(other.ID()).CanDeclareAsAttacker(g) {
			t.Fatal("typed context did not install rule")
		}
	}
}

func TestRONNextTurnAttackWindowInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	record := strings.Split(string(data), "\n")[hostUntapRecord(t, "LEG", "Wall of Dust").Position.Line-1]
	report, err := InspectFS(fstest.MapFS{"wall.ron": {Data: []byte(record)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory rejected complete card", err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "ControllerOfPermanent" && strings.Contains(signature.Context, "DuringPlayersNextTurn") {
			found = true
			if signature.Role != "player_reference" {
				t.Fatalf("operand role %s", signature.Role)
			}
		}
	}
	if !found {
		t.Fatal("missing duration operand")
	}
}
