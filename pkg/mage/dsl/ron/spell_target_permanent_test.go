package ron

import (
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONControlledPermanentCounters(t *testing.T) {
	for _, name := range []string{"Avoid Fate", "Ring of Immortals"} {
		for _, victimName := range []string{"Jump", "Holy Strength"} {
			t.Run(name+"/"+victimName, func(t *testing.T) {
				g, a, _ := ronGame()
				p := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), a.PlayerID())
				victimCard := repositoryCard(t, "LEB", victimName)
				var victim *mage.StackObject
				card := repositoryCard(t, "LEG", name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				if name == "Ring of Immortals" {
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					a.ManaPool().Add(core.Colorless, 5)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
						t.Fatal(err)
					}
					g.ResolveTopOfStack()
					victim = castRONSpell(t, g, a, victimCard, p.ID())
					a.ManaPool().Clear()
					if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, []uuid.UUID{victim.ID}); err == nil {
						t.Fatal("unpaid activation")
					}
					if g.FindPermanent(card.ID()).Tapped || g.StackSize() != 1 {
						t.Fatal("failed activation changed state")
					}
					a.ManaPool().Add(core.Colorless, 3)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, []uuid.UUID{victim.ID}); err != nil {
						t.Fatal(err)
					}
					if !g.FindPermanent(card.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
						t.Fatal("activation costs")
					}
				} else {
					victim = castRONSpell(t, g, a, victimCard, p.ID())
					a.ManaPool().Clear()
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID}, 0); err == nil {
						t.Fatal("unpaid spell")
					}
					a.ManaPool().Add(core.Green, 1)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID}, 0); err != nil {
						t.Fatal(err)
					}
				}
				if g.Stack().FindByID(victim.ID) == nil {
					t.Fatal("acted before resolution")
				}
				g.ResolveTopOfStack()
				if g.Stack().FindByID(victim.ID) != nil || !slices.ContainsFunc(a.Graveyard(), func(c mage.Card) bool { return c.ID() == victimCard.ID() }) {
					t.Fatal("spell was not countered")
				}
			})
		}
	}
}

func TestRONControlledPermanentCounterInvalid(t *testing.T) {
	for _, filter := range []string{
		`IsEnchantmentType`, `IsEnchantmentType()`, `IsEnchantmentType(Aura())`, `IsEnchantmentType(Unknown)`,
		`TargetsAPermanent`, `TargetsAPermanent()`, `TargetsAPermanent(AnyPermanent)`,
		`TargetsAPermanent(ControlledByAPlayer(You))`, `TargetsAPermanent(ControlledByAPlayer(SinglePlayer(Opponent)))`,
		`TargetsAPermanent(ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer)))`, `TargetsAPermanent(ControlledByAPlayer(SinglePlayer(You())))`,
		`And([AnySpell, TargetsAPermanent(ControlledByAPlayer(SinglePlayer(You))), TargetsAPermanent(ControlledByAPlayer(SinglePlayer(You)))])`,
		`Or([AnySpell, TargetsAPermanent(ControlledByAPlayer(SinglePlayer(You)))])`,
		`And([AnySpell, And([TargetsAPermanent(ControlledByAPlayer(SinglePlayer(You)))])])`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetSpell(` + filter + `)],ActionList([CounterSpell(Ref_TargetSpell)])))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", filter)
		}
	}
}

func TestRONControlledPermanentCounterRechecks(t *testing.T) {
	for _, name := range []string{"Avoid Fate", "Ring of Immortals"} {
		for _, change := range []string{"none", "shroud", "control", "departed", "returned", "phased", "phase-cycle", "retargeted", "source-leaves", "source-control", "source-phases", "copy", "uncounterable", "victim-copy"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				creature := repositoryCard(t, "LEB", "Air Elemental")
				creature.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(creature, a.PlayerID())
				victim := castRONSpell(t, g, b, repositoryCard(t, "LEB", "Jump"), p.ID())
				if change == "victim-copy" {
					victim = g.CopyStackObjectDirect(victim, b.PlayerID(), false)
				}
				card := repositoryCard(t, "LEG", name)
				var source *mage.Permanent
				if name == "Ring of Immortals" {
					source = g.PutOnBattlefield(card, a.PlayerID())
					a.ManaPool().Add(core.Colorless, 3)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID}); err != nil {
						t.Fatal(err)
					}
				} else {
					castRONSpell(t, g, a, card, victim.ID)
				}
				if change == "uncounterable" {
					mage.WithUncounterable()(victim.Card.(*mage.BaseCard))
				}
				clone := g.Clone()
				switch change {
				case "shroud":
					p.GrantBaseAttr(core.Shroud)
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "departed":
					g.BouncePermanentToHand(p)
				case "returned":
					g.BouncePermanentToHand(p)
					a.RemoveFromHand(creature.ID())
					g.PutOnBattlefield(creature, a.PlayerID())
				case "phased":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "retargeted":
					victim.Targets = []uuid.UUID{b.PlayerID()}
				case "source-leaves":
					if source != nil {
						g.BouncePermanentToHand(source)
					}
				case "source-control":
					if source != nil {
						g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					}
				case "source-phases":
					if source != nil {
						g.PhaseOut(source.ID())
					}
				case "copy":
					copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
					g.ResolveTopOfStack()
					if g.Stack().FindByID(copied.ID) != nil || g.Stack().FindByID(victim.ID) == nil {
						t.Fatal("copy used original controller")
					}

				}
				g.ResolveTopOfStack()
				want := change != "control" && change != "departed" && change != "returned" && change != "phased" && change != "retargeted" && change != "uncounterable"
				if got := g.Stack().FindByID(victim.ID) == nil; got != want {
					t.Fatalf("countered = %v, want %v", got, want)
				}
				if clone.Stack().FindByID(victim.ID) == nil {
					t.Fatal("live resolution changed clone")
				}
				clone.ResolveTopOfStack()
				if got := clone.Stack().FindByID(victim.ID) == nil; got != (change != "uncounterable") {
					t.Fatal("clone lost target or counterability")
				}
			})
		}
	}
}

func TestRONControlledPermanentCounterAnnouncement(t *testing.T) {
	for _, name := range []string{"Avoid Fate", "Ring of Immortals"} {
		for _, kind := range []string{"opponent", "no-targets", "player", "sorcery", "enchantment", "artifact", "ability", "nonexistent"} {
			t.Run(name+"/"+kind, func(t *testing.T) {
				g, a, b := ronGame()
				p := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), a.PlayerID())
				victimCard := mage.NewInstant("Victim", "")
				targets := []uuid.UUID{p.ID()}
				switch kind {
				case "opponent":
					targets = []uuid.UUID{g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), b.PlayerID()).ID()}
				case "no-targets":
					targets = nil
				case "player":
					targets = []uuid.UUID{a.PlayerID()}
				case "sorcery":
					victimCard = mage.NewSorcery("Victim", "")
				case "enchantment":
					victimCard = mage.NewEnchantment("Victim", "")
				case "artifact":
					victimCard = mage.NewArtifact("Victim", "")
				}
				victim := &mage.StackObject{ID: uuid.New(), SourceID: victimCard.ID(), Card: victimCard, Controller: b.PlayerID(), Targets: targets, IsAbility: kind == "ability"}
				g.PushStack(victim)
				id := victim.ID
				if kind == "nonexistent" {
					id = uuid.New()
				}
				card := repositoryCard(t, "LEG", name)
				a.ManaPool().Add(core.Green, 1)
				a.ManaPool().Add(core.Colorless, 3)
				var err error
				if name == "Ring of Immortals" {
					source := g.PutOnBattlefield(card, a.PlayerID())
					err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{id})
					if g.FindPermanent(source.ID()).Tapped {
						t.Fatal("illegal target paid tap cost")
					}
				} else {
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					err = g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{id}, 0)
					if len(a.Hand()) != 1 {
						t.Fatal("failed cast moved card")
					}
				}
				if err == nil || a.ManaPool().TotalMana() != 4 || g.StackSize() != 1 {
					t.Fatal("illegal target changed state")
				}
			})
		}
	}
}

func TestRONControlledPermanentCounterRegistryAndInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEG.ron")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	for _, node := range nodes {
		name := node.Fields["Name"].String
		if name != "Avoid Fate" && name != "Ring of Immortals" {
			continue
		}
		files[name+".ron"] = &fstest.MapFile{Data: []byte(strings.Split(string(data), "\n")[node.Position.Line-1])}
	}
	if len(files) != 2 {
		t.Fatal("missing complete records")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Avoid Fate", "Ring of Immortals"} {
		first, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		second, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		one := first.Abilities()[0].(*mage.ActionDefinition)
		two := second.Abilities()[0].(*mage.ActionDefinition)
		if one.Targets()[0] == two.Targets()[0] {
			t.Fatal("shared factory target")
		}
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{"TargetsAPermanent": "spell_filter", "ControlledByAPlayer": "permanent_filter", "SinglePlayer": "player_filter", "You": "player_reference"}
	for name, role := range roles {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing inventory role %s/%s", name, role)
		}
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetSpell(TargetsAPermanent(AnyPermanent))],ActionList([])))])`)}
	if partial, err := LoadFS(files, "*.ron"); err == nil || partial != nil {
		t.Fatal("published partial registry")
	}
}

func TestRONControlledPermanentCounterIntrinsicComposition(t *testing.T) {
	for _, filter := range []string{
		`TargetsAPermanent(ControlledByAPlayer(SinglePlayer(You)))`,
		`And([Or([IsCardtype(Instant),IsEnchantmentType(Aura)]),TargetsAPermanent(ControlledByAPlayer(SinglePlayer(You)))])`,
		`And([TargetsAPermanent(ControlledByAPlayer(SinglePlayer(You))),And([IsColor(Blue),IsCardtype(Instant)])])`,
	} {
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetSpell(`+filter+`)],ActionList([CounterSpell(Ref_TargetSpell)])))`)
		card := factory()
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		p := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), a.PlayerID())
		victim := castRONSpell(t, g, b, repositoryCard(t, "LEB", "Jump"), p.ID())
		castRONSpell(t, g, a, card, victim.ID)
		g.ResolveTopOfStack()
		if g.Stack().FindByID(victim.ID) != nil {
			t.Fatal("composition did not counter")
		}
	}
}

func TestRONControlledPermanentCounterRetargetCopy(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), a.PlayerID())
	first := castRONSpell(t, g, b, repositoryCard(t, "LEB", "Jump"), p.ID())
	second := castRONSpell(t, g, b, repositoryCard(t, "LEB", "Jump"), p.ID())
	counter := castRONSpell(t, g, a, repositoryCard(t, "LEG", "Avoid Fate"), first.ID)
	clone := g.Clone()
	copied := g.CopyStackObjectDirect(counter, a.PlayerID(), false)
	copied.Targets[0] = second.ID
	copied.TargetZones[second.ID] = core.ZoneStack
	g.ResolveTopOfStack()
	if g.Stack().FindByID(second.ID) != nil || g.Stack().FindByID(first.ID) == nil || counter.Targets[0] != first.ID {
		t.Fatal("copy target identity")
	}
	g.ResolveTopOfStack()
	if g.Stack().FindByID(first.ID) != nil {
		t.Fatal("original target identity")
	}
	clone.ResolveTopOfStack()
	if clone.Stack().FindByID(first.ID) != nil || clone.Stack().FindByID(second.ID) == nil {
		t.Fatal("clone target identity")
	}
}

func TestRONControlledPermanentCounterSharedContexts(t *testing.T) {
	body := `Targeted([TargetSpell(And([IsCardtype(Instant),TargetsAPermanent(ControlledByAPlayer(SinglePlayer(You)))]))],ActionList([CounterSpell(Ref_TargetSpell)]))`
	for _, rule := range []string{
		`SpellActions(` + body + `)`,
		`SpellActions(Modal_ChooseOne([` + body + `,ActionList([PlayerAction(You,GainLife(Integer(1)))])]))`,
		`Activated(PayMana([]),` + body + `)`,
		`TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),` + body + `)`,
	} {
		typ := "Artifact"
		if strings.HasPrefix(rule, "SpellActions") {
			typ = "Instant"
		}
		factory := permanentScopeFactory(t, typ, rule)
		card := factory()
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}
