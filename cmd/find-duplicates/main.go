package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/nfnt/resize"
)

// Flags
var (
	NumThreads = flag.Int("threads", 8, "number of simultaneous operations")
	Similarity = flag.Int("similarity", 0, "similarity of image hashes")
	Verbose    = flag.Bool("v", false, "visualize hashes as an 8x8 text square")
	PrintHTML  = flag.Bool("html", false, "prints html output")
)

// DifferenceHash computes the dHash of an image
func DifferenceHash(img image.Image) uint64 {
	var hash uint64

	// Convert to grayscale
	ogray := image.NewGray(img.Bounds())
	draw.Draw(ogray, img.Bounds(), img, image.ZP, draw.Src)

	// Resize into an 9x8 square (64 bits) when you skip
	// The first pixel of every row
	gray := resize.Resize(9, 8, img, resize.NearestNeighbor)

	var c uint
	w, h := gray.Bounds().Dx(), gray.Bounds().Dy()
	var last uint32
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			clr, _, _, _ := gray.At(x, y).RGBA()

			// log.Println(zeroPad(strconv.FormatUint(hash, 2), 63))
			// Compare against pixel to left
			if x > 0 {
				if last < clr {
					hash |= (1 << c)
				}
				c++
			}

			last = clr
		}
	}

	return hash
}

// ImageHash is an image hash and its corresponding location
type ImageHash struct {
	Path string
	Hash uint64
}

func cut(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

func zeroPad(s string, max int) string {
	if len(s) < max {
		return strings.Repeat("0", max-len(s)) + s
	}
	return s
}

// LoadImage reads an image from the given path
func LoadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.New("Error loading image: " + path + ": " + err.Error())
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, errors.New("Error loading image: " + filepath.Base(path) + ": " + err.Error())
	}
	return img, err
}

// ErrSkipDir skips a dir
var ErrSkipDir = errors.New("Skip a directory")

// Walk recursively walks through a directory
//    dir   : directory to walk through
//    fn    : function called for every file
//            in the directory tree
func Walk(dir string, fn func(string, os.FileInfo) error) error {
	info, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, v := range info {
		if v.IsDir() {
			err = fn(dir, v)
			if err != nil {
				if err == ErrSkipDir {
					continue
				}
				return err
			}
			err = Walk(path.Join(dir, v.Name()), fn)
			if err != nil {
				return err
			}
		} else {
			err := fn(dir, v)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// VisualizeHash prints out a textual representation of the image hash
func VisualizeHash(hash uint64) {
	vhash := zeroPad(strconv.FormatUint(hash, 2), 64)
	for y := 0; y < 8; y++ {
		var line string
		for x := 0; x < 8; x++ {
			if vhash[y*8+x] == '1' {
				line += "- "
			} else {
				line += "x "
			}
		}
		log.Println(line)
	}
}

// HammingDistance calculates the hamming distance between two uint64s
func HammingDistance(a, b uint64) int {
	var diff int
	n := a ^ b
	for i := 0; i < 64; i++ {
		diff += int((n >> uint(i)) & 1)
	}
	return diff
}

func main() {
	flag.Parse()

	start := time.Now()

	tokens := make(chan struct{}, *NumThreads)
	for i := 0; i < *NumThreads; i++ {
		tokens <- struct{}{}
	}

	log.Println("Counting files")
	var numFiles int
	Walk(flag.Arg(0), func(_ string, info os.FileInfo) error {
		if info.IsDir() {
			return nil
		}
		numFiles++
		return nil
	})
	log.Println("Number of files to scan: ", numFiles)

	var wg sync.WaitGroup
	wg.Add(numFiles)

	var mu sync.Mutex
	images := []ImageHash{}
	err := Walk(flag.Arg(0), func(p string, info os.FileInfo) error {
		if info.IsDir() { // Can't read directories
			return nil
		}

		<-tokens
		go func() {
			defer func() {
				wg.Done()
				tokens <- struct{}{}
			}()
			fpath := filepath.Join(p, info.Name())
			img, err := LoadImage(fpath)
			if err != nil {
				log.Println(err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			images = append(images, ImageHash{
				Path: fpath,
				Hash: DifferenceHash(img),
			})
			log.Printf("%-100s%-70s%d\n",
				cut(filepath.Base(images[len(images)-1].Path), 90),
				zeroPad(strconv.FormatUint(images[len(images)-1].Hash, 2), 63),
				images[len(images)-1].Hash,
			)
			if *Verbose {
				VisualizeHash(images[len(images)-1].Hash)
			}
		}()

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	wg.Wait()

	duplicates := []ImageHash{}
	for i, v := range images {
	skip:
		for j, y := range images {
			if i == j {
				continue
			}
			for t := 0; t < len(duplicates); t += 2 { // Don't add an already existing duplicate pair
				a := duplicates[t]
				b := duplicates[t+1]
				if (v.Path == a.Path && y.Path == b.Path) ||
					(v.Path == b.Path && y.Path == a.Path) {
					continue skip
				}
			}
			if HammingDistance(v.Hash, y.Hash) <= *Similarity {
				duplicates = append(duplicates, v, y)
			}
		}
	}

	log.Printf("Found [%d] duplicates in [%d] files in %s", len(duplicates)/2, numFiles, time.Since(start))

	if *PrintHTML {
		urls := []string{}
		for _, v := range duplicates {
			abs, err := filepath.Abs(v.Path)
			if err != nil {
				log.Println(err)
				continue
			}
			urls = append(urls, abs)
		}
		htmlTemplate.Execute(os.Stdout, urls)
	} else {
		for i := 0; i < len(duplicates)/2; i += 2 {
			absA, err := filepath.Abs(duplicates[i].Path)
			if err != nil {
				log.Println(err)
			}
			absB, err := filepath.Abs(duplicates[i+1].Path)
			if err != nil {
				log.Println(err)
			}

			fmt.Printf("%s\t%s\n", absA, absB)
		}
	}
}

var htmlTemplate = template.Must(template.New("").Funcs(
	template.FuncMap{
		"escape": func(p string) string {
			n := []string{}
			p = filepath.ToSlash(p)
			for _, v := range strings.Split(p, "/") {
				n = append(n, url.PathEscape(v))
			}
			return strings.Join(n, "/")
		},
	},
).Parse(`<html>
<head>
	<title>Images</title>
	<link href="{{ index . 0 }}" rel="shortcut icon" />
	<style>
		body {
			background-color: #232323;
			color: white;
		}
		.grid {
			display: grid;
			width: 100%;
			grid-template-columns: 500px 500px;
		}
		
		.container {
			display: grid;
			border: none;
			color: white;
			padding-right: 30px;
			padding-bottom: 30px;
		}

		.container:hover {
			background-color: purple;
			cursor: pointer;
		}
		
		img {
			width: 300px;
		}

		code {
			color: white;
			background-color: #252525;
			width: 100%;
		}
	</style>
</head>
<body>
<h3>Click an image to copy the path to your clipboard</h4>
<div class="grid">
{{- range . }}
	<div class="container">
		<span class="path">{{.}}</span>
		<img src="file:///{{escape .}}">
	</div>
{{- end }}
</div>
<pre>
	<code>
	{{- range . }}
	{{.}}
	{{- end }}
	</code>
</pre>
<script>
	var containers = document.body.getElementsByClassName("container");
	Array.from(containers).forEach(function(e) {
		console.log(e);
		e.addEventListener('click', function() {
			var path = e.getElementsByClassName("path").item(0);
			var text = path.innerHTML;
			var input = document.createElement("input");
			input.value = text;
			document.body.appendChild(input);
			input.select();
			document.execCommand('copy');
			document.body.removeChild(input);
		});
	});
</script>
</body>
</html>
`))
