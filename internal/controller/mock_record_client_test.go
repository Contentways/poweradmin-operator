/*
Copyright 2026 Patrick Omland.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	"github.com/stretchr/testify/mock"

	poweradmin "contentways.dev/contentways/poweradmin-go/poweradmin"
)

// MockRecordClient is a mock implementation of poweradmin.IRecordClient.
type MockRecordClient struct {
	mock.Mock
}

func (m *MockRecordClient) GetByID(ctx context.Context, zoneID int, recordID int64) (*poweradmin.Record, *poweradmin.Response, error) {
	args := m.Called(ctx, zoneID, recordID)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*poweradmin.Record), nil, args.Error(2)
}

func (m *MockRecordClient) List(ctx context.Context, zoneID int, opts poweradmin.RecordListOpts) ([]*poweradmin.Record, *poweradmin.Response, error) {
	args := m.Called(ctx, zoneID, opts)
	return args.Get(0).([]*poweradmin.Record), nil, args.Error(2)
}

func (m *MockRecordClient) All(ctx context.Context, zoneID int) ([]*poweradmin.Record, error) {
	args := m.Called(ctx, zoneID)
	return args.Get(0).([]*poweradmin.Record), args.Error(1)
}

func (m *MockRecordClient) Create(ctx context.Context, zoneID int, opts poweradmin.RecordCreateOpts) (int64, *poweradmin.Response, error) {
	args := m.Called(ctx, zoneID, opts)
	return args.Get(0).(int64), nil, args.Error(2)
}

func (m *MockRecordClient) Update(ctx context.Context, zoneID int, recordID int64, opts poweradmin.RecordUpdateOpts) (*poweradmin.Record, *poweradmin.Response, error) {
	args := m.Called(ctx, zoneID, recordID, opts)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*poweradmin.Record), nil, args.Error(2)
}

func (m *MockRecordClient) Delete(ctx context.Context, zoneID int, recordID int64) (*poweradmin.Response, error) {
	args := m.Called(ctx, zoneID, recordID)
	return nil, args.Error(1)
}

func (m *MockRecordClient) Bulk(ctx context.Context, zoneID int, ops []poweradmin.BulkRecordOperation) (*poweradmin.BulkRecordsResult, *poweradmin.Response, error) {
	args := m.Called(ctx, zoneID, ops)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*poweradmin.BulkRecordsResult), nil, args.Error(2)
}
