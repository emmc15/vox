package models

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Model represents a Vosk model
type Model struct {
	Name        string
	Language    string
	Size        string
	URL         string
	Description string
}

// Available models from Vosk
var AvailableModels = []Model{
	{
		Name:        "vosk-model-small-en-us-0.15",
		Language:    "en-US",
		Size:        "40M",
		URL:         "https://alphacephei.com/vosk/models/vosk-model-small-en-us-0.15.zip",
		Description: "Lightweight English model, fast but less accurate",
	},
	{
		Name:        "vosk-model-en-us-0.22",
		Language:    "en-US",
		Size:        "1.8G",
		URL:         "https://alphacephei.com/vosk/models/vosk-model-en-us-0.22.zip",
		Description: "Large English model, slower but more accurate",
	},
	{
		Name:        "vosk-model-en-us-0.22-lgraph",
		Language:    "en-US",
		Size:        "128M",
		URL:         "https://alphacephei.com/vosk/models/vosk-model-en-us-0.22-lgraph.zip",
		Description: "Medium English model, balanced speed and accuracy",
	},
}

// DefaultModelName is the default model to use
const DefaultModelName = "vosk-model-small-en-us-0.15"

// GetModelsDir returns the directory where models are stored
func GetModelsDir() (string, error) {
	// Use ./models directory in the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return filepath.Join(cwd, "models"), nil
}

// GetDefaultModel returns the configured default model name
// If no custom default is set, returns DefaultModelName
func GetDefaultModel() (string, error) {
	modelsDir, err := GetModelsDir()
	if err != nil {
		return DefaultModelName, err
	}

	configFile := filepath.Join(modelsDir, ".default_model")
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultModelName, nil
		}
		return DefaultModelName, err
	}

	modelName := strings.TrimSpace(string(data))
	if modelName == "" {
		return DefaultModelName, nil
	}

	return modelName, nil
}

// SetDefaultModel sets the default model to use
func SetDefaultModel(modelName string) error {
	// Verify model exists in available models
	model := FindModel(modelName)
	if model == nil {
		return fmt.Errorf("unknown model: %s", modelName)
	}

	modelsDir, err := GetModelsDir()
	if err != nil {
		return err
	}

	// Create models directory if it doesn't exist
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return fmt.Errorf("failed to create models directory: %w", err)
	}

	configFile := filepath.Join(modelsDir, ".default_model")
	err = os.WriteFile(configFile, []byte(modelName), 0644)
	if err != nil {
		return fmt.Errorf("failed to save default model: %w", err)
	}

	return nil
}

// IsModelDownloaded checks if a model is already downloaded
func IsModelDownloaded(modelName string) (bool, error) {
	modelsDir, err := GetModelsDir()
	if err != nil {
		return false, err
	}

	modelPath := filepath.Join(modelsDir, modelName)
	info, err := os.Stat(modelPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return info.IsDir(), nil
}

// GetModelPath returns the path to a model directory
func GetModelPath(modelName string) (string, error) {
	modelsDir, err := GetModelsDir()
	if err != nil {
		return "", err
	}

	modelPath := filepath.Join(modelsDir, modelName)

	// Check if it exists
	downloaded, err := IsModelDownloaded(modelName)
	if err != nil {
		return "", err
	}
	if !downloaded {
		return "", fmt.Errorf("model not found: %s", modelName)
	}

	return modelPath, nil
}

// FindModel finds a model by name in the available models list
func FindModel(name string) *Model {
	for _, model := range AvailableModels {
		if model.Name == name {
			return &model
		}
	}
	return nil
}

// DownloadModel downloads a model from the Vosk website
func DownloadModel(modelName string, progress func(downloaded, total int64)) error {
	model := FindModel(modelName)
	if model == nil {
		return fmt.Errorf("unknown model: %s", modelName)
	}

	modelsDir, err := GetModelsDir()
	if err != nil {
		return err
	}

	// Create models directory if it doesn't exist
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return fmt.Errorf("failed to create models directory: %w", err)
	}

	// Download to temporary file
	zipPath := filepath.Join(modelsDir, modelName+".zip")
	defer os.Remove(zipPath) // Clean up zip file after extraction

	fmt.Printf("Downloading %s (%s)...\n", modelName, model.Size)

	// Download the file
	resp, err := http.Get(model.URL)
	if err != nil {
		return fmt.Errorf("failed to download model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Create the file
	out, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Copy with progress tracking
	total := resp.ContentLength
	var downloaded int64

	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("failed to write file: %w", writeErr)
			}
			downloaded += int64(n)
			if progress != nil {
				progress(downloaded, total)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("download error: %w", err)
		}
	}

	fmt.Println("\nExtracting model...")

	// Extract the zip file
	if err := extractZip(zipPath, modelsDir); err != nil {
		return fmt.Errorf("failed to extract model: %w", err)
	}

	fmt.Println("Model downloaded successfully!")
	return nil
}

// extractZip extracts a zip file to the specified directory
func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		// Check for ZipSlip vulnerability
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// ListDownloadedModels lists all downloaded models
func ListDownloadedModels() ([]string, error) {
	modelsDir, err := GetModelsDir()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read models directory: %w", err)
	}

	var models []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "vosk-model-") {
			models = append(models, entry.Name())
		}
	}

	return models, nil
}
