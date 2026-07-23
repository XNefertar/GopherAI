package router

import (
	"GopherAI/common/aihelper"
	"GopherAI/middleware/jwt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {

	r := gin.Default()
	enterRouter := r.Group("/api/v1")
	{
		RegisterUserRouter(enterRouter.Group("/user"))
	}
	//后续登录的接口需要jwt鉴权
	{
		AIGroup := enterRouter.Group("/AI")
		AIGroup.Use(jwt.Auth())
		AIRouter(AIGroup)
	}

	{
		ImageGroup := enterRouter.Group("/image")
		ImageGroup.Use(jwt.Auth())
		ImageRouter(ImageGroup)
	}

	{
		KBGroup := enterRouter.Group("/kb")
		KBGroup.Use(jwt.Auth())
		KBRouter(KBGroup)
	}

	// 诊断端点：暴露路由器各层命中率统计（无需鉴权，内网使用）
	r.GET("/debug/router/stats", func(c *gin.Context) {
		router := aihelper.GetGlobalRouter()
		type statsProvider interface {
			Stats() aihelper.RouterStats
		}
		if sp, ok := router.(statsProvider); ok {
			s := sp.Stats()
			total := s.KeywordShortcut + s.EmbeddingHit + s.EmbeddingMiss
			l1Total := s.EmbeddingHit + s.EmbeddingMiss
			c.JSON(http.StatusOK, gin.H{
				"raw": s,
				"derived": gin.H{
					"total_requests":        total,
					"step0_shortcut_pct":    pct(s.KeywordShortcut, total),
					"l1_embedding_hit_pct":  pct(s.EmbeddingHit, l1Total),
					"l1_embedding_miss_pct": pct(s.EmbeddingMiss, l1Total),
					"l2_llm_classified_pct": pct(s.LLMClassified, s.EmbeddingMiss),
					"l2_llm_fallback_pct":   pct(s.LLMFallback, s.EmbeddingMiss),
					"l2_low_confidence_pct": pct(s.LowConfidence, s.EmbeddingMiss),
					"desc":                  "若 l1_embedding_hit_pct < 30% 或 l2_llm_fallback_pct > 40%，说明前两层带来的延迟大于收益，需调参或降级",
				},
			})
		} else {
			c.JSON(http.StatusOK, gin.H{"note": "current router does not expose stats"})
		}
	})

	return r
}

func pct(part, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}
