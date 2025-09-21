package services

import (
	"testing"
)

func TestServiceCreation(t *testing.T) {
	// Test that all services can be created without errors
	userService := NewUserService()
	if userService == nil {
		t.Fatal("Failed to create user service")
	}

	cardService := NewCardService()
	if cardService == nil {
		t.Fatal("Failed to create card service")
	}

	transactionService := NewTransactionService()
	if transactionService == nil {
		t.Fatal("Failed to create transaction service")
	}

	paymentService := NewPaymentService()
	if paymentService == nil {
		t.Fatal("Failed to create payment service")
	}

	// Test that mock data is populated
	if userService.GetUserCount() == 0 {
		t.Error("User service should have mock data")
	}

	if cardService.GetCardCount() == 0 {
		t.Error("Card service should have mock data")
	}

	if transactionService.GetTransactionCount() == 0 {
		t.Error("Transaction service should have mock data")
	}

	if paymentService.GetPaymentCount() == 0 {
		t.Error("Payment service should have mock data")
	}
}

func TestServiceRegistry(t *testing.T) {
	// Test that the service registry can be created
	registry := NewServiceRegistry()
	if registry == nil {
		t.Fatal("Failed to create service registry")
	}

	// Test that all services are present
	if registry.UserService == nil {
		t.Error("User service not initialized in registry")
	}
	if registry.CardService == nil {
		t.Error("Card service not initialized in registry")
	}
	if registry.TransactionService == nil {
		t.Error("Transaction service not initialized in registry")
	}
	if registry.PaymentService == nil {
		t.Error("Payment service not initialized in registry")
	}

	// Test service info
	info := registry.GetServiceInfo()
	if len(info) != 4 {
		t.Errorf("Expected 4 services in info, got %d", len(info))
	}

	// Test health check
	health := registry.IsHealthy()
	for service, healthy := range health {
		if !healthy {
			t.Errorf("Service %s is not healthy", service)
		}
	}
}