package jarbles_framework

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//goland:noinspection GoUnusedGlobalVariable
var StandardActions = struct {
	ReadFile  func(string) Action
	WriteFile func(string) Action
	ListDir   func(string) Action
	Compile   func(string) Action
}{
	ReadFile: func(baseDir string) Action {
		return Action{
			Name:        "read-file",
			Description: "reads a file",
			Function:    readFile(baseDir),
			Arguments: []ActionArguments{
				{
					Name:        "dir",
					Type:        "string",
					Description: "the directory of the file",
				},
				{
					Name:        "name",
					Type:        "string",
					Description: "the name of the file",
				},
			},
			RequiredArguments: []string{"dir", "name"},
		}
	},
	WriteFile: func(baseDir string) Action {
		return Action{
			Name:        "save-file",
			Description: "saves a file",
			Function:    saveFile(baseDir),
			Arguments: []ActionArguments{
				{
					Name:        "dir",
					Type:        "string",
					Description: "the directory of the file",
				},
				{
					Name:        "name",
					Type:        "string",
					Description: "the name of the file",
				},
				{
					Name:        "content",
					Type:        "string",
					Description: "the contents of the file",
				},
			},
			RequiredArguments: []string{"dir", "name", "content"},
		}
	},
	ListDir: func(baseDir string) Action {
		return Action{
			Name:        "list-directories",
			Description: "lists the directories in a directory",
			Function:    listDir(baseDir),
		}
	},
	// Compile compiles and builds a binary from go source code.
	// The go and goimports binaries must be in the PATH.
	// The entrypoint must be main.go.
	// Requires a go.mod file.
	Compile: func(baseDir string) Action {
		return Action{
			Name:        "build",
			Description: "compiles and builds a binary from go source code",
			Function:    compile(baseDir),
			Arguments: []ActionArguments{
				{
					Name:        "dir",
					Type:        "string",
					Description: "the directory that contains the source code",
				},
				{
					Name:        "outputName",
					Type:        "string",
					Description: "the filename of the output binary",
				},
			},
			RequiredArguments: []string{"dir", "outputName"},
		}
	},
}

func readFile(baseDir string) ActionFunction {
	return func(payload string) (string, error) {
		var request struct {
			Dir  string `json:"dir"`
			Name string `json:"name"`
		}
		err := json.Unmarshal([]byte(payload), &request)
		if err != nil {
			LogError("error while unmarshaling payload", "error", err.Error())
			return "", fmt.Errorf("error while unmarshaling payload: %s", err)
		}

		LogDebug("read-file", "dir", request.Dir, "name", request.Name)

		filename := filepath.Join(baseDir, request.Dir, request.Name)
		data, err := os.ReadFile(filename)
		if err != nil {
			LogError("error while reading file", "filename", filename, "error", err.Error())
			return "", fmt.Errorf("error while reading file at %s: %s", filename, err)
		}

		LogDebug("file read successfully", "filename", filename)
		return string(data), nil
	}
}

func saveFile(baseDir string) ActionFunction {
	return func(payload string) (string, error) {
		var request struct {
			Dir     string `json:"dir"`
			Name    string `json:"name"`
			Content string `json:"content"`
		}
		err := json.Unmarshal([]byte(payload), &request)
		if err != nil {
			LogError("error while unmarshaling payload", "error", err.Error())
			return "", fmt.Errorf("error while unmarshaling payload: %s", err)
		}

		LogDebug("save-file", "dir", request.Dir, "name", request.Name)

		dirname := filepath.Join(baseDir, request.Dir)
		err = os.MkdirAll(dirname, os.ModePerm)
		if err != nil {
			LogError("error while making the destination directory ", "dir", dirname, "error", err.Error())
			return "", fmt.Errorf("error while making the destination directory at %s: %s", dirname, err)
		}

		filename := filepath.Join(dirname, request.Name)
		err = os.WriteFile(filename, []byte(request.Content), 0644)
		if err != nil {
			LogError("error while writing file", "filename", filename, "error", err.Error())
			return "", fmt.Errorf("error while writing file at %s: %s", filename, err)
		}

		LogDebug("file saved successfully", "filename", filename)
		return "file saved successfully", nil
	}
}

func listDir(baseDir string) ActionFunction {
	return func(_ string) (string, error) {
		var dirs []string
		err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				abspath, err := filepath.Abs(path)
				if err != nil {
					LogError("error while getting absolute path", "path", path, "error", err.Error())
					return fmt.Errorf("error while getting absolute path at %s: %w", path, err)
				}
				dirs = append(dirs, abspath)
			}
			return nil
		})
		if err != nil {
			LogError("error while walking directory", "path", baseDir, "error", err.Error())
			return "", fmt.Errorf("error while walking directory at %s: %s", baseDir, err)
		}
		return strings.Join(dirs, "\n"), nil
	}
}

func compile(baseDir string) ActionFunction {
	return func(payload string) (string, error) {
		var request struct {
			Dir        string `json:"dir"`
			OutputName string `json:"outputName"`
		}
		err := json.Unmarshal([]byte(payload), &request)
		if err != nil {
			LogError("error while unmarshaling payload", "error", err.Error())
			return "", fmt.Errorf("error while unmarshaling payload: %s", err)
		}

		workingDir := filepath.Join(baseDir, request.Dir)

		err = modTidyCommand(workingDir)
		if err != nil {
			return "", fmt.Errorf("error while downloading dependencies: %s", err)
		}

		err = goimportsCommand(workingDir)
		if err != nil {
			return "", fmt.Errorf("error while organizing imports: %s", err)
		}

		err = buildCommand(workingDir, request.OutputName)
		if err != nil {
			return "", fmt.Errorf("error while building: %s", err)
		}

		return "compile completed successfully", nil
	}
}

func modTidyCommand(workingDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = workingDir

	return runCommand(cmd)
}

func goimportsCommand(workingDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "goimports", "-w", filepath.Join(workingDir, "main.go"))
	cmd.Dir = workingDir

	return runCommand(cmd)
}

func buildCommand(workingDir, binaryName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "build", "-o", binaryName, filepath.Join(workingDir, "main.go"))
	cmd.Dir = workingDir

	return runCommand(cmd)
}

func runCommand(cmd *exec.Cmd) error {
	LogInfo("running command", "command", cmd)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	err := cmd.Start()
	if err != nil {
		LogError("error while starting the command", "error", err.Error())
		return fmt.Errorf("error while starting the command: %w", err)
	}

	errdata, err := io.ReadAll(stderr)
	if err != nil {
		LogError("error while reading standard error", "error", err.Error())
		return fmt.Errorf("error while reading standard error: %w", err)
	}

	outdata, err := io.ReadAll(stdout)
	if err != nil {
		LogError("error while reading standard output", "error", err.Error())
		return fmt.Errorf("error while reading standard output: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		LogDebug("STDERR", string(errdata))
		LogDebug("STDOUT", string(outdata))
		LogError("error while waiting for command to finish", "error", err.Error())
		return fmt.Errorf("%s", errdata) // return the exact error message from the command
	}

	LogDebug("DATA", string(outdata))
	return nil
}
