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
        GroundStartY groundStart      `json:"groundStartY"`
        PlayerRows   []float64        `json:"playerRows"`
        EnemyRows    []float64        `json:"enemyRows"`
        SummonRows   []float64        `json:"summonRows"`
        PvpFeetY     float64          `json:"pvpFeetY"`
        Note         string           `json:"note"`
        Formation    groundFormation  `json:"formation"`
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

// ═══════════════════════════════════════════════════════════════════════════
// FORMATION LAYER (2026-09-20) — count/mass-aware arrangement + capacity
// ═══════════════════════════════════════════════════════════════════════════
// Owner ruling: the ARRANGEMENT of players/enemies AND the choice of
// background must react to the number and mass (sprite width) of the
// combatants. Small stages (env1/2/3 cave platforms) cannot host a large
// party — when a formation does not fit its side's standable zone, the
// renderer swaps to the stage's declared overflow arena.
//
// groundPlanFormation packs sprites (real post-resize widths) into the
// background's per-side zone: front row first, greedy left→right, wrap to
// the next measured row when the row is full, zigzag stagger on odd rows
// so back-row sprites peek through the gaps. Three comfort passes
// (roomy → tight → dense) before the squeeze fallback, mirroring how real
// battle lines compress instead of floating.

type groundZone struct {
        X0   float64   `json:"x0"`
        X1   float64   `json:"x1"`
        Rows []float64 `json:"rows"`
}

type groundFormation struct {
        SizeClass          string     `json:"sizeClass"`
        Player             groundZone `json:"playerZone"`
        Enemy              groundZone `json:"enemyZone"`
        Summon             groundZone `json:"summonZone"`
        OverflowBackground string     `json:"overflowBackground"`
}

// groundFormationZone returns the side's zone ("player"/"enemy"/"summon").
func (e *groundEntry) groundFormationZone(side string) *groundZone {
        if e == nil {
                return nil
        }
        f := e.Formation
        switch side {
        case "player":
                return &f.Player
        case "enemy":
                return &f.Enemy
        case "summon":
                return &f.Summon
        }
        return nil
}

// packRows greedily fills rows front→deep within the zone.
// spread scales center-to-center pitch between neighbours (1.0 = edge
// breathing room via gap; <1 lets sprites overlap for dense packs).
// Returns per-row X centers, or ok=false when n does not fit.
func packRows(z groundZone, widths []float64, spread, gap float64) ([][]float64, bool) {
        n := len(widths)
        if n == 0 {
                return nil, true
        }
        rows := make([][]float64, 0, len(z.Rows))
        i := 0
        for r := 0; r < len(z.Rows) && i < n; r++ {
                var row []float64
                x := z.X0 + widths[i]/2
                for i < n {
                        if x+widths[i]/2 > z.X1+0.01 {
                                break // row full
                        }
                        row = append(row, x)
                        prevW := widths[i]
                        i++
                        if i < n {
                                x += (prevW+widths[i])/2*spread + gap
                        }
                }
                if len(row) == 0 {
                        // single sprite wider than the whole zone: clamp center
                        row = append(row, (z.X0+z.X1)/2)
                        i++
                }
                rows = append(rows, row)
        }
        if i < n {
                return nil, false
        }
        return rows, true
}

// squeezeRows last-resort even spread: splits n sprites across all rows
// (front rows take the remainder) and lin-spaces centers edge-to-edge.
// Always succeeds; sprites may overlap heavily but stay grounded and inside.
func squeezeRows(z groundZone, widths []float64) [][]float64 {
        n := len(widths)
        R := len(z.Rows)
        if R == 0 || n == 0 {
                return nil
        }
        base := n / R
        extra := n % R
        rows := make([][]float64, 0, R)
        idx := 0
        for r := 0; r < R && idx < n; r++ {
                k := base
                if r < extra {
                        k++
                }
                if k <= 0 {
                        break
                }
                row := make([]float64, 0, k)
                if k == 1 {
                        row = append(row, (z.X0+z.X1)/2)
                } else {
                        first := z.X0 + widths[idx]/2
                        last := z.X1 - widths[idx+k-1]/2
                        if last < first {
                                last = first
                        }
                        step := (last - first) / float64(k-1)
                        for j := 0; j < k; j++ {
                                row = append(row, first+float64(j)*step)
                        }
                }
                idx += k
                rows = append(rows, row)
        }
        return rows
}

// staggerRows shifts odd rows right by half the previous row's first pitch
// so back-row sprites peek through front-row gaps (legacy zigzag look).
// The shift is clamped so every sprite stays inside the zone.
func staggerRows(z groundZone, rows [][]float64, widths []float64) {
        // recompute flat index for widths lookup
        idx := 0
        for r := 1; r < len(rows); r++ {
                prev := rows[r-1]
                cur := rows[r]
                if len(cur) == 0 || len(prev) == 0 {
                        idx += len(cur)
                        continue
                }
                shift := 0.0
                if len(prev) >= 2 {
                        shift = (prev[1] - prev[0]) / 2
                } else {
                        shift = 24 // single-sprite row: modest nudge
                }
                if shift > 48 {
                        shift = 48
                }
                // clamp: last sprite of this row must stay inside
                lastIdx := idx + len(cur) - 1
                room := z.X1 - widths[lastIdx]/2 - cur[len(cur)-1]
                if shift > room {
                        shift = room
                }
                if shift > 0 {
                        for j := range cur {
                                cur[j] += shift
                        }
                }
                idx += len(cur)
        }
}

// groundPlanFormation packs entity widths into the background's side zone.
// Returns nil when the background has no formation data (caller falls back
// to the legacy slot tables + groundAdjustedY).
func groundPlanFormation(bgFilename, side string, widths []float64) []Slot {
        return groundPlanWithPasses(bgFilename, side, widths,
                []struct{ spread, gap float64 }{{1.0, 14}, {0.62, 6}, {0.45, 0}})
}

// groundFitsRoomy reports whether the side fits at FULL comfort (single
// roomy pass). Used by the pre-render capacity probe: a stage that can only
// host a party by dense-packing it is too small for that party — the probe
// must demand a swap instead of silently crushing everyone together.
func groundFitsRoomy(bgFilename, side string, widths []float64) bool {
        if len(widths) == 0 {
                return true
        }
        entry := groundEntryFor(bgFilename)
        z := entry.groundFormationZone(side)
        if z == nil || len(z.Rows) == 0 || z.X1 <= z.X0 {
                return true // no formation data: never trigger a swap
        }
        _, ok := packRows(*z, widths, 1.0, 14)
        return ok
}

func groundPlanWithPasses(bgFilename, side string, widths []float64, passes []struct{ spread, gap float64 }) []Slot {
        entry := groundEntryFor(bgFilename)
        z := entry.groundFormationZone(side)
        if z == nil || len(z.Rows) == 0 || z.X1 <= z.X0 {
                return nil
        }
        if len(widths) == 0 {
                return []Slot{}
        }
        var rows [][]float64
        for _, pass := range passes {
                if got, ok := packRows(*z, widths, pass.spread, pass.gap); ok {
                        rows = got
                        break
                }
        }
        if rows == nil {
                rows = squeezeRows(*z, widths)
                if rows == nil {
                        return nil
                }
                return flattenSlots(rows, *z)
        }
        staggerRows(*z, rows, widths)
        return flattenSlots(rows, *z)
}

func flattenSlots(rows [][]float64, z groundZone) []Slot {
        var out []Slot
        for r, row := range rows {
                y := z.Rows[r]
                for _, x := range row {
                        out = append(out, Slot{X: x, Y: y})
                }
        }
        return out
}

// Unit widths for the pre-render capacity probe (before sprites load).
// Conservative-high on purpose: swap fires slightly early rather than late.
const (
        groundUnitPlayerW = 120.0
        groundUnitEnemyW  = 155.0
        groundUnitBossW   = 205.0
        groundUnitSummonW = 85.0
)

// groundOverflowIfNeeded probes whether the counted party fits the stage.
// Returns the overflow background filename when a side cannot fit
// ("" = stay). Side counts are VISIBLE (rendered) combatants.
func groundOverflowIfNeeded(bgFilename string, players, enemies, summons int, hasBoss bool) string {
        entry := groundEntryFor(bgFilename)
        if entry == nil || entry.Formation.OverflowBackground == "" {
                return ""
        }
        pw := make([]float64, players)
        for i := range pw {
                pw[i] = groundUnitPlayerW
        }
        ew := make([]float64, enemies)
        for i := range ew {
                ew[i] = groundUnitEnemyW
        }
        if hasBoss && len(ew) > 0 {
                ew[0] = groundUnitBossW
        }
        sw := make([]float64, summons)
        for i := range sw {
                sw[i] = groundUnitSummonW
        }
        // Probe at ROOMY comfort only: a stage that needs dense packing is
        // too small for this party. Empty sides never block the probe.
        if !groundFitsRoomy(bgFilename, "player", pw) ||
                !groundFitsRoomy(bgFilename, "enemy", ew) ||
                !groundFitsRoomy(bgFilename, "summon", sw) {
                return entry.Formation.OverflowBackground
        }
        return ""
}

// groundClampX keeps a fixed legacy X (PvP slots) inside the background's
// standable zone for the given side. Unknown bg → x unchanged.
func groundClampX(bgFilename, side string, x float64) float64 {
        entry := groundEntryFor(bgFilename)
        z := entry.groundFormationZone(side)
        if z == nil || z.X1 <= z.X0 {
                return x
        }
        if x < z.X0 {
                return z.X0
        }
        if x > z.X1 {
                return z.X1
        }
        return x
}

// countVisibleCombatants counts the entities the renderer will actually
// draw (main player always renders; others need HP or a just-died marker).
// Used by the pre-render capacity probe.
func countVisibleCombatants(req CombatRequest) (players, enemies, summons int, hasBoss bool) {
        for i, p := range req.Players {
                if p.CurrentHP <= 0 && i > 0 {
                        continue
                }
                players++
        }
        for _, e := range req.Enemies {
                if e.CurrentHP <= 0 && !e.JustDied {
                        continue
                }
                if e.IsBoss {
                        hasBoss = true
                }
                enemies++
        }
        for _, s := range req.Summons {
                if s.CurrentHP <= 0 && !s.JustDied {
                        continue
                }
                summons++
        }
        return
}
