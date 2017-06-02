package affline

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
)

var empty = color.RGBA{0, 0, 0, 0}

type Transformer struct {
	ops []Affline
}

func NewTransformer() *Transformer {
	return &Transformer{}
}

func (t *Transformer) Translate(x, y int) {
	t.ops = append(t.ops, &Translate{x, y})
}

func (t *Transformer) ReflectHoriz() {
	t.ops = append(t.ops, &ReflectHoriz{})
}

func (t *Transformer) Add(a Affline) {
	t.ops = append(t.ops, a)
}

func (t *Transformer) Apply(src image.Image) image.Image {
	op := draw.Over
	opts := new(draw.Options)
	tx := draw.ApproxBiLinear

	rect := src.Bounds()
	dst := image.NewRGBA(rect)

	m := f64.Aff3{1, 0, 0, 0, 1, 0}
	for _, op := range t.ops {
		om := op.Aff3(src)
		m = f64.Aff3{
			m[0]*om[0] + m[1]*m[3], m[0]*om[1] + m[1]*om[4], m[2] + m[0]*om[2] + m[1]*om[5],
			m[3]*om[0] + m[4]*om[3], m[3]*om[1] + m[4]*om[4], m[5] + m[3]*om[2] + m[4]*om[5],
		}
	}
	tx.Transform(dst, m, src, src.Bounds(), op, opts)
	return dst
}

func (t *Transformer) composeOps(src image.Image) f64.Aff3 {
	m := f64.Aff3{1, 0, 0, 0, 1, 0}
	for _, op := range t.ops {
		om := op.Aff3(src)
		m = f64.Aff3{
			m[0]*om[0] + m[1]*m[3], m[0]*om[1] + m[1]*om[4], m[2] + m[0]*om[2] + m[1]*om[5],
			m[3]*om[0] + m[4]*om[3], m[3]*om[1] + m[4]*om[4], m[5] + m[3]*om[2] + m[4]*om[5],
		}
	}
	return m
}

func getColorMatrix(img image.Image) [][]color.RGBA {
	b := img.Bounds()
	width := b.Max.X - b.Min.X
	height := b.Max.Y - b.Min.Y

	result := make([][]color.RGBA, height)

	for y := b.Min.Y; y < b.Max.Y; y++ {
		result[y] = make([]color.RGBA, width)

		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			result[y][x] = color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
		}
	}
	return result
}

// {a x + b y + c z, d x + e y + f z, z
func (t *Transformer) Aff3(src *image.RGBA) [][]color.RGBA {
	b := src.Bounds()
	width := b.Max.X - b.Min.X
	height := b.Max.Y - b.Min.Y
	pixels := getColorMatrix(src)
	m := t.composeOps(src)
	// {a x + b y + c z, d x + e y + f z, z}

	result := make([][]color.RGBA, height)
	for y := range result {
		result[y] = make([]color.RGBA, width)
		for x := range result[0] {
			fx, fy := float64(x), float64(y)
			tx := (m[0] * fx) + (m[1] * fy) + (m[2] * 1)
			ty := (m[3] * fx) + (m[4] * fy) + (m[5] * 1)
			if tx >= 0 && tx < float64(width) && ty >= 0 && ty < float64(height) {
				result[int(ty)][int(tx)] = pixels[y][x]
			}
		}
	}
	return result
}

type Affline interface {
	Aff3(img image.Image) f64.Aff3
	MTransform([][]color.RGBA) [][]color.RGBA
}

type Translate struct {
	X, Y int
}

func (t Translate) Aff3(img image.Image) f64.Aff3 {
	return f64.Aff3{1, 0, float64(t.X), 0, 1, float64(t.Y)}
}

func (t Translate) MTransform(pixels [][]color.RGBA) [][]color.RGBA {
	height := len(pixels)
	width := len(pixels[0])

	result := make([][]color.RGBA, height)

	for y := 0; y < height; y++ {
		result[y] = make([]color.RGBA, width)
		for x := 0; x < width; x++ {
			// Compute the source pixel
			sx := x - t.X
			sy := y - t.Y

			// Set the pixel at (x,y) from (sx,sy)
			if sx >= width || sx < 0 {
				result[y][x] = empty
			} else if sy >= height || sy < 0 {
				result[y][x] = empty
			} else {
				result[y][x] = pixels[sy][sx]
			}
		}
	}
	return result
}

type ReflectHoriz struct{}

func (r ReflectHoriz) Aff3(img image.Image) f64.Aff3 {
	width := float64(img.Bounds().Size().X)
	return f64.Aff3{-1, 0, width, 0, 1, 0}
}

func (r ReflectHoriz) MTransform(pixels [][]color.RGBA) [][]color.RGBA {
	height := len(pixels)
	width := len(pixels[0])

	result := make([][]color.RGBA, height)
	for y := 0; y < height; y++ {
		result[y] = make([]color.RGBA, width)

		for x := 0; x < width; x++ {
			result[y][x] = pixels[y][width-1-x]
		}
	}
	return result
}

type ReflectVert struct{}

func (r ReflectVert) Aff3(img image.Image) f64.Aff3 {
	height := float64(img.Bounds().Size().Y)
	return f64.Aff3{1, 0, 0, 0, -1, height}
}

func (r ReflectVert) MTransform(pixels [][]color.RGBA) [][]color.RGBA {
	height := len(pixels)
	width := len(pixels[0])
	result := make([][]color.RGBA, height)

	for y := 0; y < height; y++ {
		result[y] = make([]color.RGBA, width)

		for x := 0; x < width; x++ {
			result[y][x] = pixels[height-1-y][x]
		}
	}
	return result
}

type Rotate struct {
	Turns        int
	ExpandCanvas bool
}

func (r Rotate) Aff3(img image.Image) f64.Aff3 {
	rad := (math.Pi / 2) * float64(r.Turns)
	size := img.Bounds().Size()
	scale := 1.0

	cos := scale * math.Cos(rad)
	sin := scale * math.Sin(rad)
	cx := float64(size.X) / 2
	cy := float64(size.Y) / 2

	return f64.Aff3{
		cos, sin, ((1.0 - cos) * cx) - (sin - cy),
		-1 * sin, cos, (sin * cx) + ((1.0 - cos) * cy),
	}
}

func (r Rotate) MTransform(pixels [][]color.RGBA) {
}
