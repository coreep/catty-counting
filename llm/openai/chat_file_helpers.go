package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/llm"
	"github.com/EPecherkin/catty-counting/log"
	"github.com/EPecherkin/catty-counting/prompts"
	"github.com/google/uuid"
	"github.com/openai/openai-go/v2"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// Handles all files in the message: expose, extract and persist
func (chat *Chat) handleFiles(ctx context.Context, message *db.Message) error {
	if len(message.Files) == 0 {
		chat.deps.Logger.Debug("Message doesn't have files, skipping")
		return nil
	}
	chat.deps.Logger.With("files", len(message.Files)).Debug("handling files")
	for i, file := range message.Files {
		logger := chat.deps.Logger.With(log.FILE_ID, file.ID)

		fileURL, err := chat.exposeFile(&file)
		if err != nil {
			return fmt.Errorf("exposing file: %w", err)
		}
		logger.Debug("file exposed", "url", fileURL)

		// TODO: retry
		parsedData, err := chat.parseFile(ctx, fileURL, logger)
		if err != nil {
			return fmt.Errorf("parsing file: %w", err)
		}

		if err := chat.updateFileWithParsed(&file, parsedData, logger); err != nil {
			return fmt.Errorf("saving parsed data on file: %w", err)
		}
		message.Files[i] = file
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
	return fmt.Sprintf("%s/api/file/%s", config.Host(), key), nil
}

// Use OpenAI to extract structured JSON from the file URL.
func (chat *Chat) parseFile(ctx context.Context, fileURL string, logger *slog.Logger) (llm.File4Llm, error) {
	logger.Debug("sending file for parsing")
	var parsedFile llm.File4Llm

	params := openai.ChatCompletionNewParams{
		Model: VISION_MODEL,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(
				[]openai.ChatCompletionContentPartUnionParam{
					openai.TextContentPart(prompts.PROMPT_PARSE_FILE),
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
	assistantText := ""
	if len(resp.Choices) > 0 && resp.Choices[0].Message.Content != "" {
		assistantText = resp.Choices[0].Message.Content
	} else {
		return parsedFile, errors.New("empty extraction response")
	}
	logger.With("response", assistantText).Debug("Parsing request complete")

	if err := json.Unmarshal([]byte(assistantText), &parsedFile); err != nil {
		return parsedFile, fmt.Errorf("unmarshal parsed data: %w", errors.WithStack(err))
	}
	logger.Debug("File parsed ok")

	return parsedFile, nil
}

// Writes receipts/products/categories to DB and returns created receipts count
func (chat *Chat) updateFileWithParsed(file *db.File, parsedFile llm.File4Llm, logger *slog.Logger) error {
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
		iterLogger := logger.With(log.RECEIPT_ID, receipt.ID)
		iterLogger.Debug("Receipt created")

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
				iterLogger.With(log.ERROR, errors.WithStack(err)).Error("failed to create product")
				// TODO: handle
				continue
			}
			iterLogger = iterLogger.With(log.RECEIPT_ID, receipt.ID).With(log.PRODUCT_ID, product.ID)
			iterLogger.Debug("Product created")

			for _, c := range p.Categories {
				var category db.Category
				if err := chat.deps.DBC.Where("title = ?", c.Title).First(&category).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						iterLogger.With("category", c.Title).Warn("no such category")
					} else {
						iterLogger.With(log.ERROR, errors.WithStack(err)).Error("failed to find category")
					}
					// TODO: handle
					continue
				}
				lgr := iterLogger.With(log.CATEGORY_ID, category.ID)
				lgr.Debug("Category found")

				if err := chat.deps.DBC.Model(&product).Association("Categories").Append(&category); err != nil {
					// TODO: handle
					lgr.With(log.ERROR, errors.WithStack(err)).Error("failed to append category to product")
				}
				lgr.Debug("Category attached")
			}
			receipt.Products = append(receipt.Products, product)
		}
		file.Receipts = append(file.Receipts, receipt)
	}
	return nil
}
