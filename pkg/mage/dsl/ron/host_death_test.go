package ron

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONHostDeathCreatureBond(t *testing.T) {
	for _, change := range []string{"normal", "batch-aura-first", "batch-host-first", "control-before", "move", "return-host", "return-aura", "phase-after", "phase-before", "removed-before", "removed-after", "copy", "prevention"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			host := mage.NewCreature("Host", "", 2, 3)
			host.SetOwner(a.PlayerID())
			hp := g.PutOnBattlefield(host, b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 9), a.PlayerID())
			card := repositoryCard(t, "LEA", "Creature Bond")
			card.SetOwner(a.PlayerID())
			ap := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(ap.ID(), hp.ID())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.AttachedPermanent, mage.Fixed(0), mage.Fixed(2))), ap.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if change == "control-before" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: hp.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			if change == "phase-before" {
				g.PhaseOut(ap.ID())
			}
			if change == "removed-before" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(ap.ID(), core.Indefinite))
			}
			switch change {
			case "batch-aura-first":
				g.DestroyPermanents([]*mage.Permanent{g.FindPermanent(ap.ID()), g.FindPermanent(hp.ID())})
			case "batch-host-first":
				g.DestroyPermanents([]*mage.Permanent{g.FindPermanent(hp.ID()), g.FindPermanent(ap.ID())})
			default:
				g.DestroyPermanent(g.FindPermanent(hp.ID()))
			}
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("damage before resolution")
			}
			switch change {
			case "move":
				g.Attach(ap.ID(), other.ID())
			case "return-host":
				a.RemoveFromGraveyard(host.ID())
				g.PutOnBattlefield(host, a.PlayerID())
			case "return-aura":
				g.DestroyPermanent(g.FindPermanent(ap.ID()))
				a.RemoveFromGraveyard(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
				g.Attach(card.ID(), other.ID())
			case "phase-after":
				g.PhaseOut(ap.ID())
			case "removed-after":
				g.AddContinuousEffect(mage.RemoveAllAbilities(ap.ID(), core.Indefinite))
			}
			g.PutTriggersOnStack()
			count := 1
			if change == "phase-before" || change == "removed-before" {
				count = 0
			}
			if g.StackSize() != count {
				t.Fatalf("stack=%d want=%d", g.StackSize(), count)
			}
			clone := g.Clone()
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy")
			}
			if change == "prevention" {
				g.AddPreventionShield(b.PlayerID(), 2)
			}
			g.ResolveStack()
			damage := 5 * count
			if change == "copy" {
				damage *= 2
			}
			if change == "prevention" {
				damage -= 2
			}
			payer, untouched := b, a
			if change == "control-before" {
				payer, untouched = a, b
			}
			if payer.Life() != 20-damage || untouched.Life() != 20 {
				t.Fatalf("life=(%d,%d), damage=%d", a.Life(), b.Life(), damage)
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("shared clone damage")
			}
			clone.ResolveStack()
			if clone.GetPlayer(payer.PlayerID()).Life() != 20-5*count {
				t.Fatal("clone lost saved controller or toughness")
			}
		})
	}
}

func TestRONHostDeathCasting(t *testing.T) {
	g, a, b := ronGame()
	hostCard := mage.NewCreature("Host", "", 2, 4)
	hostCard.SetOwner(b.PlayerID())
	host := g.PutOnBattlefield(hostCard, b.PlayerID())
	c := repositoryCard(t, "LEA", "Creature Bond")
	c.SetOwner(a.PlayerID())
	a.AddToHand(c)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
		t.Fatal("unpaid cast")
	}
	a.ManaPool().Add(core.Blue, 2)
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{b.PlayerID()}, 0); err == nil || a.ManaPool().TotalMana() != 2 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("illegal target changed payment")
	}
	if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(c.ID()).AttachedTo != host.ID() || a.ManaPool().TotalMana() != 0 {
		t.Fatal("Aura cost or attachment")
	}
	g.DestroyPermanent(g.FindPermanent(host.ID()))
	g.CheckStateBasedActions()
	g.PutTriggersOnStack()
	g.ResolveStack()
	if b.Life() != 16 || a.Life() != 20 || g.FindPermanent(c.ID()) != nil {
		t.Fatal("death damage after Aura state check")
	}
}

func TestRONHostDeathInvalid(t *testing.T) {
	valid := `TriggerA(WhenAPermanentDies(SinglePermanent(HostPermanent)),ActionList([DeadGuestPermanentDealsDamage(ToughnessOfDeadPermanent,Player(ControllerOfDeadPermanent))]))`
	for _, rule := range []string{
		strings.Replace(valid, "HostPermanent", "HostPermanent()", 1), strings.Replace(valid, "SinglePermanent(HostPermanent)", "HostPermanent", 1),
		strings.Replace(valid, "ToughnessOfDeadPermanent", "ToughnessOfDeadPermanent()", 1), strings.Replace(valid, "ControllerOfDeadPermanent", "ControllerOfDeadPermanent()", 1),
		strings.Replace(valid, "HostPermanent", "Ref_TargetPermanent", 1), strings.Replace(valid, "WhenAPermanentDies", "WhenAPermanentEntersTheBattlefield", 1),
		`Activated(PayMana([]),ActionList([DeadGuestPermanentDealsDamage(Integer(1),Player(You))]))`,
		valid + `,Activated(PayMana([]),ActionList([PlayerAction(ControllerOfDeadPermanent,GainLife(Integer(1)))]))`,
		valid + `,Activated(PayMana([]),ActionList([PlayerAction(OwnerOfDeadPermanent,GainLife(Integer(1)))]))`,
		valid + `,Activated(PayMana([]),ActionList([PutGraveyardCardIntoHand(Trigger_ThatGraveyardCard)]))`,
		valid + `,Activated(PayMana([]),ActionList([PermanentDealsDamage(Trigger_ThatPermanent,Integer(1),Player(You))]))`,
		strings.Replace(valid, "Player(ControllerOfDeadPermanent)", "Player(OwnerOfDeadPermanent())", 1),
		strings.Replace(valid, "ToughnessOfDeadPermanent,Player", "ToughnessOfDeadPermanent,Integer(1),Player", 1),
		valid + `,Activated(PayMana([]),ActionList([PermanentDealsDamage(ThisPermanent,ToughnessOfDeadPermanent,Player(You))]))`,
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),%s])`, rule)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + valid + `])`
	nodes, err := Parse(strings.NewReader(src), "invalid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("non-Aura host accepted")
	}
}

func TestRONHostDeathIndependentEvents(t *testing.T) {
	node := hostUntapRecord(t, "LEA", "Creature Bond")
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	for i, player := range []*mage.BasePlayer{a, b} {
		host := mage.NewCreature("Host", "", 2, 3+i)
		host.SetOwner(player.PlayerID())
		hp := g.PutOnBattlefield(host, player.PlayerID())
		aura := factory()
		aura.SetOwner(a.PlayerID())
		ap := g.PutOnBattlefield(aura, a.PlayerID())
		g.Attach(ap.ID(), hp.ID())
		g.DestroyPermanent(g.FindPermanent(hp.ID()))
	}
	clone := g.Clone()
	for _, game := range []*mage.Game{clone, g} {
		game.PutTriggersOnStack()
		if game.StackSize() != 2 {
			t.Fatal("lost pending event")
		}
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 17 || game.GetPlayer(b.PlayerID()).Life() != 16 {
			t.Fatal("death facts shared across instances")
		}
		if game == clone && (a.Life() != 20 || b.Life() != 20) {
			t.Fatal("clone shared life")
		}
	}
}

func TestRONHostDeathSavedReferences(t *testing.T) {
	for _, change := range []string{"normal", "visit", "returned-source", "copy"} {
		t.Run(change, func(t *testing.T) {
			src := `Card(Name:"Death references",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(AnyPermanent),TriggerA(WhenAPermanentDies(SinglePermanent(HostPermanent)),ActionList([PlayerAction(OwnerOfDeadPermanent,GainLife(Integer(1))),PermanentDealsDamage(Trigger_ThatPermanent,Integer(2),Player(ControllerOfDeadPermanent)),PutGraveyardCardIntoHand(Trigger_ThatGraveyardCard)]))])`
			nodes, err := Parse(strings.NewReader(src), "death.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			host := mage.NewArtifact("Host", "")
			host.SetOwner(a.PlayerID())
			hp := g.PutOnBattlefield(host, b.PlayerID())
			aura := factory()
			aura.SetOwner(a.PlayerID())
			ap := g.PutOnBattlefield(aura, a.PlayerID())
			g.Attach(ap.ID(), hp.ID())
			if err := mage.ValidateCardDefinition(aura.Name(), aura); err != nil {
				t.Fatal(err)
			}
			g.DestroyPermanent(g.FindPermanent(hp.ID()))
			if change == "visit" {
				a.RemoveFromGraveyard(host.ID())
				a.AddToGraveyard(host)
			}
			if change == "returned-source" {
				a.RemoveFromGraveyard(host.ID())
				g.PutOnBattlefield(host, a.PlayerID())
			}
			from := mage.NewPermanentFilter("former controller", func(p *mage.Permanent, _ *mage.Game) bool {
				return p.HasType(core.TypeArtifact) && p.ControllerID() == b.PlayerID()
			})
			if err := mage.ApplyEffect(g, mage.PreventDamageThisTurn(mage.You, from), uuid.Nil, b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("host death did not trigger")
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy")
			}
			g.ResolveStack()
			gain := 1
			if change == "copy" {
				gain = 2
			}
			if a.Life() != 20+gain || b.Life() != 20 {
				t.Fatalf("lost owner, controller or departing source identity: life=(%d,%d), want=(%d,20)", a.Life(), b.Life(), 20+gain)
			}
			wantHand := 1
			if change == "visit" || change == "returned-source" {
				wantHand = 0
			}
			if len(a.Hand()) != wantHand {
				t.Fatal("moved another graveyard visit")
			}
		})
	}
}

func TestRONHostDeathDamageSource(t *testing.T) {
	for _, change := range []string{"live", "dead", "returned", "phased"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			host := mage.NewCreature("Host", "", 2, 3)
			host.SetOwner(b.PlayerID())
			hp := g.PutOnBattlefield(host, b.PlayerID())
			c := repositoryCard(t, "LEA", "Creature Bond")
			c.SetOwner(a.PlayerID())
			ap := g.PutOnBattlefield(c, a.PlayerID())
			g.Attach(ap.ID(), hp.ID())
			g.DestroyPermanent(g.FindPermanent(hp.ID()))
			switch change {
			case "dead", "returned":
				g.DestroyPermanent(g.FindPermanent(ap.ID()))
				if change == "returned" {
					a.RemoveFromGraveyard(c.ID())
					g.PutOnBattlefield(c, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(ap.ID())
			}
			from := mage.NewPermanentFilter("original Aura", func(p *mage.Permanent, _ *mage.Game) bool {
				return p.HasType(core.TypeEnchantment) && p.ControllerID() == a.PlayerID()
			})
			if err := mage.ApplyEffect(g, mage.PreventDamageThisTurn(mage.You, from), uuid.Nil, b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("trigger count")
			}
			g.ResolveStack()
			if b.Life() != 20 {
				t.Fatal("damage source was host or returned Aura")
			}
		})
	}
}

func TestRONHostDeathReprintsAndInventory(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	collection := fstest.MapFS{}
	records := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			if node.Fields["Name"].String != "Creature Bond" {
				continue
			}
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition("Creature Bond", factory()); err != nil {
				t.Fatal(err)
			}
			records++
			line := strings.Split(string(data), "\n")[node.Position.Line-1]
			collection[filepath.Base(file)] = &fstest.MapFile{Data: []byte(line)}
		}
	}
	if records != 6 {
		t.Fatalf("reprints=%d", records)
	}
	registry, err := LoadFS(collection, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("reprints counted as cards")
	}
	report, err := InspectFS(collection, "*.ron")
	if err != nil || report.Compiled != 6 {
		t.Fatal("inventory", err)
	}
	for name, role := range map[string]string{"HostPermanent": "permanent_reference", "ToughnessOfDeadPermanent": "value", "ControllerOfDeadPermanent": "player_reference", "Player": "recipient"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s/%s", name, role)
		}
	}
	collection["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Unsupported",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Unsupported])`)}
	if err := LoadInto(registry, collection, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("replaced registry after rejection")
	}
	if registry, err := LoadFS(collection, "*.ron"); err == nil || registry != nil {
		t.Fatal("published partial registry")
	}
}

func TestRONHostDeathRepeatedEvents(t *testing.T) {
	g, a, b := ronGame()
	c := repositoryCard(t, "LEA", "Creature Bond")
	c.SetOwner(a.PlayerID())
	ap := g.PutOnBattlefield(c, a.PlayerID())
	for i, player := range []*mage.BasePlayer{a, b} {
		host := mage.NewCreature("Host", "", 2, 3+i)
		host.SetOwner(player.PlayerID())
		hp := g.PutOnBattlefield(host, player.PlayerID())
		g.Attach(ap.ID(), hp.ID())
		g.DestroyPermanent(g.FindPermanent(hp.ID()))
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("lost repeated death event")
	}
	g.ResolveStack()
	if a.Life() != 17 || b.Life() != 16 {
		t.Fatal("pending events overwrote one ability's saved facts")
	}
}

func TestRONHostDeathCastingLegality(t *testing.T) {
	for _, change := range []string{"shroud-before", "shroud-after", "host-left", "land"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			hostCard := mage.NewCreature("Host", "", 2, 3)
			hostCard.SetOwner(b.PlayerID())
			hp := g.PutOnBattlefield(hostCard, b.PlayerID())
			c := repositoryCard(t, "LEA", "Creature Bond")
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			a.ManaPool().Add(core.Blue, 2)
			target := hp.ID()
			if change == "shroud-before" {
				g.MutablePermanent(hp.ID()).GrantBaseAttr(core.Shroud)
			}
			if change == "land" {
				target = g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID()).ID()
			}
			err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{target}, 0)
			if change == "land" || change == "shroud-before" {
				if err == nil || a.ManaPool().TotalMana() != 2 || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("illegal target changed payment")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if change == "shroud-after" {
				g.MutablePermanent(hp.ID()).GrantBaseAttr(core.Shroud)
			} else {
				g.DestroyPermanent(g.FindPermanent(hp.ID()))
			}
			g.ResolveStack()
			if g.FindPermanent(c.ID()) != nil || g.StackSize() != 0 || b.Life() != 20 || len(a.Graveyard()) != 1 {
				t.Fatal("illegal Aura resolved or triggered")
			}
		})
	}
}
