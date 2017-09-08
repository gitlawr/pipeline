package notification

import (
	"errors"
	"strings"

	gomail "gopkg.in/gomail.v2"
)

type EmailNotifier struct {
	Dialer *gomail.Dialer
}

func NewEmailNotifier(host string, port int, username string, password string) *EmailNotifier {
	d := gomail.NewDialer(host, port, username, password)
	return &EmailNotifier{Dialer: d}
}

func (en *EmailNotifier) SendMail(recepients string, subject string, body string) error {
	recepientArr := strings.SplitN(recepients, ",", -1)
	if len(recepientArr) == 0 {
		return errors.New("no recepient for notification email")
	}
	m := gomail.NewMessage()
	m.SetHeader("From", en.Dialer.Username)
	m.SetHeader("To", recepientArr...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)
	return en.Dialer.DialAndSend(m)
}
