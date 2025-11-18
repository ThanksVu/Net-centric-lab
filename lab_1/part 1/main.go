package main

import "fmt"

func categorizeProperty(pricePerM2 float64) string {
	if pricePerM2 > 50000000 {
		return "LUXURY"
	} else if pricePerM2 > 30000000 {
		return "PREMIUM"
	} else if pricePerM2 > 20000000 {
		return "STANDARD"
	}
	return "BUDGET"
}

func formatPrice(price float64) string {
	if price >= 1000000000 {
		return fmt.Sprintf("%.1f tỷ VND", price/1000000000)
	}
	return fmt.Sprintf("%.0f triệu VND", price/1000000)
}

func main() {
	// === Define properties ===
	// Property 1
	property1Name := "Saigon Apartment"
	property1Price := 2500000000.0
	property1Area := 75.5
	property1PricePerM2 := property1Price / property1Area

	// Property 2
	property2Name := "Hanoi Condo"
	property2Price := 2800000000.0
	property2Area := 90.0
	property2PricePerM2 := property2Price / property2Area

	// Property 3
	property3Name := "Danang Villa"
	property3Price := 4000000000.0
	property3Area := 160.0
	property3PricePerM2 := property3Price / property3Area

	// === Categorize properties ===
	cat1 := categorizeProperty(property1PricePerM2)
	cat2 := categorizeProperty(property2PricePerM2)
	cat3 := categorizeProperty(property3PricePerM2)

	// === Count categories ===
	countLuxury := 0
	countPremium := 0
	countStandard := 0
	countBudget := 0

	categories := []string{cat1, cat2, cat3}
	for _, cat := range categories {
		switch cat {
		case "LUXURY":
			countLuxury++
		case "PREMIUM":
			countPremium++
		case "STANDARD":
			countStandard++
		case "BUDGET":
			countBudget++
		}
	}

	// === Output ===
	fmt.Println("=== Property Categories ===")
	fmt.Printf("%s: %s (%s)\n", property1Name, cat1, formatPrice(property1Price))
	fmt.Printf("%s: %s (%s)\n", property2Name, cat2, formatPrice(property2Price))
	fmt.Printf("%s: %s (%s)\n", property3Name, cat3, formatPrice(property3Price))

	fmt.Println("\n=== Category Summary ===")
	fmt.Printf("LUXURY: %d properties\n", countLuxury)
	fmt.Printf("PREMIUM: %d properties\n", countPremium)
	fmt.Printf("STANDARD: %d properties\n", countStandard)
	fmt.Printf("BUDGET: %d properties\n", countBudget)
}
