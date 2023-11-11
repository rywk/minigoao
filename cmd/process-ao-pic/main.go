package main

// import (
// 	"bytes"
// 	"flag"
// 	"fmt"
// 	"image/color"
// 	"image/png"
// 	"os"
// 	"strings"
// )

// var imgPath = flag.String("img", "", "-img=...")

// func main() {
// 	flag.Parse()
// 	if imgPath == nil || *imgPath == "" {
// 		panic("empty url parameter")
// 	}
// 	pathParts := strings.Split(*imgPath, "/")
// 	if len(pathParts) == 1 {
// 		pathParts = append([]string{"."}, pathParts[0])
// 	}
// 	restOfPath := strings.Join(pathParts[:len(pathParts)-1], "/")

// 	fullFilename := pathParts[len(pathParts)-1]
// 	filenameParts := strings.Split(fullFilename, ".")
// 	filename := strings.Join(filenameParts[:len(filenameParts)-1], ".")
// 	fileExt := filenameParts[len(filenameParts)-1]

// 	newFilePath := fmt.Sprintf("%s/%s-clean.%s", restOfPath, filename, fileExt)

// 	srcImgBytes, err := os.ReadFile(*imgPath)
// 	if err != nil {
// 		panic(err)
// 	}

// 	img, err := png.Decode(bytes.NewReader(srcImgBytes))
// 	if err != nil {
// 		panic(err)
// 	}
// 	firstColor := img.At(img.Bounds().Min.X, img.Bounds().Min.Y)
// 	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
// 		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {

// 			c := img.At(x, y)

// 		}
// 	}

// 	buf := new(bytes.Buffer)
// 	if err := png.Encode(buf, img); err != nil {
// 		panic(err)
// 	}

// 	fmt.Printf("writing file %s\n", newFilePath)
// 	err = os.WriteFile(newFilePath, buf.Bytes(), os.ModePerm)
// 	if err != nil {
// 		panic(err)
// 	}
// }

// func rgbaToRGBA(r uint32, g uint32, b uint32, a uint32) color.RGBA {
// 	return color.RGBA{uint8(r / 257), uint8(g / 257), uint8(b / 257), uint8(a / 257)}
// }
