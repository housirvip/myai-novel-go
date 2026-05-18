package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/db/models"
	"myai-novel-go/internal/domain/shared"
)

const passwordHashRounds = 12

// SessionUser 是回给客户端的精简 user 视图,不包含 password_hash。
type SessionUser struct {
	ID          int64  `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Status      string `json:"status"`
}

// Result 把 register/login 结果统一成"原始 token + user"。
type Result struct {
	SessionToken string
	User         SessionUser
}

type Service struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

// Register 创建新用户 + session。如果是首个用户且存在 owner_user_id 为空的 books,
// 自动把这些 books 划归该用户(对齐原项目"先建库再注册"场景的便利逻辑)。
func (s *Service) Register(ctx context.Context, email, password, displayName string) (*Result, error) {
	email = normalizeEmail(email)
	if email == "" {
		return nil, shared.BadRequest("Email is required")
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return nil, shared.BadRequest("Display name is required")
	}
	if len(password) < 8 {
		return nil, shared.BadRequest("Password must be at least 8 characters")
	}

	var existing models.User
	err := s.db.WithContext(ctx).Where("email = ?", email).First(&existing).Error
	if err == nil {
		return nil, shared.Conflict("User already exists", nil)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), passwordHashRounds)
	if err != nil {
		return nil, err
	}
	now := shared.NowISO()
	user := models.User{
		Email:        email,
		PasswordHash: string(hash),
		DisplayName:  displayName,
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	var token string
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		// 首个用户:把 owner_user_id IS NULL 的 books 全部划给他
		var userCount int64
		if err := tx.Model(&models.User{}).Count(&userCount).Error; err != nil {
			return err
		}
		if userCount == 1 {
			if err := tx.Model(&models.Book{}).
				Where("owner_user_id IS NULL").
				Update("owner_user_id", user.ID).Error; err != nil {
				return err
			}
		}
		t, err := s.createSession(tx, user.ID)
		if err != nil {
			return err
		}
		token = t
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &Result{SessionToken: token, User: toSessionUser(&user)}, nil
}

// Login 校验密码并签发新 session。同样的密码错误 / 用户不存在 / 用户停用都返回 401,避免泄露。
func (s *Service) Login(ctx context.Context, email, password string) (*Result, error) {
	email = normalizeEmail(email)
	if email == "" || password == "" {
		return nil, shared.Unauthorized("Invalid email or password")
	}
	var user models.User
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.Unauthorized("Invalid email or password")
		}
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil ||
		user.Status != "active" {
		return nil, shared.Unauthorized("Invalid email or password")
	}
	var token string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		t, err := s.createSession(tx, user.ID)
		token = t
		return err
	})
	if err != nil {
		return nil, err
	}
	return &Result{SessionToken: token, User: toSessionUser(&user)}, nil
}

// GetSessionUser 根据原始 token 回查 user;
// 同时执行 expire / status 校验,过期或异常会顺手清掉记录。
func (s *Service) GetSessionUser(ctx context.Context, rawToken string) (*SessionUser, error) {
	if rawToken == "" {
		return nil, nil
	}
	hash := hashSessionToken(rawToken)
	var session models.UserSession
	if err := s.db.WithContext(ctx).Where("session_token_hash = ?", hash).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if session.ExpiresAt <= shared.NowISO() {
		s.db.WithContext(ctx).Delete(&session)
		return nil, nil
	}
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, session.UserID).Error; err != nil {
		s.db.WithContext(ctx).Delete(&session)
		return nil, nil
	}
	if user.Status != "active" {
		s.db.WithContext(ctx).Delete(&session)
		return nil, nil
	}
	now := shared.NowISO()
	s.db.WithContext(ctx).Model(&session).Update("last_seen_at", now)
	u := toSessionUser(&user)
	return &u, nil
}

// Logout 删除该 token 对应的 session 记录(忽略不存在的情况)。
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	return s.db.WithContext(ctx).
		Where("session_token_hash = ?", hashSessionToken(rawToken)).
		Delete(&models.UserSession{}).Error
}

// SignSessionToken 把 token 与 HMAC 签名拼成 cookie 值,
// 验证时用 verifySignedSessionToken 拆开。
func (s *Service) SignSessionToken(token string) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.AuthSessionSecret))
	mac.Write([]byte(token))
	sig := hex.EncodeToString(mac.Sum(nil))
	return token + "." + sig
}

func (s *Service) VerifySignedSessionToken(value string) string {
	if value == "" {
		return ""
	}
	idx := strings.LastIndex(value, ".")
	if idx <= 0 {
		return ""
	}
	token := value[:idx]
	sig := value[idx+1:]
	mac := hmac.New(sha256.New, []byte(s.cfg.AuthSessionSecret))
	mac.Write([]byte(token))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return ""
	}
	return token
}

func (s *Service) createSession(tx *gorm.DB, userID int64) (string, error) {
	token, err := generateSessionToken()
	if err != nil {
		return "", err
	}
	now := shared.NowISO()
	expires := time.Now().UTC().Add(time.Duration(s.cfg.AuthSessionTTLHours) * time.Hour).Format(time.RFC3339Nano)
	row := models.UserSession{
		UserID:           userID,
		SessionTokenHash: hashSessionToken(token),
		ExpiresAt:        expires,
		CreatedAt:        now,
		LastSeenAt:       &now,
	}
	if err := tx.Create(&row).Error; err != nil {
		return "", err
	}
	return token, nil
}

func generateSessionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func toSessionUser(u *models.User) SessionUser {
	return SessionUser{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Status:      u.Status,
	}
}
