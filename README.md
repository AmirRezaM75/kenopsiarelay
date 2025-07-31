```go
package main

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown gracefully
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down...")
		cancel() // Everything shuts down cleanly
	}()

	config := gameserver.Config[MyGameState]{
		Context:            ctx,
		DispatchBufferSize: 1000, // Tune for your load
		GameSlug:           "my-awesome-game",
		// ... other config
	}

	server := gameserver.NewGameServer(config)
	log.Fatal(http.ListenAndServe(":8080", server.GetRouter()))
}
```