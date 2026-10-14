package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const causePrevention = `CreatePreventDamageUntil(ASpellOrAbilityWouldCauseASourceToDealDamageToRecipient(TargetsPermanent(Ref_TargetPermanent),AnyDamageSource,Permanent(Ref_TargetPermanent)),[PreventThatDamage],UntilEndOfTurn)`

func TestRONDamageCauseSilhouette(t *testing.T) {
	for _, state := range []string{"spell", "ability", "departed-ability", "untargeted", "other-source", "copy", "clone", "return", "phase-cycle", "unpreventable", "expire"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Silhouette")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			targetCard := mage.NewCreature("Victim", "", 1, 20)
			victim := g.PutOnBattlefield(targetCard, a.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 20), b.PlayerID())
			targets := []uuid.UUID{victim.ID()}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err == nil {
				t.Fatal("unpaid casting")
			}
			a.ManaPool().Add(core.Blue, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if a.ManaPool().TotalMana() != 0 {
				t.Fatal("casting payment")
			}
			want := 0
			switch state {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
				victim = g.PutOnBattlefield(targetCard, a.PlayerID())
				want = 1
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(victim.ID()))
			case "unpreventable":
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
				want = 1
			case "expire":
				g.RunStepWithPriority(core.Cleanup)
				want = 1
			case "clone":
				g = g.Clone()
			}
			action := `SpellDealsDamage(ThisSpell,Integer(1),Permanent(Ref_TargetPermanent))`
			if state == "other-source" {
				action = `PermanentDealsDamage(Ref_TargetPermanent1,Integer(1),Permanent(Ref_TargetPermanent2))`
			}
			rules := `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + action + `])))`
			if state == "untargeted" {
				rules = `SpellActions(ActionList([SpellDealsDamage(ThisSpell,Integer(1),EachPermanent(IsCardtype(Creature)))]))`
				targets = nil
				want = 1
			}
			if state == "other-source" {
				rules = `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature)),TargetPermanent(IsCardtype(Creature))],ActionList([` + action + `])))`
				targets = []uuid.UUID{other.ID(), victim.ID()}
			}
			if state == "ability" || state == "departed-ability" {
				source := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([PermanentDealsDamage(ThisPermanent,Integer(1),Permanent(Ref_TargetPermanent))])))`)(), b.PlayerID())
				if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, targets); err != nil {
					t.Fatal(err)
				}
				if state == "departed-ability" {
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				}
			} else {
				spell := permanentScopeFactory(t, "Instant", rules)()
				spell.SetOwner(b.PlayerID())
				g.GetPlayer(b.PlayerID()).AddToHand(spell)
				if err := g.CastSpellByID(b.PlayerID(), spell.ID(), targets, 0); err != nil {
					t.Fatal(err)
				}
			}
			if state == "copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			}
			g.ResolveStack()
			if got := g.FindPermanent(victim.ID()).Damage; got != want {
				t.Fatalf("damage %d, want %d", got, want)
			}
			g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, other.ID())
			if g.FindPermanent(victim.ID()).Damage != want+1 {
				t.Fatal("unrelated damage was prevented")
			}
		})
	}
}

func TestRONDamageCauseInvalid(t *testing.T) {
	for _, action := range []string{
		strings.Replace(causePrevention, "TargetsPermanent(Ref_TargetPermanent)", "TargetsPermanent()", 1),
		strings.Replace(causePrevention, "TargetsPermanent(Ref_TargetPermanent)", "TargetsPermanent(Ref_TargetPermanent,Ref_TargetPermanent)", 1),
		strings.Replace(causePrevention, "AnyDamageSource", "AnyDamageSource()", 1),
		strings.Replace(causePrevention, "AnyDamageSource", "IsCardtype(Creature)", 1),
		strings.Replace(causePrevention, ",Permanent(Ref_TargetPermanent)", ",Player(You)", 1),
		strings.Replace(causePrevention, ",Permanent(Ref_TargetPermanent)", ",Permanent(ThisPermanent)", 1),
		strings.ReplaceAll(causePrevention, "Ref_TargetPermanent", "Trigger_ThatPermanent"),
		strings.Replace(causePrevention, "[PreventThatDamage]", "[]", 1),
		strings.Replace(causePrevention, "[PreventThatDamage]", "[PreventThatDamage()]", 1),
		strings.Replace(causePrevention, "UntilEndOfTurn", "UntilEndOfCombat", 1),
	} {
		text := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + action + `])))])`
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", action)
		}
	}
}

func TestRONDamageCauseInstallationLegality(t *testing.T) {
	for _, state := range []string{"land", "player", "shroud-before", "shroud-after", "phase-before", "return-before"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Silhouette")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			targetCard := mage.NewCreature("Victim", "", 1, 20)
			victim := g.PutOnBattlefield(targetCard, b.PlayerID())
			id := victim.ID()
			if state == "land" {
				id = g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID()).ID()
			}
			if state == "player" {
				id = b.PlayerID()
			}
			if state == "shroud-before" {
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
			}
			a.ManaPool().Add(core.Blue, 2)
			err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{id}, 0)
			if state == "land" || state == "player" || state == "shroud-before" {
				if err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 2 {
					t.Fatal("illegal declaration spent resources")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch state {
			case "shroud-after":
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
			case "phase-before":
				g.PhaseOut(victim.ID())
			case "return-before":
				g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
				g.PutOnBattlefield(targetCard, b.PlayerID())
			}
			g.ResolveStack()
			if state == "phase-before" {
				g.PhaseIn([]uuid.UUID{victim.ID()})
			}
			if len(g.WaitingPreventionPermanentSources()) != 0 {
				t.Fatal("illegal resolution installed prevention")
			}
		})
	}
}

func TestRONDamageCauseStrictLoadAndInventory(t *testing.T) {
	record := repositoryRecord(t, "LEG", "Silhouette", "silhouette.ron")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	first, second := factory(), factory()
	if first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("shared factory ability")
	}
	files := fstest.MapFS{"valid.ron": {Data: []byte(`Card(Name:"Fixture",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + causePrevention + `])))] )`)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "TargetsPermanent" && signature.Role == "damage_cause_filter" {
			found = true
		}
	}
	if !found {
		t.Fatal("cause filter role missing")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Unsupported",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnsupportedRule])`)}
	if err = LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Fixture") || registry.CardRegistered("Unsupported") {
		t.Fatal("atomic rejection")
	}
}

func TestRONDamageCauseIllegalTargetFacts(t *testing.T) {
	g, a, b := ronGame()
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 10), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 10), b.PlayerID())
	shield := repositoryCard(t, "LEG", "Silhouette")
	shield.SetOwner(a.PlayerID())
	a.AddToHand(shield)
	a.ManaPool().Add(core.Blue, 2)
	if err := g.CastSpellByID(a.PlayerID(), shield.ID(), []uuid.UUID{victim.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	spell := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature)),TargetPermanent(IsCardtype(Creature))],ActionList([SpellDealsDamage(ThisSpell,Integer(1),EachPermanent(IsCardtype(Creature)))])))`)()
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	if err := g.CastSpellByID(b.PlayerID(), spell.ID(), []uuid.UUID{victim.ID(), other.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
	g.ResolveStack()
	if g.FindPermanent(victim.ID()).Damage != 0 || g.FindPermanent(other.ID()).Damage != 1 {
		t.Fatal("illegal target lost its saved cause reference")
	}
}
