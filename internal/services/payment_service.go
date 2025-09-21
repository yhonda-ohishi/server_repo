package services

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "github.com/yhonda-ohishi/db-handler-server/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PaymentService implements the PaymentServiceServer interface
type PaymentService struct {
	pb.UnimplementedPaymentServiceServer
	mu       sync.RWMutex
	payments map[string]*pb.Payment
}

// NewPaymentService creates a new PaymentService instance with mock data
func NewPaymentService() *PaymentService {
	service := &PaymentService{
		payments: make(map[string]*pb.Payment),
	}

	// Add mock data
	service.addMockData()
	return service
}

// addMockData populates the service with mock payments for testing
func (s *PaymentService) addMockData() {
	now := time.Now()
	mockUserIds := []string{
		"user-001",
		"user-002",
		"user-003",
	}

	paymentMethods := []pb.PaymentMethod{
		pb.PaymentMethod_PAYMENT_METHOD_CREDIT_CARD,
		pb.PaymentMethod_PAYMENT_METHOD_BANK_TRANSFER,
		pb.PaymentMethod_PAYMENT_METHOD_AUTO_DEBIT,
	}

	// Generate mock payments for the last 6 months
	for i := 0; i < 30; i++ {
		userId := mockUserIds[rand.Intn(len(mockUserIds))]
		paymentMethod := paymentMethods[rand.Intn(len(paymentMethods))]

		// Random payment date within last 6 months
		paymentDate := now.Add(-time.Duration(rand.Intn(180)) * 24 * time.Hour)

		totalAmount := int64(1000 + rand.Intn(50000)) // 1,000-51,000 yen

		status := pb.PaymentProcessingStatus_PAYMENT_PROCESSING_STATUS_COMPLETED
		if rand.Float32() < 0.1 { // 10% chance of pending
			status = pb.PaymentProcessingStatus_PAYMENT_PROCESSING_STATUS_PENDING
		} else if rand.Float32() < 0.05 { // 5% chance of processing
			status = pb.PaymentProcessingStatus_PAYMENT_PROCESSING_STATUS_PROCESSING
		} else if rand.Float32() < 0.02 { // 2% chance of failed
			status = pb.PaymentProcessingStatus_PAYMENT_PROCESSING_STATUS_FAILED
		}

		// Generate mock transaction IDs
		transactionIds := []string{
			fmt.Sprintf("tx_%d_1", i),
			fmt.Sprintf("tx_%d_2", i),
		}

		payment := &pb.Payment{
			Id:              uuid.New().String(),
			UserId:          userId,
			TransactionIds:  transactionIds,
			TotalAmount:     totalAmount,
			PaymentMethod:   paymentMethod,
			PaymentDate:     timestamppb.New(paymentDate),
			Status:          status,
			ReferenceNumber: fmt.Sprintf("ref_%d_%s", time.Now().Unix(), uuid.New().String()[:8]),
		}

		s.payments[payment.Id] = payment
	}
}

// GetPayment retrieves a payment by ID
func (s *PaymentService) GetPayment(ctx context.Context, req *pb.GetPaymentRequest) (*pb.Payment, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "payment ID is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	payment, exists := s.payments[req.Id]
	if !exists {
		return nil, status.Error(codes.NotFound, "payment not found")
	}

	return payment, nil
}

// CreatePayment creates a new payment
func (s *PaymentService) CreatePayment(ctx context.Context, req *pb.CreatePaymentRequest) (*pb.Payment, error) {
	// Validate required fields
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}
	if req.TotalAmount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "total amount must be positive")
	}
	if req.PaymentMethod == pb.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "payment method is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Create new payment
	now := timestamppb.New(time.Now())
	payment := &pb.Payment{
		Id:              uuid.New().String(),
		UserId:          req.UserId,
		TransactionIds:  req.TransactionIds,
		TotalAmount:     req.TotalAmount,
		PaymentMethod:   req.PaymentMethod,
		PaymentDate:     now,
		Status:          pb.PaymentProcessingStatus_PAYMENT_PROCESSING_STATUS_PENDING,
		ReferenceNumber: fmt.Sprintf("ref_%d_%s", time.Now().Unix(), uuid.New().String()[:8]),
	}

	s.payments[payment.Id] = payment

	// Simulate payment processing (in real implementation, this would be async)
	go s.simulatePaymentProcessing(payment.Id)

	return payment, nil
}

// ListPayments lists payments for a user
func (s *PaymentService) ListPayments(ctx context.Context, req *pb.ListPaymentsRequest) (*pb.ListPaymentsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
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

	// Filter payments for the specified user
	var userPayments []*pb.Payment
	for _, payment := range s.payments {
		if payment.UserId == req.UserId {
			// Apply processing status filter if specified
			if req.Status != pb.PaymentProcessingStatus_PAYMENT_PROCESSING_STATUS_UNSPECIFIED &&
			   payment.Status != req.Status {
				continue
			}

			userPayments = append(userPayments, payment)
		}
	}

	// Sort payments by date (newest first)
	for i := 0; i < len(userPayments)-1; i++ {
		for j := i + 1; j < len(userPayments); j++ {
			if userPayments[i].PaymentDate.AsTime().Before(userPayments[j].PaymentDate.AsTime()) {
				userPayments[i], userPayments[j] = userPayments[j], userPayments[i]
			}
		}
	}

	start := skip
	end := start + int(pageSize)

	var payments []*pb.Payment
	var nextPageToken string

	if start < len(userPayments) {
		if end > len(userPayments) {
			end = len(userPayments)
		}
		payments = userPayments[start:end]
		// Generate next page token if there are more payments
		if end < len(userPayments) {
			nextPageToken = fmt.Sprintf("next_%d", end)
		}
	} else {
		payments = []*pb.Payment{}
	}

	return &pb.ListPaymentsResponse{
		Payments:      payments,
		NextPageToken: nextPageToken,
	}, nil
}

// GetMonthlyStatement generates a monthly statement for a user
func (s *PaymentService) GetMonthlyStatement(ctx context.Context, req *pb.GetMonthlyStatementRequest) (*pb.MonthlyStatement, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}
	if req.Year <= 0 {
		return nil, status.Error(codes.InvalidArgument, "year must be positive")
	}
	if req.Month <= 0 || req.Month > 12 {
		return nil, status.Error(codes.InvalidArgument, "month must be between 1 and 12")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create date range for the requested month
	startOfMonth := time.Date(int(req.Year), time.Month(req.Month), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// For demo purposes, create a mock monthly statement
	// In a real implementation, this would aggregate transaction data
	statement := &pb.MonthlyStatement{
		Id:             uuid.New().String(),
		UserId:         req.UserId,
		CardId:         "mock-card-id", // In real implementation, would come from context or parameter
		Year:           req.Year,
		Month:          req.Month,
		TotalTrips:     int32(rand.Intn(50) + 10), // 10-60 trips
		TotalDistance:  float64(rand.Intn(1000) + 100), // 100-1100 km
		TotalAmount:    int64(rand.Intn(30000) + 5000), // 5,000-35,000 yen
		DiscountAmount: int64(rand.Intn(5000)), // 0-5,000 yen discount
		FinalAmount:    0, // Will be calculated below
		GeneratedAt:    timestamppb.New(time.Now()),
		PaymentDueDate: timestamppb.New(endOfMonth.AddDate(0, 1, 15)), // 15th of next month
	}

	statement.FinalAmount = statement.TotalAmount - statement.DiscountAmount

	return statement, nil
}

// simulatePaymentProcessing simulates async payment processing
func (s *PaymentService) simulatePaymentProcessing(paymentId string) {
	// Simulate processing time (1-5 seconds)
	time.Sleep(time.Duration(1+rand.Intn(4)) * time.Second)

	s.mu.Lock()
	defer s.mu.Unlock()

	payment, exists := s.payments[paymentId]
	if !exists {
		return
	}

	// Update to processing
	payment.Status = pb.PaymentProcessingStatus_PAYMENT_PROCESSING_STATUS_PROCESSING

	// Simulate additional processing time
	time.Sleep(time.Duration(1+rand.Intn(3)) * time.Second)

	// 95% success rate, 5% failure rate
	if rand.Float32() < 0.95 {
		payment.Status = pb.PaymentProcessingStatus_PAYMENT_PROCESSING_STATUS_COMPLETED
	} else {
		payment.Status = pb.PaymentProcessingStatus_PAYMENT_PROCESSING_STATUS_FAILED
	}
}

// GetPaymentCount returns the current number of payments (helper method for testing)
func (s *PaymentService) GetPaymentCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.payments)
}

// GetPaymentsByUserId returns all payments for a user (helper method for testing)
func (s *PaymentService) GetPaymentsByUserId(userId string) []*pb.Payment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var payments []*pb.Payment
	for _, payment := range s.payments {
		if payment.UserId == userId {
			payments = append(payments, payment)
		}
	}
	return payments
}

// UpdatePaymentStatus updates the processing status of a payment (helper method)
func (s *PaymentService) UpdatePaymentStatus(paymentId string, paymentStatus pb.PaymentProcessingStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	payment, exists := s.payments[paymentId]
	if !exists {
		return status.Error(codes.NotFound, "payment not found")
	}

	payment.Status = paymentStatus
	return nil
}

// GetTotalAmountByUser returns the total completed payment amount for a user (helper method)
func (s *PaymentService) GetTotalAmountByUser(userId string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var total int64
	for _, payment := range s.payments {
		if payment.UserId == userId &&
		   payment.Status == pb.PaymentProcessingStatus_PAYMENT_PROCESSING_STATUS_COMPLETED {
			total += payment.TotalAmount
		}
	}
	return total
}