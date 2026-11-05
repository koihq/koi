package resendprovider

import (
	"context"
	"html"
	"strings"

	"github.com/google/uuid"
	"github.com/resend/resend-go/v4"
)

type Client struct {
	client    *resend.Client
	from      string
	publicURL string
}

func New(apiKey, from, publicURL string) *Client {
	return &Client{client: resend.NewClient(apiKey), from: from, publicURL: publicURL}
}

func (c *Client) SendWelcome(ctx context.Context, userID uuid.UUID, to string) error {
	text, markup := welcomeContent(c.publicURL)

	_, err := c.client.Emails.SendWithOptions(ctx, &resend.SendEmailRequest{
		From:    c.from,
		To:      []string{to},
		Subject: "Welcome to Koi",
		Text:    text,
		Html:    markup,
	}, &resend.SendEmailOptions{
		IdempotencyKey: "welcome-user/" + userID.String(),
	})

	return err
}

func welcomeContent(publicURL string) (string, string) {
	text := "Welcome to Koi.\n\nSee how AI talks about your brand.\n\n" +
		"Koi shows you when answer engines mention or cite your brand, who they recommend instead, " +
		"and where to focus next.\n\nStart by adding your brand."
	button := ""

	if publicURL != "" {
		setupURL := strings.TrimRight(publicURL, "/") + "/setup"
		text += "\n\nSet up your brand: " + setupURL
		button = `<tr><td style="padding:28px 0 0"><a href="` + html.EscapeString(setupURL) +
			`" style="display:inline-block;background:#171717;color:#ffffff;text-decoration:none;` +
			`font-size:14px;font-weight:600;padding:13px 20px;border-radius:6px">Set up your brand</a></td></tr>`
	} else {
		text += "\n\nWhen you're ready, sign in and add your brand."
		button = `<tr><td style="padding-top:28px;font-size:14px;line-height:1.6;color:#656565">` +
			`When you're ready, sign in and add your brand.</td></tr>`
	}

	text += "\n\nThe Koi team"
	markup := `<!doctype html><html lang="en"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<title>Welcome to Koi</title></head>` +
		`<body style="margin:0;padding:0;background:#f7f7f5;color:#171717;font-family:Arial,Helvetica,sans-serif">` +
		`<span style="display:none;max-height:0;overflow:hidden;opacity:0">See how AI talks about your brand.</span>` +
		`<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr><td align="center" style="padding:40px 20px">` +
		`<table role="presentation" cellpadding="0" cellspacing="0" width="100%" style="max-width:560px;background:#ffffff">` +
		`<tr><td style="padding:40px 40px 0;font-size:19px;font-weight:700;letter-spacing:-0.04em">Koi</td></tr>` +
		`<tr><td style="padding:44px 40px 40px"><table role="presentation" cellpadding="0" cellspacing="0" width="100%">` +
		`<tr><td style="font-size:13px;color:#656565;padding-bottom:12px">Welcome to Koi</td></tr>` +
		`<tr><td style="font-size:30px;line-height:1.2;font-weight:600;letter-spacing:-0.035em">See how AI talks about your brand.</td></tr>` +
		`<tr><td style="padding-top:24px;font-size:16px;line-height:1.6">Koi shows you when answer engines mention or cite your brand, who they recommend instead, and where to focus next.</td></tr>` +
		`<tr><td style="padding-top:16px;font-size:16px;line-height:1.6">Start by adding your brand.</td></tr>` +
		button +
		`</table></td></tr></table>` +
		`<table role="presentation" cellpadding="0" cellspacing="0" width="100%" style="max-width:560px"><tr><td style="padding:24px 4px;color:#656565;font-size:12px">The Koi team</td></tr></table>` +
		`</td></tr></table></body></html>`

	return text, markup
}
