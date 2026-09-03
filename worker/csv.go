package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

var csvHeader = []string{
	"アカウントID",
	"アカウント名",
	"日付",
	"表示回数",
	"クリック数",
	"コンバージョン数",
	"媒体費用",
	"マージン料率(%)",
	"顧客請求額",
}

type reportCSVRow struct {
	AccountID      string
	AccountName    string
	Date           time.Time
	Impressions    int
	Clicks         int
	Conversions    int
	Cost           int64
	MarginRate     int
	CustomerCharge int64
}

func buildReportCSV(rows []reportCSVRow) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := buf.Write(utf8BOM); err != nil {
		return nil, err
	}

	w := csv.NewWriter(&buf)
	if err := w.Write(csvHeader); err != nil {
		return nil, fmt.Errorf("write csv header: %w", err)
	}
	for _, row := range rows {
		record := []string{
			row.AccountID,
			row.AccountName,
			row.Date.Format("2006-01-02"),
			strconv.Itoa(row.Impressions),
			strconv.Itoa(row.Clicks),
			strconv.Itoa(row.Conversions),
			strconv.FormatInt(row.Cost, 10),
			strconv.Itoa(row.MarginRate),
			strconv.FormatInt(row.CustomerCharge, 10),
		}
		if err := w.Write(record); err != nil {
			return nil, fmt.Errorf("write csv row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
