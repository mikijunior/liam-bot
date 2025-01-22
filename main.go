package main

import (
	"log"
	"strings"
	"time"

	"expenses-tracker-bot/config"
	"expenses-tracker-bot/db"

	telebot "gopkg.in/telebot.v4"
)

func main() {
	cfg := config.LoadConfig()

	database := db.ConnectDB(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	defer database.Close()

	bot, err := telebot.NewBot(telebot.Settings{
		Token:  cfg.Token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatalf("Помилка створення Telegram бота: %v", err)
	}

	bot.Handle("/start", func(c telebot.Context) error {
		telegramID := c.Sender().ID

		err := db.AddUser(database, telegramID)
		if err != nil {
			log.Printf("Помилка запису користувача (%d): %v", telegramID, err)
			bot.Send(c.Sender(), "Сталася помилка при додаванні вас до бази.")
			return nil
		}

		currencies := &telebot.ReplyMarkup{}
		btnUSD := currencies.Data("USD", "set_currency:USD")
		btnEUR := currencies.Data("EUR", "set_currency:EUR")
		btnUAH := currencies.Data("UAH", "set_currency:UAH")
		currencies.Inline(
			currencies.Row(btnUSD, btnEUR, btnUAH),
		)

		bot.Send(c.Sender(), "Ласкаво просимо до трекера витрат! Оберіть валюту:", currencies)
		return nil
	})

	bot.Handle(telebot.OnCallback, func(c telebot.Context) error {
		data := strings.TrimSpace(c.Data())
	
		const prefix = "set_currency:"
	
		if strings.HasPrefix(data, prefix) {
			currency := strings.TrimPrefix(data, prefix)
			telegramID := c.Sender().ID
	
			validCurrencies := map[string]bool{"USD": true, "EUR": true, "UAH": true}
			if !validCurrencies[currency] {
				log.Printf("Отримано невідомий код валюти: %s", currency)
				return c.Respond(&telebot.CallbackResponse{Text: "Некоректна валюта. Спробуйте ще раз."})
			}

			err := db.SetUserCurrency(database, telegramID, currency)
			if err != nil {
				log.Printf("Помилка оновлення валюти для користувача (%d): %v", telegramID, err)
				return c.Respond(&telebot.CallbackResponse{Text: "Сталася помилка. Спробуйте ще раз."})
			}
	
			log.Printf("Користувач (%d) вибрав валюту: %s", telegramID, currency)
			c.Send("Валюта успішно збережена!")
			return c.Respond()
		}
	
		log.Printf("Отримано некоректний callback: %s", data)
		return c.Respond(&telebot.CallbackResponse{Text: "Некоректний запит."})
	})	

	log.Println("Бот запущено...")
	bot.Start()
}
