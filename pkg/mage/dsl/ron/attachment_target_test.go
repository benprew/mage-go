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

func attachmentTargetFixture(t *testing.T, name string) (*mage.Game, *mage.BasePlayer, *mage.BasePlayer, *mage.Permanent, *mage.Permanent, *mage.Permanent) {
	t.Helper()
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "DRK", name), a.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	hostCard := mage.NewCreature("Host", "", 2, 6)
	auraName := "Holy Strength"
	if name == "Savaen Elves" {
		hostCard = mage.NewLand("Host", mage.WithSubTypes("Forest"))
		auraName = "Wild Growth"
	}
	host := g.PutOnBattlefield(hostCard, a.PlayerID())
	auraCard := repositoryCard(t, "LEB", auraName)
	auraCard.SetOwner(b.PlayerID())
	aura := g.PutOnBattlefield(auraCard, b.PlayerID())
	g.Attach(aura.ID(), host.ID())
	return g, a, b, source, host, aura
}

func TestRONAttachmentTargetRealCards(t *testing.T) {
	for _, name := range []string{"Miracle Worker", "Savaen Elves"} {
		for _, change := range []string{"none", "aura-control", "host-control", "host-shroud", "aura-shroud", "reattach-own", "reattach-opponent", "aura-return", "host-return", "phase", "phase-cycle", "source-leaves", "source-control", "copy", "regenerate", "indestructible"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b, source, host, aura := attachmentTargetFixture(t, name)
				a.ManaPool().Add(core.Green, 2)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{aura.ID()}); err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(source.ID()).Tapped || g.FindPermanent(aura.ID()) == nil {
					t.Fatal("payment or resolution timing is wrong")
				}
				wantMana := 2
				if name == "Savaen Elves" {
					wantMana = 0
				}
				if a.ManaPool().TotalMana() != wantMana {
					t.Fatal("wrong activation mana payment")
				}
				survives := false
				switch change {
				case "aura-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: aura.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "host-shroud":
					g.MutablePermanent(host.ID()).GrantBaseAttr(core.Shroud)
				case "aura-shroud":
					g.MutablePermanent(aura.ID()).GrantBaseAttr(core.Shroud)
					survives = true
				case "host-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					survives = name == "Miracle Worker"
				case "reattach-own", "reattach-opponent":
					controller := a.PlayerID()
					if change == "reattach-opponent" {
						controller = b.PlayerID()
					}
					otherCard := mage.NewCreature("Other", "", 2, 6)
					if name == "Savaen Elves" {
						otherCard = mage.NewLand("Other", mage.WithSubTypes("Forest"))
					}
					other := g.PutOnBattlefield(otherCard, controller)
					g.Attach(aura.ID(), other.ID())
					survives = change == "reattach-opponent" && name == "Miracle Worker"
				case "aura-return":
					card := aura.Card
					g.RemoveFromBattlefield(aura)
					aura = g.PutOnBattlefield(card, b.PlayerID())
					g.Attach(aura.ID(), host.ID())
					survives = true
				case "host-return":
					card := host.Card
					g.RemoveFromBattlefield(host)
					host = g.PutOnBattlefield(card, a.PlayerID())
					g.Attach(aura.ID(), host.ID())
				case "phase", "phase-cycle":
					phased := g.PhaseOut(host.ID())
					if change == "phase-cycle" {
						g.PhaseIn(phased)
					}
					survives = change == "phase"
				case "source-leaves":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: aura.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					g.AddRegenerationShield(aura.ID())
				case "regenerate":
					g.AddRegenerationShield(aura.ID())
					survives = true
				case "indestructible":
					g.MutablePermanent(aura.ID()).GrantBaseAttr(core.Indestructible)
					survives = true
				}
				clone := g.Clone()
				clone.ResolveStack()
				if change != "phase" && g.FindPermanent(aura.ID()) == nil {
					t.Fatal("clone changed original")
				}
				g.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					if change == "phase" {
						game.PhaseIn([]uuid.UUID{host.ID(), aura.ID()})
					}
					if (game.FindPermanent(aura.ID()) != nil) != survives {
						t.Fatalf("wrong Aura survival for %s", change)
					}
					if !survives && len(game.GetPlayer(b.PlayerID()).Graveyard()) != 1 {
						t.Fatal("Aura did not go to its owner's graveyard")
					}
				}
			})
		}
	}
}

func TestRONAttachmentTargetRollback(t *testing.T) {
	for _, name := range []string{"Miracle Worker", "Savaen Elves"} {
		for _, invalid := range []string{"sick", "unattached", "wrong-host", "host-control", "shroud", "phased", "no-mana", "not-aura"} {
			if name == "Miracle Worker" && invalid == "no-mana" || name == "Savaen Elves" && invalid == "host-control" {
				continue
			}
			t.Run(name+"/"+invalid, func(t *testing.T) {
				g, a, b, source, host, aura := attachmentTargetFixture(t, name)
				if invalid != "no-mana" {
					a.ManaPool().Add(core.Green, 2)
				}
				switch invalid {
				case "sick":
					source.GrantBaseAttr(core.AttrSummonSick)
				case "unattached":
					aura.AttachedTo = uuid.Nil
				case "wrong-host":
					otherCard := mage.NewLand("Land", mage.WithSubTypes("Forest"))
					if name == "Savaen Elves" {
						otherCard = mage.NewCreature("Creature", "", 2, 6)
					}
					aura.AttachedTo = g.PutOnBattlefield(otherCard, a.PlayerID()).ID()
				case "host-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "shroud":
					aura.GrantBaseAttr(core.Shroud)
				case "phased":
					g.PhaseOut(aura.ID())
				case "not-aura":
					aura = g.PutOnBattlefield(mage.NewArtifact("Equipment", "", mage.WithSubTypes("Equipment")), a.PlayerID())
					aura.AttachedTo = host.ID()
				case "no-mana":
					g.MutablePermanent(host.ID()).Tapped = true
				}
				before := a.ManaPool().TotalMana()
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{aura.ID()}); err == nil {
					t.Fatal("illegal activation accepted")
				}
				if g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 || a.ManaPool().TotalMana() != before {
					t.Fatal("failed activation spent live resources")
				}
			})
		}
	}
}

func TestRONAttachmentTargetInvalid(t *testing.T) {
	for _, filter := range []string{
		`AttachedToAPermanent`, `AttachedToAPermanent()`, `AttachedToAPermanent(IsCardtype(Land),IsCardtype(Creature))`,
		`AttachedToAPermanent(ControlledByAPlayer(Opponent))`,
		`AttachedToAPermanent(ControlledByAPlayer(SinglePlayer(Ref_TargetPlayer)))`,
		`AttachedToAPermanent(ControlledByAPlayer(SinglePlayer(You())))`,
		`AttachedToAPermanent(Other(ThisPermanent))`,
		`AttachedToAPermanent(And([]))`,
		`AttachedToAPermanent(Or([IsCardtype(Land),ControlledByAPlayer(SinglePlayer(You))]))`,
		`AttachedToAPermanent(And([ControlledByAPlayer(SinglePlayer(You)),ControlledByAPlayer(SinglePlayer(You))]))`,
		`AttachedToAPermanent(AttachedToAPermanent(IsCardtype(Land)))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(`+filter+`)],ActionList([DestroyPermanent(Ref_TargetPermanent)])))])`), "attachment-invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid attachment filter %s", filter)
		}
	}
}

func TestRONAttachmentTargetCasting(t *testing.T) {
	for _, name := range []string{"Miracle Worker", "Savaen Elves"} {
		g, a, _ := ronGame()
		card := repositoryCard(t, "DRK", name)
		if err := mage.ValidateCardDefinition(name, card); err != nil {
			t.Fatal(err)
		}
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
			t.Fatal("unpaid casting did not roll back")
		}
		color := core.White
		if name == "Savaen Elves" {
			color = core.Green
		}
		a.ManaPool().Add(color, 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		p := g.FindPermanent(card.ID())
		if p == nil || p.CurrentPower(g) != 1 || p.CurrentToughness(g) != 1 || a.ManaPool().TotalMana() != 0 {
			t.Fatal("wrong casting cost or characteristics")
		}
	}
}

func TestRONAttachmentTargetInstancesAndCopyController(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range nodes {
		if node.Fields["Name"].String != "Miracle Worker" {
			continue
		}
		factory, err := Compile(node)
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		first, second := factory(), factory()
		one := first.Abilities()[0].(*mage.ActionDefinition)
		two := second.Abilities()[0].(*mage.ActionDefinition)
		if one == two || one.Targets()[0] == two.Targets()[0] {
			t.Fatal("factory shared an ability or target declaration")
		}
		p := g.PutOnBattlefield(first, a.PlayerID())
		q := g.PutOnBattlefield(second, a.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		q.RevokeBaseAttr(core.AttrSummonSick)
		host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 6), a.PlayerID())
		u := attachRONTestAura(t, g, host, b.PlayerID())
		v := attachRONTestAura(t, g, host, b.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{u.ID()}); err != nil {
			t.Fatal(err)
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), q.ID(), 0, []uuid.UUID{v.ID()}); err != nil {
			t.Fatal(err)
		}
		if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
		g.ResolveTopOfStack()
		if g.FindPermanent(v.ID()) == nil {
			t.Fatal("copy used the original controller for host legality")
		}
		g.AddRegenerationShield(u.ID())
		g.ResolveStack()
		if g.FindPermanent(u.ID()) == nil || g.FindPermanent(v.ID()) != nil {
			t.Fatal("factory instance target bindings crossed")
		}
		return
	}
	t.Fatal("real RON record is missing")
}

func TestRONAttachmentTargetShapesAndInventory(t *testing.T) {
	for _, host := range []string{
		`IsCardtype(Land)`, `Or([IsCardtype(Creature),IsCardtype(Land)])`,
		`And([Or([IsCardtype(Creature),IsCardtype(Land)]),ControlledByAPlayer(SinglePlayer(You))])`,
		`And([And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]),AnyPermanent])`,
		`ControlledByAPlayer(SinglePlayer(You))`,
	} {
		g, a, b := ronGame()
		card := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(AttachedToAPermanent(`+host+`))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`)()
		land := g.PutOnBattlefield(mage.NewLand("Land"), a.PlayerID())
		if strings.Contains(host, "IsCardtype(Creature)") && !strings.Contains(host, "IsCardtype(Land)") {
			land = g.PutOnBattlefield(mage.NewCreature("Creature", "", 2, 6), a.PlayerID())
		}
		aura := g.PutOnBattlefield(mage.NewAura("Attachment", "", mage.WithCastTarget(mage.TargetPermanent())), b.PlayerID())
		g.Attach(aura.ID(), land.ID())
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{aura.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if g.FindPermanent(aura.ID()) != nil {
			t.Fatalf("host filter did not resolve: %s", host)
		}
	}
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AttachedToAPermanent(And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))])))],ActionList([UnsupportedAction])))])`
	report, err := InspectFS(fstest.MapFS{"attachment.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil || report.Compiled != 0 || len(report.Issues) != 1 {
		t.Fatalf("inventory: %v", err)
	}
	roles := map[string]string{"AttachedToAPermanent": "permanent_filter", "And": "permanent_filter", "ControlledByAPlayer": "permanent_filter", "SinglePlayer": "player_filter", "You": "player_reference"}
	for _, signature := range report.Signatures {
		if want, ok := roles[signature.Name]; ok {
			if signature.Role != want || signature.Support != "unverified" {
				t.Fatalf("wrong nested signature: %+v", signature)
			}
			delete(roles, signature.Name)
		}
	}
	if len(roles) != 0 {
		t.Fatalf("missing nested roles: %v", roles)
	}
}

func TestRONAttachmentTargetTriggerPlacement(t *testing.T) {
	g, a, b := ronGame()
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 6), a.PlayerID())
	aura := attachRONTestAura(t, g, host, b.PlayerID())
	card := permanentScopeFactory(t, "Enchantment", `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),Targeted([TargetPermanent(And([IsEnchantmentType(Aura),AttachedToAPermanent(And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(You))]))]))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`)()
	g.PutOnBattlefield(card, a.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || g.StackPeek().Targets[0] != aura.ID() || g.FindPermanent(aura.ID()) == nil {
		t.Fatal("trigger target was not chosen at placement")
	}
	g.ResolveStack()
	if g.FindPermanent(aura.ID()) != nil {
		t.Fatal("targeted trigger did not resolve")
	}
}
