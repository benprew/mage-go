package ron

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextSpellCastSpellRecords(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
	}{
		{"LEA", "Crystal Rod", core.Blue}, {"LEA", "Iron Star", core.Red},
		{"LEA", "Ivory Cup", core.White}, {"LEA", "Throne of Bone", core.Black},
		{"LEA", "Wooden Sphere", core.Green}, {"ATQ", "Citanul Druid", core.Colorless},
		{"ATQ", "Urza's Chalice", core.Colorless}, {"LEG", "In the Eye of Chaos", core.Colorless},
		{"LEG", "Nether Void", core.Colorless}, {"LEG", "Presence of the Master", core.Colorless},
	} {
		sets := []string{tc.set}
		if tc.set == "LEA" {
			sets = []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"}
		}
		for _, set := range sets {
			t.Run(set+"/"+tc.name, func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
				b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				card := repositoryCard(t, set, tc.name)
				card.SetOwner(a.PlayerID())
				own := castRONSpell(t, g, a.BasePlayer, card)
				paid := g.GetPlayer(a.PlayerID()).ManaPool().TotalMana()
				change, _ := mage.ReplaceColorWord(tc.color, core.Green)
				revisedColor := core.Green
				if tc.color == core.Green {
					revisedColor = core.Blue
					change, _ = mage.ReplaceColorWord(tc.color, revisedColor)
				}
				if tc.color == core.Colorless {
					change, _ = mage.ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
				}
				if err := g.RevisePrintedSpellText(own.ID, change); err != nil {
					t.Fatal(err)
				}
				if g.StackPeek().Card != card || g.FindPermanent(card.ID()) != nil || a.offers != 0 || a.ManaPool().TotalMana() != paid {
					t.Fatal("revision changed announcement or payment")
				}
				g.ResolveStack()
				p := g.FindPermanent(card.ID())
				if p == nil || p.Card.Owner() != a.PlayerID() || p.ControllerID() != a.PlayerID() {
					t.Fatal("entry lost identity")
				}
				g.SetActivePlayerIndex(1)
				victimCard := textTriggerSpell(revisedColor)
				switch tc.name {
				case "Citanul Druid", "Urza's Chalice":
					victimCard = repositoryCard(t, "LEA", "Sol Ring")
				case "Presence of the Master":
					victimCard = repositoryCard(t, "LEA", "Crusade")
				}
				victim := castRONSpell(t, g, b.BasePlayer, victimCard)
				g.PutTriggersOnStack()
				if g.StackSize() != 2 {
					t.Fatal("continued declaration did not detect event")
				}
				a.ManaPool().Clear()
				a.ManaPool().Add(core.Colorless, 1)
				beforeB := b.ManaPool().TotalMana()
				g.ResolveTopOfStack()
				switch tc.name {
				case "Citanul Druid":
					if g.FindPermanent(p.ID()).Counters[core.P1P1] != 1 {
						t.Fatal("continued counter action failed")
					}
				case "Presence of the Master", "Nether Void", "In the Eye of Chaos":
					if g.GetStack().FindByID(victim.ID) != nil {
						t.Fatal("event spell binding lost")
					}
					if tc.name != "Presence of the Master" && (b.offers != 1 || a.offers != 0 || b.ManaPool().TotalMana() != beforeB) {
						t.Fatal("caster payment or refusal changed")
					}
				default:
					if a.Life() != 21 || a.offers != 1 || b.offers != 0 || a.ManaPool().TotalMana() != 0 {
						t.Fatal("continued payment binding failed")
					}
				}
				g.ResolveStack()
				if tc.color != core.Colorless {
					castRONSpell(t, g, b.BasePlayer, textTriggerSpell(tc.color))
					g.PutTriggersOnStack()
					if g.StackSize() != 1 {
						t.Fatal("old color still triggered")
					}
					g.ResolveStack()
					g.BouncePermanentToHand(g.FindPermanent(card.ID()))
					if a.Hand()[0] != card || g.LKI(card.ID()) == nil {
						t.Fatal("departure lost printed card or LKI")
					}
					g.SetActivePlayerIndex(0)
					castRONSpell(t, g, a.BasePlayer, card)
					g.ResolveStack()
					castRONSpell(t, g, b.BasePlayer, textTriggerSpell(tc.color))
					g.PutTriggersOnStack()
					if g.StackSize() != 2 {
						t.Fatal("recast lost printed rule")
					}
				}
			})
		}
	}
}

func TestRONTextSpellCastSpellInventory(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]int{"Crystal Rod": 6, "Iron Star": 6, "Ivory Cup": 6, "Throne of Bone": 6, "Wooden Sphere": 6, "Citanul Druid": 1, "Urza's Chalice": 1, "In the Eye of Chaos": 1, "Nether Void": 1, "Presence of the Master": 1}
	counts := map[string]int{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		records, err := Parse(strings.NewReader(string(data)), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, record := range records {
			if !spellCastSpellTextRules(record.Fields["Rules"].Args) {
				continue
			}
			if _, err := Compile(record); err != nil {
				if record.Fields["Name"].String == "Invoke Prejudice" {
					continue
				}
				t.Fatal(err)
			}
			counts[record.Fields["Name"].String]++
		}
	}
	if !maps.Equal(counts, expected) {
		t.Fatalf("unmapped inventory: %v", counts)
	}
}

func TestRONTextSpellCastSpellLifecycle(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	card := repositoryCard(t, "LEA", "Throne of Bone")
	obj := castRONSpell(t, g, a.BasePlayer, card)
	first, _ := mage.ReplaceColorWord(core.Black, core.Blue)
	second, _ := mage.ReplaceColorWord(core.Blue, core.Green)
	third, _ := mage.ReplaceColorWord(core.Green, core.Red)
	if err := g.RevisePrintedSpellText(obj.ID, first); err != nil {
		t.Fatal(err)
	}
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("spell copy failed")
	}
	if err := g.RevisePrintedSpellText(copied.ID, second); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := clone.RevisePrintedSpellText(copied.ID, third); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	clone.ResolveStack()
	token := g.FindPermanent(copied.SourceID)
	if token == nil || !token.Card.IsToken() || token.ControllerID() != b.PlayerID() {
		t.Fatal("copy entry lost token identity")
	}
	castRONSpell(t, g, a.BasePlayer, textTriggerSpell(core.Green))
	g.PutTriggersOnStack()
	castRONSpell(t, clone, clone.GetPlayer(a.PlayerID()).(*mage.SearchPlayer).BasePlayer, textTriggerSpell(core.Green))
	clone.PutTriggersOnStack()
	if g.StackSize() != 2 || clone.StackSize() != 1 {
		t.Fatal("stack clones shared continued declarations")
	}
	b.ManaPool().Clear()
	b.ManaPool().Add(core.Colorless, 1)
	g.ResolveStack()
	clone.ResolveStack()
	if b.Life() != 21 || a.Life() != 20 {
		t.Fatal("copy payment used wrong controller")
	}
	values, err := g.CaptureCopiableValues(token)
	if err != nil {
		t.Fatal(err)
	}
	printed, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	sibling := g.PutOnBattlefield(printed, a.PlayerID())
	castRONSpell(t, g, b.BasePlayer, textTriggerSpell(core.Black))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("copiable values retained ordinary text change")
	}
	a.ManaPool().Clear()
	a.ManaPool().Add(core.Colorless, 1)
	g.ResolveStack()
	g.RemoveFromBattlefield(g.FindPermanent(sibling.ID()))
	g.RemoveFromBattlefield(g.FindPermanent(token.ID()))
	p := g.FindPermanent(card.ID())
	id := p.RuntimeAbilities[0].AbilityID()
	castRONSpell(t, g, b.BasePlayer, textTriggerSpell(core.Blue))
	g.PutTriggersOnStack()
	if err := g.AddPrintedSpellCastTextEffect(p, id, core.EndOfTurn, second); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
	}
	a.ManaPool().Clear()
	a.ManaPool().Add(core.Colorless, 1)
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("pending trigger lost detection-time body")
	}
	castRONSpell(t, g, b.BasePlayer, textTriggerSpell(core.Green))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("later layer did not use continued baseline")
	}
	a.accept = false
	a.ManaPool().Add(core.Colorless, 1)
	g.ResolveStack()
	if a.Life() != 22 || a.ManaPool().TotalMana() != 1 {
		t.Fatal("decline consumed payment")
	}
	g.AddContinuousEffect(mage.RemoveAllAbilities(card.ID(), core.EndOfTurn))
	castRONSpell(t, g, b.BasePlayer, textTriggerSpell(core.Green))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("removed declaration detected event")
	}
	g.ResolveStack()
	phased := g.PhaseOut(card.ID())
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	p = g.FindPermanent(card.ID())
	if p.RuntimeAbilities[0].AbilityID() != id {
		t.Fatal("restoration lost declaration identity")
	}
	castRONSpell(t, g, b.BasePlayer, textTriggerSpell(core.Blue))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("expiry lost continued baseline")
	}
	g.ResolveStack()
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	castRONSpell(t, g, a.BasePlayer, textTriggerSpell(core.Blue))
	g.PutTriggersOnStack()
	b.ManaPool().Clear()
	b.ManaPool().Add(core.Colorless, 1)
	g.BouncePermanentToHand(g.FindPermanent(card.ID()))
	g.ResolveStack()
	if b.Life() != 22 || a.Life() != 22 {
		t.Fatal("pending departure payment lost detection controller")
	}
	lki := g.LKI(card.ID())
	if lki == nil {
		t.Fatal("missing LKI")
	}
	lkiSource := g.PutOnBattlefield(lki.Snapshot.Card.Copy(), a.PlayerID())
	castRONSpell(t, g, b.BasePlayer, textTriggerSpell(core.Blue))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("LKI lost continued rules")
	}
	g.ResolveStack()
	g.RemoveFromBattlefield(g.FindPermanent(lkiSource.ID()))
	if err := g.AddPrintedSpellCastTextEffect(p, id, core.EndOfTurn, second); err == nil {
		t.Fatal("departed visit accepted")
	}
}

func TestRONTextSpellCastSpellBoundaries(t *testing.T) {
	for _, rule := range []string{
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer(),IsColor(Red)),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer,IsColor(Red)),ActionList([PlayerAction(Trigger_ThatSpell,GainLife(Integer(1)))]))`,
		`TriggerA(WhenAPlayerCastsASpell(AnyPlayer,IsColor(Red)),ActionList([If(CostWasPaid,[GainLife(Integer(1))])]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", rule)
		}
	}
	for _, tc := range []struct{ set, name string }{{"LEG", "Sol'kanar the Swamp King"}} {
		g, a, _ := ronGame()
		card := repositoryCard(t, tc.set, tc.name)
		obj := castRONSpell(t, g, a, card)
		change, _ := mage.ReplaceColorWord(core.Black, core.Blue)
		if err := g.RevisePrintedSpellText(obj.ID, change); err == nil || g.StackPeek() != obj {
			t.Fatal("unsupported context mutated", tc.name)
		}
	}
}

func TestRONTextSpellCastSpellEventBindingsAndTargets(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerCastsASpell(AnyPlayer,IsColor(Red)),Targeted([TargetPermanent(IsColor(Red))],ActionList([DestroyPermanent(Ref_TargetPermanent),PlayerMayCost(Trigger_ThatPlayer,PayMana([ManaCostGeneric(1)])),Unless(CostWasPaid,[CounterSpell(Trigger_ThatSpell)])])))`)
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	source := factory()
	obj := castRONSpell(t, g, a.BasePlayer, source)
	change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.RevisePrintedSpellText(obj.ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	target := g.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
	first := castRONSpell(t, g, a.BasePlayer, textTriggerSpell(core.Blue))
	g.PutTriggersOnStack()
	pending := g.StackPeek()
	if len(pending.Targets) != 1 || pending.Targets[0] != target.ID() {
		t.Fatal("fresh trigger target did not use revised filter")
	}
	a.ManaPool().Clear()
	a.ManaPool().Add(core.Colorless, 1)
	g.ResolveTopOfStack()
	if g.FindPermanent(target.ID()) != nil || g.GetStack().FindByID(first.ID) == nil || a.offers != 1 || b.offers != 0 {
		t.Fatal("fresh event bindings lost payer or spell")
	}
	g.ResolveStack()
	target = g.PutOnBattlefield(textColorCard(core.Blue), a.PlayerID())
	second := castRONSpell(t, g, b.BasePlayer, textTriggerSpell(core.Blue))
	g.PutTriggersOnStack()
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.Red)), target.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(target.ID()) == nil || g.GetStack().FindByID(second.ID) == nil || b.offers != 0 {
		t.Fatal("illegal target did not stop body")
	}
}

func TestRONTextSpellCastSpellIndependentPayments(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	card := repositoryCard(t, "LEG", "Nether Void")
	obj := castRONSpell(t, g, a.BasePlayer, card)
	change, _ := mage.ReplaceColorWord(core.Black, core.Blue)
	if err := g.RevisePrintedSpellText(obj.ID, change); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	first := castRONSpell(t, g, a.BasePlayer, textTriggerSpell(core.Red))
	g.PutTriggersOnStack()
	second := castRONSpell(t, g, b.BasePlayer, textTriggerSpell(core.Green))
	g.PutTriggersOnStack()
	if g.StackSize() != 4 || a.offers != 0 || b.offers != 0 {
		t.Fatal("event or payment timing changed")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	clone := g.Clone()
	cloneA := &ronPaymentPlayer{BasePlayer: clone.GetPlayer(a.PlayerID()).(*mage.SearchPlayer).BasePlayer, accept: false}
	cloneB := &ronPaymentPlayer{BasePlayer: clone.GetPlayer(b.PlayerID()).(*mage.SearchPlayer).BasePlayer, accept: true}
	clone.SetPlayerAt(0, cloneA)
	clone.SetPlayerAt(1, cloneB)
	for _, payer := range []*ronPaymentPlayer{a, b, cloneA, cloneB} {
		payer.ManaPool().Clear()
		payer.ManaPool().Add(core.Colorless, 6)
	}
	g.ResolveTopOfStack()
	g.ResolveTopOfStack()
	if g.GetStack().FindByID(second.ID) != nil || g.GetStack().FindByID(first.ID) == nil || b.offers != 2 || a.offers != 0 || b.ManaPool().TotalMana() != 6 {
		t.Fatal("pending captures or copied refusals shared state")
	}
	g.ResolveTopOfStack()
	if g.GetStack().FindByID(first.ID) == nil || a.offers != 1 || a.ManaPool().TotalMana() != 3 {
		t.Fatal("earlier event payment lost its binding")
	}
	clone.ResolveTopOfStack()
	clone.ResolveTopOfStack()
	if clone.GetStack().FindByID(second.ID) == nil || cloneB.offers != 2 || cloneB.ManaPool().TotalMana() != 0 || cloneA.offers != 0 {
		t.Fatal("clone paid paths lost captures")
	}
	clone.ResolveTopOfStack()
	clone.ResolveTopOfStack()
	if clone.GetStack().FindByID(first.ID) != nil || cloneA.offers != 1 || cloneA.ManaPool().TotalMana() != 6 {
		t.Fatal("clone refusal shared original payment result")
	}
}

func TestRONTextSpellCastSpellCopyExceptions(t *testing.T) {
	g, a, _ := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEA", "Crystal Rod"), a.PlayerID())
	values, err := g.CaptureCopiableValues(source)
	if err != nil {
		t.Fatal(err)
	}
	modified, err := values.WithModifications(mage.CopyAddCardTypes(core.TypeLand))
	if err != nil {
		t.Fatal(err)
	}
	card, err := modified.Build()
	if err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceColorWord(core.Blue, core.Green)
	revision, err := mage.BuildTextDefinitionRevision(card, change)
	if err != nil {
		t.Fatal(err)
	}
	if !revision.Definition().HasType(core.TypeLand) {
		t.Fatal("copy lost added type")
	}
	g.PutOnBattlefield(revision.Definition(), a.PlayerID())
	castRONSpell(t, g, a, textTriggerSpell(core.Green))
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("copied Land lost its trigger rules")
	}
}
