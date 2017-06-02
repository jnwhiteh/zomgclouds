package main

import (
	"compress/gzip"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/ajstarks/svgo"
	"github.com/gosvg/gosvg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/inconsolata"
	"golang.org/x/image/math/fixed"
)

func main() {
	http.Handle("/circle.svg", http.HandlerFunc(circle))
	http.Handle("/circle2.svg", http.HandlerFunc(circle2))
	http.Handle("/text.png", http.HandlerFunc(text))
	http.Handle("/cloud", http.HandlerFunc(cloud))
	http.Handle("/", http.HandlerFunc(index))
	err := http.ListenAndServe(":3001", nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func circle(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	s := svg.New(w)
	s.Start(500, 500)
	s.Circle(250, 250, 125, "fill:none;stroke:black")
	s.Gtransform("rotate(90)")
	s.Text(250, 250, "Hello Go!", "fill:black;stroke:black;text-anchor:middle;font-size:30px")
	s.Gend()
	s.End()
	// hello
}

func circle2(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	s := gosvg.NewSVG(500, 500)
	r := s.Rect(5, 5, 490, 490)
	r.Style.Set("fill", "none")
	r.Style.Set("stroke-width", "2")
	r.Style.Set("stroke", "blue")
	c := s.Circle(250, 250, 30)
	c.Style.Set("fill", "red")
	c.Style.Set("stroke", "none")

	s.Render(w)
}

func text(w http.ResponseWriter, req *http.Request) {
	img := image.NewRGBA(image.Rect(0, 0, 500, 500))

	orange := color.RGBA{200, 100, 0, 255}
	var x = 250
	var y = 250

	point := fixed.Point26_6{fixed.Int26_6(x * 64), fixed.Int26_6(y * 64)}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(orange),
		Face: inconsolata.Regular8x16,
		Dot:  point,
	}
	d.DrawString("Hello Go")

	//	rot := image.NewRGBA(image.Rect(0, 0, 500, 500))
	//	var m f64.Aff3
	//	var sr image.Rectangle = image.Rect(0, 0, 500, 500)
	//	var op draw.Op = draw.Src
	//	var opts *draw.Options = nil
	//	draw.ApproxBiLinear.Transform(rot, m, img, sr, op, opts)

	if err := png.Encode(w, img); err != nil {
		panic(err)
	}
}

func addLabel(img *image.RGBA, x, y int, label string) {
	col := color.RGBA{200, 100, 0, 255}
	point := fixed.Point26_6{fixed.Int26_6(x * 64), fixed.Int26_6(y * 64)}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: inconsolata.Regular8x16,
		Dot:  point,
	}
	d.DrawString(label)
}

func index(w http.ResponseWriter, req *http.Request) {
	body := `
<html>
<body>
	<img src="/circle.svg"/>
	<img src="/text.png"/>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, body)
}

func cloud(w http.ResponseWriter, req *http.Request) {
	msgs, err := parse("infobot.rikers.org/#WoWUIDev/20081123.html.gz")
	if err != nil {
		fmt.Fprintf(w, "Error: %s", err)
		return
	}

	cloud := NewCloud()
	for _, msg := range msgs {
		if msg.Type == "message" {
			cloud.add(msg.Text)
		}
	}

	cloud.Print(w)
}

type Message struct {
	Type    string
	Raw     string
	Time    string
	Channel string
	User    string
	Netmask string
	Text    string
}

func parse(filename string) ([]*Message, error) {
	gzipped, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer gzipped.Close()
	file, err := gzip.NewReader(gzipped)
	if err != nil {
		return nil, err
	}

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	data := string(fileBytes)

	re := regexp.MustCompile(`<tr.*`)
	lines := re.FindAllString(data, -1)

	var msgs []*Message

	for _, line := range lines {
		msg := parseLine(line)
		if msg == nil {
			fmt.Printf("Unparsed: %s\n", line)
			break
		}
		msgs = append(msgs, msg)
	}

	return msgs, nil
}

var reJoin = regexp.MustCompile(`<tr><td><tt>(\S*)</tt></td><td colspan=2><tt><font color="\S*">(\*\*\* join/(\S*) (\S*) \(.*\)(?: \[NETSPLIT VICTIM\])?)</font></tt></td></tr>`)
var rePart = regexp.MustCompile(`<tr><td><tt>(\S*)</tt></td><td colspan=2><tt><font color="\S*">(\*\*\* part/(\S*) (\S*) \(.*\)(?: \[NETSPLIT VICTIM\])?)</font></tt></td></tr>`)
var reKick = regexp.MustCompile(`<tr><td><tt>(\S*)</tt></td><td colspan=2><tt><font color="\S*">(\*\*\* kick/(\S*) \[\S*\] by (\S*) (.*))</font></tt></td></tr>`)
var reMessage = regexp.MustCompile(`<tr bgcolor="\S*"><td><tt>(\S*)</tt></td><td><font color="\S*"><tt>(\S*)</tt></font></td><td width="100%"><tt>(.*)</tt></td></tr>`)
var reModeChange = regexp.MustCompile(`<tr><td><tt>(\S*)</tt></td><td colspan=2><tt><font color="\S*">(\*\*\* mode/(\S*) \[(\S*)(?: (.*))?\] by (\S*))</font></tt></td></tr>`)
var reTopic1 = regexp.MustCompile(`<tr><td><tt>(\S*)</tt></td><td colspan=2><tt><font color="\S*">(\*\*\* topic/(\S*) is (.*))</font></tt></td></tr>`)
var reTopic2 = regexp.MustCompile(`<tr><td><tt>(\S*)</tt></td><td colspan=2><tt><font color="\S*">(\*\*\* topic/(\S*) by (\S*) (.*))</font></tt></td></tr>`)

func parseLine(line string) *Message {
	parts := reJoin.FindAllStringSubmatch(line, -1)
	if len(parts) > 0 {
		return &Message{
			Type: "join",
			Raw:  parts[0][0],
			Time: parts[0][1],
		}
	}

	parts = rePart.FindAllStringSubmatch(line, -1)
	if len(parts) > 0 {
		return &Message{
			Type: "part",
			Raw:  parts[0][0],
			Time: parts[0][1],
		}
	}

	parts = reMessage.FindAllStringSubmatch(line, -1)
	if len(parts) > 0 {
		return &Message{
			Type: "message",
			Raw:  parts[0][0],
			Time: parts[0][1],
			User: parts[0][2],
			Text: parts[0][3],
		}
	}

	parts = reModeChange.FindAllStringSubmatch(line, -1)
	if len(parts) > 0 {
		return &Message{
			Type: "mode",
			Raw:  parts[0][0],
			Time: parts[0][1],
			Text: parts[0][2],
		}
	}

	parts = reTopic1.FindAllStringSubmatch(line, -1)
	if len(parts) > 0 {
		return &Message{
			Type: "topic",
			Raw:  parts[0][0],
			Time: parts[0][1],
		}
	}

	parts = reTopic2.FindAllStringSubmatch(line, -1)
	if len(parts) > 0 {
		return &Message{
			Type: "topic",
			Raw:  parts[0][0],
			Time: parts[0][1],
		}
	}

	parts = reKick.FindAllStringSubmatch(line, -1)
	if len(parts) > 0 {
		return &Message{
			Type: "kick",
			Raw:  parts[0][0],
			Time: parts[0][1],
		}
	}

	return nil
}

var stopWords = "the i to a is and it you that for of in my but with on so not or at it's be have just as if was like they do i'm an this get when what are then one some will more would can should is are got have january february march april may june july august september october november december"

var reWords = regexp.MustCompile(`\S*`)

type Cloud struct {
	maxFontSize int
	height      int
	width       int
	keys        []string
	Freqs       map[string]int
}

func NewCloud() *Cloud {
	return &Cloud{
		Freqs:       make(map[string]int),
		maxFontSize: 72,
		height:      800,
		width:       600,
	}
}

func (c *Cloud) add(s string) {
	words := reWords.FindAllString(strings.ToLower(s), -1)

	for i := 0; i < len(words)-1; i++ {
		word := words[i]
		if strings.Contains(stopWords, word) {
			continue
		}
		next := fmt.Sprintf("%s %s", word, words[i+1])
		c.Freqs[word] = c.Freqs[word] + 1
		c.Freqs[next] = c.Freqs[next] + 1
	}

	final := words[len(words)-1]
	c.Freqs[final] = c.Freqs[final] + 1
}

func (c *Cloud) Len() int {
	return len(c.keys)
}

func (c *Cloud) Swap(a, b int) {
	c.keys[a], c.keys[b] = c.keys[b], c.keys[a]
}

func (c *Cloud) Less(a, b int) bool {
	left := c.Freqs[c.keys[a]]
	right := c.Freqs[c.keys[b]]

	if left == right {
		return c.keys[a] < c.keys[b]
	} else {
		return left > right
	}
}

func (c *Cloud) Sort() {
	c.keys = nil

	for key, _ := range c.Freqs {
		c.keys = append(c.keys, key)
	}
	sort.Sort(c)
}

func (c *Cloud) Print(w http.ResponseWriter) {
	c.Sort()

	for _, key := range c.keys {
		if c.Freqs[key] > 2 {
			fmt.Fprintf(w, "%s => %d\n", key, c.Freqs[key])
		}
	}

	w.Header().Set("Content-Type", "text/plain")
}

func (c *Cloud) ImagePng(w http.ResponseWriter) {
	c.Sort()
	//	maxCount := c.Freqs[c.keys[0]]
}
