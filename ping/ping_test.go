package ping

import "testing"

func TestPinger_network(t *testing.T) {
	type fields struct {
		Unprivileged bool
		Count        int
		Interval     int
		Interface    string
		Timestamp    bool
		Quite        bool
		TTL          int
		Timeout      int
		Network      string
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "udp4 case 1",
			fields: fields{
				Unprivileged: true,
				Network:      "",
			},
			want:    "udp4",
			wantErr: false,
		},
		{
			name: "udp4 case 2",
			fields: fields{
				Unprivileged: true,
				Network:      "ip",
			},
			want:    "udp4",
			wantErr: false,
		},
		{
			name: "udp4 case 3",
			fields: fields{
				Unprivileged: true,
				Network:      "ip4",
			},
			want:    "udp4",
			wantErr: false,
		},
		{
			name: "udp6",
			fields: fields{
				Unprivileged: true,
				Network:      "ip6",
			},
			want:    "udp4",
			wantErr: false,
		},
		{
			name: "icmp and ip",
			fields: fields{
				Unprivileged: false,
				Network:      "ip",
			},
			want:    "ip:icmp",
			wantErr: false,
		},
		{
			name: "icmp",
			fields: fields{
				Unprivileged: false,
				Network:      "",
			},
			want:    "ip:icmp",
			wantErr: false,
		},
		{
			name: "icmp v4",
			fields: fields{
				Unprivileged: false,
				Network:      "ip4",
			},
			want:    "ip4:icmp",
			wantErr: false,
		},
		{
			name: "icmp v6",
			fields: fields{
				Unprivileged: false,
				Network:      "ip6",
			},
			want:    "ip6:ipv6-icmp",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pinger{
				Unprivileged: tt.fields.Unprivileged,
				Count:        tt.fields.Count,
				Interval:     tt.fields.Interval,
				Interface:    tt.fields.Interface,
				Timestamp:    tt.fields.Timestamp,
				Quite:        tt.fields.Quite,
				TTL:          tt.fields.TTL,
				Timeout:      tt.fields.Timeout,
				Network:      tt.fields.Network,
			}
			got, err := p.network()
			if (err != nil) != tt.wantErr {
				t.Errorf("network() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("network() got = %v, want %v", got, tt.want)
			}
		})
	}
}
