package apns

import (
	"testing"

	"github.com/sideshow/apns2/token"
)

func TestEmbeddedP8Parses(t *testing.T) {
	if _, err := token.AuthKeyFromBytes([]byte(apnsPrivateKey)); err != nil {
		t.Fatal(err)
	}
	if keyID != "54GN6CDZTS" || teamID != "Z9K94479DQ" || bundleID != "com.mistj.nodeseek" {
		t.Fatalf("creds key=%s team=%s topic=%s", keyID, teamID, bundleID)
	}
}
