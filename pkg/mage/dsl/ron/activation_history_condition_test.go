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

func TestRONActivationHistoryLibrary(t *testing.T) {
	for _, count := range []int{6, 7, 8} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "ARN", "Library of Alexandria")
			c.SetOwner(b.PlayerID())
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(c, a.PlayerID())
			for range count {
				a.AddToHand(mage.NewArtifact("Hand", ""))
			}
			for range 7 {
				b.AddToHand(mage.NewArtifact("Other hand", ""))
			}
			for range 3 {
				a.AddToLibrary(mage.NewArtifact("Draw", ""))
			}
			ability := p.Card.Abilities()[1].(mage.ActivatedAbility)
			if ability.CanActivate(a.PlayerID(), g) != (count == 7) {
				t.Fatal("query read wrong hand")
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil)
			if count != 7 {
				if err == nil || g.FindPermanent(p.ID()).Tapped || g.StackSize() != 0 || len(a.Hand()) != count {
					t.Fatal("rejected draw paid costs")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().Count(core.Colorless) != 1 || g.StackSize() != 0 {
					t.Fatal("unrestricted mana ability failed")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(p.ID()).Tapped || len(a.Hand()) != 7 || g.StackSize() != 1 {
				t.Fatal("draw timing or tap cost")
			}
			a.RemoveFromHand(a.Hand()[0].ID())
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			g.ResolveStack()
			if len(a.Hand()) != 8 || len(b.Hand()) != 7 || len(clone.GetPlayer(a.PlayerID()).Hand()) != 6 {
				t.Fatal("restriction repeated or clone changed")
			}
			clone.ResolveStack()
			if len(clone.GetPlayer(a.PlayerID()).Hand()) != 8 {
				t.Fatal("clone lost pending controller")
			}
		})
	}
}

func TestRONActivationHistoryLauncher(t *testing.T) {
	for _, change := range []string{"none", "control", "leave", "return", "phase", "regenerate", "illegal-target"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "ATQ", "Rocket Launcher")
			c.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(c, a.PlayerID())
			targetID := b.PlayerID()
			var target *mage.Permanent
			if change == "illegal-target" {
				target = g.PutOnBattlefield(mage.NewCreature("Target", "", 2, 5), b.PlayerID())
				targetID = target.ID()
			}
			a.ManaPool().Add(core.Colorless, 4)
			for _, turn := range []int{1, 2} {
				if turn == 2 {
					g.SetActivePlayerIndex(1)
					g.SetTurn(2)
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{targetID}); err == nil || a.ManaPool().TotalMana() != 4 || g.StackSize() != 0 {
					t.Fatal("early activation spent resources")
				}
			}
			g.SetActivePlayerIndex(0)
			g.SetTurn(3)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{targetID}); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 2 || b.Life() != 20 || g.StackSize() != 1 {
				t.Fatal("payment or resolution timing")
			}
			if change != "regenerate" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			switch change {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				b.ManaPool().Add(core.Colorless, 2)
				if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, []uuid.UUID{a.PlayerID()}); err == nil || b.ManaPool().TotalMana() != 2 {
					t.Fatal("new controller inherited history")
				}
			case "leave", "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if change == "return" {
					a.RemoveFromHand(c.ID())
					g.PutOnBattlefield(c, a.PlayerID())
				}
			case "phase":
				g.PhaseOut(p.ID())
			case "illegal-target":
				g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
			}
			clone := g.Clone()
			g.ResolveStack()
			want := 18
			if change == "regenerate" {
				want = 19
			}
			if change == "illegal-target" {
				want = 20
			}
			if b.Life() != want || clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("damage, target legality or clone isolation")
			}
			clone.ResolveStack()
			if clone.GetPlayer(b.PlayerID()).Life() != want {
				t.Fatal("clone damage")
			}
			if change == "regenerate" {
				g.AddRegenerationShield(p.ID())
			}
			g.RunStepWithPriority(core.EndStep)
			present := g.FindPermanentIncludingPhased(p.ID()) != nil
			if present != (change == "return" || change == "phase" || change == "illegal-target" || change == "regenerate") {
				t.Fatal("delayed destruction identity or regeneration")
			}
		})
	}
}

func TestRONActivationHistoryInvalidAndInventory(t *testing.T) {
	for _, condition := range []string{
		`PlayerPassesFilter(You,NumCardsInHandIs(EqualTo(Integer(-1))))`,
		`PlayerPassesFilter(Opponent,NumCardsInHandIs(EqualTo(Integer(7))))`,
		`PlayerPassesFilter(You,NumCardsInHandIs(GreaterThanOrEqualTo(Integer(7))))`,
		`PlayerPassesFilter(You,NumCardsInHandIs(EqualTo(ValueX)))`,
		`PlayerPassesFilter(You,NumCardsInHandIs(EqualTo(Integer(7)),Integer(8)))`,
		`PermanentPassesFilter(ThisPermanent,ControlledSinceBeginningOfMostRecentTurn())`,
		`PermanentPassesFilter(Ref_TargetPermanent,ControlledSinceBeginningOfMostRecentTurn)`,
		`PermanentPassesFilter(HostPermanent,ControlledSinceBeginningOfMostRecentTurn)`,
	} {
		assertConditionalRejected(t, `ActivatedWithModifiers(PayMana([]),ActionList([GainLife(Integer(1))]),ActivateOnlyIf(`+condition+`))`)
	}
	src := `Card(Name:"Gate",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[ActivatedWithModifiers(PayMana([]),ActionList([GainLife(Integer(1))]),ActivateOnlyIf(PlayerPassesFilter(You,NumCardsInHandIs(EqualTo(Integer(7))))))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"NumCardsInHandIs": "activation_player_filter", "EqualTo": "activation_hand_comparison", "Integer": "activation_hand_value"} {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing role %s/%s", name, role)
		}
	}
	if report.Compiled != 1 || len(report.Issues) != 0 {
		t.Fatal("valid declaration rejected")
	}
	files := fstest.MapFS{"a.ron": {Data: []byte(src)}, "bad.ron": {Data: []byte(strings.ReplaceAll(src, "NumCardsInHandIs(EqualTo(Integer(7)))", "Unknown"))}}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("strict loading published subset")
	}
}

func TestRONActivationHistoryCastingAndPlaying(t *testing.T) {
	g, a, _ := ronGame()
	c := repositoryCard(t, "ATQ", "Rocket Launcher")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
		t.Fatal("unpaid artifact cast changed state")
	}
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(c.ID()) != nil || a.ManaPool().TotalMana() != 0 {
		t.Fatal("casting cost or entry timing")
	}
	g.ResolveStack()
	if p := g.FindPermanent(c.ID()); p == nil || !p.HasType(core.TypeArtifact) {
		t.Fatal("artifact entry")
	}
	land := repositoryCard(t, "ARN", "Library of Alexandria")
	land.SetOwner(a.PlayerID())
	a.AddToHand(land)
	if err := g.PlayLand(a.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 0 || g.FindPermanent(land.ID()) == nil {
		t.Fatal("land used stack")
	}
}

func TestRONActivationHistoryManaAndInstances(t *testing.T) {
	for _, subject := range []string{"ThisPermanent", "HostPermanent"} {
		t.Run(subject, func(t *testing.T) {
			g, a, b := ronGame()
			rule := `ActivatedWithModifiers(TapPermanent(ThisPermanent),ActionList([AddMana(ManaProduceU)]),And([ActivateOnlyIf(PlayerPassesFilter(You,NumCardsInHandIs(EqualTo(Integer(1))))),ActivateOnlyIf(PermanentPassesFilter(` + subject + `,ControlledSinceBeginningOfMostRecentTurn))]))`
			var factory mage.CardFactory
			if subject == "HostPermanent" {
				src := `Card(Name:"Aura gate",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),` + rule + `])`
				nodes, err := Parse(strings.NewReader(src), "aura-gate.ron")
				if err != nil {
					t.Fatal(err)
				}
				factory, err = Compile(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
			} else {
				factory = permanentScopeFactory(t, "Artifact", rule)
			}
			c, other := factory(), factory()
			c.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			q := g.PutOnBattlefield(other, b.PlayerID())
			if subject == "HostPermanent" {
				host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 2), a.PlayerID())
				g.Attach(p.ID(), host.ID())
				hostB := g.PutOnBattlefield(mage.NewCreature("Other host", "", 2, 2), b.PlayerID())
				g.Attach(q.ID(), hostB.ID())
			}
			a.AddToHand(mage.NewArtifact("Hand", ""))
			cost := core.ParseManaCost("{U}")
			if g.CanAfford(a.PlayerID(), cost, nil) {
				t.Fatal("planner ignored control restriction")
			}
			g.SetTurn(2)
			if !g.CanAfford(a.PlayerID(), cost, nil) || g.CanAfford(b.PlayerID(), cost, nil) || g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
				t.Fatal("planner restriction, instance or query purity")
			}
			clone := g.Clone()
			clone.GetPlayer(a.PlayerID()).RemoveFromHand(a.Hand()[0].ID())
			if clone.CanAfford(a.PlayerID(), cost, nil) || !g.CanAfford(a.PlayerID(), cost, nil) {
				t.Fatal("planner shared clone hand")
			}
			g.PhaseOut(p.ID())
			if g.CanAfford(a.PlayerID(), cost, nil) {
				t.Fatal("phased source supplied mana")
			}
			g.PhaseIn([]uuid.UUID{p.ID()})
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().Count(core.Blue) != 1 || g.StackSize() != 0 || !g.FindPermanent(p.ID()).Tapped || g.FindPermanent(q.ID()).Tapped {
				t.Fatal("mana classification or instance isolation")
			}
		})
	}
}
