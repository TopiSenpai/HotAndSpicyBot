package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

var api *slack.Client
var conf config
var data save

type config struct {
	Token     string `json:"token"`
	ChannelID string `json:"channel_id"`
}

type save struct {
	Users       map[string]user `json:"users"`
	DishHistory []dishHistory   `json:"dish_history"`
}

type user struct {
	SlackID      string    `json:"slack_id"`
	LastCooked   string    `json:"last_cooked"`
	LastHelped   string    `json:"last_helped"`
	UnavailUntil time.Time `json:"unavail_until"`
}

type dishHistory struct {
	DishName string `json:"dish_name"`
	Cooked   []cooked
}

type cooked struct {
	CookedBy string            `json:"cooked_by"`
	Date     time.Time         `json:"date"`
	Rating   string            `json:"rating"`
	Voted    map[string]string `json:"voted"`
}

func loadFromJSON() save {
	b, err := ioutil.ReadFile("save.json")
	if err != nil {
		panic(err)
	}
	save := save{}
	if err := json.Unmarshal(b, &save); err != nil {
		panic(err)
	}
	return save
}

func saveToJSON() {
	b, err := json.Marshal(&data)
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile("save.json", b, 666); err != nil {
		panic(err)
	}
}

func update() {
	users, err := api.GetUserGroupMembers(conf.ChannelID)
	if err != nil {
		panic(err)
	}
	for i := range users {
		//save.Users[_] =
		fmt.Printf("%#v\n", users[i])
	}
}

func main() {

	b, err := ioutil.ReadFile("config.json")
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(b, &conf); err != nil {
		panic(err)
	}
	api := slack.New(conf.Token)
	fmt.Println("Hot & Spicy Bot starting...")
	rtm := api.NewRTM()

	go rtm.ManageConnection()

	update()

	for msg := range rtm.IncomingEvents {
		//fmt.Print("Event Received: ")
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			prefix := "<@" + rtm.GetInfo().User.ID + ">"
			if !strings.HasPrefix(ev.Msg.Text, prefix) {
				//fmt.Println("Message NOT for me: '" + ev.Text + "'")
				continue
			}
			fmt.Println("Message for me: '" + ev.Text + "'")
			args := strings.ToLower(strings.TrimPrefix(ev.Text, prefix))

			if strings.Contains(args, "wer kocht") {
				api.SendMessage(ev.Channel, slack.MsgOptionText("nicht ich", false))
			} else if strings.Contains(args, "was wird gekocht") {
				api.SendMessage(ev.Channel, slack.MsgOptionText("for me", false))
			} else if strings.Contains(args, "gerichte") {
				api.SendMessage(ev.Channel, slack.MsgOptionText("for me", false))
			} else {
				api.SendMessage(ev.Channel, slack.MsgOptionText("benutze:\n●wer kocht\n●was wird gekocht\n●Gerichte\n●@user kann nicht\n●nächste Woche fällt aus", false))
			}

		case *slack.RTMError:
			fmt.Printf("Error: %s\n", ev.Error())

		case *slack.InvalidAuthEvent:
			fmt.Printf("Invalid credentials")
			return

		default:
			//fmt.Printf("Unexpected: %v\n", msg.Data)
		}
	}

}
