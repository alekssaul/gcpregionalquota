package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	serviceusage "cloud.google.com/go/serviceusage/apiv1"
	"github.com/urfave/cli/v2"
	"google.golang.org/api/iterator"
	serviceusagepb "google.golang.org/genproto/googleapis/api/serviceusage/v1"
)

func main() {
	var projectName string
	var sourceRegion string
	var targetRegion string

	app := &cli.App{
		Name:        "GCP Regional Quota lookup",
		Usage:       " ",
		UsageText:   "gcpregionalquota --projectid project-123 --sourceregion us-east1 --targetregion us-central1",
		Description: "Small CLI tool to compare GCP region quotas",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "projectid",
				Usage:    "GCP Project ID (required)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "sourceregion",
				Usage:    "GCP Source Region (required)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "targetregion",
				Usage:    "GCP Target Region (required)",
				Required: true,
			},
		},
		Action: func(cCtx *cli.Context) error {
			if cCtx.String("projectid") != "" && cCtx.String("sourceregion") != "" && cCtx.String("targetregion") != "" {
				projectName = cCtx.String("projectid")
				sourceRegion = cCtx.String("sourceregion")
				targetRegion = cCtx.String("targetregion")
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

	serviceList, err := ListServices(projectName)
	if err != nil {
		fmt.Printf("Error getting services in: %s : %e", projectName, err)
	}

	// Itirate through ServiceList and call GetService for individual services
	for i := 0; i < len(serviceList); i++ {
		GetService(serviceList[i], sourceRegion, targetRegion)
	}
}

func GetService(serviceName string, region string, targetregion string) (err error) {
	ctx := context.Background()
	c, err := serviceusage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer c.Close()

	req := &serviceusagepb.GetServiceRequest{
		Name: serviceName,
	}
	service, err := c.GetService(ctx, req)
	if err != nil {
		return err
	}

	for i := 0; i < len(service.Config.Quota.Limits); i++ {
		ql := service.Config.Quota.Limits[i]

		// Check For zonal quota
		var sourcezonequota string
		var targetzonequota string
		zonalquota := false
		for key, value := range ql.Values {
			// Zonal Quota
			if strings.HasPrefix(key, "DEFAULT/"+region+"-") {
				sourcezonequota = sourcezonequota + strings.Trim(key, "DEFAULT/") + "=" + fmt.Sprintf("%v", value) + ","
				zonalquota = true
			}

			// Zonal Quota
			if strings.HasPrefix(key, "DEFAULT/"+targetregion+"-") {
				targetzonequota = targetzonequota + strings.Trim(key, "DEFAULT/") + "=" + fmt.Sprintf("%v", value) + ","
				zonalquota = true
			}

		}
		if zonalquota {
			fmt.Printf("Service Name: %v\n", service.Config.Name)
			fmt.Printf("Service Title: %v\n", service.Config.Title)
			fmt.Printf("Quota Name: %s\n", ql.Name)
			fmt.Printf("Quota Description: %s\n", ql.DisplayName)
			fmt.Printf("Region %s Zonal Quota: %v\n", region, sourcezonequota)
			fmt.Printf("Region %s Zonal Quota: %v\n\n", targetregion, targetzonequota)
			zonalquota = false
		}

		sourceregionquota := ql.Values["DEFAULT/"+region]
		targetregionquota := ql.Values["DEFAULT/"+targetregion]

		// There is a source region specific quota and it's not equal to target region quota
		// There is a targetregionquota, which is not same with source region and also not same as default regional value
		if (sourceregionquota != 0 && sourceregionquota != targetregionquota) ||
			(targetregionquota != 0 && sourceregionquota != targetregionquota && targetregionquota != ql.Values["DEFAULT"]) {
			fmt.Printf("Service Name: %v\n", service.Config.Name)
			fmt.Printf("Service Title: %v\n", service.Config.Title)
			fmt.Printf("Quota Name: %s\n", ql.Name)
			fmt.Printf("Quota Description: %s\n", ql.DisplayName)

			if sourceregionquota > ql.Values["DEFAULT"] {
				fmt.Printf("Region %s Quota: %v\n", region, sourceregionquota)
			} else {
				fmt.Printf("Region %s Quota: %v\n", region, ql.Values["DEFAULT"])
			}

			fmt.Printf("Region %s Quota: %v\n\n", targetregion, targetregionquota)
		}
	}

	return nil
}

// ListServices - Lists all the services within a project and returns a string array
func ListServices(projectName string) (enabledServices []string, e error) {
	ctx := context.Background()
	c, err := serviceusage.NewClient(ctx)
	if err != nil {
		return enabledServices, err
	}
	defer c.Close()

	req := &serviceusagepb.ListServicesRequest{
		Parent: "projects/" + projectName,
		Filter: "state:ENABLED",
	}

	it := c.ListServices(ctx, req)

	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return enabledServices, err
		}
		enabledServices = append(enabledServices, resp.Name)

	}
	return enabledServices, nil
}
