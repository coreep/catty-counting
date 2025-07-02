package telegram

import (
	ctx "context"
	"log"
	"os"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func Run(ctx ctx.Context) {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
		return
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	bot.Debug = true
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	for update := range updates {
		if update.Message == nil || update.Message.IsCommand() {
			continue
		}

		userMessage := update.Message.Text
		log.Printf("Received message from user: %s", userMessage)

		response, err := llm.QueryLLM(userMessage)
		if err != nil {
			log.Printf("Error querying LLM backend: %v", err)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Sorry, something went wrong.")
			bot.Send(msg)
			continue
		}
		log.Printf("Raw response from LLM %s\n", response)

		// Every answer from DeepSeek has <think/> tags with details. We don't need to show them to the user
		re := regexp.MustCompile(`<think>[\s\S]*?</think>\n?`)
		response = strings.Trim(re.ReplaceAllString(response, ""), " \n")

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending message to user: %v", err)
		}
	}
}
