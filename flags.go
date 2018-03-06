package eicplot

import (
	"fmt"
	"strconv"
)

type FloatArrayFlags struct {
	Array   []float64
	beenSet bool
}

func (f *FloatArrayFlags) Set(valueStr string) error {
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return err
	}

	if !f.beenSet {
		f.beenSet = true
		f.Array = nil
	}

	f.Array = append(f.Array, value)
	return nil
}

func (f *FloatArrayFlags) String() string {
	return fmt.Sprint(f.Array)
}
