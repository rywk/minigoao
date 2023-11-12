package audio2d_test

import (
	"bytes"
	"testing"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/wav"
	"github.com/rywk/minigoao/pkg/client/game/assets/audio"
	"github.com/stretchr/testify/require"
)

func TestSampleRate(t *testing.T) {
	sampleRate := beep.SampleRate(44100)
	for _, f := range []struct {
		file string
		f    []byte
	}{
		{file: "MeleeAir_wav", f: audio.MeleeAir_wav},
		{file: "MeleeHit_wav", f: audio.MeleeHit_wav},
		{file: "Spawn_wav", f: audio.Spawn_wav},
		{file: "Walk1_wav", f: audio.Walk1_wav},
		{file: "Walk2_wav", f: audio.Walk2_wav},
		{file: "SpellApoca_wav", f: audio.SpellApoca_wav},
		{file: "SpellInmo_wav", f: audio.SpellInmo_wav},
		{file: "SpellInmoRm_wav", f: audio.SpellInmoRm_wav},
	} {
		_, format, err := wav.Decode(bytes.NewReader(f.f))
		require.NoError(t, err, f.file)
		require.Equal(t, sampleRate, format.SampleRate, f.file)
	}
}
