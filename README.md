
# find-duplicates
Small script for finding duplicate images in a directory tree.
Will compare their perceptual hashes


## Example

### Html
`find-duplicates -html /Users/Necro/Pictures > found.html`

### List
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
