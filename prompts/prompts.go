package prompts

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/llm"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	ASSISTANT_INSTRUCTIONS = `You are an accounting helping assistant, which is capable of processing docs, receipts, building statistics and giving advices. Receiving a message from user, you should analyze if you have necessary details in your context to provide good answer. In case you need any more data about the user - ask user about it. You also have access to tools to retrieve stored information about the user and past interations. Keep your answers reasonably short. Don't propose to do something that you don't have tools to do.`
	SUMMARIZE_FILE         = `Confirm with a short symmary what files and receipts you have received. 10 words per file max.`
)

var PROMPT_PARSE_FILE = ""

func Init(dbc *gorm.DB) error {
	preFileStructure := `You are an accounting helper tool that extracts financial data from documents, receipts, invoices and etc. Focus on data useful for accounting and financial analyzis.
Parse data from the provided file to the exact JSON structure:`
	file := llm.File4Llm{
		Summary: "a short summary about the file. 50 words max",
		Receipts: []llm.Receipt4Llm{
			{
				OccuredAt:      time.Now(),
				Origin:         "store name; address; phone; email; other info about the store from the receipt",
				Recipient:      "last name first name; address; phone; email; other info about the recepient of the receipt",
				Currency:       "3 letter currency code of the receipt",
				TotalBeforeTax: decimal.NewFromFloat(0.0),
				Tax:            decimal.NewFromFloat(0.0),
				TotalWithTax:   decimal.NewFromFloat(0.0),
				Details:        "any other details of what is this receipt about that didn't fit to the other fields",
				Summary:        "a short summary about the receipt. 50 words max",
				Products: []llm.Product4Llm{
					{
						Title:          "product's title or name",
						Details:        "additional details about a particular product, if any",
						TotalBeforeTax: decimal.NewFromFloat(0.0),
						Tax:            decimal.NewFromFloat(0.0),
						TotalWithTax:   decimal.NewFromFloat(0.0),
						Categories: []llm.Category4Llm{
							{Title: "Category"},
						},
					},
				},
			},
		},
	}
	fileStructure, err := json.Marshal(file)
	if err != nil {
		return fmt.Errorf("marshaling file structure: %w", err)
	}

	explanation := `Values of the fields are for reference. If you can't parse a value for a field - omit it.
Assign 1-4 categories to each product. Do not create new categories, use this list only:`
	var categories []db.Category
	if err := dbc.Find(&categories).Error; err != nil {
		return fmt.Errorf("fetching categories: %w", err)
	}
	parsedCategories := lo.Map(categories, func(category db.Category, _ int) llm.Category4Llm {
		return llm.Category4Llm{Title: category.Title, Details: category.Details}
	})
	categoryList, err := json.Marshal(parsedCategories)
	if err != nil {
		return fmt.Errorf("marshaling categories: %w", err)
	}
	PROMPT_PARSE_FILE = preFileStructure + string(fileStructure) + explanation + string(categoryList)
	return nil
}
