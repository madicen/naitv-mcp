package entry_test

import (
	"testing"

	"github.com/madicen/naitv-mcp/pkg/entry"
)

func TestDeliveryOrDefault(t *testing.T) {
	if got := (entry.Entry{}).DeliveryOrDefault(); got != entry.DeliveryInit {
		t.Fatalf("default = %q", got)
	}
	if got := (entry.Entry{Delivery: entry.DeliveryOnDemand}).DeliveryOrDefault(); got != entry.DeliveryOnDemand {
		t.Fatalf("ondemand = %q", got)
	}
}
