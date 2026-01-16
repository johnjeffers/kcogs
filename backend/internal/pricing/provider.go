package pricing

import (
	"context"

	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cogs"
)

// FargatePricing contains the per-unit pricing for Fargate
type FargatePricing struct {
	CPUPerHour    cogs.CostValue // Price per vCPU per hour
	MemoryPerHour cogs.CostValue // Price per GB memory per hour
}

// Provider retrieves pricing information for cloud resources
type Provider interface {
	// GetEC2Price returns the hourly on-demand price for an EC2 instance type in a region
	GetEC2Price(ctx context.Context, region, instanceType string) (cogs.CostValue, error)

	// GetEC2Prices returns prices for multiple instance types (for batch efficiency)
	GetEC2Prices(ctx context.Context, region string, instanceTypes []string) (map[string]cogs.CostValue, error)

	// GetFargatePricing returns the per-unit pricing for Fargate in a region
	GetFargatePricing(ctx context.Context, region string) (*FargatePricing, error)

	// RefreshCache forces a refresh of the pricing cache
	RefreshCache(ctx context.Context) error
}
