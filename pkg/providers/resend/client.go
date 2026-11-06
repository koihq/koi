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

// SendFirstCheckReady brings back someone who left while their first check ran.
func (c *Client) SendFirstCheckReady(ctx context.Context, runID uuid.UUID, to, brand string) error {
	subject := "We asked AI about " + brand
	text, markup := firstCheckContent(c.publicURL, brand)

	_, err := c.client.Emails.SendWithOptions(ctx, &resend.SendEmailRequest{
		From:    c.from,
		To:      []string{to},
		Subject: subject,
		Text:    text,
		Html:    markup,
	}, &resend.SendEmailOptions{
		IdempotencyKey: "first-check-run/" + runID.String(),
	})

	return err
}

func firstCheckContent(publicURL, brand string) (string, string) {
	appURL := strings.TrimRight(publicURL, "/") + "/"
	safeBrand := html.EscapeString(brand)

	text := "Your first check for " + brand + " is ready.\n\n" +
		"We asked ChatGPT, Perplexity, Gemini and Google AI Overviews. See what they said: " + appURL + "\n\n" +
		"The Koi team"
	markup := `<!doctype html><html lang="en"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<title>Your first check is ready</title></head>` +
		`<body style="margin:0;padding:32px 24px;background:#f7f7f5;color:#171717;font-family:Arial,Helvetica,sans-serif">` +
		`<p style="margin:0 0 12px;font-size:20px;font-weight:600">Your first check for ` + safeBrand + ` is ready.</p>` +
		`<p style="margin:0 0 28px;font-size:14px;line-height:1.6;color:#656565">` +
		`We asked ChatGPT, Perplexity, Gemini and Google AI Overviews.</p>` +
		`<a href="` + html.EscapeString(appURL) + `" style="display:inline-block;background:#171717;color:#ffffff;` +
		`text-decoration:none;font-size:14px;font-weight:600;padding:13px 20px;border-radius:6px">See what they said</a>` +
		`<p style="margin:28px 0 0;font-size:14px;color:#656565">The Koi team</p></body></html>`

	return text, markup
}

// SendPasswordReset has no idempotency key: each retry carries a new token, which a replayed
// request would not deliver.
func (c *Client) SendPasswordReset(ctx context.Context, to, token string) error {
	resetURL := strings.TrimRight(c.publicURL, "/") + "/reset-password#" + token
	text, markup := passwordResetContent(resetURL)

	_, err := c.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    c.from,
		To:      []string{to},
		Subject: "Reset your Koi password",
		Text:    text,
		Html:    markup,
	})

	return err
}

func passwordResetContent(resetURL string) (string, string) {
	text := "Reset your Koi password.\n\n" +
		"Choose a new password: " + resetURL + "\n\n" +
		"This link works once and expires in 1 hour. " +
		"If you did not ask for it, you can ignore this email and your password stays the same.\n\n" +
		"The Koi team"
	markup := `<!doctype html><html lang="en"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<title>Reset your Koi password</title></head>` +
		`<body style="margin:0;padding:0;background:#f7f7f5;color:#171717;font-family:Arial,Helvetica,sans-serif">` +
		`<table role="presentation" cellpadding="0" cellspacing="0" width="100%"><tr><td align="center" style="padding:40px 20px">` +
		`<table role="presentation" cellpadding="0" cellspacing="0" width="100%" style="max-width:560px;background:#ffffff">` +
		`<tr><td style="padding:40px 40px 0;font-size:19px;font-weight:700;letter-spacing:-0.04em">Koi</td></tr>` +
		`<tr><td style="padding:44px 40px 40px"><table role="presentation" cellpadding="0" cellspacing="0" width="100%">` +
		`<tr><td style="font-size:30px;line-height:1.2;font-weight:600;letter-spacing:-0.035em">Reset your password</td></tr>` +
		`<tr><td style="padding-top:24px;font-size:16px;line-height:1.6">This link works once and expires in 1 hour.</td></tr>` +
		`<tr><td style="padding:28px 0 0"><a href="` + html.EscapeString(resetURL) +
		`" style="display:inline-block;background:#171717;color:#ffffff;text-decoration:none;` +
		`font-size:14px;font-weight:600;padding:13px 20px;border-radius:6px">Choose a new password</a></td></tr>` +
		`<tr><td style="padding-top:28px;font-size:14px;line-height:1.6;color:#656565">` +
		`If you did not ask for it, you can ignore this email and your password stays the same.</td></tr>` +
		`</table></td></tr></table>` +
		`<table role="presentation" cellpadding="0" cellspacing="0" width="100%" style="max-width:560px"><tr><td style="padding:24px 4px;color:#656565;font-size:12px">The Koi team</td></tr></table>` +
		`</td></tr></table></body></html>`

	return text, markup
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
