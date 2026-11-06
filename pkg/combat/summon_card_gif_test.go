package combat

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"testing"
)

func TestSummonCardVFX(t *testing.T) {
	cases := map[string]string{
		"ifrit":     "Fire-bomb",
		"leviathan": "Lightning",
		"shiva":     "Holy VFX 02",
	}
	for cardID, want := range cases {
		got, ok := summonCardVFX(cardID)
		if !ok || got != want {
			t.Fatalf("summonCardVFX(%q) = (%q, %v), want (%q, true)", cardID, got, ok, want)
		}
	}
	if _, ok := summonCardVFX("unknown"); ok {
		t.Fatal("unknown summon card must not resolve to a VFX")
	}
}

func TestEncodeSummonCardGIFProducesAnimatedGIF(t *testing.T) {
	first := image.NewRGBA(image.Rect(0, 0, 24, 24))
	second := image.NewRGBA(image.Rect(0, 0, 24, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 24; x++ {
			first.Set(x, y, color.RGBA{R: 255, A: 255})
			second.Set(x, y, color.RGBA{B: 255, A: 255})
		}
	}
	data, err := encodeSummonCardGIF([]image.Image{first, second})
	if err != nil {
		t.Fatalf("encodeSummonCardGIF: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("GIF89a")) && !bytes.HasPrefix(data, []byte("GIF87a")) {
		t.Fatalf("output is not a GIF stream: %q", data[:min(6, len(data))])
	}
	decoded, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("gif.DecodeAll: %v", err)
	}
	if len(decoded.Image) != 2 {
		t.Fatalf("decoded frame count = %d, want 2", len(decoded.Image))
	}
	if len(decoded.Delay) != 2 || decoded.Delay[0] != 8 {
		t.Fatalf("GIF delays = %v, want two 8-centisecond frames", decoded.Delay)
	}
}
