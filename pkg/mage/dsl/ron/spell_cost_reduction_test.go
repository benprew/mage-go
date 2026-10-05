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

var spellCostReductionCards = []struct {
	set, name       string
	cost, reduction int
}{
	{"DRK", "Stone Calendar", 5, 1},
	{"LEG", "Mana Matrix", 6, 2},
	{"LEG", "Planar Gate", 6, 2},
}

func costReductionProbe(kind core.CardType, cost string) *mage.BaseCard {
	switch kind {
	case core.TypeCreature:
		return mage.NewCreature("Probe", cost, 1, 1)
	case core.TypeInstant:
		return mage.NewInstant("Probe", cost, mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1))))
	case core.TypeSorcery:
		return mage.NewSorcery("Probe", cost, mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1))))
	case core.TypeEnchantment:
		return mage.NewEnchantment("Probe", cost)
	default:
		return mage.NewArtifact("Probe", cost)
	}
}

func TestRONSpellCostReductionCards(t *testing.T) {
	for _, tc := range spellCostReductionCards {
		for _, kind := range []core.CardType{core.TypeCreature, core.TypeInstant, core.TypeSorcery, core.TypeEnchantment, core.TypeArtifact} {
			for _, opponent := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%v/%v", tc.name, kind, opponent), func(t *testing.T) {
					g, a, b := ronGame()
					source := repositoryCard(t, tc.set, tc.name)
					source.SetOwner(b.PlayerID())
					g.PutOnBattlefield(source, a.PlayerID())
					spell := costReductionProbe(kind, "{4}{U}")
					caster := a
					if opponent {
						caster = b
						g.SetActivePlayerIndex(1)
					}
					spell.SetOwner(caster.PlayerID())
					caster.AddToHand(spell)
					matches := tc.name == "Stone Calendar" || (tc.name == "Mana Matrix" && (kind == core.TypeInstant || kind == core.TypeEnchantment)) || (tc.name == "Planar Gate" && kind == core.TypeCreature)
					want := 0
					if matches && !opponent {
						want = tc.reduction
					}
					if got := g.ConditionalSpellCostReduction(caster.PlayerID(), spell); got != want {
						t.Fatalf("reduction = %d, want %d", got, want)
					}
					caster.ManaPool().Add(core.Blue, 1)
					caster.ManaPool().Add(core.Colorless, 4-want)
					if err := g.CastSpellByID(caster.PlayerID(), spell.ID(), nil, 0); err != nil {
						t.Fatal(err)
					}
					if caster.ManaPool().TotalMana() != 0 {
						t.Fatal("wrong mana payment")
					}
					g.ResolveStack()
				})
			}
		}
	}
}

func TestRONSpellCostReductionLifecycle(t *testing.T) {
	for _, tc := range spellCostReductionCards {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			probe := mage.NewCreature("Probe", "{5}{G}", 1, 1)
			if tc.name == "Mana Matrix" {
				probe = mage.NewEnchantment("Probe", "{5}{G}")
			}
			check := func(game *mage.Game, player uuid.UUID, want int) {
				t.Helper()
				if got := game.ConditionalSpellCostReduction(player, probe); got != want {
					t.Fatalf("reduction = %d, want %d", got, want)
				}
			}
			check(g, a.PlayerID(), tc.reduction)
			clone := g.Clone()
			g.PhaseOut(source.ID())
			check(g, a.PlayerID(), 0)
			check(clone, a.PlayerID(), tc.reduction)
			g.PhaseIn([]uuid.UUID{source.ID()})
			check(g, a.PlayerID(), tc.reduction)
			g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
			check(g, a.PlayerID(), 0)
			g.RunStepWithPriority(core.Cleanup)
			check(g, a.PlayerID(), tc.reduction)
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), source.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			check(g, a.PlayerID(), 0)
			check(g, b.PlayerID(), tc.reduction)
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			check(g, b.PlayerID(), 0)
			g.PutOnBattlefield(card, a.PlayerID())
			check(g, a.PlayerID(), tc.reduction)
			check(g, b.PlayerID(), 0)
			check(clone, a.PlayerID(), tc.reduction)
		})
	}
}

func TestRONSpellCostReductionCasting(t *testing.T) {
	for _, tc := range spellCostReductionCards {
		t.Run(tc.name, func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Colorless, tc.cost-1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("source cast with too little mana")
			}
			if a.ManaPool().TotalMana() != tc.cost-1 || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed source cast changed resources")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			probe := mage.NewCreature("Probe", "{5}", 1, 1)
			if tc.name == "Mana Matrix" {
				probe = mage.NewEnchantment("Probe", "{5}")
			}
			if g.ConditionalSpellCostReduction(a.PlayerID(), probe) != 0 {
				t.Fatal("source applies while on the stack")
			}
			g.ResolveStack()
			if a.ManaPool().TotalMana() != 0 || g.FindPermanent(card.ID()) == nil || g.ConditionalSpellCostReduction(a.PlayerID(), probe) != tc.reduction {
				t.Fatal("source casting or entry failed")
			}
		})
	}
}

func TestRONSpellCostReductionInvalid(t *testing.T) {
	for _, rule := range []string{
		`PlayerEffect(You,[])`, `PlayerEffect(You)`, `PlayerEffect(You(),[DecreaseSpellCost(AnySpell,[CostReduceGeneric(1)])])`,
		`PlayerEffect(Opponent,[DecreaseSpellCost(AnySpell,[CostReduceGeneric(1)])])`,
		`PlayerEffect(Ref_TargetPlayer,[DecreaseSpellCost(AnySpell,[CostReduceGeneric(1)])])`,
		`PlayerEffect(You,[DecreaseSpellCost(AnySpell,[])])`,
		`PlayerEffect(You,[DecreaseSpellCost(AnySpell,[CostReduceGeneric(-1)])])`,
		`PlayerEffect(You,[DecreaseSpellCost(AnySpell,[CostReduceGeneric(Integer(1))])])`,
		`PlayerEffect(You,[DecreaseSpellCost(AnySpell,[CostReduceGeneric(1,2)])])`,
		`PlayerEffect(You,[DecreaseSpellCost(AnySpell(),[CostReduceGeneric(1)])])`,
		`PlayerEffect(You,[DecreaseSpellCost(Or([]),[CostReduceGeneric(1)])])`,
		`PlayerEffect(You,[DecreaseSpellCost(And([IsCardtype(Creature),IsTapped]),[CostReduceGeneric(1)])])`,
		`PlayerEffect(You,[DecreaseSpellCost(IsCardtype(NoType),[CostReduceGeneric(1)])])`,
		`PlayerEffect(You,[DecreaseSpellCost(IsCardtype(Land),[CostReduceGeneric(1)])])`,
		`PlayerEffect(You,[DecreaseSpellCost(AnySpell,[CostReduceGeneric(1),CostReduceGeneric(2)])])`,
		`PlayerEffect(You,[DecreaseSpellCost(AnySpell,[CostReduceGeneric(1)]),HasNoMaximumHandSize])`,
		`SpellActions(ActionList([PlayerEffect(You,[DecreaseSpellCost(AnySpell,[CostReduceGeneric(1)])])]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONSpellCostReductionInventory(t *testing.T) {
	src := `Card(Name:"Reducer",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[PlayerEffect(You,[DecreaseSpellCost(Or([IsCardtype(Instant),IsCardtype(Enchantment)]),[CostReduceGeneric(2)])])])`
	files := fstest.MapFS{"a.ron": {Data: []byte(src)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{"You": "player_reference", "DecreaseSpellCost": "player_rule_operation", "Or": "cost_spell_filter", "IsCardtype": "cost_spell_filter", "CostReduceGeneric": "spell_cost_reduction"}
	for name, role := range roles {
		found := false
		for _, s := range report.Signatures {
			if s.Name == name {
				found = true
				if s.Role != role {
					t.Fatalf("%s role = %s, want %s", name, s.Role, role)
				}
			}
		}
		if !found {
			t.Fatalf("missing %s", name)
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || registry == nil {
		t.Fatalf("valid collection: %v", err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(src, "CostReduceGeneric(2)", "CostReduceGeneric(-1)"))}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("invalid collection published a partial registry")
	}
}

func TestRONSpellCostReductionTotals(t *testing.T) {
	for _, tc := range spellCostReductionCards {
		for _, component := range []string{"printed", "X", "additional", "increase"} {
			for _, underpaid := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/%v", tc.name, component, underpaid), func(t *testing.T) {
					g, a, _ := ronGame()
					g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
					kind := core.TypeCreature
					if tc.name == "Mana Matrix" {
						kind = core.TypeInstant
					}
					cost, x := "{3}{U}", 0
					if component == "X" {
						cost, x = "{X}{U}", 3
					}
					if component == "additional" || component == "increase" {
						cost = "{U}"
					}
					spell := costReductionProbe(kind, cost)
					if component == "additional" {
						mage.WithAdditionalCost(mage.GenericCost(3))(spell)
					}
					if component == "increase" {
						g.PutOnBattlefield(mage.NewEnchantment("Tax", "", mage.WithStaticAbility(mage.IncreaseSpellCostForColor(core.Blue, 3))), a.PlayerID())
					}
					spell.SetOwner(a.PlayerID())
					a.AddToHand(spell)
					mana := 3 - tc.reduction
					if underpaid {
						mana--
					}
					a.ManaPool().Add(core.Colorless, mana)
					a.ManaPool().Add(core.Blue, 1)
					err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, x)
					if (err != nil) != underpaid {
						t.Fatalf("cast = %v, underpaid = %v", err, underpaid)
					}
					if underpaid {
						if len(a.Hand()) != 1 || g.StackSize() != 0 || a.ManaPool().TotalMana() != mana+1 {
							t.Fatal("failed total payment changed resources")
						}
						return
					}
					if a.ManaPool().TotalMana() != 0 || g.StackPeek().XValue != x {
						t.Fatal("wrong total payment or announced X")
					}
					g.ResolveStack()
				})
			}
		}
	}
}

func TestRONSpellCostReductionStackingAndColoredCosts(t *testing.T) {
	g, a, b := ronGame()
	for _, tc := range spellCostReductionCards {
		g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
	}
	g.PutOnBattlefield(repositoryCard(t, "LEG", "Mana Matrix"), b.PlayerID())
	probe := mage.NewCreature("Multi-type", "{8}{G}", 1, 1, mage.WithCardType(core.TypeEnchantment), mage.WithCardType(core.TypeArtifact))
	if got := g.ConditionalSpellCostReduction(a.PlayerID(), probe); got != 5 {
		t.Fatalf("stacked reduction = %d, want 5", got)
	}
	if got := g.ConditionalSpellCostReduction(b.PlayerID(), probe); got != 2 {
		t.Fatalf("opponent reduction = %d, want 2", got)
	}
	probe.SetOwner(a.PlayerID())
	a.AddToHand(probe)
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.CastSpellByID(a.PlayerID(), probe.ID(), nil, 0); err == nil {
		t.Fatal("generic reduction paid colored mana")
	}
	if a.ManaPool().TotalMana() != 3 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("colored cost failure changed resources")
	}
	a.ManaPool().Add(core.Green, 1)
	if err := g.CastSpellByID(a.PlayerID(), probe.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 0 {
		t.Fatal("wrong stacked payment")
	}
	g.ResolveStack()
	free := mage.NewCreature("Zero generic", "{G}", 1, 1)
	free.SetOwner(a.PlayerID())
	a.AddToHand(free)
	a.ManaPool().Add(core.Green, 1)
	if err := g.CastSpellByID(a.PlayerID(), free.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().TotalMana() != 0 {
		t.Fatal("reduction went below zero")
	}
}

func TestRONSpellCostReductionShapes(t *testing.T) {
	for _, amount := range []int{0, 2} {
		for _, filter := range []string{"AnySpell", "And([IsCardtype(Creature),IsCardtype(Artifact)])", "Or([IsCardtype(Creature),IsCardtype(Artifact)])", "And([AnySpell,Or([IsCardtype(Creature),IsCardtype(Enchantment)])])"} {
			t.Run(fmt.Sprintf("%d/%s", amount, filter), func(t *testing.T) {
				src := fmt.Sprintf(`Card(Name:"Reducer",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[PlayerEffect(You,[DecreaseSpellCost(%s,[CostReduceGeneric(%d)])])])`, filter, amount)
				nodes, err := Parse(strings.NewReader(src), "shapes.ron")
				if err != nil {
					t.Fatal(err)
				}
				factory, err := Compile(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				first, second := factory(), factory()
				if first.ID() == second.ID() || first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
					t.Fatal("factory shared source or ability identity")
				}
				g.PutOnBattlefield(first, a.PlayerID())
				g.PutOnBattlefield(second, b.PlayerID())
				probe := mage.NewCreature("Artifact creature", "{5}", 1, 1, mage.WithCardType(core.TypeArtifact))
				for _, player := range []mage.Player{a, b} {
					if got := g.ConditionalSpellCostReduction(player.PlayerID(), probe); got != amount {
						t.Fatalf("reduction = %d, want %d", got, amount)
					}
				}
			})
		}
	}
}

func TestRONSpellCostReductionStrictRecords(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"DRK", "LEG"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), set+".ron")
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.SplitAfter(string(data), "\n")
		offset := func(pos Position) int {
			n := pos.Column - 1
			for _, line := range lines[:pos.Line-1] {
				n += len(line)
			}
			return n
		}
		for i, node := range nodes {
			for _, tc := range spellCostReductionCards {
				if node.Fields["Name"].String != tc.name {
					continue
				}
				end := len(data)
				if i+1 < len(nodes) {
					end = offset(nodes[i+1].Position)
				}
				files[tc.name+".ron"] = &fstest.MapFile{Data: data[offset(node.Position):end]}
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 3 {
		t.Fatal("missing complete records")
	}
	g, a, _ := ronGame()
	for range 2 {
		card, err := registry.CreateCard("Mana Matrix")
		if err != nil {
			t.Fatal(err)
		}
		g.PutOnBattlefield(card, a.PlayerID())
	}
	probe := mage.NewInstant("Multi-type", "{5}{U}", mage.WithCardType(core.TypeEnchantment))
	for range 3 {
		g.ApplyContinuousEffects()
		if got := g.ConditionalSpellCostReduction(a.PlayerID(), probe); got != 4 {
			t.Fatalf("repeated layer reduction = %d, want 4", got)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Rules:[Unknown])`)}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("published a partial registry")
	}
}

func TestRONSpellCostReductionLockedBeforeSourcePayment(t *testing.T) {
	g, a, _ := ronGame()
	reducer := g.PutOnBattlefield(repositoryCard(t, "LEG", "Mana Matrix"), a.PlayerID())
	spell := mage.NewInstant("Pay source", "{U}", mage.Spell(mage.GainLifeFor(mage.You, mage.Fixed(1))),
		mage.WithAdditionalCost(mage.GenericCost(2)),
		mage.WithAdditionalCost(mage.SacrificeNMatchingCost(1, mage.IsArtifact, "Sacrifice an artifact")))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(reducer.ID()) != nil || a.ManaPool().TotalMana() != 0 || g.StackSize() != 1 {
		t.Fatal("cost was not locked before source sacrifice")
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("paid spell did not resolve")
	}
}
