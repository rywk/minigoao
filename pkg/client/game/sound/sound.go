package sound

import (
	"bytes"
	"log"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	audiofile "github.com/rywk/minigoao/pkg/client/game/assets/audio"
	"github.com/rywk/minigoao/proto/message/assets"
)

type Sounds struct {
	ctx          *audio.Context
	Walk1, Walk2 *audio.Player
	Spawn        *audio.Player
	MeleeAir     *audio.Player
	MeleeHit     *audio.Player
}

func NewSounds(ctx *audio.Context) *Sounds {
	s, err := &Sounds{}, (error)(nil)
	walk1d, err := wav.DecodeWithoutResampling(bytes.NewReader(audiofile.Walk1_wav))
	if err != nil {
		log.Fatal(err)
	}
	//ctx.NewPlayerFromBytes()
	s.Walk1, err = ctx.NewPlayer(walk1d)
	if err != nil {
		log.Fatal(err)
	}

	return s
}

func decodeWav(bs []byte) *wav.Stream {
	walk1d, err := wav.DecodeWithoutResampling(bytes.NewReader(bs))
	if err != nil {
		log.Fatal(err)
	}
	return walk1d
}

var assetFile = map[assets.Sound]*[]byte{
	assets.Spawn:      &audiofile.Spawn_wav,
	assets.Walk1:      &audiofile.Walk1_wav,
	assets.Walk2:      &audiofile.Walk2_wav,
	assets.MeleeAir:   &audiofile.MeleeAir_wav,
	assets.MeleeBlood: &audiofile.MeleeHit_wav,
}

func PlaySoundEffect(ctx *audio.Context, s assets.Sound) {
	sePlayer := ctx.NewPlayerFromBytes(*assetFile[s])
	sePlayer.SetVolume(.08)
	sePlayer.Play()
}
