package deployclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type CreateDeploymentRequest struct {
	App            string   `json:"app"`
	MetricsBuckets []string `json:"metricsBuckets,omitempty"`
	Provider       string   `json:"provider"`
	ServiceToken   string   `json:"serviceToken"`
	CLIVersion     string   `json:"cliVersion"`
	CLICommit      string   `json:"cliCommit"`
	Worker         struct {
		Kind string `json:"kind"`
		Code string `json:"code"`
	} `json:"worker"`
}

type DeploymentRecord struct {
	ID             string   `json:"id"`
	App            string   `json:"app"`
	MetricsBuckets []string `json:"metricsBuckets"`
	Provider       string   `json:"provider"`
	ScriptName     string   `json:"scriptName"`
	Hostname       string   `json:"hostname"`
	URL            string   `json:"url"`
	Status         string   `json:"status"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
}

type CreateDeploymentResponse struct {
	OK         bool             `json:"ok"`
	Deployment DeploymentRecord `json:"deployment"`
}

type ListDeploymentsResponse struct {
	OK          bool               `json:"ok"`
	Deployments []DeploymentRecord `json:"deployments"`
}

func New(baseURL string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) CreateDeployment(accessToken string, request CreateDeploymentRequest) (CreateDeploymentResponse, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return CreateDeploymentResponse{}, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/deployments/v1", bytes.NewReader(payload))
	if err != nil {
		return CreateDeploymentResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return CreateDeploymentResponse{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return CreateDeploymentResponse{}, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		message := strings.TrimSpace(string(body))
		var errPayload struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &errPayload) == nil {
			if errPayload.Message != "" {
				message = errPayload.Message
			} else if errPayload.Error != "" {
				message = errPayload.Error
			}
		}
		if message == "" {
			message = res.Status
		}
		return CreateDeploymentResponse{}, fmt.Errorf("deploy request failed (%s): %s", res.Status, message)
	}
	var response CreateDeploymentResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return CreateDeploymentResponse{}, err
	}
	return response, nil
}

func (c *Client) ListDeployments(accessToken string) ([]DeploymentRecord, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/deployments/v1", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	var response ListDeploymentsResponse
	if err := c.doJSON(req, &response, "deployments request failed"); err != nil {
		return nil, err
	}
	return response.Deployments, nil
}

func (c *Client) DeleteDeployment(accessToken string, deploymentID string) error {
	deploymentID = strings.TrimSpace(deploymentID)
	if deploymentID == "" {
		return fmt.Errorf("deployment id is required")
	}

	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/deployments/v1/"+deploymentID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	return c.doJSON(req, nil, "delete deployment failed")
}

func (c *Client) doJSON(req *http.Request, out any, errorPrefix string) error {
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		message := strings.TrimSpace(string(body))
		var errPayload struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &errPayload) == nil {
			if errPayload.Message != "" {
				message = errPayload.Message
			} else if errPayload.Error != "" {
				message = errPayload.Error
			}
		}
		if message == "" {
			message = res.Status
		}
		return fmt.Errorf("%s (%s): %s", errorPrefix, res.Status, message)
	}
	if out == nil || len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return err
	}
	return nil
}
