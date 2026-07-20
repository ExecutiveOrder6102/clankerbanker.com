package converter

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// KoinlyDateFormat defines the date format required by Koinly CSV.
	KoinlyDateFormat = "2006-01-02 15:04:05 Z"
	// PhoenixDateFormat defines the date format used in Phoenix CSV exports.
	PhoenixDateFormat = "2006-01-02T15:04:05.999Z"
	satsPerBTC        = 100000000
	msatsPerSat       = 1000
)

var koinlyHeader = []string{
	"Date",
	"Sent Amount",
	"Sent Currency",
	"Received Amount",
	"Received Currency",
	"Fee Amount",
	"Fee Currency",
	"Net Worth Amount",
	"Net Worth Currency",
	"Label",
	"Description",
	"TxHash",
}

var Verbose bool

// KoinlyRecord represents a single row in the Koinly CSV file.
type KoinlyRecord struct {
	Date             string
	SentAmount       string
	SentCurrency     string
	ReceivedAmount   string
	ReceivedCurrency string
	FeeAmount        string
	FeeCurrency      string
	NetWorthAmount   string
	NetWorthCurrency string
	Label            string
	Description      string
	TxHash           string
}

// PhoenixRecord represents a single row in the Phoenix CSV file.
type PhoenixRecord struct {
	Timestamp       time.Time
	Type            string
	AmountMillisats int64
	MiningFeeSat    int64
	ServiceFeeMsat  int64
	TransactionID   string
	Description     string
}

// XapoRecord represents a row from a Xapo account or interest statement.
// Xapo leaves some Amount cells blank, so HasAmount and HasUSDAmount retain
// the distinction between a blank value and a genuine zero value.
type XapoRecord struct {
	Timestamp      time.Time
	Action         string
	Currency       string
	Amount         float64
	HasAmount      bool
	USDAmount      float64
	HasUSDAmount   bool
	BTCSpotFX      float64
	HasBTCSpotFX   bool
	Counterparty   string
	SubDescription string
	StatementName  string
	StatementKind  XapoStatementKind
}

// XapoStatementKind identifies the type of Xapo statement that supplied a
// record. Xapo account CSV rows use the same shape, so this comes from the
// source filename rather than the row itself.
type XapoStatementKind string

const (
	XapoUnknownStatement     XapoStatementKind = "unknown"
	XapoUSDAccountStatement  XapoStatementKind = "usd_account"
	XapoBTCAccountStatement  XapoStatementKind = "btc_account"
	XapoBTCInterestStatement XapoStatementKind = "btc_interest"
)

// XapoStatement pairs a statement reader with its original filename. The
// filename is needed to distinguish otherwise-identical USD and BTC account
// records such as "Move to Savings".
type XapoStatement struct {
	Name   string
	Reader io.Reader
}

// ParseIntField parses an integer field with commas.
func ParseIntField(val, name string) int64 {
	v, err := strconv.ParseInt(strings.ReplaceAll(val, ",", ""), 10, 64)
	if err != nil {
		LogVerbose("Warning: failed to parse %s '%s': %v", name, val, err)
		return 0
	}
	return v
}

// FormatBTC formats sats to a BTC string.
func FormatBTC(sats float64) string {
	return fmt.Sprintf("%.8f", sats/satsPerBTC)
}

// LogVerbose prints messages only if the verbose flag is enabled.
func LogVerbose(format string, v ...interface{}) {
	if Verbose {
		log.Printf(format, v...)
	}
}

// Convert handles the core conversion logic from a reader to a writer.
func Convert(r io.Reader, w io.Writer, addRoundingCost bool) error {
	phoenixRecords, err := ReadPhoenixCSV(r)
	if err != nil {
		return fmt.Errorf("reading phoenix csv: %w", err)
	}

	if err := CreateKoinlyCSV(phoenixRecords, w, addRoundingCost); err != nil {
		return fmt.Errorf("creating koinly csv: %w", err)
	}
	return nil
}

// ConvertXapo converts and consolidates one or more Xapo statements into a
// single Koinly CSV. The statements may be BTC, USD, or BTC-interest exports.
func ConvertXapo(readers []io.Reader, w io.Writer) error {
	statements := make([]XapoStatement, 0, len(readers))
	for _, reader := range readers {
		statements = append(statements, XapoStatement{Reader: reader})
	}
	return ConvertXapoStatements(statements, w)
}

// ConvertXapoStatements converts and consolidates Xapo statements while
// retaining their original filenames for source-aware transaction handling.
func ConvertXapoStatements(statements []XapoStatement, w io.Writer) error {
	var records []*XapoRecord
	for i, statement := range statements {
		if statement.Reader == nil {
			return fmt.Errorf("reading xapo csv %d: nil reader", i+1)
		}
		statementRecords, err := ReadXapoCSVWithSource(statement.Reader, statement.Name)
		if err != nil {
			return fmt.Errorf("reading xapo csv %d: %w", i+1, err)
		}
		records = append(records, statementRecords...)
	}

	if err := CreateXapoKoinlyCSV(records, w); err != nil {
		return fmt.Errorf("creating xapo koinly csv: %w", err)
	}
	return nil
}

// ReadPhoenixCSV reads a CSV file from the given reader and parses it into a slice of PhoenixRecord.
func ReadPhoenixCSV(r io.Reader) ([]*PhoenixRecord, error) {
	reader := csv.NewReader(r)
	// Read header row to skip it.
	_, err := reader.Read()
	if err != nil {
		return nil, err
	}

	var records []*PhoenixRecord
	for {
		record, err := reader.Read()
		if err == io.EOF { // End of file reached.
			break
		}
		if err != nil {
			return nil, err
		}

		phoenixRecord, err := ParsePhoenixRecord(record)
		if err != nil {
			// Log parsing errors but continue processing other records.
			log.Printf("Error parsing record: %v. Skipping this record.", err)
			continue
		}
		records = append(records, phoenixRecord)
	}
	return records, nil
}

// CreateKoinlyCSV takes a slice of PhoenixRecord and writes them to a new CSV file
// formatted for Koinly.
func CreateKoinlyCSV(records []*PhoenixRecord, w io.Writer, addCost bool) error {
	writer := csv.NewWriter(w)
	defer writer.Flush() // Ensure all buffered writes are committed to the underlying writer.

	if err := writer.Write(koinlyHeader); err != nil {
		return err
	}

	var roundingDiff float64
	// Convert each Phoenix record to a Koinly record and write it to the CSV.
	for _, p := range records {
		koinlyRecord, diff := ToKoinlyRecord(p)
		roundingDiff += diff
		if err := writer.Write(koinlyRecord.ToStringSlice()); err != nil {
			return err
		}
	}

	if addCost {
		roundingSats := int64(math.Round(math.Abs(roundingDiff)))
		if roundingSats > 0 {
			costRecord := &KoinlyRecord{
				Date:        time.Now().UTC().Format(KoinlyDateFormat),
				FeeAmount:   FormatBTC(float64(roundingSats)),
				FeeCurrency: "BTC",
				Label:       "cost",
				Description: "Adjustment for rounding differences",
			}
			if err := writer.Write(costRecord.ToStringSlice()); err != nil {
				return err
			}
		}
	}
	return nil
}

// ReadXapoCSV reads a Xapo statement using its column headings rather than
// fixed positions. This permits account and interest exports to be combined
// even if Xapo changes column order or adds unused columns.
func ReadXapoCSV(r io.Reader) ([]*XapoRecord, error) {
	return ReadXapoCSVWithSource(r, "")
}

// ReadXapoCSVWithSource reads a Xapo CSV and records the original statement
// name and statement kind on every parsed row.
func ReadXapoCSVWithSource(r io.Reader, statementName string) ([]*XapoRecord, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}

	columns := make(map[string]int, len(header))
	for i, value := range header {
		columns[normalizeXapoHeader(value)] = i
	}

	dateColumn, ok := findXapoColumn(columns, "transactiondatetime", "processingdatetime")
	if !ok {
		return nil, fmt.Errorf("missing Transaction Date/Time or Processing Date/Time column")
	}
	actionColumn, ok := findXapoColumn(columns, "actiontaken")
	if !ok {
		return nil, fmt.Errorf("missing Action Taken column")
	}

	var records []*XapoRecord
	for line := 2; ; line++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading row %d: %w", line, err)
		}
		record, err := parseXapoRecord(row, columns, dateColumn, actionColumn)
		if err != nil {
			return nil, fmt.Errorf("parsing row %d: %w", line, err)
		}
		record.StatementName = statementName
		record.StatementKind = xapoStatementKind(statementName)
		records = append(records, record)
	}
	return records, nil
}

func parseXapoRecord(row []string, columns map[string]int, dateColumn, actionColumn int) (*XapoRecord, error) {
	timestamp, err := parseXapoTimestamp(xapoValue(row, dateColumn))
	if err != nil {
		return nil, fmt.Errorf("invalid transaction date %q: %w", xapoValue(row, dateColumn), err)
	}

	record := &XapoRecord{
		Timestamp:      timestamp,
		Action:         xapoValue(row, actionColumn),
		Currency:       strings.ToUpper(xapoValue(row, xapoColumn(columns, "currency"))),
		Counterparty:   xapoValue(row, xapoColumn(columns, "counterparty")),
		SubDescription: xapoValue(row, xapoColumn(columns, "subdescription")),
	}
	if record.Action == "" {
		return nil, fmt.Errorf("empty Action Taken")
	}

	var parseErr error
	if record.Amount, record.HasAmount, parseErr = parseOptionalXapoNumber(xapoValue(row, xapoColumn(columns, "amount"))); parseErr != nil {
		return nil, fmt.Errorf("invalid Amount: %w", parseErr)
	}
	if record.USDAmount, record.HasUSDAmount, parseErr = parseOptionalXapoNumber(xapoValue(row, xapoColumn(columns, "usdamount"))); parseErr != nil {
		return nil, fmt.Errorf("invalid USD Amount: %w", parseErr)
	}
	if record.BTCSpotFX, record.HasBTCSpotFX, parseErr = parseOptionalXapoNumber(xapoValue(row, xapoColumn(columns, "btcspotfx"))); parseErr != nil {
		return nil, fmt.Errorf("invalid BTC Spot/FX: %w", parseErr)
	}
	return record, nil
}

// CreateXapoKoinlyCSV writes the consolidated output for Xapo statements.
func CreateXapoKoinlyCSV(records []*XapoRecord, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(koinlyHeader); err != nil {
		return err
	}
	for _, record := range ToXapoKoinlyRecords(records) {
		if err := writer.Write(record.ToStringSlice()); err != nil {
			return err
		}
	}
	return writer.Error()
}

// ToXapoKoinlyRecords maps Xapo statement rows into Koinly rows. A Xapo
// USD-to-BTC exchange appears twice (a BTC receipt and a USD debit), so the
// matching legs are merged into one trade before the remaining rows are mapped.
func ToXapoKoinlyRecords(records []*XapoRecord) []*KoinlyRecord {
	groups := make(map[string][]int)
	for i, record := range records {
		if isXapoUSDToBTCExchange(record) {
			groups[xapoExchangeKey(record)] = append(groups[xapoExchangeKey(record)], i)
		}
	}

	matched := make(map[int]bool)
	var output []*KoinlyRecord
	for _, indexes := range groups {
		btcIndex, usdIndex := -1, -1
		for _, index := range indexes {
			record := records[index]
			if record.HasAmount && record.Currency == "BTC" && record.Amount > 0 {
				btcIndex = index
			}
			if record.HasUSDAmount && record.USDAmount < 0 {
				usdIndex = index
			}
		}
		if btcIndex == -1 || usdIndex == -1 {
			continue
		}
		btc := records[btcIndex]
		usd := records[usdIndex]
		output = append(output, &KoinlyRecord{
			Date:             btc.Timestamp.Format(KoinlyDateFormat),
			SentAmount:       formatXapoAmount(math.Abs(usd.USDAmount), "USD"),
			SentCurrency:     "USD",
			ReceivedAmount:   formatXapoAmount(btc.Amount, "BTC"),
			ReceivedCurrency: "BTC",
			Description:      xapoDescription(btc),
		})
		matched[btcIndex] = true
		matched[usdIndex] = true
	}

	for i, record := range records {
		if matched[i] {
			continue
		}
		if koinlyRecord := ToXapoKoinlyRecord(record); koinlyRecord != nil {
			output = append(output, koinlyRecord)
		}
	}

	sort.SliceStable(output, func(i, j int) bool {
		return output[i].Date < output[j].Date
	})
	return output
}

// ToXapoKoinlyRecord maps a standalone Xapo statement row into Koinly.
func ToXapoKoinlyRecord(record *XapoRecord) *KoinlyRecord {
	action := strings.ToLower(record.Action)
	koinlyRecord := &KoinlyRecord{
		Date:        record.Timestamp.Format(KoinlyDateFormat),
		Description: xapoDescription(record),
	}

	if strings.Contains(action, "daily btc interest") && record.HasAmount {
		koinlyRecord.ReceivedAmount = formatXapoAmount(math.Abs(record.Amount), "BTC")
		koinlyRecord.ReceivedCurrency = "BTC"
		koinlyRecord.Label = "lending interest"
		return koinlyRecord
	}
	if strings.Contains(action, "move to savings") {
		// Only the USD-account statement identifies this as a USD-to-BTC
		// conversion. The same-shaped BTC-account row is an internal movement,
		// and an unnamed source cannot be safely classified.
		if record.StatementKind == XapoUSDAccountStatement &&
			strings.EqualFold(record.Currency, "BTC") && record.HasAmount && record.Amount != 0 &&
			record.HasUSDAmount && record.USDAmount != 0 {
			koinlyRecord.SentAmount = formatXapoAmount(math.Abs(record.USDAmount), "USD")
			koinlyRecord.SentCurrency = "USD"
			koinlyRecord.ReceivedAmount = formatXapoAmount(math.Abs(record.Amount), "BTC")
			koinlyRecord.ReceivedCurrency = "BTC"
			return koinlyRecord
		}
		return nil
	}
	currency, amount, ok := xapoMovementAmount(record)
	if !ok {
		LogVerbose("Skipping Xapo row with no recognised amount: %+v", record)
		return nil
	}

	if strings.Contains(action, "subscription fee") || strings.HasSuffix(action, " fee") {
		koinlyRecord.FeeAmount = formatXapoAmount(math.Abs(amount), currency)
		koinlyRecord.FeeCurrency = currency
		koinlyRecord.Label = "cost"
		return koinlyRecord
	}
	if strings.Contains(action, "transfer to") {
		setXapoMovement(koinlyRecord, amount, currency, "withdrawal")
		return koinlyRecord
	}
	if strings.Contains(action, "received") || action == "transaction" {
		setXapoMovement(koinlyRecord, amount, currency, "deposit")
		return koinlyRecord
	}

	setXapoMovement(koinlyRecord, amount, currency, "")
	return koinlyRecord
}

func setXapoMovement(record *KoinlyRecord, amount float64, currency, label string) {
	record.Label = label
	if amount >= 0 {
		record.ReceivedAmount = formatXapoAmount(amount, currency)
		record.ReceivedCurrency = currency
		return
	}
	record.SentAmount = formatXapoAmount(math.Abs(amount), currency)
	record.SentCurrency = currency
}

func xapoMovementAmount(record *XapoRecord) (string, float64, bool) {
	if strings.EqualFold(record.Currency, "BTC") && record.HasAmount {
		return "BTC", record.Amount, true
	}
	if record.HasUSDAmount {
		return "USD", record.USDAmount, true
	}
	return "", 0, false
}

func isXapoUSDToBTCExchange(record *XapoRecord) bool {
	return strings.EqualFold(strings.TrimSpace(record.Action), "Exchange USD to BTC")
}

func xapoExchangeKey(record *XapoRecord) string {
	return record.Timestamp.UTC().Format(time.RFC3339Nano) + "|" + strings.ToLower(strings.TrimSpace(record.SubDescription))
}

func xapoDescription(record *XapoRecord) string {
	parts := []string{"Xapo: " + record.Action}
	if record.Counterparty != "" {
		parts = append(parts, record.Counterparty)
	}
	if record.SubDescription != "" {
		parts = append(parts, record.SubDescription)
	}
	return strings.Join(parts, " — ")
}

func xapoStatementKind(statementName string) XapoStatementKind {
	name := normalizeXapoHeader(filepath.Base(statementName))
	switch {
	case strings.Contains(name, "usdaccount"):
		return XapoUSDAccountStatement
	case strings.Contains(name, "btcaccount"):
		return XapoBTCAccountStatement
	case strings.Contains(name, "btcinterest"):
		return XapoBTCInterestStatement
	default:
		return XapoUnknownStatement
	}
}

func formatXapoAmount(amount float64, currency string) string {
	if currency == "BTC" {
		return FormatBTC(amount * satsPerBTC)
	}
	return strconv.FormatFloat(amount, 'f', 8, 64)
}

func normalizeXapoHeader(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, value)
}

func findXapoColumn(columns map[string]int, names ...string) (int, bool) {
	for _, name := range names {
		if index, ok := columns[name]; ok {
			return index, true
		}
	}
	return 0, false
}

func xapoColumn(columns map[string]int, name string) int {
	if index, ok := columns[name]; ok {
		return index
	}
	return -1
}

func xapoValue(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func parseOptionalXapoNumber(value string) (float64, bool, error) {
	if value == "" {
		return 0, false, nil
	}
	parsed, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", ""), 64)
	if err != nil {
		return 0, false, err
	}
	return parsed, true, nil
}

func parseXapoTimestamp(value string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02 15:04:05", time.RFC3339, time.RFC3339Nano} {
		if timestamp, err := time.Parse(layout, value); err == nil {
			return timestamp.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported date format")
}

// ParsePhoenixRecord converts a slice of strings (a row from Phoenix CSV) into a PhoenixRecord struct.
func ParsePhoenixRecord(record []string) (*PhoenixRecord, error) {
	// Parse timestamp.
	timestamp, err := time.Parse(PhoenixDateFormat, record[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp '%s': %w", record[0], err)
	}

	amountMillisats := ParseIntField(record[3], "amount_msat")
	miningFeeSat := ParseIntField(record[6], "mining_fee_sat")
	serviceFeeMsat := ParseIntField(record[8], "service_fee_msat")

	return &PhoenixRecord{
		Timestamp:       timestamp,
		Type:            record[2],
		AmountMillisats: amountMillisats,
		MiningFeeSat:    miningFeeSat,
		ServiceFeeMsat:  serviceFeeMsat,
		TransactionID:   record[11],
		Description:     record[13],
	}, nil
}

// ToKoinlyRecord converts a PhoenixRecord into a KoinlyRecord.
// It maps different Phoenix transaction types to appropriate Koinly fields (Sent, Received, Fee).
func ToKoinlyRecord(p *PhoenixRecord) (*KoinlyRecord, float64) {
	// Note: Fees are often included in the sent/received amounts in Phoenix,
	// so they are not always tracked separately in Koinly unless explicitly a fee-only transaction.
	k := &KoinlyRecord{
		Date:        p.Timestamp.Format(KoinlyDateFormat),
		TxHash:      p.TransactionID,
		Description: p.Description,
	}

	// Convert amount from millisats to sats.
	sats := float64(p.AmountMillisats) / msatsPerSat
	absSats := math.Abs(sats)
	LogVerbose("Processing Phoenix Record: %+v", p)
	LogVerbose("Calculated Sats: %.8f", sats)

	var diff float64
	// Determine the Koinly record type based on Phoenix transaction type.
	switch p.Type {
	case "lightning_received":
		amt := FormatBTC(sats)
		k.ReceivedAmount = amt
		k.ReceivedCurrency = "BTC"
		k.Label = "lightning"
		LogVerbose("Type: lightning_received -> ReceivedAmount=%s BTC", k.ReceivedAmount)
		v, _ := strconv.ParseFloat(amt, 64)
		diff = sats - v*satsPerBTC
	case "lightning_sent":
		// For sent transactions, amount_msat is negative. Use absolute value.
		amt := FormatBTC(absSats)
		k.SentAmount = amt
		k.SentCurrency = "BTC"
		k.Label = "lightning"
		LogVerbose("Type: lightning_sent -> SentAmount=%s BTC", k.SentAmount)
		v, _ := strconv.ParseFloat(amt, 64)
		diff = sats - (-v * satsPerBTC)
	case "swap_in", "legacy_swap_in":
		// Swap-in is a receipt of funds.
		amt := FormatBTC(sats)
		k.ReceivedAmount = amt
		k.ReceivedCurrency = "BTC"
		k.Label = "transfer"
		LogVerbose("Type: %s -> ReceivedAmount=%s BTC", p.Type, k.ReceivedAmount)
		v, _ := strconv.ParseFloat(amt, 64)
		diff = sats - v*satsPerBTC
	case "swap_out":
		// Swap-out is a sending of funds.
		amt := FormatBTC(absSats)
		k.SentAmount = amt
		k.SentCurrency = "BTC"
		k.Label = "transfer"
		LogVerbose("Type: swap_out -> SentAmount=%s BTC", k.SentAmount)
		v, _ := strconv.ParseFloat(amt, 64)
		diff = sats - (-v * satsPerBTC)
	case "channel_open", "legacy_pay_to_open":
		// Channel open is treated as a deposit.
		amt := FormatBTC(sats)
		k.ReceivedAmount = amt
		k.ReceivedCurrency = "BTC"
		k.Label = "deposit"
		LogVerbose("Type: %s -> ReceivedAmount=%s BTC", p.Type, k.ReceivedAmount)
		v, _ := strconv.ParseFloat(amt, 64)
		diff = sats - v*satsPerBTC
	case "channel_close":
		// Channel close is treated as a cost (fee) in Koinly, as it's often just a fee settlement.
		k.SentAmount = ""
		k.SentCurrency = ""
		k.ReceivedAmount = ""
		k.ReceivedCurrency = ""
		amt := FormatBTC(absSats)
		k.FeeAmount = amt
		k.FeeCurrency = "BTC"
		k.Label = "cost"
		LogVerbose("Type: channel_close -> FeeAmount=%s BTC", k.FeeAmount)
		v, _ := strconv.ParseFloat(amt, 64)
		diff = sats - (-v * satsPerBTC)
	default:
		// Log unknown transaction types for awareness.
		log.Printf("Unknown transaction type for Koinly conversion: %s. This transaction will not be fully converted.", p.Type)
	}

	return k, diff
}

// ToStringSlice converts a KoinlyRecord struct into a slice of strings,
// suitable for writing as a row in a CSV file.
func (k *KoinlyRecord) ToStringSlice() []string {
	return []string{
		k.Date,
		k.SentAmount,
		k.SentCurrency,
		k.ReceivedAmount,
		k.ReceivedCurrency,
		k.FeeAmount,
		k.FeeCurrency,
		k.NetWorthAmount,
		k.NetWorthCurrency,
		k.Label,
		k.Description,
		k.TxHash,
	}
}
