package ron

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextCopySpellRecords(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Clone", "Copy Artifact"} {
			t.Run(set+"/"+name, func(t *testing.T) {
				g, a, b := ronGame()
				chooser := &ronEntryCopyPlayer{Player: a, accept: true}
				g.SetPlayerAt(0, chooser)
				donorName := "Bog Wraith"
				if name == "Copy Artifact" {
					donorName = "Gauntlet of Might"
				}
				donor := g.PutOnBattlefield(repositoryCard(t, "LEA", donorName), b.PlayerID())
				chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return donor }
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || chooser.offers != 0 {
					t.Fatal("unpaid cast made an entry choice")
				}
				a.ManaPool().Add(core.Blue, 1)
				a.ManaPool().Add(core.Colorless, 3)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				paid := a.ManaPool().TotalMana()
				from := core.SwampLandType
				if name == "Copy Artifact" {
					from = core.MountainLandType
				}
				land, _ := mage.ReplaceBasicLandWord(from, core.IslandLandType)
				color, _ := mage.ReplaceColorWord(core.Red, core.Green)
				if err := g.RevisePrintedSpellText(g.StackPeek().ID, land, color); err != nil {
					t.Fatal(err)
				}
				if chooser.offers != 0 || a.ManaPool().TotalMana() != paid || g.StackPeek().Card != card {
					t.Fatal("revision changed announcement")
				}
				clone := g.Clone()
				cloneChooser := &ronEntryCopyPlayer{Player: clone.GetPlayer(a.PlayerID()), accept: true}
				cloneChooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return clone.FindPermanent(donor.ID()) }
				clone.SetPlayerAt(0, cloneChooser)
				g.ResolveStack()
				clone.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					p := game.FindPermanent(card.ID())
					if p == nil || p.Name() != donorName || p.Card.Owner() != a.PlayerID() || p.ControllerID() != a.PlayerID() || p.IsToken {
						t.Fatal("copy lost entry identity")
					}
					if name == "Clone" {
						if !p.HasKeyword(core.Islandwalk) || p.HasKeyword(core.Swampwalk) {
							t.Fatal("copy acquisition lost continued landwalk")
						}
						island := game.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
						if !mage.HasLandwalkEvasion(p, b.PlayerID(), game) || island == nil {
							t.Fatal("continued landwalk has no evasion")
						}
					} else {
						if !p.HasType(core.TypeArtifact) || !p.HasType(core.TypeEnchantment) {
							t.Fatal("text lost copy exceptions")
						}
						green := game.PutOnBattlefield(textColorCard(core.Green), a.PlayerID())
						red := game.PutOnBattlefield(textColorCard(core.Red), a.PlayerID())
						if green.CurrentPower(game) != 2 || red.CurrentPower(game) != 2 {
							t.Fatal("continued color word or independent donor rule failed")
						}
						island := game.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), a.PlayerID())
						before := game.GetPlayer(a.PlayerID()).ManaPool().Count(core.Red)
						if err := game.ActivateAbilityByIndex(a.PlayerID(), island.ID(), 0, nil); err != nil {
							t.Fatal(err)
						}
						if game.GetPlayer(a.PlayerID()).ManaPool().Count(core.Red) != before+1 {
							t.Fatal("continued mana event lost land word or changed mana symbol")
						}
					}
					values, err := game.CaptureCopiableValues(game.FindPermanent(card.ID()))
					if err != nil {
						t.Fatal(err)
					}
					printed, err := values.Build()
					if err != nil {
						t.Fatal(err)
					}
					sibling := game.PutOnBattlefield(printed, a.PlayerID())
					if name == "Clone" && (!sibling.HasKeyword(core.Swampwalk) || sibling.HasKeyword(core.Islandwalk)) {
						t.Fatal("ordinary text became copiable")
					}
					if name == "Copy Artifact" && !sibling.HasType(core.TypeEnchantment) {
						t.Fatal("later copy lost enchantment exception")
					}
				}
				g.PhaseIn(g.PhaseOut(card.ID()))
				g.ApplyContinuousEffects()
				if chooser.offers != 1 {
					t.Fatal("phasing repeated entry choice")
				}
				if name == "Clone" && !g.FindPermanent(card.ID()).HasKeyword(core.Islandwalk) {
					t.Fatal("phasing lost continued text")
				}
				g.BouncePermanentToHand(g.FindPermanent(card.ID()))
				if len(a.Hand()) != 1 || a.Hand()[0].Name() != name || g.LKI(card.ID()).Name != donorName || clone.FindPermanent(card.ID()) == nil {
					t.Fatal("departure lost printed card, LKI or clone isolation")
				}
				if name == "Clone" && !g.LKI(card.ID()).Snapshot.HasKeyword(core.Islandwalk) {
					t.Fatal("LKI lost continued landwalk")
				}
				a.ManaPool().Add(core.Blue, 1)
				a.ManaPool().Add(core.Colorless, 3)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if name == "Clone" && (!g.FindPermanent(card.ID()).HasKeyword(core.Swampwalk) || g.FindPermanent(card.ID()).HasKeyword(core.Islandwalk)) {
					t.Fatal("recast retained old continuation")
				}
			})
		}
	}
}

func TestRONTextCopySpellCopies(t *testing.T) {
	g, a, b := ronGame()
	donor := g.PutOnBattlefield(repositoryCard(t, "LEA", "Bog Wraith"), b.PlayerID())
	for i, player := range []mage.Player{a, b} {
		chooser := &ronEntryCopyPlayer{Player: player, accept: true}
		chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return donor }
		g.SetPlayerAt(i, chooser)
	}
	card := repositoryCard(t, "LEA", "Clone")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 1)
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	first, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, first); err != nil {
		t.Fatal(err)
	}
	copySpell := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	second, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	if err := g.RevisePrintedSpellText(copySpell.ID, second); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	for i, player := range []mage.Player{a, b} {
		chooser := &ronEntryCopyPlayer{Player: clone.GetPlayer(player.PlayerID()), accept: true}
		chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return clone.FindPermanent(donor.ID()) }
		clone.SetPlayerAt(i, chooser)
	}
	third, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.MountainLandType)
	if err := clone.RevisePrintedSpellText(copySpell.ID, third); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	clone.ResolveStack()
	if !g.FindPermanent(card.ID()).HasKeyword(core.Islandwalk) || !clone.FindPermanent(card.ID()).HasKeyword(core.Islandwalk) || !g.FindPermanent(copySpell.SourceID).HasKeyword(core.Forestwalk) || !clone.FindPermanent(copySpell.SourceID).HasKeyword(core.Mountainwalk) {
		t.Fatal("copies and clones shared continued text")
	}
	p := g.FindPermanent(copySpell.SourceID)
	if !p.IsToken || !p.Card.IsToken() || p.ControllerID() != b.PlayerID() {
		t.Fatal("spell copy lost token status or controller")
	}
	values, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	printed, err := values.Build()
	if err != nil || printed.AttrSeeds()[core.Swampwalk] != 1 || printed.AttrSeeds()[core.Forestwalk] != 0 {
		t.Fatal("token donor copy values included ordinary text", err)
	}
	g.BouncePermanentToHand(p)
	if len(b.Hand()) != 0 || len(b.Graveyard()) != 0 || clone.FindPermanent(copySpell.SourceID) == nil {
		t.Fatal("token departure changed physical zones or clone")
	}
}

func textCopyFactory(t *testing.T, name, rules string) mage.CardFactory {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"`+name+`",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:2),Rules:[`+rules+`])`), "copy-text.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONTextCopySpellDonorBindings(t *testing.T) {
	g, a, b := ronGame()
	donor := g.PutOnBattlefield(textCopyFactory(t, "Donor", `AsPermanentEnters(ThisPermanent,[EntersWithACounter(JavelinCounter)]),Activated(And([PayMana([]),RemoveCounters([ACounterOfTypeFromPermanent(JavelinCounter,ThisPermanent)])]),Targeted([TargetPermanent(IsColor(Red))],ActionList([TapPermanent(Ref_TargetPermanent)]))),TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),Targeted([TargetPermanent(IsLandType(Swamp))],ActionList([TapPermanent(Ref_TargetPermanent)])))`)(), b.PlayerID())
	g.ResolveStack()
	swamp := g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), a.PlayerID())
	island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), a.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), a.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), a.PlayerID())
	chooser := &copySpellTargetPlayer{ronEntryCopyPlayer: ronEntryCopyPlayer{Player: a, accept: true}, target: island.ID()}
	chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return donor }
	g.SetPlayerAt(0, chooser)
	card := repositoryCard(t, "LEA", "Clone")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 1)
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	color, _ := mage.ReplaceColorWord(core.Red, core.Green)
	land, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, color, land); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil || p.Counters[core.Javelin] != 1 || !g.FindPermanent(island.ID()).Tapped || g.FindPermanent(swamp.ID()).Tapped {
		t.Fatal("acquired entry or ETB trigger did not use fresh text")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{red.ID()}); err == nil || g.FindPermanent(p.ID()).Counters[core.Javelin] != 1 {
		t.Fatal("illegal target paid copied cost")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, []uuid.UUID{green.ID()}); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	g.ResolveStack()
	if !g.FindPermanent(green.ID()).Tapped || g.FindPermanent(donor.ID()).Counters[core.Javelin] != 1 {
		t.Fatal("pending donor action lost fresh binding or changed donor")
	}
}

type copySpellTargetPlayer struct {
	ronEntryCopyPlayer
	target uuid.UUID
}

func (p *copySpellTargetPlayer) ChooseTargets([]uuid.UUID, int, int, *mage.Game) []uuid.UUID {
	return []uuid.UUID{p.target}
}

func TestRONTextCopySpellLayers(t *testing.T) {
	g, a, b := ronGame()
	donor := g.PutOnBattlefield(repositoryCard(t, "LEA", "Bog Wraith"), b.PlayerID())
	chooser := &ronEntryCopyPlayer{Player: a, accept: true}
	chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return donor }
	g.SetPlayerAt(0, chooser)
	card := repositoryCard(t, "LEA", "Clone")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 1)
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	first, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, first); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	next, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	if err := g.AddPrintedBasicLandwalkTextEffect(p, landwalkID(t, p), core.EndOfTurn, next); err != nil {
		t.Fatal(err)
	}
	g.ApplyContinuousEffects()
	g.ApplyContinuousEffects()
	if !g.FindPermanent(p.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("later layer lost continued baseline")
	}
	g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
	if g.FindPermanent(p.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("ability removal did not suspend continued rules")
	}
	g.PhaseOut(p.ID())
	g.GetEffects().RemoveEndOfTurn()
	g.PhaseIn([]uuid.UUID{p.ID()})
	g.ApplyContinuousEffects()
	if !g.FindPermanent(p.ID()).HasKeyword(core.Islandwalk) || g.FindPermanent(p.ID()).HasKeyword(core.Forestwalk) || chooser.offers != 1 {
		t.Fatal("expiry did not restore continued baseline")
	}
}

func TestRONTextCopySpellEntryShapes(t *testing.T) {
	for _, state := range []string{"decline", "empty", "phased", "returned", "return-during-choice", "facedown", "chain"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			donor := g.PutOnBattlefield(repositoryCard(t, "LEA", "Bog Wraith"), b.PlayerID())
			chooser := &ronEntryCopyPlayer{Player: a, accept: state != "decline"}
			chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return donor }
			g.SetPlayerAt(0, chooser)
			card := repositoryCard(t, "LEA", "Clone")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 1)
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
			if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "empty":
				g.BouncePermanentToHand(donor)
			case "phased":
				g.PhaseOut(donor.ID())
			case "returned":
				g.BouncePermanentToHand(donor)
				donor = g.PutOnBattlefield(donor.Card, b.PlayerID())
			case "return-during-choice":
				chooser.pick = func(_ []*mage.Permanent, _ mage.GameReader) *mage.Permanent {
					old := donor
					g.BouncePermanentToHand(donor)
					g.PutOnBattlefield(donor.Card, b.PlayerID())
					return old
				}
			case "facedown":
				donor.FaceDown = true
			case "chain":
				chooser.accept = false
				intermediate := g.PutOnBattlefield(textCopyFactory(t, "Intermediate", `AsPermanentEnters(ThisPermanent,[MayActions([EnterAsACopyOfAPermanent(IsCardtype(Creature),NoCopyEffects)])])`)(), a.PlayerID())
				chooser.accept, chooser.offers, chooser.choices = true, 0, 0
				chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent {
					if chooser.choices == 1 {
						return intermediate
					}
					return donor
				}
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if state == "empty" || state == "phased" || state == "decline" || state == "return-during-choice" {
				if p != nil || len(a.Graveyard()) != 1 || a.Graveyard()[0].Name() != "Clone" {
					t.Fatal("declined or absent donor changed physical departure")
				}
				return
			}
			if state == "facedown" {
				if p == nil || p.Name() != "" || p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 2 || p.FaceDown || len(p.RuntimeAbilities) != 0 {
					t.Fatal("face-down copy acquired text or status")
				}
				if _, err := g.CaptureCopiableValues(p); err != nil {
					t.Fatal(err)
				}
				return
			}
			if p == nil || !p.HasKeyword(core.Islandwalk) {
				t.Fatal("entry recheck or nested copy lost continuation")
			}
			if state == "chain" && chooser.offers != 2 {
				t.Fatal("copy-entry replacement identity repeated or skipped a rule")
			}
		})
	}
}

func TestRONTextCopySpellBoundaries(t *testing.T) {
	for _, rules := range []string{
		`AsPermanentEnters(ThisPermanent,[MayActions([EnterAsACopyOfAPermanent(IsCardtype(Creature),NoCopyEffects())])])`,
		`AsPermanentEnters(ThisPermanent,[MayActions([EnterAsACopyOfAPermanent(IsCardtype(Creature),CopyEffects([]))])])`,
		`AsPermanentEnters(HostPermanent,[MayActions([EnterAsACopyOfAPermanent(IsCardtype(Creature),NoCopyEffects)])])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[`+rules+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid copy-entry declaration accepted")
		}
	}
	g, a, _ := ronGame()
	card := textCopyFactory(t, "Mixed", `Flying,AsPermanentEnters(ThisPermanent,[MayActions([EnterAsACopyOfAPermanent(IsCardtype(Creature),NoCopyEffects)])])`)()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	before := g.StackPeek()
	change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.RevisePrintedSpellText(before.ID, change); err == nil || g.StackPeek() != before {
		t.Fatal("mixed unsupported context changed stack state")
	}
	for _, name := range []string{"Magical Hack", "Sleight of Mind", "Vesuvan Doppelganger"} {
		if _, err := Compile(staticBaseRecord(t, "LEA", name)); err == nil {
			t.Fatal("incomplete dependent card accepted", name)
		}
	}
}

func TestRONTextCopySpellInventory(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
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
			if !copyEntrySpellTextRules(record.Fields["Rules"].Args) {
				continue
			}
			name := record.Fields["Name"].String
			if name != "Clone" && name != "Copy Artifact" {
				t.Fatal("copy-entry continuation lacks gameplay mapping", name)
			}
			count++
		}
	}
	if count != 12 {
		t.Fatalf("mapped %d records, want 12", count)
	}
}

func TestRONTextCopySpellRetainedColors(t *testing.T) {
	for _, cost := range []string{`[ManaCostU]`, `[]`} {
		g, a, b := ronGame()
		donor := g.PutOnBattlefield(textCopyFactory(t, "Red donor", `Activated(PayMana([]),Targeted([TargetPermanent(IsColor(Red))],ActionList([TapPermanent(Ref_TargetPermanent)])))`)(), b.PlayerID())
		chooser := &ronEntryCopyPlayer{Player: a, accept: true}
		chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return donor }
		g.SetPlayerAt(0, chooser)
		want := []core.Color{core.Blue}
		if cost == `[]` {
			want = nil
		}
		card := copyColorFactory(t, cost, "", `KeepColor,AddCardtypes([Enchantment])`)()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Blue, 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		change, _ := mage.ReplaceColorWord(core.Red, core.Green)
		if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		p := g.FindPermanent(card.ID())
		if p == nil || !p.HasType(core.TypeEnchantment) || !slices.Equal(p.Colors(), want) {
			t.Fatal("continued donor lost retained color exception")
		}
		green := g.PutOnBattlefield(textColorCard(core.Green), a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{green.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if !g.FindPermanent(green.ID()).Tapped {
			t.Fatal("fresh donor target binding lost changed word")
		}
		values, err := g.CaptureCopiableValues(g.FindPermanent(p.ID()))
		if err != nil {
			t.Fatal(err)
		}
		printed, err := values.Build()
		if err != nil {
			t.Fatal(err)
		}
		q := g.PutOnBattlefield(printed, a.PlayerID())
		if !q.HasType(core.TypeEnchantment) || !slices.Equal(q.Colors(), want) {
			t.Fatal("later copy lost retained characteristics")
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, []uuid.UUID{green.ID()}); err == nil {
			t.Fatal("later copy retained ordinary text")
		}
	}
}

func TestRONTextCopySpellFilterRevision(t *testing.T) {
	g, a, b := ronGame()
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Gray Ogre"), b.PlayerID())
	green := g.PutOnBattlefield(repositoryCard(t, "LEA", "Grizzly Bears"), b.PlayerID())
	green.GrantBaseAttr(core.Shroud)
	chooser := &ronEntryCopyPlayer{Player: a, accept: true}
	g.SetPlayerAt(0, chooser)
	card := textCopyFactory(t, "Filtered copier", `AsPermanentEnters(ThisPermanent,[MayActions([EnterAsACopyOfAPermanent(And([IsCardtype(Creature),IsColor(Red)]),NoCopyEffects)])])`)()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil || p.Name() != green.Name() || !slices.Contains(p.Colors(), core.Green) || p.HasKeyword(core.Shroud) || chooser.offers != 1 {
		t.Fatal("copy selection retained old filter, used targets or copied a grant")
	}
}

func textCopyMixedRecord(t *testing.T, set, name string) mage.Card {
	t.Helper()
	record := staticBaseRecord(t, set, name)
	rules := record.Fields["Rules"]
	rules.Args = append(rules.Args, Node{Kind: Symbol, Name: "Shroud"})
	record.Fields["Rules"] = rules
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	return factory()
}

func TestRONTextCopySpellFaceDownExceptions(t *testing.T) {
	g, a, b := ronGame()
	donor := g.PutOnBattlefield(repositoryCard(t, "LEA", "Bog Wraith"), b.PlayerID())
	donor.FaceDown = true
	chooser := &ronEntryCopyPlayer{Player: a, accept: true}
	g.SetPlayerAt(0, chooser)
	card := copyColorFactory(t, `[ManaCostU]`, "", `KeepColor,AddCardtypes([Enchantment])`)()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil || p.Name() != "" || p.FaceDown || p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 2 || !p.HasType(core.TypeEnchantment) || !slices.Equal(p.Colors(), []core.Color{core.Blue}) || len(p.RuntimeAbilities) != 0 {
		t.Fatal("face-down copy exceptions lost textless provenance")
	}
	values, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	printed, err := values.Build()
	if err != nil || printed.Name() != "" || !printed.HasType(core.TypeEnchantment) {
		t.Fatal("later copy lost face-down copy exceptions", err)
	}
}

func TestRONTextCopySpellIntrinsicDonorException(t *testing.T) {
	g, a, b := ronGame()
	donor := g.PutOnBattlefield(repositoryCard(t, "LEA", "Bog Wraith"), b.PlayerID())
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.AddCardType(mage.This, core.TypeArtifact)), donor.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	chooser := &ronEntryCopyPlayer{Player: a, accept: true}
	g.SetPlayerAt(0, chooser)
	card := repositoryCard(t, "LEA", "Copy Artifact")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 1)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil || !p.HasType(core.TypeCreature) || !p.HasType(core.TypeEnchantment) || p.HasType(core.TypeArtifact) || !p.HasKeyword(core.Islandwalk) || p.HasKeyword(core.Swampwalk) {
		t.Fatal("copiable type exception rejected or changed intrinsic donor rules")
	}
	values, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	printed, err := values.Build()
	if err != nil || !printed.HasType(core.TypeEnchantment) || printed.AttrSeeds()[core.Swampwalk] != 1 || printed.AttrSeeds()[core.Islandwalk] != 0 {
		t.Fatal("later copy lost type exception or retained ordinary text", err)
	}
}
