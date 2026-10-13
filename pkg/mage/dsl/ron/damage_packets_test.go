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

type ronPacketCard struct {
	set, name                 string
	targeted, self, mana      int
	spell, tap, selfPermanent bool
}

var ronPacketCards = []ronPacketCard{
	{"LEB", "Orcish Artillery", 2, 3, 0, false, true, false},
	{"LEG", "Psionic Entity", 2, 3, 0, false, true, true},
	{"DRK", "Brothers of Fire", 1, 1, 3, false, false, false},
	{"LEB", "Psionic Blast", 4, 2, 3, true, false, false},
}

func TestRONDamagePacketsCards(t *testing.T) {
	for _, tc := range ronPacketCards {
		for _, kind := range []string{"player", "creature", "planeswalker", "overlap"} {
			t.Run(tc.name+"/"+kind, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				var source *mage.Permanent
				if tc.spell {
					a.AddToHand(card)
				} else {
					source = g.PutOnBattlefield(card, a.PlayerID())
					source.RevokeBaseAttr(core.AttrSummonSick)
				}
				target := b.PlayerID()
				if kind == "creature" {
					target = g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 10), b.PlayerID()).ID()
				}
				if kind == "planeswalker" {
					target = g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 10), b.PlayerID()).ID()
				}
				if kind == "overlap" {
					target = a.PlayerID()
					if tc.selfPermanent {
						target = source.ID()
					}
				}
				a.ManaPool().Add(core.Blue, tc.mana)
				if !tc.spell && tc.mana > 0 {
					a.ManaPool().Clear()
					a.ManaPool().Add(core.Red, tc.mana)
				}
				if tc.spell {
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target}); err != nil {
						t.Fatal(err)
					}
				}
				if tc.tap && !g.FindPermanent(source.ID()).Tapped {
					t.Fatal("tap cost was not paid before resolution")
				}
				if a.Life() != 20 || b.Life() != 20 {
					t.Fatal("damage before resolution")
				}
				callbacks := 0
				g.SetOnDamageDealt(func(_, _ string, amount int, combat bool) {
					callbacks++
					wantLife := 20
					if !tc.selfPermanent {
						wantLife -= tc.self
						if kind == "overlap" {
							wantLife -= tc.targeted
						}
					}
					if a.Life() != wantLife || combat {
						t.Error("incomplete simultaneous results")
					}
					if kind == "player" && b.Life() != 20-tc.targeted {
						t.Error("incomplete player packet")
					}
					if kind == "creature" && g.FindPermanent(target).Damage != tc.targeted {
						t.Error("incomplete creature packet")
					}
					if kind == "planeswalker" && g.FindPermanent(target).Counters[core.Loyalty] != 10-tc.targeted {
						t.Error("incomplete walker packet")
					}
					if tc.selfPermanent && g.FindPermanent(source.ID()).Damage != tc.self+map[bool]int{true: tc.targeted}[kind == "overlap"] {
						t.Error("incomplete self packet")
					}
					if kind == "overlap" && amount != tc.self+tc.targeted {
						t.Error("overlapping packets did not merge")
					}
				})
				g.ResolveTopOfStack()
				want := 2
				if kind == "overlap" {
					want = 1
				}
				if callbacks != want {
					t.Fatalf("callbacks=%d want=%d", callbacks, want)
				}
				if tc.selfPermanent && g.FindPermanent(source.ID()) != nil {
					t.Fatal("lethal self damage did not remove source")
				}
				if tc.spell && (len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != card.ID()) {
					t.Fatal("spell graveyard")
				}
				if a.ManaPool().TotalMana() != 0 {
					t.Fatal("payment")
				}
			})
		}
	}
}

func TestRONDamagePacketsInvalid(t *testing.T) {
	for _, body := range []string{
		`SpellDealsMultipleDamage(ThisSpell,[])`, `SpellDealsMultipleDamage(ThisSpell,DamageToRecipients(Integer(1),Player(You)))`,
		`SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(1))])`, `SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(1),Player(You),Player(You))])`,
		`SpellDealsMultipleDamage(ThisSpell,[Unknown(Integer(1),Player(You))])`, `SpellDealsMultipleDamage(ThisPermanent,[DamageToRecipients(Integer(1),Player(You))])`,
		`PermanentDealsMultipleDamage(ThisPermanent,[DamageToRecipients(Integer(1),Player(You))])`,
		`SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(1),Ref_AnyTarget)])`,
		`SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(1),Permanent(ThisPermanent))])`,
		`SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(1),Player(Ref_TargetPlayer))])`,
		`SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(1),EachPlayer(AnyPlayer))])`,
		`SpellDealsMultipleDamage(ThisSpell,[DamageToRecipients(Integer(1),Player(You()))])`,
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([%s]))])`, body)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		} else if !strings.Contains(err.Error(), "invalid.ron:") {
			t.Fatalf("unpositioned error: %v", err)
		}
	}
}

func TestRONDamagePacketsLegalityAndRollback(t *testing.T) {
	for _, tc := range ronPacketCards {
		for _, scenario := range []string{"return", "phase", "phase-cycle", "shroud", "protection", "land", "sick", "unpaid"} {
			t.Run(tc.name+"/"+scenario, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				var source *mage.Permanent
				if tc.spell {
					a.AddToHand(card)
				} else {
					source = g.PutOnBattlefield(card, a.PlayerID())
					if scenario != "sick" {
						source.RevokeBaseAttr(core.AttrSummonSick)
					}
				}
				victimCard := mage.NewCreature("Victim", "", 1, 10)
				victimCard.SetOwner(b.PlayerID())
				victim := g.PutOnBattlefield(victimCard, b.PlayerID())
				target := victim.ID()
				if scenario == "land" {
					target = g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID()).ID()
				}
				color := core.Red
				if tc.spell || tc.name == "Psionic Entity" {
					color = core.Blue
				}
				if scenario != "unpaid" {
					a.ManaPool().Add(color, tc.mana)
				}
				var err error
				if tc.spell {
					err = g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0)
				} else {
					err = g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target})
				}
				invalid := scenario == "land" || scenario == "sick" && tc.tap || scenario == "unpaid" && tc.mana > 0
				if invalid {
					want := tc.mana
					if scenario == "unpaid" {
						want = 0
					}
					if err == nil || !g.Stack().IsEmpty() || a.ManaPool().TotalMana() != want || source != nil && source.Tapped {
						t.Fatalf("invalid announcement rollback: %v", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				switch scenario {
				case "return":
					g.BouncePermanentToHand(victim)
					b.RemoveFromHand(victimCard.ID())
					g.PutOnBattlefield(victimCard, b.PlayerID())
				case "phase":
					g.PhaseOut(victim.ID())
				case "phase-cycle":
					g.PhaseOut(victim.ID())
					g.PhaseIn([]uuid.UUID{victim.ID()})
				case "shroud":
					g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
				case "protection":
					g.MutablePermanent(victim.ID()).RuntimeAbilities = append(g.MutablePermanent(victim.ID()).RuntimeAbilities, mage.WrapGrantedAbility(mage.ProtectionFromColor(color)))
				}
				rejected := scenario == "return" || scenario == "phase" || scenario == "shroud" || scenario == "protection"
				callbacks := 0
				g.SetOnDamageDealt(func(string, string, int, bool) { callbacks++ })
				g.ResolveStack()
				if rejected {
					if callbacks != 0 || a.Life() != 20 || source != nil && g.FindPermanent(source.ID()) == nil {
						t.Fatal("all-illegal target still dealt untargeted damage")
					}
				} else if callbacks != 2 {
					t.Fatalf("phase cycle or valid target lost packets: %d", callbacks)
				}
			})
		}
	}
}

func TestRONDamagePacketsSourceCopiesAndClones(t *testing.T) {
	for _, tc := range ronPacketCards {
		for _, scenario := range []string{"present", "return", "phase", "control"} {
			if tc.spell && scenario != "present" {
				continue
			}
			t.Run(tc.name+"/"+scenario, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				target := b.PlayerID()
				var source *mage.Permanent
				color := core.Red
				if tc.spell {
					color = core.Blue
					a.AddToHand(card)
				} else {
					source = g.PutOnBattlefield(card, a.PlayerID())
					source.RevokeBaseAttr(core.AttrSummonSick)
					source.GrantBaseAttr(core.Lifelink)
				}
				a.ManaPool().Add(color, tc.mana)
				if tc.spell {
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target}); err != nil {
						t.Fatal(err)
					}
				}
				switch scenario {
				case "return":
					g.BouncePermanentToHand(source)
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				case "phase":
					g.PhaseOut(source.ID())
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
				if copied == nil {
					t.Fatal("copy failed")
				}
				copied.Targets[0] = a.PlayerID()
				clone := g.Clone()
				g.ResolveStack()
				self := tc.self
				if tc.selfPermanent {
					self = 0
				}
				wantA, wantB := 20-tc.targeted-self, 20-tc.targeted-self
				if !tc.spell {
					total := 2*tc.targeted + 2*tc.self
					if tc.selfPermanent && (scenario == "present" || scenario == "control") {
						total -= tc.self
					}
					if tc.selfPermanent && (scenario == "return" || scenario == "phase") {
						total = 2 * tc.targeted
					}
					if scenario == "control" {
						wantB += total
					} else {
						wantA += total
					}
				}
				if a.Life() != wantA || b.Life() != wantB || clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatalf("identity or copy controller: A=%d B=%d want=%d/%d", a.Life(), b.Life(), wantA, wantB)
				}
				if tc.selfPermanent && scenario == "return" && g.FindPermanent(source.ID()).Damage != 0 {
					t.Fatal("old self recipient followed returned source")
				}
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != wantA || clone.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatal("clone lost source or copy bindings")
				}
			})
		}
	}
}

func TestRONDamagePacketsReplacementOrder(t *testing.T) {
	for choice := range 2 {
		a := mage.NewBasePlayer("A")
		b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B"), mode: choice}
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		g.SetTurn(1)
		card := repositoryCard(t, "LEB", "Psionic Blast")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Blue, 3)
		victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 10), b.PlayerID())
		g.SetCreatureDamageRedirect(victim.ID(), b.PlayerID())
		g.AddPreventionShield(victim.ID(), 2)
		g.AddPreventionShield(a.PlayerID(), 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if a.Life() != 19 || b.Life() != 16+2*choice || victim.Damage != 0 || b.prompts != 1 {
			t.Fatalf("replacement order A=%d B=%d prompts=%d", a.Life(), b.Life(), b.prompts)
		}
	}
}

func TestRONDamagePacketsStrictReprintsAndBindings(t *testing.T) {
	names := map[string]bool{}
	for _, tc := range ronPacketCards {
		names[tc.name] = true
	}
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
			if names[node.Fields["Name"].String] {
				records++
				selected = append(selected, lines[node.Position.Line-1])
			}
		}
		if len(selected) > 0 {
			collection[filepath.Base(file)] = &fstest.MapFile{Data: []byte(strings.Join(selected, "\n"))}
		}
	}
	registry, err := LoadFS(collection, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if records != 13 || len(registry.RegisteredCardNames()) != 4 {
		t.Fatalf("records=%d names=%d", records, len(registry.RegisteredCardNames()))
	}
	for name := range names {
		first, _ := registry.CreateCard(name)
		second, _ := registry.CreateCard(name)
		left, right := first.Abilities()[0].(*mage.ActionDefinition), second.Abilities()[0].(*mage.ActionDefinition)
		if first.ID() == second.ID() || left.AbilityID() == right.AbilityID() || left.Targets()[0] == right.Targets()[0] {
			t.Fatal("factory shared instance or binding")
		}
		missing := mage.NewInstant("Missing", "", mage.WithAbility(mage.NewAction(mage.ActionSpell, mage.WithEffects(left.Effects()...))))
		if mage.ValidateCardDefinition(missing.Name(), missing) == nil {
			t.Fatal("packet target read escaped validation")
		}
	}
	report, err := InspectFS(collection, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"DamageToRecipients": "damage_packet", "Ref_AnyTarget": "recipient", "Integer": "value", "Player": "recipient", "Permanent": "recipient", "ThisPermanent": "permanent_reference", "You": "player_reference"} {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing role %s/%s", name, role)
		}
	}
	collection["gap.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Gap",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Unknown])`)}
	if r, err := LoadFS(collection, "*.ron"); err == nil || r != nil {
		t.Fatal("partial collection published")
	}
}

func TestRONDamagePacketsScopedRecipients(t *testing.T) {
	for _, context := range []string{"spell", "activation", "trigger"} {
		for _, kind := range []string{"player", "permanent"} {
			t.Run(context+"/"+kind, func(t *testing.T) {
				source, cardtype := "ThisPermanent", "Artifact"
				if context == "spell" {
					source, cardtype = "ThisSpell", "Instant"
				}
				action := "PermanentDealsMultipleDamage"
				if context == "spell" {
					action = "SpellDealsMultipleDamage"
				}
				target, recipient := "TargetPlayer(AnyPlayer)", "Player(Ref_TargetPlayer)"
				if kind == "permanent" {
					target, recipient = "TargetPermanent(IsCardtype(Creature))", "Permanent(Ref_TargetPermanent)"
				}
				body := fmt.Sprintf(`Targeted([%s],ActionList([%s(%s,[DamageToRecipients(Integer(2),%s),DamageToRecipients(Integer(1),Player(You))])]))`, target, action, source, recipient)
				rule := "SpellActions(" + body + ")"
				if context == "activation" {
					rule = "Activated(TapPermanent(ThisPermanent)," + body + ")"
				}
				if context == "trigger" {
					rule = "TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent))," + body + ")"
				}
				factory := permanentScopeFactory(t, cardtype, rule)
				a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A")}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				g.SetTurn(1)
				card := factory()
				card.SetOwner(a.PlayerID())
				targetID := b.PlayerID()
				if kind == "permanent" {
					targetID = g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 10), b.PlayerID()).ID()
				}
				a.targets = []uuid.UUID{targetID}
				if context == "spell" {
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{targetID}, 0); err != nil {
						t.Fatal(err)
					}
				} else {
					permanent := g.PutOnBattlefield(card, a.PlayerID())
					if context == "activation" {
						if err := g.ActivateAbilityByIndex(a.PlayerID(), permanent.ID(), 0, []uuid.UUID{targetID}); err != nil {
							t.Fatal(err)
						}
					} else {
						g.PutTriggersOnStack()
					}
				}
				g.ResolveStack()
				if a.Life() != 19 {
					t.Fatal("resolving controller lost")
				}
				if kind == "player" && b.Life() != 18 {
					t.Fatal("player target lost")
				}
				if kind == "permanent" && g.FindPermanent(targetID).Damage != 2 {
					t.Fatal("permanent target lost")
				}
			})
		}
	}
}
