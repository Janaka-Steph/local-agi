package actions

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mudler/LocalAgent/core/action"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/tmc/langchaingo/tools/duckduckgo"
	"mvdan.cc/xurls/v2"
)

const (
	MetadataUrls = "urls"
)

func NewSearch(config map[string]string) *SearchAction {
	results := config["results"]
	intResult := 1

	// decode int from string
	if results != "" {
		_, err := fmt.Sscanf(results, "%d", &intResult)
		if err != nil {
			fmt.Printf("error: %v", err)
		}
	}

	slog.Info("Search action with results: ", "results", intResult)
	return &SearchAction{results: intResult}
}

type SearchAction struct{ results int }

func (a *SearchAction) Run(ctx context.Context, params action.ActionParams) (action.ActionResult, error) {
	result := struct {
		Query string `json:"query"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return action.ActionResult{}, err
	}
	ddg, err := duckduckgo.New(a.results, "LocalAgent")
	if err != nil {
		fmt.Printf("error: %v", err)

		return action.ActionResult{}, err
	}
	res, err := ddg.Call(ctx, result.Query)
	if err != nil {
		fmt.Printf("error: %v", err)

		return action.ActionResult{}, err
	}

	rxStrict := xurls.Strict()
	urls := rxStrict.FindAllString(res, -1)

	return action.ActionResult{Result: res, Metadata: map[string]interface{}{MetadataUrls: urls}}, nil
}

func (a *SearchAction) Definition() action.ActionDefinition {
	return action.ActionDefinition{
		Name:        "search_internet",
		Description: "Search the internet for something.",
		Properties: map[string]jsonschema.Definition{
			"query": {
				Type:        jsonschema.String,
				Description: "The query to search for.",
			},
		},
		Required: []string{"query"},
	}
}
