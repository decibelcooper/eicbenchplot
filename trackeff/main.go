package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"

	"github.com/decibelcooper/proio/go-proio"
	"github.com/decibelcooper/proio/go-proio/model/eic"
	"go-hep.org/x/hep/hbook"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
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
	pTMin := &eicplot.FloatArrayFlags{Array: []float64{0.5}}
	pTMax := &eicplot.FloatArrayFlags{Array: []float64{100000}}
	fracCut := &eicplot.FloatArrayFlags{Array: []float64{0.01}}
	var (
		etaLimit = flag.Float64("etalimit", 4, "maximum absolute value of eta")
		nBins    = flag.Int("nbins", 80, "number of bins")
		title    = flag.String("title", "", "plot title")
		output   = flag.String("output", "out.png", "output file")
	)
	flag.Var(pTMin, "minpt", "minimum transverse momentum")
	flag.Var(pTMax, "maxpt", "maximum transverse momentum")
	flag.Var(fracCut, "frac", "maximum fractional magnitude of the difference in momentum between track and true")
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		log.Fatal("Invalid arguments")
	}

	p, _ := plot.New()
	p.Title.Text = *title
	p.X.Label.Text = "eta"
	p.X.Tick.Marker = eicplot.PreciseTicks{NSuggestedTicks: 5}
	p.Y.Tick.Marker = eicplot.PreciseTicks{NSuggestedTicks: 5}

	nSubs := 1
	nSubs = intMax(nSubs, len(pTMin.Array))
	nSubs = intMax(nSubs, len(pTMax.Array))
	nSubs = intMax(nSubs, len(fracCut.Array))

	for i, filename := range flag.Args() {
		for j := 0; j < nSubs; j++ {
			iPTMin := intMin(j, len(pTMin.Array)-1)
			iPTMax := intMin(j, len(pTMax.Array)-1)
			iFracCut := intMin(j, len(fracCut.Array)-1)

			plotters := makeTrackEffPlotters(filename, pTMin.Array[iPTMin], pTMax.Array[iPTMax], fracCut.Array[iFracCut], *etaLimit, *nBins)

			pointColor := color.RGBA{A: 255}
			switch i + j {
			case 1:
				pointColor = color.RGBA{G: 255, A: 255}
			case 2:
				pointColor = color.RGBA{B: 255, A: 255}
			case 3:
				pointColor = color.RGBA{R: 255, B: 127, G: 127, A: 255}
			}

			for _, p := range plotters {
				switch t := p.(type) {
				case *plotter.XErrorBars:
					t.LineStyle.Color = pointColor
				case *plotter.YErrorBars:
					t.LineStyle.Color = pointColor
				}
			}

			p.Add(plotters...)
		}
	}

	p.Save(6*vg.Inch, 4*vg.Inch, *output)
}

func makeTrackEffPlotters(filename string, pTMin, pTMax, fracCut, etaLimit float64, nBins int) []plot.Plotter {
	etaHist := hbook.NewH1D(nBins, -etaLimit, etaLimit)
	trueEtaHist := hbook.NewH1D(nBins, -etaLimit, etaLimit)

	reader, err := proio.Open(filename)
	if err != nil {
		log.Fatal(err)
	}

	eventNum := 0
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
					if ok {
						partCandID[simHit.Particle]++
					}

					_, ok = event.GetEntry(sourceID).(*eic.Particle)
					if ok {
						partCandID[sourceID]++
					}
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

			pMag := math.Sqrt(math.Pow(part.P.X, 2) + math.Pow(part.P.Y, 2) + math.Pow(part.P.Z, 2))
			eta := math.Atanh(part.P.Z / pMag)
			pT := math.Sqrt(math.Pow(part.P.X, 2) + math.Pow(part.P.Y, 2))
			chargeMag := math.Abs(float64(part.Charge))
			poqMag := pMag / chargeMag
			diffMag := math.Sqrt(math.Pow(track.Segment[0].Poq.X-part.P.X/chargeMag, 2) +
				math.Pow(track.Segment[0].Poq.Y-part.P.Y/chargeMag, 2) +
				math.Pow(track.Segment[0].Poq.Z-part.P.Z/chargeMag, 2))
			fracDiff := diffMag / poqMag

			// cuts
			if pT < pTMin || pT > pTMax {
				continue
			}
			if fracDiff > fracCut {
				continue
			}

			etaHist.Fill(eta, 1)
		}

		ids = event.TaggedEntries("GenStable")
		for _, id := range ids {
			part, ok := event.GetEntry(id).(*eic.Particle)
			if !ok {
				continue
			}

			pMag := math.Sqrt(math.Pow(part.P.X, 2) + math.Pow(part.P.Y, 2) + math.Pow(part.P.Z, 2))
			eta := math.Atanh(part.P.Z / pMag)
			pT := math.Sqrt(math.Pow(part.P.X, 2) + math.Pow(part.P.Y, 2))

			// cuts
			if pT < pTMin || pT > pTMax {
				continue
			}

			trueEtaHist.Fill(eta, 1)
		}

		eventNum++
	}

	reader.Close()

	points := make(plotter.XYs, nBins)
	xErrors := make(plotter.XErrors, nBins)
	yErrors := make(plotter.YErrors, nBins)
	binHalfWidth := etaLimit / float64(nBins)
	binSigma := binHalfWidth / math.Sqrt(3.)
	for i := range points {
		trueX, trueY := trueEtaHist.XY(i)

		points[i].X = trueX + binHalfWidth
		xErrors[i].Low = binSigma
		xErrors[i].High = binSigma

		_, trackY := etaHist.XY(i)
		if trueY > 0 {
			points[i].Y = trackY / trueY
			yErrors[i].Low = math.Sqrt((1 - trackY/trueY) * trackY / math.Pow(trueY, 2))
			yErrors[i].High = yErrors[i].Low
		}
	}
	errPoints := plotutil.ErrorPoints{points, xErrors, yErrors}
	xerr, _ := plotter.NewXErrorBars(errPoints)
	yerr, _ := plotter.NewYErrorBars(errPoints)

	return []plot.Plotter{xerr, yerr}
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
