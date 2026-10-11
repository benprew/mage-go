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

func TestRONZoneInterveningMiter(t *testing.T) {
	for _, change := range []string{"destroy", "sacrifice", "decline", "no-mana", "source-left", "source-returned", "source-phase", "source-control", "victim-returned", "batch", "token", "other-controller", "phase-before", "lost-before", "copy", "exile-replacement", "regenerate", "bounce", "victim-phase", "self"} {
		t.Run(change, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: change != "decline"}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			card := repositoryCard(t, "ATQ", "Urza's Miter")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			for range 5 {
				a.AddToLibrary(mage.NewLand("Draw fixture"))
			}
			victimCard := mage.NewArtifact("Victim", "")
			if change == "token" {
				victimCard = mage.NewToken("Victim", 0, 0, []core.CardType{core.TypeArtifact}, nil)
			}
			victimCard.SetOwner(b.PlayerID())
			controller := a.PlayerID()
			if change == "other-controller" {
				controller = b.PlayerID()
			}
			victim := g.PutOnBattlefield(victimCard, controller)
			if change != "no-mana" {
				a.ManaPool().Add(core.Colorless, 6)
			}
			if change == "phase-before" {
				g.PhaseOut(source.ID())
			}
			if change == "lost-before" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			}
			want := 1
			if change == "exile-replacement" {
				if err := mage.ApplyEffect(g, mage.ExileIfWouldDieThisTurn(mage.This), victim.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if change == "regenerate" {
				g.AddRegenerationShield(victim.ID())
			}
			switch change {
			case "self":
				g.DestroyPermanent(source)
			case "bounce":
				g.BouncePermanentToHand(victim)
				want = 0
			case "victim-phase":
				g.PhaseOut(victim.ID())
				want = 0
			case "sacrifice":
				g.DoSacrifice(victim)
				want = 0
			case "batch":
				g.DestroyPermanents([]*mage.Permanent{source, victim})
				want = 2
			default:
				g.DestroyPermanent(victim)
			}
			if change == "other-controller" || change == "phase-before" || change == "lost-before" || change == "exile-replacement" || change == "regenerate" {
				want = 0
			}
			switch change {
			case "source-left", "source-returned":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if change == "source-returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "source-phase":
				g.PhaseOut(source.ID())
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "victim-returned":
				b.RemoveFromGraveyard(victimCard.ID())
				g.PutOnBattlefield(victimCard, b.PlayerID())
			}
			g.PutTriggersOnStack()
			if g.StackSize() != want || a.offers != 0 {
				t.Fatalf("detection stack=%d want %d", g.StackSize(), want)
			}
			if change == "copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				want++
			}
			clone := g.Clone()
			clone.ResolveStack()
			g.ResolveStack()
			paid := want
			if change == "decline" || change == "no-mana" {
				paid = 0
			}
			hand := paid
			if change == "source-left" {
				hand++
			}
			if len(a.Hand()) != hand || len(b.Hand()) != map[bool]int{true: 1}[change == "bounce"] || len(clone.GetPlayer(a.PlayerID()).Hand()) != hand-paid {
				t.Fatalf("draw/payment context hand=%d want %d", len(a.Hand()), hand)
			}
			mana := 6 - 3*paid
			if change == "no-mana" {
				mana = 0
			}
			if a.ManaPool().TotalMana() != mana {
				t.Fatal("wrong payment")
			}
		})
	}
}

func TestRONZoneInterveningEntry(t *testing.T) {
	for _, change := range []string{"flying", "none", "lost", "gained", "source-left", "source-returned", "host-returned", "phase", "reattach", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			opts := []mage.CardOption{}
			if change != "none" && change != "gained" {
				opts = append(opts, mage.WithKeyword(core.Flying))
			}
			hostCard := mage.NewCreature("Host", "", 2, 4, opts...)
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			text := `Card(Name:"Entry watcher",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),TriggerI(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),PermanentPassesFilter(HostPermanent,HasAbility(Flying)),ActionList([GainLife(Integer(2))]))])`
			nodes, err := Parse(strings.NewReader(text), "entry.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(source.ID(), host.ID())
			g.FireEvent(core.GameEvent{Type: core.EvtZoneChange, FromZone: core.ZoneHand, ToZone: core.ZoneBattlefield, SourceID: source.ID(), PlayerID: a.PlayerID()})
			switch change {
			case "lost":
				g.AddContinuousEffect(mage.RemoveAllAbilities(host.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			case "gained":
				if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mage.GrantKeywordModification(mage.This, core.Flying)), host.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "source-left", "source-returned":
				g.RemoveFromBattlefield(source)
				if change == "source-returned" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "host-returned":
				g.RemoveFromBattlefield(host)
				q := g.PutOnBattlefield(hostCard, b.PlayerID())
				g.AddContinuousEffect(mage.RemoveAllAbilities(q.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			case "phase":
				g.PhaseOut(host.ID())
			case "reattach":
				q := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 4), b.PlayerID())
				g.Attach(source.ID(), q.ID())
			}
			g.PutTriggersOnStack()
			want := 1
			if change == "none" || change == "gained" {
				want = 0
			}
			if g.StackSize() != want {
				t.Fatalf("stack=%d want %d", g.StackSize(), want)
			}
			if change == "copy" {
				g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
				want++
			}
			if change == "lost" || change == "reattach" {
				want = 0
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			if a.Life() != 20+2*want || clone.GetPlayer(a.PlayerID()).Life() != 20+2*want {
				t.Fatal("condition was not checked with exact host information at both times")
			}
		})
	}
}

func TestRONZoneInterveningInvalid(t *testing.T) {
	valid := `TriggerI(WhenAPermanentIsPutIntoAPlayersGraveyard(IsCardtype(Artifact),AnyPlayer),PermanentPutInGraveyardPassesFilter(WasntSacrificed),ActionList([GainLife(Integer(1))]))`
	for _, rule := range []string{strings.Replace(valid, "WasntSacrificed", "WasntSacrificed()", 1), strings.Replace(valid, "WasntSacrificed", "IsTapped", 1), strings.Replace(valid, "AnyPlayer", "AnyPlayer()", 1), strings.Replace(valid, "PermanentPutInGraveyardPassesFilter(WasntSacrificed)", "PermanentPassesFilter(ThisPermanent,HasAbility(Flying))", 1), `TriggerI(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),PermanentPassesFilter(HostPermanent,HasAbility(Flying)),ActionList([GainLife(Integer(1))]))`} {
		text := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[%s])`, rule)
		nodes, err := Parse(strings.NewReader(text), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", rule)
		}
	}
}

func TestRONZoneInterveningNativeLoadAndCast(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/ATQ.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(line, `Card(Name: "Urza's Miter"`) {
			record = line
			break
		}
	}
	if record == "" {
		t.Fatal("missing complete record")
	}
	files := fstest.MapFS{"ATQ.ron": {Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	card, err := registry.CreateCard("Urza's Miter")
	if err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast succeeded")
	}
	if len(a.Hand()) != 1 || g.StackSize() != 0 || a.offers != 0 {
		t.Fatal("failed cast changed resources")
	}
	a.ManaPool().Add(core.Colorless, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	draw := mage.NewLand("Draw fixture")
	a.AddToLibrary(draw)
	victimCard := mage.NewArtifact("Victim", "")
	victimCard.SetOwner(a.PlayerID())
	victim := g.PutOnBattlefield(victimCard, a.PlayerID())
	a.ManaPool().Add(core.Colorless, 3)
	g.DestroyPermanent(victim)
	g.PutTriggersOnStack()
	g.ResolveStack()
	if len(a.Hand()) != 1 || a.Hand()[0].ID() != draw.ID() || a.ManaPool().TotalMana() != 0 || a.offers != 1 {
		t.Fatal("loaded full RON did not pay and draw")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(record, "WasntSacrificed", "WasntSacrificed()", 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection changed registry")
	}
	report, err := InspectFS(fstest.MapFS{"ATQ.ron": {Data: []byte(record)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "WasntSacrificed" && signature.Role == "death_cause_filter" {
			found = true
		}
	}
	if report.Compiled != 1 || !found {
		t.Fatal("inventory lost death cause role")
	}
}

func TestRONZoneInterveningEarthbindPrerequisite(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), set+".ron")
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, node := range nodes {
			if node.Fields["Name"].String != "Earthbind" {
				continue
			}
			found = true
			if _, err := Compile(node); err == nil || !strings.Contains(err.Error(), "unsupported temporary keyword (PermanentLayerEffect)") {
				t.Fatalf("remaining complete-card blocker: %v", err)
			}
			rules := node.Fields["Rules"]
			trigger := rules.Args[1]
			body, err := Parse(strings.NewReader(`ActionList([GainLife(Integer(2))])`), "body.ron")
			if err != nil {
				t.Fatal(err)
			}
			trigger.Args[2] = body[0]
			rules.Args[1] = trigger
			node.Fields["Rules"] = rules
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition("Earthbind prerequisite", factory()); err != nil {
				t.Fatal(err)
			}
		}
		if !found {
			t.Fatal("missing Earthbind record")
		}
	}
}

func TestRONZoneInterveningDeathBindings(t *testing.T) {
	text := `Card(Name:"Observer",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerI(WhenAPermanentIsPutIntoAPlayersGraveyard(IsCardtype(Artifact),AnyPlayer),PermanentPutInGraveyardPassesFilter(WasntSacrificed),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1))),PlayerAction(ControllerOfDeadPermanent,GainLife(Integer(2)))]))])`
	nodes, err := Parse(strings.NewReader(text), "observer.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	card := mage.NewArtifact("Victim", "")
	card.SetOwner(b.PlayerID())
	victim := g.PutOnBattlefield(card, a.PlayerID())
	g.DestroyPermanent(victim)
	b.RemoveFromGraveyard(card.ID())
	g.PutOnBattlefield(card, b.PlayerID())
	g.RemoveFromBattlefield(source)
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 21 {
		t.Fatal("guard lost captured owner and last controller")
	}
}

func TestRONZoneInterveningNativeEntry(t *testing.T) {
	for _, flying := range []bool{false, true} {
		g, a, b := ronGame()
		opts := []mage.CardOption{}
		if flying {
			opts = append(opts, mage.WithKeyword(core.Flying))
		}
		host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 4, opts...), b.PlayerID())
		text := `Card(Name:"Entry watcher",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),ManaCost:[ManaCostR],Rules:[EnchantPermanent(IsCardtype(Creature)),TriggerI(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),PermanentPassesFilter(HostPermanent,HasAbility(Flying)),ActionList([GainLife(Integer(2))]))])`
		nodes, err := Parse(strings.NewReader(text), "native-entry.ron")
		if err != nil {
			t.Fatal(err)
		}
		factory, err := Compile(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
			t.Fatal("unpaid Aura cast succeeded")
		}
		a.ManaPool().Add(core.Red, 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveTopOfStack()
		g.PutTriggersOnStack()
		want := 0
		if flying {
			want = 1
		}
		if g.StackSize() != want {
			t.Fatal("native entry did not inspect its attached host")
		}
		g.ResolveStack()
		if a.Life() != 20+2*want || g.FindPermanent(card.ID()).AttachedTo != host.ID() {
			t.Fatal("native entry lost attachment or trigger")
		}
	}
}

func TestRONZoneInterveningSourceEntry(t *testing.T) {
	for _, change := range []string{"live", "lost", "leave", "return", "phase"} {
		g, a, b := ronGame()
		text := `Card(Name:"Source watcher",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[Flying,TriggerI(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),PermanentPassesFilter(ThisPermanent,HasAbility(Flying)),ActionList([GainLife(Integer(1))]))],CardPT:(Power:2,Toughness:4))`
		nodes, err := Parse(strings.NewReader(text), "source-entry.ron")
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
		switch change {
		case "lost":
			g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
			g.ApplyContinuousEffects()
		case "leave", "return":
			g.RemoveFromBattlefield(source)
			if change == "return" {
				q := g.PutOnBattlefield(card, b.PlayerID())
				g.AddContinuousEffect(mage.RemoveAllAbilities(q.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			}
		case "phase":
			g.PhaseOut(source.ID())
		}
		g.PutTriggersOnStack()
		g.ResolveStack()
		want := 21
		if change == "lost" {
			want = 20
		}
		if a.Life() != want || b.Life() != 20 {
			t.Fatal("source entry lost current or exact departure information", change)
		}
	}
}

func TestRONZoneInterveningNativeAuraCopy(t *testing.T) {
	g, a, b := ronGame()
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 4, mage.WithKeyword(core.Flying)), b.PlayerID())
	text := `Card(Name:"Entry watcher",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),ManaCost:[ManaCostR],Rules:[EnchantPermanent(IsCardtype(Creature)),TriggerI(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),PermanentPassesFilter(HostPermanent,HasAbility(Flying)),ActionList([GainLife(Integer(2))]))])`
	nodes, err := Parse(strings.NewReader(text), "copy-entry.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("spell copy failed")
	}
	clone := g.Clone()
	g.ResolveStack()
	if a.Life() != 24 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("Aura copies lost entry conditions or clone isolation")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 24 {
		t.Fatal("cloned Aura spells lost their hosts")
	}
	if len(g.FindPermanent(host.ID()).Attachments) != 2 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("Aura copy attachment or payment changed")
	}
}
