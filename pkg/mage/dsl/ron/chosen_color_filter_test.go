package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONChosenColorFilterGameplay(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phased", "clone", "copy", "control", "late-member"} {
		t.Run(state, func(t *testing.T) {
			rule := colorEntryRule + `,Activated(PayMana([ManaCostGeneric(1)]),ActionList([TapEachPermanent(And([IsNonToken,IsColor(TheChosenColor),ControlledByAPlayer(Opponent)]))]))`
			registry, err := LoadFS(fstest.MapFS{"color.ron": {Data: []byte(colorEntryFixture(rule))}}, "*.ron")
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			chooser := &ronColorEntryPlayer{Player: a, answer: core.Red}
			g.SetPlayerAt(0, chooser)
			card, err := registry.CreateCard("Color entry")
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			red := g.PutOnBattlefield(mage.NewCreature("Red", "{R}", 1, 5), b.PlayerID())
			blue := g.PutOnBattlefield(mage.NewCreature("Blue", "{U}", 1, 5), b.PlayerID())
			tokenCard := mage.NewCreature("Token", "{R}", 1, 5)
			token := g.PutOnBattlefield(tokenCard, b.PlayerID())
			token.IsToken = true
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err == nil || chooser.calls != 1 || red.Tapped {
				t.Fatal("failed cost changed state")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			if red.Tapped {
				t.Fatal("predicate ran at announcement")
			}
			wantRed := true
			switch state {
			case "departed", "returned":
				g.BouncePermanentToHand(source)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					chooser.answer = core.Blue
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: red.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				wantRed = false
			case "late-member":
				red = g.PutOnBattlefield(mage.NewCreature("Late", "{R}", 1, 5), b.PlayerID())
			}
			clone := g.Clone()
			if state == "clone" {
				clone.MutablePermanent(source.ID()).ChosenColor = core.Blue
			}
			g.ResolveStack()
			if clone.FindPermanent(red.ID()).Tapped {
				t.Fatal("clone shares resolution")
			}
			clone.ResolveStack()
			if g.FindPermanent(red.ID()).Tapped != wantRed || blue.Tapped || token.Tapped {
				t.Fatal("wrong current membership or saved choice")
			}
			if state == "clone" && (clone.FindPermanent(red.ID()).Tapped || !clone.FindPermanent(blue.ID()).Tapped) {
				t.Fatal("clone choice ignored")
			}
		})
	}
}

func TestRONChosenColorFilterInvalid(t *testing.T) {
	for _, rules := range []string{
		`Activated(PayMana([]),ActionList([TapEachPermanent(IsColor(TheChosenColor))]))`,
		colorEntryRule + `,Activated(PayMana([]),ActionList([TapEachPermanent(IsColor(TheChosenColor()))]))`,
		colorEntryRule + `,Activated(PayMana([]),ActionList([TapEachPermanent(IsColor(TheChosenColors))]))`,
		colorEntryRule + `,Activated(PayMana([]),Targeted([TargetPermanent(IsColor(TheChosenColor))],ActionList([TapPermanent(Ref_TargetPermanent)])))`,
		colorEntryRule + `,PermanentLayerEffect(ThisPermanent,[AdjustPTForEach(1,1,IsColor(TheChosenColor))])`,
		colorEntryRule + `,EachPermanentLayerEffect(IsColor(TheChosenColor),[AdjustPT(1,1)])`,
		colorEntryRule + `,Activated(PayMana([]),ActionList([GainLife(TheChosenColor)]))`,
	} {
		nodes, err := Parse(strings.NewReader(colorEntryFixture(rules)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted invalid context: %s", rules)
		}
	}
}

func TestRONChosenColorFilterRealRequirements(t *testing.T) {
	psychic := repositoryRecord(t, "DRK", "Psychic Allergy", "DRK.ron")
	jihad := repositoryRecord(t, "ARN", "Jihad", "ARN.ron")
	for _, node := range []Node{psychic, jihad} {
		if _, err := Compile(node); err == nil {
			t.Fatal("claimed unsupported complete card")
		}
	}
	rules := psychic.Fields["Rules"].Args
	entry, err := compilePermanentEntry(rules[0])
	if err != nil {
		t.Fatal(err)
	}
	trigger := rules[1]
	trigger.Args[0].Args[0] = Node{Kind: Symbol, Name: "AnyPlayer", Position: trigger.Args[0].Args[0].Position}
	triggerOption, err := compileTriggerRule(trigger, scope{permanent: true, colorEntryChoice: true})
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	chooser := &ronColorEntryPlayer{Player: a, answer: core.Red}
	g.SetPlayerAt(0, chooser)
	sourceCard := mage.NewEnchantment("Requirement", "", entry(), triggerOption())
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	g.PutOnBattlefield(mage.NewCreature("Red", "{R}", 1, 5), b.PlayerID())
	g.PutOnBattlefield(mage.NewCreature("Multicolor", "{R}{U}", 1, 5), b.PlayerID())
	g.PutOnBattlefield(mage.NewCreature("Other controller", "{R}", 1, 5), a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("upkeep did not queue")
	}
	g.BouncePermanentToHand(source)
	a.RemoveFromHand(sourceCard.ID())
	chooser.answer = core.Blue
	g.PutOnBattlefield(sourceCard, a.PlayerID())
	g.ResolveStack()
	if b.Life() != 18 || a.Life() != 20 {
		t.Fatalf("exact real damage body: life %d/%d", a.Life(), b.Life())
	}
}

func TestRONChosenColorFilterInventoryAndAtomicLoad(t *testing.T) {
	record := colorEntryFixture(colorEntryRule + `,Activated(PayMana([]),ActionList([TapEachPermanent(IsColor(TheChosenColor))]))`)
	report, err := InspectFS(fstest.MapFS{"color.ron": {Data: []byte(record)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal(report.Issues)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "TheChosenColor" && signature.Role == "color_choice_reference" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing color choice information role")
	}
	registry := mage.NewRegistry()
	registry.Register("Retained", func() mage.Card { return mage.NewArtifact("Retained", "") })
	bad := colorEntryFixture(`Activated(PayMana([]),ActionList([TapEachPermanent(IsColor(TheChosenColor))]))`)
	if err := LoadInto(registry, fstest.MapFS{"color.ron": {Data: []byte(record + "\n" + strings.Replace(bad, "Color entry", "Invalid", 1))}}, "*.ron"); err == nil {
		t.Fatal("partial collection accepted")
	}
	if _, err := registry.CreateCard("Retained"); err != nil {
		t.Fatal("rejected collection replaced registry")
	}
}
