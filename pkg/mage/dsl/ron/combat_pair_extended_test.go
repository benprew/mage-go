package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCombatPairExtendedRealCards(t *testing.T) {
	for _, name := range []string{"Abomination", "Venom"} {
		for _, attacking := range []bool{false, true} {
			for _, color := range []core.Color{core.Green, core.White, core.Blue, core.Colorless} {
				for _, wall := range []bool{false, true} {
					t.Run(name+"/"+boolName(attacking)+"/"+color.String()+"/"+boolName(wall), func(t *testing.T) {
						g, a, b := ronGame()
						set := "LEG"
						if name == "Venom" {
							set = "DRK"
						}
						card := repositoryCard(t, set, name)
						if err := mage.ValidateCardDefinition(name, card); err != nil {
							t.Fatal(err)
						}
						p := g.PutOnBattlefield(card, a.PlayerID())
						if name == "Venom" {
							p = g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 6), a.PlayerID())
							g.Attach(card.ID(), p.ID())
						}
						var options []mage.CardOption
						if wall {
							options = append(options, mage.WithSubTypes("Wall"))
						}
						other := mage.NewCreature("Other", "", 0, 8, options...)
						other.SetColorOverride([]core.Color{color})
						q := g.PutOnBattlefield(other, b.PlayerID())
						p.RevokeBaseAttr(core.AttrSummonSick)
						q.RevokeBaseAttr(core.AttrSummonSick)
						if attacking {
							g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
							g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: q.ID(), AttackerID: p.ID()}})
						} else {
							g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{q.ID()})
							g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: p.ID(), AttackerID: q.ID()}})
						}
						g.ResolveStack()
						g.ExecuteCombatDamage()
						if g.FindPermanent(q.ID()) == nil {
							t.Fatal("destroyed before end of combat")
						}
						g.RunStepWithPriority(core.EndCombat)
						wantDestroyed := color == core.Green || color == core.White
						if name == "Venom" {
							wantDestroyed = !wall
						}
						if (g.FindPermanent(q.ID()) == nil) != wantDestroyed || g.FindPermanent(p.ID()) == nil {
							t.Fatal("wrong opposing creature destroyed")
						}
					})
				}
			}
		}
	}
}

func TestRONCombatPairExtendedIdentity(t *testing.T) {
	for _, name := range []string{"Abomination", "Venom"} {
		for _, change := range []string{"normal", "return", "phase", "phase-cycle", "source-leaves", "move", "color", "wall", "regenerate", "indestructible", "copy", "control"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				set := "LEG"
				if name == "Venom" {
					set = "DRK"
				}
				source := g.PutOnBattlefield(repositoryCard(t, set, name), a.PlayerID())
				host := source
				if name == "Venom" {
					host = g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 6), b.PlayerID())
					g.Attach(source.ID(), host.ID())
				}
				other := mage.NewCreature("Other", "", 0, 8)
				other.SetColorOverride([]core.Color{core.Green})
				q := g.PutOnBattlefield(other, b.PlayerID())
				g.FireEvent(core.GameEvent{Type: core.EvtCreatureBlockedBy, SourceID: host.ID(), TargetID: q.ID(), PlayerID: b.PlayerID()})
				if name == "Venom" {
					g.FireEvent(core.GameEvent{Type: core.EvtCombatPair, SourceID: host.ID(), TargetID: q.ID(), PlayerID: b.PlayerID()})
				}
				if change == "source-leaves" {
					g.RemoveFromBattlefield(source)
				}
				if name == "Venom" && change == "move" {
					newHost := g.PutOnBattlefield(mage.NewCreature("New host", "", 1, 5), a.PlayerID())
					g.Attach(source.ID(), newHost.ID())
				}
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatalf("stack=%d", g.StackSize())
				}
				if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				switch change {
				case "return":
					g.RemoveFromBattlefield(q)
					g.PutOnBattlefield(other, b.PlayerID())
				case "phase":
					g.PhaseOut(q.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(q.ID()))
				case "color":
					if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mage.SetColors(mage.This, core.Blue)), q.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "wall":
					g.AddContinuousEffect(mage.AddSubtypes(q.ID(), mage.SubtypeCreature, core.Indefinite, "Wall"))
					g.ApplyContinuousEffects()
				case "indestructible":
					g.MutablePermanent(q.ID()).GrantBaseAttr(core.Indestructible)
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: q.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				}
				pendingClone := g.Clone()
				for _, game := range []*mage.Game{g, pendingClone} {
					game.ResolveStack()
					if change == "regenerate" || change == "copy" {
						game.AddRegenerationShield(q.ID())
					}
					game.RunStepWithPriority(core.EndCombat)
					want := change == "return" || change == "regenerate" || change == "indestructible"
					if (game.FindPermanent(q.ID()) != nil) != want {
						t.Fatal("wrong captured identity or destruction")
					}
					if change == "phase" {
						game.PhaseIn([]uuid.UUID{q.ID()})
						if game.FindPermanent(q.ID()) == nil {
							t.Fatal("destroyed phased creature")
						}
					}
				}
			})
		}
	}
}

func TestRONCombatPairExtendedInvalid(t *testing.T) {
	blocking := `WhenACreatureBlocksACreature(SinglePermanent(HostPermanent),And([IsCardtype(Creature),IsNonCreatureType(Wall)]))`
	blocked := strings.Replace(blocking, "WhenACreatureBlocksACreature", "WhenACreatureBecomesBlockedByACreature", 1)
	union := `Or([` + blocking + `,` + blocked + `])`
	body := `ActionList([CreateFutureTrigger(AtTheNextEndOfCombat,ActionList([DestroyPermanent(Trigger_ThatOtherCreature)]))])`
	for _, event := range []string{
		`Or([])`, `Or([` + blocking + `])`, `Or([` + blocking + `,` + blocking + `])`,
		`Or([` + blocking + `,` + blocked + `,` + blocked + `])`, `Or(` + blocking + `)`,
		`Or([` + blocking + `,` + strings.Replace(blocked, "HostPermanent", "ThisPermanent", 1) + `])`,
		`Or([` + blocking + `,` + strings.Replace(blocked, "Wall", "Goblin", 1) + `])`,
		strings.Replace(union, "HostPermanent", "HostPermanent()", 1),
		strings.ReplaceAll(union, "IsNonCreatureType(Wall)", "IsColor(Purple)"),
		strings.ReplaceAll(union, "IsNonCreatureType(Wall)", "Or([])"),
		strings.ReplaceAll(union, "IsNonCreatureType(Wall)", "Or([IsColor(Green),IsNonColor(White)])"),
		strings.ReplaceAll(union, "IsNonCreatureType(Wall)", "IsColor(Green),IsColor(White)"),
		strings.ReplaceAll(union, "IsNonCreatureType(Wall)", "WasDealtDamageThisTurn"),
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),TriggerA(` + event + `,` + body + `)])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", event)
		}
	}
	for _, rule := range []string{
		`TriggerA(` + union + `,` + strings.Replace(body, "Trigger_ThatOtherCreature", "Trigger_TheBlockingCreature", 1) + `)`,
		`TriggerA(` + union + `,` + body + `),Activated(PayMana([]),ActionList([DestroyPermanent(Trigger_ThatOtherCreature)]))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),` + rule + `])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONCombatPairExtendedCasting(t *testing.T) {
	for _, name := range []string{"Abomination", "Venom"} {
		g, a, b := ronGame()
		set, generic, colored := "LEG", 3, core.Black
		if name == "Venom" {
			set, generic, colored = "DRK", 1, core.Green
		}
		card := repositoryCard(t, set, name)
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		var targets []uuid.UUID
		if name == "Venom" {
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 6), b.PlayerID())
			targets = []uuid.UUID{host.ID()}
		}
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
			t.Fatal("unpaid casting did not roll back")
		}
		a.ManaPool().Add(core.Colorless, generic)
		a.ManaPool().Add(colored, 2)
		if name == "Venom" {
			land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{land.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 3 {
				t.Fatal("illegal Aura target spent mana")
			}
		}
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		p := g.FindPermanent(card.ID())
		if p == nil || a.ManaPool().TotalMana() != 0 {
			t.Fatal("wrong casting cost or entry")
		}
		if name == "Venom" && p.AttachedTo != targets[0] {
			t.Fatal("Aura did not attach")
		}
		if name == "Abomination" && (p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 6 || !p.HasSubType("Horror")) {
			t.Fatal("wrong creature characteristics")
		}
	}
}

func TestRONCombatPairExtendedIndependentEvents(t *testing.T) {
	for _, name := range []string{"Abomination", "Venom"} {
		g, a, b := ronGame()
		set := "LEG"
		if name == "Venom" {
			set = "DRK"
		}
		victims := make([]*mage.Permanent, 0, 4)
		for i := range 2 {
			source := g.PutOnBattlefield(repositoryCard(t, set, name), a.PlayerID())
			host := source
			if name == "Venom" {
				host = g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 6), b.PlayerID())
				g.Attach(source.ID(), host.ID())
			}
			for j := range 2 {
				card := mage.NewCreature("Victim", "", 0, 8, mage.WithKeyword(core.Shroud))
				card.SetColorOverride([]core.Color{core.Green, core.White})
				q := g.PutOnBattlefield(card, b.PlayerID())
				victims = append(victims, q)
				kind := core.EvtCreatureBlockedBy
				if j == 1 {
					kind = core.EvtDeclaredBlocker
				}
				if name == "Venom" {
					kind = core.EvtCombatPair
				}
				g.FireEvent(core.GameEvent{Type: kind, SourceID: host.ID(), TargetID: q.ID(), PlayerID: a.PlayerID()})
				if i == 0 && j == 0 {
					g.AddRegenerationShield(q.ID())
				}
			}
		}
		g.PutTriggersOnStack()
		if g.StackSize() != 4 {
			t.Fatalf("%s stack=%d", name, g.StackSize())
		}
		clone := g.Clone()
		clone.ResolveStack()
		clone.RunStepWithPriority(core.EndCombat)
		if g.StackSize() != 4 {
			t.Fatal("clone changed pending stack")
		}
		g.ResolveStack()
		g.RunStepWithPriority(core.EndCombat)
		for _, game := range []*mage.Game{g, clone} {
			for i, victim := range victims {
				if (game.FindPermanent(victim.ID()) != nil) != (i == 0) {
					t.Fatal("event or factory bindings crossed")
				}
			}
		}
	}
}

func TestRONCombatPairExtendedInventory(t *testing.T) {
	src := `Card(Name:"Nested",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[TriggerA(Or([WhenACreatureBlocksACreature(SinglePermanent(HostPermanent),And([IsCardtype(Creature),Or([IsColor(Green),IsColor(White)])])),WhenACreatureBecomesBlockedByACreature(SinglePermanent(HostPermanent),And([IsCardtype(Creature),Or([IsColor(Green),IsColor(White)])]))]),ActionList([UnknownAction]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range report.Signatures {
		if strings.HasPrefix(entry.Name, "WhenACreature") && entry.Role != "event" {
			t.Fatalf("event role=%s", entry.Role)
		}
		if entry.Name == "HostPermanent" && entry.Role != "permanent_reference" {
			t.Fatalf("host role=%s", entry.Role)
		}
		if entry.Name == "IsColor" && entry.Role != "permanent_filter" {
			t.Fatalf("color role=%s", entry.Role)
		}
	}
}

func TestRONCombatPairExtendedShapes(t *testing.T) {
	for _, hostSubject := range []bool{false, true} {
		for _, shape := range []string{"blocking", "blocked", "union", "reverse-union"} {
			for _, filter := range []string{`IsCardtype(Creature)`, `And([IsCardtype(Creature),IsColor(Green)])`, `And([IsColor(Green),IsNonCreatureType(Wall),IsCardtype(Creature)])`, `And([Or([IsColor(Green),IsColor(White)]),IsCardtype(Creature)])`} {
				g, a, b := ronGame()
				subject, typeline := "ThisPermanent", `(Supertypes:[],Cardtypes:[Creature],Subtypes:[])`
				if hostSubject {
					subject, typeline = "HostPermanent", `(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura])`
				}
				blocking := `WhenACreatureBlocksACreature(SinglePermanent(` + subject + `),` + filter + `)`
				blocked := strings.Replace(blocking, "WhenACreatureBlocksACreature", "WhenACreatureBecomesBlockedByACreature", 1)
				event, ref, kind := blocking, "Trigger_ThatOtherCreature", core.EvtDeclaredBlocker
				switch shape {
				case "blocked":
					event, ref, kind = blocked, "Trigger_TheBlockingCreature", core.EvtCreatureBlockedBy
				case "union":
					event, kind = `Or([`+blocking+`,`+blocked+`])`, core.EvtCombatPair
				case "reverse-union":
					event, kind = `Or([`+blocked+`,`+blocking+`])`, core.EvtCombatPair
				}
				src := `Card(Name:"Shape",Typeline:` + typeline + `,CardPT:(Power:2,Toughness:6),Rules:[TriggerA(` + event + `,ActionList([DestroyPermanent(` + ref + `)]))])`
				if hostSubject {
					src = strings.Replace(src, `,CardPT:(Power:2,Toughness:6)`, "", 1)
					src = strings.Replace(src, `Rules:[`, `Rules:[EnchantPermanent(IsCardtype(Creature)),`, 1)
				}
				nodes, err := Parse(strings.NewReader(src), "shape.ron")
				if err != nil {
					t.Fatal(err)
				}
				factory, err := Compile(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
				card := factory()
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				p := g.PutOnBattlefield(card, a.PlayerID())
				if hostSubject {
					p = g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 6), b.PlayerID())
					g.Attach(card.ID(), p.ID())
				}
				other := mage.NewCreature("Other", "", 0, 8)
				other.SetColorOverride([]core.Color{core.Green})
				q := g.PutOnBattlefield(other, b.PlayerID())
				g.FireEvent(core.GameEvent{Type: kind, SourceID: p.ID(), TargetID: q.ID(), PlayerID: b.PlayerID()})
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("missing shape event")
				}
				g.ResolveStack()
				if g.FindPermanent(q.ID()) != nil {
					t.Fatal("wrong shape binding")
				}
			}
		}
	}
}
