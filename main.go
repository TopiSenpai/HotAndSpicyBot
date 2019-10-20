package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/nlopes/slack"
)

var api *slack.Client
var bot string
var conf config
var data save
var saveFile = "save.json"
var dishLook = sync.Mutex{}
var currentDish *dish
var votingDish *dish
var futureDish *dish
var userRegex = regexp.MustCompile("<@([A-Z0-9]+)>")

var daysOfWeek = map[string]time.Weekday{
	"sun": time.Sunday,
	"mon": time.Monday,
	"tue": time.Tuesday,
	"wed": time.Wednesday,
	"thu": time.Thursday,
	"fri": time.Friday,
	"sat": time.Saturday,
	"son": time.Sunday,
	"die": time.Tuesday,
	"mit": time.Wednesday,
	"don": time.Thursday,
	"fre": time.Friday,
	"sam": time.Saturday,
}

var wochentage = map[time.Weekday]string{
	time.Sunday:    "Sonntag",
	time.Monday:    "Montag",
	time.Tuesday:   "Dienstag",
	time.Wednesday: "Mittwoch",
	time.Thursday:  "Donnerstag",
	time.Friday:    "Freitag",
	time.Saturday:  "Samstag",
}

var monate = map[time.Month]string{
	time.January:   "Januar",
	time.February:  "Februar",
	time.March:     "März",
	time.April:     "April",
	time.May:       "Mai",
	time.June:      "Juni",
	time.July:      "July",
	time.August:    "August",
	time.September: "September",
	time.November:  "Novembwer",
	time.December:  "Dezember",
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
	Cook     string          `json:"cook"`
	Helper   string          `json:"helper"`
	Date     time.Time       `json:"date"`
	Rating   string          `json:"rating"`
	Voted    map[string]bool `json:"voted"`
}

type cookingTime struct {
	time.Time
	set bool
}

func (ct1 cookingTime) before(ct2 cookingTime) bool {
	return !ct1.set || (ct2.set && ct1.Before(ct2.Time))
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

	b, err = ioutil.ReadFile(saveFile)
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

func next(w time.Weekday) time.Time {
	cookingDate := time.Now()
	cookingDate = time.Date(cookingDate.Year(), cookingDate.Month(), cookingDate.Day(), 12, 0, 0, 0, cookingDate.Location())
	cookingDate = cookingDate.AddDate(0, 0, int(w-time.Now().Weekday()))
	for cookingDate.AddDate(0, 0, -1).Before(time.Now()) {
		cookingDate = cookingDate.AddDate(0, 0, 7)
	}
	return cookingDate
}

func saveToJSON() error {
	b, err := json.Marshal(&data)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(saveFile, b, 0644); err != nil {
		return err
	}
	return nil
}

func (d *dish) start() error {
	if d.Cook == "" || d.Helper == "" {
		group, err := api.GetGroupInfo(conf.GroupID)
		if err != nil {
			return err
		}
		lastCook := map[string]cookingTime{}
		lastHelp := map[string]cookingTime{}
		for i := range data.DishHistory {
			if date := lastCook[data.DishHistory[i].Cook]; !date.set || date.Before(data.DishHistory[i].Date) {
				lastCook[data.DishHistory[i].Cook] = cookingTime{Time: data.DishHistory[i].Date, set: true}
			}
			if date := lastCook[data.DishHistory[i].Helper]; !date.set || date.Before(data.DishHistory[i].Date) {
				lastCook[data.DishHistory[i].Helper] = cookingTime{Time: data.DishHistory[i].Date, set: true}
			}
		}
		cook := d.Cook
		helper := d.Helper
		altHelper := ""
		for i := range group.Members {
			user := group.Members[i]
			if user == "" || user == bot {
				continue
			}

			if d.Cook == "" && (cook == "" || lastCook[user].before(lastCook[d.Cook])) {
				cook = user
			}

			if d.Helper == "" && (helper == "" || lastHelp[user].before(lastHelp[d.Helper])) {
				if helper != "" && (altHelper == "" || lastHelp[helper].before(lastHelp[altHelper])) {
					altHelper = helper
				}
				helper = user
			} else if d.Helper == "" && (altHelper == "" || lastHelp[user].before(lastHelp[altHelper])) {
				altHelper = user
			}
		}
		d.Cook = cook
		d.Helper = helper
		if d.Cook == d.Helper {
			d.Helper = altHelper
		}
	}
	_, _, _, err := api.SendMessage(conf.GroupID, slack.MsgOptionText(fmt.Sprintf("Am %s den %d. %s kocht <@%s> und <@%s> hilft.", wochentage[d.Date.Weekday()], d.Date.Day(), monate[d.Date.Month()], d.Cook, d.Helper), false))
	if err != nil {
		return err
	}
	_, _, _, err = api.SendMessage(conf.GroupID, slack.MsgOptionText(fmt.Sprintf("<@%s> was möchtest du kochen?", d.Cook), false))
	if err != nil {
		return err
	}
	return nil
}

func update() error {
	dishLook.Lock()
	defer dishLook.Unlock()
	defer saveToJSON()

	if currentDish == nil {
		var lastDish *dish
		for i := range data.DishHistory {
			if lastDish == nil || lastDish.Date.Before(data.DishHistory[i].Date) {
				lastDish = &data.DishHistory[i]
			}
		}
		if lastDish == nil || lastDish.Rating != "" {
			data.DishHistory = append(data.DishHistory, dish{Date: next(conf.cookingDay)})
			currentDish = &data.DishHistory[len(data.DishHistory)-1]
		} else {
			currentDish = lastDish
		}
	}
	if currentDish != nil {
		if currentDish.Voted == nil && time.Now().After(currentDish.Date.Add(2*time.Hour)) {
			_, _, _, err := api.SendMessage(conf.GroupID, slack.MsgOptionText("Wie hat euch das essen geschmeckt bite stimmt mit :thumbsup: und :thumbsdown: ab.", false))
			if err != nil {
				return err
			}
			currentDish.Voted = map[string]bool{}
		} else if currentDish.Voted != nil && time.Now().After(currentDish.Date.AddDate(0, 0, 2)) {
			up := 0
			for i := range currentDish.Voted {
				if currentDish.Voted[i] {
					up++
				}
			}
			currentDish.Rating = fmt.Sprintf("%d/%d", up, len(currentDish.Voted))
			currentDish.Voted = nil
			_, _, _, err := api.SendMessage(conf.GroupID, slack.MsgOptionText(fmt.Sprintf("Abstimmung ist abgeschlossen Ergebnis ist: %s.", currentDish.Rating), false))
			if err != nil {
				return err
			}
			currentDish = nil
		} else if time.Now().After(currentDish.Date.AddDate(0, 0, -3)) && (currentDish.Cook == "" || currentDish.Helper == "") {
			return currentDish.start()
		}
	}
	return nil
}

func handleChangeMsg(user, args string) string {
	dishLook.Lock()
	defer dishLook.Unlock()
	defer saveToJSON()

	if currentDish == nil {
		data.DishHistory = append(data.DishHistory, dish{Date: next(conf.cookingDay)})
		currentDish = &data.DishHistory[len(data.DishHistory)-1]
	}

	fmt.Println(args)

	if strings.HasPrefix(args, "ich koche") {
		currentDish.Cook = user
		args = strings.Trim(strings.TrimPrefix(args, "ich koche"), " ")
		if strings.HasPrefix(args, "am") {
			args = strings.TrimPrefix(args, "am")
			if weekday, err := parseWeekday(strings.Trim(args, " ")); err == nil {
				currentDish.Date = next(weekday)
			} else {
				return "Entschuldige ich habe den Wochentag nicht verstanden."
			}
		} else if args != "" {
			currentDish.DishName = args
		}
		msg := fmt.Sprintf("<@%s> kocht am %d %s", currentDish.Cook, currentDish.Date.Day(), currentDish.Date.Month())
		if currentDish.DishName != "" {
			msg += " " + currentDish.DishName
		}
		return msg + "."

	} else if strings.Contains(args, "ich helfe") {
		if currentDish.Cook != user {
			currentDish.Helper = user
			return fmt.Sprintf("<@%s> hilft.", currentDish.Helper)
		} else {
			return fmt.Sprintf("<@%s> du kannst nicht helfen du kochst.", user)
		}
	} else if found := userRegex.FindStringSubmatch(args); len(found) == 2 {
		args := strings.Trim(strings.TrimPrefix(args, found[0]), " ")
		if args == "kocht" {
			currentDish.Cook = found[1]
		} else if args == "hilft" {
			if currentDish.Cook != found[1] {
				currentDish.Helper = found[1]
			} else {
				return fmt.Sprintf("<@%s> du kannst nicht helfen du kochst.", found[1])
			}
		} else {
			return "Das habe ich nicht verstanden."
		}
		return "ok"

	} else if strings.Contains(args, "fällt aus") {
		date := next(conf.cookingDay)
		for !date.After(currentDish.Date) {
			date = date.AddDate(0, 0, 7)
		}
		currentDish.Date = date
		return fmt.Sprintf("schade!\nich habe es auf den %d %s verschoben.", date.Day(), date.Month())
	} else if strings.Contains(args, ":thumbsup:") || strings.Contains(args, ":+1:") {
		if currentDish.Voted != nil {
			currentDish.Voted[user] = true
			return "ok"
		}
		return "Aktuell findet keine Abstimmung ab"
	} else if strings.Contains(args, ":thumbsdown:") || strings.Contains(args, ":-1:") {
		if currentDish.Voted != nil {
			currentDish.Voted[user] = false
			return "ok"
		}
		return "Aktuell findet keine Abstimmung ab"
	} else {
		fmt.Println(found)
		return "benutze:\nwer kocht?\nwas wird gekocht?\nwann wird gekocht?\nich koche.\nich koche am Montag/Dienstag/Mitwoch/Donnerstag/Freitag.\nich kochen Nudeln mit Tomatensoße.\n<@user> kocht.\n<@user> hilft.\ndieses mal fällt aus"
	}
}
func main() {
	err := loadFromJSON()
	if err != nil {
		panic(err)
	}

	api = slack.New(conf.Token)

	resp, err := api.AuthTest()
	if err != nil {
		panic(fmt.Errorf("Invalide Auth: %s", err))
	}
	if resp == nil {
		panic("Empty auth test response")
	}
	bot = resp.UserID
	prefix := "<@" + resp.UserID + ">"

	fmt.Println("Hot & Spicy Bot starting...")

	rtm := api.NewRTM()

	go rtm.ManageConnection()
	time.Sleep(time.Second * 5)

	fmt.Println(prefix)

	err = update()
	if err != nil {
		log.Println(err)
	}

	go func() {
		t := time.NewTicker(5 * time.Minute)
		for _ = range t.C {
			err := update()
			if err != nil {
				log.Println(err)
			}
		}
	}()

	for msg := range rtm.IncomingEvents {
		//fmt.Print("Event Received: ")
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			fmt.Printf("ev: %#v\n", ev)
			if !strings.HasPrefix(ev.Msg.Text, prefix) {
				//fmt.Println("Message NOT for me: '" + ev.Text + "'")
				continue
			}
			fmt.Println("Message for me: '" + ev.Text + "'")
			args := strings.Trim(strings.TrimPrefix(ev.Text, prefix), " !?.")

			//Questions
			if strings.Contains(args, "wer kocht") {
				if currentDish == nil || currentDish.Cook == "" {
					api.SendMessage(ev.Channel, slack.MsgOptionText("Aktuell niemand.", false))
				} else {
					msg := fmt.Sprintf("<@%s> kocht", currentDish.Cook)
					if currentDish.Helper != "" {
						msg += fmt.Sprintf(" und <@%s> hilft", currentDish.Helper)
					}
					api.SendMessage(ev.Channel, slack.MsgOptionText(msg+".", false))
				}
				break

			} else if strings.Contains(args, "wann wird gekocht") {
				if currentDish == nil {
					api.SendMessage(ev.Channel, slack.MsgOptionText("Aktuell nicht.", false))
				} else {
					api.SendMessage(ev.Channel, slack.MsgOptionText(fmt.Sprintf("<@%s> kocht und <@%s> hilft.", currentDish.Cook, currentDish.Helper), false))
				}
				break

			} else if strings.Contains(args, "was wird gekocht") {
				if currentDish == nil {
					api.SendMessage(ev.Channel, slack.MsgOptionText("Aktuell nichts.", false))
				} else if currentDish.DishName != "" {
					api.SendMessage(ev.Channel, slack.MsgOptionText(fmt.Sprintf("<@%s> kocht %s.", currentDish.Cook, currentDish.DishName), false))
				} else {
					api.SendMessage(ev.Channel, slack.MsgOptionText(fmt.Sprintf("<@%s> hat noch nicht gesagt was er kochen möchte.", currentDish.Cook), false))
				}
				break
			}
			api.SendMessage(ev.Channel, slack.MsgOptionText(handleChangeMsg(ev.User, args), false))

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
