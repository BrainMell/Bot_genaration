package combat

import (
        "path/filepath"
        "strings"
)

var CharacterSprites = map[string][]string{
        "FIGHTER":      {"Fighter1.png", "fighter2.png", "fighter3.png"},
        "SCOUT":        {"scout1.png", "scout2.png", "scout3.png", "sout4.png"},
        "APPRENTICE":   {"apprentice1.png", "apprentice2.png", "apprentice3.png", "apprentice4.png"},
        "ACOLYTE":      {"acolyte.png"},
        "WARRIOR":      {"warrior1.png", "warrior2.png", "warrior3.png", "warrior4.png"},
        "WARLORD":      {"Warlord1.png", "warlord2.png", "warlord3.png"},
        "BERSERKER":    {"Berserker1.png", "Berserker2.png", "Berserker3.png"},
        "DOOMSLAYER":   {"DoomSlayer1.png", "DoomSlayer2.png"},
        "PALADIN":      {"Paladin (1).png", "Paladin (2).png", "Paladin (3).png", "Paladin (4).png", "Paladin (5).png", "Paladin (6).png", "Paladin (7).png", "Paladin (8).png"},
        "TEMPLAR":      {"Templar (1).png", "Templar (2).png", "Templar (3).png", "Templar (4).png", "Templar (5).png", "Templar (6).png", "Templar (7).png", "Templar (8).png", "Templar (9).png"},
        "ROGUE":        {"Rogue (1).png", "Rogue (2).png", "Rogue (3).png", "Rogue (4).png"},
        "NIGHTBLADE":   {"Nightblade (1).png", "Nightblade (2).png", "Nightblade (3).png", "Nightblade (4).png", "Nightblade (5).png", "Nightblade (6).png"},
        "MONK":         {"Monk.png"},
        "ZENMASTER":    {"zenmaster.png"},
        "NINJA":        {"ninja (1).png", "ninja (2).png", "ninja (3).png", "ninja (4).png", "ninja (5).png"},
        "MAGE":         {"archmage (1).png", "archmage (2).png", "archmage (3).png", "archmage (4).png", "archmage (5).png"},
        "ARCHMAGE":     {"archmage (6).png", "archmage (7).png", "archmage (8).png", "archmage (9).png", "archmage (10).png", "archmage (11).png", "archmage (12).png"},
        "WARLOCK":      {"voidwalker (1).png", "voidwalker (2).png", "voidwalker (3).png", "voidwalker (4).png"},
        "VOIDWALKER":   {"voidwalker (5).png", "voidwalker (6).png", "voidwalker (7).png", "voidwalker (8).png", "voidwalker (9).png"},
        "ELEMENTALIST": {"elementalist (1).png", "elementalist (2).png", "elementalist (3).png", "elementalist (4).png"},
        "CLERIC":       {"cleric (1).png", "cleric (2).png", "cleric (3).png", "cleric (4).png", "cleric (5).png", "cleric (6).png"},
        "SAINT":        {"saint (1).png", "saint (2).png", "saint (3).png", "saint (4).png"},
        "DRUID":        {"druid (1).png", "druid (2).png", "druid (3).png", "druid (4).png", "druid (5).png", "druid (6).png"},
        "ARCHDRUID":    {"archdruid (1).png", "archdruid (2).png", "archdruid (3).png", "archdruid (4).png", "archdruid (5).png", "archdruid (6).png", "archdruid (7).png", "archdruid (8).png", "archdruid (9).png"},
        "NECROMANCER":  {"necromancer.png"},
        "LICH":         {"lich.png"},
        "MERCHANT":     {"merchant.png"},
        "TYCOON":       {"tycoon.png"},
        "CHRONOMANCER": {"timelord (1).png", "timelord (2).png", "timelord (3).png", "timelord (4).png", "timelord (5).png"},
        "TIMELORD":     {"timelord (1).png", "timelord (2).png", "timelord (3).png", "timelord (4).png", "timelord (5).png"},
        "SAMURAI":      {"samuri (1).png", "samuri (2).png", "samuri (3).png", "samuri (4).png", "samuri (5).png", "samuri (6).png", "samuri (7).png", "samuri (8).png", "samuri (9).png", "samuri (10).png", "samuri (11).png"},
        "GOD_HAND":     {"God_hand (1).png", "God_hand (2).png"},
        "DRAGONSLAYER": {"warrior1.png", "warrior2.png", "warrior3.png", "warrior4.png"},
        "REAPER":       {"necromancer.png"},
        "BARD":         {"acolyte.png"},
        "ARTIFICER":    {"apprentice1.png", "apprentice2.png", "apprentice3.png", "apprentice4.png"},
        "AVATAR":       {"elementalist (1).png", "elementalist (2).png", "elementalist (3).png", "elementalist (4).png"},
        // Ascended classes — mapped to closest thematic sprites
        "DRAGON_GOD":     {"DoomSlayer1.png", "DoomSlayer2.png"},
        "DRAGON_LORD":    {"Warlord1.png", "warlord2.png", "warlord3.png"},
        "KAGE":           {"ninja (1).png", "ninja (2).png", "ninja (3).png", "ninja (4).png", "ninja (5).png"},
        "SHOGUN":         {"samuri (1).png", "samuri (2).png", "samuri (3).png"},
        "VIRTUOSO":       {"acolyte.png"},
        "GRAND_INVENTOR": {"apprentice1.png", "apprentice2.png"},
}

var EnemySprites = map[string][]string{
        "FIRE_LOW":    {"fire (5).png", "fire (6).png"},
        "WATER_LOW":   {"water (4).png", "water (6).png"},
        "EARTH_MID":   {"earth (1).png", "earth (2).png", "earth (3).png"},
        "ICE_MID":     {"ice (1).png", "ice (2).png", "ice (3).png"},
        "FIRE_HIGH":   {"fire (7).png", "fire (8).png"},
        "WATER_HIGH":  {"water (7).png"},
        "EARTH_HIGH":  {"earth (4).png", "earth (5).png"},
        "MUTATED":     {"mutated (1).png", "mutated (2).png", "mutated (3).png", "mutated (4).png", "mutated (5).png", "mutated (6).png", "mutated (7).png"},
        "HYBRID":      {"hybrides (1).png", "hybrides (2).png", "hybrides (3).png", "hybrides (4).png", "hybrides (5).png", "hybrides (6).png", "hybrides (7).png"},
        "FIRE_ELITE":  {"fire (11).png"},
}

var BossSprites = map[string][]string{
        "MID_BOSSES":  {"midlevelbosses (1).png", "midlevelbosses (2).png", "midlevelbosses (3).png", "midlevelbosses (4).png", "midlevelbosses (5).png", "midlevelbosses (6).png", "midlevelbosses (7).png"},
        "HIGH_BOSSES": {"highlevelbosses (7).png", "highlevelbosses (8).png", "highlevelbosses (9).png", "highlevelbosses (10).png", "highlevelbosses (11).png", "highlevelbosses (12).png", "highlevelbosses (13).png"},
        "CALAMITY":    {"calamaties (1).png", "calamaties (2).png", "calamaties (3).png", "calamaties (4).png", "calamaties (5).png", "calamaties (6).png"},
}

// BossNameSprites maps a boss's display name (UPPERCASED) to a specific
// sprite filename. Lookup is case-insensitive. When a boss name matches,
// this specific sprite is used instead of the level-based CALAMITY bucket
// rotation — so each named boss always renders with the same distinct image.
//
// 💡 FIX (Item #9 — boss sprites not distinct): Expanded from 13 entries
// to cover ALL 40+ named bosses in classEncounters.js. Previously, bosses
// not in this map fell through to `calamaties[spriteIndex % 6]`, so many
// bosses shared the same sprite. Now every named boss has a deterministic,
// visually distinct sprite.
//
// Sprite pool (48 total in assets/rpgasset/enemies/):
//   calamaties (1-6).png      — 6 calamity-tier sprites (S/SS/SSS bosses, trials)
//   highlevelbosses (7-13).png — 7 high-level sprites (A/S bosses, some trials)
//   midlevelbosses (1-7).png   — 7 mid-level sprites (F-E-D-C-B bosses)
//   mutated (1-7).png          — 7 mutated-tier sprites (B/A bosses)
//   hybrides (1-7).png         — 7 hybrid-tier sprites (SS bosses)
//   earth (1-5).png            — 5 earth-themed sprites
//   fire (5,6,7,8,11).png      — 5 fire-themed sprites
//   ice (1,2,3).png            — 3 ice-themed sprites
//   water (4,6,7).png          — 3 water-themed sprites
//
// Mapping strategy:
//   - Rank bosses (S/SS/SSS) → calamaties sprites (visually epic)
//   - Mid bosses (F-E-D-C-B) → midlevelbosses sprites
//   - High bosses (A) → highlevelbosses sprites
//   - Trial bosses → assigned thematically (fire→fire, water→water, etc.)
//   - Special bosses (dragon→fire-themed, undead→mutated, etc.)
var BossNameSprites = map[string]string{
        // ═══ S/SS/SSS-rank dungeon bosses — use new boss_N sprites from SpriteAssets ═══
        "PRIMORDIAL CHAOS":       "boss_0_N.png",
        "ELDER CHAOS":            "boss_1_N.png",
        "VOID TITAN":             "boss_2_N.png",
        "ABYSSAL GOD":            "boss_3_N.png",
        "MUTATION PRIME":         "boss_4_N.png",
        "ELEMENTAL ARCHON":       "boss_5_N.png",

        // ═══ Mid-level bosses — use new boss sprites ═══
        "THE INFECTED COLOSSUS":  "boss_6_N.png",
        "INFECTED COLOSSUS":      "boss_6_N.png",
        "CORRUPTED GUARDIAN":     "boss_7_N.png",
        "STONE HULK":             "boss_9_N.png",
        "CRYSTAL CORRUPTED":      "boss_10_N.png",
        "EARTH WARDEN":           "boss_11_N.png",
        "FROST GHOUL":            "boss_12_N.png",
        "GLACIAL BEAST":          "boss_13_N.png",

        // ═══ High-level bosses — use new boss sprites + old highlevelbosses ═══
        "MAGMA BRUTE":            "boss_0_S.png",
        "HELLFIRE DEMON":         "boss_1_S.png",
        "ABYSSAL HORROR":         "boss_2_S.png",
        "TSUNAMI WALKER":         "boss_3_S.png",
        "BLIZZARD WRAITH":        "boss_4_S.png",
        "GRAVEYARD LORD":         "boss_5_S.png",
        "SHADOW LORD":            "boss_6_S.png",

        // ═══ Dragon bosses ═══
        "IGNEEL THE FIRE KING":   "boss_7_S.png",
        "ANCIENT DRAGON":         "boss_7_S.png",
        "ETERNAL DRAGON":         "boss_9_S.png",
        "ELDER FLAME":            "boss_10_S.png",

        // ═══ Trial bosses ═══
        "ARCANE SENTINEL":        "boss_11_S.png",
        "LICH KING":              "boss_12_S.png",
        "SHADOW STALKER":         "boss_13_S.png",
        "VOID ASSASSIN":          "highlevelbosses (9).png",
        "IRON BODY GRANDMASTER":  "midlevelbosses (3).png",
        "ANCIENT WURM":           "boss_7_S.png",
        "SOUL EATER":             "mutated (3).png",
        "ABYSSAL WHISPER":        "boss_3_S.png",
        "ELEMENTAL PRIMORDIAL":   "boss_5_N.png",
        "PRIME ELEMENT":          "boss_5_N.png",
        "VOID NECROMANCER":       "mutated (5).png",
        "CHRONOS WARDEN":         "boss_4_S.png",
        "TIME EATER":             "mutated (7).png",
        "HEAVENLY GUARDIAN":      "boss_3_N.png",
        "SERAPHIM PRIME":         "boss_0_S.png",
        "FOREST ANCESTOR":        "midlevelbosses (5).png",
        "GAIA SENTINEL":          "midlevelbosses (5).png",
        "GOLDEN GOLEM":           "midlevelbosses (3).png",
        "TREASURE HOARDER":       "boss_7_S.png",
        "SOUND REAPER":           "mutated (4).png",
        "MAESTRO OF VOID":        "boss_2_N.png",
        "CLOCKWORK TITAN":        "boss_0_S.png",
        "MECH GOD":               "boss_3_S.png",
        "DEMON LORD":             "boss_4_S.png",
        "PRIMORDIAL EVIL":        "boss_0_N.png",
        "LEVIATHAN":              "boss_3_N.png",
        "LEVIATHAN SPAWN ALPHA":  "boss_3_N.png",
        "INFERNAL OVERLORD":      "boss_1_S.png",
        "PRIMORDIAL FLAME":       "boss_7_S.png",
        "PERMAFROST TITAN":       "boss_4_S.png",
}

func GetCharacterSpritePath(class string, index int, assetsPath string) string {
        list, ok := CharacterSprites[class]
        if !ok {
                list = CharacterSprites["FIGHTER"]
        }
        filename := list[index%len(list)]
        return filepath.Join(assetsPath, "rpgasset", "characters", filename)
}

// GetEnemySpritePath picks the sprite PNG for an enemy.
//
// Signature change: now accepts `name` as the first parameter so named
// bosses (ELDER CHAOS, VOID TITAN, ABYSSAL GOD, IGNEEL THE FIRE KING, etc.)
// can be mapped to specific, deterministic sprites via BossNameSprites.
// Falls back to the original level-based bucket rotation when no name match
// is found (so unnamed / generic bosses still render correctly).
//
// The JS bot already sends enemy.name in the combat payload — it was just
// being ignored. Now it's the primary lookup key for bosses.
func GetEnemySpritePath(name string, level int, index int, isBoss bool, assetsPath string) string {
        var filename string
        if isBoss {
                // 1) Name-based lookup first (case-insensitive). If this boss has
                //    a specific sprite mapped in BossNameSprites, use it — no
                //    level/index rotation, so the boss always renders identically.
                if name != "" {
                        if specific, ok := BossNameSprites[strings.ToUpper(strings.TrimSpace(name))]; ok && specific != "" {
                                filename = specific
                                return filepath.Join(assetsPath, "rpgasset", "enemies", filename)
                        }
                }
                // 2) Fallback: level-based bucket rotation (original behavior)
                var list []string
                if level <= 60 {
                        list = BossSprites["MID_BOSSES"]
                } else if level <= 90 {
                        list = BossSprites["HIGH_BOSSES"]
                } else {
                        list = BossSprites["CALAMITY"]
                }
                filename = list[index%len(list)]
        } else {
                var list []string
                if level <= 10 {
                        list = EnemySprites["FIRE_LOW"]
                } else if level <= 20 {
                        list = EnemySprites["WATER_LOW"]
                } else if level <= 30 {
                        list = EnemySprites["EARTH_MID"]
                } else if level <= 40 {
                        list = EnemySprites["ICE_MID"]
                } else if level <= 50 {
                        list = EnemySprites["FIRE_HIGH"]
                } else if level <= 60 {
                        list = EnemySprites["WATER_HIGH"]
                } else if level <= 70 {
                        list = EnemySprites["EARTH_HIGH"]
                } else if level <= 80 {
                        list = EnemySprites["MUTATED"]
                } else if level <= 90 {
                        list = EnemySprites["HYBRID"]
                } else {
                        list = EnemySprites["FIRE_ELITE"]
                }
                filename = list[index%len(list)]
        }

        if filename == "" {
                filename = "fire (5).png"
        }

        return filepath.Join(assetsPath, "rpgasset", "enemies", filename)
}

func GetEnvironmentPath(bgName string, assetsPath string) string {
        if bgName == "" {
                bgName = "forest1.png"
        }
        return filepath.Join(assetsPath, "rpgasset", "environment", bgName)
}

// ─────────────────────────────────────────────────────────────
// 💡 Summoner System — Summon Sprites
// ─────────────────────────────────────────────────────────────
// Maps summon species to sprite filenames.
// Checks summons/digimon/ first (cached Digimon API sprites),
// then summons/ (SD sprites like Ifrit/Leviathan/Shiva),
// then summons/retromon/ (Retromon monster sprites),
// then falls back to enemy sprites.

var SummonSprites = map[string]string{
        // 💡 NEW: Sparklinlabs monster sprites (primary summon assets now)
        // These are in summons/sparklinlabs/
        "bat":             "bat.png",
        "boar":            "boar.png",
        "chest":           "chest.png",
        "dino":            "dino.png",
        "dragon":          "dragon.png",
        "ghost":           "ghost.png",
        "giant":           "giant.png",
        "mimic":           "mimic.png",
        "mushroom":        "mushroom.png",
        "octopus":         "octopus.png",
        "reptile":         "reptile.png",
        "slime":           "slime.png",
        "snake":           "snake.png",
        "yeti":            "yeti.png",

        // 💡 NEW: Grand Inventor torrent summons (space-shooter ships)
        "ship_cruiser":    "ship_cruiser.png",
        "ship_fighter":    "ship_fighter.png",
        "ship_squid":      "ship_squid.png",

        // 💡 Starter summons (use sparklinlabs sprites)
        "stoneguard":      "giant.png",
        "emberdrake":      "dragon.png",
        "mistwisp":        "ghost.png",
        "bloompixie":      "mushroom.png",

        // 💡 Evolved starters
        "iron_sentinel":       "giant.png",
        "mountain_titan":      "giant.png",
        "flare_wyrm":          "dragon.png",
        "infernal_dragon":     "dragon.png",
        "frost_spectre":       "ghost.png",
        "abyssal_phantom":     "ghost.png",
        "blossom_sylph":       "mushroom.png",
        "world_tree_spirit":   "mushroom.png",

        // 💡 UNCHANGED: Necromancer summons (keep existing assets)
        "skeleton":        "mutated (1).png",
        "skeleton_knight": "mutated (3).png",
        "lich_minion":     "mutated (4).png",
        "imp":             "hybrides (1).png",
        "void_walker":     "hybrides (3).png",
        "flame_elemental": "fire (5).png",
        "frost_elemental": "ice (1).png",
        "storm_elemental": "ice (3).png",
        "wolf":            "mutated (5).png",
        "bear":            "mutated (2).png",
        "turret_mk1":      "earth (1).png",
        "cannon_turret":   "earth (3).png",
        "wyrmling":        "highlevelbosses (7).png",
        "juvenile_dragon": "highlevelbosses (9).png",
        "ifrit_fire":      "ifrit_fire.png",
        "ifrit_nofire":    "ifrit_nofire.png",
        "leviathan":       "leviathan.png",
        "shiva_full":      "shiva_full.png",
        "shiva_ice":       "shiva_ice.png",
}

// GetSummonSpritePath returns the sprite file path for a summon species.
// Checks multiple folders in priority order:
// 1. summons/sparklinlabs/ (NEW: sparklinlabs monster sprites)
// 2. summons/digimon/ (cached Digimon API sprites, transparent PNGs)
// 3. summons/ (SD sprites like Ifrit/Leviathan/Shiva)
// 4. summons/retromon/ (Retromon monster sprites)
// 5. enemies/ (fallback to enemy sprite via SummonSprites map)
func GetSummonSpritePath(species string, assetsPath string) string {
        species = strings.ToLower(strings.TrimSpace(species))

        // 0. Check sparklinlabs (NEW: primary summon sprites)
        sparkPath := filepath.Join(assetsPath, "rpgasset", "summons", "sparklinlabs", species+".png")
        if fileExists(sparkPath) {
                return sparkPath
        }

        // 1. Check digimon cache
        digimonPath := filepath.Join(assetsPath, "rpgasset", "summons", "digimon", species+".png")
        if fileExists(digimonPath) {
                return digimonPath
        }

        // 2. Check SD summons
        sdPath := filepath.Join(assetsPath, "rpgasset", "summons", species+".png")
        if fileExists(sdPath) {
                return sdPath
        }

        // 3. Check retromon
        retromonPath := filepath.Join(assetsPath, "rpgasset", "summons", "retromon", species+".png")
        if fileExists(retromonPath) {
                return retromonPath
        }

        // 4. Fallback to SummonSprites map (which may point to sparklinlabs or enemies)
        filename, ok := SummonSprites[species]
        if !ok || filename == "" {
                filename = "slime.png"
        }
        // Check if the mapped file is in sparklinlabs first
        sparkFallback := filepath.Join(assetsPath, "rpgasset", "summons", "sparklinlabs", filename)
        if fileExists(sparkFallback) {
                return sparkFallback
        }
        return filepath.Join(assetsPath, "rpgasset", "enemies", filename)
}
