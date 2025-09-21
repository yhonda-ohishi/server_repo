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
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CardService implements the CardServiceServer interface
type CardService struct {
	pb.UnimplementedCardServiceServer
	mu    sync.RWMutex
	cards map[string]*pb.ETCCard
}

// NewCardService creates a new CardService instance with mock data
func NewCardService() *CardService {
	service := &CardService{
		cards: make(map[string]*pb.ETCCard),
	}

	// Add mock data
	service.addMockData()
	return service
}

// addMockData populates the service with mock cards for testing
func (s *CardService) addMockData() {
	now := time.Now()
	mockUserIds := []string{
		"user-001",
		"user-002",
		"user-003",
	}

	vehicleTypes := []pb.VehicleType{
		pb.VehicleType_VEHICLE_TYPE_REGULAR,
		pb.VehicleType_VEHICLE_TYPE_KEI,
		pb.VehicleType_VEHICLE_TYPE_LARGE,
	}

	// Generate mock cards for each user
	for i, userId := range mockUserIds {
		for j := 0; j < 2; j++ { // 2 cards per user
			cardNumber := fmt.Sprintf("1234-%04d-%04d-%04d", i+1, j+1, rand.Intn(10000))

			status := pb.CardStatus_CARD_STATUS_ACTIVE
			if rand.Float32() < 0.2 { // 20% chance of suspended
				status = pb.CardStatus_CARD_STATUS_SUSPENDED
			} else if rand.Float32() < 0.1 { // 10% chance of expired
				status = pb.CardStatus_CARD_STATUS_EXPIRED
			}

			vehicleType := vehicleTypes[rand.Intn(len(vehicleTypes))]

			// Random issue and expiry dates
			issueDate := now.Add(-time.Duration(rand.Intn(1095)) * 24 * time.Hour) // 0-3 years ago
			expiryDate := issueDate.Add(5 * 365 * 24 * time.Hour) // 5 years from issue

			card := &pb.ETCCard{
				Id:            uuid.New().String(),
				UserId:        userId,
				CardNumber:    cardNumber,
				Status:        status,
				VehicleType:   vehicleType,
				VehicleNumber: fmt.Sprintf("Vehicle-%d-%d", i+1, j+1),
				ExpiryDate:    timestamppb.New(expiryDate),
				CreatedAt:     timestamppb.New(issueDate),
				ActivatedAt:   timestamppb.New(issueDate),
			}

			if status != pb.CardStatus_CARD_STATUS_ACTIVE {
				card.DeactivatedAt = timestamppb.New(now.Add(-time.Duration(rand.Intn(30)) * 24 * time.Hour))
			}

			s.cards[card.Id] = card
		}
	}
}

// GetCard retrieves a card by ID
func (s *CardService) GetCard(ctx context.Context, req *pb.GetCardRequest) (*pb.ETCCard, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "card ID is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	card, exists := s.cards[req.Id]
	if !exists {
		return nil, status.Error(codes.NotFound, "card not found")
	}

	return card, nil
}

// CreateCard creates a new card
func (s *CardService) CreateCard(ctx context.Context, req *pb.CreateCardRequest) (*pb.ETCCard, error) {
	// Validate required fields
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}
	if req.VehicleType == pb.VehicleType_VEHICLE_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "vehicle type is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Use provided card number or generate one
	cardNumber := req.CardNumber
	if cardNumber == "" {
		cardNumber = s.generateCardNumber()
	}

	// Check if card number already exists
	for _, card := range s.cards {
		if card.CardNumber == cardNumber {
			return nil, status.Error(codes.AlreadyExists, "card number already exists")
		}
	}

	// Create new card
	now := timestamppb.New(time.Now())
	expiryDate := req.ExpiryDate
	if expiryDate == nil {
		expiryDate = timestamppb.New(time.Now().Add(5 * 365 * 24 * time.Hour)) // 5 years from now
	}

	card := &pb.ETCCard{
		Id:            uuid.New().String(),
		UserId:        req.UserId,
		CardNumber:    cardNumber,
		Status:        pb.CardStatus_CARD_STATUS_ACTIVE,
		VehicleType:   req.VehicleType,
		VehicleNumber: req.VehicleNumber,
		ExpiryDate:    expiryDate,
		CreatedAt:     now,
		ActivatedAt:   now,
	}

	s.cards[card.Id] = card
	return card, nil
}

// UpdateCard updates an existing card
func (s *CardService) UpdateCard(ctx context.Context, req *pb.UpdateCardRequest) (*pb.ETCCard, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "card ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	card, exists := s.cards[req.Id]
	if !exists {
		return nil, status.Error(codes.NotFound, "card not found")
	}

	// Update fields if provided
	if req.Status != pb.CardStatus_CARD_STATUS_UNSPECIFIED {
		card.Status = req.Status
		// Update activation/deactivation timestamps based on status
		now := time.Now()
		if req.Status == pb.CardStatus_CARD_STATUS_ACTIVE && card.ActivatedAt == nil {
			card.ActivatedAt = timestamppb.New(now)
			card.DeactivatedAt = nil
		} else if req.Status != pb.CardStatus_CARD_STATUS_ACTIVE && card.DeactivatedAt == nil {
			card.DeactivatedAt = timestamppb.New(now)
		}
	}
	if req.VehicleType != pb.VehicleType_VEHICLE_TYPE_UNSPECIFIED {
		card.VehicleType = req.VehicleType
	}
	if req.VehicleNumber != "" {
		card.VehicleNumber = req.VehicleNumber
	}

	return card, nil
}

// DeleteCard deletes a card by ID
func (s *CardService) DeleteCard(ctx context.Context, req *pb.DeleteCardRequest) (*emptypb.Empty, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "card ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.cards[req.Id]
	if !exists {
		return nil, status.Error(codes.NotFound, "card not found")
	}

	delete(s.cards, req.Id)
	return &emptypb.Empty{}, nil
}

// ListCards lists cards for a user
func (s *CardService) ListCards(ctx context.Context, req *pb.ListCardsRequest) (*pb.ListCardsResponse, error) {
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

	// Filter cards for the specified user
	var userCards []*pb.ETCCard
	for _, card := range s.cards {
		if card.UserId == req.UserId {
			userCards = append(userCards, card)
		}
	}

	// Sort cards by creation date (newest first)
	for i := 0; i < len(userCards)-1; i++ {
		for j := i + 1; j < len(userCards); j++ {
			if userCards[i].CreatedAt.AsTime().Before(userCards[j].CreatedAt.AsTime()) {
				userCards[i], userCards[j] = userCards[j], userCards[i]
			}
		}
	}

	start := skip
	end := start + int(pageSize)

	var cards []*pb.ETCCard
	var nextPageToken string

	if start < len(userCards) {
		if end > len(userCards) {
			end = len(userCards)
		}
		cards = userCards[start:end]
		// Generate next page token if there are more cards
		if end < len(userCards) {
			nextPageToken = fmt.Sprintf("next_%d", end)
		}
	} else {
		cards = []*pb.ETCCard{}
	}

	return &pb.ListCardsResponse{
		Cards:         cards,
		NextPageToken: nextPageToken,
	}, nil
}

// generateCardNumber generates a random card number in the format XXXX-XXXX-XXXX-XXXX
func (s *CardService) generateCardNumber() string {
	return fmt.Sprintf("%04d-%04d-%04d-%04d",
		rand.Intn(10000),
		rand.Intn(10000),
		rand.Intn(10000),
		rand.Intn(10000))
}

// Helper methods for testing and integration
func (s *CardService) GetCardCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.cards)
}

func (s *CardService) GetCardsByUserId(userId string) []*pb.ETCCard {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var cards []*pb.ETCCard
	for _, card := range s.cards {
		if card.UserId == userId {
			cards = append(cards, card)
		}
	}
	return cards
}

func (s *CardService) GetCardByNumber(cardNumber string) (*pb.ETCCard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, card := range s.cards {
		if card.CardNumber == cardNumber {
			return card, nil
		}
	}
	return nil, fmt.Errorf("card with number %s not found", cardNumber)
}