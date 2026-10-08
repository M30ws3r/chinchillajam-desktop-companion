package main

import (
	"bytes"
	"encoding/binary"
	"math"
)

// ---------- "뽀용!" sound ----------

// makePoyoWAV synthesizes a short cute "boing" (뽀 = quick pop, 용 = wobbly rising glide)
// and returns it as a 16-bit mono PCM WAV file.
func makePoyoWAV() []byte {
	const rate = 22050
	const dur = 0.36
	n := int(rate * dur)
	samples := make([]int16, n)
	phase := 0.0
	for i := 0; i < n; i++ {
		t := float64(i) / rate
		var f, env float64
		if t < 0.05 { // "뽀": bright pop, pitch drops quickly
			f = 820 - 4200*t
			env = math.Min(1, t/0.004)
		} else { // "용": soft glide upward with a jelly-like wobble
			u := t - 0.05
			f = 560 + 520*(1-math.Exp(-u/0.07)) + 60*math.Sin(2*math.Pi*18*u)*math.Exp(-u/0.12)
			env = 0.9 * math.Exp(-u/0.11)
		}
		phase += 2 * math.Pi * f / rate
		v := math.Sin(phase) + 0.25*math.Sin(2*phase) + 0.08*math.Sin(3*phase)
		if t > dur-0.02 { // fade out tail
			env *= (dur - t) / 0.02
		}
		samples[i] = int16(v / 1.33 * env * 0.55 * 32767)
	}
	return pcmWAV(samples, rate)
}

// makePyorongWAV synthesizes a bright, friendly "뾰롱!" like a cute like-notification:
// a quick upward chirp (뾰) landing on a soft bell tone a major sixth higher (롱).
func makePyorongWAV() []byte {
	const rate = 22050
	const dur = 0.42
	n := int(rate * dur)
	samples := make([]int16, n)
	phase := 0.0
	for i := 0; i < n; i++ {
		t := float64(i) / rate
		var f, env float64
		if t < 0.07 { // "뾰": short rising chirp
			f = 1050 + 280*(t/0.07)
			env = math.Min(1, t/0.005) * 0.75
		} else { // "롱": bell-like note with a gentle shimmer
			u := t - 0.07
			f = 1760 * (1 + 0.006*math.Sin(2*math.Pi*7*u))
			env = math.Min(1, u/0.006) * math.Exp(-u/0.12)
		}
		phase += 2 * math.Pi * f / rate
		v := math.Sin(phase) + 0.3*math.Sin(2*phase)*math.Exp(-t/0.08) + 0.12*math.Sin(3*phase)
		if t > dur-0.02 {
			env *= (dur - t) / 0.02
		}
		samples[i] = int16(v / 1.42 * env * 0.42 * 32767)
	}
	return pcmWAV(samples, rate)
}

func pcmWAV(samples []int16, rate int) []byte {
	var buf bytes.Buffer
	dataLen := uint32(len(samples) * 2)
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+dataLen))
	buf.WriteString("WAVEfmt ")
	for _, v := range []any{uint32(16), uint16(1), uint16(1), uint32(rate), uint32(rate * 2), uint16(2), uint16(16)} {
		binary.Write(&buf, binary.LittleEndian, v)
	}
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, dataLen)
	binary.Write(&buf, binary.LittleEndian, samples)
	return buf.Bytes()
}

// ---------- jelly wobble ----------

const (
	jellyFrames  = 24
	jellyFrameMs = 30
)

// jellyScale returns (scaleX, scaleY) at time t seconds: a damped squash-and-stretch,
// starting squashed (like a pudding being pressed), roughly area-preserving.
func jellyScale(t float64) (float64, float64) {
	amp := 0.11 * math.Exp(-t/0.2)
	sy := 1 - amp*math.Cos(2*math.Pi*4.2*t)
	sx := 1 + (1-sy)*0.7
	return sx, sy
}

// warpInto draws src (premultiplied BGRA, w×h) scaled by (sx, sy) around the bottom-center
// into dst (same size) with bilinear filtering. Used for the squash and jelly effects so
// no extra frames have to be kept in memory.
func warpInto(dst, src []byte, w, h int, sx, sy float64) {
	cx, fh := float64(w)/2, float64(h)
	isx, isy := 1/sx, 1/sy
	// per-column source positions (fixed 8-bit fractions), computed once per frame
	xs := make([]int32, w)
	xf := make([]int32, w)
	for x := 0; x < w; x++ {
		f := (float64(x)+0.5-cx)*isx + cx - 0.5
		x0 := math.Floor(f)
		xs[x] = int32(x0)
		xf[x] = int32((f - x0) * 256)
	}
	stride := w * 4
	px := func(x, y int32) (int32, int32, int32, int32) {
		if x < 0 || y < 0 || int(x) >= w || int(y) >= h {
			return 0, 0, 0, 0
		}
		o := int(y)*stride + int(x)*4
		return int32(src[o]), int32(src[o+1]), int32(src[o+2]), int32(src[o+3])
	}
	for y := 0; y < h; y++ {
		f := (float64(y)+0.5-fh)*isy + fh - 0.5
		fy0 := math.Floor(f)
		y0 := int32(fy0)
		ty := int32((f - fy0) * 256)
		row := dst[y*stride : (y+1)*stride]
		if y0 < -1 || int(y0) >= h {
			clear(row)
			continue
		}
		interiorY := y0 >= 0 && int(y0)+1 < h
		for x := 0; x < w; x++ {
			x0, tx := xs[x], xf[x]
			d := x * 4
			if x0 < -1 || int(x0) >= w {
				row[d], row[d+1], row[d+2], row[d+3] = 0, 0, 0, 0
				continue
			}
			var a0, a1, a2, a3, b0, b1, b2, b3, c0, c1, c2, c3, e0, e1, e2, e3 int32
			if interiorY && x0 >= 0 && int(x0)+1 < w {
				o := int(y0)*stride + int(x0)*4
				p := src[o : o+8 : o+8]
				q := src[o+stride : o+stride+8 : o+stride+8]
				a0, a1, a2, a3 = int32(p[0]), int32(p[1]), int32(p[2]), int32(p[3])
				b0, b1, b2, b3 = int32(p[4]), int32(p[5]), int32(p[6]), int32(p[7])
				c0, c1, c2, c3 = int32(q[0]), int32(q[1]), int32(q[2]), int32(q[3])
				e0, e1, e2, e3 = int32(q[4]), int32(q[5]), int32(q[6]), int32(q[7])
			} else {
				a0, a1, a2, a3 = px(x0, y0)
				b0, b1, b2, b3 = px(x0+1, y0)
				c0, c1, c2, c3 = px(x0, y0+1)
				e0, e1, e2, e3 = px(x0+1, y0+1)
			}
			itx, ity := 256-tx, 256-ty
			row[d] = byte(((a0*itx+b0*tx)*ity + (c0*itx+e0*tx)*ty + 32768) >> 16)
			row[d+1] = byte(((a1*itx+b1*tx)*ity + (c1*itx+e1*tx)*ty + 32768) >> 16)
			row[d+2] = byte(((a2*itx+b2*tx)*ity + (c2*itx+e2*tx)*ty + 32768) >> 16)
			row[d+3] = byte(((a3*itx+b3*tx)*ity + (c3*itx+e3*tx)*ty + 32768) >> 16)
		}
	}
}
