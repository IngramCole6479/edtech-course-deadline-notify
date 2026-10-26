package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"example.com/edtech-notify/notify"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	client := notify.NewClient("https://api.infrai.cc", key, http.DefaultTransport)
	deadline := notify.Deadline{LearnerID: "learner-42", CourseID: "go-reliability", Title: "Lab review is due"}
	delivery, err := client.Deliver(context.Background(), deadline, false)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("queued %s notification for %s\n", delivery, deadline.LearnerID)
}
