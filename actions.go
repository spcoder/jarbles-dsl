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
	CopyFile  func(string, string) Action
	ListDir   func(string) Action
	Compile   func(string, string) Action
}{
	ReadFile: func(safeDir string) Action {
		return Action{
			Name:        "read-file",
			Description: "reads a file",
			Function:    readFile(safeDir),
			Arguments: []ActionArguments{
				{
					Name:        "dir",
					Type:        "string",
					Description: "the directory of the file",
				},
				{
					Name:        "name",
					Type:        "string",
					Description: "the name of the file without the directory",
				},
			},
			RequiredArguments: []string{"dir", "name"},
		}
	},
	WriteFile: func(safeDir string) Action {
		return Action{
			Name:        "save-file",
			Description: "saves a file",
			Function:    saveFile(safeDir),
			Arguments: []ActionArguments{
				{
					Name:        "dir",
					Type:        "string",
					Description: "the directory of the file",
				},
				{
					Name:        "name",
					Type:        "string",
					Description: "the name of the file without the directory",
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
	CopyFile: func(safeSrc, safeDest string) Action {
		return Action{
			Name:        "copy-file",
			Description: "copies a file",
			Function:    copyFile(safeSrc, safeDest),
			Arguments: []ActionArguments{
				{
					Name:        "src",
					Type:        "string",
					Description: "the path of the source file",
				},
				{
					Name:        "dest",
					Type:        "string",
					Description: "the path of the destination file",
				},
			},
			RequiredArguments: []string{"src", "dest"},
		}
	},
	ListDir: func(safeDir string) Action {
		return Action{
			Name:        "list-directories",
			Description: "lists the directories in a directory",
			Function:    listDir(safeDir),
		}
	},
	// Compile compiles and builds a binary from go source code.
	// The go and goimports binaries must be in the PATH.
	// The entrypoint must be main.go.
	// Requires a go.mod file.
	Compile: func(safeSrc, safeDest string) Action {
		return Action{
			Name:        "build",
			Description: "compiles and builds a binary from go source code",
			Function:    compile(safeSrc, safeDest),
			Arguments: []ActionArguments{
				{
					Name:        "workingDir",
					Type:        "string",
					Description: "the working directory that contains the source code",
				},
				{
					Name:        "outputDir",
					Type:        "string",
					Description: "the output directory of the binary",
				},
				{
					Name:        "outputName",
					Type:        "string",
					Description: "the filename of the output binary without the directory",
				},
			},
			RequiredArguments: []string{"dir", "outputName"},
		}
	},
}

// safePath ensures that the file location specified by path is within the safeDir
func safePath(safeDir, baseDir, name string) (string, error) {
	path := filepath.Join(safeDir, strings.Replace(baseDir, safeDir, "", 1), strings.Replace(name, baseDir, "", 1))
	absPath, err := filepath.Abs(path)
	if err != nil {
		LogError("error while getting absolute path", "path", path, "error", err.Error())
		return "", fmt.Errorf("error while getting absolute path at %s: %w", path, err)
	}

	if !strings.HasPrefix(absPath, safeDir) {
		LogError("path is not within the safe directory", "safeDir", safeDir, "path", path)
		return "", fmt.Errorf("path is not within the safe directory: %s", absPath)
	}

	return absPath, nil
}

// safeDir ensures that the directory location specified by dir is within the safeDir
func safeDir(safeDir, dir string) (string, error) {
	path := filepath.Join(safeDir, strings.Replace(dir, safeDir, "", 1))
	absPath, err := filepath.Abs(path)
	if err != nil {
		LogError("error while getting absolute path", "dir", dir, "error", err.Error())
		return "", fmt.Errorf("error while getting absolute path at %s: %w", dir, err)
	}

	if !strings.HasPrefix(absPath, safeDir) {
		LogError("path is not within the safe directory", "safeDir", safeDir, "dir", dir)
		return "", fmt.Errorf("path is not within the safe directory: %s", absPath)
	}

	return absPath, nil
}

func readFile(safeDir string) ActionFunction {
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

		filename, err := safePath(safeDir, request.Dir, request.Name)
		if err != nil {
			LogError("error while getting safe path", "error", err.Error())
			return "", fmt.Errorf("error while getting safe path: %w", err)
		}

		data, err := os.ReadFile(filename)
		if err != nil {
			LogError("error while reading file", "filename", filename, "error", err.Error())
			return "", fmt.Errorf("error while reading file at %s: %s", filename, err)
		}

		LogDebug("file read successfully", "filename", filename)
		return string(data), nil
	}
}

func copyFile(safeSrc, safeDest string) ActionFunction {
	return func(payload string) (string, error) {
		var request struct {
			Src  string `json:"src"`
			Dest string `json:"dest"`
		}
		err := json.Unmarshal([]byte(payload), &request)
		if err != nil {
			LogError("error while unmarshaling payload", "error", err.Error())
			return "", fmt.Errorf("error while unmarshaling payload: %s", err)
		}

		LogDebug("copy-file", "src", request.Src, "dest", request.Dest)

		src, err := safePath(safeSrc, "", request.Src)
		if err != nil {
			LogError("error while getting safe src path", "error", err.Error())
			return "", fmt.Errorf("error while getting safe src path: %w", err)
		}

		dest, err := safePath(safeDest, "", request.Dest)
		if err != nil {
			LogError("error while getting safe dest path", "error", err.Error())
			return "", fmt.Errorf("error while getting safe dest path: %w", err)
		}

		err = os.MkdirAll(filepath.Dir(dest), os.ModePerm)
		if err != nil {
			LogError("error while making the destination directory ", "dir", filepath.Dir(dest), "error", err.Error())
			return "", fmt.Errorf("error while making the destination directory at %s: %s", filepath.Dir(dest), err)
		}

		// copy from src to dest using io.Copy
		srcFile, err := os.Open(src)
		if err != nil {
			LogError("error while opening source file", "src", src, "error", err.Error())
			return "", fmt.Errorf("error while opening source file at %s: %s", src, err)
		}
		defer func(f *os.File) {
			_ = f.Close()
		}(srcFile)

		destFile, err := os.Create(dest)
		if err != nil {
			LogError("error while creating destination file", "dest", dest, "error", err.Error())
			return "", fmt.Errorf("error while creating destination file at %s: %s", dest, err)
		}
		defer func(f *os.File) {
			_ = f.Close()
		}(destFile)

		_, err = io.Copy(destFile, srcFile)
		if err != nil {
			LogError("error while copying file", "src", src, "dest", dest, "error", err.Error())
			return "", fmt.Errorf("error while copying file from %s to %s: %s", src, dest, err)
		}

		err = destFile.Sync()
		if err != nil {
			LogError("error while syncing destination file", "dest", dest, "error", err.Error())
			return "", fmt.Errorf("error while syncing destination file at %s: %s", dest, err)
		}

		LogDebug("file copied successfully", "src", src, "dest", dest)
		return "file copied successfully", nil
	}
}

func saveFile(safeDir string) ActionFunction {
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

		filename, err := safePath(safeDir, request.Dir, request.Name)
		if err != nil {
			LogError("error while getting safe path", "error", err.Error())
			return "", fmt.Errorf("error while getting safe path: %w", err)
		}

		dirname := filepath.Dir(filename)
		err = os.MkdirAll(dirname, os.ModePerm)
		if err != nil {
			LogError("error while making the destination directory ", "dir", dirname, "error", err.Error())
			return "", fmt.Errorf("error while making the destination directory at %s: %s", dirname, err)
		}

		err = os.WriteFile(filename, []byte(request.Content), 0644)
		if err != nil {
			LogError("error while writing file", "filename", filename, "error", err.Error())
			return "", fmt.Errorf("error while writing file at %s: %s", filename, err)
		}

		LogDebug("file saved successfully", "filename", filename)
		return "file saved successfully", nil
	}
}

func listDir(safeDir string) ActionFunction {
	return func(_ string) (string, error) {
		var dirs []string
		err := filepath.WalkDir(safeDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				// # ignore .git directories
				if d.Name() == ".git" {
					return filepath.SkipDir
				}

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
			LogError("error while walking directory", "path", safeDir, "error", err.Error())
			return "", fmt.Errorf("error while walking directory at %s: %s", safeDir, err)
		}
		return strings.Join(dirs, "\n"), nil
	}
}

func compile(safeSrc, safeDest string) ActionFunction {
	return func(payload string) (string, error) {
		var request struct {
			WorkingDir string `json:"workingDir"`
			OutputDir  string `json:"outputDir"`
			OutputName string `json:"outputName"`
		}
		err := json.Unmarshal([]byte(payload), &request)
		if err != nil {
			LogError("error while unmarshaling payload", "error", err.Error())
			return "", fmt.Errorf("error while unmarshaling payload: %s", err)
		}

		workingDir, err := safeDir(safeSrc, request.WorkingDir)
		if err != nil {
			LogError("error while getting safe working directory", "error", err.Error())
			return "", fmt.Errorf("error while getting safe working directory: %w", err)
		}

		outputDir, err := safeDir(safeDest, request.OutputDir)
		if err != nil {
			LogError("error while getting safe output directory", "error", err.Error())
			return "", fmt.Errorf("error while getting safe output directory: %w", err)
		}

		LogDebug("compile", "workingDir", workingDir, "outputDir", outputDir, "outputName", request.OutputName)

		err = modTidyCommand(workingDir)
		if err != nil {
			return "", fmt.Errorf("error while downloading dependencies: %s", err)
		}

		err = goimportsCommand(workingDir)
		if err != nil {
			return "", fmt.Errorf("error while organizing imports: %s", err)
		}

		err = buildCommand(workingDir, outputDir, request.OutputName)
		if err != nil {
			return "", fmt.Errorf("error while building: %s", err)
		}

		return "compile completed successfully", nil
	}
}

func modTidyCommand(workingDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	LogDebug("downloading dependencies", "workingDir", workingDir)

	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = workingDir

	return runCommand(cmd)
}

func goimportsCommand(workingDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	mainFile := filepath.Join(workingDir, "main.go")
	LogDebug("organizing imports", "mainFile", mainFile, "workingDir", workingDir)

	cmd := exec.CommandContext(ctx, "goimports", "-w", mainFile)
	cmd.Dir = workingDir

	return runCommand(cmd)
}

func buildCommand(workingDir, outputDir, binaryName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	mainFile := filepath.Join(workingDir, "main.go")
	outputFile := filepath.Join(outputDir, binaryName)
	LogDebug("building", "workingDir", workingDir, "outputDir", outputDir, "binaryName", binaryName, "mainFile", mainFile, "outputFile", outputFile)

	cmd := exec.CommandContext(ctx, "go", "build", "-o", outputFile, mainFile)
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
		LogDebug("STDERR", "errdata", string(errdata))
		LogDebug("STDOUT", "outdata", string(outdata))
		LogError("error while waiting for command to finish", "error", err.Error())
		return fmt.Errorf("%s", errdata) // return the exact error message from the command
	}

	LogDebug("DATA", "outdata", string(outdata))
	return nil
}
