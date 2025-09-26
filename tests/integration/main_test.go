package integration

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestBrowserPluginSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping smoke test in short mode")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skipf("docker not available: %v", err)
	}

	repoRoot := repoRootDir(t)
	agentPath := filepath.Join(repoRoot, "build", "bin", "browser-agent")
	if _, err := os.Stat(agentPath); err != nil {
		buildCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(buildCtx, "make", "-C", repoRoot, "build")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if runErr := cmd.Run(); runErr != nil {
			t.Fatalf("make build failed: %v", runErr)
		}
	}

	stagedRel := filepath.ToSlash(filepath.Join("runtime", "browser-agent.bin"))
	stagedPath := filepath.Join(repoRoot, stagedRel)
	if err := copyFile(agentPath, stagedPath); err != nil {
		t.Fatalf("stage agent binary: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(stagedPath) })

	port := freePort(t)
	imageTag := fmt.Sprintf("browser-plugin-smoke:%d", time.Now().UnixNano())
	projectName := fmt.Sprintf("volant_browser_smoke_%d", time.Now().UnixNano())
	t.Setenv("BROWSER_PLUGIN_PORT", strconv.Itoa(port))
	t.Setenv("BROWSER_PLUGIN_IMAGE", imageTag)
	t.Setenv("AGENT_BINARY", stagedRel)
	t.Setenv("COMPOSE_PROJECT_NAME", projectName)

	composeDir := filepath.Join(repoRoot, "tests", "integration")
	down := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		exec.CommandContext(ctx, "docker", "compose", "down", "-v", "--remove-orphans").Run()
	}
	t.Cleanup(down)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		exec.CommandContext(ctx, "docker", "image", "rm", "-f", imageTag).Run()
	}()

	upCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	cmd := exec.CommandContext(upCtx, "docker", "compose", "up", "--build", "-d")
	cmd.Dir = composeDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		cancel()
		printComposeLogs(composeDir)
		t.Fatalf("docker compose up failed: %v", err)
	}
	cancel()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := &http.Client{Timeout: 5 * time.Second}

	if err := waitForHealthy(client, baseURL+"/healthz", 45*time.Second); err != nil {
		printComposeLogs(composeDir)
		t.Fatalf("health check failed: %v", err)
	}

	navigatePayload, _ := json.Marshal(map[string]any{"url": "https://example.com"})
	if err := postJSON(context.Background(), client, baseURL+"/v1/browser/navigate", navigatePayload); err != nil {
		printComposeLogs(composeDir)
		t.Fatalf("navigate action failed: %v", err)
	}

	screenshotPayload, _ := json.Marshal(map[string]any{"full_page": false, "format": "png"})
	resp, err := client.Post(baseURL+"/v1/browser/screenshot", "application/json", bytes.NewReader(screenshotPayload))
	if err != nil {
		printComposeLogs(composeDir)
		t.Fatalf("screenshot request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		printComposeLogs(composeDir)
		t.Fatalf("unexpected screenshot status %d: %s", resp.StatusCode, string(data))
	}

	var screenshot struct {
		Data       string `json:"data"`
		Format     string `json:"format"`
		FullPage   bool   `json:"full_page"`
		ByteLength int    `json:"byte_length"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&screenshot); err != nil {
		t.Fatalf("decode screenshot response: %v", err)
	}
	imgBytes, err := base64.StdEncoding.DecodeString(screenshot.Data)
	if err != nil {
		t.Fatalf("decode screenshot data: %v", err)
	}
	if len(imgBytes) == 0 {
		t.Fatal("empty screenshot data")
	}
	if screenshot.ByteLength != 0 && screenshot.ByteLength != len(imgBytes) {
		t.Fatalf("screenshot byte length mismatch: expected %d got %d", screenshot.ByteLength, len(imgBytes))
	}
	if screenshot.Format != "png" {
		t.Fatalf("unexpected screenshot format: %s", screenshot.Format)
	}
}

func repoRootDir(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	return filepath.Clean(filepath.Join(cwd, "../.."))
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func waitForHealthy(client *http.Client, url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		if time.Now().After(deadline) {
			if err != nil {
				return err
			}
			if resp != nil {
				return fmt.Errorf("unexpected status %d", resp.StatusCode)
			}
			return fmt.Errorf("health check timeout")
		}
		time.Sleep(1 * time.Second)
	}
}

func postJSON(ctx context.Context, client *http.Client, url string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(data))
	}
	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return dstFile.Sync()
}

func printComposeLogs(composeDir string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "compose", "logs", "--no-color")
	cmd.Dir = composeDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}
