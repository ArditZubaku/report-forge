package reports

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type LozClient struct {
	httpClient HttpClient
	baseURL    string
}

func NewClient(baseURL string, httpClient HttpClient) *LozClient {
	return &LozClient{
		httpClient: httpClient,
		baseURL:    "https://botw-compendium.herokuapp.com/api/v3/compendium",
	}
}

type Monster struct {
	Name            string   `json:"name"`
	Id              int      `json:"id"`
	Category        string   `json:"category"`
	Description     string   `json:"description"`
	Image           string   `json:"image"`
	CommonLocations []string `json:"common_locations"`
	Drops           []string `json:"drops"`
	Dlc             bool     `json:"dlc"`
}

type GetMonstersResponse struct {
	Data []Monster `json:"data"`
}

func (c *LozClient) GetMonsters(ctx context.Context) ([]Monster, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/category/monsters", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create /monsters request: %w", err)
	}

	queryParams := req.URL.Query()
	queryParams.Set("game", "totk")
	req.URL.RawQuery = queryParams.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get monsters: %w", err)
	}

	var response GetMonstersResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal monsters http response: %w", err)
	}

	return response.Data, nil
}
