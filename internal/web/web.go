package web

import (
	"bytes"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

type Page struct {
	Title   string
	Content template.HTML
}

var templates = template.Must(template.New("pages").Funcs(template.FuncMap{
	"publicLabel": publicLabel,
	"qrText":      qrText,
	"titleText":   titleText,
	"mailto":      mailto,
	"lower":       strings.ToLower,
}).Parse(pageTemplates))

func Render(c *gin.Context, status int, title, view string, data any) {
	var content bytes.Buffer
	if err := templates.ExecuteTemplate(&content, view, data); err != nil {
		c.String(http.StatusInternalServerError, "template error: %v", err)
		return
	}

	var body bytes.Buffer
	page := Page{Title: title, Content: template.HTML(content.String())}
	if err := templates.ExecuteTemplate(&body, "layout", page); err != nil {
		c.String(http.StatusInternalServerError, "template error: %v", err)
		return
	}
	c.Data(status, "text/html; charset=utf-8", body.Bytes())
}

func CSS(c *gin.Context) {
	c.Data(http.StatusOK, "text/css; charset=utf-8", []byte(styles))
}

func publicLabel(value *bool) string {
	if value == nil {
		return "UNCHECKED"
	}
	if *value {
		return "PUBLIC"
	}
	return "PRIVATE"
}

func qrText(isPublic *bool, profileURL, onlineID string) string {
	if isPublic != nil && *isPublic {
		return profileURL
	}
	return onlineID
}

func titleText(displayName, onlineID string) string {
	if strings.TrimSpace(displayName) != "" {
		return displayName
	}
	return onlineID
}

func mailto(subject, body string) string {
	values := url.Values{}
	values.Set("subject", subject)
	values.Set("body", body)
	return "mailto:?" + values.Encode()
}

const pageTemplates = `
{{ define "layout" -}}
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>{{ .Title }}</title>
    <meta name="theme-color" content="#071512">
    <link rel="stylesheet" href="/assets/app.css">
  </head>
  <body>
    <main class="shell">
      {{ .Content }}
    </main>
  </body>
</html>
{{- end }}

{{ define "new" -}}
<header class="bar">
  <a class="brand" href="/new">PSN ADD</a>
  <span class="spacer"></span>
  <span class="chip">NEW GROUP</span>
</header>
<section class="panel">
  <h1>GROUP</h1>
  <form method="post" action="/new" class="stack">
    <label>
      <span>NAME</span>
      <input name="name" maxlength="80" autocomplete="off" autofocus required>
    </label>
    <button type="submit">CREATE</button>
  </form>
  {{ if .Error }}<p class="error">{{ .Error }}</p>{{ end }}
</section>
{{- end }}

{{ define "created" -}}
<header class="bar">
  <a class="brand" href="/new">PSN ADD</a>
  <span class="spacer"></span>
  <span class="chip">CREATED</span>
</header>
<section class="panel">
  <h1>{{ .Group.Name }}</h1>
  <div class="links">
    <label><span>GROUP URL</span><input readonly value="{{ .GroupURL }}"></label>
    <label><span>ADMIN URL</span><input readonly value="{{ .AdminURL }}"></label>
    <label><span>SECRET KEY</span><input readonly value="{{ .AdminToken }}"></label>
  </div>
  <p class="note">Keep the admin URL or secret key. This app does not send email; use your own email client if you want a backup.</p>
  <div class="action-row">
    <a class="button" href="{{ .AdminURL }}">OPEN ADMIN</a>
    <a class="button" href="{{ mailto (printf "PSN Add admin link: %s" .Group.Name) (printf "Group: %s\nGroup URL: %s\nAdmin URL: %s\nSecret key: %s" .Group.Name .GroupURL .AdminURL .AdminToken) }}">EMAIL YOURSELF</a>
  </div>
</section>
{{- end }}

{{ define "group" -}}
<header class="bar sticky">
  <a class="brand" href="/g/{{ .Group.Slug }}{{ if .AdminToken }}?admin={{ .AdminToken }}{{ end }}">{{ .Group.Name }}</a>
  <span class="spacer"></span>
  <a class="chip {{ if eq .Sort "az" }}active{{ end }}" href="/g/{{ .Group.Slug }}?sort=az{{ if .AdminToken }}&admin={{ .AdminToken }}{{ end }}">A-Z</a>
  <a class="chip {{ if eq .Sort "recent" }}active{{ end }}" href="/g/{{ .Group.Slug }}?sort=recent{{ if .AdminToken }}&admin={{ .AdminToken }}{{ end }}">RECENT</a>
  {{ if .CanGroupAdmin }}<a class="chip" href="/g/{{ .Group.Slug }}/upload?admin={{ .AdminToken }}">UPLOAD</a>{{ end }}
  <a class="chip" href="#new-entry">NEW</a>
</header>

<section id="new-entry" class="panel compact">
  <form method="post" action="/g/{{ .Group.Slug }}/entries" class="inline">
    {{ if .AdminToken }}<input type="hidden" name="admin" value="{{ .AdminToken }}">{{ end }}
    <input name="display_name" placeholder="Name (optional)" autocomplete="off" maxlength="120">
    <input name="online_id" placeholder="PSN ID" autocomplete="off" required>
    <button type="submit">ADD</button>
  </form>
  {{ if .Error }}<p class="error">{{ .Error }}</p>{{ end }}
</section>

{{ if .Created }}
<section class="panel compact">
  <h2>ENTRY ADMIN</h2>
  <div class="links">
    <label><span>ENTRY URL</span><input readonly value="{{ .Created.EntryURL }}"></label>
    <label><span>ADMIN URL</span><input readonly value="{{ .Created.AdminURL }}"></label>
    <label><span>SECRET KEY</span><input readonly value="{{ .Created.AdminToken }}"></label>
  </div>
  <div class="action-row">
    <a class="button" href="{{ mailto (printf "PSN Add entry admin link: %s" .Created.Entry.OnlineID) (printf "Group: %s\nPSN ID: %s\nEntry URL: %s\nAdmin URL: %s\nSecret key: %s" .Group.Name .Created.Entry.OnlineID .Created.EntryURL .Created.AdminURL .Created.AdminToken) }}">EMAIL YOURSELF</a>
  </div>
</section>
{{ end }}

<section class="grid">
  {{ range .Entries }}
    <article class="card">
      <a href="/g/{{ $.Group.Slug }}/{{ .OnlineID }}{{ if $.AdminToken }}?admin={{ $.AdminToken }}{{ end }}">
        {{ if .AvatarURL }}<img class="avatar-badge" src="{{ .AvatarURL }}" alt="">{{ end }}
        <img class="qr" src="/qr.png?text={{ qrText .IsPublic .ProfileURL .OnlineID }}" alt="">
        <div class="card-row">
          <strong>{{ titleText .DisplayName .OnlineID }}</strong>
          <span class="status {{ lower (publicLabel .IsPublic) }}">{{ publicLabel .IsPublic }}</span>
        </div>
        {{ if .DisplayName }}<div class="handle">{{ .OnlineID }}</div>{{ end }}
      </a>
    </article>
  {{ else }}
    <div class="empty">NO ENTRIES</div>
  {{ end }}
</section>
{{- end }}

{{ define "upload" -}}
<header class="bar sticky">
  <a class="brand" href="/g/{{ .Group.Slug }}?admin={{ .AdminToken }}">{{ .Group.Name }}</a>
  <span class="spacer"></span>
  <a class="chip" href="/g/{{ .Group.Slug }}?admin={{ .AdminToken }}">LIST</a>
  <span class="chip active">UPLOAD</span>
</header>

<section class="panel">
  <h1>UPLOAD</h1>
  <form method="post" action="/g/{{ .Group.Slug }}/upload" enctype="multipart/form-data" class="stack">
    <input type="hidden" name="admin" value="{{ .AdminToken }}">
    <label>
      <span>CSV FILE</span>
      <input name="csv" type="file" accept=".csv,text/csv">
    </label>
    <label>
      <span>PASTE CSV</span>
      <textarea name="csv_text" rows="8" placeholder="Name (optional),PSN-ID&#10;Emanuele,psychic-disco2&#10;,MSnowjob"></textarea>
    </label>
    <button type="submit">IMPORT</button>
  </form>
  {{ if .Error }}<p class="error">{{ .Error }}</p>{{ end }}
</section>

{{ if .Result }}
<section class="panel compact">
  <h2>RESULT {{ .Result.Added }}/{{ len .Result.Rows }}</h2>
  <div class="upload-results">
    {{ range .Result.Rows }}
      <div class="upload-row {{ if .Added }}ok{{ else }}bad{{ end }}">
        <span>{{ .Line }}</span>
        <strong>{{ titleText .DisplayName .OnlineID }}</strong>
        <span>{{ .OnlineID }}</span>
        {{ if .Added }}
          <span class="upload-actions">
            <a href="{{ .AdminURL }}">ADMIN</a>
            <a href="{{ mailto (printf "PSN Add entry admin link: %s" .OnlineID) (printf "PSN ID: %s\nAdmin URL: %s\nSecret key: %s" .OnlineID .AdminURL .AdminToken) }}">EMAIL</a>
          </span>
        {{ else }}
          <em>{{ .Error }}</em>
        {{ end }}
      </div>
    {{ end }}
  </div>
</section>
{{ end }}
{{- end }}

{{ define "entry" -}}
<header class="bar sticky">
  <a class="brand" href="/g/{{ .Group.Slug }}{{ if .AdminToken }}?admin={{ .AdminToken }}{{ end }}">{{ .Group.Name }}</a>
  <span class="spacer"></span>
  <a class="chip" href="/g/{{ .Group.Slug }}{{ if .AdminToken }}?admin={{ .AdminToken }}{{ end }}#new-entry">NEW</a>
  {{ if .CanAdmin }}
    <form method="post" action="/g/{{ .Group.Slug }}/{{ .Entry.OnlineID }}/pull" class="action-form"><input type="hidden" name="admin" value="{{ .AdminToken }}"><button class="chip" type="submit">PULL</button></form>
    <form method="post" action="/g/{{ .Group.Slug }}/{{ .Entry.OnlineID }}/remove" class="action-form"><input type="hidden" name="admin" value="{{ .AdminToken }}"><button class="chip danger" type="submit">REMOVE</button></form>
    <form method="post" action="/g/{{ .Group.Slug }}/{{ .Entry.OnlineID }}/ban" class="action-form"><input type="hidden" name="admin" value="{{ .AdminToken }}"><button class="chip danger" type="submit">BAN</button></form>
  {{ end }}
</header>
<section class="solo">
  <article class="card large">
    {{ if .Entry.AvatarURL }}<img class="avatar-badge" src="{{ .Entry.AvatarURL }}" alt="">{{ end }}
    <img class="qr" src="/qr.png?text={{ qrText .Entry.IsPublic .Entry.ProfileURL .Entry.OnlineID }}" alt="">
    <h1>{{ titleText .Entry.DisplayName .Entry.OnlineID }}</h1>
    {{ if .Entry.DisplayName }}<div class="handle">{{ .Entry.OnlineID }}</div>{{ end }}
    <span class="status {{ lower (publicLabel .Entry.IsPublic) }}">{{ publicLabel .Entry.IsPublic }}</span>
    {{ if .Entry.IsPublic }}
      {{ if eq (publicLabel .Entry.IsPublic) "PUBLIC" }}
        <a class="button" href="{{ .Entry.ProfileURL }}" rel="noreferrer">OPEN PROFILE</a>
      {{ else }}
        <input readonly value="{{ .Entry.OnlineID }}">
      {{ end }}
    {{ else }}
      <input readonly value="{{ .Entry.OnlineID }}">
    {{ end }}
  </article>
  {{ if .Error }}<p class="error">{{ .Error }}</p>{{ end }}
</section>
{{- end }}

{{ define "message" -}}
<header class="bar">
  <a class="brand" href="/new">PSN ADD</a>
  <span class="spacer"></span>
  <span class="chip">{{ .Code }}</span>
</header>
<section class="panel">
  <h1>{{ .Title }}</h1>
  <p class="note">{{ .Message }}</p>
  {{ if .Back }}<a class="button" href="{{ .Back }}">BACK</a>{{ end }}
</section>
{{- end }}
`

const styles = `
:root {
  --bg: #071512;
  --panel: #0d211d;
  --panel-2: #102a25;
  --line: #21483f;
  --text: #c8ffe0;
  --muted: #74aa93;
  --accent: #4ade80;
  --hot: #ff6fb1;
  --warn: #ffc857;
  --danger: #ff5c5c;
  --shadow: 0 22px 60px rgba(0, 0, 0, .34);
}

* { box-sizing: border-box; }
html, body { min-height: 100%; }
body {
  margin: 0;
  background: var(--bg);
  color: var(--text);
  font-family: "Courier New", monospace;
  letter-spacing: 0;
}
a { color: inherit; text-decoration: none; }
button, input, textarea { font: inherit; }
button { cursor: pointer; }

.shell {
  width: min(1080px, 100%);
  margin: 0 auto;
  padding: 18px;
}

.bar {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 54px;
  margin-bottom: 18px;
}
.sticky {
  position: sticky;
  top: 0;
  z-index: 20;
  background: color-mix(in srgb, var(--bg) 94%, transparent);
  border-bottom: 1px solid var(--line);
  backdrop-filter: blur(10px);
}
.brand {
  font-size: 24px;
  color: var(--accent);
  max-width: 52vw;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.spacer { flex: 1; }
.chip, .button, button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 34px;
  padding: 8px 12px;
  border: 1px solid var(--line);
  border-radius: 6px;
  background: var(--panel-2);
  color: var(--text);
}
.chip.active, .button, button {
  border-color: color-mix(in srgb, var(--accent) 72%, var(--line));
  color: var(--accent);
}
.danger { color: var(--danger); }
.panel {
  padding: 22px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--panel);
  box-shadow: var(--shadow);
}
.compact { margin-bottom: 18px; padding: 14px; }
.stack, .links { display: grid; gap: 14px; }
.action-row { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 14px; }
.inline { display: grid; grid-template-columns: minmax(130px, 1fr) minmax(130px, 1fr) auto; gap: 8px; }
h1, h2 { margin: 0 0 16px; font-weight: 700; }
h1 { font-size: clamp(30px, 8vw, 72px); line-height: .9; color: var(--accent); }
h2 { font-size: 18px; color: var(--hot); }
label { display: grid; gap: 6px; }
label span { color: var(--muted); font-size: 13px; }
input, textarea {
  width: 100%;
  min-height: 44px;
  padding: 10px 12px;
  border: 1px solid var(--line);
  border-radius: 6px;
  background: #020806;
  color: var(--text);
}
textarea { resize: vertical; }
.note { color: var(--muted); max-width: 680px; }
.error { color: var(--danger); }
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(230px, 1fr));
  gap: 28px;
}
.card {
  position: relative;
  min-height: 310px;
  padding: 18px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--panel);
}
.card.large {
  width: min(460px, 100%);
  display: grid;
  gap: 14px;
}
.card.large h1 {
  margin-bottom: 0;
  font-size: clamp(28px, 7vw, 46px);
  line-height: 1;
  overflow-wrap: anywhere;
}
.qr {
  width: 100%;
  aspect-ratio: 1;
  object-fit: contain;
  border-radius: 4px;
  background: white;
}
.avatar-badge {
  position: absolute;
  top: 10px;
  right: 10px;
  width: 56px;
  height: 56px;
  border: 2px solid var(--accent);
  border-radius: 6px;
  background: var(--panel-2);
  object-fit: cover;
  box-shadow: 0 10px 22px rgba(0, 0, 0, .28);
}
.card.large .avatar-badge {
  width: 66px;
  height: 66px;
}
.card-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 14px;
}
.card-row strong { min-width: 0; overflow-wrap: anywhere; }
.handle {
  margin-top: 6px;
  color: var(--muted);
  overflow-wrap: anywhere;
}
.status {
  margin-left: auto;
  padding: 3px 7px;
  border: 1px solid var(--line);
  border-radius: 4px;
  color: var(--muted);
  font-size: 12px;
}
.status.public { color: var(--accent); }
.status.private { color: var(--warn); }
.solo {
  display: grid;
  justify-items: center;
}
.action-form { display: contents; }
.empty {
  padding: 38px;
  border: 1px dashed var(--line);
  color: var(--muted);
  text-align: center;
}
.upload-results { display: grid; gap: 8px; }
.upload-row {
  display: grid;
  grid-template-columns: 42px minmax(100px, 1fr) minmax(100px, 1fr) minmax(90px, 1fr);
  gap: 8px;
  align-items: center;
  padding: 8px;
  border: 1px solid var(--line);
  border-radius: 6px;
}
.upload-row.ok { border-color: color-mix(in srgb, var(--accent) 46%, var(--line)); }
.upload-row.bad { border-color: color-mix(in srgb, var(--danger) 46%, var(--line)); }
.upload-row em { color: var(--danger); font-style: normal; }
.upload-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
@media (max-width: 680px) {
  .shell { padding: 10px; }
  .bar { flex-wrap: wrap; align-items: flex-start; }
  .brand { max-width: 100%; flex-basis: 100%; }
  .inline { grid-template-columns: 1fr; }
  .upload-row { grid-template-columns: 1fr; }
}
`
