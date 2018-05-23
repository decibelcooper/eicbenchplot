package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/decibelcooper/proio/go-proio"
	"github.com/decibelcooper/proio/go-proio/model/eic"
	"go-hep.org/x/hep/hbook"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/palette/moreland"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"

	"github.com/decibelcooper/eicplot"
)

var (
	pTMin     = flag.Float64("minpt", 0.5, "minimum transverse momentum")
	pTMax     = flag.Float64("maxpt", 30, "maximum transverse momentum")
	etaLimit  = flag.Float64("etalimit", 4, "maximum absolute value of eta")
	pullLimit = flag.Float64("pulllimit", 0.1, "maximum momentum pull in the color map")
	nBinsPT   = flag.Int("nbinspt", 10, "number of bins in transverse momentum")
	nBinsEta  = flag.Int("nbinseta", 10, "number of bins in eta")
	title     = flag.String("title", "", "plot title")
	output    = flag.String("output", "out.png", "output file")
)

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: `+os.Args[0]+` [options] <proio-input-file>

options:
`,
	)
	flag.PrintDefaults()
}

func main() {
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() != 1 {
		printUsage()
		log.Fatal("Invalid arguments")
	}

	p, _ := plot.New()
	p.Title.Text = *title
	p.X.Label.Text = "eta"
	p.Y.Label.Text = "p_T"
	p.X.Tick.Marker = eicplot.PreciseTicks{NSuggestedTicks: 5}
	p.Y.Tick.Marker = eicplot.PreciseTicks{NSuggestedTicks: 5}

	resGrid := NewPullGrid(*nBinsEta, -*etaLimit, *etaLimit, *nBinsPT, *pTMin, *pTMax)

	filename := flag.Arg(0)
	reader, err := proio.Open(filename)
	if err != nil {
		log.Fatal(err)
	}

	for event := range reader.ScanEvents() {
		ids := event.TaggedEntries("Reconstructed")
		for _, id := range ids {
			track, ok := event.GetEntry(id).(*eic.Track)
			if !ok {
				continue
			}

			partCandID := make(map[uint64]uint64)
			for _, obsID := range track.Observation {
				eDep, ok := event.GetEntry(obsID).(*eic.EnergyDep)
				if !ok {
					continue
				}

				for _, sourceID := range eDep.Source {
					simHit, ok := event.GetEntry(sourceID).(*eic.SimHit)
					if !ok {
						continue
					}

					partCandID[simHit.GetParticle()]++
				}
			}

			partID := uint64(0)
			hitCount := uint64(0)
			for id, count := range partCandID {
				if count > hitCount {
					partID = id
					hitCount = count
				}
			}

			part, ok := event.GetEntry(partID).(*eic.Particle)
			if !ok {
				continue
			}

			if len(track.Segment) == 0 {
				continue
			}

			pMag := math.Sqrt(math.Pow(part.GetP().GetX(), 2) + math.Pow(part.GetP().GetY(), 2) + math.Pow(part.GetP().GetZ(), 2))
			eta := math.Atanh(part.GetP().GetZ() / pMag)
			pT := math.Sqrt(math.Pow(part.GetP().GetX(), 2) + math.Pow(part.GetP().GetY(), 2))
			chargeMag := math.Abs(float64(part.GetCharge()))
			poqMag := pMag / chargeMag
			trackPoqMag := math.Sqrt(math.Pow(track.Segment[0].GetPoq().GetX(), 2) +
				math.Pow(track.Segment[0].GetPoq().GetY(), 2) +
				math.Pow(track.Segment[0].GetPoq().GetZ(), 2))

			resGrid.Fill(eta, pT, trackPoqMag/poqMag)
		}
	}

	reader.Close()

	img := vgimg.New(670, 400)
	dc := draw.New(img)
	dc0 := draw.Crop(dc, 0, -70, 0, 0)
	dc1 := draw.Crop(dc, 620, 0, 0, 0)

	colorMap := moreland.SmoothBlueRed()
	colorMap.SetMin(1.0 - *pullLimit)
	colorMap.SetMax(1.0 + *pullLimit)
	heatMap := plotter.NewHeatMap(resGrid, colorMap.Palette(1000))
	heatMap.Min = 1.0 - *pullLimit
	heatMap.Max = 1.0 + *pullLimit
	p.Add(heatMap)

	p.Draw(dc0)

	p, _ = plot.New()

	colorBar := &plotter.ColorBar{ColorMap: colorMap}
	colorBar.Vertical = true
	p.Add(colorBar)
	p.HideX()
	p.Y.Padding = 0
	p.Y.Tick.Marker = eicplot.PreciseTicks{NSuggestedTicks: 3}

	p.Draw(dc1)

	w, err := os.Create(*output)
	if err != nil {
		log.Panic(err)
	}
	png := vgimg.PngCanvas{Canvas: img}
	if _, err = png.WriteTo(w); err != nil {
		log.Panic(err)
	}
}

type PullGrid struct {
	hCount, hV, hV2 *hbook.H2D
	nBinsX, nBinsY  int
	xLow, xHigh     float64
	yLow, yHigh     float64
}

func NewPullGrid(nBinsX int, xLow, xHigh float64, nBinsY int, yLow, yHigh float64) *PullGrid {
	return &PullGrid{
		hbook.NewH2D(nBinsX, xLow, xHigh, nBinsY, yLow, yHigh),
		hbook.NewH2D(nBinsX, xLow, xHigh, nBinsY, yLow, yHigh),
		hbook.NewH2D(nBinsX, xLow, xHigh, nBinsY, yLow, yHigh),
		nBinsX, nBinsY,
		xLow, xHigh,
		yLow, yHigh,
	}
}

func (g *PullGrid) Fill(x, y, z float64) {
	g.hCount.Fill(x, y, 1)
	g.hV.Fill(x, y, z)
	g.hV2.Fill(x, y, z*z)
}

func (g *PullGrid) Dims() (int, int) {
	return g.nBinsX, g.nBinsY
}

func (g *PullGrid) Z(i, j int) float64 {
	n := g.hCount.GridXYZ().Z(i, j)
	if n < 3 {
		return 0
	}
	mean := g.hV.GridXYZ().Z(i, j) / n

	return mean
}

func (g *PullGrid) X(i int) float64 {
	return g.hCount.GridXYZ().X(i)
}

func (g *PullGrid) Y(j int) float64 {
	return g.hCount.GridXYZ().Y(j)
}
