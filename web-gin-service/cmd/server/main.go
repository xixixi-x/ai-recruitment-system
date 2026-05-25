package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"final_homework/web-gin-service/internal/config"
	"final_homework/web-gin-service/internal/logicclient"
	"final_homework/web-gin-service/internal/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type Server struct {
	cfg config.Config
	cli *logicclient.Client
}

func main() {
	_ = godotenv.Load("../.env", ".env")
	cfg := config.Load()
	cli, err := logicclient.New(cfg.LogicGRPCAddr)
	if err != nil {
		panic(err)
	}
	s := &Server{cfg: cfg, cli: cli}

	r := gin.Default()
	origins := strings.Split(cfg.CORSOrigins, ",")
	r.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	api := r.Group("/api")
	api.POST("/auth/register", s.register)
	api.POST("/auth/login", s.login)
	api.GET("/jobs/public", s.publicJobs)

	hr := api.Group("/hr", middleware.Auth(cfg.JWTSecret, "hr"))
	hr.POST("/jobs", s.forward("hr.createJob"))
	hr.GET("/jobs", s.forward("hr.listJobs"))
	hr.GET("/applications", s.forward("hr.listApplications"))
	hr.GET("/applications/:id", s.hrApplicationDetail)
	hr.GET("/applications/:id/resume-url", s.hrResumeURL)
	hr.POST("/ai/chat", s.forward("hr.chat"))
	hr.GET("/ai/history", s.forward("hr.chatHistory"))

	cand := api.Group("/candidate", middleware.Auth(cfg.JWTSecret, "candidate"))
	cand.GET("/profile", s.forward("candidate.getProfile"))
	cand.PUT("/profile", s.forward("candidate.saveProfile"))
	cand.POST("/resume/sign-upload", s.forward("candidate.resumeSignUpload"))
	cand.POST("/resume/confirm", s.forward("candidate.resumeConfirm"))
	cand.POST("/jobs/:id/apply", s.candidateApply)
	cand.GET("/applications", s.forward("candidate.myApplications"))

	r.Run(":" + cfg.WebPort)
}

func bindJSON(c *gin.Context) map[string]any {
	var m map[string]any
	if c.Request.Body == nil {
		return map[string]any{}
	}
	_ = c.ShouldBindJSON(&m)
	if m == nil {
		m = map[string]any{}
	}
	return m
}

func (s *Server) ok(c *gin.Context, resp *logicclient.Response) {
	var data any
	if len(resp.Data) > 0 {
		_ = json.Unmarshal(resp.Data, &data)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": data})
}

func (s *Server) fail(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"code": status, "message": msg})
}

func (s *Server) register(c *gin.Context) {
	body := bindJSON(c)
	role, _ := body["role"].(string)
	if role != "hr" && role != "candidate" {
		s.fail(c, 400, "role must be hr or candidate")
		return
	}
	resp, err := s.cli.Call(c.Request.Context(), "auth.register", nil, body)
	if err != nil {
		s.fail(c, 400, err.Error())
		return
	}
	s.ok(c, resp)
}

func (s *Server) login(c *gin.Context) {
	body := bindJSON(c)
	resp, err := s.cli.Call(c.Request.Context(), "auth.login", nil, body)
	if err != nil {
		s.fail(c, 401, err.Error())
		return
	}
	var user struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	_ = json.Unmarshal(resp.Data, &user)
	token, err := middleware.SignToken(s.cfg.JWTSecret, user.ID, user.Username, user.Role)
	if err != nil {
		s.fail(c, 500, err.Error())
		return
	}
	data := gin.H{"token": token, "user": user}
	raw, _ := json.Marshal(data)
	resp.Data = raw
	s.ok(c, resp)
}

func (s *Server) publicJobs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	keyword := c.Query("keyword")
	resp, err := s.cli.Call(c.Request.Context(), "job.publicList", nil, gin.H{"page": page, "pageSize": pageSize, "keyword": keyword})
	if err != nil {
		s.fail(c, 400, err.Error())
		return
	}
	s.ok(c, resp)
}

func (s *Server) forward(op string) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := s.cli.Call(c.Request.Context(), op, middleware.Meta(c), bindJSON(c))
		if err != nil {
			s.fail(c, 400, err.Error())
			return
		}
		s.ok(c, resp)
	}
}

func (s *Server) hrApplicationDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	resp, err := s.cli.Call(c.Request.Context(), "hr.applicationDetail", middleware.Meta(c), gin.H{"id": id})
	if err != nil {
		s.fail(c, 404, err.Error())
		return
	}
	s.ok(c, resp)
}

func (s *Server) hrResumeURL(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	resp, err := s.cli.Call(c.Request.Context(), "hr.resumeDownloadURL", middleware.Meta(c), gin.H{"applicationId": id})
	if err != nil {
		s.fail(c, 404, err.Error())
		return
	}
	s.ok(c, resp)
}

func (s *Server) candidateApply(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	resp, err := s.cli.Call(c.Request.Context(), "candidate.applyJob", middleware.Meta(c), gin.H{"jobId": id})
	if err != nil {
		s.fail(c, 400, err.Error())
		return
	}
	s.ok(c, resp)
}
