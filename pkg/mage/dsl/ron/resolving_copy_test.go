package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONResolvingCopyGameplay(t *testing.T) {
	for _, modifications := range []string{"NoCopyEffects", "CopyEffects([KeepColor])"} {
		t.Run(modifications, func(t *testing.T) {
			g, a, b := ronGame()
			donor := g.PutOnBattlefield(repositoryCard(t, "LEA", "Air Elemental"), b.PlayerID())
			factory := textCopyFactory(t, "Copier", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentLayerEffect(ThisPermanent,[IsACopyOfPermanent(Ref_TargetPermanent,`+modifications+`)])])) )`)
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{donor.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p = g.FindPermanent(p.ID())
			if p.Name() != "Air Elemental" || len(p.RuntimeAbilities) != 0 || !p.HasAttr(core.Flying) || p.CurrentPower(g) != 4 {
				t.Fatal("copy rules missing")
			}
		})
	}
}

func TestRONResolvingCopyInvalid(t *testing.T) {
	valid := `CreatePermanentLayerEffect(ThisPermanent,[IsACopyOfPermanent(Ref_TargetPermanent,NoCopyEffects)])`
	for _, action := range []string{
		strings.Replace(valid, "NoCopyEffects", "NoCopyEffects()", 1),
		strings.Replace(valid, "NoCopyEffects", "CopyEffects([])", 1),
		strings.Replace(valid, "NoCopyEffects", "CopyEffects([KeepColor()])", 1),
		strings.Replace(valid, "NoCopyEffects", "CopyEffects([AddAbilityVariable(ThisAbility)])", 1),
		strings.Replace(valid, "Ref_TargetPermanent", "Ref_TargetPlayer", 1),
		strings.Replace(valid, "ThisPermanent", "HostPermanent", 1),
		strings.Replace(valid, ")])", ",NoCopyEffects)])", 1),
	} {
		assertConditionalRejected(t, `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+action+`])))`)
	}
}

func TestRONResolvingCopyVesuvanClause(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := repositoryRecord(t, set, "Vesuvan Doppelganger", set)
			rewrite := func(node Node) Node { return node }
			rewrite = func(node Node) Node {
				if node.Name == "MayAction" {
					node = node.Args[0]
				}
				if node.Name == "CopyEffects" && len(node.Args) == 1 && len(node.Args[0].Args) == 2 && node.Args[0].Args[1].Name == "AddAbilityVariable" {
					node.Args[0].Args = node.Args[0].Args[:1]
				}
				for i, child := range node.Args {
					node.Args[i] = rewrite(child)
				}
				for key, child := range node.Fields {
					node.Fields[key] = rewrite(child)
				}
				return node
			}
			factory, err := Compile(rewrite(record))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			donor := g.PutOnBattlefield(repositoryCard(t, "LEA", "Grizzly Bears"), b.PlayerID())
			next := g.PutOnBattlefield(repositoryCard(t, "LEA", "Air Elemental"), b.PlayerID())
			chooser := &copySpellTargetPlayer{ronEntryCopyPlayer: ronEntryCopyPlayer{Player: a, accept: true}, target: next.ID()}
			chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return donor }
			g.SetPlayerAt(0, chooser)
			card := factory()
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 2)
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			queueRONUpkeep(g, a)
			if g.StackSize() != 1 {
				t.Fatal("missing upkeep")
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p.Name() != "Air Elemental" || p.CurrentPower(g) != 4 || !p.HasAttr(core.Flying) || len(p.Colors()) != 1 || p.Colors()[0] != core.Blue {
				t.Fatal("resolving copy clause")
			}
			queueRONUpkeep(g, a)
			if g.StackSize() != 0 {
				t.Fatal("old copied trigger survived replacement")
			}
		})
	}
}

func TestRONResolvingCopyTextContinuation(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "LEA", "Bog Wraith"), a.PlayerID())
	donor := g.PutOnBattlefield(repositoryCard(t, "LEA", "Bog Wraith"), b.PlayerID())
	air := g.PutOnBattlefield(repositoryCard(t, "LEA", "Air Elemental"), b.PlayerID())
	first, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	second, _ := mage.ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
	if err := g.AddPrintedBasicLandwalkTextEffect(p, landwalkID(t, p), core.EndOfTurn, first); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedBasicLandwalkTextEffect(g.FindPermanent(p.ID()), landwalkID(t, p), core.Indefinite, second); err != nil {
		t.Fatal(err)
	}
	if err := g.BecomeCopyOfPermanent(g.FindPermanent(p.ID()), air, false); err != nil {
		t.Fatal(err)
	}
	if err := g.BecomeCopyOfPermanent(g.FindPermanent(p.ID()), donor, false); err != nil {
		t.Fatal(err)
	}
	p = g.FindPermanent(p.ID())
	if !p.HasKeyword(core.Forestwalk) || p.HasKeyword(core.Swampwalk) || p.HasKeyword(core.Islandwalk) {
		t.Fatal("continued ordered text")
	}
	clone := g.Clone()
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if !clone.FindPermanent(p.ID()).HasKeyword(core.Swampwalk) || clone.FindPermanent(p.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("expiry did not restore baseline")
	}
	if !g.FindPermanent(p.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("clone changed original")
	}
	values, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	card, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	later := g.PutOnBattlefield(card, a.PlayerID())
	if !later.HasKeyword(core.Swampwalk) || later.HasKeyword(core.Forestwalk) {
		t.Fatal("ordinary text became copiable")
	}
	for range 3 {
		g.ApplyContinuousEffects()
	}
	if !g.FindPermanent(p.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("repeated layers changed words")
	}
}

func TestRONResolvingCopyTimingAndVisits(t *testing.T) {
	for _, change := range []string{"none", "illegal", "target phase", "target return", "target phase cycle", "source return", "source phase", "source control", "clone", "stack copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			donor := g.PutOnBattlefield(repositoryCard(t, "LEA", "Air Elemental"), b.PlayerID())
			factory := textCopyFactory(t, "Copier", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentLayerEffect(ThisPermanent,[IsACopyOfPermanent(Ref_TargetPermanent,NoCopyEffects)])])))`)
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{donor.ID()}); err != nil {
				t.Fatal(err)
			}
			if p.Name() != "Copier" {
				t.Fatal("copy before resolution")
			}
			want := true
			switch change {
			case "illegal":
				donor.GrantBaseAttr(core.Shroud)
				want = false
			case "target phase":
				g.PhaseOut(donor.ID())
				want = false
			case "target return":
				g.BouncePermanentToHand(donor)
				c := b.Hand()[0]
				b.RemoveFromHand(c.ID())
				g.PutOnBattlefield(c, b.PlayerID())
				want = false
			case "target phase cycle":
				g.PhaseIn(g.PhaseOut(donor.ID()))
			case "source return":
				g.BouncePermanentToHand(p)
				c := a.Hand()[0]
				a.RemoveFromHand(c.ID())
				g.PutOnBattlefield(c, a.PlayerID())
				want = false
			case "source phase":
				g.PhaseOut(p.ID())
				g.ResolveStack()
				g.PhaseIn([]uuid.UUID{p.ID()})
				want = false
			case "source control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: donor.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "clone":
				clone := g.Clone()
				clone.ResolveStack()
				if clone.FindPermanent(p.ID()).Name() != "Air Elemental" || g.FindPermanent(p.ID()).Name() != "Copier" {
					t.Fatal("pending clone")
				}
			case "stack copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			if got := g.FindPermanent(p.ID()).Name() == "Air Elemental"; got != want {
				t.Fatal("visit or legality", got, want)
			}
			if change == "source control" && g.FindPermanent(p.ID()).ControllerID() != b.PlayerID() {
				t.Fatal("copy reset control")
			}
		})
	}
}

func TestRONResolvingCopyNewTextAndExpiry(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "LEA", "Grizzly Bears"), a.PlayerID())
	donor := g.PutOnBattlefield(repositoryCard(t, "LEA", "Bog Wraith"), b.PlayerID())
	if err := g.BecomeCopyOfPermanent(p, donor, false); err != nil {
		t.Fatal(err)
	}
	p = g.FindPermanent(p.ID())
	change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
	if err := g.AddPrintedBasicLandwalkTextEffect(p, landwalkID(t, p), core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(p.ID()).HasKeyword(core.Islandwalk) {
		t.Fatal("new text")
	}
	clone := g.Clone()
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if !g.FindPermanent(p.ID()).HasKeyword(core.Swampwalk) || !clone.FindPermanent(p.ID()).HasKeyword(core.Islandwalk) {
		t.Fatal("independent expiry")
	}
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if !clone.FindPermanent(p.ID()).HasKeyword(core.Swampwalk) {
		t.Fatal("clone baseline")
	}
}

func TestRONResolvingCopyGroupedText(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "LEA", "Gauntlet of Might"), a.PlayerID())
	donor := g.PutOnBattlefield(repositoryCard(t, "LEA", "Gauntlet of Might"), b.PlayerID())
	black := g.PutOnBattlefield(mage.NewCreature("Black fixture", "{B}", 2, 2), a.PlayerID())
	blue := g.PutOnBattlefield(mage.NewCreature("Blue fixture", "{U}", 2, 2), a.PlayerID())
	first, _ := mage.ReplaceColorWord(core.Black, core.Blue)
	second, _ := mage.ReplaceColorWord(core.Red, core.Black)
	if err := g.AddPrintedMixedStaticEventTextEffect(p, core.EndOfTurn, first, second); err != nil {
		t.Fatal(err)
	}
	if err := g.BecomeCopyOfPermanent(g.FindPermanent(p.ID()), donor, false); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(black.ID()).CurrentPower(g) != 3 || g.FindPermanent(blue.ID()).CurrentPower(g) != 2 {
		t.Fatal("one instruction applied once for each old rule")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if g.FindPermanent(black.ID()).CurrentPower(g) != 2 || g.FindPermanent(blue.ID()).CurrentPower(g) != 2 {
		t.Fatal("paired text expiry")
	}
	if err := g.AddPrintedMixedStaticEventTextEffect(g.FindPermanent(p.ID()), core.EndOfTurn, first, second); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(black.ID()).CurrentPower(g) != 3 || g.FindPermanent(blue.ID()).CurrentPower(g) != 2 {
		t.Fatal("new paired text")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if g.FindPermanent(black.ID()).CurrentPower(g) != 2 {
		t.Fatal("new paired text expiry")
	}
}

func TestRONResolvingCopyInventory(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),Targeted([TargetPermanent(AnyPermanent)],ActionList([CreatePermanentLayerEffect(ThisPermanent,[IsACopyOfPermanent(Ref_TargetPermanent,CopyEffects([KeepColor]))])])) )])`), "inventory.ron")
	if err != nil {
		t.Fatal(err)
	}
	entries := map[string]*InventorySignature{}
	walkInventory(nodes[0], "Inventory", "Card", "card", "collection", true, entries)
	for name, role := range map[string]string{"IsACopyOfPermanent": "resolving_modification", "Ref_TargetPermanent": "permanent_reference", "CopyEffects": "copy_modification", "KeepColor": "copy_modification"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatal("missing inventory operand", name, role)
		}
	}
}
