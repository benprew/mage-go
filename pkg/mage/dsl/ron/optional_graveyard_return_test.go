package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronOptionalReturnPlayer struct {
	*mage.BasePlayer
	accept  bool
	offers  int
	onOffer func()
}

func (p *ronOptionalReturnPlayer) ChooseMayAbility(string) bool {
	p.offers++
	if p.onOffer != nil {
		p.onOffer()
	}
	return p.accept
}

func TestRONOptionalGraveyardReturnNetherShadow(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, scenario := range []string{"accept", "decline", "below-only", "two-creatures", "remove-above", "depart", "return", "choice-return", "two-instances", "copy", "other-controller-copy", "pending-clone", "stack-clone", "phasing", "replacement"} {
			t.Run(set+"/"+scenario, func(t *testing.T) {
				card := repositoryCard(t, set, "Nether Shadow")
				a := &ronOptionalReturnPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "decline"}
				b := &ronOptionalReturnPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
				g := mage.NewGame(a, b)
				addCreature := func() mage.Card {
					c := mage.NewCreature("Above", "", 1, 1)
					c.SetOwner(a.PlayerID())
					a.AddToGraveyard(c)
					return c
				}
				for range 3 {
					addCreature()
				}
				card.SetOwner(a.PlayerID())
				a.AddToGraveyard(card)
				var above mage.Card
				count := 3
				if scenario == "below-only" {
					count = 0
				}
				if scenario == "two-creatures" {
					count = 2
				}
				for range count {
					above = addCreature()
				}
				filler := mage.NewArtifact("Noncreature", "")
				filler.SetOwner(a.PlayerID())
				a.AddToGraveyard(filler)
				var second mage.Card
				if scenario == "two-instances" {
					second = repositoryCard(t, set, "Nether Shadow")
					second.SetOwner(a.PlayerID())
					a.AddToGraveyard(second)
					for range 3 {
						addCreature()
					}
				}
				queueRONUpkeep(g, b)
				if g.StackSize() != 0 {
					t.Fatal("opponent upkeep triggered")
				}
				var clone *mage.Game
				if scenario == "pending-clone" {
					g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
					clone = g.Clone()
					g.PutTriggersOnStack()
				} else {
					queueRONUpkeep(g, a)
				}
				wantStack := 1
				if scenario == "below-only" || scenario == "two-creatures" {
					wantStack = 0
				}
				if scenario == "two-instances" {
					wantStack = 2
				}
				if g.StackSize() != wantStack || a.offers != 0 || b.offers != 0 {
					t.Fatalf("announcement: stack=%d offers=%d/%d", g.StackSize(), a.offers, b.offers)
				}
				if scenario == "stack-clone" {
					clone = g.Clone()
				}
				if clone != nil {
					clone.SetPlayerAt(0, &ronOptionalReturnClonePlayer{SearchPlayer: clone.GetPlayer(a.PlayerID()).(*mage.SearchPlayer)})
					clone.PutTriggersOnStack()
				}
				if scenario == "remove-above" {
					a.RemoveFromGraveyard(above.ID())
				}
				if scenario == "depart" || scenario == "return" {
					a.RemoveFromGraveyard(card.ID())
					if scenario == "return" {
						a.AddToGraveyard(card)
						for range 3 {
							addCreature()
						}
					}
				}
				if scenario == "choice-return" {
					a.onOffer = func() {
						a.RemoveFromGraveyard(card.ID())
						a.AddToGraveyard(card)
						for range 3 {
							addCreature()
						}
					}
				}
				if scenario == "copy" || scenario == "other-controller-copy" {
					controller := a.PlayerID()
					if scenario == "other-controller-copy" {
						controller = b.PlayerID()
					}
					if g.CopyStackObjectDirect(g.StackPeek(), controller, false) == nil {
						t.Fatal("copy missing")
					}
				}
				if scenario == "replacement" {
					g.AddEntersTappedRule(func(p *mage.Permanent) bool { return p.HasType(core.TypeCreature) })
				}
				g.ResolveStack()
				wantEntry := scenario == "accept" || scenario == "copy" || scenario == "other-controller-copy" || scenario == "pending-clone" || scenario == "stack-clone" || scenario == "two-instances" || scenario == "phasing" || scenario == "replacement"
				p := g.FindPermanent(card.ID())
				if (p != nil) != wantEntry {
					t.Fatalf("entry=%v, want %v", p != nil, wantEntry)
				}
				wantOffers := 1
				if !wantEntry && scenario != "decline" && scenario != "choice-return" {
					wantOffers = 0
				}
				if scenario == "two-instances" {
					wantOffers = 2
				}
				if a.offers != wantOffers || b.offers != 0 {
					t.Fatalf("offers=%d/%d, want %d/0", a.offers, b.offers, wantOffers)
				}
				if p != nil && (p.ControllerID() != a.PlayerID() || p.Card.Owner() != a.PlayerID() || !p.HasKeyword(core.Haste) || (!p.Tapped && !p.CanDeclareAsAttacker(g)) || p.CurrentPower(g) != 1 || p.CurrentToughness(g) != 1 || p.Tapped != (scenario == "replacement")) {
					t.Fatal("wrong returned characteristics")
				}
				if second != nil && g.FindPermanent(second.ID()) == nil {
					t.Fatal("factory bindings crossed")
				}
				if clone != nil {
					clone.ResolveStack()
					if clone.FindPermanent(card.ID()) == nil {
						t.Fatal("clone lost saved visit")
					}
					g.BouncePermanentToHand(p)
					if clone.FindPermanent(card.ID()) == nil {
						t.Fatal("clone shares zone state")
					}
				}
				if scenario == "phasing" {
					group := g.PhaseOut(p.ID())
					queueRONUpkeep(g, a)
					if g.StackSize() != 0 {
						t.Fatal("phased source triggered")
					}
					g.PhaseIn(group)
					if g.FindPermanent(card.ID()) == nil {
						t.Fatal("phase-in lost visit")
					}
				}
			})
		}
	}
}

func TestRONOptionalGraveyardReturnInvalid(t *testing.T) {
	const action = `MayAction(PutGraveyardCardOntoBattlefield(ThisGraveyardCard,[EntersNormally]))`
	for _, bad := range []string{
		`MayAction()`, `MayAction(PutGraveyardCardOntoBattlefield())`,
		strings.Replace(action, "ThisGraveyardCard", "ThisExiledCard", 1),
		strings.Replace(action, "ThisGraveyardCard", "ThisGraveyardCard()", 1),
		strings.Replace(action, "[EntersNormally]", "[]", 1),
		strings.Replace(action, "[EntersNormally]", "[EntersNormally,EntersNormally]", 1),
		strings.Replace(action, "EntersNormally", "EntersNormally()", 1),
		strings.Replace(action, "EntersNormally", "EntersUnderPlayersControl(Opponent)", 1),
		strings.Replace(action, "EntersNormally", "EntersTapped", 1),
	} {
		src := sourceZoneFixture("FromGraveyard", bad)
		if _, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(src)}}, "*.ron"); err == nil {
			t.Fatalf("invalid accepted: %s", bad)
		}
	}
	for _, src := range []string{
		sourceZoneFixture("FromExile", action),
		strings.Replace(strings.Replace(sourceZoneFixture("FromGraveyard", action), "Cardtypes:[Artifact],Subtypes:[]", "Cardtypes:[Enchantment],Subtypes:[Aura]", 1), "Rules:[", "Rules:[EnchantPermanent(IsCardtype(Creature)),", 1),
		fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([%s]))])`, action),
		fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[AsPermanentEnters(ThisPermanent,[%s])])`, action),
	} {
		if _, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(src)}}, "*.ron"); err == nil {
			t.Fatalf("invalid context accepted: %s", src)
		}
	}
	registry := mage.NewRegistry()
	registry.Register("Existing", func() mage.Card { return mage.NewArtifact("Existing", "") })
	if err := LoadInto(registry, fstest.MapFS{"good.ron": {Data: []byte(sourceZoneFixture("FromGraveyard", action))}, "bad.ron": {Data: []byte(sourceZoneFixture("FromExile", action))}}, "*.ron"); err == nil || !registry.CardRegistered("Existing") || registry.CardRegistered("Zone watcher") {
		t.Fatal("invalid collection replaced registry")
	}
}

type ronOptionalReturnClonePlayer struct{ *mage.SearchPlayer }

func (*ronOptionalReturnClonePlayer) ChooseMayAbility(string) bool { return true }

func TestRONOptionalGraveyardReturnFrames(t *testing.T) {
	const action = `MayAction(PutGraveyardCardOntoBattlefield(ThisGraveyardCard,[EntersNormally]))`
	for _, scenario := range []string{"decline", "absent", "two-events", "delayed", "delayed-return"} {
		t.Run(scenario, func(t *testing.T) {
			body := action + `,GainLife(Integer(1))`
			if strings.HasPrefix(scenario, "delayed") {
				body = `CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + body + `]))`
			}

			registry, err := LoadFS(fstest.MapFS{"zone.ron": {Data: []byte(sourceZoneFixture("FromGraveyard", body))}}, "*.ron")
			if err != nil {
				t.Fatal(err)
			}
			a := &ronOptionalReturnPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "decline"}
			b := &ronOptionalReturnPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			card, err := registry.CreateCard("Zone watcher")
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToGraveyard(card)
			queueRONUpkeep(g, a)
			if scenario == "two-events" {
				queueRONUpkeep(g, a)
				if g.StackSize() != 2 {
					t.Fatal("events merged")
				}
			}
			if scenario == "absent" {
				a.RemoveFromGraveyard(card.ID())
			}
			g.ResolveStack()
			if strings.HasPrefix(scenario, "delayed") {
				if a.offers != 0 || g.FindPermanent(card.ID()) != nil {
					t.Fatal("delayed action ran early")
				}
				if scenario == "delayed-return" {
					a.RemoveFromGraveyard(card.ID())
					a.AddToGraveyard(card)
				}
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
				g.ResolveStack()
			}
			wantEntry := scenario != "decline" && scenario != "absent" && scenario != "delayed-return"
			if (g.FindPermanent(card.ID()) != nil) != wantEntry {
				t.Fatal("wrong frame entry")
			}
			wantLife := 21
			if scenario == "two-events" {
				wantLife = 22
			}
			if a.Life() != wantLife || b.Life() != 20 {
				t.Fatal("independent actions lost controller")
			}
			wantA, wantB := 1, 0
			if scenario == "absent" || scenario == "delayed-return" {
				wantA = 0
			}
			if a.offers != wantA || b.offers != wantB {
				t.Fatalf("offers=%d/%d want=%d/%d", a.offers, b.offers, wantA, wantB)
			}
			if scenario == "decline" {
				a.accept = true
				queueRONUpkeep(g, a)
				g.ResolveStack()
				if g.FindPermanent(card.ID()) == nil || a.offers != 2 {
					t.Fatal("decline changed a later event")
				}
			}
		})
	}
	report, err := InspectFS(fstest.MapFS{"zone.ron": {Data: []byte(sourceZoneFixture("FromGraveyard", action))}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory: %v", err)
	}
	for name, role := range map[string]string{"MayAction": "action", "PutGraveyardCardOntoBattlefield": "action", "ThisGraveyardCard": "card_reference", "EntersNormally": "entry_operation"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
				if signature.Support != "unverified" {
					t.Fatal("inventory claimed gameplay coverage")
				}
			}
		}
		if !found {
			t.Fatalf("missing %s/%s", name, role)
		}
	}
}
