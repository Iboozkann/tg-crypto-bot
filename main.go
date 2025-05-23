package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var cryptoList = map[string]string{
	"bitcoin":  "BTC",
	"ethereum": "ETH",
	"solana":   "SOL",
	"ripple":   "XRP",
}

func main() {
	bot, err := tgbotapi.NewBotAPI("7509451523:AAGul5K56c_HIxxNQyJIoJwMWPlwm_f_tn0") // Замените на реальный токен!
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Бот запущен: @%s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			if update.CallbackQuery != nil {
				handleCallback(bot, update.CallbackQuery)
			}
			continue
		}

		chatID := update.Message.Chat.ID

		switch update.Message.Text {
		case "/start":
			sendMainMenu(bot, chatID)
		case "Курс криптовалют 📊":
			sendCryptoPrices(bot, chatID)
		case "Помощь ℹ️":
			sendHelp(bot, chatID)
		default:
			msg := tgbotapi.NewMessage(chatID, "Используйте кнопки меню или нажмите /start")
			bot.Send(msg)
		}
	}
}

func sendMainMenu(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "💰 <b>Криптотрекер</b>\nВыберите действие:")
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Курс криптовалют 📊"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Помощь ℹ️"),
		),
	)
	bot.Send(msg)
}

func sendHelp(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "ℹ️ <b>Помощь</b>\n\n"+
		"Этот бот показывает текущий курс популярных криптовалют.\n\n"+
		"Используйте кнопку <b>Курс криптовалют 📊</b> для получения актуальных цен.\n\n"+
		"Для возврата в меню нажмите /start")
	msg.ParseMode = "HTML"
	bot.Send(msg)
}

func handleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	if callback.Data == "refresh" {
		sendCryptoPrices(bot, callback.Message.Chat.ID, callback.Message.MessageID)
	}
}

func sendCryptoPrices(bot *tgbotapi.BotAPI, chatID int64, messageID ...int) {
	type Coin struct {
		Price float64 `json:"usd"`
	}

	coins := make([]string, 0, len(cryptoList))
	for coin := range cryptoList {
		coins = append(coins, coin)
	}
	apiUrl := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd", strings.Join(coins, ","))

	resp, err := http.Get(apiUrl)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при получении данных от CoinGecko 🚨"))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bot.Send(tgbotapi.NewMessage(chatID, "Сервис CoinGecko временно недоступен 🚨"))
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка обработки данных 🚨"))
		return
	}

	var data map[string]Coin
	if err := json.Unmarshal(body, &data); err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка формата данных 🚨"))
		return
	}

	var messageText strings.Builder
	messageText.WriteString("📈 <b>Текущий курс:</b>\n\n")

	for id, coin := range cryptoList {
		if priceData, exists := data[id]; exists {
			messageText.WriteString(fmt.Sprintf("• %s: <b>$%.2f</b>\n", coin, priceData.Price))
		} else {
			messageText.WriteString(fmt.Sprintf("• %s: <b>данные недоступны</b>\n", coin))
		}
	}
	messageText.WriteString("\n🔄 Обновлено: " + time.Now().Format("15:04:05"))

	// Создаем клавиатуру для кнопки обновления
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Обновить ♻️", "refresh"),
		),
	)

	if len(messageID) > 0 {
		// Редактируем существующее сообщение
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID[0], messageText.String())
		editMsg.ParseMode = "HTML"
		editMsg.ReplyMarkup = &inlineKeyboard
		_, err := bot.Send(editMsg)
		if err != nil {
			log.Printf("Ошибка при редактировании сообщения: %v", err)
		}
	} else {
		// Отправляем новое сообщение
		msg := tgbotapi.NewMessage(chatID, messageText.String())
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = inlineKeyboard
		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("Ошибка при отправке сообщения: %v", err)
		}
	}
}
