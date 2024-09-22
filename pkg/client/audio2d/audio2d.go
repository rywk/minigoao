package audio2d

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/vorbis"
	"github.com/gopxl/beep/v2/wav"
	audiofile "github.com/rywk/minigoao/pkg/client/game/assets/audio"
	"github.com/rywk/minigoao/pkg/constants/assets"
)

const (
	SampleRate    = beep.SampleRate(44100)
	SampleRateWeb = beep.SampleRate(22050)
)

type SoundBoard struct {
	sampleRate beep.SampleRate
	sounds     map[assets.Sound]*Sound
}

type Sound struct {
	sampleRate beep.SampleRate
	buffer     *beep.Buffer
}

func NewSound(bs []byte) *Sound {
	s := &Sound{}
	s.sampleRate = SampleRate
	st, f := mustDecode(bs)
	s.buffer = beep.NewBuffer(f)
	s.buffer.Append(st)
	st.Close()
	return s
}
func NewOggSound(bs []byte) *Sound {
	s := &Sound{}
	s.sampleRate = SampleRate
	st, f := mustDecodeOgg(bs)
	s.buffer = beep.NewBuffer(f)
	s.buffer.Append(st)
	st.Close()
	return s
}
func NewSoundCopies(n int, bs []byte) []*Sound {
	sounds := make([]*Sound, n)
	for i := range sounds {
		sounds[i] = NewSound(bs)
	}
	return sounds
}

func (s *Sound) Play() {
	leftCh, rightCh := beep.Dup(s.buffer.Streamer(0, s.buffer.Len()))
	leftCh = effects.Mono(multiplyChannels(1, 0, leftCh))
	rightCh = effects.Mono(multiplyChannels(0, 1, rightCh))
	NewMovingStreamer(s.sampleRate, -1, 0, leftCh).Play()
	NewMovingStreamer(s.sampleRate, +1, 0, rightCh).Play()
}

func (s *Sound) PlayFrom(x, y, sx, sy int) {
	dx, dy := x-sx, y-sy
	var mx, my float64
	if dx < 0 {
		mx = mapValue(float64(dx), 0, 30, 1, 15)
	} else {
		mx = mapValue(float64(dx), -30, 0, -15, -1)
	}
	if dy < 0 {
		my = mapValue(float64(dy), -20, 0, -10, -1)
	} else {
		my = mapValue(float64(dy), 0, 20, 1, 10)
	}
	leftCh, rightCh := beep.Dup(s.buffer.Streamer(0, s.buffer.Len()))
	leftCh = effects.Mono(multiplyChannels(1, 0, leftCh))
	rightCh = effects.Mono(multiplyChannels(0, 1, rightCh))
	NewMovingStreamer(s.sampleRate, mx, my, leftCh).Play()
	NewMovingStreamer(s.sampleRate, mx, my, rightCh).Play()
}

func NewSoundBoard(web bool) *SoundBoard {
	sb := &SoundBoard{}
	if web {
		sb.sampleRate = SampleRateWeb
		speaker.Init(sb.sampleRate, 1026)
		sb.sounds = map[assets.Sound]*Sound{
			assets.Spawn:                NewOggSound(audiofile.SpawnLow_ogg),
			assets.MeleeAir:             NewOggSound(audiofile.MeleeAirLow_ogg),
			assets.MeleeBlood:           NewOggSound(audiofile.MeleeHitLow_ogg),
			assets.Walk1:                NewOggSound(audiofile.Walk1Low_ogg),
			assets.Walk2:                NewOggSound(audiofile.Walk2Low_ogg),
			assets.SpellApocaSound:      NewOggSound(audiofile.SpellApocaLow_ogg),
			assets.SpellDescaSound:      NewOggSound(audiofile.SpellDescaLow_ogg),
			assets.SpellInmoSound:       NewOggSound(audiofile.SpellInmoLow_ogg),
			assets.SpellHealWoundsSound: NewOggSound(audiofile.SpellHealWoundsLow_ogg),
			assets.SpellResurrectSound:  NewOggSound(audiofile.SpellResurrectLow_ogg),
			assets.SpellInmoRmSound:     NewOggSound(audiofile.SpellInmoRmLow_ogg),
			assets.Potion:               NewOggSound(audiofile.PotionLow_ogg),
		}
		return sb
	}
	sb.sampleRate = SampleRate
	speaker.Init(sb.sampleRate, sb.sampleRate.N(time.Second/45))
	sb.sounds = map[assets.Sound]*Sound{
		assets.Spawn:                NewSound(audiofile.Spawn_wav),
		assets.MeleeAir:             NewSound(audiofile.MeleeAir_wav),
		assets.MeleeBlood:           NewSound(audiofile.MeleeHit_wav),
		assets.Walk1:                NewSound(audiofile.Walk1_wav),
		assets.Walk2:                NewSound(audiofile.Walk2_wav),
		assets.SpellApocaSound:      NewSound(audiofile.SpellApoca_wav),
		assets.SpellDescaSound:      NewSound(audiofile.SpellDesca_wav),
		assets.SpellInmoSound:       NewSound(audiofile.SpellInmo_wav),
		assets.SpellHealWoundsSound: NewSound(audiofile.SpellHealWounds_wav),
		assets.SpellResurrectSound:  NewSound(audiofile.SpellResurrect_wav),
		assets.SpellInmoRmSound:     NewSound(audiofile.SpellInmoRm_wav),
		assets.Potion:               NewSound(audiofile.Potion_wav),
	}
	return sb
}

func (sb *SoundBoard) Play(s assets.Sound) {
	sb.sounds[s].Play()
}

func (sb *SoundBoard) PlayFrom(s assets.Sound, x, y, sx, sy int32) {
	sb.sounds[s].PlayFrom(int(x), int(y), int(sx), int(sy))
}

func multiplyChannels(left, right float64, s beep.Streamer) beep.Streamer {
	return beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		n, ok = s.Stream(samples)
		for i := range samples[:n] {
			samples[i][0] *= left
			samples[i][1] *= right
		}
		return n, ok
	})
}

type MovingStreamer struct {
	x, y         float64
	velX, velY   float64
	leftDoppler  beep.Streamer
	rightDoppler beep.Streamer
}

func NewMovingStreamer(sr beep.SampleRate, x, y float64, streamer beep.Streamer) *MovingStreamer {
	ms := &MovingStreamer{x: x, y: y}

	const metersPerSecond = 343
	samplesPerSecond := float64(sr)
	samplesPerMeter := samplesPerSecond / metersPerSecond

	leftEar, rightEar := beep.Dup(streamer)
	leftEar = multiplyChannels(1, 0, leftEar)
	rightEar = multiplyChannels(0, 1, rightEar)

	const earDistance = 0.16
	ms.leftDoppler = effects.Doppler(2, samplesPerMeter, leftEar, func(delta int) float64 {
		dt := sr.D(delta).Seconds()
		ms.x += ms.velX * dt
		ms.y += ms.velY * dt
		return math.Max(0.25, math.Hypot(ms.x+earDistance/2, ms.y))
	})
	ms.rightDoppler = effects.Doppler(2, samplesPerMeter, rightEar, func(delta int) float64 {
		return math.Max(0.25, math.Hypot(ms.x-earDistance/2, ms.y))
	})

	return ms
}

func (ms *MovingStreamer) Play() {
	speaker.Play(ms.leftDoppler, ms.rightDoppler)
}

func report(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func mustDecode(bs []byte) (beep.StreamSeekCloser, beep.Format) {
	s, f, err := wav.Decode(bytes.NewReader(bs))
	if err != nil {
		report(err)
	}
	return s, f
}

func mustDecodeOgg(bs []byte) (beep.StreamSeekCloser, beep.Format) {
	rc := io.NopCloser(bytes.NewBuffer(bs))
	s, f, err := vorbis.Decode(rc)
	if err != nil {
		panic(err)
	}
	return s, f
}

// func mustDecodeWav(bs []byte) []byte {
// 	d, err := wavv.DecodeWithoutResampling(bytes.NewReader(bs))
// 	if err != nil {
// 		panic(err)
// 	}
// 	dbs, _ := io.ReadAll(d)
// 	return dbs
// }

func mapValue(v, start1, stop1, start2, stop2 float64) float64 {
	newval := (v-start1)/(stop1-start1)*(stop2-start2) + start2
	if start2 < stop2 {
		if newval > stop2 {
			newval = stop2
		} else if newval < start2 {
			newval = start2
		}
	} else {
		if newval > start2 {
			newval = start2
		} else if newval < stop2 {
			newval = stop2
		}
	}
	return newval
}

// p5.prototype.map = function(n, start1, stop1, start2, stop2, withinBounds) {
// 	p5._validateParameters('map', arguments);
// 	const newval = (n - start1) / (stop1 - start1) * (stop2 - start2) + start2;
// 	if (!withinBounds) {
// 	  return newval;
// 	}
// 	if (start2 < stop2) {
// 	  return this.constrain(newval, start2, stop2);
// 	} else {
// 	  return this.constrain(newval, stop2, start2);
// 	}
//   };
