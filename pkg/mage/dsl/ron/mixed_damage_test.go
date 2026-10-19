package ron

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONMixedDamageCards(t *testing.T) {
	for _, tc := range []struct {
		name, set      string
		color          core.Color
		amount, cost   int
		ground, flying bool
	}{
		{"Earthquake", "LEB", core.Red, 3, 4, true, false},
		{"Hurricane", "LEB", core.Green, 3, 4, false, true},
		{"Inferno", "DRK", core.Red, 6, 7, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			ground := g.PutOnBattlefield(mage.NewCreature("Ground", "", 1, 10, mage.WithKeyword(core.Shroud)), a.PlayerID())
			flying := g.PutOnBattlefield(mage.NewCreature("Flying", "", 1, 10, mage.WithKeyword(core.Flying)), b.PlayerID())
			land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), a.PlayerID())
			phased := g.PutOnBattlefield(mage.NewCreature("Phased", "", 1, 10), b.PlayerID())
			g.PhaseOut(phased.ID())
			card := repositoryCard(t, tc.set, tc.name)
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(tc.color, tc.cost)
			x := tc.amount
			if tc.name == "Inferno" {
				x = 0
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, x); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || a.Life() != 20 || ground.Damage != 0 {
				t.Fatal("payment or announcement timing")
			}
			callbacks := 0
			g.SetOnDamageDealt(func(source, target string, amount int, combat bool) {
				callbacks++
				gd, fd := 0, 0
				if tc.ground {
					gd = tc.amount
				}
				if tc.flying {
					fd = tc.amount
				}
				if source != tc.name || amount != tc.amount || combat || a.Life() != 20-tc.amount || b.Life() != 20-tc.amount || g.FindPermanent(ground.ID()).Damage != gd || g.FindPermanent(flying.ID()).Damage != fd {
					t.Errorf("callback saw incomplete or incorrect damage batch: %s -> %s", source, target)
				}
			})
			g.ResolveStack()
			want := 2
			if tc.ground {
				want++
			}
			if tc.flying {
				want++
			}
			if callbacks != want || land.Damage != 0 || g.FindPermanentIncludingPhased(phased.ID()).Damage != 0 {
				t.Fatal("recipient selection or notifications")
			}
			if len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != card.ID() {
				t.Fatal("spell did not finish in graveyard")
			}
		})
	}
}

func TestRONMixedDamageResolutionMembership(t *testing.T) {
	for _, name := range []string{"Earthquake", "Hurricane"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEB", name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			color := core.Red
			if name == "Hurricane" {
				color = core.Green
			}
			a.ManaPool().Add(color, 4)
			original := mage.NewCreature("Returned", "", 1, 10)
			old := g.PutOnBattlefield(original, b.PlayerID())
			changes := g.PutOnBattlefield(mage.NewCreature("Changes", "", 1, 10), b.PlayerID())
			out := g.PutOnBattlefield(mage.NewCreature("Out", "", 1, 10), b.PlayerID())
			in := g.PutOnBattlefield(mage.NewCreature("In", "", 1, 10), b.PlayerID())
			phased := g.PhaseOut(in.ID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 3); err != nil {
				t.Fatal(err)
			}
			g.BouncePermanentToHand(old)
			b.RemoveFromHand(original.ID())
			returned := g.PutOnBattlefield(original, a.PlayerID())
			g.PhaseOut(out.ID())
			g.PhaseIn(phased)
			changes = g.MutablePermanent(changes.ID())
			changes.GrantBaseAttr(core.Flying)
			later := g.PutOnBattlefield(mage.NewCreature("Later", "", 1, 10), b.PlayerID())
			g.ResolveStack()
			for _, p := range []*mage.Permanent{returned, in, later} {
				want := 3
				if name == "Hurricane" {
					want = 0
				}
				if g.FindPermanent(p.ID()).Damage != want {
					t.Fatal("collection was captured before resolution")
				}
			}
			want := 0
			if name == "Hurricane" {
				want = 3
			}
			if g.FindPermanent(changes.ID()).Damage != want || g.FindPermanentIncludingPhased(out.ID()).Damage != 0 {
				t.Fatal("current flying or phasing selection")
			}
		})
	}
}

func TestRONMixedDamageXCopiesAndRollback(t *testing.T) {
	for _, name := range []string{"Earthquake", "Hurricane"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			color := core.Red
			if name == "Hurricane" {
				color = core.Green
			}
			card := repositoryCard(t, "LEB", name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(color, 3)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 3); err == nil || a.ManaPool().TotalMana() != 3 || len(a.Hand()) != 1 || !g.Stack().IsEmpty() {
				t.Fatal("insufficient payment changed state")
			}
			a.ManaPool().Add(color, 1)
			if len(card.Abilities()[0].(*mage.ActionDefinition).Targets()) != 0 {
				t.Fatal("collection damage must not declare targets")
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 3); err != nil {
				t.Fatal(err)
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			g.ResolveStack()
			if a.Life() != 14 || b.Life() != 14 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("copy X or clone isolation")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 14 {
				t.Fatal("clone lost copied X")
			}
			zero := repositoryCard(t, "LEB", name)
			zero.SetOwner(a.PlayerID())
			a.AddToHand(zero)
			a.ManaPool().Add(color, 1)
			notifications := 0
			g.SetOnDamageDealt(func(string, string, int, bool) { notifications++ })
			if err := g.CastSpellByID(a.PlayerID(), zero.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if a.Life() != 14 || b.Life() != 14 || notifications != 0 {
				t.Fatal("zero damage produced an event")
			}
		})
	}
}

func TestRONMixedDamageReplacementOrder(t *testing.T) {
	for choice := range 2 {
		a := mage.NewBasePlayer("A")
		b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B"), mode: choice}
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		g.SetTurn(1)
		victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 10), b.PlayerID())
		card := repositoryCard(t, "DRK", "Inferno")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Red, 7)
		g.SetCreatureDamageRedirect(victim.ID(), b.PlayerID())
		g.AddPreventionShield(victim.ID(), 2)
		g.AddPreventionShield(a.PlayerID(), 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if a.Life() != 15 || b.Life() != 8+2*choice || victim.Damage != 0 || b.prompts != 1 {
			t.Fatalf("replacement ordering: A=%d B=%d damage=%d prompts=%d", a.Life(), b.Life(), victim.Damage, b.prompts)
		}
	}
}

func TestRONMixedDamageInvalid(t *testing.T) {
	for _, recipient := range []string{
		`MultipleRecipients`, `MultipleRecipients()`, `MultipleRecipients([])`, `MultipleRecipients([EachPlayer(AnyPlayer)])`,
		`MultipleRecipients([EachPermanent(IsCardtype(Creature)),EachPermanent(IsCardtype(Creature))])`,
		`MultipleRecipients([EachPlayer(AnyPlayer),EachPlayer(AnyPlayer)])`,
		`MultipleRecipients([EachPermanent(IsCardtype(Creature)),EachPlayer(AnyPlayer),EachPlayer(AnyPlayer)])`,
		`MultipleRecipients([EachPermanent(),EachPlayer(AnyPlayer)])`, `MultipleRecipients([EachPermanent(IsCardtype(Creature)),EachPlayer(AnyPlayer())])`,
		`MultipleRecipients([EachPermanent(IsCardtype(Creature)),EachPlayer(Opponent())])`,
		`MultipleRecipients([EachPermanent(IsCardtype(Creature)),Player(You())])`,
		`MultipleRecipients([EachPermanent(SinglePermanent(Ref_TargetPermanent)),EachPlayer(AnyPlayer)])`,
		`MultipleRecipients([EachPermanent(WasDealtDamageByPermanentThisGame(ThisPermanent)),EachPlayer(AnyPlayer)])`,
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([SpellDealsDamage(ThisSpell,Integer(1),%s)]))])`, recipient)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", recipient)
		}
	}
}

func TestRONMixedDamageStrictReprintsAndInventory(t *testing.T) {
	names := map[string]bool{"Earthquake": true, "Hurricane": true, "Inferno": true}
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
		lines := strings.Split(string(data), "\n")
		var selected []string
		for _, node := range nodes {
			if !names[node.Fields["Name"].String] {
				continue
			}
			records++
			selected = append(selected, lines[node.Position.Line-1])
		}
		if len(selected) > 0 {
			collection[filepath.Base(file)] = &fstest.MapFile{Data: []byte(strings.Join(selected, "\n"))}
		}
	}
	registry, err := LoadFS(collection, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if records != 13 || len(registry.RegisteredCardNames()) != 3 {
		t.Fatalf("records=%d unique=%d", records, len(registry.RegisteredCardNames()))
	}
	first, _ := registry.CreateCard("Earthquake")
	second, _ := registry.CreateCard("Earthquake")
	if first.ID() == second.ID() || first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
		t.Fatal("shared instance state")
	}
	report, err := InspectFS(collection, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"EachPermanent": "recipient_set", "EachPlayer": "recipient_set", "IsCardtype": "permanent_filter", "AnyPlayer": "player_filter"} {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s/%s inventory role", name, role)
		}
	}
	collection["gap.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Gap",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Unknown])`)}
	if registry, err := LoadFS(collection, "*.ron"); err == nil || registry != nil {
		t.Fatal("published incomplete collection")
	}
}

func TestRONMixedDamageAbilitySourceIdentity(t *testing.T) {
	for _, context := range []string{"activation", "entry-trigger"} {
		for _, scenario := range []string{"present", "return", "phase"} {
			t.Run(context+"/"+scenario, func(t *testing.T) {
				body := `ActionList([PermanentDealsDamage(ThisPermanent,Integer(2),MultipleRecipients([EachPlayer(AnyPlayer),EachPermanent(IsCardtype(Creature))]))])`
				rule := `Activated(TapPermanent(ThisPermanent),` + body + `)`
				if context == "entry-trigger" {
					rule = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),` + body + `)`
				}
				factory := permanentScopeFactory(t, "Artifact", rule)
				g, a, b := ronGame()
				card := factory()
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				source.GrantBaseAttr(core.Lifelink)
				victim := g.PutOnBattlefield(mage.NewCreature("Recipient", "", 1, 10), b.PlayerID())
				if context == "activation" {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
				} else {
					g.PutTriggersOnStack()
				}
				if a.Life() != 20 || victim.Damage != 0 {
					t.Fatal("damage before resolution")
				}
				if scenario == "return" {
					g.BouncePermanentToHand(source)
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				}
				if scenario == "phase" {
					g.PhaseOut(source.ID())
				}
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				wantA, wantB, wantDamage := 28, 16, 4
				if context == "entry-trigger" && scenario == "return" {
					wantA, wantB, wantDamage = 26, 14, 6
				}
				clone := g.Clone()
				g.ResolveStack()
				if a.Life() != wantA || b.Life() != wantB || g.FindPermanent(victim.ID()).Damage != wantDamage || clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatalf("source identity or clone: A=%d B=%d damage=%d", a.Life(), b.Life(), g.FindPermanent(victim.ID()).Damage)
				}
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != wantA {
					t.Fatal("clone lost source LKI")
				}
			})
		}
	}
}

func TestRONMixedDamageSimultaneousLethal(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Onulet"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Onulet"), b.PlayerID())
	card := repositoryCard(t, "DRK", "Inferno")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 7)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.SetOnDamageDealt(func(string, string, int, bool) {
		if g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) == nil || g.FindPermanent(first.ID()).Damage != 6 || g.FindPermanent(second.ID()).Damage != 6 {
			t.Error("lethal damage or removal occurred before complete batch")
		}
	})
	g.ResolveTopOfStack()
	if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil || a.Life() != 14 || b.Life() != 14 {
		t.Fatal("lethal damage state-based checks")
	}
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 16 || b.Life() != 16 {
		t.Fatal("death triggers lost controller or timing")
	}
}

func TestRONMixedDamageProtection(t *testing.T) {
	for _, tc := range []struct {
		name, set string
		color     core.Color
		cost, x   int
	}{
		{"Earthquake", "LEB", core.Red, 4, 3}, {"Hurricane", "LEB", core.Green, 4, 3}, {"Inferno", "DRK", core.Red, 7, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			options := []mage.CardOption{mage.WithAbility(mage.ProtectionFromColor(tc.color))}
			if tc.name == "Hurricane" {
				options = append(options, mage.WithKeyword(core.Flying))
			}
			protected := g.PutOnBattlefield(mage.NewCreature("Protected", "", 1, 10, options...), b.PlayerID())
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(tc.color, tc.cost)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, tc.x); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			amount := tc.x
			if tc.name == "Inferno" {
				amount = 6
			}
			if protected.Damage != 0 || a.Life() != 20-amount || b.Life() != 20-amount {
				t.Fatal("spell source protection or player damage")
			}
		})
	}
}
