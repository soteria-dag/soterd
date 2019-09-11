// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"github.com/soteria-dag/soterd/soterutil"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/integration/rpctest"
)

// save bytes to a file descriptor
func saveHTML(bytes []byte, fh *os.File) error {
	_, err := fh.Write(bytes)
	if err != nil {
		return err
	}

	err = fh.Close()
	if err != nil {
		return err
	}

	return nil
}

//
// runNet runs a network of miners, generates some blocks on them, taking snapshots at an interval
// and renders the dag as html at each interval
//
func runNet(minerCount int, blockTime int, 
			timeSpan int, stepInterval int, 
			runDuration int, 
			output string, 
			rankdir string, 
			keepLogs bool) (string, error) {
	
	var miners []*rpctest.Harness
	var err error

	extraArgs := []string{}

	dagNetParams := chaincfg.SimNetParams

	// if time span needs to be changed
	if (timeSpan != 0) {
		dagNetParams.TargetTimespan = time.Second * time.Duration(timeSpan)
	}

	// if block time needs to be changed
	if (blockTime != 0) {
		dagNetParams.TargetTimePerBlock = time.Millisecond * time.Duration(blockTime)
	}

	// Spawn miners
	for i := 0; i < minerCount; i++ {
		miner, err := rpctest.New(&dagNetParams, nil, extraArgs, keepLogs)
		if err != nil {
			return "", fmt.Errorf("unable to create mining node %d: %s", i, err)
		}

		if err := miner.SetUp(false, 0); err != nil {
			return "", fmt.Errorf("unable to complete mining node %d setup: %s", i, err)
		}

		if keepLogs {
			fmt.Printf("miner %d log dir: %s\n", i, miner.LogDir())
		}

		miners = append(miners, miner)
	}

	// NOTE(cedric): We'll call defer on a single anonymous function instead of minerCount times in the above loop
	defer func() {
		for _, miner := range miners {
			_ = (*miner).TearDown()
		}
	}()

	// Connect the nodes to one another
	err = rpctest.ConnectNodes(miners)
	if err != nil {
		return "", fmt.Errorf("unable to connect nodes: %s", err)
	}


	// Start mining on each miner
	for i, miner := range miners {
		err := miner.Node.SetGenerate(true, 1)
		if err != nil {
			fmt.Printf("failed to start miner on node %v: %v\n", i, err)
		}
	}

	// Recording the stepping process  
	var stepDots [][]byte
	stepCount := 0
	
	if (stepInterval != 0) {
		// Stepping if specified
		timeStart := time.Now() 
		for {
			fmt.Println("Generating Step", stepCount)
			// Render the dag in graphviz DOT file format
			dot, err := rpctest.RenderDagsDot(miners, rankdir)
			if err != nil {
				return "", fmt.Errorf("failed to render dag in graphviz DOT format: %s", err)
			}
			stepDots = append(stepDots, dot)
			timeNow := time.Now()
			timeDuration := time.Duration(runDuration) * time.Second
			time.Sleep(time.Duration(stepInterval) * time.Millisecond)
			fmt.Printf("Generating for %v\n", timeNow.Sub(timeStart))
			if (timeNow.Sub(timeStart) > timeDuration) {
				break
			}
			stepCount++
		}
	} else {
		time.Sleep(time.Duration(runDuration) * time.Second)
	}

	// Stop mining on each node
	for i, miner := range miners {
		err := miner.Node.SetGenerate(false, 0)
		if err != nil {
			fmt.Printf("failed to stop miner on node %v: %v\n", i, err)
		}
	}

	// Finalize the generation 
	fmt.Println("Finalizing")

	// Take a snap shot of the final state
	dot, err := rpctest.RenderDagsDot(miners, rankdir)
	if err != nil {
		return "", fmt.Errorf("failed to render dag in graphviz DOT format: %s", err)
	}
	stepDots = append(stepDots, dot)

	// Determine where we will save the dag steps
	var outDir string

	if len(output) == 0 {
		// Create a temporary dir as output path
		outDir, err = ioutil.TempDir("", "dagviz")
	} else {
		info, pathErr := os.Stat(output)
		if pathErr == nil && info.IsDir() {
			// output path already exists
			outDir = output
		} else {
			// Create the output path
			err = os.MkdirAll(output, 0755)
		}
	}
	if err != nil {
		return "", err
	}

	// Start the rendering process 
	for step := 0; step <= stepCount; step++ {

		fmt.Println("Rendering Step", step)

		// Render the dag in graphviz DOT file format
		dot := stepDots[step]

		// Convert DOT file contents to an SVG image
		svg, err := soterutil.DotToSvg(dot)
		if err != nil {
			return "", fmt.Errorf("failed to convert DOT file to SVG: %s", err)
		}
		
		// We're going to embed the SVG image in HTML, so strip out the xml declaration
		svgEmbed, err := soterutil.StripSvgXmlDecl(svg)
		if err != nil {
			return "", fmt.Errorf("failed to strip xml declaration from SVG image: %s", err)
		}
		
		svgBody, err := soterutil.RenderSvgHTMLFigure(svgEmbed)

		// Render the dag in an HTML document
		h, err := soterutil.RenderSteppingHTML(svgBody, "dag", step-1, step+1)
		if err != nil {
			return "", fmt.Errorf("failed to render SVG image as HTML: %s", err)
		}

		pattern := "dag_" + strconv.Itoa(step) + ".html"
		name := filepath.Join(outDir, pattern)
		fh, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return "", fmt.Errorf("failed to create output file-handle: %s", err)
		}

		// Save the HTML document
		err = saveHTML(h, fh)
		if err != nil {
			return "", fmt.Errorf("failed to save HTML file: %s", err)
		}
	}

	return outDir + "/dag_0.html", nil
}

func main() {
	var err error
	var htmlFile string 

	var stepping bool
	var output string
	var rankdir string
	var nodeCount int

	var runDuration int
	var blockTime int
	var timeSpan int

	var stepInterval int

	var keepLogs bool


	// parsing the command line parameters
	flag.StringVar(&output, "output", "", "Where to save the rendered dag")
	flag.StringVar(&rankdir, "rankdir","TB", "Orientation of the graph: TB, BT, LR, RL")
	flag.BoolVar(&stepping, "stepping", false, "Generating Stepping Results")

	flag.IntVar(&nodeCount, "nodes", 4, "Number of Nodes")
	flag.IntVar(&runDuration, "duration", 20, "Duration of the Run in seconds")
	flag.IntVar(&stepInterval, "interval", 100, "Interval in milliseconds between each step")

	flag.IntVar(&blockTime, "blocktime", 0, "Changing Mining Block Time in milliseconds")
	flag.IntVar(&timeSpan, "timespan", 0, "Changing Mining Time Span in seconds")

	flag.BoolVar(&keepLogs, "l", false, "Keep logs from soterd nodes")

	flag.Parse()

	// validate params
	if ((blockTime != 0 || timeSpan != 0) && blockTime > (timeSpan * 1000)) {
		fmt.Println("Invalid parameters: -blocktime can not be greater than -timespan.")
		syscall.Exit(1)
	}

	// everything seems alright. Let's run
	fmt.Printf("Generating dag with %d nodes for %d seconds\n", nodeCount, runDuration)
	fmt.Printf("Node Profile: block time %d msec, time span %d sec\n", blockTime, timeSpan)

	if (stepping) {
		fmt.Printf("Taking snapshots for %d seconds with %d msec interval\n", runDuration, stepInterval)
		htmlFile, err = runNet(nodeCount, blockTime, timeSpan, stepInterval, runDuration, output, rankdir, keepLogs)
	} else {
		htmlFile, err = runNet(nodeCount, blockTime, timeSpan, 0, runDuration, output, rankdir, keepLogs)
	}

	if err != nil {
		fmt.Println(err)
		syscall.Exit(1)
	}

	fmt.Println("Saved dag to", htmlFile)
}
