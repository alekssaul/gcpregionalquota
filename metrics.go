package main

type consumerQuotaMetrics struct {
	Metrics []struct {
		Name                string `json:"name,omitempty"`
		DisplayName         string `json:"displayName,omitempty"`
		ConsumerQuotaLimits []struct {
			Name         string `json:"name,omitempty"`
			Unit         string `json:"unit,omitempty"`
			IsPrecise    bool   `json:"isPrecise,omitempty"`
			Metric       string `json:"metric,omitempty"`
			QuotaBuckets []struct {
				EffectiveLimit string `json:"effectiveLimit,omitempty"`
				DefaultLimit   string `json:"defaultLimit,omitempty"`
				Dimensions     struct {
					Region string `json:"region,omitempty"`
					Zone   string `json:"zone,omitempty"`
				} `json:"dimensions,omitempty"`
			} `json:"quotaBuckets,omitempty"`
			SupportedLocations []string `json:"supportedLocations,omitempty"`
		} `json:"consumerQuotaLimits,omitempty"`
		Metric string `json:"metric,omitempty"`
	} `json:"metrics,omitempty"`
}
