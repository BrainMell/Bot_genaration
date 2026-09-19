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
        // env1 enemy deepest legacy band 290 -> must land on the platform (>= groundStart 412)
        got := groundAdjustedY("env1.png", "enemy", 290)
        if got < 412 {
                t.Fatalf("env1 enemy deep row floats over lava: got %v, want >= 412", got)
        }
}

func TestGroundAdjustedY_BeachDeepRowOnSand(t *testing.T) {
        // spark_5 ground starts ~300: legacy 290 (above horizon) must clamp to >= 302
        got := groundAdjustedY("spark_5.png", "player", 290)
        if got < 302 {
                t.Fatalf("spark_5 deep row above shoreline: got %v, want >= 302", got)
        }
}

func TestGroundAdjustedY_SummonRows(t *testing.T) {
        // spark_15 arena: summon anchors 345/305 -> 373/347 (both on tiles >= 330)
        if got := groundAdjustedY("spark_15.png", "summon", 345); !feq(got, 373) {
                t.Fatalf("spark_15 summon row0: want 373, got %v", got)
        }
        if got := groundAdjustedY("spark_15.png", "summon", 305); !feq(got, 347) {
                t.Fatalf("spark_15 summon row1: want 347, got %v", got)
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
                // For normal backgrounds rows are on open ground; for shallow-floor
                // stages (env caves) the 440 panel cap forces a flat ladder whose
                // shadows still anchor on the floor edge.
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
