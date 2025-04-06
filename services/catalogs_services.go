package services

import (
	"api/database"
	"api/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Define domain-specific errors for better error handling
var (
    ErrCatalogNotFound     = errors.New("catalog not found")
    ErrInvalidStep         = errors.New("invalid step")
    ErrAPIRequestFailed    = errors.New("API request failed")
    ErrDecodingResponse    = errors.New("failed to decode API response")
    ErrCachingFailed       = errors.New("failed to cache data")
    ErrSolutionCheckFailed = errors.New("failed to check solution")
)

// Client configuration with sensible defaults
var (
    httpClient = &http.Client{
        Timeout: 30 * time.Second,
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 20,
            IdleConnTimeout:     60 * time.Second,
        },
    }
)

// GetCatalogFromID retrieves a catalog by its ID
func GetCatalogFromID(catalogID string) (*models.Catalog, error) {
    if catalogID == "" {
        return nil, fmt.Errorf("%w: empty catalog ID", ErrCatalogNotFound)
    }

    var catalog models.Catalog
    if err := database.DB.First(&catalog, "id = ?", catalogID).Error; err != nil {
        return nil, fmt.Errorf("%w: %s", ErrCatalogNotFound, err)
    }
    
    return &catalog, nil
}

// GetAddressFromCatalogId retrieves the address of a catalog by its ID
func GetAddressFromCatalogId(catalogID string) (string, error) {
    if catalogID == "" {
        return "", fmt.Errorf("%w: empty catalog ID", ErrCatalogNotFound)
    }

    var catalog models.Catalog
    if err := database.DB.First(&catalog, "id = ?", catalogID).Error; err != nil {
        return "", fmt.Errorf("%w: %s", ErrCatalogNotFound, err)
    }
    
    if catalog.Address == "" {
        return "", fmt.Errorf("%w: catalog has no address", ErrCatalogNotFound)
    }
    
    return catalog.Address, nil
}

// CatalogProxyGet fetches data from the given URL and decodes it into the provided target.
func CatalogProxyGet[T any](url string, target *T) error {
    resp, err := httpClient.Get(url)
    if err != nil {
        return fmt.Errorf("%w: %v", ErrAPIRequestFailed, err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("%w: %s", ErrAPIRequestFailed, resp.Status)
    }

    if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
        return fmt.Errorf("%w: %v", ErrDecodingResponse, err)
    }

    return nil
}

// GetPuzzleInput retrieves puzzle input data, using cache when available
func GetPuzzleInput(catalogID string, themeName string, puzzleID string, seedID string, ctx context.Context) (map[string]interface{}, error) {
    // Validate inputs
    if catalogID == "" || themeName == "" || puzzleID == "" || seedID == "" {
        return nil, fmt.Errorf("missing required parameters: catalogID, themeName, puzzleID, or seedID")
    }

    cacheKey := fmt.Sprintf("catalog_puzzle_input:%s:%s:%s:%s", catalogID, themeName, puzzleID, seedID)

    // Step 1: Try to get puzzle input from cache
    var inputDetails map[string]interface{}
    if found, _ := database.GetFromCache(ctx, cacheKey, &inputDetails); found {
        return inputDetails, nil
    }

    // Step 2: Cache miss - fetch from database and API
    address, err := GetAddressFromCatalogId(catalogID)
    if err != nil {
        return nil, err // Error already wrapped in GetAddressFromCatalogId
    }

    // Build URL with proper escaping
    apiURL := fmt.Sprintf("%s/puzzle/generate/input?theme=%s&puzzle=%s&unique_id=%s", 
        address, 
        url.QueryEscape(themeName), 
        url.QueryEscape(puzzleID), 
        url.QueryEscape(seedID))

    if err := CatalogProxyGet(apiURL, &inputDetails); err != nil {
        return nil, fmt.Errorf("error while reaching the API: %w", err)
    }

    // Step 3: Cache the puzzle input details
    if err := database.SetToCache(ctx, cacheKey, inputDetails); err != nil {
        // Log error but don't fail the request - cache is just an optimization
        fmt.Printf("Warning: Failed to cache puzzle input: %v\n", err)
    }

    return inputDetails, nil
}

// CheckPuzzleAnswer validates a user's answer against the expected solution
func CheckPuzzleAnswer(catalogID string, themeName string, puzzleID string, step int, seedID string, answer string) (bool, error) {
    // Validate inputs
    if catalogID == "" || themeName == "" || puzzleID == "" || seedID == "" {
        return false, fmt.Errorf("missing required parameters")
    }

    // Step 1: Validate the step to redirect to the correct URL
    stepURL := ""
    switch step {
    case 1:
        stepURL = "/first"
    case 2:
        stepURL = "/second"
    default:
        return false, ErrInvalidStep
    }

    // Step 2: Fetch the catalog address
    address, err := GetAddressFromCatalogId(catalogID)
    if err != nil {
        return false, err // Error already wrapped in GetAddressFromCatalogId
    }

    // Step 3: Build the request URL for checking the solution with proper escaping
    apiURL := fmt.Sprintf("%s/puzzle/check%s?theme=%s&puzzle=%s&unique_id=%s&solution=%s",
        address,
        stepURL,
        url.QueryEscape(themeName),
        url.QueryEscape(puzzleID),
        url.QueryEscape(seedID),
        url.QueryEscape(answer),
    )

    // Step 4: Check the solution via the external service
    var result map[string]interface{}
    err = CatalogProxyGet(apiURL, &result)
    if err != nil {
        return false, fmt.Errorf("%w: %s", ErrSolutionCheckFailed, err)
    }
    
    if result == nil || result["matches"] == nil {
        return false, ErrSolutionCheckFailed
    }

    // Type assertion with safety check
    isCorrect, ok := result["matches"].(bool)
    if !ok {
        return false, fmt.Errorf("%w: unexpected response format", ErrSolutionCheckFailed)
    }

    return isCorrect, nil
}