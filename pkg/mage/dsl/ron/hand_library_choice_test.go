package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONHandLibraryChoiceConchHorn(t *testing.T) {
	for _, copyAbility := range []bool{false, true} {
		g, a, b := ronGame()
		card := repositoryCard(t, "FEM", "Conch Horn")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Colorless, 2)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		p := g.FindPermanent(card.ID())
		if p == nil {
			t.Fatal("artifact did not resolve")
		}
		for range 6 {
			a.AddToLibrary(mage.NewLand("A draw"))
			b.AddToLibrary(mage.NewLand("B draw"))
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || p.Tapped || len(a.Graveyard()) != 0 || len(a.Hand()) != 0 || len(g.StackObjects()) != 0 {
			t.Fatal("failed payment changed state")
		}
		a.ManaPool().Add(core.Colorless, 1)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(p.ID()) != nil || len(a.Graveyard()) != 1 || len(a.Hand()) != 0 || a.ManaPool().TotalMana() != 0 {
			t.Fatal("cost or action timing")
		}
		if copyAbility && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
			t.Fatal("ability copy failed")
		}
		clone := g.Clone()
		g.ResolveStack()
		wantB := 0
		if copyAbility {
			wantB = 1
		}
		if len(a.Hand()) != 1 || len(a.Library()) != 5 || len(b.Hand()) != wantB || len(b.Library()) != 6-wantB {
			t.Fatal("draw and required put-back", len(a.Hand()), len(b.Hand()))
		}
		if len(clone.GetPlayer(a.PlayerID()).Hand()) != 0 {
			t.Fatal("clone changed before resolution")
		}
		clone.ResolveStack()
		if len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 || len(clone.GetPlayer(b.PlayerID()).Hand()) != wantB {
			t.Fatal("pending clone lost action")
		}
	}
}

func TestRONHandLibraryChoiceInvalid(t *testing.T) {
	action := `PutACardFromHandOnTopOfLibrary`
	for _, rule := range []string{
		`SpellActions(ActionList([` + action + `()]))`,
		`SpellActions(ActionList([` + action + `(You)]))`,
		`SpellActions(ActionList([PlayerAction(Ref_TargetPlayer,` + action + `)]))`,
		`Activated(` + action + `,ActionList([]))`,
		`SpellActions(ActionList([EachPlayerAction(AnyPlayer,` + action + `)]))`,
		`PermanentLayerEffect(ThisPermanent,[` + action + `])`,
		`AsPermanentEnters(ThisPermanent,[` + action + `])`,
		`ReplaceWouldDraw(APlayerWouldDrawExceptFirstDrawStepDraw(AnyPlayer),[` + action + `])`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONHandLibraryChoiceScopes(t *testing.T) {
	action := `PutACardFromHandOnTopOfLibrary`
	for _, body := range []string{action, `PlayerAction(You,` + action + `)`, `PlayerActions(You,[PlayerActions(ActionPlayer,[` + action + `])])`, `PlayerAction(Ref_TargetPlayer,` + action + `)`} {
		t.Run(body, func(t *testing.T) {
			g, a, b := ronGame()
			for _, p := range []*mage.BasePlayer{a, b} {
				p.AddToHand(mage.NewLand("Hand"))
				p.AddToLibrary(mage.NewLand("Old top"))
			}
			targeted := body == `PlayerAction(Ref_TargetPlayer,`+action+`)`
			rule := `Activated(PayMana([]),ActionList([` + body + `]))`
			if targeted {
				rule = `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([` + body + `])))`
			}
			p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
			if targeted {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
			} else if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if len(a.Hand()) != 1 || len(b.Hand()) != 1 {
				t.Fatal("choice ran at announcement")
			}
			g.PhaseOut(p.ID())
			g.ResolveStack()
			wantA, wantB := 0, 1
			if targeted {
				wantA, wantB = 1, 0
			}
			if len(a.Hand()) != wantA || len(b.Hand()) != wantB {
				t.Fatal("wrong player scope")
			}
		})
	}
}

func TestRONHandLibraryChoiceReplacedDraws(t *testing.T) {
	for _, existing := range []bool{false, true} {
		g, a, b := ronGame()
		if existing {
			a.AddToHand(mage.NewLand("Existing"))
		}
		for range 3 {
			a.AddToLibrary(mage.NewLand("Undrawn"))
		}
		replacement := permanentScopeFactory(t, "Enchantment", `ReplaceWouldDraw(APlayerWouldDrawExceptFirstDrawStepDraw(AnyPlayer),[PlayerAction(WouldDrawACard_ThatPlayer,GainLife(Integer(1)))])`)()
		g.PutOnBattlefield(replacement, b.PlayerID())
		card := repositoryCard(t, "FEM", "Conch Horn")
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		a.ManaPool().Add(core.Colorless, 1)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		want := 3
		if existing {
			want = 4
		}
		if len(a.Hand()) != 0 || len(a.Library()) != want || a.Life() != 22 {
			t.Fatal("put-back did not use post-replacement hand")
		}
		if existing && a.Library()[0].Name() != "Existing" {
			t.Fatal("did not choose existing hand card")
		}
	}
}

func TestRONHandLibraryChoiceLoadAndInventory(t *testing.T) {
	source := `Card(Name:"Choice",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([PutACardFromHandOnTopOfLibrary]))])`
	files := fstest.MapFS{"choice.ron": &fstest.MapFile{Data: []byte(source)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	first, err := registry.CreateCard("Choice")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Choice")
	if err != nil {
		t.Fatal(err)
	}
	if first == nil || second == nil || first.ID() == second.ID() {
		t.Fatal("factory instances share identity")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(source, `PutACardFromHandOnTopOfLibrary`, `PutACardFromHandOnTopOfLibrary(You)`, 1))}
	if LoadInto(registry, files, "*.ron") == nil || !registry.CardRegistered("Choice") {
		t.Fatal("invalid collection replaced registry")
	}
}

func TestRONHandLibraryChoiceContexts(t *testing.T) {
	for _, kind := range []string{"spell", "trigger", "delayed"} {
		t.Run(kind, func(t *testing.T) {
			g, a, b := ronGame()
			cardtype, rule := "Instant", `SpellActions(ActionList([PutACardFromHandOnTopOfLibrary]))`
			if kind == "trigger" {
				cardtype, rule = "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([PutACardFromHandOnTopOfLibrary]))`
			}
			if kind == "delayed" {
				cardtype, rule = "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PutACardFromHandOnTopOfLibrary]))]))`
			}
			card := permanentScopeFactory(t, cardtype, rule)()
			card.SetOwner(a.PlayerID())
			switch kind {
			case "spell":
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
			case "trigger":
				g.PutOnBattlefield(card, a.PlayerID())
				g.PutTriggersOnStack()
			case "delayed":
				p := g.PutOnBattlefield(card, a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				g.DestroyPermanent(p)
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
			}
			late := mage.NewLand("Late hand")
			a.AddToHand(late)
			g.ResolveStack()
			if len(a.Hand()) != 0 || len(a.Library()) != 1 || a.Library()[0].ID() != late.ID() || len(b.Library()) != 0 {
				t.Fatal("choice did not read resolution hand")
			}
		})
	}
}

type handLibraryTimingPlayer struct {
	*mage.BasePlayer
	choose func([]mage.Card) mage.Card
	calls  int
}

func (p *handLibraryTimingPlayer) ChooseCardFromHand(cards []mage.Card, _ string, _ mage.GameReader) mage.Card {
	p.calls++
	return p.choose(cards)
}

func TestRONHandLibraryChoiceDrawTiming(t *testing.T) {
	g, base, b := ronGame()
	a := &handLibraryTimingPlayer{BasePlayer: base}
	g.SetPlayerAt(0, a)
	existing := mage.NewLand("Existing")
	a.AddToHand(existing)
	for range 4 {
		a.AddToLibrary(mage.NewLand("Drawn"))
	}
	watcher := mage.NewTriggered(core.EvtCardDrawn, false, mage.GainLifeFor(mage.You, mage.Fixed(1)))
	g.PutOnBattlefield(mage.NewArtifact("Watcher", "", mage.WithAbility(watcher)), a.PlayerID())
	card := repositoryCard(t, "FEM", "Conch Horn")
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	a.choose = func(cards []mage.Card) mage.Card {
		if len(cards) != 3 || a.Life() != 20 || g.FindPermanent(p.ID()) != nil || len(a.Graveyard()) != 1 || len(b.Hand()) != 0 {
			t.Fatal("choice did not follow draws and sacrifice before draw triggers")
		}
		return existing
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.calls != 1 || len(a.Hand()) != 2 || a.Library()[0].ID() != existing.ID() {
		t.Fatal("required choice ignored pre-existing card")
	}
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("draw triggers did not resolve after put-back")
	}
}
