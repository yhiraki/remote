package command

import (
	"os"
	"reflect"
	"testing"
)

func TestRsyncCommand_build(t *testing.T) {
	// Helper to create a dummy file for testing
	createTempFile := func(t *testing.T) string {
		f, err := os.CreateTemp("", "remote_test")
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
		return f.Name()
	}
	defer os.RemoveAll(os.TempDir()) // Not perfect cleanup but sufficient for unit tests

	tmpFile := createTempFile(t)
	defer os.Remove(tmpFile)

	tests := []struct {
		name         string
		direction    string
		remoteHost   string
		subCmdArgs   []string
		excludeFiles []string
		cwdRel       string
		wantCmd      string
		wantArgs     []string
		wantErr      bool
	}{
		{
			name:         "push existing file",
			direction:    "push",
			remoteHost:   "example.com",
			subCmdArgs:   []string{tmpFile},
			excludeFiles: nil,
			cwdRel:       ".",
			wantCmd:      "rsync",
			wantArgs:     []string{"-av", tmpFile, "example.com:" + tmpFile},
			wantErr:      false,
		},
		{
			name:         "push non-existing file",
			direction:    "push",
			remoteHost:   "example.com",
			subCmdArgs:   []string{"/non/existing/file"},
			excludeFiles: nil,
			cwdRel:       ".",
			wantErr:      true,
		},
		{
			name:         "pull file",
			direction:    "pull",
			remoteHost:   "example.com",
			subCmdArgs:   []string{"local_dest"},
			excludeFiles: nil,
			cwdRel:       ".",
			wantCmd:      "rsync",
			wantArgs:     []string{"-av", "--ignore-existing", "example.com:local_dest", "local_dest"},
			wantErr:      false,
		},
		{
			name:         "push with excludes",
			direction:    "push",
			remoteHost:   "example.com",
			subCmdArgs:   []string{tmpFile},
			excludeFiles: []string{".git", "node_modules"},
			cwdRel:       ".",
			wantCmd:      "rsync",
			wantArgs:     []string{"--exclude", ".git", "--exclude", "node_modules", "-av", tmpFile, "example.com:" + tmpFile},
			wantErr:      false,
		},
		{
			name:         "missing args",
			direction:    "push",
			remoteHost:   "example.com",
			subCmdArgs:   []string{},
			excludeFiles: nil,
			cwdRel:       ".",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &RsyncCommand{Direction: tt.direction}
			gotCmd, gotArgs, err := c.build(tt.remoteHost, tt.subCmdArgs, tt.excludeFiles, tt.cwdRel)
			if (err != nil) != tt.wantErr {
				t.Errorf("RsyncCommand.build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotCmd != tt.wantCmd {
					t.Errorf("RsyncCommand.build() cmd = %v, want %v", gotCmd, tt.wantCmd)
				}
				if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
					t.Errorf("RsyncCommand.build() args = %v, want %v", gotArgs, tt.wantArgs)
				}
			}
		})
	}
}

