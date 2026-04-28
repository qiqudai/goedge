package main

import (
	"reflect"
	"testing"
)

func TestParseMissingSharedLibraries(t *testing.T) {
	output := `
	linux-vdso.so.1 (0x00007ffc0edf5000)
	libluajit-5.1.so.2 => not found
	libpcre.so.3 => not found
	libssl.so.3 => /lib/x86_64-linux-gnu/libssl.so.3
`
	got := parseMissingSharedLibraries(output)
	want := []string{"libluajit-5.1.so.2", "libpcre.so.3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected missing libraries: got=%v want=%v", got, want)
	}
}

func TestRuntimePackagesForMissingLibrariesUbuntu(t *testing.T) {
	got := runtimePackagesForMissingLibraries(linuxDistroInfo{ID: "ubuntu", Version: "24.04"}, []string{
		"libluajit-5.1.so.2",
		"libpcre.so.3",
		"libpcre.so.3",
	})
	want := []string{"libluajit-5.1-2", "libpcre3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected packages: got=%v want=%v", got, want)
	}
}
