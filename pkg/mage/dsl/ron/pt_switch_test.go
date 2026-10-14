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
)

func checkRONSwitchPT(t *testing.T, g *mage.Game, id uuid.UUID, power, toughness int) {
	t.Helper()
	for range 3 {
		g.ApplyContinuousEffects()
		p := g.FindPermanent(id)
		if p == nil || p.CurrentPower(g) != power || p.CurrentToughness(g) != toughness {
			t.Fatalf("subject %v: got %v, want %d/%d", id, p, power, toughness)
		}
	}
}

func TestRONPTSwitchTransmutation(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "phase-cycle", "shroud", "noncreature", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := mage.NewCreature("Subject", "", 2, 5)
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			spell := repositoryCard(t, "LEG", "Transmutation")
			if err := mage.ValidateCardDefinition(spell.Name(), spell); err != nil {
				t.Fatal(err)
			}
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, 0); err == nil {
				t.Fatal("unpaid cast succeeded")
			}
			if g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("failed payment changed zones")
			}
			a.ManaPool().Add(core.Black, 1)
			a.ManaPool().Add(core.Colorless, 1)
			land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{land.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 2 {
				t.Fatal("illegal target spent mana or was accepted")
			}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			checkRONSwitchPT(t, g, p.ID(), 2, 5)
			switch change {
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				b.RemoveFromHand(card.ID())
				p = g.PutOnBattlefield(card, b.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "shroud":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
			case "noncreature":
				g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrIsCreature)
			case "control":
				if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mage.ChangeController(mage.This, mage.You)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{p.ID()})
			wantP, wantT := 2, 5
			if change == "none" || change == "phase-cycle" || change == "control" {
				wantP, wantT = 5, 2
			}
			checkRONSwitchPT(t, g, p.ID(), wantP, wantT)
			if len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != spell.ID() {
				t.Fatal("resolved spell did not leave the stack normally")
			}
			g.PhaseIn(g.PhaseOut(p.ID()))
			checkRONSwitchPT(t, g, p.ID(), wantP, wantT)
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			checkRONSwitchPT(t, g, p.ID(), 2, 5)
		})
	}
}

func TestRONPTSwitchLayersCopiesAndClones(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 5), b.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 7), b.PlayerID())
	original := castRONSpell(t, g, a, repositoryCard(t, "LEG", "Transmutation"), p.ID())
	clone := g.Clone()
	g.PhaseOut(p.ID())
	copied := g.CopySpellOnStack(original.SourceID, b.PlayerID(), true)
	if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != q.ID() {
		t.Fatal("copy did not choose its own target")
	}
	g.PhaseIn([]uuid.UUID{p.ID()})
	g.ResolveStack()
	checkRONSwitchPT(t, g, p.ID(), 5, 2)
	checkRONSwitchPT(t, g, q.ID(), 7, 3)
	checkRONSwitchPT(t, clone, p.ID(), 2, 5)
	clone.ResolveStack()
	checkRONSwitchPT(t, clone, p.ID(), 5, 2)
	checkRONSwitchPT(t, clone, q.ID(), 3, 7)
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetBasePowerToughness(mage.This, mage.Fixed(4), mage.Fixed(8)), mage.ModifyPT(mage.This, mage.Fixed(2), mage.Fixed(3))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(p.ID()).AddCounter(core.P1P1, 1)
	checkRONSwitchPT(t, g, p.ID(), 12, 7)
	castRONSpell(t, g, a, repositoryCard(t, "LEG", "Transmutation"), p.ID())
	g.ResolveStack()
	checkRONSwitchPT(t, g, p.ID(), 7, 12)
	g.GetEffects().RemoveEndOfTurn()
	checkRONSwitchPT(t, g, p.ID(), 3, 6)
	checkRONSwitchPT(t, clone, p.ID(), 5, 2)
}

func TestRONPTSwitchInvalid(t *testing.T) {
	for _, action := range []string{
		`CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[SwitchPT()],UntilEndOfTurn)`,
		`CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[SwitchPT(Integer(1))],UntilEndOfTurn)`,
		`CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[SwitchPT,Unknown],UntilEndOfTurn)`,
		`CreatePermanentLayerEffectUntil(Ref_TargetPlayer,[SwitchPT],UntilEndOfTurn)`,
		`CreatePermanentLayerEffectUntil(ThisPermanent,[SwitchPT],UntilEndOfTurn)`,
		`CreatePermanentLayerEffectUntil(Trigger_ThatCreature,[SwitchPT],UntilEndOfTurn)`,
		`CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[SwitchPT],UntilEndOfTurn())`,
		`CreatePermanentLayerEffect(Ref_TargetPermanent,[SwitchPT])`,
		`CreateEachPermanentLayerEffectUntil(IsCardtype(Creature),[SwitchPT],UntilEndOfTurn)`,
		`SwitchPT`,
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([%s])))])`, action)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", action)
		}
	}
}

func TestRONPTSwitchStrictLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	node := repositoryRecord(t, "LEG", "Transmutation", "LEG.ron")
	record := strings.Split(string(data), "\n")[node.Position.Line-1]
	registry, err := LoadFS(fstest.MapFS{"switch.ron": {Data: []byte(record)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Transmutation")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Transmutation")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID() == second.ID() || first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("factory shares identity or targets")
	}
	g, a, b := ronGame()
	p := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 5), b.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 7), a.PlayerID())
	castRONSpell(t, g, a, first, p.ID())
	castRONSpell(t, g, b, second, q.ID())
	g.ResolveStack()
	checkRONSwitchPT(t, g, p.ID(), 5, 2)
	checkRONSwitchPT(t, g, q.ID(), 7, 3)
	bad := strings.Replace(record, "[SwitchPT]", "[SwitchPT()]", 1)
	if err := LoadInto(registry, fstest.MapFS{"switch.ron": {Data: []byte(record)}, "bad.ron": {Data: []byte(bad)}}, "*.ron"); err == nil {
		t.Fatal("invalid collection accepted")
	}
	if !registry.CardRegistered("Transmutation") || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("rejected collection changed registry")
	}
}

func TestRONPTSwitchSourceAndHost(t *testing.T) {
	for _, host := range []bool{false, true} {
		t.Run(fmt.Sprint(host), func(t *testing.T) {
			g, a, b := ronGame()
			typeline, pt, subject, enchant := "Creature", ",CardPT:(Power:2,Toughness:5)", "ThisPermanent", ""
			subtypes := ""
			if host {
				typeline, pt, subject, enchant, subtypes = "Enchantment", "", "HostPermanent", "EnchantPermanent(IsCardtype(Creature)),", "Aura"
			}
			src := fmt.Sprintf(`Card(Name:"Switch context",Typeline:(Supertypes:[],Cardtypes:[%s],Subtypes:[%s])%s,Rules:[%sActivated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(%s,[SwitchPT],UntilEndOfTurn)]))])`, typeline, subtypes, pt, enchant, subject)
			nodes, err := Parse(strings.NewReader(src), "context.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			p := source
			if host {
				p = g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 5), b.PlayerID())
				g.Attach(source.ID(), p.ID())
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if host {
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			}
			g.ResolveStack()
			checkRONSwitchPT(t, g, p.ID(), 5, 2)
			clone := g.Clone()
			g.PhaseIn(g.PhaseOut(p.ID()))
			checkRONSwitchPT(t, g, p.ID(), 5, 2)
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			owner := a
			if host {
				owner = b
			}
			physical := p.Card
			owner.RemoveFromHand(physical.ID())
			returned := g.PutOnBattlefield(physical, owner.PlayerID())
			checkRONSwitchPT(t, g, returned.ID(), 2, 5)
			checkRONSwitchPT(t, clone, p.ID(), 5, 2)
		})
	}
}

func TestRONPTSwitchSignedPowerAndStateActions(t *testing.T) {
	for _, power := range []int{-2, 0, 2} {
		t.Run(fmt.Sprint(power), func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefield(mage.NewCreature("Subject", "", power, 5), b.PlayerID())
			castRONSpell(t, g, a, repositoryCard(t, "LEG", "Transmutation"), p.ID())
			g.ResolveStack()
			if power <= 0 {
				if g.FindPermanent(p.ID()) != nil || len(b.Graveyard()) != 1 {
					t.Fatal("nonpositive switched toughness missed state-based action")
				}
			} else {
				checkRONSwitchPT(t, g, p.ID(), 5, power)
			}
		})
	}
}

func TestRONPTSwitchTriggerAndDelay(t *testing.T) {
	for _, change := range []string{"none", "phase", "return"} {
		t.Run(change, func(t *testing.T) {
			g, a, _ := ronGame()
			nodes, err := Parse(strings.NewReader(`Card(Name:"Delayed switch",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:5),Rules:[TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[SwitchPT],UntilEndOfTurn),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[SwitchPT],UntilEndOfTurn)]))]))])`), "delay.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.ResolveStack()
			checkRONSwitchPT(t, g, p.ID(), 5, 2)
			clone := g.Clone()
			switch change {
			case "phase":
				g.PhaseOut(p.ID())
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.ResolveStack()
			if change != "return" {
				g.PhaseIn([]uuid.UUID{p.ID()})
				wantP, wantT := 2, 5
				if change == "phase" {
					wantP, wantT = 5, 2
				}
				checkRONSwitchPT(t, g, p.ID(), wantP, wantT)
			} else {
				a.RemoveFromHand(card.ID())
				returned := g.PutOnBattlefield(card, a.PlayerID())
				checkRONSwitchPT(t, g, returned.ID(), 2, 5)
			}
			checkRONSwitchPT(t, clone, p.ID(), 5, 2)
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			clone.ResolveStack()
			checkRONSwitchPT(t, clone, p.ID(), 2, 5)
		})
	}
}
