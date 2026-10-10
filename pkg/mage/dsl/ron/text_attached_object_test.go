package ron

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type attachedObjectCase struct {
	name, set, consumer, from, to string
	color                         core.Color
	keyword                       core.Attr
	power, toughness              int
	mana                          core.Color
}

func attachedObjectCases() []attachedObjectCase {
	return []attachedObjectCase{
		{name: "Black Ward", color: core.Black, from: "Black"}, {name: "Blue Ward", color: core.Blue, from: "Blue"}, {name: "Green Ward", color: core.Green, from: "Green"}, {name: "Red Ward", color: core.Red, from: "Red"}, {name: "White Ward", color: core.White, from: "White"},
		{name: "Burrowing", consumer: "Magical Hack", from: "Mountain", to: "Forest", keyword: core.Forestwalk}, {name: "Fishliver Oil", set: "ARN", consumer: "Magical Hack", from: "Island", to: "Forest", keyword: core.Forestwalk},
		{name: "Flight", keyword: core.Flying}, {name: "Fear", keyword: core.Fear}, {name: "Lance", keyword: core.FirstStrike}, {name: "Eternal Warrior", set: "LEG", keyword: core.Vigilance},
		{name: "Firebreathing", power: 1, mana: core.Red}, {name: "Blessing", power: 1, toughness: 1, mana: core.White}, {name: "Regeneration", mana: core.Green},
	}
}

func TestRONTextAttachedObjectRecords(t *testing.T) {
	counts := map[string]int{}
	for _, tc := range attachedObjectCases() {
		sets := []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"}
		if tc.set != "" {
			sets = []string{tc.set}
		}
		for _, set := range sets {
			counts[tc.name]++
			t.Run(set+"/"+tc.name, func(t *testing.T) {
				g, a, b := ronGame()
				host := g.PutOnBattlefield(mage.NewCreature("Host", "{G}", 2, 3), b.PlayerID())
				other := g.PutOnBattlefield(mage.NewCreature("Other", "{G}", 2, 3), a.PlayerID())
				card := repositoryCard(t, set, tc.name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(core.White, 2)
				a.ManaPool().Add(core.Blue, 2)
				a.ManaPool().Add(core.Black, 2)
				a.ManaPool().Add(core.Red, 2)
				a.ManaPool().Add(core.Green, 2)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				source := g.FindPermanent(card.ID())
				if source == nil || source.AttachedTo != host.ID() {
					t.Fatal("Aura did not enter attached")
				}
				noMatch, _ := mage.ReplaceBasicLandWord(core.PlainsLandType, core.SwampLandType)
				if err := g.AddPrintedAttachedObjectTextEffect(source, core.EndOfTurn, noMatch); err != nil {
					t.Fatal(err)
				}
				id := source.RuntimeAbilities[0].AbilityID()
				consumer, from, to := tc.consumer, tc.from, tc.to
				if consumer == "" {
					consumer = "Sleight of Mind"
				}
				if from == "" {
					from = "Red"
				}
				if to == "" {
					to = "Green"
				}
				if from == to {
					to = "Blue"
				}
				castActivatedObjectWords(t, g, a, source, consumer, from, to)
				source = g.FindPermanent(card.ID())
				if source.RuntimeAbilities[0].AbilityID() != id || source.Card.ManaCost().String() != card.ManaCost().String() {
					t.Fatal("text changed declaration identity or mana symbols")
				}
				if strings.HasSuffix(tc.name, "Ward") {
					next := core.Green
					if tc.color == core.Green {
						next = core.Blue
					}
					if !g.FindPermanent(host.ID()).HasProtectionFrom(textColorCard(next)) || g.FindPermanent(host.ID()).HasProtectionFrom(textColorCard(tc.color)) || g.FindPermanent(other.ID()).HasProtectionFrom(textColorCard(next)) {
						t.Fatal("Ward text did not change only its host")
					}
					change, _ := mage.ReplaceColorWord(next, core.White)
					if next != core.White {
						if err := g.AddPrintedAttachedObjectTextEffect(source, core.Indefinite, change); err != nil {
							t.Fatal(err)
						}
					}
					if g.FindPermanent(host.ID()).HasProtectionFrom(card) || !g.FindPermanent(host.ID()).HasProtectionFrom(repositoryCard(t, "LEA", "White Ward")) {
						t.Fatal("Ward exception lost its exact donor")
					}
				} else if tc.keyword != 0 {
					if !g.FindPermanent(host.ID()).HasKeyword(tc.keyword) || g.FindPermanent(other.ID()).HasKeyword(tc.keyword) {
						t.Fatal("host grant was lost or applied to another creature")
					}
					if tc.consumer == "Magical Hack" {
						g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
						if !mage.HasLandwalkEvasion(g.FindPermanent(host.ID()), a.PlayerID(), g) {
							t.Fatal("changed landwalk has no evasion")
						}
					}
				} else {
					a.ManaPool().Clear()
					b.ManaPool().Add(tc.mana, 1)
					if err := g.ActivateAbilityByIndex(b.PlayerID(), card.ID(), 0, nil); err == nil {
						t.Fatal("host controller activated donor ability")
					}
					if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err == nil {
						t.Fatal("unpaid donor activated")
					}
					a.ManaPool().Add(tc.mana, 1)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					if g.StackSize() != 1 || g.FindPermanent(host.ID()).CurrentPower(g) != 2 {
						t.Fatal("activation skipped stack timing")
					}
					clone := g.Clone()
					g.ResolveStack()
					clone.ResolveStack()
					for _, game := range []*mage.Game{g, clone} {
						if tc.name == "Regeneration" {
							game.DestroyPermanent(game.FindPermanent(host.ID()))
							if game.FindPermanent(host.ID()) == nil || !game.FindPermanent(host.ID()).Tapped {
								t.Fatal("changed donor lost regeneration shield")
							}
						} else {
							staticPTCheck(t, game, host, 2+tc.power, 3+tc.toughness)
						}
					}
				}
			})
		}
	}
	want := map[string]int{}
	for _, tc := range attachedObjectCases() {
		want[tc.name] = 6
		if tc.set != "" {
			want[tc.name] = 1
		}
	}
	if !maps.Equal(counts, want) {
		t.Fatalf("record mapping changed: %v", counts)
	}
}

func TestRONTextAttachedObjectLifecycle(t *testing.T) {
	for _, state := range []string{"present", "phased", "removed"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			host := g.PutOnBattlefield(repositoryCard(t, "LEA", "White Knight"), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 3), a.PlayerID())
			card := repositoryCard(t, "LEA", "Red Ward")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(source.ID(), host.ID())
			g.ApplyContinuousEffects()
			id := source.RuntimeAbilities[0].AbilityID()
			first, _ := mage.ReplaceColorWord(core.Red, core.Blue)
			second, _ := mage.ReplaceColorWord(core.Blue, core.White)
			if err := g.AddPrintedAttachedObjectTextEffect(source, core.EndOfTurn, first); err != nil {
				t.Fatal(err)
			}
			if err := g.AddPrintedAttachedObjectTextEffect(g.FindPermanent(card.ID()), core.Indefinite, second); err != nil {
				t.Fatal(err)
			}
			check := func(game *mage.Game, target *mage.Permanent, color core.Color, want bool) {
				t.Helper()
				if game.FindPermanent(target.ID()).HasProtectionFrom(textColorCard(color)) != want {
					t.Fatalf("protection %v: want %v", color, want)
				}
			}
			check(g, host, core.White, true)
			check(g, host, core.Red, false)
			check(g, host, core.Black, true)
			castActivatedObjectWords(t, g, a, host, "Sleight of Mind", "Black", "Green")
			check(g, host, core.Green, true)
			check(g, host, core.Black, false)
			check(g, host, core.White, true)
			clone := g.Clone()
			clone.GetEffects().RemoveEndOfTurn()
			clone.ApplyContinuousEffects()
			check(clone, host, core.Red, true)
			check(clone, host, core.White, false)
			check(g, host, core.White, true)
			g.Attach(source.ID(), other.ID())
			g.ApplyContinuousEffects()
			check(g, host, core.White, false)
			check(g, other, core.White, true)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			if g.FindPermanent(card.ID()).ControllerID() != b.PlayerID() || g.FindPermanent(card.ID()).RuntimeAbilities[0].AbilityID() != id {
				t.Fatal("control change lost declaration")
			}
			check(g, other, core.White, true)
			var phased []uuid.UUID
			var remover *mage.Permanent
			switch state {
			case "phased":
				phased = g.PhaseOut(other.ID())
				g.ApplyContinuousEffects()
			case "removed":
				remover = g.PutOnBattlefield(mage.NewArtifact("Remover", ""), a.PlayerID())
				removal := mage.RemoveAllAbilities(card.ID(), core.WhileOnBattlefield)
				removal.SetSourceID(remover.ID())
				g.AddContinuousEffect(removal)
				check(g, other, core.White, false)
			}
			if state != "present" {
				if err := g.AddPrintedAttachedObjectTextEffect(source, core.Indefinite, first); err == nil {
					t.Fatal("inactive donor accepted text")
				}
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if state == "phased" {
				g.PhaseIn(phased)
				g.ApplyContinuousEffects()
			}
			if state == "removed" {
				g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
			}
			check(g, other, core.Red, true)
			check(g, other, core.White, false)
			third, _ := mage.ReplaceColorWord(core.Red, core.Blue)
			if err := g.AddPrintedAttachedObjectTextEffect(g.FindPermanent(card.ID()), core.Indefinite, third); err != nil {
				t.Fatal(err)
			}
			values, err := g.CaptureCopiableValues(g.FindPermanent(card.ID()))
			if err != nil {
				t.Fatal(err)
			}
			printed, err := values.Build()
			if err != nil {
				t.Fatal(err)
			}
			printed.SetOwner(a.PlayerID())
			donor := g.PutOnBattlefield(printed, a.PlayerID())
			g.Attach(donor.ID(), host.ID())
			g.ApplyContinuousEffects()
			check(g, host, core.Red, true)
			check(g, host, core.Blue, false)
			g.BouncePermanentToHand(g.FindPermanent(other.ID()))
			if g.LKI(other.ID()) == nil || !g.LKI(other.ID()).Snapshot.HasProtectionFrom(textColorCard(core.Blue)) {
				t.Fatal("host departure lost final protection LKI")
			}
			g.BouncePermanentToHand(g.FindPermanent(card.ID()))
			returned := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(returned.ID(), host.ID())
			g.ApplyContinuousEffects()
			if err := g.AddPrintedAttachedObjectTextEffect(source, core.Indefinite, first); err == nil {
				t.Fatal("old Aura visit followed return")
			}
			check(g, host, core.Red, true)
			check(g, host, core.Blue, false)
		})
	}
}

func TestRONTextAttachedObjectActivationBindings(t *testing.T) {
	for _, state := range []string{"reattached", "source-left", "host-returned", "phased-host", "control"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			hostCard := mage.NewCreature("Host", "", 2, 3)
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 3), a.PlayerID())
			card := repositoryCard(t, "LEA", "Firebreathing")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(source.ID(), host.ID())
			g.ApplyContinuousEffects()
			change, _ := mage.ReplaceColorWord(core.Red, core.Green)
			if err := g.AddPrintedAttachedObjectTextEffect(source, core.Indefinite, change); err != nil {
				t.Fatal(err)
			}
			a.ManaPool().Add(core.Red, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			pending := g.StackPeek()
			if err := g.AddPrintedAttachedObjectTextEffect(g.FindPermanent(source.ID()), core.EndOfTurn, change); err != nil {
				t.Fatal(err)
			}
			if g.StackPeek() != pending {
				t.Fatal("text installation replaced pending activation")
			}
			switch state {
			case "reattached":
				g.Attach(source.ID(), other.ID())
				g.ApplyContinuousEffects()
			case "source-left":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			case "host-returned":
				g.BouncePermanentToHand(g.FindPermanent(host.ID()))
				g.PutOnBattlefield(hostCard, b.PlayerID())
			case "phased-host":
				g.PhaseOut(host.ID())
				g.ApplyContinuousEffects()
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if state == "phased-host" {
					if game.FindPermanent(host.ID()) != nil {
						t.Fatal("phased host became present")
					}
					continue
				}
				power := 3
				if state == "host-returned" || state == "reattached" {
					power = 2
				}
				staticPTCheck(t, game, host, power, 3)
				otherPower := 2
				if state == "reattached" {
					otherPower = 3
				}
				staticPTCheck(t, game, other, otherPower, 3)
			}
			if state == "control" {
				b.ManaPool().Add(core.Red, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err == nil {
					t.Fatal("former controller activated donor")
				}
				if err := g.ActivateAbilityByIndex(b.PlayerID(), card.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				staticPTCheck(t, g, host, 4, 3)
				staticPTCheck(t, clone, host, 3, 3)
			}
		})
	}
}

func TestRONTextAttachedObjectBoundaries(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"LEA", "Consecrate Land"}, {"LEA", "Holy Armor"}, {"LEA", "Farmstead"}, {"LEA", "Goblin King"}, {"LEA", "Holy Strength"}, {"LEA", "Evil Presence"}, {"LEG", "Spectral Cloak"}, {"FEM", "Thrull Retainer"}} {
		g, a, _ := ronGame()
		card := repositoryCard(t, tc.set, tc.name)
		p := g.PutOnBattlefield(card, a.PlayerID())
		change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
		if err := g.AddPrintedAttachedObjectTextEffect(p, core.Indefinite, change); err == nil {
			t.Fatalf("unsupported complete context accepted: %s", tc.name)
		}
	}
	for _, body := range []string{
		`CreatePermanentLayerEffect(HostPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheSecondChosenColorWord)])`,
		`ChooseTwoColorWords,CreatePermanentLayerEffect(HostPermanent,[ReplaceLandTypeVariableWithNewLandTypeVariable(TheFirstChosenColorWord,TheSecondChosenColorWord)])`,
		`ChooseTwoColorWords,CreatePermanentLayerEffect(HostPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheFirstChosenColorWord)])`,
		`ChooseTwoColorWords,CreatePermanentLayerEffect(HostPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheSecondChosenColorWord),AdjustPT(1,1)])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),Activated(PayMana([]),ActionList([`+body+`]))])`), "invalid-attached.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid word declaration accepted")
		}
	}
	for _, rule := range []string{
		`EnchantPermanent(IsCardtype(Creature)),PermanentLayerEffect(HostPermanent,[AddAbility([])])`,
		`EnchantPermanent(IsCardtype(Creature)),PermanentLayerEffect(HostPermanent,[AddAbility([ProtectionAndDoesntRemovePermanents(FromColor(Colors([Red])),SinglePermanent(HostPermanent))])])`,
		`EnchantPermanent(IsCardtype(Creature)),PermanentLayerEffect(HostPermanent,[AddAbility([Landwalk(IsLandType(Invalid))])])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[`+rule+`])`), "invalid-attached.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid Aura declaration accepted")
		}
	}
}

func TestRONTextAttachedObjectCopiesAndLegality(t *testing.T) {
	g, a, b := ronGame()
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 3), b.PlayerID())
	land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	card := repositoryCard(t, "LEA", "Blue Ward")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{land.ID()}, 0); err == nil || a.ManaPool().Count(core.White) != 1 {
		t.Fatal("illegal enchant target spent mana")
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("Aura spell copy failed")
	}
	g.ResolveStack()
	token := g.FindPermanent(copied.SourceID)
	if token == nil || !token.Card.IsToken() || token.AttachedTo != host.ID() {
		t.Fatal("Aura copy lost token identity or attachment")
	}
	change, _ := mage.ReplaceColorWord(core.Blue, core.White)
	if err := g.AddPrintedAttachedObjectTextEffect(g.FindPermanent(card.ID()), core.Indefinite, change); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(host.ID()).HasProtectionFrom(textColorCard(core.Blue)) || !g.FindPermanent(host.ID()).HasProtectionFrom(textColorCard(core.White)) {
		t.Fatal("Aura instances shared their changed grant")
	}
	if g.FindPermanent(host.ID()).HasProtectionFrom(card) || !g.FindPermanent(host.ID()).HasProtectionFrom(token.Card) {
		t.Fatal("copied Aura received original exception")
	}
	g.CheckStateBasedActions()
	if g.FindPermanent(token.ID()) != nil || g.FindPermanent(card.ID()) == nil || g.FindPermanent(host.ID()).HasProtectionFrom(textColorCard(core.Blue)) {
		t.Fatal("white protection did not remove only the other Aura")
	}
}

func TestRONTextAttachedObjectInventory(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
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
			if !attachedObjectTextRules(node.Fields["Rules"].Args) {
				continue
			}
			counts[node.Fields["Name"].String]++
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 3), a.PlayerID())
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			g.Attach(source.ID(), host.ID())
			g.ApplyContinuousEffects()
			change, _ := mage.ReplaceColorWord(core.Red, core.Green)
			if err := g.AddPrintedAttachedObjectTextEffect(source, core.EndOfTurn, change); err != nil {
				t.Fatalf("%s: %v", node.Fields["Name"].String, err)
			}
		}
	}
	want := map[string]int{}
	for _, tc := range attachedObjectCases() {
		want[tc.name] = 6
		if tc.set != "" {
			want[tc.name] = 1
		}
	}
	if !maps.Equal(counts, want) {
		t.Fatalf("complete context inventory changed: %v", counts)
	}
}
