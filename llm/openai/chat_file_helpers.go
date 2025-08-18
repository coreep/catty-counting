package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/log"
	"github.com/google/uuid"
	"github.com/openai/openai-go/v2"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type parsedProduct struct {
	Title          string   `json:"title"`
	Details        string   `json:"details"`
	Summary        string   `json:"summary"`
	TotalBeforeTax string   `json:"totalBeforeTax"`
	Tax            string   `json:"tax"`
	TotalAfterTax  string   `json:"totalAfterTax"`
	Categories     []string `json:"categories"`
}

type parsedReceipt struct {
	TotalBeforeTax string          `json:"totalBeforeTax"`
	Tax            string          `json:"tax"`
	TotalAfterTax  string          `json:"totalAfterTax"`
	Origin         string          `json:"origin"`
	Recipient      string          `json:"recipient"`
	Details        string          `json:"details"`
	Summary        string          `json:"summary"`
	Products       []parsedProduct `json:"products"`
}

type parsedFile struct {
	Receipts []parsedReceipt `json:"receipts"`
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

// Handles all files in the message: expose, extract and persist
func (chat *Chat) processFiles(ctx context.Context, message *db.Message) error {
	// TODO: we intend to mutate db.Message all the way down to the categories
	// any mutaton to the structure should affect message. Message -> Files -> Receipts -> Products -> Categories
	for _, file := range message.Files {
		logger := chat.deps.Logger.With("file_id", file.ID, "file_blob", file.BlobKey)

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

		if err := chat.attachParsedOnFile(file, parsedData, logger); err != nil {
			return fmt.Errorf("saving parsed data on file: %w", err)
		}
	}
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
func (chat *Chat) parseFile(ctx context.Context, fileURL string, logger *slog.Logger) (parsedFile, error) {
	logger.Debug("Sending file for parsing")
	var res parsedFile
	params := openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT5,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(PROMPT_SYSTEM),
			openai.UserMessage(
				[]openai.ChatCompletionContentPartUnionParam{
					openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{URL: fileURL, Detail: "auto"}),
					openai.TextContentPart(PRMOPT_PARSE_FILE),
				},
			),
		},
	}

	resp, err := chat.oClient.Chat.Completions.New(ctx, params)
	if err != nil {
		return res, fmt.Errorf("extraction call failed: %w", errors.WithStack(err))
	}
	logger.Debug("Parsing finished")

	assistantText := ""
	if len(resp.Choices) > 0 && resp.Choices[0].Message.Content != "" {
		assistantText = resp.Choices[0].Message.Content
	}
	if assistantText == "" {
		return res, fmt.Errorf("empty extraction response")
	}

	if err := json.Unmarshal([]byte(assistantText), &res); err != nil {
		return res, fmt.Errorf("unmarshal extraction json: %w", errors.WithStack(err))
	}
	logger.Debug("Seems to parsed fine")

	return res, nil
}

// Writes receipts/products/categories to DB and returns created receipts count
func (chat *Chat) attachParsedOnFile(file db.File, res parsedFile, logger *slog.Logger) error {
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
			return fmt.Errorf("create receipt: %w", errors.WithStack(err))
		}

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
				logger.With(log.ERROR, errors.WithStack(err)).Error("failed to create product")
				continue
			}

			for _, c := range p.Categories {
				var cat db.Category
				if err := chat.deps.DBC.Where("title = ?", c).First(&cat).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						cat = db.Category{Title: c}
						if err := chat.deps.DBC.Create(&cat).Error; err != nil {
							logger.With(log.ERROR, errors.WithStack(err)).Error("failed to create category")
							continue
						}
					} else {
						logger.With(log.ERROR, errors.WithStack(err)).Error("failed to query category")
						continue
					}
				}

				if err := chat.deps.DBC.Model(&prod).Association("Categories").Append(&cat); err != nil {
					logger.With(log.ERROR, errors.WithStack(err)).Error("failed to append category to product")
				}
			}
		}
	}
	return nil
}
