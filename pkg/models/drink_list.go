package models

import "errors"

type (
	DrinkList struct {
		list map[string]Drink `json:"list"`
	}

	Drink struct {
		Name  string  `db:"name" json:"name"`
		Price int     `db:"price" json:"price"`
		Ppm   float64 `db:"ppm" json:"ppm"`
	}
)

var ErrDrinkAlreadyExist error = errors.New("drink already exist")

func (dl *DrinkList) Init() {
	dl.list = make(map[string]Drink)
}

func (dl *DrinkList) DrinkContain(name string) bool {
	_, ok := dl.list[name]
	return ok
}

func (dl *DrinkList) NewDrink(drink Drink) {
	dl.list[drink.Name] = drink
}

func (dl *DrinkList) Drink(name string) *Drink {
	val, _ := dl.list[name]
	return &val
}

func (dl *DrinkList) DrinkList() map[string]Drink {
	return dl.list
}

func (dl *DrinkList) containDrink(name string) (*Drink, bool) {
	v, ok := dl.list[name]
	if ok {
		return &v, ok
	}

	return &Drink{}, false
}
