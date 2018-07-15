
# find-duplicates
Small script for finding duplicate images in a directory tree.
Will compare their perceptual hashes using the difference hash algorithm

```go
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
```


## Example

#### html

`find-duplicates -html /Users/Necro/Pictures > found.html`

#### list

`find-duplicates /Users/Necro/Pictures > found.txt`

## Installing
`go get -u github.com/Necroforger/cmd/find-duplicates`

## Arguments

| Flag       | Description                                      |
|------------|--------------------------------------------------|
| threads    | number of goroutines to use                      |
| similarity | maximum difference of hashes to indicate match   |
| v          | visualize hashes while searching                 |
| html       | output in html format so you can view the images |
