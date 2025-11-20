package command

import (
	"reflect"
	"testing"
)

func TestSSHCommand_build(t *testing.T) {
	tests := []struct {
		name       string
		remoteHost string
		subCmdArgs []string
		envVars    []string
		cwdRel     string
		wantCmd    string
		wantArgs   []string
	}{
		{
			name:       "default shell",
			remoteHost: "example.com",
			subCmdArgs: []string{},
			envVars:    nil,
			cwdRel:     ".",
			wantCmd:    "ssh",
			wantArgs:   []string{"example.com", "-t", "cd '.'; exec $SHELL"},
		},
		{
			name:       "simple command",
			remoteHost: "example.com",
			subCmdArgs: []string{"ls", "-la"},
			envVars:    nil,
			cwdRel:     "src",
			wantCmd:    "ssh",
			wantArgs:   []string{"example.com", "-t", "cd 'src'; exec  sh -c 'ls -la'"},
		},
		{
			name:       "command with env vars",
			remoteHost: "example.com",
			subCmdArgs: []string{"env"},
			envVars:    []string{"FOO=BAR", "BAZ=QUX"},
			cwdRel:     ".",
			wantCmd:    "ssh",
			wantArgs:   []string{"example.com", "-t", "cd '.'; exec env 'FOO=BAR' 'BAZ=QUX'  sh -c 'env'"},
		},
		{
			name:       "command with single quotes",
			remoteHost: "example.com",
			subCmdArgs: []string{"echo", "'hello'"},
			envVars:    nil,
			cwdRel:     ".",
			wantCmd:    "ssh",
			wantArgs:   []string{"example.com", "-t", "cd '.'; exec  sh -c 'echo '\\''hello'\\'''"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &SSHCommand{}
			gotCmd, gotArgs, err := c.build(tt.remoteHost, tt.subCmdArgs, tt.envVars, tt.cwdRel)
			if err != nil {
				t.Errorf("SSHCommand.build() error = %v", err)
				return
			}
			if gotCmd != tt.wantCmd {
				t.Errorf("SSHCommand.build() cmd = %v, want %v", gotCmd, tt.wantCmd)
			}
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("SSHCommand.build() args = %v, want %v", gotArgs, tt.wantArgs)
			}
		})
	}
}

