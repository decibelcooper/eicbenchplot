package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"

	"github.com/proio-org/go-proio"
	"github.com/proio-org/go-proio-pb/model/eic"
	"go-hep.org/x/hep/hbook"
	"go-hep.org/x/hep/hplot"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"

	"github.com/decibelcooper/eicplot"
)

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: `+os.Args[0]+` [options] <proio-input-file>

options:
`,
	)
	flag.PrintDefaults()
}

func main() {
	var (
		title  = flag.String("title", "", "plot title")
		output = flag.String("output", "out.png", "output file")
	)
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		log.Fatal("Invalid arguments")
	}

	p, _ := plot.New()
	p.Title.Text = *title
	p.X.Label.Text = "Transverse Momentum Transfer (GeV)"
	p.X.Tick.Marker = eicplot.PreciseTicks{NSuggestedTicks: 5}
	p.Y.Tick.Marker = eicplot.LogTicks{}
	p.Y.Scale = eicplot.LogScale{}

	var hists []*hbook.H1D
	for _, filename := range flag.Args() {
		hists = append(hists, makeHists(filename)...)
	}

	for i, hist := range hists {
		lineColor := color.RGBA{A: 255}
		switch i {
		case 1:
			lineColor = color.RGBA{G: 255, A: 255}
		case 2:
			lineColor = color.RGBA{B: 255, A: 255}
		case 3:
			lineColor = color.RGBA{R: 255, B: 127, G: 127, A: 255}
		}

		h := hplot.NewH1D(hist)
		h.FillColor = nil
		h.LineStyle.Color = lineColor
		h.Infos.Style = hplot.HInfoNone

		p.Add(h)
	}

	p.Save(6*vg.Inch, 4*vg.Inch, *output)
}

const jpsiMass = 3.096916

func makeHists(filename string) []*hbook.H1D {
	deltaPTHist := hbook.NewH1D(50, 0, 4)
	deltaPTTruthHist := hbook.NewH1D(50, 0, 4)

	reader, err := proio.Open(filename)
	if err != nil {
		log.Fatal(err)
	}

	for event := range reader.ScanEvents() {
		ids := event.TaggedEntries("Reconstructed")
		tracks := []*eic.Track{}
		for _, id := range ids {
			track, ok := event.GetEntry(id).(*eic.Track)
			if ok && len(track.Segment) > 0 {
				tracks = append(tracks, track)
			}
		}

		if len(tracks) == 3 {
			var p [2]float64
			for _, track := range tracks {
				p[0] += *track.Segment[0].Poq.X
				p[1] += *track.Segment[0].Poq.Y
			}
			deltaPT := math.Sqrt(math.Pow(p[0], 2) + math.Pow(p[1], 2))
			deltaPTHist.Fill(deltaPT, 1)
		}

		ids = event.TaggedEntries("GenStable")
		protons := []*eic.Particle{}
		for _, id := range ids {
			part, ok := event.GetEntry(id).(*eic.Particle)
			if ok && *part.Pdg == 2212 {
				protons = append(protons, part)
			}
		}

		if len(protons) == 1 {
			deltaPT := math.Sqrt(math.Pow(float64(*protons[0].P.X), 2) + math.Pow(float64(*protons[0].P.Y), 2))
			deltaPTTruthHist.Fill(deltaPT, 1)
		}
	}

	reader.Close()
	return []*hbook.H1D{deltaPTTruthHist, deltaPTHist}
}
