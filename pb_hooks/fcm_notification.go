package pb_hooks

import (
	"context"
	"log"

	"firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

var FCM *messaging.Client

func InitFirebase() {
	opt := option.WithCredentialsFile("pb_hooks/firebase-key.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatal(err)
	}
	FCM, err = app.Messaging(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}

func SendNotification(token, title, body string) {
	ctx := context.Background()
	msg := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: map[string]string{
			"click_action": "FLUTTER_NOTIFICATION_CLICK",
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				ChannelID: "default_channel",
				Sound:     "default",
			},
		},
	}
	_, err := FCM.Send(ctx, msg)
	if err != nil {
		log.Println("FCM error:", err)
	}
}