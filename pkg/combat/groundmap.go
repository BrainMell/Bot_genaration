package combat

import (
        _ "embed"
        "encoding/json"
        "log"
        "math"
        "sync"
)

// ═══════════════════════════════════════════════════════════════════════════
// PER-BACKGROUND GROUND MAP (2026-09-20)
// ═══════════════════════════════════════════════════════════════════════════
// Vision-measured standable-ground mapping for every battle background.
//
// PROBLEM: the slot tables in layout.go hardcode one depth ladder
// (feet Y = 440/390/340/290) for ALL backgrounds. On backgrounds whose
// ground starts lower (env caves: stone floor ≈ Y 410-465) back rows
// stood in mid-air over the lava/pool; on beaches (spark_5/6, ground
// ≈ Y 300-310) the deepest rows floated above the horizon.
//
// FIX: groundmap.json holds, per background, the measured ground start
// (canvas space, 1024x687, AFTER imaging.Fill cover+center-crop and the
// 31% dark overlay - i.e. exactly what the renderer paints on) and a
// recomputed depth ladder whose every row still lands on standable
// ground. groundAdjustedY() remaps each legacy slot Y onto the new
// ladder, preserving the zigzag formation (X untouched, relative depth
// preserved via linear scale for wrap-shifted rows).
//
// Measurements: vision pass over canvas-space renders + annotated
// verification overlays (sprite silhouettes at the mapped slots).
// Verification overlays: qa/groundmap/annotated/*.png
//
// Safety: if the JSON is missing/corrupt or a background has no entry,
// every helper returns its input unchanged → legacy behaviour.

type groundStart struct {
        Player int `json:"player"`
        Enemy  int `json:"enemy"`
}

type groundEntry struct {
        GroundStartY groundStart `json:"groundStartY"`
        PlayerRows   []float64   `json:"playerRows"`
        EnemyRows    []float64   `json:"enemyRows"`
        SummonRows   []float64   `json:"summonRows"`
        PvpFeetY     float64     `json:"pvpFeetY"`
        Note         string      `json:"note"`
}

type groundDefaults struct {
        PlayerRows []float64 `json:"playerRows"`
        EnemyRows  []float64 `json:"enemyRows"`
        SummonRows []float64 `json:"summonRows"`
        PvpFeetY   float64   `json:"pvpFeetY"`
}

type groundMapFile struct {
        Version     int                     `json:"version"`
        Defaults    groundDefaults          `json:"defaults"`
        Backgrounds map[string]*groundEntry `json:"backgrounds"`
}

//go:embed groundmap.json
var groundMapJSON []byte

var (
        groundMapOnce sync.Once
        groundMap     *groundMapFile
)

func loadGroundMap() *groundMapFile {
        groundMapOnce.Do(func() {
                groundMap = &groundMapFile{}
                if err := json.Unmarshal(groundMapJSON, groundMap); err != nil {
                        log.Printf("[groundmap] ⚠️ parse failed (%v) - legacy slot tables in effect", err)
                        groundMap = &groundMapFile{}
                }
                if len(groundMap.Backgrounds) == 0 {
                        log.Printf("[groundmap] ⚠️ no background entries - legacy slot tables in effect")
                } else {
                        log.Printf("[groundmap] loaded %d background ground entries (v%d)",
                                len(groundMap.Backgrounds), groundMap.Version)
                }
        })
        return groundMap
}

// groundEntryFor returns the entry for a background filename ("" safe).
func groundEntryFor(bgFilename string) *groundEntry {
        gm := loadGroundMap()
        if gm == nil || len(gm.Backgrounds) == 0 || bgFilename == "" {
                return nil
        }
        return gm.Backgrounds[bgFilename]
}

// Legacy depth-ladder anchors = the layout.go slot table Y values.
// playerRows/enemyRows anchors (PvE): 440 front, 390, 340, 290 deepest.
// summonRows anchors: 345 companion row, 305 deep row.
var (
        legacySideAnchors   = [4]float64{440, 390, 340, 290}
        legacySummonAnchors = [2]float64{345, 305}
)

// groundAdjustedY remaps a legacy slot feet-Y onto the background's
// measured ground rows.
//
//   - side "player"/"enemy": legacy anchors 440/390/340/290
//   - side "summon":         legacy anchors 345/305
//   - Y below the deepest anchor (slotFor wrap shifts, -30/row) is
//     extrapolated linearly so wrapped rows keep their extra depth.
//   - No entry / parse failure → y unchanged (legacy behaviour).
func groundAdjustedY(bgFilename, side string, y float64) float64 {
        entry := groundEntryFor(bgFilename)
        if entry == nil {
                return y
        }
        var rows []float64
        var anchors []float64
        n := 0
        switch side {
        case "player":
                rows, n = entry.PlayerRows, len(entry.PlayerRows)
                anchors = legacySideAnchors[:]
        case "enemy":
                rows, n = entry.EnemyRows, len(entry.EnemyRows)
                anchors = legacySideAnchors[:]
        case "summon":
                rows, n = entry.SummonRows, len(entry.SummonRows)
                anchors = legacySummonAnchors[:]
        default:
                return y
        }
        if n == 0 {
                return y
        }
        // Exact anchor bands, partitioned at midpoints between adjacent
        // anchors (slot Ys always equal an anchor exactly; midpoint splits
        // keep ±1px sprite jitter safe). Deeper = smaller Y.
        deepest := anchors[n-1]
        if y < deepest-1 {
                // Wrapped row (slotFor shift, -30/row): extrapolate linearly so
                // relative depth survives (scale = canvas-px per legacy-px).
                top := anchors[0]
                span := deepest - top
                if span <= 0 {
                        return rows[n-1]
                }
                scale := (rows[n-1] - rows[0]) / span
                return rows[n-1] + (y-deepest)*scale
        }
        for i := 0; i < n; i++ {
                lower := math.Inf(-1)
                if i < n-1 {
                        lower = (anchors[i] + anchors[i+1]) / 2
                }
                upper := math.Inf(1)
                if i > 0 {
                        upper = (anchors[i-1] + anchors[i]) / 2
                }
                if y > lower && y <= upper {
                        return rows[i]
                }
        }
        return rows[0]
}

// groundPvpFeetY returns the per-background PvP feet line (default 455).
func groundPvpFeetY(bgFilename string) float64 {
        entry := groundEntryFor(bgFilename)
        if entry == nil || entry.PvpFeetY <= 0 {
                return 455
        }
        return entry.PvpFeetY
}
