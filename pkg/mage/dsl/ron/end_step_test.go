package ron

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONEndStepBallLightning(t *testing.T) {
	for _, player := range []int{0, 1} {
		for _, change := range []string{"none", "returned", "control", "phased-before", "phased-after", "phase-in", "countered"} {
			t.Run(fmt.Sprintf("%d/%s", player, change), func(t *testing.T) {
				g, a, b := ronGame()
				c := repositoryCard(t, "DRK", "Ball Lightning")
				if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
					t.Fatal(err)
				}
				c.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
				if p.CurrentPower(g) != 6 || p.CurrentToughness(g) != 1 || !p.HasKeyword(core.Haste) || !p.HasKeyword(core.Trample) {
					t.Fatal("wrong creature characteristics")
				}
				if change == "phased-before" {
					g.PhaseOut(p.ID())
				}
				if change == "phase-in" {
					g.PhaseIn(g.PhaseOut(p.ID()))
				}
				g.SetActivePlayerIndex(player)
				g.SetStep(core.EndStep)
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: g.ActivePlayerObj().PlayerID()})
				g.PutTriggersOnStack()
				want := 1
				if change == "phased-before" {
					want = 0
				}
				if g.StackSize() != want || len(a.Graveyard()) != 0 {
					t.Fatal("end step trigger timing")
				}
				if want == 0 {
					return
				}
				current := p.ID()
				switch change {
				case "returned":
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					a.RemoveFromHand(c.ID())
					current = g.PutOnBattlefield(c, b.PlayerID()).ID()
					if g.StackSize() != 1 {
						t.Fatal("entry during end step triggered")
					}
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "phased-after":
					g.PhaseOut(p.ID())
				case "countered":
					g.GetStack().RemoveByID(g.StackPeek().ID)
				}
				clone := g.Clone()
				clone.ResolveStack()
				if g.FindPermanentIncludingPhased(current) == nil {
					t.Fatal("clone changed live game")
				}
				g.ResolveStack()
				survives := change == "returned" || change == "control" || change == "phased-after" || change == "countered"
				for _, game := range []*mage.Game{g, clone} {
					if (game.FindPermanentIncludingPhased(current) != nil) != survives {
						t.Fatal("sacrifice followed a new incarnation or wrong controller")
					}
					if !survives && (len(game.GetPlayer(a.PlayerID()).Graveyard()) != 1 || len(game.GetPlayer(b.PlayerID()).Graveyard()) != 0) {
						t.Fatal("wrong graveyard")
					}
				}
			})
		}
	}
}

func TestRONEndStepBreedingPit(t *testing.T) {
	for _, change := range []string{"none", "departed", "returned", "control-before-placement", "phased-after", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "FEM", "Breeding Pit")
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent end step triggered")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			if g.StackSize() != 0 || len(g.FilterBattlefield(mage.IsToken)) != 0 {
				t.Fatal("trigger resolved at detection")
			}
			if change == "control-before-placement" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || g.StackPeek().Controller != a.PlayerID() {
				t.Fatal("pending trigger lost controller")
			}
			switch change {
			case "departed", "returned":
				g.DestroyPermanent(g.FindPermanent(p.ID()))
				if change == "returned" {
					g.PutOnBattlefield(c, b.PlayerID())
				}
			case "phased-after":
				g.PhaseOut(p.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			g.ResolveStack()
			tokens := g.FilterBattlefield(mage.IsToken)
			want := 1
			if change == "copy" {
				want = 2
			}
			if len(tokens) != want {
				t.Fatal("wrong token count")
			}
			owners := map[uuid.UUID]int{}
			for _, token := range tokens {
				owners[token.Card.Owner()]++
				if token.ControllerID() != token.Card.Owner() || token.CurrentPower(g) != 0 || token.CurrentToughness(g) != 1 || !token.HasSubType("Thrull") || !slices.Contains(token.Colors(), core.Black) || !token.IsToken {
					t.Fatal("wrong Thrull characteristics")
				}
			}
			if owners[a.PlayerID()] != 1 || (change == "copy" && owners[b.PlayerID()] != 1) {
				t.Fatal("wrong resolving controller")
			}
		})
	}
	for _, choice := range []string{"pay", "decline", "short"} {
		t.Run("upkeep/"+choice, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: choice != "decline"}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			c := repositoryCard(t, "FEM", "Breeding Pit")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			a.ManaPool().Add(core.Black, 2)
			if choice == "short" {
				a.ManaPool().Clear()
				a.ManaPool().Add(core.Black, 1)
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep triggered")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || a.offers != 0 {
				t.Fatal("payment timing")
			}
			g.ResolveStack()
			if (g.FindPermanent(p.ID()) != nil) != (choice == "pay") {
				t.Fatal("wrong upkeep payment outcome")
			}
			want := 0
			if choice == "decline" {
				want = 2
			}
			if choice == "short" {
				want = 1
			}
			if a.ManaPool().TotalMana() != want {
				t.Fatal("wrong mana payment")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			want = 0
			if choice == "pay" {
				want = 1
			}
			if len(g.FilterBattlefield(mage.IsToken)) != want {
				t.Fatal("upkeep and end step interaction")
			}
		})
	}
}

func TestRONEndStepEventBindings(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(AtTheBeginningOfAPlayersEndStep(AnyPlayer), ActionList([PlayerAction(Trigger_ThatPlayer, GainLife(Integer(2))), GainLife(Integer(1))]))`)
	for _, copyAbility := range []bool{false, true} {
		t.Run(fmt.Sprint(copyAbility), func(t *testing.T) {
			g, a, b := ronGame()
			first, second := factory(), factory()
			if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
				t.Fatal(err)
			}
			first.SetOwner(a.PlayerID())
			second.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(first, a.PlayerID())
			q := g.PutOnBattlefield(second, b.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if copyAbility && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			wantStack := 4
			if copyAbility {
				wantStack = 5
			}
			if g.StackSize() != wantStack || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("independent pending events")
			}
			g.DestroyPermanent(g.FindPermanent(p.ID()))
			g.DestroyPermanent(g.FindPermanent(q.ID()))
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone mutated live bindings")
			}
			g.ResolveStack()
			wantA, wantB := 26, 26
			if copyAbility {
				wantA = 29
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatal("event player or resolving controller was overwritten")
			}

			if clone.GetPlayer(a.PlayerID()).Life() != a.Life() || clone.GetPlayer(b.PlayerID()).Life() != b.Life() {
				t.Fatal("clone lost event snapshots")
			}
		})
	}
}

func TestRONEndStepTargetsAndPhaseMembership(t *testing.T) {
	for _, change := range []string{"none", "returned", "phased", "source-phased"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			factory := permanentScopeFactory(t, "Artifact", `TriggerA(AtTheBeginningOfAPlayersEndStep(AnyPlayer), Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([TapPermanent(Ref_TargetPermanent), PlayerAction(Trigger_ThatPlayer, GainLife(Integer(1)))])))`)
			c := factory()
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(c, a.PlayerID())
			victim := repositoryCard(t, "LEB", "Air Elemental")
			victim.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(victim, b.PlayerID())
			if change == "source-phased" {
				g.PhaseOut(source.ID())
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if change == "source-phased" {
				if g.StackSize() != 0 {
					t.Fatal("phased source triggered")
				}
				return
			}
			if g.StackSize() != 1 || len(g.StackPeek().Targets) != 1 || g.StackPeek().Targets[0] != target.ID() {
				t.Fatal("trigger target declaration")
			}
			current := target.ID()
			if change == "returned" {
				g.BouncePermanentToHand(g.FindPermanent(target.ID()))
				b.RemoveFromHand(victim.ID())
				current = g.PutOnBattlefield(victim, b.PlayerID()).ID()
			}
			if change == "phased" {
				g.PhaseOut(target.ID())
			}
			g.ResolveStack()
			if (b.Life() == 21) != (change == "none") || g.FindPermanentIncludingPhased(current).Tapped != (change == "none") {
				t.Fatalf("target legality: life=%d tapped=%v", b.Life(), g.FindPermanentIncludingPhased(current).Tapped)
			}
		})
	}
}

func TestRONEndStepInvalidDeclarations(t *testing.T) {
	for _, rule := range []string{
		`TriggerA(AtTheBeginningOfAPlayersEndStep(AnyPlayer()),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(AtTheBeginningOfAPlayersEndStep(SinglePlayer(Opponent)),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(AtTheBeginningOfAPlayersEndStep(SinglePlayer(HostController)),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(AtTheBeginningOfAPlayersEndStep(SinglePlayer(You())),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(AtTheBeginningOfAPlayersEndStep(),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(AtTheBeginningOfAPlayersEndStep(AnyPlayer),ActionList([PlayerAction(Trigger_ThatPlayer(),GainLife(Integer(1)))]))`,
		`TriggerA(AtTheBeginningOfAPlayersEndStep(AnyPlayer),ActionList([TapPermanent(Trigger_ThatPlayer)]))`,
		`TriggerA(AtTheBeginningOfAPlayersEndStep(AnyPlayer),ActionList([PlayerAction(ControllerOfSpell(Trigger_ThatSpell),GainLife(Integer(1)))]))`,
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))`,
		`TriggerA(AtTheBeginningOfAPlayersEndStep(AnyPlayer),ActionList([GainLife(Integer(1))])),Activated(PayMana([]),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))`,
		`TriggerI(AtTheBeginningOfAPlayersEndStep(AnyPlayer),PermanentPassesFilter(ThisPermanent,IsTapped),ActionList([GainLife(Integer(1))]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid-end-step.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", rule)
		}
	}
}

func TestRONEndStepInventory(t *testing.T) {
	source := `Card(Name:"Step",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(AtTheBeginningOfAPlayersEndStep(SinglePlayer(You)),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))])`
	report, err := InspectFS(fstest.MapFS{"step.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"AtTheBeginningOfAPlayersEndStep": "event", "SinglePlayer": "player_filter", "You": "player_reference", "Trigger_ThatPlayer": "player_reference"}
	for name, role := range want {
		found := false
		for _, s := range report.Signatures {
			if s.Name == name && s.Role == role {
				found = true
				if s.Support != "unverified" {
					t.Fatal("inventory claimed gameplay")
				}
			}
		}
		if !found {
			t.Errorf("missing %s role %s", name, role)
		}
	}
}

func TestRONEndStepNativeTurnAndCombat(t *testing.T) {
	g, a, b := ronGame()
	ball := repositoryCard(t, "DRK", "Ball Lightning")
	ball.SetOwner(a.PlayerID())
	a.AddToHand(ball)
	if err := g.CastSpellByID(a.PlayerID(), ball.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast accepted")
	}
	if g.StackSize() != 0 || len(a.Hand()) != 1 {
		t.Fatal("failed casting changed state")
	}
	a.ManaPool().Add(core.Red, 3)
	if err := g.CastSpellByID(a.PlayerID(), ball.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(ball.ID())
	if p == nil || !p.CanDeclareAsAttacker(g) {
		t.Fatal("haste did not permit immediate attack")
	}
	blocker := repositoryCard(t, "ATQ", "Ornithopter")
	blocker.SetOwner(b.PlayerID())
	q := g.PutOnBattlefield(blocker, b.PlayerID())
	g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
	g.GetCombat().AddBlocker(q.ID(), p.ID())
	g.GetCombat().ResolveDamage(g, false)
	g.CheckStateBasedActions()
	if b.Life() != 16 || g.FindPermanent(q.ID()) != nil || g.FindPermanent(p.ID()) == nil {
		t.Fatal("trample damage or blocker destruction")
	}
	pit := repositoryCard(t, "FEM", "Breeding Pit")
	pit.SetOwner(a.PlayerID())
	a.AddToHand(pit)
	g.SetStep(core.PostcombatMain)
	a.ManaPool().Add(core.Black, 1)
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.CastSpellByID(a.PlayerID(), pit.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.RunStepWithPriority(core.EndStep)
	if g.FindPermanent(p.ID()) != nil || len(g.FilterBattlefield(mage.IsToken)) != 1 || g.StackSize() != 0 {
		t.Fatal("turn step did not dispatch and resolve triggers")
	}
	late := repositoryCard(t, "DRK", "Ball Lightning")
	late.SetOwner(a.PlayerID())
	later := g.PutOnBattlefield(late, a.PlayerID())
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("late entry replayed end step")
	}
	g.RunStepWithPriority(core.Cleanup)
	if g.FindPermanent(later.ID()) == nil {
		t.Fatal("cleanup acted as end step")
	}
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.EndStep)
	if g.FindPermanent(later.ID()) != nil || len(g.FilterBattlefield(mage.IsToken)) != 1 {
		t.Fatal("next opponent end step")
	}
}

func TestRONEndStepExplicitPlayerConsumers(t *testing.T) {
	for _, action := range []string{"GainLife(Integer(2))", "LoseLife(Integer(2))", "DrawACard", "DiscardACard"} {
		t.Run(action, func(t *testing.T) {
			g, a, b := ronGame()
			for range 3 {
				c := repositoryCard(t, "LEB", "Forest")
				c.SetOwner(b.PlayerID())
				b.AddToLibrary(c)
			}
			inHand := repositoryCard(t, "LEB", "Island")
			inHand.SetOwner(b.PlayerID())
			b.AddToHand(inHand)
			factory := permanentScopeFactory(t, "Artifact", `TriggerA(AtTheBeginningOfAPlayersEndStep(AnyPlayer),ActionList([PlayerAction(Trigger_ThatPlayer,`+action+`)]))`)
			source := factory()
			source.SetOwner(a.PlayerID())
			g.PutOnBattlefield(source, a.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if len(g.StackPeek().Targets) != 0 {
				t.Fatal("event player became target")
			}
			g.ResolveStack()
			if a.Life() != 20 || len(a.Hand()) != 0 || len(a.Graveyard()) != 0 {
				t.Fatal("action used ability controller")
			}
			switch action {
			case "GainLife(Integer(2))":
				if b.Life() != 22 {
					t.Fatal("wrong life recipient")
				}
			case "LoseLife(Integer(2))":
				if b.Life() != 18 {
					t.Fatal("wrong life recipient")
				}
			case "DrawACard":
				if len(b.Hand()) != 2 || len(b.Library()) != 2 {
					t.Fatal("wrong draw recipient")
				}
			case "DiscardACard":
				if len(b.Hand()) != 0 || len(b.Graveyard()) != 1 {
					t.Fatal("wrong discard recipient")
				}
			}
		})
	}
}

func TestRONEndStepCurrentController(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "FEM", "Breeding Pit")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("your end step used owner")
	}
	g.PhaseOut(source.ID())
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.StackPeek().Controller != b.PlayerID() {
		t.Fatal("new controller end step did not trigger")
	}
	g.ResolveStack()
	tokens := g.FilterBattlefield(mage.IsToken)
	if len(tokens) != 1 || tokens[0].ControllerID() != b.PlayerID() || tokens[0].Card.Owner() != b.PlayerID() {
		t.Fatal("token followed source owner")
	}
}
