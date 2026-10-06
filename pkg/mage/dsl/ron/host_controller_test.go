package ron

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONHostControllerParalyze(t *testing.T) {
	for _, choice := range []string{"pay", "decline", "unaffordable", "mana-source"} {
		t.Run(choice, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: choice != "decline"}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 8), b.PlayerID())
			card := repositoryCard(t, "LEB", "Paralyze")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			a.ManaPool().Add(core.Black, 1)
			land := g.PutOnBattlefield(mage.NewLand("Invalid host"), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{land.ID()}, 0); err == nil {
				t.Fatal("noncreature target accepted")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 || a.ManaPool().TotalMana() != 1 {
				t.Fatal("failed casting changed resources")
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if host.Tapped || a.offers+b.offers != 0 {
				t.Fatal("entry action or payment ran before resolution")
			}
			g.ResolveStack()
			if !g.FindPermanent(host.ID()).Tapped || g.FindPermanent(card.ID()).AttachedTo != host.ID() {
				t.Fatal("entry did not attach and tap")
			}
			g.SetActivePlayerIndex(1)
			g.RunStepWithPriority(core.Untap)
			if !g.FindPermanent(host.ID()).Tapped {
				t.Fatal("normal untap bypassed restriction")
			}
			a.ManaPool().Add(core.Colorless, 4)
			var manaSources []uuid.UUID
			if choice == "mana-source" {
				for range 4 {
					forest := repositoryCard(t, "LEB", "Forest")
					forest.SetOwner(b.PlayerID())
					manaSources = append(manaSources, g.PutOnBattlefield(forest, b.PlayerID()).ID())
				}
			} else if choice != "unaffordable" {
				b.ManaPool().Add(core.Colorless, 4)
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("Aura controller upkeep triggered")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 || a.offers+b.offers != 0 {
				t.Fatal("payment timing or trigger targets")
			}
			g.ResolveStack()
			paid := choice == "pay" || choice == "mana-source"
			if g.FindPermanent(host.ID()).Tapped == paid || a.offers != 0 || a.ManaPool().TotalMana() != 4 {
				t.Fatal("wrong payer or untap result")
			}
			wantOffers := 1
			if choice == "unaffordable" {
				wantOffers = 0
			}
			if b.offers != wantOffers || g.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != map[string]int{"decline": 4}[choice] {
				t.Fatal("payment did not preserve or spend the expected mana")
			}
			for _, id := range manaSources {
				if !g.FindPermanent(id).Tapped {
					t.Fatal("resolution payment did not use mana abilities")
				}
			}
			if !g.UntapPermanent(g.FindPermanent(host.ID())) && !paid {
				t.Fatal("other untap effects prohibited")
			}
		})
	}
}

func TestRONHostControllerResolutionIdentity(t *testing.T) {
	for _, scenario := range []string{"normal", "control-before", "control-after", "move", "source-control", "source-left", "source-return", "host-return", "phase-before", "phase-after", "phase-cycle", "removed-before", "removed-after", "unattached", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			hostCard := mage.NewCreature("Host", "", 2, 8)
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 8), a.PlayerID())
			card := repositoryCard(t, "LEB", "Paralyze")
			source := g.PutOnBattlefield(card, a.PlayerID())
			if scenario != "unattached" {
				g.Attach(source.ID(), host.ID())
			}
			g.ResolveStack()
			g.TapPermanent(g.FindPermanent(host.ID()))
			g.TapPermanent(g.FindPermanent(other.ID()))
			control := func(id, player uuid.UUID) {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: id, ControllerID: player, Duration: core.Indefinite})
			}
			if scenario == "control-before" {
				control(host.ID(), a.PlayerID())
			}
			if scenario == "phase-before" {
				g.PhaseOut(host.ID())
			}
			if scenario == "removed-before" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
			}
			eventPlayer := b.PlayerID()
			if scenario == "control-before" {
				eventPlayer = a.PlayerID()
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: eventPlayer})
			g.PutTriggersOnStack()
			switch scenario {
			case "control-after":
				control(host.ID(), a.PlayerID())
			case "move":
				g.Attach(source.ID(), other.ID())
			case "source-control":
				control(source.ID(), b.PlayerID())
			case "source-left", "source-return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if scenario == "source-return" {
					g.PutOnBattlefield(card, a.PlayerID())
					g.Attach(card.ID(), other.ID())
				}
			case "host-return":
				g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
				g.PutOnBattlefield(hostCard, a.PlayerID())
				g.TapPermanent(g.FindPermanent(host.ID()))
			case "phase-after":
				g.PhaseOut(host.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "removed-after":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
			}
			g.PutTriggersOnStack()
			if scenario == "source-return" {
				if g.StackSize() != 2 {
					t.Fatal("returned Aura did not queue a separate entry trigger")
				}
				g.ResolveTopOfStack()
			}
			want := 1
			if scenario == "phase-before" || scenario == "removed-before" || scenario == "unattached" {
				want = 0
			}
			if g.StackSize() != want {
				t.Fatalf("triggers=%d want=%d", g.StackSize(), want)
			}
			if scenario == "source-return" {
				g.TapPermanent(g.FindPermanent(other.ID()))
			}
			if scenario == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			a.ManaPool().Add(core.Colorless, 8)
			b.ManaPool().Add(core.Colorless, 8)
			clone := g.Clone()
			clone.ResolveStack()
			if !g.FindPermanentIncludingPhased(host.ID()).Tapped || a.ManaPool().TotalMana() != 8 || b.ManaPool().TotalMana() != 8 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			payer := b.PlayerID()
			if scenario == "control-before" || scenario == "control-after" || scenario == "move" {
				payer = a.PlayerID()
			}
			amount := 4 * want
			if scenario == "copy" {
				amount = 8
			}
			for _, game := range []*mage.Game{g, clone} {
				for _, player := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
					spent := 0
					if player == payer && game == g {
						spent = amount
					}
					if game.GetPlayer(player).ManaPool().TotalMana() != 8-spent {
						t.Fatalf("payer changed: %s/%s", scenario, player)
					}
				}
				target := host.ID()
				if scenario == "move" {
					target = other.ID()
				}
				untapped := game == g && want > 0 && scenario != "host-return" && scenario != "phase-after"
				if game.FindPermanentIncludingPhased(target).Tapped == untapped {
					t.Fatal("action did not keep attachment identity")
				}
			}
		})
	}
}

func TestRONHostControllerDeclarations(t *testing.T) {
	for _, tc := range []struct{ kind, ref string }{
		{"Aura", "HostController()"}, {"Aura", "HostPermanent"}, {"Aura", "Trigger_ThatPlayer"}, {"Artifact", "HostController"},
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[%s],Subtypes:[%s]),Rules:[Activated(PayMana([]),ActionList([PlayerAction(%s,GainLife(Integer(1)))]))])`, map[string]string{"Aura": "Enchantment", "Artifact": "Artifact"}[tc.kind], map[string]string{"Aura": "Aura"}[tc.kind], tc.ref)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("invalid reference accepted: %s", src)
		}
	}
}

func TestRONHostControllerNativeUpkeep(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
	g := mage.NewGame(a, b)
	g.SetTurn(1)
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 8), b.PlayerID())
	card := repositoryCard(t, "LEB", "Paralyze")
	g.PutOnBattlefield(card, a.PlayerID())
	g.Attach(card.ID(), host.ID())
	g.ResolveStack()
	g.TapPermanent(g.FindPermanent(host.ID()))
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(host.ID()).Tapped {
		t.Fatal("native untap ignored Paralyze")
	}
	b.ManaPool().Add(core.Colorless, 4)
	g.RunStepWithPriority(core.Upkeep)
	if g.FindPermanent(host.ID()).Tapped || g.StackSize() != 0 || b.offers != 1 || a.offers != 0 || b.ManaPool().TotalMana() != 0 {
		t.Fatal("native upkeep payment or untap failed")
	}
}

func TestRONHostControllerActivation(t *testing.T) {
	src := `Card(Name:"Attachment query",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(AnyPermanent),Activated(PayMana([]),ActionList([PlayerAction(HostController,GainLife(Integer(3)))]))])`
	nodes, err := Parse(strings.NewReader(src), "attachment.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Attachment query", factory()); err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	host := g.PutOnBattlefield(mage.NewArtifact("Host", ""), b.PlayerID())
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	g.Attach(source.ID(), host.ID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 23 {
		t.Fatal("activation read the source controller")
	}
}

func TestRONHostControllerInstancesAndReprints(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	var factory mage.CardFactory
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range nodes {
			if n.Fields["Name"].String != "Paralyze" {
				continue
			}
			factory, err = Compile(n)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition("Paralyze", factory()); err != nil {
				t.Fatal(err)
			}
			count++
		}
	}
	if count != 6 {
		t.Fatalf("reprints=%d", count)
	}
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
	g := mage.NewGame(a, b)
	hosts := make([]uuid.UUID, 0, 2)
	for _, player := range []*ronPaymentPlayer{a, b} {
		host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 8), player.PlayerID())
		card := factory()
		card.SetOwner(a.PlayerID())
		g.PutOnBattlefield(card, a.PlayerID())
		g.Attach(card.ID(), host.ID())
		hosts = append(hosts, host.ID())
		player.ManaPool().Add(core.Colorless, 4)
	}
	g.ResolveStack()
	for _, id := range hosts {
		g.TapPermanent(g.FindPermanent(id))
	}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("independent pending events lost")
	}
	g.ResolveStack()
	if a.offers != 1 || b.offers != 1 || a.ManaPool().TotalMana()+b.ManaPool().TotalMana() != 0 {
		t.Fatal("instance payment references collided")
	}
	for _, id := range hosts {
		if g.FindPermanent(id).Tapped {
			t.Fatal("instance untap references collided")
		}
	}
}

func TestRONHostControllerCurseDamageClause(t *testing.T) {
	n := hostUntapRecord(t, "DRK", "Curse Artifact")
	if _, err := Compile(n); err == nil || !strings.Contains(err.Error(), "optional payment cost (SacrificePermanent)") {
		t.Fatalf("full card blocker: %v", err)
	}
	erosion := hostUntapRecord(t, "DRK", "Erosion")
	if _, err := Compile(erosion); err == nil || !strings.Contains(err.Error(), "optional payment cost (Or)") {
		t.Fatalf("alternative payment blocker: %v", err)
	}
	fields := maps.Clone(n.Fields)
	rules := fields["Rules"]
	rules.Args = append([]Node(nil), rules.Args...)
	trigger := rules.Args[1]
	trigger.Args = append([]Node(nil), trigger.Args...)
	body := trigger.Args[1]
	body.Args = append([]Node(nil), body.Args...)
	body.Args[0] = body.Args[0].Args[1].Args[1]
	trigger.Args[1] = body
	rules.Args[1] = trigger
	fields["Rules"] = rules
	n.Fields = fields
	factory, err := Compile(n)
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Curse Artifact damage clause", factory()); err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"normal", "control", "move", "departed", "returned-host", "phase", "copy", "prevented"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			hostCard := mage.NewArtifact("Host", "")
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			other := g.PutOnBattlefield(mage.NewArtifact("Other", ""), a.PlayerID())
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(source.ID(), host.ID())
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || len(g.StackPeek().Targets) != 0 {
				t.Fatal("damage clause trigger missing or targeted")
			}
			switch scenario {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "move":
				g.Attach(source.ID(), other.ID())
			case "departed":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "returned-host":
				g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
				g.PutOnBattlefield(hostCard, a.PlayerID())
			case "phase":
				g.PhaseOut(host.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("damage clause copy failed")
				}
			case "prevented":
				g.AddPreventionShield(b.PlayerID(), 2)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original damage recipient")
			}
			g.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				recipient := b.PlayerID()
				if scenario == "control" || scenario == "move" {
					recipient = a.PlayerID()
				}
				amount := 2
				switch scenario {
				case "copy":
					amount = 4
				case "prevented":
					amount = 0
				}
				for _, player := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
					damage := 0
					if player == recipient {
						damage = amount
					}
					if game.GetPlayer(player).Life() != 20-damage {
						t.Fatal("resolution recipient did not use current attachment or exact LKI")
					}
				}
			}
		})
	}
}
