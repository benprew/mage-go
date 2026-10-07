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

func TestRONPaidUntapCards(t *testing.T) {
	for _, tc := range []struct {
		name, set string
		cost      int
		color     core.Color
	}{
		{"Magnetic Mountain", "3ED", 4, core.Colorless}, {"Magnetic Mountain", "ARN", 4, core.Colorless}, {"Thelon's Curse", "FEM", 1, core.Blue},
	} {
		for _, scenario := range []string{"paid", "decline", "unaffordable", "source-departed", "source-control", "control", "phase", "returned", "copy", "clone"} {
			t.Run(tc.name+"/"+tc.set+"/"+scenario, func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "decline"}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				g.SetStep(core.PrecombatMain)
				g.SetActivePlayerIndex(1)
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(b.PlayerID())
				b.AddToHand(card)
				if err := g.CastSpellByID(b.PlayerID(), card.ID(), nil, 0); err == nil {
					t.Fatal("unpaid cast accepted")
				}
				if tc.name == "Magnetic Mountain" {
					b.ManaPool().Add(core.Red, 2)
					b.ManaPool().Add(core.Colorless, 1)
				} else {
					b.ManaPool().Add(core.Green, 2)
				}
				if err := g.CastSpellByID(b.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				fixture := repositoryCard(t, "LEB", "Prodigal Sorcerer")
				fixture.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(fixture, a.PlayerID())
				other := g.PutOnBattlefield(repositoryCard(t, "LEB", "Prodigal Sorcerer"), b.PlayerID())
				nonblue := g.PutOnBattlefield(repositoryCard(t, "LEB", "Grizzly Bears"), a.PlayerID())
				for _, p := range []*mage.Permanent{target, other, nonblue} {
					g.TapPermanent(p)
				}
				g.SetActivePlayerIndex(0)
				g.RunStepWithPriority(core.Untap)
				if !g.FindPermanent(target.ID()).Tapped || g.FindPermanent(nonblue.ID()).Tapped {
					t.Fatal("untap-step restriction")
				}
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
				if g.StackSize() != 1 || !g.FindPermanent(target.ID()).Tapped {
					t.Fatal("trigger timing")
				}
				if scenario != "unaffordable" {
					a.ManaPool().Add(tc.color, tc.cost)
				}
				switch scenario {
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: card.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "source-departed":
					g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: target.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "phase":
					g.PhaseOut(target.ID())
				case "returned":
					g.BouncePermanentToHand(target)
					b.RemoveFromHand(fixture.ID())
					target = g.PutOnBattlefield(fixture, a.PlayerID())
					g.TapPermanent(target)
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				if scenario == "clone" {
					clone.SetPlayerAt(0, &ronPaymentPlayer{BasePlayer: clone.GetPlayer(a.PlayerID()).(*mage.SearchPlayer).BasePlayer, accept: true})
					clone.ResolveStack()
					if clone.FindPermanent(target.ID()).Tapped || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
						t.Fatal("clone lost captured event player or payment")
					}
					if !g.FindPermanent(target.ID()).Tapped || a.ManaPool().TotalMana() != tc.cost {
						t.Fatal("clone changed original")
					}
				}
				g.ResolveStack()
				g.PhaseIn([]uuid.UUID{target.ID()})
				paid := scenario != "decline" && scenario != "unaffordable" && scenario != "control" && scenario != "phase"
				if g.FindPermanent(target.ID()).Tapped == paid || !g.FindPermanent(other.ID()).Tapped {
					t.Fatal("wrong player or collection")
				}
				want := tc.cost
				if paid || scenario == "unaffordable" {
					want = 0
				}
				if a.ManaPool().TotalMana() != want {
					t.Fatalf("remaining mana=%d want=%d", a.ManaPool().TotalMana(), want)
				}
			})
		}
	}
}

func TestRONPaidUntapInvalid(t *testing.T) {
	choose := `PlayerAction(You,ChooseAnyNumberPermanentsAndPayManaForEach(And([IsTapped,ControlledByPlayer(PlayerAction_ThatPlayer)]),[ManaCostU]))`
	consume := `PlayerAction(You,UntapEachPermanent(TheChosenPermanents))`
	for _, body := range []string{
		choose, consume, consume + "," + choose, choose + `,GainLife(Integer(1)),` + consume,
		choose + `,` + strings.Replace(consume, "PlayerAction(You,", "PlayerAction(Ref_TargetPlayer,", 1),
		strings.Replace(choose, "PlayerAction_ThatPlayer", "Trigger_ThatPlayer", 1) + "," + consume,
		strings.Replace(choose, "[ManaCostU]", "[ManaCostX]", 1) + "," + consume,
		strings.Replace(consume, "TheChosenPermanents", "TheChosenPermanents()", 1) + "," + choose,
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([%s]))])`, body)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONPaidUntapPlayerAndProcedureIsolation(t *testing.T) {
	a := mage.NewBasePlayer("A")
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
	g := mage.NewGame(a, b)
	g.SetTurn(1)
	g.SetStep(core.PrecombatMain)
	g.PutOnBattlefield(repositoryCard(t, "FEM", "Thelon's Curse"), a.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "3ED", "Magnetic Mountain"), a.PlayerID())
	first := g.PutOnBattlefield(repositoryCard(t, "LEB", "Prodigal Sorcerer"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEB", "Prodigal Sorcerer"), b.PlayerID())
	g.TapPermanent(first)
	g.TapPermanent(second)
	b.ManaPool().Add(core.Blue, 1)
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("each upkeep must trigger each rule")
	}
	g.ResolveStack()
	if !g.FindPermanent(first.ID()).Tapped || g.FindPermanent(second.ID()).Tapped || b.ManaPool().TotalMana() != 0 {
		t.Fatal("event player or independent payment result")
	}
}

func TestRONPaidUntapInventoryAndStrictLoading(t *testing.T) {
	src := `Card(Name:"Paid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([PlayerAction(You,ChooseAnyNumberPermanentsAndPayManaForEach(And([IsTapped,ControlledByPlayer(PlayerAction_ThatPlayer)]),[ManaCostU])),PlayerAction(You,UntapEachPermanent(TheChosenPermanents))]))])`
	fs := fstest.MapFS{"paid.ron": {Data: []byte(src)}}
	registry, err := LoadFS(fs, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("missing complete card")
	}
	report, err := InspectFS(fs, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"ChooseAnyNumberPermanentsAndPayManaForEach": "player_action", "ControlledByPlayer": "permanent_filter", "PlayerAction_ThatPlayer": "player_reference", "TheChosenPermanents": "permanent_collection_reference", "ManaCostU": "metadata"} {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s/%s", name, role)
		}
	}
	fs["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(src, "Name:\"Paid\"", "Name:\"Bad\"", 1))}
	fs["bad.ron"].Data = []byte(strings.Replace(string(fs["bad.ron"].Data), "TheChosenPermanents", "TheChosenPermanents()", 1))
	if partial, err := LoadFS(fs, "*.ron"); err == nil || partial != nil {
		t.Fatal("invalid collection published a partial registry")
	}
}

func TestRONPaidUntapControllerReference(t *testing.T) {
	src := `Card(Name:"Controller",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([PlayerAction(ControllerOfPermanent(ThisPermanent),ChooseAnyNumberPermanentsAndPayManaForEach(And([IsTapped,ControlledByPlayer(PlayerAction_ThatPlayer)]),[])),PlayerAction(ControllerOfPermanent(ThisPermanent),UntapEachPermanent(TheChosenPermanents))]))])`
	nodes, err := Parse(strings.NewReader(src), "controller.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	first, second := factory(), factory()
	if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
		t.Fatal(err)
	}
	if first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("factory shared declaration")
	}
}
