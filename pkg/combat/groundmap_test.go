package combat

import (
        "math"
        "testing"
)

// Tolerant float compare
func feq(a, b float64) bool { return math.Abs(a-b) < 0.001 }

func TestGroundAdjustedY_EnvCaveEnemyFront(t *testing.T) {
        // env1 enemy front row: legacy 440 stays 440 (front row already on floor edge)
        if got := groundAdjustedY("env1.png", "enemy", 440); !feq(got, 440) {
                t.Fatalf("env1 enemy front: want 440, got %v", got)
        }
}

func TestGroundAdjustedY_EnvCaveEnemyBackNoMoreLavaFloating(t *testing.T) {
        // env1 enemy deepest legacy band 290 -> platform ladder row 410 (v2 map)
        if got := groundAdjustedY("env1.png", "enemy", 290); !feq(got, 410) {
                t.Fatalf("env1 enemy deep row: want 410 (on platform), got %v", got)
        }
}

func TestGroundAdjustedY_BeachDeepRowOnSand(t *testing.T) {
        // spark_5 ground starts ~310: legacy 290 (above shoreline) -> 314 (wet sand)
        if got := groundAdjustedY("spark_5.png", "player", 290); !feq(got, 314) {
                t.Fatalf("spark_5 deep row: want 314 (on sand), got %v", got)
        }
}

func TestGroundAdjustedY_SummonRows(t *testing.T) {
        // spark_15 arena: summon anchors 345/305 -> 364/324 (both on tiles >= 315)
        if got := groundAdjustedY("spark_15.png", "summon", 345); !feq(got, 364) {
                t.Fatalf("spark_15 summon row0: want 364, got %v", got)
        }
        if got := groundAdjustedY("spark_15.png", "summon", 305); !feq(got, 324) {
                t.Fatalf("spark_15 summon row1: want 324, got %v", got)
        }
}

func TestGroundAdjustedY_WrapExtrapolation(t *testing.T) {
        // slotFor wrap shifts legacy 290 by -30 -> 260; env1 must extrapolate
        // deeper than the deepest row but never re-enter the sky.
        got := groundAdjustedY("env1.png", "player", 260)
        deep := groundAdjustedY("env1.png", "player", 290)
        if got > deep {
                t.Fatalf("wrapped row must be deeper (higher Y == closer): got %v, deep %v", got, deep)
        }
}

func TestGroundAdjustedY_FallbackUnknownBg(t *testing.T) {
        // Unknown background -> unchanged legacy Y.
        if got := groundAdjustedY("does_not_exist.png", "player", 340); !feq(got, 340) {
                t.Fatalf("unknown bg must fall through: got %v", got)
        }
        if got := groundAdjustedY("", "enemy", 290); !feq(got, 290) {
                t.Fatalf("empty bg must fall through: got %v", got)
        }
}

func TestGroundPvpFeetY(t *testing.T) {
        if got := groundPvpFeetY("spark_15.png"); !feq(got, 455) {
                t.Fatalf("spark_15 pvp feet: want 455, got %v", got)
        }
        if got := groundPvpFeetY("unknown.png"); !feq(got, 455) {
                t.Fatalf("unknown bg pvp feet default: want 455, got %v", got)
        }
}

func TestGroundMapCoversKnownBattleBackgrounds(t *testing.T) {
        required := []string{
                "spark_1.png", "spark_2.png", "spark_3.png", "spark_4.png",
                "spark_5.png", "spark_6.png", "spark_7.png", "spark_8.png",
                "spark_10.png", "spark_15.png",
                "spark_1-night.png", "spark_2-night.png", "spark_3-night.png",
                "env1.png", "env2.png", "env3.png",
                "forest.png", "background1.png", "ice.png", "background3.png",
                "sand.png", "background2.png",
        }
        for _, bg := range required {
                e := groundEntryFor(bg)
                if e == nil {
                        t.Errorf("ground map missing entry for %s", bg)
                        continue
                }
                if len(e.PlayerRows) != 4 || len(e.EnemyRows) != 4 || len(e.SummonRows) != 2 {
                        t.Errorf("%s: bad row table P=%d E=%d S=%d", bg, len(e.PlayerRows), len(e.EnemyRows), len(e.SummonRows))
                }
                // Shadow-anchored grounding: every row's feet (+12px shadow ellipse,
                // ±12px perspective tolerance) must reach the measured ground start.
                gs := float64(e.GroundStartY.Player)
                for i, y := range e.PlayerRows {
                        if y+24 < gs {
                                t.Errorf("%s playerRows[%d]=%v not shadow-anchored on groundStart %v", bg, i, y, gs)
                        }
                }
                gse := float64(e.GroundStartY.Enemy)
                for i, y := range e.EnemyRows {
                        if y+24 < gse {
                                t.Errorf("%s enemyRows[%d]=%v not shadow-anchored on groundStart %v", bg, i, y, gse)
                        }
                }
                // Front rows must respect the PvE panel cap.
                if e.PlayerRows[0] > 440 || e.EnemyRows[0] > 440 {
                        t.Errorf("%s front row breaches PvE panel cap 440", bg)
                }
        }
}

// ═══════════════════════════════════════════════════════════════════════════
// FORMATION LAYER TESTS (2026-09-20) — count/mass-aware arrangement
// ═══════════════════════════════════════════════════════════════════════════

// Cave platforms: a 3-player party must land on the measured 2-row platform
// ladder (440/396), inside the platform's X zone — never over the lava.
func TestGroundPlanFormation_CavePartyOnPlatform(t *testing.T) {
        widths := []float64{120, 110, 130}
        slots := groundPlanFormation("env1.png", "player", widths)
        if slots == nil {
                t.Fatalf("env1 player formation missing for 3 players")
        }
        if len(slots) != 3 {
                t.Fatalf("want 3 slots, got %d", len(slots))
        }
        for i, s := range slots {
                if s.Y != 440 && s.Y != 396 {
                        t.Errorf("slot %d feet Y=%v off the platform ladder (440/396)", i, s.Y)
                }
                if s.X < 160 || s.X > 470 {
                        t.Errorf("slot %d X=%v outside standable platform zone [160,470]", i, s.X)
                }
        }
        // zigzag: with 2-per-front-row the third wraps to the deeper row
        if slots[2].Y != 396 {
                t.Errorf("third sprite should wrap to row 396, got %v", slots[2].Y)
        }
}

// Wide fields keep the legacy look: 6 players = 2 per row, 3 rows.
func TestGroundPlanFormation_WideZigzag(t *testing.T) {
        widths := []float64{120, 120, 120, 120, 120, 120}
        slots := groundPlanFormation("spark_1.png", "player", widths)
        if slots == nil {
                t.Fatalf("spark_1 player formation missing for 6 players")
        }
        perRow := map[float64]int{}
        for _, s := range slots {
                perRow[s.Y]++
                if s.X < 40 || s.X > 380 {
                        t.Errorf("slot X=%v outside wide player zone [40,380]", s.X)
                }
        }
        for y, n := range perRow {
                if n != 2 {
                        t.Errorf("row Y=%v holds %d sprites, want 2 (legacy zigzag)", y, n)
                }
        }
        if len(perRow) != 3 {
                t.Errorf("6 players should use 3 rows, got %d", len(perRow))
        }
}

// Dense packs (10 enemies) must still fit a wide field — congested but grounded.
func TestGroundPlanFormation_DensePackFits(t *testing.T) {
        widths := make([]float64, 10)
        for i := range widths {
                widths[i] = 155
        }
        slots := groundPlanFormation("spark_1.png", "enemy", widths)
        if slots == nil {
                t.Fatalf("10-enemy pack must dense-fit spark_1, got no plan")
        }
        if len(slots) != 10 {
                t.Fatalf("want 10 slots, got %d", len(slots))
        }
        for i, s := range slots {
                if s.X < 640 || s.X > 990 {
                        t.Errorf("slot %d X=%v outside enemy zone", i, s.X)
                }
                if s.Y != 440 && s.Y != 392 && s.Y != 346 && s.Y != 302 {
                        t.Errorf("slot %d Y=%v off the measured row ladder", i, s.Y)
                }
        }
}

// Boss + minion in a cave: tight fit, no overflow (boss mass absorbs the row).
func TestGroundPlanFormation_BossPlusMinionCave(t *testing.T) {
        slots := groundPlanFormation("env1.png", "enemy", []float64{205, 155})
        if slots == nil {
                t.Fatalf("boss+minion must fit env1 (dense pass), got no plan")
        }
        for i, s := range slots {
                if s.Y != 440 && s.Y != 396 {
                        t.Errorf("slot %d Y=%v off platform ladder", i, s.Y)
                }
        }
        if slots[0].Y == slots[1].Y && math.Abs(slots[0].X-slots[1].X) < 100 {
                t.Errorf("boss and minion fully stacked: %v/%v", slots[0].X, slots[1].X)
        }
}

// Unknown background -> no formation (legacy tables).
func TestGroundPlanFormation_LegacyFallback(t *testing.T) {
        if got := groundPlanFormation("does_not_exist.png", "player", []float64{120}); got != nil {
                t.Errorf("unknown bg must return nil (legacy), got %v", got)
        }
}

// Overflow probes: caves swap, wide fields stay.
func TestGroundOverflowIfNeeded(t *testing.T) {
        // 6v6 (+1 summon) in a lava cave -> swap to crimson plains
        if got := groundOverflowIfNeeded("env1.png", 6, 6, 1, false); got != "spark_2.png" {
                t.Errorf("env1 6v6 should overflow to spark_2, got %q", got)
        }
        if got := groundOverflowIfNeeded("env2.png", 6, 6, 0, false); got != "spark_7.png" {
                t.Errorf("env2 6v6 should overflow to spark_7, got %q", got)
        }
        if got := groundOverflowIfNeeded("env3.png", 6, 6, 0, false); got != "spark_4.png" {
                t.Errorf("env3 6v6 should overflow to spark_4, got %q", got)
        }
        // small skirmish stays in the cave
        if got := groundOverflowIfNeeded("env1.png", 3, 3, 1, false); got != "" {
                t.Errorf("env1 3v3+1 must stay, got %q", got)
        }
        // boss + 1 minion stays (dense pass fits)
        if got := groundOverflowIfNeeded("env1.png", 2, 2, 0, true); got != "" {
                t.Errorf("env1 boss+minion must stay, got %q", got)
        }
        // wide fields never swap
        if got := groundOverflowIfNeeded("spark_1.png", 6, 6, 2, false); got != "" {
                t.Errorf("spark_1 6v6 must stay, got %q", got)
        }
        if got := groundOverflowIfNeeded("spark_15.png", 4, 4, 0, false); got != "" {
                t.Errorf("spark_15 4v4 must stay, got %q", got)
        }
}

// PvE capacity probe counts visible combatants like the renderer does.
func TestCountVisibleCombatants(t *testing.T) {
        req := CombatRequest{}
        req.Players = []Player{
                {CurrentHP: 0},   // main player always renders
                {CurrentHP: 100}, // alive
                {CurrentHP: 0},   // dead-and-unseen: skipped
        }
        req.Enemies = []Enemy{
                {CurrentHP: 500, IsBoss: true},
                {CurrentHP: 0, JustDied: true}, // renders tinted: counted
                {CurrentHP: 0},                 // skipped
        }
        req.Summons = []Summon{{CurrentHP: 200}, {CurrentHP: 0}}
        p, e, s, boss := countVisibleCombatants(req)
        if p != 2 || e != 2 || s != 1 || !boss {
                t.Errorf("counts p=%d e=%d s=%d boss=%v, want 2/2/1/true", p, e, s, boss)
        }
}

// Cave-safe clamps for fixed PvP positions.
func TestGroundClampX(t *testing.T) {
        if got := groundClampX("env1.png", "player", 100); !feq(got, 160) {
                t.Errorf("env1 player X=100 must clamp to 160, got %v", got)
        }
        if got := groundClampX("env1.png", "enemy", 800); !feq(got, 800) {
                t.Errorf("env1 enemy X=800 already inside zone, got %v", got)
        }
        if got := groundClampX("spark_1.png", "player", 220); !feq(got, 220) {
                t.Errorf("wide zone must not move inside X, got %v", got)
        }
        if got := groundClampX("unknown.png", "player", 220); !feq(got, 220) {
                t.Errorf("unknown bg must not clamp, got %v", got)
        }
}

// Every formation zone must be standable: rows above ground start (shadow
// tolerance) and below the panel cap; zones ordered; overflow targets exist.
func TestFormationZonesStandable(t *testing.T) {
        gm := loadGroundMap()
        for bg, e := range gm.Backgrounds {
                f := e.Formation
                zones := map[string]groundZone{
                        "player": f.Player, "enemy": f.Enemy, "summon": f.Summon,
                }
                for side, z := range zones {
                        if z.X1 <= z.X0 {
                                t.Errorf("%s %s zone inverted (%v..%v)", bg, side, z.X0, z.X1)
                        }
                        if len(z.Rows) == 0 {
                                t.Errorf("%s %s zone has no rows", bg, side)
                        }
                        gs := float64(e.GroundStartY.Player)
                        if side == "enemy" {
                                gs = float64(e.GroundStartY.Enemy)
                        }
                        for i, y := range z.Rows {
                                if y > 440 {
                                        t.Errorf("%s %s rows[%d]=%v breaches panel cap 440", bg, side, i, y)
                                }
                                if y+24 < gs {
                                        t.Errorf("%s %s rows[%d]=%v floats above ground start %v", bg, side, i, y, gs)
                                }
                        }
                        for i := 1; i < len(z.Rows); i++ {
                                if z.Rows[i] >= z.Rows[i-1] {
                                        t.Errorf("%s %s rows not strictly deepening: %v -> %v", bg, side, z.Rows[i-1], z.Rows[i])
                                }
                        }
                }
                if ob := f.OverflowBackground; ob != "" {
                        if _, ok := gm.Backgrounds[ob]; !ok {
                                t.Errorf("%s overflow target %q has no ground map entry", bg, ob)
                        }
                        if gm.Backgrounds[ob].Formation.SizeClass == "s" {
                                t.Errorf("%s overflow target %q is itself a small stage", bg, ob)
                        }
                }
        }
}
