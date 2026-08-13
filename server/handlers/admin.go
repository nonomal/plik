package handlers

import (
	"net/http"
	"strconv"

	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// GetUsers return users
func GetUsers(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {

	// Double check authorization
	if !ctx.IsAdmin() {
		ctx.Forbidden("you need administrator privileges")
		return
	}

	pagingQuery := ctx.GetPagingQuery()

	provider := req.URL.Query().Get("provider")

	var admin *bool
	if adminStr := req.URL.Query().Get("admin"); adminStr != "" {
		isAdmin := adminStr == "true"
		admin = &isAdmin
	}

	// Get users
	users, cursor, err := ctx.GetMetadataBackend().GetUsers(provider, admin, false, pagingQuery)
	if err != nil {
		ctx.InternalServerError("unable to get users : %s", err)
		return
	}

	// Count total users matching the filters
	// Note: not in the same transaction as the paginated query above, so the total
	// may be slightly inconsistent with the results if users are created/deleted
	// concurrently. This is acceptable for an admin UI counter.
	total, err := ctx.GetMetadataBackend().CountUsers(provider, admin)
	if err != nil {
		ctx.InternalServerError("unable to count users : %s", err)
		return
	}

	pagingResponse := common.NewPagingResponse(users, cursor).WithTotal(total)
	common.WriteJSONResponse(resp, pagingResponse)
}

// SearchUsers search users by query string
func SearchUsers(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {

	// Double check authorization
	if !ctx.IsAdmin() {
		ctx.Forbidden("you need administrator privileges")
		return
	}

	q := req.URL.Query().Get("q")
	if len(q) < 2 {
		ctx.BadRequest("search query must be at least 2 characters")
		return
	}

	provider := req.URL.Query().Get("provider")

	var admin *bool
	if adminStr := req.URL.Query().Get("admin"); adminStr != "" {
		isAdmin := adminStr == "true"
		admin = &isAdmin
	}

	limit := 5
	if limitStr := req.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if limit > 20 {
		limit = 20
	}

	users, err := ctx.GetMetadataBackend().SearchUsers(q, provider, admin, limit)
	if err != nil {
		ctx.InternalServerError("unable to search users : %s", err)
		return
	}

	common.WriteJSONResponse(resp, users)
}

// GetUploads return uploads
func GetUploads(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	// Double check authorization
	if !ctx.IsAdmin() {
		ctx.Forbidden("you need administrator privileges")
		return
	}

	pagingQuery := ctx.GetPagingQuery()
	sort := req.URL.Query().Get("sort")

	filters := parseBadgeFilters(req)
	filters.User = req.URL.Query().Get("user")
	filters.Token = req.URL.Query().Get("token")

	var uploads []*common.Upload
	var cursor *paginator.Cursor
	var err error

	if sort == "size" {
		uploads, cursor, err = ctx.GetMetadataBackend().GetUploadsSortedBySize(filters, true, pagingQuery)
		if err != nil {
			ctx.InternalServerError("unable to get uploads : %s", err)
			return
		}
	} else {
		uploads, cursor, err = ctx.GetMetadataBackend().GetUploads(filters, true, pagingQuery)
		if err != nil {
			ctx.InternalServerError("unable to get uploads : %s", err)
			return
		}
	}

	// Count total uploads matching the filters
	// Note: not in the same transaction as the paginated query above, so the total
	// may be slightly inconsistent with the results if uploads are cleaned up
	// concurrently. This is acceptable for an admin UI counter.
	total, err := ctx.GetMetadataBackend().CountUploads(filters)
	if err != nil {
		ctx.InternalServerError("unable to count uploads : %s", err)
		return
	}

	pagingResponse := common.NewPagingResponse(uploads, cursor).WithTotal(total)
	common.WriteJSONResponse(resp, pagingResponse)
}

// GetServerStatistics return the server statistics
func GetServerStatistics(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {

	// Double check authorization
	if !ctx.IsAdmin() {
		ctx.Forbidden("you need administrator privileges")
		return
	}

	// Get server statistics
	stats, err := ctx.GetMetadataBackend().GetServerStatistics()
	if err != nil {
		ctx.InternalServerError("unable to get server statistics : %s", err)
		return
	}

	common.WriteJSONResponse(resp, stats)
}
