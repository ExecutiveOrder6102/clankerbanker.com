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

func TestConvertXapoConsolidatesStatements(t *testing.T) {
	btcAccount := `Processing Date/Time,Transaction Date/Time,Action Taken,Currency,Amount,BTC Spot/FX,USD Amount,Counterparty,Sub Description
2030-02-04 22:08:12,2030-02-04 22:08:12,Transaction,BTC,0.00012345,51000.00,6.29,,test-btc-credit
2030-02-04 20:05:08,2030-02-04 20:04:58,Move to Savings,BTC,-0.00420000,51200.00,-215.04,,test-internal-allocation
2030-02-04 19:58:13,2030-02-04 19:58:01,Exchange USD to BTC,BTC,0.00800000,52500.00,420.00,,BTC 0.00800000 exchanged
`
	usdAccount := `Processing Date/Time,Transaction Date/Time,Action Taken,Currency,Amount,BTC Spot/FX,USD Amount,Counterparty,Sub Description
2030-02-11 17:12:33,2030-02-11 17:11:54,GBP Transfer to Example Bank #..0000,,,1.25,-156.25,,GBP 125.00 sent (test fee included)
2030-02-10 19:41:01,2030-02-10 19:40:54,Received GBP from SAMPLE TEST BANK,,,1.25,937.50,,GBP 750.00 received (test fee included)
2030-02-10 19:39:00,2030-02-10 19:38:54,Move to Savings,BTC,-0.01250000,52000.00,-650.00,,BTC 0.01250000 moved to savings
2030-02-04 19:58:13,2030-02-04 19:58:01,Exchange USD to BTC,BTC,-0.00800000,52500.00,-420.00,,BTC 0.00800000 exchanged
2030-02-04 19:57:06,2030-02-04 19:57:06,Subscription fee,,,1.00,-8.00,,test-subscription-fee
`
	interest := `Processing Date/Time,Transaction Date/Time,Action Taken,Currency,Amount,BTC Spot/FX,USD Amount,Counterparty,Sub Description
2030-02-11 00:32:57,2030-02-11 00:32:57,Daily BTC interest,BTC,0.00000031,53000.00,0.02,Test Savings,test-interest-credit
`

	var output bytes.Buffer
	err := ConvertXapoStatements([]XapoStatement{
		{Name: "BTC_account_sample.csv", Reader: strings.NewReader(btcAccount)},
		{Name: "USD_account_sample.csv", Reader: strings.NewReader(usdAccount)},
		{Name: "BTC_interest_sample.csv", Reader: strings.NewReader(interest)},
	}, &output)
	if err != nil {
		t.Fatalf("ConvertXapoStatements returned an error: %v", err)
	}

	rows, err := csv.NewReader(&output).ReadAll()
	if err != nil {
		t.Fatalf("reading converted CSV: %v", err)
	}
	// Header, two USD-to-BTC trades (including the USD-account savings
	// conversion), BTC transaction, subscription fee, USD receipt from GBP,
	// USD withdrawal from GBP, and BTC interest. The BTC-account savings row is
	// intentionally omitted as an internal transfer.
	if len(rows) != 8 {
		t.Fatalf("expected 8 output rows, got %d: %#v", len(rows), rows)
	}

	var exchange, savingsConversion, interestRow, receivedGBP, sentGBP, feeRow []string
	for _, row := range rows[1:] {
		switch {
		case row[2] == "USD" && row[4] == "BTC" && row[1] == "420.00000000":
			exchange = row
		case row[2] == "USD" && row[4] == "BTC" && row[1] == "650.00000000":
			savingsConversion = row
		case row[9] == "lending interest":
			interestRow = row
		case row[4] == "USD" && row[9] == "deposit":
			receivedGBP = row
		case row[2] == "USD" && row[9] == "withdrawal":
			sentGBP = row
		case row[9] == "cost":
			feeRow = row
		}
	}
	if exchange == nil || exchange[1] != "420.00000000" || exchange[3] != "0.00800000" {
		t.Errorf("expected consolidated USD-to-BTC exchange, got %#v", exchange)
	}
	if savingsConversion == nil || savingsConversion[3] != "0.01250000" || !strings.Contains(savingsConversion[10], "Move to Savings") {
		t.Errorf("expected USD-account savings conversion, got %#v", savingsConversion)
	}
	for _, row := range rows[1:] {
		if row[1] == "215.04000000" && row[2] == "USD" && row[4] == "BTC" {
			t.Errorf("BTC-account Move to Savings must not become a trade: %#v", row)
		}
	}
	if interestRow == nil || interestRow[3] != "0.00000031" || interestRow[7] != "" || interestRow[8] != "" {
		t.Errorf("expected BTC lending interest without a fiat value, got %#v", interestRow)
	}
	if receivedGBP == nil || receivedGBP[3] != "937.50000000" || receivedGBP[5] != "" {
		t.Errorf("expected GBP receipt settled as USD, got %#v", receivedGBP)
	}
	if sentGBP == nil || sentGBP[1] != "156.25000000" || sentGBP[5] != "" {
		t.Errorf("expected GBP transfer settled as USD, got %#v", sentGBP)
	}
	if feeRow == nil || feeRow[5] != "8.00000000" || feeRow[6] != "USD" {
		t.Errorf("expected USD subscription cost, got %#v", feeRow)
	}
}

func TestConvertXapoConsolidatesBTCToUSDCardSpending(t *testing.T) {
	btcAccount := `Processing Date/Time,Transaction Date/Time,Action Taken,Currency,Amount,BTC Spot/FX,USD Amount,Counterparty,Sub Description
2030-03-05 12:00:00,2030-03-05 12:00:00,Exchange BTC to USD,BTC,-0.00025000,60000.00,-15.00,,BTC 0.00025000 exchanged
`
	usdAccount := `Processing Date/Time,Transaction Date/Time,Action Taken,Currency,Amount,BTC Spot/FX,USD Amount,Counterparty,Sub Description
2030-03-05 12:00:06,2030-03-05 12:00:04,Card ···· 1234 transaction,GBP,-12.00,1.25,-15.00,Example Merchant,GBP 12.00 charged
2030-03-05 12:00:00,2030-03-05 12:00:00,Exchange BTC to USD,BTC,0.00025000,60000.00,15.00,,BTC 0.00025000 exchanged
`

	var output bytes.Buffer
	err := ConvertXapoStatements([]XapoStatement{
		{Name: "BTC_account_sample.csv", Reader: strings.NewReader(btcAccount)},
		{Name: "USD_account_sample.csv", Reader: strings.NewReader(usdAccount)},
	}, &output)
	if err != nil {
		t.Fatalf("ConvertXapoStatements returned an error: %v", err)
	}

	rows, err := csv.NewReader(&output).ReadAll()
	if err != nil {
		t.Fatalf("reading converted CSV: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected header plus BTC-to-USD trade and card payment, got %#v", rows)
	}

	var exchange, cardPayment []string
	for _, row := range rows[1:] {
		switch {
		case row[2] == "BTC" && row[4] == "USD":
			exchange = row
		case row[2] == "USD" && row[9] == "payment":
			cardPayment = row
		}
	}
	if exchange == nil || exchange[1] != "0.00025000" || exchange[3] != "15.00000000" {
		t.Errorf("expected consolidated BTC-to-USD exchange, got %#v", exchange)
	}
	if cardPayment == nil || cardPayment[1] != "15.00000000" ||
		!strings.Contains(cardPayment[10], "Card ···· 1234 transaction") ||
		!strings.Contains(cardPayment[10], "Example Merchant") ||
		!strings.Contains(cardPayment[10], "GBP 12.00 charged") {
		t.Errorf("expected USD card payment with original context, got %#v", cardPayment)
	}
}

func TestToXapoKoinlyRecordOnlyConvertsUSDSavingsStatement(t *testing.T) {
	record := &XapoRecord{
		Timestamp:      time.Date(2030, time.February, 4, 20, 5, 8, 0, time.UTC),
		Action:         "Move to Savings",
		Currency:       "BTC",
		Amount:         -0.0042,
		HasAmount:      true,
		USDAmount:      -215.04,
		HasUSDAmount:   true,
		StatementKind:  XapoBTCAccountStatement,
		SubDescription: "test BTC-account internal transfer",
	}

	if koinlyRecord := ToXapoKoinlyRecord(record); koinlyRecord != nil {
		t.Fatalf("expected BTC-account Move to Savings row to be skipped, got %#v", koinlyRecord)
	}

	record.StatementKind = XapoUSDAccountStatement
	koinlyRecord := ToXapoKoinlyRecord(record)
	if koinlyRecord == nil || koinlyRecord.SentAmount != "215.04000000" || koinlyRecord.ReceivedAmount != "0.00420000" {
		t.Fatalf("expected USD-account Move to Savings trade, got %#v", koinlyRecord)
	}
}

func TestReadXapoCSVUsesHeadersRatherThanColumnOrder(t *testing.T) {
	input := `Amount,Unused,Action Taken,Transaction Date/Time,Sub Description,Currency,USD Amount
0.00000031,ignored,Daily BTC interest,2030-02-11 00:32:57,test-reference,BTC,0.02
`
	records, err := ReadXapoCSVWithSource(strings.NewReader(input), "/exports/USD_account_sample.csv")
	if err != nil {
		t.Fatalf("ReadXapoCSV returned an error: %v", err)
	}
	if len(records) != 1 || !records[0].HasAmount || records[0].Amount != 0.00000031 || records[0].Action != "Daily BTC interest" ||
		records[0].StatementName != "/exports/USD_account_sample.csv" || records[0].StatementKind != XapoUSDAccountStatement {
		t.Fatalf("unexpected parsed Xapo records: %#v", records)
	}
}

func TestToXapoKoinlyRecordUsesUSDAmountForNonBTC(t *testing.T) {
	record := &XapoRecord{
		Timestamp:      time.Date(2030, 2, 4, 20, 15, 56, 0, time.UTC),
		Action:         "Received transfer",
		Currency:       "USDC",
		Amount:         15,
		HasAmount:      true,
		USDAmount:      14.75,
		HasUSDAmount:   true,
		Counterparty:   "Example sender",
		SubDescription: "USDC 15.00 received",
	}

	koinlyRecord := ToXapoKoinlyRecord(record)
	if koinlyRecord == nil || koinlyRecord.ReceivedAmount != "14.75000000" || koinlyRecord.ReceivedCurrency != "USD" || koinlyRecord.Label != "deposit" {
		t.Fatalf("expected non-BTC receipt to use USD Amount, got %#v", koinlyRecord)
	}
	if koinlyRecord.Description != "Xapo: Received transfer — Example sender — USDC 15.00 received" {
		t.Fatalf("expected original Xapo context in description, got %q", koinlyRecord.Description)
	}
}

func TestToXapoKoinlyRecordUsesBTCAmountForBTC(t *testing.T) {
	record := &XapoRecord{
		Timestamp:    time.Date(2030, 2, 4, 20, 15, 56, 0, time.UTC),
		Action:       "Transaction",
		Currency:     "BTC",
		Amount:       0.00012345,
		HasAmount:    true,
		USDAmount:    6.29,
		HasUSDAmount: true,
	}

	koinlyRecord := ToXapoKoinlyRecord(record)
	if koinlyRecord == nil || koinlyRecord.ReceivedAmount != "0.00012345" || koinlyRecord.ReceivedCurrency != "BTC" || koinlyRecord.Label != "deposit" {
		t.Fatalf("expected BTC receipt to use BTC Amount, got %#v", koinlyRecord)
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
