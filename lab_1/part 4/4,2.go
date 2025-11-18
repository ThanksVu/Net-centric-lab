package main

import (
    "fmt"
    "sort"
)

type Property struct {
    Name     string
    District string
    Size     float64 // m²
    Price    float64 // total VND
    ROI      float64 // %
}

func optimizePortfolio(properties []Property, totalBudget float64) []Property {
    var portfolio []Property
    remainingBudget := totalBudget

    // Sort properties by ROI descending
    sort.Slice(properties, func(i, j int) bool {
        return properties[i].ROI > properties[j].ROI
    })

    // Greedily select properties while budget allows
    for _, prop := range properties {
        if prop.Price <= remainingBudget {
            portfolio = append(portfolio, prop)
            remainingBudget -= prop.Price
        }
    }

    return portfolio
}

func printPortfolio(portfolio []Property, totalBudget float64) {
    fmt.Println("=== Portfolio Optimization ===")
    fmt.Printf("Budget: %.1f tỷ VND\n", totalBudget/1_000_000_000)

    totalInvested := 0.0
    totalROI := 0.0
    for i, p := range portfolio {
        fmt.Printf("%d. %s: %.1f tỷ VND (ROI: %.1f%%)\n",
            i+1, p.Name, p.Price/1_000_000_000, p.ROI)
        totalInvested += p.Price
        totalROI += p.ROI
    }

    remaining := totalBudget - totalInvested
    avgROI := 0.0
    if len(portfolio) > 0 {
        avgROI = totalROI / float64(len(portfolio))
    }

    fmt.Printf("\nTotal Invested: %.1f tỷ VND\n", totalInvested/1_000_000_000)
    fmt.Printf("Remaining Budget: %.1f triệu VND\n", remaining/1_000_000)
    fmt.Printf("Portfolio Average ROI: %.1f%%\n", avgROI)
}

func main() {
    properties := []Property{
        {"Budget Studio", "District 12", 35, 800_000_000, 18.0},
        {"Saigon Apartment", "District 7", 70, 2_500_000_000, 12.0},
        {"HCMC House", "District 3", 90, 4_200_000_000, 10.0},
        {"Luxury Condo", "District 1", 100, 6_000_000_000, 8.0},
    }

    budget := 8_000_000_000.0 // 8 billion VND
    portfolio := optimizePortfolio(properties, budget)
    printPortfolio(portfolio, budget)
}
