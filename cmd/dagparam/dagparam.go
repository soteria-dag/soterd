// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/soteria-dag/soterd/soterutil"
	"html/template"
	"io/ioutil"
	"log"
	"math/big"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/soteria-dag/soterd/blockdag"
	"github.com/soteria-dag/soterd/chaincfg"
	"github.com/soteria-dag/soterd/integration/rpctest"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
)

const (
	defaultMinerCount = 4
	defaultMineTime = time.Second * 30
	defaultWait = time.Minute
)

var (
	// Set to debug or trace to produce more logging output from miners.
	// Note that you'll also want to have the test harness not clean up data/log files after the test suite runs.
	defaultSoterdExtraArgs = []string{
		//"--debuglevel=debug",
	}
)

// Metrics relating to the node that we're interested in
type nodeMetrics struct {
	// Block-generation metrics
	blkGenCount int64
	blkGenTimes []float64
	blkHashes []string

	// DAG metrics
	multiParentBlocks int
	meanMultiParentBlockDistance float64
}

// Represent the information gathered from running a network of nodes
type runResult struct {
	// Profile used for run
	profile *runProfile

	// Metrics for each miner
	nodeStats map[int]*nodeMetrics

	// Graph of DAG produced by test
	dagSvg []byte

	// TODO: Propagation time of blocks

	dagSynced bool
}

// Represent parameter changes, and measurements from running a network with those changes
type changeResult struct {
	fields  []string
	amounts []int64
	units   []string
	result  *runResult
}

// Represent a collection of changes and their metrics from the network
type resultSet struct {
	changes []*changeResult
}

type runProfile struct {
	// DAG Network parameters to use
	dagNetParams chaincfg.Params

	// Modifications to make, to dagNetParams
	dagNetMods []func(*chaincfg.Params) error

	// Number of miners to spawn
	minerCount int

	// How long to let miners mine for
	mineTime time.Duration

	// Max time to wait for blocks to sync between nodes, once all are generated
	wait time.Duration

	// Any extra args to pass to soterd processes
	soterdExtraArgs []string

	// If logs should be kept from soterd nodes
	soterdKeepLogs bool
}

// absInt returns the absolute value of x
func absInt(x int) int {
	if x < 0 {
		return -x
	} else {
		return x
	}
}

// meanFloat64 returns the mean value of the values in x
func meanFloat64(x []float64) float64 {
	total := float64(0)

	if len(x) == 0 {
		return total
	}

	for _, v := range x {
		total += v
	}

	mean := total / float64(len(x))

	return mean
}

// pctBigInt returns x value changed by pct amount
func pctBigInt(x *big.Int, pct int) (n *big.Int) {
	bigHundred := big.NewInt(100)
	bigPct := big.NewInt(int64(absInt(pct)))
	change := new(big.Int).Mul(
		new(big.Int).Div(x, bigHundred),
		bigPct)

	if pct < 0 {
		// Decrease value
		n = new(big.Int).Sub(x, change)
	} else {
		// Increase value
		n = new(big.Int).Add(x, change)
	}

	return n
}

// save bytes to a dynamically-named file based on the provided pattern
func save(bytes []byte, pattern string) (string, error) {
	t, err := ioutil.TempFile("", pattern)
	if err != nil {
		return "", err
	}

	_, err = t.Write(bytes)
	if err != nil {
		return "", err
	}

	err = t.Close()
	if err != nil {
		return "", err
	}

	return t.Name(), nil
}

// modPowLimit modifies the PowLimit and PowLimitBits values by pct amount
func modPowLimit(params *chaincfg.Params, pct int) {
	params.PowLimit = pctBigInt(params.PowLimit, pct)
	params.PowLimitBits = blockdag.BigToCompact(params.PowLimit)
}

// waitForDagTick calls rpctest.WaitForDAG, and outputs dots at regular intervals. After wait time is reached,
// the function returns whether or not rpctest.WaitForDAG completed.
func waitForDAGTick(nodes []*rpctest.Harness, wait time.Duration) {
	threshold := time.Now().Add(wait)
	tick := time.Tick(time.Second * 5)
	stop := make(chan int)

	go func() {
		_ = rpctest.WaitForDAG(nodes, wait)
		close(stop)
	}()

	for {
		select {
			case <-tick:
				if time.Now().After(threshold) {
					fmt.Println("!")
					return
				} else {
					fmt.Print(".")
				}
			case <-stop:
				fmt.Println()
				return
		}
	}
}

// amountStrings returns a list of strings for the change amounts
func (c *changeResult) amountStrings() []string {
	s := make([]string, 0)
	for _, amount := range c.amounts {
		s = append(s, fmt.Sprintf("%d", amount))
	}

	return s
}

// changeStrings returns a list of strings for the change and values
func (c *changeResult) changeStrings() []string {
	s := make([]string, 0)
	for i, field := range c.fields {
		s = append(s, fmt.Sprintf("%s %d%s", field, c.amounts[i], c.units[i]))
	}

	return s
}

// filePattern returns a string relating to the changeResult, that could be used with ioutil.TempFile to generate a file
func (c *changeResult) filePattern(prefix string) string {
	return fmt.Sprintf("%s_%s_%s_*.svg",
		prefix,
		strings.Join(c.fields, "-"),
		strings.Join(c.amountStrings(), "_"))
}

// GraphNodeBlkGen returns a bar graph of blocks generated per node
func (c *changeResult) GraphNodeBlkGen() ([]byte, error) {
	var svg bytes.Buffer

	title := fmt.Sprintf("Blocks Generated Per Node (%s)",
		strings.Join(c.changeStrings(), ", "))

	values := make([]chart.Value, 0)
	for _, m := range c.result.miners() {
		v := chart.Value{
			Label: fmt.Sprintf("node %d", m),
			Value: float64(c.result.minerBlockCount(m)),
		}
		values = append(values, v)
	}

	ch := chart.BarChart{
		Title: title,
		TitleStyle: chart.StyleShow(),
		XAxis: chart.StyleShow(),
		YAxis: chart.YAxis{
			Name: "block count",
			NameStyle: chart.StyleShow(),
			Style: chart.StyleShow(),
		},
		Bars: values,
	}

	err := ch.Render(chart.SVG, &svg)
	if err != nil {
		return svg.Bytes(), err
	}

	return svg.Bytes(), nil
}

// Print prints the changeResult
func (c *changeResult) Print() {
	fmt.Printf("blocks (total %v):\n", c.result.totalBlockCount())
	for _, m := range c.result.miners() {
		fmt.Printf("\tminer %v\tblkGen count %v\tblkGen mean %v ms\n",
			m, c.result.minerBlockCount(m), c.result.minerMeanGenTime(m))
	}
}

// SaveNodeBlkGen saves a bar graph of blocks generated per node to an svg file
func (c *changeResult) SaveNodeBlkGen() (string, error) {
	svg, err := c.GraphNodeBlkGen()
	if err != nil {
		return "", err
	}

	return save(svg, c.filePattern("nodeBlkGen"))
}

// SaveDAG saves the dag of the change result to an svg file
func (c *changeResult) SaveDAG() (string, error) {
	return save(c.result.dagSvg, c.filePattern("dag"))
}


// SaveGraphs saves all graphs related to the change
func (c *changeResult) SaveGraphs() (string, string, error) {
	d, err := c.SaveDAG()
	if err != nil {
		return "", "", err
	}

	bg, err := c.SaveNodeBlkGen()
	if err != nil {
		return "", "", err
	}

	return d, bg, nil
}

// newRunResult returns an instance of runResult type
func newRunResult() runResult {
	return runResult{
		nodeStats: make(map[int]*nodeMetrics),
	}
}

// minerBlockCount returns the number of blocks that the miner generated
func (rr *runResult) minerBlockCount(m int) int64 {
	return rr.nodeStats[m].blkGenCount
}

// minerMeanGenTime returns the mean block-generation time in milliseconds for the miner
func (rr *runResult) minerMeanGenTime(m int) float64 {
	return meanFloat64(rr.nodeStats[m].blkGenTimes)
}

// meanMpbDistance returns the mean multi-parent block distance across nodes.
// This will be a mean of means, since each node's value is also a mean.
func (rr *runResult) meanMpbDistance() float64 {
	d := make([]float64, 0)
	for _, ns := range rr.nodeStats {
		d = append(d, ns.meanMultiParentBlockDistance)
	}

	return meanFloat64(d)
}

// miners returns the miners in the runResult type
func (rr *runResult) miners() []int {
	var miners []int
	for m := range rr.nodeStats {
		miners = append(miners, m)
	}

	sort.Ints(miners)

	return miners
}

// totalBlockCount returns the total number of blocks generated
func (rr *runResult) totalBlockCount() int64 {
	total := int64(0)
	for _, ns := range rr.nodeStats {
		total += ns.blkGenCount
	}

	return total
}

// totalMpbCount returns the total number of multi-parent blocks
func (rr *runResult) totalMpbCount() int {
	total := 0
	for _, ns := range rr.nodeStats {
		total += ns.multiParentBlocks
	}

	return total
}

// newResultSet returns a new instance of resultSet type
func newResultSet() resultSet {
	return resultSet{
		changes: make([]*changeResult, 0),
	}
}

// GraphBlkGenTime returns scatter plots of mean block-generation time across changes in the result set.
// There is one scatter plot per change, because the x-axis is used to plot the change values across the result set.
// x-axis will have the change values across the result set
// y-axis will contain the mean gen time data-points for nodes within a change-result
func (rs *resultSet) GraphMeanBlkGenTime() ([][]byte, error) {
	if len(rs.changes) == 0 {
		err := fmt.Errorf("No changes in result-set to graph!")
		return [][]byte{}, err
	}

	// Create lists of things per change
	// NOTE(cedric): Here we're assuming that the same set of changes is consistent
	titles := make([]string, 0)
	xvalues := make([][]float64, 0)
	yvalues := make([][]float64, 0)
	//gridLines := make([]chart.GridLine, 0)
	for _, chgString := range rs.changes[0].fields {
		t := fmt.Sprintf("Mean Block-Generation Time in ms (%s)", chgString)
		titles = append(titles, t)
		xvalues = append(xvalues, make([]float64, 0))
		yvalues = append(yvalues, make([]float64, 0))

		//gl := chart.GridLine{
		//	IsMinor: false,
		//	Style: chart.StyleShow(),
		//	Value: float64(rs.fields[0].amounts[i]),
		//}
		//gridLines = append(gridLines, gl)
	}

	// Populate x, y axis values for each change.
	// There are multiple entries per change value, one per node in the network
	for _, cr := range rs.changes {
		for i, v := range cr.amounts {
			// There should be one change-value entry for each node in the change-result
			for j := 0; j < len(cr.result.miners()); j++ {
				xvalues[i] = append(xvalues[i], float64(v))
			}

			for _, m := range cr.result.miners() {
				yvalues[i] = append(yvalues[i], float64(cr.result.minerMeanGenTime(m)))
			}
		}
	}

	// This is used for coloring the y-values
	viridisByY := func(xr, yr chart.Range, index int, x, y float64) drawing.Color {
		return chart.Viridis(y, yr.GetMin(), yr.GetMax())
	}

	// Render the charts
	svgs := make([][]byte, 0)
	for i, t := range titles {
		ch := chart.Chart{
			Title: t,
			TitleStyle: chart.StyleShow(),
			XAxis: chart.XAxis{
				Name: fmt.Sprintf("%s (in %s)", rs.changes[0].fields[i], rs.changes[0].units[i]),
				NameStyle: chart.StyleShow(),
				Style: chart.StyleShow(),
				//GridLines: gridLines,
				GridMinorStyle: chart.Style{
					Show: false,
				},
			},
			YAxis: chart.YAxis{
				Name: "block-gen time (in ms)",
				NameStyle: chart.StyleShow(),
				Style: chart.StyleShow(),
			},
			Series: []chart.Series{
				chart.ContinuousSeries{
					Style: chart.Style{
						Show: true,
						StrokeWidth: chart.Disabled,
						DotWidth: 5,
						DotColorProvider: viridisByY,
					},
					XValues: xvalues[i],
					YValues: yvalues[i],
				},
			},
		}

		var svg bytes.Buffer
		err := ch.Render(chart.SVG, &svg)
		if err != nil {
			return [][]byte{}, err
		}

		svgs = append(svgs, svg.Bytes())
	}

	return svgs, nil
}

// saveDagHTML saves an HTML document representing the dag, and returns the file name
func saveDagHTML(svg []byte) (string, error) {
	html, err := soterutil.RenderSvgHTML(svg, "dag")
	if err != nil {
		return "", err
	}

	return save(html, "dag_*.html")
}

// RenderHTML returns an HTML document representing the result set
func (rs *resultSet) RenderHTML() ([]byte, error) {
	var render bytes.Buffer

	tmpl := `
<html> 
	<head>
		<title>{{ .title }}</title>
	</head>
	<body>
		<h2>Mean block-generation times (one graph per value-changed)</h3>
		{{- range .mbgtGraphs -}}
		<figure>
			{{ . }}
		</figure>
		{{- end}}
		<br>
		<br>
		<h2>Per-change graphs</h3>
		{{- range .crGraphs -}}
			<h3>{{ .Title }}</h4>
			<p>dag synced? {{ .DagSynced }}</p>
			<a href="{{ .DagFile }}">dag</a>
			<figure>
				{{ .BlkGenGraph }}
			</figure>
		{{- end}}
	</body>
</html>
`
	t := template.New("dagparam result set")
	t, err := t.Parse(tmpl)

	svgs, err := rs.GraphMeanBlkGenTime()
	if err != nil {
		return []byte{}, err
	}
	htmlSvgs := make([]template.HTML, 0)
	for _, svg := range svgs {
		htmlSvgs = append(htmlSvgs, template.HTML(svg))
	}

	crSvgs := make([]struct{
		Title string
		DagFile string
		BlkGenGraph template.HTML
		DagSynced bool
	}, 0)
	for i, cr := range rs.changes {
		title := fmt.Sprintf("Run %d changes (%s)",
			i, strings.Join(cr.changeStrings(), ", "))

		dagFile, err := saveDagHTML(cr.result.dagSvg)
		if err != nil {
			return []byte{}, err
		}
		fmt.Println("saved dag:", dagFile)

		// Use relative path when linking between results and change-result dag figure
		relDagFile := filepath.Base(dagFile)

		svg, err := cr.GraphNodeBlkGen()
		if err != nil {
			return []byte{}, err
		}
		crSvgs = append(crSvgs, struct{
			Title string
			DagFile string
			BlkGenGraph template.HTML
			DagSynced bool
		}{
			Title: title,
			DagFile: relDagFile,
			BlkGenGraph: template.HTML(svg),
			DagSynced: cr.result.dagSynced,
		})
	}

	data := map[string]interface{}{
		"title": "dagparam result-set",
		"mbgtGraphs": htmlSvgs,
		"crGraphs": crSvgs,
	}

	err = t.Execute(&render, data)
	if err != nil {
		return []byte{}, err
	}

	return render.Bytes(), nil
}

// SaveHTML saves an HTML document representing the result set, and returns the file name
func (rs *resultSet) SaveHTML() (string, error) {
	html, err := rs.RenderHTML()
	if err != nil {
		return "", err
	}

	return save(html, "resultSet_*.html")
}


// SaveMeanBlkGenTime saves scatter plots of mean block-generation time across changes in the result set.
// There is one scatter plot per change, because the x-axis is used to plot the change values across the result set.
func (rs *resultSet) SaveMeanBlkGenTime() ([]string, error) {
	svgs, err := rs.GraphMeanBlkGenTime()
	if err != nil {
		return []string{}, err
	}

	fileNames := make([]string, 0)
	for _, svg := range svgs {
		name, err := save(svg, "meanBlkGenTime_*.svg")
		if err != nil {
			return fileNames, err
		}

		fileNames = append(fileNames, name)
	}

	return fileNames, nil
}

// SaveGraphs saves all graphs related to the resultSet
func (rs *resultSet) SaveGraphs() ([]string, error) {
	fileNames := make([]string, 0)

	mbgt, err := rs.SaveMeanBlkGenTime()
	if err != nil {
		return fileNames, err
	}

	fileNames = append(fileNames, mbgt...)

	// Save change-specific graphs for each change in the result-set
	for _, cr := range rs.changes {
		d, bg, err := cr.SaveGraphs()
		if err != nil {
			return fileNames, err
		}

		fileNames = append(fileNames, d, bg)
	}

	return fileNames, nil
}

// newRunProfile returns a new instance of runProfile with default values
func newRunProfile() runProfile {
	return runProfile{
		dagNetParams: chaincfg.SimNetParams,
		dagNetMods: make([]func(*chaincfg.Params) error, 0),
		minerCount: defaultMinerCount,
		mineTime: defaultMineTime,
		wait: defaultWait,
		soterdExtraArgs: defaultSoterdExtraArgs,
	}
}

// runNet runs a network of nodes with the given changes to parameters, and returns the result of the run
func runNet(profile runProfile) (runResult, error) {
	var miners []*rpctest.Harness
	result := newRunResult()

	// Modify params based on profile passed to runNet
	for _, modFn := range profile.dagNetMods {
		err := modFn(&profile.dagNetParams)
		if err != nil {
			return result, err
		}
	}

	result.profile = &profile

	// Spawn miners
	for i := 0; i < profile.minerCount; i++ {
		miner, err := rpctest.New(&profile.dagNetParams, nil, profile.soterdExtraArgs, profile.soterdKeepLogs)
		if err != nil {
			fmt.Printf("unable to create mining node %v: %v\n", i, err)
			return result, err
		}

		if err := miner.SetUp(false, 0); err != nil {
			fmt.Printf("unable to complete mining node %v setup: %v\n", i, err)
			return result, err
		}

		if profile.soterdKeepLogs {
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

	// TODO: Apply connectivity profile that was passed to runNet
	// Connect the nodes to one another
	err := rpctest.ConnectNodes(miners)
	if err != nil {
		fmt.Printf("unable to connect nodes: %v\n", err)
		return result, err
	}

	// Put each miner in generate mode
	for i, miner := range miners {
		err := miner.Node.SetGenerate(true, 1)
		if err != nil {
			fmt.Printf("failed to start miner on node %v: %v\n", i, err)
		}
	}

	// Let miners generate blocks
	fmt.Println("letting nodes generate blocks")
	time.Sleep(profile.mineTime)

	// Stop mining on each node
	for i, miner := range miners {
		err := miner.Node.SetGenerate(false, 0)
		if err != nil {
			fmt.Printf("failed to stop miner on node %v: %v\n", i, err)
		}
	}


	fmt.Println("waiting for blocks to sync between nodes")
	waitForDAGTick(miners, profile.wait)

	// Collect metrics from miners
	fmt.Println("collecting metrics from nodes")
	for i, miner := range miners {
		resp, err := miner.Node.GetBlockMetrics()
		if err != nil {
			fmt.Printf("failed to getblockmetrics on miner %v: %v\n", i, err)
			return result, err
		}

		metrics := nodeMetrics{
			blkGenCount: resp.BlkGenCount,
			blkGenTimes: resp.BlkGenTimes,
			blkHashes: resp.BlkHashes,
		}

		// Count multi-parent blocks and distance from one another
		mpbCount, mpbMeanDistance, err := rpctest.CountDAGBlocks(miner)
		if err != nil {
			fmt.Printf("failed to count multi-parent blocks on miner %v: %v", i, err)
			return result, err
		}

		metrics.multiParentBlocks = mpbCount
		metrics.meanMultiParentBlockDistance = mpbMeanDistance

		result.nodeStats[i] = &metrics
	}

	// Render the dag in graphviz DOT file format
	dot, err := rpctest.RenderDagsDot(miners, "TB")
	if err != nil {
		return result, err
	}

	// Convert DOT file contents to an SVG image
	svg, err := soterutil.DotToSvg(dot)
	if err != nil {
		return result, err
	}

	// We're going to embed the SVG image in HTML, so strip out the xml declaration
	svgEmbed, err := soterutil.StripSvgXmlDecl(svg)
	if err != nil {
		return result, err
	}

	result.dagSvg = svgEmbed

	// Check if DAG on all miners is identical
	fmt.Println("comparing dag between nodes")
	err = rpctest.CompareDAG(miners)
	if err != nil {
		result.dagSynced = false
		fmt.Printf("dag not identical between all nodes :(\t%s\n", err)
	} else {
		result.dagSynced = true
		fmt.Println("dag same on all nodes :)")
	}

	return result, nil
}

// runModPowLimit runs a network with TargetTimespan and TargetTimePerBlock changed
func runModTargetTime(targetTimespan time.Duration, targetTimePerBlock time.Duration, keepLogs bool) (*changeResult, error) {
	// Create a profile for this parameter run
	profile := newRunProfile()

	profile.soterdKeepLogs = keepLogs

	// Modify SimNetParams for test
	timeMod := func (params *chaincfg.Params) error {
		params.TargetTimespan = targetTimespan
		params.TargetTimePerBlock = targetTimePerBlock
		return nil
	}

	profile.dagNetMods = append(profile.dagNetMods, timeMod)

	fmt.Printf("run with TargetTimeSpan %s\tTargetTimePerBlock %s\n", targetTimespan, targetTimePerBlock)
	result, err := runNet(profile)
	if err != nil {
		newErr := fmt.Errorf("runNet: %v", err)
		// We'll retry the run instead of aborting the whole suite of runs
		return nil, newErr
	}

	change := changeResult{
		fields:  []string{"TargetTimeSpan", "TargetTimePerBlock"},
		amounts: []int64{int64(targetTimespan.Seconds() * 1000), int64(targetTimePerBlock.Seconds() * 1000)},
		units: []string{"ms", "ms"},
		result:  &result,
	}

	return &change, nil
}

// runModPowLimit runs a network with PowLimit and PowLimitBits changed
func runModPowLimit(pctChange int, keepLogs bool) (*changeResult, error) {
	// Create a profile for this parameter run
	profile := newRunProfile()

	profile.soterdKeepLogs = keepLogs

	// Modify SimNetParams for test
	powMod := func (params *chaincfg.Params) error {
		modPowLimit(params, pctChange)
		return nil
	}

	profile.dagNetMods = append(profile.dagNetMods, powMod)

	fmt.Printf("run with PowLimit and PowLimitBits modified by %v%%\n", pctChange)
	result, err := runNet(profile)
	if err != nil {
		newErr := fmt.Errorf("runNet: %v", err)
		// We'll retry the run instead of aborting the whole suite of runs
		return nil, newErr
	}

	change := changeResult{
		fields:  []string{"PowLimit"},
		amounts: []int64{int64(pctChange)},
		units:   []string{"%"},
		result:  &result,
	}

	return &change, nil
}

// exploreBlockTargetTime runs miners with TargetTimespan and TargetTimePerBlock changed from a series of value-pairs.
func exploreBlockTargetTime(keepLogs bool) resultSet {
	results := newResultSet()

	type timePair struct {
		targetTimespan, targetTimePerBlock time.Duration
	}

	// Series of changed values we want to try out
	// NOTE(cedric): The TargetTimespan value must always be *larger* than the TargetTimePerBlock value.
	// This is because blockdag.calcNextRequiredDifficulty modulos a value against BlockDAG.blocksPerRetarget, and
	// blocksPerRetarget := targetTimespan / targetTimePerBlock
	// If TargetTimespan is lower than TargetTimePerBlock, this will trigger a divide-by-zero panic because the
	// fractional value will round down to zero.
	pairs := []timePair{
		// NOTE(cedric): 1 Millisecond is currently the minimum targetTimePerBlock that is supported by soterd, because
		// difficulty calculations use millisecond precision (x / time.Millisecond)
		{
			targetTimespan:     time.Second,
			targetTimePerBlock: time.Millisecond,
		},
		{
			targetTimespan:     time.Second,
			targetTimePerBlock: time.Millisecond * 2,
		},
		{
			targetTimespan:     time.Second,
			targetTimePerBlock: time.Millisecond * 4,
		},
		{
			targetTimespan:     time.Second,
			targetTimePerBlock: time.Millisecond * 8,
		},
		{
			targetTimespan:     time.Second,
			targetTimePerBlock: time.Millisecond * 16,
		},
		{
			targetTimespan:     time.Second,
			targetTimePerBlock: time.Millisecond * 32,
		},
		{
			targetTimespan:     time.Second,
			targetTimePerBlock: time.Millisecond * 64,
		},
		{
			targetTimespan:     time.Second,
			targetTimePerBlock: time.Millisecond * 128,
		},
		{
			targetTimespan:     time.Second * 5,
			targetTimePerBlock: time.Millisecond * 250,
		},
		{
			targetTimespan:     time.Second * 5,
			targetTimePerBlock: time.Millisecond * 500,
		},
		{
			targetTimespan:     time.Second * 10,
			targetTimePerBlock: time.Second,
		},
		{
			targetTimespan:     time.Second * 10,
			targetTimePerBlock: time.Second * 2,
		},
		{
			targetTimespan:     time.Second * 20,
			targetTimePerBlock: time.Second * 5,
		},
		{
			targetTimespan:     time.Second * 20,
			targetTimePerBlock: time.Second * 10,
		},
		// The defaultMineTime is too short for testing these values
		//{
		//	targetTimespan:     time.Second * 60,
		//	targetTimePerBlock: time.Second * 30,
		//},
		//{
		//	targetTimespan:     time.Minute * 2,
		//	targetTimePerBlock: time.Minute,
		//},
		//{
		//	targetTimespan:     time.Minute * 3,
		//	targetTimePerBlock: time.Minute + (time.Second * 30),
		//},
		//{
		//	targetTimespan:     time.Minute * 4,
		//	targetTimePerBlock: time.Minute * 2,
		//},
		//{
		//	targetTimespan:     time.Minute * 10,
		//	targetTimePerBlock: time.Minute * 5,
		//},
	}

	// Explore changes
	for i, pair := range pairs {
		if pair.targetTimespan < pair.targetTimePerBlock {
			log.Fatalf("exploreBlockTargetTime: pair %d\ttargetTimespan %d < targetTimePerBlock %d. This could trigger panic in soterd! (divide by zero)", i, pair.targetTimespan, pair.targetTimePerBlock)
		}

		change, err := runModTargetTime(pair.targetTimespan, pair.targetTimePerBlock, keepLogs)
		if err != nil {
			fmt.Printf("Error during runModTargetTime: %s\n", err)
			// We'll retry the run instead of aborting the whole set of runs
			continue
		}

		results.changes = append(results.changes, change)
		change.Print()
		fmt.Println()

		//dagName, blkGenName, err := change.Save()
		//if err != nil {
		//	log.Fatalf("Error while saving graphs: %v", err)
		//}
		//fmt.Println("dag graph:", dagName)
		//fmt.Println("nodeBlkGen graph:", blkGenName)
	}

	return results
}

// explorePowLimit runs miners with PowLimit & PowLimitBits changed from lowerPctBound to upperPctBound from default values.
func explorePowLimit(keepLogs bool) resultSet {
	results := newResultSet()

	// How many changes we'll move away from the defaults
	deviations := 3
	// How much we'll adjust the change per run
	pctChangeInc := 10
	// The lower and upper bounds of the percent we'll change PowLimit values
	lowerPctBound := -(pctChangeInc * deviations)
	upperPctBound := pctChangeInc * deviations
	pctChange := lowerPctBound

	// Explore changes in PowLimit
	for pctChange <= upperPctBound {
		change, err := runModPowLimit(pctChange, keepLogs)
		if err != nil {
			fmt.Printf("Error during runModPowLimit: %v\n", err)
			// We'll retry the run instead of aborting the whole set of runs
			continue
		}

		results.changes = append(results.changes, change)
		change.Print()
		fmt.Println()

		//dagName, blkGenName, err := change.Save()
		//if err != nil {
		//	log.Fatalf("Error while saving graphs: %v", err)
		//}
		//fmt.Println("dag graph:", dagName)
		//fmt.Println("nodeBlkGen graph:", blkGenName)

		// Update how much we'll change values for next run
		pctChange += pctChangeInc
	}

	return results
}

func main() {
	var keepLogs bool

	flag.BoolVar(&keepLogs, "l", false, "Keep logs from soterd nodes")

	flag.Parse()

	//powLimitResults := explorePowLimit(keepLogs)
	//graphFiles, err := powLimitResults.Save()
	//if err != nil {
	//	log.Fatalf("Failed to save graphs from PowLimit result-set: %v", err)
	//}
	//
	//for _, name := range graphFiles {
	//	fmt.Println("saved:", name)
	//}

	//powLimitHTML, err := powLimitResults.SaveHTML()
	//if err != nil {
	//	log.Fatalf("Failed to render HTML for PowLimit result-set: %v", err)
	//}
	//fmt.Println("saved PowLimit results:", powLimitHTML)

	timeResults := exploreBlockTargetTime(keepLogs)
	timeHtml, err := timeResults.SaveHTML()
	if err != nil {
		log.Fatalf("Failed to render HTML for TargetTime result-set: %s", err)
	}
	fmt.Println("saved TargetTime results:", timeHtml)
}
