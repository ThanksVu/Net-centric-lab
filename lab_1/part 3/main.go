package main

import (
	"fmt"
	"math"
)

type Property struct {
	Name  string
	Price float64
	Area  float64
}

func (p Property) CalculateROI(monthlyRent float64) float64 {
	return (monthlyRent * 12 / p.Price) * 100
}

func (p Property) InvestmentGrade() string {
	roi := p.CalculateROI(0)
	switch {
	case roi > 8:
		return "EXCELLENT"
	case roi >= 5:
		return "GOOD"
	case roi >= 3:
		return "FAIR"
	default:
		return "POOR"
	}
}

func (p Property) PricePerM2() float64 {
	if p.Area == 0 {
		return 0
	}
	return p.Price / p.Area
}

func (p Property) IsAffordable(budget float64) bool {
	return p.Price <= budget
}

type LoanInfo struct {
	LoanAmount     float64
	MonthlyPayment float64
	TotalInterest  float64
}

func (p Property) CalculateLoan(downPaymentPercent, interestRate float64, years int) LoanInfo {
	loanAmount := p.Price * (1 - downPaymentPercent/100)
	monthlyRate := interestRate / 100 / 12
	numPayments := float64(years * 12)

	if interestRate == 0 {
		return LoanInfo{
			LoanAmount:     loanAmount,
			MonthlyPayment: loanAmount / numPayments,
			TotalInterest:  0,
		}
	}

	monthlyPayment := loanAmount * monthlyRate * math.Pow(1+monthlyRate, numPayments) / (math.Pow(1+monthlyRate, numPayments) - 1)
	totalInterest := monthlyPayment*numPayments - loanAmount

	return LoanInfo{
		LoanAmount:     loanAmount,
		MonthlyPayment: monthlyPayment,
		TotalInterest:  totalInterest,
	}
}

func main() {
	properties := []Property{
		{"Saigon Apartment", 2500000000, 100},
		{"HCMC House", 3500000000, 150},
		{"Budget Studio", 1200000000, 50},
	}

	monthlyRent := 25000000.0
	for _, prop := range properties {
		fmt.Printf("%s: ROI %.2f%% per year - %s\n", prop.Name, prop.CalculateROI(monthlyRent), prop.InvestmentGrade())
	}

	fmt.Printf("\n=== Loan Analysis ===\n")
	for _, prop := range properties {
		loanInfo := prop.CalculateLoan(20, 8.5, 20)
		fmt.Printf("%s:\n", prop.Name)
		fmt.Printf("  Loan Amount: %.2f VND (80%% of price)\n", loanInfo.LoanAmount)
		fmt.Printf("  Monthly Payment: %.2f triệu VND\n", loanInfo.MonthlyPayment/1000000)
		fmt.Printf("  Total Interest: %.2f tỷ VND over 20 years\n", loanInfo.TotalInterest/1000000000)
	}
}
