package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "github.com/yhonda-ohishi/db-handler-server/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UserService implements the UserServiceServer interface
type UserService struct {
	pb.UnimplementedUserServiceServer
	mu    sync.RWMutex
	users map[string]*pb.User
}

// NewUserService creates a new UserService instance with mock data
func NewUserService() *UserService {
	service := &UserService{
		users: make(map[string]*pb.User),
	}

	// Add mock data
	service.addMockData()
	return service
}

// addMockData populates the service with mock users for testing
func (s *UserService) addMockData() {
	mockUsers := []*pb.User{
		{
			Id:          uuid.New().String(),
			Email:       "john.doe@example.com",
			Name:        "John Doe",
			PhoneNumber: "+81-80-1234-5678",
			Address:     "1-1-1 Shibuya, Shibuya-ku, Tokyo 150-0002, Japan",
			CreatedAt:   timestamppb.New(time.Now().Add(-30 * 24 * time.Hour)),
			UpdatedAt:   timestamppb.New(time.Now()),
			Status:      pb.UserStatus_USER_STATUS_ACTIVE,
		},
		{
			Id:          uuid.New().String(),
			Email:       "jane.smith@example.com",
			Name:        "Jane Smith",
			PhoneNumber: "+81-90-9876-5432",
			Address:     "2-2-2 Shinjuku, Shinjuku-ku, Tokyo 160-0022, Japan",
			CreatedAt:   timestamppb.New(time.Now().Add(-15 * 24 * time.Hour)),
			UpdatedAt:   timestamppb.New(time.Now()),
			Status:      pb.UserStatus_USER_STATUS_ACTIVE,
		},
		{
			Id:          uuid.New().String(),
			Email:       "suspended.user@example.com",
			Name:        "Suspended User",
			PhoneNumber: "+81-70-1111-2222",
			Address:     "3-3-3 Harajuku, Shibuya-ku, Tokyo 150-0001, Japan",
			CreatedAt:   timestamppb.New(time.Now().Add(-60 * 24 * time.Hour)),
			UpdatedAt:   timestamppb.New(time.Now()),
			Status:      pb.UserStatus_USER_STATUS_SUSPENDED,
		},
	}

	for _, user := range mockUsers {
		s.users[user.Id] = user
	}
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[req.Id]
	if !exists {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return user, nil
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
	// Validate required fields
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	// Basic email validation
	if !strings.Contains(req.Email, "@") {
		return nil, status.Error(codes.InvalidArgument, "invalid email format")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if email already exists
	for _, user := range s.users {
		if user.Email == req.Email {
			return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
		}
	}

	// Create new user
	now := timestamppb.New(time.Now())
	user := &pb.User{
		Id:          uuid.New().String(),
		Email:       req.Email,
		Name:        req.Name,
		PhoneNumber: req.PhoneNumber,
		Address:     req.Address,
		CreatedAt:   now,
		UpdatedAt:   now,
		Status:      pb.UserStatus_USER_STATUS_ACTIVE,
	}

	s.users[user.Id] = user
	return user, nil
}

// UpdateUser updates an existing user
func (s *UserService) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.User, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[req.Id]
	if !exists {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	// Update fields if provided
	if req.Email != "" {
		// Check if new email already exists (but not for the same user)
		for id, existingUser := range s.users {
			if id != req.Id && existingUser.Email == req.Email {
				return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
			}
		}

		// Basic email validation
		if !strings.Contains(req.Email, "@") {
			return nil, status.Error(codes.InvalidArgument, "invalid email format")
		}

		user.Email = req.Email
	}
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.PhoneNumber != "" {
		user.PhoneNumber = req.PhoneNumber
	}
	if req.Address != "" {
		user.Address = req.Address
	}

	user.UpdatedAt = timestamppb.New(time.Now())
	return user, nil
}

// DeleteUser deletes a user by ID
func (s *UserService) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*emptypb.Empty, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "user ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.users[req.Id]
	if !exists {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	delete(s.users, req.Id)
	return &emptypb.Empty{}, nil
}

// ListUsers lists users with pagination
func (s *UserService) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
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

	// Convert map to slice for pagination
	allUsers := make([]*pb.User, 0, len(s.users))
	for _, user := range s.users {
		allUsers = append(allUsers, user)
	}

	// Sort by creation date (newest first)
	for i := 0; i < len(allUsers)-1; i++ {
		for j := i + 1; j < len(allUsers); j++ {
			if allUsers[i].CreatedAt.AsTime().Before(allUsers[j].CreatedAt.AsTime()) {
				allUsers[i], allUsers[j] = allUsers[j], allUsers[i]
			}
		}
	}

	start := skip
	end := start + int(pageSize)

	var users []*pb.User
	var nextPageToken string

	if start < len(allUsers) {
		if end > len(allUsers) {
			end = len(allUsers)
		}
		users = allUsers[start:end]
		// Generate next page token if there are more users
		if end < len(allUsers) {
			nextPageToken = fmt.Sprintf("next_%d", end)
		}
	} else {
		users = []*pb.User{}
	}

	return &pb.ListUsersResponse{
		Users:         users,
		NextPageToken: nextPageToken,
	}, nil
}

// GetUserCount returns the current number of users (helper method for testing)
func (s *UserService) GetUserCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users)
}

// GetUserByEmail retrieves a user by email (helper method for testing)
func (s *UserService) GetUserByEmail(email string) (*pb.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user with email %s not found", email)
}