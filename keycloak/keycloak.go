package keycloak

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"go-agent-manager/config"
	"go-agent-manager/models"

	"github.com/Nerzal/gocloak/v13"
)

var (
	kcClient      *gocloak.GoCloak
	adminToken    *gocloak.JWT
	tokenMutex    sync.RWMutex
	tokenRefreshC chan bool
)

// InitKeycloak 初始化 Keycloak 客户端
func InitKeycloak() {
	kcClient = gocloak.NewClient(config.AppConfig.Keycloak.AuthServerURL)
	tokenRefreshC = make(chan bool, 1)
	go startAdminTokenRefresher()
	tokenRefreshC <- true
}

// getAdminAccessToken 获取管理员 Access Token
func getAdminAccessToken() (string, error) {
	tokenMutex.RLock()
	// 简单判断：如果 token 存在，直接返回
	// 依赖后台协程定时刷新来保证有效性
	if adminToken != nil {
		tokenMutex.RUnlock()
		return adminToken.AccessToken, nil
	}
	tokenMutex.RUnlock()

	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	// 双重检查
	if adminToken != nil {
		return adminToken.AccessToken, nil
	}

	log.Println("Acquiring/Refreshing Keycloak Admin Access Token...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	// LoginClient 使用 Client Credentials Grant
	adminToken, err = kcClient.LoginClient(
		ctx,
		config.AppConfig.Keycloak.AdminClientID,
		config.AppConfig.Keycloak.AdminClientSecret,
		config.AppConfig.Keycloak.Realm,
	)
	if err != nil {
		return "", err
	}
	log.Println("Keycloak Admin Access Token acquired successfully.")
	return adminToken.AccessToken, nil
}

// startAdminTokenRefresher 启动一个协程定时刷新管理员 token
func startAdminTokenRefresher() {
	for range tokenRefreshC {
		tokenMutex.Lock()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		token, err := kcClient.LoginClient(
			ctx,
			config.AppConfig.Keycloak.AdminClientID,
			config.AppConfig.Keycloak.AdminClientSecret,
			config.AppConfig.Keycloak.Realm,
		)
		cancel()

		if err != nil {
			tokenMutex.Unlock()
			log.Printf("Failed to refresh Keycloak Admin token: %v. Retrying in 10 seconds...", err)
			time.AfterFunc(10*time.Second, func() { tokenRefreshC <- true })
			continue
		}

		adminToken = token
		tokenMutex.Unlock()

		// 计算下次刷新时间：提前 30 秒刷新
		expiresIn := token.ExpiresIn - 30
		if expiresIn < 1 {
			expiresIn = 1
		}
		log.Printf("Keycloak Admin token will refresh in %d seconds.", expiresIn)
		time.AfterFunc(time.Duration(expiresIn)*time.Second, func() { tokenRefreshC <- true })
	}
}

// ValidateAccessToken 验证从前端传来的用户 Access Token
func ValidateAccessToken(ctx context.Context, tokenString string) (string, []string, error) {
	// 调用 getAdminAccessToken 主要是为了确保 Keycloak 服务本身是通的，或者 introspect 需要 token
	// 但 v13 的 RetrospectToken 只需要 clientID/Secret，不需要 admin token。
	// 不过为了保险起见，或者如果将来使用其他 API，保留这个调用也无妨，
	// 但既然编译器报错未使用的变量，我们需要用到它或者移除它。
	// 在这里，RetrospectToken 确实不需要 adminToken。
	
	// 如果您使用的是 Confidential Client (有 secret)，Retrospect 不需要 Admin Token。
	
	// 1. 验证 Token 有效性 (Introspection)
	result, err := kcClient.RetrospectToken(
		ctx,
		tokenString,
		config.AppConfig.Keycloak.FrontendClientID,
		config.AppConfig.Keycloak.AdminClientSecret,
		config.AppConfig.Keycloak.Realm,
	)
	if err != nil {
		return "", nil, err
	}

	if !*result.Active {
		return "", nil, errors.New("token is not active")
	}

	// 2. 解析 Token 获取用户信息 (Decode)
	// DecodeAccessToken 不需要额外的权限，只需要 JWT 字符串
	_, claims, err := kcClient.DecodeAccessToken(ctx, tokenString, config.AppConfig.Keycloak.Realm)
	if err != nil {
		return "", nil, err
	}

	// claims 类型是 *jwt.MapClaims，解引用后就是 map[string]interface{}
	claimsMap := *claims

	// 获取 User ID (sub)
	sub, ok := claimsMap["sub"].(string)
	if !ok {
		return "", nil, errors.New("sub claim not found or invalid")
	}

	// 获取 Roles
	var roles []string
	if realmAccess, ok := claimsMap["realm_access"].(map[string]interface{}); ok {
		if realmRoles, ok := realmAccess["roles"].([]interface{}); ok {
			for _, role := range realmRoles {
				if rStr, ok := role.(string); ok {
					roles = append(roles, rStr)
				}
			}
		}
	}

	return sub, roles, nil
}

// FetchKeycloakUsers 从 Keycloak 获取所有用户
func FetchKeycloakUsers(ctx context.Context) ([]models.KeycloakUser, error) {
	// 这里必须使用 Admin Token
	adminAccessToken, err := getAdminAccessToken()
	if err != nil {
		return nil, err
	}

	params := gocloak.GetUsersParams{
		// 默认获取所有
	}

	kcUsers, err := kcClient.GetUsers(ctx, adminAccessToken, config.AppConfig.Keycloak.Realm, params)
	if err != nil {
		return nil, err
	}

	var users []models.KeycloakUser
	for _, kcu := range kcUsers {
		user := models.KeycloakUser{
			ID:            gocloak.PString(kcu.ID),
			Username:      gocloak.PString(kcu.Username),
			Email:         gocloak.PString(kcu.Email),
			FirstName:     gocloak.PString(kcu.FirstName),
			LastName:      gocloak.PString(kcu.LastName),
			Enabled:       gocloak.PBool(kcu.Enabled),
			EmailVerified: gocloak.PBool(kcu.EmailVerified),
		}
		// 暂时忽略 FederatedIdentities 以简化
		users = append(users, user)
	}

	return users, nil
}

// UpdateKeycloakUserStatus 启用/禁用 Keycloak 用户
func UpdateKeycloakUserStatus(ctx context.Context, userID string, enable bool) error {
	adminAccessToken, err := getAdminAccessToken()
	if err != nil {
		return err
	}

	user, err := kcClient.GetUserByID(ctx, adminAccessToken, config.AppConfig.Keycloak.Realm, userID)
	if err != nil {
		return err
	}

	user.Enabled = gocloak.BoolP(enable)

	err = kcClient.UpdateUser(ctx, adminAccessToken, config.AppConfig.Keycloak.Realm, *user)
	if err != nil {
		return err
	}
	return nil
}
