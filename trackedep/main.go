package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/decibelcooper/eicplot"
	"github.com/proio-org/go-proio"
	"github.com/proio-org/go-proio-pb/model/eic"
	"go-hep.org/x/hep/hbook"
	"go-hep.org/x/hep/hplot"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
)

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: `+os.Args[0]+` [options] <proio-input-file>

options:
`,
	)
	flag.PrintDefaults()
}

var (
	output = flag.String("output", "out.png", "output file")
)

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() != 1 {
		printUsage()
		log.Fatal("Invalid arguments")
	}

	reader, err := proio.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	p, _ := plot.New()
	p.X.Label.Text = "log_10{E dep. (MeV)}"
	p.X.Tick.Marker = eicplot.PreciseTicks{5}
	p.Y.Tick.Marker = eicplot.LogTicks{}
	p.Y.Scale = eicplot.LogScale{}

	hist := hbook.NewH1D(100, -9, 2)

	for event := range reader.ScanEvents() {
		trackerIDs := event.TaggedEntries("Tracker")
		for _, id := range trackerIDs {
			eDep, ok := event.GetEntry(id).(*eic.EnergyDep)
			if !ok {
				continue
			}

			hist.Fill(math.Log10(float64(eDep.GetMean()*1000)), 1)
		}
	}

	hPlot := hplot.NewH1D(hist)
	p.Add(hPlot)

	p.Save(6*vg.Inch, 4*vg.Inch, *output)
}
