package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/EPecherkin/catty-counting/db"
	"github.com/EPecherkin/catty-counting/deps"
	"github.com/EPecherkin/catty-counting/logger"
	"github.com/google/uuid"
	"github.com/openai/openai-go/v2"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Chat struct {
	oClient *openai.Client
	deps    deps.Deps
	history []openai.ChatCompletionMessageParamUnion
}

func newChat(oClient *openai.Client, deps deps.Deps) *Chat {
	deps.Logger = deps.Logger.With(logger.CALLER, "openai.Chat")
	deps.Logger.Debug("Creating openai chat")
	return &Chat{oClient: oClient, deps: deps}
}

const systemPrompt = `You are an accounting helping assistant, which is capable of processing docs, receipts, building statistics and giving advices.`

func (chat *Chat) Talk(ctx context.Context, message db.Message, responseChan chan<- string) {
	chat.deps.Logger = chat.deps.Logger.With(logger.USER_ID, message.UserID, logger.MESSAGE_ID, message.ID)
	chat.deps.Logger.Debug("starting talking")
	defer func() {
		if err := recover(); err != nil {
			chat.deps.Logger.With(logger.ERROR, err).Error("failed talking")
		} else {
			chat.deps.Logger.Debug("finished talking")
		}
	}()

	// preload files and exposed files
	var fullMsg db.Message
	if err := chat.deps.DBC.Preload("Files.ExposedFile").Preload("Files").First(&fullMsg, message.ID).Error; err != nil {
		chat.deps.Logger.With(logger.ERROR, errors.WithStack(err)).Debug("failed to preload message files, falling back to provided message")
		fullMsg = message
	}

	// Process files using extracted helper methods and return a short summary
	totalReceiptsCreated, _ := chat.processFiles(ctx, fullMsg)
	summaryText := fmt.Sprintf("Processed %d files, created %d receipts.", len(fullMsg.Files), totalReceiptsCreated)

	responseMessage := db.Message{
		UserID:    message.UserID,
		Text:      summaryText,
		Direction: db.MessageDirectionToUser,
	}
	if err := chat.deps.DBC.Create(&responseMessage).Error; err != nil {
		chat.deps.Logger.With(logger.ERROR, errors.WithStack(err)).Error("failed to save summary message")
	}

	responseChan <- summaryText
	assistantMessage := openai.AssistantMessage(responseMessage.Text)
	chat.history = append(chat.history, assistantMessage)
	return

	// For every file create ExposedFile if missing and ask OpenAI to extract structured data
	totalReceiptsCreated := 0
	for _, file := range fullMsg.Files {
		chat.deps.Logger = chat.deps.Logger.With("file_id", file.ID, "file_blob", file.BlobKey)
		if file.ExposedFile == nil || file.ExposedFile.Key == "" {
			ek := uuid.New().String()
			ef := db.ExposedFile{FileID: file.ID, Key: ek}
			if err := chat.deps.DBC.Create(&ef).Error; err != nil {
				chat.deps.Logger.With(logger.ERROR, errors.WithStack(err)).Error("failed to create exposed file")
				continue
			}
			file.ExposedFile = &ef
		}

		fileURL := fmt.Sprintf("%s/%s", config.FileBucket(), file.ExposedFile.Key)
		chat.deps.Logger.Debug("requesting extraction for file", "url", fileURL)

		extractPrompt := fmt.Sprintf("Please analyze the document at %s and extract receipts data as a JSON object with structure: {\"receipts\": [ {\"totalBeforeTax\": \"string amount\", \"tax\": \"string amount\", \"totalAfterTax\": \"string amount\", \"origin\": \"string\", \"recipient\": \"string\", \"details\": \"string\", \"summary\": \"string\", \"products\": [ {\"title\":\"\",\"details\":\"\",\"summary\":\"\",\"totalBeforeTax\":\"\",\"tax\":\"\",\"totalAfterTax\":\"\",\"categories\": [\"\"] } ] } ] }", fileURL)

		params := openai.ChatCompletionNewParams{
			Model: openai.ChatModelGPT5,
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(systemPrompt),
				openai.SystemMessage(extractPrompt),
			},
		}

		resp, err := chat.oClient.Chat.Completions.Create(ctx, params)
		if err != nil {
			chat.deps.Logger.With(logger.ERROR, errors.WithStack(err)).Error("extraction call failed")
			continue
		}

		assistantText := ""
		if len(resp.Choices) > 0 && resp.Choices[0].Message.Content != nil {
			assistantText = resp.Choices[0].Message.Content.GetText()
		}

		if assistantText == "" {
			chat.deps.Logger.Debug("empty extraction response")
			continue
		}

		var extracted struct {
			Receipts []struct {
				TotalBeforeTax string `json:"totalBeforeTax"`
				Tax            string `json:"tax"`
				TotalAfterTax  string `json:"totalAfterTax"`
				Origin         string `json:"origin"`
				Recipient      string `json:"recipient"`
				Details        string `json:"details"`
				Summary        string `json:"summary"`
				Products       []struct {
					Title          string   `json:"title"`
					Details        string   `json:"details"`
					Summary        string   `json:"summary"`
					TotalBeforeTax string   `json:"totalBeforeTax"`
					Tax            string   `json:"tax"`
					TotalAfterTax  string   `json:"totalAfterTax"`
					Categories     []string `json:"categories"`
				} `json:"products"`
			} `json:"receipts"`
		}

		if err := json.Unmarshal([]byte(assistantText), &extracted); err != nil {
			chat.deps.Logger.With(logger.ERROR, errors.WithStack(err)).Error("failed to unmarshal extracted json")
			continue
		}

		for _, r := range extracted.Receipts {
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
				chat.deps.Logger.With(logger.ERROR, errors.WithStack(err)).Error("failed to create receipt")
				continue
			}
			totalReceiptsCreated++

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
	}

	// After processing files, ask the model to prepare a short human-readable summary
	summaryText := fmt.Sprintf("Processed %d files, created %d receipts.", len(fullMsg.Files), totalReceiptsCreated)

	// Create response message and save
	responseMessage := db.Message{
		UserID:    message.UserID,
		Text:      summaryText,
		Direction: db.MessageDirectionToUser,
	}
	if err := chat.deps.DBC.Create(&responseMessage).Error; err != nil {
		chat.deps.Logger.With(logger.ERROR, errors.WithStack(err)).Error("failed to save summary message")
	}

	// Stream back the summary
	responseChan <- summaryText

	// Also append to history
	assistantMessage := openai.AssistantMessage(responseMessage.Text)
	chat.history = append(chat.history, assistantMessage)
}

func (chat *Chat) loadHistory(userID uint) error {
	if len(chat.history) > 0 {
		return nil
	}

	var messages []db.Message
	if err := chat.deps.DBC.Where("user_id = ?", userID).Order("created_at asc").Find(&messages).Error; err != nil {
		return fmt.Errorf("loading messages from db: %w", errors.WithStack(err))
	}

	chat.history = []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
	}

	for _, msg := range messages {
		if msg.Direction == db.MessageDirectionFromUser {
			chat.history = append(chat.history, openai.UserMessage(chat.buildContent(msg)))
		} else {
			chat.history = append(chat.history, openai.AssistantMessage(msg.Text))
		}
	}

	return nil
}

func (chat *Chat) buildContent(message db.Message) []openai.ChatCompletionContentPartUnionParam {
	content := []openai.ChatCompletionContentPartUnionParam{
		openai.TextContentPart(message.Text),
	}

	return content
}

func (chat *Chat) handleError(err error, details string) string {
	chat.deps.Logger.With(logger.ERROR, err).Error(details)
	if config.LogDebug() {
		return fmt.Sprintf(details+": %+v", err)
	} else {
		return "Sorry, something went wrong"
	}
}
