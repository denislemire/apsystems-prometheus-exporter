package exporter

import "testing"

func TestParseBatchItem(t *testing.T) {
	uid, ch, kwh, ok := parseBatchItem("804000060846-2-2.446396")
	if !ok {
		t.Fatal("expected ok")
	}
	if uid != "804000060846" || ch != "2" || kwh < 2.44 || kwh > 2.45 {
		t.Fatalf("got uid=%s ch=%s kwh=%v", uid, ch, kwh)
	}
}
