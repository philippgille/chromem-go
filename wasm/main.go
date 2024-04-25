//go:build js

package main

import (
	"context"
	"errors"
	"syscall/js"

	"github.com/philippgille/chromem-go"
)

var c *chromem.Collection

func main() {
	js.Global().Set("initDB", js.FuncOf(initDB))
	js.Global().Set("addDocument", js.FuncOf(addDocument))
	js.Global().Set("query", js.FuncOf(query))

	select {} // prevent main from exiting
}

// Exported function to initialize the database and collection.
// Takes an OpenAI API key as argument.
func initDB(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return "expected 1 argument with the OpenAI API key"
	}

	openAIAPIKey := args[0].String()
	embeddingFunc := chromem.NewEmbeddingFuncOpenAI(openAIAPIKey, chromem.EmbeddingModelOpenAI3Small)

	db := chromem.NewDB()
	var err error
	c, err = db.CreateCollection("chromem", nil, embeddingFunc)
	if err != nil {
		return err.Error()
	}

	return nil
}

// Exported function to add documents to the collection.
// Takes the document ID and content as arguments.
func addDocument(this js.Value, args []js.Value) interface{} {
	ctx := context.Background()

	var id string
	var content string
	var err error
	if len(args) != 2 {
		err = errors.New("expected 2 arguments with the document ID and content")
	} else {
		id = args[0].String()
		content = args[1].String()
	}

	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolve := args[0]
		reject := args[1]
		go func() {
			if err != nil {
				handleErr(err, reject)
				return
			}

			err = c.AddDocument(ctx, chromem.Document{
				ID:      id,
				Content: content,
			})
			if err != nil {
				handleErr(err, reject)
				return
			}
			resolve.Invoke()
		}()
		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// Exported function to query the collection
// Takes the query string and the number of documents to return as argument.
func query(this js.Value, args []js.Value) interface{} {
	ctx := context.Background()

	var q string
	var err error
	if len(args) != 1 {
		err = errors.New("expected 1 argument with the query string")
	} else {
		q = args[0].String()
	}

	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolve := args[0]
		reject := args[1]
		go func() {
			if err != nil {
				handleErr(err, reject)
				return
			}

			res, err := c.Query(ctx, q, 1, nil, nil)
			if err != nil {
				handleErr(err, reject)
				return
			}

			// Convert response to JS values
			// TODO: Return more than one result
			o := js.Global().Get("Object").New()
			o.Set("ID", res[0].ID)
			o.Set("Similarity", res[0].Similarity)
			o.Set("Content", res[0].Content)

			resolve.Invoke(o)
		}()
		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

func handleErr(err error, reject js.Value) {
	errorConstructor := js.Global().Get("Error")
	errorObject := errorConstructor.New(err.Error())
	reject.Invoke(errorObject)
}
