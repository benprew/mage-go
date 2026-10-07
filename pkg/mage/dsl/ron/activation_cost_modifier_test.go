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

func TestRONActivationCostRealCards(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEA", "Gloom"}, {"ATQ", "Power Artifact"}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			hostCard := mage.NewArtifact("Host", "", mage.WithAbility(mage.Activate(mage.Costs(mage.ManaCostOf("{4}"), mage.Tap()), mage.GainLifeFor(mage.You, mage.Fixed(1)))))
			if tc.name == "Gloom" {
				hostCard = mage.NewEnchantment("Host", "{W}", mage.WithActivatedAbility(mage.GainLifeFor(mage.You, mage.Fixed(1)), mage.Tap()))
			}
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			targets := []uuid.UUID(nil)
			if tc.name == "Gloom" {
				a.ManaPool().Add(core.Colorless, 2)
				a.ManaPool().Add(core.Black, 1)
			} else {
				a.ManaPool().Add(core.Blue, 2)
				targets = []uuid.UUID{host.ID()}
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			want := 2
			if tc.name == "Gloom" {
				want = 3
			}
			b.ManaPool().Add(core.Colorless, want-1)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), host.ID(), 0, nil); err == nil {
				t.Fatal("underpayment accepted")
			}
			if g.FindPermanent(host.ID()).Tapped || b.ManaPool().TotalMana() != want-1 {
				t.Fatal("rollback")
			}
			b.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), host.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if b.Life() != 21 || b.ManaPool().TotalMana() != 0 {
				t.Fatal("incorrect activation")
			}
		})
	}
}

func TestRONActivationCostInvalid(t *testing.T) {
	for _, rule := range []string{
		`ActivatedAbilityEffect(AbilityOfPermanent(HostPermanent),ReduceManaCostNotLessThanOne([CostReduceGeneric(-1)]))`,
		`ActivatedAbilityEffect(AbilityOfPermanent(ThisPermanent),ReduceManaCostNotLessThanOne([CostReduceGeneric(2)]))`,
		`ActivatedAbilityEffect(AbilityOfPermanent(HostPermanent()),ReduceManaCostNotLessThanOne([CostReduceGeneric(2)]))`,
		`ActivatedAbilityEffect(AbilityOfPermanent(Ref_TargetPermanent),ReduceManaCostNotLessThanOne([CostReduceGeneric(2)]))`,
		`ActivatedAbilityEffect(AbilityOfPermanent(HostPermanent),ReduceManaCostNotLessThanOne([CostReduceGeneric(2),CostReduceGeneric(1)]))`,
		`ActivatedAbilityEffect(AbilityOfPermanent(HostPermanent),ReduceManaCostNotLessThanOne([CostReduceGeneric(Integer(2))]))`,
		`ActivatedAbilityEffect(AbilityOfAPermanent(And([])),IncreaseManaCost([ManaCostGeneric(3)]))`,
		`ActivatedAbilityEffect(AbilityOfAPermanent(IsColor(White)),IncreaseManaCost([ManaCostX]))`,
		`ActivatedAbilityEffect(AbilityOfAPermanent(IsColor(White())),IncreaseManaCost([ManaCostGeneric(3)]))`,
		`ActivatedAbilityEffect(AbilityOfAPermanent(IsCardtype(Enchantment)),IncreaseManaCost([ManaCostGeneric(-1)]))`,
		`ActivatedAbilityEffect(AbilityOfAPermanent(AnyPermanent),IncreaseManaCost(ManaCostW))`,
		`ActivatedAbilityEffect(AbilityOfAPermanent(IsColor(White)),IncreaseManaCost([ManaCostGeneric(3)]),UntilEndOfTurn)`,
		`ActivatedAbilityEffect(AbilityOfPermanent(HostPermanent),IncreaseManaCost([ManaCostGeneric(3)]))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Artifact)),` + rule + `])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid modifier", rule)
		}
	}
	for _, typ := range []string{"Artifact", "Instant"} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[` + typ + `],Subtypes:[]),Rules:[ActivatedAbilityEffect(AbilityOfPermanent(HostPermanent),ReduceManaCostNotLessThanOne([CostReduceGeneric(2)]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("unbound host", typ)
		}
	}
}

func TestRONActivationCostRecordsAndInventory(t *testing.T) {
	entries, err := os.ReadDir("../../../../cards/mtgish")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	counts := map[string]int{}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".ron") {
			continue
		}
		data, err := os.ReadFile("../../../../cards/mtgish/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		var selected []string
		for line := range strings.SplitSeq(string(data), "\n") {
			if !strings.Contains(line, `Name: "Gloom"`) && !strings.Contains(line, `Name: "Power Artifact"`) {
				continue
			}
			nodes, err := Parse(strings.NewReader(line), entry.Name())
			if err != nil {
				t.Fatal(err)
			}
			name := nodes[0].Fields["Name"].String
			counts[name]++
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			first, second := factory(), factory()
			if first.ID() == second.ID() {
				t.Fatal("factory identity shared")
			}
			for _, card := range []mage.Card{first, second} {
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
			}
			selected = append(selected, line)
		}
		if len(selected) > 0 {
			files[entry.Name()] = &fstest.MapFile{Data: []byte(strings.Join(selected, "\n"))}
		}
	}
	if counts["Gloom"] != 6 || counts["Power Artifact"] != 1 {
		t.Fatal(counts)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 7 || len(report.Issues) != 0 {
		t.Fatal("complete records rejected")
	}
	for _, signature := range report.Signatures {
		if signature.Role == "unknown" {
			t.Fatalf("unknown descendant %+v", signature)
		}
	}
	old := mage.DefaultRegistry
	if registry, err := LoadFS(files, "*.ron"); err != nil || registry == nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnknownRule])`)}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial registry")
	}
	if mage.DefaultRegistry != old {
		t.Fatal("global registry changed")
	}
}

func TestRONActivationCostGloomBothRules(t *testing.T) {
	for _, opponent := range []bool{false, true} {
		g, a, b := ronGame()
		g.PutOnBattlefield(repositoryCard(t, "LEA", "Gloom"), a.PlayerID())
		caster := a
		if opponent {
			caster = b
		}
		spell := repositoryCard(t, "LEA", "Healing Salve")
		spell.SetOwner(caster.PlayerID())
		caster.AddToHand(spell)
		caster.ManaPool().Add(core.White, 1)
		if err := g.CastSpellByID(caster.PlayerID(), spell.ID(), []uuid.UUID{caster.PlayerID()}, 0); err == nil {
			t.Fatal("white spell tax")
		}
		caster.ManaPool().Add(core.Colorless, 3)
		if err := g.CastSpellByID(caster.PlayerID(), spell.ID(), []uuid.UUID{caster.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if caster.Life() != 23 || caster.ManaPool().TotalMana() != 0 {
			t.Fatal("white spell payment")
		}
		for _, tc := range []struct {
			typ, cost string
			want      int
		}{{"Enchantment", "{W}", 3}, {"Enchantment", "{U}", 0}, {"Artifact", "{W}", 0}} {
			var hostCard mage.Card
			option := mage.WithActivatedAbility(mage.GainLifeFor(mage.You, mage.Fixed(1)), mage.Tap())
			if tc.typ == "Enchantment" {
				hostCard = mage.NewEnchantment("Host", tc.cost, option)
			} else {
				hostCard = mage.NewArtifact("Host", tc.cost, option)
			}
			host := g.PutOnBattlefield(hostCard, caster.PlayerID())
			caster.ManaPool().Add(core.Colorless, tc.want)
			if err := g.ActivateAbilityByIndex(caster.PlayerID(), host.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if caster.ManaPool().TotalMana() != 0 {
				t.Fatal("nonmatching activation taxed")
			}
		}
	}
}

func TestRONActivationCostPowerArtifactManaAndFloor(t *testing.T) {
	g, a, _ := ronGame()
	monolith := g.PutOnBattlefield(repositoryCard(t, "LEA", "Basalt Monolith"), a.PlayerID())
	for range 2 {
		aura := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Power Artifact"), a.PlayerID())
		g.Attach(aura.ID(), monolith.ID())
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), monolith.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 3 || !g.FindPermanent(monolith.ID()).Tapped {
		t.Fatal("zero mana cost changed")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), monolith.ID(), 2, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.ManaPool().TotalMana() != 2 || g.FindPermanent(monolith.ID()).Tapped {
		t.Fatal("one mana minimum")
	}
}
