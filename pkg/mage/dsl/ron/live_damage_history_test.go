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

func TestRONLiveDamageHistoryRunesword(t *testing.T) {
	g, a, b := ronGame()
	sword := g.PutOnBattlefield(repositoryCard(t, "DRK", "Runesword"), a.PlayerID())
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 5), a.PlayerID())
	attacker.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
	if err := g.ActivateAbilityByIndex(a.PlayerID(), sword.ID(), 0, []uuid.UUID{attacker.ID()}); err == nil {
		t.Fatal("unpaid activation")
	}
	if sword.Tapped || g.StackSize() != 0 {
		t.Fatal("failed payment changed state")
	}
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), sword.ID(), 0, []uuid.UUID{attacker.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if attacker.CurrentPower(g) != 4 {
		t.Fatal("missing power bonus")
	}
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 5), b.PlayerID())
	g.AddRegenerationShield(victim.ID())
	g.DealDamageToPermanent(victim, 1, attacker.ID())
	g.DestroyPermanent(victim)
	if g.FindExiledCard(victim.Card.ID()) == nil || g.CreatureDeaths() != 0 {
		t.Fatal("live damage rule failed")
	}
	g.BouncePermanentToHand(attacker)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("missing leave trigger")
	}
	g.ResolveStack()
	if g.FindPermanent(sword.ID()) != nil {
		t.Fatal("Runesword not sacrificed")
	}
}

func TestRONLiveDamageHistoryInvalid(t *testing.T) {
	for _, action := range []string{
		`CreateEachPermanentRuleEffectUntil()`,
		`CreateEachPermanentRuleEffectUntil(WasDealtDamageByPermanentThisTurn(ThisPermanent,IsCardtype(Artifact)),[CantBeRegenerated],UntilEndOfTurn)`,
		`CreateEachPermanentRuleEffectUntil(WasDealtDamageByPermanentThisTurn(ThisPermanent,IsCardtype(Creature),IsCardtype(Creature)),[CantBeRegenerated],UntilEndOfTurn)`,
		`CreateReplaceWouldPutIntoGraveyardUntil(APermanentWouldDie(WasDealtDamageByPermanentThisTurn(ThisPermanent,IsCardtype(Creature))),[ExileItInstead],UntilEndOfTurn)`,
		`CreateEachPermanentRuleEffectUntil(WasDealtDamageByPermanentThisTurn(),[CantBeRegenerated],UntilEndOfTurn)`,
		`CreateEachPermanentRuleEffectUntil(WasDealtDamageByPermanentThisTurn(You),[CantBeRegenerated],UntilEndOfTurn)`,
		`CreateEachPermanentRuleEffectUntil(WasDealtDamageByPermanentThisTurn(Ref_TargetPermanent),[CantBeRegenerated],UntilEndOfTurn)`,
		`CreateEachPermanentRuleEffectUntil(WasDealtDamageByPermanentThisTurn(ThisPermanent),[CantBeRegenerated()],UntilEndOfTurn)`,
		`CreateEachPermanentRuleEffectUntil(WasDealtDamageByPermanentThisTurn(ThisPermanent),[CantBeRegenerated,Flying],UntilEndOfTurn)`,
		`CreateEachPermanentRuleEffectUntil(WasDealtDamageByPermanentThisTurn(ThisPermanent),[CantBeRegenerated],UntilEndOfCombat)`,
		`CreateEachPermanentRuleEffectUntil(And([WasDealtDamageByPermanentThisTurn(ThisPermanent),WasDealtDamageByPermanentThisTurn(ThisPermanent)]),[CantBeRegenerated],UntilEndOfTurn)`,
		`CreateReplaceWouldPutIntoGraveyardUntil(APermanentWouldDie(WasDealtDamageByPermanentThisTurn(ThisPermanent)),[ExileItInstead()],UntilEndOfTurn)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+action+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONLiveDamageHistoryTiming(t *testing.T) {
	for _, change := range []string{"prior-damage", "prevented", "zero", "other-source", "illegal", "phase-before", "return-before", "phase-cycle", "donor-departure", "source-departure", "recipient-return", "cleanup", "clone", "copy", "abilities", "sacrifice", "walker"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			sword := g.PutOnBattlefield(repositoryCard(t, "DRK", "Runesword"), a.PlayerID())
			sword.Card.SetOwner(a.PlayerID())
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 5), a.PlayerID())
			attacker.Card.SetOwner(a.PlayerID())
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
			victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 5), b.PlayerID())
			victim.Card.SetOwner(b.PlayerID())
			if change == "walker" {
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 5), b.PlayerID())
				victim.Card.SetOwner(b.PlayerID())
			}
			if change == "prior-damage" {
				g.DealDamageToPermanent(victim, 1, attacker.ID())
			}
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), sword.ID(), 0, []uuid.UUID{attacker.ID()}); err != nil {
				t.Fatal(err)
			}
			saved := g.CaptureDamageSource(attacker.ID())
			switch change {
			case "illegal":
				attacker.GrantBaseAttr(core.Shroud)
			case "phase-before":
				g.PhaseOut(attacker.ID())
			case "return-before":
				g.RemoveFromBattlefield(attacker)
				attacker = g.PutOnBattlefield(attacker.Card, a.PlayerID())
				saved = g.CaptureDamageSource(attacker.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			if change == "copy" && attacker.CurrentPower(g) != 6 {
				t.Fatal("copy lost power bonus")
			}
			amount := 1
			switch change {
			case "prevented":
				g.AddPreventionShield(victim.ID(), 1)
			case "zero":
				amount = 0
			case "other-source":
				saved = g.CaptureDamageSource(g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 5), a.PlayerID()).ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(attacker.ID()))
			case "donor-departure":
				g.RemoveFromBattlefield(g.FindPermanent(sword.ID()))
			case "source-departure":
				g.RemoveFromBattlefield(attacker)
			case "abilities":
				g.AddContinuousEffect(mage.RemoveAllAbilities(victim.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
			}
			if change != "prior-damage" {
				g.DealDamageToPermanentFromSource(victim, amount, saved)
			}
			switch change {
			case "recipient-return":
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(victim.Card, b.PlayerID())
			case "cleanup":
				g.RunStepWithPriority(core.Cleanup)
			case "clone":
				clone := g.Clone()
				clone.RunStepWithPriority(core.Cleanup)
				clone.AddRegenerationShield(victim.ID())
				clone.DestroyPermanent(clone.FindPermanent(victim.ID()))
				if clone.FindPermanent(victim.ID()) == nil {
					t.Fatal("clone cleanup retained rule")
				}
			}
			g.AddRegenerationShield(victim.ID())
			if change == "sacrifice" || change == "walker" {
				g.DoSacrifice(victim)
			} else {
				g.DestroyPermanent(victim)
			}
			want := change != "prevented" && change != "zero" && change != "other-source" && change != "illegal" && change != "phase-before" && change != "return-before" && change != "recipient-return" && change != "cleanup" && change != "walker"
			if (g.FindExiledCard(victim.Card.ID()) != nil) != want {
				t.Fatalf("exile=%v want=%v", g.FindExiledCard(victim.Card.ID()) != nil, want)
			}
		})
	}
}

func TestRONLiveDamageHistoryNativeAndBindings(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	record := ""
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Runesword"`) {
			record = line
		}
	}
	if record == "" {
		t.Fatal("missing record")
	}
	files := fstest.MapFS{"DRK.ron": &fstest.MapFile{Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	card, err := registry.CreateCard("Runesword")
	if err != nil {
		t.Fatal(err)
	}
	other, err := registry.CreateCard("Runesword")
	if err != nil {
		t.Fatal(err)
	}
	if card.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == other.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("factories share targets")
	}
	g, a, b := ronGame()
	first := g.PutOnBattlefield(card, a.PlayerID())
	second := g.PutOnBattlefield(other, a.PlayerID())
	targets := make([]*mage.Permanent, 2)
	for i := range targets {
		targets[i] = g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 5), a.PlayerID())
		targets[i].RevokeBaseAttr(core.AttrSummonSick)
	}
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{targets[0].ID(), targets[1].ID()})
	a.ManaPool().Add(core.Colorless, 6)
	for i, sword := range []*mage.Permanent{first, second} {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), sword.ID(), 0, []uuid.UUID{targets[i].ID()}); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	for _, attacker := range targets {
		victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 5), b.PlayerID())
		g.DealDamageToPermanent(victim, 1, attacker.ID())
		g.AddRegenerationShield(victim.ID())
		g.DestroyPermanent(victim)
		if g.FindExiledCard(victim.Card.ID()) == nil {
			t.Fatal("independent binding lost its source")
		}
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnknownRule])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection published")
	}
	report, err := InspectFS(fstest.MapFS{"DRK.ron": files["DRK.ron"]}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 || report.Records != 1 {
		t.Fatal("native record not compiled")
	}
	for _, signature := range report.Signatures {
		if signature.Name == "Ref_TargetPermanent" && strings.Contains(signature.Context, "WasDealtDamageByPermanentThisTurn") && signature.Role != "permanent_reference" {
			t.Fatalf("wrong history source role: %s", signature.Role)
		}
	}
}

func TestRONLiveDamageHistoryCollection(t *testing.T) {
	card := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([TapEachPermanent(WasDealtDamageByPermanentThisTurn(Ref_TargetPermanent))])))`)()
	g, a, b := ronGame()
	donor := g.PutOnBattlefield(card, a.PlayerID())
	source := g.PutOnBattlefield(mage.NewCreature("Source", "", 1, 5), a.PlayerID())
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 5), b.PlayerID())
	g.DealDamageToPermanent(victim, 1, source.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), donor.ID(), 0, []uuid.UUID{source.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !victim.Tapped || source.Tapped {
		t.Fatal("wrong history collection")
	}
}
