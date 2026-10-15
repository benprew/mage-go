package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCopyAbilityRetention(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			factory, err := Compile(repositoryRecord(t, set, "Vesuvan Doppelganger", set))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			bear := g.PutOnBattlefield(repositoryCard(t, "LEA", "Grizzly Bears"), b.PlayerID())
			air := g.PutOnBattlefield(repositoryCard(t, "LEA", "Air Elemental"), b.PlayerID())
			chooser := &copySpellTargetPlayer{ronEntryCopyPlayer: ronEntryCopyPlayer{Player: a, accept: true}, target: air.ID()}
			chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return bear }
			g.SetPlayerAt(0, chooser)
			card := factory()
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 2)
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			for _, target := range []*mage.Permanent{air, bear, air} {
				chooser.target = target.ID()
				queueRONUpkeep(g, a)
				if g.StackSize() != 1 {
					t.Fatalf("upkeep triggers=%d", g.StackSize())
				}
				g.ResolveStack()
				p := g.FindPermanent(card.ID())
				if p.Name() != target.Name() || len(p.Colors()) != 1 || p.Colors()[0] != core.Blue {
					t.Fatal("copy or retained color failed")
				}
			}
			chooser.target = card.ID()
			queueRONUpkeep(g, a)
			g.ResolveStack()
			queueRONUpkeep(g, a)
			if g.StackSize() != 2 {
				t.Fatalf("self copy must add an independent ability: %d", g.StackSize())
			}
			chooser.accept = false
			g.ResolveStack()
		})
	}
}

func TestRONCopyAbilityRetentionLifecycle(t *testing.T) {
	for _, change := range []string{"decline", "illegal", "target return", "target phase", "phase cycle", "source return", "source phase", "control", "stack copy", "clone", "other doppelganger", "face down", "ability removed"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			bear := g.PutOnBattlefield(repositoryCard(t, "LEA", "Grizzly Bears"), b.PlayerID())
			air := g.PutOnBattlefield(repositoryCard(t, "LEA", "Air Elemental"), b.PlayerID())
			chooser := &copySpellTargetPlayer{ronEntryCopyPlayer: ronEntryCopyPlayer{Player: a, accept: true}, target: air.ID()}
			chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return bear }
			g.SetPlayerAt(0, chooser)
			p := g.PutOnBattlefield(repositoryCard(t, "LEA", "Vesuvan Doppelganger"), a.PlayerID())
			if change == "other doppelganger" {
				otherChooser := &ronEntryCopyPlayer{Player: b, accept: true}
				otherChooser.pick = chooser.pick
				g.SetPlayerAt(1, otherChooser)
				other := g.PutOnBattlefield(repositoryCard(t, "LEA", "Vesuvan Doppelganger"), b.PlayerID())
				if other == nil {
					t.Fatal("second copy entry failed")
				}
				chooser.target = other.ID()
			}
			queueRONUpkeep(g, a)
			if g.StackSize() != 1 || chooser.offers != 1 {
				t.Fatal("target announcement must precede the decision")
			}
			wantName := "Air Elemental"
			wantAbilities := 1
			switch change {
			case "decline":
				chooser.accept = false
				wantName = "Grizzly Bears"
			case "illegal":
				air.GrantBaseAttr(core.Shroud)
				wantName = "Grizzly Bears"
			case "target return":
				g.BouncePermanentToHand(air)
				card := b.Hand()[0]
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
				wantName = "Grizzly Bears"
			case "target phase":
				g.PhaseOut(air.ID())
				wantName = "Grizzly Bears"
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
				g.PhaseIn(g.PhaseOut(air.ID()))
			case "source return":
				g.BouncePermanentToHand(p)
				card := a.Hand()[0]
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
				wantName = "Grizzly Bears"
			case "source phase":
				g.PhaseOut(p.ID())
				wantName = "Grizzly Bears"
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: bear.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "stack copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("stack copy")
				}
			case "clone":
				clone := g.Clone()
				clone.SetPlayerAt(0, &ronEntryCopyPlayer{Player: clone.GetPlayer(a.PlayerID()), accept: false})
				clone.ResolveStack()
				if clone.FindPermanent(p.ID()).Name() != "Grizzly Bears" {
					t.Fatal("clone decision")
				}
				queueRONUpkeep(clone, clone.GetPlayer(a.PlayerID()))
				if clone.StackSize() != 1 {
					t.Fatal("clone lost its independent trigger")
				}
			case "other doppelganger":
				wantName = "Grizzly Bears"
				wantAbilities = 2
			case "face down":
				air.FaceDown = true
				wantName = ""
			case "ability removed":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				wantAbilities = 0
			}
			g.ResolveStack()
			p = g.FindPermanentIncludingPhased(p.ID())
			if p.Name() != wantName || len(p.RuntimeAbilities) != wantAbilities {
				t.Fatalf("name=%q abilities=%d, want %q/%d", p.Name(), len(p.RuntimeAbilities), wantName, wantAbilities)
			}
			if change == "source phase" {
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			if change == "ability removed" {
				return
			}
			chooser.accept = false
			chooser.target = bear.ID()
			if change == "control" {
				queueRONUpkeep(g, a)
				if g.StackSize() != 0 {
					t.Fatal("stale trigger controller")
				}
				queueRONUpkeep(g, b)
			} else {
				queueRONUpkeep(g, a)
			}
			if g.StackSize() != wantAbilities {
				t.Fatalf("next upkeep has %d triggers, want %d", g.StackSize(), wantAbilities)
			}
			g.ResolveStack()
		})
	}
}

func TestRONCopyAbilityRetentionIndependentDeclarations(t *testing.T) {
	g, a, b := ronGame()
	bear := g.PutOnBattlefield(repositoryCard(t, "LEA", "Grizzly Bears"), b.PlayerID())
	chooser := &copySpellTargetPlayer{ronEntryCopyPlayer: ronEntryCopyPlayer{Player: a, accept: true}, target: bear.ID()}
	chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return bear }
	g.SetPlayerAt(0, chooser)
	addition := func(n string) string {
		return `AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([MayAction(CreatePermanentLayerEffect(ThisPermanent,[IsACopyOfPermanent(Ref_TargetPermanent,CopyEffects([KeepColor,AddAbilityVariable(ThisAbility)]))])),GainLife(Integer(` + n + `))])))])`
	}
	factory := copyColorFactory(t, `[ManaCostU]`, "", `KeepColor,`+addition("1")+`,`+addition("2"))
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	queueRONUpkeep(g, a)
	if g.StackSize() != 2 {
		t.Fatal("independent additions")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || a.Life() != 20 {
		t.Fatal("pending bindings escaped clone")
	}
	g.ResolveTopOfStack()
	firstGain := a.Life() - 20
	g.ResolveStack()
	if a.Life() != 23 || len(g.FindPermanent(p.ID()).RuntimeAbilities) != 1 {
		t.Fatal("pending declarations read current rules")
	}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if a.Life() != 23+(3-firstGain) {
		t.Fatal("retained declaration changed its body")
	}
	delta := a.Life() - 23
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if a.Life() != 23+2*delta {
		t.Fatal("retained declaration switched between lexical instances")
	}
}

func TestRONCopyAbilityRetentionInvalid(t *testing.T) {
	body := func(mod string) string {
		return `Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([MayAction(CreatePermanentLayerEffect(ThisPermanent,[IsACopyOfPermanent(Ref_TargetPermanent,CopyEffects([` + mod + `]))]))]))`
	}
	for _, mod := range []string{
		`AddAbilityVariable`, `AddAbilityVariable()`, `AddAbilityVariable(ThisAbility())`,
		`AddAbilityVariable(You)`, `AddAbilityVariable(TheChosenAbility)`,
		`AddAbilityVariable(ThisAbility,ThisAbility)`,
		`AddAbilityVariable(ThisAbility),AddAbilityVariable(ThisAbility)`,
		`KeepColor,KeepColor,AddAbilityVariable(ThisAbility)`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[AsPermanentEnters(ThisPermanent,[MayActions([EnterAsACopyOfAPermanent(IsCardtype(Creature),CopyEffects([AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),` + body(mod) + `)])]))])])])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid retention accepted", mod)
		}
	}
	assertConditionalRejected(t, `Activated(PayMana([]),`+body(`AddAbilityVariable(ThisAbility)`)+`)`)
}

func TestRONCopyAbilityRetentionShapes(t *testing.T) {
	for _, optional := range []bool{false, true} {
		for _, keep := range []bool{false, true} {
			g, a, b := ronGame()
			bear := g.PutOnBattlefield(repositoryCard(t, "LEA", "Grizzly Bears"), b.PlayerID())
			chooser := &copySpellTargetPlayer{ronEntryCopyPlayer: ronEntryCopyPlayer{Player: a, accept: true}, target: bear.ID()}
			chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return bear }
			g.SetPlayerAt(0, chooser)
			mods := `AddAbilityVariable(ThisAbility)`
			if keep {
				mods = `KeepColor,` + mods
			}
			action := `CreatePermanentLayerEffect(ThisPermanent,[IsACopyOfPermanent(Ref_TargetPermanent,CopyEffects([` + mods + `]))])`
			if optional {
				action = `MayAction(` + action + `)`
			}
			factory := copyUpkeepFactory(t, `Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+action+`]))`)
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			queueRONUpkeep(g, a)
			g.ResolveStack()
			p = g.FindPermanent(p.ID())
			color := core.Green
			if keep {
				color = core.Blue
			}
			if len(p.Colors()) != 1 || p.Colors()[0] != color || len(p.RuntimeAbilities) != 1 {
				t.Fatal("retention shape or colors")
			}
			queueRONUpkeep(g, a)
			if g.StackSize() != 1 {
				t.Fatal("retained shape lost its trigger")
			}
			g.ResolveStack()
		}
	}
}

func TestRONCopyAbilityRetentionInventory(t *testing.T) {
	record := repositoryRecord(t, "LEA", "Vesuvan Doppelganger", "inventory")
	entries := map[string]*InventorySignature{}
	walkInventory(record, "Vesuvan Doppelganger", "Card", "card", "collection", true, entries)
	for name, role := range map[string]string{"ThisAbility": "ability_reference", "AddAbilityVariable": "copy_modification"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatal("missing inventory role", name, role)
		}
	}
}
