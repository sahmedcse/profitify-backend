package domain

// Canonical sector buckets. These are intentionally coarse so the dashboard
// sidebar can show a single 3-4 character badge per ticker.
const (
	SectorTechnology             = "Technology"
	SectorFinancial              = "Financial"
	SectorHealthcare             = "Healthcare"
	SectorConsumerCyclical       = "Consumer Cyclical"
	SectorConsumerDefensive      = "Consumer Defensive"
	SectorIndustrials            = "Industrials"
	SectorEnergy                 = "Energy"
	SectorUtilities              = "Utilities"
	SectorRealEstate             = "Real Estate"
	SectorBasicMaterials         = "Basic Materials"
	SectorCommunicationServices  = "Communication Services"
	SectorUnknown                = "Unknown"
)

// SICToSector maps a SIC (Standard Industrial Classification) code to a
// canonical sector bucket. Returns SectorUnknown for empty or unrecognized
// codes. The mapping uses the first two digits (major group) which is the
// broadest grouping in the SIC hierarchy.
//
// Reference: https://www.osha.gov/data/sic-manual
func SICToSector(code string) string {
	if len(code) < 2 {
		return SectorUnknown
	}
	prefix := code[:2]
	switch prefix {
	// Agriculture, Forestry, Fishing
	case "01", "02", "07", "08", "09":
		return SectorBasicMaterials
	// Mining
	case "10", "12", "14":
		return SectorBasicMaterials
	case "13": // Oil & gas extraction
		return SectorEnergy
	// Construction
	case "15", "16", "17":
		return SectorIndustrials
	// Manufacturing 20-39
	case "20", "21": // Food, tobacco
		return SectorConsumerDefensive
	case "22", "23": // Textiles, apparel
		return SectorConsumerCyclical
	case "24", "25": // Lumber, furniture
		return SectorConsumerCyclical
	case "26": // Paper
		return SectorBasicMaterials
	case "27": // Printing & publishing
		return SectorCommunicationServices
	case "28": // Chemicals / pharmaceuticals
		if len(code) >= 4 && code[:4] == "2834" {
			return SectorHealthcare
		}
		return SectorBasicMaterials
	case "29": // Petroleum refining
		return SectorEnergy
	case "30", "31", "32", "33": // Rubber, leather, stone, primary metals
		return SectorBasicMaterials
	case "34": // Fabricated metals
		return SectorIndustrials
	case "35": // Industrial machinery & computer equipment
		// 3570-3579 is computer & office equipment -> Technology
		if len(code) >= 3 && code[:3] == "357" {
			return SectorTechnology
		}
		return SectorIndustrials
	case "36": // Electronic & electrical equipment
		return SectorTechnology
	case "37": // Transportation equipment
		return SectorConsumerCyclical
	case "38": // Instruments & medical instruments
		return SectorHealthcare
	case "39": // Miscellaneous manufacturing
		return SectorConsumerCyclical

	// Transportation & Communication 40-49
	case "40", "41", "42", "44", "45", "47":
		return SectorIndustrials
	case "46": // Pipelines
		return SectorEnergy
	case "48": // Communications
		return SectorCommunicationServices
	case "49": // Electric, gas, sanitary services
		return SectorUtilities

	// Wholesale & Retail 50-59
	case "50", "51": // Wholesale
		return SectorIndustrials
	case "52", "53", "55", "56", "57", "59": // Retail
		return SectorConsumerCyclical
	case "54": // Food stores
		return SectorConsumerDefensive
	case "58": // Eating & drinking places
		return SectorConsumerCyclical

	// Finance, Insurance, Real Estate 60-67
	case "60", "61", "62", "63", "64", "67":
		return SectorFinancial
	case "65": // Real estate
		return SectorRealEstate

	// Services 70-89
	case "70", "72", "75", "76", "78", "79": // Hotels, personal, auto services, amusement
		return SectorConsumerCyclical
	case "73": // Business services (includes software 7370-7379)
		if len(code) >= 3 && code[:3] == "737" {
			return SectorTechnology
		}
		return SectorIndustrials
	case "80": // Health services
		return SectorHealthcare
	case "81", "82", "83", "84", "86", "87", "89": // Legal, education, social, engineering
		return SectorIndustrials

	// Public Administration 91-99
	case "91", "92", "93", "94", "95", "96", "97", "99":
		return SectorIndustrials
	}
	return SectorUnknown
}
