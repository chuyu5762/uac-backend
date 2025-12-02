package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/pu-ac-cn/uac-backend/internal/model"
	"github.com/pu-ac-cn/uac-backend/internal/repository"
)

var (
	ErrUserIDEmpty       = errors.New("用户 ID 不能为空")
	ErrUsernameEmpty     = errors.New("用户名不能为空")
	ErrUsernameInvalid   = errors.New("用户名只能包含字母、数字和下划线")
	ErrUsernameTooShort  = errors.New("用户名长度不能少于 3 个字符")
	ErrEmailEmpty        = errors.New("邮箱不能为空")
	ErrEmailInvalid      = errors.New("邮箱格式无效")
	ErrPasswordEmpty     = errors.New("密码不能为空")
	ErrPasswordTooShort  = errors.New("密码长度不能少于 8 个字符")
	ErrUserLocked        = errors.New("用户已被锁定")
	ErrUserDisabled      = errors.New("用户已被禁用")
	ErrPasswordIncorrect = errors.New("密码错误")
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

type UserService interface {
	Create(ctx context.Context, user *model.User, password string) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter *repository.UserFilter, page *repository.Pagination) ([]*model.User, int64, error)
	Authenticate(ctx context.Context, username, password string) (*model.User, error)
	ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error
	BindOrganization(ctx context.Context, userID, orgID string) error
	UnbindOrganization(ctx context.Context, userID, orgID string) error
	ListUserOrganizations(ctx context.Context, userID string) ([]*model.UserOrgBinding, error)
	HasOrgAccess(ctx context.Context, userID, orgID string) (bool, error)
}

type userService struct {
	userRepo    repository.UserRepository
	bindingRepo repository.UserOrgBindingRepository
	orgRepo     repository.OrganizationRepository
}

func NewUserService(userRepo repository.UserRepository, bindingRepo repository.UserOrgBindingRepository, orgRepo repository.OrganizationRepository) UserService {
	return &userService{userRepo: userRepo, bindingRepo: bindingRepo, orgRepo: orgRepo}
}

func (s *userService) Create(ctx context.Context, user *model.User, password string) error {
	if err := s.validateUser(user); err != nil {
		return err
	}
	if err := s.validatePassword(password); err != nil {
		return err
	}
	if err := user.SetPassword(password); err != nil {
		return errors.New("密码加密失败")
	}
	if user.Status == "" {
		user.Status = model.StatusActive
	}
	return s.userRepo.Create(ctx, user)
}

func (s *userService) GetByID(ctx context.Context, id string) (*model.User, error) {
	if id == "" {
		return nil, ErrUserIDEmpty
	}
	return s.userRepo.GetByID(ctx, id)
}

func (s *userService) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	if username == "" {
		return nil, ErrUsernameEmpty
	}
	return s.userRepo.GetByUsername(ctx, username)
}

func (s *userService) Update(ctx context.Context, user *model.User) error {
	if user.ID == "" {
		return ErrUserIDEmpty
	}
	return s.userRepo.Update(ctx, user)
}

func (s *userService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return ErrUserIDEmpty
	}
	return s.userRepo.Delete(ctx, id)
}

func (s *userService) List(ctx context.Context, filter *repository.UserFilter, page *repository.Pagination) ([]*model.User, int64, error) {
	if page == nil {
		page = &repository.Pagination{Page: 1, PageSize: 20}
	}
	return s.userRepo.List(ctx, filter, page)
}

func (s *userService) Authenticate(ctx context.Context, username, password string) (*model.User, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, ErrPasswordIncorrect
	}
	if user.IsLocked() {
		return nil, ErrUserLocked
	}
	if !user.IsActive() {
		return nil, ErrUserDisabled
	}
	if !user.VerifyPassword(password) {
		user.IncrementFailedLogin()
		_ = s.userRepo.Update(ctx, user)
		return nil, ErrPasswordIncorrect
	}
	user.ResetFailedLogin()
	_ = s.userRepo.Update(ctx, user)
	return user, nil
}

func (s *userService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if !user.VerifyPassword(oldPassword) {
		return ErrPasswordIncorrect
	}
	if err := s.validatePassword(newPassword); err != nil {
		return err
	}
	if err := user.SetPassword(newPassword); err != nil {
		return errors.New("密码加密失败")
	}
	return s.userRepo.Update(ctx, user)
}

func (s *userService) BindOrganization(ctx context.Context, userID, orgID string) error {
	if userID == "" {
		return ErrUserIDEmpty
	}
	if orgID == "" {
		return ErrOrgIDEmpty
	}
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return err
	}
	if s.orgRepo != nil {
		if _, err := s.orgRepo.GetByID(ctx, orgID); err != nil {
			return errors.New("组织不存在")
		}
	}
	binding := &model.UserOrgBinding{UserID: userID, OrgID: orgID}
	return s.bindingRepo.Create(ctx, binding)
}

func (s *userService) UnbindOrganization(ctx context.Context, userID, orgID string) error {
	if userID == "" {
		return ErrUserIDEmpty
	}
	if orgID == "" {
		return ErrOrgIDEmpty
	}
	return s.bindingRepo.Delete(ctx, userID, orgID)
}

func (s *userService) ListUserOrganizations(ctx context.Context, userID string) ([]*model.UserOrgBinding, error) {
	if userID == "" {
		return nil, ErrUserIDEmpty
	}
	return s.bindingRepo.ListByUserID(ctx, userID)
}

func (s *userService) HasOrgAccess(ctx context.Context, userID, orgID string) (bool, error) {
	if userID == "" || orgID == "" {
		return false, nil
	}
	return s.bindingRepo.Exists(ctx, userID, orgID)
}

func (s *userService) validateUser(user *model.User) error {
	if user == nil {
		return errors.New("用户信息不能为空")
	}
	user.Username = strings.TrimSpace(user.Username)
	if user.Username == "" {
		return ErrUsernameEmpty
	}
	if len(user.Username) < 3 {
		return ErrUsernameTooShort
	}
	if !usernameRegex.MatchString(user.Username) {
		return ErrUsernameInvalid
	}
	user.Email = strings.TrimSpace(user.Email)
	if user.Email == "" {
		return ErrEmailEmpty
	}
	if !emailRegex.MatchString(user.Email) {
		return ErrEmailInvalid
	}
	return nil
}

func (s *userService) validatePassword(password string) error {
	if password == "" {
		return ErrPasswordEmpty
	}
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	return nil
}
