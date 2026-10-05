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

func seedRONDrawLibrary(p *mage.BasePlayer) {
	for range 12 {
		c := mage.NewLand("Library")
		c.SetOwner(p.PlayerID())
		p.AddToLibrary(c)
	}
}

func TestRONDrawStepCards(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, name := range []string{"Howling Mine", "Mana Vault"} {
			t.Run(set+"/"+name, func(t *testing.T) {
				g, a, b := ronGame()
				seedRONDrawLibrary(a)
				seedRONDrawLibrary(b)
				c := repositoryCard(t, set, name)
				c.SetOwner(a.PlayerID())
				if err := mage.ValidateCardDefinition(name, c); err != nil {
					t.Fatal(err)
				}
				a.AddToHand(c)
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err == nil {
					t.Fatal("unpaid cast accepted")
				}
				if len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("failed cast changed state")
				}
				cost := 2
				if name == "Mana Vault" {
					cost = 1
				}
				a.ManaPool().Add(core.Colorless, cost)
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				p := g.FindPermanent(c.ID())
				if name == "Mana Vault" {
					if !mage.IsActivatedManaAbility(c.Abilities()[3]) {
						t.Fatal("missing mana classification")
					}
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 3, nil); err != nil {
						t.Fatal(err)
					}
					if g.StackSize() != 0 || !g.FindPermanent(p.ID()).Tapped || a.ManaPool().TotalMana() != 3 {
						t.Fatal("mana ability did not resolve immediately")
					}
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 3, nil); err == nil {
						t.Fatal("tapped mana activation accepted")
					}
					if a.ManaPool().TotalMana() != 3 {
						t.Fatal("failed activation changed mana")
					}
					g.RunStepWithPriority(core.Untap)
					if !g.FindPermanent(p.ID()).Tapped {
						t.Fatal("vault untapped during untap step")
					}
				}
				g.SetTurn(2)
				g.RunStepWithPriority(core.Draw)
				wantHand := 2
				wantLife := 20
				if name == "Mana Vault" {
					wantHand = 1
					wantLife = 19
				}
				if len(a.Hand()) != wantHand || a.Life() != wantLife || len(b.Hand()) != 0 || b.Life() != 20 {
					t.Fatal("native draw step used wrong player or timing")
				}
				g.SetActivePlayerIndex(1)
				g.RunStepWithPriority(core.Draw)
				wantHand = 2
				if name == "Mana Vault" {
					wantHand = 1
				}
				if len(b.Hand()) != wantHand || b.Life() != 20 || a.Life() != wantLife {
					t.Fatal("opponent draw step")
				}
			})
		}
	}
}

func TestRONDrawStepConditionsAndIdentity(t *testing.T) {
	for _, name := range []string{"Howling Mine", "Mana Vault"} {
		for _, change := range []string{"none", "false-at-event", "false-at-resolution", "restore", "leave", "leave-false", "return", "phase-before", "phase-after", "phase-cycle", "control", "copy", "remove-before", "remove-after"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				seedRONDrawLibrary(a)
				seedRONDrawLibrary(b)
				c := repositoryCard(t, "LEA", name)
				c.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
				tapped := name == "Mana Vault"
				if tapped {
					g.TapPermanent(p)
				}
				setCondition := func(valid bool) {
					t.Helper()
					if tapped == valid {
						g.TapPermanent(g.FindPermanent(p.ID()))
					} else {
						g.UntapPermanent(g.FindPermanent(p.ID()))
					}
				}
				if change == "false-at-event" {
					setCondition(false)
				}
				if change == "phase-before" {
					g.PhaseOut(p.ID())
				}
				if change == "remove-before" {
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
				if change == "false-at-event" || change == "phase-before" || change == "remove-before" {
					if g.StackSize() != 0 {
						t.Fatal("invalid detection triggered")
					}
					return
				}
				if g.StackSize() != 1 || len(a.Hand()) != 0 || a.Life() != 20 {
					t.Fatal("trigger timing")
				}
				switch change {
				case "false-at-resolution", "restore", "leave-false":
					setCondition(false)
					if change == "restore" {
						setCondition(true)
					}
					if change == "leave-false" {
						g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					}
				case "leave", "return":
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					if change == "return" {
						a.RemoveFromHand(c.ID())
						p = g.PutOnBattlefield(c, a.PlayerID())
						setCondition(false)
					}
				case "phase-after":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				case "remove-after":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				clone := g.Clone()
				before := len(a.Hand())
				clone.ResolveStack()
				if a.Life() != 20 || len(a.Hand()) != before {
					t.Fatal("clone changed live game")
				}
				g.ResolveStack()
				count := 1
				if change == "false-at-resolution" || change == "leave-false" {
					count = 0
				}
				if change == "copy" {
					count = 2
				}
				if name == "Howling Mine" {
					if len(a.Hand()) != before+count || len(b.Hand()) != 0 {
						t.Fatal("lost event player or source LKI")
					}
				} else {
					wantA, wantB := 20-count, 20
					if change == "copy" {
						wantA = 19
						wantB = 19
					}
					if a.Life() != wantA || b.Life() != wantB {
						t.Fatalf("damage controller or LKI: %d/%d want %d/%d", a.Life(), b.Life(), wantA, wantB)
					}
				}
				if clone.GetPlayer(a.PlayerID()).Life() != a.Life() || len(clone.GetPlayer(a.PlayerID()).Hand()) != len(a.Hand()) {
					t.Fatal("clone lost captured context")
				}
			})
		}
	}
}

func TestRONDrawStepVaultPayment(t *testing.T) {
	for _, choice := range []string{"pay", "decline", "short", "control"} {
		t.Run(choice, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: choice != "decline"}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			c := repositoryCard(t, "LEA", "Mana Vault")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.TapPermanent(p)
			payer := a
			if choice == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				payer = b
			}
			amount := 4
			if choice == "short" {
				amount = 3
			}
			payer.ManaPool().Add(core.Colorless, amount)
			other := b
			if payer == b {
				other = a
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: other.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("wrong controller upkeep")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: payer.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || payer.offers != 0 {
				t.Fatal("upkeep choice timing")
			}
			g.ResolveStack()
			paid := choice == "pay" || choice == "control"
			if g.FindPermanent(p.ID()).Tapped == paid {
				t.Fatal("wrong untap choice")
			}
			want := amount
			if paid {
				want = 0
			}
			if payer.ManaPool().TotalMana() != want {
				t.Fatal("failed payment did not roll back")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: payer.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			life := 20
			if !paid {
				life = 19
			}
			if payer.Life() != life || other.Life() != 20 {
				t.Fatal("upkeep did not affect draw-step condition")
			}
		})
	}
}

func TestRONDrawStepBindings(t *testing.T) {
	g, a, b := ronGame()
	seedRONDrawLibrary(a)
	seedRONDrawLibrary(b)
	for range 2 {
		g.PutOnBattlefield(repositoryCard(t, "LEA", "Howling Mine"), a.PlayerID())
	}
	g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 4 {
		t.Fatal("separate instances/events lost")
	}
	g.ResolveStack()
	if len(a.Hand()) != 2 || len(b.Hand()) != 2 {
		t.Fatal("event binding overwritten")
	}
	g, a, b = ronGame()
	c := repositoryCard(t, "LEA", "Mana Vault")
	p := g.PutOnBattlefield(c, a.PlayerID())
	g.TapPermanent(p)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("own draw step used owner")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 19 {
		t.Fatal("current controller did not trigger")
	}
}

func TestRONDrawStepInvalid(t *testing.T) {
	valid := `TriggerI(AtTheBeginningOfAPlayersDrawStep(AnyPlayer),PermanentPassesFilter(ThisPermanent,IsUntapped),ActionList([PlayerAction(Trigger_ThatPlayer,DrawACard)]))`
	for _, rule := range []string{
		strings.Replace(valid, "AnyPlayer", "AnyPlayer()", 1), strings.Replace(valid, "AnyPlayer", "SinglePlayer(Opponent)", 1), strings.Replace(valid, "AnyPlayer", "SinglePlayer(You())", 1), strings.Replace(valid, "AnyPlayer", "SinglePlayer(HostController)", 1), strings.Replace(valid, "ThisPermanent", "HostPermanent", 1), strings.Replace(valid, "ThisPermanent", "Ref_TargetPermanent", 1), strings.Replace(valid, "IsUntapped", "IsUntapped()", 1), strings.Replace(valid, "IsUntapped", "AnyPermanent", 1), strings.Replace(valid, "Trigger_ThatPlayer", "Trigger_ThatSpell", 1), valid + `,Activated(PayMana([]),ActionList([PlayerAction(Trigger_ThatPlayer,DrawACard)]))`, strings.Replace(valid, "PermanentPassesFilter(ThisPermanent,IsUntapped),", "", 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid-draw.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONDrawStepInventoryAndReprints(t *testing.T) {
	fs := fstest.MapFS{}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), set)
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range nodes {
			if slices.Contains([]string{"Howling Mine", "Mana Vault"}, n.Fields["Name"].String) {
				lines := strings.Split(string(data), "\n")
				fs[fmt.Sprintf("%s-%s.ron", set, n.Fields["Name"].String)] = &fstest.MapFile{Data: []byte(lines[n.Position.Line-1])}
			}
		}
	}
	registry, err := LoadFS(fs, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("reprints not canonical")
	}
	for _, name := range []string{"Howling Mine", "Mana Vault"} {
		first, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		second, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		if first.ID() == second.ID() {
			t.Fatal("factory shared card identity")
		}
		for i, ability := range first.Abilities() {
			if ability == second.Abilities()[i] {
				t.Fatal("factory shared ability state")
			}
		}
	}
	report, err := InspectFS(fs, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 12 {
		t.Fatal("reprints rejected")
	}
	for name, role := range map[string]string{"AnyPlayer": "player_filter", "SinglePlayer": "player_filter", "You": "player_reference", "Trigger_ThatPlayer": "player_reference", "IsUntapped": "permanent_filter"} {
		if !slices.ContainsFunc(report.Signatures, func(s InventorySignature) bool { return s.Name == name && s.Role == role && s.Support == "unverified" }) {
			t.Fatal("missing role", name, role)
		}
	}
	fs["gap.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Gap",Rules:[Unsupported])`)}
	if r, err := LoadFS(fs, "*.ron"); err == nil || r != nil {
		t.Fatal("partial registry published")
	}
}

func TestRONDrawStepBeforeNormalDrawReplacement(t *testing.T) {
	for _, initiallyTapped := range []bool{false, true} {
		t.Run(fmt.Sprint(initiallyTapped), func(t *testing.T) {
			g, a, _ := ronGame()
			g.SetTurn(2)
			card := repositoryCard(t, "LEA", "Howling Mine")
			card.SetOwner(a.PlayerID())
			mine := g.PutOnBattlefield(card, a.PlayerID())
			if initiallyTapped {
				g.TapPermanent(mine)
			}
			replacement := g.PutOnBattlefield(mage.NewArtifact("Draw replacement", ""), a.PlayerID())
			calls := 0
			g.AddEmptyLibraryDrawReplacement(replacement.ID(), a.PlayerID(), func(game *mage.Game, _ uuid.UUID) {
				calls++
				if source := game.FindPermanent(mine.ID()); source != nil {
					if initiallyTapped {
						game.UntapPermanent(source)
					} else {
						game.DestroyPermanent(source)
					}
				}
			})
			g.RunStepWithPriority(core.Draw)
			want := 2
			if initiallyTapped {
				want = 1
			}
			if calls != want {
				t.Fatalf("replacement calls=%d want %d; detection must precede normal draw", calls, want)
			}
		})
	}
}

func TestRONDrawStepVaultDamagePrevention(t *testing.T) {
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mana Vault"), a.PlayerID())
	g.TapPermanent(p)
	g.AddPreventionShield(a.PlayerID(), 1)
	g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("damage bypassed prevention")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtDrawStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 19 {
		t.Fatal("shield was not consumed")
	}
}
