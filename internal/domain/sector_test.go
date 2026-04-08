package domain_test

import (
	"testing"

	"github.com/profitify/profitify-backend/internal/domain"
)

func TestSICToSector(t *testing.T) {
	tests := []struct {
		name string
		code string
		want string
	}{
		{"empty", "", domain.SectorUnknown},
		{"too short", "1", domain.SectorUnknown},
		{"unknown prefix", "9999", domain.SectorIndustrials},
		{"truly unknown", "00", domain.SectorUnknown},

		// Technology
		{"prepackaged software 7372", "7372", domain.SectorTechnology},
		{"computer services 7370", "7370", domain.SectorTechnology},
		{"computer equipment 3571", "3571", domain.SectorTechnology},
		{"electronic equipment 3674", "3674", domain.SectorTechnology},

		// Financial
		{"depository banks 6020", "6020", domain.SectorFinancial},
		{"insurance carriers 6311", "6311", domain.SectorFinancial},
		{"holding companies 6770", "6770", domain.SectorFinancial},

		// Healthcare
		{"pharmaceutical preparations 2834", "2834", domain.SectorHealthcare},
		{"health services 8011", "8011", domain.SectorHealthcare},
		{"medical instruments 3841", "3841", domain.SectorHealthcare},

		// Energy
		{"oil & gas extraction 1311", "1311", domain.SectorEnergy},
		{"petroleum refining 2911", "2911", domain.SectorEnergy},
		{"pipelines 4612", "4612", domain.SectorEnergy},

		// Utilities
		{"electric services 4911", "4911", domain.SectorUtilities},

		// Real Estate
		{"real estate operators 6512", "6512", domain.SectorRealEstate},

		// Consumer Defensive
		{"food stores 5411", "5411", domain.SectorConsumerDefensive},
		{"food manufacturing 2000", "2000", domain.SectorConsumerDefensive},
		{"tobacco 2111", "2111", domain.SectorConsumerDefensive},

		// Consumer Cyclical
		{"apparel 2300", "2300", domain.SectorConsumerCyclical},
		{"furniture 2511", "2511", domain.SectorConsumerCyclical},
		{"motor vehicles 3711", "3711", domain.SectorConsumerCyclical},
		{"general merchandise 5311", "5311", domain.SectorConsumerCyclical},
		{"eating places 5812", "5812", domain.SectorConsumerCyclical},
		{"hotels 7011", "7011", domain.SectorConsumerCyclical},

		// Communication Services
		{"telecommunications 4813", "4813", domain.SectorCommunicationServices},
		{"newspapers 2711", "2711", domain.SectorCommunicationServices},

		// Industrials
		{"general construction 1500", "1500", domain.SectorIndustrials},
		{"fabricated metals 3411", "3411", domain.SectorIndustrials},
		{"industrial machinery 3523", "3523", domain.SectorIndustrials},
		{"air transport 4512", "4512", domain.SectorIndustrials},
		{"wholesale durable 5040", "5040", domain.SectorIndustrials},
		{"business services non-software 7311", "7311", domain.SectorIndustrials},

		// Basic Materials
		{"agriculture 0100", "0100", domain.SectorBasicMaterials},
		{"metal mining 1040", "1040", domain.SectorBasicMaterials},
		{"chemicals non-pharma 2812", "2812", domain.SectorBasicMaterials},
		{"primary metals 3312", "3312", domain.SectorBasicMaterials},
		{"paper 2611", "2611", domain.SectorBasicMaterials},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.SICToSector(tt.code)
			if got != tt.want {
				t.Errorf("SICToSector(%q) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}
