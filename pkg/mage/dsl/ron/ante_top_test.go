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

func TestRONAnteTopCards(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Demonic Attorney", "Contract from Below"} {
			for _, active := range []int{0, 1} {
				t.Run(fmt.Sprintf("%s/%s/%d", set, name, active), func(t *testing.T) {
					a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
					g, err := mage.NewGameWithAnte(a, b, nil, nil)
					if err != nil {
						t.Fatal(err)
					}
					g.SetActivePlayerIndex(active)
					g.SetStep(core.PrecombatMain)
					g.SetTurn(1)
					caster, other := a, b
					if active == 1 {
						caster, other = b, a
					}
					fillRONLibrary(t, caster, 10)
					fillRONLibrary(t, other, 10)
					top, otherTop := caster.Library()[0].ID(), other.Library()[0].ID()
					card := repositoryCard(t, set, name)
					if err := mage.ValidateCardDefinition(name, card); err != nil {
						t.Fatal(err)
					}
					card.SetOwner(caster.PlayerID())
					caster.AddToHand(card)
					caster.ManaPool().Add(core.Black, 1)
					if name == "Demonic Attorney" {
						caster.ManaPool().Add(core.Colorless, 2)
					}
					if name == "Demonic Attorney" {
						if err := g.CastSpellByID(caster.PlayerID(), card.ID(), nil, 0); err == nil {
							t.Fatal("insufficient black mana accepted")
						}
						if len(caster.Hand()) != 1 || caster.ManaPool().TotalMana() != 3 || g.StackSize() != 0 || len(caster.Library()) != 10 {
							t.Fatal("failed payment changed state")
						}
						caster.ManaPool().Add(core.Black, 1)
					}
					if err := g.CastSpellByID(caster.PlayerID(), card.ID(), nil, 0); err != nil {
						t.Fatal(err)
					}
					hand := repositoryCard(t, "LEA", "Forest")
					hand.SetOwner(caster.PlayerID())
					caster.AddToHand(hand)
					clone := g.Clone()
					clone.ResolveStack()
					if len(caster.Library()) != 10 || len(caster.Ante()) != 0 || len(caster.Hand()) != 1 {
						t.Fatal("clone changed original")
					}
					g.ResolveStack()
					if len(caster.Ante()) != 1 || caster.Ante()[0].ID() != top {
						t.Fatal("wrong ante top")
					}
					if name == "Contract from Below" {
						if len(caster.Hand()) != 7 || len(caster.Library()) != 2 || len(caster.Graveyard()) != 2 || len(other.Library()) != 10 || len(other.Ante()) != 0 {
							t.Fatal("contract order or player scope")
						}
					} else if len(other.Ante()) != 1 || other.Ante()[0].ID() != otherTop || len(caster.Library()) != 9 || len(caster.Hand()) != 1 {
						t.Fatal("all-player ante")
					}
				})
			}
		}
	}
}

func TestRONAnteTopInvalid(t *testing.T) {
	for _, rule := range []string{
		`SpellActions(ActionList([AnteTopCardOfLibrary()]))`,
		`SpellActions(ActionList([EachPlayerAction(AnyPlayer,AnteTopCardOfLibrary(Integer(1)))]))`,
		`SpellActions(ActionList([PlayerAction(ActionPlayer,AnteTopCardOfLibrary)]))`,
		`Activated(AnteTopCardOfLibrary,ActionList([]))`,
		`AsPermanentEnters(ThisPermanent,[AnteTopCardOfLibrary])`,
		`PermanentLayerEffect(ThisPermanent,AnteTopCardOfLibrary)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONAnteTopPlayerScopes(t *testing.T) {
	for _, body := range []string{
		`PlayerAction(Ref_TargetPlayer,AnteTopCardOfLibrary)`,
		`PlayerActions(Ref_TargetPlayer,[AnteTopCardOfLibrary,PlayerAction(ActionPlayer,AnteTopCardOfLibrary)])`,
	} {
		a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
		g, err := mage.NewGameWithAnte(a, b, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		g.SetStep(core.PrecombatMain)
		g.SetTurn(1)
		fillRONLibrary(t, a, 3)
		fillRONLibrary(t, b, 3)
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([`+body+`])))`)
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		want := 1
		if strings.HasPrefix(body, "PlayerActions") {
			want = 2
		}
		if len(a.Library()) != 3 || len(a.Ante()) != 0 || len(b.Ante()) != want {
			t.Fatal("wrong scoped player")
		}
	}
}

func TestRONAnteTopSequenceEvents(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g, err := mage.NewGameWithAnte(a, b, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	fillRONLibrary(t, a, 9)
	top := a.Library()[0].ID()
	card := repositoryCard(t, "LEA", "Contract from Below")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	hand := repositoryCard(t, "LEA", "Forest")
	hand.SetOwner(a.PlayerID())
	a.AddToHand(hand)
	discards, antes, draws := 0, 0, 0
	for _, kind := range []core.EventType{core.EvtDiscard, core.EvtZoneChange, core.EvtCardDrawn} {
		watcher := mage.NewTriggered(kind, false).SetCondition(func(e *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
			switch e.Type {
			case core.EvtDiscard:
				discards++
				if len(a.Hand()) != 0 || len(a.Ante()) != 0 || len(a.Library()) != 9 {
					t.Error("discard sequence")
				}
			case core.EvtZoneChange:
				if e.ToZone == core.ZoneAnte {
					antes++
					if discards != 1 || draws != 0 || len(a.Library()) != 8 || a.Ante()[0].ID() != top {
						t.Error("ante sequence")
					}
				}
			case core.EvtCardDrawn:
				draws++
				if antes != 1 {
					t.Error("draw preceded ante")
				}
			}
			return false
		})
		g.PutOnBattlefield(mage.NewArtifact("observer", "", mage.WithAbility(watcher)), a.PlayerID())
	}
	a.ManaPool().Add(core.Black, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if discards != 1 || antes != 1 || draws != 7 {
		t.Fatalf("events %d/%d/%d", discards, antes, draws)
	}
}

func TestRONAnteTopDelayedIdentity(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phased"} {
		t.Run(state, func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g, err := mage.NewGameWithAnte(a, b, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			fillRONLibrary(t, a, 2)
			fillRONLibrary(t, b, 2)
			factory := ronDelayedCombatFactory(t, `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerAction(ControllerOfPermanent(ThisPermanent),AnteTopCardOfLibrary)]))]))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if state == "departed" || state == "returned" {
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			}
			if state == "phased" {
				g.PhaseOut(p.ID())
			}
			sources := g.WaitingDelayedPermanentSources()
			if len(sources) != 1 || !sources[0].SameObject(saved) || len(a.Library()) != 2 {
				t.Fatal("waiting query lost identity or executed action")
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{clone, g} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				if len(game.GetPlayer(a.PlayerID()).Ante()) != 1 || len(game.GetPlayer(b.PlayerID()).Ante()) != 0 || len(game.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("wrong delayed player or lifetime")
				}
				if game == clone && len(a.Ante()) != 0 {
					t.Fatal("clone changed original")
				}
			}
		})
	}
}

func TestRONAnteTopCopiesAndEmpty(t *testing.T) {
	for _, name := range []string{"Demonic Attorney", "Contract from Below"} {
		t.Run(name, func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g, err := mage.NewGameWithAnte(a, b, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			fillRONLibrary(t, a, 1)
			top := a.Library()[0].ID()
			card := repositoryCard(t, "LEA", name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Black, 2)
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			if len(a.Ante()) != 1 || a.Ante()[0].ID() != top || len(b.Ante()) != 0 || len(a.Library()) != 0 {
				t.Fatal("copy or empty library")
			}
			if name == "Contract from Below" && (len(a.Hand()) != 0 || a.IsAlive() || b.IsAlive()) {
				t.Fatal("ante failure stopped draws")
			}
		})
	}
}

func TestRONAnteTopInventory(t *testing.T) {
	source := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([AnteTopCardOfLibrary,EachPlayerAction(AnyPlayer,AnteTopCardOfLibrary)]))])`
	report, err := InspectFS(fstest.MapFS{"ante.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := make(map[string]bool)
	for _, signature := range report.Signatures {
		if signature.Name == "AnteTopCardOfLibrary" {
			roles[signature.Role] = true
		}
	}
	if report.Compiled != 1 || !roles["action"] || !roles["player_action"] {
		t.Fatalf("compiled=%d roles=%v", report.Compiled, roles)
	}
}
