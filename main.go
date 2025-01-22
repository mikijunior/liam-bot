package main

import (
	"fmt"
	"log"
	"strconv"
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

		showCurrencyButtons(bot, c, "Ласкаво просимо до трекера витрат! Оберіть валюту:")
		return nil
	})

	bot.Handle("/setcurrency", func(c telebot.Context) error {
		showCurrencyButtons(bot, c, "Оберіть нову валюту:")
		return nil
	})

	var userStates = make(map[int64]string)

	bot.Handle("/setbudget", func(c telebot.Context) error {
		telegramID := c.Sender().ID
		userStates[telegramID] = "awaiting_budget"
		return c.Send("Будь ласка, введіть бажаний місячний бюджет:")
	})

	bot.Handle(telebot.OnText, func(c telebot.Context) error {
		telegramID := c.Sender().ID

		if userStates[telegramID] == "awaiting_budget" {
			budget, err := strconv.ParseFloat(c.Text(), 64)
			if err != nil || budget <= 0 {
				return c.Send("Будь ласка, введіть коректну суму бюджету.")
			}

			err = db.SetUserMonthlyBudget(database, telegramID, budget)
			if err != nil {
				log.Printf("Помилка збереження бюджету для користувача (%d): %v", telegramID, err)
				return c.Send("Сталася помилка при збереженні бюджету. Спробуйте ще раз.")
			}

			delete(userStates, telegramID)
			return c.Send(fmt.Sprintf("Місячний бюджет успішно встановлено: %.2f", budget))
		}

		return nil
	})

	bot.Handle(telebot.OnCallback, func(c telebot.Context) error {
		data := strings.TrimSpace(c.Data())
		const prefix = "set_currency:"

		if strings.HasPrefix(data, prefix) {
			currency := strings.TrimPrefix(data, prefix)
			telegramID := c.Sender().ID

			validCurrencies := map[string]bool{"USD": true, "EUR": true, "UAH": true, "PLN": true}
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

func showCurrencyButtons(bot *telebot.Bot, c telebot.Context, message string) {
	currencies := &telebot.ReplyMarkup{}
	btnUSD := currencies.Data("USD", "set_currency:USD")
	btnEUR := currencies.Data("EUR", "set_currency:EUR")
	btnUAH := currencies.Data("UAH", "set_currency:UAH")
	btnPLN := currencies.Data("PLN", "set_currency:PLN")
	currencies.Inline(
		currencies.Row(btnUSD, btnEUR, btnUAH, btnPLN),
	)

	bot.Send(c.Sender(), message, currencies)
}
