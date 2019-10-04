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
var currentDish *dish
var now = func() cookingTime { return cookingTime{Time: time.Now(), set: true} }

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
	DishHistory []dish `json:"dish_history"`
}

type dish struct {
	DishName string          `json:"dish_name"`
	CookedBy string          `json:"cooked_by"`
	Helped   string          `json:"helped"`
	Date     cookingTime     `json:"date"`
	Rating   string          `json:"rating"`
	Voted    map[string]bool `json:"voted"`
}

type cookingTime struct {
	time.Time
	set bool
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

func TodayAt(hour, min, sec, nsec int) cookingTime {
	today := now()
	today.Time = time.Date(today.Year(), today.Month(), today.Day(), hour, min, sec, nsec, today.Location())
	return today
}

func nextCookinDay() cookingTime {
	cookingDate := TodayAt(12, 0, 0, 0)
	cookingDate.Time = cookingDate.AddDate(0, 0, int(conf.cookingDay-now().Weekday()))
	for cookingDate.AddDate(0, 0, -1).Before(now().Time) {
		cookingDate.Time = cookingDate.AddDate(0, 0, 7)
	}
	return cookingDate
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

func update() error {
	defer saveToJSON()
	if currentDish == nil {
		var lastDish *dish
		for i := range data.DishHistory {
			if lastDish == nil || lastDish.Date.Before(data.DishHistory[i].Date.Time) {
				lastDish = &data.DishHistory[i]
			}
		}
		if lastDish == nil || lastDish.Rating != "" {
			cookingDate := nextCookinDay()
			if cookingDate.AddDate(0, 0, -3).After(now().Time) {
				group, err := api.GetGroupInfo(conf.GroupID)
				if err != nil {
					return err
				}
				newDish := dish{Date: cookingDate}
				altHelper := ""
				lastCook := map[string]cookingTime{}
				lastHelp := map[string]cookingTime{}
				for i := range data.DishHistory {
					if date := lastCook[data.DishHistory[i].CookedBy]; !date.set || date.Before(data.DishHistory[i].Date.Time) {
						lastCook[data.DishHistory[i].CookedBy] = data.DishHistory[i].Date
					}
					if date := lastCook[data.DishHistory[i].Helped]; !date.set || date.Before(data.DishHistory[i].Date.Time) {
						lastCook[data.DishHistory[i].Helped] = data.DishHistory[i].Date
					}
				}
				for i := range group.Members {
					user := group.Members[i]
					if newCookLastDate := lastCook[user]; newDish.CookedBy == "" {
						newDish.CookedBy = user
					} else if currentCookLastDate := lastCook[newDish.CookedBy]; !newCookLastDate.set || (currentCookLastDate.set && newCookLastDate.Before(currentCookLastDate.Time)) {
						newDish.CookedBy = user
					}
					if newHelperLastDate, newHelperOk := lastHelp[user]; user == newDish.CookedBy {

					}
					if newHelperLastDate, newHelperOk := lastHelp[user]; true {

					}
				}
			}
		} else {
			currentDish = lastDish
		}
	}
	if currentDish.Voted == nil && currentDish.Date.Add(2*time.Hour).After(now().Time) {
		api.SendMessage(conf.GroupID, slack.MsgOptionText("Wie hat euch das essen geschmeckt bite stimmt mit :thumbsup: und :thumbsdown: ab.", false))
		currentDish.Voted = map[string]bool{}
	} else if currentDish.Voted != nil && currentDish.Date.AddDate(0, 0, 2).After(now().Time) {
		up := 0
		for i := range currentDish.Voted {
			if currentDish.Voted[i] {
				up++
			}
		}
		currentDish.Rating = fmt.Sprintf("%d/%d", up, len(currentDish.Voted))
		currentDish.Voted = nil
		api.SendMessage(conf.GroupID, slack.MsgOptionText(fmt.Sprintf("Abstimmung ist abgeschlossen Ergebnis ist: %s.", currentDish.Rating), false))
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
	return nil
}

func newDish(cookingDate time.Time) {

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

	go func() {
		t := time.NewTicker(5 * time.Minute)
		for _ = range t.C {
			update()
		}
	}()

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
				if currentDish == nil {
					api.SendMessage(ev.Channel, slack.MsgOptionText("Aktuell niemand.", false))
				} else {
					api.SendMessage(ev.Channel, slack.MsgOptionText(fmt.Sprintf("<@%s> kocht und <@%s> hilft.", currentDish.CookedBy, currentDish.Helped), false))
				}
			} else if strings.Contains(args, "was wird gekocht") {
				if currentDish == nil {
					api.SendMessage(ev.Channel, slack.MsgOptionText("Aktuell nichts.", false))
				} else if currentDish.DishName != "" {
					api.SendMessage(ev.Channel, slack.MsgOptionText(fmt.Sprintf("<@%s> kocht %s.", currentDish.CookedBy, currentDish.DishName), false))
				} else {
					api.SendMessage(ev.Channel, slack.MsgOptionText(fmt.Sprintf("<@%s> hat noch nicht gesagt was er kochen möchte.", currentDish.CookedBy), false))
				}
			} else if strings.Contains(args, "gerichte") {
				api.SendMessage(ev.Channel, slack.MsgOptionText("test", false))
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
