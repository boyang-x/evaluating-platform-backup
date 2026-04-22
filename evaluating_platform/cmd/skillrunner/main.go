package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	skillpkg "evaluating_platform/internal/skill"
)

func main() {
	port := 8091
	if raw := strings.TrimSpace(os.Getenv("SKILL_RUNNER_PORT")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			port = parsed
		}
	}

	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.POST("/api/v1/run", func(c *gin.Context) {
		var req skillpkg.RuntimeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := runSkill(c.Request.Context(), &req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": resp})
	})

	addr := fmt.Sprintf(":%d", port)
	log.Printf("skill runner listening on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func runSkill(ctx context.Context, req *skillpkg.RuntimeRequest) (*skillpkg.RuntimeResponse, error) {
	archiveBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.ArchiveBase64))
	if err != nil {
		return nil, fmt.Errorf("decode archive: %w", err)
	}
	if _, err := skillpkg.ParsePackage(archiveBytes, skillpkg.DefaultMaxArchiveBytes); err != nil {
		return nil, fmt.Errorf("validate skill package: %w", err)
	}
	if strings.TrimSpace(req.Entrypoint) == "" {
		return nil, fmt.Errorf("entrypoint is required")
	}

	runCtx := ctx
	if req.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	workdir, err := os.MkdirTemp("", "skill-run-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(workdir)

	if err := extractArchive(archiveBytes, workdir); err != nil {
		return nil, err
	}

	inputPath := filepath.Join(workdir, "runtime_input.json")
	inputBytes := []byte(`{}`)
	if len(bytes.TrimSpace(req.Input)) > 0 {
		inputBytes = req.Input
	}
	if err := os.WriteFile(inputPath, inputBytes, 0o644); err != nil {
		return nil, fmt.Errorf("write runtime input: %w", err)
	}
	outputDir := filepath.Join(workdir, "output")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	containerName := "skill-run-" + uuid.NewString()
	defer func() {
		_, _ = dockerCommand(context.Background(), "rm", "-f", containerName)
	}()

	command := "set -e; if [ -f requirements.txt ]; then python -m pip install --no-cache-dir -r requirements.txt; fi; python " + shQuote(req.Entrypoint)
	createArgs := []string{
		"create",
		"--name", containerName,
		"--workdir", "/workspace",
		"-e", "SKILL_MODE=" + req.RunType,
		"-e", "SKILL_INPUT_PATH=/workspace/runtime_input.json",
		"-e", "SKILL_OUTPUT_PATH=/workspace/output/result.json",
		"-e", "SKILL_VALIDATION_REPORT_PATH=/workspace/output/validation-report.json",
	}
	createArgs = appendEnvIfPresent(createArgs,
		"LLM_API_KEY",
		"LLM_BASE_URL",
		"LLM_MODEL",
		"SKILL_LLM_API_KEY",
		"SKILL_LLM_BASE_URL",
		"SKILL_LLM_MODEL",
	)
	createArgs = appendExplicitEnv(createArgs, req.Env)
	if network := strings.TrimSpace(os.Getenv("SKILL_RUNNER_DOCKER_NETWORK")); network != "" {
		createArgs = append(createArgs, "--network", network)
	}
	createArgs = append(createArgs, "python:3.12-slim", "sh", "-lc", command)
	if _, err := dockerCommand(runCtx, createArgs...); err != nil {
		return nil, fmt.Errorf("docker create failed: %w", err)
	}

	if _, err := dockerCommand(runCtx, "cp", filepath.ToSlash(workdir)+"/.", containerName+":/workspace"); err != nil {
		return nil, fmt.Errorf("docker cp workspace failed: %w", err)
	}
	if _, err := dockerCommand(runCtx, "start", containerName); err != nil {
		return nil, fmt.Errorf("docker start failed: %w", err)
	}

	waitOutput, waitErr := dockerCommand(runCtx, "wait", containerName)
	logsOutput, _ := dockerCommand(context.Background(), "logs", containerName)
	if waitErr != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return &skillpkg.RuntimeResponse{
				Status: "timeout",
				Stdout: logsOutput,
				Error:  "skill execution timed out",
			}, nil
		}
		return nil, fmt.Errorf("docker wait failed: %w", waitErr)
	}

	exitCode := 1
	if parsed, err := strconv.Atoi(strings.TrimSpace(waitOutput)); err == nil {
		exitCode = parsed
	}

	resultData, _ := copyOut(containerName, "/workspace/output/result.json", filepath.Join(workdir, "result.json"))
	validationData, _ := copyOut(containerName, "/workspace/output/validation-report.json", filepath.Join(workdir, "validation-report.json"))

	status := "completed"
	if exitCode != 0 {
		status = "failed"
	}
	return &skillpkg.RuntimeResponse{
		Status:           status,
		ExitCode:         &exitCode,
		Stdout:           logsOutput,
		Result:           json.RawMessage(defaultJSON(resultData, `{}`)),
		ValidationReport: json.RawMessage(defaultJSON(validationData, `{}`)),
		Error:            errorFromExitCode(exitCode),
	}, nil
}

func dockerCommand(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		if text == "" {
			return "", err
		}
		return text, fmt.Errorf("%w: %s", err, text)
	}
	return text, nil
}

func extractArchive(archive []byte, dest string) error {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	for _, file := range reader.File {
		name := strings.ReplaceAll(file.Name, "\\", "/")
		cleaned := filepath.Clean(name)
		if strings.Contains(cleaned, "..") || filepath.IsAbs(cleaned) {
			return fmt.Errorf("unsafe archive path: %s", file.Name)
		}
		target := filepath.Join(dest, cleaned)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("create dir %s: %w", cleaned, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create parent dir %s: %w", cleaned, err)
		}
		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("open %s: %w", cleaned, err)
		}
		out, err := os.Create(target)
		if err != nil {
			_ = rc.Close()
			return fmt.Errorf("create %s: %w", cleaned, err)
		}
		if _, err := io.Copy(out, rc); err != nil {
			_ = out.Close()
			_ = rc.Close()
			return fmt.Errorf("write %s: %w", cleaned, err)
		}
		_ = out.Close()
		_ = rc.Close()
	}
	return nil
}

func copyOut(containerName, sourcePath, targetPath string) ([]byte, error) {
	cmd := exec.Command("docker", "cp", containerName+":"+sourcePath, targetPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		text := strings.TrimSpace(string(output))
		if strings.Contains(text, "No such container:path") || strings.Contains(text, "Could not find the file") {
			return nil, nil
		}
		return nil, err
	}
	data, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func shQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func defaultJSON(data []byte, fallback string) []byte {
	if len(bytes.TrimSpace(data)) == 0 {
		return []byte(fallback)
	}
	return data
}

func errorFromExitCode(exitCode int) string {
	if exitCode == 0 {
		return ""
	}
	return fmt.Sprintf("skill process exited with code %d", exitCode)
}

func appendEnvIfPresent(args []string, keys ...string) []string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			args = append(args, "-e", key+"="+value)
		}
	}
	return args
}

func appendExplicitEnv(args []string, env map[string]string) []string {
	if len(env) == 0 {
		return args
	}
	keys := make([]string, 0, len(env))
	for key := range env {
		if strings.TrimSpace(key) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		args = append(args, "-e", key+"="+env[key])
	}
	return args
}
