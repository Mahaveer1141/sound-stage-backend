package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"time"
)

// baseLayout is the shared email layout. Content-specific templates inject their
// markup via the {{.Content}} slot, keeping duplication to zero.
const baseLayout = `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>{{.Title}}</title>
</head>

<body style="margin:0; padding:0;">
<table width="100%" cellpadding="0" cellspacing="0" role="presentation" style="background-color:#f2f4f6;">
	<tr>
		<td align="center" style="padding:30px;">
			<table width="100%" cellpadding="0" cellspacing="0" role="presentation"
				style="margin:0 auto; background-color:#ffffff; border-radius:6px;">
				<tr>
					<td style="padding:30px;">
						<table width="100%" cellpadding="0" cellspacing="0" role="presentation">
							<tr>
								<td align="center" style="padding-bottom:20px;">
									<img
										src="{{.LogoURL}}"
										alt="Logo"
										style="display:block; border:0; outline:none; text-decoration:none; height:100px; width:auto;"
									/>
								</td>
							</tr>
						</table>
						<table width="100%" cellpadding="0" cellspacing="0" role="presentation">
							<tr>
								<td style="font-size:16px; line-height:1.6;">
									{{.Content}}
								</td>
							</tr>
						</table>
					</td>
				</tr>
				<tr>
					<td align="center"
						style="padding:20px; font-size:12px; color:#666666; border-top:1px solid #f0f0f0;">
						© {{.Year}} - {{.AppName}}
					</td>
				</tr>
			</table>
		</td>
	</tr>
</table>
</body>
</html>`

type emailData struct {
	Title   string
	Content template.HTML
	Year    string
	LogoURL string
	AppName string
}

func renderWithBaseLayout(title string, content string) string {
	tmpl, err := template.New("base").Parse(baseLayout)
	if err != nil {
		panic(fmt.Sprintf("mailer: failed to parse base layout: %v", err))
	}

	var buf bytes.Buffer
	data := emailData{
		Title:   title,
		Content: template.HTML(content),
		Year:    time.Now().Format("2006"),
		LogoURL: "https://images.pexels.com/photos/34663573/pexels-photo-34663573.jpeg",
		AppName: "Sound Stage",
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		panic(fmt.Sprintf("mailer: failed to execute base layout: %v", err))
	}

	return buf.String()
}

// ---------------------------------------------------------------------------
// Content renderers — each returns ONLY the inner HTML for its email type.
// ---------------------------------------------------------------------------

func renderOTPEmailHTML(otp string) string {
	content := fmt.Sprintf(`<p style="margin:0 0 12px 0;">Hi,</p>
<p style="margin:0;">
	Your OTP is:
	<span style="font-size:18px; font-weight:bold;">%s</span>
</p>`, otp)

	return renderWithBaseLayout("OTP Email", content)
}
