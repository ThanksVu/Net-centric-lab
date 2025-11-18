package main

import (
    "fmt"
    "strings"
    "math"
)

type Property struct {
    Name     string
    District string
    Size     float64 // in m²
    Price    float64 // total price in VND
}

// --- Mocked helper methods for this challenge ---
func (p Property) IsAffordable(budget float64) bool {
    return p.Price <= budget
}

type Loan struct {
    MonthlyPayment float64
}

func (p Property) CalculateLoan(downPaymentPercent, rate float64, years int) Loan {
    loanAmount := p.Price * (1 - downPaymentPercent/100)
    monthlyRate := rate / 100 / 12
    months := float64(years * 12)
    payment := loanAmount * (monthlyRate * math.Pow(1+monthlyRate, months)) / (math.Pow(1+monthlyRate, months) - 1)
    return Loan{MonthlyPayment: payment}
}

func (p Property) CalculateROI(monthlyRent float64) float64 {
    return (monthlyRent * 12 / p.Price) * 100
}

// --- SMART RECOMMENDER FUNCTION ---
func smartRecommendProperty(p Property, budget, maxMonthlyPayment float64) (string, string) {
    // --- Basic affordability check ---
    if !p.IsAffordable(budget) {
        return "❌ SKIP - Over budget", "Property exceeds your total budget."
    }

    // --- Loan check (same as base function) ---
    loanInfo := p.CalculateLoan(20, 8.5, 20)
    if loanInfo.MonthlyPayment > maxMonthlyPayment {
        return "⚠️ CONSIDER - High monthly payment", fmt.Sprintf("Monthly payment: %.0f VND exceeds your max (%.0f)", loanInfo.MonthlyPayment, maxMonthlyPayment)
    }

    // --- ROI check (same as base function) ---
    roi := p.CalculateROI(p.Price * 0.012) // Assume 1.2% monthly rent
    baseRecommendation := "🤔 MAYBE - Average investment"
    if roi > 10 {
        baseRecommendation = "🔥 BUY NOW - Excellent ROI"
    } else if roi > 6 {
        baseRecommendation = "✅ GOOD BUY - Solid investment"
    }

    // --- Extra Smart Criteria ---
    var bonus []string
    var warnings []string

    // (1) Location premium
    district := strings.ToLower(p.District)
    if district == "district 1" || district == "district 2" || district == "district 7" {
        bonus = append(bonus, "Premium location")
    }

    // (2) Size efficiency
    if p.Size >= 50 && p.Size <= 100 {
        bonus = append(bonus, "Optimal size (50–100m²)")
    } else if p.Size < 50 {
        warnings = append(warnings, "Too small (<50m²)")
    } else if p.Size > 100 {
        warnings = append(warnings, "Too large (>100m²)")
    }

    // (3) Price per m² reasonableness
    pricePerM2 := p.Price / p.Size
    if pricePerM2 > 60000000 {
        warnings = append(warnings, fmt.Sprintf("High price per m² (%.1fM VND)", pricePerM2/1_000_000))
    }

    // --- Adjust overall recommendation ---
    score := len(bonus) - len(warnings)
    finalRecommendation := baseRecommendation
    if score >= 2 && strings.HasPrefix(baseRecommendation, "✅") {
        finalRecommendation = "🏆 TOP PICK - Strong overall value"
    } else if score < 0 {
        finalRecommendation = "⚠️ BE CAUTIOUS - Too many downsides"
    }

    // --- Compose detailed message ---
    details := fmt.Sprintf("Bonus: %v | Warnings: %v", bonus, warnings)
    return finalRecommendation, details
}

// --- Example test ---
func main() {
    p := Property{
        Name:     "Luxury Apartment D7",
        District: "District 7",
        Size:     85,
        Price:    4800000000, // 4.8B VND
    }

    rec, details := smartRecommendProperty(p, 5000000000, 30000000)
    fmt.Println("🏠 Property:", p.Name)
    fmt.Println("📊 Recommendation:", rec)
    fmt.Println("📋 Details:", details)
}
