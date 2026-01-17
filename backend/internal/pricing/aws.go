package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	"github.com/aws/aws-sdk-go-v2/service/pricing/types"

	"github.com/johnjeffers/infra-utilities/kcogs/backend/internal/cogs"
)

// AWSProvider implements Provider using the AWS Price List API
type AWSProvider struct {
	client        *pricing.Client
	cache         map[string]cogs.CostValue // key: "region:instanceType"
	fargateCache  map[string]*FargatePricing // key: region
	cacheMu       sync.RWMutex
	cacheExpiry   time.Time
	cacheDuration time.Duration
}

// NewAWSProvider creates a new AWS pricing provider
func NewAWSProvider(ctx context.Context, cacheDurationMinutes int) (*AWSProvider, error) {
	// AWS Pricing API is only available in us-east-1 and ap-south-1
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	client := pricing.NewFromConfig(cfg)

	// Validate credentials by making a test API call
	if err := validateCredentials(ctx, client); err != nil {
		return nil, err
	}

	return &AWSProvider{
		client:        client,
		cache:         make(map[string]cogs.CostValue),
		fargateCache:  make(map[string]*FargatePricing),
		cacheDuration: time.Duration(cacheDurationMinutes) * time.Minute,
	}, nil
}

// validateCredentials checks that AWS credentials are configured and have access to the Pricing API
func validateCredentials(ctx context.Context, client *pricing.Client) error {
	// Make a minimal API call to validate credentials
	_, err := client.DescribeServices(ctx, &pricing.DescribeServicesInput{
		ServiceCode: aws.String("AmazonEC2"),
		MaxResults:  aws.Int32(1),
	})
	if err != nil {
		return fmt.Errorf("AWS credentials not found or invalid: %w", err)
	}
	return nil
}

// GetEC2Price returns the hourly on-demand price for an EC2 instance type
func (p *AWSProvider) GetEC2Price(ctx context.Context, region, instanceType string) (cogs.CostValue, error) {
	cacheKey := fmt.Sprintf("%s:%s", region, instanceType)

	// Check cache first
	p.cacheMu.RLock()
	if price, ok := p.cache[cacheKey]; ok && time.Now().Before(p.cacheExpiry) {
		p.cacheMu.RUnlock()
		return price, nil
	}
	p.cacheMu.RUnlock()

	// Fetch from API
	price, err := p.fetchPrice(ctx, region, instanceType)
	if err != nil {
		return 0, err
	}

	// Update cache
	p.cacheMu.Lock()
	p.cache[cacheKey] = price
	if p.cacheExpiry.IsZero() || time.Now().After(p.cacheExpiry) {
		p.cacheExpiry = time.Now().Add(p.cacheDuration)
	}
	p.cacheMu.Unlock()

	return price, nil
}

// GetEC2Prices returns prices for multiple instance types
func (p *AWSProvider) GetEC2Prices(ctx context.Context, region string, instanceTypes []string) (map[string]cogs.CostValue, error) {
	result := make(map[string]cogs.CostValue)

	for _, instanceType := range instanceTypes {
		price, err := p.GetEC2Price(ctx, region, instanceType)
		if err != nil {
			// Log error but continue with other instance types
			result[instanceType] = 0
			continue
		}
		result[instanceType] = price
	}

	return result, nil
}

// RefreshCache forces a refresh of the pricing cache
func (p *AWSProvider) RefreshCache(ctx context.Context) error {
	p.cacheMu.Lock()
	p.cache = make(map[string]cogs.CostValue)
	p.fargateCache = make(map[string]*FargatePricing)
	p.cacheExpiry = time.Time{}
	p.cacheMu.Unlock()
	return nil
}

// GetFargatePricing returns the per-unit pricing for Fargate in a region
func (p *AWSProvider) GetFargatePricing(ctx context.Context, region string) (*FargatePricing, error) {
	// Check cache first
	p.cacheMu.RLock()
	if pricing, ok := p.fargateCache[region]; ok && time.Now().Before(p.cacheExpiry) {
		p.cacheMu.RUnlock()
		return pricing, nil
	}
	p.cacheMu.RUnlock()

	// Fetch from API
	pricing, err := p.fetchFargatePricing(ctx, region)
	if err != nil {
		return nil, err
	}

	// Update cache
	p.cacheMu.Lock()
	p.fargateCache[region] = pricing
	if p.cacheExpiry.IsZero() || time.Now().After(p.cacheExpiry) {
		p.cacheExpiry = time.Now().Add(p.cacheDuration)
	}
	p.cacheMu.Unlock()

	return pricing, nil
}

// fetchFargatePricing queries the AWS Price List API for Fargate pricing
func (p *AWSProvider) fetchFargatePricing(ctx context.Context, region string) (*FargatePricing, error) {
	locationName, ok := regionToLocation[region]
	if !ok {
		return nil, fmt.Errorf("unknown region: %s", region)
	}

	result := &FargatePricing{}

	// Fetch vCPU pricing
	cpuPrice, err := p.fetchFargateComponentPrice(ctx, locationName, "vCPU-Hours")
	if err != nil {
		return nil, fmt.Errorf("fetching Fargate vCPU price: %w", err)
	}
	result.CPUPerHour = cpuPrice

	// Fetch Memory pricing
	memPrice, err := p.fetchFargateComponentPrice(ctx, locationName, "GB-Hours")
	if err != nil {
		return nil, fmt.Errorf("fetching Fargate memory price: %w", err)
	}
	result.MemoryPerHour = memPrice

	return result, nil
}

// fetchFargateComponentPrice fetches a specific Fargate pricing component
func (p *AWSProvider) fetchFargateComponentPrice(ctx context.Context, location, usageType string) (cogs.CostValue, error) {
	input := &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEKS"),
		Filters: []types.Filter{
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("location"),
				Value: aws.String(location),
			},
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("usagetype"),
				Value: aws.String(usageType),
			},
		},
		MaxResults: aws.Int32(10),
	}

	output, err := p.client.GetProducts(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("calling GetProducts: %w", err)
	}

	// Find the Fargate pricing in results
	for _, priceJSON := range output.PriceList {
		var product map[string]interface{}
		if err := json.Unmarshal([]byte(priceJSON), &product); err != nil {
			continue
		}

		// Check if this is a Fargate product
		attrs, ok := product["product"].(map[string]interface{})
		if !ok {
			continue
		}
		attributes, ok := attrs["attributes"].(map[string]interface{})
		if !ok {
			continue
		}

		// Look for Fargate in the usagetype or product family
		usageTypeAttr, _ := attributes["usagetype"].(string)
		if !containsFargate(usageTypeAttr) {
			continue
		}

		price, err := parsePriceFromProduct(priceJSON)
		if err != nil {
			continue
		}

		return price, nil
	}

	// If we couldn't find pricing from API, use fallback prices (us-east-1 prices as baseline)
	// These are approximate and should be updated periodically
	switch usageType {
	case "vCPU-Hours":
		return cogs.CostValue(0.04048), nil // Fargate vCPU per hour
	case "GB-Hours":
		return cogs.CostValue(0.004445), nil // Fargate Memory per GB per hour
	}

	return 0, fmt.Errorf("no Fargate pricing found for %s in %s", usageType, location)
}

// containsFargate checks if a string contains Fargate-related identifiers
func containsFargate(s string) bool {
	return len(s) > 0 && (contains(s, "Fargate") || contains(s, "fargate"))
}

// contains is a simple case-sensitive substring check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// fetchPrice queries the AWS Price List API for EC2 pricing
func (p *AWSProvider) fetchPrice(ctx context.Context, region, instanceType string) (cogs.CostValue, error) {
	locationName, ok := regionToLocation[region]
	if !ok {
		return 0, fmt.Errorf("unknown region: %s", region)
	}

	input := &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEC2"),
		Filters: []types.Filter{
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("instanceType"),
				Value: aws.String(instanceType),
			},
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("location"),
				Value: aws.String(locationName),
			},
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("operatingSystem"),
				Value: aws.String("Linux"),
			},
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("tenancy"),
				Value: aws.String("Shared"),
			},
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("preInstalledSw"),
				Value: aws.String("NA"),
			},
			{
				Type:  types.FilterTypeTermMatch,
				Field: aws.String("capacitystatus"),
				Value: aws.String("Used"),
			},
		},
		MaxResults: aws.Int32(1),
	}

	output, err := p.client.GetProducts(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("calling GetProducts: %w", err)
	}

	if len(output.PriceList) == 0 {
		return 0, fmt.Errorf("no pricing found for %s in %s", instanceType, region)
	}

	return parsePriceFromProduct(output.PriceList[0])
}

// parsePriceFromProduct extracts the hourly on-demand price from the AWS pricing JSON
func parsePriceFromProduct(priceListJSON string) (cogs.CostValue, error) {
	var product map[string]interface{}
	if err := json.Unmarshal([]byte(priceListJSON), &product); err != nil {
		return 0, fmt.Errorf("parsing price list JSON: %w", err)
	}

	terms, ok := product["terms"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("no terms in price list")
	}

	onDemand, ok := terms["OnDemand"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("no OnDemand terms in price list")
	}

	// Get the first (and usually only) offer
	for _, offerVal := range onDemand {
		offer, ok := offerVal.(map[string]interface{})
		if !ok {
			continue
		}

		priceDimensions, ok := offer["priceDimensions"].(map[string]interface{})
		if !ok {
			continue
		}

		// Get the first price dimension
		for _, dimVal := range priceDimensions {
			dim, ok := dimVal.(map[string]interface{})
			if !ok {
				continue
			}

			pricePerUnit, ok := dim["pricePerUnit"].(map[string]interface{})
			if !ok {
				continue
			}

			usdStr, ok := pricePerUnit["USD"].(string)
			if !ok {
				continue
			}

			price, err := strconv.ParseFloat(usdStr, 64)
			if err != nil {
				return 0, fmt.Errorf("parsing USD price: %w", err)
			}

			return cogs.CostValue(price), nil
		}
	}

	return 0, fmt.Errorf("could not extract price from product")
}

// regionToLocation maps AWS region codes to pricing API location names
var regionToLocation = map[string]string{
	"us-east-1":      "US East (N. Virginia)",
	"us-east-2":      "US East (Ohio)",
	"us-west-1":      "US West (N. California)",
	"us-west-2":      "US West (Oregon)",
	"af-south-1":     "Africa (Cape Town)",
	"ap-east-1":      "Asia Pacific (Hong Kong)",
	"ap-south-1":     "Asia Pacific (Mumbai)",
	"ap-south-2":     "Asia Pacific (Hyderabad)",
	"ap-southeast-1": "Asia Pacific (Singapore)",
	"ap-southeast-2": "Asia Pacific (Sydney)",
	"ap-southeast-3": "Asia Pacific (Jakarta)",
	"ap-southeast-4": "Asia Pacific (Melbourne)",
	"ap-northeast-1": "Asia Pacific (Tokyo)",
	"ap-northeast-2": "Asia Pacific (Seoul)",
	"ap-northeast-3": "Asia Pacific (Osaka)",
	"ca-central-1":   "Canada (Central)",
	"ca-west-1":      "Canada West (Calgary)",
	"eu-central-1":   "EU (Frankfurt)",
	"eu-central-2":   "EU (Zurich)",
	"eu-west-1":      "EU (Ireland)",
	"eu-west-2":      "EU (London)",
	"eu-west-3":      "EU (Paris)",
	"eu-south-1":     "EU (Milan)",
	"eu-south-2":     "EU (Spain)",
	"eu-north-1":     "EU (Stockholm)",
	"il-central-1":   "Israel (Tel Aviv)",
	"me-south-1":     "Middle East (Bahrain)",
	"me-central-1":   "Middle East (UAE)",
	"sa-east-1":      "South America (Sao Paulo)",
}
