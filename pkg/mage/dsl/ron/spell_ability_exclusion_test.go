package ron

import (
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSpellAbilityExclusionGameplay(t *testing.T) {
	for _, change := range []string{"win", "lose", "other", "self", "opponent-source", "noncreature", "wrong-name", "departed", "returned", "phased", "removed", "late-blocker", "control", "source-return", "source-phased", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Goblin Artisans"), a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			a.SetLibrary([]mage.Card{mage.NewArtifact("Drawn", "")})
			victim := &mage.StackObject{ID: uuid.New(), SourceID: uuid.New(), Card: mage.NewArtifact("Victim", ""), Controller: a.PlayerID()}
			g.PushStack(victim)
			otherCard := mage.NewCreature("Goblin Artisans", "", 1, 1)
			if change == "noncreature" {
				otherCard = mage.NewArtifact("Goblin Artisans", "")
			}
			if change == "wrong-name" {
				otherCard = mage.NewCreature("Different", "", 1, 1)
			}
			controller := a.PlayerID()
			if change == "opponent-source" {
				controller = b.PlayerID()
			}
			other := g.PutOnBattlefield(otherCard, controller)
			blockerSource := other
			if change == "self" {
				blockerSource = source
			}
			blocker := &mage.StackObject{ID: uuid.New(), SourceID: blockerSource.ID(), IsAbility: true, Controller: b.PlayerID(), Targets: []uuid.UUID{victim.ID}}
			blocked := slices.Contains([]string{"other", "opponent-source", "departed", "returned", "phased"}, change)
			if blocked || change == "self" || change == "noncreature" || change == "wrong-name" || change == "removed" {
				g.PushStack(blocker)
			}
			if change == "departed" || change == "returned" {
				g.BouncePermanentToHand(other)
				if change == "returned" {
					a.RemoveFromHand(other.Card.ID())
					g.PutOnBattlefield(other.Card, a.PlayerID())
				}
			}
			if change == "phased" {
				g.PhaseOut(other.ID())
			}
			if change == "removed" {
				g.GetStack().RemoveByID(blocker.ID)
			}
			g.SetCoinFlipResults([]bool{change == "win", true, false})
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID})
			if blocked {
				if err == nil || source.Tapped || g.FlipCoin(a.PlayerID()) {
					t.Fatal("illegal activation paid or consumed coin")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !source.Tapped {
				t.Fatal("tap cost not paid")
			}
			ability := g.StackPeek()
			switch change {
			case "late-blocker":
				g.PushStack(blocker)
				g.GetStack().RemoveByID(ability.ID)
				g.PushStack(ability)
			case "control":
				victim.Controller = b.PlayerID()
			case "source-return":
				g.BouncePermanentToHand(source)
				a.RemoveFromHand(source.Card.ID())
				g.PutOnBattlefield(source.Card, a.PlayerID())
			case "source-phased":
				g.PhaseOut(source.ID())
			case "copy":
				g.CopyStackObjectDirect(ability, a.PlayerID(), false)
			}
			clone := g.Clone()
			g.ResolveTopOfStack()
			clone.ResolveTopOfStack()
			survives := change == "win" || change == "late-blocker" || change == "control"
			if (g.GetStack().FindByID(victim.ID) != nil) != survives || (clone.GetStack().FindByID(victim.ID) != nil) != survives {
				t.Fatal("wrong counter or legality result")
			}
			draws := 0
			if change == "win" {
				draws = 1
			}
			if len(a.Hand()) != draws || len(clone.GetPlayer(a.PlayerID()).Hand()) != draws {
				t.Fatal("wrong draw branch")
			}
			if survives && change != "win" {
				if g.FlipCoin(a.PlayerID()) {
					t.Fatal("illegal target consumed coin")
				}
			}
		})
	}
}

func TestRONSpellAbilityExclusionInvalid(t *testing.T) {
	for _, filter := range []string{`IsntTheTargetOfAnAbility`, `IsntTheTargetOfAnAbility()`, `IsntTheTargetOfAnAbility(AbilityOfAPermanent())`, `IsntTheTargetOfAnAbility(AbilityOfASource(IsCardtype(Creature)))`, `IsntTheTargetOfAnAbility(AbilityOfAPermanent(And([])))`, `IsntTheTargetOfAnAbility(AbilityOfAPermanent(And([IsCardtype(Creature),Or([])])))`, `IsntTheTargetOfAnAbility(AbilityOfAPermanent(Other(Ref_TargetPermanent)))`, `IsntTheTargetOfAnAbility(AbilityOfAPermanent(Or([Other(ThisPermanent),IsCardtype(Creature)])))`, `IsntTheTargetOfAnAbility(AbilityOfAPermanent(IsNamed(NamedCard(""))))`, `IsntTheTargetOfAnAbility(AbilityOfAPermanent(ControlledByAPlayer(SinglePlayer(You))))`} {
		node := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),Targeted([TargetSpell(`+filter+`)],ActionList([CounterSpell(Ref_TargetSpell)])))])`)
		if _, err := Compile(node); err == nil {
			t.Fatalf("accepted %s", filter)
		}
	}
}

func TestRONSpellAbilityExclusionContexts(t *testing.T) {
	filter := `IsntTheTargetOfAnAbility(AbilityOfAPermanent(IsCardtype(Creature)))`
	for _, rule := range []string{
		`SpellActions(Targeted([TargetSpell(` + filter + `)],ActionList([CounterSpell(Ref_TargetSpell)])))`,
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer,` + filter + `),ActionList([CounterSpell(Trigger_ThatSpell)]))`,
	} {
		typ := "Instant"
		if strings.HasPrefix(rule, "Trigger") {
			typ = "Enchantment"
		}
		node := nameScopeNode(t, `Card(Name:"Context",Typeline:(Supertypes:[],Cardtypes:[`+typ+`],Subtypes:[]),Rules:[`+rule+`])`)
		if _, err := Compile(node); err == nil {
			t.Fatal("accepted unsupported declaration context", rule)
		}
	}
}

func TestRONSpellAbilityExclusionRecords(t *testing.T) {
	record := staticBaseRecord(t, "ATQ", "Goblin Artisans")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	first := factory()
	if first.Abilities()[0] == factory().Abilities()[0] {
		t.Fatal("factories share ability state")
	}
	for range 2 {
		card := factory()
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile("../../../../cards/mtgish/ATQ.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var text string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Goblin Artisans"`) {
			text = line
		}
	}
	fs := fstest.MapFS{"card.ron": {Data: []byte(text)}}
	registry, err := LoadFS(fs, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatalf("real record load: %v", err)
	}
	report, err := InspectFS(fs, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal("inventory did not compile record")
	}

	roles := map[string]string{"IsntTheTargetOfAnAbility": "spell_filter", "AbilityOfAPermanent": "stack_ability_filter", "IsNamed": "permanent_filter", "Other": "permanent_filter"}
	for name, role := range roles {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name {
				found = true
				if signature.Role != role {
					t.Fatalf("%s role = %s", name, signature.Role)
				}
			}
		}
		if !found {
			t.Fatal("missing semantic role", name)
		}
	}
	invalid := strings.Replace(text, "Other(ThisPermanent)", "Other(Ref_TargetPermanent)", 1)
	fs["invalid.ron"] = &fstest.MapFile{Data: []byte(invalid)}
	if partial, err := LoadFS(fs, "*.ron"); err == nil || partial != nil {
		t.Fatal("invalid declaration published partial registry")
	}
	if err := LoadInto(registry, fs, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("failed reload replaced accepted registry")
	}
}

func TestRONSpellAbilityExclusionCastingAndInstances(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "ATQ", "Goblin Artisans")
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast accepted")
	}
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	source := g.FindPermanent(card.ID())
	if source == nil || source.CurrentPower(g) != 1 || source.CurrentToughness(g) != 1 || !source.HasSubType("Goblin") || !source.HasSubType("Artificer") {
		t.Fatal("wrong printed creature")
	}
	victim := &mage.StackObject{ID: uuid.New(), SourceID: uuid.New(), Card: mage.NewArtifact("Victim", ""), Controller: a.PlayerID()}
	g.PushStack(victim)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID}); err == nil || source.Tapped {
		t.Fatal("summoning sickness ignored")
	}
	source.RevokeBaseAttr(core.AttrSummonSick)
	victim.Controller = b.PlayerID()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID}); err == nil || source.Tapped {
		t.Fatal("opponent spell accepted")
	}
	victim.Controller = a.PlayerID()
	a.SetLibrary([]mage.Card{mage.NewArtifact("One", ""), mage.NewArtifact("Two", "")})
	g.SetCoinFlipResults([]bool{true, true, false})
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID}); err == nil {
		t.Fatal("tapped source paid twice")
	}
	source.Tapped = false
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID}); err != nil {
		t.Fatal("same source excluded its earlier ability", err)
	}
	other := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Goblin Artisans"), a.PlayerID())
	other.RevokeBaseAttr(core.AttrSummonSick)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), other.ID(), 0, []uuid.UUID{victim.ID}); err == nil || other.Tapped {
		t.Fatal("other source shared pending targets")
	}
	g.ResolveTopOfStack()
	g.ResolveTopOfStack()
	if len(a.Hand()) != 2 || g.GetStack().FindByID(victim.ID) == nil || g.FlipCoin(a.PlayerID()) {
		t.Fatal("independent source abilities or draws failed")
	}
}

func TestRONSpellAbilityExclusionSourceVisit(t *testing.T) {
	for _, newVisit := range []bool{false, true} {
		t.Run(map[bool]string{false: "old-source", true: "new-source"}[newVisit], func(t *testing.T) {
			g, a, _ := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Goblin Artisans"), a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			victim := &mage.StackObject{ID: uuid.New(), SourceID: uuid.New(), Card: mage.NewArtifact("Victim", ""), Controller: a.PlayerID()}
			g.PushStack(victim)
			blocker := &mage.StackObject{ID: uuid.New(), SourceID: source.ID(), IsAbility: true, Targets: []uuid.UUID{victim.ID}}
			g.PushStack(blocker)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID}); err != nil {
				t.Fatal(err)
			}
			ability := g.StackPeek()
			g.BouncePermanentToHand(source)
			a.RemoveFromHand(source.Card.ID())
			g.PutOnBattlefield(source.Card, a.PlayerID())
			if newVisit {
				g.GetStack().RemoveByID(blocker.ID)
				g.PushStack(&mage.StackObject{ID: uuid.New(), SourceID: source.ID(), IsAbility: true, Targets: []uuid.UUID{victim.ID}})
				g.GetStack().RemoveByID(ability.ID)
				g.PushStack(ability)
			}
			g.SetCoinFlipResults([]bool{false, true})
			g.ResolveTopOfStack()
			if (g.GetStack().FindByID(victim.ID) != nil) != newVisit {
				t.Fatal("source exclusion used physical card instead of exact visit")
			}
			if g.FlipCoin(a.PlayerID()) != !newVisit {
				t.Fatal("wrong legality timing")
			}
		})
	}
}
