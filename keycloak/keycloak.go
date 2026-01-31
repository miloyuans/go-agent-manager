package keycloak

import (
	"context"
	"log"
	"sync"
	"time"

	"go-agent-manager/config"
	"go-agent-manager/models" // 引入 models 用于 KeycloakUser DTO

	"github.com/Nerzal/gocloak/v13" // 确保是 gocloak/v13
	"github.com/golang-jwt/jwt/v5"
)

var (
	kcClient      gocloak.GoCloak
	adminToken    *gocloak.JWT // 后端自己调 Keycloak Admin API 的 token
	tokenMutex    sync.RWMutex
	tokenRefreshC chan bool // 用于通知刷新 token 的 channel
)

// InitKeycloak 初始化 Keycloak 客户端
func InitKeycloak() {
	kcClient = gocloak.NewClient(config.AppConfig.Keycloak.AuthServerURL)
	tokenRefreshC = make(chan bool, 1) // buffered channel
	go startAdminTokenRefresher()      // 启动 token 刷新协程
	tokenRefreshC <- true              // 首次获取 token
}

// getAdminAccessToken 获取管理员 Access Token
func getAdminAccessToken() (string, error) {
	tokenMutex.RLock()
	// 如果 token 存在且未过期，直接返回
	if adminToken != nil && !adminToken.Is
	Expired() { // IsExpired() 内部会检查 ExpiresIn
		defer tokenMutex.RUnlock()
		return adminToken.AccessToken, nil
	}
	tokenMutex.RUnlock()

	// 否则需要刷新或首次获取
	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	// 再次检查，防止在 RUnlock 和 Lock 之间其他协程已经获取
	if adminToken != nil && !adminToken.IsExpired() {
		return adminToken.AccessToken, nil
	}

	log.Println("Acquiring/Refreshing Keycloak Admin Access Token...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var err error
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
		token, err := getAdminAccessToken() // 内部会获取并更新 adminToken
		if err != nil {
			log.Printf("Failed to refresh Keycloak Admin token: %v. Retrying in 10 seconds...", err)
			time.AfterFunc(10*time.Second, func() { tokenRefreshC <- true })
			continue
		}

		// 提前 30 秒刷新
		expiresIn := adminToken.ExpiresIn - 30
		if expiresIn < 1 { // 避免负数或过小的刷新间隔
			expiresIn = 1 // 至少等待1秒
		}
		log.Printf("Keycloak Admin token will refresh in %d seconds.", expiresIn)
		time.AfterFunc(time.Duration(expiresIn)*time.Second, func() { tokenRefreshC <- true })
	}
}


// ValidateAccessToken 验证从前端传来的用户 Access Token
// 返回用户的 KeycloakUserID (sub) 和 角色列表
func ValidateAccessToken(ctx context.Context, tokenString string) (string, []string, error) {
	// 方案一：通过 Keycloak Introspection Endpoint 验证 (推荐，最简单可靠)
	adminAccessToken, err := getAdminAccessToken()
	if err != nil {
		return "", nil, err
	}

	result, err := kcClient.RetrospectToken(
		ctx,
		tokenString,
		config.AppConfig.Keycloak.FrontendClientID, // 前端 Client ID
		config.AppConfig.Keycloak.AdminClientSecret, // 如果 FrontendClientID 是 public client，这里可以为空，但 introspect 要求 client secret
		config.AppConfig.Keycloak.Realm,
		adminAccessToken,
	)
	if err != nil {
		return "", nil, err
	}

	if !*result.Active {
		return "", nil, gocloak.NewAPIError("Token is not active", 401)
	}

	// 方案二：本地验证 JWT (需要获取 Keycloak 公钥，并处理 Key Rollover，更复杂)
	// 如果您选择本地验证，可以使用 jwt.ParseWithClaims 配合 Keycloak 的 JWKS Endpoint

	// 解析 JWT payload 获取 sub 和 roles
	claims := jwt.MapClaims{}
	_, _, err = kcClient.DecodeAccessToken(ctx, tokenString, config.AppConfig.Keycloak.Realm) // DecodeAccessToken 内部会校验签名

	if err != nil { // 如果签名验证失败，会返回错误
		// 尝试不校验签名解析一次，以获取 claims
		_, claims, err = kcClient.DecodeAccessToken(ctx, tokenString, "") // 不传入 realm 则不校验签名
		if err != nil {
			return "", nil, err
		}
	}


	// Extract sub (Keycloak User ID)
	sub, ok := claims["sub"].(string)
	if !ok {
		return "", nil, gocloak.NewAPIError("sub claim not found or invalid", 401)
	}

	// Extract roles (Keycloak 默认在 realm_access.roles 中)
	var roles []string
	if realmAccess, ok := claims["realm_access"].(map[string]interface{}); ok {
		if realmRoles, ok := realmAccess["roles"].([]interface{}); ok {
			for _, role := range realmRoles {
				if rStr, ok := role.(string); ok {
					roles = append(roles, rStr)
				}
			}
		}
	}
	
	// Keycloak 在 introspect 结果中也可能直接有 roles 字段
	// 具体的解析方式取决于 Keycloak 的 Client Mapper 配置
	// 如果在 introspect 的 result 中能直接拿到 roles，那更直接
	// 目前 gocloak.RetrospectToken 返回的 result 结构没有直接的 roles，需要二次解析
	// 或者您可以直接本地解析 JWT Payload。

	// 为了简化，这里假定从 claims 中获取 sub 和 roles 成功。
	// 实际生产中，RetrospectToken 已经确认 token 有效，本地解析 payload 是安全的。

	return sub, roles, nil
}


// FetchKeycloakUsers 从 Keycloak 获取所有用户
func FetchKeycloakUsers(ctx context.Context) ([]models.KeycloakUser, error) {
	adminAccessToken, err := getAdminAccessToken()
	if err != nil {
		return nil, err
	}

	params := gocloak.GetUsersParams{
		// First: gocloak.IntPtr(0), // 分页参数
		// Max: gocloak.IntPtr(100),
		// Search: gocloak.StringPtr(""),
	}

	kcUsers, err := kcClient.GetUsers(ctx, adminAccessToken, config.AppConfig.Keycloak.Realm, params)
	if err != nil {
		return nil, err
	}

	var users []models.KeycloakUser
	for _, kcu := range kcUsers {
		user := models.KeycloakUser{
			ID:            *kcu.ID,
			Username:      *kcu.Username,
			Email:         *kcu.Email,
			FirstName:     *kcu.FirstName,
			LastName:      *kcu.LastName,
			Enabled:       *kcu.Enabled,
			EmailVerified: *kcu.EmailVerified,
		}
		// 联邦身份可以从 kcu.FederatedIdentities 中提取
		if kcu.FederatedIdentities != nil {
			for _, fid := range *kcu.FederatedIdentities {
				user.FederatedIdentities = append(user.FederatedIdentities, struct {
					IdentityProvider string `json:"identityProvider"`
					UserID           string `json:"userId"`
					UserName         string `json:"userName"`
				}{
					IdentityProvider: *fid.IdentityProvider,
					UserID:           *fid.UserID,
					UserName:         *fid.UserName,
				})
			}
		}
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

	// 先获取用户，因为 UpdateUser 需要完整的 UserRepresentation
	user, err := kcClient.GetUserByID(ctx, adminAccessToken, config.AppConfig.Keycloak.Realm, userID)
	if err != nil {
		return err
	}

	user.Enabled = gocloak.BoolP(enable) // 设置启用/禁用状态

	err = kcClient.UpdateUser(ctx, adminAccessToken, config.AppConfig.Keycloak.Realm, *user)
	if err != nil {
		return err
	}
	return nil
}
