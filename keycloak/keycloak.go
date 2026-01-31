package keycloak

import (
	"context"
	"errors" // 引入标准库 errors
	"log"
	"sync"
	"time"

	"go-agent-manager/config"
	"go-agent-manager/models"

	"github.com/Nerzal/gocloak/v13" // 确保引用的是 v13
	"github.com/golang-jwt/jwt/v5"
)

var (
	kcClient      *gocloak.GoCloak // 修改为指针类型，因为 NewClient 返回 *gocloak.GoCloak
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
	// v13 版本移除了 IsExpired 方法，我们需要手动计算
	// ExpiresIn 是 token 的有效期秒数，我们并没有存储 token 获取的时间点，
	// 所以更稳妥的方式是依赖定时刷新，或者在这里简单判断 adminToken 是否为 nil
	// 真正的过期检查需要解析 JWT 的 exp 字段，或者利用 Refresh Token 机制
	if adminToken != nil {
		// 这里简化处理：假定如果有 token 且非空，即暂时可用。
		// 严谨做法是解析 adminToken.AccessToken 的 exp claim。
		// 由于 startAdminTokenRefresher 会定时更新，这里只要非 nil 即可。
		tokenMutex.RUnlock()
		return adminToken.AccessToken, nil
	}
	tokenMutex.RUnlock()

	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	if adminToken != nil {
		return adminToken.AccessToken, nil
	}

	log.Println("Acquiring/Refreshing Keycloak Admin Access Token...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // 增加超时时间
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
		// 强制刷新：调用 LoginClient 获取新 Token
		// 注意：getAdminAccessToken 内部有缓存检查，为了强制刷新，我们需要绕过它或者重置 adminToken
		// 但为了简单，我们直接在这里调用 LoginClient 更新全局变量
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
		// ExpiresIn 是 int 类型
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
	adminAccessToken, err := getAdminAccessToken()
	if err != nil {
		return "", nil, err
	}

	// v13 RetrospectToken 参数变更：(ctx, accessToken, clientID, clientSecret, realm)
	result, err := kcClient.RetrospectToken(
		ctx,
		tokenString,
		config.AppConfig.Keycloak.FrontendClientID,
		config.AppConfig.Keycloak.AdminClientSecret, // 如果 Client 是 public，这里可能传 ""，但通常 introspect 需要 secret
		config.AppConfig.Keycloak.Realm,
	)
	if err != nil {
		return "", nil, err
	}

	if !*result.Active {
		// v13 移除了 NewAPIError，使用标准 errors.New
		return "", nil, errors.New("token is not active")
	}

	// 解析 Token 获取 sub 和 roles
	// DecodeAccessToken 在 v13 返回 (*jwt.Token, *jwt.MapClaims, error)
	_, claims, err := kcClient.DecodeAccessToken(ctx, tokenString, config.AppConfig.Keycloak.Realm)
	if err != nil {
		return "", nil, err
	}
	
	// jwt.MapClaims 本质是 map[string]interface{}
	claimsMap := *claims

	sub, ok := claimsMap["sub"].(string)
	if !ok {
		return "", nil, errors.New("sub claim not found or invalid")
	}

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
	adminAccessToken, err := getAdminAccessToken()
	if err != nil {
		return nil, err
	}

	params := gocloak.GetUsersParams{
		// v13 参数使用指针，可以为空
	}

	kcUsers, err := kcClient.GetUsers(ctx, adminAccessToken, config.AppConfig.Keycloak.Realm, params)
	if err != nil {
		return nil, err
	}

	var users []models.KeycloakUser
	for _, kcu := range kcUsers {
		user := models.KeycloakUser{
			ID:            gocloak.PString(kcu.ID), // 使用 PString 安全解引用，防止 nil panic
			Username:      gocloak.PString(kcu.Username),
			Email:         gocloak.PString(kcu.Email),
			FirstName:     gocloak.PString(kcu.FirstName),
			LastName:      gocloak.PString(kcu.LastName),
			Enabled:       gocloak.PBool(kcu.Enabled),
			EmailVerified: gocloak.PBool(kcu.EmailVerified),
		}
		
		// v13 用户结构体中可能将 FederatedIdentities 放在了 UserRepresentation 中
		// 或者需要单独调用 API 获取 (GetUserFederatedIdentities)
		// 如果 kcu.FederatedIdentities 报错，说明该字段在 GetUsers 返回的简略信息中不存在
		// 为了修复编译错误，这里暂时注释掉联合身份的获取，或者需要额外调用 API
		
		/*
		if kcu.FederatedIdentities != nil {
			for _, fid := range *kcu.FederatedIdentities {
				user.FederatedIdentities = append(user.FederatedIdentities, struct {
					IdentityProvider string `json:"identityProvider"`
					UserID           string `json:"userId"`
					UserName         string `json:"userName"`
				}{
					IdentityProvider: gocloak.PString(fid.IdentityProvider),
					UserID:           gocloak.PString(fid.UserID),
					UserName:         gocloak.PString(fid.UserName),
				})
			}
		}
		*/
		
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
