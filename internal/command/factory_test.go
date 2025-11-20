package command

import (
	"testing"
)

func TestNewCommand(t *testing.T) {
	tests := []struct {
		name    string
		subCmd  string
		want    interface{} // We check type
		wantErr bool
	}{
		{
			name:    "ssh command (empty)",
			subCmd:  "",
			want:    &SSHCommand{},
			wantErr: false,
		},
		{
			name:    "ssh command (sh)",
			subCmd:  "sh",
			want:    &SSHCommand{},
			wantErr: false,
		},
		{
			name:    "ip command",
			subCmd:  "ip",
			want:    &IPCommand{},
			wantErr: false,
		},
		{
			name:    "push command",
			subCmd:  "push",
			want:    &RsyncCommand{},
			wantErr: false,
		},
		{
			name:    "pull command",
			subCmd:  "pull",
			want:    &RsyncCommand{},
			wantErr: false,
		},
		{
			name:    "tunnel command",
			subCmd:  "tunnel",
			want:    &TunnelCommand{},
			wantErr: false,
		},
		{
			name:    "unknown command",
			subCmd:  "unknown",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCommand(tt.subCmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Simple type check
				switch wantType := tt.want.(type) {
				case *SSHCommand:
					if _, ok := got.(*SSHCommand); !ok {
						t.Errorf("NewCommand() got = %T, want %T", got, wantType)
					}
				case *RsyncCommand:
					if _, ok := got.(*RsyncCommand); !ok {
						t.Errorf("NewCommand() got = %T, want %T", got, wantType)
					}
				case *IPCommand:
					if _, ok := got.(*IPCommand); !ok {
						t.Errorf("NewCommand() got = %T, want %T", got, wantType)
					}
				case *TunnelCommand:
					if _, ok := got.(*TunnelCommand); !ok {
						t.Errorf("NewCommand() got = %T, want %T", got, wantType)
					}
				}
			}
		})
	}
}

