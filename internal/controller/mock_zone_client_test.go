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

// MockZoneClient is a mock implementation of poweradmin.IZoneClient.
type MockZoneClient struct {
	mock.Mock
}

func (m *MockZoneClient) GetByID(ctx context.Context, id int) (*poweradmin.Zone, *poweradmin.Response, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*poweradmin.Zone), nil, args.Error(2)
}

func (m *MockZoneClient) GetByName(ctx context.Context, name string) (*poweradmin.Zone, *poweradmin.Response, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*poweradmin.Zone), nil, args.Error(2)
}

func (m *MockZoneClient) List(ctx context.Context, opts poweradmin.ListOpts) ([]*poweradmin.Zone, *poweradmin.Response, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).([]*poweradmin.Zone), nil, args.Error(2)
}

func (m *MockZoneClient) All(ctx context.Context) ([]*poweradmin.Zone, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*poweradmin.Zone), args.Error(1)
}

func (m *MockZoneClient) Create(ctx context.Context, opts poweradmin.ZoneCreateOpts) (int, *poweradmin.Response, error) {
	args := m.Called(ctx, opts)
	return args.Int(0), nil, args.Error(2)
}

func (m *MockZoneClient) Update(ctx context.Context, id int, opts poweradmin.ZoneUpdateOpts) (*poweradmin.Zone, *poweradmin.Response, error) {
	args := m.Called(ctx, id, opts)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*poweradmin.Zone), nil, args.Error(2)
}

func (m *MockZoneClient) Delete(ctx context.Context, id int) (*poweradmin.Response, error) {
	args := m.Called(ctx, id)
	return nil, args.Error(1)
}

func (m *MockZoneClient) Owners(ctx context.Context, zoneID int) ([]*poweradmin.ZoneOwner, *poweradmin.Response, error) {
	args := m.Called(ctx, zoneID)
	return args.Get(0).([]*poweradmin.ZoneOwner), nil, args.Error(2)
}

func (m *MockZoneClient) AddOwner(ctx context.Context, zoneID, userID int) (*poweradmin.Response, error) {
	args := m.Called(ctx, zoneID, userID)
	return nil, args.Error(1)
}

func (m *MockZoneClient) AddOwners(ctx context.Context, zoneID int, userIDs []int) (*poweradmin.Response, error) {
	args := m.Called(ctx, zoneID, userIDs)
	return nil, args.Error(1)
}

func (m *MockZoneClient) RemoveOwner(ctx context.Context, zoneID, userID int) (*poweradmin.Response, error) {
	args := m.Called(ctx, zoneID, userID)
	return nil, args.Error(1)
}
