package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

var damagePlayerCards = []struct{ set, name string }{{"LEA", "Hypnotic Specter"}, {"LEB", "Hypnotic Specter"}, {"2ED", "Hypnotic Specter"}, {"3ED", "Hypnotic Specter"}, {"CED", "Hypnotic Specter"}, {"CEI", "Hypnotic Specter"}, {"LEG", "Nicol Bolas"}}

func TestRONDamagePlayerCards(t *testing.T) {
	for _, tc := range damagePlayerCards {
		for _, state := range []string{"normal", "self", "permanent", "other", "zero", "prevented", "partial", "twice", "leave", "return", "phase-after", "phase-before", "control-before", "control-after", "removed", "copy", "empty"} {
			t.Run(tc.set+"/"+tc.name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				randomRONHand(t, a, 4)
				hand := randomRONHand(t, b, 4)
				if state == "empty" {
					b.SetHand(nil)
				}
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(tc.name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				if !source.HasKeyword(core.Flying) {
					t.Fatal("missing flying")
				}
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 9), b.PlayerID())
				if state == "phase-before" {
					g.PhaseOut(source.ID())
				}
				if state == "removed" {
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
				}
				if state == "control-before" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				if state == "prevented" {
					g.AddPreventionShield(b.PlayerID(), 3)
				}
				if state == "partial" {
					g.AddPreventionShield(b.PlayerID(), 2)
				}
				g.SetRandomResults([]int{2, 0, 1, 0})
				switch state {
				case "self":
					g.DealDamageToPlayer(a, 3, source.ID())
				case "permanent":
					g.DealDamageToPermanent(other, 3, source.ID())
				case "other":
					g.DealDamageToPlayer(b, 3, other.ID())
				case "zero":
					g.DealDamageToPlayer(b, 0, source.ID())
				default:
					g.DealDamageToPlayer(b, 3, source.ID())
				}
				expected := 1
				switch state {
				case "self", "permanent", "other", "zero", "prevented", "phase-before", "control-before", "removed":
					expected = 0
				}
				if state == "twice" {
					g.DealDamageToPlayer(b, 1, source.ID())
					expected = 2
				}
				if len(b.Hand()) != len(hand) && state != "empty" {
					t.Fatal("early discard")
				}
				if state == "leave" || state == "return" {
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					if state == "return" {
						g.PutOnBattlefield(card, b.PlayerID())
					}
				}
				if state == "phase-after" {
					g.PhaseOut(source.ID())
				}
				if state == "control-after" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				clone := g.Clone()
				g.PutTriggersOnStack()
				if g.StackSize() != expected {
					t.Fatalf("stack %d, want %d", g.StackSize(), expected)
				}
				if state == "copy" {
					g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
					expected++
				}
				g.ResolveStack()
				want := 0
				if expected > 0 {
					want = expected
					if tc.name == "Nicol Bolas" {
						want = 4
					}
				}
				if state == "empty" {
					want = 0
				}
				if len(a.Hand()) != 4 || len(b.Graveyard()) != want || len(b.Hand()) != len(hand)-want && state != "empty" {
					t.Fatalf("hands %d/%d grave %d, want %d", len(a.Hand()), len(b.Hand()), len(b.Graveyard()), want)
				}
				if want > 0 && tc.name == "Hypnotic Specter" && b.Graveyard()[0].ID() != hand[2].ID() {
					t.Fatal("random card not selected")
				}
				if len(clone.GetPlayer(b.PlayerID()).Graveyard()) != 0 {
					t.Fatal("live resolution changed clone")
				}
				clone.PutTriggersOnStack()
				clone.ResolveStack()
				cloneWant := want
				if state == "copy" && tc.name == "Hypnotic Specter" {
					cloneWant--
				}
				if len(clone.GetPlayer(b.PlayerID()).Graveyard()) != cloneWant {
					t.Fatal("pending clone lost event player")
				}
			})
		}
	}
}

func TestRONDamagePlayerInvalid(t *testing.T) {
	for _, event := range []string{`WhenAPermanentDealsDamageToAPlayer`, `WhenAPermanentDealsDamageToAPlayer(SinglePermanent(ThisPermanent))`, `WhenAPermanentDealsDamageToAPlayer(SinglePermanent(ThisPermanent),Opponent())`, `WhenAPermanentDealsDamageToAPlayer(SinglePermanent(ThisPermanent),AnyPlayer)`, `WhenAPermanentDealsDamageToAPlayer(SinglePermanent(HostPermanent),Opponent)`, `WhenAPermanentDealsDamageToAPlayer(SinglePermanent(Ref_TargetPermanent),Opponent)`, `WhenAPermanentDealsDamageToAPlayer(IsCardtype(Creature),Opponent)`} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[TriggerA(` + event + `,ActionList([PlayerAction(Trigger_ThatPlayer,DiscardHand)]))])`
		nodes, err := Parse(strings.NewReader(src), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", event)
		}
	}
	for _, rules := range []string{`Activated(PayMana([]),ActionList([PlayerAction(Trigger_ThatPlayer,DiscardHand)]))`, `TriggerA(WhenAPermanentDealsDamageToAPlayer(SinglePermanent(ThisPermanent),Opponent),ActionList([PlayerAction(Trigger_ThatPlayer(),DiscardHand)]))`, `TriggerA(WhenAPermanentDealsDamageToAPlayer(SinglePermanent(ThisPermanent),Opponent),ActionList([PlayerAction(Trigger_ThatPlayer,DiscardHand())]))`, `TriggerA(WhenAPermanentDealsDamageToAPlayer(SinglePermanent(ThisPermanent),Opponent),ActionList([GainLife(Trigger_AmountOfDamageDealt)]))`} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rules + `])`
		nodes, err := Parse(strings.NewReader(src), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rules)
		}
	}
}

func TestRONDamagePlayerInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(WhenAPermanentDealsDamageToAPlayer(SinglePermanent(ThisPermanent),Opponent),ActionList([PlayerAction(Trigger_ThatPlayer,DiscardHand)]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal(report.Issues)
	}
	for name, role := range map[string]string{"SinglePermanent": "permanent_filter", "ThisPermanent": "permanent_reference", "Opponent": "player_filter", "Trigger_ThatPlayer": "player_reference", "DiscardHand": "player_action"} {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s: %s", name, role)
		}
	}
}

func TestRONDamagePlayerCombat(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEB", "Hypnotic Specter"}, {"LEG", "Nicol Bolas"}} {
		for _, blocked := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%v", tc.name, blocked), func(t *testing.T) {
				a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				for range 3 {
					c := mage.NewLand("Hand")
					c.SetOwner(b.PlayerID())
					b.AddToHand(c)
				}
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 9, mage.WithKeyword(core.Flying)), b.PlayerID())
				a.SetAttackers(1, []string{tc.name})
				if blocked {
					b.SetBlockers(1, map[string]string{"Blocker": tc.name})
				}
				g.RunStepWithPriority(core.DeclareAttackers)
				g.RunStepWithPriority(core.DeclareBlockers)
				g.RunStepWithPriority(core.CombatDamage)
				want := 1
				if tc.name == "Nicol Bolas" {
					want = 3
				}
				if blocked {
					want = 0
				}
				if len(b.Graveyard()) != want {
					t.Fatal("combat discard", len(b.Graveyard()), want)
				}
				life := 20 - card.Power()
				if blocked {
					life = 20
				}
				if b.Life() != life {
					t.Fatal("combat damage", b.Life())
				}
			})
		}
	}
}

func TestRONDamagePlayerCasting(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		colors    []core.Color
	}{{"LEB", "Hypnotic Specter", []core.Color{core.Colorless, core.Black, core.Black}}, {"LEG", "Nicol Bolas", []core.Color{core.Colorless, core.Colorless, core.Blue, core.Blue, core.Black, core.Black, core.Red, core.Red}}} {
		for _, paid := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%v", tc.name, paid), func(t *testing.T) {
				g, a, b := ronGame()
				randomRONHand(t, b, 2)
				c := repositoryCard(t, tc.set, tc.name)
				c.SetOwner(a.PlayerID())
				a.AddToHand(c)
				for i, color := range tc.colors {
					if paid || i < len(tc.colors)-1 {
						a.ManaPool().Add(color, 1)
					}
				}
				before := a.ManaPool().TotalMana()
				err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0)
				if !paid {
					if err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 || a.ManaPool().TotalMana() != before {
						t.Fatal("failed payment changed state")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().TotalMana() != 0 || len(a.Hand()) != 0 {
					t.Fatal("wrong payment")
				}
				g.ResolveStack()
				p := g.FindPermanent(c.ID())
				if p == nil {
					t.Fatal("missing entry")
				}
				g.DealDamageToPlayer(b, 1, p.ID())
				g.PutTriggersOnStack()
				g.ResolveStack()
				want := 1
				if tc.name == "Nicol Bolas" {
					want = 2
				}
				if len(b.Graveyard()) != want {
					t.Fatal("cast trigger failed")
				}
			})
		}
	}
}

func TestRONDamagePlayerInstances(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEB", "Hypnotic Specter"}, {"LEG", "Nicol Bolas"}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			randomRONHand(t, a, 3)
			randomRONHand(t, b, 3)
			data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
			if err != nil {
				t.Fatal(err)
			}
			nodes, err := Parse(strings.NewReader(string(data)), tc.set+".ron")
			if err != nil {
				t.Fatal(err)
			}
			var factory mage.CardFactory
			for _, node := range nodes {
				if node.Fields["Name"].String == tc.name {
					factory, err = Compile(node)
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			if factory == nil {
				t.Fatal("missing factory")
			}
			first, second := factory(), factory()
			first.SetOwner(a.PlayerID())
			second.SetOwner(b.PlayerID())
			p, q := g.PutOnBattlefield(first, a.PlayerID()), g.PutOnBattlefield(second, b.PlayerID())
			g.DealSimultaneousDamage(mage.PlayerDamageAssignment(b, 1, g.CaptureDamageSource(p.ID())), mage.PlayerDamageAssignment(a, 2, g.CaptureDamageSource(q.ID())))
			randomRONHand(t, b, 1)
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.PutTriggersOnStack()
				if game.StackSize() != 2 {
					t.Fatal("missing independent events")
				}
				game.ResolveStack()
				wantA, wantB := 1, 1
				if tc.name == "Nicol Bolas" {
					wantA, wantB = 3, 4
				}
				if len(game.GetPlayer(a.PlayerID()).Graveyard()) != wantA || len(game.GetPlayer(b.PlayerID()).Graveyard()) != wantB {
					t.Fatal("shared recipient or stale hand selection")
				}
			}
		})
	}
}

func TestRONDamagePlayerDiscardReplacement(t *testing.T) {
	a := &ronDiscardPlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.PutOnBattlefield(mage.NewArtifact("Replacement", "", mage.WithStaticAbility(mage.DiscardToLibraryReplacement())), a.PlayerID())
	randomRONHand(t, a.BasePlayer, 3)
	c := repositoryCard(t, "LEG", "Nicol Bolas")
	c.SetOwner(b.PlayerID())
	source := g.PutOnBattlefield(c, b.PlayerID())
	g.DealDamageToPlayer(a, 1, source.ID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if len(a.Hand()) != 0 || len(a.Library()) != 3 || len(a.Graveyard()) != 0 || g.PlayerDiscardCountThisTurn(a.PlayerID()) != 3 {
		t.Fatal("hand replacement batch failed")
	}
}

func TestRONDamagePlayerHandBatch(t *testing.T) {
	g, a, b := ronGame()
	randomRONHand(t, b, 3)
	c := repositoryCard(t, "LEG", "Nicol Bolas")
	c.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(c, a.PlayerID())
	observed := 0
	g.PutOnBattlefield(mage.NewEnchantment("Observer", "", mage.WithAbility(mage.NewTriggered(core.EvtDiscard, false, mage.DrawCards(mage.Fixed(0))).SetCondition(func(_ *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
		observed++
		if len(b.Hand()) != 0 || len(b.Graveyard()) != 4 {
			t.Fatal("discard event saw a partial hand move")
		}
		return false
	}))), a.PlayerID())
	g.DealDamageToPlayer(b, 1, source.ID())
	randomRONHand(t, b, 1)
	g.PutTriggersOnStack()
	g.ResolveStack()
	if observed != 4 {
		t.Fatal("discard event count", observed)
	}
}
