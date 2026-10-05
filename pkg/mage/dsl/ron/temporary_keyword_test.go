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

func TestRONTemporaryKeywordRealCards(t *testing.T) {
	for _, tc := range []struct {
		name, set string
		index     int
		removed   []core.Attr
	}{
		{"Radjan Spirit", "LEG", 0, []core.Attr{core.Flying}},
		{"Hammerheim", "LEG", 1, []core.Attr{core.Plainswalk, core.Islandwalk, core.Swampwalk, core.Mountainwalk, core.Forestwalk, core.Desertwalk, core.LegendaryLandwalk}},
		{"Scarwood Hag", "DRK", 1, []core.Attr{core.Forestwalk}},
	} {
		for _, change := range []string{"none", "returned", "phased", "phase-in", "installed-return", "installed-phase", "control", "shroud"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				opts := []mage.CardOption{mage.WithKeyword(core.Vigilance), mage.WithKeyword(core.Banding)}
				for _, kw := range tc.removed {
					opts = append(opts, mage.WithKeyword(kw))
				}
				if tc.name == "Scarwood Hag" {
					opts = append(opts, mage.WithKeyword(core.Islandwalk))
				}
				fixture := mage.NewCreature("Recipient", "", 2, 2, opts...)
				fixture.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(fixture, b.PlayerID())
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 2, opts...), b.PlayerID())
				land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
				if tc.name != "Hammerheim" {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), tc.index, []uuid.UUID{target.ID()}); err == nil {
						t.Fatal("summoning sickness did not restrict tap cost")
					}
				} else {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if a.ManaPool().Count(core.Red) != 1 || !g.Stack().IsEmpty() {
						t.Fatal("land mana ability did not resolve immediately")
					}
					g.FindPermanent(source.ID()).Tapped = false
				}
				g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), tc.index, []uuid.UUID{land.ID()}); err == nil {
					t.Fatal("noncreature accepted")
				}
				if g.FindPermanent(source.ID()).Tapped || !g.Stack().IsEmpty() {
					t.Fatal("illegal announcement paid tap cost")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), tc.index, []uuid.UUID{target.ID()}); err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(target.ID()).HasKeyword(tc.removed[0]) {
					t.Fatal("removal started before resolution")
				}
				clone := g.Clone()
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("ability copy failed")
				}
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				current := target.ID()
				returnTarget := func() {
					g.BouncePermanentToHand(g.FindPermanent(current))
					b.RemoveFromHand(fixture.ID())
					current = g.PutOnBattlefield(fixture, b.PlayerID()).ID()
				}
				switch change {
				case "returned":
					returnTarget()
				case "phased":
					g.PhaseOut(current)
				case "phase-in":
					g.PhaseIn(g.PhaseOut(current))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: current, TargetID: current, ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "shroud":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Shroud)), current, b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				g.ResolveStack()
				if change == "installed-return" {
					returnTarget()
				}
				if change == "installed-phase" {
					g.PhaseIn(g.PhaseOut(current))
				}
				g.PhaseIn([]uuid.UUID{current})
				removed := change == "none" || change == "phase-in" || change == "installed-phase" || change == "control"
				for _, kw := range tc.removed {
					if g.FindPermanent(current).HasKeyword(kw) == removed {
						t.Fatalf("wrong removal for %s", kw.String())
					}
					if !g.FindPermanent(other.ID()).HasKeyword(kw) || !clone.FindPermanent(target.ID()).HasKeyword(kw) {
						t.Fatal("shared target or clone state")
					}
				}
				if !g.FindPermanent(current).HasKeyword(core.Vigilance) || !g.FindPermanent(current).HasKeyword(core.Banding) {
					t.Fatal("unrelated keyword removed")
				}
				if tc.name == "Scarwood Hag" && !g.FindPermanent(current).HasKeyword(core.Islandwalk) {
					t.Fatal("other landwalk removed")
				}
				clone.ResolveStack()
				if clone.FindPermanent(target.ID()).HasKeyword(tc.removed[0]) {
					t.Fatal("clone lost pending removal")
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				if !g.FindPermanent(current).HasKeyword(tc.removed[0]) || clone.FindPermanent(target.ID()).HasKeyword(tc.removed[0]) {
					t.Fatal("cleanup did not isolate games")
				}
			})
		}
	}
}

func TestRONTemporaryKeywordScarwoodOrder(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "DRK", "Scarwood Hag")
	source := g.PutOnBattlefield(card, a.PlayerID())
	g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
	target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), a.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
	activate := func(index int) {
		t.Helper()
		g.FindPermanent(source.ID()).Tapped = false
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, []uuid.UUID{target.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
		t.Fatal("unpaid grant accepted")
	}
	if g.FindPermanent(source.ID()).Tapped || !g.Stack().IsEmpty() {
		t.Fatal("failed grant paid tap cost")
	}
	a.ManaPool().Add(core.Green, 12)
	activate(0)
	activate(0)
	if !mage.HasLandwalkEvasion(g.FindPermanent(target.ID()), b.PlayerID(), g) {
		t.Fatal("grant has no combat effect")
	}
	activate(1)
	if g.FindPermanent(target.ID()).HasKeyword(core.Forestwalk) || mage.HasLandwalkEvasion(g.FindPermanent(target.ID()), b.PlayerID(), g) {
		t.Fatal("removal did not erase all earlier grants")
	}
	activate(0)
	if !g.FindPermanent(target.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("later grant did not win")
	}
	activate(1)
	if g.FindPermanent(target.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("later removal did not win")
	}
	if a.ManaPool().TotalMana() != 0 {
		t.Fatal("grant mana cost not paid")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if g.FindPermanent(target.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("grant survived cleanup")
	}
}

func TestRONTemporaryKeywordInvalid(t *testing.T) {
	for _, modification := range []string{
		`LosesAbility()`, `LosesAbility(Flying, Banding)`, `LosesAbility(Flying())`,
		`LosesAbility(TheChosenAbility)`, `LosesAbility(AnyBandsWithOthers)`,
		`LosesAbility(AnyLandwalk())`, `LosesAbility(Landwalk(IsLandType(Forest())))`,
		`LosesAbility(Landwalk(IsLandType(Snow)))`, `LosesAbility(Landwalk(AnyPermanent))`,
		`LosesAbility(Enchant(IsCardtype(Creature)))`, `AddAbility([AnyLandwalk])`,
		`AddAbility([Landwalk(IsLandType(Forest()))])`,
	} {
		t.Run(modification, func(t *testing.T) {
			src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), ActionList([CreatePermanentLayerEffectUntil(ThisPermanent, [%s], UntilEndOfTurn)]))])`, modification)
			nodes, err := Parse(strings.NewReader(src), "invalid-keyword.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid keyword accepted")
			}
		})
	}
	for _, body := range []string{
		`Activated(PayMana([]), ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [LosesAbility(Flying)], UntilEndOfTurn)]))`,
		`Activated(PayMana([]), Targeted([TargetPlayer(AnyPlayer)], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [LosesAbility(Flying)], UntilEndOfTurn)])))`,
		`Activated(PayMana([]), ActionList([CreatePermanentLayerEffectUntil(HostPermanent, [LosesAbility(Flying)], UntilEndOfTurn)]))`,
		`Activated(PayMana([]), ActionList([CreatePermanentLayerEffectUntil(ThisPermanent, [LosesAbility(Flying)], UntilEndOfCombat())]))`,
		`EachPermanentLayerEffect(IsCardtype(Creature), [LosesAbility(Flying())])`,
	} {
		src := fmt.Sprintf(`Card(Name: "Invalid scope", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [%s])`, body)
		nodes, err := Parse(strings.NewReader(src), "invalid-keyword-scope.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("invalid context accepted: %s", body)
		}
	}
}

func TestRONTemporaryKeywordSourceLandwalk(t *testing.T) {
	for _, tc := range []struct {
		name, set, land     string
		index, cost, damage int
		color               core.Color
		keyword             core.Attr
	}{
		{"River Merfolk", "FEM", "Mountain", 0, 1, 0, core.Blue, core.Mountainwalk},
		{"Wormwood Treefolk", "DRK", "Forest", 0, 2, 2, core.Green, core.Forestwalk},
		{"Wormwood Treefolk", "DRK", "Swamp", 1, 2, 2, core.Black, core.Swampwalk},
	} {
		for _, change := range []string{"none", "returned", "phased", "phase-in", "control", "prevented"} {
			t.Run(tc.name+"/"+tc.land+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				g.PutOnBattlefield(repositoryCard(t, "LEB", tc.land), b.PlayerID())
				a.ManaPool().Add(core.Colorless, tc.cost)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), tc.index, nil); err == nil {
					t.Fatal("wrong color accepted")
				}
				if a.ManaPool().TotalMana() != tc.cost || !g.Stack().IsEmpty() {
					t.Fatal("failed payment changed state")
				}
				a.ManaPool().Clear()
				a.ManaPool().Add(tc.color, tc.cost)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), tc.index, nil); err != nil {
					t.Fatal(err)
				}
				if a.Life() != 20 || source.HasKeyword(tc.keyword) || a.ManaPool().TotalMana() != 0 {
					t.Fatal("announcement timing or payment")
				}
				clone := g.Clone()
				current := source.ID()
				switch change {
				case "returned":
					g.BouncePermanentToHand(g.FindPermanent(current))
					a.RemoveFromHand(card.ID())
					current = g.PutOnBattlefield(card, a.PlayerID()).ID()
				case "phased":
					g.PhaseOut(current)
				case "phase-in":
					g.PhaseIn(g.PhaseOut(current))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: current, TargetID: current, ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "prevented":
					g.AddPreventionShield(a.PlayerID(), 2)
				}
				g.ResolveStack()
				g.PhaseIn([]uuid.UUID{current})
				granted := change != "returned" && change != "phased"
				if g.FindPermanent(current).HasKeyword(tc.keyword) != granted || mage.HasLandwalkEvasion(g.FindPermanent(current), b.PlayerID(), g) != granted {
					t.Fatal("source incarnation or evasion")
				}
				wantDamage := tc.damage
				if change == "prevented" {
					wantDamage = 0
				}
				if a.Life() != 20-wantDamage || b.Life() != 20 {
					t.Fatal("damage did not keep ability controller or source LKI")
				}
				if clone.FindPermanent(source.ID()).HasKeyword(tc.keyword) || clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("clone changed before resolution")
				}
				clone.ResolveStack()
				if !clone.FindPermanent(source.ID()).HasKeyword(tc.keyword) || clone.GetPlayer(a.PlayerID()).Life() != 20-tc.damage {
					t.Fatal("clone lost pending context")
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				if g.FindPermanent(current).HasKeyword(tc.keyword) || !clone.FindPermanent(source.ID()).HasKeyword(tc.keyword) {
					t.Fatal("grant cleanup isolation")
				}
			})
		}
	}
}

func TestRONTemporaryKeywordCopyRetarget(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [LosesAbility(Flying)], UntilEndOfTurn)])))`)
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 2, mage.WithKeyword(core.Flying)), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 2, mage.WithKeyword(core.Flying)), b.PlayerID())
	spell := factory()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{first.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	phased := g.PhaseOut(first.ID())
	copied := g.CopySpellOnStack(spell.ID(), b.PlayerID(), true)
	if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != second.ID() {
		t.Fatal("copy did not choose independent target")
	}
	g.PhaseIn(phased)
	g.ResolveTopOfStack()
	if !g.FindPermanent(first.ID()).HasKeyword(core.Flying) || g.FindPermanent(second.ID()).HasKeyword(core.Flying) {
		t.Fatal("copy used original target")
	}
	g.ResolveStack()
	if g.FindPermanent(first.ID()).HasKeyword(core.Flying) {
		t.Fatal("original target binding lost")
	}
}

func TestRONTemporaryKeywordInstancesAndInventory(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [LosesAbility(Flying)], UntilEndOfTurn)])))`)
	first, second := factory(), factory()
	if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("factory shared target")
	}
	if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
		t.Fatal(err)
	}
	src := `Card(Name: "Inventory", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), ActionList([CreatePermanentLayerEffectUntil(ThisPermanent, [LosesAbility(Landwalk(IsLandType(Forest)))], UntilEndOfTurn)]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{"LosesAbility": "temporary_modification", "Landwalk": "keyword", "IsLandType": "landwalk_filter", "Forest": "land_subtype"}
	for name, role := range roles {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing role %s/%s", name, role)
		}
	}
}
