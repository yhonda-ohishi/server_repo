package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	proto "github.com/yhonda-ohishi/db-handler-server/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ETCServiceServer implements the ETC明細 gRPC service
type ETCServiceServer struct {
	proto.UnimplementedETCServiceServer
	// In a real implementation, this would connect to a database
	// For now, we'll use in-memory storage for testing
	etcData map[int64]*proto.ETCMeisai
	nextID  int64
}

// NewETCServiceServer creates a new ETC service server
func NewETCServiceServer() *ETCServiceServer {
	server := &ETCServiceServer{
		etcData: make(map[int64]*proto.ETCMeisai),
		nextID:  1,
	}

	// Add some test data
	server.seedTestData()

	return server
}

// seedTestData adds test ETC明細 data
func (s *ETCServiceServer) seedTestData() {
	now := timestamppb.Now()

	testData := []*proto.ETCMeisai{
		{
			Id:             1,
			Hash:           s.generateHashForData("2024-01-15", "Tokyo", "Osaka", "1234"),
			Date:           "2024-01-15",
			Time:           "08:30:00",
			CarType:        "普通車",
			CarNumber:      "品川 500 あ 1234",
			EntranceIc:     "首都高速道路 入口",
			ExitIc:         "名神高速道路 出口",
			Distance:       450,
			TollAmount:     8500,
			DiscountAmount: 500,
			FinalAmount:    8000,
			PaymentMethod:  "ETC",
			CardNumber:     "****-****-****-1234",
			UserId:         "user001",
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		{
			Id:             2,
			Hash:           s.generateHashForData("2024-01-20", "Yokohama", "Nagoya", "5678"),
			Date:           "2024-01-20",
			Time:           "14:15:00",
			CarType:        "普通車",
			CarNumber:      "横浜 301 さ 5678",
			EntranceIc:     "第三京浜道路 入口",
			ExitIc:         "東名高速道路 出口",
			Distance:       320,
			TollAmount:     6200,
			DiscountAmount: 300,
			FinalAmount:    5900,
			PaymentMethod:  "ETC",
			CardNumber:     "****-****-****-5678",
			UserId:         "user002",
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		{
			Id:             3,
			Hash:           s.generateHashForData("2024-02-01", "Kyoto", "Fukuoka", "9999"),
			Date:           "2024-02-01",
			Time:           "10:45:00",
			CarType:        "大型車",
			CarNumber:      "京都 100 き 9999",
			EntranceIc:     "名神高速道路 入口",
			ExitIc:         "九州自動車道 出口",
			Distance:       680,
			TollAmount:     15400,
			DiscountAmount: 1000,
			FinalAmount:    14400,
			PaymentMethod:  "ETC",
			CardNumber:     "****-****-****-9999",
			UserId:         "user001",
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}

	for _, data := range testData {
		s.etcData[data.Id] = data
		if data.Id >= s.nextID {
			s.nextID = data.Id + 1
		}
	}
}

// generateHashForData generates a SHA256 hash for ETC data
func (s *ETCServiceServer) generateHashForData(date, entrance, exit, carNumber string) string {
	data := fmt.Sprintf("%s-%s-%s-%s", date, entrance, exit, carNumber)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// CreateETCMeisai creates a new ETC明細 record
func (s *ETCServiceServer) CreateETCMeisai(ctx context.Context, req *proto.CreateETCMeisaiRequest) (*proto.ETCMeisaiResponse, error) {
	if req.EtcMeisai == nil {
		return nil, status.Error(codes.InvalidArgument, "ETC明細 data is required")
	}

	etcMeisai := req.EtcMeisai
	etcMeisai.Id = s.nextID
	s.nextID++

	// Generate hash if not provided
	if etcMeisai.Hash == "" {
		etcMeisai.Hash = s.generateHashForData(etcMeisai.Date, etcMeisai.EntranceIc, etcMeisai.ExitIc, etcMeisai.CarNumber)
	}

	now := timestamppb.Now()
	etcMeisai.CreatedAt = now
	etcMeisai.UpdatedAt = now

	s.etcData[etcMeisai.Id] = etcMeisai

	return &proto.ETCMeisaiResponse{EtcMeisai: etcMeisai}, nil
}

// GetETCMeisai retrieves an ETC明細 record by ID
func (s *ETCServiceServer) GetETCMeisai(ctx context.Context, req *proto.GetETCMeisaiRequest) (*proto.ETCMeisaiResponse, error) {
	etcMeisai, exists := s.etcData[req.Id]
	if !exists {
		return nil, status.Error(codes.NotFound, "ETC明細 not found")
	}

	return &proto.ETCMeisaiResponse{EtcMeisai: etcMeisai}, nil
}

// UpdateETCMeisai updates an existing ETC明細 record
func (s *ETCServiceServer) UpdateETCMeisai(ctx context.Context, req *proto.UpdateETCMeisaiRequest) (*proto.ETCMeisaiResponse, error) {
	existing, exists := s.etcData[req.Id]
	if !exists {
		return nil, status.Error(codes.NotFound, "ETC明細 not found")
	}

	if req.EtcMeisai == nil {
		return nil, status.Error(codes.InvalidArgument, "ETC明細 data is required")
	}

	updated := req.EtcMeisai
	updated.Id = req.Id
	updated.CreatedAt = existing.CreatedAt
	updated.UpdatedAt = timestamppb.Now()

	s.etcData[req.Id] = updated

	return &proto.ETCMeisaiResponse{EtcMeisai: updated}, nil
}

// DeleteETCMeisai deletes an ETC明細 record
func (s *ETCServiceServer) DeleteETCMeisai(ctx context.Context, req *proto.DeleteETCMeisaiRequest) (*emptypb.Empty, error) {
	if _, exists := s.etcData[req.Id]; !exists {
		return nil, status.Error(codes.NotFound, "ETC明細 not found")
	}

	delete(s.etcData, req.Id)

	return &emptypb.Empty{}, nil
}

// ListETCMeisai lists ETC明細 records with pagination
func (s *ETCServiceServer) ListETCMeisai(ctx context.Context, req *proto.ListETCMeisaiRequest) (*proto.ListETCMeisaiResponse, error) {
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var allRecords []*proto.ETCMeisai
	for _, record := range s.etcData {
		allRecords = append(allRecords, record)
	}

	// Simple pagination logic
	startIndex := 0
	if req.PageToken != "" {
		if parsed, err := strconv.Atoi(req.PageToken); err == nil {
			startIndex = parsed
		}
	}

	endIndex := startIndex + int(pageSize)
	if endIndex > len(allRecords) {
		endIndex = len(allRecords)
	}

	var paginatedRecords []*proto.ETCMeisai
	if startIndex < len(allRecords) {
		paginatedRecords = allRecords[startIndex:endIndex]
	}

	var nextPageToken string
	if endIndex < len(allRecords) {
		nextPageToken = strconv.Itoa(endIndex)
	}

	return &proto.ListETCMeisaiResponse{
		EtcMeisaiList:   paginatedRecords,
		NextPageToken:   nextPageToken,
		TotalCount:      int32(len(allRecords)),
	}, nil
}

// BulkCreateETCMeisai creates multiple ETC明細 records
func (s *ETCServiceServer) BulkCreateETCMeisai(ctx context.Context, req *proto.BulkCreateETCMeisaiRequest) (*proto.BulkCreateETCMeisaiResponse, error) {
	var created []*proto.ETCMeisai
	var errorMessages []string
	successCount := 0

	for _, etcMeisai := range req.EtcMeisaiList {
		createReq := &proto.CreateETCMeisaiRequest{EtcMeisai: etcMeisai}
		resp, err := s.CreateETCMeisai(ctx, createReq)
		if err != nil {
			errorMessages = append(errorMessages, err.Error())
		} else {
			created = append(created, resp.EtcMeisai)
			successCount++
		}
	}

	return &proto.BulkCreateETCMeisaiResponse{
		CreatedEtcMeisaiList: created,
		SuccessCount:         int32(successCount),
		ErrorCount:           int32(len(errorMessages)),
		ErrorMessages:        errorMessages,
	}, nil
}

// BulkUpdateETCMeisai updates multiple ETC明細 records
func (s *ETCServiceServer) BulkUpdateETCMeisai(ctx context.Context, req *proto.BulkUpdateETCMeisaiRequest) (*proto.BulkUpdateETCMeisaiResponse, error) {
	var updated []*proto.ETCMeisai
	var errorMessages []string
	successCount := 0

	for _, etcMeisai := range req.EtcMeisaiList {
		updateReq := &proto.UpdateETCMeisaiRequest{Id: etcMeisai.Id, EtcMeisai: etcMeisai}
		resp, err := s.UpdateETCMeisai(ctx, updateReq)
		if err != nil {
			errorMessages = append(errorMessages, err.Error())
		} else {
			updated = append(updated, resp.EtcMeisai)
			successCount++
		}
	}

	return &proto.BulkUpdateETCMeisaiResponse{
		UpdatedEtcMeisaiList: updated,
		SuccessCount:         int32(successCount),
		ErrorCount:           int32(len(errorMessages)),
		ErrorMessages:        errorMessages,
	}, nil
}

// GetETCMeisaiByDateRange retrieves ETC明細 records within a date range
func (s *ETCServiceServer) GetETCMeisaiByDateRange(ctx context.Context, req *proto.GetETCMeisaiByDateRangeRequest) (*proto.ListETCMeisaiResponse, error) {
	var filteredRecords []*proto.ETCMeisai

	for _, record := range s.etcData {
		if req.StartDate != "" && record.Date < req.StartDate {
			continue
		}
		if req.EndDate != "" && record.Date > req.EndDate {
			continue
		}
		filteredRecords = append(filteredRecords, record)
	}

	// Apply pagination
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	startIndex := 0
	if req.PageToken != "" {
		if parsed, err := strconv.Atoi(req.PageToken); err == nil {
			startIndex = parsed
		}
	}

	endIndex := startIndex + int(pageSize)
	if endIndex > len(filteredRecords) {
		endIndex = len(filteredRecords)
	}

	var paginatedRecords []*proto.ETCMeisai
	if startIndex < len(filteredRecords) {
		paginatedRecords = filteredRecords[startIndex:endIndex]
	}

	var nextPageToken string
	if endIndex < len(filteredRecords) {
		nextPageToken = strconv.Itoa(endIndex)
	}

	return &proto.ListETCMeisaiResponse{
		EtcMeisaiList:   paginatedRecords,
		NextPageToken:   nextPageToken,
		TotalCount:      int32(len(filteredRecords)),
	}, nil
}

// GetETCMeisaiByHash retrieves ETC明細 record by hash
func (s *ETCServiceServer) GetETCMeisaiByHash(ctx context.Context, req *proto.GetETCMeisaiByHashRequest) (*proto.ETCMeisaiResponse, error) {
	for _, record := range s.etcData {
		if record.Hash == req.Hash {
			return &proto.ETCMeisaiResponse{EtcMeisai: record}, nil
		}
	}

	return nil, status.Error(codes.NotFound, "ETC明細 with specified hash not found")
}

// GetUnmappedETCMeisai retrieves ETC明細 records that are not mapped (example implementation)
func (s *ETCServiceServer) GetUnmappedETCMeisai(ctx context.Context, req *proto.GetUnmappedETCMeisaiRequest) (*proto.ListETCMeisaiResponse, error) {
	// For demonstration, we'll consider records without UserId as unmapped
	var unmappedRecords []*proto.ETCMeisai

	for _, record := range s.etcData {
		if record.UserId == "" {
			unmappedRecords = append(unmappedRecords, record)
		}
	}

	// Apply pagination
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	startIndex := 0
	if req.PageToken != "" {
		if parsed, err := strconv.Atoi(req.PageToken); err == nil {
			startIndex = parsed
		}
	}

	endIndex := startIndex + int(pageSize)
	if endIndex > len(unmappedRecords) {
		endIndex = len(unmappedRecords)
	}

	var paginatedRecords []*proto.ETCMeisai
	if startIndex < len(unmappedRecords) {
		paginatedRecords = unmappedRecords[startIndex:endIndex]
	}

	var nextPageToken string
	if endIndex < len(unmappedRecords) {
		nextPageToken = strconv.Itoa(endIndex)
	}

	return &proto.ListETCMeisaiResponse{
		EtcMeisaiList:   paginatedRecords,
		NextPageToken:   nextPageToken,
		TotalCount:      int32(len(unmappedRecords)),
	}, nil
}

// CheckDuplicatesByHash checks for duplicate hashes
func (s *ETCServiceServer) CheckDuplicatesByHash(ctx context.Context, req *proto.CheckDuplicatesByHashRequest) (*proto.CheckDuplicatesResponse, error) {
	var duplicates []string

	for _, hash := range req.Hashes {
		for _, record := range s.etcData {
			if record.Hash == hash {
				duplicates = append(duplicates, hash)
				break
			}
		}
	}

	return &proto.CheckDuplicatesResponse{
		DuplicateHashes: duplicates,
		DuplicateCount:  int32(len(duplicates)),
	}, nil
}

// GenerateHash generates a hash for ETC明細 data
func (s *ETCServiceServer) GenerateHash(ctx context.Context, req *proto.GenerateHashRequest) (*proto.GenerateHashResponse, error) {
	if req.EtcMeisai == nil {
		return nil, status.Error(codes.InvalidArgument, "ETC明細 data is required")
	}

	hash := s.generateHashForData(
		req.EtcMeisai.Date,
		req.EtcMeisai.EntranceIc,
		req.EtcMeisai.ExitIc,
		req.EtcMeisai.CarNumber,
	)

	return &proto.GenerateHashResponse{Hash: hash}, nil
}

// GetETCSummary returns summary statistics for ETC明細 data
func (s *ETCServiceServer) GetETCSummary(ctx context.Context, req *proto.GetETCSummaryRequest) (*proto.GetETCSummaryResponse, error) {
	var filteredRecords []*proto.ETCMeisai

	for _, record := range s.etcData {
		// Filter by date range
		if req.StartDate != "" && record.Date < req.StartDate {
			continue
		}
		if req.EndDate != "" && record.Date > req.EndDate {
			continue
		}
		// Filter by user ID
		if req.UserId != "" && record.UserId != req.UserId {
			continue
		}
		filteredRecords = append(filteredRecords, record)
	}

	var totalAmount, totalToll, totalDiscount int64
	monthlyStats := make(map[string]*proto.ETCMonthlySummary)

	for _, record := range filteredRecords {
		totalAmount += int64(record.FinalAmount)
		totalToll += int64(record.TollAmount)
		totalDiscount += int64(record.DiscountAmount)

		// Extract year-month from date
		dateParts := strings.Split(record.Date, "-")
		if len(dateParts) >= 2 {
			yearStr, monthStr := dateParts[0], dateParts[1]
			key := fmt.Sprintf("%s-%s", yearStr, monthStr)

			if _, exists := monthlyStats[key]; !exists {
				year, _ := strconv.Atoi(yearStr)
				month, _ := strconv.Atoi(monthStr)
				monthlyStats[key] = &proto.ETCMonthlySummary{
					Year:             int32(year),
					Month:            int32(month),
					TransactionCount: 0,
					TotalAmount:      0,
				}
			}

			monthlyStats[key].TransactionCount++
			monthlyStats[key].TotalAmount += int64(record.FinalAmount)
		}
	}

	var monthlySummaries []*proto.ETCMonthlySummary
	for _, summary := range monthlyStats {
		monthlySummaries = append(monthlySummaries, summary)
	}

	return &proto.GetETCSummaryResponse{
		TotalTransactions: int32(len(filteredRecords)),
		TotalAmount:       totalAmount,
		TotalToll:         totalToll,
		TotalDiscount:     totalDiscount,
		MonthlySummaries:  monthlySummaries,
	}, nil
}

// GetMonthlyStats returns detailed monthly statistics
func (s *ETCServiceServer) GetMonthlyStats(ctx context.Context, req *proto.GetMonthlyStatsRequest) (*proto.GetMonthlyStatsResponse, error) {
	targetMonth := fmt.Sprintf("%04d-%02d", req.Year, req.Month)

	var monthlyRecords []*proto.ETCMeisai
	for _, record := range s.etcData {
		// Filter by user ID
		if req.UserId != "" && record.UserId != req.UserId {
			continue
		}

		// Filter by year-month
		if len(record.Date) >= 7 && record.Date[:7] == targetMonth {
			monthlyRecords = append(monthlyRecords, record)
		}
	}

	var totalAmount int64
	dailyStats := make(map[int32]*proto.ETCDailyStat)

	for _, record := range monthlyRecords {
		totalAmount += int64(record.FinalAmount)

		// Extract day from date
		dateParts := strings.Split(record.Date, "-")
		if len(dateParts) >= 3 {
			day, _ := strconv.Atoi(dateParts[2])
			dayKey := int32(day)

			if _, exists := dailyStats[dayKey]; !exists {
				dailyStats[dayKey] = &proto.ETCDailyStat{
					Day:              dayKey,
					TransactionCount: 0,
					TotalAmount:      0,
				}
			}

			dailyStats[dayKey].TransactionCount++
			dailyStats[dayKey].TotalAmount += int64(record.FinalAmount)
		}
	}

	var dailyStatsList []*proto.ETCDailyStat
	for _, stat := range dailyStats {
		dailyStatsList = append(dailyStatsList, stat)
	}

	var averageAmount int64
	if len(monthlyRecords) > 0 {
		averageAmount = totalAmount / int64(len(monthlyRecords))
	}

	return &proto.GetMonthlyStatsResponse{
		Year:             req.Year,
		Month:            req.Month,
		TransactionCount: int32(len(monthlyRecords)),
		TotalAmount:      totalAmount,
		AverageAmount:    averageAmount,
		DailyStats:       dailyStatsList,
	}, nil
}