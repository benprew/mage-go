package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCreatureDeathSengir(t *testing.T) {
	for _, change := range []string{"normal", "none", "prevented", "other-source", "victim-return", "source-return", "source-phase", "victim-phase-cycle", "cleanup", "control", "return-after", "phase-after", "phase-cycle-after", "copy", "two-victims", "simultaneous-source", "abilities-lost"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEB", "Sengir Vampire")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			if !source.HasAttr(core.Flying) || source.CurrentPower(g) != 4 || source.CurrentToughness(g) != 4 {
				t.Fatal("creature metadata")
			}
			victimCard := mage.NewCreature("Victim", "", 1, 8)
			victimCard.SetOwner(b.PlayerID())
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			damageSource := source.ID()
			if change == "other-source" {
				damageSource = g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 8), a.PlayerID()).ID()
			}
			if change == "prevented" {
				g.AddPreventionShield(victim.ID(), 1)
			}
			if change != "none" {
				g.DealDamageToPermanent(victim, 1, damageSource)
			}
			switch change {
			case "victim-return":
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(victimCard, b.PlayerID())
			case "source-return":
				g.RemoveFromBattlefield(source)
				source = g.PutOnBattlefield(card, a.PlayerID())
			case "source-phase":
				g.PhaseOut(source.ID())
			case "victim-phase-cycle":
				g.PhaseIn(g.PhaseOut(victim.ID()))
			case "cleanup":
				g.RunStepWithPriority(core.Cleanup)
			case "abilities-lost":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
			}
			wantStack, wantCounters := 1, 1
			switch change {
			case "none", "prevented", "other-source", "victim-return", "source-return", "source-phase", "cleanup", "abilities-lost":
				wantStack, wantCounters = 0, 0
			}
			switch change {
			case "two-victims":
				other := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 8), a.PlayerID())
				g.DealDamageToPermanent(other, 1, source.ID())
				g.DestroyPermanents([]*mage.Permanent{victim, other, other})
				wantStack, wantCounters = 2, 2
			case "simultaneous-source":
				g.DestroyPermanents([]*mage.Permanent{source, victim})
				wantCounters = 0
			default:
				g.DestroyPermanent(victim)
			}
			g.PutTriggersOnStack()
			if g.StackSize() != wantStack {
				t.Fatalf("stack=%d want=%d", g.StackSize(), wantStack)
			}
			if change == "simultaneous-source" {
				g.ResolveStack()
				if g.FindPermanent(source.ID()) != nil {
					t.Fatal("dead source returned")
				}
				return
			}
			if source.Counters[core.P1P1] != 0 {
				t.Fatal("counter before resolution")
			}
			switch change {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "return-after":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
				wantCounters = 0
			case "phase-after":
				g.PhaseOut(source.ID())
				wantCounters = 0
			case "phase-cycle-after":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				wantCounters = 2
			}
			clone := g.Clone()
			clone.ResolveStack()
			if g.FindPermanentIncludingPhased(source.ID()).Counters[core.P1P1] != 0 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				p := game.FindPermanentIncludingPhased(source.ID())
				if p.Counters[core.P1P1] != wantCounters {
					t.Fatalf("counters=%d want=%d", p.Counters[core.P1P1], wantCounters)
				}
				if !p.PhasedOut && p.CurrentPower(game) != 4+wantCounters {
					t.Fatal("counter layer")
				}
			}
		})
	}
}

func TestRONCreatureDeathSoulNet(t *testing.T) {
	for _, change := range []string{"pay", "decline", "unaffordable", "sacrifice", "exile", "bounce", "regenerate", "noncreature", "token", "animated", "phase-victim", "phase-source", "control-after", "departed-source", "two-deaths", "simultaneous-source", "copy"} {
		t.Run(change, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: change != "decline"}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, "LEB", "Soul Net")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			victimCard := mage.NewCreature("Victim", "", 1, 8)
			victimCard.SetOwner(b.PlayerID())
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			if change == "noncreature" {
				victimCard = mage.NewArtifact("Noncreature", "")
				victimCard.SetOwner(b.PlayerID())
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(victimCard, b.PlayerID())
			}
			if change == "animated" {
				g.RemoveFromBattlefield(victim)
				victimCard = mage.NewArtifact("Animated", "")
				victimCard.SetOwner(b.PlayerID())
				victim = g.PutOnBattlefield(victimCard, b.PlayerID())
				factory := permanentScopeFactory(t, "Enchantment", `EachPermanentLayerEffect(IsCardtype(Artifact),[AddCardtype(Creature),SetPT(PT(2,2))])`)
				g.PutOnBattlefield(factory(), b.PlayerID())
				g.ApplyContinuousEffects()
			}
			if change == "token" {
				victim.IsToken = true
			}
			if change != "unaffordable" {
				a.ManaPool().Add(core.Colorless, 2)
			}
			b.ManaPool().Add(core.Colorless, 2)
			wantTriggers := 1
			switch change {
			case "sacrifice":
				g.DoSacrifice(victim)
			case "exile":
				g.ExilePermanent(victim)
				wantTriggers = 0
			case "bounce":
				g.BouncePermanentToHand(victim)
				wantTriggers = 0
			case "regenerate":
				g.AddRegenerationShield(victim.ID())
				g.DestroyPermanent(victim)
				wantTriggers = 0
			case "noncreature":
				g.DestroyPermanent(victim)
				wantTriggers = 0
			case "phase-victim":
				g.PhaseOut(victim.ID())
				wantTriggers = 0
			case "phase-source":
				g.PhaseOut(source.ID())
				g.DestroyPermanent(victim)
				wantTriggers = 0
			case "simultaneous-source":
				g.DestroyPermanents([]*mage.Permanent{source, victim})
			case "two-deaths":
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 8), a.PlayerID())
				g.DestroyPermanents([]*mage.Permanent{victim, other, other})
				wantTriggers = 2
			default:
				g.DestroyPermanent(victim)
			}
			g.PutTriggersOnStack()
			if g.StackSize() != wantTriggers || a.offers != 0 || a.Life() != 20 {
				t.Fatal("event or payment timing")
			}
			if change == "copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			if change == "control-after" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if change == "departed-source" {
				g.ExilePermanent(g.FindPermanent(source.ID()))
			}
			clone := g.Clone()
			clone.ResolveStack()
			beforeMana := 2
			if change == "unaffordable" {
				beforeMana = 0
			}
			if a.Life() != 20 || a.ManaPool().TotalMana() != beforeMana {
				t.Fatal("clone changed resources")
			}
			g.ResolveStack()
			gain := wantTriggers
			if change == "decline" || change == "unaffordable" {
				gain = 0
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != beforeMana || clone.GetPlayer(b.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 2 || clone.StackSize() != 0 {
				t.Fatal("search clone did not decline its own optional payments")
			}
			if g.GetPlayer(a.PlayerID()).Life() != 20+gain {
				t.Fatalf("life=%d gain=%d", g.GetPlayer(a.PlayerID()).Life(), gain)
			}
			mana := 2 - gain
			if change == "unaffordable" {
				mana = 0
			}
			if g.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != mana {
				t.Fatal("optional payment amount or rollback")
			}
			otherGain := 0
			if change == "copy" {
				otherGain = 1
			}
			if g.GetPlayer(b.PlayerID()).Life() != 20+otherGain || g.GetPlayer(b.PlayerID()).ManaPool().TotalMana() != 2-otherGain {
				t.Fatal("wrong resolving payer")
			}
		})
	}
}

func TestRONCreatureDeathDeclarations(t *testing.T) {
	for _, filter := range []string{
		`IsCardtype`, `IsCardtype(Creature())`, `IsCardtype(Creature,Artifact)`, `IsCardtype(Artifact)`,
		`And([IsCardtype(Creature)])`, `And([IsCardtype(Creature),WasDealtDamageByPermanentThisTurn(ThisPermanent),IsCardtype(Creature)])`,
		`And([IsCardtype(Creature),IsCardtype(Creature)])`, `Or([IsCardtype(Creature),WasDealtDamageByPermanentThisTurn(ThisPermanent)])`,
		`And([IsCardtype(Creature),WasDealtDamageByPermanentThisTurn(HostPermanent)])`,
		`And([IsCardtype(Creature),WasDealtDamageByPermanentThisTurn(Ref_TargetPermanent)])`,
		`And([IsCardtype(Creature),WasDealtDamageByPermanentThisTurn(ThisPermanent())])`,
		`And([IsCardtype(Creature),WasDealtDamageByPermanentThisTurn(ThisPermanent,ThisPermanent)])`,
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(WhenAPermanentDies(%s),ActionList([GainLife(Integer(1))]))])`, filter)
		nodes, err := Parse(strings.NewReader(src), "invalid-death.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", filter)
		}
	}
	for _, body := range []string{`TapPermanent(Trigger_ThatPermanent)`, `PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))`} {
		src := fmt.Sprintf(`Card(Name:"Unbound",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(WhenAPermanentDies(IsCardtype(Creature)),ActionList([%s]))])`, body)
		nodes, err := Parse(strings.NewReader(src), "unbound.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("event invented a binding")
		}
	}
}

func TestRONCreatureDeathReprintsAndCasting(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Soul Net", "Sengir Vampire"} {
			t.Run(set+"/"+name, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
					t.Fatal("unpaid cast changed state")
				}
				amount := 1
				if name == "Sengir Vampire" {
					amount = 5
				}
				if name == "Sengir Vampire" {
					a.ManaPool().Add(core.Green, amount)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || a.ManaPool().TotalMana() != amount || g.StackSize() != 0 {
						t.Fatal("wrong-color payment changed state")
					}
					a.ManaPool().Clear()
				}
				a.ManaPool().Add(core.Black, amount)
				g.SetStep(core.Upkeep)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || g.StackSize() != 0 || a.ManaPool().TotalMana() != amount {
					t.Fatal("illegal timing changed state")
				}
				g.SetStep(core.PrecombatMain)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if g.FindPermanent(card.ID()) == nil || a.ManaPool().TotalMana() != 0 {
					t.Fatal("cast or cost")
				}
				second := g.PutOnBattlefield(repositoryCard(t, set, name), b.PlayerID())
				victimCard := mage.NewCreature("Victim", "", 1, 8)
				victimCard.SetOwner(a.PlayerID())
				victim := g.PutOnBattlefield(victimCard, a.PlayerID())
				if name == "Sengir Vampire" {
					g.DealDamageToPermanent(victim, 1, card.ID())
				}
				g.DestroyPermanent(victim)
				g.PutTriggersOnStack()
				g.ResolveStack()
				if name == "Sengir Vampire" && (g.FindPermanent(card.ID()).Counters[core.P1P1] != 1 || g.FindPermanent(second.ID()).Counters[core.P1P1] != 0) {
					t.Fatal("independent sources shared history")
				}
			})
		}
	}
}

func TestRONCreatureDeathInventoryAndRegistry(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		var records []string
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "Sengir Vampire"`) || strings.Contains(line, `Name: "Soul Net"`) {
				records = append(records, line)
			}
		}
		if len(records) != 2 {
			t.Fatal("missing complete reprint records")
		}
		files[set+".ron"] = &fstest.MapFile{Data: []byte(strings.Join(records, "\n"))}
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Records != 12 || report.Compiled != 12 || report.UniqueCards != 2 || len(report.Issues) != 0 {
		t.Fatal("reprint inventory")
	}
	found := false
	for _, sig := range report.Signatures {
		if sig.Name == "ThisPermanent" && strings.Contains(sig.Context, "WasDealtDamageByPermanentThisTurn") {
			found = true
			if sig.Role != "permanent_reference" {
				t.Fatalf("damage-history source role=%s", sig.Role)
			}
		}
	}
	if !found {
		t.Fatal("missing nested death source")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("reprints were not canonical")
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(WhenAPermanentDies(IsCardtype(Artifact)),ActionList([]))])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("invalid collection replaced the registry")
	}
}

func TestRONCreatureDeathFilterOrder(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPermanentDies(And([WasDealtDamageByPermanentThisTurn(ThisPermanent),IsCardtype(Creature)])),ActionList([GainLife(Integer(1))]))`)
	g, a, b := ronGame()
	card := factory()
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(card, a.PlayerID())
	victimCard := mage.NewCreature("Victim", "", 1, 8)
	victimCard.SetOwner(b.PlayerID())
	victim := g.PutOnBattlefield(victimCard, b.PlayerID())
	g.DealDamageToPermanent(victim, 1, source.ID())
	g.DestroyPermanent(victim)
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("filter order changed damage history")
	}
}

type creatureDeathLifeChooser struct{ *counterReplacementChooser }

func (*creatureDeathLifeChooser) ChooseMayAbility(string) bool { return true }

func TestRONCreatureDeathLifeReplacementOrder(t *testing.T) {
	for _, index := range []int{0, 1} {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			a := &creatureDeathLifeChooser{&counterReplacementChooser{BasePlayer: mage.NewBasePlayer("A"), index: index}}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			g.SetStep(core.PrecombatMain)
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Soul Net"), a.PlayerID())
			first := g.PutOnBattlefield(mage.NewArtifact("Double", ""), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewArtifact("Add", ""), b.PlayerID())
			g.AddReplacementEffect(&informationLifeReplacement{source: first.ID(), double: true})
			g.AddReplacementEffect(&informationLifeReplacement{source: second.ID()})
			a.ManaPool().Add(core.Colorless, 1)
			victim := mage.NewCreature("Victim", "", 1, 8)
			victim.SetOwner(b.PlayerID())
			g.DestroyPermanent(g.PutOnBattlefield(victim, b.PlayerID()))
			g.PutTriggersOnStack()
			g.ResolveStack()
			if a.Life() != 23+index || a.choices != 1 || a.ManaPool().TotalMana() != 0 || b.Life() != 20 {
				t.Fatal("life replacement ordering or payment")
			}
		})
	}
}
