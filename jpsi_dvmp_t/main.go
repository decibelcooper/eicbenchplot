package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"

	"github.com/pkg/profile"
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
	defer profile.Start().Stop()

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
	p.X.Label.Text = "-t (GeV^2)"
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
	tHist := hbook.NewH1D(50, -1, 4)
	tPTruthHist := hbook.NewH1D(50, -1, 4)
	tETruthHist := hbook.NewH1D(50, -1, 4)

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
			var p [4]float64
			p[2] += 5. // 5 GeV e- beam
			p[3] -= 5.
			for _, track := range tracks {
				p[0] += *track.Segment[0].Poq.X
				p[1] += *track.Segment[0].Poq.Y
				p[2] += *track.Segment[0].Poq.Z
				p[3] += math.Sqrt(pSquareD(track.Segment[0].Poq))
			}
			t := math.Pow(p[0], 2) + math.Pow(p[1], 2) + math.Pow(p[2], 2)
			t -= math.Pow(p[3], 2)
			tHist.Fill(t, 1)
		}

		ids = event.TaggedEntries("GenStable")
		protons := []*eic.Particle{}
		leptons := []*eic.Particle{}
		for _, id := range ids {
			part, ok := event.GetEntry(id).(*eic.Particle)
			if ok {
				if *part.Pdg == 2212 {
					protons = append(protons, part)
				} else if *part.Pdg == 11 || *part.Pdg == -11 {
					leptons = append(leptons, part)
				}
			}
		}

		if len(protons) == 1 {
			t := math.Pow(float64(*protons[0].P.X), 2) + math.Pow(float64(*protons[0].P.Y), 2) + math.Pow(float64(*protons[0].P.Z-100.), 2) //  100 GeV p+ beam
			t -= math.Pow(math.Sqrt(pSquareF(protons[0].P)+math.Pow(float64(*protons[0].Mass), 2))-100., 2)
			tPTruthHist.Fill(t, 1)
		}

		if len(leptons) == 3 {
			var p [4]float64
			p[2] += 5. // 5 GeV e- beam
			p[3] -= 5.
			for _, lepton := range leptons {
				p[0] += float64(*lepton.P.X)
				p[1] += float64(*lepton.P.Y)
				p[2] += float64(*lepton.P.Z)
				p[3] += math.Sqrt(pSquareF(lepton.P))
			}
			t := math.Pow(p[0], 2) + math.Pow(p[1], 2) + math.Pow(p[2], 2)
			t -= math.Pow(p[3], 2)
			tETruthHist.Fill(t, 1)
		}
	}

	reader.Close()
	return []*hbook.H1D{tPTruthHist, tETruthHist, tHist}
}

func pSquareD(p *eic.XYZD) float64 {
	px2 := math.Pow(float64(*p.X), 2)
	py2 := math.Pow(float64(*p.Y), 2)
	pz2 := math.Pow(float64(*p.Z), 2)
	return px2 + py2 + pz2
}

func pSquareF(p *eic.XYZF) float64 {
	px2 := math.Pow(float64(*p.X), 2)
	py2 := math.Pow(float64(*p.Y), 2)
	pz2 := math.Pow(float64(*p.Z), 2)
	return px2 + py2 + pz2
}
