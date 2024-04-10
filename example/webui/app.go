package main

import (
	"fmt"
	"net/http"

	. "github.com/mudler/local-agent-framework/agent"

	"github.com/donseba/go-htmx"
	fiber "github.com/gofiber/fiber/v2"
)

type (
	App struct {
		htmx *htmx.HTMX
		pool *AgentPool
	}
)

func (a *App) KnowledgeBase(db *InMemoryDatabase) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		payload := struct {
			URL       string `form:"url"`
			ChunkSize int    `form:"chunk_size"`
		}{}

		if err := c.BodyParser(&payload); err != nil {
			return err
		}

		website := payload.URL
		if website == "" {
			return fmt.Errorf("please enter a URL")
		}
		chunkSize := defaultChunkSize
		if payload.ChunkSize > 0 {
			chunkSize = payload.ChunkSize
		}

		go WebsiteToKB(website, chunkSize, db)

		return c.Redirect("/knowledgebase")
	}
}

func (a *App) Notify(pool *AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		payload := struct {
			Message string `form:"message"`
		}{}

		if err := c.BodyParser(&payload); err != nil {
			return err
		}

		query := payload.Message
		if query == "" {
			_, _ = c.Write([]byte("Please enter a message."))
			return nil
		}

		agent := pool.GetAgent(c.Params("name"))
		agent.Ask(
			WithText(query),
		)
		_, _ = c.Write([]byte("Message sent"))

		return nil
	}
}

func (a *App) Delete(pool *AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		if err := pool.Remove(c.Params("name")); err != nil {
			fmt.Println("Error removing agent", err)
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		}
		return c.Redirect("/agents")
	}
}

func (a *App) Create(pool *AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		config := AgentConfig{}
		if err := c.BodyParser(&config); err != nil {
			return err
		}

		fmt.Printf("Agent configuration: %+v\n", config)

		if config.Name == "" {
			c.Status(http.StatusBadRequest).SendString("Name is required")
			return nil
		}
		if err := pool.CreateAgent(config.Name, &config); err != nil {
			c.Status(http.StatusInternalServerError).SendString(err.Error())
			return nil
		}
		return c.Redirect("/agents")
	}
}

func (a *App) Chat(pool *AgentPool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		payload := struct {
			Message string `json:"message"`
		}{}

		if err := c.BodyParser(&payload); err != nil {
			return err
		}
		agentName := c.Params("name")
		manager := pool.GetManager(agentName)

		query := payload.Message
		if query == "" {
			_, _ = c.Write([]byte("Please enter a message."))
			return nil
		}
		manager.Send(
			NewMessage(
				chatDiv(query, "gray"),
			).WithEvent("messages"))

		go func() {
			agent := pool.GetAgent(agentName)
			if agent == nil {
				fmt.Println("Agent not found in pool", c.Params("name"))
				return
			}
			res := agent.Ask(
				WithText(query),
			)
			fmt.Println("response is", res.Response)
			manager.Send(
				NewMessage(
					chatDiv(res.Response, "blue"),
				).WithEvent("messages"))
			manager.Send(
				NewMessage(
					disabledElement("inputMessage", false), // show again the input
				).WithEvent("message_status"))

			//result := `<i>done</i>`
			//	_, _ = w.Write([]byte(result))
		}()

		manager.Send(
			NewMessage(
				loader() + disabledElement("inputMessage", true),
			).WithEvent("message_status"))

		return nil
	}
}
