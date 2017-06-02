package affline

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
)

var (
	z = color.RGBA{0, 0, 0, 0}
	R = color.RGBA{255, 0, 0, 255}
	G = color.RGBA{0, 255, 0, 255}
	B = color.RGBA{0, 0, 255, 255}
)

var basic = [][]color.RGBA{
	{z, z, z, z, z, z, z, z, z, z},
	{z, z, R, R, G, G, B, z, z, z},
	{z, z, z, R, z, z, B, z, z, z},
	{z, z, z, R, G, G, B, B, z, z},
	{z, z, z, z, z, z, z, z, z, z},
}

func BasicImage() *image.RGBA {
	const width, height = 10, 5
	rect := image.Rect(0, 0, width, height)
	box := image.NewRGBA(rect)
	for y, row := range basic {
		for x, color := range row {
			box.Set(x, y, color)
		}
	}
	return box
}

func TestBasic(t *testing.T) {
	img := BasicImage()
	compare(t, img, basic)
}

func compare(t *testing.T, i image.Image, expected [][]color.RGBA) {
	img := i.(*image.RGBA)
	b := img.Bounds()
	height := b.Max.Y - b.Min.Y
	width := b.Max.X - b.Min.X

	if height != len(expected) {
		t.Fatalf("Heights do not match: %d != %d", height, len(expected))
	}
	if width != len(expected[0]) {
		t.Fatalf("Widths do not match: %d != %d", width, len(expected[0]))
	}

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			testPixelColor(t, img, x, y, expected[y][x])
		}
	}
}

func TestIdentity(t *testing.T) {
	src := BasicImage()
	tx := NewTransformer()
	result := tx.Apply(src)
	compare(t, result, basic)
}

func TestTranslate(t *testing.T) {
	src := BasicImage()
	tx := NewTransformer()
	tx.Translate(1, 1)
	result := tx.Apply(src)

	expected := tx.Aff3(src)
	compare(t, result, expected)
}

func TestReflectHoriz(t *testing.T) {
	src := BasicImage()
	tx := NewTransformer()
	tx.ReflectHoriz()
	result := tx.Apply(src)

	expected := tx.Aff3(src)
	compare(t, result, expected)
}

func TestReflectVert(t *testing.T) {
	src := BasicImage()
	trans := NewAffTrans(src)
	trans.ReflectVert()
	result := trans.Apply()

	expected := [][]color.RGBA{
		{z, z, z, z, z, z, z, z, z, z},
		{z, z, z, R, G, G, B, B, z, z},
		{z, z, z, R, z, z, B, z, z, z},
		{z, z, R, R, G, G, B, z, z, z},
		{z, z, z, z, z, z, z, z, z, z},
	}
	compare(t, result, expected)
}

func TestRotateNoChange(t *testing.T) {
	src := BasicImage()
	trans := NewAffTrans(src)
	trans.Rotate90(0)
	result := trans.Apply()
	compare(t, result, basic)
}

func TestRotate90CCW(t *testing.T) {
	src := BasicImage()
	trans := NewAffTrans(src)
	trans.Rotate90(1)
	result := trans.Apply()

	expected := [][]color.RGBA{
		{z, z, z, z, z},
		{z, z, z, z, z},
		{z, z, z, B, z},
		{z, B, B, B, z},
		{z, G, z, G, z},
		{z, G, z, G, z},
		{z, R, R, R, z},
		{z, R, z, z, z},
		{z, z, z, z, z},
		{z, z, z, z, z},
	}

	compare(t, result, expected)
}

func TestRotate180(t *testing.T) {
	// Rotate 180 degrees
	src := BasicImage()
	trans := NewAffTrans(src)
	trans.Rotate90(2)
	result := trans.Apply()

	// flip vertically and horizontally
	t2 := NewAffTrans(src)
	t2.ReflectVert()
	t2.ReflectHoriz()
	flipped := t2.Apply()
	expected := getColorMatrix(flipped)

	compare(t, result, expected)
}

func TestRotate360(t *testing.T) {
	// Rotate 180 degrees
	src := BasicImage()
	trans := NewAffTrans(src)
	trans.Rotate90(4)
	result := trans.Apply()

	compare(t, result, basic)
}

type AffTrans struct {
	src         image.Image
	height      int
	width       int
	m           *f64.Aff3
	op          draw.Op
	opts        *draw.Options
	transformer draw.Transformer
}

func NewAffTrans(src image.Image) *AffTrans {
	return &AffTrans{
		src:         src,
		height:      src.Bounds().Size().Y,
		width:       src.Bounds().Size().X,
		m:           &f64.Aff3{1, 0, 0, 0, 1, 0},
		op:          draw.Over,
		opts:        nil,
		transformer: draw.ApproxBiLinear,
	}
}

func (t *AffTrans) Translate(x, y float64) {
	t.m[2] = x
	t.m[5] = y
}

func (t *AffTrans) ReflectHoriz() {
	t.m[0] = -1
	t.m[2] = float64(t.src.Bounds().Size().X)
}

func (t *AffTrans) ReflectVert() {
	t.m[4] = -1
	t.m[5] = float64(t.src.Bounds().Size().Y)
}

func (t *AffTrans) Rotate90(count int) {
	rad := (math.Pi / 2) * float64(count)
	cx, cy := float64(t.src.Bounds().Size().X)/2, float64(t.src.Bounds().Size().Y)/2

	if (count % 2) > 0 {
		// Rotate the canvas
		t.height, t.width = t.width, t.height
		cy = cx
	}

	scale := 1.0
	cos := scale * math.Cos(rad)
	sin := scale * math.Sin(rad)

	t.m[0] = cos
	t.m[1] = sin
	t.m[2] = ((1.0 - cos) * cx) - (sin * cy)
	t.m[3] = -1 * sin
	t.m[4] = cos
	t.m[5] = (sin * cx) + ((1.0 - cos) * cy)
}

func (t *AffTrans) Apply() *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, t.width, t.height))
	t.transformer.Transform(dst, *t.m, t.src, t.src.Bounds(), t.op, t.opts)
	return dst
}

func testPixelColor(t *testing.T, i image.Image, x, y int, c color.Color) {
	pixel := i.At(x, y)
	if !coloreq(pixel, c) {
		t.Errorf("Colors do not match at (%d, %d): %v != %v", x, y, pixel, c)
	}
}

func coloreq(a, b color.Color) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()

	return (ar == br && ag == bg && ab == bb && aa == ba)
}

func ascii(c color.Color) string {
	if coloreq(c, R) {
		return "R"
	} else if coloreq(c, G) {
		return "G"
	} else if coloreq(c, B) {
		return "B"
	} else {
		return " "
	}
}

func print(img image.Image) {
	b := img.Bounds()
	width := b.Max.X - b.Min.X
	header := fmt.Sprintf("%s\n", strings.Repeat("*", width+2))

	for y := b.Min.Y; y < b.Max.Y; y++ {
		if y == b.Min.Y {
			fmt.Printf(header)
		}

		fmt.Printf("*")
		for x := b.Min.X; x < b.Max.X; x++ {
			fmt.Printf("%s", ascii(img.At(x, y)))
		}
		fmt.Printf("*\n")
	}
	fmt.Printf(header)
}

func printMatrix(pixels [][]color.RGBA) {
	width := len(pixels[0])
	header := fmt.Sprintf("%s\n", strings.Repeat("*", width+2))

	for y := range pixels {
		if y == 0 {
			fmt.Printf(header)
		}

		fmt.Printf("*")
		for x := range pixels[y] {
			fmt.Printf("%s", ascii(pixels[y][x]))
		}
		fmt.Printf("*\n")
	}
	fmt.Printf(header)
}
