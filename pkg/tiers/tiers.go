// pkg/tiers/tiers.go
package tiers

type ResourceTier struct {
	PVCStorage    string
	MemoryRequest string
	MemoryLimit   string
	CPURequest    string
	CPULimit      string
}

var Tiers = map[string]ResourceTier{
	"free": {
		PVCStorage:    "1Gi",
		MemoryRequest: "1Gi",
		MemoryLimit:   "2Gi",
		CPURequest:    "500m",
		CPULimit:      "1",
	},
	"professional": {
		PVCStorage:    "10Gi",
		MemoryRequest: "2Gi",
		MemoryLimit:   "4Gi",
		CPURequest:    "1",
		CPULimit:      "2",
	},
	"enterprise": {
		PVCStorage:    "50Gi",
		MemoryRequest: "4Gi",
		MemoryLimit:   "8Gi",
		CPURequest:    "2",
		CPULimit:      "4",
	},
}

func GetTier(tierName string) (ResourceTier, bool) {
	tier, exists := Tiers[tierName]
	return tier, exists
}