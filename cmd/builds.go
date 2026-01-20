package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var buildsCmd = &cobra.Command{
	Use:   "builds",
	Short: "Manage builds",
	Long: `Manage builds on the Cozy platform.

Subcommands:
  list    List recent builds
  logs    View build logs
  cancel  Cancel a running build`,
}

var buildsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent builds",
	RunE:  runBuildsList,
}

var buildsLogsCmd = &cobra.Command{
	Use:   "logs <build_id>",
	Short: "View build logs",
	Args:  cobra.ExactArgs(1),
	RunE:  runBuildsLogs,
}

var buildsCancelCmd = &cobra.Command{
	Use:   "cancel <build_id>",
	Short: "Cancel a running build",
	Args:  cobra.ExactArgs(1),
	RunE:  runBuildsCancel,
}

var (
	buildsListLimit  int
	buildsLogsFollow bool
)

func init() {
	buildsCmd.AddCommand(buildsListCmd)
	buildsCmd.AddCommand(buildsLogsCmd)
	buildsCmd.AddCommand(buildsCancelCmd)

	buildsListCmd.Flags().IntVarP(&buildsListLimit, "limit", "n", 10, "number of builds to show")
	buildsLogsCmd.Flags().BoolVarP(&buildsLogsFollow, "follow", "f", false, "follow logs (stream)")
}

func runBuildsList(cmd *cobra.Command, args []string) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	url := fmt.Sprintf("%s/v1/builds?limit=%d", strings.TrimRight(cfg.BuilderURL, "/"), buildsListLimit)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to builder: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to list builds (%d): %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	var result struct {
		Builds []buildInfo `json:"builds"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Builds) == 0 {
		fmt.Println("No builds found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "BUILD ID\tDEPLOYMENT\tSTATUS\tCREATED")
	for _, b := range result.Builds {
		created := b.CreatedAt.Format("2006-01-02 15:04")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", b.ID, b.Deployment, b.Status, created)
	}
	w.Flush()

	return nil
}

type buildInfo struct {
	ID         string    `json:"id"`
	Deployment string    `json:"deployment"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

func runBuildsLogs(cmd *cobra.Command, args []string) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	buildID := args[0]
	follow := ""
	if buildsLogsFollow {
		follow = "?follow=true"
	}

	url := fmt.Sprintf("%s/v1/builds/%s/logs%s", strings.TrimRight(cfg.BuilderURL, "/"), buildID, follow)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	if buildsLogsFollow {
		req.Header.Set("Accept", "text/event-stream")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to builder: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("build not found: %s", buildID)
	}
	if resp.StatusCode != 200 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to get logs (%d): %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return readSSELogsBuilds(resp.Body)
	}

	_, err = io.Copy(os.Stdout, resp.Body)
	return err
}

func readSSELogsBuilds(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return nil
			}
			fmt.Println(data)
		}
	}
	return scanner.Err()
}

func runBuildsCancel(cmd *cobra.Command, args []string) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	buildID := args[0]
	url := fmt.Sprintf("%s/v1/builds/%s/cancel", strings.TrimRight(cfg.BuilderURL, "/"), buildID)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to builder: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("build not found: %s", buildID)
	}
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to cancel build (%d): %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	fmt.Printf("Build %s cancelled.\n", buildID)
	return nil
}
