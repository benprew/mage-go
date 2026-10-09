package ron

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPermanentInformationSpells(t *testing.T) {
	for _, name := range []string{"Swords to Plowshares", "Crumble"} {
		for _, scenario := range []string{"normal", "control", "power", "negative", "indestructible", "regeneration", "absent", "returned", "phase", "phase-cycle", "shroud"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				g, a, b := ronGame()
				set, color := "LEA", core.White
				if name == "Crumble" {
					set, color = "ATQ", core.Green
				}
				card := repositoryCard(t, set, name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(color, 1)
				victim := mage.NewCreature("Victim", "{3}{G}", 3, 8, mage.WithCardType(core.TypeArtifact))
				victim.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(victim, b.PlayerID())
				p.AddCounter(core.P1P1, 2)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().TotalMana() != 0 || a.Life() != 20 || b.Life() != 20 || g.FindPermanent(p.ID()) == nil {
					t.Fatal("wrong announcement timing")
				}
				want := 5
				if name == "Crumble" {
					want = 4
				}
				receiver := b.PlayerID()
				switch scenario {
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
					receiver = a.PlayerID()
				case "power":
					p.AddCounter(core.P1P1, 3)
					if name != "Crumble" {
						want = 8
					}
				case "negative":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(-9), mage.Fixed(0))), p.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					if name != "Crumble" {
						want = 0
					}
				case "indestructible":
					p.GrantBaseAttr(core.Indestructible)
				case "regeneration":
					g.AddRegenerationShield(p.ID())
				case "absent", "returned":
					g.BouncePermanentToHand(p)
					if scenario == "returned" {
						a.RemoveFromHand(victim.ID())
						g.PutOnBattlefield(victim, b.PlayerID())
					}
					want = 0
				case "phase":
					g.PhaseOut(p.ID())
					want = 0
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "shroud":
					p.GrantBaseAttr(core.Shroud)
					want = 0
				}
				clone := g.Clone()
				clone.ResolveStack()
				if a.Life() != 20 || b.Life() != 20 {
					t.Fatal("clone changed original")
				}
				g.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					if game.GetPlayer(receiver).Life() != 20+want {
						t.Fatalf("life=%d want=%d", game.GetPlayer(receiver).Life(), 20+want)
					}
					other := a.PlayerID()
					if receiver == other {
						other = b.PlayerID()
					}
					if game.GetPlayer(other).Life() != 20 {
						t.Fatal("wrong player gained life")
					}
					removed := scenario != "absent" && scenario != "returned" && scenario != "phase" && scenario != "shroud" && (name != "Crumble" || scenario != "indestructible")
					if removed && game.FindPermanentIncludingPhased(p.ID()) != nil {
						t.Fatal("target not removed")
					}
					if removed && name == "Swords to Plowshares" && game.FindExiledCard(victim.ID()) == nil {
						t.Fatal("target not exiled")
					}
					if removed && name == "Crumble" && !slices.ContainsFunc(game.GetPlayer(a.PlayerID()).Graveyard(), func(c mage.Card) bool { return c.ID() == victim.ID() }) {
						t.Fatal("destruction did not use the owner's graveyard")
					}
				}
			})
		}
	}
}

func TestRONPermanentInformationVenom(t *testing.T) {
	for _, scenario := range []string{"normal", "control", "departed", "returned", "second-departure", "phase", "phase-cycle", "reattach", "source-left", "copy", "twice"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEA", "Psychic Venom")
			card.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			hostCard := mage.NewLand("Host")
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			aura := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			g.TapPermanent(host)
			receiver := b.PlayerID()
			damage := 2
			switch scenario {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				receiver = a.PlayerID()
			case "departed", "returned", "second-departure":
				g.BouncePermanentToHand(host)
				if scenario != "departed" {
					b.RemoveFromHand(hostCard.ID())
					q := g.PutOnBattlefield(hostCard, a.PlayerID())
					if scenario == "second-departure" {
						g.BouncePermanentToHand(q)
					}
				}
			case "phase":
				g.PhaseOut(host.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "reattach":
				g.Attach(aura.ID(), g.PutOnBattlefield(mage.NewLand("Other"), a.PlayerID()).ID())
			case "source-left":
				g.BouncePermanentToHand(aura)
			case "twice":
				g.UntapPermanent(host)
				g.TapPermanent(host)
				damage = 4
			}
			g.PutTriggersOnStack()
			if scenario == "copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				damage = 4
			}
			clone := g.Clone()
			clone.ResolveStack()
			g.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if game.GetPlayer(receiver).Life() != 20-damage {
					t.Fatalf("life=%d want=%d", game.GetPlayer(receiver).Life(), 20-damage)
				}
			}
		})
	}
}

func TestRONPermanentInformationInvalid(t *testing.T) {
	for _, action := range []string{
		`PlayerAction(ControllerOfPermanent(Ref_TargetPlayer), GainLife(Integer(1)))`,
		`PlayerAction(ControllerOfPermanent(Ref_TargetPermanent()), GainLife(Integer(1)))`,
		`PlayerAction(ControllerOfPermanent(), GainLife(Integer(1)))`,
		`PlayerAction(ControllerOfPermanent(Ref_TargetPermanent,ThisPermanent), GainLife(Integer(1)))`,
		`PlayerAction(ControllerOfPermanent(Trigger_ThatPermanent), GainLife(Integer(1)))`,
		`PlayerAction(You, GainLife(PowerOfPermanent(Ref_TargetPlayer)))`,
		`PlayerAction(You, GainLife(PowerOfPermanent()))`,
		`Exile([Permanent(Ref_TargetPermanent)], [FaceDown])`,
		`Exile([], [])`, `Exile([Permanent(Ref_TargetPermanent),CardInGraveyards(Ref_TargetPermanent)], [])`,
		`Exile([CardInGraveyards(Ref_TargetPermanent)], [])`,
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([%s])))])`, action)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", action)
		}
	}
}

func TestRONPermanentInformationCopiesAndRollback(t *testing.T) {
	for _, name := range []string{"Swords to Plowshares", "Crumble"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronModalGame(0)
			set, color := "LEA", core.White
			if name == "Crumble" {
				set, color = "ATQ", core.Green
			}
			first := g.PutOnBattlefield(mage.NewCreature("First", "{2}{G}", 3, 8, mage.WithCardType(core.TypeArtifact)), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "{5}{G}", 7, 8, mage.WithCardType(core.TypeArtifact)), a.PlayerID())
			card := repositoryCard(t, set, name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID()}, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			a.ManaPool().Add(color, 1)
			wrong := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{wrong.ID()}, 0); err == nil {
				t.Fatal("wrong target accepted")
			}
			if a.ManaPool().TotalMana() != 1 || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed announcement spent resources")
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			original := g.StackPeek()
			a.targets = []uuid.UUID{second.ID()}
			copied := g.CopyStackObjectDirect(original, a.PlayerID(), true)
			if copied == nil || copied.Targets[0] != second.ID() || original.Targets[0] != first.ID() {
				t.Fatal("copy target isolation")
			}
			g.ResolveTopOfStack()
			wantA, wantB := 27, 23
			if name == "Crumble" {
				wantA, wantB = 26, 23
			}
			if a.Life() != wantA || b.Life() != 20 {
				t.Fatal("copy used original information")
			}
			g.ResolveTopOfStack()
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatal("original lost information")
			}
			if repositoryCard(t, set, name).Abilities()[0].(*mage.ActionDefinition).Targets()[0] == card.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
				t.Fatal("factory shared target declaration")
			}
		})
	}
}

func TestRONPermanentInformationContexts(t *testing.T) {
	body := `Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([Exile([Permanent(Ref_TargetPermanent)],[]),PlayerAction(ControllerOfPermanent(Ref_TargetPermanent),GainLife(PowerOfPermanent(Ref_TargetPermanent)))]))`
	for _, context := range []string{"activation", "trigger", "modal"} {
		t.Run(context, func(t *testing.T) {
			g, a, b := ronGame()
			typ, rule := "Artifact", `Activated(PayMana([]),`+body+`)`
			if context == "trigger" {
				rule = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),` + body + `)`
			}
			if context == "modal" {
				typ = "Instant"
				rule = `SpellActions(Modal_ChooseOne([` + body + `,ActionList([GainLife(Integer(1))])]))`
			}
			factory := permanentScopeFactory(t, typ, rule)
			card := factory()
			card.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			p := g.PutOnBattlefield(mage.NewCreature("Victim", "", 4, 8), b.PlayerID())
			if context == "modal" {
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
					t.Fatal(err)
				}
			} else {
				source := g.PutOnBattlefield(card, a.PlayerID())
				if context == "activation" {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
						t.Fatal(err)
					}
				} else {
					g.PutTriggersOnStack()
				}
			}
			g.ResolveStack()
			if b.Life() != 24 || a.Life() != 20 || g.FindPermanent(p.ID()) != nil {
				t.Fatal("context lost target information")
			}
		})
	}
	for _, rule := range []string{
		`SpellActions(Modal_ChooseOne([Targeted([TargetPermanent(AnyPermanent)],ActionList([])),ActionList([GainLife(PowerOfPermanent(Ref_TargetPermanent))])]))`,
		`Activated(PayMana([]),Targeted([TargetPermanent(AnyPermanent)],ActionList([]))),Activated(PayMana([]),ActionList([PlayerAction(ControllerOfPermanent(Ref_TargetPermanent),GainLife(Integer(1)))]))`,
	} {
		assertGraveyardInvalid(t, rule)
	}
}

func TestRONPermanentInformationReprintsAndInventory(t *testing.T) {
	names := []string{"Swords to Plowshares", "Crumble", "Psychic Venom", "Divine Offering"}
	paths, err := os.ReadDir("../../../../cards/mtgish")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	records := 0
	for _, path := range paths {
		if !strings.HasSuffix(path.Name(), ".ron") {
			continue
		}
		data, err := os.ReadFile("../../../../cards/mtgish/" + path.Name())
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), path.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			if !slices.Contains(names, node.Fields["Name"].String) {
				continue
			}
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			files[fmt.Sprintf("%s-%d.ron", path.Name(), records)] = &fstest.MapFile{Data: []byte(strings.Split(string(data), "\n")[node.Position.Line-1])}
			records++
		}
	}
	if records != 15 {
		t.Fatalf("records=%d", records)
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 4 {
		t.Fatal("reprints did not merge")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 15 {
		t.Fatal("reprint inventory rejected record")
	}
	for name, role := range map[string]string{"PowerOfPermanent": "value", "ControllerOfPermanent": "player_reference", "Permanent": "exile_subject", "Ref_TargetPermanent": "permanent_reference"} {
		if !slices.ContainsFunc(report.Signatures, func(s InventorySignature) bool { return s.Name == name && s.Role == role }) {
			t.Errorf("missing %s role %s", name, role)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Unsupported",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Unknown])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 4 {
		t.Fatal("incomplete collection published")
	}
}

func TestRONPermanentInformationDivineOffering(t *testing.T) {
	for _, scenario := range []string{"destroy", "regenerate", "indestructible", "illegal", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Divine Offering")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			a.ManaPool().Add(core.White, 2)
			victim := mage.NewArtifact("Victim", "{5}{U}")
			victim.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(victim, b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			want, removed := 26, true
			switch scenario {
			case "regenerate":
				g.AddRegenerationShield(p.ID())
				removed = false
			case "indestructible":
				p.GrantBaseAttr(core.Indestructible)
				removed = false
			case "illegal":
				g.PhaseOut(p.ID())
				want, removed = 20, false
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				want = 20
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				wantB := 20
				if scenario == "copy" {
					wantB = 26
				}
				if game.GetPlayer(a.PlayerID()).Life() != want || game.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatal("offering used wrong mana value or player")
				}
				if (game.FindPermanentIncludingPhased(p.ID()) == nil) != removed {
					t.Fatal("wrong destruction result")
				}
				if scenario == "regenerate" && (!game.FindPermanent(p.ID()).Tapped || game.HasRegenerationShield(p.ID())) {
					t.Fatal("regeneration not applied")
				}
			}
		})
	}
}

type informationLifeReplacement struct {
	source uuid.UUID
	double bool
}

func (r *informationLifeReplacement) SourceID() uuid.UUID           { return r.source }
func (*informationLifeReplacement) GetDuration() core.Duration      { return core.Indefinite }
func (*informationLifeReplacement) IsActive(mage.GameReader) bool   { return true }
func (r *informationLifeReplacement) Clone() mage.ReplacementEffect { c := *r; return &c }
func (*informationLifeReplacement) Matches(action mage.Action, _ mage.GameReader) bool {
	_, ok := action.(*mage.LifeGainAction)
	return ok
}
func (r *informationLifeReplacement) Replace(action mage.Action, _ *mage.Game) mage.Action {
	gain := action.(*mage.LifeGainAction)
	amount := gain.Amount() + 1
	if r.double {
		amount = gain.Amount() * 2
	}
	return mage.NewLifeGainAction(gain.ActionSource(), gain.PlayerID(), amount)
}

func TestRONPermanentInformationReplacementOrder(t *testing.T) {
	for _, index := range []int{0, 1} {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			a := mage.NewBasePlayer("A")
			b := &counterReplacementChooser{BasePlayer: mage.NewBasePlayer("B"), index: index}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, "ATQ", "Crumble")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Green, 1)
			victim := mage.NewArtifact("Victim", "{3}{G}")
			victim.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(victim, b.PlayerID())
			first := g.PutOnBattlefield(mage.NewArtifact("Double", ""), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewArtifact("Add", ""), a.PlayerID())
			g.AddReplacementEffect(&informationLifeReplacement{source: first.ID(), double: true})
			g.AddReplacementEffect(&informationLifeReplacement{source: second.ID()})
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if b.Life() != 29+index || a.Life() != 20 || b.choices != 1 || g.FindPermanent(p.ID()) != nil {
				t.Fatalf("life=%d choices=%d", b.Life(), b.choices)
			}
		})
	}
}

func TestRONPermanentInformationVenomCasting(t *testing.T) {
	for _, scenario := range []string{"land", "artifact", "shroud", "invalidated"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEA", "Psychic Venom")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 2)
			host := mage.NewLand("Host")
			if scenario == "artifact" {
				host = mage.NewArtifact("Host", "")
			}
			host.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(host, b.PlayerID())
			if scenario == "shroud" {
				p.GrantBaseAttr(core.Shroud)
			}
			err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0)
			if scenario == "artifact" || scenario == "shroud" {
				if err == nil || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 2 || g.StackSize() != 0 {
					t.Fatal("illegal aura cast did not roll back")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "invalidated" {
				g.PhaseOut(p.ID())
			}
			g.ResolveStack()
			if scenario == "invalidated" {
				if g.FindPermanent(card.ID()) != nil {
					t.Fatal("aura entered with illegal target")
				}
				return
			}
			if g.FindPermanent(card.ID()).AttachedTo != p.ID() {
				t.Fatal("aura did not attach to target land")
			}
			g.TapPermanent(g.FindPermanent(p.ID()))
			g.PutTriggersOnStack()
			g.ResolveStack()
			if b.Life() != 18 || a.Life() != 20 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("cast aura did not execute damage")
			}
		})
	}
}
