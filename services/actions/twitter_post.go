package actions

import (
	"context"
	"fmt"

	"github.com/mudler/LocalAgent/core/action"
	"github.com/mudler/LocalAgent/services/connectors/twitter"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func NewPostTweet(config map[string]string) *PostTweetAction {
	return &PostTweetAction{
		token:            config["token"],
		noCharacterLimit: config["noCharacterLimits"] == "true",
	}
}

type PostTweetAction struct {
	token            string
	noCharacterLimit bool
}

func (a *PostTweetAction) Run(ctx context.Context, params action.ActionParams) (action.ActionResult, error) {
	result := struct {
		Text string `json:"text"`
	}{}
	err := params.Unmarshal(&result)
	if err != nil {
		fmt.Printf("error: %v", err)

		return action.ActionResult{}, err
	}

	if !a.noCharacterLimit && len(result.Text) > 280 {
		return action.ActionResult{}, fmt.Errorf("tweet is too long, max 280 characters")
	}

	client := twitter.NewTwitterClient(a.token)

	if err := client.Post(result.Text); err != nil {
		return action.ActionResult{}, err
	}

	return action.ActionResult{Result: fmt.Sprintf("twitter post created")}, nil
}

func (a *PostTweetAction) Definition() action.ActionDefinition {
	return action.ActionDefinition{
		Name:        "post_tweet",
		Description: "Post a tweet",
		Properties: map[string]jsonschema.Definition{
			"text": {
				Type:        jsonschema.String,
				Description: "The text to send.",
			},
		},
		Required: []string{"text"},
	}
}

func (a *PostTweetAction) Plannable() bool {
	return true
}
