package providers

import "testing"

func TestDNSPodTC3RecordLineInvalidIsNotIgnorable(t *testing.T) {
	provider := &DNSPodProvider{}
	if provider.isIgnorableTC3("InvalidParameter.RecordLineInvalid", "") {
		t.Fatalf("RecordLineInvalid must not be ignored")
	}
}

func TestDNSPodTC3MissingRecordIsIgnorableForDeletePaths(t *testing.T) {
	provider := &DNSPodProvider{}
	if !provider.isIgnorableTC3("ResourceNotFound.NoDataOfRecord", "") {
		t.Fatalf("missing records should remain ignorable for delete/idempotent paths")
	}
}
