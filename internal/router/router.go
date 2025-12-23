package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/dig"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/Tencent/WeKnora/internal/handler/session"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types/interfaces"

	_ "github.com/Tencent/WeKnora/docs" // swagger docs
)

// RouterParams defines the router parameters
type RouterParams struct {
	dig.In

	Config                *config.Config
	UserService           interfaces.UserService
	KBService             interfaces.KnowledgeBaseService
	KnowledgeService      interfaces.KnowledgeService
	ChunkService          interfaces.ChunkService
	SessionService        interfaces.SessionService
	MessageService        interfaces.MessageService
	ModelService          interfaces.ModelService
	EvaluationService     interfaces.EvaluationService
	KBHandler             *handler.KnowledgeBaseHandler
	KnowledgeHandler      *handler.KnowledgeHandler
	TenantHandler         *handler.TenantHandler
	TenantService         interfaces.TenantService
	ChunkHandler          *handler.ChunkHandler
	SessionHandler        *session.Handler
	MessageHandler        *handler.MessageHandler
	ModelHandler          *handler.ModelHandler
	EvaluationHandler     *handler.EvaluationHandler
	AuthHandler           *handler.AuthHandler
	InitializationHandler *handler.InitializationHandler
	SystemHandler         *handler.SystemHandler
	MCPServiceHandler     *handler.MCPServiceHandler
	WebSearchHandler      *handler.WebSearchHandler
	FAQHandler            *handler.FAQHandler
	TagHandler            *handler.TagHandler
}

// NewRouter creates a new router
func NewRouter(params RouterParams) *gin.Engine {
	r := gin.New()

	// CORS middleware should be placed first
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-API-Key", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "Access-Control-Allow-Origin"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Basic middleware (no authentication required)
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.ErrorHandler())

	// Health check (no authentication required)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Swagger API docs (enabled only in non-production environments)
	// Determined by GIN_MODE environment variable: disabled in release mode
	if gin.Mode() != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
			ginSwagger.DefaultModelsExpandDepth(-1),
			ginSwagger.DocExpansion("list"),
			ginSwagger.DeepLinking(true),
			ginSwagger.PersistAuthorization(true),
		))
	}

	// Authentication middleware
	r.Use(middleware.Auth(params.TenantService, params.UserService, params.Config))

	// OpenTelemetry tracing middleware
	r.Use(middleware.TracingMiddleware())

	// API routes requiring authentication
	v1 := r.Group("/api/v1")
	{
		RegisterAuthRoutes(v1, params.AuthHandler)
		RegisterTenantRoutes(v1, params.TenantHandler)
		RegisterKnowledgeBaseRoutes(v1, params.KBHandler)
		RegisterKnowledgeTagRoutes(v1, params.TagHandler)
		RegisterKnowledgeRoutes(v1, params.KnowledgeHandler)
		RegisterFAQRoutes(v1, params.FAQHandler)
		RegisterChunkRoutes(v1, params.ChunkHandler)
		RegisterSessionRoutes(v1, params.SessionHandler)
		RegisterChatRoutes(v1, params.SessionHandler)
		RegisterMessageRoutes(v1, params.MessageHandler)
		RegisterModelRoutes(v1, params.ModelHandler)
		RegisterEvaluationRoutes(v1, params.EvaluationHandler)
		RegisterInitializationRoutes(v1, params.InitializationHandler)
		RegisterSystemRoutes(v1, params.SystemHandler)
		RegisterMCPServiceRoutes(v1, params.MCPServiceHandler)
		RegisterWebSearchRoutes(v1, params.WebSearchHandler)
	}

	return r
}

// RegisterChunkRoutes registers chunk-related routes
func RegisterChunkRoutes(r *gin.RouterGroup, handler *handler.ChunkHandler) {
	chunks := r.Group("/chunks")
	{
		chunks.GET("/:knowledge_id", handler.ListKnowledgeChunks)
		chunks.GET("/by-id/:id", handler.GetChunkByIDOnly)
		chunks.DELETE("/:knowledge_id/:id", handler.DeleteChunk)
		chunks.DELETE("/:knowledge_id", handler.DeleteChunksByKnowledgeID)
		chunks.PUT("/:knowledge_id/:id", handler.UpdateChunk)
		chunks.DELETE("/by-id/:id/questions", handler.DeleteGeneratedQuestion)
	}
}

// RegisterKnowledgeRoutes registers knowledge-related routes
func RegisterKnowledgeRoutes(r *gin.RouterGroup, handler *handler.KnowledgeHandler) {
	kb := r.Group("/knowledge-bases/:id/knowledge")
	{
		kb.POST("/file", handler.CreateKnowledgeFromFile)
		kb.POST("/url", handler.CreateKnowledgeFromURL)
		kb.POST("/manual", handler.CreateManualKnowledge)
		kb.GET("", handler.ListKnowledge)
	}

	k := r.Group("/knowledge")
	{
		k.GET("/batch", handler.GetKnowledgeBatch)
		k.GET("/:id", handler.GetKnowledge)
		k.DELETE("/:id", handler.DeleteKnowledge)
		k.PUT("/:id", handler.UpdateKnowledge)
		k.PUT("/manual/:id", handler.UpdateManualKnowledge)
		k.GET("/:id/download", handler.DownloadKnowledgeFile)
		k.PUT("/image/:id/:chunk_id", handler.UpdateImageInfo)
		k.PUT("/tags", handler.UpdateKnowledgeTagBatch)
		k.GET("/search", handler.SearchKnowledge)
	}
}

// RegisterFAQRoutes registers FAQ-related routes
func RegisterFAQRoutes(r *gin.RouterGroup, handler *handler.FAQHandler) {
	if handler == nil {
		return
	}
	faq := r.Group("/knowledge-bases/:id/faq")
	{
		faq.GET("/entries", handler.ListEntries)
		faq.GET("/entries/export", handler.ExportEntries)
		faq.POST("/entries", handler.UpsertEntries)
		faq.POST("/entry", handler.CreateEntry)
		faq.PUT("/entries/:entry_id", handler.UpdateEntry)
		// Unified batch update API - supports is_enabled, is_recommended, tag_id
		faq.PUT("/entries/fields", handler.UpdateEntryFieldsBatch)
		faq.PUT("/entries/tags", handler.UpdateEntryTagBatch)
		faq.DELETE("/entries", handler.DeleteEntries)
		faq.POST("/search", handler.SearchFAQ)
	}
	// FAQ import progress route (outside of knowledge-base scope)
	faqImport := r.Group("/faq/import")
	{
		faqImport.GET("/progress/:task_id", handler.GetImportProgress)
	}
}

// RegisterKnowledgeBaseRoutes registers knowledge base routes
func RegisterKnowledgeBaseRoutes(r *gin.RouterGroup, handler *handler.KnowledgeBaseHandler) {
	kb := r.Group("/knowledge-bases")
	{
		kb.POST("", handler.CreateKnowledgeBase)
		kb.GET("", handler.ListKnowledgeBases)
		kb.GET("/:id", handler.GetKnowledgeBase)
		kb.PUT("/:id", handler.UpdateKnowledgeBase)
		kb.DELETE("/:id", handler.DeleteKnowledgeBase)
		kb.GET("/:id/hybrid-search", handler.HybridSearch)
		kb.POST("/copy", handler.CopyKnowledgeBase)
		kb.GET("/copy/progress/:task_id", handler.GetKBCloneProgress)
	}
}

// RegisterKnowledgeTagRoutes registers knowledge tag routes
func RegisterKnowledgeTagRoutes(r *gin.RouterGroup, tagHandler *handler.TagHandler) {
	if tagHandler == nil {
		return
	}
	kbTags := r.Group("/knowledge-bases/:id/tags")
	{
		kbTags.GET("", tagHandler.ListTags)
		kbTags.POST("", tagHandler.CreateTag)
		kbTags.PUT("/:tag_id", tagHandler.UpdateTag)
		kbTags.DELETE("/:tag_id", tagHandler.DeleteTag)
	}
}

// RegisterMessageRoutes registers message-related routes
func RegisterMessageRoutes(r *gin.RouterGroup, handler *handler.MessageHandler) {
	messages := r.Group("/messages")
	{
		messages.GET("/:session_id/load", handler.LoadMessages)
		messages.DELETE("/:session_id/:id", handler.DeleteMessage)
	}
}

// RegisterSessionRoutes registers session routes
func RegisterSessionRoutes(r *gin.RouterGroup, handler *session.Handler) {
	sessions := r.Group("/sessions")
	{
		sessions.POST("", handler.CreateSession)
		sessions.GET("/:id", handler.GetSession)
		sessions.GET("", handler.GetSessionsByTenant)
		sessions.PUT("/:id", handler.UpdateSession)
		sessions.DELETE("/:id", handler.DeleteSession)
		sessions.POST("/:session_id/generate_title", handler.GenerateTitle)
		sessions.POST("/:session_id/stop", handler.StopSession)
		sessions.GET("/continue-stream/:session_id", handler.ContinueStream)
	}
}

// RegisterChatRoutes registers chat routes
func RegisterChatRoutes(r *gin.RouterGroup, handler *session.Handler) {
	knowledgeChat := r.Group("/knowledge-chat")
	{
		knowledgeChat.POST("/:session_id", handler.KnowledgeQA)
	}

	agentChat := r.Group("/agent-chat")
	{
		agentChat.POST("/:session_id", handler.AgentQA)
	}

	knowledgeSearch := r.Group("/knowledge-search")
	{
		knowledgeSearch.POST("", handler.SearchKnowledge)
	}
}

// RegisterTenantRoutes registers tenant-related routes
func RegisterTenantRoutes(r *gin.RouterGroup, handler *handler.TenantHandler) {
	r.GET("/tenants/all", handler.ListAllTenants)
	r.GET("/tenants/search", handler.SearchTenants)
	tenantRoutes := r.Group("/tenants")
	{
		tenantRoutes.POST("", handler.CreateTenant)
		tenantRoutes.GET("/:id", handler.GetTenant)
		tenantRoutes.PUT("/:id", handler.UpdateTenant)
		tenantRoutes.DELETE("/:id", handler.DeleteTenant)
		tenantRoutes.GET("", handler.ListTenants)

		// Generic KV configuration management (tenant-level)
		// Tenant ID is obtained from authentication context
		tenantRoutes.GET("/kv/:key", handler.GetTenantKV)
		tenantRoutes.PUT("/kv/:key", handler.UpdateTenantKV)
	}
}

// RegisterModelRoutes registers model-related routes
func RegisterModelRoutes(r *gin.RouterGroup, handler *handler.ModelHandler) {
	models := r.Group("/models")
	{
		models.POST("", handler.CreateModel)
		models.GET("", handler.ListModels)
		models.GET("/:id", handler.GetModel)
		models.PUT("/:id", handler.UpdateModel)
		models.DELETE("/:id", handler.DeleteModel)
	}
}

func RegisterEvaluationRoutes(r *gin.RouterGroup, handler *handler.EvaluationHandler) {
	evaluationRoutes := r.Group("/evaluation")
	{
		evaluationRoutes.POST("/", handler.Evaluation)
		evaluationRoutes.GET("/", handler.GetEvaluationResult)
	}
}

// RegisterAuthRoutes registers authentication routes
func RegisterAuthRoutes(r *gin.RouterGroup, handler *handler.AuthHandler) {
	r.POST("/auth/register", handler.Register)
	r.POST("/auth/login", handler.Login)
	r.POST("/auth/refresh", handler.RefreshToken)
	r.GET("/auth/validate", handler.ValidateToken)
	r.POST("/auth/logout", handler.Logout)
	r.GET("/auth/me", handler.GetCurrentUser)
	r.POST("/auth/change-password", handler.ChangePassword)
}

func RegisterInitializationRoutes(r *gin.RouterGroup, handler *handler.InitializationHandler) {
	r.GET("/initialization/config/:kbId", handler.GetCurrentConfigByKB)
	r.POST("/initialization/initialize/:kbId", handler.InitializeByKB)
	r.PUT("/initialization/config/:kbId", handler.UpdateKBConfig)

	// Ollama endpoints
	r.GET("/initialization/ollama/status", handler.CheckOllamaStatus)
	r.GET("/initialization/ollama/models", handler.ListOllamaModels)
	r.POST("/initialization/ollama/models/check", handler.CheckOllamaModels)
	r.POST("/initialization/ollama/models/download", handler.DownloadOllamaModel)
	r.GET("/initialization/ollama/download/progress/:taskId", handler.GetDownloadProgress)
	r.GET("/initialization/ollama/download/tasks", handler.ListDownloadTasks)

	// Remote API endpoints
	r.POST("/initialization/remote/check", handler.CheckRemoteModel)
	r.POST("/initialization/embedding/test", handler.TestEmbeddingModel)
	r.POST("/initialization/rerank/check", handler.CheckRerankModel)
	r.POST("/initialization/multimodal/test", handler.TestMultimodalFunction)

	r.POST("/initialization/extract/text-relation", handler.ExtractTextRelations)
	r.POST("/initialization/extract/fabri-tag", handler.FabriTag)
	r.POST("/initialization/extract/fabri-text", handler.FabriText)
}

// RegisterSystemRoutes registers system information routes
func RegisterSystemRoutes(r *gin.RouterGroup, handler *handler.SystemHandler) {
	systemRoutes := r.Group("/system")
	{
		systemRoutes.GET("/info", handler.GetSystemInfo)
	}
}

// RegisterMCPServiceRoutes registers MCP service routes
func RegisterMCPServiceRoutes(r *gin.RouterGroup, handler *handler.MCPServiceHandler) {
	mcpServices := r.Group("/mcp-services")
	{
		// Create MCP service
		mcpServices.POST("", handler.CreateMCPService)
		// List MCP services
		mcpServices.GET("", handler.ListMCPServices)
		// Get MCP service by ID
		mcpServices.GET("/:id", handler.GetMCPService)
		// Update MCP service
		mcpServices.PUT("/:id", handler.UpdateMCPService)
		// Delete MCP service
		mcpServices.DELETE("/:id", handler.DeleteMCPService)
		// Test MCP service connection
		mcpServices.POST("/:id/test", handler.TestMCPService)
		// Get MCP service tools
		mcpServices.GET("/:id/tools", handler.GetMCPServiceTools)
		// Get MCP service resources
		mcpServices.GET("/:id/resources", handler.GetMCPServiceResources)
	}
}

// RegisterWebSearchRoutes registers web search routes
func RegisterWebSearchRoutes(r *gin.RouterGroup, webSearchHandler *handler.WebSearchHandler) {
	// Web search providers
	webSearch := r.Group("/web-search")
	{
		// Get available providers
		webSearch.GET("/providers", webSearchHandler.GetProviders)
	}
}
