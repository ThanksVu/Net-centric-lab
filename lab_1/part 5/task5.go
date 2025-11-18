package main

import (
	"fmt"
)

type Property struct {
	Name     string
	Price    float64
	Area     float64
	Bedrooms int
	District string
}

func viewAllProperties(properties []Property) {
	fmt.Println("\n=== All Properties ===")
	for _, prop := range properties {
		pricePerM2 := prop.Price / prop.Area
		fmt.Printf("%s: %.0f VND (%.0f VND/m²)\n", prop.Name, prop.Price, pricePerM2)
	}
}

func findPropertiesInBudget(properties []Property, maxBudget float64) []Property {
	var result []Property
	for _, prop := range properties {
		if prop.Price <= maxBudget {
			result = append(result, prop)
		}
	}
	return result
}

func investmentAnalysis(properties []Property, rents []float64) {
	fmt.Println("\n=== Investment Analysis ===")
	for i, prop := range properties {
		roi := (rents[i] * 12 / prop.Price) * 100
		fmt.Printf("%s: ROI %.2f%%\n", prop.Name, roi)
	}
}

func loanCalculator(properties []Property, downPaymentPercent float64, interestRate float64, years int) {
	fmt.Println("\n=== Loan Analysis ===")
	for _, prop := range properties {
		loanAmount := prop.Price * (1 - downPaymentPercent/100)
		monthlyPayment := calculateMonthlyPayment(loanAmount, interestRate, years)
		totalInterest := (monthlyPayment * float64(years*12)) - loanAmount
		fmt.Printf("%s: Loan Amount: %.0f VND, Monthly Payment: %.0f VND, Total Interest: %.0f VND\n",
			prop.Name, loanAmount, monthlyPayment, totalInterest)
	}
}

func calculateMonthlyPayment(loanAmount, annualRate float64, years int) float64 {
	monthlyRate := annualRate / 100 / 12
	numPayments := float64(years * 12)
	return loanAmount * monthlyRate * (1 + monthlyRate) * numPayments / (1 + monthlyRate)
}

func main() {
	properties := []Property{
		{"Saigon Apartment", 2500000000, 75.5, 2, "District 1"},
		{"HCMC House", 4200000000, 120.0, 3, "District 7"},
		{"Budget Studio", 800000000, 35.0, 1, "Binh Thanh"},
	}

	for {
		fmt.Println("\n=== Property Analyzer Menu ===")
		fmt.Println("1. View all properties")
		fmt.Println("2. Search by budget")
		fmt.Println("3. Investment analysis")
		fmt.Println("4. Loan calculator")
		fmt.Println("5. Get recommendations")
		fmt.Println("6. Optimize portfolio")
		fmt.Println("0. Exit")
		fmt.Print("Choose option: ")

		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			viewAllProperties(properties)
		case 2:
			var budget float64
			fmt.Print("Enter your budget: ")
			fmt.Scanln(&budget)
			affordableProperties := findPropertiesInBudget(properties, budget)
			fmt.Printf("\nProperties under %.0f VND:\n", budget)
			viewAllProperties(affordableProperties)
		case 3:
			rents := []float64{25000000, 35000000, 12000000}
			investmentAnalysis(properties, rents)
		case 4:
			loanCalculator(properties, 20, 8.5, 20)
		case 5:
			// Gọi hàm gợi ý
			// Giả sử bạn đã có hàm recommendProperty
		case 6:
			// Tối ưu danh mục đầu tư
			// Giả sử bạn đã có hàm optimizePortfolio
		case 0:
			fmt.Println("Goodbye!")
			return
		default:
			fmt.Println("Invalid option!")
		}
	}
}
