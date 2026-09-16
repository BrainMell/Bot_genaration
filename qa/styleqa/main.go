package main

// styleqa - renders EVERY rebuilt kind x EVERY style to PNG for visual QA.
// Usage: go run ./qa/styleqa -out /tmp/styleqa
// Prints one line per render: S<style>_<kind>.png <bytes>

import (
	"encoding/json"
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"image-service/pkg/combat"
	"image-service/pkg/economy"
)

// portrait payloads (wire format of /api/cards/portrait), one per kind.
func portraitPayloads() map[string]string {
	rows7 := `[{"label":"HP","value":"2","sub":"+15/pt"},{"label":"ATK","value":"1","sub":"+3/pt"},{"label":"DEF","value":"0","sub":"+2/pt"},{"label":"MAG","value":"1","sub":"+3/pt"},{"label":"SPD","value":"0","sub":"+2/pt"},{"label":"LUCK","value":"0","sub":"+2/pt"},{"label":"CRIT","value":"0","sub":"+1/pt"}]`
	standing := `[{"label":"ZENI","value":"12,450"},{"label":"PVP WINS","value":"38"},{"label":"DUELS LOST","value":"9"},{"label":"QUESTS","value":"21"},{"label":"ITEMS CRAFTED","value":"104"}]`
	progress := `[{"label":"LEVEL","cur":14,"max":20,"done":false,"valueText":"14 / 20"},{"label":"WINS TO B-RANK","cur":38,"max":50,"done":false,"valueText":"38 / 50"},{"label":"STORY ARC","cur":9,"max":9,"done":true,"valueText":"COMPLETE"}]`
	entries := `[{"title":"Iron Longsword","icon":"sword","sub":"ATK +12 · COMMON","value":"850 Z","runes":"ATK"},{"title":"Traveler Garb","icon":"armor","sub":"DEF +8","value":"420 Z","runes":"DEF"},{"title":"HP Tonic","icon":"potion","sub":"Restores 50 HP","value":"60 Z","runes":"HEAL"}]`
	slots := `[` +
		`{"slot":"MAIN HAND","icon":"sword","name":"Iron Longsword","tierLabel":"RARE","tier":3,"dur":44,"durMax":60,"empty":false},` +
		`{"slot":"OFF HAND","icon":"shield","name":"Buckler","tierLabel":"COMMON","tier":1,"dur":31,"durMax":40,"empty":false},` +
		`{"slot":"ARMOR","icon":"armor","name":"Traveler Garb","tierLabel":"UNCOMMON","tier":2,"dur":18,"durMax":55,"empty":false},` +
		`{"slot":"HELMET","icon":"helmet","name":"","tierLabel":"","tier":0,"dur":0,"durMax":0,"empty":true},` +
		`{"slot":"AMULET","icon":"amulet","name":"Bone Charm","tierLabel":"RARE","tier":3,"dur":50,"durMax":50,"empty":false},` +
		`{"slot":"RING","icon":"ring","name":"Signet","tierLabel":"COMMON","tier":1,"dur":22,"durMax":30,"empty":false},` +
		`{"slot":"CAPE","icon":"cape","name":"","tierLabel":"","tier":0,"dur":0,"durMax":0,"empty":true},` +
		`{"slot":"BOOTS","icon":"boots","name":"Leather Boots","tierLabel":"COMMON","tier":1,"dur":12,"durMax":35,"empty":false},` +
		`{"slot":"RELIC","icon":"relic","name":"Old Compass","tierLabel":"UNCOMMON","tier":2,"dur":40,"durMax":45,"empty":false}]`
	branches := `[` +
		`{"name":"BLADE","skills":[{"name":"Swift Cut","cur":3,"max":5,"state":"learned","tier":1},{"name":"Parry","cur":2,"max":5,"state":"learned","tier":2},{"name":"Cleave","cur":1,"max":3,"state":"open","tier":3},{"name":"Executioner","cur":0,"max":1,"state":"locked","tier":4}]},` +
		`{"name":"GUARD","skills":[{"name":"Stone Stance","cur":5,"max":5,"state":"maxed","tier":1},{"name":"Iron Skin","cur":2,"max":5,"state":"learned","tier":2},{"name":"Bulwark","cur":0,"max":3,"state":"locked","tier":3}]},` +
		`{"name":"RAGE","skills":[{"name":"Fury","cur":4,"max":5,"state":"learned","tier":1},{"name":"Berserk","cur":0,"max":4,"state":"open","tier":2},{"name":"Undying","cur":0,"max":1,"state":"locked","tier":3}]}]`
	groups := `[` +
		`{"name":"SIGNATURE ARTS","sub":"learned techniques","items":[{"title":"Swift Cut","icon":"blade","sub":"ATK 140% · ignores guard","value":"LVL 3","runes":"ATK DOT"},{"title":"Parry Stance","icon":"shield","sub":"Blocks the next hit","value":"LVL 2","runes":"DEF STUN"},{"title":"Rising Fang","icon":"claw","sub":"ATK 180%","value":"LVL 1","runes":"ATK"}]},` +
		`{"name":"PASSIVE DISCIPLINE","sub":"always active","items":[{"title":"Iron Body","icon":"body","sub":"DEF +8%","value":"LVL 2","runes":"DEF BUFF"}]}]`
	out := map[string]string{
		"RANK":      `{"kind":"RANK","style":@STYLE@,"nickname":"Kaelen","level":24,"rankLetter":"B","rankLine":"B-RANK ADVENTURER","xpNow":"12.4K","xpLeft":"17.2K","xpPercent":42,"standing":` + standing + `,"progressTitle":"THE PATH AHEAD","progress":` + progress + `,"sealText":"B","caption":"the guild watches your climb"}`,
		"ALLOCATE":  `{"kind":"ALLOCATE","style":@STYLE@,"nickname":"Kaelen","pill":"Scout · Starter","pointsBig":"5 POINTS","spentNow":"SPENT 0","spentLeft":"5 LEFT TO SPEND","spentPercent":0,"sealText":"5","caption":"spent points are permanent - choose wisely","rows":` + rows7 + `,"ctaLabel":".j allocate <stat> [amount]","ctaSub":"e.g. .j allocate ATK 5  ·  .j allocate HP 3"}`,
		"SKILLUP":   `{"kind":"SKILLUP","style":@STYLE@,"nickname":"Kaelen","skillName":"Swift Cut","skillMax":5,"cur":3,"tier":2,"ascended":false,"sealText":"3","caption":"the blade remembers"}`,
		"ABILITIES": `{"kind":"ABILITIES","style":@STYLE@,"docTitle":"COMBAT CODEX","docQuote":"Every scar is a lesson. Every battle, a page.","pageLabel":"PAGE 1 / 3","startNumber":1,"groups":` + groups + `}`,
		"SKILLTREE": `{"kind":"SKILLTREE","style":@STYLE@,"className":"Fighter","level":24,"skillPoints":2,"branches":` + branches + `}`,
		"EQUIP":     `{"kind":"EQUIP","style":@STYLE@,"nickname":"Kaelen","sealText":"B","caption":"7/9 slots filled - repair at .j blacksmith","playerClass":"FIGHTER","playerIndex":0,"slots":` + slots + `}`,
		"SHOP":      `{"kind":"SHOP","style":@STYLE@,"nickname":"Kaelen","caption":".j buy <item>","entries":` + entries + `}`,
		"GUILDINFO": `{"kind":"GUILDINFO","style":@STYLE@,"nickname":"Ember Watch","motto":"We hold the line so others may run.","sealText":"L7","level":7,"xpPercent":64,"hexColor":"#7a2c3a","rows":[{"label":"ARCHETYPE","value":"MILITANT"},{"label":"MEMBERS","value":"18 / 30"},{"label":"TREASURY","value":"54,200 Z"},{"label":"FOUNDED","value":"2026-01-30"}],"buildings":[{"name":"Forge","level":3},{"name":"Vault","level":5},{"name":"Garden","level":2}]}`,
	}
	return out
}

func econPayloads() map[string]economy.TransactionCardRequest {
	base := economy.TransactionCardRequest{
		Nickname: "Kaelen", Amount: 1, NewWallet: 12450, ZeniSymbol: "Z", SealText: "B",
	}
	out := map[string]economy.TransactionCardRequest{}
	for _, t := range []string{"CRAFT", "BREW", "COOK", "FORGE", "FISH"} {
		r := base
		r.Type = t
		r.ItemName = map[string]string{"CRAFT": "Iron Longsword", "BREW": "Emerald Tonic", "COOK": "Hunter's Stew", "FORGE": "Ember Edge", "FISH": "Silverfin"}[t]
		if t == "FISH" {
			r.Details = "a fine catch under the morning fog"
		}
		out[t] = r
	}
	d := base
	d.Type = "DECREE"
	d.ItemName = "A-RANK"
	d.Details = "B-RANK -> A-RANK"
	d.SealText = "A"
	out["DECREE"] = d
	// money movement kinds (2026-09-16): one flow per direction
	mv := map[string]struct {
		amount float64
		wallet float64
		bank   float64
		det    string
	}{
		"TRANSFER": {2500, 9950, 4200, "to a fellow adventurer"},
		"DEPOSIT":  {4000, 8450, 8200, "locked in the vault"},
		"WITHDRAW": {1500, 13950, 2700, "coin in hand again"},
	}
	for t, m := range mv {
		r := base
		r.Type = t
		r.Amount = m.amount
		r.NewWallet = m.wallet
		r.NewBank = m.bank
		r.Details = m.det
		out[t] = r
	}
	return out
}

// balancePayload - the themed treasury register (econ_style_money.go).
func balancePayload() economy.EconomyCardRequest {
	return economy.EconomyCardRequest{
		Nickname: "Kaelen", Wallet: 12450, Bank: 68000, Total: 80450,
		Frozen: 0, ZeniSymbol: "Z", Rank: "B", Level: 24, Style: 1,
	}
}

func main() {
	out := flag.String("out", "/tmp/styleqa", "output dir")
	flag.Parse()
	if err := os.MkdirAll(*out, 0o755); err != nil {
		panic(err)
	}
	styles := []int{1, 2, 3, 4, 5, 6, 8, 9, 10}
	count := 0
	for _, st := range styles {
		for kind, tpl := range portraitPayloads() {
			payload := strings.ReplaceAll(tpl, "@STYLE@", strconv.Itoa(st))
			var check map[string]any
			if err := json.Unmarshal([]byte(payload), &check); err != nil {
				fmt.Println("BAD PAYLOAD", kind, err)
				continue
			}
			img := combat.QAStyledRender([]byte(payload))
			if img == nil {
				fmt.Println("NIL", st, kind)
				continue
			}
			name := filepath.Join(*out, fmt.Sprintf("S%02d_%s.png", st, kind))
			f, err := os.Create(name)
			if err != nil {
				panic(err)
			}
			if err := png.Encode(f, img); err != nil {
				panic(err)
			}
			f.Close()
			count++
			fmt.Println(name)
		}
		for _, t := range []string{"CRAFT", "BREW", "COOK", "FORGE", "FISH", "DECREE", "TRANSFER", "DEPOSIT", "WITHDRAW"} {
			req := econPayloads()[t]
			req.Style = st
			img := economy.QAStyledEconomy(&req)
			if img == nil {
				fmt.Println("NIL", st, t)
				continue
			}
			name := filepath.Join(*out, fmt.Sprintf("S%02d_%s.png", st, t))
			f, err := os.Create(name)
			if err != nil {
				panic(err)
			}
			if err := png.Encode(f, img); err != nil {
				panic(err)
			}
			f.Close()
			count++
			fmt.Println(name)
		}
		// themed balance card
		b := balancePayload()
		b.Style = st
		if img := economy.QAStyledBalance(&b); img != nil {
			name := filepath.Join(*out, fmt.Sprintf("S%02d_BALANCE.png", st))
			f, err := os.Create(name)
			if err != nil {
				panic(err)
			}
			if err := png.Encode(f, img); err != nil {
				panic(err)
			}
			f.Close()
			count++
			fmt.Println(name)
		}
	}
	fmt.Println("TOTAL RENDERED:", count)
}
