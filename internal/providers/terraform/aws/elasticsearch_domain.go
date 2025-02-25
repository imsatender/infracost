package aws

import (
	"fmt"

	"github.com/infracost/infracost/internal/schema"
	"github.com/shopspring/decimal"
)

func GetElasticsearchDomainRegistryItem() *schema.RegistryItem {
	return &schema.RegistryItem{
		Name:  "aws_elasticsearch_domain",
		RFunc: NewElasticsearchDomain,
	}
}

func NewElasticsearchDomain(d *schema.ResourceData, u *schema.UsageData) *schema.Resource {
	region := d.Get("region").String()
	clusterConfig := d.Get("cluster_config").Array()[0]
	instanceType := clusterConfig.Get("instance_type").String()
	instanceCount := clusterConfig.Get("instance_count").Int()
	dedicatedMasterEnabled := clusterConfig.Get("dedicated_master_enabled").Bool()
	dedicatedMasterType := clusterConfig.Get("dedicated_master_type").String()
	dedicatedMasterCount := clusterConfig.Get("dedicated_master_count").Int()
	ultrawarmEnabled := clusterConfig.Get("warm_enabled").Bool()
	ultrawarmType := clusterConfig.Get("warm_type").String()
	ultrawarmCount := clusterConfig.Get("warm_count").Int()

	ebsOptions := d.Get("ebs_options").Array()[0]

	ebsTypeMap := map[string]string{
		"gp2":      "GP2",
		"io1":      "PIOPS-Storage",
		"standard": "Magnetic",
	}

	gbVal := decimal.NewFromInt(int64(defaultVolumeSize))
	if ebsOptions.Get("volume_size").Exists() {
		gbVal = decimal.NewFromFloat(ebsOptions.Get("volume_size").Float())
	}

	ebsType := "gp2"
	if ebsOptions.Get("volume_type").Exists() {
		ebsType = ebsOptions.Get("volume_type").String()
	}

	ebsFilter := "gp2"
	if val, ok := ebsTypeMap[ebsType]; ok {
		ebsFilter = val
	}

	iopsVal := decimal.NewFromInt(1)
	if ebsOptions.Get("iops").Exists() {
		iopsVal = decimal.NewFromFloat(ebsOptions.Get("iops").Float())

		if iopsVal.LessThan(decimal.NewFromInt(1)) {
			iopsVal = decimal.NewFromInt(1)
		}
	}

	costComponents := []*schema.CostComponent{
		{
			Name:           fmt.Sprintf("Instance (on-demand, %s)", instanceType),
			Unit:           "hours",
			UnitMultiplier: 1,
			HourlyQuantity: decimalPtr(decimal.NewFromInt(instanceCount)),
			ProductFilter: &schema.ProductFilter{
				VendorName:    strPtr("aws"),
				Region:        strPtr(region),
				Service:       strPtr("AmazonES"),
				ProductFamily: strPtr("Elastic Search Instance"),
				AttributeFilters: []*schema.AttributeFilter{
					{Key: "usagetype", ValueRegex: strPtr("/ESInstance/")},
					{Key: "instanceType", Value: &instanceType},
				},
			},
			PriceFilter: &schema.PriceFilter{
				PurchaseOption: strPtr("on_demand"),
			},
		},
		{
			Name:            fmt.Sprintf("Storage (%s)", ebsType),
			Unit:            "GB-months",
			UnitMultiplier:  1,
			MonthlyQuantity: &gbVal,
			ProductFilter: &schema.ProductFilter{
				VendorName:    strPtr("aws"),
				Region:        strPtr(region),
				Service:       strPtr("AmazonES"),
				ProductFamily: strPtr("Elastic Search Volume"),
				AttributeFilters: []*schema.AttributeFilter{
					{Key: "usagetype", ValueRegex: strPtr("/ES.+-Storage/")},
					{Key: "storageMedia", Value: strPtr(ebsFilter)},
				},
			},
			PriceFilter: &schema.PriceFilter{
				PurchaseOption: strPtr("on_demand"),
			},
		},
	}

	if ebsType == "io1" {
		costComponents = append(costComponents, &schema.CostComponent{
			Name:            fmt.Sprintf("Storage IOPS (%s)", ebsType),
			Unit:            "IOPS-months",
			UnitMultiplier:  1,
			MonthlyQuantity: &iopsVal,
			ProductFilter: &schema.ProductFilter{
				VendorName:    strPtr("aws"),
				Region:        strPtr(region),
				Service:       strPtr("AmazonES"),
				ProductFamily: strPtr("Elastic Search Volume"),
				AttributeFilters: []*schema.AttributeFilter{
					{Key: "usagetype", ValueRegex: strPtr("/ES:PIOPS/")},
					{Key: "storageMedia", Value: strPtr("PIOPS")},
				},
			},
			PriceFilter: &schema.PriceFilter{
				PurchaseOption: strPtr("on_demand"),
			},
		})
	}

	if dedicatedMasterEnabled {
		costComponents = append(costComponents, &schema.CostComponent{
			Name:           fmt.Sprintf("Dedicated Master Instance (on-demand, %s)", dedicatedMasterType),
			Unit:           "hours",
			UnitMultiplier: 1,
			HourlyQuantity: decimalPtr(decimal.NewFromInt(dedicatedMasterCount)),
			ProductFilter: &schema.ProductFilter{
				VendorName:    strPtr("aws"),
				Region:        strPtr(region),
				Service:       strPtr("AmazonES"),
				ProductFamily: strPtr("Elastic Search Instance"),
				AttributeFilters: []*schema.AttributeFilter{
					{Key: "usagetype", ValueRegex: strPtr("/ESInstance/")},
					{Key: "instanceType", Value: &dedicatedMasterType},
				},
			},
			PriceFilter: &schema.PriceFilter{
				PurchaseOption: strPtr("on_demand"),
			},
		})
	}

	if ultrawarmEnabled {
		costComponents = append(costComponents, &schema.CostComponent{
			Name:           fmt.Sprintf("Ultrawarm Instance (on-demand, %s)", ultrawarmType),
			Unit:           "hours",
			UnitMultiplier: 1,
			HourlyQuantity: decimalPtr(decimal.NewFromInt(ultrawarmCount)),
			ProductFilter: &schema.ProductFilter{
				VendorName:    strPtr("aws"),
				Region:        strPtr(region),
				Service:       strPtr("AmazonES"),
				ProductFamily: strPtr("Elastic Search Instance"),
				AttributeFilters: []*schema.AttributeFilter{
					{Key: "usagetype", ValueRegex: strPtr("/ESInstance/")},
					{Key: "instanceType", Value: &ultrawarmType},
				},
			},
			PriceFilter: &schema.PriceFilter{
				PurchaseOption: strPtr("on_demand"),
			},
		})
	}

	return &schema.Resource{
		Name:           d.Address,
		CostComponents: costComponents,
	}
}
