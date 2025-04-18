package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maritime_traffic/pkg/handlers"
	"maritime_traffic/pkg/traffic"
	"net/http"
)

type Client struct {
	Address string
}

func NewClient(address string, port int) *Client {
	return &Client{
		Address: fmt.Sprintf("%s:%d", address, port),
	}
}

func (c *Client) Flush() error {
	resp, err := http.Post(fmt.Sprintf("%s/api/v1/flush", c.Address), "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to flush: %s", resp.Status)
	}

	return nil
}

func (c *Client) GetShips() ([]traffic.Ship, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/ships", c.Address))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get ships: %s", resp.Status)
	}

	var ships []traffic.Ship
	if err := json.NewDecoder(resp.Body).Decode(&ships); err != nil {
		return nil, err
	}

	return ships, nil
}

func (c *Client) GetShip(id string) (traffic.Ship, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/ships/%s", c.Address, id))
	if err != nil {
		return traffic.Ship{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return traffic.Ship{}, fmt.Errorf("failed to get ship: %s", resp.Status)
	}

	var ship traffic.Ship
	if err := json.NewDecoder(resp.Body).Decode(&ship); err != nil {
		return traffic.Ship{}, err
	}

	return ship, nil
}

func (c *Client) PositionShip(id string, time int, position handlers.Position) (traffic.PositionResult, error) {
	reqBody, err := json.Marshal(handlers.PositionShipRequest{
		Time: time,
		X:    position.X,
		Y:    position.Y,
	})
	if err != nil {
		return traffic.PositionResult{}, err
	}

	resp, err := http.Post(fmt.Sprintf("%s/api/v1/ships/%s/position", c.Address, id), "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return traffic.PositionResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return traffic.PositionResult{}, fmt.Errorf("failed to position ship: %s", resp.Status)
	}

	var result traffic.PositionResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return traffic.PositionResult{}, err
	}

	return result, nil
}
