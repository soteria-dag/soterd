// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package soterutil

import (
	"bytes"
	"fmt"
	"html/template"
	"os/exec"
)

// DotToSvg returns a rendering of the graphviz DOT file contents in SVG format
//
// This function makes use of the graphviz `dot` command, so graphviz needs to be installed.
// Graphviz: http://graphviz.org/
//
// NOTE(cedric): If you're embedding the svg file contents in an HTML document, you'll want to use the StripSvgXmlDecl
// function to strip the xml declaration tag from the svg contents before embedding the svg as a <figure>.
func DotToSvg(dot []byte) ([]byte, error) {
	var in, out, stderr bytes.Buffer

	cmdName := "dot"
	args := []string{"-Tsvg"}

	// Check if the graphviz "dot" program is available
	cmdPath, found := Which(cmdName)
	if !found {
		return out.Bytes(), fmt.Errorf("Couldn't find %s command in path. Is graphviz installed?", cmdName)
	}

	_, err := in.Write(dot)
	if err != nil {
		return out.Bytes(), err
	}

	// Run the `dot` command and pass the rendered dot file contents to its stdin
	cmd := exec.Command(cmdPath, args...)
	cmd.Stdin = &in
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return out.Bytes(), fmt.Errorf("%s\n%s", stderr.String(), err)
	}

	return out.Bytes(), nil
}


// RenderSvgHTML returns an HTML document containing the svg
func RenderSvgHTML(svg []byte, title string) ([]byte, error) {
       var render []byte
       var err error

       render, err = RenderSvgHTMLFigure(svg)
       if err != nil {
               return []byte{}, err
       }
       return (RenderHTML(render, title))
}

// RenderSvgHTMLFigure returns an HTML section containing the svg
func RenderSvgHTMLFigure(svg []byte) ([]byte, error) {
       var render bytes.Buffer

       tmpl := `
<figure>
{{ .svg }}
</figure>
`

       t := template.New("dag")
       t, err := t.Parse(tmpl)

       data := map[string]interface{}{
               "svg": template.HTML(svg),
       }

       err = t.Execute(&render, data)
       if err != nil {
               return []byte{}, err
       }

       return render.Bytes(), nil
}

// RenderSvgHTML returns an HTML document containing the svg
func RenderHTML(body []byte, title string) ([]byte, error) {
	var render bytes.Buffer

	tmpl := `
<html>
	<head>
		<title>{{ .title }}</title>
	</head>
	<body>
		{{ .body }}
	</body>
</html>
`

	t := template.New("dag")
	t, err := t.Parse(tmpl)

	data := map[string]interface{}{
		"title": title,
		"body": template.HTML(body),
	}

	err = t.Execute(&render, data)
	if err != nil {
		return []byte{}, err
	}

	return render.Bytes(), nil
}

// renderSteppingHTML returns an HTML document containing the body
// with stepping backward/forward hyperlinks
func RenderSteppingHTML(body []byte, title string, prev int, next int) ([]byte, error) {
       var render bytes.Buffer

       tmpl := `
<html> 
       <head>
               <title>{{ .title }}</title>
       </head>
       <body>
           <a href="dag_{{ .prev}}.html">prev</a>
           <a href="dag_{{ .next}}.html">next</a>
               {{ .body }}
               <a href="dag_{{ .prev}}.html">prev</a>
           <a href="dag_{{ .next}}.html">next</a>
       </body>
</html>
`

       t := template.New("dag")
       t, err := t.Parse(tmpl)

       data := map[string]interface{}{
               "title": title,
               "body": template.HTML(body),
               "prev": prev,
               "next": next,
       }

       err = t.Execute(&render, data)
       if err != nil {
               return []byte{}, err
       }

       return render.Bytes(), nil
}

// StripSvgXmlDecl is for stripping the xml declaration tag from svg file contents.
// It's available as a helper command in case you're embedding an svg file in an HTML document.
func StripSvgXmlDecl(svg []byte) ([]byte, error) {
	// Check that the bytes is an svg file, by looking for the opening svg xml element tag
	i := bytes.Index(svg, []byte("<svg"))
	if i < 0 {
		return []byte{}, fmt.Errorf("Svg tag not found in bytes")
	}

	// We return bytes starting from the <svg> xml tag, to cut-out the <?xml> declaration tag,
	// which isn't needed when the svg element will be rendered within another document (HTML file), instead of
	// being a stand-alone document.
	svg = svg[i:]

	return svg, nil
}