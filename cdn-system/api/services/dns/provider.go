package dns

type Provider interface {
	GetDomains() ([]string, error)
	GetRecords(domain string) ([]DNSRecord, error)
	AddRecord(domain string, record DNSRecord) error
	DeleteRecord(domain string, record DNSRecord) error
}

// LineRecordDeleter optionally deletes all records for the same name + line.
type LineRecordDeleter interface {
	DeleteRecordsByLine(domain string, record DNSRecord) error
}

// RecordSetUpdater optionally updates a record set with multiple values.
// providers like Huawei/GoDaddy support replacing all values for a name/type.
type RecordSetUpdater interface {
	UpsertRecordSet(domain string, record DNSRecord, values []string) error
}

// RecordValueReplacer optionally updates a single record value in place.
// Implementations should match by name/type/line and optionally record.Value if provided.
type RecordValueReplacer interface {
	ReplaceRecordValue(domain string, record DNSRecord, newValue string) error
}

type DNSRecord struct {
	Type   string // A, CNAME, TXT
	Name   string // @, www, etc.
	Value  string
	TTL    int
	Line   string // vendor-specific line value
	Weight int    // vendor-specific weight
}

type ProviderFactory func(credentials string) (Provider, error)

var providers = make(map[string]ProviderFactory)

func RegisterProvider(name string, factory ProviderFactory) {
	providers[name] = factory
}

func GetProvider(name string, credentials string) (Provider, error) {
	if factory, ok := providers[name]; ok {
		return factory(credentials)
	}
	return nil, nil
}
