package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONModalTriggerRelicBind(t *testing.T) {
	for _, scenario := range []string{"damage", "life", "walker", "shroud", "phase", "phase-cycle", "return", "source-left", "source-phase", "control", "copy", "retap", "prevention", "source-disabled", "wrong-host"} {
		t.Run(scenario, func(t *testing.T) {
			mode := 0
			if scenario == "life" {
				mode = 1
			}
			g, a, b := ronModalGame(mode)
			g.SetActivePlayerIndex(0)
			host := g.PutOnBattlefield(mage.NewArtifact("Host", ""), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewArtifact("Other", ""), a.PlayerID())
			card := repositoryCard(t, "LEG", "Relic Bind")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed cast changed zones")
			}
			a.ManaPool().Add(core.Blue, 1)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{other.ID()}, 0); err == nil {
				t.Fatal("own artifact accepted")
			}
			if a.ManaPool().TotalMana() != 3 {
				t.Fatal("illegal cast spent mana")
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			if source == nil || source.AttachedTo != host.ID() || a.ManaPool().TotalMana() != 0 {
				t.Fatal("Aura cast or attachment failed")
			}
			walkerCard := mage.NewPlaneswalker("Walker", "", 9)
			walkerCard.SetOwner(b.PlayerID())
			walker := g.PutOnBattlefield(walkerCard, b.PlayerID())
			a.targets = []uuid.UUID{b.PlayerID()}
			permanentTarget := scenario == "walker" || scenario == "shroud" || scenario == "phase" || scenario == "phase-cycle" || scenario == "return"
			if permanentTarget {
				a.targets = []uuid.UUID{walker.ID()}
			}
			if scenario == "source-disabled" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
			}
			tapped := host
			if scenario == "wrong-host" {
				tapped = other
			}
			g.TapPermanent(tapped)
			if a.prompts != 0 || b.Life() != 20 {
				t.Fatal("mode chosen or resolved at event time")
			}
			if scenario == "retap" {
				g.TapPermanent(host)
			}
			switch scenario {
			case "source-left":
				g.RemoveFromBattlefield(source)
			case "source-phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.PutTriggersOnStack()
			if scenario == "source-disabled" || scenario == "wrong-host" {
				if g.StackSize() != 0 || a.prompts != 0 {
					t.Fatal("inactive event triggered")
				}
				return
			}
			obj := g.StackPeek()
			if g.StackSize() != 1 || obj.ModeChoice != mode || len(obj.Targets) != 1 || a.prompts != 1 {
				t.Fatal("mode or target placement")
			}
			switch scenario {
			case "shroud":
				g.FindPermanent(walker.ID()).GrantBaseAttr(core.Shroud)
			case "phase":
				g.PhaseOut(walker.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(walker.ID()))
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(walker.ID()))
				walker = g.PutOnBattlefield(walkerCard, b.PlayerID())
			case "copy":
				a.targets = []uuid.UUID{a.PlayerID()}
				copied := g.CopyStackObjectDirect(obj, a.PlayerID(), true)
				if copied == nil || copied.ModeChoice != mode || obj.Targets[0] != b.PlayerID() {
					t.Fatal("copy changed original mode or targets")
				}
			case "prevention":
				g.AddPreventionShield(b.PlayerID(), 1)
			}
			clone := g.Clone()
			prompts := a.prompts
			clone.ResolveStack()
			if b.Life() != 20 || g.FindPermanentIncludingPhased(walker.ID()).Counters[core.Loyalty] != 9 {
				t.Fatal("clone shared state")
			}
			g.ResolveStack()
			if a.prompts != prompts {
				t.Fatal("resolution chose a mode")
			}
			wantLife := 19
			if scenario == "life" {
				wantLife = 21
			}
			if permanentTarget || scenario == "prevention" {
				wantLife = 20
			}
			if b.Life() != wantLife {
				t.Fatalf("life=%d want=%d", b.Life(), wantLife)
			}
			wantLoyalty := 9
			if scenario == "walker" || scenario == "phase-cycle" {
				wantLoyalty = 8
			}
			if g.FindPermanentIncludingPhased(walker.ID()).Counters[core.Loyalty] != wantLoyalty {
				t.Fatal("target legality or identity")
			}
			if scenario == "copy" && a.Life() != 19 {
				t.Fatal("copy did not use separate target")
			}
		})
	}
}

func TestRONModalTriggerInvalid(t *testing.T) {
	event := `WhenAPermanentBecomesTapped(SinglePermanent(HostPermanent))`
	life := `Targeted([TargetPlayer(AnyPlayer)], ActionList([PlayerAction(Ref_TargetPlayer, GainLife(Integer(1)))]))`
	for _, body := range []string{
		`Modal_ChooseOne()`, `Modal_ChooseOne([])`, `Modal_ChooseOne([` + life + `])`,
		`Modal_ChooseOne([` + life + `, ActionList([PlayerAction(Ref_TargetPlayer, GainLife(Integer(1)))])])`,
		`Modal_ChooseOne([` + life + `, ActionList([DestroyPermanent(Trigger_ThatPlayer)])])`,
		`Modal_ChooseOne([` + life + `, ` + life + `], [])`,
	} {
		src := fmt.Sprintf(`Card(Name: "Bad", Typeline: (Supertypes: [], Cardtypes: [Enchantment], Subtypes: [Aura]), Rules: [EnchantPermanent(AnyPermanent), TriggerA(%s, %s)])`, event, body)
		nodes, err := Parse(strings.NewReader(src), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONModalTriggerCollection(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Relic Bind"`) {
			record = line
		}
	}
	registry, err := LoadFS(fstest.MapFS{"relic.ron": {Data: []byte(record)}}, "*.ron")
	if err != nil || registry == nil {
		t.Fatalf("strict collection: %v", err)
	}
}

func TestRONModalTriggerBindingsAndSource(t *testing.T) {
	for _, subject := range []string{"HostPermanent", "ThisPermanent"} {
		t.Run(subject, func(t *testing.T) {
			g, a, b := ronModalGame(0)
			effectRef := "ThisPermanent"
			if subject == "HostPermanent" {
				effectRef = "Trigger_ThatPermanent"
			}
			src := fmt.Sprintf(`Card(Name: "Tap modes", Typeline: (Supertypes: [], Cardtypes: [Enchantment], Subtypes: [Aura]), Rules: [EnchantPermanent(AnyPermanent), TriggerA(WhenAPermanentBecomesTapped(SinglePermanent(%s)), Modal_ChooseOne([ActionList([PutCounters([ACounterOfTypeOnPermanent(ChargeCounter, %s)])]), ActionList([GainLife(Integer(2))])]))])`, subject, effectRef)
			nodes, err := Parse(strings.NewReader(src), "bindings.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			firstHost := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 5), b.PlayerID())
			secondHost := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 5), b.PlayerID())
			first := g.PutOnBattlefield(factory(), a.PlayerID())
			second := g.PutOnBattlefield(factory(), a.PlayerID())
			g.Attach(first.ID(), firstHost.ID())
			g.Attach(second.ID(), secondHost.ID())
			target, untouched := firstHost, secondHost
			if subject == "ThisPermanent" {
				target, untouched = first, second
			}
			g.TapPermanent(target)
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("instance trigger mismatch")
			}
			if subject == "HostPermanent" {
				g.Attach(first.ID(), secondHost.ID())
			}
			clone := g.Clone()
			clone.ResolveStack()
			if g.FindPermanent(target.ID()).Counters[core.Charge] != 0 {
				t.Fatal("clone shared event binding")
			}
			g.ResolveStack()
			if g.FindPermanent(target.ID()).Counters[core.Charge] != 1 || g.FindPermanent(untouched.ID()).Counters[core.Charge] != 0 {
				t.Fatal("mode followed a different instance or attachment")
			}
		})
	}
}

func TestRONModalTriggerLegalMenu(t *testing.T) {
	for _, choice := range []int{0, -1, 2} {
		t.Run(fmt.Sprint(choice), func(t *testing.T) {
			g, a, _ := ronModalGame(choice)
			src := `Card(Name: "Legal modes", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [TriggerA(WhenAPermanentBecomesTapped(SinglePermanent(ThisPermanent)), Modal_ChooseOne([Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([DestroyPermanent(Ref_TargetPermanent)])), ActionList([GainLife(Integer(2))])]))])`
			nodes, err := Parse(strings.NewReader(src), "menu.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			g.TapPermanent(source)
			g.PutTriggersOnStack()
			if choice != 0 {
				if g.StackSize() != 0 {
					t.Fatal("invalid choice left a trigger")
				}
				return
			}
			if g.StackSize() != 1 || g.StackPeek().ModeChoice != 1 || len(g.StackPeek().Targets) != 0 {
				t.Fatal("unavailable mode entered stack")
			}
			g.ResolveStack()
			if a.Life() != 22 {
				t.Fatal("filtered mode index changed")
			}
		})
	}
}
