package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTypedSacrificeCurseArtifact(t *testing.T) {
	for _, scenario := range []string{"pay", "decline", "host-control", "move", "source-left", "source-return", "host-return", "host-phase", "phase-cycle", "source-phase", "removed-before", "removed-after", "copy", "replaced", "prevented"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "decline" && scenario != "prevented"}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: a.accept}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			hostCard := mage.NewArtifact("Host", "")
			hostCard.SetOwner(a.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			other := g.PutOnBattlefield(mage.NewArtifact("Other", ""), a.PlayerID())
			c := repositoryCard(t, "DRK", "Curse Artifact")
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			a.ManaPool().Add(core.Black, 2)
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			invalid := g.PutOnBattlefield(mage.NewCreature("Invalid", "", 1, 1), b.PlayerID())
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{invalid.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 4 || len(a.Hand()) != 1 {
				t.Fatal("target legality or rollback")
			}
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(c.ID())
			if source == nil || source.AttachedTo != host.ID() {
				t.Fatal("attachment")
			}
			if scenario == "removed-before" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
			}
			queueRONUpkeep(g, a)
			if g.StackSize() != 0 {
				t.Fatal("wrong upkeep")
			}
			queueRONUpkeep(g, b)
			if scenario == "removed-before" {
				if g.StackSize() != 0 {
					t.Fatal("removed ability triggered")
				}
				return
			}
			if g.StackSize() != 1 || a.offers+b.offers != 0 || g.FindPermanent(host.ID()) == nil {
				t.Fatal("payment timing")
			}
			switch scenario {
			case "host-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "move":
				g.Attach(source.ID(), other.ID())
			case "source-left", "source-return":
				g.RemoveFromBattlefield(source)
				if scenario == "source-return" {
					g.PutOnBattlefield(c, a.PlayerID())
					g.Attach(c.ID(), other.ID())
				}
			case "host-return":
				g.RemoveFromBattlefield(host)
				g.PutOnBattlefield(hostCard, b.PlayerID())
			case "host-phase":
				g.PhaseOut(host.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "source-phase":
				g.PhaseOut(source.ID())
			case "removed-after":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			case "replaced":
				g.AddExileIfWouldGoToGraveyardThisTurn(host.ID(), uuid.New())
			case "prevented":
				g.AddPreventionShield(b.PlayerID(), 2)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				paid := game != clone && scenario != "decline" && scenario != "prevented" && scenario != "host-return" && scenario != "host-phase" && scenario != "host-control"
				subject := host.ID()

				if paid && game.FindPermanentIncludingPhased(subject) != nil {
					t.Fatalf("declared subject not sacrificed clone=%t life=%d/%d offers=%d/%d", game == clone, game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life(), a.offers, b.offers)
				}
				if !paid && game.FindPermanentIncludingPhased(host.ID()) == nil {
					t.Fatal("failed payment consumed subject")
				}
				damage := 0
				if scenario == "decline" || scenario == "host-return" || scenario == "host-phase" || scenario == "host-control" {
					damage = 2
				}
				if scenario == "copy" {
					damage = 2
				}
				recipient := b.PlayerID()
				if game == clone {
					damage = 2
					if scenario == "copy" {
						damage = 4
					}
					if scenario == "prevented" {
						damage = 0
					}

				}
				for _, id := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
					want := 20
					if id == recipient {
						want -= damage
					}
					if game.GetPlayer(id).Life() != want {
						t.Fatalf("life=%d want=%d clone=%t", game.GetPlayer(id).Life(), want, game == clone)
					}
				}
				if scenario == "replaced" && game != clone && len(game.GetExile()) != 1 {
					t.Fatal("exile replacement")
				}
				if scenario == "source-return" && game.FindPermanent(other.ID()) == nil {
					t.Fatal("old trigger followed returned aura")
				}
			}
		})
	}
}

func TestRONTypedSacrificeDeclarationsAndLocalResults(t *testing.T) {
	for _, cost := range []string{"SacrificePermanent()", "SacrificePermanent(ThisPermanent())", "SacrificePermanent(HostPermanent)", "SacrificePermanent(Ref_TargetPermanent)", "SacrificePermanent(Trigger_ThatCreature)", "SacrificePermanent(You)", "SacrificePermanent(SinglePermanent(ThisPermanent))"} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([MayCost(` + cost + `),If(CostWasPaid,[GainLife(Integer(1))])]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", cost)
		}
	}
	for _, condition := range []string{"If(CostWasPaid", "Unless(CostWasntPaid", "If(CostWasntPaid", "Unless(CostWasPaid"} {
		for _, accept := range []bool{false, true} {
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([MayCost(SacrificePermanent(ThisPermanent)),`+condition+`,[GainLife(Integer(3))])]))`)
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: accept}
			g := mage.NewGame(a, mage.NewBasePlayer("B"))
			g.SetStep(core.PrecombatMain)
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			want := 20
			positive := condition == "If(CostWasPaid" || condition == "Unless(CostWasntPaid"
			if positive == accept {
				want = 23
			}
			if a.Life() != want || (g.FindPermanent(p.ID()) == nil) != accept {
				t.Fatal("branch polarity")
			}
		}
	}
}

func TestRONTypedSacrificeTargetsAndInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Artifact))],ActionList([MayCost(SacrificePermanent(Ref_TargetPermanent)),If(CostWasPaid,[MayCost(SacrificePermanent(Ref_TargetPermanent)),Unless(CostWasPaid,[GainLife(Integer(2))]),GainLife(Integer(3))])])) )`)
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	first := g.PutOnBattlefield(factory(), a.PlayerID())
	second := g.PutOnBattlefield(factory(), a.PlayerID())
	one := g.PutOnBattlefield(mage.NewArtifact("One", ""), a.PlayerID())
	two := g.PutOnBattlefield(mage.NewArtifact("Two", ""), a.PlayerID())
	for _, source := range []*mage.Permanent{first, second} {
		if err := mage.ValidateCardDefinition(source.Name(), source.Card); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, []uuid.UUID{one.ID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), second.ID(), 0, []uuid.UUID{two.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(one.ID()) == nil || g.FindPermanent(two.ID()) != nil || a.Life() != 25 {
		t.Fatal("instance binding or nested result")
	}
	g.ResolveStack()
	if g.FindPermanent(one.ID()) != nil || a.Life() != 30 || a.offers != 2 {
		t.Fatal("independent target or payment result")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, []uuid.UUID{second.ID()}); err != nil {
		t.Fatal(err)
	}
	copyObject := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
	if copyObject == nil {
		t.Fatal("copy")
	}
	copyObject.Targets[0] = first.ID()
	copyObject.TargetZones[first.ID()] = core.ZoneBattlefield
	g.ResolveStack()
	if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil || a.Life() != 40 {
		t.Fatal("retargeted copy changed original binding")
	}
}

func TestRONTypedSacrificeNativeUpkeep(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
	g := mage.NewGame(a, b)
	c := repositoryCard(t, "DRK", "Curse Artifact")
	c.SetOwner(a.PlayerID())
	host := g.PutOnBattlefield(mage.NewArtifact("Host", ""), b.PlayerID())
	source := g.PutOnBattlefield(c, a.PlayerID())
	g.Attach(source.ID(), host.ID())
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Upkeep)
	if g.FindPermanent(host.ID()) != nil || a.Life() != 20 || b.Life() != 20 || b.offers != 1 || g.StackSize() != 0 {
		t.Fatal("native upkeep payment")
	}
}

func TestRONTypedSacrificeUnblockedSourceClause(t *testing.T) {
	n := hostUntapRecord(t, "FEM", "Mindstab Thrull")
	if _, err := Compile(n); err == nil || !strings.Contains(err.Error(), "Trigger_DefendingPlayer") {
		t.Fatalf("remaining player binding: %v", err)
	}
	nodes, err := Parse(strings.NewReader(`Card(Name:"Source event contract",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:2),Rules:[TriggerA(WhenACreatureAttacksAndIsntBlocked(SinglePermanent(ThisPermanent)),ActionList([MayCost(SacrificePermanent(Trigger_ThatCreature)),If(CostWasPaid,[GainLife(Integer(3))])]))])`), "source-event.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	g := mage.NewGame(a, mage.NewBasePlayer("B"))
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtAttacksUnblocked, SourceID: p.ID(), PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.FindPermanent(p.ID()) == nil {
		t.Fatal("event timing")
	}
	g.ResolveStack()
	if a.Life() != 23 || g.FindPermanent(p.ID()) != nil {
		t.Fatal("source event payment")
	}
}
