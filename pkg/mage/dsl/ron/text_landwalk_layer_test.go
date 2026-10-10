package ron

import (
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextLandwalkLayerOrderingAndExpiry(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Bog Wraith")
	p := g.PutOnBattlefieldAttacking(card, a.PlayerID(), b.PlayerID())
	id := landwalkID(t, p)
	first, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	second, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	if err := g.AddPrintedBasicLandwalkTextEffect(p, id, core.EndOfTurn, first); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedBasicLandwalkTextEffect(g.FindPermanent(p.ID()), id, core.Indefinite, second); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
	}
	if !g.FindPermanent(p.ID()).HasKeyword(core.Forestwalk) || g.FindPermanent(p.ID()).HasKeyword(core.Swampwalk) {
		t.Fatal("ordered text effects failed")
	}
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	blocker := g.PutOnBattlefield(repositoryCard(t, "LEA", "Grizzly Bears"), b.PlayerID())
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: p.ID()}})
	if g.IsBlockingInCombat(blocker.ID()) {
		t.Fatal("combat ignored layer-3 landwalk")
	}
	clone := g.Clone()
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if !clone.FindPermanent(p.ID()).HasKeyword(core.Swampwalk) || clone.FindPermanent(p.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("expiry did not recompute from baseline")
	}
	if !g.FindPermanent(p.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("clone expiry changed original")
	}
	if card.AttrSeeds()[core.Swampwalk] != 1 || card.AttrSeeds()[core.Forestwalk] != 0 {
		t.Fatal("text changed copy information")
	}
}

func TestRONTextLandwalkLayerRemovalAndPhasing(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "LEA", "Bog Wraith"), a.PlayerID())
	id := landwalkID(t, p)
	first, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.AddPrintedBasicLandwalkTextEffect(p, id, core.EndOfTurn, first); err != nil {
		t.Fatal(err)
	}
	aura := g.PutOnBattlefield(repositoryCard(t, "ARN", "Fishliver Oil"), a.PlayerID())
	g.Attach(aura.ID(), p.ID())
	remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	removal := mage.RemoveAllAbilities(p.ID(), core.WhileOnBattlefield)
	removal.SetSourceID(remover.ID())
	g.AddContinuousEffect(removal)
	if g.FindPermanent(p.ID()).HasKeyword(core.Islandwalk) {
		t.Fatal("layer 6 removal failed")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
	if !g.FindPermanent(p.ID()).HasKeyword(core.Swampwalk) || !g.FindPermanent(p.ID()).HasKeyword(core.Islandwalk) {
		t.Fatal("expiry changed grant or restoration")
	}
	if err := g.AddPrintedBasicLandwalkTextEffect(g.FindPermanent(p.ID()), id, core.EndOfTurn, first); err != nil {
		t.Fatal(err)
	}
	phased := g.PhaseOut(p.ID())
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	if !g.FindPermanent(p.ID()).HasKeyword(core.Swampwalk) {
		t.Fatal("phased expiry did not restore baseline")
	}
}

func TestRONTextLandwalkLayerIdentityAndBaseline(t *testing.T) {
	g, a, _ := ronGame()
	card := repositoryCard(t, "LEA", "Bog Wraith")
	p := g.PutOnBattlefield(card, a.PlayerID())
	id := landwalkID(t, p)
	island, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	forest, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	if err := g.RevisePrintedBasicLandwalk(p, id, island); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedBasicLandwalkTextEffect(g.FindPermanent(p.ID()), id, core.EndOfTurn, forest); err != nil {
		t.Fatal(err)
	}
	if err := g.RevisePrintedBasicLandwalk(g.FindPermanent(p.ID()), id, forest); err == nil {
		t.Fatal("direct revision bypassed layered baseline")
	}
	sibling := g.PutOnBattlefield(card.Copy(), a.PlayerID())
	if !sibling.HasKeyword(core.Swampwalk) {
		t.Fatal("copy inherited noncopiable text")
	}
	clone := g.Clone()
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if !clone.FindPermanent(p.ID()).HasKeyword(core.Islandwalk) {
		t.Fatal("expiry lost prior direct revision")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	if lki := g.LKI(p.ID()); lki == nil || !lki.Snapshot.HasKeyword(core.Forestwalk) {
		t.Fatal("departure LKI lost text")
	}
	returned := g.PutOnBattlefield(card, a.PlayerID())
	g.ApplyContinuousEffects()
	if !returned.HasKeyword(core.Swampwalk) || returned.HasKeyword(core.Forestwalk) {
		t.Fatal("returned object inherited layer")
	}
	if err := g.AddPrintedBasicLandwalkTextEffect(p, id, core.Indefinite, island); err == nil {
		t.Fatal("old incarnation accepted")
	}
}

func TestRONTextLandwalkLayerInvalid(t *testing.T) {
	for _, scenario := range []string{"nil", "id", "empty", "change", "color", "duration", "face-down", "phased", "departed", "removed", "grant"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, _ := ronGame()
			p := g.PutOnBattlefield(repositoryCard(t, "LEA", "Bog Wraith"), a.PlayerID())
			id := landwalkID(t, p)
			change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
			changes := []mage.TextChange{change}
			duration := core.EndOfTurn
			input := p
			switch scenario {
			case "nil":
				input = nil
			case "id":
				id = landwalkID(t, g.PutOnBattlefield(repositoryCard(t, "LEA", "Bog Wraith"), a.PlayerID()))
			case "empty":
				changes = nil
			case "change":
				changes = []mage.TextChange{{}}
			case "color":
				change, _ = mage.ReplaceColorWord(core.White, core.Red)
				changes = []mage.TextChange{change}
			case "duration":
				duration = core.WhileOnBattlefield
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
			case "phased":
				g.PhaseOut(p.ID())
			case "departed":
				g.BouncePermanentToHand(p)
			case "removed":
				effect := mage.RemoveAllAbilities(p.ID(), core.EndOfTurn)
				g.AddContinuousEffect(effect)
			case "grant":
				g.MutablePermanent(p.ID()).RuntimeAbilities[0] = mage.WrapGrantedAbility(p.RuntimeAbilities[0])
			}
			if err := g.AddPrintedBasicLandwalkTextEffect(input, id, duration, changes...); err == nil {
				t.Fatal("invalid installation accepted")
			}
			if present := g.FindPermanent(p.ID()); present != nil && !present.FaceDown && present.HasKeyword(core.Islandwalk) {
				t.Fatal("failed installation changed text")
			}
		})
	}
	for _, rule := range []string{`Landwalk()`, `Landwalk(IsLandType(Unknown))`, `Landwalk(IsLandType(Island),Flying)`, `Landwalk(IsLandTypeVariable(TheChosenLandType))`} {
		records, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(records[0]); err == nil {
			t.Fatal("invalid declaration accepted", rule)
		}
	}
}

func TestRONTextLandwalkLayerDuplicateDeclarations(t *testing.T) {
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Landwalk(IsLandType(Swamp)),Landwalk(IsLandType(Swamp))`)(), a.PlayerID())
	first, second := p.RuntimeAbilities[0].AbilityID(), p.RuntimeAbilities[1].AbilityID()
	island, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	forest, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.ForestLandType)
	if err := g.AddPrintedBasicLandwalkTextEffect(p, first, core.EndOfTurn, island); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedBasicLandwalkTextEffect(g.FindPermanent(p.ID()), second, core.Indefinite, forest); err != nil {
		t.Fatal(err)
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	current := g.FindPermanent(p.ID())
	if !current.HasKeyword(core.Swampwalk) || !current.HasKeyword(core.Forestwalk) || current.HasKeyword(core.Islandwalk) {
		t.Fatal("sibling lifetimes mixed")
	}
}

func TestRONTextLandwalkLayerRecords(t *testing.T) {
	cases := []struct{ set, name string }{
		{"LEA", "Bog Wraith"}, {"LEA", "Shanodin Dryads"}, {"DRK", "Marsh Goblins"},
		{"LEG", "Cat Warriors"}, {"LEG", "Devouring Deep"}, {"LEG", "Lost Soul"},
		{"LEG", "Mountain Yeti"}, {"LEG", "Righteous Avengers"}, {"LEG", "Segovian Leviathan"}, {"LEG", "Sol'kanar the Swamp King"},
	}
	landOf := map[core.Attr]core.BasicLandType{core.Plainswalk: core.PlainsLandType, core.Islandwalk: core.IslandLandType, core.Swampwalk: core.SwampLandType, core.Mountainwalk: core.MountainLandType, core.Forestwalk: core.ForestLandType}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			p := g.PutOnBattlefield(card, a.PlayerID())
			var from core.BasicLandType
			var old core.Attr
			for kw, word := range landOf {
				if p.HasKeyword(kw) {
					from, old = word, kw
					break
				}
			}
			if from.String() == "" {
				t.Fatal("missing printed word")
			}
			to, next := core.IslandLandType, core.Islandwalk
			if from == to {
				to, next = core.ForestLandType, core.Forestwalk
			}
			g.PutOnBattlefield(repositoryCard(t, "LEA", to.String()), b.PlayerID())
			change, _ := mage.ReplaceBasicLandWord(from, to)
			id := landwalkID(t, p)
			if err := g.AddPrintedBasicLandwalkTextEffect(p, id, core.EndOfTurn, change); err != nil {
				t.Fatal(err)
			}
			for range 2 {
				g.ApplyContinuousEffects()
			}
			current := g.FindPermanent(p.ID())
			if landwalkID(t, current) != id || !current.HasKeyword(next) || current.HasKeyword(old) || !mage.HasLandwalkEvasion(current, b.PlayerID(), g) {
				t.Fatal("layer did not use real RON declaration")
			}
			if tc.name == "Mountain Yeti" && !current.HasProtectionFrom(textColorCard(core.White)) {
				t.Fatal("landwalk changed protection")
			}
			if tc.name == "Sol'kanar the Swamp King" {
				castRONSpell(t, g, b, textTriggerSpell(core.Black))
				g.PutTriggersOnStack()
				g.ResolveStack()
				if a.Life() != 21 {
					t.Fatal("landwalk changed spell-cast trigger")
				}
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if !g.FindPermanent(p.ID()).HasKeyword(old) || g.FindPermanent(p.ID()).HasKeyword(next) {
				t.Fatal("real declaration did not restore")
			}
		})
	}
}
