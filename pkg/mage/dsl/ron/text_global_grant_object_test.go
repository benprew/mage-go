package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func grantObjectConsumer(t *testing.T, name string) mage.CardFactory {
	t.Helper()
	record := staticBaseRecord(t, "LEA", name)
	record.Fields["Rules"].Args[0].Args[0] = record.Fields["Rules"].Args[0].Args[0].Args[0].Args[1]
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONTextGlobalGrantObjectRecords(t *testing.T) {
	for _, tc := range []struct {
		set, name, subtype, from string
		keyword                  core.Attr
		controlled               bool
		power, toughness         int
	}{
		{"LEA", "Goblin King", "Goblin", "Mountain", core.Mountainwalk, false, 1, 1},
		{"LEA", "Lord of Atlantis", "Merfolk", "Island", core.Islandwalk, false, 1, 1},
		{"LEA", "Zombie Master", "Zombie", "Swamp", core.Swampwalk, false, 0, 0},
		{"DRK", "Hidden Path", "Goblin", "Forest", core.Forestwalk, false, 0, 0},
		{"LEG", "Fortified Area", "Wall", "", core.Banding, true, 1, 0},
		{"LEG", "Kobold Drill Sergeant", "Kobold", "", core.Trample, true, 0, 1},
		{"LEG", "Kobold Overlord", "Kobold", "", core.FirstStrike, true, 0, 0},
		{"LEG", "Concordant Crossroads", "Goblin", "", core.Haste, false, 0, 0},
		{"FEM", "Goblin War Drums", "Goblin", "", core.Menace, true, 0, 0},
	} {
		sets := []string{tc.set}
		if tc.set == "LEA" {
			sets = []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"}
		}
		for _, set := range sets {
			t.Run(set+"/"+tc.name, func(t *testing.T) {
				g, base, b := ronGame()
				name, from, to := "Magical Hack", tc.from, "Plains"
				keyword := core.Plainswalk
				if from == "" {
					name, from, to, keyword = "Sleight of Mind", "Red", "Blue", tc.keyword
				}
				a := &intrinsicWordTester{Player: base, words: []string{from, to}}
				g.SetPlayerAt(0, a)
				card := repositoryCard(t, set, tc.name)
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				own := g.PutOnBattlefield(mage.NewCreature("Own", "{G}", 1, 2, mage.WithSubTypes(tc.subtype)), a.PlayerID())
				other := g.PutOnBattlefield(mage.NewCreature("Other", "{G}", 1, 2, mage.WithSubTypes(tc.subtype)), b.PlayerID())
				spell := grantObjectConsumer(t, name)()
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				a.ManaPool().Add(core.Blue, 1)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{source.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				if a.calls != 0 {
					t.Fatal("word choices ran during announcement")
				}
				g.ResolveStack()
				if a.calls != 2 {
					t.Fatal("word choices did not resolve")
				}
				for range 3 {
					g.ApplyContinuousEffects()
				}
				for i, p := range []*mage.Permanent{own, other} {
					active := !tc.controlled || i == 0
					current := g.FindPermanent(p.ID())
					if current.HasKeyword(keyword) != active {
						t.Fatalf("wrong revised grant on recipient %d", i)
					}
					if tc.from != "" && current.HasKeyword(tc.keyword) {
						t.Fatal("old landwalk remains")
					}
					power, toughness := 1, 2
					if active {
						power += tc.power
						toughness += tc.toughness
					}
					staticPTCheck(t, g, p, power, toughness)
					if tc.name == "Zombie Master" {
						if len(current.RuntimeAbilities) != 1 {
							t.Fatal("missing regeneration sibling")
						}
						actor := current.ControllerID()
						g.GetPlayer(actor).ManaPool().Add(core.Black, 1)
						if err := g.ActivateAbilityByIndex(actor, p.ID(), 0, nil); err != nil {
							t.Fatal(err)
						}
						g.ResolveStack()
						if !g.HasRegenerationShield(p.ID()) || g.HasRegenerationShield(source.ID()) {
							t.Fatal("grant body lost recipient source")
						}
					}
				}
				if tc.name == "Kobold Overlord" && !g.FindPermanent(source.ID()).HasKeyword(core.FirstStrike) {
					t.Fatal("fixed intrinsic keyword lost")
				}
				clone := g.Clone()
				g.PhaseOut(source.ID())
				g.ApplyContinuousEffects()
				if g.FindPermanent(own.ID()).HasKeyword(keyword) || !clone.FindPermanent(own.ID()).HasKeyword(keyword) {
					t.Fatal("phase or clone isolation failed")
				}
				g.PhaseIn([]uuid.UUID{source.ID()})
				g.ApplyContinuousEffects()
				if !g.FindPermanent(own.ID()).HasKeyword(keyword) {
					t.Fatal("phase cycle lost text")
				}
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if g.LKI(source.ID()) == nil || a.Hand()[0] != card {
					t.Fatal("departure lost LKI or physical card")
				}
				g.PutOnBattlefield(card, a.PlayerID())
				if tc.from != "" && (!g.FindPermanent(own.ID()).HasKeyword(tc.keyword) || g.FindPermanent(own.ID()).HasKeyword(keyword)) {
					t.Fatal("return kept prior text")
				}
			})
		}
	}
}

func TestRONTextGlobalGrantObjectZombieLifecycle(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEA", "Zombie Master")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	zombie := mage.NewCreature("Zombie", "", 2, 3, mage.WithSubTypes("Zombie"))
	zombie.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(zombie, b.PlayerID())
	artifact := g.PutOnBattlefield(mage.NewArtifact("Zombie artifact", "", mage.WithSubTypes("Zombie")), b.PlayerID())
	external := g.PutOnBattlefield(mage.NewEnchantment("External walk", "", mage.WithAbility(mage.StaticAbility(mage.GrantKeywordToAllIncludingSource(core.Plainswalk, mage.IsCreature)))), a.PlayerID())
	b.ManaPool().Add(core.Black, 2)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	id := g.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID()
	first, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.ForestLandType)
	second, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	color, _ := mage.ReplaceColorWord(core.Black, core.Blue)
	if err := g.AddPrintedGlobalGrantTextEffect(source, core.EndOfTurn, first, color); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedGlobalGrantTextEffect(g.FindPermanent(source.ID()), core.Indefinite, second); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
		if !g.FindPermanent(p.ID()).HasKeyword(core.Islandwalk) || g.FindPermanent(p.ID()).HasKeyword(core.Forestwalk) || g.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID() != id {
			t.Fatal("ordered text lost grant identity")
		}
	}
	if !g.FindPermanent(source.ID()).HasKeyword(core.Plainswalk) || g.FindPermanent(source.ID()).HasKeyword(core.Islandwalk) {
		t.Fatal("text changed granted words or source exclusion")
	}
	if len(g.FindPermanent(artifact.ID()).RuntimeAbilities) != 1 || g.FindPermanent(artifact.ID()).HasKeyword(core.Islandwalk) {
		t.Fatal("sibling filters merged")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), artifact.ID(), 0, nil); err != nil {
		t.Fatal("black mana symbol changed", err)
	}
	clone := g.Clone()
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if !clone.FindPermanent(p.ID()).HasKeyword(core.Swampwalk) || clone.FindPermanent(p.ID()).HasKeyword(core.Islandwalk) {
		t.Fatal("expiry did not recompute remaining words")
	}
	if !g.FindPermanent(p.ID()).HasKeyword(core.Islandwalk) {
		t.Fatal("clone changed original text")
	}
	values, err := g.CaptureCopiableValues(g.FindPermanent(source.ID()))
	if err != nil {
		t.Fatal(err)
	}
	printed, err := values.Build()
	if err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	lki := g.LKI(p.ID())
	if lki == nil || !lki.Snapshot.HasKeyword(core.Islandwalk) || !lki.Snapshot.HasKeyword(core.Plainswalk) {
		t.Fatal("LKI lost final recipient rules")
	}
	returned := g.PutOnBattlefield(zombie, b.PlayerID())
	g.ResolveStack()
	if g.HasRegenerationShield(returned.ID()) || !g.HasRegenerationShield(artifact.ID()) {
		t.Fatal("pending grants lost separate recipient incarnations")
	}
	clone.ResolveStack()
	if !clone.HasRegenerationShield(p.ID()) || !clone.HasRegenerationShield(artifact.ID()) {
		t.Fatal("clone lost pending grants")
	}
	phases := g.PhaseOut(source.ID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(returned.ID()).RuntimeAbilities) != 0 || g.FindPermanent(returned.ID()).HasKeyword(core.Islandwalk) {
		t.Fatal("phasing retained donor grants")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	g.PhaseIn(phases)
	g.ApplyContinuousEffects()
	if !g.FindPermanent(returned.ID()).HasKeyword(core.Swampwalk) {
		t.Fatal("phased expiry lost baseline")
	}
	g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
	if len(g.FindPermanent(returned.ID()).RuntimeAbilities) != 0 || g.FindPermanent(returned.ID()).HasKeyword(core.Swampwalk) {
		t.Fatal("ability removal retained donor")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if !g.FindPermanent(returned.ID()).HasKeyword(core.Swampwalk) {
		t.Fatal("removal expiry lost grant")
	}
	g.RemoveFromBattlefield(g.FindPermanent(external.ID()))
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	if err := g.AddPrintedGlobalGrantTextEffect(source, core.Indefinite, first); err == nil {
		t.Fatal("departed visit accepted")
	}
	g.PutOnBattlefield(printed, a.PlayerID())
	if !g.FindPermanent(returned.ID()).HasKeyword(core.Swampwalk) || g.FindPermanent(returned.ID()).HasKeyword(core.Islandwalk) {
		t.Fatal("copy values included ordinary text changes")
	}
}

func TestRONTextGlobalGrantObjectControl(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Kobold Overlord"), a.PlayerID())
	own := g.PutOnBattlefield(mage.NewCreature("Own", "", 1, 2, mage.WithSubTypes("Kobold")), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 2, mage.WithSubTypes("Kobold")), b.PlayerID())
	change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.AddPrintedGlobalGrantTextEffect(source, core.Indefinite, change); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if g.FindPermanent(own.ID()).HasKeyword(core.FirstStrike) || !g.FindPermanent(other.ID()).HasKeyword(core.FirstStrike) || !g.FindPermanent(source.ID()).HasKeyword(core.FirstStrike) {
		t.Fatal("changed control lost live selection or intrinsic rule")
	}
	if !clone.FindPermanent(own.ID()).HasKeyword(core.FirstStrike) || clone.FindPermanent(other.ID()).HasKeyword(core.FirstStrike) {
		t.Fatal("changed control leaked into clone")
	}
}

func TestRONTextGlobalGrantObjectPendingBindings(t *testing.T) {
	g, base, b := ronGame()
	a := &intrinsicWordTester{Player: base, words: []string{"Green", "Blue", "Green", "White"}}
	g.SetPlayerAt(0, a)
	first := g.PutOnBattlefield(repositoryCard(t, "DRK", "Hidden Path"), b.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "DRK", "Hidden Path"), b.PlayerID())
	factory := grantObjectConsumer(t, "Sleight of Mind")
	a.ManaPool().Add(core.Blue, 3)
	cast := func(p *mage.Permanent) {
		t.Helper()
		c := factory()
		c.SetOwner(a.PlayerID())
		a.AddToHand(c)
		if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
			t.Fatal(err)
		}
	}
	cast(first)
	cast(second)
	clone := g.Clone()
	clone.SetPlayerAt(0, &intrinsicWordTester{Player: clone.GetPlayer(a.PlayerID()), words: []string{"Green", "Red", "Green", "Black"}})
	g.ResolveStack()
	clone.ResolveStack()
	for _, tc := range []struct {
		game   *mage.Game
		color  core.Color
		active bool
	}{{g, core.Blue, true}, {g, core.White, true}, {g, core.Green, false}, {g, core.Red, false}, {clone, core.Red, true}, {clone, core.Black, true}, {clone, core.Blue, false}} {
		p := tc.game.PutOnBattlefield(textColorCard(tc.color), b.PlayerID())
		if tc.game.FindPermanent(p.ID()).HasKeyword(core.Forestwalk) != tc.active {
			t.Fatal("pending word bindings leaked", tc.color)
		}
	}
	cast(first)
	g.BouncePermanentToHand(g.FindPermanent(first.ID()))
	g.PutOnBattlefield(first.Card, b.PlayerID())
	calls := a.calls
	g.ResolveStack()
	if a.calls != calls {
		t.Fatal("returned target made resolution choices")
	}
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	if !g.FindPermanent(green.ID()).HasKeyword(core.Forestwalk) {
		t.Fatal("returned source inherited old text")
	}
}

func TestRONTextGlobalGrantObjectBoundaries(t *testing.T) {
	change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	for _, tc := range []struct{ set, name string }{{"LEA", "Gauntlet of Might"}, {"LEA", "Conversion"}, {"ARN", "Kird Ape"}, {"LEA", "Serra Angel"}, {"LEA", "Green Ward"}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, _ := ronGame()
			p := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
			before := g.FindPermanent(p.ID())
			if err := g.AddPrintedGlobalGrantTextEffect(p, core.Indefinite, change); err == nil {
				t.Fatal("unsupported complete context accepted")
			}
			if g.FindPermanent(p.ID()) != before {
				t.Fatal("rejection changed state")
			}
		})
	}
}

func TestRONTextGlobalGrantObjectActivatedBody(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `EachPermanentLayerEffect(IsCardtype(Creature),[AddAbility([Activated(PayMana([ManaCostR]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AddAbility([Protection(FromColor(Colors([Red])))])],UntilEndOfTurn)]))])])`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	p := g.PutOnBattlefield(mage.NewCreature("Recipient", "{G}", 1, 2), b.PlayerID())
	b.ManaPool().Add(core.Red, 2)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	id := g.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID()
	change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.AddPrintedGlobalGrantTextEffect(source, core.Indefinite, change); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID() != id {
		t.Fatal("changed body lost recipient identity")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal("red payment symbol changed", err)
	}
	if b.ManaPool().TotalMana() != 0 {
		t.Fatal("grant used donor payment")
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.ResolveTopOfStack()
	clone.ResolveTopOfStack()
	for _, game := range []*mage.Game{g, clone} {
		current := game.FindPermanent(p.ID())
		if !current.HasProtectionFrom(textColorCard(core.Blue)) || current.HasProtectionFrom(textColorCard(core.Red)) {
			t.Fatal("fresh grant body kept old words")
		}
	}
	g.ResolveTopOfStack()
	clone.ResolveTopOfStack()
	for _, game := range []*mage.Game{g, clone} {
		current := game.FindPermanent(p.ID())
		if !current.HasProtectionFrom(textColorCard(core.Blue)) || !current.HasProtectionFrom(textColorCard(core.Red)) {
			t.Fatal("pending activation used new words")
		}
	}
}
