package inboxuser

import (
	"context"
	"errors"
	"fmt"
	"mailboxsweep"
	"strings"
	"time"

	"github.com/mcnijman/go-emailaddress"
)

type Service interface {
	Store(ctx context.Context, address string) (*InboxUser, error)
	FindByAddress(ctx context.Context, address string) (*InboxUser, error)
	FindByUUID(ctx context.Context, uuid string) (*InboxUser, error)
	SetName(ctx context.Context, nameFirst, nameLast string) error
	Consent(ctx context.Context, agreements []Agreement) error
	ListUserAgreements(ctx context.Context) ([]UserAgreement, error)
	GetAgreement(ctx context.Context, code Agreement) (*UserAgreement, error)

	StorePayment(ctx context.Context, payment Payment) error
}

func NewService(repo InboxUserRepository) (DefaultService, error) {
	agreements, err := readAgreements()
	if err != nil {
		return DefaultService{}, err
	}
	return DefaultService{
		repo:       repo,
		agreements: agreements,
	}, nil
}

type DefaultService struct {
	repo       InboxUserRepository
	agreements map[Agreement]AgreementDocument
}

func (s DefaultService) Store(ctx context.Context, address string) (*InboxUser, error) {
	_, err := emailaddress.Parse(address)
	if err != nil {
		return nil, err
	}

	user, err := NewInboxUser(strings.TrimSpace(address))
	if err != nil {
		return nil, err
	}

	if _, err = s.repo.Store(ctx, *user); err != nil {
		return nil, err
	}

	return s.repo.FindByAddress(ctx, address)
}

func (s DefaultService) Consent(ctx context.Context, agreements []Agreement) error {
	userAddress, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	user, err := s.repo.FindByAddress(ctx, userAddress)
	if err != nil {
		return err
	}

	for _, agreement := range agreements {
		err = s.repo.Consent(ctx, user.Address, agreement)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s DefaultService) FindByAddress(ctx context.Context, address string) (*InboxUser, error) {
	return s.repo.FindByAddress(ctx, address)
}

func (s DefaultService) FindByUUID(ctx context.Context, uuid string) (*InboxUser, error) {
	return s.repo.FindByUUID(ctx, uuid)
}

func (s DefaultService) SetName(ctx context.Context, nameFirst, nameLast string) error {
	if strings.TrimSpace(nameFirst) == "" {
		return errors.New("first name is required")
	}

	if strings.TrimSpace(nameLast) == "" {
		return errors.New("last name is required")
	}

	userAddress, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	user, err := s.repo.FindByAddress(ctx, userAddress)
	if err != nil {
		return err
	}

	return s.repo.SetName(ctx, user.Address, nameFirst, nameLast)
}

type UserAgreement struct {
	Agreement   Agreement
	Description string
	Agreed      bool
	AgreedAt    time.Time
	Required    bool
}

func (s DefaultService) ListUserAgreements(ctx context.Context) ([]UserAgreement, error) {
	userAddress, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindByAddress(ctx, userAddress)
	if err != nil {
		return nil, err
	}

	userAgreements := make([]UserAgreement, 0, len(user.Agreements))

	for _, agr := range allAgreements {
		doc, ok := s.agreements[agr]
		if !ok {
			return nil, fmt.Errorf("no document loaded for agreement %s", agr)
		}

		agreed, ok := user.Agreements[agr]
		userAgreements = append(userAgreements, UserAgreement{
			Agreement:   agr,
			Agreed:      ok,
			AgreedAt:    agreed,
			Description: doc.Name,
			Required:    contains(requiredAgreements, agr),
		})
	}
	return userAgreements, nil
}

func (s DefaultService) StorePayment(ctx context.Context, payment Payment) error {
	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return err
	}

	return s.repo.StorePayment(ctx, addr, payment)
}

func (s DefaultService) GetAgreement(ctx context.Context, code Agreement) (*UserAgreement, error) {
	userAddress, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindByAddress(ctx, userAddress)
	if err != nil {
		return nil, err
	}

	agr, ok := s.agreements[code]
	if !ok {
		err = fmt.Errorf("agreement %s not recognized", code)
		return nil, mailboxsweep.NotFoundError(err)
	}

	res := &UserAgreement{
		Agreement:   agr.Code,
		Description: agr.Body,
		Required:    contains(requiredAgreements, agr.Code),
	}

	userAgr, ok := user.Agreements[code]
	if ok {
		res.Agreed = true
		res.AgreedAt = userAgr
	}

	return res, nil
}
