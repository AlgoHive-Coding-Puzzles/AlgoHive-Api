package services

import (
	"api/database"
	"api/models"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func GetCatalogFromID(catalogID string) (*models.Catalog, error) {
	var catalog models.Catalog
	if err := database.DB.First(&catalog, "id = ?", catalogID).Error; err != nil {
		return nil, err
	}
	return &catalog, nil
}

func GetAddressFromCatalogId(catalogID string) (string, error) {
	var catalog models.Catalog
	if err := database.DB.First(&catalog, "id = ?", catalogID).Error; err != nil {
		return "", err
	}
	return catalog.Address, nil
}

// CatalogProxyGet fetches data from the given URL and decodes it into the provided target.
func CatalogProxyGet[T any](url string, target *T) error {
    resp, err := http.Get(url)
    if err != nil {
        return fmt.Errorf("failed to fetch data from catalog proxy: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("failed to fetch data from catalog proxy: %s", resp.Status)
    }

    if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
        return fmt.Errorf("failed to decode JSON response: %v", err)
    }

    return nil
}

func GetPuzzleInput(catalogID string, themeName string, puzzleID string, seedID string, ctx context.Context) (map[string]interface{}, error) {
	cacheKey := "catalog_puzzle_input:" + catalogID + ":" + themeName + ":" + puzzleID + ":" + seedID

	// Step 1: Try to get puzzle input from cache
	var inputDetails map[string]interface{}
    if found, _ := database.GetFromCache(ctx, cacheKey, &inputDetails); found {
        return inputDetails, nil
    }

	// Step 2: Cache miss - fetch from database and API
	address, err := GetAddressFromCatalogId(catalogID)
	if err != nil {
		return nil, fmt.Errorf("catalog not found: %w", err)
	}

	apiURL := fmt.Sprintf("%s/puzzle/generate/input?theme=%s&puzzle=%s&unique_id=%s", address, themeName, puzzleID, seedID)
	if err := CatalogProxyGet(apiURL, &inputDetails); err != nil {
		return nil, fmt.Errorf("error while reaching the API: %w", err)
	}

	// Step 3: Cache the puzzle input details
	if err := database.SetToCache(ctx, cacheKey, inputDetails); err != nil {
		return nil, fmt.Errorf("error while caching the puzzle input: %w", err)
	}

	return inputDetails, nil
}

func CheckPuzzleAnswer(catalogID string, themeName string, puzzleID string, step int, seedID string, answer string) (bool, error) {
	// Step 1: Validate the step to redirect to the correct URL
	stepURL := ""
	switch step {
	case 1:
		stepURL = "/first"
	case 2:
		stepURL = "/second"
	default:
		return false, fmt.Errorf("invalid step")
		
	}

	// Step 2: Fetch the catalog address
	address, err := GetAddressFromCatalogId(catalogID)
	if err != nil {
		return false, fmt.Errorf("catalog not found: %w", err)
	}

	// Step 3: Build the request URL for checking the solution
    apiURL := fmt.Sprintf("%s/puzzle/check%s?theme=%s&puzzle=%s&unique_id=%s&solution=%s",
		address,
        stepURL,
        themeName,
        puzzleID,
        seedID,
        answer,
    )

	// Step 4: Check the solution via the external service
	var result map[string]interface{}
	err = CatalogProxyGet(apiURL, &result)
	if err != nil || result == nil || result["matches"] == nil {
		return false, fmt.Errorf("failed to check solution")
	}

	isCorrect := result["matches"].(bool)

	return isCorrect, nil
}