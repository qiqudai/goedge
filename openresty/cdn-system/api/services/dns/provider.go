package dns

type Provider interface {
	GetDomains() ([]string, error)
	AddRecord(domain string, record DNSRecord) error
	DeleteRecord(domain string, record DNSRecord) error
}

type DNSRecord struct {
	Type  string // A, CNAME, TXT
	Name  string // @, www, etc.
	Value string
	TTL   int
	Line  string // vendor-specific line value
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
