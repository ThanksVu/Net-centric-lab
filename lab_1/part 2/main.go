package main

import (
	"fmt"
	"sort"
)

// Define the Property struct
type Property struct {
	Name      string
	District  string
	Price     float64 // in VND
	Bedrooms  int
}

// 1️⃣ Function: Find properties within a budget
func findPropertiesInBudget(properties []Property, maxBudget float64) []Property {
	var result []Property

	for _, prop := range properties {
		if prop.Price <= maxBudget {
			result = append(result, prop)
		}
	}

	return result
}

// 2️⃣ Function: Find properties by number of bedrooms
func findPropertiesByBedrooms(properties []Property, bedrooms int) []Property {
	var result []Property

	for _, prop := range properties {
		if prop.Bedrooms == bedrooms {
			result = append(result, prop)
		}
	}

	return result
}

// 3️⃣ Analyze properties by district
func analyzeByDistrict(properties []Property) map[string][]Property {
	districtMap := make(map[string][]Property)

	for _, prop := range properties {
		districtMap[prop.District] = append(districtMap[prop.District], prop)
	}

	return districtMap
}

// 4️⃣ Calculate statistics per district
func calculateDistrictStats(districtMap map[string][]Property) map[string]map[string]interface{} {
	stats := make(map[string]map[string]interface{})

	for district, props := range districtMap {
		totalPrice := 0.0
		mostExpensive := props[0]

		for _, prop := range props {
			totalPrice += prop.Price
			if prop.Price > mostExpensive.Price {
				mostExpensive = prop
			}
		}

		avgPrice := totalPrice / float64(len(props))

		stats[district] = map[string]interface{}{
			"count":         len(props),
			"avgPrice":      avgPrice,
			"mostExpensive": mostExpensive.Name,
		}
	}

	return stats
}

func main() {
	
	// 🏘️ Sample data
	properties := []Property{
		{"Saigon Apartment", "District 1", 2500000000, 2},
		{"HCMC House", "District 7", 4200000000, 4},
		{"Budget Studio", "Binh Thanh", 800000000, 1},
		{"Luxury Villa", "District 7", 6500000000, 5},
	}

	// 🔹 Test 1: Find by Budget
	budget := 3000000000.0 // 3 billion
	affordable := findPropertiesInBudget(properties, budget)
	fmt.Printf("\nProperties under %.0f VND:\n", budget)
	for _, p := range affordable {
		fmt.Printf("- %s (%s): %.0f VND\n", p.Name, p.District, p.Price)
	}

	// 🔹 Test 2: Find by Bedrooms
	bedroomCount := 2
	twoBedroom := findPropertiesByBedrooms(properties, bedroomCount)
	fmt.Printf("\nProperties with %d bedrooms:\n", bedroomCount)
	for _, p := range twoBedroom {
		fmt.Printf("- %s (%s): %.0f VND\n", p.Name, p.District, p.Price)
	}

	// 🔹 Test 3: District Analysis
	fmt.Println("\n=== District Analysis ===")
	districtMap := analyzeByDistrict(properties)
	stats := calculateDistrictStats(districtMap)

	// Display district stats
	for district, info := range stats {
		fmt.Printf("%s: %d properties, Avg: %.1f tỷ VND, Most expensive: %s\n",
			district,
			info["count"].(int),
			info["avgPrice"].(float64)/1e9,
			info["mostExpensive"].(string),
		)
	}

	// 🔹 Sort districts by average price (descending)
	type districtRanking struct {
		name  string
		avg   float64
	}
	var ranking []districtRanking

	for d, info := range stats {
		ranking = append(ranking, districtRanking{d, info["avgPrice"].(float64)})
	}

	sort.Slice(ranking, func(i, j int) bool {
		return ranking[i].avg > ranking[j].avg
	})

	fmt.Println("\nRanking by Average Price:")
	for i, r := range ranking {
		fmt.Printf("%d. %s: %.1f tỷ VND\n", i+1, r.name, r.avg/1e9)
	}
}
