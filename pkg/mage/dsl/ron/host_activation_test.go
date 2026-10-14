package ron

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONHostActivationImprison(t *testing.T) {
	for _, scenario := range []string{"paid", "declined", "short", "reattach", "source-left", "source-control", "host-return", "host-phase", "phase-cycle", "copy", "absent", "two", "indestructible"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "declined"}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			hostCard := mage.NewCreature("Host", "", 1, 5, mage.WithAbility(mage.Activate(mage.Costs(mage.Tap()), mage.GainLifeFor(mage.You, mage.Fixed(5)))))
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			host.RevokeBaseAttr(core.AttrSummonSick)
			card := repositoryCard(t, "LEG", "Imprison")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Black, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			aura := g.FindPermanent(card.ID())
			if aura == nil || aura.AttachedTo != host.ID() {
				t.Fatal("Aura did not attach")
			}
			if scenario != "short" {
				a.ManaPool().Add(core.White, 2)
			}
			if err := g.ActivateAbilityByIndex(b.PlayerID(), host.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			original := g.StackPeek()
			if g.StackSize() != 1 || a.offers != 0 {
				t.Fatal("payment ran at activation")
			}
			switch scenario {
			case "reattach":
				g.Attach(aura.ID(), g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 5), b.PlayerID()).ID())
			case "source-left":
				g.RemoveFromBattlefield(aura)
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: aura.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "host-return":
				g.RemoveFromBattlefield(host)
				g.PutOnBattlefield(hostCard, b.PlayerID())
			case "host-phase":
				g.PhaseOut(host.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "absent":
				g.CounterStackObject(original.ID)
			case "two":
				g.UntapPermanent(host)
				if err := g.ActivateAbilityByIndex(b.PlayerID(), host.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			case "indestructible":
				a.accept = false
				aura.GrantBaseAttr(core.Indestructible)
			}
			g.PutTriggersOnStack()
			if scenario == "copy" {
				b.ManaPool().Add(core.White, 1)
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			cloneA := &ronExclusionPaymentPlayer{Player: clone.GetPlayer(a.PlayerID()), accept: a.accept}
			cloneB := &ronExclusionPaymentPlayer{Player: clone.GetPlayer(b.PlayerID()), accept: true}
			clone.SetPlayerAt(0, cloneA)
			clone.SetPlayerAt(1, cloneB)
			g.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 2 && scenario != "short" {
				t.Fatal("live payment changed clone")
			}
			clone.ResolveStack()
			paid := scenario != "declined" && scenario != "short" && scenario != "indestructible"
			life, mana := 20, 2
			if paid {
				mana--
			} else if scenario != "short" {
				life = 25
			}
			if scenario == "short" {
				mana, life = 0, 25
			}
			if scenario == "two" {
				mana = 0
			}
			for _, game := range []*mage.Game{g, clone} {
				if game.GetPlayer(b.PlayerID()).Life() != life || game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != mana {
					t.Fatalf("life=%d mana=%d want=%d,%d", game.GetPlayer(b.PlayerID()).Life(), game.GetPlayer(a.PlayerID()).ManaPool().TotalMana(), life, mana)
				}
				if scenario == "paid" && game.FindPermanent(aura.ID()) == nil {
					t.Fatal("paid trigger destroyed Aura")
				}
				if (scenario == "short" || scenario == "declined") && game.FindPermanent(aura.ID()) != nil {
					t.Fatal("unpaid trigger kept Aura")
				}
				if scenario == "indestructible" && game.FindPermanent(aura.ID()) == nil {
					t.Fatal("destruction ignored indestructible")
				}
			}
			if scenario == "source-control" && (a.offers != 1 || b.offers != 0) {
				t.Fatal("pending trigger changed payer")
			}
			if scenario == "copy" && b.ManaPool().TotalMana() != 0 {
				t.Fatal("copied trigger lost its controller")
			}
		})
	}
}

func TestRONHostActivationInvalid(t *testing.T) {
	event := `WhenAPlayerActivatesAnAbility(AnyPlayer,And([AbilityOfPermanent(HostPermanent),HasTapSelfInCost,NonManaAbility]))`
	for _, bad := range []string{
		strings.Replace(event, "AnyPlayer", "Opponent", 1), strings.Replace(event, "AnyPlayer", "AnyPlayer()", 1),
		strings.Replace(event, "HostPermanent", "ThisPermanent", 1), strings.Replace(event, "HostPermanent", "HostPermanent()", 1),
		strings.Replace(event, "HasTapSelfInCost", "HasTapSelfInCost()", 1), strings.Replace(event, "NonManaAbility", "NonManaAbility()", 1),
		strings.Replace(event, ",NonManaAbility", "", 1), strings.Replace(event, "NonManaAbility", "HasTapSelfInCost", 1),
		strings.Replace(event, "And([", "Or([", 1), strings.Replace(event, "AbilityOfPermanent", "AbilityOfAPermanent", 1),
		strings.Replace(event, "HasTapSelfInCost", "DoesntHaveTapSelfInCost", 1),
	} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),TriggerA(` + bad + `,ActionList([CounterAbility(Trigger_ThatActivatedAbility)]))])`
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	for _, body := range []string{
		`CounterAbility(Trigger_ThatActivatedAbility())`, `CounterAbility(Trigger_ThatSpell)`,
		`TapPermanent(Trigger_ThatActivatedAbility)`, `CounterAbility(Ref_TargetAbility)`,
		`PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))`,
		`CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([CounterAbility(Trigger_ThatActivatedAbility)]))`,
	} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),TriggerA(` + event + `,ActionList([` + body + `]))])`
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	assertConditionalRejected(t, `TriggerA(AtTheBeginningOfAPlayersUpkeep(You),ActionList([CounterAbility(Trigger_ThatActivatedAbility)]))`)
	assertConditionalRejected(t, `TriggerA(`+event+`,ActionList([GainLife(Integer(1))]))`)
}

func TestRONHostActivationBindings(t *testing.T) {
	factory := hostCombatFactory(t, `WhenAPlayerActivatesAnAbility(AnyPlayer,And([NonManaAbility,HasTapSelfInCost,AbilityOfPermanent(HostPermanent)]))`, `PlayerAction(ControllerOfAbility(Trigger_ThatActivatedAbility),GainLife(Integer(2))),CounterAbility(Trigger_ThatActivatedAbility)`)
	g, a, b := ronGame()
	for _, player := range []mage.Player{a, b} {
		host := g.PutOnBattlefield(mage.NewCreature("Host", "", 1, 5, mage.WithAbility(mage.Activate(mage.Costs(mage.Tap()), mage.GainLifeFor(mage.You, mage.Fixed(5))))), player.PlayerID())
		host.RevokeBaseAttr(core.AttrSummonSick)
		controller := a.PlayerID()
		if player.PlayerID() == a.PlayerID() {
			controller = b.PlayerID()
		}
		watcher := g.PutOnBattlefield(factory(), controller)
		g.Attach(watcher.ID(), host.ID())
		if err := g.ActivateAbilityByIndex(player.PlayerID(), host.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatal("independent declarations lost triggers")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatalf("life=%d,%d", a.Life(), b.Life())
	}
}

func TestRONHostActivationStrictLoad(t *testing.T) {
	data, err := fs.ReadFile(repositoryFixtures.files, "LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var source string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Imprison"`) {
			source = line
			break
		}
	}
	if source == "" {
		t.Fatal("missing complete record")
	}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, fstest.MapFS{"valid.ron": {Data: []byte(source)}}, "*.ron"); err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(source, "Trigger_ThatActivatedAbility", "Trigger_ThatSpell", 1)
	if err := LoadInto(registry, fstest.MapFS{"bad.ron": {Data: []byte(bad)}}, "*.ron"); err == nil {
		t.Fatal("invalid collection loaded")
	}
	if !registry.CardRegistered("Imprison") || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("failed load changed registry")
	}
}
