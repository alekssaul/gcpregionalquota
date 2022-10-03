package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	serviceusage "cloud.google.com/go/serviceusage/apiv1"
	"github.com/urfave/cli/v2"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iterator"
	serviceusagepb "google.golang.org/genproto/googleapis/api/serviceusage/v1"
)

func main() {
	var projectID string
	var sourceRegion string
	var targetRegion string
	var authtoken string
	var projectnumber string

	app := &cli.App{
		Name:        "GCP Regional Quota lookup",
		Usage:       " ",
		UsageText:   "gcpregionalquota --projectid project-123 --sourceregion us-east1 --targetregion us-central1 --token $(gcloud auth print-access-token)",
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
			&cli.StringFlag{
				Name:     "token",
				Usage:    "GCP auth token (required)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "projectnumber",
				Usage:    "GCP Project Number (optional)",
				Required: false,
			},
		},
		Action: func(cCtx *cli.Context) error {
			if cCtx.String("projectid") != "" && cCtx.String("sourceregion") != "" && cCtx.String("targetregion") != "" && cCtx.String("token") != "" {
				projectID = cCtx.String("projectid")
				sourceRegion = cCtx.String("sourceregion")
				targetRegion = cCtx.String("targetregion")
				authtoken = cCtx.String("token")
				projectnumber = cCtx.String("projectnumber")
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

	// Get Project Number from ProjectID via gcloud sdk if Project Number is not specified
	if projectnumber == "" {
		p, err := GetProjectNumber(projectID)
		if err != nil {
			fmt.Printf("Error getting Project Number for: %s : %s", projectID, err.Error())
		}
		projectnumber = fmt.Sprintf("%v", p)
	}

	// Get a List of Services enabled in the Project
	serviceList, err := ListServices(projectID)
	if err != nil {
		fmt.Printf("Error getting services in: %s : %s", projectID, err.Error())
	}

	// Enumurate the Lists and execute the main region check function
	for i := 0; i < len(serviceList); i++ {
		GetconsumerQuotaMetrics(serviceList[i], authtoken, projectnumber, sourceRegion, targetRegion)
	}

}

func GetProjectNumber(projectid string) (projectnumber int64, err error) {
	cloudresourcemanagerService, err := cloudresourcemanager.NewService(context.Background())
	if err != nil {
		return 0, err
	}

	project, err := cloudresourcemanagerService.Projects.Get(projectid).Do()
	if err != nil {
		return 0, err
	}
	return project.ProjectNumber, nil
}

// GetconsumerQuotaMetrics
func GetconsumerQuotaMetrics(service string, authtoken string, projectnumber string, sourceRegion string, targetRegion string) (err error) {
	// Hand construct the http call as needed data seem to be only part of v1beta1 version of serviceusage API
	endpoint := "https://serviceusage.googleapis.com/v1beta1/projects/" + projectnumber + "/services/" + service + "/consumerQuotaMetrics"
	request, err := http.NewRequest("GET", endpoint, nil)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+authtoken)

	client := &http.Client{}
	response, error := client.Do(request)
	if error != nil {
		panic(error)
	}
	defer response.Body.Close()

	// Unmarshal Response
	var cqm consumerQuotaMetrics
	if response.StatusCode == http.StatusOK {
		err := json.NewDecoder(response.Body).Decode(&cqm)
		if err != nil {
			fmt.Printf("%T\n%s\n%#v\n", err, err, err)
		}

		CheckLimitDifferences(cqm)
		CompareRegions(cqm, sourceRegion, targetRegion)

	}

	return nil
}

func CompareRegions(cqm consumerQuotaMetrics, sourceRegion string, targetRegion string) {
	for i := 0; i < len(cqm.Metrics); i++ {
		for q := 0; q < len(cqm.Metrics[i].ConsumerQuotaLimits); q++ {
			sourceRegionLimit := ""
			targetRegionLimit := ""
			for b := 0; b < len(cqm.Metrics[i].ConsumerQuotaLimits[q].QuotaBuckets); b++ {
				displayname := cqm.Metrics[i].DisplayName
				metric := cqm.Metrics[i].Metric
				region := cqm.Metrics[i].ConsumerQuotaLimits[q].QuotaBuckets[b].Dimensions.Region
				if region == sourceRegion {
					sourceRegionLimit = cqm.Metrics[i].ConsumerQuotaLimits[q].QuotaBuckets[b].EffectiveLimit
				}
				if region == targetRegion {
					targetRegionLimit = cqm.Metrics[i].ConsumerQuotaLimits[q].QuotaBuckets[b].EffectiveLimit
				}
				if targetRegionLimit != "" && sourceRegionLimit != "" && targetRegionLimit < sourceRegionLimit {
					fmt.Printf("Metric: %s\ndisplayname: %s\n%s Region Limit: %s\n%s Region Limit: %s\n\n", metric, displayname, sourceRegion, sourceRegionLimit, targetRegion, targetRegionLimit)
					sourceRegionLimit = ""
					targetRegionLimit = ""
				}
			}

		}

	}

}

// CheckLimitDifferences - Checks to see if any limits in region has been changed
func CheckLimitDifferences(cqm consumerQuotaMetrics) {
	for i := 0; i < len(cqm.Metrics); i++ {
		for q := 0; q < len(cqm.Metrics[i].ConsumerQuotaLimits); q++ {
			for b := 0; b < len(cqm.Metrics[i].ConsumerQuotaLimits[q].QuotaBuckets); b++ {
				effectiveLimit := cqm.Metrics[i].ConsumerQuotaLimits[q].QuotaBuckets[b].EffectiveLimit
				defaultLimit := cqm.Metrics[i].ConsumerQuotaLimits[q].QuotaBuckets[b].DefaultLimit
				dimensions := cqm.Metrics[i].ConsumerQuotaLimits[q].QuotaBuckets[b].Dimensions
				displayname := cqm.Metrics[i].DisplayName
				metric := cqm.Metrics[i].Metric
				if effectiveLimit != defaultLimit {
					if dimensions.Zone != "" {
						fmt.Printf("Metric:%s\ndisplayname:%s\ndefaultLimit:%s\neffectiveLimit:%s\ndimension:%s\n\n", metric, displayname, defaultLimit, effectiveLimit, dimensions.Zone)
					} else {
						fmt.Printf("Metric:%s\ndisplayname:%s\ndefaultLimit:%s\neffectiveLimit:%s\ndimension:%s\n\n", metric, displayname, defaultLimit, effectiveLimit, dimensions.Region)
					}
				}
			}
		}
	}
}

// ListServices - Lists all the services within a project and returns a string array
func ListServices(projectID string) (enabledServices []string, e error) {
	ctx := context.Background()
	c, err := serviceusage.NewClient(ctx)
	if err != nil {
		return enabledServices, err
	}
	defer c.Close()

	req := &serviceusagepb.ListServicesRequest{
		Parent: "projects/" + projectID,
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
		enabledServices = append(enabledServices, resp.Config.Name)

	}
	return enabledServices, nil
}
