package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// noAuthAPI is the list of APIs that don't require authentication
var noAuthAPI = map[string][]string{
	"/health":               {"GET"},
	"/api/v1/auth/register": {"POST"},
	"/api/v1/auth/login":    {"POST"},
	"/api/v1/auth/refresh":  {"POST"},
}

// isNoAuthAPI checks if the request is in the no-auth API list
func isNoAuthAPI(path string, method string) bool {
	for api, methods := range noAuthAPI {
		// If ends with *, match by prefix; otherwise match full path
		if strings.HasSuffix(api, "*") {
			if strings.HasPrefix(path, strings.TrimSuffix(api, "*")) && slices.Contains(methods, method) {
				return true
			}
		} else if path == api && slices.Contains(methods, method) {
			return true
		}
	}
	return false
}

// canAccessTenant checks if a user can access a target tenant
func canAccessTenant(user *types.User, targetTenantID uint64, cfg *config.Config) bool {
	// 1. Check if feature is enabled
	if cfg == nil || cfg.Tenant == nil || !cfg.Tenant.EnableCrossTenantAccess {
		return false
	}
	// 2. Check user permissions
	if !user.CanAccessAllTenants {
		return false
	}
	// 3. If target tenant is user's own tenant, allow access
	if user.TenantID == targetTenantID {
		return true
	}
	// 4. User has cross-tenant access permission
	return true
}

// Auth authentication middleware
func Auth(
	tenantService interfaces.TenantService,
	userService interfaces.UserService,
	cfg *config.Config,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ignore OPTIONS request
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Check if request is in the no-auth API list
		if isNoAuthAPI(c.Request.URL.Path, c.Request.Method) {
			c.Next()
			return
		}

		// Try JWT Token authentication
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			user, err := userService.ValidateToken(c.Request.Context(), token)
			if err == nil && user != nil {
				// JWT Token authentication successful
				// Check for cross-tenant access request
				targetTenantID := user.TenantID
				tenantHeader := c.GetHeader("X-Tenant-ID")
				if tenantHeader != "" {
					// Parse target tenant ID
					parsedTenantID, err := strconv.ParseUint(tenantHeader, 10, 64)
					if err == nil {
						// Check if user has cross-tenant access permission
						if canAccessTenant(user, parsedTenantID, cfg) {
							// Verify target tenant exists
							targetTenant, err := tenantService.GetTenantByID(c.Request.Context(), parsedTenantID)
							if err == nil && targetTenant != nil {
								targetTenantID = parsedTenantID
								log.Printf("User %s switching to tenant %d", user.ID, targetTenantID)
							} else {
								log.Printf("Error getting target tenant by ID: %v, tenantID: %d", err, parsedTenantID)
								c.JSON(http.StatusBadRequest, gin.H{
									"error": "Invalid target tenant ID",
								})
								c.Abort()
								return
							}
						} else {
							// User doesn't have permission to access target tenant
							log.Printf("User %s attempted to access tenant %d without permission", user.ID, parsedTenantID)
							c.JSON(http.StatusForbidden, gin.H{
								"error": "Forbidden: insufficient permissions to access target tenant",
							})
							c.Abort()
							return
						}
					}
				}

				// Get tenant info (using target tenant ID)
				tenant, err := tenantService.GetTenantByID(c.Request.Context(), targetTenantID)
				if err != nil {
					log.Printf("Error getting tenant by ID: %v, tenantID: %d, userID: %s", err, targetTenantID, user.ID)
					c.JSON(http.StatusUnauthorized, gin.H{
						"error": "Unauthorized: invalid tenant",
					})
					c.Abort()
					return
				}

				// Store user and tenant info in context
				c.Set(types.TenantIDContextKey.String(), targetTenantID)
				c.Set(types.TenantInfoContextKey.String(), tenant)
				c.Set("user", user)
				c.Request = c.Request.WithContext(
					context.WithValue(
						context.WithValue(
							context.WithValue(c.Request.Context(), types.TenantIDContextKey, targetTenantID),
							types.TenantInfoContextKey, tenant,
						),
						"user", user,
					),
				)
				c.Next()
				return
			}
		}

		// Try X-API-Key authentication (compatibility mode)
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			// Get tenant information
			tenantID, err := tenantService.ExtractTenantIDFromAPIKey(apiKey)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Unauthorized: invalid API key format",
				})
				c.Abort()
				return
			}

			// Verify API key validity (matches the one in database)
			t, err := tenantService.GetTenantByID(c.Request.Context(), tenantID)
			if err != nil {
				log.Printf("Error getting tenant by ID: %v, tenantID: %d", err, tenantID)
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Unauthorized: invalid API key",
				})
				c.Abort()
				return
			}

			if t == nil || t.APIKey != apiKey {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Unauthorized: invalid API key",
				})
				c.Abort()
				return
			}

			// Store tenant ID in context
			c.Set(types.TenantIDContextKey.String(), tenantID)
			c.Set(types.TenantInfoContextKey.String(), t)
			c.Request = c.Request.WithContext(
				context.WithValue(
					context.WithValue(c.Request.Context(), types.TenantIDContextKey, tenantID),
					types.TenantInfoContextKey, t,
				),
			)
			c.Next()
			return
		}

		// No authentication info provided
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing authentication"})
		c.Abort()
	}
}

// GetTenantIDFromContext helper function to get tenant ID from context
func GetTenantIDFromContext(ctx context.Context) (uint64, error) {
	tenantID, ok := ctx.Value("tenantID").(uint64)
	if !ok {
		return 0, errors.New("tenant ID not found in context")
	}
	return tenantID, nil
}
