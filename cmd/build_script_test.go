package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildScriptDoesNotPackageFailedBuilds(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("build.sh requires a POSIX shell")
	}

	script, err := filepath.Abs("../build.sh")
	if err != nil {
		t.Fatal(err)
	}

	for _, failedCommand := range []string{"go get", "go build", "npm install", "npm run build", ""} {
		name := failedCommand
		if name == "" {
			name = "successful package"
		}
		t.Run(name, func(t *testing.T) {
			workDir := t.TempDir()
			binDir := filepath.Join(workDir, "bin")
			for _, dir := range []string{"bin", "dist", "conf", "templates"} {
				if err := os.Mkdir(filepath.Join(workDir, dir), 0700); err != nil {
					t.Fatal(err)
				}
			}

			files := map[string]string{
				"package.json":  "{\"version\": \"1.0.0\"}\n",
				"ezbookkeeping": "stale executable\n",
				"LICENSE":       "test license\n",
				"bin/git":       "#!/bin/sh\nprintf 'abc1234\\n'\n",
				"bin/go":        "#!/bin/sh\nif [ \"go $1\" = \"$FAILED_COMMAND\" ]; then exit 42; fi\n",
				"bin/npm":       "#!/bin/sh\nif [ \"npm $*\" = \"$FAILED_COMMAND\" ]; then exit 42; fi\n",
				"bin/gcc":       "#!/bin/sh\nexit 0\n",
				"bin/clang":     "#!/bin/sh\nexit 0\n",
				"bin/node":      "#!/bin/sh\nexit 0\n",
			}
			for name, content := range files {
				if err := os.WriteFile(filepath.Join(workDir, name), []byte(content), 0700); err != nil {
					t.Fatal(err)
				}
			}

			command := exec.Command("sh", script, "package", "--no-lint", "--no-test", "-o", "artifact.tar.gz")
			command.Dir = workDir
			for _, entry := range os.Environ() {
				if !strings.HasPrefix(entry, "PATH=") && !strings.HasPrefix(entry, "FAILED_COMMAND=") {
					command.Env = append(command.Env, entry)
				}
			}
			command.Env = append(command.Env, "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"), "FAILED_COMMAND="+failedCommand)
			output, err := command.CombinedOutput()
			_, archiveErr := os.Stat(filepath.Join(workDir, "artifact.tar.gz"))

			if failedCommand == "" {
				if err != nil || archiveErr != nil {
					t.Fatalf("successful build did not produce an archive: command=%v, archive=%v\n%s", err, archiveErr, output)
				}
			} else {
				if err == nil {
					t.Errorf("failed %s did not stop the build\n%s", failedCommand, output)
				}
				if !os.IsNotExist(archiveErr) {
					t.Errorf("failed %s produced an archive from stale output: %v", failedCommand, archiveErr)
				}
			}
		})
	}
}
