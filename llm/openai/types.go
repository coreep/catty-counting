package openai

import (
	"time"

	"github.com/EPecherkin/catty-counting/db"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

type File4Llm struct {
	ID       uint          `json:"id,omitempty"`
	Summary  string        `json:"summary"`
	Receipts []Receipt4Llm `json:"receipts"`
}

type Receipt4Llm struct {
	ID             uint            `json:"id,omitempty"`
	TotalBeforeTax decimal.Decimal `json:"total_before_tax"`
	Tax            decimal.Decimal `json:"tax"`
	TotalWithTax   decimal.Decimal `json:"total_with_tax"`
	Currency       string          `json:"currency"`
	Origin         string          `json:"origin"`
	Recipient      string          `json:"recipient"`
	Details        string          `json:"details"`
	Summary        string          `json:"summary"`
	OccuredAt      time.Time       `json:"occured_at"`
	Products       []Product4Llm   `json:"products"`
}

type Product4Llm struct {
	ID             uint            `json:"id,omitempty"`
	Title          string          `json:"title"`
	Details        string          `json:"details"`
	TotalBeforeTax decimal.Decimal `json:"total_before_tax"`
	Tax            decimal.Decimal `json:"tax"`
	TotalWithTax   decimal.Decimal `json:"total_with_tax"`
	Categories     []Category4Llm  `json:"categories"`
}

type Category4Llm struct {
	ID      uint   `json:"id,omitempty"`
	Title   string `json:"title"`
	Details string `json:"details,omitempty"`
}

func dbFileToLlm(file db.File) File4Llm {
	f4l := File4Llm{
		ID:       file.ID,
		Summary:  file.Summary,
		Receipts: lo.Map(file.Receipts, func(receipt db.Receipt, _ int) Receipt4Llm { return dbReceiptToLlm(receipt) }),
	}
	return f4l
}

func dbReceiptToLlm(receipt db.Receipt) Receipt4Llm {
	r4l := Receipt4Llm{
		ID:             receipt.ID,
		TotalBeforeTax: receipt.TotalBeforeTax,
		Tax:            receipt.Tax,
		TotalWithTax:   receipt.TotalWithTax,
		Currency:       receipt.Currency,
		Origin:         receipt.Origin,
		Recipient:      receipt.Recipient,
		Details:        receipt.Details,
		Summary:        receipt.Summary,
		OccuredAt:      receipt.OccuredAt,
		Products:       lo.Map(receipt.Products, func(product db.Product, _ int) Product4Llm { return dbProductToLlm(product) }),
	}
	return r4l
}

func dbProductToLlm(product db.Product) Product4Llm {
	p4l := Product4Llm{
		ID:             product.ID,
		Title:          product.Title,
		Details:        product.Details,
		TotalBeforeTax: product.TotalBeforeTax,
		Tax:            product.Tax,
		TotalWithTax:   product.TotalWithTax,
		Categories:     lo.Map(product.Categories, func(category db.Category, _ int) Category4Llm { return dbCategoryToLlm(category) }),
	}
	return p4l
}

func dbCategoryToLlm(category db.Category) Category4Llm {
	c4l := Category4Llm{
		ID:      category.ID,
		Title:   category.Title,
		Details: category.Details,
	}
	return c4l
}
