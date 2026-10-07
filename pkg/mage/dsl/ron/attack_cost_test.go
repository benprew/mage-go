package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONAttackCostBrainwash(t *testing.T) {
	for _, change := range []string{"pay", "short", "clone", "donor-control", "host-control", "donor-left", "donor-phase", "phase-cycle", "host-return", "removed", "host-removed", "reattach", "stack-copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 3), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 3), b.PlayerID())
			g.MutablePermanent(host.ID()).RevokeBaseAttr(core.AttrSummonSick)
			card := repositoryCard(t, "DRK", "Brainwash")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			a.ManaPool().Add(core.White, 1)
			land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{land.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 1 {
				t.Fatal("illegal target spent mana")
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if !g.PayAttackCosts(host, b.PlayerID(), false) {
				t.Fatal("Aura affected host before resolution")
			}
			if change == "stack-copy" {
				g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
			}
			g.ResolveStack()
			taxed := true
			controller := b.PlayerID()
			switch change {
			case "donor-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "host-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				controller = a.PlayerID()
			case "donor-left":
				g.BouncePermanentToHand(g.FindPermanent(card.ID()))
				taxed = false
			case "donor-phase":
				g.PhaseOut(card.ID())
				taxed = false
			case "phase-cycle":
				p := g.PhaseOut(host.ID())
				g.PhaseIn(p)
			case "host-return":
				g.BouncePermanentToHand(host)
				host = g.PutOnBattlefield(host.Card, b.PlayerID())
				g.MutablePermanent(host.ID()).RevokeBaseAttr(core.AttrSummonSick)
				taxed = false
			case "removed":
				g.AddContinuousEffect(mage.RemoveAllAbilities(card.ID(), core.Indefinite))
				taxed = false
			case "host-removed":
				g.AddContinuousEffect(mage.RemoveAllAbilities(host.ID(), core.Indefinite))
			case "reattach":
				g.Attach(card.ID(), other.ID())
				if !g.PayAttackCosts(g.FindPermanent(host.ID()), controller, false) {
					t.Fatal("former host kept attack cost")
				}
				host = other
				g.MutablePermanent(host.ID()).RevokeBaseAttr(core.AttrSummonSick)
			}
			g.ApplyContinuousEffects()
			amount := 3
			if change == "short" {
				amount = 2
			}
			if change == "stack-copy" {
				amount = 6
			}
			g.MutablePermanent(host.ID()).RevokeBaseAttr(core.AttrSummonSick)
			payer := g.GetPlayer(controller)
			payer.ManaPool().Add(core.Colorless, amount)
			before := g.Clone()
			g.ExecuteAttackers(controller, []uuid.UUID{host.ID()})
			success := change != "short"
			if g.GetCombat().IsAttacking(host.ID()) != success || g.FindPermanent(host.ID()).Tapped != success {
				t.Fatal("attack legality or payment timing")
			}
			want := amount
			if taxed && success {
				want = 0
			}
			if payer.ManaPool().TotalMana() != want {
				t.Fatalf("mana = %d, want %d", payer.ManaPool().TotalMana(), want)
			}
			if before.GetPlayer(controller).ManaPool().TotalMana() != amount || before.GetCombat().IsAttacking(host.ID()) || before.FindPermanent(host.ID()).Tapped {
				t.Fatal("attack changed clone")
			}
			if change == "clone" {
				before.ExecuteAttackers(controller, []uuid.UUID{host.ID()})
				if !before.GetCombat().IsAttacking(host.ID()) || before.GetPlayer(controller).ManaPool().TotalMana() != 0 {
					t.Fatal("clone lost tax")
				}
			}
		})
	}
}

func TestRONAttackCostLeviathan(t *testing.T) {
	for _, change := range []string{"pay", "one", "wrong", "opponent", "phased-island", "sick", "tapped", "source-phase", "control", "clone", "removed"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Leviathan")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			a.ManaPool().Add(core.Colorless, 5)
			a.ManaPool().Add(core.Blue, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("early entry")
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if !p.Tapped || !p.HasKeyword(core.Trample) || p.CurrentPower(g) != 10 || p.CurrentToughness(g) != 10 {
				t.Fatal("metadata or entry")
			}
			g.SetTurn(1)
			g.RunStepWithPriority(core.Untap)
			if !g.FindPermanent(p.ID()).Tapped {
				t.Fatal("untapped in untap step")
			}
			g.UntapPermanent(g.FindPermanent(p.ID()))
			if change != "sick" {
				g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
			} else {
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.AttrSummonSick)
			}
			payer := a.PlayerID()
			if change == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				payer = b.PlayerID()
				g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
			}
			for i := range 2 {
				c := repositoryCard(t, "LEB", "Island")
				c.SetOwner(b.PlayerID())
				if change == "wrong" && i == 1 {
					c = repositoryCard(t, "LEB", "Forest")
					c.SetOwner(b.PlayerID())
				}
				if change == "one" && i == 1 {
					continue
				}
				ctrl := payer
				if change == "opponent" && i == 1 {
					ctrl = b.PlayerID()
				}
				land := g.PutOnBattlefield(c, ctrl)
				if change == "phased-island" && i == 1 {
					g.PhaseOut(land.ID())
				}
			}
			if change == "tapped" {
				g.TapPermanent(g.FindPermanent(p.ID()))
			}
			if change == "source-phase" {
				g.PhaseOut(p.ID())
			}
			if change == "removed" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
			}
			before := g.Clone()
			g.ExecuteAttackers(payer, []uuid.UUID{p.ID()})
			success := change == "pay" || change == "control" || change == "clone" || change == "removed"
			if g.GetCombat().IsAttacking(p.ID()) != success {
				t.Fatal("attack ignored cost or legality")
			}
			want := 0
			if success && change != "removed" {
				want = 2
			}
			if len(b.Graveyard()) != want {
				t.Fatalf("sacrifices = %d, want %d", len(b.Graveyard()), want)
			}
			if len(before.GetPlayer(b.PlayerID()).Graveyard()) != 0 || before.GetCombat().IsAttacking(p.ID()) {
				t.Fatal("clone changed")
			}
			if change == "clone" {
				before.ExecuteAttackers(payer, []uuid.UUID{p.ID()})
				if len(before.GetPlayer(b.PlayerID()).Graveyard()) != 2 {
					t.Fatal("clone lost sacrifice cost")
				}
			}
		})
	}
}

func TestRONAttackCostLeviathanUpkeep(t *testing.T) {
	for _, accept := range []bool{false, true} {
		t.Run(fmt.Sprint(accept), func(t *testing.T) {
			a := &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: accept}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			p := g.PutOnBattlefield(repositoryCard(t, "DRK", "Leviathan"), a.PlayerID())
			for range 2 {
				c := repositoryCard(t, "LEB", "Island")
				c.SetOwner(b.PlayerID())
				g.PutOnBattlefield(c, a.PlayerID())
			}
			queueRONUpkeep(g, b)
			if g.StackSize() != 0 {
				t.Fatal("wrong player's upkeep")
			}
			queueRONUpkeep(g, a)
			if g.StackSize() != 1 || !g.FindPermanent(p.ID()).Tapped || len(b.Graveyard()) != 0 {
				t.Fatal("upkeep payment timing")
			}
			g.ResolveStack()
			want := 0
			if accept {
				want = 2
			}
			if g.FindPermanent(p.ID()).Tapped == accept || len(b.Graveyard()) != want {
				t.Fatal("optional upkeep payment")
			}
			g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
			if g.GetCombat().IsAttacking(p.ID()) {
				t.Fatal("upkeep payment paid attack cost")
			}
		})
	}
}

func TestRONAttackCostInvalidAndInventory(t *testing.T) {
	for _, operation := range []string{`CantAttackUnlessCost`, `CantAttackUnlessCost()`, `CantAttackUnlessCost(PayMana([]),PayMana([]))`, `CantAttackUnlessCost(PayMana([ManaCostX]))`, `CantAttackUnlessCost(PayManaX([ManaCostX],ValueX))`, `CantAttackUnlessCost(SacrificeNumberPermanents(Integer(0),AnyPermanent))`, `CantAttackUnlessCost(SacrificeNumberPermanents(ValueX,AnyPermanent))`, `CantAttackUnlessCost(SacrificeNumberPermanents(Integer(2),SinglePermanent(Ref_TargetPermanent)))`, `CantAttackUnlessCost(TapPermanent(ThisPermanent))`, `CantAttackUnlessCost(And([PayMana([])]))`} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:2),Rules:[PermanentRuleEffect(ThisPermanent,[` + operation + `])])`
		nodes, err := Parse(strings.NewReader(src), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", operation)
		}
	}
	for _, subject := range []string{`HostPermanent`, `Ref_TargetPermanent`, `ThisPermanent()`, `You`} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[PermanentRuleEffect(` + subject + `,[CantAttackUnlessCost(PayMana([]))])])`
		nodes, err := Parse(strings.NewReader(src), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", subject)
		}
	}
	fs := fstest.MapFS{}
	for _, name := range []string{"Brainwash", "Leviathan"} {
		n := hostUntapRecord(t, "DRK", name)
		f, err := Compile(n)
		if err != nil {
			t.Fatal(err)
		}
		if err = mage.ValidateCardDefinition(name, f()); err != nil {
			t.Fatal(err)
		}
		data := fmt.Sprintf(`Card(Name:"%s",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[PermanentRuleEffect(ThisPermanent,[CantAttackUnlessCost(SacrificeNumberPermanents(Integer(2),IsLandType(Island)))])])`, name)
		fs[name+".ron"] = &fstest.MapFile{Data: []byte(data)}
	}
	report, err := InspectFS(fs, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, signature := range report.Signatures {
		if signature.Name == "SacrificeNumberPermanents" && signature.Role != "cost" {
			t.Fatal("nested cost role", signature.Role)
		}
	}
}

type ronAttackCostPlayer struct {
	*optionalSacrificePlayer
	attackers []uuid.UUID
}

func (p *ronAttackCostPlayer) DeclareAttackers(*mage.Game) []uuid.UUID { return p.attackers }

func TestRONAttackCostNativeDeclaration(t *testing.T) {
	for _, name := range []string{"Brainwash", "Leviathan"} {
		for _, accept := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", name, accept), func(t *testing.T) {
				a := &ronAttackCostPlayer{optionalSacrificePlayer: &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: accept}}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				var p *mage.Permanent
				if name == "Brainwash" {
					p = g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 3), a.PlayerID())
					c := repositoryCard(t, "DRK", name)
					c.SetOwner(b.PlayerID())
					b.AddToHand(c)
					b.ManaPool().Add(core.White, 1)
					g.SetActivePlayerIndex(1)
					g.SetStep(core.PrecombatMain)
					if err := g.CastSpellByID(b.PlayerID(), c.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
					g.SetActivePlayerIndex(0)
					a.ManaPool().Add(core.Colorless, 3)
				} else {
					p = g.PutOnBattlefield(repositoryCard(t, "DRK", name), a.PlayerID())
					g.UntapPermanent(p)
					for range 2 {
						c := repositoryCard(t, "LEB", "Island")
						c.SetOwner(a.PlayerID())
						g.PutOnBattlefield(c, a.PlayerID())
					}
				}
				g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
				a.attackers = []uuid.UUID{p.ID()}
				g.RunStepWithPriority(core.DeclareAttackers)
				if g.GetCombat().IsAttacking(p.ID()) != accept || g.FindPermanent(p.ID()).Tapped != accept {
					t.Fatal("native attack ignored payment decision")
				}
				if a.offers != 1 {
					t.Fatalf("payment offers = %d", a.offers)
				}
				if name == "Leviathan" {
					want := 0
					if accept {
						want = 2
					}
					if len(a.Graveyard()) != want {
						t.Fatal("native sacrifice timing")
					}
					if accept {
						blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 2), b.PlayerID())
						g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: p.ID()}})
						g.ExecuteCombatDamage()
						if b.Life() != 12 || g.FindPermanent(p.ID()).Damage != 1 {
							t.Fatal("complete Leviathan did not trample")
						}
					}
				}
			})
		}
	}
}

func TestRONAttackCostStrictLoading(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var records []string
	for line := range strings.SplitSeq(string(data), "\n") {
		for _, name := range []string{"Brainwash", "Leviathan"} {
			if strings.Contains(line, `Name: "`+name+`"`) {
				records = append(records, line)
			}
		}
	}
	if len(records) != 2 {
		t.Fatal("record count", len(records))
	}
	files := fstest.MapFS{"cards.ron": {Data: []byte(strings.Join(records, "\n"))}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Brainwash", "Leviathan"} {
		first, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		second, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		if first.ID() == second.ID() {
			t.Fatal("shared factory instance")
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Rules:[Unsupported])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("accepted partial collection")
	}
	if _, err := registry.CreateCard("Brainwash"); err != nil {
		t.Fatal("failed load replaced registry")
	}
}

func TestRONAttackCostAtomicChoices(t *testing.T) {
	for _, change := range []string{"pay", "short-mana", "invalid", "stale"} {
		t.Run(change, func(t *testing.T) {
			a := &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			nodes, err := Parse(strings.NewReader(`Card(Name:"Atomic",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:3),Rules:[PermanentRuleEffect(ThisPermanent,[CantAttackUnlessCost(PayMana([ManaCostGeneric(1)]))]),PermanentRuleEffect(ThisPermanent,[CantAttackUnlessCost(SacrificeNumberPermanents(Integer(2),IsCardtype(Artifact)))])])`), "atomic.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
			victims := make([]*mage.Permanent, 0, 2)
			for range 2 {
				c := mage.NewArtifact("Payment", "")
				c.SetOwner(b.PlayerID())
				victims = append(victims, g.PutOnBattlefield(c, a.PlayerID()))
			}
			if change != "short-mana" {
				a.ManaPool().Add(core.Colorless, 1)
			}
			if change == "invalid" {
				a.invalid = p
			}
			if change == "stale" {
				a.invalid = victims[0]
				g.RemoveFromBattlefield(victims[0])
				g.PutOnBattlefield(victims[0].Card, a.PlayerID())
			}
			before := g.Clone()
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
			success := change == "pay"
			if g.GetCombat().IsAttacking(p.ID()) != success || g.FindPermanent(p.ID()).Tapped != success {
				t.Fatal("invalid choice declared attack")
			}
			want := 0
			if success {
				want = 2
			}
			if len(b.Graveyard()) != want {
				t.Fatal("partial sacrifice")
			}
			mana := before.GetPlayer(a.PlayerID()).ManaPool().TotalMana()
			if success {
				mana = 0
			}
			if a.ManaPool().TotalMana() != mana {
				t.Fatal("failed payment spent mana")
			}
			if len(before.GetPlayer(b.PlayerID()).Graveyard()) != 0 {
				t.Fatal("batch changed clone")
			}
		})
	}
}

func TestRONAttackCostSacrificedAttacker(t *testing.T) {
	for _, native := range []bool{false, true} {
		t.Run(fmt.Sprint(native), func(t *testing.T) {
			a := &ronAttackCostPlayer{optionalSacrificePlayer: &optionalSacrificePlayer{ronSacrificeChooser: &ronSacrificeChooser{BasePlayer: mage.NewBasePlayer("A")}, accept: true}}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			c := repositoryCard(t, "DRK", "Leviathan")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			g.UntapPermanent(p)
			g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
			if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mage.AddCardType(mage.This, core.TypeLand), mage.AddSubtypeFamily(mage.This, mage.SubtypeLand, "Island")), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			land := repositoryCard(t, "LEB", "Island")
			land.SetOwner(a.PlayerID())
			g.PutOnBattlefield(land, a.PlayerID())
			a.choice = p.ID()
			a.attackers = []uuid.UUID{p.ID()}
			if native {
				g.RunStepWithPriority(core.DeclareAttackers)
			} else {
				g.ExecuteAttackers(a.PlayerID(), a.attackers)
			}
			if g.FindPermanent(p.ID()) != nil || len(a.Graveyard()) != 2 {
				t.Fatal("attack cost did not sacrifice matching source")
			}
			if g.GetCombat().IsAttacking(p.ID()) {
				t.Fatal("sacrificed source became an attacker")
			}
		})
	}
}
