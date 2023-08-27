package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var tokenFile = flag.String("token", ".token", "Telegram token.")
var voucher string

func main() {
	flag.Parse()

	tPath, err := filepath.Abs(*tokenFile)
	fatalOnErr(err)

	t, err := os.ReadFile(tPath)
	fatalOnErr(err)

	bot, err := tgbotapi.NewBotAPI(strings.TrimSpace(string(t)))
	fatalOnErr(err)

	updCfg := tgbotapi.NewUpdate(0)
	updCfg.Timeout = 60
	bot.Debug = true

	log.Printf("Authorized on account %s\n", bot.Self.UserName)

	db, err := newDb("app.db")
	fatalOnErr(err)
	defer func() {
		_ = db.Close()
	}()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, os.Kill)

	stopChan := make(chan struct{})

	updChan := bot.GetUpdatesChan(updCfg)

	for {
		select {
		case sig := <-sigChan:
			log.Printf("got signal: %s", sig)
			close(stopChan)

			return
		case upd := <-updChan:
			handleUpdate(upd, db, bot)
		}
	}
}

func fatalOnErr(err error) {
	if err != nil {
		log.Printf("fatal error: %s", err)
		os.Exit(1)
	}
}

func handleUpdate(upd tgbotapi.Update, db *database, bot *tgbotapi.BotAPI) {
	if upd.Message == nil {
		log.Printf("got empty #%d update, skip\n", upd.UpdateID)
		return
	}

	m := strings.ToLower(upd.Message.Text)
	chatID := upd.Message.Chat.ID

	if chatID != 5004371578 && chatID != 841547487 {
		err := sendMessage(bot, chatID, "Можно только на ленинградской")
		if err != nil {
			log.Printf("Не смог отправить ответ на сообщение %s, err:%s", m, err)
		}
		return
	}
	switch {
	case m == "/start":
		err := sendMessage(bot, chatID, "Здесь нужно отсканировать ваучер и отправить его.\nСледующим сообщением ввести сумму, которую нужно снять с ваучера.\nCумма должна быть целочисленной(без копеек)")
		if err != nil {
			log.Printf("Не смог ответить на сообщение %s, err:%s", m, err)
		}
	case m == "2023111202" || m == "2023111201" || m == "v":
		voucher = m
		balance32, err := db.withdrawMoney(voucher, 0)
		balanceInt := int(balance32)
		balanceInt = 2000 - balanceInt
		balanceString := strconv.Itoa(balanceInt)
		err = sendMessage(bot, chatID, "Баланс :"+balanceString+"\nТеперь введите сумму без копеек, которую нужно снять с ваучера")
		if err != nil {
			log.Printf("Не смог ответить на сообщение %s, err: %s", m, err)
		}
	case m == "/stop":
		if voucher != "" {
			voucher = ""
			err := sendMessage(bot, chatID, "Закончил работу с ваучером")
			if err != nil {
				log.Printf("Не смог ответить на сообщение %s, err: %s", m, err)
			}
		} else {
			err := sendMessage(bot, chatID, "Работа с ваучером уже закончена")
			if err != nil {
				log.Printf("Не смог ответить на сообщение %s, err: %s", m, err)
			}
		}
	default:
		if voucher != "" {
			cash, err := strconv.ParseUint(m, 10, 32)
			if err != nil {
				log.Printf("Не смог преобразовать %s в uint32, err :%s", m, err)
				err = sendMessage(bot, chatID, "Проверьте правильно ли введена сумма! \nДолжно быть целое число!")
				return
			}
			cash32 := uint32(cash)
			cash32, err = db.withdrawMoney(voucher, cash32)
			cashInt := int(cash32)
			balance := 2000 - cashInt
			cashString := strconv.Itoa(balance)
			balanceString := strconv.Itoa(cashInt)
			if balance < 0 {
				err = sendMessage(bot, chatID, "Недостаточно средств, не хватает "+cashString+" Рублей\n"+cashString+" рублей нужно доплатить. С ваучера средства сняты")
			} else {
				err = sendMessage(bot, chatID, "Деньги успешно сняты - остаток по счету : "+cashString+" Рублей")
			}
			err = sendMessage(bot, chatID, "Закончить работу с этим ваучером?\nНажмите на /stop или введите сумму, которую еще нужно снять\nНедостаток по кассе по этому ваучеру дожен составлять: "+balanceString+" Рублей")
		}

	}
}

func sendMessage(b *tgbotapi.BotAPI, chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)

	_, err := b.Send(msg)
	return err
}
