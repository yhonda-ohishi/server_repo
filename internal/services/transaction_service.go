package services

import (
	"fmt"
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "github.com/yhonda-ohishi/db-handler-server/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TransactionService implements the TransactionServiceServer interface
type TransactionService struct {
	pb.UnimplementedTransactionServiceServer
	mu           sync.RWMutex
	transactions map[string]*pb.Transaction
}

// NewTransactionService creates a new TransactionService instance with mock data
func NewTransactionService() *TransactionService {
	service := &TransactionService{
		transactions: make(map[string]*pb.Transaction),
	}

	// Add mock data
	service.addMockData()
	// Add specific test transaction with known ID
	service.addTestTransaction()
	return service
}

// addMockData populates the service with mock transactions for testing
func (s *TransactionService) addMockData() {
	now := time.Now()
	mockCardIds := []string{
		"card-1",
		"card-2",
		"card-3",
	}

	gateNames := []string{
		"Tokyo IC",
		"Shibuya IC",
		"Shinjuku IC",
		"Harajuku IC",
		"Osaka IC",
		"Kyoto IC",
		"Nagoya IC",
	}

	// Generate 20 mock transactions
	for i := 0; i < 20; i++ {
		cardId := mockCardIds[rand.Intn(len(mockCardIds))]
		entryGate := gateNames[rand.Intn(len(gateNames))]
		exitGate := gateNames[rand.Intn(len(gateNames))]

		// Ensure entry and exit gates are different
		for exitGate == entryGate {
			exitGate = gateNames[rand.Intn(len(gateNames))]
		}

		entryTime := now.Add(-time.Duration(rand.Intn(720)) * time.Hour) // Random time within last 30 days
		exitTime := entryTime.Add(time.Duration(30+rand.Intn(180)) * time.Minute) // 30 min to 3.5 hours later

		distance := float64(10 + rand.Intn(200)) // 10-210 km
		tollAmount := int64(300 + rand.Intn(2000)) // 300-2300 yen base toll
		discountAmount := int64(rand.Intn(int(tollAmount) / 4)) // 0-25% discount
		finalAmount := tollAmount - discountAmount

		paymentStatus := pb.PaymentStatus_PAYMENT_STATUS_COMPLETED
		if rand.Float32() < 0.1 { // 10% chance of pending
			paymentStatus = pb.PaymentStatus_PAYMENT_STATUS_PENDING
		} else if rand.Float32() < 0.05 { // 5% chance of failed
			paymentStatus = pb.PaymentStatus_PAYMENT_STATUS_FAILED
		}

		transaction := &pb.Transaction{
			Id:              uuid.New().String(),
			CardId:          cardId,
			EntryGateId:     entryGate,
			ExitGateId:      exitGate,
			EntryTime:       timestamppb.New(entryTime),
			ExitTime:        timestamppb.New(exitTime),
			Distance:        distance,
			TollAmount:      tollAmount,
			DiscountAmount:  discountAmount,
			FinalAmount:     finalAmount,
			PaymentStatus:   paymentStatus,
			TransactionDate: timestamppb.New(exitTime),
		}

		s.transactions[transaction.Id] = transaction
	}
}

// addTestTransaction adds specific test transactions with known IDs for testing
func (s *TransactionService) addTestTransaction() {
	now := time.Now()

	// Add specific test transaction with known ID "txn-1"
	testTransaction := &pb.Transaction{
		Id:              "txn-1",
		CardId:          "card-1",
		EntryGateId:     "gate-001",
		ExitGateId:      "gate-002",
		EntryTime:       timestamppb.New(now.Add(-2 * time.Hour)),
		ExitTime:        timestamppb.New(now.Add(-1 * time.Hour)),
		Distance:        45.5,
		TollAmount:      1200,
		DiscountAmount:  100,
		FinalAmount:     1100,
		PaymentStatus:   pb.PaymentStatus_PAYMENT_STATUS_COMPLETED,
		TransactionDate: timestamppb.New(now.Add(-1 * time.Hour)),
	}

	s.transactions[testTransaction.Id] = testTransaction
}

// GetTransaction retrieves a single transaction by ID
func (s *TransactionService) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.Transaction, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "transaction ID is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	transaction, exists := s.transactions[req.Id]
	if !exists {
		return nil, status.Error(codes.NotFound, "transaction not found")
	}

	return transaction, nil
}

// GetTransactionHistory retrieves transaction history for a card
func (s *TransactionService) GetTransactionHistory(ctx context.Context, req *pb.GetTransactionHistoryRequest) (*pb.TransactionList, error) {
	if req.CardId == "" {
		return nil, status.Error(codes.InvalidArgument, "card ID is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Default pagination values
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}

	// For simplicity, ignore page token for now in mock implementation
	skip := 0
	if req.PageToken != "" {
		// In real implementation, decode page token to get skip value
		skip = 0
	}

	// Filter transactions for the specified card
	var cardTransactions []*pb.Transaction
	for _, transaction := range s.transactions {
		if transaction.CardId == req.CardId {
			// Apply date filters if specified
			if req.StartDate != nil && transaction.TransactionDate.AsTime().Before(req.StartDate.AsTime()) {
				continue
			}
			if req.EndDate != nil && transaction.TransactionDate.AsTime().After(req.EndDate.AsTime()) {
				continue
			}

			cardTransactions = append(cardTransactions, transaction)
		}
	}

	// Sort transactions by date (newest first)
	for i := 0; i < len(cardTransactions)-1; i++ {
		for j := i + 1; j < len(cardTransactions); j++ {
			if cardTransactions[i].TransactionDate.AsTime().Before(cardTransactions[j].TransactionDate.AsTime()) {
				cardTransactions[i], cardTransactions[j] = cardTransactions[j], cardTransactions[i]
			}
		}
	}

	start := skip
	end := start + int(pageSize)

	var transactions []*pb.Transaction
	var nextPageToken string

	if start < len(cardTransactions) {
		if end > len(cardTransactions) {
			end = len(cardTransactions)
		}
		transactions = cardTransactions[start:end]
		// Generate next page token if there are more transactions
		if end < len(cardTransactions) {
			nextPageToken = fmt.Sprintf("next_%d", end)
		}
	} else {
		transactions = []*pb.Transaction{}
	}

	// Calculate total amount
	var totalAmount int64
	for _, tx := range cardTransactions {
		totalAmount += tx.FinalAmount
	}

	return &pb.TransactionList{
		Transactions:  transactions,
		NextPageToken: nextPageToken,
		TotalAmount:   totalAmount,
	}, nil
}

// CreateTransaction creates a new transaction (helper method for testing)
func (s *TransactionService) CreateTransaction(cardId, entryGateId, exitGateId string, entryTime, exitTime time.Time, distance float64, tollAmount int64) (*pb.Transaction, error) {
	if cardId == "" {
		return nil, status.Error(codes.InvalidArgument, "card ID is required")
	}
	if entryGateId == "" {
		return nil, status.Error(codes.InvalidArgument, "entry gate ID is required")
	}
	if exitGateId == "" {
		return nil, status.Error(codes.InvalidArgument, "exit gate ID is required")
	}
	if tollAmount < 0 {
		return nil, status.Error(codes.InvalidArgument, "toll amount must be non-negative")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Calculate discount (simple random discount for testing)
	discountAmount := int64(rand.Intn(int(tollAmount) / 4))
	finalAmount := tollAmount - discountAmount

	transaction := &pb.Transaction{
		Id:              uuid.New().String(),
		CardId:          cardId,
		EntryGateId:     entryGateId,
		ExitGateId:      exitGateId,
		EntryTime:       timestamppb.New(entryTime),
		ExitTime:        timestamppb.New(exitTime),
		Distance:        distance,
		TollAmount:      tollAmount,
		DiscountAmount:  discountAmount,
		FinalAmount:     finalAmount,
		PaymentStatus:   pb.PaymentStatus_PAYMENT_STATUS_COMPLETED,
		TransactionDate: timestamppb.New(exitTime),
	}

	s.transactions[transaction.Id] = transaction
	return transaction, nil
}

// GetTransactionCount returns the current number of transactions (helper method for testing)
func (s *TransactionService) GetTransactionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.transactions)
}

// GetTransactionsByCardId returns all transactions for a card (helper method for testing)
func (s *TransactionService) GetTransactionsByCardId(cardId string) []*pb.Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var transactions []*pb.Transaction
	for _, transaction := range s.transactions {
		if transaction.CardId == cardId {
			transactions = append(transactions, transaction)
		}
	}
	return transactions
}

// UpdateTransactionPaymentStatus updates the payment status of a transaction (helper method)
func (s *TransactionService) UpdateTransactionPaymentStatus(transactionId string, paymentStatus pb.PaymentStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	transaction, exists := s.transactions[transactionId]
	if !exists {
		return status.Error(codes.NotFound, "transaction not found")
	}

	transaction.PaymentStatus = paymentStatus
	return nil
}