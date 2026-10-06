package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONStaticValueAspect(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 3), b.PlayerID())
			card := repositoryCard(t, set, "Aspect of Wolf")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("payment rollback")
			}
			a.ManaPool().Add(core.Green, 1)
			a.ManaPool().Add(core.Colorless, 1)
			land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{land.ID()}, 0); err == nil {
				t.Fatal("land target accepted")
			}
			if a.ManaPool().TotalMana() != 2 || len(a.Hand()) != 1 {
				t.Fatal("target rollback")
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			checkConditionalPT(t, g, host.ID(), 2, 3)
			g.ResolveStack()
			checkConditionalPT(t, g, host.ID(), 2, 4)
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Swamp"), a.PlayerID())
			checkConditionalPT(t, g, host.ID(), 2, 4)
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Tropical Island"), a.PlayerID())
			checkConditionalPT(t, g, host.ID(), 3, 4)
			third := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			checkConditionalPT(t, g, host.ID(), 3, 5)
			clone := g.Clone()
			phased := g.PhaseOut(third.ID())
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, host.ID(), 3, 4)
			checkConditionalPT(t, clone, host.ID(), 3, 5)
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, host.ID(), 3, 5)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			checkConditionalPT(t, g, host.ID(), 2, 4)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: card.ID(), TargetID: land.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			checkConditionalPT(t, g, host.ID(), 3, 4)
			phased = g.PhaseOut(card.ID())
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, host.ID(), 2, 3)
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, host.ID(), 3, 4)
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 4, 5), a.PlayerID())
			g.Attach(card.ID(), other.ID())
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, host.ID(), 2, 3)
			checkConditionalPT(t, g, other.ID(), 5, 6)
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), card.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, other.ID(), 4, 5)
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, other.ID(), 5, 6)
			g.BouncePermanentToHand(g.FindPermanent(card.ID()))
			checkConditionalPT(t, g, other.ID(), 4, 5)
		})
	}
}

func TestRONStaticValueWombat(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Rabid Wombat")
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Green, 1)
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid wombat")
	}
	if a.ManaPool().TotalMana() != 3 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("payment rollback")
	}
	a.ManaPool().Add(core.Green, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil {
		t.Fatal("early entry")
	}
	g.ResolveStack()
	checkConditionalPT(t, g, card.ID(), 0, 1)
	if !g.FindPermanent(card.ID()).HasKeyword(core.Vigilance) {
		t.Fatal("missing vigilance")
	}
	otherCard := repositoryCard(t, "LEG", "Rabid Wombat")
	other := g.PutOnBattlefield(otherCard, b.PlayerID())
	aura := g.PutOnBattlefield(repositoryCard(t, "LEB", "Regeneration"), b.PlayerID())
	g.Attach(aura.ID(), card.ID())
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 2, 3)
	checkConditionalPT(t, g, other.ID(), 0, 1)
	equipment := g.PutOnBattlefield(mage.NewArtifact("Equipment", "", mage.WithSubTypes("Equipment")), a.PlayerID())
	g.Attach(equipment.ID(), card.ID())
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 2, 3)
	second := g.PutOnBattlefield(repositoryCard(t, "LEB", "Regeneration"), a.PlayerID())
	g.Attach(second.ID(), card.ID())
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 4, 5)
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(aura.ID()))
	clone.ApplyContinuousEffects()
	checkConditionalPT(t, clone, card.ID(), 2, 3)
	checkConditionalPT(t, g, card.ID(), 4, 5)
	phased := g.PhaseOut(aura.ID())
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 2, 3)
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 4, 5)
	g.Attach(aura.ID(), other.ID())
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 2, 3)
	checkConditionalPT(t, g, other.ID(), 2, 3)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	checkConditionalPT(t, g, card.ID(), 2, 3)
	phased = g.PhaseOut(card.ID())
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 2, 3)
	g.MutablePermanent(card.ID()).AddCounter(core.P1P1, 1)
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 3, 4)
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetBasePowerToughness(mage.This, mage.Fixed(5), mage.Fixed(7)), mage.SwitchPowerToughness(mage.This)), card.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 10, 8)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 3, 4)
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), card.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 1, 2)
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 3, 4)
	g.MutablePermanent(card.ID()).RevokeBaseAttr(core.AttrSummonSick)
	g.SetStep(core.DeclareAttackers)
	g.SetActivePlayerIndex(1)
	g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{card.ID()})
	if !g.GetCombat().IsAttacking(card.ID()) || g.FindPermanent(card.ID()).Tapped {
		t.Fatal("vigilance attack")
	}
	g.SetStep(core.CombatDamage)
	g.ExecuteCombatDamage()
	if a.Life() != 17 {
		t.Fatal("combat did not use current bonus")
	}
	g.BouncePermanentToHand(g.FindPermanent(card.ID()))
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, b.PlayerID())
	g.ApplyContinuousEffects()
	checkConditionalPT(t, g, card.ID(), 0, 1)
}

func TestRONStaticValueInvalid(t *testing.T) {
	count := `TheNumberOfPermanentsOnTheBattlefield(And([IsEnchantmentType(Aura),AttachedToPermanent(ThisPermanent)]))`
	for i, rule := range []string{
		`PermanentLayerEffect(Ref_TargetPermanent,[AdjustPTForEach(2,2,` + count + `)])`,
		`PermanentLayerEffect(HostPermanent,[AdjustPTForEach(2,2,` + count + `)])`,
		`PermanentLayerEffect(ThisPermanent,[AdjustPTForEach(2,2,ValueX)])`,
		`PermanentLayerEffect(ThisPermanent,[AdjustPTForEach(2,2,PowerOfPermanent(ThisPermanent))])`,
		`PermanentLayerEffect(ThisPermanent,[AdjustPTForEach(2,2,` + count + `,Integer(1))])`,
		`PermanentLayerEffect(ThisPermanent,[AdjustPTForEach(Integer(2),2,` + count + `)])`,
		`PermanentLayerEffect(ThisPermanent,[AdjustPTXY(PlusX,PlusX,Integer(1),Integer(2))])`,
		`PermanentLayerEffect(ThisPermanent,[AdjustPTXY(PlusX,PlusY,Integer(1))])`,
		`PermanentLayerEffect(ThisPermanent,[AdjustPTForEach(2,2,TheNumberOfPermanentsOnTheBattlefield(AttachedToPermanent(ThisPermanent)))])`,
		`PermanentLayerEffect(ThisPermanent,[AdjustPTForEach(2,2,TheNumberOfPermanentsOnTheBattlefield(And([IsEnchantmentType(Aura),AttachedToPermanent(Ref_TargetPermanent)])))])`,
		`PermanentLayerEffect(ThisPermanent,[AdjustPTForEach(2,2,TheNumberOfPermanentsOnTheBattlefield(And([IsEnchantmentType(Aura),AttachedToPermanent(ThisPermanent),IsColor(Green)])))])`,
		`PermanentLayerEffect(ThisPermanent,[AdjustPTForEach(2,2,TheNumberOfPermanentsOnTheBattlefield(And([])))])`,
		`PermanentLayerEffect(ThisPermanent,[AdjustPTForEach(2,2,` + count + `),AddAbility([Flying])])`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[AdjustPTForEach(2,2,` + count + `)])`,
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[`+rule+`])`), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Compile(nodes[0]); err == nil {
				t.Fatal("accepted invalid static value")
			}
		})
	}
}

func TestRONStaticValueAspectResolutionAndCopies(t *testing.T) {
	for _, change := range []string{"shroud", "departed", "returned", "phased", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			hostCard := mage.NewCreature("Host", "", 2, 3)
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 4, 5), a.PlayerID())
			for range 3 {
				g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			}
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
			card := repositoryCard(t, "LEB", "Aspect of Wolf")
			castRONSpell(t, g, a, card, host.ID())
			switch change {
			case "shroud":
				g.MutablePermanent(host.ID()).GrantBaseAttr(core.Shroud)
			case "departed", "returned":
				g.BouncePermanentToHand(g.FindPermanent(host.ID()))
				if change == "returned" {
					b.RemoveFromHand(hostCard.ID())
					g.PutOnBattlefield(hostCard, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(host.ID())
			case "copy":
				phased := g.PhaseOut(host.ID())
				copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), true)
				if copied == nil {
					t.Fatal("missing copy")
				}
				if len(copied.Targets) != 1 || copied.Targets[0] != other.ID() {
					t.Fatal("copy target")
				}
				g.PhaseIn(phased)
			}
			clone := g.Clone()
			clone.ResolveStack()
			g.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if change == "copy" {
					checkConditionalPT(t, game, host.ID(), 3, 5)
					checkConditionalPT(t, game, other.ID(), 4, 6)
				} else {
					if game.FindPermanent(card.ID()) != nil {
						t.Fatal("illegal Aura resolved")
					}
					if change == "returned" || change == "shroud" {
						checkConditionalPT(t, game, host.ID(), 2, 3)
					}
				}
			}
		})
	}
}

func TestRONStaticValueSignedAndLiveCounts(t *testing.T) {
	for _, tc := range []struct {
		value            string
		power, toughness int
	}{
		{`AdjustPTXY(PlusX,PlusY,HalfRoundedDown(Integer(-3)),HalfRoundedUp(Integer(-3)))`, 8, 9},
		{`AdjustPTForEach(-2,3,Twice(TheNumberOfPermanentsOnTheBattlefield(And([IsLandType(Forest),ControlledByAPlayer(SinglePlayer(You))]))))`, 6, 16},
	} {
		t.Run(tc.value, func(t *testing.T) {
			g, a, b := ronGame()
			nodes, err := Parse(strings.NewReader(`Card(Name:"Numbers",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:10,Toughness:10),Rules:[PermanentLayerEffect(ThisPermanent,[`+tc.value+`])])`), "numbers.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			first := factory()
			second := factory()
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			p := g.PutOnBattlefield(first, a.PlayerID())
			q := g.PutOnBattlefield(second, b.PlayerID())
			checkConditionalPT(t, g, p.ID(), tc.power, tc.toughness)
			if strings.Contains(tc.value, "Integer(-3)") {
				checkConditionalPT(t, g, q.ID(), 8, 9)
			} else {
				checkConditionalPT(t, g, q.ID(), 10, 10)
			}
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			if !strings.Contains(tc.value, "Integer(-3)") {
				checkConditionalPT(t, g, p.ID(), 2, 22)
			}
		})
	}
}

func TestRONStaticValueInventoryAndStrictLoad(t *testing.T) {
	source := `Card(Name:"Live",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:0,Toughness:1),Rules:[PermanentLayerEffect(ThisPermanent,[AdjustPTForEach(2,2,TheNumberOfPermanentsOnTheBattlefield(And([IsEnchantmentType(Aura),AttachedToPermanent(ThisPermanent)])))])])`
	files := fstest.MapFS{"a.ron": {Data: []byte(source)}, "reprint.ron": {Data: []byte(source)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 2 || report.UniqueCards != 1 || len(report.Issues) != 0 {
		t.Fatal("valid records rejected")
	}
	found := false
	for _, entry := range report.Signatures {
		if entry.Role == "unknown" {
			t.Fatalf("unknown descendant: %s", entry.Name)
		}
		if entry.Name == "TheNumberOfPermanentsOnTheBattlefield" && entry.Role == "static_value" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing live static value role")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Live")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Live")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID() == second.ID() {
		t.Fatal("shared factory identity")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(source, "Name:\"Live\"", "Name:\"Invalid\"", 1) + "\nNotACard")}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("partial collection accepted")
	}
	if names := registry.RegisteredCardNames(); len(names) != 1 || names[0] != "Live" {
		t.Fatal("failed load changed registry")
	}
}
