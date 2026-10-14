package ron

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func landwalkID(t *testing.T, p *mage.Permanent) uuid.UUID {
	t.Helper()
	for _, a := range p.RuntimeAbilities {
		if _, ok := a.(*mage.PrintedBasicLandwalkAbility); ok {
			return a.AbilityID()
		}
	}
	t.Fatal("missing printed landwalk declaration")
	return uuid.Nil
}

func TestRONTextLandwalkRecords(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	count := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		records, err := Parse(bytes.NewReader(data), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, record := range records {
			for _, rule := range record.Fields["Rules"].Args {
				if rule.Name != "Landwalk" || rule.Args[0].Name != "IsLandType" || rule.Args[0].Args[0].Name == "Desert" {
					continue
				}
				factory, err := Compile(record)
				if err != nil {
					t.Fatal(err)
				}
				card := factory()
				names[card.Name()] = true
				count++
				t.Run(filepath.Base(file)+"/"+card.Name(), func(t *testing.T) {
					g, a, b := ronGame()
					p := g.PutOnBattlefield(card, a.PlayerID())
					id := landwalkID(t, p)
					old, err := compileLandwalkKeyword(rule)
					if err != nil {
						t.Fatal(err)
					}
					from := map[core.Attr]core.BasicLandType{core.Plainswalk: core.PlainsLandType, core.Islandwalk: core.IslandLandType, core.Swampwalk: core.SwampLandType, core.Mountainwalk: core.MountainLandType, core.Forestwalk: core.ForestLandType}[old]
					to, walk := core.IslandLandType, core.Islandwalk
					if old == core.Islandwalk {
						to, walk = core.ForestLandType, core.Forestwalk
					}

					g.PutOnBattlefield(repositoryCard(t, "LEA", from.String()), b.PlayerID())
					if !mage.HasLandwalkEvasion(p, b.PlayerID(), g) {
						t.Fatal("printed landwalk permits blocking")
					}
					change, _ := mage.ReplaceBasicLandWord(from, to)
					if err := g.RevisePrintedBasicLandwalk(p, id, change); err != nil {
						t.Fatal(err)
					}
					current := g.FindPermanent(p.ID())
					if landwalkID(t, current) != id || current.HasKeyword(old) || !current.HasKeyword(walk) || mage.HasLandwalkEvasion(current, b.PlayerID(), g) {
						t.Fatal("revision did not replace intrinsic contribution")
					}
					g.PutOnBattlefield(repositoryCard(t, "LEA", to.String()), b.PlayerID())
					if !mage.HasLandwalkEvasion(g.FindPermanent(p.ID()), b.PlayerID(), g) {
						t.Fatal("revised evasion permits blocking")
					}
					if card.Name() != current.Name() || card.AttrSeeds()[old] != 1 || card.AttrSeeds()[walk] != 0 {
						t.Fatal("revision changed printed copy information")
					}
				})
			}
		}
	}
	if len(names) != 13 || count != 23 {
		t.Fatalf("coverage: %d names / %d records", len(names), count)
	}
}

func TestRONTextLandwalkLifetime(t *testing.T) {
	g, a, _ := ronGame()
	card := repositoryCard(t, "LEA", "Bog Wraith")
	p := g.PutOnBattlefield(card, a.PlayerID())
	id := landwalkID(t, p)
	sibling := g.PutOnBattlefield(repositoryCard(t, "LEA", "Bog Wraith"), a.PlayerID())
	red, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.MountainLandType)
	blue, _ := mage.ReplaceBasicLandWord(core.MountainLandType, core.IslandLandType)
	if err := g.RevisePrintedBasicLandwalk(p, id, red); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := clone.RevisePrintedBasicLandwalk(clone.FindPermanent(p.ID()), id, blue); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(p.ID()).HasKeyword(core.Mountainwalk) || !clone.FindPermanent(p.ID()).HasKeyword(core.Islandwalk) || !sibling.HasKeyword(core.Swampwalk) {
		t.Fatal("revision state shared")
	}
	phased := g.PhaseOut(p.ID())
	if err := g.RevisePrintedBasicLandwalk(p, id, blue); err == nil {
		t.Fatal("phased source accepted")
	}
	g.PhaseIn(phased)
	if !g.FindPermanent(p.ID()).HasKeyword(core.Mountainwalk) {
		t.Fatal("phasing lost revision")
	}
	remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
	effect := mage.RemoveAllAbilities(p.ID(), core.WhileOnBattlefield)
	effect.SetSourceID(remover.ID())
	g.AddContinuousEffect(effect)
	if g.FindPermanent(p.ID()).HasKeyword(core.Mountainwalk) {
		t.Fatal("removed landwalk active")
	}
	if err := g.RevisePrintedBasicLandwalk(g.FindPermanent(p.ID()), id, blue); err == nil {
		t.Fatal("removed declaration accepted")
	}
	removedClone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
	removedClone.RemoveFromBattlefield(removedClone.FindPermanent(remover.ID()))
	if !g.FindPermanent(p.ID()).HasKeyword(core.Mountainwalk) || !removedClone.FindPermanent(p.ID()).HasKeyword(core.Mountainwalk) {
		t.Fatal("restoration lost revision")
	}
	if err := removedClone.RevisePrintedBasicLandwalk(removedClone.FindPermanent(p.ID()), id, blue); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(p.ID()).HasKeyword(core.Mountainwalk) {
		t.Fatal("restored clone changed original")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	if lki := g.LKI(p.ID()); lki == nil || !lki.Snapshot.HasKeyword(core.Mountainwalk) {
		t.Fatal("LKI lost revision")
	}
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.RevisePrintedBasicLandwalk(p, id, blue); err == nil {
		t.Fatal("returned source accepted")
	}
	if !returned.HasKeyword(core.Swampwalk) || returned.HasKeyword(core.Mountainwalk) {
		t.Fatal("return lost printed origin")
	}
}

func TestRONTextLandwalkDeclarationsAndGrants(t *testing.T) {
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Landwalk(IsLandType(Swamp)),Landwalk(IsLandType(Swamp))`)(), a.PlayerID())
	first, second := p.RuntimeAbilities[0].AbilityID(), p.RuntimeAbilities[1].AbilityID()
	red, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.MountainLandType)
	blue, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.RevisePrintedBasicLandwalk(p, first, red); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(p.ID()).HasKeyword(core.Swampwalk) {
		t.Fatal("revision removed sibling contribution")
	}
	if err := g.RevisePrintedBasicLandwalk(g.FindPermanent(p.ID()), second, blue); err != nil {
		t.Fatal(err)
	}
	current := g.FindPermanent(p.ID())
	if current.HasKeyword(core.Swampwalk) || !current.HasKeyword(core.Mountainwalk) || !current.HasKeyword(core.Islandwalk) {
		t.Fatal("declarations shared history")
	}
	creature := g.PutOnBattlefield(repositoryCard(t, "LEA", "Bog Wraith"), a.PlayerID())
	aura := g.PutOnBattlefield(repositoryCard(t, "ARN", "Fishliver Oil"), a.PlayerID())
	g.Attach(aura.ID(), creature.ID())
	g.ApplyContinuousEffects()
	island, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.ForestLandType)
	if err := g.RevisePrintedBasicLandwalk(creature, landwalkID(t, creature), island); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(creature.ID()).HasKeyword(core.Islandwalk) || !g.FindPermanent(creature.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("revision changed grant")
	}
	g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
	if g.FindPermanent(creature.ID()).HasKeyword(core.Islandwalk) || !g.FindPermanent(creature.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("grant departure changed printed contribution")
	}
}

func TestRONTextLandwalkInvalid(t *testing.T) {
	for _, rule := range []string{`Landwalk()`, `Landwalk(IsLandType())`, `Landwalk(IsLandType(Island),Flying)`, `Landwalk(IsLandType(UnknownLand))`, `Landwalk(IsLandTypeVariable(TheChosenLandType))`} {
		records, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[`+rule+`],CardPT:(Power:1,Toughness:1))`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(records[0]); err == nil {
			t.Fatal("invalid landwalk accepted", rule)
		}
	}
	for _, tc := range []struct {
		set, name string
		kw        core.Attr
	}{{"ARN", "Desert Nomads", core.Desertwalk}, {"LEG", "Livonya Silone", core.LegendaryLandwalk}} {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
		if !p.HasKeyword(tc.kw) {
			t.Fatal("nonbasic landwalk lost")
		}
		for _, ability := range p.RuntimeAbilities {
			if _, ok := ability.(*mage.PrintedBasicLandwalkAbility); ok {
				t.Fatal("nonbasic declaration opted in")
			}
		}
	}
}

func TestRONTextLandwalkCastAndRemoval(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Bog Wraith")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 1)
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil {
		t.Fatal("creature entered before resolution")
	}
	g.ResolveTopOfStack()
	p := g.FindPermanent(card.ID())
	id := landwalkID(t, p)
	p.GrantBaseAttr(core.Swampwalk)
	island, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.RevisePrintedBasicLandwalk(p, id, island); err != nil {
		t.Fatal(err)
	}
	current := g.FindPermanent(p.ID())
	if !current.HasKeyword(core.Swampwalk) || !current.HasKeyword(core.Islandwalk) {
		t.Fatal("revision removed separate base contribution")
	}
	current.RevokeBaseAttr(core.Swampwalk)
	remover := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	removal := mage.RevokeKeywordFromAll(core.Islandwalk, mage.AnyPermanent)
	removal.SetSourceID(remover.ID())
	g.AddContinuousEffect(removal)
	if g.FindPermanent(p.ID()).HasKeyword(core.Islandwalk) {
		t.Fatal("keyword removal ignored revision")
	}
	g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
	if !g.FindPermanent(p.ID()).HasKeyword(core.Islandwalk) || g.FindPermanent(p.ID()).HasKeyword(core.Swampwalk) {
		t.Fatal("keyword restoration lost revision")
	}
}

func TestRONTextLandwalkBlockDeclaration(t *testing.T) {
	for _, land := range []string{"Swamp", "Island"} {
		t.Run(land, func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefieldAttacking(repositoryCard(t, "LEA", "Bog Wraith"), a.PlayerID(), b.PlayerID())
			blocker := g.PutOnBattlefield(repositoryCard(t, "LEA", "Grizzly Bears"), b.PlayerID())
			g.PutOnBattlefield(repositoryCard(t, "LEA", land), b.PlayerID())
			island, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
			if err := g.RevisePrintedBasicLandwalk(p, landwalkID(t, p), island); err != nil {
				t.Fatal(err)
			}
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: p.ID()}})
			if g.IsBlockingInCombat(blocker.ID()) != (land == "Swamp") {
				t.Fatal("block declaration did not use revised evasion")
			}
		})
	}
}
