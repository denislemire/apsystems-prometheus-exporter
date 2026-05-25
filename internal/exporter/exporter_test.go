package exporter

import (
	"encoding/json"
	"testing"
)

func TestParseBatchItem(t *testing.T) {
	uid, ch, kwh, ok := parseBatchItem("804000060846-2-2.446396")
	if !ok {
		t.Fatal("expected ok")
	}
	if uid != "804000060846" || ch != "2" || kwh < 2.44 || kwh > 2.45 {
		t.Fatalf("got uid=%s ch=%s kwh=%v", uid, ch, kwh)
	}
}

func TestSummaryDataDecode(t *testing.T) {
	raw := `{"today":"58.29","month":"1124.40","year":"3293.03","lifetime":"44254.31"}`
	var summary summaryData
	if err := json.Unmarshal([]byte(raw), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Year != "3293.03" || summary.Lifetime != "44254.31" {
		t.Fatalf("got year=%q lifetime=%q", summary.Year, summary.Lifetime)
	}
}
