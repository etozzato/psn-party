package handlers

import (
	"encoding/csv"
	"errors"
	"io"
	"net/http"
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
}

type newPageData struct {
	Error string
}

type createdPageData struct {
	Group      groups.Group
	GroupURL   string
	AdminURL   string
	AdminToken string
}

type groupPageData struct {
	Group         groups.Group
	Entries       []groups.Entry
	Sort          string
	AdminToken    string
	CanGroupAdmin bool
	Error         string
	Created       *groups.AddEntryResult
}

type uploadPageData struct {
	Group      groups.Group
	AdminToken string
	Error      string
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

func New(cfg config.Config, groupService *groups.Service) *Handler {
	return &Handler{cfg: cfg, groups: groupService}
}

func (h *Handler) Register(r *gin.Engine) {
	r.GET("/assets/app.css", web.CSS)
	r.GET("/qr.png", h.qr)
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusSeeOther, "/new") })
	r.GET("/new", h.newGroupPage)
	r.POST("/new", h.createGroup)
	r.GET("/g/:slug", h.groupPage)
	r.POST("/g/:slug/entries", h.addEntry)
	r.GET("/g/:slug/upload", h.uploadPage)
	r.POST("/g/:slug/upload", h.uploadEntries)
	r.GET("/g/:slug/:online_id", h.entryPage)
	r.POST("/g/:slug/:online_id/pull", h.pullEntry)
	r.POST("/g/:slug/:online_id/remove", h.removeEntry)
	r.POST("/g/:slug/:online_id/ban", h.banEntry)
}

func (h *Handler) newGroupPage(c *gin.Context) {
	web.Render(c, http.StatusOK, "PSN Add / New", "new", newPageData{})
}

func (h *Handler) createGroup(c *gin.Context) {
	result, err := h.groups.CreateGroup(c.Request.Context(), h.cfg.PublicBaseURL, c.PostForm("name"))
	if err != nil {
		if utils.WantsJSON(c) {
			utils.WriteError(c, err)
			return
		}
		web.Render(c, utils.AsAppError(err).Status, "PSN Add / New", "new", newPageData{Error: utils.AsAppError(err).Message})
		return
	}
	if utils.WantsJSON(c) {
		utils.WriteData(c, http.StatusCreated, result)
		return
	}
	web.Render(c, http.StatusCreated, "PSN Add / Created", "created", createdPageData(result))
}

func (h *Handler) groupPage(c *gin.Context) {
	group, entries, sortBy, adminToken, err := h.loadGroupEntries(c)
	if err != nil {
		h.renderError(c, err, "/new")
		return
	}
	canGroupAdmin, err := h.groups.GroupAdmin(c.Request.Context(), group.ID, adminToken)
	if err != nil {
		h.renderError(c, err, "/g/"+group.Slug)
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

	result, err := h.groups.AddEntryWithName(c.Request.Context(), h.cfg.PublicBaseURL, group, c.PostForm("display_name"), c.PostForm("online_id"))
	if err != nil {
		entries, listErr := h.groups.ListEntries(c.Request.Context(), group.ID, sortFromRequest(c))
		if listErr != nil {
			h.renderError(c, listErr, "/g/"+group.Slug)
			return
		}
		web.Render(c, utils.AsAppError(err).Status, group.Name, "group", groupPageData{
			Group:      group,
			Entries:    entries,
			Sort:       sortFromRequest(c),
			AdminToken: formOrQuery(c, "admin"),
			Error:      utils.AsAppError(err).Message,
		})
		return
	}

	entries, err := h.groups.ListEntries(c.Request.Context(), group.ID, sortFromRequest(c))
	if err != nil {
		h.renderError(c, err, "/g/"+group.Slug)
		return
	}
	adminToken := formOrQuery(c, "admin")
	web.Render(c, http.StatusCreated, group.Name, "group", groupPageData{
		Group:      group,
		Entries:    entries,
		Sort:       sortFromRequest(c),
		AdminToken: adminToken,
		Created:    &result,
	})
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
	if err != nil {
		web.Render(c, utils.AsAppError(err).Status, group.Name+" / Upload", "upload", uploadPageData{
			Group:      group,
			AdminToken: adminToken,
			Error:      utils.AsAppError(err).Message,
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

func (h *Handler) entryPage(c *gin.Context) {
	data, err := h.loadEntryPage(c)
	if err != nil {
		h.renderError(c, err, "/new")
		return
	}
	web.Render(c, http.StatusOK, data.Entry.OnlineID, "entry", data)
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

func (h *Handler) loadGroupEntries(c *gin.Context) (groups.Group, []groups.Entry, string, string, error) {
	group, err := h.groups.GetGroup(c.Request.Context(), c.Param("slug"))
	if err != nil {
		return groups.Group{}, nil, "", "", err
	}
	sortBy := sortFromRequest(c)
	entries, err := h.groups.ListEntries(c.Request.Context(), group.ID, sortBy)
	if err != nil {
		return groups.Group{}, nil, "", "", err
	}
	return group, entries, sortBy, formOrQuery(c, "admin"), nil
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

	adminToken := formOrQuery(c, "admin")
	groupAdmin, err := h.groups.GroupAdmin(c.Request.Context(), group.ID, adminToken)
	if err != nil {
		return entryPageData{}, err
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
	adminToken := formOrQuery(c, "admin")
	ok, err := h.groups.GroupAdmin(c.Request.Context(), group.ID, adminToken)
	if err != nil {
		return groups.Group{}, "", err
	}
	if !ok {
		return groups.Group{}, "", utils.Forbidden("group admin token required")
	}
	return group, adminToken, nil
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
	fileHeader, err := c.FormFile("csv")
	if err == nil {
		file, err := fileHeader.Open()
		if err != nil {
			return nil, utils.BadRequest("CSV file could not be opened")
		}
		return file, nil
	}
	if !errors.Is(err, http.ErrMissingFile) {
		return nil, utils.BadRequest("CSV upload could not be read")
	}

	text := strings.TrimSpace(c.PostForm("csv_text"))
	if text == "" {
		return nil, utils.BadRequest("CSV file or pasted CSV is required")
	}
	return io.NopCloser(strings.NewReader(text)), nil
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
	value := strings.TrimSpace(c.PostForm(key))
	if value != "" {
		return value
	}
	return strings.TrimSpace(c.Query(key))
}
