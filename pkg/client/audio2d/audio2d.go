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
	web        bool
	sampleRate beep.SampleRate
	sounds     map[assets.Sound]*Sound
	Volume     float64
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
	s.sampleRate = SampleRateWeb
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
func (s *Sound) PlayFlat(vol float64) {
	speaker.Play(&effects.Volume{
		Streamer: s.buffer.Streamer(0, s.buffer.Len()),
		Base:     2,
		Volume:   vol,
		Silent:   false,
	})
}

func (s *Sound) Play(vol float64) {
	leftCh, rightCh := beep.Dup(s.buffer.Streamer(0, s.buffer.Len()))

	const earDistance = 0.16
	const metersPerSecond = 343
	samplesPerSecond := float64(s.sampleRate)
	samplesPerMeter := samplesPerSecond / metersPerSecond

	leftCh = effects.Mono(multiplyChannels(1, 0, leftCh))
	leftEar1, rightEar1 := beep.Dup(leftCh)
	s1 := effects.Doppler(2, samplesPerMeter, multiplyChannels(1, 0, leftEar1), func(delta int) float64 {
		return math.Max(0.25, math.Hypot(1+earDistance/2, 0))
	})
	s2 := effects.Doppler(2, samplesPerMeter, multiplyChannels(0, 1, rightEar1), func(delta int) float64 {
		return math.Max(0.25, math.Hypot(1-earDistance/2, 0))
	})

	rightCh = effects.Mono(multiplyChannels(0, 1, rightCh))
	leftEar2, rightEar2 := beep.Dup(rightCh)
	s3 := effects.Doppler(2, samplesPerMeter, multiplyChannels(1, 0, leftEar2), func(delta int) float64 {
		return math.Max(0.25, math.Hypot(-1+earDistance/2, 0))
	})
	s4 := effects.Doppler(2, samplesPerMeter, multiplyChannels(0, 1, rightEar2), func(delta int) float64 {
		return math.Max(0.25, math.Hypot(-1-earDistance/2, 0))
	})
	// silent := false
	// if vol <= 0 {
	// 	silent = true
	// }

	speaker.Play(&effects.Volume{
		Streamer: beep.Mix(s1, s2, s3, s4),
		Base:     2,
		Volume:   vol,
		Silent:   false,
	})
}

func (s *Sound) PlayFrom(vol float64, x, y, sx, sy int) {
	dx, dy := sx-x, sy-y
	var mx, my float64
	if dx < 0 {
		mx = mapValue(float64(dx), -40, 0, -7, -1)
	} else {
		mx = mapValue(float64(dx), 0, 40, 1, 7)
	}
	if dy < 0 {
		my = mapValue(float64(dy), -40, 0, -7, -1)
	} else {
		my = mapValue(float64(dy), 0, 40, 1, 7)
	}
	s.PlayMovingStreamers(vol, mx, my)
}

func (s *Sound) PlayMovingStreamers(vol, mx, my float64) {
	leftCh, rightCh := beep.Dup(s.buffer.Streamer(0, s.buffer.Len()))

	const earDistance = 0.16
	const metersPerSecond = 343
	samplesPerSecond := float64(s.sampleRate)
	samplesPerMeter := samplesPerSecond / metersPerSecond

	leftCh = effects.Mono(multiplyChannels(1, 0, leftCh))
	leftEar1, rightEar1 := beep.Dup(leftCh)
	s1 := effects.Doppler(2, samplesPerMeter, multiplyChannels(1, 0, leftEar1), func(delta int) float64 {
		return math.Max(0.25, math.Hypot(mx+earDistance/2, my))
	})
	s2 := effects.Doppler(2, samplesPerMeter, multiplyChannels(0, 1, rightEar1), func(delta int) float64 {
		return math.Max(0.25, math.Hypot(mx-earDistance/2, my))
	})

	rightCh = effects.Mono(multiplyChannels(0, 1, rightCh))
	leftEar2, rightEar2 := beep.Dup(rightCh)
	s3 := effects.Doppler(2, samplesPerMeter, multiplyChannels(1, 0, leftEar2), func(delta int) float64 {
		return math.Max(0.25, math.Hypot(mx+earDistance/2, my))
	})
	s4 := effects.Doppler(2, samplesPerMeter, multiplyChannels(0, 1, rightEar2), func(delta int) float64 {
		return math.Max(0.25, math.Hypot(mx-earDistance/2, my))
	})
	// silent := false
	// if vol <= 0 {
	// 	silent = true
	// }
	speaker.Play(&effects.Volume{
		Streamer: beep.Mix(s1, s2, s3, s4),
		Base:     2,
		Volume:   vol,
		Silent:   false,
	})
}

func NewSoundBoard(web bool) *SoundBoard {
	sb := &SoundBoard{
		web:    web,
		Volume: 0,
	}
	if web {
		sb.sampleRate = SampleRateWeb
		speaker.Init(sb.sampleRate, 1027)
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
			assets.Death:                NewOggSound(audiofile.Death_ogg),
			assets.KillBell:             NewOggSound(audiofile.KillBell_ogg),
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
		assets.Death:                NewSound(audiofile.Death_wav),
		assets.KillBell:             NewSound(audiofile.KillBell_wav),
	}
	return sb
}

type AudioMixer interface {
	Play(s assets.Sound)
	PlayFrom(s assets.Sound, x, y, sx, sy int32)
	SetVolume(float64)
}

var _ AudioMixer = (*SoundBoard)(nil)

func (sb *SoundBoard) SetVolume(v float64) {
	sb.Volume = mapValue(v, 0, 200, -8, 2)
}
func (sb *SoundBoard) Play(s assets.Sound) {
	sb.sounds[s].Play(sb.Volume)
}

func (sb *SoundBoard) PlayFrom(s assets.Sound, x, y, sx, sy int32) {
	sb.sounds[s].PlayFrom(sb.Volume, int(x), int(y), int(sx), int(sy))
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

type Sound struct {
	sampleRate beep.SampleRate
	buffer     *beep.Buffer
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
