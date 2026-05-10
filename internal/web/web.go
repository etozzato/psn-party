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
	"gmail":       gmail,
	"outlook":     outlook,
	"hey":         hey,
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

func gmail(subject, body string) string {
	values := url.Values{}
	values.Set("view", "cm")
	values.Set("fs", "1")
	values.Set("su", subject)
	values.Set("body", body)
	return "https://mail.google.com/mail/?" + values.Encode()
}

func outlook(subject, body string) string {
	values := url.Values{}
	values.Set("subject", subject)
	values.Set("body", body)
	return "https://outlook.office.com/mail/deeplink/compose?" + values.Encode()
}

func hey(subject, body string) string {
	values := url.Values{}
	values.Set("subject", subject)
	values.Set("body", body)
	return "https://app.hey.com/messages/new?" + values.Encode()
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
    <div class="theme-picker" data-theme-picker>
      <button class="theme-button" type="button" aria-label="Select theme" aria-expanded="false" data-theme-toggle>
        <span class="theme-swatch" aria-hidden="true"></span>
        <span class="theme-label" data-theme-label>PHOSPHOR</span>
        <span class="theme-chevron" aria-hidden="true">v</span>
      </button>
      <div class="theme-menu" data-theme-menu hidden>
        <button class="theme-menu-item" type="button" data-palette="phosphor" aria-pressed="false"><span class="theme-check" data-palette-check>○</span><span>PHOSPHOR</span></button>
        <button class="theme-menu-item" type="button" data-palette="plastic" aria-pressed="false"><span class="theme-check" data-palette-check>○</span><span>PLASTIC</span></button>
        <button class="theme-menu-item" type="button" data-palette="deepNavy" aria-pressed="false"><span class="theme-check" data-palette-check>○</span><span>DEEP NAVY</span></button>
        <button class="theme-menu-item" type="button" data-palette="paper" aria-pressed="false"><span class="theme-check" data-palette-check>○</span><span>PAPER</span></button>
        <button class="theme-menu-item" type="button" data-palette="ink" aria-pressed="false"><span class="theme-check" data-palette-check>○</span><span>INK</span></button>
      </div>
    </div>
    <main class="shell">
      {{ .Content }}
    </main>
    <div class="loading-overlay" data-loading-overlay hidden>
      <div class="loading-box">
        <span class="spinner" aria-hidden="true"></span>
        <strong data-loading-message>CHECKING PSN</strong>
      </div>
    </div>
    <script>
      var palettes = {
        phosphor: {
          label: "PHOSPHOR",
          vars: { bg: "#071512", panel: "#0d211d", "panel-2": "#102a25", line: "#21483f", text: "#c8ffe0", muted: "#74aa93", accent: "#4ade80", hot: "#ff6fb1", warn: "#ffc857", danger: "#ff5c5c" }
        },
        plastic: {
          label: "PLASTIC",
          vars: { bg: "#ece8de", panel: "#ddd7c9", "panel-2": "#f7f3e9", line: "#b8ae9d", text: "#2a261e", muted: "#726a5c", accent: "#7f7864", hot: "#9b4d73", warn: "#9a6a15", danger: "#c73934" }
        },
        deepNavy: {
          label: "DEEP NAVY",
          vars: { bg: "#101820", panel: "#18222c", "panel-2": "#0b1117", line: "#2b465d", text: "#e3eaf0", muted: "#9fb0c1", accent: "#5fa8d3", hot: "#d678b6", warn: "#e0b95a", danger: "#ff6b6b" }
        },
        paper: {
          label: "PAPER",
          vars: { bg: "#f7f7f5", panel: "#ffffff", "panel-2": "#ededeb", line: "#c9c9c4", text: "#1d1d1d", muted: "#6d6d6d", accent: "#4f6f52", hot: "#9b3d6a", warn: "#9b6a00", danger: "#c33b35" }
        },
        ink: {
          label: "INK",
          vars: { bg: "#121212", panel: "#1c1c1c", "panel-2": "#000000", line: "#3a3a3a", text: "#ededed", muted: "#8a8a8a", accent: "#bfbfbf", hot: "#f07db4", warn: "#d8b85d", danger: "#ff6767" }
        }
      };
      var defaultPalette = "phosphor";
      var storageKey = "psn_add_theme_palette";

      function setPalette(id) {
        var palette = palettes[id] || palettes[defaultPalette];
        var root = document.documentElement;
        root.dataset.palette = palettes[id] ? id : defaultPalette;
        Object.keys(palette.vars).forEach(function (key) {
          root.style.setProperty("--" + key, palette.vars[key]);
        });
        document.querySelectorAll("[data-theme-label]").forEach(function (node) {
          node.textContent = palette.label;
        });
        document.querySelectorAll("[data-palette]").forEach(function (node) {
          var selected = node.dataset.palette === root.dataset.palette;
          node.classList.toggle("is-selected", selected);
          node.setAttribute("aria-pressed", selected ? "true" : "false");
          var check = node.querySelector("[data-palette-check]");
          if (check) check.textContent = selected ? "●" : "○";
        });
        var themeMeta = document.querySelector('meta[name="theme-color"]');
        if (themeMeta) themeMeta.setAttribute("content", palette.vars.bg);
        try { localStorage.setItem(storageKey, root.dataset.palette); } catch (_) {}
      }

      function initialPalette() {
        try { return localStorage.getItem(storageKey) || defaultPalette; } catch (_) { return defaultPalette; }
      }

      function initThemePicker() {
        var picker = document.querySelector("[data-theme-picker]");
        if (!picker) return;
        var toggle = picker.querySelector("[data-theme-toggle]");
        var menu = picker.querySelector("[data-theme-menu]");
        if (!toggle || !menu) return;

        function closeMenu() {
          menu.hidden = true;
          toggle.setAttribute("aria-expanded", "false");
        }

        function openMenu() {
          menu.hidden = false;
          toggle.setAttribute("aria-expanded", "true");
        }

        toggle.addEventListener("click", function () {
          if (menu.hidden) openMenu(); else closeMenu();
        });
        picker.querySelectorAll("[data-palette]").forEach(function (node) {
          node.addEventListener("click", function () {
            setPalette(node.dataset.palette || defaultPalette);
            closeMenu();
          });
        });
        document.addEventListener("click", function (event) {
          if (!picker.contains(event.target)) closeMenu();
        });
        document.addEventListener("keydown", function (event) {
          if (event.key === "Escape") closeMenu();
        });
      }

      setPalette(initialPalette());
      document.addEventListener("DOMContentLoaded", initThemePicker);

      document.addEventListener("submit", function (event) {
        var form = event.target;
        if (!form || !form.matches("[data-loading]")) return;
        if (form.dataset.submitting === "true") {
          event.preventDefault();
          return;
        }
        form.dataset.submitting = "true";
        var loadingMessage = form.dataset.loadingMessage || "WORKING";
        window.setTimeout(function () {
          var overlay = document.querySelector("[data-loading-overlay]");
          var message = document.querySelector("[data-loading-message]");
          if (message) message.textContent = loadingMessage;
          if (overlay) overlay.hidden = false;
          form.querySelectorAll("button").forEach(function (button) {
            button.setAttribute("aria-disabled", "true");
          });
        }, 0);
      });
      document.addEventListener("input", function (event) {
        var field = event.target;
        if (!field || !field.closest) return;
        var panel = field.closest("#new-entry");
        if (!panel) return;
        var error = panel.querySelector(".error");
        if (error) error.hidden = true;
      });
    </script>
  </body>
</html>
{{- end }}

{{ define "new" -}}
<header class="bar">
  <a class="brand" href="#">PSN PARTY</a>
  <span class="spacer"></span>
  <span class="chip"><a class="brand" href="/new">NEW PARTY</a></span>
</header>
<section class="panel">
  <h1>GROUP</h1>
  <div class="tallies">
    <span>{{ .Stats.Groups }} PARTIES</span>
    <span>{{ .Stats.Entries }} PSN IDS</span>
  </div>
  <form method="post" action="/new" class="stack">
    <label>
      <span>NAME</span>
      <input name="name" maxlength="80" autocomplete="off" autofocus required>
    </label>
    <div class="segmented" role="group" aria-label="Group visibility">
      <label>
        <input type="radio" name="visibility" value="public" checked>
        <span>PUBLIC</span>
      </label>
      <label>
        <input type="radio" name="visibility" value="private">
        <span>PRIVATE</span>
      </label>
    </div>
    <button type="submit">CREATE</button>
  </form>
  {{ if .Error }}<p class="error">{{ .Error }}{{ if .FormOnlineID }} · Received: {{ .FormOnlineID }}{{ end }}</p>{{ end }}
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
    {{ if .PIN }}<label><span>GROUP PIN</span><input readonly value="{{ .PIN }}"></label>{{ end }}
  </div>
  <p class="note">Keep the admin URL or secret key. This app does not send email; use your own email client if you want a backup.</p>
  <div class="action-row">
    <a class="button" href="{{ .AdminURL }}">OPEN ADMIN</a>
    {{ $subject := printf "PSN Add admin link: %s" .Group.Name }}
    {{ $body := printf "Group: %s\nGroup URL: %s\nAdmin URL: %s\nSecret key: %s" .Group.Name .GroupURL .AdminURL .AdminToken }}
    {{ if .PIN }}{{ $body = printf "%s\nGroup PIN: %s" $body .PIN }}{{ end }}
    <a class="button" href="{{ mailto $subject $body }}">EMAIL YOURSELF</a>
    <a class="button" href="{{ gmail $subject $body }}" target="_blank" rel="noreferrer">GMAIL</a>
    <a class="button" href="{{ outlook $subject $body }}" target="_blank" rel="noreferrer">OUTLOOK</a>
    <a class="button" href="{{ hey $subject $body }}" target="_blank" rel="noreferrer">HEY</a>
  </div>
</section>
{{- end }}

{{ define "pin" -}}
<header class="bar">
  <a class="brand" href="/new">PSN ADD</a>
  <span class="spacer"></span>
  <span class="chip">PRIVATE</span>
</header>
<section class="panel">
  <h1>PIN</h1>
  <p class="note">{{ .Group.Name }} is private.</p>
  <form method="post" action="/g/{{ .Group.Slug }}/unlock" class="inline pin-form">
    <input type="hidden" name="redirect" value="{{ .Redirect }}">
    <input name="pin" inputmode="numeric" pattern="[0-9]{5}" maxlength="5" placeholder="5 digit PIN" autocomplete="off" autofocus required>
    <button type="submit">OPEN</button>
  </form>
  {{ if .Error }}<p class="error">{{ .Error }}{{ if .FormOnlineID }} · Received: {{ .FormOnlineID }}{{ else }} · Received empty PSN ID{{ end }}</p>{{ end }}
</section>
{{- end }}

{{ define "group" -}}
<header class="bar sticky">
  <a class="brand" href="/g/{{ .Group.Slug }}{{ if .AdminToken }}?admin={{ .AdminToken }}{{ end }}">{{ .Group.Name }}</a>
  <span class="spacer"></span>
  <a class="chip {{ if eq .Sort "az" }}active{{ end }}" href="/g/{{ .Group.Slug }}?sort=az{{ if .AdminToken }}&admin={{ .AdminToken }}{{ end }}">A-Z</a>
  <a class="chip {{ if eq .Sort "recent" }}active{{ end }}" href="/g/{{ .Group.Slug }}?sort=recent{{ if .AdminToken }}&admin={{ .AdminToken }}{{ end }}">RECENT</a>
  <span class="chip">{{ len .Entries }} IDS</span>
  {{ if .CanGroupAdmin }}<form method="post" action="/g/{{ .Group.Slug }}/admin/off" class="action-form"><input type="hidden" name="redirect" value="/g/{{ .Group.Slug }}?sort={{ .Sort }}"><button class="chip admin-off" type="submit">ADMIN OFF</button></form>{{ end }}
  {{ if .CanGroupAdmin }}<a class="chip" href="/g/{{ .Group.Slug }}/export.csv?admin={{ .AdminToken }}">CSV</a><a class="chip" href="/g/{{ .Group.Slug }}/upload?admin={{ .AdminToken }}">UPLOAD</a>{{ end }}
  <a class="chip" href="#new-entry">NEW</a>
</header>

<section id="new-entry" class="panel compact">
  <form method="post" action="/g/{{ .Group.Slug }}/entries" class="inline" data-loading data-loading-message="CHECKING PSN">
    {{ if .AdminToken }}<input type="hidden" name="admin" value="{{ .AdminToken }}">{{ end }}
    <input name="display_name" placeholder="Name (optional)" autocomplete="off" maxlength="120" value="{{ .FormName }}">
    <input name="online_id" placeholder="PSN ID" autocomplete="off" minlength="3" maxlength="16" pattern="[A-Za-z][A-Za-z0-9_-]{2,15}" title="3-16 characters, starts with a letter, letters/numbers/underscore/hyphen only" required value="{{ .FormOnlineID }}">
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
    {{ $subject := printf "PSN Add entry admin link: %s" .Created.Entry.OnlineID }}
    {{ $body := printf "Group: %s\nPSN ID: %s\nEntry URL: %s\nAdmin URL: %s\nSecret key: %s" .Group.Name .Created.Entry.OnlineID .Created.EntryURL .Created.AdminURL .Created.AdminToken }}
    <a class="button" href="{{ .Created.AdminURL }}">OPEN ADMIN</a>
    <a class="button" href="{{ mailto $subject $body }}">EMAIL YOURSELF</a>
    <a class="button" href="{{ gmail $subject $body }}" target="_blank" rel="noreferrer">GMAIL</a>
    <a class="button" href="{{ outlook $subject $body }}" target="_blank" rel="noreferrer">OUTLOOK</a>
    <a class="button" href="{{ hey $subject $body }}" target="_blank" rel="noreferrer">HEY</a>
  </div>
</section>
{{ end }}

{{ if .CanGroupAdmin }}
<section class="panel compact">
  <h2>PRIVACY {{ if .Group.HasPIN }}PRIVATE{{ else }}PUBLIC{{ end }}</h2>
  {{ if .PIN }}
    <div class="links">
      <label><span>NEW GROUP PIN</span><input readonly value="{{ .PIN }}"></label>
    </div>
    <div class="action-row">
      {{ $subject := printf "PSN Add group PIN: %s" .Group.Name }}
      {{ $body := printf "Group: %s\nGroup URL: %s\nPIN: %s" .Group.Name (printf "/g/%s" .Group.Slug) .PIN }}
      <a class="button" href="{{ mailto $subject $body }}">EMAIL YOURSELF</a>
      <a class="button" href="{{ gmail $subject $body }}" target="_blank" rel="noreferrer">GMAIL</a>
      <a class="button" href="{{ outlook $subject $body }}" target="_blank" rel="noreferrer">OUTLOOK</a>
      <a class="button" href="{{ hey $subject $body }}" target="_blank" rel="noreferrer">HEY</a>
    </div>
  {{ end }}
  <div class="action-row">
    {{ if .Group.HasPIN }}
      <form method="post" action="/g/{{ .Group.Slug }}/pin/remove" class="action-form"><input type="hidden" name="admin" value="{{ .AdminToken }}"><button class="chip" type="submit">MAKE PUBLIC</button></form>
      <form method="post" action="/g/{{ .Group.Slug }}/pin/update" class="action-form"><input type="hidden" name="admin" value="{{ .AdminToken }}"><button class="chip" type="submit">UPDATE PIN</button></form>
    {{ else }}
      <form method="post" action="/g/{{ .Group.Slug }}/pin/update" class="action-form"><input type="hidden" name="admin" value="{{ .AdminToken }}"><button class="chip" type="submit">MAKE PRIVATE</button></form>
    {{ end }}
  </div>
</section>
{{ end }}

<section class="grid">
  {{ range .Entries }}
    <article class="card has-avatar">
      <a href="/g/{{ $.Group.Slug }}/{{ .OnlineID }}{{ if $.AdminToken }}?admin={{ $.AdminToken }}{{ end }}">
        {{ if .AvatarURL }}<img class="avatar-badge" src="{{ .AvatarURL }}" alt="">{{ else }}<span class="avatar-badge avatar-placeholder">PSN</span>{{ end }}
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
  <form method="post" action="/g/{{ .Group.Slug }}/admin/off" class="action-form"><input type="hidden" name="redirect" value="/g/{{ .Group.Slug }}"><button class="chip admin-off" type="submit">ADMIN OFF</button></form>
  <a class="chip" href="/g/{{ .Group.Slug }}?admin={{ .AdminToken }}">LIST</a>
  <a class="chip" href="/g/{{ .Group.Slug }}/export.csv?admin={{ .AdminToken }}">CSV</a>
  <span class="chip active">UPLOAD</span>
</header>

<section class="panel">
  <h1>UPLOAD</h1>
  <form method="post" action="/g/{{ .Group.Slug }}/upload?admin={{ .AdminToken }}" class="stack" data-loading data-loading-message="CHECKING PSN">
    <input type="hidden" name="admin" value="{{ .AdminToken }}">
    <label>
      <span>PASTE CSV</span>
      <textarea name="csv_text" rows="8" placeholder="Name (optional),PSN-ID" required>{{ .CSVText }}</textarea>
    </label>
    <button type="submit">IMPORT CSV</button>
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
          {{ $subject := printf "PSN Add entry admin link: %s" .OnlineID }}
          {{ $body := printf "PSN ID: %s\nAdmin URL: %s\nSecret key: %s" .OnlineID .AdminURL .AdminToken }}
          <span class="upload-actions">
            <a href="{{ .AdminURL }}">ADMIN</a>
            <a href="{{ mailto $subject $body }}">EMAIL</a>
            <a href="{{ gmail $subject $body }}" target="_blank" rel="noreferrer">GMAIL</a>
            <a href="{{ outlook $subject $body }}" target="_blank" rel="noreferrer">OUTLOOK</a>
            <a href="{{ hey $subject $body }}" target="_blank" rel="noreferrer">HEY</a>
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
    <form method="post" action="/g/{{ .Group.Slug }}/admin/off" class="action-form"><input type="hidden" name="redirect" value="/g/{{ .Group.Slug }}/{{ .Entry.OnlineID }}"><button class="chip admin-off" type="submit">ADMIN OFF</button></form>
    <form method="post" action="/g/{{ .Group.Slug }}/{{ .Entry.OnlineID }}/pull" class="action-form" data-loading data-loading-message="CHECKING PSN"><input type="hidden" name="admin" value="{{ .AdminToken }}"><button class="chip" type="submit">PULL</button></form>
    <form method="post" action="/g/{{ .Group.Slug }}/{{ .Entry.OnlineID }}/remove" class="action-form"><input type="hidden" name="admin" value="{{ .AdminToken }}"><button class="chip danger" type="submit">REMOVE</button></form>
    <form method="post" action="/g/{{ .Group.Slug }}/{{ .Entry.OnlineID }}/ban" class="action-form"><input type="hidden" name="admin" value="{{ .AdminToken }}"><button class="chip danger" type="submit">BAN</button></form>
  {{ end }}
</header>
<section class="solo">
  <article class="card large has-avatar">
    {{ if .Entry.AvatarURL }}<img class="avatar-badge" src="{{ .Entry.AvatarURL }}" alt="">{{ else }}<span class="avatar-badge avatar-placeholder">PSN</span>{{ end }}
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
button:disabled { cursor: wait; opacity: .64; }

.shell {
  width: min(1080px, 100%);
  margin: 0 auto;
  padding: 18px;
}

.theme-picker {
  position: fixed;
  right: 14px;
  bottom: 14px;
  z-index: 80;
}

.theme-button,
.theme-menu-item {
  transition: background-color 180ms ease, border-color 180ms ease, color 180ms ease, transform 180ms ease;
}

.theme-button {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 34px;
  padding: 8px 12px;
  border: 1px solid var(--line);
  border-radius: 999px;
  background: color-mix(in srgb, var(--panel-2) 88%, transparent);
  color: var(--text);
  box-shadow: 0 12px 28px rgba(0, 0, 0, .22);
}

.theme-button:hover,
.theme-menu-item:hover {
  transform: translateY(-1px);
}

.theme-swatch {
  width: 15px;
  height: 15px;
  border-radius: 999px;
  background: linear-gradient(135deg, var(--accent), var(--hot));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--text) 20%, transparent);
}

.theme-label {
  font-size: 11px;
}

.theme-chevron {
  color: var(--muted);
  font-size: 11px;
}

.theme-menu {
  position: absolute;
  right: 0;
  bottom: calc(100% + 8px);
  display: grid;
  gap: 6px;
  min-width: 176px;
  padding: 8px;
  border: 1px solid color-mix(in srgb, var(--accent) 34%, var(--line));
  border-radius: 8px;
  background: var(--panel);
  box-shadow: var(--shadow);
}

.theme-menu[hidden] { display: none; }

.theme-menu-item {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 8px;
  width: 100%;
  padding: 8px 10px;
  border-radius: 6px;
  color: var(--text);
  background: transparent;
  text-align: left;
}

.theme-menu-item.is-selected {
  background: color-mix(in srgb, var(--accent) 14%, transparent);
}

.theme-check {
  width: 14px;
  color: var(--accent);
  font-size: 12px;
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
.admin-off {
  min-height: 42px;
  padding: 10px 16px;
  border-color: color-mix(in srgb, var(--danger) 82%, var(--line));
  background: color-mix(in srgb, var(--danger) 18%, var(--panel-2));
  color: var(--danger);
  font-weight: 700;
}
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
.tallies {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 0 0 18px;
}
.tallies span {
  padding: 8px 10px;
  border: 1px solid var(--line);
  border-radius: 6px;
  color: var(--accent);
  background: var(--panel-2);
}
.inline { display: grid; grid-template-columns: minmax(130px, 1fr) minmax(130px, 1fr) auto; gap: 8px; }
.pin-form { grid-template-columns: minmax(130px, 1fr) auto; }
.segmented {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}
.segmented label { display: block; }
.segmented input { position: absolute; opacity: 0; pointer-events: none; }
.segmented span {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 44px;
  border: 1px solid var(--line);
  border-radius: 6px;
  background: var(--panel-2);
  color: var(--muted);
}
.segmented input:checked + span {
  border-color: color-mix(in srgb, var(--accent) 72%, var(--line));
  color: var(--accent);
}
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
.card.has-avatar { padding-top: 86px; }
.card.large.has-avatar { padding-top: 96px; }
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
.avatar-placeholder {
  display: grid;
  place-items: center;
  border-color: var(--line);
  color: var(--muted);
  font-size: 13px;
  font-weight: 700;
  box-shadow: none;
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
.loading-overlay {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: grid;
  place-items: center;
  padding: 18px;
  background: rgba(2, 8, 6, .72);
  backdrop-filter: blur(4px);
}
.loading-overlay[hidden] { display: none; }
.loading-box {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 58px;
  padding: 14px 18px;
  border: 1px solid color-mix(in srgb, var(--accent) 60%, var(--line));
  border-radius: 8px;
  background: var(--panel);
  color: var(--accent);
  box-shadow: var(--shadow);
}
.spinner {
  width: 24px;
  height: 24px;
  border: 3px solid var(--line);
  border-top-color: var(--accent);
  border-radius: 999px;
  animation: spin .8s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
@media (max-width: 680px) {
  .shell { padding: 10px; }
  .bar { flex-wrap: wrap; align-items: flex-start; }
  .brand { max-width: 100%; flex-basis: 100%; }
  .inline { grid-template-columns: 1fr; }
  .upload-row { grid-template-columns: 1fr; }
}
`
