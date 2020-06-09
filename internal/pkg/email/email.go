package email

import (
	"bytes"
	"errors"
	"time"

	"github.com/ic3network/mccs-alpha-api/global"
	"github.com/ic3network/mccs-alpha-api/internal/app/types"
	"github.com/ic3network/mccs-alpha-api/internal/pkg/template"
	"github.com/ic3network/mccs-alpha-api/util/l"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var e *Email

func init() {
	global.Init()
	e = New()
}

// Email is a prioritized configuration registry.
type Email struct {
	serverAddr string
	from       *mail.Email
	client     *sendgrid.Client
}

// New returns an initialized Email instance.
func New() *Email {
	e := new(Email)
	e.serverAddr = viper.GetString("url")
	// Always send from MCCS
	e.from = mail.NewEmail(viper.GetString("email_from"), viper.GetString("sendgrid.sender_email"))
	e.client = sendgrid.NewSendClient(viper.GetString("sendgrid.key"))
	return e
}

// emailData contains all the information to compose an email.
type emailData struct {
	receiver      string
	receiverEmail string
	replyToName   string
	replyToEmail  string
	subject       string
	text          string
	html          string
}

func (e *Email) send(d emailData) error {
	if d.receiver == "" || d.receiverEmail == "" {
		return errors.New("receiver is empty")
	}

	to := mail.NewEmail(d.receiver, d.receiverEmail)
	message := mail.NewSingleEmail(e.from, d.subject, to, d.text, d.html)
	if d.replyToEmail != "" && d.replyToName != "" {
		replyTo := mail.NewEmail(d.replyToName, d.replyToEmail)
		message.SetReplyTo(replyTo)
	}

	info, err := e.client.Send(message)
	if err != nil {
		l.Logger.Error("error sending email", zap.String("info", info.Body))
		return err
	}
	return nil
}

// External APIs

type WelcomeEmail struct {
	EntityName string
	Email      string
	Receiver   string
}

// SendWelcomeEmail sends the welcome email once a new account is created.
func SendWelcomeEmail(input WelcomeEmail) error {
	if !viper.GetBool("receive_email.signup_notifications") {
		return nil
	}
	return e.sendWelcomeEmail(input)
}
func (e *Email) sendWelcomeEmail(input WelcomeEmail) error {
	t, err := template.NewEmailView("welcome")
	if err != nil {
		return err
	}

	data := struct {
		EntityName string
	}{
		EntityName: input.EntityName,
	}

	var tpl bytes.Buffer
	if err := t.ExecuteTemplate(&tpl, "welcome", data); err != nil {
		return err
	}
	html := tpl.String()

	d := emailData{
		receiver:      input.Receiver,
		receiverEmail: input.Email,
		subject:       "Welcome to The Open Credit Network directory!",
		text:          "Welcome to The Open Credit Network directory!",
		html:          html,
	}

	if err := e.send(d); err != nil {
		return err
	}
	return nil
}

// SendThankYouEmail sends the thank you email once the user completes the trading member signup form.
func SendThankYouEmail(firstName, lastName, email string) error {
	return e.sendThankYouEmail(firstName, lastName, email)
}
func (e *Email) sendThankYouEmail(firstName, lastName, email string) error {
	t, err := template.NewEmailView("thankYou")
	if err != nil {
		return err
	}

	data := struct {
		FirstName string
	}{
		FirstName: firstName,
	}

	var tpl bytes.Buffer
	if err := t.ExecuteTemplate(&tpl, "thankYou", data); err != nil {
		return err
	}
	html := tpl.String()

	d := emailData{
		receiver:      firstName + " " + lastName,
		receiverEmail: email,
		subject:       "Thank You for Your Application",
		text:          "Thank You for Your Application",
		html:          html,
	}

	if err := e.send(d); err != nil {
		return err
	}
	return nil
}

// SendNewMemberSignupEmail sends the email to the OCN Admin email address.
func SendNewMemberSignupEmail(entityName, email string) error {
	return e.sendNewMemberSignupEmail(entityName, email)
}
func (e *Email) sendNewMemberSignupEmail(entityName, email string) error {
	d := emailData{
		receiver:      viper.GetString("email_from"),
		receiverEmail: viper.GetString("sendgrid.sender_email"),
		subject:       "New Trading Member Application",
		text:          "New Trading Member Application",
		html:          "Entity Name: " + entityName + ", Email Address: " + email,
	}
	if err := e.send(d); err != nil {
		return err
	}
	return nil
}

// SendResetEmail sends the reset email.
func SendResetEmail(receiver string, email string, token string) error {
	return e.sendResetEmail(receiver, email, token)
}
func (e *Email) sendResetEmail(receiver string, email string, token string) error {
	text := "Your password reset link is: " + e.serverAddr + "/password-reset/" + token
	d := emailData{
		receiver:      receiver,
		receiverEmail: email,
		subject:       "Password Reset",
		text:          text,
		html:          text,
	}
	err := e.send(d)
	if err != nil {
		return err
	}
	return nil
}

func AdminResetPassword(receiver string, email string, token string) error {
	return e.adminResetPassword(receiver, email, token)
}
func (e *Email) adminResetPassword(receiver string, email string, token string) error {
	text := "Your password reset link is: " + e.serverAddr + "/admin/password-reset/" + token
	d := emailData{
		receiver:      receiver,
		receiverEmail: email,
		subject:       "Password Reset",
		text:          text,
		html:          text,
	}
	err := e.send(d)
	if err != nil {
		return err
	}
	return nil
}

// SendDailyEmailList sends the matching tags for a user.
func SendDailyEmailList(entity *types.Entity, matchedTags *types.MatchedTags, lastNotificationSentDate time.Time) error {
	return e.sendDailyEmailList(entity, matchedTags, lastNotificationSentDate)
}
func (e *Email) sendDailyEmailList(entity *types.Entity, matchedTags *types.MatchedTags, lastNotificationSentDate time.Time) error {
	t, err := template.NewEmailView("dailyEmail")
	if err != nil {
		return err
	}

	data := struct {
		Entity                   *types.Entity
		MatchedOffers            map[string][]string
		MatchedWants             map[string][]string
		LastNotificationSentDate time.Time
		URL                      string
	}{
		Entity:                   entity,
		MatchedOffers:            matchedTags.MatchedOffers,
		MatchedWants:             matchedTags.MatchedWants,
		LastNotificationSentDate: lastNotificationSentDate,
		URL:                      viper.GetString("url"),
	}

	var tpl bytes.Buffer
	if err := t.ExecuteTemplate(&tpl, "dailyEmail", data); err != nil {
		return err
	}
	html := tpl.String()

	d := emailData{
		receiver:      entity.Name,
		receiverEmail: entity.Email,
		subject:       "Potential trades via the Open Credit Network",
		text:          "Good news! There are new matches on The Open Credit Network for your offers and/or wants. Please login to your account to view them: https://trade.opencredit.network",
		html:          html,
	}

	if err := e.send(d); err != nil {
		return err
	}
	return nil
}

// SendContactEntity sends the contact to the entity owner.
func SendContactEntity(receiver, receiverEmail, replyToName, replyToEmail, body string) error {
	return e.sendContactEntity(receiver, receiverEmail, replyToName, replyToEmail, body)
}
func (e *Email) sendContactEntity(receiver, receiverEmail, replyToName, replyToEmail, body string) error {
	d := emailData{
		receiver:      receiver,
		receiverEmail: receiverEmail,
		replyToName:   replyToName,
		replyToEmail:  replyToEmail,
		subject:       "Contact from OCN directory member",
		text:          body,
		html:          body,
	}
	err := e.send(d)
	if err != nil {
		return err
	}

	// Send a copy of the email to the sengrid: sender_email address.
	go func() {
		if !viper.GetBool("receive_email.trade_contact_emails") {
			return
		}
		d := emailData{
			receiver:      viper.GetString("email_from"),
			receiverEmail: viper.GetString("sendgrid.sender_email"),
			subject:       "Contact from OCN directory member " + replyToName + " to " + receiver,
			text:          body,
			html:          body,
		}
		err := e.send(d)
		if err != nil {
			l.Logger.Error("error sending email: ", zap.Error(err))
		}
	}()

	return nil
}

// SendSignupNotification sends an email notification as each new signup occurs.
func SendSignupNotification(entityName string, contactEmail string) error {
	return e.sendSignupNotification(entityName, contactEmail)
}
func (e *Email) sendSignupNotification(entityName string, contactEmail string) error {
	body := "Entity Name: " + entityName + ", Contact Email: " + contactEmail
	d := emailData{
		receiver:      viper.GetString("email_from"),
		receiverEmail: viper.GetString("sendgrid.sender_email"),
		subject:       "A new entity has been signed up!",
		text:          body,
		html:          body,
	}
	err := e.send(d)
	if err != nil {
		return err
	}
	return nil
}
