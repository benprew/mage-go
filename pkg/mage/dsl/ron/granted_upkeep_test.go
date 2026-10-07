package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONGrantedUpkeepCards(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		cost      int
		color     core.Color
	}{
		{"ATQ", "Energy Flux", 2, core.Blue},
		{"LEG", "The Tabernacle at Pendrell Vale", 1, core.Blue},
		{"LEA", "Farmstead", 2, core.White},
	} {
		for _, choice := range []string{"paid", "declined", "short", "wrong color", "donor leaves", "recipient returns", "returned unpaid", "control", "copy", "clone", "regenerate", "indestructible"} {
			t.Run(tc.name+"/"+choice, func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: choice != "declined" && choice != "regenerate" && choice != "indestructible" && choice != "returned unpaid"}
				b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
				g := mage.NewGame(a, b)
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(b.PlayerID())
				var subjectCard mage.Card
				switch tc.name {
				case "Energy Flux":
					subjectCard = mage.NewArtifact("Subject", "")
				case "Farmstead":
					subjectCard = mage.NewLand("Subject")
				default:
					subjectCard = mage.NewCreature("Subject", "", 2, 4)
				}
				subjectCard.SetOwner(a.PlayerID())
				subject := g.PutOnBattlefield(subjectCard, a.PlayerID())
				donor := g.PutOnBattlefield(card, b.PlayerID())
				if tc.name == "Farmstead" {
					g.Attach(donor.ID(), subject.ID())
				}
				queueRONUpkeep(g, b)
				if g.StackSize() != 0 {
					t.Fatal("used donor controller")
				}
				queueRONUpkeep(g, a)
				if g.StackSize() != 1 || g.StackPeek().SourceID != subject.ID() || a.offers != 0 {
					t.Fatal("recipient source or payment timing")
				}
				amount := tc.cost
				if choice == "short" {
					amount--
				}
				color := tc.color
				if choice == "wrong color" {
					color = core.Red
				}
				a.ManaPool().Add(color, amount)
				switch choice {
				case "donor leaves":
					g.DestroyPermanent(donor)
				case "recipient returns", "returned unpaid":
					g.BouncePermanentToHand(subject)
					a.RemoveFromHand(subjectCard.ID())
					subject = g.PutOnBattlefield(subjectCard, a.PlayerID())
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: donor.ID(), TargetID: subject.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy")
					}
				case "regenerate":
					g.AddRegenerationShield(subject.ID())
				case "indestructible":
					g.MutablePermanent(subject.ID()).GrantBaseAttr(core.Indestructible)
				}
				clone := g.Clone()
				if choice == "clone" {
					clone.ResolveStack()
					if g.StackSize() != 1 || a.ManaPool().TotalMana() != amount || a.Life() != 20 {
						t.Fatal("clone changed original")
					}
				}
				g.ResolveStack()
				paid := choice != "short" && choice != "declined" && choice != "regenerate" && choice != "indestructible" && choice != "returned unpaid" && (choice != "wrong color" || tc.name != "Farmstead")
				if tc.name == "Farmstead" {
					want := 20
					if paid {
						want++
					}
					if a.Life() != want || b.Life() != 20 || g.FindPermanent(subject.ID()) == nil {
						t.Fatal("life or payer")
					}
				} else {
					present := paid || choice == "recipient returns" || choice == "returned unpaid" || choice == "control"
					if choice == "copy" && tc.name != "Energy Flux" {
						present = false
					}
					if tc.name == "The Tabernacle at Pendrell Vale" && (choice == "regenerate" || choice == "indestructible") {
						present = true
					}
					if (g.FindPermanent(subject.ID()) != nil) != present {
						t.Fatal("wrong payment branch or incarnation")
					}
					if tc.name == "The Tabernacle at Pendrell Vale" && choice == "regenerate" && !g.FindPermanent(subject.ID()).Tapped {
						t.Fatal("regeneration did not replace destruction")
					}
				}
				wantMana := amount
				if paid {
					wantMana -= tc.cost
				}
				if a.ManaPool().TotalMana() != wantMana {
					t.Fatal("failed payment spent mana")
				}
			})
		}
	}
}

func TestRONGrantedUpkeepInvalid(t *testing.T) {
	for _, rule := range []string{
		`PermanentLayerEffect(ThisPermanent,[AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([GainLife(Integer(1))]))])])`,
		`PermanentLayerEffect(HostPermanent,[AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(HostController)),ActionList([GainLife(Integer(1))]))])])`,
		`PermanentLayerEffect(HostPermanent,[AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([TapPermanent(HostPermanent)]))])])`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),ActionList([GainLife(Integer(1))]))])])`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([TapPermanent(Ref_TargetPermanent)]))])])`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([If(CostWasPaid,[GainLife(Integer(1))])]))])])`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)))])])`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[AddAbility([TriggerA])])`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[AddAbility([TriggerA(WhenAPermanentDies(SinglePermanent(ThisPermanent)),ActionList([GainLife(Integer(1))]))])])`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[AdjustPT(1,1),AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([GainLife(Integer(1))]))])])`,
		`EachPermanentLayerEffect(IsCardtype(Creature),[AddAbility([Flying,TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([GainLife(Integer(1))]))])])`,
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),%s])`, rule)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONGrantedUpkeepLifecycle(t *testing.T) {
	for _, name := range []string{"Farmstead", "Energy Flux", "The Tabernacle at Pendrell Vale"} {
		for _, change := range []string{"source phase", "source phase cycle", "source return", "recipient phase", "recipient phase cycle", "source removal", "recipient removal", "source control", "recipient control", "attachment", "pending phase", "two donors", "later entry"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: false}
				b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
				g := mage.NewGame(a, b)
				set := map[string]string{"Farmstead": "LEA", "Energy Flux": "ATQ", "The Tabernacle at Pendrell Vale": "LEG"}[name]
				c := repositoryCard(t, set, name)
				c.SetOwner(a.PlayerID())
				victim := func() mage.Card {
					switch name {
					case "Farmstead":
						return mage.NewLand("Recipient")
					case "Energy Flux":
						return mage.NewArtifact("Recipient", "")
					default:
						return mage.NewCreature("Recipient", "", 2, 4)
					}
				}
				p := g.PutOnBattlefield(victim(), a.PlayerID())
				q := g.PutOnBattlefield(victim(), b.PlayerID())
				donor := g.PutOnBattlefield(c, a.PlayerID())
				if name == "Farmstead" {
					g.Attach(donor.ID(), p.ID())
				}
				want := 1
				eventPlayer := mage.Player(a)
				switch change {
				case "source phase":
					g.PhaseOut(donor.ID())
					want = 0
				case "source phase cycle":
					g.PhaseIn(g.PhaseOut(donor.ID()))
				case "source return":
					g.BouncePermanentToHand(donor)
					a.RemoveFromHand(c.ID())
					donor = g.PutOnBattlefield(c, a.PlayerID())
					if name == "Farmstead" {
						g.Attach(donor.ID(), q.ID())
						want = 0
					}
				case "recipient phase":
					g.PhaseOut(p.ID())
					want = 0
				case "recipient phase cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "source removal":
					g.AddContinuousEffect(mage.RemoveAllAbilities(donor.ID(), core.Indefinite))
					want = 0
				case "recipient removal":
					g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
					want = 0
				case "source control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: q.ID(), TargetID: donor.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "recipient control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: donor.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					eventPlayer = b
					if name != "Farmstead" {
						want = 2
					}
				case "attachment":
					if name == "Farmstead" {
						g.Attach(donor.ID(), q.ID())
						want = 0
					}
				case "two donors":
					second := repositoryCard(t, set, name)
					second.SetOwner(a.PlayerID())
					r := g.PutOnBattlefield(second, a.PlayerID())
					if name == "Farmstead" {
						g.Attach(r.ID(), p.ID())
					}
					want = 2
				case "later entry":
					g.PutOnBattlefield(victim(), a.PlayerID())
					if name != "Farmstead" {
						want = 2
					}
				}
				g.ApplyContinuousEffects()
				queueRONUpkeep(g, eventPlayer)
				if g.StackSize() != want || a.offers != 0 || b.offers != 0 {
					t.Fatalf("stack=%d want %d", g.StackSize(), want)
				}
				if change == "pending phase" {
					g.PhaseOut(p.ID())
				}
				g.ResolveStack()
				if name != "Farmstead" && want > 0 && change != "pending phase" && g.FindPermanent(p.ID()) != nil {
					t.Fatal("unpaid branch missed recipient")
				}
				if change != "recipient control" && g.FindPermanent(q.ID()) == nil {
					t.Fatal("opponent recipient changed")
				}
			})
		}
	}
}

func TestRONGrantedUpkeepCastingAndNativeStep(t *testing.T) {
	for _, name := range []string{"Energy Flux", "Farmstead", "The Tabernacle at Pendrell Vale"} {
		t.Run(name, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(2)
			set := map[string]string{"Energy Flux": "ATQ", "Farmstead": "LEA", "The Tabernacle at Pendrell Vale": "LEG"}[name]
			c := repositoryCard(t, set, name)
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			var subject mage.Card
			switch name {
			case "Farmstead":
				subject = repositoryCard(t, "LEA", "Plains")
			case "Energy Flux":
				subject = mage.NewArtifact("Recipient", "")
			default:
				subject = mage.NewCreature("Recipient", "", 2, 4)
			}
			subject.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(subject, b.PlayerID())
			var targets []uuid.UUID
			if name == "Farmstead" {
				targets = []uuid.UUID{p.ID()}
			}
			if name == "The Tabernacle at Pendrell Vale" {
				if !c.HasType(core.TypeLand) || !c.HasSuperType(core.SuperLegendary) || c.ManaCost().CMC() != 0 {
					t.Fatal("land metadata")
				}
				if err := g.PlayLand(a.PlayerID(), c.ID()); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), targets, 0); err == nil {
					t.Fatal("unpaid cast accepted")
				}
				if len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("cast rollback")
				}
				a.ManaPool().Add(core.Blue, 1)
				a.ManaPool().Add(core.White, 3)
				if name == "Farmstead" {
					bad := g.PutOnBattlefield(mage.NewCreature("Not a land", "", 2, 4), b.PlayerID())
					if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{bad.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 4 {
						t.Fatal("illegal target or payment rollback")
					}
				}
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), targets, 0); err != nil {
					t.Fatal(err)
				}
				queueRONUpkeep(g, b)
				if g.StackSize() != 1 {
					t.Fatal("grant started before spell resolution")
				}
				g.ResolveStack()
				if a.ManaPool().TotalMana() != 1 {
					t.Fatal("casting cost")
				}
			}
			g.SetActivePlayerIndex(1)
			g.RunStepWithPriority(core.Upkeep)
			if name != "Farmstead" && g.FindPermanent(p.ID()) != nil {
				t.Fatal("native unpaid upkeep branch")
			}
			if name == "Farmstead" && (b.offers != 0 || b.Life() != 20 || g.FindPermanent(c.ID()).AttachedTo != p.ID()) {
				t.Fatal("unaffordable native payment or attachment")
			}
		})
	}
}

func TestRONGrantedUpkeepInventoryAndAtomicLoad(t *testing.T) {
	src := `Card(Name:"Grant",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[EachPermanentLayerEffect(IsCardtype(Creature),[AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))])])])`
	fs := fstest.MapFS{"a.ron": {Data: []byte(src)}}
	r, err := InspectFS(fs, "*.ron")
	if err != nil || r.Compiled != 1 {
		t.Fatalf("inventory: %v %v", err, r)
	}
	for _, occurrence := range r.Signatures {
		if occurrence.Role == "unknown" {
			t.Fatal("unassigned nested role", occurrence)
		}
	}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, fs, "*.ron"); err != nil {
		t.Fatal(err)
	}
	fs["b.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(src, `Name:"Grant"`, `Name:"Invalid"`, 1))}
	fs["b.ron"].Data = []byte(strings.Replace(string(fs["b.ron"].Data), "SinglePlayer(You)", "SinglePlayer(HostController)", 1))
	if err := LoadInto(registry, fs, "*.ron"); err == nil {
		t.Fatal("invalid collection loaded")
	}
	if len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("registry changed")
	}
}

func TestRONGrantedUpkeepSeparatePayments(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
	g := mage.NewGame(a, b)
	for range 2 {
		g.PutOnBattlefield(repositoryCard(t, "ATQ", "Energy Flux"), b.PlayerID())
	}
	c := mage.NewArtifact("Recipient", "")
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	a.ManaPool().Add(core.Blue, 2)
	b.ManaPool().Add(core.Blue, 4)
	queueRONUpkeep(g, a)
	if g.StackSize() != 2 {
		t.Fatal("independent donors did not grant two triggers")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(p.ID()) == nil || a.ManaPool().TotalMana() != 0 || g.StackSize() != 1 {
		t.Fatal("first payment")
	}
	g.ResolveStack()
	if g.FindPermanent(p.ID()) != nil || b.ManaPool().TotalMana() != 4 || a.offers != 1 || b.offers != 0 {
		t.Fatal("second payment used the first result or donor's mana")
	}
}
