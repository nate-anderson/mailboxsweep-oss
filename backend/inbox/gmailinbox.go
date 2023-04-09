package inbox

import (
	"fmt"
	"mailboxsweep"

	"google.golang.org/api/gmail/v1"
)

type GmailInbox struct {
	address string
	client  *gmail.Service
}

func (i GmailInbox) Name() string {
	return i.address
}

func (i GmailInbox) Folders() ([]string, error) {
	result, err := i.client.Users.Labels.List(i.address).Do()
	if err != nil {
		return nil, err
	}
	labels := make([]string, len(result.Labels))
	for i, l := range result.Labels {
		labels[i] = l.Name
	}
	return labels, nil
}

func (i GmailInbox) ValidityKeys() (map[string]uint32, error) {
	folders, err := i.Folders()
	if err != nil {
		return nil, fmt.Errorf("failed to get folders in janky ValidityKeys call: %w", err)
	}

	keys := make(map[string]uint32)
	for _, f := range folders {
		keys[f] = 0
	}

	return keys, nil
}

func (i GmailInbox) Folder(name string) (Folder, error) {
	labelsRes, err := i.client.Users.Labels.List(i.address).Do()
	if err != nil {
		return nil, err
	}

	var labelID string
	for _, label := range labelsRes.Labels {
		if label.Name == name {
			labelID = label.Id
		}
	}
	if labelID == "" {
		return nil, mailboxsweep.NotFoundError(fmt.Errorf("label %s not found", name))
	}

	label, err := i.client.Users.Labels.Get(i.address, labelID).Do()
	if err != nil {
		return nil, err
	}

	return GmailFolder{
		label:   label,
		client:  i.client,
		address: i.address,
	}, nil
}
