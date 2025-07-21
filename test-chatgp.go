package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const (
	// imageFile    = "/Users/epec/fileshare/Public/Screenshot 2025-07-18 at 19.39.51.png"
	model        = openai.ChatModelGPT4o // use GPT‑4o Vision-capable
	inputRate    = 5.0                   // $5.00 per 1M input tokens :contentReference[oaicite:5]{index=5}
	outputRate   = 20.0                  // $20.00 per 1M output tokens :contentReference[oaicite:6]{index=6}
	costPerImage = 0.04                  // ~$0.04 per image tile :contentReference[oaicite:7]{index=7}
)

func _main() {
	client := openai.NewClient(option.WithAPIKey(os.Getenv("OPENAI_API_KEY")))
	var err error

	imageUrl := "https://files.evgenii-p.com/seafhttp/f/f51399e7bcd449139b07/?op=view"
	reqTxt := "Tell me what do you see on this image."
	// reqTxt := "Tell me what do you see on this image. Try to extract as much data as you can and return it in a json array, where each item is an hash with keys: {what: what-is-it-you-extracted, name: if-any, value: if-any, comment: any-context-you-want-me-to-know} "

	// Load and encode image
	// imgBytes, err := ioutil.ReadFile(imageFile)
	// if err != nil {
	// 	log.Fatalf("read img.jpg: %v", err)
	// }
	// imgB64 := base64.StdEncoding.EncodeToString(imgBytes)

	// Cost estimate (rough token guess)
	// estInputTokens := len(reqTxt)/4 + len(imgBytes)/100 // approx
	// estOutputTokens := 500                              // estimate
	// estCost := (float64(estInputTokens)*inputRate+float64(estOutputTokens)*outputRate)/1e6 + costPerImage
	// fmt.Printf("🔎 Estimated cost: $%.6f (in: %d tokens, out: %d tokens)\n",
	// 	estCost, estInputTokens, estOutputTokens)

	// Prepare and send streaming chat
	params := openai.ChatCompletionNewParams{
		Model: model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
				openai.TextContentPart(reqTxt),
				openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{URL: imageUrl}),
			}),
		},
	}

	stream := client.Chat.Completions.NewStreaming(context.Background(), params)
	err = stream.Err()
	if err != nil {
		log.Fatalf("stream error: %v", err)
	}
	defer stream.Close()

	fmt.Println("🚨 AI response: ")
	for stream.Next() {
		resp := stream.Current()
		chunk := resp.Choices[0].Delta.Content
		fmt.Print(chunk)
	}

	err = stream.Err()
	if err != nil {
		log.Fatalf("stream last error: %v", err)
	}

	// Retrieve final usage for more accurate billing
	// if resp, err := stream.GetResponse(); err == nil {
	// 	used := resp.Usage
	// 	realCost := (float64(used.PromptTokens)*inputRate+float64(used.CompletionTokens)*outputRate)/1e6 + costPerImage
	// 	fmt.Printf("\n✅ Done in %v. Input tokens: %d, Output tokens: %d, Cost: $%.6f\n",
	// 		time.Since(stream.StartedAt()), used.PromptTokens, used.CompletionTokens, realCost)
	// }
}
