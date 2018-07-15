package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

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

func main() {
	flag.Parse()

	var f []byte

	if flag.Arg(0) == "-" {
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		f = b
	} else {
		b, err := ioutil.ReadFile(flag.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
		f = b
	}

	var urls = []string{}

	lines := strings.Split(string(f), "\n")
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		urls = append(urls, parts[0], parts[1])
	}

	htmlTemplate.Execute(os.Stdout, urls)
}
