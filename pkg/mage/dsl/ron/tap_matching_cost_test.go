package ron

import (
	"fmt"
	"maps"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTapMatchingHandOfJustice(t *testing.T) {
	for _, scenario := range []string{"normal", "insufficient", "sick-source", "shroud", "target-return", "target-phase", "source-left", "source-phase", "target-shroud-after", "indestructible", "regenerate", "copy", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "FEM", "Hand of Justice")
			card.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			a.AddToHand(card)
			a.ManaPool().Add(core.White, 1)
			a.ManaPool().Add(core.Colorless, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			if a.ManaPool().TotalMana() != 5 || len(a.Hand()) != 1 {
				t.Fatal("cast rollback")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.MutablePermanent(card.ID())
			if scenario != "sick-source" {
				source.RevokeBaseAttr(core.AttrSummonSick)
			}
			payers := make([]uuid.UUID, 0, 3)
			for range 3 {
				p := g.PutOnBattlefield(mage.NewCreature("White", "{W}", 1, 2), a.PlayerID())
				payers = append(payers, p.ID())
			}
			blue := g.PutOnBattlefield(mage.NewCreature("Blue", "{U}", 1, 2), a.PlayerID())
			enemy := g.PutOnBattlefield(mage.NewCreature("Enemy", "{W}", 1, 2), b.PlayerID())
			victim := mage.NewCreature("Victim", "", 4, 4)
			victim.SetOwner(b.PlayerID())
			g.PutOnBattlefield(victim, b.PlayerID())
			if scenario == "insufficient" {
				g.TapPermanent(g.FindPermanent(payers[0]))
			}
			if scenario == "shroud" {
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()})
			if scenario == "insufficient" || scenario == "sick-source" || scenario == "shroud" {
				if err == nil || g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped {
					t.Fatal("invalid activation or rollback")
				}
				for _, id := range payers[1:] {
					if g.FindPermanent(id).Tapped {
						t.Fatal("partial payment")
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source.ID()).Tapped || g.StackSize() != 1 || g.FindPermanent(victim.ID()) == nil {
				t.Fatal("activation timing")
			}
			for _, id := range payers {
				if !g.FindPermanent(id).Tapped {
					t.Fatal("untapped payer")
				}
			}
			if g.FindPermanent(blue.ID()).Tapped || g.FindPermanent(enemy.ID()).Tapped {
				t.Fatal("ineligible payer")
			}
			switch scenario {
			case "target-return":
				g.BouncePermanentToHand(g.FindPermanent(victim.ID()))
				b.RemoveFromHand(victim.ID())
				g.PutOnBattlefield(victim, b.PlayerID())
			case "target-phase":
				g.PhaseOut(victim.ID())
			case "target-shroud-after":
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
			case "indestructible":
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Indestructible)
			case "regenerate":
				g.AddRegenerationShield(victim.ID())
			case "source-phase":
				g.PhaseOut(source.ID())
			case "source-left":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			if scenario == "clone" {
				clone.PhaseOut(victim.ID())
			}
			g.ResolveStack()
			clone.ResolveStack()
			survives := scenario == "target-return" || scenario == "target-phase" || scenario == "target-shroud-after" || scenario == "indestructible" || scenario == "regenerate"
			if (g.FindPermanentIncludingPhased(victim.ID()) != nil) != survives {
				t.Fatal("resolution identity")
			}
			if scenario == "regenerate" && !g.FindPermanent(victim.ID()).Tapped {
				t.Fatal("regeneration did not tap target")
			}
			if scenario == "clone" && clone.FindPermanentIncludingPhased(victim.ID()) == nil {
				t.Fatal("clone target lost")
			}
		})
	}
}

func TestRONTapMatchingSourceAndIndependentFactories(t *testing.T) {
	for _, cost := range []string{`TapAPermanent(And([IsUntapped,IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]))`, `TapNumberPermanents(Integer(1),IsCardtype(Creature))`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Tap source",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:3),Rules:[Activated(`+cost+`,ActionList([GainLife(Integer(1))]))])`), "source.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		first, second := factory(), factory()
		g.PutOnBattlefield(first, a.PlayerID())
		g.PutOnBattlefield(second, b.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if !g.FindPermanent(first.ID()).Tapped || g.FindPermanent(second.ID()).Tapped {
			t.Fatal("source or instance")
		}
		clone := g.Clone()
		g.ResolveStack()
		if a.Life() != 21 || b.Life() != 20 {
			t.Fatal("payer controller")
		}
		clone.ResolveStack()
		if clone.GetPlayer(a.PlayerID()).Life() != 21 {
			t.Fatal("clone resolution")
		}
	}
}

func TestRONTapMatchingInvalid(t *testing.T) {
	for i, cost := range []string{`TapAPermanent`, `TapAPermanent()`, `TapAPermanent(AnyPermanent,AnyPermanent)`, `TapNumberPermanents(Integer(0),AnyPermanent)`, `TapNumberPermanents(Integer(-1),AnyPermanent)`, `TapNumberPermanents(ValueX,AnyPermanent)`, `TapNumberPermanents(Integer(1),And([]))`, `TapNumberPermanents(Integer(1))`, `TapAPermanent(ControlledByAPlayer(SinglePlayer(Opponent)))`, `TapAPermanent(ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer)))`, `TapAPermanent(Or([ControlledByAPlayer(SinglePlayer(You)),AnyPermanent]))`, `TapAPermanent(And([ControlledByAPlayer(SinglePlayer(You)),ControlledByAPlayer(SinglePlayer(You))]))`} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(` + cost + `,ActionList([GainLife(Integer(1))]))])`
			nodes, err := Parse(strings.NewReader(src), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid cost accepted", cost)
			}
		})
	}
}

func TestRONTapMatchingWarMachineActivationClauses(t *testing.T) {
	record := chosenDiscardRecord(t, "FEM", "Vodalian War Machine")
	for _, scenario := range []string{"normal", "none", "tapped", "enemy", "phase", "copy", "source-return", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			node := record
			node.Fields = make(map[string]Node, len(record.Fields))
			maps.Copy(node.Fields, record.Fields)
			node.Fields["Rules"] = Node{Kind: List, Args: append([]Node(nil), record.Fields["Rules"].Args[:3]...)}
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
			if g.FindPermanent(source.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("defender lost")
			}
			var merfolk *mage.Permanent
			if scenario != "none" {
				controller := a.PlayerID()
				if scenario == "enemy" {
					controller = b.PlayerID()
				}
				merfolk = g.PutOnBattlefield(mage.NewCreature("Merfolk", "{U}", 1, 3, mage.WithSubTypes("Merfolk")), controller)
				if scenario == "tapped" {
					g.TapPermanent(merfolk)
				}
				if scenario == "phase" {
					g.PhaseOut(merfolk.ID())
				}
			}
			err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			invalid := scenario == "none" || scenario == "enemy" || scenario == "tapped" || scenario == "phase"
			if invalid {
				if err == nil || g.StackSize() != 0 {
					t.Fatal("ineligible payer accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(merfolk.ID()).Tapped || g.FindPermanent(source.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("payment or resolution timing")
			}
			g.ResolveStack()
			if !g.FindPermanent(source.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("attack permission missing")
			}
			if !g.FindPermanent(source.ID()).HasKeyword(core.Defender) {
				t.Fatal("permission removed defender")
			}
			g.UntapPermanent(g.FindPermanent(merfolk.ID()))
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			if scenario == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			if scenario == "source-return" {
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			}
			clone := g.Clone()
			g.ResolveStack()
			bonus := 2
			if scenario == "copy" {
				bonus = 4
			}
			if scenario == "source-return" {
				bonus = 0
			}
			checkConditionalPT(t, g, source.ID(), bonus, 4+bonus/2)
			checkConditionalPT(t, clone, source.ID(), 0, 4)
			clone.ResolveStack()
			checkConditionalPT(t, clone, source.ID(), bonus, 4+bonus/2)
			g.RunStepWithPriority(core.Cleanup)
			checkConditionalPT(t, g, source.ID(), 0, 4)
			g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
			if g.FindPermanent(source.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("permission did not expire")
			}
		})
	}
	if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "WasTappedToPayForAbilitiesOfPermanent") {
		t.Fatalf("full card must retain history gap: %v", err)
	}
}

func TestRONTapMatchingInventoryAndAtomicLoad(t *testing.T) {
	record := chosenDiscardRecord(t, "FEM", "Hand of Justice")
	data, err := os.ReadFile("../../../../cards/mtgish/FEM.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var source string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Hand of Justice"`) {
			source = line
		}
	}
	files := fstest.MapFS{"first.ron": {Data: []byte(source)}, "reprint.ron": {Data: []byte(source)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 2 || report.UniqueCards != 1 || len(report.Issues) != 0 {
		t.Fatal("reprint inspection", report.Issues)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Role == "unknown" {
			t.Fatal("unknown nested role", signature.Name)
		}
		if signature.Name == "TapNumberPermanents" && signature.Role == "cost" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing cost role")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard(record.Fields["Name"].String)
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard(first.Name())
	if err != nil {
		t.Fatal(err)
	}
	if first.ID() == second.ID() {
		t.Fatal("shared factory instance")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Unsupported",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[MissingRule])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("partial load accepted")
	}
	if len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("failed load replaced registry")
	}
}
