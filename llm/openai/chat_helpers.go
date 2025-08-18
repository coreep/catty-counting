package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/google/uuid"
	"github.com/openai/openai-go/v2"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type extractedProduct struct {
	Title          string   `json:"title"`
	Details        string   `json:"details"`
	Summary        string   `json:"summary"`
	TotalBeforeTax string   `json:"totalBeforeTax"`
	Tax            string   `json:"tax"`
	TotalAfterTax  string   `json:"totalAfterTax"`
	Categories     []string `json:"categories"`
}

type extractedReceipt struct {
	TotalBeforeTax string             `json:"totalBeforeTax"`
	Tax            string             `json:"tax"`
	TotalAfterTax  string             `json:"totalAfterTax"`
	Origin         string             `json:"origin"`
	Recipient      string             `json:"recipient"`
	Details        string             `json:"details"`
	Summary        string             `json:"summary"`
	Products       []extractedProduct `json:"products"`
}

type extractionResult struct {
	Receipts []extractedReceipt `json:"receipts"`
}

// processFiles handles all files in the message: ensure exposed, extract and persist
func (chat *Chat) processFiles(ctx context.Context, fullMsg db.Message) ([]db.Receipt, error) {
	totalReceiptsCreated := 0
	for _, file := range fullMsg.Files {
		chat.deps.Logger = chat.deps.Logger.With("file_id", file.ID, "file_blob", file.BlobKey)

		fileURL, err := chat.ensureExposedFile(&file)
		if err != nil {
			// TODO:  return as error
			chat.deps.Logger.With(logger.ERROR, err).Error("problems exposing the file")
			continue
		}
		chat.deps.Logger.Debug("file exposed. parsing...", "url", fileURL)

		parsed, err := chat.parseFile(ctx, fileURL)
		if err != nil {
			chat.deps.Logger.With(logger.ERROR, err).Error("parsing failed")
			continue
		}

		created, err := chat.persistParsed(file, parsed)
		if err != nil {
			chat.deps.Logger.With(logger.ERROR, err).Error("failed to persist parsing results")
			continue
		}
		totalReceiptsCreated += created
	}
	return totalReceiptsCreated, nil
}

// ensureExposedFile creates ExposedFile record when missing and returns the key
func (chat *Chat) ensureExposedFile(file *db.File) (string, error) {
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

const PRMOPT_PARSE_FILE = `Parse receipts and products from the provided file to the exact JSON structure:
{
  "receipts": [
    {
      "totalBeforeTax": "string amount",
      "tax": "string amount",
      "totalAfterTax": "string amount",
      "origin": "string",
      "recipient": "string",
      "details": "string",
      "summary": "string",
      "products": [
        {
          "title": "",
          "details": "",
          "summary": "",
          "totalBeforeTax": "",
          "tax": "",
          "totalAfterTax": "",
          "categories": [
            ""
          ]
        }
      ]
    }
  ]
}`

// parseFile asks OpenAI to extract structured JSON from the file URL
func (chat *Chat) parseFile(ctx context.Context, fileURL string) (extractionResult, error) {
	var res extractionResult

	params := openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT5,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(PROMPT_SYSTEM),
			openai.SystemMessage(
				[]openai.ChatCompletionContentPartUnionParam{
					openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{URL: fileURL, Detail: "auto"}),
					openai.TextContentPart(PRMOPT_PARSE_FILE),
				},
			),
		},
	}

	resp, err := chat.oClient.Chat.Completions.Create(ctx, params)
	if err != nil {
		return res, fmt.Errorf("extraction call failed: %w", errors.WithStack(err))
	}

	assistantText := ""
	if len(resp.Choices) > 0 && resp.Choices[0].Message.Content != nil {
		assistantText = resp.Choices[0].Message.Content.GetText()
	}
	if assistantText == "" {
		return res, fmt.Errorf("empty extraction response")
	}

	if err := json.Unmarshal([]byte(assistantText), &res); err != nil {
		return res, fmt.Errorf("unmarshal extraction json: %w", errors.WithStack(err))
	}

	return res, nil
}

// persistExtraction writes receipts/products/categories to DB and returns created receipts count
func (chat *Chat) persistParsed(file db.File, res extractionResult) (int, error) {
	created := 0
	for _, r := range res.Receipts {
		totalBefore, _ := decimal.NewFromString(r.TotalBeforeTax)
		tax, _ := decimal.NewFromString(r.Tax)
		totalAfter, _ := decimal.NewFromString(r.TotalAfterTax)

		receipt := db.Receipt{
			FileID:         file.ID,
			TotalBeforeTax: totalBefore,
			Tax:            tax,
			TotalAfterTax:  totalAfter,
			Origin:         r.Origin,
			Recipient:      r.Recipient,
			Details:        r.Details,
			Summary:        r.Summary,
		}
		if err := chat.deps.DBC.Create(&receipt).Error; err != nil {
			return created, fmt.Errorf("create receipt: %w", errors.WithStack(err))
		}
		created++

		for _, p := range r.Products {
			pb, _ := decimal.NewFromString(p.TotalBeforeTax)
			ptax, _ := decimal.NewFromString(p.Tax)
			pa, _ := decimal.NewFromString(p.TotalAfterTax)

			prod := db.Product{
				ReceiptID:      receipt.ID,
				Title:          p.Title,
				Details:        p.Details,
				Summary:        p.Summary,
				TotalBeforeTax: pb,
				Tax:            ptax,
				TotalAfterTax:  pa,
			}
			if err := chat.deps.DBC.Create(&prod).Error; err != nil {
				chat.deps.Logger.With(logger.ERROR, errors.WithStack(err)).Error("failed to create product")
				continue
			}

			for _, c := range p.Categories {
				var cat db.Category
				if err := chat.deps.DBC.Where("title = ?", c).First(&cat).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						cat = db.Category{Title: c}
						if err := chat.deps.DBC.Create(&cat).Error; err != nil {
							chat.deps.Logger.With(logger.ERROR, errors.WithStack(err)).Error("failed to create category")
							continue
						}
					} else {
						chat.deps.Logger.With(logger.ERROR, errors.WithStack(err)).Error("failed to query category")
						continue
					}
				}

				if err := chat.deps.DBC.Model(&prod).Association("Categories").Append(&cat); err != nil {
					chat.deps.Logger.With(logger.ERROR, errors.WithStack(err)).Error("failed to append category to product")
				}
			}
		}
	}
	return created, nil
}
