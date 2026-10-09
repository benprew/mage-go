package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextHostPTRecords(t *testing.T) {
	for _, tc := range []struct {
		name             string
		power, toughness int
	}{{"Goblin Caves", 0, 2}, {"Goblin Shrine", 1, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", tc.name)
			source := g.PutOnBattlefield(card, a.PlayerID())
			mountain := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), b.PlayerID())
			forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
			nonbasic := g.PutOnBattlefield(mage.NewLand("Nonbasic Forest", mage.WithSubTypes("Forest")), b.PlayerID())
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 3, mage.WithSubTypes("Goblin")), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 3, mage.WithSubTypes("Goblin")), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 3, mage.WithSubTypes("Elf")), a.PlayerID())
			noncreature := g.PutOnBattlefield(mage.NewArtifact("Goblin artifact", "", mage.WithSubTypes("Goblin")), b.PlayerID())
			check := func(game *mage.Game, enabled bool) {
				t.Helper()
				power, toughness := 2, 3
				if enabled {
					power += tc.power
					toughness += tc.toughness
				}
				for _, p := range []*mage.Permanent{first, second} {
					staticPTCheck(t, game, p, power, toughness)
				}
				staticPTCheck(t, game, other, 2, 3)
				staticPTCheck(t, game, noncreature, 0, 0)
			}
			g.Attach(source.ID(), mountain.ID())
			check(g, true)
			id := textStaticID(t, source)
			change, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.ForestLandType)
			next, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
			if err := g.RevisePrintedStaticPTAbility(source, id, change); err != nil {
				t.Fatal(err)
			}
			if textStaticID(t, g.FindPermanent(source.ID())) != id {
				t.Fatal("declaration identity changed")
			}
			check(g, false)
			g.Attach(source.ID(), forest.ID())
			check(g, true)
			clone := g.Clone()
			if err := clone.RevisePrintedStaticPTAbility(clone.FindPermanent(source.ID()), id, next); err != nil {
				t.Fatal(err)
			}
			check(clone, false)
			check(g, true)
			g.Attach(source.ID(), nonbasic.ID())
			check(g, false)
			g.Attach(source.ID(), forest.ID())
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			check(g, true)
			phased := g.PhaseOut(forest.ID())
			g.ApplyContinuousEffects()
			check(g, false)
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			check(g, true)
			phased = g.PhaseOut(source.ID())
			g.ApplyContinuousEffects()
			check(g, false)
			if err := g.RevisePrintedStaticPTAbility(source, id, next); err == nil {
				t.Fatal("phased declaration revised")
			}
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			check(g, true)
			remover := g.PutOnBattlefield(mage.NewArtifact("Remover", ""), a.PlayerID())
			removal := mage.RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
			removal.SetSourceID(remover.ID())
			g.AddContinuousEffect(removal)
			check(g, false)
			if err := g.RevisePrintedStaticPTAbility(source, id, next); err == nil {
				t.Fatal("removed declaration revised")
			}
			g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
			check(g, true)
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetSubtypeFamily(mage.This, mage.SubtypeLand, "Island")), forest.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.ApplyContinuousEffects()
			check(g, false)
			if err := g.RevisePrintedStaticPTAbility(g.FindPermanent(source.ID()), id, next); err != nil {
				t.Fatal(err)
			}
			check(g, true)
			late := g.PutOnBattlefield(mage.NewCreature("Late", "", 1, 4, mage.WithSubTypes("Goblin")), b.PlayerID())
			staticPTCheck(t, g, late, 1+tc.power, 4+tc.toughness)
			if err := mage.ApplyEffect(g, mage.CreateTokenCopies(mage.You, mage.Fixed(1), mage.DefineToken("Goblin", mage.TokenTypes(core.TypeCreature), mage.TokenSubtypes("Goblin"), mage.TokenPT(1, 4))), source.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.ApplyContinuousEffects()
			for _, p := range g.FilterBattlefield(mage.IsToken) {
				staticPTCheck(t, g, p, 1+tc.power, 4+tc.toughness)
			}
			g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			check(g, false)
			if err := g.RevisePrintedStaticPTAbility(source, id, change); err == nil {
				t.Fatal("departed source revised")
			}
			g.PutTriggersOnStack()
			g.ResolveStack()
			wantDamage := 0
			if tc.name == "Goblin Shrine" {
				wantDamage = 1
			}
			if g.FindPermanent(first.ID()).Damage != wantDamage || g.FindPermanent(second.ID()).Damage != wantDamage || g.FindPermanent(other.ID()).Damage != 0 {
				t.Fatal("separate departure rule changed")
			}
			returned := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(returned.ID(), mountain.ID())
			check(g, true)
			if err := g.RevisePrintedStaticPTAbility(source, id, next); err == nil {
				t.Fatal("old visit revised returned source")
			}
			if returned.Card.ManaCost().String() != repositoryCard(t, "DRK", tc.name).ManaCost().String() {
				t.Fatal("printed cost changed")
			}
		})
	}
}

func hostPTRule(condition, recipients, changes string) string {
	return `If(PermanentPassesFilter(HostPermanent,` + condition + `),[EachPermanentLayerEffect(` + recipients + `,[` + changes + `])])`
}

func hostPTFactory(t *testing.T, rule string) mage.CardFactory {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Rule",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Land)),`+rule+`])`), "host.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONTextHostPTBoundaries(t *testing.T) {
	condition := `And([IsSupertype(Basic),IsLandType(Mountain)])`
	recipients := `And([IsCreatureType(Goblin),IsCardtype(Creature)])`
	for _, rule := range []string{
		hostPTRule(`And([IsSupertype(Basic()),IsLandType(Mountain)])`, recipients, `AdjustPT(1,2)`),
		hostPTRule(condition, recipients, `AdjustPT(1)`),
		hostPTRule(condition, recipients, `AdjustPT(ValueX,2)`),
		hostPTRule(condition, recipients, `AdjustPT(1,2),AddAbility([Flying])`),
		strings.Replace(hostPTRule(condition, recipients, `AdjustPT(1,2)`), "HostPermanent", "Ref_TargetPermanent", 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid rule accepted", rule)
		}
	}
	for _, rule := range []string{
		hostPTRule(`IsLandType(Mountain)`, recipients, `AdjustPT(1,2)`),
		hostPTRule(`And([IsSupertype(Basic),IsLandType(Desert)])`, recipients, `AdjustPT(1,2)`),
		hostPTRule(condition, `IsCardtype(Creature)`, `AdjustPT(1,2)`),
		hostPTRule(condition, recipients, `AdjustPT(1,2),AdjustPT(2,1)`),
		strings.Replace(hostPTRule(condition, recipients, `AdjustPT(1,2)`), "If(", "Unless(", 1),
	} {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(hostPTFactory(t, rule)(), a.PlayerID())
		change, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.ForestLandType)
		if err := g.RevisePrintedStaticPTAbility(p, textStaticID(t, p), change); err == nil {
			t.Fatal("unmarked context revised", rule)
		}
	}
}

func TestRONTextHostPTIndependentDeclarations(t *testing.T) {
	rule := hostPTRule(`And([IsLandType(Mountain),IsSupertype(Basic)])`, `And([IsCardtype(Creature),IsCreatureType(Goblin)])`, `AdjustPT(1,2)`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(hostPTFactory(t, rule+","+rule)(), a.PlayerID())
	sibling := g.PutOnBattlefield(hostPTFactory(t, rule)(), b.PlayerID())
	forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	g.Attach(source.ID(), forest.ID())
	g.Attach(sibling.ID(), forest.ID())
	member := g.PutOnBattlefield(mage.NewCreature("Member", "", 1, 3, mage.WithSubTypes("Goblin")), a.PlayerID())
	change, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.ForestLandType)
	if err := g.RevisePrintedStaticPTAbility(source, textStaticID(t, sibling), change); err == nil {
		t.Fatal("foreign declaration revised")
	}
	if err := g.RevisePrintedStaticPTAbility(source, source.RuntimeAbilities[0].AbilityID(), change); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, member, 2, 5)
	if err := g.RevisePrintedStaticPTAbility(g.FindPermanent(source.ID()), source.RuntimeAbilities[1].AbilityID(), change); err != nil {
		t.Fatal(err)
	}
	staticPTCheck(t, g, member, 3, 7)
}

func TestRONTextHostPTVisitsAndLKI(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "DRK", "Goblin Caves"), a.PlayerID())
	card := repositoryCard(t, "LEA", "Forest")
	host := g.PutOnBattlefield(card, b.PlayerID())
	g.Attach(source.ID(), host.ID())
	memberCard := mage.NewCreature("Member", "", 2, 3, mage.WithSubTypes("Goblin"))
	member := g.PutOnBattlefield(memberCard, b.PlayerID())
	change, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.ForestLandType)
	if err := g.RevisePrintedStaticPTAbility(source, textStaticID(t, source), change); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(g.FindPermanent(member.ID()))
	if lki := g.LKI(member.ID()); lki == nil || lki.Power != 2 || lki.Toughness != 5 {
		t.Fatal("departure lost revised P/T")
	}
	member = g.PutOnBattlefield(memberCard, b.PlayerID())
	staticPTCheck(t, g, member, 2, 5)
	g.BouncePermanentToHand(g.FindPermanent(host.ID()))
	staticPTCheck(t, g, member, 2, 3)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	staticPTCheck(t, g, member, 2, 3)
	g.Attach(source.ID(), returned.ID())
	staticPTCheck(t, g, member, 2, 5)
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetSubtypeFamily(mage.This, mage.SubtypeCreature, "Elf")), member.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	staticPTCheck(t, g, member, 2, 3)
}
