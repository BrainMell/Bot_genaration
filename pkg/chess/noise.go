package chess

import "math"

// Tiny deterministic value-noise toolkit for the procedural board textures
// (no assets, no RNG state: the same square always renders the same grain).

func hash2(ix, iy int, seed uint32) float64 {
	h := uint32(ix)*374761393 + uint32(iy)*668265263 + seed*2246822519
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h&0xffff) / 65535.0
}

func smooth(t float64) float64 { return t * t * (3 - 2*t) }

func vnoise(x, y float64, seed uint32) float64 {
	ix, iy := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := smooth(x-float64(ix)), smooth(y-float64(iy))
	a, b := hash2(ix, iy, seed), hash2(ix+1, iy, seed)
	c, d := hash2(ix, iy+1, seed), hash2(ix+1, iy+1, seed)
	return (a*(1-fx)+b*fx)*(1-fy) + (c*(1-fx)+d*fx)*fy
}

func fbm(x, y float64, oct int, seed uint32) float64 {
	sum, amp, norm := 0.0, 0.5, 0.0
	for i := 0; i < oct; i++ {
		sum += vnoise(x, y, seed+uint32(i)*101) * amp
		norm += amp
		x, y, amp = x*2.03, y*2.03, amp*0.5
	}
	return sum / norm
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }
