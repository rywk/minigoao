package assets

import (
	_ "embed"
)

var (
	//go:embed grass_patches.png
	GrassPatches_png []byte

	//go:embed body.png
	Body_png []byte

	//go:embed axe.png
	Axe_png []byte

	//go:embed head.png
	Head_png []byte

	//go:embed ongo.png
	Ongo_png []byte

	//go:embed tiletest.png
	Tiletest_png []byte
)
