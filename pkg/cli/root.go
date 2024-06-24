// pkg/cli/root.go
package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"minislot/pkg/tiers"

	"github.com/spf13/cobra"
)

type DeploymentConfig struct {
	ID           string `json:"id"`
	Namespace    string `json:"namespace"`
	Version      string `json:"version"`
	Seed         int    `json:"seed"`
	ChainID      int    `json:"chainId"`
	BlockTime    int    `json:"blockTime"`
	Tier         string `json:"tier"`
	StorageClass string `json:"storageClass"`
}

var (
	serverURL string
	config    DeploymentConfig
)

var rootCmd = &cobra.Command{
	Use:   "katana-cli",
	Short: "CLI for Katana Deployment Server",
	Long:  `A CLI tool to interact with the Katana Deployment Server and create new deployments.`,
	RunE:  runDeploy,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Deployment server URL")
	rootCmd.Flags().StringVar(&config.ID, "id", "", "Deployment ID")
	rootCmd.Flags().StringVar(&config.Namespace, "namespace", "default", "Kubernetes namespace")
	rootCmd.Flags().StringVar(&config.Version, "version", "latest", "Katana version")
	rootCmd.Flags().IntVar(&config.Seed, "seed", 0, "Seed value")
	rootCmd.Flags().IntVar(&config.ChainID, "chain-id", 1, "Chain ID")
	rootCmd.Flags().IntVar(&config.BlockTime, "block-time", 0, "Block time")
	rootCmd.Flags().StringVar(&config.Tier, "tier", "free", "Resource tier (free, professional, enterprise)")
	rootCmd.Flags().StringVar(&config.StorageClass, "storage-class", "standard", "Storage class")

	rootCmd.MarkFlagRequired("id")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	// Validate tier
	if _, exists := tiers.GetTier(config.Tier); !exists {
		return fmt.Errorf("invalid tier: %s", config.Tier)
	}

	jsonData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshalling config: %w", err)
	}

	resp, err := http.Post(serverURL+"/deploy", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	fmt.Println("Deployment created successfully")
	return nil
}