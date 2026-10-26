package notify

import "testing"

func TestDeliveryForLearnerPresence(t *testing.T) {
	tests := []struct {
		name   string
		online bool
		want   Delivery
	}{
		{name: "connected learner receives the live event", online: true, want: Realtime},
		{name: "offline learner keeps a durable notification", online: false, want: Durable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DeliveryFor(test.online); got != test.want {
				t.Fatalf("DeliveryFor(%v) = %q, want %q", test.online, got, test.want)
			}
		})
	}
}
