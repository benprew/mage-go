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

func TestRONOriginalPrintingNamesSylex(t *testing.T) {
	factory, err := Compile(staticBaseRecord(t, "ATQ", "Golgothian Sylex"))
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"normal", "no-mana", "control", "phased", "cycle", "returned", "face-down", "source-left", "source-phased", "clone", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			card := mage.NewCreature("Ornithopter", "", 1, 1)
			card.SetOwner(a.PlayerID())
			subject := g.PutOnBattlefield(card, b.PlayerID())
			unrelated := g.PutOnBattlefield(mage.NewArtifact("Black Lotus", ""), a.PlayerID())
			token := g.PutOnBattlefield(mage.NewToken("Ornithopter", 1, 1, []core.CardType{core.TypeCreature}, nil), b.PlayerID())
			hidden := g.PutOnBattlefield(mage.NewCreature("Atog", "", 2, 2), b.PlayerID())
			g.MutablePermanent(hidden.ID()).FaceDown = true
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			if err == nil || g.StackSize() != 0 || g.FindPermanent(source.ID()).Tapped {
				t.Fatal("failed payment changed state")
			}
			if state == "no-mana" {
				return
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || !g.FindPermanent(source.ID()).Tapped || g.FindPermanent(subject.ID()) == nil {
				t.Fatal("payment or resolution timing")
			}
			late := g.PutOnBattlefield(mage.NewLand("Mishra's Factory"), a.PlayerID())
			switch state {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: subject.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "phased":
				g.PhaseOut(subject.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(subject.ID()))
			case "returned":
				g.BouncePermanentToHand(subject)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "face-down":
				g.MutablePermanent(subject.ID()).FaceDown = true
			case "source-left":
				g.BouncePermanentToHand(source)
			case "source-phased":
				g.PhaseOut(source.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			if state == "clone" {
				clone.PhaseOut(subject.ID())
			}
			g.ResolveStack()
			remains := state == "phased" || state == "face-down"
			if (g.FindPermanentIncludingPhased(subject.ID()) != nil) != remains || g.FindPermanent(late.ID()) != nil {
				t.Fatal("resolution did not select current names")
			}
			if g.FindPermanent(unrelated.ID()) == nil || g.FindPermanent(token.ID()) == nil || g.FindPermanent(hidden.ID()) == nil {
				t.Fatal("unlisted, token, or nameless subject moved")
			}
			if (g.FindPermanentIncludingPhased(source.ID()) != nil) != (state == "source-phased") {
				t.Fatal("source sacrifice")
			}
			if clone.FindPermanentIncludingPhased(subject.ID()) == nil || clone.FindPermanent(late.ID()) == nil {
				t.Fatal("original changed clone")
			}
			clone.ResolveStack()
			if (clone.FindPermanentIncludingPhased(subject.ID()) != nil) != (remains || state == "clone") {
				t.Fatal("clone selection changed")
			}
			found := false
			for _, c := range a.Graveyard() {
				if c.ID() == card.ID() {
					found = true
				}
			}
			if found == remains {
				t.Fatal("sacrifice did not use current controller and owner")
			}
		})
	}
}

func TestRONOriginalPrintingNamesBottleFilter(t *testing.T) {
	record := staticBaseRecord(t, "ARN", "City in a Bottle")
	filter := record.Fields["Rules"].Args[0].Args[0].Args[0].Args[0].Args[1]
	rule := nameScopeNode(t, `Activated(PayMana([]),ActionList([ControllersSacrificeEachPermanent(AnyPermanent)]))`)
	rule.Args[1].Args[0].Args[0].Args[0] = filter
	factory := staticBaseSubrule(t, record, rule)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	victim := g.PutOnBattlefield(mage.NewArtifact("Aladdin's Ring", ""), b.PlayerID())
	land := g.PutOnBattlefield(mage.NewLand("Mountain"), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(victim.ID()) != nil || g.FindPermanent(land.ID()) == nil || g.FindPermanent(source.ID()) != nil {
		t.Fatal("Arabian names use set membership or omit self")
	}
	if _, err := Compile(staticBaseRecord(t, "ARN", "City in a Bottle")); err != nil {
		t.Fatal("complete state trigger did not compile", err)
	}
}

func TestRONOriginalPrintingNamesDeclarations(t *testing.T) {
	for _, name := range []string{"SharesANameOriginallyPrintedInAntiquities", "SharesANameOriginallyPrintedInArabianNights"} {
		for _, filter := range []string{name + `()`, name + `(AnyPermanent)`, name + `(Integer(1),Integer(2))`} {
			text := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([TapEachPermanent(` + filter + `)]))])`
			if registry, err := LoadFS(fstest.MapFS{"a.ron": {Data: []byte(text)}}, "*.ron"); err == nil || registry != nil {
				t.Fatal("invalid declaration published", filter)
			}
		}
		for _, rule := range []string{`Activated(%s,ActionList([]))`, `Activated(PayMana([]),ActionList([%s]))`} {
			text := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + fmt.Sprintf(rule, name) + `])`
			if registry, err := LoadFS(fstest.MapFS{"a.ron": {Data: []byte(text)}}, "*.ron"); err == nil || registry != nil {
				t.Fatal("invalid role published", rule)
			}
		}
	}
}

func TestRONOriginalPrintingNamesTargets(t *testing.T) {
	for _, state := range []string{"normal", "face-down", "changed-name", "returned", "phased", "cycle", "clone", "copy"} {
		t.Run(state, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(SharesANameOriginallyPrintedInAntiquities)],ActionList([TapPermanent(Ref_TargetPermanent)])))`)
			g, a, b := ronGame()
			card := mage.NewCreature("Atog", "", 2, 2)
			subject := g.PutOnBattlefield(card, b.PlayerID())
			spell := factory()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{subject.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "face-down":
				g.MutablePermanent(subject.ID()).FaceDown = true
			case "changed-name":
				changed := mage.NewCreature("Other", "", 2, 2)
				changed.SetID(subject.ID())
				changed.SetOwner(subject.Card.Owner())
				g.MutablePermanent(subject.ID()).Card = changed
			case "returned":
				g.BouncePermanentToHand(subject)
				b.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "phased":
				g.PhaseOut(subject.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(subject.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			if state == "clone" {
				clone.MutablePermanent(subject.ID()).FaceDown = true
			}
			g.ResolveStack()
			want := state == "normal" || state == "cycle" || state == "clone" || state == "copy"
			if g.FindPermanentIncludingPhased(subject.ID()).Tapped != want {
				t.Fatal("target name or visit was not checked again")
			}
			clone.ResolveStack()
			if clone.FindPermanentIncludingPhased(subject.ID()).Tapped != (want && state != "clone") {
				t.Fatal("clone target state shared")
			}
		})
	}
}

func TestRONOriginalPrintingNamesSharedContexts(t *testing.T) {
	for _, name := range []string{"SharesANameOriginallyPrintedInAntiquities", "SharesANameOriginallyPrintedInArabianNights"} {
		member := "Atog"
		if strings.HasSuffix(name, "ArabianNights") {
			member = "Kird Ape"
		}
		t.Run(name+"/static", func(t *testing.T) {
			factory := permanentScopeFactory(t, "Enchantment", `EachPermanentLayerEffect(`+name+`,[AdjustPT(1,1)])`)
			g, a, b := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			subject := g.PutOnBattlefield(mage.NewCreature(member, "", 2, 2), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 2), b.PlayerID())
			checkConditionalPT(t, g, subject.ID(), 3, 3)
			checkConditionalPT(t, g, other.ID(), 2, 2)
			g.MutablePermanent(subject.ID()).FaceDown = true
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, subject.ID(), 2, 2)
			g.MutablePermanent(subject.ID()).FaceDown = false
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, subject.ID(), 3, 3)
			g.PhaseOut(source.ID())
			g.ApplyContinuousEffects()
			checkConditionalPT(t, g, subject.ID(), 2, 2)
		})
		t.Run(name+"/cost", func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(And([PayMana([ManaCostGeneric(1)]),SacrificeNumberPermanents(Integer(1),`+name+`)]),ActionList([GainLife(Integer(1))]))`)
			g, a, _ := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			subject := g.PutOnBattlefield(mage.NewCreature(member, "", 2, 2), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || g.FindPermanent(subject.ID()) == nil {
				t.Fatal("failed payment moved subject")
			}
			a.ManaPool().Add(core.Colorless, 1)
			g.MutablePermanent(subject.ID()).FaceDown = true
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || a.ManaPool().TotalMana() != 1 {
				t.Fatal("nameless payment spent resources")
			}
			g.MutablePermanent(subject.ID()).FaceDown = false
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(subject.ID()) != nil || a.Life() != 20 {
				t.Fatal("cost timing")
			}
			g.ResolveStack()
			if a.Life() != 21 {
				t.Fatal("paid continuation")
			}
		})
		t.Run(name+"/information", func(t *testing.T) {
			record := staticBaseRecord(t, "ATQ", "Golgothian Sylex")
			record.Fields["Name"] = Node{Kind: String, String: member}
			record.Fields["Rules"] = nameScopeNode(t, `[Activated(PayMana([]),ActionList([If(PermanentPassesFilter(ThisPermanent,`+name+`),[GainLife(Integer(2))])]))]`)
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.BouncePermanentToHand(source)
			g.ResolveStack()
			if a.Life() != 22 {
				t.Fatal("retained name was not read after departure")
			}
		})
	}
	text := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([TapEachPermanent(SharesANameOriginallyPrintedInAntiquities)]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(text)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	found := false
	for _, sig := range report.Signatures {
		if sig.Name == "SharesANameOriginallyPrintedInAntiquities" && sig.Role == "permanent_filter" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing filter inventory role")
	}
}

func TestRONOriginalPrintingNamesSylexSimultaneous(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Golgothian Sylex"), a.PlayerID())
	victim := g.PutOnBattlefield(mage.NewCreature("Atog", "", 2, 2, mage.WithKeyword(core.Indestructible)), b.PlayerID())
	g.AddRegenerationShield(victim.ID())
	events := 0
	watcher := mage.NewTriggered(core.EvtZoneChange, false).SetCondition(func(evt *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
		if evt.SourceID == source.ID() || evt.SourceID == victim.ID() {
			events++
			if g.FindPermanent(source.ID()) != nil || g.FindPermanent(victim.ID()) != nil {
				t.Error("event saw partial sacrifice")
			}
		}
		return false
	})
	g.PutOnBattlefield(mage.NewArtifact("Observer", "", mage.WithAbility(watcher)), a.PlayerID())
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if events != 2 || g.FindPermanent(victim.ID()) != nil {
		t.Fatal("sacrifice used destruction or partial batch", events)
	}
}

func TestRONOriginalPrintingNamesSylexCast(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "ATQ", "Golgothian Sylex")
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || a.ManaPool().TotalMana() != 3 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("failed cast changed resources")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil || a.ManaPool().TotalMana() != 0 {
		t.Fatal("cast timing")
	}
	g.ResolveStack()
	if g.FindPermanent(card.ID()) == nil {
		t.Fatal("Sylex did not enter")
	}
	victim := g.PutOnBattlefield(repositoryCard(t, "3ED", "Ornithopter"), b.PlayerID())
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(card.ID()) != nil || g.FindPermanent(victim.ID()) != nil {
		t.Fatal("reprint did not share original Antiquities name")
	}
}
