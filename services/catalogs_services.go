package services

import (
	"api/database"
	"api/models"
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

func CatalogProxyGet(url string) (map[string]interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch data from catalog proxy: %s", resp.Status)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	return data, nil
}