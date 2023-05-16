// Admin dashboard

package admin

import (
	"fmt"
	"mailboxsweep/changeset"
	"mailboxsweep/pkg/sssadmin"
)

func Dashboard(
	environmentName string,
	changesetService changeset.Service,
) *sssadmin.Dashboard {
	widgets := []sssadmin.Widget{
		ClearChangesetsWidget{changesetService},
	}

	dashTitle := fmt.Sprintf("MailboxSweep (%s) Admin Dashboard", environmentName)
	description := "Admin actions for MailboxSweep"
	return sssadmin.New(dashTitle, description, widgets...)
}
