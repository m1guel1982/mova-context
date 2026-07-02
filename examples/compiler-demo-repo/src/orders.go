package orders

import "fmt"

// CreateOrder creates a new order for a client.
func CreateOrder(clientID string, amount float64) (string, error) {
	if amount <= 0 {
		return "", fmt.Errorf("invalid amount")
	}
	id := generateID()
	fmt.Println("order created", id, clientID)
	return id, nil
}

// CancelOrder cancels an existing order.
func CancelOrder(orderID string) error {
	if orderID == "" {
		return fmt.Errorf("missing order id")
	}
	return nil
}

func generateID() string {
	return "ord_123"
}
