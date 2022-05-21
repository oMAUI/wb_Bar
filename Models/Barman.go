package Models

type Barman struct {
	DrinkList *DrinkList
	ID        int `db:"id" json:"id"`
}

// TODO: хранить список напитков в структуре или передавать как параметр?

func (b *Barman) CreateDrink(drink Drink) {
	newDrink := Drink{
		Name:  drink.Name,
		Price: drink.Price,
		Ppm:   drink.Ppm,
	}

	b.DrinkList.NewDrink(newDrink)
}

func (b *Barman) GetDrinkLIst(drinkList DrinkList) []Drink {
	var list []Drink
	for _, val := range drinkList.list {
		list = append(list, val)
	}

	return list
}
