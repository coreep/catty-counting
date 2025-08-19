package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/log"
	"github.com/google/uuid"
	"github.com/openai/openai-go/v2"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// TODO: memoize
func (chat *Chat) prepareParseFilePrompt() (string, error) {
	preFileStructure := `You are an accounting helper tool that extracts financial data from documents, receipts, invoices and etc. Focus on data that could be useful for accounting and financial analyzis.
Parse receipts and products from the provided file to the exact JSON structure:`
	file := db.File{
		Summary: "a short summary about the file. 50 words max",
		Receipts: []db.Receipt{
			{
				TotalBeforeTax: decimal.NewFromFloat(0.0),
				Tax:            decimal.NewFromFloat(0.0),
				TotalWithTax:   decimal.NewFromFloat(0.0),
				OccuredAt:      func() *time.Time { t := time.Now(); return &t }(),
				Origin:         "store name; address; phone; email; other info about the store from the receipt",
				Recipient:      "last name first name; address; phone; email; other info about the recepient of the receipt",
				Details:        "any other details of what is this receipt about that didn't fit to the other fields",
				Summary:        "a short summary about the receipt. 50 words max",
				Products: []db.Product{
					{
						Title:          "product's title or name",
						Details:        "additional details about a particular product, if any",
						TotalBeforeTax: decimal.NewFromFloat(0.0),
						Tax:            decimal.NewFromFloat(0.0),
						TotalWithTax:   decimal.NewFromFloat(0.0),
						Categories: []db.Category{
							{Title: "Category"},
						},
					},
				},
			},
		},
	}
	fileStructure, err := json.Marshal(file)
	if err != nil {
		return "", fmt.Errorf("marshaling file structure: %w", err)
	}

	explanation := `Values of the fields are for reference. If you can't parse a value for a field - omit it.
Assign 1-4 categories to each product. Do not create new categories, use this list only:`
	var categories []db.Category
	if err := chat.deps.DBC.Find(categories).Error; err != nil {
		return "", fmt.Errorf("fetching categories: %w", err)
	}
	categoryList, err := json.Marshal(categories)
	if err != nil {
		return "", fmt.Errorf("marshaling categories: %w", err)
	}
	return preFileStructure + string(fileStructure) + explanation + string(categoryList), nil
}

// Handles all files in the message: expose, extract and persist
func (chat *Chat) handleFiles(ctx context.Context, message *db.Message) error {
	if len(message.Files) == 0 {
		chat.deps.Logger.Debug("Message doesn't have files, skipping")
		return nil
	}
	// TODO: we intend to mutate db.Message all the way down to the categories
	// any mutaton to the structure should affect message. Message -> Files -> Receipts -> Products -> Categories
	for _, file := range message.Files {
		logger := chat.deps.Logger.With(log.FILE_ID, file.ID)

		fileURL, err := chat.exposeFile(&file)
		if err != nil {
			return fmt.Errorf("exposing file: %w", err)
		}
		logger.Debug("file exposed. parsing...", "url", fileURL)

		// TODO: retry
		parsedData, err := chat.parseFile(ctx, fileURL, logger)
		if err != nil {
			return fmt.Errorf("parsing file: %w", err)
		}

		if err := chat.addParsedToFile(file, parsedData, logger); err != nil {
			return fmt.Errorf("saving parsed data on file: %w", err)
		}
	}
	var receipts []db.Receipt
	for _, f := range message.Files {
		receipts = append(receipts, f.Receipts...)
	}
	var products []db.Product
	for _, r := range receipts {
		products = append(products, r.Products...)
	}
	categoriesCount := 0
	for _, p := range products {
		categoriesCount += len(p.Categories)
	}
	chat.deps.Logger.
		With("files", len(message.Files)).
		With("receipts", len(receipts)).
		With("products", len(products)).
		With("categories", categoriesCount).
		Debug("Files processed")
	return nil
}

// Make file available for API retrieval. Returns API URL for downloading the file
func (chat *Chat) exposeFile(file *db.File) (string, error) {
	var key string
	if file.ExposedFile != nil && file.ExposedFile.Key != "" {
		key = file.ExposedFile.Key
	} else {
		key = uuid.New().String()
		ef := db.ExposedFile{FileID: file.ID, Key: key}
		if err := chat.deps.DBC.Create(&ef).Error; err != nil {
			return "", fmt.Errorf("creating exposed file: %w", errors.WithStack(err))
		}
		file.ExposedFile = &ef
	}
	return fmt.Sprintf("%s/%s", config.Host(), key), nil
}

// Use OpenAI to extract structured JSON from the file URL
func (chat *Chat) parseFile(ctx context.Context, fileURL string, logger *slog.Logger) (unpersisted db.File, err error) {
	logger.Debug("Sending file for parsing")
	var parsedFile db.File

	parseFilePrompt, err := chat.prepareParseFilePrompt()
	if err != nil {
		return parsedFile, fmt.Errorf("preparing parse file prompt: %w", err)
	}

	params := openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT5,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(parseFilePrompt),
			openai.UserMessage(
				[]openai.ChatCompletionContentPartUnionParam{
					openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{URL: fileURL, Detail: "auto"}),
				},
			),
		},
	}

	// TODO: preserve request/response from OpenAI as db.Message
	// Use MessageDirectionSystemToLlm / MessageDirectionLlmToSystem
	resp, err := chat.oClient.Chat.Completions.New(ctx, params)
	if err != nil {
		return parsedFile, fmt.Errorf("extraction call failed: %w", errors.WithStack(err))
	}
	logger.Debug("Parsing request complete")

	assistantText := ""
	if len(resp.Choices) > 0 && resp.Choices[0].Message.Content != "" {
		assistantText = resp.Choices[0].Message.Content
	}
	if assistantText == "" {
		return parsedFile, errors.New("empty extraction response")
	}

	if err := json.Unmarshal([]byte(assistantText), &parsedFile); err != nil {
		return parsedFile, fmt.Errorf("unmarshal parsed data: %w", errors.WithStack(err))
	}
	logger.Debug("File parsed ok")

	return parsedFile, nil
}

// Writes receipts/products/categories to DB and returns created receipts count
func (chat *Chat) addParsedToFile(file db.File, parsedFile db.File, logger *slog.Logger) error {
	file.Summary = parsedFile.Summary
	if err := chat.deps.DBC.Save(file).Error; err != nil {
		// TODO: handle
		logger.With(log.ERROR, errors.WithStack(err)).Error("failed to update file's summary")
	}

	for _, r := range parsedFile.Receipts {
		receipt := db.Receipt{
			FileID:         file.ID,
			TotalBeforeTax: r.TotalBeforeTax,
			Tax:            r.Tax,
			TotalWithTax:   r.TotalWithTax,
			Origin:         r.Origin,
			Recipient:      r.Recipient,
			Details:        r.Details,
			Summary:        r.Summary,
		}
		if err := chat.deps.DBC.Create(&receipt).Error; err != nil {
			return fmt.Errorf("create receipt: %w", errors.WithStack(err))
		}
		lgr := logger.With(log.RECEIPT_ID, receipt.ID)
		lgr.Debug("Receipt created")

		for _, p := range r.Products {
			product := db.Product{
				ReceiptID:      receipt.ID,
				Title:          p.Title,
				Details:        p.Details,
				TotalBeforeTax: p.TotalBeforeTax,
				Tax:            p.Tax,
				TotalWithTax:   p.TotalWithTax,
			}
			if err := chat.deps.DBC.Create(&product).Error; err != nil {
				lgr.With(log.ERROR, errors.WithStack(err)).Error("failed to create product")
				// TODO: handle
				continue
			}
			lgr = lgr.With(log.RECEIPT_ID, receipt.ID).With(log.PRODUCT_ID, product.ID)
			lgr.Debug("Product created")

			for _, c := range p.Categories {
				var category db.Category
				if err := chat.deps.DBC.Where("title = ?", c.Title).First(&category).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						lgr.With("category", c.Title).Warn("no such category")
					} else {
						lgr.With(log.ERROR, errors.WithStack(err)).Error("failed to find category")
					}
					// TODO: handle
					continue
				}
				lgr := lgr.With(log.CATEGORY_ID, category.ID)
				lgr.Debug("Category found")

				if err := chat.deps.DBC.Model(&product).Association("Categories").Append(&category); err != nil {
					// TODO: handle
					lgr.With(log.ERROR, errors.WithStack(err)).Error("failed to append category to product")
				}
				lgr.Debug("Category attached")
			}
		}
	}
	return nil
}
