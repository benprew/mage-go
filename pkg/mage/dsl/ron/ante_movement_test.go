package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONAnteMovementJeweledBird(t *testing.T) {
	for _, state := range []string{"pay", "no-other", "foreign-owner", "control", "phased", "departed", "returned", "clone", "copy"} {
		t.Run(state, func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g, _ := mage.NewGameWithAnte(a, b, nil, nil)
			fillRONLibrary(t, a, 3)
			fillRONLibrary(t, b, 3)
			top := a.Library()[0].ID()
			owned, foreign := mage.NewArtifact("owned", ""), mage.NewArtifact("foreign", "")
			owned.SetOwner(a.PlayerID())
			foreign.SetOwner(b.PlayerID())
			if state != "no-other" {
				a.AddToAnte(owned)
			}
			b.AddToAnte(foreign)
			card := repositoryCard(t, "ARN", "Jeweled Bird")
			card.SetOwner(a.PlayerID())
			if state == "foreign-owner" {
				card.SetOwner(b.PlayerID())
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if !p.Tapped || len(a.Hand()) != 0 || len(a.Graveyard()) != 0 {
				t.Fatal("action ran before resolution")
			}
			switch state {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phased":
				g.PhaseOut(p.ID())
			case "departed", "returned":
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "clone":
				clone := g.Clone()
				clone.ResolveStack()
				if len(clone.GetPlayer(a.PlayerID()).Ante()) != 1 || clone.GetPlayer(a.PlayerID()).Ante()[0].ID() != card.ID() || len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 1 || len(a.Graveyard()) != 0 || g.StackSize() != 1 {
					t.Fatal("clone movement leaked")
				}
			}
			g.ResolveStack()
			paid := state == "pay" || state == "no-other" || state == "control" || state == "clone" || state == "copy"
			if paid {
				want := 1
				if state == "no-other" {
					want = 0
				}
				if len(a.Ante()) != 1 || a.Ante()[0].ID() != card.ID() || len(a.Graveyard()) != want || len(a.Hand()) != 1 || a.Hand()[0].ID() != top || len(a.Library()) != 2 {
					t.Fatal("wrong payment, source exclusion, movement or draw")
				}
			} else {
				wantHand := 0
				if state == "departed" {
					wantHand = 1
				}
				if len(a.Ante()) != 1 || a.Ante()[0].ID() != owned.ID() || len(a.Graveyard()) != 0 || len(a.Library()) != 3 || len(a.Hand()) != wantHand {
					t.Fatal("unpaid branch moved cards or drew")
				}
			}
			if len(b.Ante()) != 1 || b.Ante()[0].ID() != foreign.ID() || len(b.Hand()) != 0 || len(b.Library()) != 3 {
				t.Fatal("affected wrong player")
			}
		})
	}
}

func TestRONAnteMovementDeclarations(t *testing.T) {
	for _, subject := range []string{`OwnedByAPlayer`, `OwnedByAPlayer(You)`, `OwnedByAPlayer(SinglePlayer())`, `OwnedByAPlayer(SinglePlayer(ActionPlayer))`, `OwnedByAPlayer(SinglePlayer(Ref_TargetPlayer))`, `Other()`, `Other(AnyCard)`, `Other(OwnedByAPlayer(SinglePlayer(ThisPermanent)))`, `AnyCard`, `Ref_TargetAnteCard`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([PutEachCardFromAnteInGraveyard(` + subject + `)]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", subject)
		}
	}
	for _, rule := range []string{`SpellActions(ActionList([PutEachCardFromAnteInGraveyard]))`, `SpellActions(ActionList([PutEachCardFromAnteInGraveyard()]))`, `Activated(PutEachCardFromAnteInGraveyard(OwnedByAPlayer(SinglePlayer(You))),ActionList([DrawACard]))`, `PermanentLayerEffect(ThisPermanent,PutEachCardFromAnteInGraveyard(OwnedByAPlayer(SinglePlayer(You))))`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONAnteMovementSourceScope(t *testing.T) {
	move := `PutEachCardFromAnteInGraveyard(Other(OwnedByAPlayer(SinglePlayer(You))))`
	for _, body := range []string{
		move,
		`MustCost(AnteAPermanent(SinglePermanent(ThisPermanent))),Unless(CostWasPaid,[` + move + `])`,
		`MustCost(AnteAPermanent(SinglePermanent(ThisPermanent))),If(CostWasPaid,[DrawACard]),` + move,
		`MustCost(AnteAPermanent(SinglePermanent(ThisPermanent))),If(CostWasPaid,[CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + move + `]))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(TapPermanent(ThisPermanent),ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted out-of-scope source exclusion: %s", body)
		}
	}
}

func TestRONAnteMovementBirdSequenceAndOrder(t *testing.T) {
	a := &ronMillOrderPlayer{ronPaymentPlayer: &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A")}}
	b := mage.NewBasePlayer("B")
	g, _ := mage.NewGameWithAnte(a, b, nil, nil)
	fillRONLibrary(t, a.BasePlayer, 2)
	first, second := mage.NewArtifact("first", ""), mage.NewArtifact("second", "")
	first.SetOwner(a.PlayerID())
	second.SetOwner(a.PlayerID())
	a.AddToAnte(first)
	a.AddToAnte(second)
	card := repositoryCard(t, "ARN", "Jeweled Bird")
	card.SetOwner(a.PlayerID())
	if retained, excluded := mage.PrepareCardsForAnte([]mage.Card{card}, false); len(retained) != 0 || len(excluded) != 1 {
		t.Fatal("missing deck restriction")
	}
	stages := []core.Zone{}
	observe := func(e *core.GameEvent, reader mage.GameReader, _, _ uuid.UUID) bool {
		if reader != g {
			return false
		}
		if e.SourceID == card.ID() && e.ToZone == core.ZoneAnte {
			stages = append(stages, core.ZoneAnte)
			if len(a.Ante()) != 3 || len(a.Graveyard()) != 0 || len(a.Hand()) != 0 {
				t.Error("wrong payment timing")
			}
		}
		if e.FromZone == core.ZoneAnte {
			stages = append(stages, core.ZoneGraveyard)
			if len(a.Ante()) != 1 || len(a.Graveyard()) != 2 || len(a.Hand()) != 0 {
				t.Error("movement was partial or followed draw")
			}
		}
		if e.Type == core.EvtCardDrawn {
			stages = append(stages, core.ZoneHand)
			if len(a.Graveyard()) != 2 || len(a.Ante()) != 1 {
				t.Error("draw preceded movement")
			}
		}
		return false
	}
	for _, kind := range []core.EventType{core.EvtZoneChange, core.EvtCardDrawn} {
		watcher := mage.NewTriggered(kind, false).SetCondition(observe)
		g.PutOnBattlefield(mage.NewArtifact("observer", "", mage.WithAbility(watcher)), a.PlayerID())
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil || g.StackSize() != 1 {
		t.Fatal("tapped source paid again")
	}
	g.ResolveStack()
	if a.orders != 1 || a.Graveyard()[0].ID() != second.ID() || a.Graveyard()[1].ID() != first.ID() || len(stages) != 4 || stages[0] != core.ZoneAnte || stages[1] != core.ZoneGraveyard || stages[2] != core.ZoneGraveyard || stages[3] != core.ZoneHand {
		t.Fatalf("wrong sequence or owner order: stages=%v orders=%d", stages, a.orders)
	}
}

func TestRONAnteMovementExplicitPlayerAndInventory(t *testing.T) {
	src := `Card(Name:"Move",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([PutEachCardFromAnteInGraveyard(OwnedByAPlayer(SinglePlayer(Ref_TargetPlayer)))])))])`
	nodes, err := Parse(strings.NewReader(src), "move.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g, _ := mage.NewGameWithAnte(a, b, nil, nil)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	owned := mage.NewArtifact("owned", "")
	owned.SetOwner(b.PlayerID())
	a.AddToAnte(owned)
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(a.Ante()) != 0 || len(b.Graveyard()) != 1 || b.Graveyard()[0].ID() != owned.ID() {
		t.Fatal("wrong explicit owner")
	}
	report, err := InspectFS(fstest.MapFS{"move.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{}
	for _, signature := range report.Signatures {
		roles[signature.Name] = signature.Role
	}
	if roles["OwnedByAPlayer"] != "ante_card_filter" || roles["SinglePlayer"] != "player_filter" || roles["Ref_TargetPlayer"] != "player_reference" {
		t.Fatal("incorrect movement roles", roles)
	}
}
