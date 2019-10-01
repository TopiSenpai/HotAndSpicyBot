package main

import (
	"reflect"
	"testing"
	"time"
)

func Test_parseWeekday(t *testing.T) {
	type args struct {
		v string
	}
	tests := []struct {
		name    string
		args    args
		want    time.Weekday
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseWeekday(tt.args.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseWeekday() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseWeekday() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_loadFromJSON(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := loadFromJSON(); (err != nil) != tt.wantErr {
				t.Errorf("loadFromJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_nextCookinDay(t *testing.T) {
	tests := []struct {
		name       string
		now        time.Time
		cookingDay time.Weekday
		want       time.Time
	}{
		{name: "Sunday 24h30 before", now: time.Date(2019, 10, 5, 11, 30, 0, 0, time.Local), cookingDay: time.Sunday, want: time.Date(2019, 10, 6, 12, 0, 0, 0, time.Local)},
		{name: "Sunday 23h30 before", now: time.Date(2019, 10, 5, 12, 30, 0, 0, time.Local), cookingDay: time.Sunday, want: time.Date(2019, 10, 13, 12, 0, 0, 0, time.Local)},
		{name: "Monday 24h30 before", now: time.Date(2019, 10, 6, 11, 30, 0, 0, time.Local), cookingDay: time.Monday, want: time.Date(2019, 10, 7, 12, 0, 0, 0, time.Local)},
		{name: "Monday 00h30 before", now: time.Date(2019, 10, 7, 11, 30, 0, 0, time.Local), cookingDay: time.Monday, want: time.Date(2019, 10, 14, 12, 0, 0, 0, time.Local)},
		{name: "Monday 23:30 after", now: time.Date(2019, 10, 8, 11, 30, 0, 0, time.Local), cookingDay: time.Monday, want: time.Date(2019, 10, 14, 12, 0, 0, 0, time.Local)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now = func() time.Time { return tt.now }
			conf.cookingDay = tt.cookingDay
			if got := nextCookinDay(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("nextCookinDay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_saveToJSON(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := saveToJSON(); (err != nil) != tt.wantErr {
				t.Errorf("saveToJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_update(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update()
		})
	}
}

func Test_newDish(t *testing.T) {
	type args struct {
		cookingDate time.Time
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newDish(tt.args.cookingDate)
		})
	}
}

func Test_main(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			main()
		})
	}
}
