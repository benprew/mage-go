package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONHandInspectionGlasses(t *testing.T) {
	for _, scenario := range []string{"normal", "self", "empty", "departed", "phased", "returned", "control", "copy", "clone", "illegal", "unpaid"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "LEA", "Glasses of Urza")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			for _, player := range []*mage.BasePlayer{a, b} {
				card := mage.NewLand(player.Name() + " hand")
				card.SetOwner(player.PlayerID())
				if scenario != "empty" {
					player.AddToHand(card)
				}
			}
			target := b.PlayerID()
			if scenario == "self" {
				target = a.PlayerID()
			}
			if scenario == "illegal" {
				target = uuid.New()
			}
			if scenario == "unpaid" {
				g.TapPermanent(p)
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{target})
			if scenario == "illegal" || scenario == "unpaid" {
				if err == nil || g.StackSize() != 0 || len(g.HandLooks(a.PlayerID())) != 0 || (scenario == "illegal" && p.Tapped) {
					t.Fatal("failed activation changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !p.Tapped || len(g.HandLooks(a.PlayerID())) != 0 {
				t.Fatal("timing or tap payment")
			}
			late := mage.NewLand("Added after announcement")
			late.SetOwner(b.PlayerID())
			if scenario == "normal" {
				b.AddToHand(late)
			}
			switch scenario {
			case "departed", "returned":
				g.BouncePermanentToHand(p)
				if scenario == "returned" {
					a.RemoveFromHand(c.ID())
					g.PutOnBattlefield(c, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "clone":
				clone := g.Clone()
				clone.ResolveStack()
				if len(clone.HandLooks(a.PlayerID())) != 1 || len(g.HandLooks(a.PlayerID())) != 0 {
					t.Fatal("clone shared private history")
				}
			}
			g.ResolveStack()
			want := 1
			if scenario == "normal" {
				want = 2
			}
			if scenario == "empty" {
				want = 0
			}
			looks := g.HandLooks(a.PlayerID())
			if len(looks) != want || len(g.PublicReveals()) != 0 || g.IsHandRevealed(target) {
				t.Fatal("wrong private inspection", len(looks))
			}
			if want > 0 && looks[0].Owner != target {
				t.Fatal("wrong hand")
			}
			copyWant := 0
			if scenario == "copy" {
				copyWant = 1
			}
			if len(g.HandLooks(b.PlayerID())) != copyWant {
				t.Fatal("wrong copied action viewer")
			}
		})
	}
}

func TestRONHandInspectionCasting(t *testing.T) {
	g, a, b := ronGame()
	g.SetStep(core.PrecombatMain)
	c := repositoryCard(t, "LEA", "Glasses of Urza")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
		t.Fatal("unpaid artifact cast")
	}
	if len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("failed casting changed state")
	}
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(c.ID())
	if p == nil || a.ManaPool().TotalMana() != 0 {
		t.Fatal("casting cost or artifact entry")
	}
	b.AddToHand(mage.NewLand("Hidden"))
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if !p.Tapped || len(g.HandLooks(a.PlayerID())) != 1 {
		t.Fatal("new artifact could not inspect")
	}
}

func TestRONHandInspectionInvalidAndReprints(t *testing.T) {
	entries := map[string]*InventorySignature{}
	walkInventory(staticBaseRecord(t, "LEA", "Glasses of Urza"), "Glasses of Urza", "Card", "card", "collection", true, entries)
	found := false
	for _, entry := range entries {
		if entry.Name == "Ref_TargetPlayer" {
			found = true
			if entry.Role != "player_reference" {
				t.Fatal("wrong inspection operand role", entry.Role)
			}
		}
	}
	if !found {
		t.Fatal("inspection operand not inventoried")
	}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		c := repositoryCard(t, set, "Glasses of Urza")
		if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
			t.Fatal(err)
		}
	}
	for _, body := range []string{`LookAtPlayersHand`, `LookAtPlayersHand()`, `LookAtPlayersHand(You,Opponent)`, `LookAtPlayersHand(You())`, `LookAtPlayersHand(Ref_TargetPlayer)`, `LookAtPlayersHand(ThisPermanent)`, `LookAtPlayersHand(Trigger_ThatPlayer)`} {
		if _, err := Compile(parseHandInspection(t, `SpellActions(ActionList([`+body+`]))`)); err == nil {
			t.Fatal("invalid action accepted", body)
		}
	}
	for _, rule := range []string{`LookAtPlayersHand(You)`, `Activated(LookAtPlayersHand(You),ActionList([DrawACard]))`, `AsPermanentEnters(ThisPermanent,[LookAtPlayersHand(You)])`, `PermanentLayerEffect(ThisPermanent,[LookAtPlayersHand(You)])`} {
		if _, err := Compile(parseHandInspection(t, rule)); err == nil {
			t.Fatal("wrong role accepted", rule)
		}
	}
	if _, err := Compile(staticBaseRecord(t, "LEA", "Word of Command")); err == nil || !strings.Contains(err.Error(), "LookAtPlayersHandAndChooseACardForPlayerToPlayControllingThemToDoSo") {
		t.Fatal("incomplete controlled play accepted", err)
	}
}

func parseHandInspection(t *testing.T, rule string) Node {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "hand.ron")
	if err != nil {
		t.Fatal(err)
	}
	return nodes[0]
}

func TestRONHandInspectionInstancesAndRetarget(t *testing.T) {
	a := &ronLibraryViewer{BasePlayer: mage.NewBasePlayer("A")}
	b := &ronLibraryViewer{BasePlayer: mage.NewBasePlayer("B"), retarget: a.PlayerID()}
	g := mage.NewGame(a, b)
	for _, player := range []*ronLibraryViewer{a, b} {
		card := mage.NewLand(player.Name() + " hand")
		card.SetOwner(player.PlayerID())
		player.AddToHand(card)
	}
	first, second := repositoryCard(t, "LEA", "Glasses of Urza"), repositoryCard(t, "LEA", "Glasses of Urza")
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	p, q := g.PutOnBattlefield(first, a.PlayerID()), g.PutOnBattlefield(second, b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	cp := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	if cp == nil || cp.Targets[0] != a.PlayerID() {
		t.Fatal("copy retarget failed")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), q.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(g.HandLooks(a.PlayerID())) != 1 || g.HandLooks(a.PlayerID())[0].Name != "B hand" || len(g.HandLooks(b.PlayerID())) != 2 || g.HandLooks(b.PlayerID())[0].Name != "A hand" || g.HandLooks(b.PlayerID())[1].Name != "A hand" {
		t.Fatal("instances or retargeted copy shared target state")
	}
}

func TestRONHandInspectionDelayedReads(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phased", "control"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			for _, player := range []*mage.BasePlayer{a, b} {
				c := mage.NewLand(player.Name() + " hand")
				c.SetOwner(player.PlayerID())
				player.AddToHand(c)
			}
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([LookAtPlayersHand(ControllerOfPermanent(ThisPermanent))]))]))`)
			c := factory()
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			switch state {
			case "departed", "returned":
				g.BouncePermanentToHand(p)
				if state == "returned" {
					a.RemoveFromHand(c.ID())
					g.PutOnBattlefield(c, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			for range 3 {
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) || len(g.HandLooks(a.PlayerID())) != 0 {
					t.Fatal("query inspected hand or lost source")
				}
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{clone, g} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				owner, want := a.PlayerID(), 1
				if state == "control" {
					owner = b.PlayerID()
				}
				if state == "departed" {
					want = 2
				}
				looks := game.HandLooks(a.PlayerID())
				if len(looks) != want || looks[0].Owner != owner || len(game.HandLooks(b.PlayerID())) != 0 {
					t.Fatal("delayed subject or viewer changed", len(looks))
				}
				if game == clone && len(g.HandLooks(a.PlayerID())) != 0 {
					t.Fatal("clone shared delayed observation")
				}
			}
		})
	}
}

func TestRONHandInspectionSpellAndTrigger(t *testing.T) {
	for _, rule := range []string{`SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([LookAtPlayersHand(Ref_TargetPlayer)])))`, `TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),ActionList([LookAtPlayersHand(Trigger_ThatPlayer)]))`} {
		g, a, b := ronGame()
		c := mage.NewLand("B hand")
		c.SetOwner(b.PlayerID())
		b.AddToHand(c)
		record := parseHandInspection(t, rule)
		spell := strings.HasPrefix(rule, "SpellActions")
		if spell {
			record.Fields["Typeline"].Fields["Cardtypes"] = Node{Kind: List, Args: []Node{{Kind: Symbol, Name: "Instant"}}}
		}
		factory, err := Compile(record)
		if err != nil {
			t.Fatal(err)
		}
		probe := factory()
		probe.SetOwner(a.PlayerID())
		if spell {
			a.AddToHand(probe)
			if err := g.CastSpellByID(a.PlayerID(), probe.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
		} else {
			g.PutOnBattlefield(probe, a.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
		}
		if len(g.HandLooks(a.PlayerID())) != 0 {
			t.Fatal("inspection ran before resolution")
		}
		g.ResolveStack()
		if len(g.HandLooks(a.PlayerID())) != 1 || g.HandLooks(a.PlayerID())[0].Name != "B hand" || len(g.HandLooks(b.PlayerID())) != 0 {
			t.Fatal("spell or event player scope lost")
		}
	}
}
