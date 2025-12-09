package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/emmett/diaz/internal/models"
)

type ModelManager struct{}

func NewModelManager() *ModelManager {
	return &ModelManager{}
}

func (m *ModelManager) ListModels() error {
	fmt.Println("Available models for download:")
	fmt.Println()

	for i, model := range models.AvailableModels {
		fmt.Printf("%d. %s\n", i+1, model.Name)
		fmt.Printf("   Language: %s\n", model.Language)
		fmt.Printf("   Size:     %s\n", model.Size)
		fmt.Printf("   Info:     %s\n", model.Description)

		downloaded, _ := models.IsModelDownloaded(model.Name)
		if downloaded {
			fmt.Printf("   Status:   ✓ Downloaded\n")
		} else {
			fmt.Printf("   Status:   Not downloaded\n")
		}
		fmt.Println()
	}

	fmt.Println("To download a model, use:")
	fmt.Println("  diaz --download-model <model-name>")
	return nil
}

func (m *ModelManager) ListDownloaded() error {
	downloaded, err := models.ListDownloadedModels()
	if err != nil {
		return fmt.Errorf("error listing models: %w", err)
	}

	if len(downloaded) == 0 {
		fmt.Println("No models downloaded yet.")
		fmt.Println()
		fmt.Println("Use 'diaz --list-models' to see available models")
		fmt.Println("Use 'diaz --download-model <name>' to download a model")
		return nil
	}

	fmt.Printf("Downloaded models (%d):\n", len(downloaded))
	fmt.Println()

	for i, modelName := range downloaded {
		fmt.Printf("%d. %s", i+1, modelName)
		if modelName == models.DefaultModelName {
			fmt.Printf(" [DEFAULT]")
		}
		fmt.Println()

		modelPath, err := models.GetModelPath(modelName)
		if err == nil {
			fmt.Printf("   Path: %s\n", modelPath)
		}
	}
	fmt.Println()
	fmt.Println("To use a model, run:")
	fmt.Println("  diaz --model <model-name>")
	return nil
}

func (m *ModelManager) Download(name string) error {
	model := models.FindModel(name)
	if model == nil {
		fmt.Fprintf(os.Stderr, "Error: Unknown model '%s'\n", name)
		fmt.Println()
		fmt.Println("Use 'diaz --list-models' to see available models")
		return fmt.Errorf("unknown model: %s", name)
	}

	downloaded, err := models.IsModelDownloaded(name)
	if err != nil {
		return fmt.Errorf("error checking model: %w", err)
	}

	if downloaded {
		fmt.Printf("Model '%s' is already downloaded.\n", name)
		modelPath, _ := models.GetModelPath(name)
		fmt.Printf("Location: %s\n", modelPath)
		return nil
	}

	fmt.Printf("Downloading model: %s (%s)\n", model.Name, model.Size)
	fmt.Printf("Description: %s\n", model.Description)
	fmt.Println()

	err = models.DownloadModel(name, func(downloaded, total int64) {
		percent := float64(downloaded) / float64(total) * 100
		fmt.Printf("\rProgress: %.1f%% (%d/%d bytes)", percent, downloaded, total)
	})

	if err != nil {
		return fmt.Errorf("error downloading model: %w", err)
	}

	fmt.Println()
	fmt.Printf("✓ Model '%s' downloaded successfully!\n", name)
	return nil
}

func (m *ModelManager) SetDefault(name string) error {
	model := models.FindModel(name)
	if model == nil {
		fmt.Fprintf(os.Stderr, "Error: Unknown model '%s'\n", name)
		fmt.Println()
		fmt.Println("Use 'diaz --list-models' to see available models")
		return fmt.Errorf("unknown model: %s", name)
	}

	err := models.SetDefaultModel(name)
	if err != nil {
		return fmt.Errorf("error setting default model: %w", err)
	}

	fmt.Printf("✓ Default model set to: %s\n", name)
	fmt.Printf("  Description: %s\n", model.Description)
	fmt.Printf("  Size: %s\n", model.Size)
	fmt.Println()

	downloaded, _ := models.IsModelDownloaded(name)
	if !downloaded {
		fmt.Println("Note: This model is not yet downloaded.")
		fmt.Printf("Run 'diaz --download-model %s' to download it.\n", name)
	}
	return nil
}

func (m *ModelManager) SelectInteractive() (string, error) {
	fmt.Println("Select a model to use:")
	fmt.Println()

	downloadedModels, err := models.ListDownloadedModels()
	if err != nil {
		return "", err
	}

	downloadedMap := make(map[string]bool)
	for _, m := range downloadedModels {
		downloadedMap[m] = true
	}

	for i, model := range models.AvailableModels {
		status := "Not downloaded"
		if downloadedMap[model.Name] {
			status = "✓ Downloaded"
		}

		fmt.Printf("%d. %s (%s)\n", i+1, model.Name, model.Size)
		fmt.Printf("   %s\n", model.Description)
		fmt.Printf("   Status: %s\n", status)
		fmt.Println()
	}

	fmt.Print("Enter number (1-3): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSpace(input)
	var choice int
	_, err = fmt.Sscanf(input, "%d", &choice)
	if err != nil || choice < 1 || choice > len(models.AvailableModels) {
		return "", fmt.Errorf("invalid selection")
	}

	selected := models.AvailableModels[choice-1].Name
	fmt.Printf("\nSelected: %s\n", selected)

	if !downloadedMap[selected] {
		fmt.Println("This model is not downloaded.")
		fmt.Print("Download now? (y/n): ")
		response, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response == "y" || response == "yes" {
			err = models.DownloadModel(selected, func(downloaded, total int64) {
				percent := float64(downloaded) / float64(total) * 100
				fmt.Printf("\rProgress: %.1f%% (%d/%d bytes)", percent, downloaded, total)
			})
			if err != nil {
				return "", fmt.Errorf("failed to download model: %w", err)
			}
			fmt.Println()
		} else {
			return "", fmt.Errorf("cannot proceed without model")
		}
	}

	return selected, nil
}

func (m *ModelManager) EnsureModel(name string, autoDownload bool) (string, error) {
	downloaded, err := models.IsModelDownloaded(name)
	if err != nil {
		return "", fmt.Errorf("failed to check for model: %w", err)
	}

	if downloaded {
		return name, nil
	}

	if autoDownload {
		fmt.Printf("Model '%s' not found. Downloading automatically...\n", name)
		err = models.DownloadModel(name, func(downloaded, total int64) {
			percent := float64(downloaded) / float64(total) * 100
			fmt.Printf("\rProgress: %.1f%% (%d/%d bytes)", percent, downloaded, total)
		})
		if err != nil {
			return "", fmt.Errorf("failed to download model: %w", err)
		}
		fmt.Println()
		return name, nil
	}

	// Prompt user
	fmt.Printf("Model '%s' not found.\n", name)
	fmt.Println()
	fmt.Println("Available models:")
	for i, model := range models.AvailableModels {
		marker := ""
		if model.Name == name {
			marker = " (selected)"
		}
		fmt.Printf("  %d. %s (%s) - %s%s\n", i+1, model.Name, model.Size, model.Description, marker)
	}
	fmt.Println()
	fmt.Printf("Download '%s'? (y/n): ", name)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println()
		fmt.Println("You can download models using:")
		fmt.Println("  diaz --list-models          # List available models")
		fmt.Println("  diaz --download-model <name> # Download a specific model")
		return "", fmt.Errorf("model download declined")
	}

	err = models.DownloadModel(name, func(downloaded, total int64) {
		percent := float64(downloaded) / float64(total) * 100
		fmt.Printf("\rProgress: %.1f%% (%d/%d bytes)", percent, downloaded, total)
	})
	if err != nil {
		return "", fmt.Errorf("failed to download model: %w", err)
	}
	fmt.Println()

	return name, nil
}

func (m *ModelManager) SelectModel(modelName string, selectInteractive bool) (string, error) {
	if modelName != "" {
		return modelName, nil
	}

	if selectInteractive {
		return m.SelectInteractive()
	}

	return models.GetDefaultModel()
}
