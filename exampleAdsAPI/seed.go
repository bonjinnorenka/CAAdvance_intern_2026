package main

import (
	"hash/fnv"
	"math"
	"math/rand"
)

type accountBase struct {
	impressions int
	ctr         float64
	cpc         float64
	cvr         float64
}

type metrics struct {
	Impressions int
	Clicks      int
	Cost        int
	Conversions int
}

func hash64(s string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return int64(h.Sum64())
}

func accountBases(accountID string) accountBase {
	v := uint64(hash64(accountID))
	return accountBase{
		impressions: 8000 + int(v%25000),
		ctr:         0.015 + float64(v%100)/10000.0,
		cpc:         80 + float64(v%120),
		cvr:         0.020 + float64((v/100)%80)/10000.0,
	}
}

func dailyRNG(accountID, date string) *rand.Rand {
	return rand.New(rand.NewSource(hash64(accountID + "|" + date)))
}

func clamp(v, min, max float64) float64 {
	return math.Min(max, math.Max(min, v))
}

func generateMetrics(accountID, date string) metrics {
	base := accountBases(accountID)
	rng := dailyRNG(accountID, date)

	impressions := int(math.Round(float64(base.impressions) * (0.80 + rng.Float64()*0.40)))
	if impressions < 0 {
		impressions = 0
	}

	ctr := clamp(base.ctr*(0.85+rng.Float64()*0.30), 0.005, 0.08)
	clicks := int(math.Round(float64(impressions) * ctr))
	if clicks < 0 {
		clicks = 0
	}
	if clicks > impressions {
		clicks = impressions
	}

	cpc := base.cpc * (0.85 + rng.Float64()*0.30)
	cost := int(math.Round(float64(clicks) * cpc))
	if cost < 0 {
		cost = 0
	}

	cvr := clamp(base.cvr*(0.70+rng.Float64()*0.60), 0.005, 0.15)
	conversions := int(math.Round(float64(clicks) * cvr))
	if conversions < 0 {
		conversions = 0
	}
	if conversions > clicks {
		conversions = clicks
	}

	return metrics{
		Impressions: impressions,
		Clicks:      clicks,
		Cost:        cost,
		Conversions: conversions,
	}
}
