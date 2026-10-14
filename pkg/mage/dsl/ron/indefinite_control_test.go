package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONIndefiniteControlOgre(t *testing.T) {
	for _, change := range []string{"none", "tie-detection", "tie-resolution", "new-leader", "opponent-upkeep", "phase", "phase-cycle", "return", "copy", "abilities"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ARN", "Ghazbán Ogre")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			a.ManaPool().Add(core.Green, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if p == nil || p.Card.Power() != 2 || p.Card.Toughness() != 2 {
				t.Fatal("creature metadata")
			}
			if change != "tie-detection" {
				b.GainLife(2)
			}
			player := mage.Player(a)
			if change == "opponent-upkeep" {
				player = b
			}
			queueRONUpkeep(g, player)
			if change == "tie-detection" || change == "opponent-upkeep" {
				if g.StackSize() != 0 {
					t.Fatal("incorrect detection")
				}
				return
			}
			if g.StackSize() != 1 {
				t.Fatal("missing trigger")
			}
			switch change {
			case "tie-resolution":
				a.GainLife(2)
			case "new-leader":
				a.GainLife(3)
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "abilities":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
			}
			clone := g.Clone()
			clone.GetPlayer(a.PlayerID()).GainLife(10)
			g.ResolveStack()
			clone.ResolveStack()
			want := b.PlayerID()
			if change == "tie-resolution" || change == "new-leader" || change == "phase" || change == "return" {
				want = a.PlayerID()
			}
			if g.FindPermanentIncludingPhased(card.ID()).ControllerID() != want {
				t.Fatal("wrong controller", change)
			}
			if clone.FindPermanentIncludingPhased(card.ID()).ControllerID() != a.PlayerID() {
				t.Fatal("clone resolution shared life or control")
			}
			b.GainLife(10)
			a.GainLife(30)
			g.ApplyContinuousEffects()
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if g.FindPermanentIncludingPhased(card.ID()).ControllerID() != want {
				t.Fatal("controller was not saved indefinitely")
			}
		})
	}
}

func TestRONIndefiniteControlRainbowVale(t *testing.T) {
	for _, change := range []string{"none", "phase", "phase-cycle", "return", "control", "abilities", "two", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "FEM", "Rainbow Vale")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			if !mage.IsActivatedManaAbility(p.RuntimeAbilities[0]) {
				t.Fatal("not a mana ability")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
				t.Fatal("mana timing")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
				t.Fatal("tapped payment accepted")
			}
			switch change {
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "abilities":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
			case "two":
				g.UntapPermanent(g.FindPermanent(p.ID()))
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			wantStack := 1
			if change == "two" {
				wantStack = 2
			}
			if g.StackSize() != wantStack {
				t.Fatal("delayed trigger count", g.StackSize())
			}
			if change == "copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			if change == "copy" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			clone := g.Clone()
			if change == "copy" {
				g.ResolveTopOfStack()
				clone.ResolveTopOfStack()
				if g.FindPermanent(p.ID()).ControllerID() != a.PlayerID() || clone.FindPermanent(p.ID()).ControllerID() != a.PlayerID() {
					t.Fatal("copied delayed controller")
				}
			}
			g.ResolveStack()
			clone.ResolveStack()
			want := b.PlayerID()
			if change == "phase" || change == "return" {
				want = a.PlayerID()
			}
			if g.FindPermanentIncludingPhased(p.ID()).ControllerID() != want || clone.FindPermanentIncludingPhased(p.ID()).ControllerID() != want {
				t.Fatal("saved source or controller")
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if g.FindPermanentIncludingPhased(p.ID()).ControllerID() != want {
				t.Fatal("control expired")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("delayed trigger repeated")
			}
		})
	}
}

func TestRONIndefiniteControlInvalid(t *testing.T) {
	for _, body := range []string{
		`CreatePermanentLayerEffect(ThisPermanent,[SetController])`,
		`CreatePermanentLayerEffect(ThisPermanent,[SetController()])`,
		`CreatePermanentLayerEffect(ThisPermanent,[SetController(You,Opponent)])`,
		`CreatePermanentLayerEffect(ThisPermanent,[SetController(ThePlayerWithTheMostLife())])`,
		`CreatePermanentLayerEffect(ThisPermanent,[SetController(TheChosenPlayer)])`,
		`CreatePermanentLayerEffect(ThisPermanent,[SetController(Unknown)])`,
		`CreatePermanentLayerEffect(ThisPermanent,[SetController(You),SetColor(SimpleColorList([Red]))])`,
		`GainControlOfPermanent()`, `GainControlOfPermanent(ThisPermanent,ThisPermanent)`,
		`GainControlOfPermanent(Ref_TargetPlayer)`, `GainControlOfPermanent(Ref_TargetPermanent)`,
		`PlayerAction(TheChosenPlayer,GainControlOfPermanent(ThisPermanent))`,
	} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + body + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", body)
		}
	}
	for _, rule := range []string{
		`AsPermanentEnters(ThisPermanent,[GainControlOfPermanent(ThisPermanent)])`,
		`PermanentLayerEffect(ThisPermanent,[SetController(ThePlayerWithTheMostLife)])`,
	} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rule + `])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONIndefiniteControlRainbowPayment(t *testing.T) {
	for _, color := range []string{"W", "U", "B", "R", "G", "U}{U"} {
		t.Run(color, func(t *testing.T) {
			g, a, b := ronGame()
			vale := g.PutOnBattlefield(repositoryCard(t, "FEM", "Rainbow Vale"), a.PlayerID())
			spell := mage.NewArtifact("Payment fixture", "{"+color+"}")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0)
			if color == "U}{U" {
				if err == nil || g.FindPermanent(vale.ID()).Tapped || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 || len(a.Hand()) != 1 {
					t.Fatal("payment rollback", err)
				}
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("failed payment installed a delayed trigger")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(vale.ID()).Tapped || a.ManaPool().TotalMana() != 0 || g.StackSize() != 1 {
				t.Fatal("automatic mana payment")
			}
			g.ResolveStack()
			refs := g.WaitingDelayedPermanentSources()
			if len(refs) != 1 || refs[0].ID() != vale.ID() {
				t.Fatal("waiting source query", refs)
			}
			if g.FindPermanent(vale.ID()).ControllerID() != a.PlayerID() || g.StackSize() != 0 {
				t.Fatal("query changed control")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if g.FindPermanent(vale.ID()).ControllerID() != b.PlayerID() {
				t.Fatal("automatic mana rider missing")
			}
		})
	}
}

func TestRONIndefiniteControlInventoryAndLoad(t *testing.T) {
	files := fstest.MapFS{}
	for _, entry := range []struct{ set, name string }{{"ARN", "Ghazbán Ogre"}, {"FEM", "Rainbow Vale"}} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + entry.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "`+entry.name+`"`) {
				files[entry.set+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Records != 2 || report.Compiled != 2 || len(report.Issues) != 0 {
		t.Fatal("complete records", report)
	}
	roles := map[string]bool{}
	for _, signature := range report.Signatures {
		roles[signature.Role+"/"+signature.Name] = true
	}
	for _, key := range []string{"resolving_modification/SetController", "player_reference/ThePlayerWithTheMostLife", "player_action/GainControlOfPermanent", "permanent_reference/ThisPermanent"} {
		if !roles[key] {
			t.Fatal("missing inventory role", key)
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Ghazbán Ogre", "Rainbow Vale"} {
		first, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		second, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		if first.ID() == second.ID() || first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
			t.Fatal("shared instance")
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([GainControlOfPermanent(Ref_TargetPermanent)]))])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Rainbow Vale") || registry.CardRegistered("Bad") {
		t.Fatal("atomic load", err)
	}
}

func TestRONIndefiniteControlOgreNewController(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "ARN", "Ghazbán Ogre"), a.PlayerID())
	b.GainLife(2)
	queueRONUpkeep(g, a)
	g.ResolveStack()
	a.GainLife(5)
	queueRONUpkeep(g, a)
	if g.StackSize() != 0 {
		t.Fatal("old controller still supplies upkeep")
	}
	queueRONUpkeep(g, b)
	if g.StackSize() != 1 {
		t.Fatal("new controller upkeep missing")
	}
	g.ResolveStack()
	if g.FindPermanent(p.ID()).ControllerID() != a.PlayerID() {
		t.Fatal("subsequent control change")
	}
}
