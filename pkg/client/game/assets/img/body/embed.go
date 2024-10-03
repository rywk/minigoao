package body

import _ "embed"

var (
	//go:embed head.png
	Head_png []byte
	//go:embed body_naked.png
	BodyNaked_png []byte
	//go:embed dead_body.png
	DeadBody_png []byte
	//go:embed dead_head.png
	DeadHead_png []byte
)
