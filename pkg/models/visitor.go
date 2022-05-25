package models

import (
	"errors"
	"math"
	"time"
)

type Visitor struct {
	DrinkList *DrinkList
	ID        int       `db:"id" json:"id"`
	Login     string    `db:"login" json:"name"`
	Money     int       `db:"money" json:"money"`
	Ppm       float64   `db:"ppm" json:"ppm"`
	IsAlive   bool      `db:"is_alive" json:"is_alive"`
	LastDrink time.Time `db:"last_drink" json:"-"`
}

var (
	ErrNoMoney error = errors.New("no money")
	ErrDead    error = errors.New("died")
)

func (v *Visitor) UpdatePpm() {
	dur := time.Now().Sub(v.LastDrink)
	v.Ppm -= math.Round(dur.Minutes())
	if v.Ppm < 0 {
		v.Ppm = 0
	}
}

func (v *Visitor) AvailableDrinkList(list DrinkList) []Drink {
	var availableList []Drink
	for _, val := range list.list {
		if v.Money >= val.Price {
			availableList = append(availableList, val)
		}
	}

	return availableList
}

func (v *Visitor) BuyDrink(drink Drink) error {
	if drink.Price > v.Money {
		return ErrNoMoney
	}
	v.Money -= drink.Price
	v.Ppm += drink.Ppm
	if v.Ppm >= lethalDose {
		v.IsAlive = false
		return ErrDead
	}
	v.LastDrink = time.Now()
	return nil
}
