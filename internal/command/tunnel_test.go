package command

import (
	"reflect"
	"testing"
)

func TestTunnelCommand_build(t *testing.T) {
	tests := []struct {
		name         string
		remoteHost   string
		subCmdArgs   []string
		isBackground bool
		isVerbose    bool
		wantCmd      string
		wantArgs     []string
		wantErr      bool
	}{
		{
			name:         "single port",
			remoteHost:   "example.com",
			subCmdArgs:   []string{"8080"},
			isBackground: false,
			isVerbose:    false,
			wantCmd:      "ssh",
			wantArgs:     []string{"-N", "-L", "8080:localhost:8080", "example.com"},
			wantErr:      false,
		},
		{
			name:         "multiple ports",
			remoteHost:   "example.com",
			subCmdArgs:   []string{"8080", "3000"},
			isBackground: false,
			isVerbose:    false,
			wantCmd:      "ssh",
			wantArgs:     []string{"-N", "-L", "8080:localhost:8080", "-L", "3000:localhost:3000", "example.com"},
			wantErr:      false,
		},
		{
			name:         "background mode",
			remoteHost:   "example.com",
			subCmdArgs:   []string{"8080"},
			isBackground: true,
			isVerbose:    false,
			wantCmd:      "ssh",
			wantArgs:     []string{"-N", "-f", "-L", "8080:localhost:8080", "example.com"},
			wantErr:      false,
		},
		{
			name:         "missing ports",
			remoteHost:   "example.com",
			subCmdArgs:   []string{},
			isBackground: false,
			isVerbose:    false,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &TunnelCommand{}
			gotCmd, gotArgs, err := c.build(tt.remoteHost, tt.subCmdArgs, tt.isBackground, tt.isVerbose)
			if (err != nil) != tt.wantErr {
				t.Errorf("TunnelCommand.build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotCmd != tt.wantCmd {
					t.Errorf("TunnelCommand.build() cmd = %v, want %v", gotCmd, tt.wantCmd)
				}
				if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
					t.Errorf("TunnelCommand.build() args = %v, want %v", gotArgs, tt.wantArgs)
				}
			}
		})
	}
}

