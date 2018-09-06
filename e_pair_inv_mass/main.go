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
	fmt.Fprintf(os.Stderr, `Usage: `+os.Args[0]+` [options] <proio-input-files>...

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
	p.X.Label.Text = "Mass (GeV)"
	p.X.Tick.Marker = eicplot.PreciseTicks{NSuggestedTicks: 5}
	p.Y.Tick.Marker = eicplot.PreciseTicks{NSuggestedTicks: 5}

	for i, filename := range flag.Args() {
		hist := makeInvMassHist(filename)

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
		h.LineStyle.Color = lineColor
		if len(flag.Args()) == 1 {
			h.Infos.Style = hplot.HInfoSummary
		}

		p.Add(h)
	}

	p.Save(6*vg.Inch, 4*vg.Inch, *output)
}

func makeInvMassHist(filename string) *hbook.H1D {
	invMassHist := hbook.NewH1D(50, 2.9, 3.3)

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

		for i := 0; i < len(tracks); i++ {
			for j := i + 1; j < len(tracks); j++ {
				if (*tracks[i].Segment[0].Chargesign)*(*tracks[j].Segment[0].Chargesign) > 0 {
					continue
				}

				poqi := tracks[i].Segment[0].Poq
				poqj := tracks[j].Segment[0].Poq
				p := []float64{*poqi.X + *poqj.X, *poqi.Y + *poqj.Y, *poqi.Z + *poqj.Z}

				p2 := math.Pow(p[0], 2) + math.Pow(p[1], 2) + math.Pow(p[2], 2)
				Ei := math.Sqrt(math.Pow(*poqi.X, 2) + math.Pow(*poqi.Y, 2) + math.Pow(*poqi.Z, 2))
				Ej := math.Sqrt(math.Pow(*poqj.X, 2) + math.Pow(*poqj.Y, 2) + math.Pow(*poqj.Z, 2))
				E2 := math.Pow(Ei+Ej, 2)

				// eta := math.Atanh(p[2] / math.Sqrt(p2))
				invMass := math.Sqrt(E2 - p2)

				if invMass > 2.9 && invMass < 3.3 {
					invMassHist.Fill(invMass, 1)
				}
			}
		}
	}

	reader.Close()
	return invMassHist
}
