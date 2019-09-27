package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

var api *slack.Client
var conf config
var data save

var daysOfWeek = map[string]time.Weekday{
	"sun": time.Sunday,
	"mon": time.Monday,
	"tue": time.Tuesday,
	"wed": time.Wednesday,
	"thu": time.Thursday,
	"fri": time.Friday,
	"sat": time.Saturday,
}

func parseWeekday(v string) (time.Weekday, error) {
	if len(v) < 3 {
		return -1, fmt.Errorf("Invalid weekday (%s) to short", v)
	}
	v = strings.ToLower(v[:3])
	if d, ok := daysOfWeek[v]; ok {
		return d, nil
	}

	return -1, fmt.Errorf("invalid weekday '%s'", v)
}

type config struct {
	Token      string `json:"token"`
	GroupID    string `json:"group_id"`
	CookingDay string `json:"cooking_day"`
	cookingDay time.Weekday
}

type save struct {
	DishHistory []dishHistory `json:"dish_history"`
}

type dishHistory struct {
	DishName string   `json:"dish_name"`
	Cooked   []cooked `json:"cooked"`
}

type cooked struct {
	CookedBy string            `json:"cooked_by"`
	Date     time.Time         `json:"date"`
	Rating   string            `json:"rating"`
	Voted    map[string]string `json:"voted"`
}

func loadFromJSON() error {
	b, err := ioutil.ReadFile("config.json")
	if os.IsNotExist(err) {
		return fmt.Errorf("Please deposit valid config: %s", err.Error())
	} else if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &conf); err != nil {
		return err
	}
	fmt.Println([]byte("[]"))

	if conf.GroupID == "" || (conf.GroupID[0] == 91 && conf.GroupID[len(conf.GroupID)-1] == 93) {
		return fmt.Errorf("Invalide config: Invalide group_id")
	}

	if conf.Token == "" || (conf.Token[0] == 91 && conf.Token[len(conf.Token)-1] == 93) {
		return fmt.Errorf("Invalide config: Invalide token")
	}

	conf.cookingDay, err = parseWeekday(conf.CookingDay)
	if err != nil {
		return fmt.Errorf("Invalide config: %s", err)
	}

	b, err = ioutil.ReadFile("save.json")
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	return nil
}

func nextCookinDay() time.Time {
	days := int(conf.cookingDay - time.Now().Weekday())
	fmt.Println(days)
	if days < 0 {
		days = 7 + days
	}
	fmt.Println(days)
	return time.Now().AddDate(0, 0, days)
}

func saveToJSON() error {
	b, err := json.Marshal(&data)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile("save.json", b, 777); err != nil {
		return err
	}
	return nil
}

func update() {
	defer saveToJSON()
	if data.DishHistory == nil || len(data.DishHistory) < 1 {
		newDish()
	}
	fmt.Printf("%#v\n", data.DishHistory)
	fmt.Println(conf.GroupID)
	group, err := api.GetGroupInfo(conf.GroupID)
	if err != nil {
		panic(err)
	}
	for i := range group.Members {
		fmt.Println(group.Members[i])
		user, err := api.GetUserInfo(group.Members[i])
		if err != nil {
			fmt.Println(err)
			continue
		}
		//save.Users[_] =
		fmt.Printf("%#v\n", user)
	}
}

func newDish() {

}

func main() {
	err := loadFromJSON()
	if err != nil {
		panic(err)
	}
	d := nextCookinDay()
	fmt.Println(d)
	fmt.Println(conf.cookingDay)
	fmt.Println(d.Weekday())
	panic(nil)

	api = slack.New(conf.Token)
	if api == nil {
		panic("empty api")
	}
	resp, err := api.AuthTest()
	if err != nil {
		panic(fmt.Errorf("Invalide Auth: %s", err))
	}
	if resp == nil {
		panic("Empty auth test response")
	}

	prefix := "<@" + resp.UserID + ">"

	fmt.Println("Hot & Spicy Bot starting...")

	rtm := api.NewRTM()

	go rtm.ManageConnection()
	time.Sleep(time.Second * 5)

	fmt.Println(prefix)

	update()

	panic("Test")

	for msg := range rtm.IncomingEvents {
		//fmt.Print("Event Received: ")
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			if !strings.HasPrefix(ev.Msg.Text, prefix) {
				//fmt.Println("Message NOT for me: '" + ev.Text + "'")
				continue
			}
			fmt.Println("Message for me: '" + ev.Text + "'")
			args := strings.ToLower(strings.TrimPrefix(ev.Text, prefix))

			if strings.Contains(args, "wer kocht") {
				api.SendMessage(ev.Channel, slack.MsgOptionText("nicht ich", false))
			} else if strings.Contains(args, "was wird gekocht") {
				api.SendMessage(ev.Channel, slack.MsgOptionText("Schniposa", false))
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
