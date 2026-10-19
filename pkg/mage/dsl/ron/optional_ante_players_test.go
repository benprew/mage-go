package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronAntePlayersPlayer struct {
	*mage.BasePlayer
	accept bool
}

func (p *ronAntePlayersPlayer) ChooseMayAbility(string) bool { return p.accept }

func TestRONOptionalAntePlayersGameplay(t *testing.T) {
	for _, choice := range [][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
		a := &ronAntePlayersPlayer{mage.NewBasePlayer("A"), choice[0]}
		b := &ronAntePlayersPlayer{mage.NewBasePlayer("B"), choice[1]}
		g, err := mage.NewGameWithAnte(a, b, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		a.SetLife(5)
		b.SetLife(30)
		g.SetStep(core.PrecombatMain)
		g.SetTurn(1)
		for _, p := range []*ronAntePlayersPlayer{a, b} {
			c := mage.NewArtifact("top", "")
			c.SetOwner(p.PlayerID())
			p.SetLibrary([]mage.Card{c})
		}
		c := repositoryCard(t, "LEG", "Rebirth")
		a.AddToHand(c)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
			t.Fatal("unpaid casting")
		}
		a.ManaPool().Add(core.Green, 6)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 5 || len(a.Ante()) != 0 {
			t.Fatal("payment before resolution")
		}
		g.ResolveStack()
		wantA, wantB := 5, 30
		if choice[0] {
			wantA = 20
		}
		if choice[1] {
			wantB = 20
		}
		if a.Life() != wantA || b.Life() != wantB || (len(a.Ante()) == 1) != choice[0] || (len(b.Ante()) == 1) != choice[1] {
			t.Fatal("paid-player continuation", choice, a.Life(), b.Life())
		}
	}
}

const optionalAntePlayersBody = `EachPlayerMayCost(AnyPlayer,AnteTopCardOfLibrary),EachPlayerAction(PaidCost,SetLifeTotal(Integer(20)))`

func TestRONOptionalAntePlayersInvalid(t *testing.T) {
	for _, body := range []string{
		`EachPlayerAction(PaidCost,SetLifeTotal(Integer(20)))`,
		strings.Replace(optionalAntePlayersBody, "AnyPlayer", "Opponent", 1),
		strings.Replace(optionalAntePlayersBody, "AnyPlayer", "AnyPlayer()", 1),
		strings.Replace(optionalAntePlayersBody, "AnteTopCardOfLibrary)", "AnteTopCardOfLibrary())", 1),
		strings.Replace(optionalAntePlayersBody, "AnteTopCardOfLibrary", "PayLife(Integer(1))", 1),
		strings.Replace(optionalAntePlayersBody, "PaidCost,", "PaidCost(),", 1),
		strings.Replace(optionalAntePlayersBody, "Integer(20)", "X", 1),
		`If(PlayerPassesFilter(You,IsTheirTurn),[` + optionalAntePlayersBody + `]),EachPlayerAction(PaidCost,SetLifeTotal(Integer(20)))`,
	} {
		source := `Card(Name:"invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + body + `]))])`
		if _, err := LoadFS(fstest.MapFS{"invalid.ron": {Data: []byte(source)}}, "*.ron"); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONOptionalAntePlayersCopiesAndDelay(t *testing.T) {
	for _, mode := range []string{"copy", "clone", "delay", "source-left", "source-phased", "repeat"} {
		t.Run(mode, func(t *testing.T) {
			a := &ronAntePlayersPlayer{mage.NewBasePlayer("A"), true}
			b := &ronAntePlayersPlayer{mage.NewBasePlayer("B"), false}
			g, _ := mage.NewGameWithAnte(a, b, nil, nil)
			g.SetTurn(2)
			a.SetLife(5)
			b.SetLife(30)
			for _, p := range []*ronAntePlayersPlayer{a, b} {
				for range 2 {
					c := mage.NewArtifact("top", "")
					c.SetOwner(p.PlayerID())
					p.SetLibrary(append(p.Library(), c))
				}
			}
			body := optionalAntePlayersBody
			if mode == "delay" {
				body = `EachPlayerMayCost(AnyPlayer,AnteTopCardOfLibrary),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([EachPlayerAction(PaidCost,SetLifeTotal(Integer(20)))]))`
			}
			if mode == "repeat" {
				body += `,EachPlayerMayCost(AnyPlayer,AnteTopCardOfLibrary),EachPlayerAction(PaidCost,SetLifeTotal(Integer(10)))`
			}
			factory := permanentScopeFactory(t, "Artifact", `Activated(TapPermanent(ThisPermanent),ActionList([`+body+`]))`)
			c := factory()
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "clone":
				clone := g.Clone()
				clone.SetPlayerAt(0, &ronAntePlayersPlayer{clone.GetPlayer(a.PlayerID()).(*mage.SearchPlayer).BasePlayer, true})
				clone.ResolveStack()
				if a.Life() != 5 || len(a.Ante()) != 0 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("pending clone isolation")
				}
			case "source-left":
				g.BouncePermanentToHand(p)
			case "source-phased":
				g.PhaseOut(p.ID())
			}
			g.ResolveStack()
			if mode == "delay" {
				if a.Life() != 5 || b.Life() != 30 || len(a.Ante()) != 1 {
					t.Fatal("early delayed result")
				}
				a.accept = false
				b.accept = true
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				clone := g.Clone()
				clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				clone.PutTriggersOnStack()
				clone.ResolveStack()
				if a.Life() != 5 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("saved result clone")
				}
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
				g.ResolveStack()
			}
			want, antes := 20, 1
			if mode == "repeat" {
				want = 10
				antes = 2
			}
			if mode == "copy" {
				antes = 2
			}
			if a.Life() != want || b.Life() != 30 || len(a.Ante()) != antes || len(b.Ante()) != 0 {
				t.Fatal("saved payments", a.Life(), b.Life(), len(a.Ante()))
			}
			second := factory()
			if second.ID() == c.ID() {
				t.Fatal("shared factory card")
			}
		})
	}
}

func TestRONOptionalAntePlayersInventoryAndAtomicLoad(t *testing.T) {
	src := `Card(Name:"payment",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + optionalAntePlayersBody + `]))])`
	fs := fstest.MapFS{"valid.ron": {Data: []byte(src)}}
	registry, err := LoadFS(fs, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, _ := registry.CreateCard("payment")
	second, _ := registry.CreateCard("payment")
	if first.ID() == second.ID() {
		t.Fatal("shared definition")
	}
	report, err := InspectFS(fs, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	roles := map[string]string{}
	for _, s := range report.Signatures {
		roles[s.Name] = s.Role
	}
	if roles["AnteTopCardOfLibrary"] != "resolution_cost" || roles["PaidCost"] != "player_collection_reference" || roles["SetLifeTotal"] != "player_action" || roles["Integer"] != "value" {
		t.Fatal("inventory roles", roles)
	}
	old := mage.DefaultRegistry
	fs["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(src, "payment", "invalid", 1) + `Card(Name:"bad",Rules:[Unknown])`)}
	if _, err := LoadFS(fs, "*.ron"); err == nil {
		t.Fatal("partial collection accepted")
	}
	if mage.DefaultRegistry != old {
		t.Fatal("registry changed")
	}
}

func TestRONOptionalAntePlayersContexts(t *testing.T) {
	for _, rules := range []string{
		`AsPermanentEnters(ThisPermanent,[` + optionalAntePlayersBody + `])`,
		`PermanentLayerEffect(ThisPermanent,[` + optionalAntePlayersBody + `])`,
		`Activated(EachPlayerMayCost(AnyPlayer,AnteTopCardOfLibrary),ActionList([DrawACard]))`,
		`Activated(PayMana([]),ActionList([CreateFutureReplaceWouldDraw(You,[` + optionalAntePlayersBody + `])]))`,
	} {
		src := `Card(Name:"invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rules + `])`
		if _, err := LoadFS(fstest.MapFS{"invalid.ron": {Data: []byte(src)}}, "*.ron"); err == nil {
			t.Fatal("invalid context", rules)
		}
	}
}
