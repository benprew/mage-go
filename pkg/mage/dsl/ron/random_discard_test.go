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

func randomRONHand(t *testing.T, p *mage.BasePlayer, count int) []mage.Card {
	t.Helper()
	cards := make([]mage.Card, count)
	for i := range cards {
		cards[i] = repositoryCard(t, "LEB", "Forest")
		cards[i].SetOwner(p.PlayerID())
		p.AddToHand(cards[i])
	}
	return cards
}

func TestRONRandomDiscardSpells(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		x, amount int
	}{
		{"FEM", "Hymn to Tourach", 0, 2},
		{"LEA", "Mind Twist", 3, 3}, {"LEB", "Mind Twist", 3, 3}, {"2ED", "Mind Twist", 3, 3},
		{"3ED", "Mind Twist", 3, 3}, {"CED", "Mind Twist", 3, 3}, {"CEI", "Mind Twist", 3, 3},
		{"LEB", "Mind Twist", 0, 0},
	} {
		for _, initial := range []int{0, 1, 4} {
			t.Run(fmt.Sprintf("%s/%s/%d/%d", tc.set, tc.name, tc.x, initial), func(t *testing.T) {
				a, b := &ronDiscardPlayer{BasePlayer: mage.NewBasePlayer("A")}, &ronDiscardPlayer{BasePlayer: mage.NewBasePlayer("B")}
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				hand := randomRONHand(t, b.BasePlayer, initial)
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(core.Black, 2)
				a.ManaPool().Add(core.Colorless, tc.x)
				g.SetRandomResults([]int{2, 0, 0, 7})
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, tc.x); err != nil {
					t.Fatal(err)
				}
				if len(b.Hand()) != initial || len(b.Graveyard()) != 0 {
					t.Fatal("discard occurred during casting")
				}
				clone := g.Clone()
				g.ResolveStack()
				want := min(initial, tc.amount)
				if len(b.Hand()) != initial-want || len(b.Graveyard()) != want || len(a.Graveyard()) != 1 || len(a.requests)+len(b.requests) != 0 {
					t.Fatal("wrong discard recipient or choice")
				}
				if want > 0 && b.Graveyard()[0].ID() != hand[2%initial].ID() {
					t.Fatal("wrong first random card")
				}
				seen := map[uuid.UUID]bool{}
				for _, c := range b.Graveyard() {
					if seen[c.ID()] {
						t.Fatal("duplicate discard")
					}
					seen[c.ID()] = true
				}
				if len(clone.GetPlayer(b.PlayerID()).Hand()) != initial {
					t.Fatal("live resolution changed clone")
				}
				clone.ResolveStack()
				cp := clone.GetPlayer(b.PlayerID())
				for i, c := range cp.Graveyard() {
					if c.ID() != b.Graveyard()[i].ID() {
						t.Fatal("clone lost random queue or target")
					}
				}
				if len(cp.Hand()) != len(b.Hand()) {
					t.Fatal("clone lost amount")
				}
				if clone.RandIntn(10) != g.RandIntn(10) {
					t.Fatal("clone random consumption differs")
				}
			})
		}
	}
}

func TestRONRandomDiscardRingLifecycle(t *testing.T) {
	for _, change := range []string{"none", "departed", "returned", "control", "phased", "empty"} {
		t.Run(change, func(t *testing.T) {
			a := &ronDiscardPlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			initial := 3
			if change == "empty" {
				initial = 0
			}
			hand := randomRONHand(t, a.BasePlayer, initial)
			fillRONLibrary(t, a.BasePlayer, 4)
			card := repositoryCard(t, "FEM", "Ring of Renewal")
			card.SetOwner(b.PlayerID())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			a.ManaPool().Add(core.Colorless, 5)
			g.SetRandomResults([]int{2, 7})
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if !p.Tapped || a.ManaPool().TotalMana() != 0 || len(a.Library()) != 4 || len(a.Hand()) != initial {
				t.Fatal("activation timing")
			}
			switch change {
			case "departed", "returned":
				g.DestroyPermanent(p)
				if change == "returned" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "control":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), p.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "phased":
				g.PhaseOut(p.ID())
			}
			clone := g.Clone()
			g.ResolveStack()
			want := min(initial, 1)
			if len(a.Hand()) != initial-want+2 || len(a.Library()) != 2 || len(a.Graveyard()) != want || len(b.Hand()) != 0 || len(a.requests) != 0 {
				t.Fatal("discard/draw order or captured controller")
			}
			if want > 0 && a.Graveyard()[0].ID() != hand[2].ID() {
				t.Fatal("selected a newly drawn card")
			}
			clone.ResolveStack()
			if len(clone.GetPlayer(a.PlayerID()).Hand()) != len(a.Hand()) || len(clone.GetPlayer(a.PlayerID()).Library()) != 2 {
				t.Fatal("clone lost activation")
			}
			next := 7
			if initial == 0 {
				next = 2
			}
			if g.RandIntn(10) != next {
				t.Fatal("empty discard consumed randomness")
			}
		})
	}
}

func TestRONRandomDiscardInvalid(t *testing.T) {
	for _, action := range []string{
		`DiscardACardAtRandom()`, `DiscardACardAtRandom(Integer(1))`, `DiscardNumberCardsAtRandom`,
		`DiscardNumberCardsAtRandom(Integer(-1))`, `DiscardNumberCardsAtRandom(2)`,
		`DiscardNumberCardsAtRandom(Integer())`, `DiscardNumberCardsAtRandom(Integer(1),Integer(2))`,
		`DiscardNumberCardsAtRandom(ValueX)`, `DiscardNumberCardsAtRandom(TheNumberOfCountersRemovedThisWay)`,
		`PlayerAction(Ref_TargetPlayer,DiscardACardAtRandom)`, `PlayerAction(ThisPermanent,DiscardACardAtRandom)`,
		`PlayerAction(Trigger_ThatPlayer,DiscardACardAtRandom)`,
	} {
		src := `Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), ActionList([` + action + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", action)
		}
	}
}

func TestRONRandomDiscardInventory(t *testing.T) {
	src := `Card(Name: "Discard", Typeline: (Supertypes: [], Cardtypes: [Sorcery], Subtypes: []), Rules: [SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,DiscardNumberCardsAtRandom(Integer(2)))])))])`
	report, err := InspectFS(fstest.MapFS{"discard.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	seen := false
	for _, s := range report.Signatures {
		if s.Name == "Integer" {
			seen = true
			if s.Role != "value" {
				t.Fatalf("amount role = %s", s.Role)
			}
		}
	}
	if !seen {
		t.Fatal("missing amount")
	}
}

func TestRONRandomDiscardCopyAndInstances(t *testing.T) {
	g, a, b := ronGame()
	randomRONHand(t, a, 5)
	randomRONHand(t, b, 5)
	first, second := repositoryCard(t, "LEB", "Mind Twist"), repositoryCard(t, "LEB", "Mind Twist")
	if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("factory shared target state")
	}
	nodes, err := Parse(strings.NewReader(`Card(Name: "Instant Discard", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), ManaCost: [ManaCostX, ManaCostB], Rules: [SpellActions(Targeted([TargetPlayer(AnyPlayer)], ActionList([PlayerAction(Ref_TargetPlayer, DiscardNumberCardsAtRandom(ValueX))])))])`), "copy.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	second = factory()
	for _, c := range []mage.Card{first, second} {
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
	}
	a.ManaPool().Add(core.Black, 2)
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{b.PlayerID()}, 2); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByID(a.PlayerID(), second.ID(), []uuid.UUID{a.PlayerID()}, 1); err != nil {
		t.Fatal(err)
	}
	original := g.FindStackObject(first.ID())
	g.SetRandomResults([]int{0, 0, 0, 0})
	clone := g.Clone()
	copied := g.CopySpellOnStack(first.ID(), b.PlayerID(), true)
	if copied == nil || copied.XValue != 2 || copied.Targets[0] != a.PlayerID() || original.Targets[0] != b.PlayerID() {
		t.Fatal("copy lost target or X")
	}
	g.ResolveStack()
	if len(a.Hand()) != 2 || len(b.Hand()) != 3 {
		t.Fatalf("copy/instance hands %d/%d", len(a.Hand()), len(b.Hand()))
	}
	clone.ResolveStack()
	if len(clone.GetPlayer(a.PlayerID()).Hand()) != 4 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 3 || len(a.Hand()) != 2 {
		t.Fatal("clone shared copied state")
	}
}

func TestRONRandomDiscardRollback(t *testing.T) {
	for _, failure := range []string{"mana", "target", "ring-mana", "tapped", "phased"} {
		t.Run(failure, func(t *testing.T) {
			g, a, b := ronGame()
			randomRONHand(t, b, 3)
			g.SetRandomResults([]int{7})
			card := repositoryCard(t, "LEB", "Mind Twist")
			card.SetOwner(a.PlayerID())
			if failure == "mana" || failure == "target" {
				a.AddToHand(card)
				a.ManaPool().Add(core.Black, 1)
				target := b.PlayerID()
				if failure == "target" {
					a.ManaPool().Add(core.Colorless, 3)
					target = uuid.New()
				}
				before := a.ManaPool().TotalMana()
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 3); err == nil {
					t.Fatal("illegal cast succeeded")
				}
				if len(a.Hand()) != 1 || a.ManaPool().TotalMana() != before {
					t.Fatal("cast did not roll back")
				}
			} else {
				card = repositoryCard(t, "FEM", "Ring of Renewal")
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				mana := 5
				if failure == "ring-mana" {
					mana = 4
				}
				a.ManaPool().Add(core.Colorless, mana)
				switch failure {
				case "tapped":
					g.TapPermanent(p)
				case "phased":
					g.PhaseOut(p.ID())
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err == nil {
					t.Fatal("illegal activation succeeded")
				}
				if a.ManaPool().TotalMana() != mana {
					t.Fatal("activation consumed mana")
				}
			}
			if !g.GetStack().IsEmpty() || len(b.Hand()) != 3 || len(b.Graveyard()) != 0 || g.RandIntn(10) != 7 {
				t.Fatal("failure changed resolution or randomness")
			}
		})
	}
}

func TestRONRandomDiscardReplacement(t *testing.T) {
	a, b := &ronDiscardPlayer{BasePlayer: mage.NewBasePlayer("A")}, &ronDiscardPlayer{BasePlayer: mage.NewBasePlayer("B")}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	hand := randomRONHand(t, b.BasePlayer, 3)
	g.PutOnBattlefield(mage.NewArtifact("Replacement", "", mage.WithStaticAbility(mage.DiscardToLibraryReplacement())), b.PlayerID())
	events := 0
	watcher := mage.NewTriggered(core.EvtDiscard, false).SetCondition(func(evt *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
		events++
		if evt.PlayerID != b.PlayerID() || len(b.Hand()) != 1 || len(b.Library()) != 2 || len(b.Graveyard()) != 0 {
			t.Error("event observed partial replacement")
		}
		return false
	})
	g.PutOnBattlefield(mage.NewArtifact("Watcher", "", mage.WithAbility(watcher)), a.PlayerID())
	card := repositoryCard(t, "FEM", "Hymn to Tourach")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 2)
	g.SetRandomResults([]int{2, 0, 7})
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if events != 2 || b.Hand()[0].ID() != hand[1].ID() || b.Library()[0].ID() != hand[2].ID() || b.Library()[1].ID() != hand[0].ID() || g.RandIntn(10) != 7 {
		t.Fatal("replacement bypassed selected batch")
	}
}

func TestRONRandomDiscardGwendlyn(t *testing.T) {
	for _, scenario := range []string{"self", "opponent", "wrong-turn", "sick", "tapped", "phased", "new-controller", "departed", "returned", "control-after"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			randomRONHand(t, a, 3)
			hand := randomRONHand(t, b, 3)
			card := repositoryCard(t, "LEG", "Gwendlyn Di Corci")
			card.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			if p.CurrentPower(g) != 3 || p.CurrentToughness(g) != 5 || !mage.IsLegendary.Match(p, g) {
				t.Fatal("creature metadata")
			}
			target, payer := b, a
			switch scenario {
			case "self":
				target = a
			case "wrong-turn":
				g.SetActivePlayerIndex(1)
			case "sick":
				p.GrantBaseAttr(core.AttrSummonSick)
			case "tapped":
				g.TapPermanent(p)
			case "phased":
				g.PhaseOut(p.ID())
			case "new-controller":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
				g.SetActivePlayerIndex(1)
				payer = b
				target = a
			}
			g.SetStep(core.DeclareBlockers)
			g.SetRandomResults([]int{2, 7})
			err := g.ActivateAbilityByIndex(payer.PlayerID(), p.ID(), 0, []uuid.UUID{target.PlayerID()})
			if scenario == "wrong-turn" || scenario == "sick" || scenario == "tapped" || scenario == "phased" {
				if err == nil || !g.GetStack().IsEmpty() || len(a.Hand()) != 3 || len(b.Hand()) != 3 || g.RandIntn(10) != 2 {
					t.Fatal("illegal activation changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(target.Hand()) != 3 || !g.FindPermanent(p.ID()).Tapped {
				t.Fatal("activation timing")
			}
			switch scenario {
			case "departed", "returned":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(card.ID())
				if scenario == "returned" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "control-after":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.SetActivePlayerIndex(1 - g.ActivePlayerIndex())
			g.SetStep(core.EndStep)
			clone := g.Clone()
			g.ResolveStack()
			if len(target.Hand()) != 2 || len(target.Graveyard()) != 1 || g.RandIntn(10) != 7 {
				t.Fatal("lost target, amount, or resolution timing")
			}
			if target == b && target.Graveyard()[0].ID() != hand[2].ID() {
				t.Fatal("wrong random selection")
			}
			clone.ResolveStack()
			if len(clone.GetPlayer(target.PlayerID()).Hand()) != 2 || len(target.Hand()) != 2 {
				t.Fatal("clone lost target or shared hand")
			}
		})
	}
}

func TestRONRandomDiscardContexts(t *testing.T) {
	for _, rule := range []string{
		`SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,DiscardACardAtRandom),DiscardACardAtRandom])))`,
		`Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,DiscardACardAtRandom),DiscardACardAtRandom])))`,
		`TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([DiscardACardAtRandom]))`,
		`TriggerA(AtTheBeginningOfAPlayersEndStep(AnyPlayer),ActionList([PlayerAction(Trigger_ThatPlayer,DiscardNumberCardsAtRandom(Integer(2))),DiscardACardAtRandom]))`,
	} {
		t.Run(rule, func(t *testing.T) {
			kind := "Artifact"
			if strings.HasPrefix(rule, "SpellActions") {
				kind = "Instant"
			}
			factory := permanentScopeFactory(t, kind, rule)
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			randomRONHand(t, a, 3)
			randomRONHand(t, b, 3)
			card.SetOwner(a.PlayerID())
			wantA, wantB := 1, 0
			if kind == "Instant" {
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
				wantB = 1
			} else {
				p := g.PutOnBattlefield(card, a.PlayerID())
				if strings.HasPrefix(rule, "Activated") {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
						t.Fatal(err)
					}
					wantB = 1
				}
				if strings.Contains(rule, "EndStep") {
					g.SetActivePlayerIndex(1)
					g.SetStep(core.EndStep)
					g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
					g.PutTriggersOnStack()
					wantB = 2
				}
			}
			g.SetRandomResults([]int{0, 0, 0, 7})
			g.ResolveStack()
			if len(a.Hand()) != 3-wantA || len(b.Hand()) != 3-wantB {
				t.Fatalf("scoped hands %d/%d", len(a.Hand()), len(b.Hand()))
			}
		})
	}
}

func TestRONRandomDiscardModes(t *testing.T) {
	for _, mode := range []int{0, 1} {
		g, a, b := ronModalGame(mode)
		randomRONHand(t, a.BasePlayer, 3)
		randomRONHand(t, b, 3)
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Modal_ChooseOne([Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,DiscardNumberCardsAtRandom(Integer(2)))])),ActionList([DiscardACardAtRandom])]))`)
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		targets := []uuid.UUID{b.PlayerID()}
		if mode == 1 {
			targets = nil
		}
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err != nil {
			t.Fatal(err)
		}
		if a.prompts != 1 || len(a.Hand()) != 3 || len(b.Hand()) != 3 {
			t.Fatal("mode did not select at announcement")
		}
		g.SetRandomResults([]int{0, 0, 7})
		g.ResolveStack()
		wantA, wantB := 3, 1
		if mode == 1 {
			wantA, wantB = 2, 3
		}
		if a.prompts != 1 || len(a.Hand()) != wantA || len(b.Hand()) != wantB {
			t.Fatal("inactive mode executed or lost its player")
		}
	}
}
