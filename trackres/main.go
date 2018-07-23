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
	pTMin    = flag.Float64("minpt", 0.5, "minimum transverse momentum")
	pTMax    = flag.Float64("maxpt", 30, "maximum transverse momentum")
	etaLimit = flag.Float64("etalimit", 4, "maximum absolute value of eta")
	resLimit = flag.Float64("reslimit", 0.1, "maximum momentum resolution in the color map")
	nBinsPT  = flag.Int("nbinspt", 10, "number of bins in transverse momentum")
	nBinsEta = flag.Int("nbinseta", 10, "number of bins in eta")
	title    = flag.String("title", "", "plot title")
	output   = flag.String("output", "out.png", "output file")
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

	resGrid := NewResGrid(*nBinsEta, -*etaLimit, *etaLimit, *nBinsPT, *pTMin, *pTMax)

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

			pMag := math.Sqrt(math.Pow(float64(part.GetP().GetX()), 2) + math.Pow(float64(part.GetP().GetY()), 2) + math.Pow(float64(part.GetP().GetZ()), 2))
			eta := math.Atanh(float64(part.GetP().GetZ()) / pMag)
			pT := math.Sqrt(math.Pow(float64(part.GetP().GetX()), 2) + math.Pow(float64(part.GetP().GetY()), 2))
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

	colorMap := moreland.ExtendedBlackBody()
	colorMap.SetMin(0)
	colorMap.SetMax(*resLimit)
	pal := colorMap.Palette(1000)
	heatMap := plotter.NewHeatMap(resGrid, pal)
	heatMap.Min = 0
	heatMap.Max = *resLimit
	p.Add(heatMap)

	p.Draw(dc0)

	p, _ = plot.New()

	colorBar := &plotter.ColorBar{ColorMap: colorMap}
	colorBar.Vertical = true
	p.Add(colorBar)
	p.HideX()
	p.Y.Padding = 0

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

type ResGrid struct {
	hCount, hV, hV2 *hbook.H2D
	nBinsX, nBinsY  int
	xLow, xHigh     float64
	yLow, yHigh     float64
}

func NewResGrid(nBinsX int, xLow, xHigh float64, nBinsY int, yLow, yHigh float64) *ResGrid {
	return &ResGrid{
		hbook.NewH2D(nBinsX, xLow, xHigh, nBinsY, yLow, yHigh),
		hbook.NewH2D(nBinsX, xLow, xHigh, nBinsY, yLow, yHigh),
		hbook.NewH2D(nBinsX, xLow, xHigh, nBinsY, yLow, yHigh),
		nBinsX, nBinsY,
		xLow, xHigh,
		yLow, yHigh,
	}
}

func (g *ResGrid) Fill(x, y, z float64) {
	g.hCount.Fill(x, y, 1)
	g.hV.Fill(x, y, z)
	g.hV2.Fill(x, y, z*z)
}

func (g *ResGrid) Dims() (int, int) {
	return g.nBinsX, g.nBinsY
}

func (g *ResGrid) Z(i, j int) float64 {
	n := g.hCount.GridXYZ().Z(i, j)
	if n < 3 {
		return 1
	}
	mean := g.hV.GridXYZ().Z(i, j) / n
	mean2 := g.hV2.GridXYZ().Z(i, j) / n

	stddev := math.Sqrt(mean2 - mean*mean)
	return stddev
}

func (g *ResGrid) X(i int) float64 {
	return g.hCount.GridXYZ().X(i)
}

func (g *ResGrid) Y(j int) float64 {
	return g.hCount.GridXYZ().Y(j)
}
