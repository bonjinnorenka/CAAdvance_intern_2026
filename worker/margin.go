package main

import (
	"fmt"
	"math"
)

func customerChargeJPY(cost int64, marginRate int) (int64, error) {
	if marginRate < 0 || marginRate >= 100 {
		return 0, fmt.Errorf("invalid margin_rate: %d", marginRate)
	}
	if marginRate == 0 {
		return cost, nil
	}
	charge := math.Round(float64(cost) / (1 - float64(marginRate)/100))
	return int64(charge), nil
}
