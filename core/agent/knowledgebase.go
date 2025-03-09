package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mudler/LocalAgent/pkg/xlog"
	"github.com/sashabaranov/go-openai"
)

func (a *Agent) knowledgeBaseLookup() {
	if (!a.options.enableKB && !a.options.enableLongTermMemory && !a.options.enableSummaryMemory) ||
		len(a.currentConversation) <= 0 {
		xlog.Debug("[Knowledge Base Lookup] Disabled, skipping", "agent", a.Character.Name)
		return
	}

	// Walk conversation from bottom to top, and find the first message of the user
	// to use it as a query to the KB
	var userMessage string
	userMessage = a.currentConversation.GetLatestUserMessage().Content

	xlog.Info("[Knowledge Base Lookup] Last user message", "agent", a.Character.Name, "message", userMessage, "lastMessage", a.currentConversation.GetLatestUserMessage())

	if userMessage == "" {
		xlog.Info("[Knowledge Base Lookup] No user message found in conversation", "agent", a.Character.Name)
		return
	}

	results, err := a.options.ragdb.Search(userMessage, a.options.kbResults)
	if err != nil {
		xlog.Info("Error finding similar strings inside KB:", "error", err)
	}

	if len(results) == 0 {
		xlog.Info("[Knowledge Base Lookup] No similar strings found in KB", "agent", a.Character.Name)
		return
	}

	formatResults := ""
	for _, r := range results {
		formatResults += fmt.Sprintf("- %s \n", r)
	}
	xlog.Info("[Knowledge Base Lookup] Found similar strings in KB", "agent", a.Character.Name, "results", formatResults)

	// a.currentConversation = append(a.currentConversation,
	// 	openai.ChatCompletionMessage{
	// 		Role:    "system",
	// 		Content: fmt.Sprintf("Given the user input you have the following in memory:\n%s", formatResults),
	// 	},
	// )
	a.currentConversation = append([]openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: fmt.Sprintf("Given the user input you have the following in memory:\n%s", formatResults),
		}}, a.currentConversation...)
}

func (a *Agent) saveConversation(m Messages, prefix string) error {
	if a.options.conversationsPath == "" {
		return nil
	}
	dateTime := time.Now().Format("2006-01-02-15-04-05")
	fileName := a.Character.Name + "-" + dateTime + ".json"
	if prefix != "" {
		fileName = prefix + "-" + fileName
	}
	os.MkdirAll(a.options.conversationsPath, os.ModePerm)
	return m.Save(filepath.Join(a.options.conversationsPath, fileName))
}

func (a *Agent) saveCurrentConversation() {

	if err := a.saveConversation(a.currentConversation, ""); err != nil {
		xlog.Error("Error saving conversation", "error", err)
	}

	if !a.options.enableLongTermMemory && !a.options.enableSummaryMemory {
		xlog.Debug("Long term memory is disabled", "agent", a.Character.Name)
		return
	}

	xlog.Info("Saving conversation", "agent", a.Character.Name, "conversation size", len(a.currentConversation))

	if a.options.enableSummaryMemory && len(a.currentConversation) > 0 {
		msg, err := a.askLLM(a.context.Context, []openai.ChatCompletionMessage{{
			Role:    "user",
			Content: "Summarize the conversation below, keep the highlights as a bullet list:\n" + Messages(a.currentConversation).String(),
		}})
		if err != nil {
			xlog.Error("Error summarizing conversation", "error", err)
		}

		if err := a.options.ragdb.Store(msg.Content); err != nil {
			xlog.Error("Error storing into memory", "error", err)
		}
	} else {
		for _, message := range a.currentConversation {
			if message.Role == "user" {
				if err := a.options.ragdb.Store(message.Content); err != nil {
					xlog.Error("Error storing into memory", "error", err)
				}
			}
		}
	}
}
