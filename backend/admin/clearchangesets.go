package admin

import (
	"context"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/changeset"
	"net/url"
	"strings"
)

type ClearChangesetsWidget struct {
	service changeset.Service
}

func (w ClearChangesetsWidget) Title() string {
	return "Clear User Changesets"
}

func (w ClearChangesetsWidget) HTML() string {
	return `<div>
		<form method="POST" action={{.Endpoint}}>
			<input type="text" class="form-control form-group" name="user_email" placeholder="User Email Address">

			<div class="form-check form-group">
				<input class="form-check-input" type="radio" name="operation" value="delete" id="clear_changeset_delete">
				<label class="form-check-label" for="clear_changeset_delete">Delete</label>
			</div>
			<div class="form-check form-group">
				<input class="form-check-input" type="radio" name="operation" value="mark_read" id="clear_changeset_mark_read">
				<label class="form-check-label" for="clear_changeset_mark_read">Mark Read</label>
			</div>
			<input type="submit" value="Submit" class="btn btn-primary">

			{{with .Response}}
				<p>Deleted changesets for user {{.Email}} and operation {{.Operation}}</p>
			{{end}}
		</form>
	</div>`
}

func (w ClearChangesetsWidget) Description() *string {
	desc := "Clear the selected changesets for a user"
	return &desc
}

type clearResult struct {
	Operation changeset.UpdateOperation
	Email     string
}

func (w ClearChangesetsWidget) HandleAction(ctx context.Context, data url.Values) (any, error) {
	if len(data) == 0 {
		return nil, nil
	}

	email := data.Get("user_email")
	operationStr := data.Get("operation")
	var operation changeset.UpdateOperation

	switch strings.ToUpper(operationStr) {
	case string(changeset.Delete):
		operation = changeset.Delete
	case string(changeset.MarkRead):
		operation = changeset.MarkRead
	default:
		return nil, fmt.Errorf("unknown update operation %s", operationStr)
	}

	adminCtx := mailboxsweep.AuthorizeAdminContext(ctx, email)
	err := w.service.ClearSelection(adminCtx, operation)
	if err != nil {
		return nil, err
	}

	return clearResult{
		Email:     email,
		Operation: operation,
	}, nil
}
