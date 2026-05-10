package handlers

import (
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	qrcode "github.com/skip2/go-qrcode"

	"psnadd/internal/config"
	"psnadd/internal/services/groups"
	"psnadd/internal/utils"
	"psnadd/internal/web"
)

type Handler struct {
	cfg    config.Config
	groups *groups.Service
	logger *slog.Logger
}

type newPageData struct {
	Error string
	Stats groups.Stats
}

type createdPageData struct {
	Group      groups.Group
	GroupURL   string
	AdminURL   string
	AdminToken string
	PIN        string
}

type groupPageData struct {
	Group         groups.Group
	Entries       []groups.Entry
	Sort          string
	AdminToken    string
	CanGroupAdmin bool
	PIN           string
	Error         string
	FormName      string
	FormOnlineID  string
	Created       *groups.AddEntryResult
}

type pinPageData struct {
	Group    groups.Group
	Redirect string
	Error    string
}

type uploadPageData struct {
	Group      groups.Group
	AdminToken string
	Error      string
	CSVText    string
	Result     *groups.BatchAddResult
}

type entryPageData struct {
	Group      groups.Group
	Entry      groups.Entry
	AdminToken string
	CanAdmin   bool
	Error      string
}

type messagePageData struct {
	Code    string
	Title   string
	Message string
	Back    string
}

func New(cfg config.Config, groupService *groups.Service, logger *slog.Logger) *Handler {
	return &Handler{cfg: cfg, groups: groupService, logger: logger}
}

func (h *Handler) Register(r *gin.Engine) {
	r.GET("/assets/app.css", web.CSS)
	r.GET("/qr.png", h.qr)
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusSeeOther, "/new") })
	r.GET("/new", h.newGroupPage)
	r.POST("/new", h.createGroup)
	r.POST("/g/:slug/unlock", h.unlockGroup)
	r.GET("/g/:slug", h.groupPage)
	r.POST("/g/:slug/entries", h.addEntry)
	r.GET("/g/:slug/upload", h.uploadPage)
	r.POST("/g/:slug/upload", h.uploadEntries)
	r.GET("/g/:slug/export.csv", h.exportEntries)
	r.POST("/g/:slug/pin/remove", h.removeGroupPIN)
	r.POST("/g/:slug/pin/update", h.updateGroupPIN)
	r.GET("/g/:slug/:online_id", h.entryPage)
	r.POST("/g/:slug/:online_id/pull", h.pullEntry)
	r.POST("/g/:slug/:online_id/remove", h.removeEntry)
	r.POST("/g/:slug/:online_id/ban", h.banEntry)
}

func (h *Handler) newGroupPage(c *gin.Context) {
	stats, err := h.groups.Stats(c.Request.Context())
	if err != nil {
		h.renderError(c, err, "/new")
		return
	}
	web.Render(c, http.StatusOK, "PSN Add / New", "new", newPageData{Stats: stats})
}

func (h *Handler) createGroup(c *gin.Context) {
	result, err := h.groups.CreateGroup(c.Request.Context(), h.cfg.PublicBaseURL, c.PostForm("name"), c.PostForm("visibility") == "private")
	if err != nil {
		if utils.WantsJSON(c) {
			utils.WriteError(c, err)
			return
		}
		stats, statsErr := h.groups.Stats(c.Request.Context())
		if statsErr != nil {
			h.renderError(c, statsErr, "/new")
			return
		}
		web.Render(c, utils.AsAppError(err).Status, "PSN Add / New", "new", newPageData{Error: utils.AsAppError(err).Message, Stats: stats})
		return
	}
	if utils.WantsJSON(c) {
		utils.WriteData(c, http.StatusCreated, result)
		return
	}
	h.setAdminCookie(c, result.Group, result.AdminToken)
	web.Render(c, http.StatusCreated, "PSN Add / Created", "created", createdPageData(result))
}

func (h *Handler) groupPage(c *gin.Context) {
	group, entries, sortBy, adminToken, canGroupAdmin, err := h.loadGroupEntries(c)
	if err != nil {
		h.renderError(c, err, "/new")
		return
	}
	if !canGroupAdmin && !h.ensureGroupAccess(c, group, "") {
		return
	}
	web.Render(c, http.StatusOK, group.Name, "group", groupPageData{
		Group:         group,
		Entries:       entries,
		Sort:          sortBy,
		AdminToken:    adminToken,
		CanGroupAdmin: canGroupAdmin,
	})
}

func (h *Handler) addEntry(c *gin.Context) {
	group, err := h.groups.GetGroup(c.Request.Context(), c.Param("slug"))
	if err != nil {
		h.renderError(c, err, "/new")
		return
	}
	adminToken := h.adminTokenForGroup(c, group)
	canGroupAdmin, err := h.groups.GroupAdmin(c.Request.Context(), group.ID, adminToken)
	if err != nil {
		h.renderError(c, err, "/g/"+group.Slug)
		return
	}
	if canGroupAdmin {
		h.setAdminCookie(c, group, adminToken)
	}
	if !canGroupAdmin && !h.ensureGroupAccess(c, group, "") {
		return
	}

	displayName := strings.TrimSpace(c.PostForm("display_name"))
	onlineID := strings.TrimSpace(c.PostForm("online_id"))
	h.logger.Info("entry add form parsed",
		"slug", group.Slug,
		"content_type", c.GetHeader("Content-Type"),
		"content_length", c.Request.ContentLength,
		"form_keys", formKeys(c),
		"display_name_len", len(displayName),
		"online_id_len", len(onlineID),
		"online_id", onlineID,
		"has_admin_query", strings.TrimSpace(c.Query("admin")) != "",
		"has_admin_form", strings.TrimSpace(c.PostForm("admin")) != "",
	)
	result, err := h.groups.AddEntryWithName(c.Request.Context(), h.cfg.PublicBaseURL, group, displayName, onlineID)
	if err != nil {
		entries, listErr := h.groups.ListEntries(c.Request.Context(), group.ID, sortFromRequest(c))
		if listErr != nil {
			h.renderError(c, listErr, "/g/"+group.Slug)
			return
		}
		web.Render(c, utils.AsAppError(err).Status, group.Name, "group", groupPageData{
			Group:         group,
			Entries:       entries,
			Sort:          sortFromRequest(c),
			AdminToken:    adminToken,
			CanGroupAdmin: canGroupAdmin,
			Error:         utils.AsAppError(err).Message,
			FormName:      displayName,
			FormOnlineID:  onlineID,
		})
		return
	}

	entries, err := h.groups.ListEntries(c.Request.Context(), group.ID, sortFromRequest(c))
	if err != nil {
		h.renderError(c, err, "/g/"+group.Slug)
		return
	}
	web.Render(c, http.StatusCreated, group.Name, "group", groupPageData{
		Group:         group,
		Entries:       entries,
		Sort:          sortFromRequest(c),
		AdminToken:    adminToken,
		CanGroupAdmin: canGroupAdmin,
		Created:       &result,
	})
}

func (h *Handler) unlockGroup(c *gin.Context) {
	group, err := h.groups.GetGroup(c.Request.Context(), c.Param("slug"))
	if err != nil {
		h.renderError(c, err, "/new")
		return
	}
	redirect := cleanGroupRedirect(c.PostForm("redirect"), group.Slug)
	ok, err := h.groups.VerifyPIN(c.Request.Context(), group.ID, c.PostForm("pin"))
	if err != nil {
		h.renderError(c, err, "/g/"+group.Slug)
		return
	}
	if !ok {
		web.Render(c, http.StatusForbidden, group.Name+" / PIN", "pin", pinPageData{
			Group:    group,
			Redirect: redirect,
			Error:    "PIN did not match",
		})
		return
	}
	h.setPINCookie(c, group, c.PostForm("pin"))
	c.Redirect(http.StatusSeeOther, redirect)
}

func (h *Handler) uploadPage(c *gin.Context) {
	group, adminToken, err := h.requireGroupAdmin(c)
	if err != nil {
		h.renderError(c, err, "/g/"+c.Param("slug"))
		return
	}
	web.Render(c, http.StatusOK, group.Name+" / Upload", "upload", uploadPageData{
		Group:      group,
		AdminToken: adminToken,
	})
}

func (h *Handler) uploadEntries(c *gin.Context) {
	group, adminToken, err := h.requireGroupAdmin(c)
	if err != nil {
		h.renderError(c, err, "/g/"+c.Param("slug"))
		return
	}

	rows, err := parseUploadRows(c)
	h.logger.Info("csv upload form parsed",
		"slug", group.Slug,
		"content_type", c.GetHeader("Content-Type"),
		"content_length", c.Request.ContentLength,
		"form_keys", formKeys(c),
		"csv_text_len", len(uploadCSVText(c)),
		"has_admin_query", strings.TrimSpace(c.Query("admin")) != "",
		"has_admin_form", strings.TrimSpace(c.PostForm("admin")) != "",
		"row_count", len(rows),
		"parse_error", err != nil,
	)
	if err != nil {
		web.Render(c, utils.AsAppError(err).Status, group.Name+" / Upload", "upload", uploadPageData{
			Group:      group,
			AdminToken: adminToken,
			Error:      utils.AsAppError(err).Message,
			CSVText:    uploadCSVText(c),
		})
		return
	}

	result := h.groups.AddEntriesBatch(c.Request.Context(), h.cfg.PublicBaseURL, group, rows)
	web.Render(c, http.StatusOK, group.Name+" / Upload", "upload", uploadPageData{
		Group:      group,
		AdminToken: adminToken,
		Result:     &result,
	})
}

func (h *Handler) exportEntries(c *gin.Context) {
	group, _, err := h.requireGroupAdmin(c)
	if err != nil {
		h.renderError(c, err, "/g/"+c.Param("slug"))
		return
	}
	entries, err := h.groups.ListEntries(c.Request.Context(), group.ID, groups.SortAZ)
	if err != nil {
		h.renderError(c, err, "/g/"+group.Slug)
		return
	}

	filename := slugFileName(group.Name) + ".csv"
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"Name (optional)", "PSN-ID"})
	for _, entry := range entries {
		_ = writer.Write([]string{entry.DisplayName, entry.OnlineID})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		h.renderError(c, utils.Internal("could not export CSV", err), "/g/"+group.Slug)
		return
	}
}

func (h *Handler) entryPage(c *gin.Context) {
	data, err := h.loadEntryPage(c)
	if err != nil {
		h.renderError(c, err, "/new")
		return
	}
	if !data.CanAdmin && !h.ensureGroupAccess(c, data.Group, c.Request.URL.RequestURI()) {
		return
	}
	web.Render(c, http.StatusOK, data.Entry.OnlineID, "entry", data)
}

func (h *Handler) removeGroupPIN(c *gin.Context) {
	group, adminToken, err := h.requireGroupAdmin(c)
	if err != nil {
		h.renderError(c, err, "/g/"+c.Param("slug"))
		return
	}
	if _, err := h.groups.ClearPIN(c.Request.Context(), group.ID); err != nil {
		h.renderError(c, err, "/g/"+group.Slug+"?admin="+adminToken)
		return
	}
	c.SetCookie(pinCookieName(group), "", -1, "/", "", false, true)
	c.Redirect(http.StatusSeeOther, "/g/"+group.Slug+"?admin="+adminToken)
}

func (h *Handler) updateGroupPIN(c *gin.Context) {
	group, adminToken, err := h.requireGroupAdmin(c)
	if err != nil {
		h.renderError(c, err, "/g/"+c.Param("slug"))
		return
	}
	updated, pin, err := h.groups.RotatePIN(c.Request.Context(), group.ID)
	if err != nil {
		h.renderError(c, err, "/g/"+group.Slug+"?admin="+adminToken)
		return
	}
	h.setPINCookie(c, updated, pin)

	entries, err := h.groups.ListEntries(c.Request.Context(), updated.ID, sortFromRequest(c))
	if err != nil {
		h.renderError(c, err, "/g/"+updated.Slug+"?admin="+adminToken)
		return
	}
	web.Render(c, http.StatusOK, updated.Name, "group", groupPageData{
		Group:         updated,
		Entries:       entries,
		Sort:          sortFromRequest(c),
		AdminToken:    adminToken,
		CanGroupAdmin: true,
		PIN:           pin,
	})
}

func (h *Handler) pullEntry(c *gin.Context) {
	data, err := h.loadEntryPage(c)
	if err != nil {
		h.renderError(c, err, "/new")
		return
	}
	if !data.CanAdmin {
		h.renderError(c, utils.Forbidden("admin token required"), "/g/"+data.Group.Slug+"/"+data.Entry.OnlineID)
		return
	}
	entry, err := h.groups.PullEntry(c.Request.Context(), data.Entry)
	if err != nil {
		h.renderError(c, err, "/g/"+data.Group.Slug+"/"+data.Entry.OnlineID)
		return
	}
	c.Redirect(http.StatusSeeOther, "/g/"+data.Group.Slug+"/"+entry.OnlineID+"?admin="+data.AdminToken)
}

func (h *Handler) removeEntry(c *gin.Context) {
	data, err := h.loadEntryPage(c)
	if err != nil {
		h.renderError(c, err, "/new")
		return
	}
	if !data.CanAdmin {
		h.renderError(c, utils.Forbidden("admin token required"), "/g/"+data.Group.Slug+"/"+data.Entry.OnlineID)
		return
	}
	if err := h.groups.RemoveEntry(c.Request.Context(), data.Entry); err != nil {
		h.renderError(c, err, "/g/"+data.Group.Slug+"/"+data.Entry.OnlineID)
		return
	}
	c.Redirect(http.StatusSeeOther, "/g/"+data.Group.Slug+"?admin="+data.AdminToken)
}

func (h *Handler) banEntry(c *gin.Context) {
	data, err := h.loadEntryPage(c)
	if err != nil {
		h.renderError(c, err, "/new")
		return
	}
	if !data.CanAdmin {
		h.renderError(c, utils.Forbidden("admin token required"), "/g/"+data.Group.Slug+"/"+data.Entry.OnlineID)
		return
	}
	if err := h.groups.BanEntry(c.Request.Context(), data.Group, data.Entry); err != nil {
		h.renderError(c, err, "/g/"+data.Group.Slug+"/"+data.Entry.OnlineID)
		return
	}
	c.Redirect(http.StatusSeeOther, "/g/"+data.Group.Slug+"?admin="+data.AdminToken)
}

func (h *Handler) qr(c *gin.Context) {
	text := strings.TrimSpace(c.Query("text"))
	if text == "" {
		c.Status(http.StatusBadRequest)
		return
	}
	png, err := qrcode.Encode(text, qrcode.Medium, 512)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	c.Data(http.StatusOK, "image/png", png)
}

func (h *Handler) loadGroupEntries(c *gin.Context) (groups.Group, []groups.Entry, string, string, bool, error) {
	group, err := h.groups.GetGroup(c.Request.Context(), c.Param("slug"))
	if err != nil {
		return groups.Group{}, nil, "", "", false, err
	}
	adminToken := h.adminTokenForGroup(c, group)
	canGroupAdmin, err := h.groups.GroupAdmin(c.Request.Context(), group.ID, adminToken)
	if err != nil {
		return groups.Group{}, nil, "", "", false, err
	}
	if canGroupAdmin {
		h.setAdminCookie(c, group, adminToken)
	}
	sortBy := sortFromRequest(c)
	entries, err := h.groups.ListEntries(c.Request.Context(), group.ID, sortBy)
	if err != nil {
		return groups.Group{}, nil, "", "", false, err
	}
	return group, entries, sortBy, adminToken, canGroupAdmin, nil
}

func (h *Handler) loadEntryPage(c *gin.Context) (entryPageData, error) {
	group, err := h.groups.GetGroup(c.Request.Context(), c.Param("slug"))
	if err != nil {
		return entryPageData{}, err
	}
	entry, err := h.groups.GetEntry(c.Request.Context(), group.ID, c.Param("online_id"))
	if err != nil {
		return entryPageData{}, err
	}

	adminToken := h.adminTokenForGroup(c, group)
	groupAdmin, err := h.groups.GroupAdmin(c.Request.Context(), group.ID, adminToken)
	if err != nil {
		return entryPageData{}, err
	}
	if groupAdmin {
		h.setAdminCookie(c, group, adminToken)
	}
	entryAdmin, err := h.groups.EntryAdmin(c.Request.Context(), entry.ID, adminToken)
	if err != nil {
		return entryPageData{}, err
	}

	return entryPageData{
		Group:      group,
		Entry:      entry,
		AdminToken: adminToken,
		CanAdmin:   groupAdmin || entryAdmin,
	}, nil
}

func (h *Handler) requireGroupAdmin(c *gin.Context) (groups.Group, string, error) {
	group, err := h.groups.GetGroup(c.Request.Context(), c.Param("slug"))
	if err != nil {
		return groups.Group{}, "", err
	}
	adminToken := h.adminTokenForGroup(c, group)
	ok, err := h.groups.GroupAdmin(c.Request.Context(), group.ID, adminToken)
	if err != nil {
		return groups.Group{}, "", err
	}
	if !ok {
		return groups.Group{}, "", utils.Forbidden("group admin token required")
	}
	h.setAdminCookie(c, group, adminToken)
	return group, adminToken, nil
}

func (h *Handler) ensureGroupAccess(c *gin.Context, group groups.Group, redirect string) bool {
	if !group.HasPIN {
		return true
	}
	if h.hasPINAccess(c, group) {
		return true
	}
	if redirect == "" {
		redirect = "/g/" + group.Slug
	}
	web.Render(c, http.StatusForbidden, group.Name+" / PIN", "pin", pinPageData{
		Group:    group,
		Redirect: redirect,
	})
	return false
}

func (h *Handler) hasPINAccess(c *gin.Context, group groups.Group) bool {
	pin := strings.TrimSpace(c.Query("pin"))
	if pin != "" {
		ok, err := h.groups.VerifyPIN(c.Request.Context(), group.ID, pin)
		if err == nil && ok {
			h.setPINCookie(c, group, pin)
			return true
		}
	}
	cookie, err := c.Cookie(pinCookieName(group))
	if err != nil {
		return false
	}
	ok, err := h.groups.VerifyPINHash(c.Request.Context(), group.ID, cookie)
	return err == nil && ok
}

func (h *Handler) setPINCookie(c *gin.Context, group groups.Group, pin string) {
	c.SetCookie(pinCookieName(group), utils.HashToken(pin), 60*60*24*30, "/", "", false, true)
}

func pinCookieName(group groups.Group) string {
	return "psn_add_pin_" + group.Slug[:16]
}

func (h *Handler) adminTokenForGroup(c *gin.Context, group groups.Group) string {
	token := formOrQuery(c, "admin")
	if token != "" {
		return token
	}
	cookie, err := c.Cookie(adminCookieName(group))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie)
}

func (h *Handler) setAdminCookie(c *gin.Context, group groups.Group, token string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}
	c.SetCookie(adminCookieName(group), token, 60*60*24*30, "/", "", false, true)
}

func adminCookieName(group groups.Group) string {
	return "psn_add_admin_" + group.Slug[:16]
}

func cleanGroupRedirect(raw, slug string) string {
	if strings.TrimSpace(raw) == "" {
		return "/g/" + slug
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.IsAbs() || !strings.HasPrefix(parsed.Path, "/g/"+slug) {
		return "/g/" + slug
	}
	if parsed.RawQuery != "" {
		return parsed.Path + "?" + parsed.RawQuery
	}
	return parsed.Path
}

func parseUploadRows(c *gin.Context) ([]groups.BatchAddRow, error) {
	reader, err := uploadReader(c)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true
	csvReader.FieldsPerRecord = -1

	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, utils.BadRequest("CSV could not be parsed")
	}

	rows := make([]groups.BatchAddRow, 0, len(records))
	for idx, record := range records {
		line := idx + 1
		if len(record) == 0 || blankRecord(record) {
			continue
		}
		if len(rows) == 0 && isUploadHeader(record) {
			continue
		}
		if len(record) != 2 {
			return nil, utils.BadRequest("CSV must have exactly two columns: NAME,PSN-ID")
		}
		rows = append(rows, groups.BatchAddRow{
			Line:        line,
			DisplayName: strings.TrimSpace(record[0]),
			OnlineID:    strings.TrimSpace(record[1]),
		})
	}
	if len(rows) == 0 {
		return nil, utils.BadRequest("CSV has no entries")
	}
	return rows, nil
}

func uploadReader(c *gin.Context) (io.ReadCloser, error) {
	text := uploadCSVText(c)
	if text == "" {
		c.Set("csv_text_empty", true)
	}
	if text == "" {
		return nil, utils.BadRequest("CSV file or pasted CSV is required")
	}
	return io.NopCloser(strings.NewReader(text)), nil
}

func uploadCSVText(c *gin.Context) string {
	return strings.TrimSpace(c.PostForm("csv_text"))
}

func formKeys(c *gin.Context) []string {
	if c.Request.Form == nil && c.Request.PostForm == nil {
		_ = c.Request.ParseForm()
	}
	keys := make([]string, 0, len(c.Request.PostForm))
	for key := range c.Request.PostForm {
		keys = append(keys, key)
	}
	return keys
}

func blankRecord(record []string) bool {
	for _, cell := range record {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func isUploadHeader(record []string) bool {
	if len(record) != 2 {
		return false
	}
	first := strings.ToLower(strings.TrimSpace(record[0]))
	second := strings.ToLower(strings.TrimSpace(record[1]))
	return (first == "name" || first == "name(optional)" || first == "name (optional)" || first == "name optional" || first == "display_name") &&
		(second == "psn-id" || second == "psn_id" || second == "psn id" || second == "online_id" || second == "online id")
}

func (h *Handler) renderError(c *gin.Context, err error, back string) {
	appErr := utils.AsAppError(err)
	web.Render(c, appErr.Status, "PSN Add / Error", "message", messagePageData{
		Code:    appErr.Code,
		Title:   http.StatusText(appErr.Status),
		Message: appErr.Message,
		Back:    back,
	})
}

func sortFromRequest(c *gin.Context) string {
	sortBy := strings.ToLower(strings.TrimSpace(c.DefaultQuery("sort", c.PostForm("sort"))))
	if sortBy == groups.SortAZ {
		return groups.SortAZ
	}
	return groups.SortRecent
}

func formOrQuery(c *gin.Context, key string) string {
	value := strings.TrimSpace(c.Query(key))
	if value != "" {
		return value
	}
	return strings.TrimSpace(c.PostForm(key))
}

var fileUnsafePattern = regexp.MustCompile(`[^a-z0-9_-]+`)

func slugFileName(value string) string {
	name := strings.ToLower(strings.TrimSpace(value))
	name = fileUnsafePattern.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	if name == "" {
		return "party"
	}
	return name
}
