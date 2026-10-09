package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONHostDamagePlayerBackfire(t *testing.T) {
	for _, state := range []string{"normal", "partial", "prevented", "zero", "other-player", "other-source", "creature", "life-loss", "reattach", "host-return", "host-control", "host-phase", "aura-leave", "aura-phase-before", "removed", "twice", "copy"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Backfire")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			hostCard := mage.NewCreature("Host", "", 3, 9)
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 9), a.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed cast changed state")
			}
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("early Aura entry")
			}
			g.ResolveStack()
			aura := g.FindPermanent(card.ID())
			if aura == nil || aura.AttachedTo != host.ID() {
				t.Fatal("missing attachment")
			}
			if state == "partial" {
				g.AddPreventionShield(a.PlayerID(), 2)
			}
			if state == "prevented" {
				g.AddPreventionShield(a.PlayerID(), 3)
			}
			if state == "aura-phase-before" {
				g.PhaseOut(aura.ID())
			}
			if state == "removed" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(aura.ID(), core.Indefinite))
			}
			wantA, wantB, triggers := 17, 17, 1
			switch state {
			case "zero":
				g.DealDamageToPlayer(a, 0, host.ID())
				wantA, wantB, triggers = 20, 20, 0
			case "other-player":
				g.DealDamageToPlayer(b, 3, host.ID())
				wantA, wantB, triggers = 20, 17, 0
			case "other-source":
				g.DealDamageToPlayer(a, 3, other.ID())
				wantB, triggers = 20, 0
			case "creature":
				g.DealDamageToPermanent(other, 3, host.ID())
				wantA, wantB, triggers = 20, 20, 0
			case "life-loss":
				a.LoseLife(3)
				wantB, triggers = 20, 0
			default:
				g.DealDamageToPlayer(a, 3, host.ID())
			}
			switch state {
			case "partial":
				wantA, wantB = 19, 19
			case "prevented":
				wantA, wantB, triggers = 20, 20, 0
			case "aura-phase-before", "removed":
				wantB, triggers = 20, 0
			case "reattach":
				g.Attach(aura.ID(), other.ID())
			case "host-return":
				g.RemoveFromBattlefield(host)
				g.PutOnBattlefield(hostCard, a.PlayerID())
			case "host-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				wantA, wantB = 14, 20
			case "host-phase":
				g.PhaseOut(host.ID())
			case "aura-leave":
				g.RemoveFromBattlefield(aura)
			case "twice":
				g.DealDamageToPlayer(a, 1, host.ID())
				wantA, wantB, triggers = 16, 16, 2
			}
			if b.Life() != 20 && state != "other-player" {
				t.Fatal("early reflected damage")
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.PutTriggersOnStack()
				if game.StackSize() != triggers {
					t.Fatalf("triggers %d, want %d", game.StackSize(), triggers)
				}
				if state == "copy" {
					game.CopyStackObjectDirect(game.StackPeek(), b.PlayerID(), false)
				}
				game.ResolveStack()
				expectedB := wantB
				if state == "copy" {
					expectedB -= 3
				}
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != expectedB {
					t.Fatalf("life %d/%d, want %d/%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life(), wantA, expectedB)
				}
			}
		})
	}
}

func TestRONHostDamagePlayerInvalid(t *testing.T) {
	for _, event := range []string{
		`WhenAPermanentDealsDamageToAPlayer(SinglePermanent(HostPermanent),SinglePlayer(You()))`,
		`WhenAPermanentDealsDamageToAPlayer(SinglePermanent(HostPermanent),SinglePlayer(Opponent))`,
		`WhenAPermanentDealsDamageToAPlayer(SinglePermanent(HostPermanent),SinglePlayer(You),AnyPlayer)`,
		`WhenAPermanentDealsDamageToAPlayer(SinglePermanent(Trigger_ThatPermanent),AnyPlayer)`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),TriggerA(` + event + `,ActionList([GainLife(Trigger_AmountOfDamageDealt)]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", event)
		}
	}
}

func TestRONHostDamagePlayerReturnedDamageSource(t *testing.T) {
	g, a, b := ronGame()
	card := mage.NewCreature("Host", "", 2, 9)
	old := g.PutOnBattlefield(card, b.PlayerID())
	saved := g.CaptureDamageSource(old.ID())
	g.RemoveFromBattlefield(old)
	returned := g.PutOnBattlefield(card, b.PlayerID())
	aura := g.PutOnBattlefield(repositoryCard(t, "LEG", "Backfire"), a.PlayerID())
	g.Attach(aura.ID(), returned.ID())
	g.DealDamageToPlayerFromSource(a, 2, saved)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("new host inherited old damage source")
	}
	g.DealDamageToPlayer(a, 1, returned.ID())
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("new host did not trigger")
	}
	g.ResolveStack()
	if b.Life() != 19 {
		t.Fatal("wrong reflected damage", b.Life())
	}
}

func TestRONHostDamagePlayerSourceContexts(t *testing.T) {
	for _, filter := range []string{"AnyPlayer", "Opponent", "SinglePlayer(You)"} {
		t.Run(filter, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPermanentDealsDamageToAPlayer(SinglePermanent(ThisPermanent),`+filter+`),ActionList([GainLife(Trigger_AmountOfDamageDealt)]))`)
			g, a, b := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			recipient := b
			if filter == "SinglePlayer(You)" {
				recipient = a
			}
			g.DealDamageToPlayer(recipient, 2, source.ID())
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("missing source trigger")
			}
			g.ResolveStack()
			wantA, wantB := 22, 18
			if recipient == a {
				wantA, wantB = 20, 20
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatal("wrong amount binding")
			}
		})
	}
}

func TestRONHostDamagePlayerSimultaneous(t *testing.T) {
	g, a, b := ronGame()
	assignments := make([]mage.DamageAssignment, 0, 2)
	for _, amount := range []int{1, 3} {
		host := g.PutOnBattlefield(mage.NewCreature("Host", "", 1, 9), b.PlayerID())
		aura := g.PutOnBattlefield(repositoryCard(t, "LEG", "Backfire"), a.PlayerID())
		g.Attach(aura.ID(), host.ID())
		assignments = append(assignments, mage.PlayerDamageAssignment(a, amount, g.CaptureDamageSource(host.ID())))
	}
	g.DealSimultaneousDamage(assignments...)
	g.AddPreventionShield(b.PlayerID(), 2)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("missing independent batch triggers")
	}
	g.ResolveStack()
	if a.Life() != 16 || b.Life() != 18 {
		t.Fatal("batch amount or reflection prevention", a.Life(), b.Life())
	}
}

func TestRONHostDamagePlayerInventoryAndAtomicLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Backfire"`) {
			record = line
		}
	}
	if record == "" {
		t.Fatal("missing complete Backfire record")
	}
	files := fstest.MapFS{"one.ron": {Data: []byte(record)}, "two.ron": {Data: []byte(record)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 2 || report.UniqueCards != 1 {
		t.Fatalf("inspection %v: %+v", err, report)
	}
	for name, role := range map[string]string{"HostPermanent": "permanent_reference", "Trigger_ThatPermanent": "permanent_reference", "Trigger_AmountOfDamageDealt": "value", "SinglePlayer": "player_filter"} {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatal("missing semantic role", name, role)
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([GainLife(Trigger_AmountOfDamageDealt)]))])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("invalid collection accepted")
	}
	if _, err := registry.CreateCard("Backfire"); err != nil {
		t.Fatal("failed load replaced valid registry", err)
	}
}

func TestRONHostDamagePlayerTokenAmountIsolation(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([CreateTokens([TokenWithPT(PT(1,1),Colorless,[],[Creature],SubtypesList([Snake]),[TriggerA(WhenAPermanentDealsDamageToAPlayer(SinglePermanent(ThisPermanent),AnyPlayer),ActionList([GainLife(Trigger_AmountOfDamageDealt)]))])],[])]))`)
	g, a, b := ronGame()
	for _, player := range []*mage.BasePlayer{a, b} {
		card := factory()
		card.SetOwner(player.PlayerID())
		player.AddToHand(card)
		if err := g.CastSpellByID(player.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	tokens := g.FilterBattlefield(mage.IsToken)
	if len(tokens) != 2 {
		t.Fatal("missing tokens")
	}
	for _, token := range tokens {
		recipient, amount := b, 1
		if token.ControllerID() == b.PlayerID() {
			recipient, amount = a, 3
		}
		g.DealDamageToPlayer(recipient, amount, token.ID())
	}
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 22 {
		t.Fatal("token amount bindings shared", a.Life(), b.Life())
	}
}
