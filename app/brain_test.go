package main

import (
	"image"
	"image/color"
	"testing"
	"time"
	"unsafe"
)

func TestAlternateAndPhone(t *testing.T) {
	b := NewBrain(DefaultConfig())
	t0 := time.Unix(0, 0)
	var got []Sprite
	for i := 0; i < 10; i++ { // slow typing: 300ms apart -> never "fast"
		a, ok := b.Key(t0.Add(time.Duration(i) * 300 * time.Millisecond))
		if !ok {
			t.Fatalf("press %d ignored", i)
		}
		got = append(got, a.Show)
	}
	want := []Sprite{SpLeft, SpRight, SpLeft, SpRight, SpLeft, SpRight, SpLeft, SpRight, SpLeft, SpPhone}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("press %d: got %v want %v (all %v)", i, got[i], want[i], got)
		}
	}
	// phone is sticky for PhoneHoldMs
	if _, ok := b.Key(t0.Add(2700*time.Millisecond + 500*time.Millisecond)); ok {
		t.Fatal("key during phone hold should be ignored")
	}
	// after a pause the next press starts with the left paw again
	a, _ := b.Key(t0.Add(10 * time.Second))
	if a.Show != SpLeft {
		t.Fatalf("after pause got %v", a.Show)
	}
}

func TestFast(t *testing.T) {
	b := NewBrain(DefaultConfig())
	t0 := time.Unix(0, 0)
	var last Action
	for i := 0; i < 7; i++ {
		last, _ = b.Key(t0.Add(time.Duration(i) * 100 * time.Millisecond))
	}
	if last.Show != SpFastA && last.Show != SpFastB {
		t.Fatalf("expected fast pose, got %v", last.Show)
	}
}

func TestStructSizes(t *testing.T) {
	checks := map[string][2]uintptr{
		"MSG": {unsafe.Sizeof(msgT{}), 48}, "WNDCLASSEXW": {unsafe.Sizeof(wndClassEx{}), 80},
		"NOTIFYICONDATAW": {unsafe.Sizeof(notifyIconData{}), 976}, "KBDLLHOOKSTRUCT": {unsafe.Sizeof(kbdllHook{}), 24},
		"MSLLHOOKSTRUCT": {unsafe.Sizeof(msllHook{}), 32}, "BITMAPINFOHEADER": {unsafe.Sizeof(bitmapInfoHeader{}), 40},
		"BLENDFUNCTION": {unsafe.Sizeof(blendFunc{}), 4},
	}
	for n, c := range checks {
		if c[0] != c[1] {
			t.Errorf("%s size %d want %d", n, c[0], c[1])
		}
	}
}

func TestRender(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 80, 60))
	for y := 0; y < 60; y++ {
		for x := 0; x < 80; x++ {
			src.SetNRGBA(x, y, color.NRGBA{200, 100, 50, 128})
		}
	}
	out := renderBGRA(src, 40, 30, 0.97)
	// bottom row fully covered, premultiplied: R = 200*128/255 ≈ 100
	d := (29*40 + 20) * 4
	if out[d+3] < 126 || out[d+3] > 130 || out[d+2] < 98 || out[d+2] > 102 || out[d] < 23 || out[d] > 27 {
		t.Fatalf("bad pixel %v", out[d:d+4])
	}
	if out[3] != 0 { // top row empty because of squash
		t.Fatalf("top row should be transparent, alpha=%d", out[3])
	}
}

func TestPat(t *testing.T) {
	b := NewBrain(DefaultConfig())
	t0 := time.Unix(0, 0)
	if a := b.Pat(t0); a.Show != SpPatted {
		t.Fatal("pat should show patted")
	}
	if _, ok := b.Key(t0.Add(500 * time.Millisecond)); ok {
		t.Fatal("typing during pat should keep the happy face")
	}
	if _, ok := b.Key(t0.Add(2 * time.Second)); !ok {
		t.Fatal("typing after pat should work")
	}
	if got := scaleNearest([]byte{1, 2, 3, 4}, 1, 1, 2, 2); len(got) != 16 || got[12] != 1 {
		t.Fatal("scaleNearest")
	}
}

func TestFX(t *testing.T) {
	for _, w := range [][]byte{makePoyoWAV(), makePyorongWAV()} {
		if string(w[:4]) != "RIFF" || string(w[8:12]) != "WAVE" || len(w) < 10000 {
			t.Fatal("bad wav")
		}
	}
	sx, sy := jellyScale(0)
	if sy >= 1 || sx <= 1 {
		t.Fatalf("jelly should start squashed: %v %v", sx, sy)
	}
	if _, sy := jellyScale(float64(jellyFrames*jellyFrameMs) / 1000); sy < 0.97 || sy > 1.03 {
		t.Fatalf("jelly should settle, sy=%v", sy)
	}
	src := make([]byte, 25*20*4)
	for i := range src {
		src[i] = 255
	}
	dst := make([]byte, len(src))
	warpInto(dst, src, 25, 20, 1.1, 1.1)
	if dst[(10*25+12)*4+3] != 255 {
		t.Fatal("stretched frame should fill the box")
	}
	warpInto(dst, src, 25, 20, 1, 0.8)
	if dst[3] != 0 || dst[(19*25+12)*4+3] != 255 {
		t.Fatal("squashed frame: top should be empty, bottom filled")
	}
}

func BenchmarkWarpLarge(b *testing.B) {
	w, h := 1200, 893
	src := make([]byte, w*h*4)
	dst := make([]byte, w*h*4)
	for i := 0; i < b.N; i++ {
		warpInto(dst, src, w, h, 1.05, 0.92)
	}
}

func BenchmarkWarpDefault(b *testing.B) {
	w, h := 450, 335
	src := make([]byte, w*h*4)
	dst := make([]byte, w*h*4)
	for i := 0; i < b.N; i++ {
		warpInto(dst, src, w, h, 1.05, 0.92)
	}
}
