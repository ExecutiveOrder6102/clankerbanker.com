package converter

import (
	"bytes"
	"encoding/csv"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestToKoinlyRecordTransferAndDepositTypes(t *testing.T) {
	for _, tc := range []struct {
		typeName string
		amount   int64
		label    string
		sent     bool
	}{
		{"swap_in", 100000000, "transfer", false},
		{"legacy_swap_in", 100000000, "transfer", false},
		{"swap_out", -100000000, "transfer", true},
		{"channel_open", 100000000, "deposit", false},
		{"legacy_pay_to_open", 100000000, "deposit", false},
	} {
		t.Run(tc.typeName, func(t *testing.T) {
			p := &PhoenixRecord{
				Timestamp: time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC), Type: tc.typeName,
				AmountMillisats: tc.amount, TransactionID: "tx-alias", Description: "alias mapping",
			}
			k, diff := ToKoinlyRecord(p)
			if math.Abs(diff) > 1e-9 {
				t.Fatalf("expected zero rounding diff, got %f", diff)
			}
			if k.Label != tc.label || k.Date != "2024-05-01 12:00:00 Z" || k.TxHash != p.TransactionID || k.Description != p.Description {
				t.Errorf("unexpected metadata: %+v", k)
			}
			if tc.sent {
				if k.SentAmount != "0.00100000" || k.SentCurrency != "BTC" || k.ReceivedAmount != "" || k.ReceivedCurrency != "" {
					t.Errorf("unexpected outgoing transfer: %+v", k)
				}
			} else if k.ReceivedAmount != "0.00100000" || k.ReceivedCurrency != "BTC" || k.SentAmount != "" || k.SentCurrency != "" {
				t.Errorf("unexpected incoming transfer/deposit: %+v", k)
			}
			if k.FeeAmount != "" || k.FeeCurrency != "" {
				t.Errorf("unexpected separate fee: %+v", k)
			}
		})
	}
}

func TestConvertSampleCSV(t *testing.T) {
	f, err := os.Open(filepath.Join("..", "testdata", "sample_phoenix.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var output bytes.Buffer
	if err := Convert(f, &output, false); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(&output).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"Date", "Sent Amount", "Sent Currency", "Received Amount", "Received Currency", "Fee Amount", "Fee Currency", "Net Worth Amount", "Net Worth Currency", "Label", "Description", "TxHash"},
		{"2025-01-01 00:00:00 Z", "", "", "0.00050000", "BTC", "", "", "", "", "lightning", "First receive", "tx1"},
		{"2025-01-02 00:00:00 Z", "0.00020000", "BTC", "", "", "", "", "", "", "lightning", "Sent payment", "tx2"},
		{"2025-01-03 00:00:00 Z", "", "", "0.00100000", "BTC", "", "", "", "", "transfer", "Swap in", "tx3"},
		{"2025-01-04 00:00:00 Z", "0.00050000", "BTC", "", "", "", "", "", "", "transfer", "Swap out", "tx4"},
		{"2025-01-05 00:00:00 Z", "", "", "0.00080000", "BTC", "", "", "", "", "deposit", "Open channel", "tx5"},
		{"2025-01-06 00:00:00 Z", "", "", "", "", "0.00003000", "BTC", "", "", "cost", "Close channel", "tx6"},
	}
	if len(rows) != len(want) {
		t.Fatalf("expected %d CSV rows, got %d", len(want), len(rows))
	}
	for i, row := range rows {
		if strings.Join(row, "\x00") != strings.Join(want[i], "\x00") {
			t.Errorf("row %d = %q, want %q", i, row, want[i])
		}
	}
}

func TestConvertRoundingCostOption(t *testing.T) {
	const header = "Timestamp,ref,type,amount_msat,unused1,unused2,MiningFeeSat,unused3,ServiceFeeMsat,unused4,unused5,transaction_id,unused6,description\n"
	for _, tc := range []struct {
		name       string
		amount     string
		addCost    bool
		adjustment bool
	}{
		{"disabled", "1400", false, false},
		{"enabled", "1400", true, true},
		{"negative difference", "1600", true, true},
		{"no difference", "1000", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := header + "2025-01-01T00:00:00.000Z,x,lightning_received," + tc.amount + ",x,x,0,x,0,x,x,tx1,x,First\n" +
				"2025-01-02T00:00:00.000Z,x,lightning_received," + tc.amount + ",x,x,0,x,0,x,x,tx2,x,Second\n"
			var output bytes.Buffer
			before := time.Now().UTC().Truncate(time.Second)
			if err := Convert(strings.NewReader(input), &output, tc.addCost); err != nil {
				t.Fatal(err)
			}
			after := time.Now().UTC()
			rows, err := csv.NewReader(&output).ReadAll()
			if err != nil {
				t.Fatal(err)
			}
			wantRows := 3
			if tc.adjustment {
				wantRows++
			}
			if len(rows) != wantRows {
				t.Fatalf("expected %d rows, got %d: %q", wantRows, len(rows), rows)
			}
			if !tc.adjustment {
				return
			}
			adjustment := rows[3]
			wantFields := []string{"", "", "", "", "0.00000001", "BTC", "", "", "cost", "Adjustment for rounding differences", ""}
			if strings.Join(adjustment[1:], "\x00") != strings.Join(wantFields, "\x00") {
				t.Errorf("unexpected adjustment fields: %q", adjustment)
			}
			date, err := time.Parse(KoinlyDateFormat, adjustment[0])
			if err != nil || date.Before(before) || date.After(after) {
				t.Errorf("adjustment date %q is not the current UTC time: %v", adjustment[0], err)
			}
		})
	}
}

func TestParsePhoenixRecord(t *testing.T) {
	record := []string{
		"2024-05-01T12:00:00.000Z", // timestamp
		"unused1",
		"lightning_received", // type
		"123456789",          // amount_msat
		"unused2",
		"unused3",
		"0", // mining fee sat
		"unused4",
		"0", // service fee msat
		"unused5",
		"unused6",
		"txid123", // transaction id
		"unused7",
		"test description", // description
	}
	p, err := ParsePhoenixRecord(record)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.Timestamp.Equal(time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("timestamp parsed incorrectly: %v", p.Timestamp)
	}
	if p.Type != "lightning_received" || p.AmountMillisats != 123456789 || p.MiningFeeSat != 0 || p.ServiceFeeMsat != 0 || p.TransactionID != "txid123" || p.Description != "test description" {
		t.Errorf("parsed struct mismatch: %+v", p)
	}
}

func TestToKoinlyRecordLightningReceived(t *testing.T) {
	p := &PhoenixRecord{
		Timestamp:       time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC),
		Type:            "lightning_received",
		AmountMillisats: 1000000000, // 1,000,000 sats
		TransactionID:   "tx1",
		Description:     "desc",
	}
	k, diff := ToKoinlyRecord(p)
	if math.Abs(diff) > 1e-9 {
		t.Errorf("expected zero rounding diff, got %f", diff)
	}
	if k.ReceivedAmount != "0.01000000" || k.ReceivedCurrency != "BTC" || k.Label != "lightning" {
		t.Errorf("unexpected koinly record: %+v", k)
	}
}

func TestToKoinlyRecordLightningSent(t *testing.T) {
	p := &PhoenixRecord{
		Timestamp:       time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC),
		Type:            "lightning_sent",
		AmountMillisats: -200000000, // -200,000 sats
		TransactionID:   "tx2",
		Description:     "desc",
	}
	k, diff := ToKoinlyRecord(p)
	if math.Abs(diff) > 1e-9 {
		t.Errorf("expected zero rounding diff, got %f", diff)
	}
	if k.SentAmount != "0.00200000" || k.SentCurrency != "BTC" || k.Label != "lightning" {
		t.Errorf("unexpected koinly record: %+v", k)
	}
}

func TestToKoinlyRecordChannelClose(t *testing.T) {
	p := &PhoenixRecord{
		Timestamp:       time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC),
		Type:            "channel_close",
		AmountMillisats: -150000, // -150 sats
		TransactionID:   "tx3",
		Description:     "desc",
	}
	k, diff := ToKoinlyRecord(p)
	if math.Abs(diff) > 1e-9 {
		t.Errorf("expected zero rounding diff, got %f", diff)
	}
	if k.FeeAmount != "0.00000150" || k.FeeCurrency != "BTC" || k.Label != "cost" {
		t.Errorf("unexpected koinly record: %+v", k)
	}
}

func TestFinalBalanceSampleCSV(t *testing.T) {
	// Need to fix path to testdata since we are in converter package
	f, err := os.Open(filepath.Join("..", "testdata", "sample_phoenix.csv"))
	if err != nil {
		t.Fatalf("failed to read csv: %v", err)
	}
	defer f.Close()

	records, err := ReadPhoenixCSV(f)
	if err != nil {
		t.Fatalf("failed to read csv records: %v", err)
	}
	var total float64
	for _, p := range records {
		k, diff := ToKoinlyRecord(p)
		if math.Abs(diff) > 1e-9 {
			t.Errorf("unexpected rounding diff %f", diff)
		}
		if k.ReceivedAmount != "" {
			v, err := strconv.ParseFloat(k.ReceivedAmount, 64)
			if err != nil {
				t.Fatalf("bad received amount: %v", err)
			}
			total += v
		}
		if k.SentAmount != "" {
			v, err := strconv.ParseFloat(k.SentAmount, 64)
			if err != nil {
				t.Fatalf("bad sent amount: %v", err)
			}
			total -= v
		}
		if k.FeeAmount != "" {
			v, err := strconv.ParseFloat(k.FeeAmount, 64)
			if err != nil {
				t.Fatalf("bad fee amount: %v", err)
			}
			total -= v
		}
	}
	expected := 0.00157
	if math.Abs(total-expected) > 1e-8 {
		t.Errorf("expected final balance %.8f BTC, got %.8f BTC", expected, total)
	}
}
