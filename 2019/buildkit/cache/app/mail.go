package app

import (
	"bytes"
	"fmt"
	"html/template"
	"log"

	"bitbucket.org/okteto/okteto/backend/model"
	"github.com/pkg/errors"
	mailgun "gopkg.in/mailgun/mailgun-go.v1"
)

const (
	inviteEmailTitle = "You've been invited to okteto!"

	inviteEmailBody = `You've been invited to participate in the {{.Project}} project in okteto. 

Please click  on the link below to create your account: 
{{.Subscribe}}
	
If you didn't expect this email please unsubscribe by clicking on the link below: 
{{.Unsubscribe}}
	
Thanks, 
The okteto team`

	inviteEmailBodyHTML = `
<!DOCTYPE html>
<html>
	<head>
		<title>You've been invited to participate in okteto.</title> 
	</head>
	<body>
<p>You've been invited to participate in the {{.Project}} project in okteto.</p>

<p>Please click <a href="{{.Subscribe}}">here</a> to create your account.</p>
	
<p>If you didn't expect this email please <a href="{{.Unsubscribe}}">unsubscribe here</a>.</p>
	
<p>Thanks,</p>
<p>The okteto team</p>
</body>
</html>`

	projectInviteEmailTitle = "You've been invited to collaborate in the %s project"

	projectInviteEmailBody = `Hi,
	
You've been invited to collaborate in the "{{.Project}}" project. Go to {{.URL}} to access it.

Thanks, 
The okteto team`

	projectInviteEmailBodyHTML = `
<!DOCTYPE html>
<html>
	<head>
		<title>You've been invited to collaborate!</title> 
	</head>
	<body>
<p>You've been invited to collaborate in the <a href="{{.URL}}">{{.Project}}</a> project.</p>
	
<p>Thanks,</p>
<p>The okteto team</p>
</body>
</html>`
)

// EmailProvider manages sending emails and the templates
type EmailProvider struct {
	FromEmail         string
	projectInvite     *template.Template
	projectInviteHTML *template.Template
	userInvite        *template.Template
	userInviteHTML    *template.Template
	Sender            EmailSender
}

// EmailSender is an interface used to send in-app emails
type EmailSender interface {
	send(from string, title string, body string, bodyHTML string, to ...string) error
}

// Mailgun sends emails via mailgun.com, implements the EmailSender interface
type Mailgun struct {
	Client mailgun.Mailgun
}

// NoopMail is a mail client that won't send messages, implements the EmailSender interface
type NoopMail struct{}

// Send wont'send a message
func (n *NoopMail) send(from string, title string, body string, bodyHTML string, to ...string) error {
	log.Printf("would've sent email from=%s to=%s with title=%s", from, to[0], title)
	return nil
}

// NewMail creates a new instance of the mail provider and initializes the templates. It will panic
// if a template is missing.
func NewMail(fromEmail string, sender EmailSender) *EmailProvider {
	e := EmailProvider{
		FromEmail: fromEmail,
		Sender:    sender,
	}

	e.userInvite = template.Must(template.New("userInvite").Parse(inviteEmailBody))
	e.userInviteHTML = template.Must(template.New("userInviteHTML").Parse(inviteEmailBodyHTML))
	e.projectInvite = template.Must(template.New("projectInvite").Parse(projectInviteEmailBody))
	e.projectInviteHTML = template.Must(template.New("projectInviteHTML").Parse(projectInviteEmailBodyHTML))

	return &e
}

// NewMailgunSender returns an instance of mailgun.Mailgun
func NewMailgunSender(apiKey, domain string) EmailSender {
	return &Mailgun{Client: mailgun.NewMailgun(domain, apiKey, "")}
}

func (e *EmailProvider) send(from, title, body, bodyHTML string, to string) error {
	return e.Sender.send(from, title, body, bodyHTML, to)
}

// sendInviteEmail sends the invite email template to the user using mailgun's API
func (e *EmailProvider) sendInviteEmail(email string, projectName string, inviteLink string, unsubscribeLink string) error {
	if email == "" {
		return errors.New(string(model.InvalidEmail))
	}

	if projectName == "" {
		return errors.New(string(model.InvalidProject))
	}

	data := struct {
		Project     string
		Subscribe   string
		Unsubscribe string
	}{
		projectName, inviteLink, unsubscribeLink,
	}

	buf := new(bytes.Buffer)
	err := e.userInvite.Execute(buf, data)
	if err != nil {
		return err
	}

	htmlBuf := new(bytes.Buffer)
	err = e.userInviteHTML.Execute(htmlBuf, data)
	if err != nil {
		return err
	}

	return e.send(e.FromEmail, inviteEmailTitle, buf.String(), htmlBuf.String(), email)
}

// sendProjectInviteEmail sends the project invite email template to the user using mailgun's API
func (e *EmailProvider) sendProjectInviteEmail(email string, projectName string, inviteLink string) error {
	if email == "" {
		return errors.New(string(model.InvalidEmail))
	}

	if projectName == "" {
		return errors.New(string(model.InvalidProject))
	}

	data := struct {
		Project string
		URL     string
	}{
		projectName, inviteLink,
	}

	buf := new(bytes.Buffer)
	err := e.projectInvite.Execute(buf, data)
	if err != nil {
		return err
	}

	title := fmt.Sprintf(projectInviteEmailTitle, projectName)
	htmlBuf := new(bytes.Buffer)
	err = e.projectInviteHTML.Execute(htmlBuf, data)
	if err != nil {
		return err
	}

	return e.send(e.FromEmail, title, buf.String(), htmlBuf.String(), email)
}

// Send sends an email using mailgun
func (m *Mailgun) send(from string, title string, body string, bodyHTML string, to ...string) error {
	message := m.Client.NewMessage(from, title, body, to...)
	message.SetHtml(bodyHTML)
	_, _, err := m.Client.Send(message)
	return err
}
