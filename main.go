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
	var tempExpenses = make(map[int64]map[string]string)

	bot.Handle("/setbudget", func(c telebot.Context) error {
		telegramID := c.Sender().ID
		userStates[telegramID] = "awaiting_budget"
		return c.Send("Будь ласка, введіть бажаний місячний бюджет:")
	})

	bot.Handle(telebot.OnText, func(c telebot.Context) error {
		telegramID := c.Sender().ID

		switch userStates[telegramID] {
		case "awaiting_budget":
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

		case "awaiting_amount":
			amount, err := strconv.ParseFloat(c.Text(), 64)
			if err != nil || amount <= 0 {
				return c.Send("Будь ласка, введіть коректну суму.")
			}

			tempExpenses[telegramID]["amount"] = c.Text()
			userStates[telegramID] = "awaiting_category"

			return c.Send("Вкажіть категорію витрати:")

		case "awaiting_category":
			category := strings.TrimSpace(c.Text())
			if category == "" {
				return c.Send("Категорія не може бути порожньою.")
			}

			tempExpenses[telegramID]["category"] = category
			userStates[telegramID] = "awaiting_note"

			return c.Send("Додайте коментар до витрати (або напишіть \"пропустити\"):")

		case "awaiting_note":
			note := strings.TrimSpace(c.Text())
			if strings.ToLower(note) == "пропустити" {
				note = ""
			}

			tempExpenses[telegramID]["note"] = note

			amount, _ := strconv.ParseFloat(tempExpenses[telegramID]["amount"], 64)
			category := tempExpenses[telegramID]["category"]

			userID, err := db.GetUserID(database, telegramID)
			if err != nil {
				delete(userStates, telegramID)
				delete(tempExpenses, telegramID)
				return c.Send("Ваш профіль не знайдено. Будь ласка, скористайтесь командою /start для реєстрації.")
			}

			err = db.AddExpense(database, userID, amount, category, note)
			if err != nil {
				delete(userStates, telegramID)
				delete(tempExpenses, telegramID)
				return c.Send("Сталася помилка при збереженні витрати. Спробуйте ще раз.")
			}

			monthlyExpenses, err := db.GetMonthlyExpenses(database, userID)
			if err != nil {
				delete(userStates, telegramID)
				delete(tempExpenses, telegramID)
				return c.Send("Сталася помилка при розрахунку місячних витрат.")
			}

			response := fmt.Sprintf("Витрату %.2f %s успішно додано!\nЗагальна сума витрат за поточний місяць: %.2f", amount, category, monthlyExpenses)

			budget, err := db.GetUserMonthlyBudget(database, telegramID)
			if err == nil && budget > 0 {
				remaining := budget - monthlyExpenses
				response += fmt.Sprintf("\nЗалишок у місячному бюджеті: %.2f", remaining)

				if remaining > 0 && remaining <= budget*0.1 {
					response += "\n⚠️ Увага! У вашому бюджеті залишилось менше 10%."
				}
			}

			delete(userStates, telegramID)
			delete(tempExpenses, telegramID)

			return c.Send(response)

		default:
			return nil
		}
	})

	bot.Handle("/addexpense", func(c telebot.Context) error {
		telegramID := c.Sender().ID

		tempExpenses[telegramID] = make(map[string]string)
		userStates[telegramID] = "awaiting_amount"

		return c.Send("Вкажіть суму витрати:")
	})

	bot.Handle(telebot.OnCallback, func(c telebot.Context) error {
		data := strings.TrimSpace(c.Data())
		const prefix = "set_currency:"

		if strings.HasPrefix(data, prefix) {
			currency := strings.TrimPrefix(data, prefix)
			telegramID := c.Sender().ID

			validCurrencies := map[string]bool{"USD": true, "EUR": true, "UAH": true, "PLN": true, "CAD": true}
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

	go func() {
		for {
			userIDs, err := db.GetUsersWithBudget(database)
			if err != nil {
				log.Printf("Помилка отримання користувачів для перевірки бюджету: %v", err)
				time.Sleep(1 * time.Hour)
				continue
			}

			for _, telegramID := range userIDs {
				remaining, percentageSpent, lastNotification, err := db.GetBudgetStatusWithNotification(database, telegramID)
				if err != nil {
					log.Printf("Помилка перевірки бюджету для користувача (%d): %v", telegramID, err)
					continue
				}

				currentYear, currentMonth, _ := time.Now().Date()
				notificationYear, notificationMonth, _ := lastNotification.Date()

				if percentageSpent >= 70 &&
					(notificationYear != currentYear || notificationMonth != currentMonth) {
					message := fmt.Sprintf("⚠️ Ви витратили %.2f%% вашого бюджету. Залишок: %.2f.", percentageSpent, remaining)
					_, err := bot.Send(&telebot.User{ID: telegramID}, message)
					if err != nil {
						log.Printf("Помилка відправки повідомлення користувачу (%d): %v", telegramID, err)
						continue
					}

					err = db.UpdateLastNotification(database, telegramID)
					if err != nil {
						log.Printf("Помилка оновлення часу нагадування для користувача (%d): %v", telegramID, err)
					}
				}
			}

			time.Sleep(1 * time.Hour)
		}
	}()

	log.Println("Бот запущено...")
	bot.Start()
}

func showCurrencyButtons(bot *telebot.Bot, c telebot.Context, message string) {
	currencies := &telebot.ReplyMarkup{}
	btnUSD := currencies.Data("USD", "set_currency:USD")
	btnEUR := currencies.Data("EUR", "set_currency:EUR")
	btnUAH := currencies.Data("UAH", "set_currency:UAH")
	btnPLN := currencies.Data("PLN", "set_currency:PLN")
	btnCAD := currencies.Data("CAD", "set_currency:CAD")
	currencies.Inline(
		currencies.Row(btnUSD, btnEUR, btnUAH, btnPLN, btnCAD),
	)

	bot.Send(c.Sender(), message, currencies)
}
