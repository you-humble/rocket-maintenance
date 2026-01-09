package model

import "time"

type Category int32

const (
	CategoryUnknown Category = iota
	CategoryEngine
	CategoryFuel
	CategoryPorthole
	CategoryWing
)

type Part struct {
	// Globally unique identifier of the part.
	ID string
	// Human-readable part name.
	Name string
	// Detailed description of the part.
	Description string
	// Unit price of the part.
	Price float64
	// Quantity of this part currently available in stock.
	StockQuantity int64
	// Category of the part.
	Category Category
	// Physical dimensions and weight of the part.
	Dimensions *Dimensions
	// Manufacturer information for this part.
	Manufacturer *Manufacturer
	// Free-form tags used for quick search and classification.
	Tags []string
	// Flexible key–value metadata associated with the part.
	// Each entry can store a string, integer, double, or boolean value.
	Metadata map[string]any
	// Timestamp when the part was created.
	CreatedAt *time.Time
	// Timestamp when the part was last updated.
	UpdatedAt *time.Time
}

type Dimensions struct {
	// Length in centimeters.
	Length float64
	// Width in centimeters.
	Width float64
	// Height in centimeters.
	Height float64
	// Weight in kilograms.
	Weight float64
}

type Manufacturer struct {
	// Manufacturer name.
	Name string
	// Country of origin of the manufacturer.
	Country string
	// Official website of the manufacturer.
	Website string
}

type PartsFilter struct {
	IDs                   []string
	Names                 []string
	Categories            []Category
	ManufacturerCountries []string
	Tags                  []string
}

func (f PartsFilter) Empty() bool {
	return len(f.IDs) == 0 &&
		len(f.Names) == 0 &&
		len(f.Categories) == 0 &&
		len(f.ManufacturerCountries) == 0 &&
		len(f.Tags) == 0
}

type ListPartsResponse struct {
	Parts []*Part
}
