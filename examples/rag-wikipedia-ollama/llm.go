package main

import (
	"context"
	"net/http"
	"strings"
	"text/template"

	"github.com/sashabaranov/go-openai"
)

const (
	// We use a local LLM running in Ollama to ask a question: https://github.com/ollama/ollama
	ollamaBaseURL = "http://localhost:11434/v1"
	// We use Google's Gemma (2B), a very small model that doesn't need many resources
	// and is fast, but doesn't have much knowledge: https://huggingface.co/google/gemma-2b
	// We found Gemma 2B to be superior to TinyLlama (1.1B), Stable LM 2 (1.6B)
	// and Phi-2 (2.7B) for the retrieval augmented generation (RAG) use case.
	llmModel = "gemma:2b"
)

// There are many different ways to provide the context to the LLM.
// You can pass each context as user message, or the list as one user message,
// or pass it in the system prompt. The system prompt itself also has a big impact
// on how well the LLM handles the context, especially for LLMs with < 7B parameters.
// The prompt engineering is up to you, it's out of scope for the vector database.
var systemPromptTpl = template.Must(template.New("system_prompt").Parse(`
You are an assistant with access to a knowledge base, tasked with answering questions CarGurus' and ATA's norms and procedures.

ATA is a division of CarGurus responsible for the development and maintenance of CarGurus' automated advertising platform. ATA stands for Automated Traffic Aquistion.

Answer the question in a very concise manner. Use an unbiased and journalistic tone. Do not repeat text. Don't make anything up. If you are not sure about something, just say that you don't know.
{{- /* Stop here if no context is provided. The rest below is for handling contexts. */ -}}
{{- if . -}}
Answer the question solely based on the provided search results from the knowledge base. If the search results from the knowledge base are not relevant to the question at hand, just say that you don't know. Don't make anything up.

Anything between the following 'contexts' XML blocks is retrieved from the knowledge base, not part of the conversation with the user. The 'contexts' tag contains multiple 'context' ordered by relevance, so the first one is the most relevant.

Each 'context' tag has two children: 'content' and 'url'. The 'content' tag contains the text of the search result, and the 'url' tag contains the URL of the search result. The value of the 'content' should be used to answer questions.

You must end your response with the URLs (from the 'context' tag) of the search results that you used to answer the question. If you used multiple search results, separate the URLs with new lines.

<contexts>
    {{- if . -}}
		{{- range $context := .}}
		<context>
			<content>{{index $context "content"}}</content>
			<url>{{index $context "url"}}</url>
		</context>
		{{end}}
   {{- end}}
</contexts>
{{- end -}}

`))

func askLLM(ctx context.Context, contexts []map[string]string, question string) string {
	// We can use the OpenAI client because Ollama is compatible with OpenAI's API.
	openAIClient := openai.NewClientWithConfig(openai.ClientConfig{
		BaseURL:    ollamaBaseURL,
		HTTPClient: http.DefaultClient,
	})
	sb := &strings.Builder{}
	err := systemPromptTpl.Execute(sb, contexts)
	if err != nil {
		panic(err)
	}
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: sb.String(),
		}, {
			Role:    openai.ChatMessageRoleUser,
			Content: "Question: " + question,
		},
	}
	res, err := openAIClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    llmModel,
		Messages: messages,
	})
	if err != nil {
		panic(err)
	}
	reply := res.Choices[0].Message.Content
	reply = strings.TrimSpace(reply)

	return reply
}
